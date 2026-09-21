package server

import (
	"bufio"
	"fmt"
	"net"
	"net-cat/internal/session"
	"strings"
	"sync"
)

type Server struct {
	activeConnections int
	mu                sync.Mutex
	sessionStart      session.Starter
	room              session.Room
}

func (s *Server) HandleConnection(conn net.Conn) {
	s.mu.Lock()

	if s.activeConnections >= 10 {
		s.mu.Unlock()
		defer conn.Close()
		_, err := fmt.Fprint(conn, "Chat is full\n")
		if err != nil {
			return
		}
		return
	}
	s.activeConnections++
	s.mu.Unlock()

	var once sync.Once
	release := func() {
		once.Do(func() {
			s.mu.Lock()
			s.activeConnections--
			s.mu.Unlock()
		})
	}
	handedOff := false
	defer func() {
		if !handedOff {
			conn.Close()
			release()
		}
	}()

	_, err := fmt.Fprintln(conn, "Welcome to TCP-Chat!")
	if err != nil {
		return
	}
	_, err = fmt.Fprint(conn,
		"         _nnnn_\n"+
			"        dGGGGMMb\n"+
			"       @p~qp~~qMb\n"+
			"       M|@||@) M|\n"+
			"       @,----.JM|\n"+
			"      JS^\\__/  qKL\n"+
			"     dZP        qKRb\n"+
			"    dZP          qKKb\n"+
			"   fZP            SMMb\n"+
			"   HZM            MMMM\n"+
			"   FqM            MMMM\n"+
			" __| \".        |\\dS\"qML\n"+
			" |    `.       | `' \\Zq\n"+
			"_)      \\.___.,|     .'\n"+
			"\\____   )MMMMMP|   .'\n"+
			"     `-'       `--'\n"+
			"[ENTER YOUR NAME]: ",
	)
	if err != nil {
		return
	}
	reader := bufio.NewReader(conn)
	var name string
	tooLong := false
	for {
		name, tooLong, err = readName(reader)
		if err != nil {
			return
		}

		if name == "" {
			_, err = fmt.Fprintln(conn, "Invalid name. Please enter a non-empty name.")

			if err != nil {
				return
			}

			_, err = fmt.Fprint(conn, "[ENTER YOUR NAME]: ")

			if err != nil {
				return
			}
			continue
		}
		if tooLong {
			_, err = fmt.Fprintln(conn, "Name too long. Maximum is 64 bytes.")

			if err != nil {
				return
			}

			_, err = fmt.Fprint(conn, "[ENTER YOUR NAME]: ")

			if err != nil {
				return
			}
			continue
		}
		break
	}
	err = s.sessionStart(conn, name, reader, release, s.room)
	if err != nil {
		return
	}
	handedOff = true
}

func NewServer(starter session.Starter, room session.Room) *Server {
	srv := Server{sessionStart: starter, room: room}
	return &srv
}

func readName(r *bufio.Reader) (string, bool, error) {
	var name []byte
	tooLong := false
	byteCount := 0
	for {

		ch, size, err := r.ReadRune()

		if err != nil {
			return "", false, err
		}

		if ch == '\n' {
			break
		}
		isSpace := isNameSpace(ch)
		if len(name) == 0 && isSpace {
			continue
		}

		if byteCount <= 64 {
			byteCount += size
		}

		if byteCount > 64 && !isSpace {
			tooLong = true
		}

		if byteCount <= 64 {
			if ch == '\uFFFD' && size == 1 {
				err = r.UnreadRune()
				if err != nil {
					return "", false, err
				}

				b, err := r.ReadByte()
				if err != nil {
					return "", false, err
				}
				name = append(name, b)
			} else {
				name = append(name, string(ch)...)
			}
		}

	}

	return strings.TrimSpace(string(name)), tooLong, nil
}

func isNameSpace(ch rune) bool {
	trimmed := strings.TrimSpace(string(ch))
	if trimmed == "" {
		return true
	}
	return false
}
