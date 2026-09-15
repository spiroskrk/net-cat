# Architecture — NetCat TCP Chat

## Ownership and independent work

| Owner | Files and responsibilities | Independent test substitutes | Reviewer |
| --- | --- | --- | --- |
| [Kostis](../kostis-tasks.md) | `main.go`, `main_test.go`, `internal/server/`: CLI, listening, reservation, welcome/name validation, handoff | Fake session starter and local TCP clients | Aris |
| [Aris](../aris-tasks.md) | `internal/chat/`: IDs, membership, timestamps, rendering, history, ordered delivery, announcements | Fake destinations and controlled clock | Spyros |
| [Spyros](../spyros-tasks.md) | `internal/session/`: room registration, line input, serialized socket output, cleanup | Fake room, fake release operation, `net.Pipe` | Kostis |

Keep root logic small. Each source file gets matching tests where appropriate. Baseline packages are server, chat and session; there is no separate protocol package. Detailed policies and exact errors are in [notes.md](notes.md).

Planned module: `net-cat`; planned toolchain: Go 1.26.2, pending evaluator verification. No implementation or module file is created by this documentation update. Bonus paths are `internal/activitylog/` (Kostis), and proposed `cmd/tcpchat-client/` and `internal/tui/` (Spyros).

## Shared Go API contract

The following snippets describe declarations to prepare together before independent implementation; they are not complete source files.

Aris owns these shared types in `internal/chat`:

```go
type ClientID uint64

type Destination interface {
    Begin(history []string) error
    Enqueue(text string) error
    Fail(err error)
}
```

Spyros owns this interface and entry point in `internal/session`. It imports `net-cat/internal/chat` for shared types; Aris's package must not import session.

```go
type Room interface {
    Join(name string, output chat.Destination) (chat.ClientID, error)
    Submit(id chat.ClientID, message string) error
    Leave(id chat.ClientID) error
}

func Start(conn net.Conn, name string, reader *bufio.Reader,
    release func(), room Room) error
```

Aris's concrete room implements those three methods. Its internal representation and construction can be chosen within this contract; room tests need no socket or session implementation.

Kostis tests admission with an injected starter matching Start's signature. Application wiring supplies the real starter and room at integration time. The planned shared skeleton contains agreed types/interfaces and the injectable starter boundary, not completed implementations. Prepare it together once, then work independently against fakes.

## Handoff semantics

Kostis accepts and reserves a connection, finishes welcome/name output, then calls Start with the accepted trimmed name, original buffered reader, connection, idempotent release function and room.

- Start returns nil when Spyros takes ownership. It does not wait for the chat session to end or for room registration to finish.
- On a returned error, ownership has not transferred: Kostis closes and releases. The rejected start must not leave active workers using those resources.
- After acceptance, Spyros handles registration and all subsequent failures. Start the output worker before calling Join.
- The release function can be called repeatedly but releases one reservation only once. No two components may write concurrently across handoff.

Admission never registers room membership. No valid-name disconnect before registration should invent a departure announcement.

## Room and destination semantics

Join accepts the name and destination, assigns an ID, and establishes one history/live ordering boundary. It returns that ID on successful registration, or an error with provisional membership rolled back. Duplicate display names have distinct IDs. Submit accepts complete message content, suppresses whitespace-only input, creates one timestamped record and sends it to all registered members including the sender. Unknown IDs return an error. Leave removes membership once; repeated Leave returns nil.

- Begin accepts the initial immutable history batch once, including an empty batch. It returns without waiting for socket writes.
- Enqueue accepts one complete formatted event, including its final newline; it returns an error when unavailable or full rather than waiting indefinitely.
- Fail signals Spyros's cleanup asynchronously; it must not wait for cleanup or synchronously call back into Leave while room state is held.
- Spyros implements Destination. Aris formats newline-terminated messages/notices; the session writes them unchanged and does not add a second newline.
- The history batch is written before the newcomer's own join event and subsequent queued events. Begin does not consume one live-queue slot per historical message.

A destination failure is reported through Fail. Failure before registration completes rolls back provisional membership without a departure notice. After successful registration, normal cleanup emits one departure notice. Ensure cleanup racing with Join's return cannot lose the returned ID and leave a ghost member. Join success means registration/queue acceptance, not proof that the client has already received all bytes.

## Delivery and failure flow

One writer serializes history, events and session-local error lines. The latter never become room messages or history. Ten-second deadlines apply separately to each outgoing message. The live-event queue holds 256 events; queue overflow or write failure ends only that session. A quiet client is not timed out for lack of input.

Spyros closes the socket to unblock pending I/O, terminates workers, removes registered membership, and releases capacity once. Aris never closes sockets, releases admission capacity, or performs blocking network writes under room state. Capacity accounting must also avoid holding its lock during I/O.

Test failures before handoff, during Join, during replay, and after successful registration independently. Use explicit completion signals, not arbitrary sleeps, to prove workers terminate.

## Bonuses

Aris owns rename and independent rooms: preserve IDs and historical authors, isolate messages/history/notices, and keep the server-wide ten-slot limit when switching rooms. Kostis owns agreed flags and activity/file logging through a shared event sink. Spyros owns the gocui client: separate display/input areas, preserve unfinished text and cursor, show the server echo once, restore terminal state on exit, and keep plain nc compatible.

Agree on bonus interfaces before implementing those features. All three integrate and run the [golden checks](golden_tests.md) and [supplied audit](audit_test.md); independent unit tests do not replace integration or LAN testing.
