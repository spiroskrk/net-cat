# Project progress — Kostis

Last updated: 2026-09-18. Latest session reviewed the saved source on the `kostis` branch, based on commit `b034139`, with an uncommitted `readName` helper in `internal/server/server.go`. The earlier runtime verification applies to the baseline before this helper was added; no tests were run during this mentoring session.
Update these sections as implementation and verification advance; record evidence before marking work complete.

## Completed work

- **Project setup:** root module `net-cat`, with Go `1.26.2` declared in [go.mod](go.mod). Architecture, policies, task plans, test fixtures, audit checklist, and build/run/LAN instructions exist.
- **Startup — Kostis:** [main.go](main.go) implements default port `8989`, digits-only ports `1–65535`, leading zeros, argument errors, stderr/exit handling, TCP listening on `:port`, and an accept loop. Connections run in goroutines sharing one `Server`. Port parsing has tests in [main_test.go](main_test.go); executable startup scenarios are not yet verified.
- **Admission — Kostis:** [server.go](internal/server/server.go) implements a mutex-protected limit of ten connections, including clients entering names; exact full-capacity rejection; welcome/logo/prompt; trimmed nonempty names; LF/CRLF support; duplicate names; rejection and retry above 64 bytes after trimming; and discard of unfinished input at EOF. The memory-limit gap is tracked below.
- **Handoff boundary — Kostis/shared:** `NewServer` injects a starter and room. Admission passes the existing connection, trimmed name, buffered reader, and `sync.Once`-protected release callback. Failed admission/handoff closes and releases; successful handoff transfers ownership. `session.Room` and `session.Starter` are declared in [session.go](internal/session/session.go).
- **Room component — Aris:** [chat.go](internal/chat/chat.go) implements mutex-protected membership with unique IDs, duplicate names, timestamped sender-inclusive delivery to destinations, whitespace-only message suppression, in-memory history, history-before-join ordering, join/leave notices, registration rollback when `Begin` fails, idempotent departure, and failed-destination notification. These are component behaviors, not integrated socket chat.
- **Component tests:** [server_test.go](internal/server/server_test.go) covers rejected/successful handoff, disconnect before naming, and repeated release. [admission_test.go](internal/server/admission_test.go) covers buffered messages and handoff arguments, usable sockets after handoff, name boundaries/retries/duplicates, exact welcome, concurrent capacity, loopback TCP slot reuse, and write-failure cleanup. [chat_test.go](internal/chat/chat_test.go) covers formatting, IDs, history/live sequencing, delivery, filtering, rollback, departure, and destination failures using fakes.

## Currently in progress

- **Bounded-memory name reading — Kostis: partial helper implemented, not integrated or tested.** `readName(r *bufio.Reader) (string, bool, error)` now exists below `NewServer` in [server.go](internal/server/server.go). The student wrote it incrementally during this session. It reads bounded fragments with `ReadSlice('\n')`, stores at most 64 bytes in its name slice, drains through the newline, and reports a length flag separately from reading errors. Its current whitespace checks recognize only the ordinary ASCII space; broader whitespace handling remains unfinished.
- **Admission still has the original memory gap.** `HandleConnection` continues to use `ReadString('\n')` and does not call `readName`. The new helper has therefore not changed admission behavior. Existing long-input admission tests verify rejection/retry, not the helper or bounded storage.

## Progress made in this session

### Saved helper behavior, reviewed from source

- Declares `name` as a byte slice and `tooLong` as a boolean before the outer loop, preserving their state between reads.
- Uses an outer loop to call `ReadSlice('\n')` on the supplied reader and an inner `range` loop to inspect each returned byte.
- Stops processing the current fragment at LF. A nil reading error ends the outer loop; `bufio.ErrBufferFull` continues reading the same line.
- Returns an empty name, `false`, and the reading error for other failures, including EOF before a newline. Unfinished input is discarded.
- Skips leading ordinary spaces while the name slice is empty.
- Appends bytes only while `len(name) < 64`. Once full, a subsequent byte other than an ordinary space sets `tooLong` to true. Reading continues so the rest of that line is drained.
- Returns `strings.TrimSpace(string(name)), tooLong, nil` after a complete line. For ordinary-space inputs, this preserves internal spaces and removes surrounding spaces. The stored prefix and the separate flag must be considered together; a returned prefix is not an accepted name when `tooLong` is true.
- Keeps the original reader, so buffered bytes from later lines remain available. This property has been reviewed in the code, not verified by a new test.

### Learning checkpoints covered

Worked through function return types versus returned values, `:=` versus `=`, equality comparisons, variable scope, the effect of nested-loop braces and `break`, `continue`, byte slices, `append`, and returning the correct boolean flag.

Traced why `bufio.ErrBufferFull` means another fragment is needed, why EOF before LF is an incomplete line, and why rejecting a name must drain the rest of that line. Discussed the 64-byte limit after trimming, preservation of internal spaces, and the fact that a chunk boundary is not a line boundary. Continue reinforcing these concepts with short traces; do not assume every distinction is already secure.

### Known limitations of the saved helper

- The two reading-time whitespace checks still use `b == ' '` and `b != ' '`. The proposed `isSpace` boolean has **not** been added.
- A 64-byte name followed by CRLF is incorrectly flagged as too long when `\r` is read. Long surrounding tabs can also produce a false length rejection even though the trimmed name would fit.
- Unicode whitespace handling is unfinished. Recognizing whitespace byte by byte is insufficient for multibyte characters, especially across fragment boundaries. A retained 64-byte prefix can also end partway through a trailing whitespace character; final `TrimSpace` alone cannot repair that truncated encoding.
- No direct `readName` tests were added or run. The helper is still disconnected from `HandleConnection`, so existing admission tests do not exercise it.

## Exact next checkpoint

Continue the mentor workflow in [AGENTS.md](AGENTS.md): let the student implement and review one small step at a time. The last instruction given, not yet implemented, was:

1. Inside the byte loop, after the newline check, declare a boolean named `isSpace`.
2. Make it true when `b` equals any of `' '`, `'\t'`, `'\r'`, `'\v'`, or `'\f'`. Join the comparisons with `||`.
3. Replace `b == ' '` in the leading-whitespace check with `isSpace`.
4. Replace `b != ' '` in the overflow check with `!isSpace`.

Review that change and trace a 64-byte name followed by CRLF, plus a short name surrounded by many tabs. This completes only the ASCII whitespace checkpoint.

Then finish Unicode-aware processing while preserving the byte limit, original input bytes, and reader. Do not trim fragments independently or reject long surrounding whitespace merely because the raw line exceeds 64 bytes. The Unicode implementation approach has not yet been chosen with the student.

Add meaningful helper tests for length boundaries, surrounding/internal whitespace, fragmented input, unfinished EOF, and bounded storage. Include an oversized line, a valid retry, and a first chat message supplied together, verifying that subsequent lines remain readable. When the helper is ready, replace admission's `ReadString` path and use the returned `tooLong` flag rather than relying on the length of its capped result. Preserve exact error/prompt text, retry behavior, the existing reader, and cleanup. Run the affected tests and record actual results before marking this work complete.

## Planned / not yet implemented or verified

- **Session runtime — Spyros:** implement `session.Start`, the `chat.Destination` adapter, registration, input framing and the 4,096-byte message limit, serialized output, separate history delivery, the 256-event live queue, write deadlines, and coordinated worker/socket/membership/capacity cleanup. `session_test.go` currently contains only its package declaration.
- **Application integration:** create the real room and initialize the shared server with `NewServer` and the real starter. `main` currently uses `server.Server{}`, leaving its dependencies nil; a valid name reaches a nil starter call. End-to-end chat is therefore not complete.
- **Remaining verification:** compiled CLI stdout/stderr/exit behavior and bind failures; concurrent room join/submit and send-order scenarios; session failure/replay/queue tests; integrated multi-client chat and continued operation after departures; actual different-computer LAN checks; and recorded functional audit outcomes. Component tests do not establish these results.
- **Documentation maintenance:** refresh stale README/docs statements that implementation or `go.mod` is absent. Existing instructions and audit plans are not evidence that their scenarios passed.
- **Bonuses:** extra flags and activity/file logging (Kostis), rename/multiple rooms (Aris), and the `gocui` client (Spyros). No implementations are present; behavior/API decisions remain to be agreed after required integration.
- **External checks:** evaluator compatibility with Go `1.26.2` and standard-library test helpers remains unverified.

## Verification record

| Date | Check | Observed result |
| --- | --- | --- |
| 2026-09-18 | `go test -race ./... -count=1` — earlier baseline | Previously recorded as passed for root, chat, and server, including loopback TCP admission tests. Session reported `[no tests to run]`. Local socket access was required. This predates the new helper and was not rerun in this session. |
| 2026-09-18 | Current mentoring session — source review | Confirmed the saved partial `readName` helper and that admission still calls `ReadString`. No tests added or run; bounded-storage runtime verification remains outstanding. |
| 2026-09-18 | Full application / different-computer audit | Not verified; no completed audit evidence recorded. |

Scope follows [Kostis's tasks](tasks/kostis-tasks.md), [Aris's tasks](tasks/aris-tasks.md), [Spyros's tasks](tasks/spyros-tasks.md), [architecture](docs/architecture.md), and [agreed policies](docs/notes.md). Passing race checks covers exercised paths only.
