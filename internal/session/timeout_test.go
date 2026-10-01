package session

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net-cat/internal/chat"
	"sync"
	"testing"
	"time"
)

type timeoutDeadlineRecord struct {
	requested time.Time
	observed  time.Time
}

// Record the production deadline before optionally shortening the pipe's actual
// write deadline, so timeout tests stay fast without weakening the policy check.
type timeoutObservedConn struct {
	net.Conn
	mu                  sync.Mutex
	deadlines           []timeoutDeadlineRecord
	writes              int
	readDeadlines       int
	combinedDeadlines   int
	shortenedWriteLimit time.Duration
}

func (c *timeoutObservedConn) SetWriteDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.deadlines = append(c.deadlines, timeoutDeadlineRecord{deadline, time.Now()})
	c.mu.Unlock()
	if c.shortenedWriteLimit > 0 {
		deadline = time.Now().Add(c.shortenedWriteLimit)
	}
	return c.Conn.SetWriteDeadline(deadline)
}

func (c *timeoutObservedConn) SetReadDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.readDeadlines++
	c.mu.Unlock()
	return c.Conn.SetReadDeadline(deadline)
}

func (c *timeoutObservedConn) SetDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.combinedDeadlines++
	c.mu.Unlock()
	return c.Conn.SetDeadline(deadline)
}

func (c *timeoutObservedConn) Write(p []byte) (int, error) {
	c.mu.Lock()
	c.writes++
	c.mu.Unlock()
	return c.Conn.Write(p)
}

func timeoutAssertDeadlines(t *testing.T, conn *timeoutObservedConn, wantWrites int) {
	t.Helper()
	conn.mu.Lock()
	defer conn.mu.Unlock()
	if conn.writes != wantWrites || len(conn.deadlines) != wantWrites {
		t.Fatalf("writes/deadlines = %d/%d, want %d/%d", conn.writes, len(conn.deadlines), wantWrites, wantWrites)
	}
	for i, record := range conn.deadlines {
		// Allow scheduling overhead between time.Now in write and this wrapper.
		remaining := record.requested.Sub(record.observed)
		if remaining < 9*time.Second || remaining > 10*time.Second {
			t.Fatalf("write %d requested deadline in %v, want approximately 10s", i, remaining)
		}
		if i > 0 && !record.requested.After(conn.deadlines[i-1].requested) {
			t.Fatalf("write %d reused the previous message's deadline", i)
		}
	}
	if conn.readDeadlines != 0 || conn.combinedDeadlines != 0 {
		t.Fatalf("session imposed input deadlines: read=%d, combined=%d", conn.readDeadlines, conn.combinedDeadlines)
	}
}

func timeoutWait(t *testing.T, done <-chan struct{}, description string) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", description)
	}
}

func timeoutReadLine(t *testing.T, reader *bufio.Reader, want string) {
	t.Helper()
	got, err := reader.ReadString('\n')
	if err != nil || got != want {
		t.Fatalf("read = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestWriteDeadlineRenewedForHistoryAndLiveEvents(t *testing.T) {
	server, client := net.Pipe()
	conn := &timeoutObservedConn{Conn: server}
	s := &session{
		conn:    conn,
		history: make(chan []string, 1),
		queue:   make(chan string, queueSize),
		done:    make(chan struct{}),
	}
	writerDone := make(chan struct{})
	t.Cleanup(func() {
		s.Fail(errors.New("test cleanup"))
		client.Close()
		timeoutWait(t, writerDone, "output worker cleanup")
	})
	go func() {
		s.writeLoop()
		close(writerDone)
	}()
	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := s.Begin([]string{"old one\n", "old two\n"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Enqueue("joined\n"); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(client)
	for _, line := range []string{"old one\n", "old two\n", "joined\n"} {
		timeoutReadLine(t, reader, line)
	}
	if err := s.Enqueue("later live event\n"); err != nil {
		t.Fatal(err)
	}
	timeoutReadLine(t, reader, "later live event\n")
	timeoutAssertDeadlines(t, conn, 4)
}

type timeoutRecordingRoom struct {
	*chat.Room
	mu     sync.Mutex
	ids    map[string]chat.ClientID
	leaves map[chat.ClientID]int
}

func (r *timeoutRecordingRoom) Join(name string, out chat.Destination) (chat.ClientID, error) {
	id, err := r.Room.Join(name, out)
	if err == nil {
		r.mu.Lock()
		r.ids[name] = id
		r.mu.Unlock()
	}
	return id, err
}

func (r *timeoutRecordingRoom) Leave(id chat.ClientID) error {
	r.mu.Lock()
	r.leaves[id]++
	r.mu.Unlock()
	return r.Room.Leave(id)
}

type timeoutRunningSession struct {
	s        *session
	done     chan struct{}
	releases int // Read only after done closes.
}

func timeoutStartSession(t *testing.T, conn net.Conn, name string, room Room) *timeoutRunningSession {
	t.Helper()
	running := &timeoutRunningSession{done: make(chan struct{})}
	running.s = &session{
		conn:    conn,
		room:    room,
		release: func() { running.releases++ },
		history: make(chan []string, 1),
		queue:   make(chan string, queueSize),
		done:    make(chan struct{}),
	}
	t.Cleanup(func() {
		conn.Close()
		timeoutWait(t, running.done, name+" session cleanup")
	})
	go func() {
		running.s.run(name, bufio.NewReader(conn))
		// run waits for its writer; done also makes the release count safe to read.
		close(running.done)
	}()
	return running
}

func TestWriteTimeoutCleansUpOnlyAffectedSession(t *testing.T) {
	fixed := time.Date(2020, 1, 20, 16, 3, 43, 0, time.UTC)
	room := &timeoutRecordingRoom{
		Room:   chat.NewRoom(func() time.Time { return fixed }),
		ids:    make(map[string]chat.ClientID),
		leaves: make(map[chat.ClientID]int),
	}
	healthyServer, healthyClient := net.Pipe()
	t.Cleanup(func() { healthyClient.Close() })
	if err := healthyClient.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	healthyConn := &timeoutObservedConn{Conn: healthyServer}
	healthy := timeoutStartSession(t, healthyConn, "Healthy", room)
	healthyReader := bufio.NewReader(healthyClient)
	timeoutReadLine(t, healthyReader, "Healthy has joined our chat...\n")
	if _, err := io.WriteString(healthyClient, "seed\n"); err != nil {
		t.Fatal(err)
	}
	timeoutReadLine(t, healthyReader, "[2020-01-20 16:03:43][Healthy]:seed\n")

	slowServer, slowClient := net.Pipe()
	t.Cleanup(func() { slowClient.Close() })
	slowConn := &timeoutObservedConn{Conn: slowServer, shortenedWriteLimit: 30 * time.Millisecond}
	slow := timeoutStartSession(t, slowConn, "Slow", room)
	// Slow never reads, so its replay write must end through an actual deadline.
	timeoutReadLine(t, healthyReader, "Slow has joined our chat...\n")
	timeoutWait(t, slow.done, "timed-out session and its output worker")
	var timeoutError net.Error
	if !errors.As(slow.s.err, &timeoutError) || !timeoutError.Timeout() {
		t.Fatalf("session failure = %v, want a socket timeout", slow.s.err)
	}
	timeoutAssertDeadlines(t, slowConn, 1)
	if slow.releases != 1 {
		t.Fatalf("timed-out session released capacity %d times, want 1", slow.releases)
	}
	room.mu.Lock()
	slowID := room.ids["Slow"]
	leaves := room.leaves[slowID]
	room.mu.Unlock()
	if slowID == 0 || leaves != 1 {
		t.Fatalf("timed-out membership ID/Leave count = %d/%d, want nonzero/1", slowID, leaves)
	}
	if err := room.Submit(slowID, "must not remain registered"); err == nil {
		t.Fatal("timed-out client still has room membership")
	}
	timeoutReadLine(t, healthyReader, "Slow has left our chat...\n")

	// Healthy was quiet while Slow failed, and still has both input and output.
	if _, err := io.WriteString(healthyClient, "after timeout\n"); err != nil {
		t.Fatal(err)
	}
	timeoutReadLine(t, healthyReader, "[2020-01-20 16:03:43][Healthy]:after timeout\n")
	timeoutAssertDeadlines(t, healthyConn, 5)
	select {
	case <-healthy.done:
		t.Fatal("healthy participant was disconnected by another client's timeout")
	default:
	}
}
