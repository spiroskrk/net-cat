# Technical Notes — net-cat (TCPChat)

## 1. Core Idea

One server holds a shared conversation. Each connected terminal has its own session, but everyone participates in the same ordered chat. The server accepts a name, receives message text, assigns metadata, remembers the message, and delivers the same record to every named participant, including the sender.

The supplied [task subject](../zone01-doc-agent-prompt.md) and [audit checklist](audit_test.md) define the required behavior. The package layout and concurrency approach in [architecture.md](architecture.md) are proposals for implementing it. No program has been implemented by this documentation task.

## 2. Important Concepts

| Concept | Simple explanation | Project relevance |
| --- | --- | --- |
| TCP listener | A door where clients can connect | Server waits on the selected port |
| Connection | One conversation channel with one client | Read, write, and close are session operations |
| Byte stream | Bytes arrive without chat-message boundaries | One network read is not necessarily one input line |
| Goroutine | Work that can progress alongside other work | Clients can read/write while the listener keeps accepting |
| Channel | A way to pass work or events between goroutines | Proposed room ownership and client output queues |
| Mutex | A lock protecting shared values | Proposed atomic connection reservation |
| History | Ordered records of earlier chat messages | New participants see the earlier conversation |
| Cleanup | Ending a session and releasing its resources | One failure must not leave a slot occupied or disconnect others |

Keep the TCP requirement distinct from the background description of NetCat's UDP and UNIX-socket features. Neither is a required transport here.

Audit F10 requires connections from two or three different computers. Their `nc` commands must target the server's reachable IP. A listener restricted to loopback cannot pass this check; choose a reachable interface. Local loopback remains appropriate for isolated package tests, which do not replace the multi-computer audit.

## 3. Input Rules

- No server argument is valid and selects `8989`.
- One port argument is supported, as in `go run . 2525`.
- Extra positional arguments produce `[USAGE]: ./TCPChat $port`.
- A connection must supply a nonempty name before participating.
- Subsequent nonempty lines are candidate chat messages.
- Empty messages must not be broadcast or saved as chat messages.
- The body is text: digits, negative-looking values, and decimal-looking values are not numbers to calculate.

Whitespace-only input is not the same as an empty string unless a trimming policy makes it so. The subject does not settle that policy. Separate removal of line endings from removal of meaningful spaces in a message.

## 4. Parsing Rules

Use `bufio` for a clearly documented line-reading approach. Read a complete name/message before passing it to the room. TCP can split one line across reads or combine several lines in one read; tests must cover both.

Keep responsibility boundaries clear:

- Root argument handling chooses configuration; it does not parse chat input.
- `session` owns input framing and name validation.
- `chat` accepts already parsed events and rejects empty chat content at the shared boundary as a safeguard.
- `protocol` formats output; it does not parse or print input.

Decide how LF, CRLF, EOF without a newline, very long lines, and invalid text are handled. A reader's default behavior is not automatically a requirement from the subject. Check and propagate terminal read errors; distinguish a normal disconnect from an unexpected I/O failure.

Do not casually add `strconv` for ports or `regexp` for validation: neither appears in the allowed package list. Choose an approach using permitted packages after the desired port syntax is agreed.

## 5. Core Logic Rules

### Connection lifetime

Proposed states, expressed as responsibilities rather than solved code:

```txt
accepted socket
    -> capacity reserved
    -> waiting for name
    -> named participant with history queued
    -> live chat
    -> removed and capacity released
```

An excess connection is refused before joining. A socket that closes before naming still releases its reservation, but it has no named membership to announce as leaving. Repeated read/write failures for one session must cause cleanup only once.

Counting pending unnamed sockets toward ten is the proposed interpretation of “Maximum 10 connections.” Confirm it for evaluation. Distinguish a reserved socket from an active named participant so capacity checks and join/leave notices use the right state.

### Room ownership

Proposed design: one room-owner goroutine handles membership and publication events in order. The server separately protects its admission count. Each session has one writer that serializes its welcome, replay, notices, chat text, and prompts. These choices make ownership explicit; channels and mutexes are tools, not goals in themselves.

Do not let room-state work block on a client's socket. Choose a slow-writer/queue-overflow policy before implementing asynchronous delivery. There is no subject-provided timeout, queue capacity, or disconnect message to copy. Never silently discard history while claiming that all previous messages are replayed.

Go permits concurrent operations on a `net.Conn`. Application-level output ordering is still a separate design responsibility; using one writer per session makes complete output records easier to order. See the [Go connection documentation](https://pkg.go.dev/net#Conn).

### History and live delivery

Store enough information to reproduce every earlier chat message: original author, timestamp, body, and room order. Choose either immutable structured records or a complete rendered representation consistently. A chat message is not a join notice or a prompt; decide separately whether notices belong in history.

Joining must establish a clear boundary: earlier messages are replayed, later messages arrive live. A message arriving during replay must appear once, in its room position. Copying history and registering later without synchronization can lose a message between the two operations. Registering first without ordered output can let live text overtake history.

Use a small teaching example: history contains A and B, Lee joins, and C arrives during replay. Lee must receive A, B, C once each in the server's chosen order. This describes the invariant to preserve; design the mechanism step by step.

Messages can share the same displayed second. Preserve the room's accepted event order rather than sorting by formatted timestamps. Simultaneous clients have no guaranteed real-world arrival order that a golden test can assume.

### Audit recipient rules

Audit F09 is explicit: when client two sends in a three-client room, clients one, two, and three all receive the same message. Do not exclude the sender from recipient selection. Assign metadata once and use it for every recipient and the history record. A visible local input line is not a server-delivered message.

For F06, the plan follows the literal “all Clients” wording by delivering a join notice to every named participant, including the newly joined client. Queue that client's prior history before its own join notice and later live events. This is an inclusive interpretation of the audit wording; existing clients still receive the notice as the subject requires. Departures notify only remaining participants. The decision about retaining notices in history remains separate.

## 6. Required Formulas

No numerical formulas or statistics are required. The relevant algorithms are line framing, admission control, publication order, replay, and cleanup. Teach each with a small state example before discussing Go implementation.

## 7. Output Formatting Rules

Known strings:

```txt
Listening on the port :8989
[USAGE]: ./TCPChat $port
Welcome to TCP-Chat!
[ENTER YOUR NAME]:
Lee has joined our chat...
Lee has left our chat...
```

These are separate fixtures, not a single expected program run. The penguin and full manual examples are in [golden_tests.md](golden_tests.md).

At the supplied fixed time, name `Yenlik` and body `hello` render as:

```txt
[2020-01-20 16:03:43][Yenlik]:hello
```

Use the Go time layout `2006-01-02 15:04:05` to produce the required date/time shape. Go layouts express the reference time, rather than `%Y`/`%m` tokens. Pass a fixed time to formatter tests; live tests use actual timestamps. See [Go time formatting](https://pkg.go.dev/time#Time.Format).

Keep exact protocol strings in `internal/protocol`. Do not use a default time representation that adds fractions or timezone fields. Do not add a space after the colon, square brackets around the body, or debug text to the sample message shape. Decide line terminators and prompt suffixes explicitly; a Markdown code block does not settle whether a prompt ends with a space or newline.

The provided terminal transcript includes locally typed text and prompts. A prompt with an empty message field is not permission to broadcast an empty message. Audit F09 resolves sender delivery: the sender must receive the same server message as peers. Prompt redraw details remain unspecified.

## 8. Decimal Precision Rules

Decimal precision and numeric rounding do not apply. Timestamps display seconds without fractional seconds. Format the supplied time; do not add numerical rounding logic. Timezone and the exact timestamp-assignment event remain open questions.

## 9. Edge Cases

| Situation | What to reason about |
| --- | --- |
| Occupied port | Report startup failure without a misleading successful-listen line |
| Empty name or EOF while naming | Do not join; clean up a closed socket and its reservation |
| Whitespace-only or duplicate name | Resolve policy before writing a specific expected result |
| Empty message | No broadcast and no history entry |
| CRLF, fragmented lines, or multiple lines | Preserve logical message boundaries and agreed content |
| Partial line at EOF | Decide whether to process or discard before cleanup |
| Tenth and eleventh connections race | Capacity reservation must be atomic |
| Join overlaps publication | No history gap, duplication, or live message overtaking replay |
| Read and write both fail | Remove once; announce once if the client had joined |
| Slow client | Keep healthy clients responsive under a documented policy |
| Two messages share a timestamp | Use room order, not timestamp uniqueness |
| Listener stops | Let owned sessions/goroutines terminate cleanly in tests |

## 10. Common Mistakes

- Copying package names, audit commands, or numeric formulas from an unrelated project.
- Treating a network read as one complete message.
- Trimming all message spaces merely because names need validation.
- Letting an unnamed client receive history or publish chat text.
- Counting only joined users while allowing unlimited unnamed sockets under a ten-connection claim.
- Checking capacity and reserving a slot in separate unsynchronized actions.
- Holding a global lock or room event handler during blocking socket writes.
- Rebuilding history timestamps at replay time.
- Filtering the sender out of broadcasts, or using local input echo as evidence that the server delivered a message.
- Generating separate timestamps for each recipient instead of sharing one accepted message record.
- Testing only with `localhost` and claiming the different-computers audit has passed.
- Losing messages between history capture and live membership.
- Sending output from several goroutines with no application ordering policy.
- Closing shared channels from several places or releasing a slot twice.
- Treating a client write failure as a reason to terminate the entire server.
- Treating a passing race check as proof of message ordering or complete correctness.

## 11. Implementation Reminders

Keep the root small, name responsibilities clearly, and add tests alongside each source file. Explain exported APIs, parsing choices, state ownership, and cleanup with useful short comments. Avoid comments that restate obvious code.

The exact implementation allowlist is `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, `reflect`. The unit-testing recommendation conflicts with the omission of `testing`; resolve that exception before deciding test imports. No Go version or module path is supplied.

Use bounded waits and connection deadlines in tests so failures report instead of hanging. Prefer observable acknowledgments to arbitrary sleeps. Close test sockets/listeners and wait for owned goroutines to finish. Use local loopback listeners with operating-system-assigned ports in package tests as a test setup choice; this does not establish whether CLI port `0` is accepted.

The Go race detector can check the concurrent paths that a run actually executes; it cannot establish coverage of unexecuted paths or prove ordering. See the [official race detector guide](https://go.dev/doc/articles/race_detector).

After implementation, from the project root:

```bash
go fmt ./...
go test ./...
```

On a supported environment:

```bash
go test -race ./...
```

Recommend a future `.gitignore` for actual generated paths such as `/TCPChat`, `/bin/`, and `/tmp/`. Add a selected log output path only if file logging is implemented. Do not create that file in this documentation task. Keep the eventual README short, with runnable examples and links to [prd.md](prd.md) and [golden_tests.md](golden_tests.md).

## 12. Official Audit Reminders

The [supplied manual audit](audit_test.md) contains functional checks F01–F18 and bonus checks B01–B09. Run its compiled `./TCPChat` commands and the procedures mapped in [golden_tests.md](golden_tests.md) after implementation. No automated comparator, archive, or downloadable audit tool was supplied; do not invent one. These checks have not been run during documentation work.

Keep evidence for the exact scenarios: client two sends to all three clients; two or three different computers connect; four clients lose one and the remaining three stay usable; three clients lose one and the remaining two receive a departure notice. Continue checking the subject's ten-connection cap, name requirement, and empty-message suppression even though the pasted audit does not repeat each one.

The audit separately scores activity-log production and log-file saving. Other bonuses are renaming, rename announcements, separate concurrent rooms, extra flags, and a TUI using only the permitted `gocui` UI package. Its bonus section also asks about good practices and a test file; good practices remain required by the subject, and tests remain part of this plan.

Record actual outcomes or **Not run**. For audit F18, explain any observed failure using the supplied categories: Empty Work, Incomplete Work, Invalid compilation, Cheating, Crashing, Leaks. Do not invent a pass, diagnose a leak without evidence, or infer misconduct from code style.

Be ready to explain input, line framing, storage, package boundaries, goroutines, synchronization, history order, exact formatting, failure handling, and why each test is useful. Keep generated audit assets, temporary files, logs, and binaries out of the submission.

## 13. Review Checklist

- [ ] Does the root contain only essential entry/configuration files?
- [ ] Are detailed docs in `docs/`, with the mentor file placed at the eventual project root?
- [ ] Does each Go file have a matching test in its package where appropriate?
- [ ] Do the tests cover the behavior in [golden_tests.md](golden_tests.md)?
- [ ] Are exact package permissions and the chosen Go version respected?
- [ ] Does parsing preserve the agreed message content and logical line order?
- [ ] Are capacity, membership, and history state owned and synchronized clearly?
- [ ] Are empty messages suppressed and invalid inputs handled without panic?
- [ ] Does replay retain original author/time/content and avoid gaps or duplicates?
- [ ] Does every named client, including the sender, receive the same server-delivered message?
- [ ] Have actual clients on two or three different computers connected through the server's reachable IP?
- [ ] Do disconnects release resources once while other clients keep chatting?
- [ ] Are known output strings exact, with seconds precision and no debug output?
- [ ] Does the README explain how to run and test?
- [ ] Are generated/downloaded artifacts absent from the final project?
- [ ] Can the student explain the design and the remaining decisions?

## Open Questions

These are gaps in the subject, not invented requirements. Clear requirements can be documented now. Resolve each policy before implementing or asserting its exact outcome; record the answer and update all affected docs/tests.

| ID | Question | Planning recommendation or effect |
| --- | --- | --- |
| OQ-01 | Is a custom client mode required beyond the shown `nc` usage? | Plan the demonstrated server and manual `nc` client first; do not invent client flags |
| OQ-02 | Which Go version/module path will the evaluation use, and are test-only imports such as `testing` exempt? | Keep implementation imports within the exact list; test plans await this interpretation |
| OQ-03 | Which one-argument ports are valid: numeric range, `0`, signs, spaces, service names? What are error text, stream, and exit status? | Zero args and the shown positive port examples are clear; invalid cases must not start a misleading success path |
| OQ-04 | Which reachable interfaces/address families should the server bind? | Audit F10 requires clients on different computers; loopback-only binding is insufficient. Exact interface/family is still a choice; no extra host argument is required by the audit |
| OQ-05 | Do unnamed connections count toward ten? What should an excess client see? | Proposed: reserve pending and named sockets together; exact refusal text and naming timeout are undecided |
| OQ-06 | Are names trimmed, unique, length-limited, case-sensitive, or character-restricted? Re-prompt or disconnect on invalid name? | Nonempty is required; other rules need agreement |
| OQ-07 | Should whitespace-only messages be suppressed, and should nonempty message spaces be preserved? | Separate emptiness validation from modifying content |
| OQ-08 | What are the accepted line endings, oversized-line behavior, and partial-line-at-EOF policy? | Test the chosen framing and limits; do not silently inherit reader defaults |
| OQ-09 | How are interactive prompts refreshed/terminated? | Sender delivery is resolved by audit F09: all three clients receive the same message. Prompt suffix/redraw details remain unspecified |
| OQ-10 | Which timezone and event define a message's timestamp? | Proposed: assign once when the room accepts a message and retain it; do not impose UTC/local time without agreement |
| OQ-11 | Are join/leave notices part of history, and is replay across server restarts required? | Earlier chat messages are required; in-memory process-lifetime history is the baseline proposal; durable logging is a bonus |
| OQ-12 | How should slow writers, queue overflow, and idle unnamed sockets be handled? | Keep room work off blocking sockets; choose limits/timeouts explicitly without silently losing messages |
| OQ-13 | Does the supplied audit prescribe exact prompt suffixes, error streams/status, or a test-import exception? | A manual audit is now supplied; these details remain unspecified. Use F01–F18 and B01–B09 without inventing extra exact-output rules |

The delivery location is resolved: this bundle keeps the original six documents and the added audit reference inside `docs/` beside the source Markdown. Selecting the eventual Go project path and installing its root `AGENTS.md` is separate from this documentation delivery.
