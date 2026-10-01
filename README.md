# TCPChat — Zone01 NetCat project

A Go TCP group chat for up to **10 simultaneous connections**, designed for use from separate terminals on one computer or different computers on the same local network.

**Status:** startup, admission, room behavior, and the real session runtime are integrated. The full race suite and scripted multi-client TCP checks on localhost passed on 2026-10-01. Different-computer LAN testing and the final manual audit remain **Not run**. The commands below describe how to build, run, and verify the current implementation.

## Team and task plans

| Developer | Required work | Task plan |
| --- | --- | --- |
| Kostis | Server startup, ports, welcome/name entry, capacity and admission | [Kostis's tasks](tasks/kostis-tasks.md) |
| Aris | Membership, broadcasts, history, announcements and synchronization | [Aris's tasks](tasks/aris-tasks.md) |
| Spyros | Client sessions, ordered delivery, disconnect handling and cleanup | [Spyros's tasks](tasks/spyros-tasks.md) |

Shared interfaces are defined so that each component can be developed and tested independently with fake dependencies. Aris reviews Kostis, Spyros reviews Aris, and Kostis reviews Spyros. All three participate in final integration, LAN testing, and the audit walkthrough.

## Required behavior

- Welcome clients with the subject's Linux logo and ask for a non-empty name. Empty or whitespace-only names receive an explanation and another prompt; duplicate names are allowed with unique internal client IDs.
- Broadcast timestamped messages to every chat client, including the sender. Ignore empty messages and exclude them from history.
- Replay previous chat messages to newcomers without missing or duplicating messages during concurrent activity.
- Send join announcements to everyone, including the newcomer; send departure announcements to the remaining clients.
- Keep other clients connected when one leaves or fails. Release connections, session goroutines, membership, and capacity exactly once.
- Use goroutines and channels or mutexes. Quiet clients stay connected; a client that cannot receive messages may be disconnected under the agreed delivery policy.

## Build and start the server

Prerequisites: Go and a Netcat client (`nc`). The project currently declares Go `1.26.2` in `go.mod`.

Open a terminal and change into the project's `net-cat` directory, where `go.mod` is located. Build the executable there:

```bash
go build -o TCPChat .
```

Start on the default port **8989**:

```bash
./TCPChat
```

Or select a custom port:

```bash
./TCPChat 2525
```

Expected startup output for that example:

```txt
Listening on the port :2525
```

Keep this terminal open while the server runs. Use separate terminals for chat clients; messages are typed in the client terminals.

During development, the corresponding `go run` forms are:

```bash
go run .
go run . 2525
```

Excess arguments, such as:

```bash
./TCPChat 2525 localhost
```

must print:

```txt
[USAGE]: ./TCPChat $port
```

and exit without starting the server.

An invalid port value must print:

```txt
Invalid port. Please use a port number between 1 and 65535.
[USAGE]: ./TCPChat $port
```

and exit without starting the server.

Valid custom ports contain digits only and must resolve to a value from `1` through `65535`. Leading zeros are accepted.

## Connect and chat

Leave the server running in the first terminal. Open a **second terminal** for your first chat client. If you started the server with `./TCPChat`, connect to its default port:

```bash
nc localhost 8989
```

If you started it with `./TCPChat 2525`, use that port instead:

```bash
nc localhost 2525
```

To chat with another local client, open a third terminal and run the same `nc` command. Each client needs its own terminal and name entry.

For another computer on the same LAN, use the server computer's actual LAN IP instead of `localhost`. For example, if the server computer's LAN IP is `192.168.1.10`:

```bash
nc 192.168.1.10 2525
```

The server listens on `:port`, allowing network-accessible TCP connections when the host network configuration permits them. The server computer's firewall must allow incoming TCP connections on the selected port. Use the same port in the server and client commands.

Actual multi-computer LAN operation is **Not run**: a second computer was not available for the current verification. Localhost checks do not complete audit F10.

After connecting, the server displays its welcome message and name prompt. Type your name and press **Enter** at:

```txt
[ENTER YOUR NAME]:
```

For an empty or whitespace-only name, the server responds with:

```txt
Invalid name. Please enter a non-empty name.
```

and prompts for the name again.

Names are limited to **64 bytes after trimming**. Oversized names are rejected and the client remains connected so another name can be entered.

Once your name is accepted, type a message, such as `Hello everyone!`, and press **Enter** to send it. Repeat for each message. New clients receive earlier chat messages before joining the live conversation.

A message sent by a client is delivered to all chat clients, including the sender, in this form:

```txt
[2026-09-14 15:30:05][Aris]:Hello everyone!
```

Actual timestamps depend on when the message is sent.

Plain `nc` may show both locally typed text and the server's formatted response, and incoming output can visually interrupt typing.

## Leave the chat and stop the server

- **Leave as a client:** press **Ctrl+C** in that client's `nc` terminal. This closes that connection; the server and other clients continue running. Remaining clients receive a departure announcement once a named client leaves.
- **Stop the server:** press **Ctrl+C** in the terminal running `./TCPChat` or `go run .`. This stops the server and disconnects all connected clients.
- **Start again:** run the server command again, then reconnect each client with `nc` and enter a name. Chat history is kept only in memory, so restarting the server starts a fresh conversation.

## Current implementation status

### Implemented

- Default server port `8989`.
- Custom digits-only ports from `1` to `65535`, including leading-zero input.
- Argument and invalid-port validation.
- TCP listener and concurrent connection acceptance.
- Server-wide capacity limit of 10 connections, including clients still entering their names.
- Exact welcome/logo/name-prompt admission flow.
- Empty and whitespace-only name rejection with retry.
- Duplicate display names.
- Maximum name length of 64 bytes after trimming.
- Bounded-memory name reading while draining rejected input through the terminating newline.
- Preservation of the existing buffered reader across the session handoff boundary.
- Capacity release and connection cleanup for admission failures and disconnects before successful handoff.
- Shared room/session handoff contracts.
- Room membership, message submission, history, join/leave behavior, and related synchronization.
- A single shared room and the real `session.Start` injected through `NewServer` in the root application.
- Session input framing, the 4,096-byte message limit, and recovery after oversized messages.
- Ordered history and live output, per-message write deadlines, and session cleanup.

### Remaining required verification

- Different-computer LAN testing (F10): **Not run**, because no second computer was available.
- The manual `nc` audit walkthrough and final functional verdict.
- Evaluator toolchain compatibility and acceptance of the agreed test-only imports.

## Tests and verification

Run from the Go project root:

```bash
go fmt ./...
go test ./...
go test -race ./...
```

The current server/admission tests include coverage for name handling, capacity, cleanup, handoff arguments, buffered input preservation, malformed input, Unicode whitespace, and the 64-byte name boundary.

The room component has independent tests for membership, message submission, history/live sequencing, filtering, departure behavior, and destination failures.

Session tests cover normal replay/input/disconnect behavior, oversized-message recovery, rejection of a second history handoff after the first batch is consumed, rejection of output after failure, and concurrent failure during blocked history replay with one departure and one capacity release. Additional failure and deadline checks are in [failure_test.go](internal/session/failure_test.go) and [timeout_test.go](internal/session/timeout_test.go).

### Local verification — 2026-10-01

These results apply to the working tree based on `ea4d1e6`, including the local session fixes, regression tests, and application wiring.

`go test -race ./... -count=1 -timeout=60s` passed for the root, chat, server, and session packages. `go vet ./...` also passed. Focused session checks additionally verified:

- A 257-line history batch does not consume the 256 live-event slots. Once those slots fill during blocked replay, another event triggers cleanup of the affected session; its membership is removed, its capacity is released once, and a healthy participant can continue.
- Registration errors before history delivery and during blocked replay close the connection and release capacity once without a false departure.
- Failure before `Join` returns its assigned ID still results in exactly one departure using that ID.
- Every tested history/live write receives a renewed ten-second deadline. An actual socket write timeout cleans up only the affected session, while another real session can still exchange messages.

The timeout test records the requested production deadline, then shortens only the test transport's deadline to 30 milliseconds. It exercises a real `net.Pipe` timeout without waiting ten seconds. The healthy-session checks also confirm that session code does not install read or combined read/write deadlines in the exercised paths.

A temporary race-enabled build of the root executable also passed scripted checks using real TCP sockets on localhost:

- Default port `8989`, custom port `2525`, and leading-zero port selection.
- Exact stderr, exit status 1, and no startup output for excess arguments and invalid ports; controlled failure when a port is occupied.
- Name validation/retry, name trimming, and preservation of a message sent together with the name.
- Three-client delivery including the sender, identical message metadata, and ordered history before the newcomer's join notice.
- Empty-message filtering, preservation of message spaces, LF/CRLF framing, split and batched input, the 4,096-byte boundary, and recovery after oversized messages.
- Three- and four-client departure scenarios, discarded unfinished EOF input, and independent sessions with duplicate names.
- Ten named or unnamed connections, exact eleventh-client rejection and closure, slot reuse, and continued chat through repeated disconnect/reconnect cycles.

No race reports or server errors occurred during these executable checks. Passing tests cover only the exercised paths; they do not establish different-computer connectivity or a final audit verdict.

## Final integration and audit checks

Use this checklist for the remaining manual audit walkthrough. The local scripted checks above cover many of these behaviors; different-computer testing and the final audit are still pending:

- Connect three clients and verify welcome/name entry and join notifications.
- Send from the second client and verify all three receive the same formatted message.
- Join after messages have been sent and verify history replay.
- Disconnect a client and verify the others receive a departure notice and continue chatting.
- Fill all 10 slots, disconnect one client, and verify a replacement can join.
- Repeat connection/disconnection cycles and verify cleanup, including disconnects during name entry.
- Verify empty-message filtering and invalid-name handling.
- Run session failure, replay, queue, and cleanup scenarios.
- Test clients from two or more different computers on the same LAN.
- Record the final functional audit results.

Each task plan contains component-specific tests and failure cases. The supplied audit is a manual checklist; no automated audit comparator was supplied.

Evaluator compatibility with Go `1.26.2` and the use of standard-library test helpers should still be confirmed if required by the evaluation environment.

## Documentation

Project documentation currently includes:

- `docs/prd.md`
- `docs/architecture.md`
- `docs/workflow.md`
- `docs/notes.md`
- `docs/golden_tests.md`
- `docs/audit_test.md`
- `tasks/kostis-tasks.md`
- `tasks/aris-tasks.md`
- `tasks/spyros-tasks.md`

Shared API signatures and ownership are documented in [architecture.md](docs/architecture.md). Agreed capacity, input handling, history, delivery, and prompt policies are documented in [notes.md](docs/notes.md).

The current implementation status and observed local verification are recorded above. The manual audit checklist and its results are maintained in [audit_test.md](docs/audit_test.md).

## Implementation constraints

Use only the subject's permitted implementation packages:

`io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, and `reflect`.

Keep the root entry point small and package application logic under `internal/`.

Generated executables, temporary test data, logs, and generated audit artifacts should not be committed unless explicitly required by the subject.
