# Kostis — Server startup, admission, and operations

## Goal and ownership

Build the server entry flow so clients on the same computer or different computers on the same local network can enter the chat safely.

Own the planned `main.go`, `main_test.go`, `internal/server/`, and matching package tests. For bonuses, own command-line options and `internal/activitylog/` with its tests. These are implementation targets; this document does not create Go code.

Related plans: [Aris](aris-tasks.md) and [Spyros](spyros-tasks.md).

## Required tasks

1. Parse startup arguments. No argument means port `8989`; one valid port selects that port. Accept port numbers from 1 through 65535.
2. For excess arguments, print only the required usage line and exit without starting the server:

   ```txt
   [USAGE]: ./TCPChat $port
   ```

3. For an invalid port value, print both lines and exit:

   ```txt
   Invalid port. Please use a port number between 1 and 65535.
   [USAGE]: ./TCPChat $port
   ```

4. Listen on a network-accessible interface, supporting local `nc` clients and clients on other LAN computers. Report listener failures cleanly. On successful startup, print the selected port, for example:

   ```txt
   Listening on the port :2525
   ```

5. Accept connections concurrently and enforce a server-wide maximum of 10. Coordinate capacity reservation and release safely.
6. Send the exact welcome text, Linux logo, and `[ENTER YOUR NAME]:` prompt from the fixture in [golden_tests.md](docs/golden_tests.md#client-welcome), using the agreed trailing-space name prompt.
7. Reject empty or whitespace-only names while keeping the connection open. Print the following message, then show the name prompt again:

   ```txt
   Invalid name. Please enter a non-empty name.
   ```

8. Allow duplicate display names. Hand admitted connections to Spyros's session component; Aris's room assigns unique internal client IDs.
9. Release capacity and close the connection on admission failure or disconnect before handoff.
10. Prepare build/run and LAN testing instructions for later inclusion in the project README. Explain using the server's LAN IP and permitting the selected port through its firewall when necessary. Do not automatically change firewall settings.

## Shared integration contract

Follow the agreed [Go API and ownership contract](docs/architecture.md#shared-go-api-contract) and [required policies](docs/notes.md#agreed-required-contract). Prepare the shared declarations together once before independent implementation; test against fakes until integration. Changes to shared signatures or meanings require team agreement.

Inject a session starter matching session.Start for independent tests. Pass connection, trimmed name, existing buffered reader, release function and room. Nil means ownership transferred; an error means admission still owns closure/release. Release must be idempotent. Count pending names toward ten and send `Chat is full\n` before closing excess connections. Accept digits-only ports 1–65535, including leading zeros; CLI failures use stderr and exit 1. Trim names and enforce 64 bytes after trimming; oversized names get `Name too long. Maximum is 64 bytes.\n` then another `[ENTER YOUR NAME]: ` prompt. Drain oversized input incrementally. No admission code registers membership or announces departures.

## Independent tests and acceptance criteria

Use a fake session starter and local TCP connections; no real chat room is needed.

| Test | Expected result |
| --- | --- |
| No port; explicit `2525` | Select `8989`; select `2525` |
| `2525 localhost` | Exact usage line; no listener started |
| Non-numeric, zero, negative, or out-of-range port | Invalid-port explanation followed by usage; no server started |
| Listener cannot bind | Controlled startup error, no panic |
| New connection | Exact welcome logo and name prompt |
| Empty or whitespace-only name | Explanation and repeated prompt; no session handoff |
| Two identical names | Both may be admitted |
| Ten occupied slots and one extra connection | Extra client cannot join; existing clients remain connected |
| Disconnect during name entry | Connection closes and reserved slot becomes reusable |
| Successful handoff | Correct name, connection, reader, and release operation reach fake session |
| Name and first message arrive together | Handoff preserves the first message |
| Handoff fails | Admission resources released exactly once |
| Repeated admission/disconnection | Capacity remains reusable; admission workers terminate |

Keep tests deterministic using explicit completion signals and bounded waits. Count name-entry connections toward ten and assert the exact full-capacity response from the contract.

## Bonus tasks — after required integration passes

1. Add selected Netcat-style flags without breaking the three audited startup forms. Decide supported flags and syntax before implementation; no specific flags have been approved yet.
2. Provide activity logging and file persistence. Define a shared event sink that Aris and Spyros can replace with a fake in tests.
3. Agree on event fields, log content, destination, opening/closing ownership, and failure behavior. Avoid letting a log failure crash a healthy chat session.
4. Test each flag, concurrent log events, file persistence, and log-write failures. Recommend ignoring generated binaries and log files in version control; do not create `.gitignore` as part of this task document.

## Development checkpoints

1. Prepare the agreed handoff boundary; demonstrate tests with a fake session.
2. Complete argument and startup tests.
3. Complete welcome/name validation tests.
4. Complete capacity and admission cleanup tests.
5. Integrate with Spyros and Aris; run required audit scenarios.
6. Implement and test agreed flags and logging.

## Open Questions

Baseline signatures, limits, input policies, replay, prompts and cleanup rules are agreed in [notes.md](docs/notes.md). Planned module is `net-cat`, toolchain Go 1.26.2. Standard-library test helpers are approved for test files by the team; evaluator acceptance and toolchain compatibility remain unverified. Remaining bonus/API and LAN deployment choices are listed in [Open Questions](docs/notes.md#open-questions).

## Shared verification and review

Build the audit executable from the eventual Go project root:

```bash
go build -o TCPChat .
go fmt ./...
go test ./...
go test -race ./...
```

Verify `./TCPChat`, `./TCPChat 2525`, and `./TCPChat 2525 localhost`. Test clients from at least two LAN computers as well as multiple terminals on one computer.

Implementation packages allowed by the subject: `io`, `log`, `os`, `fmt`, `net`, `sync`, `time`, `bufio`, `errors`, `strings`, `reflect`. The `gocui` exception applies to the bonus terminal UI.

Aris reviews Kostis's work. Kostis reviews Spyros's work and explains session cleanup back to him. Every developer participates in final integration and the audit walkthrough. Use small functions, useful comments, and matching test files where appropriate. Required functionality comes before bonuses.
