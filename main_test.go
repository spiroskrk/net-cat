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
