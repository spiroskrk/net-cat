# TCPChat — Zone01 NetCat project

A planned Go TCP group chat for up to **10 simultaneous connections**, usable from separate terminals on one computer or different computers on the same local network.

**Status:** this directory currently contains the project prompt and team task plans. The Go server and terminal client have not been implemented. The commands below describe how to build, run, and verify the eventual implementation from its project root.

## Team and task plans

| Developer | Required work | Bonus work | Task plan |
| --- | --- | --- | --- |
| Kostis | Server startup, ports, welcome/name entry, capacity and admission | Additional flags, activity logging and log files | [Kostis's tasks](kostis-tasks.md) |
| Aris | Membership, broadcasts, history, announcements and synchronization | Renaming and separate chat rooms | [Aris's tasks](aris-tasks.md) |
| Spyros | Client sessions, ordered delivery, disconnect handling and cleanup | Terminal client using `gocui` | [Spyros's tasks](spyros-tasks.md) |

Agree on shared interfaces first, then develop each component with fake dependencies. Aris reviews Kostis, Spyros reviews Aris, and Kostis reviews Spyros. All three participate in integration, LAN testing, and the audit walkthrough.

## Required behavior

- Welcome clients with the subject's Linux logo and ask for a non-empty name. Empty or whitespace-only names receive an explanation and another prompt; duplicate names are allowed with unique internal client IDs.
- Broadcast timestamped messages to every chat client, including the sender. Ignore empty messages and exclude them from history.
- Replay previous chat messages to newcomers without missing or duplicating messages during concurrent activity.
- Send join announcements to everyone, including the newcomer; send departure announcements to the remaining clients.
- Keep other clients connected when one leaves or fails. Release connections, session goroutines, membership, and capacity exactly once.
- Use goroutines and channels or mutexes. Quiet clients stay connected; a client that cannot receive messages may be disconnected under the agreed delivery policy.

## Build and start the server

Prerequisites for the eventual implementation: Go and a Netcat client (`nc`). The project Go version is still to be selected and recorded in `go.mod`.

```bash
go build -o TCPChat .
```

Start on the default port **8989**:

```bash
./TCPChat
```

Or select a port:

```bash
./TCPChat 2525
```

Expected startup output for that example:

```txt
Listening on the port :2525
```

During development, `go run .` and `go run . 2525` provide the corresponding startup forms.

Excess arguments, such as `./TCPChat 2525 localhost`, must print only the following message and exit:

```txt
[USAGE]: ./TCPChat $port
```

An invalid port value must print both lines and exit without starting the server:

```txt
Invalid port. Please use a port number between 1 and 65535.
[USAGE]: ./TCPChat $port
```

## Connect and chat

With the server running on port 2525, open a separate terminal for each client:

```bash
nc localhost 2525
```

For another computer on the same LAN, use the server computer's actual LAN IP instead of `localhost`. For example, if its IP is `192.168.1.10`:

```bash
nc 192.168.1.10 2525
```

The server must listen on a network-accessible interface, and its firewall must allow incoming TCP connections on the chosen port. Use the same port in server and client commands.

Enter your name at `[ENTER YOUR NAME]:`, then type messages and press Enter. For an empty or whitespace-only name, expect this feedback followed by another name prompt:

```txt
Invalid name. Please enter a non-empty name.
```

A message sent by Aris is delivered to all chat clients, including Aris, in this form:

```txt
[2026-09-14 15:30:05][Aris]:Hello everyone!
```

Actual timestamps depend on when the message is sent. Plain `nc` may show both locally typed text and the server's formatted response; incoming output can visually interrupt typing. The planned bonus UI will preserve input separately from the chat display.

## Tests and audit

Run from the implemented Go project root:

```bash
go fmt ./...
go test ./...
go test -race ./...
```

Manual acceptance checks:

- Verify default/custom ports, excess arguments, and invalid port values using the compiled executable.
- Connect three clients; verify welcome/name entry and join notifications.
- Send from the second client; verify all three receive the same formatted message.
- Join after messages have been sent; verify history replay.
- Test with clients on two or three different computers.
- Disconnect a client; verify the others receive a departure notice and continue chatting.
- Fill 10 slots, disconnect one client, and verify a replacement can join. Repeat connection/disconnection cycles and verify cleanup, including disconnects during name entry.
- Verify empty-message filtering and invalid-name handling.

Each task plan contains independent component tests and failure cases. The supplied audit is a manual checklist; no automated audit comparator was supplied. Confirm whether test-only imports such as `testing` are exempt from the implementation package whitelist.

## Planned bonuses

After required functionality passes integration tests, implement name changes with announcements, separate rooms, additional Netcat-style flags, activity logging with file persistence, and a terminal UI using `github.com/jroimartin/gocui`.

The UI must preserve unfinished input and display each server-delivered message once. Plain `nc` and UI clients must work together on the same server. UI startup commands will be documented once its interface is agreed and implemented.

## Documentation and remaining decisions

See the [project prompt and task subject](../../../zone01-tools/zone01-doc-agent-prompt.md) and the three linked task plans for scope, boundaries, and open questions. Shared API signatures, full-capacity feedback, treatment of name-entry slots, whitespace-only chat messages, history retention, delivery limits, prompt behavior, bonus command syntax, and logging details still need final decisions.

The planned documentation scaffold includes `AGENTS.md`, `docs/prd.md`, `docs/architecture.md`, `docs/workflow.md`, `docs/notes.md`, and `docs/golden_tests.md`. These files have not yet been generated; the PRD will hold detailed requirements and golden tests will hold expected test cases.

Use only the subject's permitted implementation packages: `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, and `reflect`, with the explicit `gocui` exception for the terminal UI bonus. Keep the root entry point small and package logic under `internal/`. Recommend a `.gitignore` for generated executables, temporary test data, and log files once their paths are chosen; do not submit generated audit assets or binaries.
