package main

import "testing"

func TestParsePortDefault(t *testing.T) {
	port, err := parsePort([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 8989 {
		t.Fatalf("got port %d, want 8989", port)
	}
}

func TestParsePortExplicit(t *testing.T) {
	port, err := parsePort([]string{"2525"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 2525 {
		t.Fatalf("got port %d, want 2525", port)
	}
}

func TestParsePortTooManyArguments(t *testing.T) {
	port, err := parsePort([]string{"2525", "localhost"})
	if err == nil {
		t.Fatalf("expected a usage error, got nil")
	}
	if port != 0 {
		t.Fatalf("got port %d, want 0", port)
	}
	if err.Error() != "[USAGE]: ./TCPChat $port" {
		t.Fatalf("got error %q, want %q", err.Error(), "[USAGE]: ./TCPChat $port")
	}
}

func TestParsePortInvalid(t *testing.T) {
	inputs := []string{"", "25x5", "0", "65536"}
	wantError := "Invalid port. Please use a port number between 1 and 65535.\n[USAGE]: ./TCPChat $port"

	for _, v := range inputs {
		port, err := parsePort([]string{v})

		if err == nil {
			t.Fatalf("input %q: expected an invalid-port error, got nil", v)
		}

		if port != 0 {
			t.Fatalf("input %q: got port %d, want 0", v, port)
		}

		if err.Error() != wantError {
			t.Fatalf("input %q: got error %q, want %q", v, err.Error(), wantError)
		}
	}
}
