# Spyros — Client sessions, cleanup, and terminal UI

## Goal and ownership

Connect admitted TCP clients to the chat room, deliver messages reliably, and clean up failed connections without disrupting other clients.

Own the planned `internal/session/` and matching tests. For the bonus, own the terminal client, with a proposed entry point at `cmd/tcpchat-client/` and interface logic under `internal/tui/`. This keeps the root executable dedicated to the audited server; confirm the final layout before implementation.

Related plans: [Kostis](kostis-tasks.md) and [Aris](aris-tasks.md).

## Required tasks

**Initial declarations — Spyros:** Define `session.Room` and the exported `session.Starter` function type in `internal/session/` before implementing session behavior. `Starter` matches the agreed `Start` signature: connection, accepted name, existing buffered reader, capacity-release function, and `Room`, returning an error. It lets Kostis inject either a fake starter or the eventual real `session.Start`. Share these declarations with the team; the source filename is an implementation choice, not a required `contract.go` filename.

1. Receive an admitted connection, accepted name, existing buffered reader, and capacity-release operation from Kostis.
2. Register with Aris's room and retain the returned unique client ID. Duplicate names are permitted.
3. Read complete input lines and pass messages to the room using that ID. Preserve already-buffered input from admission.
4. Deliver room output to the socket in order. Prevent overlapping writes from corrupting messages or announcements.
5. Support simultaneous reading and outgoing delivery with goroutines and channels or mutexes.
6. Handle EOF, read failures, write failures, and slow receivers. A delivery timeout or agreed queue limit disconnects only the affected client.
7. Do not disconnect a client merely because they are not typing. The agreed timeout concerns outgoing delivery, not idle chat participation.
8. On cleanup, close the connection, terminate session workers, leave the room, and release capacity exactly once. Closing the connection must unblock pending network work.
9. Keep plain `nc` clients functional on the same computer and on other LAN computers.

## Shared integration contract

Follow the agreed [Go API and ownership contract](../docs/architecture.md#shared-go-api-contract) and [required policies](../docs/notes.md#agreed-required-contract). Prepare the shared declarations together once before independent implementation; test against fakes until integration. Changes to shared signatures or meanings require team agreement.

Own session.Room, session.Start and the Destination implementation. Start returns nil on ownership acceptance, then registration runs under session ownership; returned startup errors leave resources with Kostis. Start the output worker before Join. Preserve the admission reader; use LF/CRLF framing and discard unfinished EOF input. Ignore whitespace-only messages while preserving other spaces; enforce 4,096 bytes and send `Message too long. Maximum is 4096 bytes.\n` after discarding oversized input. Use one output writer, a separate initial history batch, a 256-live-event queue and a ten-second deadline per message. No repeated nc chat prompt or idle-input timeout. Fail signals cleanup without waiting. Cleanup must handle a failure racing with Join's returned ID, remove registered membership and release capacity once.

## Independent tests and acceptance criteria

Use `net.Pipe`, a fake room, and explicit worker-completion signals. Use bounded waits to detect hangs; do not rely on arbitrary sleeps as proof of cleanup.

| Test | Expected result |
| --- | --- |
| Valid session startup | Name and outgoing destination reach fake room; assigned ID is retained |
| Input split across reads | One complete message submitted when the line completes |
| Several lines in one read | Each message submitted once, in order |
| Admission reader already contains chat bytes | First message is preserved |
| Several outgoing events | Client receives intact, ordered output with no overlapping writes |
| EOF or read failure | Connection, membership, workers, and capacity are cleaned up |
| Write failure | Same cleanup; other sessions remain operational |
| Slow receiver exceeds approved policy | Only that session disconnects |
| Quiet but healthy receiver | Remains connected |
| Simultaneous cleanup requests | One room departure and one capacity release |
| Registration fails | Resources released; no false departure announcement |
| Repeated connect/disconnect | Session workers terminate and capacity stays reusable |

Shared integration tests must fill 10 slots, disconnect one client, admit a replacement, and prove the remaining clients still exchange messages. Also test disconnects during name entry with Kostis and confirm repeated cycles do not leak room membership with Aris.

## Bonus terminal UI — agreed scope

1. Build a custom terminal client using `github.com/jroimartin/gocui`, the UI library permitted by the subject and audit. Do not substitute another UI library.
2. Keep separate chat-display and editable-input areas. Incoming messages must preserve unfinished text and cursor position in the input field.
3. Send entered text to the same server used by `nc`. After submission, clear the input field and display the server-returned formatted message once in the chat area. Do not append a second local copy.
4. Handle welcome/name entry, history, join/leave events, errors, and connection closure appropriately.
5. Support Aris's approved rename and room commands, once their syntax is agreed.
6. Keep the server compatible with plain `nc`: required audit checks must not depend on the custom UI.
7. Coordinate any prompt or protocol differences before coding. Do not infer structured messages through fragile string matching without an agreed protocol design.

### Bonus acceptance tests

- An `nc` client and a `gocui` client exchange messages in the same room.
- The UI receives the sender's own server broadcast and displays it once.
- An incoming message leaves partially typed input intact.
- History and announcements appear in the chat area, not the input area.
- Connection failure closes client workers and exits or reports failure cleanly; terminal state is restored on exit.
- Rename and room operations behave consistently between `nc` and the custom client.

Automate transport and input-state logic where practical; manually verify terminal rendering and cursor preservation.

## Development checkpoints

1. Define and share `session.Room` and `session.Starter`, using Aris's shared chat types and the agreed handoff and failure-signaling contract.
2. Demonstrate input/output tests using `net.Pipe` and a fake room.
3. Complete disconnect, simultaneous failure, and worker-termination tests.
4. Integrate required chat with Kostis and Aris; run the audit using `nc`.
5. Implement and verify the bonus UI, then integrate rename/room features.

## Open Questions

Baseline signatures, limits, input policies, replay, prompts and cleanup rules are agreed in [notes.md](../docs/notes.md). Planned module is `net-cat`, toolchain Go 1.26.2. Standard-library test helpers are approved for test files by the team; evaluator acceptance and toolchain compatibility remain unverified. Remaining bonus/API and LAN deployment choices are listed in [Open Questions](../docs/notes.md#open-questions).

## Shared verification and review

From the eventual Go project root:

```bash
go fmt ./...
go test ./...
go test -race ./...
```

Implementation packages allowed by the subject: `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, `reflect`, with the explicit `gocui` exception for the terminal UI bonus.

Kostis reviews Spyros's work. Spyros reviews Aris's work and explains room ordering/history back to him. All three participate in LAN testing and the final audit walkthrough. Use small functions, useful comments, and matching tests where appropriate. Complete required chat behavior before bonus implementation.
