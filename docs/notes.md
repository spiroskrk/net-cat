# Technical Notes — NetCat

The [task files](../aris-tasks.md), [Kostis plan](../kostis-tasks.md), and [Spyros plan](../spyros-tasks.md) divide the work. This page records agreed behavior; [architecture.md](architecture.md) defines shared APIs and ownership. No runtime checks were performed by this documentation update.

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

## Concepts and implementation reminders

TCP is a byte stream: one read may contain half a line or several lines. Admission and session must share the same buffered reader to avoid losing the first message. Input limits must bound retained data, not merely reject a huge string after allocating it.

A reservation is not room membership. An unnamed connection occupies a slot but has no departure announcement. A room ID is not a display name: identical names remain independent members.

Keep room operations independent of blocking socket writes. If history contains A and B and C is published during join, the newcomer must see A, B, C once each in accepted order. Equal displayed seconds do not imply equal ordering positions.

Failure signaling must not wait for cleanup while holding room state: cleanup may need to call Leave. A failure during Join can race with the returned ID; ensure a successfully returned registration is eventually removed even if cleanup began first. Test this explicitly.

The history batch is immutable after acceptance. The writer sends it before queued events; the 10-second limit is renewed per message, not for the whole replay. Queue-full behavior during a busy replay is distinct from history itself occupying queue slots.

## Open Questions

Required project policies above and API signatures are agreed. Remaining external checks and bonus decisions:

- Verify evaluator support for Go 1.26.2 and standard-library test-only imports.
- Select the reachable bind address/address family for the actual LAN and verify from two or three computers. Do not automatically change firewall settings.
- Bonuses: rename/room commands, notice wording, default room, room lifecycle/retention, switching failures, extra flags, logging event fields/file location/failure policy, custom-client arguments/layout, and UI/control-event protocol.

Choose internal data structures and synchronization within the agreed observable contract. A change to a shared signature or its meaning needs team agreement. Bonus decisions do not block independent baseline implementation.
