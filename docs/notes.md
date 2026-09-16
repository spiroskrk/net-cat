# Technical Notes — NetCat

The [task files](../tasks/aris-tasks.md), [Kostis plan](../tasks/kostis-tasks.md), and [Spyros plan](../tasks/spyros-tasks.md) divide the work. This page records agreed behavior; [architecture.md](architecture.md) defines shared APIs and ownership. No runtime checks were performed by this documentation update.

## Agreed required contract

These are team-approved project policies. Numerical limits, API choices, and error details supplement the supplied subject/audit; they are not claimed to be evaluator-mandated rules.

| Area | Agreement |
| --- | --- |
| Module/toolchain | Planned module `net-cat`; installed/planned toolchain Go 1.26.2 (`linux/amd64`). No `go.mod` has been created; evaluator compatibility is unverified. |
| Imports | Production: `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, `reflect`. Standard-library test helpers are allowed in `_test.go` files by the team; evaluator acceptance is unverified. `gocui` is the bonus UI exception. |
| Ports | Default 8989; one ASCII-digits-only argument with numeric value 1–65535. Leading zeros allowed; signs, spaces, zero and out-of-range values rejected. |
| CLI errors | Excess arguments: usage only. Invalid port: explanation then usage. CLI failures and startup errors go to stderr with exit status 1; successful listening output goes to stdout. |
| Admission | Ten reserved connections server-wide, including clients entering names. Excess connections receive `Chat is full\n`, then close. Reserve/release atomically; never hold the capacity lock across network I/O. |
| Names | Trim surrounding whitespace; reject empty names; allow duplicate display names with distinct room IDs. Maximum 64 bytes after trimming. Retry invalid/oversized names on the same connection. |
| Input framing | Accept LF and CRLF; remove line delimiters. Discard unfinished input at EOF. Preserve bytes buffered during admission by transferring the existing reader. |
| Messages | Ignore empty/whitespace-only lines; preserve surrounding spaces on nonempty messages. Maximum 4,096 bytes excluding the line ending. Discard oversized lines incrementally, report the error, and continue on the same connection. |
| History | Keep all accepted chat messages in memory for the current server run, excluding announcements, errors and prompts. Restart clears history. Preserve original author, timestamp and room order. |
| Recipients/order | All members, including the sender, receive identical formatted messages. Newcomer: history, own join notice, subsequent events. Existing members receive the join notice; remaining members receive one departure notice. |
| Time | Assign server-local time once when the room accepts a message. Display seconds using `2006-01-02 15:04:05`; keep original timestamps in replay. |
| Delivery | Ten-second deadline per outgoing message, including each history message; 256 pending live events per client. Disconnect only the affected client on timeout, delivery failure or full queue. Quiet clients remain connected. |
| Replay buffering | Accept history as one initial immutable batch, separate from the live queue. Events arriving during replay still count toward the 256-event limit. Long history alone must not consume those slots. |
| Output | Chat, notices and errors end with `\n`. Name prompt is `[ENTER YOUR NAME]: `, with one trailing space and no newline. No repeated chat-input prompt after admission. |
| Ownership | Kostis owns closure/release until successful handoff; Spyros owns them afterward, including registration failure. Close sockets to unblock work; terminate workers and release capacity exactly once. |
| Room identity | Join returns a unique client ID. Submit with an unknown ID returns an error. Repeated Leave is successful and does not repeat announcements. |
| Formatting | Kostis owns startup/admission text; Aris owns chat/notice rendering. No separate `internal/protocol/` package for the baseline. |

### Exact error text

Each line below ends with a newline when emitted. The two invalid-port lines are emitted together; other lines describe separate cases.

```txt
Invalid port. Please use a port number between 1 and 65535.
[USAGE]: ./TCPChat $port
```

```txt
Invalid name. Please enter a non-empty name.
```

```txt
Name too long. Maximum is 64 bytes.
```

```txt
Message too long. Maximum is 4096 bytes.
```

After either name error, send the name prompt again. Drain an oversized line before processing the next one. Count bytes, not Unicode characters; do not confuse CRLF delimiter bytes with content length. Name trimming and bounded input storage must both hold, including very long surrounding whitespace.

## Beginner glossary

| Term | Plain explanation | In this project |
| --- | --- | --- |
| TCP listener | A door where new connections arrive | Kostis listens on the selected port |
| Connection | One client's two-way byte stream | Transferred from admission to Spyros |
| Buffered reader | Reads ahead and keeps unused bytes | Must survive name entry so the first message is not lost |
| Line framing | Finding complete lines in incoming bytes | One socket read can contain part of a line or several lines |
| Goroutine | A task that can progress alongside others | Reading must not stop outgoing messages |
| Channel | A way for goroutines to pass values/signals | Can carry room events, queued output or completion signals |
| Mutex | A lock protecting shared state | Keeps checking/reserving capacity one atomic operation |
| Race | Concurrent access whose result depends on timing without proper synchronization | Two admissions must not claim the last slot together |
| Ownership | Responsibility for changing or cleaning up a resource | Kostis before handoff, Spyros afterward |
| Idempotent cleanup | Calling cleanup again has no extra effect | One released slot and one departure notice |
| Queue / backpressure | Pending work accumulates when a receiver cannot keep up | 256 pending live events, then disconnect that receiver |
| Deadlock | Tasks wait on each other and cannot progress | Fail must not wait for cleanup that needs the room's lock |
| Fake | A small test substitute for another component | Each teammate tests without waiting for the others' implementation |
| History/live boundary | The point separating replay from new events | Every message appears once when someone joins |

## Concepts and implementation reminders

TCP is a byte stream: one read may contain half a line or several lines. Admission and session must share the same buffered reader to avoid losing the first message. Input limits must bound retained data, not merely reject a huge string after allocating it.

A reservation is not room membership. An unnamed connection occupies a slot but has no departure announcement. A room ID is not a display name: identical names remain independent members.

Keep room operations independent of blocking socket writes. If history contains A and B and C is published during join, the newcomer must see A, B, C once each in accepted order. Equal displayed seconds do not imply equal ordering positions.

Failure signaling must not wait for cleanup while holding room state: cleanup may need to call Leave. A failure during Join can race with the returned ID; ensure a successfully returned registration is eventually removed even if cleanup began first. Test this explicitly.

The history batch is immutable after acceptance. The writer sends it before queued events; the 10-second limit is renewed per message, not for the whole replay. Queue-full behavior during a busy replay is distinct from history itself occupying queue slots.

## Common mistakes to avoid

- Treating a socket read as a complete message rather than framing lines.
- Creating a new reader after name entry and losing already-buffered chat bytes.
- Using display names as unique membership keys when duplicates are allowed.
- Letting unnamed sockets evade the ten-slot count, or checking and reserving in separate unsynchronized operations.
- Trimming message bodies when only whitespace-only suppression is intended.
- Allocating an entire oversized line before checking its size.
- Holding a shared lock while writing to a slow socket or waiting for cleanup.
- Making Fail synchronously wait for Leave and deadlocking room processing.
- Missing the returned client ID when cleanup races with Join, leaving a ghost member.
- Recomputing timestamps during replay or separately for each recipient.
- Excluding the sender, or mistaking local terminal echo for server delivery.
- Copying history and registering in separate unordered steps, losing or duplicating messages between them.
- Filling the live queue with every history item, or applying one ten-second deadline to the entire replay.
- Writing from multiple goroutines without serialization, or adding an extra newline to already-formatted output.
- Treating a quiet client as a slow receiver; the deadline applies to outgoing writes, not typing.
- Closing shared channels from multiple places or releasing capacity more than once.
- Claiming localhost tests prove LAN access, or a passing race detector proves message ordering.

## Open Questions

Required project policies above and API signatures are agreed. Remaining external checks and bonus decisions:

- Verify evaluator support for Go 1.26.2 and standard-library test-only imports.
- Select the reachable bind address/address family for the actual LAN and verify from two or three computers. Do not automatically change firewall settings.
- Bonuses: rename/room commands, notice wording, default room, room lifecycle/retention, switching failures, extra flags, logging event fields/file location/failure policy, custom-client arguments/layout, and UI/control-event protocol.

Choose internal data structures and synchronization within the agreed observable contract. A change to a shared signature or its meaning needs team agreement. Bonus decisions do not block independent baseline implementation.
