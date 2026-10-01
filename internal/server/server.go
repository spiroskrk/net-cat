package server

import (
	"bufio"
	"fmt"
	"net"
	"net-cat/internal/session"
	"strings"
	"sync"
)

// Server coordinates admission across all accepted connections.
// Share one instance so clients entering names count toward the global limit.
type Server struct {
	activeConnections int
	mu                sync.Mutex
	sessionStart      session.Starter
	room              session.Room
}

// HandleConnection reserves capacity, validates a name, and starts a session.
// Admission owns closure and release until the starter accepts the handoff.
func (s *Server) HandleConnection(conn net.Conn) {
	// Reserve before name entry, and release the lock before any network I/O.
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

	// Admission and session cleanup may both call release; return the slot once.
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
	// Pass this same reader to the session to preserve bytes sent with the name.
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

// NewServer connects admission to a session starter and a shared room.
// Production supplies session.Start; independent tests can inject a fake starter.
func NewServer(starter session.Starter, room session.Room) *Server {
	srv := Server{sessionStart: starter, room: room}
	return &srv
}

// readName consumes one LF-terminated name with bounded retained storage.
// It returns the trimmed name and an oversize flag; reject the prefix when true.
// An unfinished name at EOF is discarded.
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
		// Leading whitespace does not count toward the trimmed name's byte limit.
		if len(name) == 0 && isSpace {
			continue
		}

		// Stop counting after crossing the limit, but keep draining to the newline.
		if byteCount <= 64 {
			byteCount += size
		}

		// Trailing whitespace is trimmed; only more non-space content proves oversize.
		if byteCount > 64 && !isSpace {
			tooLong = true
		}

		if byteCount <= 64 {
			// ReadRune replaces an invalid UTF-8 byte with U+FFFD of width one.
			// Reread that byte so accepted names preserve the original input.
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

// isNameSpace follows the same whitespace rules as the final name trimming.
func isNameSpace(ch rune) bool {
	trimmed := strings.TrimSpace(string(ch))
	if trimmed == "" {
		return true
	}
	return false
}
