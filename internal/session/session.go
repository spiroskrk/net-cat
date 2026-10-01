package session

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net-cat/internal/chat"
	"sync"
	"time"
)

const queueSize = 256

const writeTimeout = 10 * time.Second

const maxMessage = 4096

// Room is the membership and message API needed by a session.
// The room owns client IDs and formatting; the session owns socket I/O.
type Room interface {
	Join(name string, output chat.Destination) (chat.ClientID, error)
	Submit(id chat.ClientID, message string) error
	Leave(id chat.ClientID) error
}

// Starter is the admission handoff. A nil return transfers cleanup ownership
// to the session; an error leaves closure and capacity release with admission.
type Starter func(
	conn net.Conn,
	name string,
	reader *bufio.Reader,
	release func(),
	room Room,
) error

type session struct {
	conn    net.Conn
	room    Room
	release func()

	// A separate replay batch prevents long history from using live-event slots.
	history chan []string
	queue   chan string

	done        chan struct{}
	closeOnce   sync.Once
	historyOnce sync.Once
	err         error // First failure; inspect only after session cleanup completes.
}

// Start accepts ownership of an admitted connection and begins its session.
// Reusing admission's reader preserves buffered chat input. Registration happens
// in the background; the session handles any later failure and calls release.
func Start(conn net.Conn, name string, reader *bufio.Reader, release func(), room Room) error {
	s := &session{
		conn:    conn,
		room:    room,
		release: release,
		history: make(chan []string, 1),
		queue:   make(chan string, queueSize),
		done:    make(chan struct{}),
	}
	go s.run(name, reader)
	return nil // ownership is ours now; don't wait for Join
}

// Begin accepts the initial immutable history batch once, without waiting
// for the writer to deliver it to the socket.
func (s *session) Begin(history []string) error {
	select {
	case <-s.done:
		return errors.New("session closed")
	default:
	}
	// Once remembers the handoff even after the writer empties the channel.
	// Only the call that queues the batch clears its own local error.
	err := errors.New("session: history already delivered")

	s.historyOnce.Do(func() {
		select {
		case s.history <- history:
			err = nil
		default:
		}
	})

	return err
}

// Enqueue accepts a live event without waiting for socket delivery.
// A full queue returns an error so the caller can fail this session.
func (s *session) Enqueue(text string) error {
	// Check closure separately: one select could choose a ready send over done.
	// A failure concurrent with this call is still handled by socket cleanup.
	select {
	case <-s.done:
		return errors.New("session closed")
	default:
	}
	select {
	case s.queue <- text:
		return nil
	default:
		return errors.New("session: output queue full")
	}
}

// Fail records the first failure and stops socket I/O without waiting for
// room removal. run owns that cleanup, avoiding a callback into Leave here.
func (s *session) Fail(err error) {
	s.closeOnce.Do(func() {
		s.err = err
		close(s.done)
		// Closing done alone cannot unblock a goroutine inside a socket Read or Write.
		s.conn.Close()
	})
}

// run owns registration and final cleanup, so Leave and release are not
// duplicated by the reader, writer, or external failure callbacks.
func (s *session) run(name string, reader *bufio.Reader) {
	writerDone := make(chan struct{})
	go func() {
		s.writeLoop() // on exit it closes conn, which wakes the reader
		close(writerDone)
	}()

	// Join can trigger Fail before returning. Retain its successful ID so that
	// even this early failure removes the registered member during cleanup.
	id, err := s.room.Join(name, s)
	if err != nil {
		s.Fail(err) // wakes the writer; never joined, so no leave
	} else {
		s.readLoop(id, reader) // returns once the client is gone (it calls Fail)
	}

	// Reading has ended or never started; wait for the writer before freeing the slot.
	<-writerDone
	if err == nil {
		s.room.Leave(id)
	}
	s.release()
}

// writeLoop is the only socket writer after the admission handoff.
func (s *session) writeLoop() {
	defer s.conn.Close() // on any exit: wake the reader blocked in a read

	// Finish replay before the newcomer's own join notice and later live events.
	select {
	case history := <-s.history:
		for _, line := range history {
			if err := s.write(line); err != nil {
				s.Fail(err)
				return
			}
		}
	case <-s.done:
		return
	}

	// Live output includes room events and session-local validation errors.
	for {
		select {
		case line := <-s.queue:
			if err := s.write(line); err != nil {
				s.Fail(err)
				return
			}
		case <-s.done:
			return
		}
	}
}

// write gives each line its own delivery deadline, including each history line.
// Input stays untimed so quiet clients remain connected.
func (s *session) write(line string) error {
	s.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	_, err := io.WriteString(s.conn, line)
	return err
}

// readLoop turns socket input into Submit calls until the client is gone.
func (s *session) readLoop(id chat.ClientID, reader *bufio.Reader) {
	for {
		line, tooLong, err := readLine(reader)
		if err != nil {
			s.Fail(err) // EOF or closed conn: the client is gone
			return
		}
		if tooLong {
			// Use the same writer for errors; never broadcast them or add them to history.
			if err := s.Enqueue("Message too long. Maximum is 4096 bytes.\n"); err != nil {
				s.Fail(err)
				return
			}
			continue
		}
		if err := s.room.Submit(id, line); err != nil {
			s.Fail(err)
			return
		}
	}
}

// readLine drains through LF while retaining at most maxMessage+1 bytes.
// The extra byte allows CRLF at the limit. When tooLong is true, the returned
// prefix must not be submitted as a message.
func readLine(r *bufio.Reader) (string, bool, error) {
	var buf []byte
	n := 0 // content bytes seen, including a possible trailing '\r'
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", false, err // unfinished line at EOF is discarded
		}
		if b == '\n' {
			break
		}
		n++
		if n <= maxMessage+1 { // +1 leaves room for a '\r'
			buf = append(buf, b)
		}
	}
	// Only strip CR when the whole line fit; a truncated prefix may end in a
	// CR that was content rather than the actual line ending.
	if n > 0 && n == len(buf) && buf[n-1] == '\r' {
		buf = buf[:n-1] // CRLF: drop the '\r'
		n--
	}
	return string(buf), n > maxMessage, nil
}
