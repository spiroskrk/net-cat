**# Project progress — Kostis**

Last updated: 2026-09-21. Latest session continued on the \`kostis\` branch at commit \`6cc5344\`, with uncommitted changes to \`internal/server/server.go\` and its tests. The student completed the Unicode-aware \`readName\` implementation, including preservation of malformed original bytes. At the student's explicit request, the assistant added direct helper tests and an allocation benchmark to \`internal/server/server_test.go\`. The focused tests passed, and the bounded-memory `readName` helper has now been integrated into admission. Post-integration server tests and the full race suite passed.

Update these sections as implementation and verification advance; record evidence before marking work complete.

**## Completed work**

\- **\*\*Project setup:\*\*** root module \`net-cat\`, with Go \`1.26.2\` declared in [go.mod]\(go.mod). Architecture, policies, task plans, test fixtures, audit checklist, and build/run/LAN instructions exist.

\- **\*\*Startup — Kostis:\*\*** [main.go]\(main.go) implements default port \`8989\`, digits-only ports \`1–65535\`, leading zeros, argument errors, stderr/exit handling, TCP listening on \`:port\`, and an accept loop. Connections run in goroutines sharing one \`Server\`. Port parsing has tests in [main_test.go]\(main_test.go); executable startup scenarios are not yet verified.

\- **\*\*Admission — Kostis:\*\*** [server.go]\(internal/server/server.go) implements a mutex-protected limit of ten connections, including clients entering names; exact full-capacity rejection; welcome/logo/prompt; trimmed nonempty names; LF/CRLF support; duplicate names; rejection and retry above 64 bytes after trimming; and discard of unfinished input at EOF. Admission now uses the bounded-memory `readName` helper instead of `ReadString` for name input.

\- **\*\*Handoff boundary — Kostis/shared:\*\*** \`NewServer\` injects a starter and room. Admission passes the existing connection, trimmed name, buffered reader, and \`sync.Once\`-protected release callback. Failed admission/handoff closes and releases; successful handoff transfers ownership. \`session.Room\` and \`session.Starter\` are declared in [session.go]\(internal/session/session.go).

\- **\*\*Room component — Aris:\*\*** [chat.go]\(internal/chat/chat.go) implements mutex-protected membership with unique IDs, duplicate names, timestamped sender-inclusive delivery to destinations, whitespace-only message suppression, in-memory history, history-before-join ordering, join/leave notices, registration rollback when \`Begin\` fails, idempotent departure, and failed-destination notification. These are component behaviors, not integrated socket chat.

\- **\*\*Component tests:\*\*** [server_test.go]\(internal/server/server_test.go) covers rejected/successful handoff, disconnect before naming, and repeated release. [admission_test.go]\(internal/server/admission_test.go) covers buffered messages and handoff arguments, usable sockets after handoff, name boundaries/retries/duplicates, exact welcome, concurrent capacity, loopback TCP slot reuse, and write-failure cleanup. [chat_test.go]\(internal/chat/chat_test.go) covers formatting, IDs, history/live sequencing, delivery, filtering, rollback, departure, and destination failures using fakes.

\- **\*\*Direct name-parser tests:\*\*** [server_test.go]\(internal/server/server_test.go) now tests \`readName\` and \`isNameSpace\` directly: exact accepted bytes, empty names, ASCII/Unicode whitespace, LF/CRLF, 64-byte boundaries, malformed bytes versus genuine U+FFFD, fragmented input, EOF/read failures, large generated streams, and preservation of retry/chat lines. An allocation benchmark checks increasing stream lengths without constructing whole input lines in memory. These tests passed, and admission now uses the helper.

**## Currently in progress**

- **Kostis admission work: verification/cleanup checkpoint.** The bounded-memory name-reading integration is complete and verified by the current server tests and full race suite. Remaining Kostis work is outside this checkpoint, including executable startup verification and later application integration/bonus work.

**## Progress made in this session**

**### Saved helper behavior, reviewed from source**

\- Declares \`name\`, \`tooLong\`, and \`byteCount\` before the loop so their state survives successive reads. Calls \`ReadRune\` on the supplied reader to obtain a rune, its original byte width, and a reading error.

\- Returns an empty name, \`false\`, and the error when reading fails, including EOF before LF. A successfully read LF ends the loop; unfinished input is discarded.

\- Uses \`isNameSpace\`, implemented with \`strings.TrimSpace(string(ch))\`, to recognize whitespace consistently with final name trimming. Leading whitespace is skipped while the name slice is empty, before counting or storing it.

\- Adds the rune's byte width to \`byteCount\` only while the counter is at most 64. Once it crosses the limit, it stays above 64, avoiding unbounded counter growth. A nonwhitespace rune encountered above the limit sets \`tooLong\`; reading still continues through LF. Trailing whitespace alone does not cause rejection, but whitespace followed by more name content counts toward the limit.

\- Stores a rune only when \`byteCount <= 64\`. Valid UTF-8 is retained through \`append(name, string(ch)...)\`, so a multibyte rune is never partly stored. A complete trailing whitespace rune that crosses the cap can be omitted without damaging the retained encoding.

\- Distinguishes an actual U+FFFD character from malformed input: \`ch == '\uFFFD' && size == 1\` means one original invalid byte was decoded as the replacement rune. In that branch, \`UnreadRune\` rewinds the read, \`ReadByte\` retrieves the original byte, and one \`append\` stores it. Both reading errors are checked. A genuine U+FFFD uses the normal branch and retains its three bytes.

\- Returns \`strings.TrimSpace(string(name)), tooLong, nil\` after a complete line. Internal whitespace and original nonwhitespace bytes are preserved for accepted names. A capped prefix is not an accepted name when \`tooLong\` is true.

\- Keeps the original reader and consumes only through the name's LF, leaving later buffered lines available. The new direct tests verify this with fragmented streams and rejected names followed by valid retries and chat messages.

**### Learning checkpoints covered**

Continued from the ASCII whitespace checkpoint to the distinction between \`ReadSlice\` byte fragments and \`ReadRune\` results, including the difference between a rune and its byte width. Worked through boolean comparisons and \`&&\`, variable scope, placement of braces and \`else\`, and \`append(name, string(ch)...)\` for adding a string's bytes to a byte slice.

Traced why consumed-byte counting and retained storage are separate, why trailing whitespace can extend beyond the cap without rejecting a name, and why later nonwhitespace changes that result. Discussed Unicode whitespace at the boundary and preserving malformed input with \`UnreadRune\` followed by \`ReadByte\`. Reinforced discarding unfinished EOF input, draining rejected lines, and keeping the existing reader. Continue with short traces and very small implementation steps; do not assume all of these distinctions are secure. The student had not yet answered why an invalid byte is stored exactly once when they explicitly asked the assistant to add all needed tests to \`server_test.go\`; that authorized test implementation without requiring another permission question.

**### Verification and remaining gaps**

\- Source review found no correctness defect in the saved helper against the agreed name policy. Seven new test functions and one benchmark now exercise the helper directly; the earlier socket tests remain in place.

\- Focused helper tests passed. Their coverage report showed \`readName\` at 92.6% and \`isNameSpace\` at 100%. The two defensive error returns during \`UnreadRune\`/\`ReadByte\` recovery were not exercised; normal buffer operation retains the byte that is immediately reread.

\- \`go test -race ./... -count=1 -timeout=60s\` passed for root, chat, server, and session. Session still reports \`[no tests to run]\`. A passing race detector covers only exercised paths.

\- Generated long-stream tests passed. \`BenchmarkReadNameLongInput\` measured 64, 65,536, and 1,048,576 repetitions for ASCII names, malformed-byte names, and surrounding Unicode whitespace. Allocations stayed about 4.4–4.5 KB per operation across sizes, including reader/fixture setup. This is empirical regression evidence, not a universal bound or a timing threshold.

\- \`HandleConnection\` still calls \`ReadString\`; admission therefore retains its original unbounded-line allocation gap. The helper has not yet changed connection behavior.

**## Exact next checkpoint**

Continue the mentor workflow in [AGENTS.md]\(AGENTS.md): let the student implement and review one small step at a time.

The bounded-memory admission integration is complete. Do not repeat the `readName` integration task unless a regression is found. Before moving into another implementation area, choose the next task from Kostis's remaining scope and consult only the documentation relevant to that task.

**## Planned / not yet implemented or verified**

\- **\*\*Session runtime — Spyros:\*\*** implement \`session.Start\`, the \`chat.Destination\` adapter, registration, input framing and the 4,096-byte message limit, serialized output, separate history delivery, the 256-event live queue, write deadlines, and coordinated worker/socket/membership/capacity cleanup. \`session_test.go\` currently contains only its package declaration.

\- **\*\*Application integration:\*\*** create the real room and initialize the shared server with \`NewServer\` and the real starter. \`main\` currently uses \`server.Server{}\`, leaving its dependencies nil; a valid name reaches a nil starter call. End-to-end chat is therefore not complete.

\- **\*\*Remaining verification:\*\*** compiled CLI stdout/stderr/exit behavior and bind failures; concurrent room join/submit and send-order scenarios; session failure/replay/queue tests; integrated multi-client chat and continued operation after departures; actual different-computer LAN checks; and recorded functional audit outcomes. Component tests do not establish these results.

\- **\*\*Documentation maintenance:\*\*** refresh stale README/docs statements that implementation or \`go.mod\` is absent. Existing instructions and audit plans are not evidence that their scenarios passed.

\- **\*\*Bonuses:\*\*** extra flags and activity/file logging (Kostis), rename/multiple rooms (Aris), and the \`gocui\` client (Spyros). No implementations are present; behavior/API decisions remain to be agreed after required integration.

\- **\*\*External checks:\*\*** evaluator compatibility with Go \`1.26.2\` and standard-library test helpers remains unverified.

**## Verification record**

\| Date | Check | Observed result |

\| --- | --- | --- |

\| 2026-09-18 | \`go test -race ./... -count=1\` — earlier baseline | Previously recorded as passed for root, chat, and server, including loopback TCP admission tests. Session reported \`[no tests to run]\`. Local socket access was required. This predates the new helper and was not rerun in this session. |

\| 2026-09-18 | Earlier mentoring session — source review | Confirmed the saved partial \`ReadSlice\`-based \`readName\` helper and that admission still calls \`ReadString\`. No tests added or run in that session. |

\| 2026-09-18 | Full application / different-computer audit | Not verified; no completed audit evidence recorded. |

\| 2026-09-21 | Initial source review, before direct tests | Reviewed the saved \`ReadRune\`-based helper, Unicode whitespace classification, capped storage, malformed-byte recovery, EOF handling, and reader reuse. No correctness defect found by inspection; helper integration and direct runtime verification were still outstanding at this point. |

\| 2026-09-21 | \`gofmt -d internal/server/server.go\` | No output; no formatting differences reported. |

\| 2026-09-21 | \`go test ./internal/server -count=1\` — before direct tests | Passed: \`ok net-cat/internal/server 0.017s\`. The then-existing tests did not call \`readName\`, so this verified existing server behavior and compilation, not the new helper's runtime behavior or bounded retention. No tests had been added yet. |

\| 2026-09-21 | Direct helper tests, after explicit test-writing request | \`go test ./internal/server -run 'Test(ReadName\\|IsNameSpace)' -count=1 -timeout=30s -coverprofile=/tmp/net-cat-name-coverage.out\` passed in 0.100s. Function coverage: \`readName\` 92.6%, \`isNameSpace\` 100%. |

\| 2026-09-21 | Full race suite after new tests | \`go test -race ./... -count=1 -timeout=60s\` passed for root, chat, server, and session; session reported \`[no tests to run]\`. Admission still uses its original reading path. |

\| 2026-09-21 | Allocation benchmark after new tests | \`go test ./internal/server -run '^$' -bench '^BenchmarkReadNameLongInput$' -benchmem -benchtime=3x -count=1 -timeout=60s\` passed. ASCII: 4,466–4,472 B/op and 11 allocs/op; malformed bytes: 4,408 B/op and 8 allocs/op; surrounding whitespace: 4,498 B/op and 12 allocs/op across the three repetition counts. |

\| 2026-09-21 | Admission integration | `HandleConnection` now uses `readName(reader)` instead of `ReadString`, uses the returned `tooLong` flag for oversized-name rejection, removes the redundant second trim, and preserves the same buffered reader for session handoff. |

\| 2026-09-21 | `go test ./internal/server -count=1` — after integration | Passed: `ok net-cat/internal/server 0.102s`. The current server/admission and direct helper tests pass after integrating `readName`. |

\| 2026-09-21 | `go test -race ./... -count=1 -timeout=60s` — after integration | Passed for root, chat, server, and session; session reported `[no tests to run]`. No race was detected in the exercised paths. |

Scope follows [Kostis's tasks]\(tasks/kostis-tasks.md), [Aris's tasks]\(tasks/aris-tasks.md), [Spyros's tasks]\(tasks/spyros-tasks.md), [architecture]\(docs/architecture.md), and [agreed policies]\(docs/notes.md). Passing race checks covers exercised paths only.