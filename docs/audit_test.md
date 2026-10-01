# Audit Test — NetCat

## Team contract

The [architecture](architecture.md) and [agreed policies](notes.md#agreed-required-contract) supplement this checklist. Kostis owns CLI/admission/names, Aris owns room behavior/rendering, and Spyros owns session I/O/cleanup. Those team choices do not alter the supplied audit questions below.

## Source and Status

Source: the audit checklist supplied by the user in this conversation, followed by “this is the audit test. so take it in mind also.” The functional and bonus questions below preserve that supplied text; stable IDs and formatting have been added for reference.

**Overall status: Partially verified locally.** The local build, race suite, scripted TCP and real `nc` terminal checks, and source review passed at commit `ad2d51a` on 2026-10-01. Different-computer testing, evaluator compatibility, and the final audit verdict remain pending. Individual statuses below distinguish observed results from checks that were not run. No official automated audit comparator was supplied.

Use [golden_tests.md](golden_tests.md) for fixtures, package-test coverage, manual procedures, and traceability. The [PRD](prd.md), [architecture](architecture.md), [workflow](workflow.md), and [notes](notes.md) describe the implementation plan.

## Preparation and Interpretation Notes

The following notes explain how to use the supplied checklist; they are separate from its preserved questions.

- After implementation, run the build command below from the Go project root. The audit invokes the resulting `TCPChat` executable.
- `<port>` and `<host ip>` in the preserved audit are placeholders. Substitute the chosen listening port and the server IP reachable from each client computer.
- F09 explicitly names the first, second, and third clients. The server must deliver each accepted chat message to all named clients, including its sender, with the same timestamp, author, and body. Local terminal echo of typed input cannot satisfy this check.
- For F06, the plan interprets “all Clients” inclusively: existing named participants and the newcomer receive the join notice. The newcomer receives earlier history before its own join notice and later live messages. This is the plan's literal interpretation; the audit does not enumerate recipients as explicitly as F09.
- F10 requires two or three different computers. A localhost-only demonstration is insufficient; the server listener must be reachable through a non-loopback network IP.
- F13's placeholder text is preserved exactly below. The concrete subject example `[2020-01-20 16:03:43][Yenlik]:hello` remains the formatter fixture in [golden_tests.md](golden_tests.md); placeholder brackets around `client.message` do not replace that concrete example.
- The subject's nonempty-name rule, empty-message suppression, maximum of ten connections, and error handling remain required alongside this checklist.
- Bonus features remain optional. The original subject requires good practices and recommends tests despite their placement in this audit's Bonus section. Matching meaningful tests remain part of this project's development plan.
- Prompt endings, local timestamps, input limits, full-capacity text and test-import project policy are agreed in [notes.md](notes.md). Evaluator acceptance of test imports/toolchain and bonus decisions remain under [Open Questions](notes.md#open-questions).

Build command:

```bash
go build -o TCPChat .
```

For the different-computer check, start `./TCPChat 2525` on the server computer. On each client computer, this Bash command obtains the actual server IP instead of embedding an invented address:

```bash
read -r -p 'Server IP: ' tcpchat_server_ip
nc "$tcpchat_server_ip" 2525
```

Run listener commands one at a time as directed by [workflow.md](workflow.md). The questions below preserve the supplied audit; their statuses describe the evidence actually collected.

## Local verification record — 2026-10-01

Checked commit: `ad2d51a`. Installed toolchain: Go `1.26.2`, `linux/amd64`.

- Built a fresh race-enabled executable in a temporary directory with `go build -race -o <temporary-directory>/TCPChat .`.
- `go test -race ./... -count=1 -timeout=60s` passed for the root, chat, server, and session packages. `go vet ./...` passed; `gofmt -l` reported no unformatted Go files.
- Scripted real TCP checks passed for exact CLI errors, occupied-port failure, custom/leading-zero ports, name validation and buffered handoff, history order, three-client sender-inclusive delivery, three-/four-client departures, message boundaries and recovery, and duplicate names.
- Real `/usr/bin/nc` clients in separate pseudo-terminals passed F04–F09 and F11–F13 on loopback port `34911`. The script checked the exact 372-byte welcome/logo/prompt, history before each own join notice, identical broadcasts including the sender, Enter submission, and three-/four-client Ctrl+C departures with continued chat. Terminal echo was disabled so local typing could not masquerade as server delivery.
- Stopping the test server closed its clients' TCP connections. This installed `nc` continued waiting for terminal input afterward; Ctrl+C exited the client processes. All audit-owned servers and clients were stopped after the checks.
- Ten named connections and ten unnamed connections each exhausted capacity; the eleventh received `Chat is full\n` followed by EOF. Disconnecting a client released its slot, and replacement clients could chat through repeated reconnect cycles.
- No race report or server error occurred in these test runs. Existing servers occupied ports `8989` and `2525`, so those exact startup cases were skipped in this run; isolated chat scenarios used available ports. Earlier successful default/custom startup checks are recorded in [README.md](../README.md#local-verification--2026-10-01).
- Source review confirmed goroutines, mutexes/channels, connection handoff, and cleanup ownership. All production standard-library imports match the permitted list. Tests additionally import `testing` and `testing/iotest`, which the team permits; evaluator acceptance remains unconfirmed.
- No second computer was available. These observations do not establish LAN connectivity, complete a human manual walkthrough, or constitute the final evaluator verdict.

## Functional

### F01 — Default Port

Try running `./TCPChat`.

Is the server listening for connections on the default port?

**Status:** Not rerun in this session: an existing server occupied port `8989`. The earlier successful default-port check is recorded in README.md.

### F02 — Usage

Try running `./TCPChat 2525 localhost`.

```txt
[USAGE]: ./TCPChat $port
```

Did the server respond with usage, as above?

**Status:** Passed locally. The fresh executable returned exit status 1, exact usage on stderr, and no startup output for `2525 localhost`.

### F03 — Custom Port

Try running `./TCPChat 2525`.

Is the server listening for connections on the port 2525?

**Status:** Not rerun on port `2525` in this session because an existing server occupied it. Startup and connections passed on available custom ports, including leading-zero input; the earlier `2525` pass is recorded in README.md.

### F04 — Two-Client Connection

Try opening 3 terminals, run on the first terminal the command `./TCPChat <port>` and on the second and third terminal run the command `nc <host ip> <port>`.

Do both clients connect to the server with success?

**Status:** Passed with both scripted TCP and real `nc` clients. Two `nc` clients in separate pseudo-terminals connected concurrently to the fresh server, completed admission, and exchanged messages.

### F05 — Logo and Name

Try creating a server and 2 Clients.

Did the server respond with a linux logo and ask for the name?

**Status:** Passed locally. Admission tests and real `nc` sessions checked the exact welcome/logo/prompt. Scripted TCP checks also verified validation retries.

### F06 — Join Notification

Try creating a server and 2 Clients.

Do all Clients receive a message informing them that the Client joined the chat?

**Status:** Passed locally. Existing clients and each newcomer received its join notice; the newcomer received history first.

### F07 — First Client to Second Client

Try creating a server and 2 Clients and send a message using the first Client.

Does the second Client receive the message?

**Status:** Passed locally. A message from the first client reached the second client and the sender.

### F08 — Previous Messages

Try creating a server and 1 Client and send some messages using this Client. Then create a new Client.

Can the new Client see all the previous messages?

**Status:** Passed locally. Newcomers received the stored messages with unchanged metadata and order before their own join notice.

### F09 — Message Received by All Three Clients

Try creating a server and 3 Clients and send a message using the second Client.

Did all the Clients (first, second and third) received the same message?

**Status:** Passed locally. All three clients received byte-identical server output when the second client sent a message; the check read received bytes rather than terminal echo.

### F10 — Different Computers

Try creating a server and use 2 or 3 different computers and create one Client for each computer.

Did the server/Clients connect with success?

**Status:** Not run. No second computer was available. Localhost evidence does not satisfy this check.

### F11 — Four Clients and One Disconnect

Try creating a server and 4 Clients and disconnect one of the Clients.

Do the rest of the Clients stay connected?

**Status:** Passed locally. After a disconnect from four named clients, all three remaining clients continued exchanging messages.

### F12 — Departure Notification

Try creating a server and 3 Clients and disconnect one of the Clients.

Do the rest of the Clients receive a message notifying that the Client left?

**Status:** Passed locally. After a disconnect from three clients, both remaining clients received the departure notice and continued chatting.

### F13 — Message Identification

Try creating a server and 3 Clients. Then send messages between the Clients.

```txt
[2020-01-20 15:48:41][client.name]:[client.message]
```

Are the messages identified by the name of each Client and the time that the messages were sent, as above?

**Status:** Passed locally. Live output matched the timestamp/name/body format and was identical across recipients; the room tests also verify formatting against a controlled timestamp.

### F14 — Established Connections

Are the connections between server and Clients well established?

**Status:** Passed for the exercised local scenarios. Connections remained usable through broadcasts, history delivery, input errors, and repeated disconnect/reconnect cycles. Different-computer reachability remains covered by pending F10.

### F15 — Goroutines

Does the project present go routines?

**Status:** Passed source review. `main.go` starts a goroutine per accepted connection; `session.Start` starts the session lifecycle, which starts a separate output worker.

### F16 — Channels or Mutexes

Does the project use channels or mutexes?

**Status:** Passed source review. Admission and room state use mutexes; sessions use history/live/completion channels and `sync.Once` for history handoff and failure cleanup.

### F17 — Allowed Packages

Are the students using only the allowed packages?

**Status:** Production imports passed source review: `bufio`, `errors`, `fmt`, `io`, `net`, `os`, `strings`, `sync`, and `time`, plus internal project packages. Final evaluator acceptance of the additional test-only imports `testing` and `testing/iotest` remains pending.

### F18 — Final Audit Verdict

As an auditor, is this project up to every standard? If not, why are you failing the project?(Empty Work, Incomplete Work, Invalid compilation, Cheating, Crashing, Leaks)

**Status:** Pending. Local checks found no failure in the exercised scenarios, but F10, exact-port startup reruns, evaluator toolchain/test-import acceptance, and the final audit review remain outstanding. No final pass/fail verdict is recorded.

## Bonus

### B01 — Rename

Can the Clients change their names?

**Status:** Not run.

### B02 — Rename Announcement

Is the chat group informed if a Client changes his name?

**Status:** Not run.

### B03 — Separate Group Chats

Is the server capable of handling multiple separate group chats simultaneously?

**Status:** Not run.

### B04 — Additional NetCat Flags

Is there more NetCat flags implemented?

**Status:** Not run.

### B05 — Client Activity Logs

Does the server produce logs about Clients activities?

**Status:** Not run.

### B06 — Logs Saved to a File

Are the server logs saved into a file?

**Status:** Not run.

### B07 — Terminal UI

Does the project present a Terminal UI using JUST this package : [https://github.com/jroimartin/gocui](https://github.com/jroimartin/gocui)?

**Status:** Not run.

### B08 — Good Practices

Does the code obey the good practices?

**Status:** Not run.

### B09 — Test File

Is there a test file for this code?

**Status:** Passed. Matching Go test files exist for the root, chat, server, and session packages; all passed the race-enabled suite in this run.

## Recording Results

After implementation, record each item's observed behavior and update its status only from actual evidence. Keep unperformed checks marked **Not run**. Record optional features that were not implemented separately from failed required checks. Complete F18 only after reviewing the applicable functional results; retain the supplied failure categories when explaining a failure.

Keep this checklist as project documentation. Generated binaries, local logs, temporary captures, and downloaded audit assets do not belong in the submitted source.
