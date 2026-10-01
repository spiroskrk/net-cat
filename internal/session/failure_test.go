package session

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net-cat/internal/chat"
	"strings"
	"testing"
	"time"
)

type failureRoom struct {
	joinFn   func(string, chat.Destination) (chat.ClientID, error)
	submitFn func(chat.ClientID, string) error
	leaveFn  func(chat.ClientID) error
}

func (r *failureRoom) Join(name string, output chat.Destination) (chat.ClientID, error) {
	return r.joinFn(name, output)
}

func (r *failureRoom) Submit(id chat.ClientID, message string) error {
	if r.submitFn == nil {
		return errors.New("unexpected Submit in failure test")
	}
	return r.submitFn(id, message)
}

func (r *failureRoom) Leave(id chat.ClientID) error {
	return r.leaveFn(id)
}

// This destination consumes events immediately; only the tested socket is slow.
type failureObserver struct {
	lines    chan string
	failures chan error
}

func (o *failureObserver) Begin([]string) error { return nil }

func (o *failureObserver) Enqueue(line string) error {
	select {
	case o.lines <- line:
		return nil
	default:
		return errors.New("test observer buffer full")
	}
}

func (o *failureObserver) Fail(err error) {
	select {
	case o.failures <- err:
	default:
	}
}

func startFailureSession(t *testing.T, conn, client net.Conn, room Room) (*session, <-chan struct{}, <-chan struct{}) {
	t.Helper()
	released := make(chan struct{}, 2)
	s := &session{
		conn:    conn,
		room:    room,
		release: func() { released <- struct{}{} },
		history: make(chan []string, 1),
		queue:   make(chan string, queueSize),
		done:    make(chan struct{}),
	}
	stopped := make(chan struct{})
	t.Cleanup(func() {
		client.Close()
		conn.Close()
		select {
		case <-stopped:
		case <-time.After(2 * time.Second):
			t.Error("failure-test session did not stop during cleanup")
		}
	})
	go func() {
		s.run("Slow", bufio.NewReader(conn))
		close(stopped)
	}()
	return s, stopped, released
}

func assertFailureSessionStopped(t *testing.T, client net.Conn, stopped, released <-chan struct{}) {
	t.Helper()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("session workers did not stop")
	}
	if got := len(released); got != 1 {
		t.Fatalf("capacity released %d times, want 1", got)
	}
	// net.Pipe rejects deadline changes when its peer is already closed.
	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil && !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal(err)
	}
	var data [1]byte
	if n, err := client.Read(data[:]); n != 0 || err != io.EOF {
		t.Fatalf("client read after cleanup = (%d, %v), want (0, EOF)", n, err)
	}
}

func TestSessionQueueOverflowDuringReplay(t *testing.T) {
	realRoom := chat.NewRoom(time.Now)
	healthy := &failureObserver{
		lines:    make(chan string, 1024),
		failures: make(chan error, 1),
	}
	healthyID, err := realRoom.Join("Healthy", healthy)
	if err != nil {
		t.Fatal(err)
	}
	// More history lines than live slots must not exhaust the live queue.
	for i := 0; i < 257; i++ {
		if err := realRoom.Submit(healthyID, "historical message"); err != nil {
			t.Fatal(err)
		}
	}

	joined := make(chan chat.ClientID, 1)
	left := make(chan chat.ClientID, 2)
	room := &failureRoom{
		joinFn: func(name string, output chat.Destination) (chat.ClientID, error) {
			id, err := realRoom.Join(name, output)
			if err == nil {
				joined <- id
			}
			return id, err
		},
		submitFn: realRoom.Submit,
		leaveFn: func(id chat.ClientID) error {
			err := realRoom.Leave(id)
			left <- id
			return err
		},
	}
	server, client := net.Pipe()
	conn := &replayBlockingConn{Conn: server, writeStarted: make(chan struct{})}
	s, stopped, released := startFailureSession(t, conn, client, room)
	var slowID chat.ClientID
	select {
	case slowID = <-joined:
	case <-time.After(2 * time.Second):
		t.Fatal("slow session did not join")
	}
	select {
	case <-conn.writeStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("history replay did not start")
	}
	// The writer is in history, so only the own-join notice uses a live slot.
	if got := len(s.queue); got != 1 {
		t.Fatalf("live queue after starting 257-line history = %d, want 1", got)
	}
	for i := 0; i < 255; i++ {
		if err := realRoom.Submit(healthyID, "live message"); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(s.queue); got != 256 {
		t.Fatalf("pending live events = %d, want 256", got)
	}
	select {
	case <-s.done:
		t.Fatal("session failed before the 256-event queue overflowed")
	default:
	}

	// The real room must turn an Enqueue error into Fail without blocking.
	submitted := make(chan error, 1)
	go func() { submitted <- realRoom.Submit(healthyID, "overflow event") }()
	select {
	case err := <-submitted:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("overflow blocked a healthy sender")
	}
	assertFailureSessionStopped(t, client, stopped, released)
	if len(left) != 1 {
		t.Fatalf("Leave called %d times, want 1", len(left))
	}
	if id := <-left; id != slowID {
		t.Fatalf("Leave got ID %d, want %d", id, slowID)
	}
	if err := realRoom.Submit(slowID, "ghost member"); err == nil {
		t.Fatal("failed session retained room membership")
	}
	if err := realRoom.Submit(healthyID, "still usable"); err != nil {
		t.Fatal(err)
	}
	departures, continued := 0, false
	for len(healthy.lines) > 0 {
		line := <-healthy.lines
		if line == "Slow has left our chat...\n" {
			departures++
		}
		if strings.HasSuffix(line, "[Healthy]:still usable\n") {
			continued = true
		}
	}
	if departures != 1 || !continued {
		t.Fatalf("healthy participant: departures=%d, continued=%v", departures, continued)
	}
	select {
	case err := <-healthy.failures:
		t.Fatalf("healthy participant failed: %v", err)
	default:
	}
}

func TestSessionRegistrationFailure(t *testing.T) {
	for _, duringReplay := range []bool{false, true} {
		name := "before_history"
		if duringReplay {
			name = "during_history"
		}
		t.Run(name, func(t *testing.T) {
			server, client := net.Pipe()
			conn := &replayBlockingConn{Conn: server, writeStarted: make(chan struct{})}
			rejected := errors.New("registration rejected")
			left := make(chan chat.ClientID, 2)
			room := &failureRoom{
				joinFn: func(_ string, output chat.Destination) (chat.ClientID, error) {
					if duringReplay {
						if err := output.Begin([]string{"old message\n"}); err != nil {
							return 0, err
						}
						select {
						case <-conn.writeStarted:
						case <-time.After(2 * time.Second):
							return 0, errors.New("history replay did not start")
						}
					}
					return 0, rejected
				},
				leaveFn: func(id chat.ClientID) error { left <- id; return nil },
			}
			s, stopped, released := startFailureSession(t, conn, client, room)
			assertFailureSessionStopped(t, client, stopped, released)
			if s.err != rejected {
				t.Fatalf("failure reason = %v, want registration rejection", s.err)
			}
			if len(left) != 0 {
				t.Fatal("failed registration caused a false room departure")
			}
		})
	}
}

func TestSessionFailureBeforeJoinReturnsID(t *testing.T) {
	server, client := net.Pipe()
	failed := errors.New("failed before Join returned")
	left := make(chan chat.ClientID, 2)
	room := &failureRoom{
		joinFn: func(_ string, output chat.Destination) (chat.ClientID, error) {
			if err := output.Begin(nil); err != nil {
				return 0, err
			}
			output.Fail(failed)
			return 7, nil
		},
		leaveFn: func(id chat.ClientID) error { left <- id; return nil },
	}
	s, stopped, released := startFailureSession(t, server, client, room)
	assertFailureSessionStopped(t, client, stopped, released)
	if s.err != failed {
		t.Fatalf("failure reason = %v, want pre-return failure", s.err)
	}
	if len(left) != 1 {
		t.Fatalf("Leave called %d times after Join returned, want 1", len(left))
	}
	if id := <-left; id != 7 {
		t.Fatalf("Leave got ID %d, want 7", id)
	}
}
