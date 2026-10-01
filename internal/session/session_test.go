package session

import (
	"bufio"
	"net"
	"net-cat/internal/chat"
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
