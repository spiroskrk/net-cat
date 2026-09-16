# Architecture — NetCat TCP Chat

## Ownership and independent work

| Owner | Files and responsibilities | Independent test substitutes | Reviewer |
| --- | --- | --- | --- |
| [Kostis](../tasks/kostis-tasks.md) | `main.go`, `main_test.go`, `internal/server/`: CLI, listening, reservation, welcome/name validation, handoff | Fake session starter and local TCP clients | Aris |
| [Aris](../tasks/aris-tasks.md) | `internal/chat/`: IDs, membership, timestamps, rendering, history, ordered delivery, announcements | Fake destinations and controlled clock | Spyros |
| [Spyros](../tasks/spyros-tasks.md) | `internal/session/`: room registration, line input, serialized socket output, cleanup | Fake room, fake release operation, `net.Pipe` | Kostis |

Keep root logic small. Each source file gets matching tests where appropriate. Baseline packages are server, chat and session; there is no separate protocol package. Detailed policies and exact errors are in [notes.md](notes.md).

Planned module: `net-cat`; planned toolchain: Go 1.26.2, pending evaluator verification. No implementation or module file is created by this documentation update. Bonus paths are `internal/activitylog/` (Kostis), and proposed `cmd/tcpchat-client/` and `internal/tui/` (Spyros).

## Project directory tree

Target layout for the required implementation. `AGENTS.md`, `README.md`, `tasks/`, and `docs/` already exist in the locations shown. `go.mod`, Go source files, tests, and `internal/` are planned; this tree does not create them. Source/test names illustrate the starting layout and may be split into smaller files as needed.

```txt
net-cat/
├── AGENTS.md
├── README.md
├── go.mod                       # Planned module: net-cat
├── main.go                      # Kostis: CLI and application wiring
├── main_test.go
├── tasks/
│   ├── aris-tasks.md
│   ├── kostis-tasks.md
│   └── spyros-tasks.md
├── internal/
│   ├── server/                  # Kostis: listener, admission, welcome, names
│   │   ├── server.go
│   │   └── server_test.go
│   ├── chat/                    # Aris: room, shared types, rendering, history
│   │   ├── chat.go
│   │   └── chat_test.go
│   └── session/                 # Spyros: Room interface, I/O, cleanup
│       ├── session.go
│       └── session_test.go
└── docs/
    ├── architecture.md
    ├── prd.md
    ├── workflow.md
    ├── notes.md
    ├── golden_tests.md
    └── audit_test.md
```

Proposed bonus additions, after required integration passes; confirm the terminal client's final layout before implementation:

```txt
net-cat/
├── cmd/
│   └── tcpchat-client/          # Spyros: bonus client entry point
└── internal/
    ├── activitylog/             # Kostis: activity/file logging and tests
    └── tui/                     # Spyros: gocui interface and tests
```

Aris's rename and room bonuses extend `internal/chat/`. Keep the audited server entry point at root `main.go`; the bonus client has a separate entry point. Generated binaries and logs are excluded from these source trees.

## Data flow

```txt
CLI arguments
    |
    v
main.go (Kostis): validate port, wire components
    |
    v
server (Kostis): listen -> accept -> reserve slot
    |
    v
welcome -> read/trim/validate name
    |
    | connection + name + existing reader + release function
    v
session.Start (Spyros): accept ownership -> start output worker -> Join
    |                                                        |
    | complete input lines -> Submit(client ID, message)      |
    +--------------------------+-----------------------------+
                               v
chat (Aris): membership -> timestamp/render -> history/order
                               |
                    Begin(history), then Enqueue(events)
                               |
                               v
session writer (Spyros): history first -> queued output -> client
```

The same session reads input and sends output concurrently. Room methods receive complete messages, never raw socket reads. A new member receives history, its own join notice, then later events; the sender receives its own server-formatted messages too.

Before a successful handoff, failures return to Kostis's admission cleanup. After handoff, read/write/room failures converge on Spyros's cleanup: close the connection, finish workers, leave registered membership, and release the slot once. See the error table below for each boundary.

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

## Error ownership and response

| Failure or condition | Owner and response | Effect on other clients |
| --- | --- | --- |
| Extra CLI arguments | Kostis: usage only on stderr, exit 1; no listener | No new server starts |
| Invalid port | Kostis: invalid-port explanation plus usage on stderr, exit 1 | No new server starts |
| Bind failure | Kostis: startup error on stderr, exit 1; no successful-listen line | An already-running server is unaffected |
| Accept failure after startup | Kostis: handle/report listener failure; any retry must be deliberate and bounded; fatal shutdown must clean up owned resources | Do not leave reservations or workers behind |
| All ten slots occupied | Kostis: send `Chat is full\n`, close excess connection | Existing clients continue |
| Empty/whitespace-only or oversized name | Kostis: exact name error, drain oversized line when needed, prompt again | Reservation remains with that connection |
| Disconnect before handoff or rejected Start | Kostis: close and release once; no room departure notice | Slot becomes reusable |
| Oversized chat line | Spyros: discard incrementally, send exact size error through serialized output, continue | No broadcast/history entry |
| Empty/whitespace-only chat | Aris suppresses it; Spyros may filter it too | No broadcast/history entry |
| Join fails before registration completes | Aris rolls back provisional membership; Spyros closes/releases after handoff | No false departure or ghost member |
| Submit uses unknown ID | Aris returns an error; session handles failure through its lifecycle path | Room remains usable |
| Repeated Leave | Aris returns success without a second notice | Membership changes once |
| Socket read/write failure, timeout, or full live queue | Spyros cleans up once; Aris reports failed destinations through Fail without blocking | Only the affected client disconnects |

Exact client/CLI error strings are in [notes.md](notes.md#exact-error-text). Unexpected I/O diagnostics must not become broadcast/history entries. Avoid duplicate error reporting and avoid holding room/capacity locks while waiting for network I/O or cleanup.

## Bonuses

Aris owns rename and independent rooms: preserve IDs and historical authors, isolate messages/history/notices, and keep the server-wide ten-slot limit when switching rooms. Kostis owns agreed flags and activity/file logging through a shared event sink. Spyros owns the gocui client: separate display/input areas, preserve unfinished text and cursor, show the server echo once, restore terminal state on exit, and keep plain nc compatible.

Agree on bonus interfaces before implementing those features. All three integrate and run the [golden checks](golden_tests.md) and [supplied audit](audit_test.md); independent unit tests do not replace integration or LAN testing.
