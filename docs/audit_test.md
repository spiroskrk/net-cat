# Audit Test — NetCat

## Team contract

The [architecture](architecture.md) and [agreed policies](notes.md#agreed-required-contract) supplement this checklist. Kostis owns CLI/admission/names, Aris owns room behavior/rendering, and Spyros owns session I/O/cleanup. Those team choices do not alter the supplied audit questions below.

## Source and Status

Source: the audit checklist supplied by the user in this conversation, followed by “this is the audit test. so take it in mind also.” The functional and bonus questions below preserve that supplied text; stable IDs and formatting have been added for reference.

**Overall status: Not run.** No implementation, compilation, network test, code audit, or runtime verdict has been performed by this documentation update. No automated comparator or downloadable audit tool was supplied. Do not treat this checklist as evidence that the project passes.

Use [golden_tests.md](golden_tests.md) for fixtures, package-test coverage, manual procedures, and traceability. The [PRD](prd.md), [architecture](architecture.md), [workflow](workflow.md), [notes](notes.md), and [mentoring guide](AGENTS.md) describe the implementation plan.

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

Future build command:

```bash
go build -o TCPChat .
```

For the different-computer check, start `./TCPChat 2525` on the server computer. On each client computer, this Bash command obtains the actual server IP instead of embedding an invented address:

```bash
read -r -p 'Server IP: ' tcpchat_server_ip
nc "$tcpchat_server_ip" 2525
```

Run listener commands one at a time as directed by [workflow.md](workflow.md). Each command and question below remains unrun until the implemented project is available.

## Functional

### F01 — Default Port

Try running `./TCPChat`.

Is the server listening for connections on the default port?

**Status:** Not run.

### F02 — Usage

Try running `./TCPChat 2525 localhost`.

```txt
[USAGE]: ./TCPChat $port
```

Did the server respond with usage, as above?

**Status:** Not run.

### F03 — Custom Port

Try running `./TCPChat 2525`.

Is the server listening for connections on the port 2525?

**Status:** Not run.

### F04 — Two-Client Connection

Try opening 3 terminals, run on the first terminal the command `./TCPChat <port>` and on the second and third terminal run the command `nc <host ip> <port>`.

Do both clients connect to the server with success?

**Status:** Not run.

### F05 — Logo and Name

Try creating a server and 2 Clients.

Did the server respond with a linux logo and ask for the name?

**Status:** Not run.

### F06 — Join Notification

Try creating a server and 2 Clients.

Do all Clients receive a message informing them that the Client joined the chat?

**Status:** Not run.

### F07 — First Client to Second Client

Try creating a server and 2 Clients and send a message using the first Client.

Does the second Client receive the message?

**Status:** Not run.

### F08 — Previous Messages

Try creating a server and 1 Client and send some messages using this Client. Then create a new Client.

Can the new Client see all the previous messages?

**Status:** Not run.

### F09 — Message Received by All Three Clients

Try creating a server and 3 Clients and send a message using the second Client.

Did all the Clients (first, second and third) received the same message?

**Status:** Not run.

### F10 — Different Computers

Try creating a server and use 2 or 3 different computers and create one Client for each computer.

Did the server/Clients connect with success?

**Status:** Not run.

### F11 — Four Clients and One Disconnect

Try creating a server and 4 Clients and disconnect one of the Clients.

Do the rest of the Clients stay connected?

**Status:** Not run.

### F12 — Departure Notification

Try creating a server and 3 Clients and disconnect one of the Clients.

Do the rest of the Clients receive a message notifying that the Client left?

**Status:** Not run.

### F13 — Message Identification

Try creating a server and 3 Clients. Then send messages between the Clients.

```txt
[2020-01-20 15:48:41][client.name]:[client.message]
```

Are the messages identified by the name of each Client and the time that the messages were sent, as above?

**Status:** Not run.

### F14 — Established Connections

Are the connections between server and Clients well established?

**Status:** Not run.

### F15 — Goroutines

Does the project present go routines?

**Status:** Not run.

### F16 — Channels or Mutexes

Does the project use channels or mutexes?

**Status:** Not run.

### F17 — Allowed Packages

Are the students using only the allowed packages?

**Status:** Not run.

### F18 — Final Audit Verdict

As an auditor, is this project up to every standard? If not, why are you failing the project?(Empty Work, Incomplete Work, Invalid compilation, Cheating, Crashing, Leaks)

**Status:** Not run. No pass/fail verdict has been recorded.

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

**Status:** Not run.

## Recording Results

After implementation, record each item's observed behavior and update its status only from actual evidence. Keep unperformed checks marked **Not run**. Record optional features that were not implemented separately from failed required checks. Complete F18 only after reviewing the applicable functional results; retain the supplied failure categories when explaining a failure.

Keep this checklist as project documentation. Generated binaries, local logs, temporary captures, and downloaded audit assets do not belong in the submitted source.
