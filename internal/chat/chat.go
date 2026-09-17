package chat

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

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

// timeLayout is the layout required by the subject's message format.
const timeLayout = "2006-01-02 15:04:05"

const (
	joinEvent  = "has joined our chat..."
	leaveEvent = "has left our chat..."
)

// formatMessage renders one accepted chat message, newline included. The
// session writes the result unchanged, so no caller adds a second newline.
func formatMessage(t time.Time, name, body string) string {
	return fmt.Sprintf("[%s][%s]:%s\n", t.Format(timeLayout), name, body)
}

// formatNotice renders a join or leave announcement, newline included.
// Notices never enter history.
func formatNotice(name, event string) string {
	return fmt.Sprintf("%s %s\n", name, event)
}

// member is one registered participant. The room keeps it until Leave.
type member struct {
	id   ClientID
	name string
	out  Destination
}

// Room holds chat membership and history for one group chat. All exported
// methods are safe for concurrent use; mu guards every field below it.
type Room struct {
	mu      sync.Mutex
	nextID  ClientID
	members map[ClientID]*member
	history []string
	now     func() time.Time
}

// NewRoom returns an empty room. Production passes time.Now; tests pass a
// controlled clock so formatted timestamps are predictable.
func NewRoom(now func() time.Time) *Room {
	return &Room{
		nextID:  1,
		members: make(map[ClientID]*member),
		now:     now,
	}
}

// Join registers a member, replays the history it must not miss, then announces
// the arrival to everyone including the newcomer. Registration and the history
// snapshot happen under one lock: split apart, a message published in between
// would either reach the newcomer twice or not at all.
func (r *Room) Join(name string, output Destination) (ClientID, error) {
	if output == nil {
		return 0, errors.New("chat: join needs a destination")
	}

	r.mu.Lock()
	id := r.nextID
	r.nextID++
	r.members[id] = &member{id: id, name: name, out: output}

	// Copy the history: the caller must never hold a slice the room mutates.
	history := append([]string(nil), r.history...)

	if err := output.Begin(history); err != nil {
		// Registration never completed, so roll back silently.
		// A member who never joined must not produce a departure notice.
		delete(r.members, id)
		r.mu.Unlock()
		return 0, err
	}

	failed := r.broadcastLocked(formatNotice(name, joinEvent))
	r.mu.Unlock()

	reportFailures(failed)
	return id, nil
}

// Submit accepts one complete message from a registered member, records it in
// history with a single acceptance timestamp, and delivers it to everyone
// including the sender.
func (r *Room) Submit(id ClientID, message string) error {
	if strings.TrimSpace(message) == "" {
		return nil // whitespace-only input never reaches the chat or history
	}

	r.mu.Lock()

	m, ok := r.members[id]
	if !ok {
		r.mu.Unlock()
		return errors.New("chat: unknown client ID")
	}

	line := formatMessage(r.now(), m.name, message)
	r.history = append(r.history, line)
	failed := r.broadcastLocked(line)

	r.mu.Unlock()

	reportFailures(failed)
	return nil
}

// Leave removes a member and tells the others once. Calling it again for the
// same ID succeeds silently: the session may reach cleanup from two paths.
func (r *Room) Leave(id ClientID) error {
	r.mu.Lock()

	m, ok := r.members[id]
	if !ok {
		r.mu.Unlock()
		return nil
	}

	name := m.name
	delete(r.members, id)
	failed := r.broadcastLocked(formatNotice(name, leaveEvent))

	r.mu.Unlock()

	reportFailures(failed)
	return nil
}

// failure pairs a broken destination with the error it reported, so Fail can be
// called after the room lock is released.
type failure struct {
	dest Destination
	err  error
}

// broadcastLocked hands one formatted line to every member and returns the
// destinations that refused it. It runs under mu so that concurrent senders
// cannot interleave: every member observes the same room order. This is only
// safe because Enqueue is contractually non-blocking.
func (r *Room) broadcastLocked(line string) []failure {
	var failed []failure
	for _, m := range r.members {
		if err := m.out.Enqueue(line); err != nil {
			failed = append(failed, failure{dest: m.out, err: err})
		}
	}
	return failed
}

// reportFailures signals cleanup for broken destinations. It runs with no lock
// held, because cleanup will call back into Leave.
func reportFailures(failed []failure) {
	for _, f := range failed {
		f.dest.Fail(f.err)
	}
}
