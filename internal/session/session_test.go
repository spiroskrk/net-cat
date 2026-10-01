package session

import (
	"bufio"
	"errors"
	"net"
	"net-cat/internal/chat"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeRoom stands in for Aris's room and reports what the session did.
type fakeRoom struct {
	submits chan string
	left    chan chat.ClientID
}

func (r *fakeRoom) Join(name string, out chat.Destination) (chat.ClientID, error) {
	out.Begin([]string{"old message\n"})
	out.Enqueue(name + " has joined\n")
	return 7, nil
}

func (r *fakeRoom) Submit(id chat.ClientID, message string) error {
	r.submits <- message
	return nil
}

func (r *fakeRoom) Leave(id chat.ClientID) error {
	r.left <- id
	return nil
}

func TestSessionLifecycle(t *testing.T) {
	server, client := net.Pipe()
	room := &fakeRoom{submits: make(chan string, 1), left: make(chan chat.ClientID, 1)}
	released := make(chan struct{})

	err := Start(server, "Maria", bufio.NewReader(server), func() { close(released) }, room)
	if err != nil {
		t.Fatalf("Start returned %v", err)
	}

	// 1. History first, then live events.
	out := bufio.NewReader(client)
	for _, want := range []string{"old message\n", "Maria has joined\n"} {
		got, err := out.ReadString('\n')
		if err != nil || got != want {
			t.Fatalf("got %q (err %v), want %q", got, err, want)
		}
	}

	// 2. Input reaches Submit without its CRLF.
	client.Write([]byte("hello\r\n"))
	select {
	case got := <-room.submits:
		if got != "hello" {
			t.Fatalf("Submit got %q, want %q", got, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("Submit was never called")
	}

	// 3. Client disconnects: Leave with the right ID, then release.
	client.Close()
	select {
	case id := <-room.left:
		if id != 7 {
			t.Fatalf("Leave got ID %d, want 7", id)
		}
	case <-time.After(time.Second):
		t.Fatal("Leave was never called")
	}
	select {
	case <-released:
	case <-time.After(time.Second):
		t.Fatal("release was never called")
	}
}

func TestTooLongMessage(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	room := &fakeRoom{submits: make(chan string, 1), left: make(chan chat.ClientID, 1)}

	Start(server, "Maria", bufio.NewReader(server), func() {}, room)

	out := bufio.NewReader(client)
	out.ReadString('\n') // history
	out.ReadString('\n') // join notice

	// 4097 bytes: one over the limit.
	client.Write([]byte(strings.Repeat("a", maxMessage+1) + "\n"))
	client.SetReadDeadline(time.Now().Add(time.Second))
	got, _ := out.ReadString('\n')
	if got != "Message too long. Maximum is 4096 bytes.\n" {
		t.Fatalf("got %q, want the too-long error", got)
	}

	// The connection must still work afterwards.
	client.Write([]byte("hi\n"))
	select {
	case msg := <-room.submits:
		if msg != "hi" {
			t.Fatalf("Submit got %q, want %q", msg, "hi")
		}
	case <-time.After(time.Second):
		t.Fatal("Submit was never called after the long line")
	}
}

func TestBeginOnlyOnce(t *testing.T) {
	s := &session{
		history: make(chan []string, 1),
		done:    make(chan struct{}),
	}

	if err := s.Begin([]string{"old message\n"}); err != nil {
		t.Fatalf("first Begin returned %v", err)
	}

	// Consume the batch as the writer would, so the channel has space again.
	select {
	case history := <-s.history:
		if len(history) != 1 || history[0] != "old message\n" {
			t.Fatalf("first history batch = %q, want [\"old message\\n\"]", history)
		}
	default:
		t.Fatal("first Begin did not deliver history")
	}

	if err := s.Begin([]string{"second message\n"}); err == nil {
		t.Fatal("second Begin succeeded after the first batch was consumed")
	}

	select {
	case history := <-s.history:
		t.Fatalf("rejected Begin delivered another history batch: %q", history)
	default:
	}
}

func TestFailedSessionRejectsOutput(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	s := &session{
		conn:    server,
		history: make(chan []string, 1),
		queue:   make(chan string, queueSize),
		done:    make(chan struct{}),
	}
	s.Fail(errors.New("forced failure"))

	// Both buffers have space; rejection must come from the failed state.
	if err := s.Begin([]string{"old message\n"}); err == nil {
		t.Error("Begin accepted history after failure")
	}
	if err := s.Enqueue("live message\n"); err == nil {
		t.Error("Enqueue accepted an event after failure")
	}
	if len(s.history) != 0 || len(s.queue) != 0 {
		t.Fatal("failed session retained rejected output")
	}
}

type replayBlockingConn struct {
	net.Conn
	writeStarted chan struct{}
	writeOnce    sync.Once
}

func (c *replayBlockingConn) Write(p []byte) (int, error) {
	c.writeOnce.Do(func() { close(c.writeStarted) })
	return c.Conn.Write(p)
}

// This regression must unblock through Close, never a write timeout.
func (c *replayBlockingConn) SetWriteDeadline(time.Time) error {
	return nil
}

func TestFailDuringHistoryReplay(t *testing.T) {
	server, client := net.Pipe()
	conn := &replayBlockingConn{
		Conn:         server,
		writeStarted: make(chan struct{}),
	}
	room := &fakeRoom{
		submits: make(chan string, 1),
		left:    make(chan chat.ClientID, 2),
	}
	released := make(chan struct{}, 2)
	s := &session{
		conn:    conn,
		room:    room,
		release: func() { released <- struct{}{} },
		history: make(chan []string, 1),
		queue:   make(chan string, queueSize),
		done:    make(chan struct{}),
	}
	runDone := make(chan struct{})
	t.Cleanup(func() {
		client.Close()
		server.Close()
		select {
		case <-runDone:
		case <-time.After(2 * time.Second):
			t.Error("session did not stop after test cleanup")
		}
	})
	go func() {
		s.run("Maria", bufio.NewReader(conn))
		close(runDone)
	}()

	// The peer stays open without reading, so the first replay write cannot finish.
	select {
	case <-conn.writeStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("history replay never started")
	}

	failure := errors.New("forced replay failure")
	failureReturned := make(chan struct{}, 2)
	for i := 0; i < 2; i++ {
		go func() {
			s.Fail(failure)
			failureReturned <- struct{}{}
		}()
	}
	for i := 0; i < 2; i++ {
		select {
		case <-failureReturned:
		case <-time.After(2 * time.Second):
			t.Fatal("Fail did not return")
		}
	}
	select {
	case <-runDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Fail did not stop the session during blocked replay")
	}

	// run waits for its writer before returning, so both workers have finished.
	if got := len(room.left); got != 1 {
		t.Fatalf("Leave called %d times, want 1", got)
	}
	if id := <-room.left; id != 7 {
		t.Fatalf("Leave got ID %d, want 7", id)
	}
	if got := len(released); got != 1 {
		t.Fatalf("capacity released %d times, want 1", got)
	}
}
