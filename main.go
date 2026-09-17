package main

import "errors"

func parsePort(p []string) (int, error) {
	if len(p) == 0 {
		return 8989, nil
	}
	if len(p) > 1 {
		return 0, errors.New("[USAGE]: ./TCPChat $port")
	}
	if p[0] == "" {
		return 0, errors.New("Invalid port. Please use a port number between 1 and 65535.\n[USAGE]: ./TCPChat $port")
	}
	port := 0
	for _, d := range p[0] {
		if d < '0' || d > '9' {
			return 0, errors.New("Invalid port. Please use a port number between 1 and 65535.\n[USAGE]: ./TCPChat $port")
		}
		port = port * 10
		dig := int(d - '0')
		port += dig
		if port > 65535 {
			return 0, errors.New("Invalid port. Please use a port number between 1 and 65535.\n[USAGE]: ./TCPChat $port")
		}
	}
	if port == 0 {
		return 0, errors.New("Invalid port. Please use a port number between 1 and 65535.\n[USAGE]: ./TCPChat $port")
	}
	return port, nil
}
