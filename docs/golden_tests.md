# Golden Tests — NetCat

## 1. Test ownership and status

All runtime and audit checks are **Not run**. This is a specification, not executable tests. Follow the [contract](notes.md#agreed-required-contract) and [shared APIs](architecture.md#shared-go-api-contract). Standard-library test helpers are approved by the team; evaluator acceptance remains unverified.

| Source/test location | Owner | Independent setup |
| --- | --- | --- |
| main.go / main_test.go | Kostis | Callable CLI helpers |
| internal/server/server.go / server_test.go | Kostis | Fake session starter, local TCP |
| internal/chat/chat.go / chat_test.go | Aris | Fake destinations, controlled clock |
| internal/session/session.go / session_test.go | Spyros | Fake room/release, net.Pipe |

Add matching tests when splitting source files. No protocol package is planned.

## 2. Boundary tests

| Scenario | Expected result | Owner |
| --- | --- | --- |
| Name and first message arrive together | Existing reader reaches session; first message is retained | Kostis/Spyros |
| Start returns error | Admission closes and releases once; no session workers retained | Kostis/Spyros |
| Start returns nil, Join later fails | Spyros cleans up; no false departure or ghost member | Spyros/Aris |
| Failure races with successful Join return | Returned ID is removed; slot/workers released once | Spyros/Aris |
| Begin receives long history | One immutable batch, before own join notice/live events; no per-history-message queue slots | Aris/Spyros |
| Destination fails | Fail signals cleanup without blocking room progress | Aris/Spyros |

## 3. CLI tests

| Input | Expected result |
| --- | --- |
| No argument | Listen on 8989; stdout listening line |
| 2525; 1; 65535; 02525 | Accept numeric port (binding can independently fail) |
| Extra arguments | Only usage on stderr, exit 1; no listener |
| banana; 0; -1; +1; 65536; surrounding spaces | Exact invalid-port explanation then usage on stderr; exit 1 |
| Occupied port | Startup error on stderr, exit 1; no false success line |

Exact text is in notes and Section 8. Test numeric boundaries without requiring successful binding to privileged ports. Successful startup prints the selected numeric port.

## 4. Input tests

| Input | Expected result | Owner |
| --- | --- | --- |
| `  Kostis  ` | Accepted as `Kostis` | Kostis |
| Empty/whitespace-only name | Exact invalid-name line and repeated prompt; connection stays open | Kostis |
| Two equal names | Both admitted with distinct room IDs | Kostis/Aris |
| 64-byte name after trimming | Accepted | Kostis |
| 65-byte name after trimming | Exact name-length error, drain line, prompt again | Kostis |
| Huge surrounding whitespace or oversized line | Bounded retained input; trimming/validation still follow contract | Kostis |
| Empty/whitespace-only message | No broadcast/history entry | Spyros/Aris |
| `  hello  ` | Spaces preserved in broadcast/history | Spyros/Aris |
| 4096-byte message | Accepted | Spyros |
| 4097-byte message | Drain incrementally, exact error, keep connection open; next valid line works | Spyros |
| LF and CRLF | Both delimit lines; delimiter bytes excluded from limits | Both input owners |
| Split reads; several lines in one read | Complete lines submitted once in order | Spyros |
| EOF with unfinished name/message | Discard partial line and clean up | Both input owners |

Count bytes rather than Unicode characters. Cover errors and the next valid line in the same read, including admission handoff. Name prompt ends in a space with no newline; error lines end in newline.

## 5. Room and lifecycle tests

- Three clients, client two sends: all three get identical server-delivered output once; terminal echo is not evidence.
- Known clock/name/body gives the exact fixture below. Duplicate names have independent IDs. Unknown Submit returns an error; repeated Leave succeeds without another announcement.
- Joining receives all previous chat messages, original metadata/order, then own join notice and later events. No notices, prompts, errors or empty messages in history. Restart starts empty.
- Join racing with submit: no lost/duplicate message. Concurrent senders produce consistent accepted room order even in the same displayed second.
- Four clients lose one: remaining three continue chatting. Three clients lose one: remaining two receive one departure notice.
- Ten sockets including unnamed ones reserve all slots; eleventh gets `Chat is full\n` and closes. Disconnect releases a slot for a replacement. Repeated cycles leave no workers/membership behind.

## 6. Formatting tests

At a controlled server-local acceptance time, author Yenlik and body hello produce exactly:

```txt
[2020-01-20 16:03:43][Yenlik]:hello
```

Aris includes one final newline, no space after the colon and no body brackets. Layout is `2006-01-02 15:04:05`. Replay retains that timestamp. Join/leave examples are `Lee has joined our chat...\n` and `Lee has left our chat...\n`.

Kostis tests the complete welcome/penguin below and exact name/error strings from notes. Spyros verifies unchanged serialized output, with no doubled newlines or repeated nc chat prompt. Error output is local to the affected client, never room history.

## 7. Delivery and failure tests

- A write that cannot complete within ten seconds disconnects only that client; a quiet healthy receiver stays connected.
- Each history message gets its own write deadline; the whole batch has no single ten-second deadline.
- Hold the writer to fill 256 pending live events; the next Enqueue returns an error and triggers cleanup. Distinguish pending entries from any message currently being written.
- Long history alone does not fill the live queue. Live events during replay do count; their overflow follows normal failure handling.
- EOF, read error, write error and room failure converge on one cleanup. Close unblocks pending socket work; explicit completion signals prove termination.
- Registration/replay failures do not leave ghost members or duplicate departures. Fail must not deadlock with Leave.

Use bounded test deadlines and observable readiness/completion signals. Tests may use controllable time/deadline helpers to avoid unnecessary real-time waits while preserving the production ten-second policy. A passing race detector does not prove correct ordering or cover unexercised paths.

## 8. Manual Test Inputs and Expected Outputs

### Build and server startup

After implementation, build the audit executable from the project root:

```bash
go build -o TCPChat .
```

In one terminal, run F01:

```bash
./TCPChat
```

Expected startup line:

```txt
Listening on the port :8989
```

Stop that server with Ctrl+C before starting the explicit-port example:

```bash
./TCPChat 2525
```

Expected startup line:

```txt
Listening on the port :2525
```

In another terminal, verify invalid argument count:

```bash
./TCPChat 2525 localhost
```

Expected program text:

```txt
[USAGE]: ./TCPChat $port
```

### Client welcome

Keep the server on port `2525` running. In a separate terminal:

```bash
nc localhost 2525
```

The required visible welcome and prompt are shown below. The wire prompt ends with one trailing space and no newline; the code block shows its visible text only.

```txt
Welcome to TCP-Chat!
         _nnnn_
        dGGGGMMb
       @p~qp~~qMb
       M|@||@) M|
       @,----.JM|
      JS^\__/  qKL
     dZP        qKRb
    dZP          qKKb
   fZP            SMMb
   HZM            MMMM
   FqM            MMMM
 __| ".        |\dS"qML
 |    `.       | `' \Zq
_)      \.___.,|     .'
\____   )MMMMMP|   .'
     `-'       `--'
[ENTER YOUR NAME]:
```

### Multi-client conversation and history

1. Name the first client `Yenlik`. Send `hello` and then `How are you?`. Verify Yenlik receives both formatted messages back from the server.
2. In another terminal, run `nc localhost 2525` and enter `Lee`.
3. Verify Yenlik receives `Lee has joined our chat...`. Verify Lee receives both earlier chat messages with their original timestamps, names, and order, followed by `Lee has joined our chat...` under the documented F06 interpretation.
4. From Lee, send `Hi everyone!`. Verify Yenlik and Lee each receive one server-delivered formatted message containing the same accepted timestamp, Lee's name, and that text. The current timestamp will differ from the subject's historical example.
5. Press Enter on an empty message line. Verify no empty chat message reaches either client or later appears in history.
6. Connect and name a third client. Verify it receives the earlier nonempty messages in order. Send from each participant and check server delivery to all three, including the sender, with identical message metadata. This is F09.
7. Press Ctrl+C in Lee's terminal. Verify the other two clients receive `Lee has left our chat...` and can continue exchanging messages. This covers F12.
8. For F11, restore three clients and add a fourth named client. Disconnect any one client, then verify all remaining three can still exchange messages without reconnecting.

The source transcripts mix terminal input echo, input prompts, received messages, and copied shell lines. Their timestamps also differ across client views. Do not use the entire transcripts as exact network-output fixtures or copy their shell artifacts. Audit F09 independently requires server delivery to the sender; locally typed text alone cannot pass that check.

### Different computers — F10

Use two or three distinct computers. Start `./TCPChat 2525` on the server computer with a listener reachable through its network IP. A listener available only on loopback cannot satisfy F10. Use the actual server IP reachable from the client computers; do not substitute `localhost` for this check.

On each client computer, in Bash:

```bash
read -r -p 'Server IP: ' tcpchat_server_ip
nc "$tcpchat_server_ip" 2525
```

Enter names, exchange messages in both directions, and verify identical message metadata reaches every named client, including each sender. Confirm that a remote newcomer receives history and a remote departure leaves the other clients usable. Localhost-only tests do not complete F10. Record the computers used and observed result when this test is actually performed.

### Capacity and recovery

1. Restart the server for a fresh manual session.
2. Open ten client terminals, each running `nc localhost 2525`, and give each a distinct nonempty name.
3. Verify all ten can remain connected and receive a server-delivered message, including at its sender.
4. Attempt the same connection from an eleventh terminal. Verify it cannot participate as an eleventh client. Expect `Chat is full\n` followed by closure.
5. Close one admitted client and verify the remaining nine stay connected.
6. Connect a new client and verify it can take the released slot.
7. Repeat with sockets that connect but do not yet submit a name; they occupy reserved slots too.

## 9. Supplied Audit and Comparator Guidance

The user supplied a manual audit checklist after the original scaffold was generated. Its questions, commands, and expected text are preserved in [audit_test.md](audit_test.md). No automated comparator, audit executable, or downloadable checking tool was supplied. Use the built `TCPChat` program and `nc` for the manual checks; do not invent a checker command.

### Functional traceability

Every row currently has status **Not run**. The cited test files are future targets; code-review checks and different-computer checks also require the indicated manual evidence.

| Audit ID | Check and evidence | Planned coverage |
| --- | --- | --- |
| F01 | Built executable defaults to `8989` | Section 3; `main_test.go`; Section 8 startup |
| F02 | Built executable rejects extra arguments with exact usage text | Section 3; `main_test.go` |
| F03 | Built executable accepts a custom port | Section 3; `main_test.go` |
| F04 | Two clients connect using `nc` | Sections 2 and 8; `internal/server/server_test.go` |
| F05 | Welcome logo and nonempty-name exchange | Sections 4, 6, and 8; server tests |
| F06 | Join notification | Section 5; chat tests; existing named clients and newcomer, with newcomer history first, under the documented interpretation |
| F07 | First client's message reaches second | Sections 5 and 8; chat/session tests |
| F08 | New client receives earlier conversation | Sections 5 and 8; chat/session history tests |
| F09 | One message reaches all three, including sender | Sections 5, 6, and 8; chat/session tests prove server delivery and identical metadata |
| F10 | Chat works across two or three different computers | Section 8; manual evidence using the server's reachable IP |
| F11 | One of four clients disconnects; three remain usable | Sections 5 and 8; session/chat tests |
| F12 | One of three clients leaves; others receive notice | Sections 5 and 8; chat tests |
| F13 | Time, name, and message format | Section 6; `internal/chat/chat_test.go` |
| F14 | Connections remain established during chat | Sections 5 and 8; session/server integration checks |
| F15 | Goroutines are used | Source review of listener/session/room responsibilities; concurrent behavior tests |
| F16 | Channels or mutexes are used | Source review of shared-state ownership; exercised race checks |
| F17 | Imports follow the allowed package list | Import review plus the test-only exception question in Section 1 |
| F18 | Final failure verdict | Evaluate only after all applicable functional checks; no verdict has been recorded |

### Bonus traceability

These are the supplied checklist's bonus items. Do not add optional feature implementations during documentation work. The subject requires good practices and recommends tests even though the audit also lists them as bonuses. Matching meaningful tests remain part of this project's development plan.

| Audit ID | Bonus area | Future review or test |
| --- | --- | --- |
| B01 | Client rename | Aris/Spyros: session/chat tests after rename syntax is specified |
| B02 | Rename announcement | Aris: chat tests for agreed old/new-name notice |
| B03 | Multiple chat groups | Membership, delivery isolation, and room history tests after a room design is chosen |
| B04 | Additional NetCat flags | CLI tests for each explicitly documented flag |
| B05 | Client activity logs | Kostis: test the shared event sink and concurrent activity events |
| B06 | Logs saved to a file | Kostis: file-output success/error checks after logging policy is agreed |
| B07 | Terminal UI using only the allowed `gocui` package | Spyros: nc/UI interoperability, unfinished input/cursor preservation, one server echo, terminal restoration, and dependency inspection |
| B08 | Good practices | Review package boundaries, useful comments, clear names, errors, and synchronization |
| B09 | Test file | Verify meaningful test files exist and run the approved suite |

The subject's maximum-ten limit, nonempty names, empty-message suppression, and safe error handling remain required even where the audit checklist does not repeat them. Use Sections 2–8 alongside the audit.

After implementation, prepare to explain admission, goroutines, channels or mutexes, line framing, cleanup, replay, timestamps, and why a failed client leaves other clients connected. Complete the supplied checklist after reviewing its instructions, then record observations and verdicts without treating an unrun check as passed.

Keep temporary captures, generated binaries, and optional local log files out of the submitted project. See [workflow.md](workflow.md) for the final review and task-specific `.gitignore` recommendations; this documentation does not create a `.gitignore`.

## 10. Test Coverage Rule

Every Go source file should have a matching test file in the same package where appropriate. Use the source/test ownership mapping in Section 1; add mappings when responsibilities are split into further source files.

After implementation and tests exist, run from the project root:

```bash
go fmt ./...
go test ./...
```

Where the installed Go toolchain and platform support the race detector, also check concurrent code:

```bash
go test -race ./...
```

The race detector supplements behavior tests; it does not prove history ordering, complete delivery, or freedom from deadlocks. Complete the manual `nc` checks and applicable F01–F18/B01–B09 items in [audit_test.md](audit_test.md), including the distinct-computer check.

Record each unresolved policy in [notes.md](notes.md), update its expected result here when agreed, and then add the corresponding test. Neither Go tests nor live networking checks have been run as part of this documentation-only scaffold.
