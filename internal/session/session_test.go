package session

import (
	"bufio"
	"net"
	"net-cat/internal/chat"
	"strings"
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
