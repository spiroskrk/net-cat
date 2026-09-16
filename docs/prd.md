# PRD — NetCat TCP Chat

## Scope and authority

Build a Go TCP group-chat server for up to ten connections, usable through plain `nc` on the same computer and on different LAN computers. Require a name before participation, deliver each accepted message to every member including its sender, replay prior chat messages, announce joins/leaves, and keep other clients operating after a disconnect.

The supplied [audit](audit_test.md) records evaluation questions. [Aris](../tasks/aris-tasks.md), [Kostis](../tasks/kostis-tasks.md), and [Spyros](../tasks/spyros-tasks.md) own the implementation tasks. [notes.md](notes.md#agreed-required-contract) records the user-approved project policies, including choices beyond the audit. [architecture.md](architecture.md) defines the agreed interfaces. This is documentation, not evidence of implemented behavior.

## Responsibilities

Kostis owns root startup and server admission, including welcome and names. Spyros takes ownership after handoff and handles session I/O and cleanup. Aris owns room IDs, membership, rendering, timestamps, history and order. Keep logic in `internal/server`, `internal/session`, and `internal/chat`, with matching tests. No separate protocol package is planned.

## Required acceptance criteria

- Default port 8989; explicit digits-only ports 1–65535; agreed invalid-input text, stderr, and exit status 1. Successful listening output goes to stdout.
- LAN-accessible listener; maximum ten reserved sockets including pending names. An excess client gets `Chat is full\n` and closes.
- Exact penguin/welcome fixture; `[ENTER YOUR NAME]: ` prompt. Trim names, allow duplicate display names, reject empty or over-64-byte names with the agreed error and retry.
- Preserve admission-buffered messages across handoff. Accept LF/CRLF complete lines, discard unfinished EOF input, ignore whitespace-only messages, and preserve other message spaces.
- Maximum 4,096 message bytes excluding delimiters; discard oversized lines incrementally and send the agreed error without closing a healthy connection.
- One server-local acceptance timestamp and current author per message; identical sender-inclusive delivery and original history metadata.
- All chat history in memory for the current run, without notices/prompts. Newcomer receives history, own join notice, then subsequent events exactly once in room order.
- Ten-second write deadline per message, 256 queued live events, initial history batch separate from that queue. Quiet clients stay connected; slow/failed clients do not disrupt healthy peers.
- One cleanup path after handoff; no leaked membership, slots, sockets or workers. Repeated Leave is harmless; unknown Submit IDs fail.
- Goroutines and channels or mutexes; exact production allowlist from notes. Planned module `net-cat`, toolchain Go 1.26.2; evaluator version/test-import compatibility remains unverified.

## Verification and completion

Use [golden_tests.md](golden_tests.md) for exact fixtures and owner-specific cases. All three perform integration and audit F01–F18: build/startup, actual different-computer connections, three-client sender-inclusive broadcast, four-client departure stability, and three-client departure notices. Record observed outcomes only; all runtime checks are currently Not run.

Each person can implement and test their package using fakes after preparing the minimal shared declarations together. Contract changes require coordination. Follow [workflow.md](workflow.md) and the learning process in [AGENTS.md](../AGENTS.md).

## Bonuses and exclusions

After required integration passes: Aris owns rename/rooms; Kostis owns extra flags and activity/file logging; Spyros owns the gocui client. Bonus commands and shared event/UI contracts remain to be agreed. Keep nc usable and the audited server at the root entry point. UDP, UNIX-domain sockets, and durable chat history are not baseline requirements.

Good practices and meaningful tests remain part of the plan even though the audit lists them among bonuses. Keep README instructions concise when separately requested. Do not submit generated executables, logs, temporary captures or downloaded audit assets. Preserve the audit document itself. No Go files, README, gitignore, or module are created by this update.
