# PRD — net-cat (TCPChat)

## 1. Project Overview

Build a Go TCP group chat server that several terminal clients can use at the same time. The project combines a command-line entry point, network server, line-based input parsing, shared chat state, and concurrent client sessions.

The sources of project requirements are the **TASK SUBJECT** in [zone01-doc-agent-prompt.md](../zone01-doc-agent-prompt.md) and the [user-supplied audit checklist](audit_test.md). This is a documentation scaffold, not an implemented or tested program. **Required** statements come from those sources; **proposed** choices are planning recommendations. The audit clarifies that messages reach the sender as well as peers and that clients on different computers must connect. Unspecified behavior is recorded under [Open Questions](notes.md#open-questions).

At the user's request, the original six generated documents are delivered together in this `docs/` folder, including [AGENTS.md](AGENTS.md). The subsequently supplied checklist is also preserved here as [audit_test.md](audit_test.md). The target project structure below describes their eventual placement in a Go project; it does not describe Go files already created.

## 2. Project Goal

Let up to ten connected clients join with a nonempty name, exchange messages identified by timestamp and author, and receive earlier chat messages when joining. Every named participant, including the sender, receives the same accepted message. Support clients on different computers. A client leaving must not interrupt the remaining clients.

The supplied client commands use the existing `nc` program. The introduction also mentions client mode, but specifies no custom client command or protocol beyond these examples; resolve that ambiguity before planning a separate client executable.

## 3. Non-Goals

- Implementing Go code during documentation generation.
- Recreating every NetCat feature, UDP support, or UNIX-domain sockets for the required TCP chat.
- Adding optional bonuses before the required chat behavior works.
- Inventing an official audit tool, error text, persistence requirement, or additional package permission.

Optional features are name changes with announcements, multiple independent chat groups, more NetCat flags, a terminal UI, producing client activity logs, and saving those logs to a file. The audit assesses producing logs and saving them separately. The `gocui` package exception applies only to the terminal UI bonus, whose audit allows just that UI package. These features need a separate plan after the required implementation passes review.

The audit also lists good practices and the presence of a test file as bonus checks. Good practices remain required by the subject, and matching meaningful tests remain part of this project's development plan. The [golden tests](golden_tests.md) map all nine bonus checks without weakening the baseline requirements.

## 4. User Story

As a terminal user, I can connect to a running chat server with `nc`, enter a name, read earlier messages, and talk to other users. I can leave without disconnecting anyone else.

As the server operator, I can start on the default port or supply one port, limit connections, and handle network errors without crashing the whole chat.

## 5. Project Structure Requirements

Proposed eventual layout; create implementation files progressively during learning:

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

The delivered `docs/AGENTS.md` is intended for the eventual project root. Keep this bundle together until choosing that project location. No Go files, module, README, or `.gitignore` are created by this documentation task.

## 6. Root Folder Rules

- Keep only essential project entry and configuration files in the root.
- `main.go` selects command-line configuration, wires packages together, starts the server, and reports startup failures.
- `main_test.go` tests root-level argument handling and coordination where appropriate.
- `README.md` should stay short and practical: purpose, prerequisites, run/test commands, input/output examples, and links to the detailed docs.
- `go.mod` records the chosen module path and supported Go version.
- `AGENTS.md` contains the mentoring and collaboration rules.
- Keep package logic in `internal/` and detailed documentation in `docs/`.
- Do not place extra Go logic files in the root besides `main.go` and matching root tests.

## 7. Internal Folder Rules

These four packages are a proposed responsibility split, not a subject-mandated API.

| Package | Responsibility and expected files | Must not do |
| --- | --- | --- |
| `internal/server` | `server.go`, `server_test.go`: listen, accept, reserve/release capacity, coordinate session lifetime | Parse chat lines, construct chat text, or own message history |
| `internal/session` | `session.go`, `session_test.go`: welcome/name exchange, read complete lines, deliver serialized output, clean up one connection | Own global membership/history or duplicate protocol formatting |
| `internal/chat` | `chat.go`, `chat_test.go`: named membership, ordered messages, history replay, join/leave events, recipients | Read sockets, parse CLI arguments, or block the room on socket writes |
| `internal/protocol` | `protocol.go`, `protocol_test.go`: pure rendering of startup/usage text, banner, prompts, chat messages, and notices | Open sockets, print directly, or mutate room state |

Use small functions and explicit inputs/outputs. Add source/test file pairs only when a responsibility needs splitting. Keep dependencies one-way as explained in [architecture.md](architecture.md).

## 8. Docs Folder Rules

| Document | Purpose |
| --- | --- |
| [prd.md](prd.md) | Scope, requirements, acceptance criteria, and milestones |
| [architecture.md](architecture.md) | Package boundaries, ownership, data flow, and error flow |
| [workflow.md](workflow.md) | Small development steps and checkpoints |
| [notes.md](notes.md) | Practical concepts, decisions, open questions, and review reminders |
| [golden_tests.md](golden_tests.md) | Expected behavior, exact known text, and manual/package test cases |
| [audit_test.md](audit_test.md) | Supplied functional and bonus audit checklist, with stable check IDs |

Keep requirements and decisions consistent across these documents. Record resolved questions in `notes.md` before making their behavior an acceptance test. `audit_test.md` is the additional source reference supplied after the original six-file scaffold; keep it in the eventual project's `docs/` folder too.

## 9. File Responsibilities

The root coordinates, `server` manages admitted sockets, `session` handles one client's conversation, `chat` manages the shared room, and `protocol` owns exact visible text. Corresponding tests verify those boundaries. Documentation describes the intended behavior; it must not claim implementation milestones or test results that have not happened.

Generated executables and local test artifacts are not source files. Recommend a future `.gitignore` containing `/TCPChat`, `/bin/`, and `/tmp/` if those output paths are used. Add a specific log path only if the optional file-logging bonus is implemented. Do not create `.gitignore` as part of this task or add unrelated statistical data/audit patterns.

## 10. Input Requirements

Run these commands from the eventual Go project root after implementation:

The audit runs the compiled executable. Build it once, then run each server invocation separately, stopping the previous server when needed:

```bash
go build -o TCPChat .
```

```bash
./TCPChat
```

```bash
./TCPChat 2525
```

```bash
./TCPChat 2525 localhost
```

These are the audit's default-port, explicit-port, and usage checks. The subject also demonstrates the corresponding development commands below.

```bash
go run .
```

No positional argument selects port `8989`.

```bash
go run . 2525
```

One positional port starts the server on that port if it can bind successfully.

```bash
go run . 2525 localhost
```

Additional positional arguments must produce the subject's usage message. The valid-port examples resolve the unclear word “Otherwise” in the prose. Invalid single-port syntax and its exact error contract remain open questions.

Clients connect from separate terminals:

```bash
nc localhost 2525
```

For audit F10, repeat the client connection on two or three different computers using the server computer's reachable IP and selected port. `localhost` addresses the client computer itself and is only suitable for the same-machine examples. The listener must accept connections through a reachable non-loopback interface; a loopback-only server does not satisfy this audit. The exact interface/address-family choice is still a design decision. [workflow.md](workflow.md) provides the multi-computer procedure.

The first input is a name. A client cannot participate without a nonempty name. Later input consists of chat messages; empty messages must not be broadcast. Line-based input follows the terminal examples. Trimming, CRLF handling, partial lines at EOF, and size limits need explicit decisions. Client input supplies the name and text, not trusted timestamp or author prefixes.

## 11. Output Requirements

Known startup lines, for the corresponding successful commands:

```txt
Listening on the port :8989
```

```txt
Listening on the port :2525
```

Known invalid-argument usage text; `$port` is literal:

```txt
[USAGE]: ./TCPChat $port
```

On an admitted connection, show `Welcome to TCP-Chat!`, the exact supplied penguin, and `[ENTER YOUR NAME]:`. The complete banner fixture is in [golden_tests.md](golden_tests.md). Name acceptance must precede participation and history delivery.

Chat message example from the subject:

```txt
[2020-01-20 16:03:43][Yenlik]:hello
```

The timestamp has second precision; there is no extra bracket around `hello`. Preserve the author, content, and original timestamp when replaying history. Current runs use their actual times, not the dates in the subject.

Join/leave text illustrated by the subject:

```txt
Lee has joined our chat...
Lee has left our chat...
```

For join notices, this plan takes audit F06's “all Clients” literally: notify all named participants, including the newcomer, after its earlier history is delivered. This inclusive interpretation also satisfies the subject's requirement to inform existing clients. Departure notices go to the remaining participants.

Audit F09 explicitly requires the same chat message on all three clients when the second sends. Broadcast each accepted message to the sender too, with identical timestamp, author, and content; local terminal echo is not evidence of server delivery. Empty-input prompts visible in the example transcript are not empty messages to broadcast. Prompt refresh, exact prompt suffix, output stream, and error status are not fully specified. Keep diagnostics separate from client chat data; never add debug text to required successful output.

## 12. Functional Requirements

| ID | Required behavior | Verification focus |
| --- | --- | --- |
| FR-01 | Listen and accept TCP client connections | Default and explicit-port startup; real TCP connection |
| FR-02 | Support concurrent clients using goroutines and channels or mutexes | Simultaneous sessions and safe shared state |
| FR-03 | Enforce a maximum of ten connections | Tenth succeeds; eleventh cannot join; capacity released on departure |
| FR-04 | Display the supplied welcome/logo and require a nonempty name | Banner, naming gate, empty name, EOF during naming |
| FR-05 | Deliver a client's nonempty messages to all named participants, including the sender | Audit F09: client two sends; all three receive the same server message once |
| FR-06 | Do not broadcast empty messages | Empty line produces no chat event or history message |
| FR-07 | Identify each chat message by timestamp and author | Fixed formatter fixture; structural live-time check |
| FR-08 | Give a newcomer all previous chat messages | Ordered replay with original metadata and a complete transition to live messages |
| FR-09 | Inform all named clients of joins and remaining clients of departures | F06 inclusive join interpretation; F12 departure notification; one notice per transition |
| FR-10 | Keep other clients connected when one leaves | Remaining users continue exchanging messages |
| FR-11 | Apply the demonstrated command-line port/usage behavior | Zero, one, and extra arguments |
| FR-12 | Handle server and client-side network errors | Bind/accept/read/write failures; safe cleanup; no expected-input panic |
| FR-13 | Support clients connecting from two or three different computers | F10: connect using the server's reachable IP; loopback-only success is insufficient |

Proposed safeguards: reserve capacity for unnamed as well as named sockets; serialize room events; give each connection one writer; preserve a single replay/live ordering boundary; remove failed sessions once. These are design choices supporting the required behavior, not additional supplied protocol rules.

## 13. Technical Requirements

- **Language:** Go is required. The subject gives no version; select one supported by the evaluation environment and record it in `go.mod`.
- **Allowed implementation packages:** `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, `reflect`.
- Use no unnecessary external libraries. Do not silently import unlisted packages such as `strconv`, `context`, `regexp`, or `os/exec`.
- The subject recommends unit tests but omits `testing` from the list. Confirm the evaluator's test-only package policy; do not infer permission for arbitrary test helpers.
- Use goroutines and channels or mutexes. Explain who owns each shared value and how admission, publishing, replay, and removal are synchronized.
- A slow or failing connection must not cause unrelated clients to disconnect. Choose and document a slow-writer policy; the subject gives no timeout or queue-size values.
- Keep protocol rendering pure and network I/O out of room-state operations.
- Run `go fmt ./...` and `go test ./...` when implementation files exist. Use the race detector for exercised concurrent scenarios where supported.
- Match each Go source file with a test in the same package where appropriate, and cover the cases in [golden_tests.md](golden_tests.md).

## 14. Code Comments Rule

Use short, useful comments for exported functions/types, shared package APIs, ownership of room state, important parsing decisions, replay ordering, cleanup behavior, and audit-sensitive formatting. Explain why a rule exists or what responsibility an API has. Avoid comments that repeat obvious assignments or every line of a loop. Update comments when the behavior changes.

There are no numerical formulas to document. Explain concurrency and ordering decisions with small examples instead.

## 15. Edge Cases

- No argument (valid default), extra arguments, malformed single port, occupied/unavailable port.
- Ten simultaneous admissions, an eleventh connection, and a new connection after a slot is released.
- Empty name; whitespace-only or duplicate names as unresolved policy cases.
- EOF before naming, immediately after joining, or during a write.
- Empty messages, whitespace-only messages, CRLF, fragmented input, several lines in one read, and an unterminated last line.
- An oversized line, stalled writer, or client that never submits a name; limits and timeouts are undecided.
- Concurrent publication and joining, repeated disconnect signals, and message history with no entries.
- Messages containing colons/brackets or non-ASCII text; do not interpret content as protocol metadata.
- Two messages with the same displayed second; ordering must not depend on unique timestamps.
- Failed listener/read/write operations and cleanup that could otherwise leak slots or goroutines.

## 16. Testing Plan

1. **Protocol:** compare the exact known strings and penguin; format messages with a fixed supplied time.
2. **Input/session:** validate naming, complete-line framing, empty-message suppression, and read/write failure cleanup.
3. **Room:** verify all recipients including the sender, identical message metadata, history, ordered replay/live transition, inclusive join notices, departures, and concurrent operations.
4. **Server:** verify listener behavior, atomic admission at ten, refusal of excess clients, and capacity recovery.
5. **Root CLI:** test default/explicit/extra-argument coordination without embedding network or room logic in `main.go`.
6. **Manual integration:** build `TCPChat` and run the audit's binary commands; use separate `nc` terminals for history, sender-inclusive delivery, empty input, disconnect isolation, and capacity. Repeat on two or three different computers. Exercise four clients with one disconnect and three clients with a departure notice separately.
7. **Concurrency review:** run exercised scenarios with the race detector where available; use bounded test deadlines and observable events instead of arbitrary sleeps.

These are planned checks, not reported passing tests. See [workflow.md](workflow.md) for commands and [golden_tests.md](golden_tests.md) for expected behavior and test ownership.

## 17. Official Audit or Comparator Guidance

The user supplied a manual functional and bonus [audit checklist](audit_test.md). Its functional checks F01–F18 and bonus checks B01–B09 are mapped to procedures and evidence in [golden_tests.md](golden_tests.md). Run the checklist after implementation; no audit checks have been executed during documentation work.

No automated comparator, audit executable, or downloadable tool was provided. `nc` is the demonstrated client. Do not invent a checker or download unrelated assets.

For the final verdict (F18), record observed evidence and any applicable supplied failure category: **Empty Work**, **Incomplete Work**, **Invalid compilation**, **Cheating**, **Crashing**, or **Leaks**. Assess compilation, completed requirements, crashes, and resource cleanup through actual checks. Integrity judgments require evidence; do not infer cheating from style or unfamiliar code. Mark checks not performed as **Not run** and record unavailable multi-computer testing as a limitation, not a pass.

## 18. Acceptance Criteria

- [ ] Required TCP behavior FR-01 through FR-13 and functional audit checks F01–F18 have recorded evidence.
- [ ] `go build -o TCPChat .` succeeds and the audit's three binary startup/usage cases behave correctly.
- [ ] All three clients, including client two as sender, receive the same server-authored timestamp/name/body record.
- [ ] Two or three different computers connect using the server's reachable IP.
- [ ] The tenth connection works, the eleventh cannot participate, and capacity is recovered safely.
- [ ] Names gate participation; empty messages are absent from delivery and history.
- [ ] History and live delivery preserve the agreed order with no replay gap or duplication.
- [ ] Required strings match, timestamp fields are correct, and successful output has no debug text.
- [ ] Expected invalid inputs and network failures do not panic or terminate unrelated sessions.
- [ ] Package responsibilities and root structure follow the documented plan.
- [ ] Unit and integration tests pass under the resolved test-import policy; supported race checks show no detected races in tested paths.
- [ ] Behavior-affecting open questions have been resolved or explicitly accepted for evaluation.
- [ ] The practical README explains how to run and test and links to the PRD and golden tests.
- [ ] No generated executables, temporary data, logs, downloaded archives, or audit artifacts are submitted.

## 19. Implementation Milestones

| Milestone | Small, reviewable outcome |
| --- | --- |
| 1 — Understand the subject | Review all docs and resolve the first implementation-blocking questions |
| 2 — Prepare the project | Choose module/version and establish the root/package plan |
| 3 — CLI and visible text | Test argument counts and pure protocol rendering |
| 4 — TCP admission | Listen, accept, and test the connection limit |
| 5 — One named session | Welcome, name input, line parsing, and cleanup |
| 6 — Shared room | Join/leave events and nonempty message delivery |
| 7 — History | Preserve metadata and verify replay during live publication |
| 8 — Integration and errors | Wire packages, handle failures, exercise concurrent clients |
| 9 — Review and audit preparation | Run all checks, explain the design, and prepare a clean submission |

Each milestone is implemented in small mentoring steps; [workflow.md](workflow.md) gives checkpoints. Bonus work starts only after the required project is complete.

## 20. Important Notes

The subject's transcript mixes local terminal echo, server prompts, and received messages, and the two clients' example timestamps are not mutually consistent. Do not treat the complete transcript as a byte-for-byte network fixture. Sender-inclusive delivery is established by audit F09; verify the server sends that record rather than relying on typed text appearing locally. Use exact isolated text plus behavioral tests.

Do not silently choose policies for unspecified inputs. Review [Open Questions](notes.md#open-questions) before implementing the affected behavior. The documentation itself is complete even while genuine subject ambiguities remain visible.

## 21. Definition of Done

The implemented project is done when its acceptance criteria pass, the user can explain the network flow and ownership of state, and the final repository contains readable code, appropriate matching tests, practical run/test instructions, and clean documentation. This scaffold does not claim that implementation or verification has occurred.
