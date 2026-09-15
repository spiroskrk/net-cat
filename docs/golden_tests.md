# Golden Tests — NetCat

## 1. Golden Tests Overview

This is a test specification for the future Go TCP chat server. It does not create implementation files or executable tests. Read [prd.md](prd.md), [architecture.md](architecture.md), [workflow.md](workflow.md), [notes.md](notes.md), and the mentoring rules in [AGENTS.md](AGENTS.md).

The user has now supplied a manual audit checklist, preserved in [audit_test.md](audit_test.md). Its stable functional IDs are F01–F18 and bonus IDs are B01–B09. All runtime checks are **Not run**. No automated comparator was supplied.

Audit F09 requires a server-delivered message at **all named clients, including its sender**. Every recipient must receive the same accepted timestamp, author, and body; locally echoed typing does not establish server delivery. Audit F10 additionally requires communication between two or three different computers.

For F06, this plan reads “all Clients” inclusively: send the join notice to existing named participants and the newcomer. This is a documented interpretation of the audit's wording. At the newcomer, earlier history must precede its own join notice and later live messages.

A golden test compares actual behavior with an agreed expected result. Use exact comparisons for specified text, fixed times for message-format tests, and event-based checks for concurrent networking. An unspecified detail must remain an Open Question until a policy is agreed.

The planned test boundaries are:

| Future source file | Matching test file | Responsibility |
| --- | --- | --- |
| `main.go` | `main_test.go` | Argument handling and startup coordination |
| `internal/server/server.go` | `internal/server/server_test.go` | Listener, connection admission, capacity |
| `internal/session/session.go` | `internal/session/session_test.go` | TCP line input, names, serialized client output, cleanup |
| `internal/chat/chat.go` | `internal/chat/chat_test.go` | Membership, broadcasts, message order, history |
| `internal/protocol/protocol.go` | `internal/protocol/protocol_test.go` | Welcome, prompt, message, and notification formatting |

The subject permits only `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, and `reflect`. It recommends tests but omits `testing`. Confirm whether test-only imports are exempt before writing automated tests; do not silently assume permission for `testing`, `os/exec`, `bytes`, or other unlisted packages. The `gocui` allowance applies only to the optional terminal UI.

## 2. Package Boundary Tests

| Area | What to test | Why it matters | Expected behavior | Test file |
| --- | --- | --- | --- | --- |
| Root | Arguments reach startup correctly | Keeps CLI behavior separate from chat logic | Defaults and explicit ports follow Section 3; no room logic in root | `main_test.go` |
| Server | Listener initialization succeeds or fails | Distinguishes startup failure from client failure | Success accepts TCP connections; failure is reported without panic | `internal/server/server_test.go` |
| Server | Admission and slot release | Prevents overflow and permanently lost slots | No more than 10 connections admitted; a closed connection releases its slot | `internal/server/server_test.go` |
| Session | Name and message lines are read separately | A name must not accidentally become a chat message | Nonempty name is required before message participation | `internal/session/session_test.go` |
| Session | Output for one connection is serialized | Concurrent writes can corrupt the welcome, history, or live messages | Complete protocol output units remain intact on that connection | `internal/session/session_test.go` |
| Chat | Membership and history are independent of terminal formatting | Makes room behavior testable | Events contain the correct participants and messages; formatting stays in protocol | `internal/chat/chat_test.go` |
| Protocol | Rendering receives explicit message data and time | Avoids nondeterministic golden files | Equal inputs produce equal text without network operations | `internal/protocol/protocol_test.go` |
| Error boundary | One failed client stays isolated | Other clients must remain connected | Session is cleaned up; room continues serving remaining clients | `internal/session/session_test.go`, `internal/chat/chat_test.go` |

## 3. CLI Tests

Commands below are for the future implemented project, run from its root after building `TCPChat` with the command in Section 8. The subject's `go run .` forms remain useful during development; the supplied audit uses the built executable.

| Input or condition | Why it matters | Expected behavior | Test file |
| --- | --- | --- | --- |
| `./TCPChat` | F01: checks the specified default | Starts TCP listening on `8989` and prints `Listening on the port :8989` | `main_test.go` |
| `./TCPChat 2525` | F03: checks the supported explicit port | Starts TCP listening on `2525` and prints `Listening on the port :2525` | `main_test.go` |
| `./TCPChat 2525 localhost` | F02: checks argument count | Prints `[USAGE]: ./TCPChat $port`; does not start the normal server | `main_test.go` |
| More than two user arguments | Checks the same usage rule consistently | Usage response; no normal server startup | `main_test.go` |
| One malformed port argument | Avoids panic and accidental startup | Controlled failure; exact validation policy and response remain open | `main_test.go` |
| Requested port already occupied | Checks listener error handling | Controlled startup failure; no false success announcement | `internal/server/server_test.go` |

The wording “otherwise” in the subject is clarified by its examples: zero arguments and one valid port argument are supported. The usage text is fixed; its output stream and exit status are not specified. `go run` may print its own process-failure text, so distinguish wrapper output from the program's output.

## 4. Parser/Input Tests

| Input or condition | Why it matters | Expected behavior | Test file |
| --- | --- | --- | --- |
| Name `Yenlik` followed by a line ending | Establishes normal admission | Client may participate under `Yenlik` | `internal/session/session_test.go` |
| Empty name line | Enforces the explicit name requirement | Client cannot join with an empty name; reprompt versus close is open | `internal/session/session_test.go` |
| Disconnect before sending a name | Avoids leaked connections and false events | Resources and capacity are released; no named participant is invented | `internal/session/session_test.go` |
| Empty chat line | Enforces empty-message suppression | No empty chat message is broadcast or added to chat history | `internal/session/session_test.go`, `internal/chat/chat_test.go` |
| `hello` or `How are you?` | Checks ordinary message input | Message content is retained in the formatted broadcast | `internal/session/session_test.go` |
| One line arrives in several TCP reads | TCP does not preserve message boundaries | Reads are assembled into one line, without premature messages | `internal/session/session_test.go` |
| Several complete lines arrive in one TCP read | One read is not necessarily one message | Each complete line is processed once and in input order | `internal/session/session_test.go` |
| Whitespace-only name or message | Trimming affects validity and content | Pending policy; do not invent a required golden result | `internal/session/session_test.go` |
| Duplicate name | Identity policy is unspecified | Pending policy; no arbitrary uniqueness requirement | `internal/session/session_test.go` |
| CRLF input, partial final line at EOF, oversized line | Framing and limits affect real clients | Follow the documented policy after clarification; always handle errors without panic | `internal/session/session_test.go` |

Line-based input follows the demonstrated terminal chat. Whether line-ending normalization, surrounding whitespace, or a final unterminated line is accepted must be documented before asserting exact bytes.

## 5. Core Logic Tests

| Scenario | Why it matters | Expected behavior | Test file |
| --- | --- | --- | --- |
| Yenlik sends while Lee is connected | F07 and F09: establishes chat delivery | Lee and Yenlik receive the same server-delivered nonempty message with name and timestamp | `internal/chat/chat_test.go`, `internal/session/session_test.go` |
| Three named clients; one sends | F09: checks all-client delivery | All three, including the sender, receive the message once with the same accepted timestamp, author, and body | `internal/chat/chat_test.go`, `internal/session/session_test.go` |
| Sender sees typed input locally | F09: local echo can disguise a missing broadcast | Verify the sender receives the formatted message from the server connection; local input echo is insufficient | `internal/session/session_test.go` |
| Lee joins an existing room | F06: checks membership events under the documented inclusive interpretation | Existing named clients and Lee receive a join event naming Lee; Lee receives earlier history first; protocol tests verify `Lee has joined our chat...` | `internal/chat/chat_test.go`, `internal/protocol/protocol_test.go` |
| Lee leaves an existing room | Checks departure behavior | Remaining clients receive a leave event naming Lee and remain connected; protocol tests verify `Lee has left our chat...` | `internal/chat/chat_test.go`, `internal/protocol/protocol_test.go` |
| New client joins after two chat messages | Checks mandatory history | Both earlier messages are replayed in room order, retaining original names, text, and timestamps | `internal/chat/chat_test.go` |
| Client joins an empty room | Avoids fabricated history | No prior chat messages are invented | `internal/chat/chat_test.go` |
| A message arrives during history replay | Finds the replay/live transition race | New client receives the message exactly once, through history or live delivery, in a coherent room order | `internal/chat/chat_test.go`, `internal/session/session_test.go` |
| Several clients send concurrently | Shared state must remain consistent | No lost or duplicate messages; all named clients, including each sender, observe a consistent room order and identical message metadata | `internal/chat/chat_test.go` |
| Four named clients; one disconnects | F11: checks continued group operation | Remaining three stay connected and can send and receive messages, including their own server-delivered messages | `internal/session/session_test.go`, `internal/chat/chat_test.go` |
| Two departures or read/write failures coincide | Cleanup can be triggered more than once | Membership and capacity change once per client; one departure notice per departing participant | `internal/chat/chat_test.go`, `internal/session/session_test.go` |
| Tenth named connection is admitted | Checks the permitted boundary | Ten connected participants can exchange messages | `internal/server/server_test.go` |
| Eleventh connection attempts admission | Enforces the hard maximum | It cannot become an eleventh participant; exact rejection text is open | `internal/server/server_test.go` |
| Multiple attempts compete for the last slot | Finds check-then-admit races | Admitted connection count never exceeds 10 | `internal/server/server_test.go` |
| A client leaves at capacity, then another connects | Confirms capacity recovery | A new client can occupy the released slot | `internal/server/server_test.go` |
| One client stops reading or its write fails | A client must not freeze or terminate the group | Healthy clients remain usable; exact slow-client policy must be documented | `internal/session/session_test.go`, `internal/chat/chat_test.go` |

Count pending unnamed sockets together with named clients for admission as a recommended design decision, pending clarification. Include pending-handshake capacity tests once that policy is agreed.

One room-owner goroutine and a serialized writer for each client are proposed architecture choices. Tests should verify membership, ordering, and uninterrupted delivery rather than demand a particular channel arrangement. Simultaneous messages have no predetermined cross-client order before the room accepts them; assert agreement on the resulting order, not an invented wall-clock ordering.

## 6. Output Formatting Tests

### Fixed message fixture

Supply the formatter with the date/time `2020-01-20 16:03:43`, author `Yenlik`, and message `hello`. The exact message text is:

```txt
[2020-01-20 16:03:43][Yenlik]:hello
```

The brackets surround the timestamp and author; there is no added space after the colon and no bracket pair around `hello`. Go's corresponding time layout is `2006-01-02 15:04:05`. The time zone is unspecified. Audit F13's original placeholder notation is preserved in [audit_test.md](audit_test.md); this concrete fixture follows the subject's actual message example.

| Check | Why it matters | Expected behavior | Test file |
| --- | --- | --- | --- |
| Fixed message fixture | Makes formatting deterministic | Exact text above, with transport line termination checked separately | `internal/protocol/protocol_test.go` |
| Fixed join and leave names | Preserves demonstrated notice text | `Lee has joined our chat...` and `Lee has left our chat...` | `internal/protocol/protocol_test.go` |
| Welcome and penguin | Checks the required greeting | Text and ASCII spacing match Section 8 | `internal/protocol/protocol_test.go` |
| Name prompt | Ensures the client knows the next action | Literal `[ENTER YOUR NAME]:` is present; trailing space/newline policy remains open | `internal/protocol/protocol_test.go` |
| Live timestamps | Real executions cannot reuse a historical fixture | Correct date/time shape and agreed time policy; avoid exact current-second expectations | `internal/protocol/protocol_test.go`, `internal/session/session_test.go` |
| Replay timestamps | History must remain historical | Replay preserves original message time, not replay time | `internal/chat/chat_test.go` |
| Broadcast metadata at three clients | F09: formatting must represent one accepted message | Sender and both peers receive the same timestamp, author, and body; do not assign a new timestamp for each recipient | `internal/chat/chat_test.go`, `internal/session/session_test.go` |
| Clean successful output | Debug text corrupts the chat stream | Client output contains only agreed protocol content | `internal/session/session_test.go` |

Choose and document line termination separately from the visible text fixtures. Do not treat a Markdown code block's final newline as proof of an unspecified prompt newline.

## 7. Edge Case Tests

### Open Questions

Agree these policies before promoting their tests to strict acceptance checks:

- Custom Go client mode: the introduction mentions it, but only `nc` usage is specified.
- Whitespace trimming, whitespace-only values, duplicate names, and empty-name retry behavior.
- Whether pending unnamed sockets count toward the maximum of 10; exact full-server response.
- Invalid single-port handling, permitted port values, error streams, and exit status.
- Interactive prompt suffixes, prompt refresh, and any timestamp on join/leave notices. Sender delivery is resolved by F09 and is required.
- Exact notification formatting beyond the demonstrated text. F06's recipient policy is the documented inclusive interpretation above; the newcomer's own join notice follows its history.
- Time zone and the event that determines a message's send timestamp.
- Maximum line sizes, CRLF handling, final unterminated lines at EOF, and slow-client handling.
- Inclusion of notices in history, history lifetime across restarts, and optional persistent logging.
- Go version and permission for test-only imports outside the supplied allowlist.

Additional failure checks belong in `internal/server/server_test.go` and `internal/session/session_test.go`: failed accept, failed read, failed write, abrupt disconnect, and connection cleanup after handshake failure. Expect controlled handling and continued service where recovery is possible; do not invent exact error strings.

Networking tests must have bounded deadlines and clean up listeners, connections, and goroutines. Synchronize on observable events instead of arbitrary sleeps. Negative checks, such as “no empty broadcast,” need a bounded observation window so they cannot hang forever.

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

The required visible welcome and prompt are:

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
4. Attempt the same connection from an eleventh terminal. Verify it cannot participate as an eleventh client. Do not require an invented rejection message.
5. Close one admitted client and verify the remaining nine stay connected.
6. Connect a new client and verify it can take the released slot.
7. After resolving pending-handshake policy, repeat with sockets that connect but do not yet submit a name.

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
| F05 | Welcome logo and nonempty-name exchange | Sections 4, 6, and 8; session/protocol tests |
| F06 | Join notification | Section 5; chat/protocol tests; existing named clients and newcomer, with newcomer history first, under the documented interpretation |
| F07 | First client's message reaches second | Sections 5 and 8; chat/session tests |
| F08 | New client receives earlier conversation | Sections 5 and 8; chat/session history tests |
| F09 | One message reaches all three, including sender | Sections 5, 6, and 8; chat/session tests prove server delivery and identical metadata |
| F10 | Chat works across two or three different computers | Section 8; manual evidence using the server's reachable IP |
| F11 | One of four clients disconnects; three remain usable | Sections 5 and 8; session/chat tests |
| F12 | One of three clients leaves; others receive notice | Sections 5 and 8; chat/protocol tests |
| F13 | Time, name, and message format | Section 6; `internal/protocol/protocol_test.go` |
| F14 | Connections remain established during chat | Sections 5 and 8; session/server integration checks |
| F15 | Goroutines are used | Source review of listener/session/room responsibilities; concurrent behavior tests |
| F16 | Channels or mutexes are used | Source review of shared-state ownership; exercised race checks |
| F17 | Imports follow the allowed package list | Import review plus the test-only exception question in Section 1 |
| F18 | Final failure verdict | Evaluate only after all applicable functional checks; no verdict has been recorded |

### Bonus traceability

These are the supplied checklist's bonus items. Do not add optional feature implementations during documentation work. The subject requires good practices and recommends tests even though the audit also lists them as bonuses. Matching meaningful tests remain part of this project's development plan.

| Audit ID | Bonus area | Future review or test |
| --- | --- | --- |
| B01 | Client rename | Session/chat tests after a rename command is specified |
| B02 | Rename announcement | Chat/protocol tests for agreed old/new-name notice |
| B03 | Multiple chat groups | Membership, delivery isolation, and room history tests after a room design is chosen |
| B04 | Additional NetCat flags | CLI tests for each explicitly documented flag |
| B05 | Client activity logs | Review the chosen activity events and their recorded output |
| B06 | Logs saved to a file | File-output success/error checks once a logging path and policy are agreed |
| B07 | Terminal UI using only the allowed `gocui` package | Manual UI review and dependency inspection |
| B08 | Good practices | Review package boundaries, useful comments, clear names, errors, and synchronization |
| B09 | Test file | Verify meaningful test files exist and run the approved suite |

The subject's maximum-ten limit, nonempty names, empty-message suppression, and safe error handling remain required even where the audit checklist does not repeat them. Use Sections 2–8 alongside the audit.

After implementation, prepare to explain admission, goroutines, channels or mutexes, line framing, cleanup, replay, timestamps, and why a failed client leaves other clients connected. Complete the supplied checklist after reviewing its instructions, then record observations and verdicts without treating an unrun check as passed.

Keep temporary captures, generated binaries, and optional local log files out of the submitted project. See [workflow.md](workflow.md) for the final review and task-specific `.gitignore` recommendations; this documentation does not create a `.gitignore`.

## 10. Test Coverage Rule

Every Go source file should have a matching test file in the same package where appropriate. Use the five source/test mappings in Section 1; add mappings when responsibilities are split into further source files.

After the implementation and approved test setup exist, run from the project root:

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
