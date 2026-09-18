package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net-cat/internal/session"
	"testing"
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
