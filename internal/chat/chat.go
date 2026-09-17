package chat

// ClientID identifies one registered chat member. The room assigns it on Join
// and it stays stable even if the member later changes name.
type ClientID uint64

// Destination is one member's outgoing channel, implemented by the session
// package and called by the room. Every method returns immediately; none of
// them waits for the socket write to complete.
type Destination interface {
	// Begin delivers the history batch once, before any live event.
	Begin(history []string) error

	// Enqueue delivers one complete, newline-terminated line.
	// It reports an error instead of blocking when the queue is unavailable.
	Enqueue(text string) error

	// Fail signals session cleanup asynchronously. It returns nothing so the
	// room never waits for cleanup that needs the room's own lock.
	Fail(err error)
}
