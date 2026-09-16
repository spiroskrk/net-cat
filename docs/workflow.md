# Workflow — NetCat TCP Chat

## 1. Plan and setup

Read [architecture.md](architecture.md), [notes.md](notes.md), [golden_tests.md](golden_tests.md), [prd.md](prd.md) and [audit_test.md](audit_test.md). Follow the mentoring process in [AGENTS.md](../AGENTS.md). This update creates documentation only; no implementation checks have run.

Prepare the agreed shared ClientID/Destination declarations, session.Room interface and injectable session starter boundary together before splitting implementation. Planned module: net-cat; toolchain: Go 1.26.2. These declarations let everyone compile against the same types and test with fakes. Shared API changes need coordination; ordinary internal implementation choices do not require waiting for another person's code.

## 2. Independent development and integration

1. **Kostis:** CLI and exact errors, listener, ten-slot admission including unnamed clients, welcome/name validation, bounded input, original-reader transfer, handoff errors and slot recovery. Test with a fake starter and local TCP connections.
2. **Aris:** unique IDs, duplicate names, membership, controlled timestamp/rendering, whitespace suppression, sender-inclusive broadcast, history batch/order, repeated Leave and failed destination signaling. Test with fake destinations and a controlled clock.
3. **Spyros:** fake-room registration, existing-reader framing, size errors, serialized output, Begin/Enqueue/Fail, deadlines, quiet clients, simultaneous failures, registration races and complete cleanup. Test using net.Pipe, fake release and completion signals.
4. **All three:** integrate, prove name-plus-message preservation, registration rollback, long replay with live events, ten-slot recovery and continued healthy chat. Execute the manual audit on nc, including actual LAN computers.
5. **Review:** Aris reviews Kostis and explains capacity back to him; Kostis reviews Spyros and explains cleanup back to him; Spyros reviews Aris and explains history/order back to him.
6. **Bonuses after required integration passes:** agree on rename/room commands and switching (Aris), logging events/files and flags (Kostis), and UI protocol/client arguments (Spyros). Keep nc working. Test cursor/input preservation, single server echo and terminal restoration for the gocui client.

### Small implementation checkpoints

These are steps within each owner's work, not a requirement to wait for another owner's completed implementation. Prepare the shared declarations first, then use fakes. At every step, explain the idea, implement a small piece, run its check, and explain the result before moving on.

| Owner / step | Files or package | Completion checkpoint |
| --- | --- | --- |
| Together: shared setup | Planned go.mod, chat shared types, session.Room and starter boundary | Same types/signatures available to all three; each owner can describe who owns a connection before/after handoff |
| Kostis 1: arguments | main.go and main_test.go | Default/custom ports and exact failures tested without opening sockets |
| Kostis 2: listener | internal/server | Successful listen and occupied-port failure tested; listener closes cleanly |
| Kostis 3: reservation | internal/server | Ten unnamed sockets fill capacity; excess response and released-slot reuse verified |
| Kostis 4: welcome/names | internal/server | Exact penguin/prompt, trimming, duplicate names, 64-byte boundary and retry verified |
| Kostis 5: handoff | internal/server with fake starter | Name plus first message in one read survives; accepted/rejected transfer has correct ownership |
| Aris 1: identity/membership | internal/chat with fake destinations | Equal names get distinct IDs; unknown Submit and repeated Leave have agreed results |
| Aris 2: rendering/broadcast | internal/chat with controlled clock | Exact message/notice text; sender included; whitespace-only messages excluded |
| Aris 3: replay/order | internal/chat | History A, B and concurrent message C reach newcomer once in order, with original metadata |
| Aris 4: failures | internal/chat | Failed destination signals cleanup; registration rolls back; healthy destinations continue |
| Spyros 1: startup | internal/session with fake room/release | Accepted ownership, output worker before Join, assigned ID retained; rejected start leaks no workers |
| Spyros 2: framing | internal/session with net.Pipe | Split/coalesced lines, admission-buffered input, LF/CRLF, EOF and size boundaries verified |
| Spyros 3: output | internal/session | One writer, initial replay batch, 256 pending live events, per-message deadlines and quiet clients verified |
| Spyros 4: cleanup | internal/session | Concurrent read/write/room failures release once; failure racing with Join return leaves no membership |
| Together: integration | All packages and manual audit | Capacity recovery, long replay/live traffic, sender-inclusive chat, distinct-computer connections and departures verified |

Keep source/test files paired when splitting a package further. The exact test cases and visible strings remain in [golden_tests.md](golden_tests.md); bonus work follows passing required integration.

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

Use up to ten client terminals, including at least one waiting at the name prompt under the agreed capacity policy. An eleventh connection must not be admitted to the chat. The agreed response is `Chat is full\n`, then close. Close one accepted connection and verify the next attempt succeeds.

With several named clients, send messages close together and join another client while messages are arriving. Check complete lines, consistent order, and no missing or duplicate replay/live messages. Use automated integration tests for reliable slow-reader, fragmented-input, and simultaneous-admission checks; terminal timing alone is insufficient evidence.

## 5. Package Testing Steps

These commands apply once packages and tests exist. Standard-library test helpers are approved by the team; evaluator acceptance remains unverified. Run the relevant command after a small change:

```bash
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

Tests should use controllable clocks, explicit readiness/completion signals, local listeners with available test ports, and bounded deadlines. Clean up every listener, socket, and started goroutine. Avoid tests that depend on arbitrary sleeps or on fixed ports already in use. Use standard-library test helpers in test files; keep production imports within the allowlist.

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

No listener should start for that invocation. The agreed project policy uses stderr and exit status 1. The usage text still refers to `./TCPChat $port` even when the program is launched with `go run .` during development.

Check representative invalid single-port inputs under Kostis's task policy:

```bash
./TCPChat banana
./TCPChat -1
./TCPChat 65536
```

Each prints `Invalid port. Please use a port number between 1 and 65535.` followed by `[USAGE]: ./TCPChat $port`, then exits without starting a server. Also test `0` and valid range boundaries; use stderr and exit status 1; reject signs and surrounding spaces, and allow leading zeros.

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
- [ ] Production imports follow the exact allowlist; the planned Go 1.26.2 toolchain and team-approved test imports have been checked for evaluator compatibility.
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
