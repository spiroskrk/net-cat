# Workflow — NetCat TCP Chat

## 1. Workflow Overview

Build the Go TCP chat server in small, reviewed steps. This file plans future implementation; this documentation task creates no Go code, README, or `.gitignore`, and none of the future commands below are claimed to have run.

Read [prd.md](prd.md), [architecture.md](architecture.md), [notes.md](notes.md), [golden_tests.md](golden_tests.md), the supplied manual audit in [audit_test.md](audit_test.md), and the mentoring rules in [AGENTS.md](AGENTS.md) before implementation. This delivery keeps all documentation together as requested. In a future Go project, install the mentoring file at the root and keep the other documents in `docs/`.

The baseline is the supplied TCP server used with `nc`, at most ten connections, required names, messages, history, and join/leave notifications. The supplied audit also requires a `TCPChat` executable, message delivery to all three connected named clients including the sender, and connections from two or three different computers. Defer optional features until baseline checks pass. The introductory custom-client question remains open because the supplied invocations specify only the server and `nc` clients.

Follow the mentor cycle for each step: understand the idea, attempt a small implementation, run the relevant check, explain the result, and review before proceeding. Resolve only the Open Questions that affect the next piece of behavior; continue independent work on requirements already clear.

## 2. Step-by-Step Development

### Step 1 — Confirm the baseline contract

**Files:** `docs/prd.md`, `docs/notes.md`, `docs/golden_tests.md`, `docs/audit_test.md`.

Read the subject and supplied audit, distinguishing fixed output from illustrative timestamps and terminal echo. Record that the sender must receive server delivery and that loopback-only listening cannot pass the multi-computer audit. For join notices, adopt the plan's literal interpretation of F06: all named participants receive the notice, including the newcomer after replay. Record unresolved port, whitespace, duplicate-name, timestamp, prompt, history-event, line-limit, exact reachable bind-address, custom-client, and excess-connection policies. Confirm the Go version and whether `testing` is allowed in tests.

**Checkpoint:** Explain the difference between a TCP connection, a named chat member, and a stored message. Do not invent an audit rule for an unresolved case.

### Step 2 — Prepare the eventual Go project

**Files:** `go.mod`, `main.go`, `main_test.go`.

When implementation is authorized, establish the chosen module and a minimal entry point with its test file. Keep root Go logic limited to `main.go` and matching tests. Recommend `.gitignore` entries for artifacts actually expected, such as `/TCPChat`, `bin/`, `tmp/`, and `*.test`; create that file only when requested.

**Checkpoint:** The chosen Go version is documented and the root is easy to read. There is no chat implementation hidden in setup.

### Step 3 — Handle command-line arguments

**Files:** `main.go`, `main_test.go`, `internal/protocol/protocol.go`, `internal/protocol/protocol_test.go`.

Implement only port selection and argument validation: default `8989`, one supplied port, and the exact extra-argument usage `[USAGE]: ./TCPChat $port`. Put pure usage/startup text rendering in `protocol`; the root selects when to print it. Resolve invalid single-port handling before asserting its exact stream or exit status. Stay within the allowed packages; do not assume `strconv` is available.

**Checkpoint:** Argument tests pass without opening a socket. The student can explain which cases are specified by examples and which were agreed as policy.

### Step 4 — Create the welcome formatter

**Files:** `internal/protocol/protocol.go`, `internal/protocol/protocol_test.go`.

Add pure formatting for the supplied welcome line, Linux logo, and name prompt. Keep the exact banner in one place. Do not implement socket writes in this package.

**Checkpoint:** An exact fixture comparison catches missing backslashes, changed spaces, extra debug text, and unintended newlines. Use [golden_tests.md](golden_tests.md) as the fixture source.

### Step 5 — Start and close a listener

**Files:** `internal/server/server.go`, `internal/server/server_test.go`, `main.go`, `main_test.go`.

Introduce listener startup and its error return. Bind to a reachable non-loopback interface or suitable wildcard address so other computers can connect to the server host. Wire only this behavior into the root. Print `Listening on the port :8989` or the selected port only after a successful bind.

**Checkpoint:** A local test connection succeeds; an occupied port produces a handled startup error and no false listening message. Test cleanup closes the listener. Record the intended reachable bind configuration, then verify it from other computers in the manual audit stage.

### Step 6 — Add connection admission

**Files:** `internal/server/server.go`, `internal/server/server_test.go`.

Add the ten-connection admission boundary and safe accounting before adding chat behavior. Recommended policy counts sockets awaiting names as well as named clients. Keep slot reservation and release together under a clear mutex ownership rule.

**Checkpoint:** Ten sockets can be admitted, an eleventh cannot join, and disconnecting an admitted socket permits a replacement. Closing an unnamed socket releases its slot once.

### Step 7 — Read a name and manage one session

**Files:** `internal/session/session.go`, `internal/session/session_test.go`, `internal/server/server.go`.

Create session lifecycle, its single writer, welcome delivery, and a buffered line reader. Require a non-empty name before named membership. Handle name-stage EOF and read/write errors. Decide whitespace and empty-name response policy before implementing their exact behavior.

**Checkpoint:** A client sees the correct greeting, can submit a valid name, and cannot join with an empty name. Test a name split across socket writes and clean termination before naming.

### Step 8 — Parse chat message lines

**Files:** `internal/session/session.go`, `internal/session/session_test.go`.

Extend the reader from the name phase to message lines. Suppress empty messages. Decide CRLF normalization, whitespace preservation, final incomplete-line behavior, and maximum-line handling without silently adding an arbitrary task requirement.

**Checkpoint:** Two lines arriving together remain two messages; a fragmented line remains one message. Empty messages produce no room request. Read errors trigger bounded cleanup.

### Step 9 — Format timestamped messages and membership events

**Files:** `internal/protocol/protocol.go`, `internal/protocol/protocol_test.go`.

Add pure formatters for message records and the demonstrated join/leave notices. Use supplied timestamp values in tests. Every recipient, including the sender, must receive the same accepted message. Resolve prompt-refresh behavior before adding prompt-specific assertions.

**Checkpoint:** The fixed-time message `[2020-01-20 16:03:43][Yenlik]:hello` and notices `Lee has joined our chat...` and `Lee has left our chat...` match their fixtures. No formatter reads a socket or chooses its own current time.

### Step 10 — Introduce named room membership

**Files:** `internal/chat/chat.go`, `internal/chat/chat_test.go`.

Add room-owned named membership and ordered join/leave requests. Recommended design uses one room goroutine and channels. Use an opaque session identity so lifecycle cleanup does not depend on a name being globally unique unless that policy is explicitly chosen.

**Checkpoint:** All named participants, including a new member, get one join event; this is the plan's literal interpretation of audit F06. A leave event goes to all remaining members. A failed unnamed session produces no named leave event. Tests can exercise room logic without network I/O. Once replay is added, the newcomer's own join notice must follow its history.

### Step 11 — Broadcast accepted messages

**Files:** `internal/chat/chat.go`, `internal/chat/chat_test.go`, `internal/session/session.go`, `internal/session/session_test.go`.

Connect parsed messages to the room, assign one server-side timestamp per accepted message, and deliver the same message record to all connected named clients, including the sender. Keep each client's output serialized by its writer.

**Checkpoint:** With clients 1, 2, and 3, a message from client 2 arrives once at all three via the server, with identical timestamp, sender name, and text. A socket-level test verifies delivery to client 2 rather than mistaking terminal input echo for a received message. Concurrent senders produce whole message lines in a consistent accepted order. Empty lines never broadcast.

### Step 12 — Store and replay history

**Files:** `internal/chat/chat.go`, `internal/chat/chat_test.go`, `internal/session/session.go`, `internal/session/session_test.go`.

Store accepted messages with original timestamp, sender name, and text. Add the ordered join transition described in [architecture.md](architecture.md): queue a history snapshot before later live deliveries and activate live membership in the same room-owned transition. Keep network writes outside shared-state ownership.

**Checkpoint:** A later joiner gets all prior messages in order with unchanged metadata, then its own join notice, then later live deliveries. A message racing with the join appears exactly once, either in replay or in live delivery. Empty messages are absent from history.

### Step 13 — Complete failure and disconnect handling

**Files:** `internal/session/session.go`, `internal/session/session_test.go`, `internal/server/server.go`, `internal/server/server_test.go`, `internal/chat/chat.go`, `internal/chat/chat_test.go`.

Review one cleanup path for EOF, read failure, write failure, and rejected handshakes. Add the agreed bounded-queue and slow-client policy. Make repeated cleanup harmless, prevent send-on-closed-channel failures, and ensure a blocked socket operation can be stopped.

**Checkpoint:** With four named clients, one departure leaves the remaining three connected and able to exchange messages. In a separate three-client case, one departure notifies both remaining clients. Membership, admission slots, and goroutines are released once. A replacement client can connect and read valid history; an unresponsive client cannot block the room.

### Step 14 — Run complete conversations and review

**Files:** All implementation files and matching tests; `docs/golden_tests.md`, `docs/audit_test.md`, `docs/notes.md`; eventual `README.md` when requested.

Build the `TCPChat` executable. Run the supplied functional audit F01–F18, including the manual conversation, package checks, exact binary invocations, multi-computer connections, and capacity/replay cases below. Review bonus checks B01–B09 for features actually attempted. Record resolved decisions and actual outcomes, leaving unrun checks unrun. Prepare a short README with build/run/test commands, input/output expectations, and links to the PRD and golden tests when README creation is requested.

**Checkpoint:** Every baseline acceptance criterion has evidence. Resolve failures before refactoring, then repeat only the checks affected by a refactor. Prepare to explain the architecture and tests before considering bonuses.

## 3. Checkpoints After Each Step

- Explain what changed and why that file or package owns it.
- Keep the work to the step's responsibility; list remaining uncertainty in Open Questions.
- Add or update the matching test file where appropriate and run the affected package tests.
- Check the production import allowlist: `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, `reflect`.
- Check errors, cleanup, output bytes, and useful comments. Do not add debug lines to successful server/client output.
- Review the test's assertion: it should demonstrate a required behavior or failure boundary, not merely repeat the implementation.
- Verify understanding before starting the next teaching step, as required by the mentoring guide.

## 4. Manual Testing Steps

Run commands from the future Go project root after the relevant implementation exists. Use separate terminals and an installed `nc` command. Ports `8989` and `2525` must be available for the corresponding examples.

### Build the audit executable

```bash
go build -o TCPChat .
```

Build success is evidence only after this command has actually completed successfully. Keep the generated executable out of the submitted source tree.

### Start the default server

Terminal A:

```bash
./TCPChat
```

Expected server startup line:

```txt
Listening on the port :8989
```

### Connect the first client

Terminal B:

```bash
nc localhost 8989
```

Verify the complete welcome/logo/name prompt from [golden_tests.md](golden_tests.md). Type `Yenlik` and press Enter. Send `hello`, then `How are you?`. The server must deliver these messages back to Yenlik as well as to any other named clients. Press Enter on an empty line; it must not become a chat message or history entry. Prompt presentation follows the resolved policy; locally displayed typing is not proof that the server sent a message back.

### Join, replay, and converse

Terminal C:

```bash
nc localhost 8989
```

Enter `Lee`. Yenlik receives `Lee has joined our chat...`. Lee receives the two previous messages with their original timestamp, sender, text, and order, followed by its own join notice under this plan's literal interpretation of audit F06.

Open a third client in Terminal D:

```bash
nc localhost 8989
```

Enter `Mira`. All three named clients receive the join notice; Mira receives it after prior messages. Send `Hi everyone!` from client 2, Lee. Yenlik, Lee, and Mira must each receive the same timestamped message from the server, including identical timestamp, name, and text. Send replies and verify all three connections remain usable. Pair this manual check with a socket-level test that reads client 2's received bytes so terminal echo cannot produce a false pass.

The live timestamps vary. Compare their `YYYY-MM-DD HH:MM:SS` shape and preserve their actual values when checking replay; do not expect the subject's 2020 examples during a live run.

### Disconnect and reconnect

With Yenlik, Lee, and Mira connected, press Ctrl-C in Lee's terminal. Both Yenlik and Mira receive `Lee has left our chat...` and continue chatting. This is the audit's three-client departure-notice case.

Reconnect Lee and add a fourth named client, `Sam`, using another `nc localhost 8989` session. Disconnect Sam. The remaining three clients must stay connected and exchange messages successfully. This is the separate four-client stability case. Reconnect another client and verify that the released connection slot is reusable and history is still available.

### Connect from different computers

After the local checks, stop the default server and start the custom-port server on the chosen host:

```bash
./TCPChat 2525
```

Use two or three different computers on a network that can reach that host. On each client computer, enter the actual reachable IP address of the server host when prompted:

```bash
read -r -p 'Server host IP: ' TCPCHAT_SERVER_IP
nc "$TCPCHAT_SERVER_IP" 2525
```

Each client must connect, complete the welcome/name handshake, and exchange messages with clients on the other computers. Verify message metadata, sender delivery, join notices, and continued connections. `localhost` on a remote computer identifies that computer and cannot substitute for the server host IP.

Record the server address, port, client computers, and observed results. If remote clients cannot connect, inspect the selected bind address and network reachability; use the environment's authorized network configuration. Do not mark this case passed from same-machine terminal tests, and do not assume or automatically change firewall rules.

### Exercise capacity and simultaneous activity

Use up to ten client terminals, including at least one waiting at the name prompt under the recommended capacity policy. An eleventh connection must not be admitted to the chat. The subject does not specify its rejection text. Close one accepted connection and verify the next attempt succeeds.

With several named clients, send messages close together and join another client while messages are arriving. Check complete lines, consistent order, and no missing or duplicate replay/live messages. Use automated integration tests for reliable slow-reader, fragmented-input, and simultaneous-admission checks; terminal timing alone is insufficient evidence.

## 5. Package Testing Steps

These commands apply once the packages and tests exist and the standard test-package allowance has been resolved. Run the relevant command after a small change:

```bash
go test ./internal/protocol
go test ./internal/session
go test ./internal/chat
go test ./internal/server
go test .
```

After the baseline is integrated:

```bash
go fmt ./...
go test ./...
```

Where the chosen Go installation and platform support the race detector:

```bash
go test -race ./...
```

The race detector can report races in paths exercised by the run; passing does not prove that untested paths are safe. Include actual concurrent admission, join/replay, delivery, and disconnect scenarios. [Go race detector documentation](https://go.dev/doc/articles/race_detector)

Tests should use controllable clocks, explicit readiness/completion signals, local listeners with available test ports, and bounded deadlines. Clean up every listener, socket, and started goroutine. Avoid tests that depend on arbitrary sleeps or on fixed ports already in use. Do not add disallowed imports or external test frameworks without resolving permission.

## 6. Full CLI Testing

Build with `go build -o TCPChat .` after implementation, as shown above. Stop earlier servers before changing the tested port. The supplied audit requires these binary invocations; the subject's `go run .` invocations remain useful during development.

First, check the no-argument invocation in Terminal A:

```bash
./TCPChat
```

It must listen on `8989` and print `Listening on the port :8989`. Connect with `nc localhost 8989` to verify an established client connection, then stop this server before the custom-port case:

```bash
./TCPChat 2525
```

Expected startup line:

```txt
Listening on the port :2525
```

Connect from another terminal:

```bash
nc localhost 2525
```

The welcome/name/chat behavior should match the default-port run.

From another terminal, test the supplied invalid-arity example:

```bash
./TCPChat 2525 localhost
```

Expected application usage text:

```txt
[USAGE]: ./TCPChat $port
```

No listener should start for that invocation. The subject and supplied audit do not fix its exit status or output stream. The usage text still refers to `./TCPChat $port` even when the program is launched with `go run .` during development.

Check representative invalid single-port inputs after agreeing on their policy:

```bash
./TCPChat banana
./TCPChat -1
./TCPChat 65536
```

Each must be handled safely without panic or a false successful-start announcement. Do not invent a byte-exact error line from these examples. If the selected policy rejects port zero or surrounding whitespace, add explicit cases to `main_test.go` and the golden test table.

While the first server still occupies `2525`, try a second invocation in another terminal:

```bash
./TCPChat 2525
```

It should report the listen failure safely and leave the existing server and clients working. Exact operating-system error text is variable.

CLI unit tests should call testable argument/startup helpers in `main_test.go`; this plan does not assume permission for `os/exec` process-launch tests. Manual terminal checks cover the full invocation.

## 7. Audit/Comparator Testing

The user supplied a manual audit checklist, preserved in [audit_test.md](audit_test.md). It contains functional checks F01–F18, bonus checks B01–B09, and the verdict reasons Empty Work, Incomplete Work, Invalid compilation, Cheating, Crashing, and Leaks. No automated comparator or downloadable audit tool was provided. Do not invent one or claim that reading this checklist constitutes passing an audit.

Use the mapping in [golden_tests.md](golden_tests.md) for package coverage and exact fixtures, then execute each applicable manual audit item after implementation. Check compilation, the exact binary invocations, established client connections, names, nonempty messages, ten-connection admission, goroutines and channels or mutexes, and allowed imports. Preserve live timestamps when comparing history. Keep a record of decisions for unspecified formatting and input cases so reviewers can distinguish requirements from implementation policy.

Record the three-client sender-inclusive broadcast, two/three-computer connection case, four-client departure with three stable survivors, and three-client departure with two notified survivors separately. A check without observed runtime evidence remains unrun; no such checks were run while updating this documentation. Build failures, crashes, or missing behavior need fixes before their corresponding checks can pass. Review socket/goroutine cleanup and distinguish required retained message history from leaked resources. The student should be able to explain the work and reproduce its results.

The supplied bonus checklist covers nine items: changing a name, notifying the group of a name change, multiple independent groups, extra NetCat flags, activity logs, saved logs, a terminal UI using only the permitted `gocui` exception, good practices, and a test file. Good practices remain mandatory in the original subject, and tests remain part of this workflow despite their bonus placement. Plan and verify optional features only after the baseline works; do not silently expand imports or create those features during documentation work.

Keep generated binaries, temporary captures, and any downloaded audit assets outside the submitted project or remove those generated artifacts before submission. The source documentation `audit_test.md` is a review checklist and belongs with the project docs.

Prepare to explain how names are read, why TCP needs line framing, how ten sockets are counted, who owns room state, how a join avoids losing messages, why each client has one writer, how disconnect cleanup works, and what each test proves.

## 8. Final Review Checklist

- [ ] Root contains only essential project entry files; Go logic outside `main.go` and matching root tests lives in `internal/`.
- [ ] Detailed docs and the supplied audit checklist are complete and consistent; this documentation delivery is entirely inside `docs/` as requested.
- [ ] Every Go source file has a matching `_test.go` file where appropriate.
- [ ] Production imports follow the exact allowlist; the test-only `testing` exception and chosen Go version are resolved.
- [ ] `go fmt ./...` has been run and `go test ./...` passes after implementation.
- [ ] Concurrent paths have been checked with `go test -race ./...` where supported, with any limitation recorded.
- [ ] Package and manual checks cover [golden_tests.md](golden_tests.md) and functional audit F01–F18 in [audit_test.md](audit_test.md); actual outcomes are recorded separately from unrun checks.
- [ ] `go build -o TCPChat .` succeeds, and `./TCPChat`, `./TCPChat 2525`, and `./TCPChat 2525 localhost` have been checked against the supplied audit.
- [ ] Default/custom ports and invalid input are handled correctly, with no panic on expected failures.
- [ ] Welcome/logo, name prompt, timestamp/name/message format, usage, and supplied membership notices match the subject.
- [ ] Join notices reach all named participants, including the newcomer after replay, following this plan's literal interpretation of audit F06; leave notices reach remaining participants.
- [ ] Empty messages are neither broadcast nor stored; unnamed clients cannot send chat messages.
- [ ] At most ten connections are admitted, including pending-name sockets under the chosen policy; slots are reusable after termination.
- [ ] With three named clients, client 2's accepted message is delivered by the server to all three, including client 2, with identical timestamp, name, and text; terminal echo is not counted as delivery.
- [ ] Two or three different computers connect to the reachable server host IP, complete the handshake, and exchange messages; loopback-only listening is not used for this audit case.
- [ ] New clients receive all earlier messages in original order with original metadata; join-time traffic is neither lost nor duplicated.
- [ ] With four clients, one disconnect leaves three stable connections; with three clients, one disconnect notifies both remaining clients.
- [ ] Client errors leave the remaining clients connected; cleanup occurs once, sockets/goroutines are released, and slow clients cannot block the room indefinitely.
- [ ] Useful comments explain exported APIs, non-obvious concurrency, parsing, cleanup, and audit-sensitive formatting without repeating obvious code.
- [ ] Remaining Open Questions are resolved for all shipped behavior and are reflected in requirements and tests.
- [ ] The eventual README stays short, explains how to run and test, and links to the PRD and golden tests; it is created only when requested.
- [ ] Binaries, temporary captures, and downloaded/generated audit assets are absent from the submitted project.
- [ ] The audit's Empty Work, Incomplete Work, Invalid compilation, Cheating, Crashing, and Leaks verdict reasons have been reviewed without inventing results or passing thresholds.
- [ ] Bonus B01–B09 outcomes are recorded only for checks actually performed; optional features do not displace baseline requirements or tests.
- [ ] Refactoring follows passing checks, and the student can explain the implementation before bonus work begins.
