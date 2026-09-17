package session

import (
	"bufio"
	"net"
	"net-cat/internal/chat"
)

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
