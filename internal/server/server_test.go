package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net-cat/internal/session"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func rejectSession(conn net.Conn, name string, reader *bufio.Reader, release func(), room session.Room) error {

	return errors.New("test: rejected handoff")
}

func TestHandleConnectionRejectedHandoff(t *testing.T) {
	srv := NewServer(rejectSession, nil)
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	deadline := time.Now().Add(2 * time.Second)
	err := clientConn.SetDeadline(deadline)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		srv.HandleConnection(serverConn)
		close(done)
	}()

	reader := bufio.NewReader(clientConn)
	_, err = reader.ReadString(':')

	if err != nil {
		t.Fatal(err)
	}
	_, err = reader.ReadByte()

	if err != nil {
		t.Fatal(err)
	}
	_, err = fmt.Fprintln(clientConn, "Kostis")

	if err != nil {
		t.Fatal(err)
	}
	_, err = reader.ReadByte()

	if err != io.EOF {
		t.Fatalf("expected EOF after rejected handoff, got %v", err)
	}

	select {
	case <-done:

	case <-time.After(2 * time.Second):
		t.Fatal("handler did not finish after rejected handoff")
	}

	if srv.activeConnections != 0 {
		t.Fatalf("expected 0 active connections, got %d", srv.activeConnections)
	}
}

func TestHandleConnectionDisconnectBeforeName(t *testing.T) {
	srv := NewServer(rejectSession, nil)
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	deadline := time.Now().Add(2 * time.Second)
	err := clientConn.SetDeadline(deadline)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		srv.HandleConnection(serverConn)
		close(done)
	}()

	reader := bufio.NewReader(clientConn)
	_, err = reader.ReadString(':')

	if err != nil {
		t.Fatal(err)
	}
	_, err = reader.ReadByte()

	if err != nil {
		t.Fatal(err)
	}

	err = clientConn.Close()

	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:

	case <-time.After(2 * time.Second):
		t.Fatal("handler did not finish after disconnect before name")
	}

	if srv.activeConnections != 0 {
		t.Fatalf("expected 0 active connections, got %d", srv.activeConnections)
	}
}

func TestHandleConnectionSuccessfulHandoff(t *testing.T) {
	var sessionRelease func()
	starter := func(conn net.Conn, name string, reader *bufio.Reader, release func(), room session.Room) error {
		sessionRelease = release
		return nil
	}
	srv := NewServer(starter, nil)

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()

	deadline := time.Now().Add(2 * time.Second)
	err := clientConn.SetDeadline(deadline)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		srv.HandleConnection(serverConn)
		close(done)
	}()

	reader := bufio.NewReader(clientConn)

	_, err = reader.ReadString(':')

	if err != nil {
		t.Fatal(err)
	}
	_, err = reader.ReadByte()

	if err != nil {
		t.Fatal(err)
	}
	_, err = fmt.Fprintln(clientConn, "Kostis")

	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:

	case <-time.After(2 * time.Second):
		t.Fatal("handler did not finish after successful handoff")
	}

	if srv.activeConnections != 1 {
		t.Fatalf("expected 1 active connections, got %d", srv.activeConnections)
	}

	if sessionRelease == nil {
		t.Fatalf("starter did not provide a release callback")
	}

	sessionRelease()
	if srv.activeConnections != 0 {
		t.Fatalf("expected 0 active connections after first release, got %d", srv.activeConnections)
	}
	sessionRelease()
	if srv.activeConnections != 0 {
		t.Fatalf("expected 0 active connections after repeated release, got %d", srv.activeConnections)
	}
}

func TestReadName(t *testing.T) {
	a62 := strings.Repeat("a", 62)
	a63 := strings.Repeat("a", 63)
	a64 := strings.Repeat("a", 64)
	unicodeSpaces := "\u0085\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000"
	longSpaces := strings.Repeat(" \t\u00a0\u3000", 2048)
	malformed := "\xff\x80\xc0\xaf\xed\xa0\x80\xf4\x90\x80\x80\xe2\x80"

	for _, tc := range []struct {
		label       string
		input       string
		want        string
		wantTooLong bool
	}{
		{"empty LF", "\n", "", false},
		{"empty CRLF", "\r\n", "", false},
		{"ASCII whitespace only", " \t\v\f\r\n", "", false},
		{"Unicode whitespace only", unicodeSpaces + "\n", "", false},
		{"plain name", "Kostis\n", "Kostis", false},
		{"CRLF", "Kostis\r\n", "Kostis", false},
		{"ASCII trimming", "\t \vKostis\f\r \n", "Kostis", false},
		{"Unicode trimming", unicodeSpaces + "Κώστας" + unicodeSpaces + "\n", "Κώστας", false},
		{"internal whitespace", " A \t\r\u00a0\u3000B \n", "A \t\r\u00a0\u3000B", false},
		{"non-whitespace controls", " \x00\u180e\u200b\ufeff \n", "\x00\u180e\u200b\ufeff", false},
		{"63 ASCII bytes", a63 + "\n", a63, false},
		{"64 ASCII bytes", a64 + "\n", a64, false},
		{"64 bytes with CRLF", a64 + "\r\n", a64, false},
		{"65 ASCII bytes", a64 + "a\n", "", true},
		{"two-byte rune fits", a62 + "α\n", a62 + "α", false},
		{"two-byte rune crosses cap", a63 + "α\n", "", true},
		{"32 Greek letters", strings.Repeat("α", 32) + "\n", strings.Repeat("α", 32), false},
		{"33 Greek letters", strings.Repeat("α", 33) + "\n", "", true},
		{"replacement rune fits", strings.Repeat("a", 61) + "\ufffd\n", strings.Repeat("a", 61) + "\ufffd", false},
		{"replacement rune crosses cap", a62 + "\ufffd\n", "", true},
		{"four-byte rune fits", strings.Repeat("a", 60) + "😀\n", strings.Repeat("a", 60) + "😀", false},
		{"four-byte rune crosses cap", strings.Repeat("a", 61) + "😀\n", "", true},
		{"two-byte trailing space crosses cap", a63 + "\u00a0\n", a63, false},
		{"three-byte trailing space crosses cap", a62 + "\u3000\n", a62, false},
		{"two-byte internal space crosses cap", a63 + "\u00a0B\n", "", true},
		{"three-byte internal space crosses cap", a62 + "\u3000B\n", "", true},
		{"internal space fits exactly", strings.Repeat("a", 61) + "\u00a0B\n", strings.Repeat("a", 61) + "\u00a0B", false},
		{"long whitespace only", longSpaces + "\n", "", false},
		{"long surrounding whitespace", longSpaces + a64 + longSpaces + "\n", a64, false},
		{"long whitespace becomes internal", a64 + longSpaces + "B\n", "", true},
		{"single invalid byte", "\xff\n", "\xff", false},
		{"invalid byte with surrounding whitespace", "\u3000\xff\u00a0\n", "\xff", false},
		{"malformed UTF-8 sequences", malformed + "\n", malformed, false},
		{"incomplete UTF-8 before LF", "\xc2\n", "\xc2", false},
		{"mixed valid and invalid UTF-8", "α\xff\ufffd\x80B\n", "α\xff\ufffd\x80B", false},
		{"invalid byte fits", a63 + "\xff\n", a63 + "\xff", false},
		{"invalid byte crosses cap", a64 + "\xff\n", "", true},
		{"64 invalid bytes", strings.Repeat("\xff", 64) + "\n", strings.Repeat("\xff", 64), false},
		{"65 invalid bytes", strings.Repeat("\xff", 65) + "\n", "", true},
	} {
		t.Run(tc.label, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(tc.input))
			name, tooLong, err := readName(reader)
			if err != nil {
				t.Fatalf("readName: %v", err)
			}
			if tooLong != tc.wantTooLong {
				t.Fatalf("tooLong = %v, want %v", tooLong, tc.wantTooLong)
			}
			// A rejected prefix is not an accepted name; its exact contents are unspecified.
			if !tc.wantTooLong && name != tc.want {
				t.Fatalf("name = %q, want %q", name, tc.want)
			}
			if len(name) > 64 {
				t.Fatalf("returned name contains %d bytes, maximum is 64", len(name))
			}
		})
	}
}

func TestReadNameFragmentedInput(t *testing.T) {
	for _, tc := range []struct {
		label string
		input string
		want  string
	}{
		{"Unicode and CRLF", "\u3000Κώστας 😀\u00a0\r\n", "Κώστας 😀"},
		{"rune across buffer boundary", strings.Repeat("a", 15) + "α😀\n", strings.Repeat("a", 15) + "α😀"},
		{"malformed and replacement rune", "\xff\ufffd\x80α\xe2\x80\n", "\xff\ufffd\x80α\xe2\x80"},
		{"trailing whitespace across cap", strings.Repeat("a", 63) + "\u3000\n", strings.Repeat("a", 63)},
	} {
		for _, transport := range []struct {
			label string
			wrap  func(io.Reader) io.Reader
		}{
			{"small buffer", func(r io.Reader) io.Reader { return r }},
			{"one byte reads", iotest.OneByteReader},
			{"half reads", iotest.HalfReader},
		} {
			t.Run(tc.label+"/"+transport.label, func(t *testing.T) {
				const next = "first message\n"
				reader := bufio.NewReaderSize(transport.wrap(strings.NewReader(tc.input+next)), 16)
				name, tooLong, err := readName(reader)
				if err != nil || tooLong || name != tc.want {
					t.Fatalf("readName = (%q, %v, %v), want (%q, false, nil)", name, tooLong, err, tc.want)
				}
				rest, err := io.ReadAll(reader)
				if err != nil || string(rest) != next {
					t.Fatalf("remaining input = (%q, %v), want (%q, nil)", rest, err, next)
				}
			})
		}
	}
}

func TestReadNameDiscardsUnfinishedInput(t *testing.T) {
	readErr := errors.New("test: name input failed")
	for _, terminalErr := range []error{io.EOF, readErr} {
		for _, tc := range []struct {
			label string
			input string
		}{
			{"empty", ""},
			{"partial name", "Kostis"},
			{"CR without LF", "Kostis\r"},
			{"whitespace", " \u3000\t"},
			{"64 bytes", strings.Repeat("a", 64)},
			{"oversized", strings.Repeat("a", 8192)},
			{"invalid bytes", "\xff\x80"},
			{"incomplete UTF-8", "Kostis\xe2\x80"},
		} {
			t.Run(terminalErr.Error()+"/"+tc.label, func(t *testing.T) {
				reader := bufio.NewReaderSize(&nameTerminalErrorReader{data: tc.input, err: terminalErr}, 16)
				name, tooLong, err := readName(reader)
				if name != "" || tooLong || !errors.Is(err, terminalErr) {
					t.Fatalf("readName = (%q, %v, %v), want (empty, false, %v)", name, tooLong, err, terminalErr)
				}
			})
		}
	}
}

func TestReadNameCompleteLineBeforeReadError(t *testing.T) {
	readErr := errors.New("test: error after complete name")
	for _, terminalErr := range []error{io.EOF, readErr} {
		t.Run(terminalErr.Error(), func(t *testing.T) {
			// A Reader may return data and an error together. The complete line still counts.
			reader := bufio.NewReader(&nameTerminalErrorReader{data: "α\xff\r\n", err: terminalErr})
			name, tooLong, err := readName(reader)
			if name != "α\xff" || tooLong || err != nil {
				t.Fatalf("readName = (%q, %v, %v), want (α\\xff, false, nil)", name, tooLong, err)
			}
			name, tooLong, err = readName(reader)
			if name != "" || tooLong || !errors.Is(err, terminalErr) {
				t.Fatalf("next readName = (%q, %v, %v), want (empty, false, %v)", name, tooLong, err, terminalErr)
			}
		})
	}
}

func TestReadNameDrainsOversizedLineAndPreservesFollowingInput(t *testing.T) {
	for _, tc := range []struct {
		label string
		first string
	}{
		{"prefetched together", strings.Repeat("a", 65) + "\n"},
		{"across reader buffers", strings.Repeat("α\xff", 8192) + "\r\n"},
		{"long internal whitespace", strings.Repeat("a", 64) + strings.Repeat("\u3000", 8192) + "B\n"},
	} {
		t.Run(tc.label, func(t *testing.T) {
			const retry = "\u3000Retry α\t\r\n"
			const chat = "first message\nsecond message\r\n"
			reader := bufio.NewReader(strings.NewReader(tc.first + retry + chat))
			name, tooLong, err := readName(reader)
			if err != nil || !tooLong || len(name) > 64 {
				t.Fatalf("oversized name = (%q, %v, %v), want capped name, true, nil", name, tooLong, err)
			}
			name, tooLong, err = readName(reader)
			if err != nil || tooLong || name != "Retry α" {
				t.Fatalf("retry = (%q, %v, %v), want (Retry α, false, nil)", name, tooLong, err)
			}
			rest, err := io.ReadAll(reader)
			if err != nil || string(rest) != chat {
				t.Fatalf("remaining input = (%q, %v), want (%q, nil)", rest, err, chat)
			}
		})
	}
}

func TestReadNameLongStreams(t *testing.T) {
	for _, kind := range []string{"ASCII name", "malformed name", "surrounding whitespace"} {
		t.Run(kind, func(t *testing.T) {
			const repetitions = 1 << 20
			input, want, wantTooLong := longNameInput(kind, repetitions)
			reader := bufio.NewReader(input)
			name, tooLong, err := readName(reader)
			if err != nil || tooLong != wantTooLong || len(name) > 64 {
				t.Fatalf("readName = (%q, %v, %v), want <=64 bytes, %v, nil", name, tooLong, err, wantTooLong)
			}
			if !wantTooLong && name != want {
				t.Fatalf("name = %q, want %q", name, want)
			}
			rest, err := io.ReadAll(reader)
			if err != nil || string(rest) != "next line\n" {
				t.Fatalf("remaining input = (%q, %v), want (next line\\n, nil)", rest, err)
			}
		})
	}
}

func TestIsNameSpace(t *testing.T) {
	for _, ch := range "\t\n\v\f\r \u0085\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000" {
		if !isNameSpace(ch) {
			t.Errorf("isNameSpace(%U) = false, want true", ch)
		}
	}
	for _, ch := range []rune{'A', 'α', '😀', '\x00', '\u180e', '\u200b', '\ufeff', '\ufffd'} {
		if isNameSpace(ch) {
			t.Errorf("isNameSpace(%U) = true, want false", ch)
		}
	}
}

// Allocation measurements complement the storage-cap source review: a short returned
// prefix alone cannot reveal an oversized temporary allocation inside the parser.
func BenchmarkReadNameLongInput(b *testing.B) {
	for _, kind := range []string{"ASCII name", "malformed name", "surrounding whitespace"} {
		for _, repetitions := range []int{64, 1 << 16, 1 << 20} {
			b.Run(fmt.Sprintf("%s/%d", kind, repetitions), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					input, want, wantTooLong := longNameInput(kind, repetitions)
					name, tooLong, err := readName(bufio.NewReader(input))
					if err != nil || tooLong != wantTooLong || len(name) > 64 || (!wantTooLong && name != want) {
						b.Fatalf("unexpected readName result: (%q, %v, %v)", name, tooLong, err)
					}
				}
			})
		}
	}
}

// Supply the final bytes together with the terminal error, as io.Reader permits.
type nameTerminalErrorReader struct {
	data string
	err  error
}

func (r *nameTerminalErrorReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	if len(r.data) == 0 {
		return n, r.err
	}
	return n, nil
}

// Generate long inputs without allocating a string proportional to the input size.
type repeatingNameReader struct {
	pattern   string
	remaining int
	offset    int
}

func (r *repeatingNameReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if n > r.remaining {
		n = r.remaining
	}
	for i := 0; i < n; i++ {
		p[i] = r.pattern[r.offset]
		r.offset = (r.offset + 1) % len(r.pattern)
	}
	r.remaining -= n
	return n, nil
}

func longNameInput(kind string, repetitions int) (io.Reader, string, bool) {
	repeated := func(pattern string) io.Reader {
		return &repeatingNameReader{pattern: pattern, remaining: len(pattern) * repetitions}
	}
	switch kind {
	case "ASCII name":
		return io.MultiReader(repeated("a"), strings.NewReader("\nnext line\n")), strings.Repeat("a", 64), repetitions > 64
	case "malformed name":
		return io.MultiReader(repeated("\xff"), strings.NewReader("\nnext line\n")), strings.Repeat("\xff", 64), repetitions > 64
	case "surrounding whitespace":
		return io.MultiReader(repeated("\u3000"), strings.NewReader("Kostis"), repeated("\u00a0"), strings.NewReader("\nnext line\n")), "Kostis", false
	default:
		panic("unknown long-name fixture")
	}
}
