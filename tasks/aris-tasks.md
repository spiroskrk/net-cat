# Aris — Chat room, history, and room bonuses

## Goal and ownership

Implement chat behavior independently of TCP sockets: membership, message formatting, delivery, history, and concurrency.

Own the planned `internal/chat/` and matching tests. Related plans: [Kostis](kostis-tasks.md) and [Spyros](spyros-tasks.md).

## Required tasks

1. Assign a unique internal client ID when registering a session. Duplicate display names are allowed; never use names as unique membership keys.
2. Maintain room membership and synchronize shared state using channels or mutexes.
3. Accept complete message lines from Spyros's session component. Ignore empty messages and do not store them in history. Ignore whitespace-only messages too, preserving spaces in other messages.
4. Create one formatted message using its server-assigned send/acceptance timestamp and current sender name:

   ```txt
   [2020-01-20 15:48:41][client.name]:client.message
   ```

   The final `client.message` is the message text; do not add literal brackets around it. Use the subject's examples to verify formatting.

5. Deliver the same formatted message to every registered chat client, including its sender, following the audit. Terminal echo does not count as server delivery.
6. Retain chat messages and replay previous messages to new clients. Preserve original names, timestamps, and order.
7. Announce a successful join to all chat clients, including the newcomer, as agreed from the audit:

   ```txt
   Spyros has joined our chat...
   ```

8. On departure, remove membership and notify the remaining clients once:

   ```txt
   Spyros has left our chat...
   ```

9. Establish an ordering boundary between history replay and live messages so a joining client neither misses nor receives duplicate chat messages.
10. Keep healthy clients operating when another client's outgoing delivery fails. Report the failed destination to the session cleanup mechanism.

## Shared integration contract

Follow the agreed [Go API and ownership contract](../docs/architecture.md#shared-go-api-contract) and [required policies](../docs/notes.md#agreed-required-contract). Prepare the shared declarations together once before independent implementation; test against fakes until integration. Changes to shared signatures or meanings require team agreement.

Own `chat.ClientID`, `chat.Destination` and the concrete room implementing Join, Submit and Leave. Join returns an ID or rolls back provisional membership; repeated Leave returns nil, unknown-ID Submit returns an error. Use server-local acceptance time and format each message once, including its newline. Deliver immutable history through Begin, then enqueue the newcomer's own join notice and later events. Report failed destinations through nonblocking Fail; never close sockets or release capacity. Empty/whitespace-only input is suppressed; preserve spaces in other messages. Keep all chat history for the current run, excluding notices and prompts.

## Independent tests and acceptance criteria

Use fake outgoing destinations and a controlled clock. A server or real socket is unnecessary.

| Test | Expected result |
| --- | --- |
| Two clients with identical names | Distinct IDs and independent membership |
| Sender submits a message with three members present | All three receive the same formatted message, including the sender |
| Empty message | No broadcast and no history entry |
| Known clock and display name | Exact timestamp/name/message formatting |
| New member after several messages | Receives previous messages once, in original order |
| Join | All members, including newcomer, receive the join announcement |
| Leave and repeated leave | Remaining members receive exactly one departure announcement |
| Concurrent join and submit | History/live boundary produces no missing or duplicated chat messages |
| Concurrent sends | Clients observe a consistent room message order |
| Failed destination | Healthy destinations continue receiving; cleanup is notified |
| Departure followed by more traffic | No further delivery to departed member; room continues working |
| Failure during registration/replay | No leaked membership or deadlock |

History uses one separate initial batch; it does not occupy the 256 live-event slots. Test events arriving during replay and overflow separately.

## Bonus tasks — after required integration passes

1. Add name changes. Validate new names using the agreed non-empty rule, preserve the client ID, and announce the change to the affected room. Previously stored messages retain their original names.
2. Support multiple independent rooms. Isolate membership, chat messages, history, and announcements.
3. Agree on a default room and commands for rename, room creation, switching, and listing before implementing command parsing. Plain `nc` clients must still be supported.
4. Preserve the server-wide maximum of 10 connections across rooms; moving rooms does not consume another connection slot.
5. Emit agreed room/activity events through Kostis's logging interface; test using a fake sink.
6. Test rename validation, duplicate names after rename, announcement recipients, room isolation, room switching, and failures during switching. Decide empty-room/history retention policy first.

## Development checkpoints

1. Prepare the agreed shared declarations and demonstrate fake-client tests.
2. Implement and test membership and unique IDs.
3. Implement formatting, empty-message filtering, and sender-inclusive broadcast.
4. Implement history and concurrent join ordering.
5. Complete failure/removal tests and integrate required chat behavior.
6. Add approved rename and room commands, then logging events.

## Open Questions

Baseline signatures, limits, input policies, replay, prompts and cleanup rules are agreed in [notes.md](../docs/notes.md). Planned module is `net-cat`, toolchain Go 1.26.2. Standard-library test helpers are approved for test files by the team; evaluator acceptance and toolchain compatibility remain unverified. Remaining bonus/API and LAN deployment choices are listed in [Open Questions](../docs/notes.md#open-questions).

## Shared verification and review

From the eventual Go project root:

```bash
go fmt ./...
go test ./...
go test -race ./...
```

Verify the audit's three-client sender-inclusive broadcast, history, join/leave announcements, and multi-computer chat. Required cleanup tests must confirm no membership leaks after repeated connections and disconnections.

Implementation packages allowed by the subject: `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, `reflect`. Keep logic beginner-friendly, with small functions, clear package boundaries, useful comments, and matching tests where appropriate.

Spyros reviews Aris's work. Aris reviews Kostis's work and explains admission/capacity handling back to him. All three participate in integration and the final audit walkthrough.
