# Architecture — NetCat TCP Chat

## 1. Architecture Overview

This is a Go TCP server with a small command-line entry point and multiple concurrent client sessions. The supplied usage connects clients with `nc`. Implementing a separate custom client is an Open Question because the introduction mentions client mode but gives no corresponding CLI contract.

The supplied manual audit, preserved in [audit_test.md](audit_test.md), also requires building a `TCPChat` executable, connecting from two or three different computers, and delivering a client's message to all three connected named clients, including its sender. These are requirements, not optional design choices. No automated comparator was provided.

The subject requires goroutines and channels or mutexes. The package layout and concurrency model below are recommended design choices, not additional subject requirements. Start with one connection and add each responsibility in the order described in [workflow.md](workflow.md).

Keep the root small so command-line behavior can be understood independently of sockets and shared chat state. Put implementation packages in `internal/` so their responsibilities and tests remain clear. There are no mathematical formulas in this task.

## 2. Target Structure

The following is the intended structure of a future implemented project; these Go files are not created by this documentation task.

```txt
net-cat/
├── AGENTS.md
├── main.go
├── main_test.go
├── README.md
├── go.mod
├── internal/
│   ├── server/
│   │   ├── server.go
│   │   └── server_test.go
│   ├── session/
│   │   ├── session.go
│   │   └── session_test.go
│   ├── chat/
│   │   ├── chat.go
│   │   └── chat_test.go
│   └── protocol/
│       ├── protocol.go
│       └── protocol_test.go
└── docs/
    ├── prd.md
    ├── architecture.md
    ├── workflow.md
    ├── notes.md
    ├── golden_tests.md
    └── audit_test.md
```

For this delivery, the user requested the documentation inside one `docs/` folder, including [AGENTS.md](AGENTS.md). The supplied audit checklist is added there as [audit_test.md](audit_test.md). If this bundle is later installed into a Go project, place the mentoring file at the project root as shown above. The other documents stay in `docs/`.

`README.md` should eventually explain the project, run and test commands, expected input and output, and link to the PRD and golden tests. It should stay short. Do not create it, `.gitignore`, or implementation files as part of this documentation-only task.

## 3. Root Package

`main.go` owns basic argument handling and application wiring:

- No port argument selects `8989`.
- One port argument selects that port after validation under the documented port policy.
- Extra arguments produce the exact usage text `[USAGE]: ./TCPChat $port`.
- Start the server and print the listening line only after listening succeeds: for example, `Listening on the port :2525`.
- Report startup or fatal listener errors safely, without a panic or a misleading successful-start message.

The eventual build command is `go build -o TCPChat .`. The audit invokes `./TCPChat`, `./TCPChat 2525`, and `./TCPChat 2525 localhost`; these must follow the same argument contract as the subject's `go run .` examples. The executable is a generated local artifact and should not be committed.

`main_test.go` tests argument selection, usage formatting, and startup coordination where appropriate. Make these behaviors callable without launching an external process or terminating the test process. Do not assume permission to import `os/exec` for CLI tests; it is absent from the supplied allowlist.

`go.mod` declares the project module and chosen supported Go version. The subject does not fix a Go version or module path. Do not choose dependencies merely to avoid learning a small piece of standard-library logic.

## 4. Internal Packages

| Package | Expected source and test | Owns | Must not do |
| --- | --- | --- | --- |
| `internal/server` | `server.go`, `server_test.go` | Listener lifecycle on a reachable interface, accepting sockets, connection admission, active-socket accounting, wiring session and chat dependencies | Parse chat lines, format messages, own message history |
| `internal/session` | `session.go`, `session_test.go` | Per-client line reading, name handshake, input validation, session lifecycle, serialized client writes | Decide global broadcast order, directly mutate room membership/history, parse CLI arguments |
| `internal/chat` | `chat.go`, `chat_test.go` | Named membership, accepted messages, authoritative timestamps and order, history, join/leave events, recipient selection | Accept sockets, read client input, perform blocking network writes, print logs or prompts |
| `internal/protocol` | `protocol.go`, `protocol_test.go` | Pure startup/usage text, welcome, name prompt, message, membership event, and chat prompt formatting | Read or write sockets, change state, decide membership, obtain the current time independently |

Every Go source file should have a matching `_test.go` file in the same package where appropriate. Add further files only when a clear responsibility justifies them, with corresponding tests; do not grow the root with extra Go logic files.

## 5. Package Responsibilities

### Input and validation

The root validates the small CLI contract. Session code interprets a TCP byte stream as complete text lines, first for a name and then for chat messages. A read from a socket is not a message boundary: a line can span reads, and one read can contain several lines.

Sessions remove line delimiters according to the agreed LF/CRLF policy. Reject an empty name and suppress an empty chat message. Whether whitespace-only input is empty and whether ordinary surrounding spaces are preserved remain explicit decisions. Do not silently trim every message into a different message.

The server enforces at most ten connections. Recommended policy: count both sockets awaiting names and named sessions. Reserve a slot before starting the handshake, guard admission accounting with a mutex, and release the slot exactly once on every termination path. A pending client that disconnects should not create a named-user leave announcement.

The listener must accept TCP connections addressed to the server host from other computers. A listener bound only to `localhost` or a loopback address cannot satisfy the supplied multi-computer audit. Choose a reachable non-loopback interface or suitable wildcard binding; the exact address and address family remain deployment decisions. Local tests establish basic behavior, while separate computers must verify actual network reachability.

### Shared chat logic

Recommended model: one room-owner goroutine handles join, message, and leave requests through channels. It alone changes the member set and ordered history. Give a message its timestamp once when the server accepts it into the room order; store that timestamp, sender name, and text with the message. This is a design choice for an `nc` text protocol that supplies no sender timestamp.

For every accepted nonempty chat message, deliver the same message record to all currently connected named clients, including the sender. With clients 1, 2, and 3 present, client 2's message must reach all three over their server connections with identical timestamp, name, and text. The terminal displaying locally typed input is not evidence of server delivery to the sender.

For join notices, this plan interprets the audit's F06 wording "all Clients" literally: notify every named participant, including the newcomer after its history replay. This includes the existing clients required by the original subject. A leave notice goes only to the remaining clients.

Store messages in that accepted order. Replay preserves original metadata; it does not recompute timestamps. Do not silently truncate history, since the subject requires all previous chat messages for a new client. The subject does not require history to survive a server restart.

### Formatting and client output

`protocol` owns the exact startup/usage text, welcome banner and Linux logo, `[ENTER YOUR NAME]:`, timestamped message shape, and membership notices. The root decides when to print startup/usage text; the formatter only returns text. The subject's message example is `[2020-01-20 15:48:41][client.name]:[client.message]`; its transcripts show actual text immediately after the colon, such as `[2020-01-20 16:03:43][Yenlik]:hello`. Use the exact fixtures and interpretation in [golden_tests.md](golden_tests.md).

Each session has one writer goroutine that serializes welcome output, replay, live messages, and any agreed prompt refresh. This protects complete application output from interleaving. Concurrent `net.Conn` method calls are supported by Go; the single writer is an application ordering choice. [Go net.Conn documentation](https://pkg.go.dev/net#Conn)

Pass message fields into pure formatting functions so `protocol` does not depend on `chat`. `server` wires dependencies; `session` uses the room interface and formatter. Keep dependencies in one direction and avoid package import cycles.

## 6. Data Flow

```txt
CLI arguments
    |
    v
main.go: select port and start application
    |
    v
server: listen -> accept -> reserve connection slot
    |
    v
session: welcome -> read and validate name -> read message lines
    |                                             |
    | join / leave requests                       | message requests
    +----------------------+----------------------+
                           v
                    chat room owner
               membership + ordered history
                           |
          ordered deliveries, including to sender
                           v
                session writer -> protocol formatting
                           |
                           v
                   client TCP connection
```

### Joining without losing messages

1. The server admits the socket; the session sends the welcome and obtains a non-empty name.
2. The session asks the room owner to join. The room owner establishes a single position for that join among message events.
3. As one ordered transition, make an immutable snapshot of previous message records, place its replay delivery before any live delivery for that client, and make the client eligible for subsequent room messages. A replay delivery may carry the snapshot as one item; a bounded queue must not need one slot for every historical message.
4. Queue the join notice to every named participant, including the newcomer. Its writer finishes replay before writing its own join notice and subsequent live deliveries. Newcomer inclusion is this plan's literal interpretation of audit F06; existing-member notification also satisfies the subject.

A concurrent message therefore appears either in the snapshot or after it as a live delivery, exactly once. Do not register the client after an uncoordinated history copy, which could miss messages. Do not perform socket writes while holding the admission mutex or while the room owner processes shared state.

### Leaving and slow clients

EOF, a read error, or a write error ends that client's session through one cleanup path. Close its socket, remove named membership once, notify remaining clients once if it had joined, stop its writer, and release its admission slot once. Other clients remain connected.

Recommended design: bounded output queues and a documented write deadline or equivalent slow-client policy keep an unresponsive reader from blocking the room. Queue capacity, timeout values, and the exact disconnect rule are Open Questions. A full queue must trigger explicit handling; silently dropping chat messages violates the delivery requirement. Verify that closing a session wakes any blocked read/write and that shutdown does not send on a closed channel.

## 7. Error Flow

| Failure | Owner and response | Effect on the rest of the program |
| --- | --- | --- |
| Wrong argument count | Root returns the supplied usage line | Do not start listening |
| Invalid port text or unavailable bind address | Root/server reports validation or listen failure | No successful listening announcement; exact invalid-port stream/status needs a decision |
| Eleventh connection | Server refuses admission and closes the excess socket safely | Existing ten connections continue; rejection wording is unspecified |
| Empty name | Session does not admit named membership | Re-prompt versus close is a decision; release any slot if the socket closes |
| Empty chat message | Session ignores it | No broadcast and no history entry |
| Client read/write failure or disconnect | Session reports the failure to lifecycle cleanup; room removes membership if present | Remaining clients stay connected and are informed of a named user's departure |
| Slow client exceeds the chosen output policy | Session/server cleans up that client and records a useful diagnostic | Room must remain able to serve other clients |
| Listener failure | Server returns a fatal error to `main.go`, or applies a documented retry policy for recoverable cases | Do not retry indefinitely without a reason or leak existing sessions |

Return errors at package boundaries and add context where it identifies the failed operation. Use a single reporting point for each error to avoid duplicate log noise. Per-client errors are handled at the session/server boundary; they should not terminate the entire server. Fatal startup/listener errors travel to the root. Keep server diagnostics separate from exact client protocol output and successful CLI output. The subject does not fix diagnostic wording or every output stream.

## 8. Testing Boundaries

| Test file | Evidence it should provide |
| --- | --- |
| `main_test.go` | Default/custom port, extra-argument usage, agreed invalid-port policy, no false success on startup failure |
| `internal/server/server_test.go` | Real listener accepts established clients, ten-connection cap including pending names, slot reuse, occupied-port error, four-client departure leaves three usable connections; supplement with a manual multi-computer check |
| `internal/session/session_test.go` | Name handshake, fragmented/coalesced line input, empty input, CRLF decision, EOF/read/write cleanup, serialized writes, slow-client behavior |
| `internal/chat/chat_test.go` | Three-client broadcast includes sender with identical metadata/text, original history order/metadata, concurrent join with live traffic, single join/leave events, three-client departure notifies both remaining clients, no empty history entries |
| `internal/protocol/protocol_test.go` | Exact logo/banner, name prompt, deterministic timestamp/name/message formatting, membership notices, chosen prompt rules |

Keep pure formatting tests free of networking. Supply fixed times to message-format tests and a controllable clock to room tests where needed. Use local TCP listeners and `net.Pipe` where appropriate; socket deadlines and explicit readiness signals should bound tests without relying on arbitrary sleeps. Close all test connections and stop spawned goroutines during cleanup.

The subject recommends tests but lists no `testing` package. Treat permission for the standard `testing` package in `_test.go` files as an Open Question, and do not infer permission for additional test helpers or external dependencies. The audit places a test-file check and good-practice check in its bonus section; good practices remain a baseline subject requirement and progressive tests remain part of this development plan.

Use the supplied functional audit F01–F18 and bonus checks B01–B09 in [audit_test.md](audit_test.md), together with the mapping in [golden_tests.md](golden_tests.md). The audit includes the verdict reasons Empty Work, Incomplete Work, Invalid compilation, Cheating, Crashing, and Leaks. Plan evidence for build success, all required behavior, student understanding, failure handling, and resource cleanup. Retained chat history is required state; distinguish it from leaked sockets, listeners, abandoned queues, or goroutines after a session closes. No audit or runtime result has been verified by generating these documents.

## 9. Design Rules

- Use Go and only the supplied production package allowlist: `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, `reflect`.
- Do not assume that familiar alternatives such as `strconv`, `context`, `regexp`, or `os/exec` are allowed. Confirm exceptions before adding imports.
- Keep state ownership explicit: server admission, room membership/history, and session I/O each have one clear owner.
- Keep parsing in sessions, shared chat decisions in `chat`, and exact output formatting in `protocol`.
- Keep network writes and printing out of pure parsing/formatting and shared-state logic.
- Use small functions, clear names, and comments that explain package APIs, ordering guarantees, cleanup ownership, and formatting decisions. Do not comment obvious lines.
- Add one concurrency responsibility at a time, test it, and explain it before continuing. Avoid advanced abstractions until they solve an observed problem.
- Do not silently add history retention limits, custom commands, a client executable, multiple rooms, file logging, or a terminal UI. Bonuses follow a complete baseline.
- Recommend a future `.gitignore` for actual generated artifacts such as `/TCPChat`, `bin/`, `tmp/`, and `*.test`; add log/output patterns only if those artifacts are introduced. Do not create it during this documentation task.

### Open Questions

Resolve affected behavior in the PRD, golden tests, and implementation together:

- Does the introductory client-mode description require a custom client beyond the supplied `nc` usage?
- Which invalid single-port arguments show usage; what port range, stdout/stderr choice, and exit status are expected? Which non-loopback or wildcard bind address and address family should the server use to meet the required multi-computer access?
- Does whitespace-only input count as empty? Are surrounding spaces retained? Are duplicate names allowed? What happens after an empty name?
- What timestamp timezone and timestamp capture point are expected?
- Exactly when and how are chat prompts refreshed? Sender delivery is required by the audit; only prompt presentation remains unresolved. Distinguish terminal input echo from server output.
- Does replay include join/leave notices, or only chat messages? The mandatory requirement explicitly names previous messages, while event retention is unspecified.
- What text, if any, should an excess client receive? What line-length limit and slow-client queue/deadline policy are permitted?
- Are incomplete final lines at EOF accepted, and how should CRLF input be normalized?
- Is the standard `testing` package permitted in tests despite its omission from the allowlist? What Go version should be used?

Do not invent expected audit behavior for these points. Clear requirements can be developed while the affected decisions remain documented.

## 10. What Should Not Go Inside `main.go`

Do not put the accept loop, admission mutex, client goroutines, line reader, name handshake, message-history storage, broadcast loop, replay coordination, membership cleanup, Linux logo literal, message formatter, or socket-write policy in `main.go`.

The root should show how the application starts and how fatal errors are reported. A reader should be able to follow it without understanding the internal chat algorithm.
