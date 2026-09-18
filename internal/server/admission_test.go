package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net-cat/internal/chat"
	"net-cat/internal/session"
	"strings"
	"sync"
	"testing"
	"time"
)

const admissionTimeout = 5 * time.Second
const namePrompt = "[ENTER YOUR NAME]: "

// Independent fixture from docs/golden_tests.md, including the prompt's space.
const welcomeFixture = "Welcome to TCP-Chat!\n" +
	"         _nnnn_\n" +
	"        dGGGGMMb\n" +
	"       @p~qp~~qMb\n" +
	"       M|@||@) M|\n" +
	"       @,----.JM|\n" +
	"      JS^\\__/  qKL\n" +
	"     dZP        qKRb\n" +
	"    dZP          qKKb\n" +
	"   fZP            SMMb\n" +
	"   HZM            MMMM\n" +
	"   FqM            MMMM\n" +
	" __| \".        |\\dS\"qML\n" +
	" |    `.       | `' \\Zq\n" +
	"_)      \\.___.,|     .'\n" +
	"\\____   )MMMMMP|   .'\n" +
	"     `-'       `--'\n" + namePrompt

type admissionRecord struct {
	conn    net.Conn
	name    string
	reader  *bufio.Reader
	release func()
	room    session.Room
}

// The fake accepts ownership; cleanup stands in for the real session's cleanup.
func recordingServer(t *testing.T, room session.Room) (*Server, <-chan admissionRecord) {
	t.Helper()
	records := make(chan admissionRecord, 32)
	starter := func(conn net.Conn, name string, reader *bufio.Reader, release func(), room session.Room) error {
		t.Cleanup(func() {
			conn.Close()
			release()
		})
		records <- admissionRecord{conn, name, reader, release, room}
		return nil
	}
	return NewServer(starter, room), records
}

type admissionPeer struct {
	server net.Conn
	client net.Conn
	reader *bufio.Reader
	done   chan struct{}
}

// A pipe gives deterministic read-ahead; capacity also exercises real local TCP.
func startAdmission(t *testing.T, srv *Server, tcp bool, start <-chan struct{}) *admissionPeer {
	t.Helper()
	var serverConn, clientConn net.Conn
	if tcp {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		if err := listener.(*net.TCPListener).SetDeadline(time.Now().Add(admissionTimeout)); err != nil {
			t.Fatal(err)
		}
		clientConn, err = net.DialTimeout("tcp", listener.Addr().String(), admissionTimeout)
		if err != nil {
			t.Fatal(err)
		}
		serverConn, err = listener.Accept()
		if err != nil {
			clientConn.Close()
			t.Fatal(err)
		}
	} else {
		serverConn, clientConn = net.Pipe()
	}
	p := &admissionPeer{serverConn, clientConn, bufio.NewReader(clientConn), make(chan struct{})}
	// Register cleanup before any operation that can fail the test.
	t.Cleanup(func() {
		clientConn.Close()
		serverConn.Close()
		select {
		case <-p.done:
		case <-time.After(admissionTimeout):
			t.Error("admission handler did not stop during cleanup")
		}
	})
	deadline := time.Now().Add(admissionTimeout)
	for _, conn := range []net.Conn{serverConn, clientConn} {
		if err := conn.SetDeadline(deadline); err != nil {
			close(p.done) // No handler has started, so cleanup need not wait.
			t.Fatal(err)
		}
	}
	go func() {
		defer close(p.done)
		if start != nil {
			<-start
		}
		srv.HandleConnection(serverConn)
	}()
	return p
}

func waitAdmission(t *testing.T, p *admissionPeer) {
	t.Helper()
	select {
	case <-p.done:
	case <-time.After(admissionTimeout):
		t.Fatal("admission handler did not finish")
	}
}

func readExact(t *testing.T, reader io.Reader, want string) {
	t.Helper()
	got := make([]byte, len(want))
	if _, err := io.ReadFull(reader, got); err != nil {
		t.Fatalf("reading %q: %v", want, err)
	}
	if string(got) != want {
		t.Fatalf("output mismatch\nwant: %q\n got: %q", want, got)
	}
}

func writeInput(t *testing.T, conn net.Conn, input string) {
	t.Helper()
	if _, err := io.WriteString(conn, input); err != nil {
		t.Fatalf("writing client input: %v", err)
	}
}

func requireCapacity(t *testing.T, srv *Server, want int) {
	t.Helper()
	// Other clients may still be entering names, so read under the same mutex.
	srv.mu.Lock()
	got := srv.activeConnections
	srv.mu.Unlock()
	if got != want {
		t.Fatalf("expected %d active connections, got %d", want, got)
	}
}

func receiveHandoff(t *testing.T, records <-chan admissionRecord) admissionRecord {
	t.Helper()
	select {
	case record := <-records:
		return record
	case <-time.After(admissionTimeout):
		t.Fatal("starter was not called")
		return admissionRecord{}
	}
}

func requireNoHandoff(t *testing.T, records <-chan admissionRecord) {
	t.Helper()
	select {
	case record := <-records:
		t.Fatalf("unexpected handoff for name %q", record.name)
	default:
	}
}

// Admission must forward the room, without registering or announcing clients.
type admissionRoom struct{ t *testing.T }

func (r *admissionRoom) Join(string, chat.Destination) (chat.ClientID, error) {
	r.t.Error("admission called Room.Join")
	return 0, errors.New("unexpected Join")
}
func (r *admissionRoom) Submit(chat.ClientID, string) error {
	r.t.Error("admission called Room.Submit")
	return errors.New("unexpected Submit")
}
func (r *admissionRoom) Leave(chat.ClientID) error {
	r.t.Error("admission called Room.Leave")
	return errors.New("unexpected Leave")
}

func TestHandleConnectionPreservesBufferedMessage(t *testing.T) {
	room := &admissionRoom{t}
	srv, records := recordingServer(t, room)
	p := startAdmission(t, srv, false, nil)
	readExact(t, p.reader, welcomeFixture)
	// One pipe write puts both lines into the admission reader's buffer.
	writeInput(t, p.client, "  Kostis  \r\nhello\n")
	waitAdmission(t, p)
	got := receiveHandoff(t, records)
	if got.conn != p.server || got.room != room || got.name != "Kostis" || got.reader == nil || got.release == nil {
		t.Fatalf("incorrect handoff: %+v", got)
	}
	readExact(t, got.reader, "hello\n")
	requireCapacity(t, srv, 1)

	// Prove the socket remains usable after HandleConnection returns.
	written := make(chan error, 1)
	go func() {
		_, err := io.WriteString(got.conn, "session is open\n")
		written <- err
	}()
	readExact(t, p.reader, "session is open\n")
	select {
	case err := <-written:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(admissionTimeout):
		t.Fatal("session write did not finish")
	}
	got.conn.Close()
	got.release()
	requireCapacity(t, srv, 0)
}

func TestHandleConnectionNameValidation(t *testing.T) {
	for _, tc := range []struct{ label, input, name, response string }{
		{"trim LF", " \tKostis \t\n", "Kostis", ""},
		{"CRLF", "Aris\r\n", "Aris", ""},
		{"64 bytes", strings.Repeat("a", 64) + "\n", strings.Repeat("a", 64), ""},
		{"64 UTF-8 bytes", strings.Repeat("α", 32) + "\n", strings.Repeat("α", 32), ""},
		{"long surrounding whitespace", strings.Repeat(" ", 8192) + "Kostis" + strings.Repeat("\t", 8192) + "\n", "Kostis", ""},
		{"empty", "\n", "", "Invalid name. Please enter a non-empty name.\n"},
		{"whitespace", " \t\r\n", "", "Invalid name. Please enter a non-empty name.\n"},
		{"65 bytes", strings.Repeat("a", 65) + "\n", "", "Name too long. Maximum is 64 bytes.\n"},
		{"66 UTF-8 bytes", strings.Repeat("α", 33) + "\n", "", "Name too long. Maximum is 64 bytes.\n"},
		{"oversized across buffers", strings.Repeat("a", 16384) + "\n", "", "Name too long. Maximum is 64 bytes.\n"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			srv, records := recordingServer(t, nil)
			p := startAdmission(t, srv, false, nil)
			readExact(t, p.reader, welcomeFixture)
			writeInput(t, p.client, tc.input)
			want := tc.name
			if tc.response != "" {
				readExact(t, p.reader, tc.response+namePrompt)
				requireNoHandoff(t, records)
				requireCapacity(t, srv, 1)
				// A valid retry on the same connection must still be admitted.
				want = "Retry"
				writeInput(t, p.client, want+"\n")
			}
			waitAdmission(t, p)
			got := receiveHandoff(t, records)
			if got.name != want {
				t.Fatalf("expected name %q, got %q", want, got.name)
			}
			requireNoHandoff(t, records)
		})
	}
	t.Run("duplicate names", func(t *testing.T) {
		srv, records := recordingServer(t, nil)
		for i := 0; i < 2; i++ {
			p := startAdmission(t, srv, false, nil)
			readExact(t, p.reader, welcomeFixture)
			writeInput(t, p.client, "Kostis\n")
			waitAdmission(t, p)
			if got := receiveHandoff(t, records); got.name != "Kostis" {
				t.Fatalf("expected duplicate name Kostis, got %q", got.name)
			}
		}
		requireCapacity(t, srv, 2)
	})
	t.Run("discard unfinished name at EOF", func(t *testing.T) {
		srv, records := recordingServer(t, nil)
		p := startAdmission(t, srv, false, nil)
		readExact(t, p.reader, welcomeFixture)
		writeInput(t, p.client, "unfinished")
		p.client.Close()
		waitAdmission(t, p)
		requireNoHandoff(t, records)
		requireCapacity(t, srv, 0)
	})
}

func TestHandleConnectionCapacity(t *testing.T) {
	t.Run("ten pending names and TCP slot reuse", func(t *testing.T) {
		srv, records := recordingServer(t, nil)
		var peers []*admissionPeer
		for i := 0; i < 10; i++ {
			p := startAdmission(t, srv, true, nil)
			readExact(t, p.reader, welcomeFixture)
			peers = append(peers, p)
		}
		requireCapacity(t, srv, 10)
		for cycle := 0; cycle < 3; cycle++ {
			extra := startAdmission(t, srv, true, nil)
			readExact(t, extra.reader, "Chat is full\n")
			if _, err := extra.reader.ReadByte(); err != io.EOF {
				t.Fatalf("full server should close excess connection, got %v", err)
			}
			waitAdmission(t, extra)
			requireCapacity(t, srv, 10)
			peers[cycle].client.Close()
			waitAdmission(t, peers[cycle])
			requireCapacity(t, srv, 9)
			replacement := startAdmission(t, srv, true, nil)
			readExact(t, replacement.reader, welcomeFixture)
			peers[cycle] = replacement
			requireCapacity(t, srv, 10)
		}
		// All ten remaining clients still work after rejection and replacement.
		for i, p := range peers {
			writeInput(t, p.client, fmt.Sprintf("client%d\n", i))
			waitAdmission(t, p)
			receiveHandoff(t, records)
		}
		requireCapacity(t, srv, 10)
	})
	t.Run("simultaneous admission", func(t *testing.T) {
		srv, records := recordingServer(t, nil)
		start := make(chan struct{})
		var once sync.Once
		open := func() { once.Do(func() { close(start) }) }
		defer open() // Also unblock handlers if setup fails.
		var peers []*admissionPeer
		for i := 0; i < 20; i++ {
			peers = append(peers, startAdmission(t, srv, false, start))
		}
		open()
		accepted, rejected := 0, 0
		for _, p := range peers {
			line, err := p.reader.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			switch line {
			case "Welcome to TCP-Chat!\n":
				accepted++
				readExact(t, p.reader, strings.TrimPrefix(welcomeFixture, line))
			case "Chat is full\n":
				rejected++
				if _, err := p.reader.ReadByte(); err != io.EOF {
					t.Fatalf("rejected connection remained open: %v", err)
				}
				waitAdmission(t, p)
			default:
				t.Fatalf("unexpected admission response %q", line)
			}
		}
		if accepted != 10 || rejected != 10 {
			t.Fatalf("expected 10 accepted and 10 rejected; got %d and %d", accepted, rejected)
		}
		requireCapacity(t, srv, 10)
		requireNoHandoff(t, records)
		for _, p := range peers {
			p.client.Close()
			waitAdmission(t, p)
		}
		requireCapacity(t, srv, 0)
	})
}

func TestHandleConnectionWelcome(t *testing.T) {
	srv, records := recordingServer(t, nil)
	p := startAdmission(t, srv, false, nil)
	readExact(t, p.reader, welcomeFixture)
	requireCapacity(t, srv, 1)
	requireNoHandoff(t, records)
	p.client.Close()
	waitAdmission(t, p)
	requireCapacity(t, srv, 0)
}

// Fail on a selected output call to exercise each admission write-error branch.
type failingAdmissionConn struct {
	net.Conn
	failAt int
	writes int
}

func (c *failingAdmissionConn) Write(p []byte) (int, error) {
	c.writes++
	if c.writes == c.failAt {
		return 0, errors.New("test: failed admission write")
	}
	return c.Conn.Write(p)
}

func TestHandleConnectionWriteFailureCleanup(t *testing.T) {
	for _, tc := range []struct {
		label     string
		failAt    int
		input     string
		errorLine string
	}{
		{"greeting", 1, "", ""},
		{"logo and prompt", 2, "", ""},
		{"empty name error", 3, "\n", ""},
		{"empty name reprompt", 4, "\n", "Invalid name. Please enter a non-empty name.\n"},
		{"oversized name error", 3, strings.Repeat("a", 65) + "\n", ""},
		{"oversized name reprompt", 4, strings.Repeat("a", 65) + "\n", "Name too long. Maximum is 64 bytes.\n"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			srv, records := recordingServer(t, nil)
			serverConn, clientConn := net.Pipe()
			conn := &failingAdmissionConn{Conn: serverConn, failAt: tc.failAt}
			p := &admissionPeer{conn, clientConn, bufio.NewReader(clientConn), make(chan struct{})}
			t.Cleanup(func() {
				serverConn.Close()
				clientConn.Close()
				waitAdmission(t, p)
			})
			for _, endpoint := range []net.Conn{serverConn, clientConn} {
				if err := endpoint.SetDeadline(time.Now().Add(admissionTimeout)); err != nil {
					t.Fatal(err)
				}
			}
			go func() {
				defer close(p.done)
				srv.HandleConnection(conn)
			}()
			if tc.failAt == 2 {
				readExact(t, p.reader, "Welcome to TCP-Chat!\n")
			}
			if tc.input != "" {
				readExact(t, p.reader, welcomeFixture)
				writeInput(t, clientConn, tc.input)
				readExact(t, p.reader, tc.errorLine)
			}
			if _, err := p.reader.ReadByte(); err != io.EOF {
				t.Fatalf("expected EOF after write failure, got %v", err)
			}
			waitAdmission(t, p)
			requireNoHandoff(t, records)
			requireCapacity(t, srv, 0)
		})
	}
}
