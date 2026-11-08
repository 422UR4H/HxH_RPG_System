# Story Clock, Combat Close and End Match — Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every match a server-owned story clock (start, adjust, set, real-time play, limits, 1 s per counted Race round), extend `change_scene` into a combat close with a combat summary (rounds → seconds, hiatuses), and add `end_match`.

**Architecture:** A pure domain entity `clock.Clock` lives inside `MatchSession` and is mutated only under `r.mu`. Use cases stay thin; the `Room` persists ("one master command, one transaction") and broadcasts `clock_changed`. Clock state is a snapshot row in `match_clocks`; clock history goes to `match_events`.

**Tech Stack:** Go 1.23, PostgreSQL + goose, gorilla/websocket, huma (REST), standard `testing`.

**Spec:** `docs/superpowers/specs/2026-11-08-story-clock-design.pt-br.md` (Portuguese; read it whole first — §3 rules, §4 combat close, §5 end match, §6 data, §7 protocol). EN version: `…-design.md`.

## Global Constraints

- Work ONLY in the worktree `System_X_System/.worktrees/story-clock` on branch `feat/story-clock`. Before every commit run `git branch --show-current` and stop if it is not `feat/story-clock`. Only `git add <your files>` + `git commit`; never stash/checkout/reset.
- Language: identifiers, file names and test names in **English**; code comments in **Portuguese** (match the surrounding files); wire field names camelCase.
- Story time is **naive**: DB `TIMESTAMP` (never `TIMESTAMPTZ`), Go `time.Time` in UTC used as wall clock, wire string layout `"2006-01-02T15:04:05"` (no `Z`, no offset). Real instants (`game_end_at`, `running_since`) are `TIMESTAMPTZ` / RFC 3339.
- Clock resolution is whole seconds (`Truncate(time.Second)` everywhere a value is produced).
- Long skip threshold: `172800` seconds (2 days). Forward moves **strictly greater** than it need `confirmLongSkip: true`. Baseline = the clock **before** the command (without the automatic round second).
- Checkpoint interval while running: 60 s. History merge window for consecutive `adjust_clock`: 10 s.
- Close code for a finished match: `4002`, reason `"match ended"` (4001 stays "lobby not open").
- Migrations additive only (local Postgres is shared with other sessions). Run `make migrate-up` on the dev DB after Task 1 and say so in the PR.
- After every task touching `internal/`: `go vet ./... && go vet -tags=integration ./internal/gateway/pg/...`. Integration tests: `set -a && . ./.env && set +a && export TEST_DATABASE_URL="postgres://$PG_DB_USER:$PG_DB_PASS@localhost:5432/hxh_rpg_test?sslmode=disable"` then `go test -tags=integration -p 1 ./internal/gateway/pg/<pkg>/...`.
- NEVER remove TODO comments. Follow `.github/instructions/*.instructions.md` (game-server, gateway, integration tests).

## Review Focus

1. A rejected command must not mutate the clock: `change_scene`/`end_match` with `storyAt` below the minimum, or an unconfirmed long skip, must leave scene, round and clock exactly as they were (validate first, mutate after). Test in Task 4.
2. Clock running across a server restart must come back **paused at the last checkpoint**, never with hours of downtime added. Test in Task 5.
3. Naive time round-trip: a `storyAt` of `"2000-01-01T23:59:59"` written and read back (DB and wire) must not shift by the server's time zone. Tests in Tasks 2 and 3.
4. The master's tab being replaced by another tab of the same account (B4, `connection_replaced`) is NOT "master left": no auto-pause. Test in Task 6.
5. A turn open raises the limit only in memory; a restart mid-turn returns the limit to the last persisted value (no `turns.story_at`/limit written at open). Test in Task 3.

---

### Task 1: Schema, match fields, `IsFinished` guards, REST start time

**Files:**
- Create: `migrations/20261108000000_story_clock.sql`
- Modify: `internal/domain/match/match.go` (fields + `IsFinished`), `internal/gateway/pg/match/{read_match.go,create_match.go,update_match.go}`, `internal/application/match/{create_match.go,update_match.go,start_match.go,add_match_npc.go,remove_match_npc.go}`, `internal/application/enrollment/*.go` guards, `internal/application/campaign/update_campaign.go`, `internal/app/api/match/{create_match.go,update_match.go,get_match.go,match_summary_response.go}`, `docs/dev/api/match.md`
- Test: existing `*_test.go` next to each changed file; `internal/gateway/pg/match/match_integration_test.go`

**Interfaces:**
- Produces: `match.Match.StoryStartTime *time.Duration` (offset from midnight, whole seconds; nil = no time), `match.Match.GameEndAt *time.Time`, `func (m *Match) IsFinished() bool { return m.GameEndAt != nil }`. REST `storyStartTime` string `"HH:MM"` or `"HH:MM:SS"` (create: optional; update: optional pointer, `""` clears). GET match adds `storyStartTime` (`"HH:MM:SS"`, omitted when nil) and `gameEndAt` (RFC 3339, omitted when nil). Error `ErrInvalidStoryStartTime` → 422 with detail `"storyStartTime must be HH:MM or HH:MM:SS"`.

Migration (all `IF NOT EXISTS`, with a symmetric `-- +goose Down`):
- `matches`: `story_start_time TIME NULL`, `game_end_at TIMESTAMPTZ NULL`
- `match_clocks(match_uuid UUID PRIMARY KEY REFERENCES matches(uuid) ON DELETE CASCADE, story_at TIMESTAMP NULL, limit_at TIMESTAMP NULL, limit_reason VARCHAR(16) NOT NULL DEFAULT '', running_since TIMESTAMPTZ NULL, updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`
- `scenes`: `story_started_at TIMESTAMP NULL`, `story_finished_at TIMESTAMP NULL` (`brief_final_description` already exists)
- `rounds`: `counted BOOLEAN NOT NULL DEFAULT false`, `drained BOOLEAN NOT NULL DEFAULT false`
- `turns`: `story_at TIMESTAMP NULL`

- [ ] **Step 1:** Failing tests: `TestMatchIsFinished` (nil → false; set → true; `StoryEndAt` set but `GameEndAt` nil → false). Handler tests `TestCreateMatchStoryStartTime` (cases `"08:30"` → 8h30m stored, `"08:30:15"`, `""`/absent → nil, `"25:00"` → 422, `"8h"` → 422) and `TestUpdateMatchStoryStartTime` (`""` clears; refused after start like `storyStartAt` today). Gateway integration `TestMatchStoryStartTimeAndGameEndAt` (round trip of `story_start_time` = 08:30:15 and `game_end_at` nil).
- [ ] **Step 2:** Run; expect FAIL.
- [ ] **Step 3:** Implement. Every guard that today reads `StoryEndAt != nil` as "finished" switches to `IsFinished()` — find them with `grep -rn "StoryEndAt != nil\|StoryEndAt == nil" internal/application` and keep the same error values. Campaign `story_end_at` is a different concept: do not touch it. Write migration; run `make migrate-up` against the dev DB.
- [ ] **Step 4:** Run unit + `go test -tags=integration ./internal/gateway/pg/match/...`; PASS. Existing guard tests that set `StoryEndAt` to mean finished must now set `GameEndAt`.
- [ ] **Step 5:** Update `docs/dev/api/match.md` (new fields, error). Commit `feat(match): story start time, game_end_at and IsFinished guards`.

### Task 2: Domain clock entity

**Files:**
- Create: `internal/domain/match/entity/clock/clock.go`, `internal/domain/match/entity/clock/error.go`, `internal/domain/match/entity/clock/wire.go`
- Test: `internal/domain/match/entity/clock/clock_test.go`

**Interfaces:**
- Produces:
```go
type LimitReason string // "start" | "turn" | "scene" | "roundMode" | "round"
type PauseReason string // "manual" | "roundMode" | "scene" | "masterLeft" | "end" | "restart"
const LongSkip = 48 * time.Hour
type Snapshot struct { StoryAt *time.Time; Running bool; RunningSince *time.Time; LimitAt *time.Time; LimitReason LimitReason }
type Pause struct { From, To time.Time; Elapsed time.Duration; Reason PauseReason }
func NewUnset() *Clock
func NewAt(storyAt time.Time) *Clock                      // paused; limit = storyAt, reason start
func Reconstruct(s Snapshot) *Clock
func (c *Clock) IsSet() bool
func (c *Clock) Running() bool
func (c *Clock) Now(real time.Time) time.Time            // storyAt + (real − runningSince) when running; truncated to second. Panics never: unset returns zero time.
func (c *Clock) Limit() (time.Time, LimitReason)
func (c *Clock) Start(at time.Time) error                // ErrAlreadySet
func (c *Clock) Adjust(delta time.Duration, confirmLong bool, real time.Time) (from, to time.Time, err error)
func (c *Clock) Set(to time.Time, confirmLong bool, real time.Time) (from time.Time, err error)
func (c *Clock) Play(real time.Time) error               // ErrNotSet, ErrAlreadyRunning
func (c *Clock) Pause(reason PauseReason, real time.Time) *Pause   // nil when it was not running
func (c *Clock) RaiseLimit(reason LimitReason, real time.Time)     // limit = Now(real)
func (c *Clock) AddRoundSecond(real time.Time)                     // +1 s, then RaiseLimit(round)
func (c *Clock) Minimum(extra time.Duration, real time.Time) time.Time // max(limit, Now+extra)
func (c *Clock) CheckMove(to time.Time, minimum time.Time, confirmLong bool, real time.Time) error // shared validation, no mutation
func (c *Clock) Snapshot(real time.Time) Snapshot        // StoryAt = Now(real); RunningSince = real when running (a checkpoint)
func RoundEquivalents(from, to time.Time) int            // whole seconds; 1 round = 1 s (spec §1.4)
const WireLayout = "2006-01-02T15:04:05"
func FormatStory(t time.Time) string; func ParseStory(s string) (time.Time, error) // reject offsets/Z
```
- Errors (domain errors via `domain.NewDomainError`): `ErrNotSet`, `ErrAlreadySet`, `ErrBelowLimit`, `ErrLongSkipUnconfirmed`, `ErrZeroDelta`, `ErrAlreadyRunning`.

- [ ] **Step 1:** Failing table tests in `clock_test.go`, real time fixed at `R = 2026-11-08T12:00:00Z`, story at `S = 2000-01-01T08:00:00`:
  - `TestNowWhileRunning`: Play at R, Now(R+90s) == S+90s; Now(R+90.7s) == S+90s.
  - `TestAdjustRespectsLimit`: after RaiseLimit(turn) at S, Adjust(−1s) → `ErrBelowLimit`, clock unchanged; Adjust(+3600s) → to = 09:00:00.
  - `TestAdjustWhileRunningKeepsRunning`: Play at R; Adjust(+60s, R+30s) → Now(R+30s) == S+90s, Running() true.
  - `TestLongSkipNeedsConfirm`: Adjust(+172800s) OK; Adjust(+172801s) → `ErrLongSkipUnconfirmed`; same with confirm → OK. Same rule through `Set`.
  - `TestRewindRealTimeDownToLimit`: limit at S, Play, Pause at R+600s, Set(S+60s) OK; Set(S−1s) → `ErrBelowLimit`.
  - `TestPauseReturnsElapsed`: Play R, Pause(manual, R+192s) → From S, To S+192s, Elapsed 192s; second Pause → nil.
  - `TestAddRoundSecondRaisesLimit`: AddRoundSecond → Now == S+1s, Limit == (S+1s, round).
  - `TestMinimum`: limit S+5s, Now S+2s (after a rewind) → Minimum(1s) == S+5s; limit S, Now S+10s → Minimum(1s) == S+11s.
  - `TestStoryWireRoundTrip`: `FormatStory(ParseStory("2000-01-01T23:59:59"))` identical; `ParseStory("2000-01-01T23:59:59Z")` and `"…-03:00"` → error.
  - `TestRoundEquivalents`: 90 s apart → 90.
- [ ] **Step 2:** `go test ./internal/domain/match/entity/clock/` → FAIL.
- [ ] **Step 3:** Implement. Mutating methods validate first and mutate only on success.
- [ ] **Step 4:** PASS. Commit `feat(clock): domain story clock`.

### Task 3: Session wiring — counted/drained rounds, turn story time, limits

**Files:**
- Modify: `internal/domain/match/entity/round/round.go` (`counted`, `drained`, `MarkCounted()`, `IsCounted()`, `IsDrained()`, `ReconstructRound` gains both), `internal/domain/match/entity/turn/turn.go` (`storyAt *time.Time`, `SetStoryAt`, `GetStoryAt`), `internal/domain/match/entity/scene/scene.go` (`BriefFinalDescription *string`, `storyStartedAt`, `storyFinishedAt` with getters/setters), `internal/domain/match/matchsession/match_session.go`
- Modify persistence of these fields: `internal/gateway/pg/round/{ensure_scene_and_round.go,persist_turn_close.go,persist_round_close.go,find_active_session.go}` (+ `matchsession.ActiveSessionData` fields)
- Test: `internal/domain/match/matchsession/clock_session_test.go`, `internal/gateway/pg/round/round_integration_test.go`

**Interfaces:**
- Consumes: Task 2 `clock.Clock`.
- Produces on `MatchSession`: `Clock() *clock.Clock`, `SetClock(c *clock.Clock)` (used by Init), `ClockSnapshot(real time.Time) clock.Snapshot`. Session uses an injectable `now func() time.Time` (default `time.Now`; tests set it), exposed as `SetNowFunc`.
- Behavior (all under the caller's `r.mu`):
  - `OpenNextAction`/`PullAction`: refuse with `clock.ErrNotSet` when the clock is unset (before any mutation). When a turn opens: `turn.SetStoryAt(clock.Now)`; if the round's mode is Race → `round.MarkCounted()`; `clock.RaiseLimit(turn)`. When a turn closes: `clock.RaiseLimit(turn)`.
  - `CloseRound()`: `drained = activeQueue is empty` stored on the closing round; if the closing round `IsCounted()` → `clock.AddRoundSecond`. The new round is not counted.
  - Persistence writes `rounds.counted/drained`, `turns.story_at` (the value set at open, written only at close — nothing at open), and the clock snapshot: add `Clock *clock.Snapshot` to `appmatch.TurnCloseData` and a `clk *clock.Snapshot` parameter to `PersistRoundClose`; both upsert `match_clocks` inside their transaction. Create the shared upsert in a new `internal/gateway/pg/clock` package (`UpsertTx(ctx, tx, matchUUID, snap)` + `Load(ctx, matchUUID) (*clock.Snapshot, error)`), used by round and by later tasks.
- [ ] **Step 1:** Failing tests: `TestOpenTurnStampsStoryTimeAndRaisesLimit`, `TestRaceRoundCountsOneSecond` (Race, open+close one action, queue empties → `CloseRound` → Now == S+1s, round drained=true), `TestRoundWithActionsLeftIsNotDrained` (an action that cannot pay stays queued → drained=false), `TestFreeRoundDoesNotCount`, `TestModeSwitchMidRoundStillCounts` (turn opened in Race, mode → Free, close → +1 s), `TestOpenRefusedWhenClockUnset`. Integration: `TestPersistTurnCloseWritesClockAndStoryAt`, `TestPersistRoundCloseWritesCountedDrained`, `TestClockSnapshotNaiveRoundTrip` (23:59:59 survives with the DB session in `America/Belem`).
- [ ] **Step 2:** FAIL. **Step 3:** Implement; `room.go` call sites that build `TurnCloseData`/call `PersistRoundClose` pass `session.ClockSnapshot(now)` taken under `r.mu` in the same critical section as the other copies. **Step 4:** unit + `go test -tags=integration ./internal/gateway/pg/round/... ./internal/gateway/pg/clock/...` PASS; `go test ./internal/app/game/...` PASS (fix fakes).
- [ ] **Step 5:** Commit `feat(match): rounds count seconds, turns carry story time`.

### Task 4: Session — scene change, mode change and match end move the clock

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`, `internal/application/match/{change_scene.go,change_round_mode.go,error.go}`; create `internal/application/match/end_match.go`
- Test: `internal/domain/match/matchsession/clock_session_test.go`, `internal/application/match/end_match_test.go`, existing change_scene/change_round_mode tests

**Interfaces:**
- Produces:
```go
type SceneClose struct { BriefFinalDescription *string; StoryAt *time.Time; ConfirmLongSkip bool }
func (s *MatchSession) ChangeScene(category enum.SceneCategory, briefInit string, close SceneClose) (oldScene *scene.Scene, oldRound *round.Round, pause *clock.Pause, err error)
func (s *MatchSession) SetRoundMode(mode enum.RoundMode) (*clock.Pause, error) // ErrNotSet when unset; pauses (reason roundMode) and RaiseLimit(roundMode) only when the mode actually changes
func (s *MatchSession) EndMatch(storyAt time.Time, confirmLong bool) (*MatchEnd, error)
type MatchEnd struct { Scene *scene.Scene; Round *round.Round; Pause *clock.Pause; StoryAt time.Time }
var ErrMatchInBattle // "cannot end the match during a battle scene"
```
- `ChangeScene` order (spec §4): unset → `ErrNotSet`; open turn → `ErrRoundHasOpenTurn`; `minimum = clock.Minimum(1s if activeRound.IsCounted() else 0)`; if `StoryAt` given → `CheckMove(*StoryAt, minimum, confirm)` (baseline for long skip = `Now` before). Only then: pause (scene), `CloseRound`-equivalent (+1 s when counted), move clock to `max(StoryAt, now)`, `RaiseLimit(scene)`, stamp old scene `storyFinishedAt` and final description (trimmed, ≤255 else `ErrBriefFinalTooLong`), new scene `storyStartedAt`.
- `EndMatch`: unset → `ErrNotSet`; active scene `battle` → `ErrMatchInBattle`; open turn → `ErrRoundHasOpenTurn`; same minimum/CheckMove; then pause (end), close round (+1 s when counted) and scene, clock to `storyAt`, discard queue. Use case `EndMatchUC.Execute(ctx, session, masterUUID, callerUUID, storyAt, briefFinal *string, confirm) (*matchsession.MatchEnd, error)` — master check, ≤255 check, delegates.
- [ ] **Step 1:** Failing tests: `TestChangeSceneRejectsBelowMinimumWithoutMutating` (scene id, round id, Now, limit all unchanged after the error), `TestChangeSceneAddsActiveRoundSecond`, `TestChangeScenePausesRealTime`, `TestChangeSceneLongSkipNeedsConfirm`, `TestSetRoundModePausesAndRaisesLimit`, `TestSetRoundModeSameModeNoPause`, `TestEndMatchRefusedInBattle`, `TestEndMatchRefusedWithOpenTurn`, `TestEndMatchClosesAndFreezes`, `TestEndMatchUCRejectsNonMaster`.
- [ ] **Step 2:** FAIL. **Step 3:** Implement; adapt room call sites to the new signatures (keep their behavior; the pause/broadcast wiring is Task 6/7). **Step 4:** `go test ./internal/...` PASS. **Step 5:** Commit `feat(match): scene, regime and match end move the clock`.

### Task 5: Session init loads or creates the clock

**Files:**
- Modify: `internal/application/match/init_match_session.go`, `internal/application/match/i_repository.go` (new `IClockRepository`), `cmd/game/main.go` (wiring), `internal/gateway/pg/clock/repository.go`
- Test: `internal/application/match/init_match_session_test.go`, `internal/gateway/pg/clock/clock_integration_test.go`

**Interfaces:**
- Produces: `type IClockRepository interface { Load(ctx, matchUUID) (*clock.Snapshot, error); Save(ctx, matchUUID, snap clock.Snapshot, ev *ClockEvent) error }` where `ClockEvent struct { Event matchevent.Event; MergeInto uuid.UUID }` (MergeInto non-nil → UPDATE that event's payload instead of INSERT). `NewInitMatchSessionUC(..., clocks IClockRepository)` (nil allowed: unset clock).
- Rules: no row → if `match.StoryStartTime != nil` → `NewAt(StoryStartAt + *StoryStartTime)` and `Save`; else `NewUnset()` and `Save`. Row with `RunningSince != nil` → reconstruct **paused at `StoryAt`** (the last checkpoint), `Save`, and log `clock of match X was running at restart: paused at <t>`.
- [ ] **Step 1:** Failing tests: `TestInitCreatesClockFromStartTime`, `TestInitCreatesUnsetClockWithoutTime`, `TestInitPausesRunningClockAtCheckpoint` (row running_since = 3 h ago, story 08:00 → session clock 08:00 paused), `TestInitKeepsExistingClock`. Integration `TestClockRepositorySaveLoadAndMerge` (insert event, then MergeInto updates the same row; `ListByMatch` returns one event).
- [ ] **Step 2–4:** FAIL → implement → PASS. **Step 5:** Commit `feat(clock): load or create the clock when the session starts`.

### Task 6: WS clock commands, state and broadcasts

**Files:**
- Modify: `internal/app/game/message.go`, `internal/app/game/room.go`, `internal/app/game/room_deps.go`, `cmd/game/main.go`; create `internal/app/game/clock.go` (handlers + helpers, keep room.go from growing), `internal/application/match/clock_commands.go` (thin UCs: master check + delegate)
- Test: `internal/app/game/clock_e2e_test.go`

**Interfaces:**
- Consumes: Tasks 2–5. `RoomDeps.ClockRepo IClockRepository` (nil = no persistence, the room still works in memory).
- Messages (constants `MsgType…`): in `set_clock_start {time:"HH:MM[:SS]"}`, `adjust_clock {deltaSeconds:int, confirmLongSkip?:bool}`, `set_clock {storyAt, confirmLongSkip?}`, `play_clock {}`, `pause_clock {}`; out `clock_changed {clock: ClockPayload, change: ClockChangePayload}`.
```go
type ClockPayload struct { StoryAt *string `json:"storyAt"`; Running bool `json:"running"`; LimitAt *string `json:"limitAt"`; LimitReason string `json:"limitReason"` }
type ClockChangePayload struct { Kind string `json:"kind"`; From *string `json:"from,omitempty"`; To *string `json:"to,omitempty"`; DeltaSeconds int64 `json:"deltaSeconds,omitempty"`; Reason string `json:"reason,omitempty"` }
```
  `change.kind` ∈ `start|adjust|set|play|pause|round|scene|end|limit`. `MatchFullStatePayload.Clock *ClockPayload` (`json:"clock,omitempty"`, absent only before a session exists).
- Error codes (`NewErrorMessage(code, msg)`): `clock_not_set`, `clock_already_set`, `clock_below_limit`, `clock_long_skip_unconfirmed`, `clock_play_in_race`, `clock_already_running`, `invalid_payload`, `forbidden`, `match_not_started`. Map domain errors in one helper `clockErrorCode(err) string`. Also map `clock.ErrNotSet` from `open_next_action`, `pull_action`, `change_round_mode`, `change_scene`, `end_match` to `clock_not_set` (today they would say `game_error`).
- Room rules: every clock mutation happens under `r.mu` with the session; snapshot under the same lock; then persist (`ClockRepo.Save` with the event, own transaction — needs the active scene/round rows: call `ensureSceneAndRoundRows` first), then broadcast to the table. Events: `clockStarted {at}`, `clockAdjusted {from,to,deltaSeconds}` (merge: the room keeps `lastAdjust {eventUUID, at time.Time}`; a new adjust within 10 s with no other clock event or turn/scene/mode change in between → `MergeInto`, `from` kept, `to`/`deltaSeconds` accumulated), `clockRealTime {from,to,elapsedSeconds,reason}` written on every pause. `set_clock` writes a `clockAdjusted` too (delta = to − from).
- `play_clock` in Race → `clock_play_in_race`. Broadcast `kind: limit` after a turn opens (the limit moved) — no event. After `round_closed`, broadcast `kind: round` with from/to.
- Checkpoint: a `time.Ticker` (60 s) in `Run`'s select; when the clock is running → `ClockRepo.Save` (no event, no broadcast).
- Master leaves: in the `unregister` branch, when the removed client is the master **and** it was not replaced (B4 — check how `removeClient` distinguishes; a replaced connection must not pause) and the clock is running → pause(masterLeft), persist with event, broadcast to remaining clients (before the empty-room return).
- [ ] **Step 1:** Failing e2e tests (pattern of `regency_e2e_test.go`): `TestClockStartSetsTimeOnce` (second → `clock_already_set`), `TestAdjustClockBroadcastsToTable` (player receives `clock_changed` kind adjust with from/to), `TestAdjustBelowLimitRefused`, `TestLongSkipRequiresConfirm`, `TestPlayRefusedInRace`, `TestModeChangeToRacePausesClock` (player gets kind pause reason roundMode), `TestMasterLeavingPausesClock`, `TestReplacedMasterTabDoesNotPause`, `TestOpenNextActionRefusedWhenClockUnset` (code `clock_not_set`), `TestMatchFullStateCarriesClock`, `TestConsecutiveAdjustsMergeInHistory` (two adjusts 3 s apart → one `clockAdjusted` event with summed delta), `TestNonMasterClockCommandForbidden`.
- [ ] **Step 2–4:** FAIL → implement → `go test ./internal/app/game/...` PASS (whole package, it is large — run it once at the end of the task).
- [ ] **Step 5:** Commit `feat(game): clock commands and clock_changed`.

### Task 7: Combat close — `get_combat_summary`, `change_scene` fields, history

**Files:**
- Modify: `internal/app/game/message.go` (`ChangeScenePayload` gains `briefFinalDescription *string`, `storyAt *string`, `confirmLongSkip bool`), `internal/app/game/clock.go`/`room.go` (change_scene arm), `internal/application/match/get_match_history.go` + `i_repository.go` (`HistoryScene.BriefFinalDescription/StoryStartedAt/StoryFinishedAt`, `HistoryRound.Counted/Drained`, `HistoryTurn.StoryAt`, new `HistoryEvent` kinds `clockStarted|clockAdjusted|clockRealTime`), `internal/gateway/pg/round/find_match_history.go`, REST history mapping in `internal/app/api/match/`
- Create: `internal/application/match/combat_summary.go`, `internal/gateway/pg/round/combat_summary.go`
- Test: `internal/application/match/combat_summary_test.go`, `internal/gateway/pg/round/round_integration_test.go`, `internal/app/game/combat_close_e2e_test.go`

**Interfaces:**
- Produces: `combat_summary` payload
```go
type CombatSummaryPayload struct { SceneID uuid.UUID `json:"sceneId"`; CountedRounds int `json:"countedRounds"`; CountedSeconds int `json:"countedSeconds"`; ActiveRoundCounts bool `json:"activeRoundCounts"`; MinimumStoryAt string `json:"minimumStoryAt"`; Hiatuses []HiatusPayload `json:"hiatuses"` }
type HiatusPayload struct { Before HiatusTurn `json:"before"`; After HiatusTurn `json:"after"` }
type HiatusTurn struct { RoundNumber int `json:"roundNumber"`; TurnID uuid.UUID `json:"turnId"`; ActorID uuid.UUID `json:"actorId"`; ActionKind string `json:"actionKind"` }
```
  `CountedRounds`/`CountedSeconds` = closed counted rounds of the scene (already on the clock); `ActiveRoundCounts` = the active round is counted (its second is added at close); `MinimumStoryAt` = `clock.Minimum(1 s if ActiveRoundCounts)`. `RoundNumber` = 1-based position of the round in the scene by `created_at`. `ActionKind` = the same discriminator the history/turn payloads already use for an action's type (reuse it, do not invent one). Hiatus = a **Race** round with `drained = true` whose next round in the scene has a turn (spec §3.6); before = its last turn, after = the next round's first turn. The active round takes part with its in-memory turns.
- Pure function `BuildCombatSummary(rounds []SummaryRound, active SummaryRound, minimum time.Time) CombatSummary` in `combat_summary.go` (no I/O) + gateway `ListSceneRoundsForSummary(ctx, sceneUUID) ([]SummaryRound, error)`.
- `get_combat_summary`: master only; active scene not battle → `not_battle_scene`; answer only to the sender.
- `change_scene`: passes `SceneClose`; maps errors (`clock_below_limit`, `clock_long_skip_unconfirmed`, `clock_not_set`, brief too long → `invalid_payload`); persists old scene's `brief_final_description`/`story_finished_at`, new scene's `story_started_at` and the clock snapshot in the same write path (extend `CloseSceneAndRound` with those fields + the snapshot in one transaction, or make a `PersistSceneChange`); broadcasts `scene_changed` then `clock_changed` (kind `scene`, plus a `pause` change first when real time was running, with its `clockRealTime` event).
- [ ] **Step 1:** Failing tests: `TestBuildCombatSummaryCountsAndHiatuses` (rounds: R1 Race counted drained=false; R2 Race counted drained=true with turns T3,T4; R3 Race counted with T5 → one hiatus before T4 (round 2) after T5 (round 3); CountedSeconds 3), `TestDrainedLastRoundIsNotHiatus`, `TestFreeDrainedRoundIsNotHiatus`, `TestActiveRoundJoinsSummary`. Integration `TestListSceneRoundsForSummary`, `TestHistoryCarriesClockFields`. E2E `TestCombatSummaryOnlyInBattle`, `TestChangeSceneClosesCombatWithStoryAt` (clock = given storyAt; history scene has final description), `TestChangeSceneBelowMinimumRefused`.
- [ ] **Step 2–4:** FAIL → implement → PASS. **Step 5:** Commit `feat(match): combat close with summary and final description`.

### Task 8: `end_match`, close code 4002, REST end fields

**Files:**
- Modify: `internal/app/game/{message.go,room.go,handler.go,clock.go}`, `internal/gateway/pg/match/` (new `end_match.go`: one transaction — `matches.game_end_at/story_end_at/brief_final_description`, close round and scene with story times, clock snapshot, `clockRealTime` event if paused, and `UPDATE campaigns SET story_current_at = GREATEST(COALESCE(story_current_at, $t), $t)` for the match's campaign, `$t` = the final story instant — never regresses, spec §5), `internal/application/match/i_repository.go` (`PersistMatchEnd`), `internal/app/api/match/get_match.go` (`storyEndTime` "HH:MM:SS" from `match_clocks.story_at` when finished)
- Test: `internal/app/game/end_match_e2e_test.go`, `internal/gateway/pg/match/match_integration_test.go`, `internal/app/api/match/get_match_test.go`

**Interfaces:**
- In: `end_match {storyAt, briefFinalDescription?, confirmLongSkip?}`. Out: `match_ended {matchId, storyAt, gameEndAt}` (whole table), then every connection is closed with `websocket.FormatCloseMessage(4002, "match ended")` and the room stops (`RoomStateClosed`; no dropped-turn log since there is no open turn).
- Errors: `match_in_battle`, `game_error` (open turn, same text as change_scene), `clock_not_set`, `clock_below_limit`, `clock_long_skip_unconfirmed`, `invalid_payload` (brief > 255), `forbidden`.
- Handler: on connect, if the match `IsFinished()` → write `match_ended` (same payload, from the DB) and close with 4002 (mirror `rejectLobbyNotOpen`, including the drain goroutine), for master and players, lobby and game paths alike.
- Persist failure: answer the master `game_error` "could not end the match", keep the room running and the session untouched (end is only applied in memory after the write succeeds: call `session.EndMatch` on a dry-run first? — no: compute in the session, snapshot, write, and on failure rebuild is not possible; therefore validate with `CheckMove` and preconditions first, write the end, and only then apply `EndMatch` to the session).
- [ ] **Step 1:** Failing tests: `TestEndMatchBroadcastsAndCloses4002` (player gets `match_ended` then close code 4002), `TestEndMatchRefusedInBattleScene`, `TestReconnectToFinishedMatchGets4002`, integration `TestPersistMatchEnd` (game_end_at set, story_end_at = date of storyAt, scene/round closed, clock = storyAt paused, campaign story_current_at = storyAt), `TestPersistMatchEndNeverRegressesCampaign` (campaign already at a later instant → unchanged), `TestGetMatchFinishedHasStoryEndTime`.
- [ ] **Step 2–4:** FAIL → implement → PASS. **Step 5:** Commit `feat(match): end match over the table socket`.

### Task 9: Contracts and docs

**Files:**
- Create: `docs/dev/api/match-clock.md` (every message of Tasks 6–8: direction, who, payload JSON examples with naive times, triggers, errors, persistence, reconnect behavior, the naive-time warning for the front, the 4002 code), `docs/game/tempo.md` (rule for players/masters: the story clock, 1 round = 1 s, hiatuses, real time, limits; PT-BR, no code; developer footer), `docs/dev/match/clock.md` (design rationale: where the clock lives, transactions, merge, checkpoints)
- Modify: `docs/dev/api/match-combat-ws.md` (index; `change_scene` new fields; `match_full_state.clock`; error catalogue; §9 restart table row for the clock), `docs/dev/api/match-history.md` (new fields and event kinds), `docs/dev/api/game-lobby.md` (4002), `docs/documentation-map.yaml` (register new docs and map `internal/domain/match/entity/clock/`, `internal/gateway/pg/clock/`), `docs/game/cenas-e-turnos.md` link, `AGENTS.md` Known Issues one paragraph, `.github/instructions/domain-map.instructions.md` row for clock
- [ ] **Step 1:** Write docs. **Step 2:** Check every message/field in code appears in `match-clock.md` (`grep -o '"[a-z_]*"' internal/app/game/message.go` for the new types). **Step 3:** Commit `docs(clock): contract, game rule and dev notes`.

### Task 10: Final verification

- [ ] `go vet ./... && go vet -tags=integration ./... && go vet -tags=smoke ./...`
- [ ] `go test ./...` and `go test -tags=integration -p 1 ./internal/gateway/pg/...` PASS
- [ ] `golangci-lint cache clean && ~/go/bin/golangci-lint run --max-same-issues 0 --max-issues-per-linter 0` → 0 issues
- [ ] Smoke on own ports (API `SERVER_ADDR=localhost:5001`, game `GAME_SERVER_ADDR=localhost:8082`, from this worktree): `curl` create/update/get match with `storyStartTime`; a Node `ws` script as master + player (seed accounts, `docs/dev/seed.md`) covering set_clock_start → adjust → play/pause → Race round +1 s → get_combat_summary → change_scene with storyAt → end_match → reconnect gets 4002. Save the script under the session scratchpad, not the repo. Kill the processes you started.
