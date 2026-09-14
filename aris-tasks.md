# Aris — Chat room, history, and room bonuses

## Goal and ownership

Implement chat behavior independently of TCP sockets: membership, message formatting, delivery, history, and concurrency.

Own the planned `internal/chat/` and matching tests. Related plans: [Kostis](kostis-tasks.md) and [Spyros](spyros-tasks.md).

## Required tasks

1. Assign a unique internal client ID when registering a session. Duplicate display names are allowed; never use names as unique membership keys.
2. Maintain room membership and synchronize shared state using channels or mutexes.
3. Accept complete message lines from Spyros's session component. Ignore empty messages and do not store them in history. Confirm whitespace-only message handling before finalizing those tests.
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

- Agree on exact Go signatures with Spyros before coding: join, submit message, leave, and outgoing delivery.
- Join accepts a display name and an outgoing destination and returns a unique client ID or failure. Messages and departures identify clients by ID.
- The room owns membership, message timestamps, formatted chat output, announcements, history, and ordering. It does not read sockets, close connections, or release server capacity slots.
- Spyros owns connection lifecycle and invokes leave during cleanup. Repeated leave requests must not duplicate announcements.
- Outgoing delivery is ordered and reports failure without indefinitely blocking the room. No socket writes under a room-state lock.
- Agree how registration rollback works when delivery fails during history replay or a join announcement. Do not leave ghost members behind.
- Define history/live sequencing and prompt ownership together. Proposed sequence for a newcomer: previous chat history, own join announcement, then later events. Treat this ordering as a design proposal until signatures and behavior are finalized.
- Provide a controllable time source for deterministic tests without requiring live TCP clients.

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

Coordinate history size and outgoing queue capacity with Spyros: valid history replay must not accidentally exceed an ordinary live-message queue and disconnect every newcomer.

## Bonus tasks — after required integration passes

1. Add name changes. Validate new names using the agreed non-empty rule, preserve the client ID, and announce the change to the affected room. Previously stored messages retain their original names.
2. Support multiple independent rooms. Isolate membership, chat messages, history, and announcements.
3. Agree on a default room and commands for rename, room creation, switching, and listing before implementing command parsing. Plain `nc` clients must still be supported.
4. Preserve the server-wide maximum of 10 connections across rooms; moving rooms does not consume another connection slot.
5. Emit agreed room/activity events through Kostis's logging interface; test using a fake sink.
6. Test rename validation, duplicate names after rename, announcement recipients, room isolation, room switching, and failures during switching. Decide empty-room/history retention policy first.

## Development checkpoints

1. Agree on room/session contracts and demonstrate fake-client tests.
2. Implement and test membership and unique IDs.
3. Implement formatting, empty-message filtering, and sender-inclusive broadcast.
4. Implement history and concurrent join ordering.
5. Complete failure/removal tests and integrate required chat behavior.
6. Add approved rename and room commands, then logging events.

## Open Questions

- Recommended: treat whitespace-only chat messages as empty. Decide whether to preserve surrounding whitespace on non-empty messages.
- Recommended: history holds chat messages in memory for the current server run, excluding announcements. Persistence, any history limit, and announcement retention have not been agreed.
- Finalize outgoing delivery failure signaling, queue limits, and replay strategy with Spyros.
- Finalize prompt behavior for `nc` and the terminal UI without changing mandatory message delivery.
- Decide bonus command syntax, room lifecycle, rename announcement wording, and logged events.
- Confirm test-only packages against the evaluator's whitelist interpretation.

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
