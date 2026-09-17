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
	for {
		name, err = reader.ReadString('\n')
		if err != nil {
			return
		}
		name = strings.TrimSpace(name)
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
		if len(name) > 64 {
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
