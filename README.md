# TCPChat — Zone01 NetCat project

A Go TCP group chat for up to **10 simultaneous connections**, designed for use from separate terminals on one computer or different computers on the same local network.

**Status:** server startup, port handling, connection admission, capacity control, welcome/name entry, and the shared session handoff boundary are implemented. The room component is also implemented independently. Full end-to-end chat is not yet complete because the real session runtime and final application integration are still pending. The commands below describe how to build, run, and verify the current implementation.

## Team and task plans

| Developer | Required work | Bonus work | Task plan |
| --- | --- | --- | --- |
| Kostis | Server startup, ports, welcome/name entry, capacity and admission | Additional flags, activity logging and log files | [Kostis's tasks](tasks/kostis-tasks.md) |
| Aris | Membership, broadcasts, history, announcements and synchronization | Renaming and separate chat rooms | [Aris's tasks](tasks/aris-tasks.md) |
| Spyros | Client sessions, ordered delivery, disconnect handling and cleanup | Terminal client using `gocui` | [Spyros's tasks](tasks/spyros-tasks.md) |

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

Build the executable from the project root:

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

With the server running on port `2525`, a client on the same computer can connect with:

```bash
nc localhost 2525
```

For another computer on the same LAN, use the server computer's actual LAN IP instead of `localhost`. For example, if the server computer's LAN IP is `192.168.1.10`:

```bash
nc 192.168.1.10 2525
```

The server listens on `:port`, allowing network-accessible TCP connections when the host network configuration permits them. The server computer's firewall must allow incoming TCP connections on the selected port. Use the same port in the server and client commands.

Actual multi-computer LAN operation must still be verified during final integration testing.

During admission, enter a name at:

```txt
[ENTER YOUR NAME]:
```

For an empty or whitespace-only name, the server responds with:

```txt
Invalid name. Please enter a non-empty name.
```

and prompts for the name again.

Names are limited to **64 bytes after trimming**. Oversized names are rejected and the client remains connected so another name can be entered.

Once full session integration is complete, a message sent by a client is expected to be delivered to all chat clients, including the sender, in this form:

```txt
[2026-09-14 15:30:05][Aris]:Hello everyone!
```

Actual timestamps depend on when the message is sent.

Plain `nc` may show both locally typed text and the server's formatted response, and incoming output can visually interrupt typing. The planned bonus UI is intended to preserve unfinished input separately from the chat display.

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
- Room membership, message submission, history, join/leave behavior, and related synchronization as an independently tested component.

### Pending required integration

- Real session runtime.
- Wiring the real room and session starter into the application through `NewServer`.
- End-to-end multi-client chat verification.
- Continued operation after client departures.
- Session failure, replay, and queue scenarios.
- Actual different-computer LAN verification.
- Final functional audit.

The root application currently cannot complete a valid client handoff because the real session runtime has not yet been integrated.

## Tests and verification

Run from the Go project root:

```bash
go fmt ./...
go test ./...
go test -race ./...
```

The current server/admission tests include coverage for name handling, capacity, cleanup, handoff arguments, buffered input preservation, malformed input, Unicode whitespace, and the 64-byte name boundary.

The room component has independent tests for membership, message submission, history/live sequencing, filtering, departure behavior, and destination failures.

The full race suite has passed for the currently exercised paths. A passing race detector only covers code paths exercised by the tests.

Manual executable verification has also been performed for:

- default port `8989`;
- custom port `2525`;
- excess arguments;
- invalid ports including `abc`, `0`, `-1`, and `65536`;
- listener bind failure when the requested port is already occupied.

Full client-to-chat behavior is not yet considered verified because application session integration remains pending.

## Final integration and audit checks

After the real session runtime is integrated:

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

## Planned bonuses

Bonus work should begin only after the required functionality is integrated and verified.

Planned bonuses include:

- name changes with announcements;
- separate chat rooms;
- additional Netcat-style flags;
- activity logging and log-file persistence;
- a terminal UI using `github.com/jroimartin/gocui`.

The UI should preserve unfinished input and display each server-delivered message once. Plain `nc` and UI clients should work with the same server. UI startup commands can be documented once its interface is agreed and implemented.

## Documentation

Project documentation currently includes:

- `AGENTS.md`
- `docs/prd.md`
- `docs/architecture.md`
- `docs/workflow.md`
- `docs/notes.md`
- `docs/golden_tests.md`
- `tasks/kostis-tasks.md`
- `tasks/aris-tasks.md`
- `tasks/spyros-tasks.md`
- `progress_log_kostis.md`

Shared API signatures and ownership are documented in [architecture.md](docs/architecture.md). Agreed capacity, input handling, history, delivery, and prompt policies are documented in [notes.md](docs/notes.md).

The detailed current status of Kostis's work and recorded verification evidence are maintained in [progress_log_kostis.md](progress_log_kostis.md).

## Implementation constraints

Use only the subject's permitted implementation packages:

`io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, and `reflect`

with the explicit `gocui` exception for the terminal UI bonus.

Keep the root entry point small and package application logic under `internal/`.

Generated executables, temporary test data, logs, and generated audit artifacts should not be committed unless explicitly required by the subject.