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

type Room interface {
	Join(name string, output chat.Destination) (chat.ClientID, error)
	Submit(id chat.ClientID, message string) error
	Leave(id chat.ClientID) error
}

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

	history chan []string // begin hands the replay batch to the writer, once
	queue   chan string   // live events

	done        chan struct{}
	closeOnce   sync.Once
	historyOnce sync.Once
	err         error
}

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

func (s *session) Begin(history []string) error {
	select {
	case <-s.done:
		return errors.New("session closed")
	default:
	}
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

func (s *session) Enqueue(text string) error {
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

func (s *session) Fail(err error) {
	s.closeOnce.Do(func() {
		s.err = err
		close(s.done)
		s.conn.Close()
	})
}

// run owns the whole lifecycle, so Leace and release happen once by construction.
func (s *session) run(name string, reader *bufio.Reader) {
	writerDone := make(chan struct{})
	go func() {
		s.writeLoop() // on exit it closes conn, which wakes the reader
		close(writerDone)
	}()

	id, err := s.room.Join(name, s)
	if err != nil {
		s.Fail(err) // wakes the writer; never joined, so no leave
	} else {
		s.readLoop(id, reader) // returns once the client is gone (it calls Fail)
	}

	<-writerDone
	if err == nil {
		s.room.Leave(id)
	}
	s.release()
}

func (s *session) writeLoop() {
	defer s.conn.Close() // on any exit: wake the reader blocked in a read

	// 1. History first. the room calls Begin during Join.
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

	// 2. Live events until the session ends.
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

// write sends one line with its own 10-second deadline.
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

// readLine reads one line but never store more than maxMessage+1 bytes.
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
	if n > 0 && n == len(buf) && buf[n-1] == '\r' {
		buf = buf[:n-1] // CRLF: drop the '\r'
		n--
	}
	return string(buf), n > maxMessage, nil
}
