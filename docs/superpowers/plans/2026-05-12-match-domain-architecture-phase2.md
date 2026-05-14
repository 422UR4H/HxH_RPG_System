# Match Domain Architecture — Phase 2: MatchSession + Use Cases + Room Integration

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `MatchSession` (in-memory game state), the six in-session use cases, and wire them into the WebSocket `Room` so the game engine becomes operational over WebSocket.

**Architecture:** `MatchSession` lives in `internal/domain/match/matchsession/` and holds the active `Round`, the action queue, and the character sheet cache. Six use cases in `internal/application/match/` orchestrate it. The `Room` in `internal/app/game/` holds the session and dispatches incoming WebSocket messages to the right use case. Turn/Round persistence is deferred to Phase 3.

**Tech Stack:** Go 1.23, `testing` standard package, table-driven tests with `t.Run()`, external test packages (`package <pkg>_test`).

> **Scope:** Phase 2 of `docs/superpowers/specs/2026-05-11-match-domain-architecture-design.md`.
> **Phase 3 (separate plan):** Turn/Round DB persistence (INSERT on CloseTurn), Scene management, Initiative.

---

## Key Types Reference

Keep this open while working. Every task in this plan uses these types.

| Symbol | Package import alias | Full path |
|--------|---------------------|-----------|
| `*round.Round` | `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"` | — |
| `*turn.Turn` | `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"` | — |
| `action.Action` / `*action.PriorityQueue` | `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"` | — |
| `service.RoundOrchestrator` / `service.CombatResolver` / `*service.TurnResolution` | `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"` | — |
| `*match.Participant` | `"github.com/422UR4H/HxH_RPG_System/internal/domain/match"` | — |
| `*csSheet.CharacterSheet` | `csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"` | — |
| `csEntity.Summary` | `csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"` | — |
| `enum.Free` / `enum.Race` | `"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"` | — |
| `*matchsession.MatchSession` | `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"` | — |

---

## File Map

### Created by this plan

| File | Package | Responsibility |
|------|---------|----------------|
| `internal/domain/match/matchsession/error.go` | `matchsession` | Session-level domain errors |
| `internal/domain/match/matchsession/match_session.go` | `matchsession` | MatchSession struct + all methods |
| `internal/domain/match/matchsession/match_session_test.go` | `matchsession_test` | Unit tests (no mocks — pure in-memory) |
| `internal/application/match/i_session_loader.go` | `match` | `ICharSheetLoader` interface |
| `internal/application/match/init_match_session.go` | `match` | `InitMatchSessionUC` + `IInitMatchSession` interface |
| `internal/application/match/open_next_action.go` | `match` | `OpenNextActionUC` + result type + interface |
| `internal/application/match/pull_action.go` | `match` | `PullActionUC` + interface |
| `internal/application/match/enqueue_action.go` | `match` | `EnqueueActionUC` + interface |
| `internal/application/match/attach_reaction.go` | `match` | `AttachReactionUC` + interface |
| `internal/application/match/close_turn.go` | `match` | `CloseTurnUC` + interface |
| `internal/application/match/close_round.go` | `match` | `CloseRoundUC` + interface |

### Modified by this plan

| File | Change |
|------|--------|
| `internal/app/game/message.go` | Add 10 new message type constants + 8 new payload structs |
| `internal/app/game/room.go` | Add `session` field + 6 new UC interfaces + updated constructor + new handlers |
| `internal/app/game/hub.go` | Update `GetOrCreateRoom` signature |
| `internal/app/game/handler.go` | Inject 6 new UCs |
| `internal/app/game/game_test.go` | Update mocks for new constructor signature |
| `AGENTS.md` | Update Known Issues |

---

## Task 1 — Baseline Verification

**Files:** no changes

- [ ] **Step 1: Verify clean build**

```bash
go build ./...
```

Expected: zero errors.

- [ ] **Step 2: Verify tests pass**

```bash
go test ./...
```

Expected: all pass.

---

## Task 2 — MatchSession: struct, constructor, and errors (TDD)

**Files:**
- Create: `internal/domain/match/matchsession/error.go`
- Create: `internal/domain/match/matchsession/match_session.go`
- Create: `internal/domain/match/matchsession/match_session_test.go`

- [ ] **Step 1: Create `error.go`**

```go
// internal/domain/match/matchsession/error.go
package matchsession

import "errors"

var (
	ErrParticipantNotFound = errors.New("participant not found in match session")
	ErrActionActorMismatch = errors.New("action actor does not match player")
	ErrRoundHasOpenTurn    = errors.New("cannot close round: current turn is still open")
	ErrCharSheetNotFound   = errors.New("character sheet not found in session")
)
```

- [ ] **Step 2: Write the failing constructor test**

Create `internal/domain/match/matchsession/match_session_test.go`:

```go
package matchsession_test

import (
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

func TestNewMatchSession(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()

	participant := makeParticipant(matchUUID, &playerUUID)
	sheet := &csSheet.CharacterSheet{}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{playerUUID: sheet}

	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})

	if s == nil {
		t.Fatal("expected non-nil MatchSession")
	}
	if s.GetActiveRound() == nil {
		t.Error("expected non-nil activeRound on new session")
	}
	if s.GetActiveRound().GetMode() != enum.Free {
		t.Error("expected initial round mode to be Free")
	}
}

func TestMatchSession_GetCharSheet(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	sheet := &csSheet.CharacterSheet{}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{playerUUID: sheet}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})

	t.Run("returns sheet for known player", func(t *testing.T) {
		got, err := s.GetCharSheet(playerUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != sheet {
			t.Error("expected same sheet pointer")
		}
	})

	t.Run("returns ErrCharSheetNotFound for unknown player", func(t *testing.T) {
		_, err := s.GetCharSheet(uuid.New())
		if err != matchsession.ErrCharSheetNotFound {
			t.Errorf("expected ErrCharSheetNotFound, got %v", err)
		}
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func makeParticipant(matchUUID uuid.UUID, playerUUID *uuid.UUID) *match.Participant {
	return &match.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet: csEntity.Summary{
			UUID:       uuid.New(),
			PlayerUUID: playerUUID,
		},
	}
}
```

- [ ] **Step 3: Run to confirm failure**

```bash
go test ./internal/domain/match/matchsession/... -v
```

Expected: FAIL — `matchsession` package not found.

- [ ] **Step 4: Implement `match_session.go`**

```go
// internal/domain/match/matchsession/match_session.go
package matchsession

import (
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type MatchSession struct {
	matchUUID    uuid.UUID
	activeRound  *round.Round
	activeQueue  action.PriorityQueue
	charSheets   map[uuid.UUID]*csSheet.CharacterSheet // keyed by playerUUID
	participants map[uuid.UUID]*match.Participant       // keyed by playerUUID
	roundOrch    service.RoundOrchestrator
	combatRes    service.CombatResolver
}

// NewMatchSession creates an in-memory game session for the given match.
// charSheets is keyed by playerUUID. participants is the list from the DB.
// Participants without a playerUUID (NPC sheets) are omitted from the map.
func NewMatchSession(
	matchUUID uuid.UUID,
	charSheets map[uuid.UUID]*csSheet.CharacterSheet,
	participants []*match.Participant,
) *MatchSession {
	pMap := make(map[uuid.UUID]*match.Participant, len(participants))
	for _, p := range participants {
		if p.Sheet.PlayerUUID != nil {
			pMap[*p.Sheet.PlayerUUID] = p
		}
	}
	return &MatchSession{
		matchUUID:    matchUUID,
		activeRound:  round.NewRound(enum.Free),
		activeQueue:  action.NewActionPriorityQueue(nil),
		charSheets:   charSheets,
		participants: pMap,
		roundOrch:    service.RoundOrchestrator{},
		combatRes:    service.CombatResolver{},
	}
}

func (s *MatchSession) GetActiveRound() *round.Round { return s.activeRound }

func (s *MatchSession) GetCurrentTurn() *round.Round {
	// Intentionally returns the round — callers use GetActiveRound().CurrentTurn()
	return s.activeRound
}

func (s *MatchSession) GetCharSheet(playerUUID uuid.UUID) (*csSheet.CharacterSheet, error) {
	sheet, ok := s.charSheets[playerUUID]
	if !ok {
		return nil, ErrCharSheetNotFound
	}
	return sheet, nil
}
```

> **Note on GetCurrentTurn:** The spec describes it as returning `*turn.Turn` via `activeRound.CurrentTurn()`. In this implementation, callers access it as `session.GetActiveRound().CurrentTurn()` to avoid duplicating the delegating method. Add `GetCurrentTurn() *turn.Turn` if a caller in Phase 2 needs it directly.

- [ ] **Step 5: Run tests**

```bash
go test ./internal/domain/match/matchsession/... -v
```

Expected: all PASS.

- [ ] **Step 6: Full build**

```bash
go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/domain/match/matchsession/
git commit -m "$(cat <<'EOF'
feat: add MatchSession struct, constructor and read accessors

Initializes with a Free-mode Round and empty action queue.
charSheets and participants keyed by playerUUID for O(1) lookup.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 3 — MatchSession: EnqueueAction (TDD)

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Modify: `internal/domain/match/matchsession/match_session_test.go`

- [ ] **Step 1: Write failing tests**

Append to `match_session_test.go`:

```go
func TestMatchSession_EnqueueAction(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	t.Run("enqueues action for known participant", func(t *testing.T) {
		a := makeAction(playerUUID)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrParticipantNotFound for unknown player", func(t *testing.T) {
		a := makeAction(uuid.New())
		err := s.EnqueueAction(uuid.New(), a)
		if err != matchsession.ErrParticipantNotFound {
			t.Errorf("expected ErrParticipantNotFound, got %v", err)
		}
	})

	t.Run("returns ErrActionActorMismatch when actorID does not match playerUUID", func(t *testing.T) {
		a := makeAction(uuid.New()) // actorID is a different UUID
		err := s.EnqueueAction(playerUUID, a)
		if err != matchsession.ErrActionActorMismatch {
			t.Errorf("expected ErrActionActorMismatch, got %v", err)
		}
	})
}

func makeAction(actorID uuid.UUID) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
}
```

Also add `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"` to the test imports.

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/domain/match/matchsession/... -run TestMatchSession_EnqueueAction -v
```

Expected: FAIL — `EnqueueAction` undefined.

- [ ] **Step 3: Add `EnqueueAction` to `match_session.go`**

```go
// EnqueueAction adds a player's action to the priority queue.
// playerUUID must be a known participant and must match a.GetActorID().
func (s *MatchSession) EnqueueAction(playerUUID uuid.UUID, a *action.Action) error {
	if _, ok := s.participants[playerUUID]; !ok {
		return ErrParticipantNotFound
	}
	if a.GetActorID() != playerUUID {
		return ErrActionActorMismatch
	}
	s.activeQueue.Insert(a)
	return nil
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/domain/match/matchsession/... -v
```

Expected: all PASS.

- [ ] **Step 5: Full build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/matchsession/
git commit -m "$(cat <<'EOF'
feat(matchsession): add EnqueueAction with participant and actor validation

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 4 — MatchSession: OpenNextAction + PullAction (TDD)

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Modify: `internal/domain/match/matchsession/match_session_test.go`

- [ ] **Step 1: Write failing tests**

Append to `match_session_test.go`:

```go
func TestMatchSession_OpenNextAction(t *testing.T) {
	t.Run("opens Turn from highest-priority action in queue", func(t *testing.T) {
		s := emptySession()
		playerA := uuid.New()
		playerB := uuid.New()
		s2 := sessionWithParticipants(playerA, playerB)

		aHigh := makeActionWithSpeed(playerA, 10)
		aLow := makeActionWithSpeed(playerB, 3)
		s2.EnqueueAction(playerA, aHigh) //nolint:errcheck
		s2.EnqueueAction(playerB, aLow)  //nolint:errcheck

		closed, opened, err := s2.OpenNextAction()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed != nil {
			t.Error("expected nil closed turn on first OpenNextAction")
		}
		if opened == nil {
			t.Fatal("expected non-nil opened turn")
		}
		if opened.GetAction().Speed.Result != 10 {
			t.Errorf("expected speed 10, got %d", opened.GetAction().Speed.Result)
		}
		_ = s
	})

	t.Run("closes previous open turn before opening next", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s := sessionWithParticipants(playerA, playerB)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 10)) //nolint:errcheck
		s.EnqueueAction(playerB, makeActionWithSpeed(playerB, 5))  //nolint:errcheck

		_, first, _ := s.OpenNextAction()
		closed, _, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed == nil {
			t.Fatal("expected closed turn to be non-nil on second call")
		}
		if closed != first {
			t.Error("expected closed turn to be the first opened turn")
		}
		if first.GetFinishedAt() == nil {
			t.Error("expected first turn to be closed")
		}
	})

	t.Run("returns service.ErrQueueEmpty when queue is empty", func(t *testing.T) {
		s := emptySession()
		_, _, err := s.OpenNextAction()
		if err != service.ErrQueueEmpty {
			t.Errorf("expected ErrQueueEmpty, got %v", err)
		}
	})
}

func TestMatchSession_PullAction(t *testing.T) {
	t.Run("opens Turn for specific action UUID", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s := sessionWithParticipants(playerA, playerB)
		aTarget := makeActionWithSpeed(playerA, 3)
		aOther := makeActionWithSpeed(playerB, 10)
		s.EnqueueAction(playerA, aTarget) //nolint:errcheck
		s.EnqueueAction(playerB, aOther)  //nolint:errcheck
		targetID := aTarget.GetID()

		_, opened, err := s.PullAction(targetID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opened.GetAction().GetID() != targetID {
			t.Errorf("expected action %v, got %v", targetID, opened.GetAction().GetID())
		}
	})

	t.Run("returns service.ErrActionNotFound for unknown UUID", func(t *testing.T) {
		s := emptySession()
		_, _, err := s.PullAction(uuid.New())
		if err != service.ErrActionNotFound {
			t.Errorf("expected ErrActionNotFound, got %v", err)
		}
	})
}

// ── additional helpers ────────────────────────────────────────────────────────

func emptySession() *matchsession.MatchSession {
	return matchsession.NewMatchSession(uuid.New(), nil, nil)
}

func sessionWithParticipants(playerUUIDs ...uuid.UUID) *matchsession.MatchSession {
	matchUUID := uuid.New()
	participants := make([]*match.Participant, len(playerUUIDs))
	for i, id := range playerUUIDs {
		pID := id
		participants[i] = makeParticipant(matchUUID, &pID)
	}
	return matchsession.NewMatchSession(matchUUID, nil, participants)
}

func makeActionWithSpeed(actorID uuid.UUID, speed int) *action.Action {
	a := action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{Result: speed}, nil, nil, nil, nil, nil, nil)
	return a
}
```

Add `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"` to test imports.

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/domain/match/matchsession/... -run "TestMatchSession_OpenNextAction|TestMatchSession_PullAction" -v
```

Expected: FAIL — `OpenNextAction`, `PullAction` undefined.

- [ ] **Step 3: Add `OpenNextAction` and `PullAction` to `match_session.go`**

```go
// OpenNextAction closes any open Turn, then extracts the highest-priority
// Action from the queue and opens a new Turn. Returns the closed turn (nil
// if there was no open turn) and the newly opened turn.
func (s *MatchSession) OpenNextAction() (closed *turn.Turn, opened *turn.Turn, err error) {
	if s.activeRound.HasOpenTurn() {
		closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
	}
	opened, err = s.roundOrch.NextAction(s.activeRound, &s.activeQueue)
	return
}

// PullAction closes any open Turn, then extracts the Action with the given
// UUID from the queue and opens a new Turn.
func (s *MatchSession) PullAction(id uuid.UUID) (closed *turn.Turn, opened *turn.Turn, err error) {
	if s.activeRound.HasOpenTurn() {
		closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
	}
	opened, err = s.roundOrch.PullAction(s.activeRound, &s.activeQueue, id)
	return
}
```

Add the missing imports to `match_session.go`:

```go
import (
	"time"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/domain/match/matchsession/... -v
```

Expected: all PASS.

- [ ] **Step 5: Full build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/matchsession/
git commit -m "$(cat <<'EOF'
feat(matchsession): add OpenNextAction and PullAction

Delegates queue extraction to RoundOrchestrator. Closes the previous
open Turn before opening the next one.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 5 — MatchSession: AttachReaction + CloseTurn + CloseRound (TDD)

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Modify: `internal/domain/match/matchsession/match_session_test.go`

- [ ] **Step 1: Write failing tests**

Append to `match_session_test.go`:

```go
func TestMatchSession_AttachReaction(t *testing.T) {
	t.Run("attaches reaction to current turn and returns resolution", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s := sessionWithParticipants(playerA, playerB)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 10)) //nolint:errcheck
		_, opened, _ := s.OpenNextAction()
		actionID := opened.GetAction().GetID()

		reaction := makeReactionTo(playerB, actionID)
		res, err := s.AttachReaction(reaction)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatal("expected non-nil TurnResolution")
		}
		if len(opened.GetReactions()) != 1 {
			t.Errorf("expected 1 reaction, got %d", len(opened.GetReactions()))
		}
	})

	t.Run("returns ErrReactionNotCompatible for wrong target", func(t *testing.T) {
		playerA := uuid.New()
		s := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck

		reaction := makeReactionTo(playerA, uuid.New()) // wrong target
		_, err := s.AttachReaction(reaction)
		if err != service.ErrReactionNotCompatible {
			t.Errorf("expected ErrReactionNotCompatible, got %v", err)
		}
	})
}

func TestMatchSession_CloseTurn(t *testing.T) {
	t.Run("closes current open turn", func(t *testing.T) {
		playerA := uuid.New()
		s := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
		_, opened, _ := s.OpenNextAction()

		closed, err := s.CloseTurn()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed == nil {
			t.Fatal("expected non-nil closed turn")
		}
		if closed != opened {
			t.Error("expected closed turn to be the opened turn")
		}
		if closed.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set")
		}
	})

	t.Run("returns ErrNoCurrentTurn when no turns exist", func(t *testing.T) {
		s := emptySession()
		_, err := s.CloseTurn()
		if err != service.ErrNoCurrentTurn {
			t.Errorf("expected ErrNoCurrentTurn, got %v", err)
		}
	})
}

func TestMatchSession_CloseRound(t *testing.T) {
	t.Run("closes round and starts a new one with same mode", func(t *testing.T) {
		playerA := uuid.New()
		s := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck
		s.CloseTurn()                                              //nolint:errcheck

		closedRound, err := s.CloseRound()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closedRound == nil {
			t.Fatal("expected non-nil closed round")
		}
		if closedRound.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set on closed round")
		}
		if s.GetActiveRound() == closedRound {
			t.Error("expected activeRound to be a new round after CloseRound")
		}
	})

	t.Run("returns ErrRoundHasOpenTurn when a turn is still open", func(t *testing.T) {
		playerA := uuid.New()
		s := sessionWithParticipants(playerA)
		s.EnqueueAction(playerA, makeActionWithSpeed(playerA, 5)) //nolint:errcheck
		s.OpenNextAction()                                         //nolint:errcheck
		// turn is still open — no CloseTurn called

		_, err := s.CloseRound()
		if err != matchsession.ErrRoundHasOpenTurn {
			t.Errorf("expected ErrRoundHasOpenTurn, got %v", err)
		}
	})
}

func makeReactionTo(actorID, targetActionID uuid.UUID) *action.Action {
	return action.NewAction(actorID, nil, targetActionID, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
}
```

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/domain/match/matchsession/... -run "TestMatchSession_AttachReaction|TestMatchSession_CloseTurn|TestMatchSession_CloseRound" -v
```

Expected: FAIL — methods undefined.

- [ ] **Step 3: Add `AttachReaction`, `CloseTurn`, `CloseRound` to `match_session.go`**

```go
// AttachReaction validates the reaction targets the current Turn's Action,
// adds it, and re-calculates the combat resolution snapshot.
func (s *MatchSession) AttachReaction(r *action.Action) (*service.TurnResolution, error) {
	if err := s.roundOrch.AttachReaction(s.activeRound, r); err != nil {
		return nil, err
	}
	t := s.activeRound.CurrentTurn()
	return s.combatRes.Resolve(t, s.charSheets), nil
}

// CloseTurn finalizes the current Turn. Returns ErrNoCurrentTurn if the
// round has no turns.
func (s *MatchSession) CloseTurn() (*turn.Turn, error) {
	return s.roundOrch.CloseTurnErr(s.activeRound, time.Now())
}

// CloseRound finalizes the current Round and starts a new one with the same
// mode. Returns ErrRoundHasOpenTurn if the current Turn is still open.
func (s *MatchSession) CloseRound() (*round.Round, error) {
	if s.activeRound.HasOpenTurn() {
		return nil, ErrRoundHasOpenTurn
	}
	mode := s.activeRound.GetMode()
	closed := s.roundOrch.CloseRound(s.activeRound, time.Now())
	s.activeRound = round.NewRound(mode)
	return closed, nil
}
```

- [ ] **Step 4: Run all matchsession tests**

```bash
go test ./internal/domain/match/matchsession/... -v
```

Expected: all PASS.

- [ ] **Step 5: Full build and all tests**

```bash
go build ./...
go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/matchsession/
git commit -m "$(cat <<'EOF'
feat(matchsession): add AttachReaction, CloseTurn, CloseRound

AttachReaction validates target and returns updated TurnResolution.
CloseRound guards against open turns and resets activeRound.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 6 — InitMatchSession use case (TDD)

This use case is called from the Room when the match starts. It loads participants from the DB, loads their full character sheets, and creates a `MatchSession`.

**Files:**
- Create: `internal/application/match/i_session_loader.go`
- Create: `internal/application/match/init_match_session.go`

- [ ] **Step 1: Create `i_session_loader.go`**

```go
// internal/application/match/i_session_loader.go
package match

import (
	"context"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/google/uuid"
)

type ICharSheetLoader interface {
	GetCharacterSheetByUUID(ctx context.Context, uuid uuid.UUID) (*csSheet.CharacterSheet, bool, error)
}
```

- [ ] **Step 2: Write the failing test**

Create `internal/application/match/init_match_session_test.go`:

```go
package match_test

import (
	"context"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/google/uuid"
)

func TestInitMatchSession(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	sheetUUID := uuid.New()

	t.Run("creates session with loaded char sheets", func(t *testing.T) {
		pUUID := playerUUID
		repo := &mockMatchRepo{
			participants: []*matchDomain.Participant{
				{
					UUID:      uuid.New(),
					MatchUUID: matchUUID,
					Sheet: csEntity.Summary{
						UUID:       sheetUUID,
						PlayerUUID: &pUUID,
					},
				},
			},
		}
		loader := &mockSheetLoader{
			sheet: &csSheet.CharacterSheet{},
		}

		uc := match.NewInitMatchSessionUC(repo, loader)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected non-nil session")
		}
	})

	t.Run("creates session even when sheet not found (NPC case)", func(t *testing.T) {
		repo := &mockMatchRepo{
			participants: []*matchDomain.Participant{
				{
					UUID:      uuid.New(),
					MatchUUID: matchUUID,
					Sheet:     csEntity.Summary{UUID: sheetUUID}, // no PlayerUUID
				},
			},
		}
		loader := &mockSheetLoader{found: false}

		uc := match.NewInitMatchSessionUC(repo, loader)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected non-nil session")
		}
	})
}

// ── mocks ────────────────────────────────────────────────────────────────────

type mockMatchRepo struct {
	participants []*matchDomain.Participant
	err          error
	// embed the full IRepository to satisfy the interface without implementing all methods
	IRepository
}

func (m *mockMatchRepo) ListParticipantsByMatchUUID(_ context.Context, _ uuid.UUID) ([]*matchDomain.Participant, error) {
	return m.participants, m.err
}

type mockSheetLoader struct {
	sheet *csSheet.CharacterSheet
	found bool
	err   error
}

func (m *mockSheetLoader) GetCharacterSheetByUUID(_ context.Context, _ uuid.UUID) (*csSheet.CharacterSheet, bool, error) {
	return m.sheet, m.found, m.err
}
```

> **Note:** `mockMatchRepo` embeds `IRepository` so it only needs to implement `ListParticipantsByMatchUUID`. Any other IRepository method called would panic, catching unintended usage in tests.

- [ ] **Step 3: Run to confirm failure**

```bash
go test ./internal/application/match/... -run TestInitMatchSession -v
```

Expected: FAIL — `InitMatchSessionUC`, `NewInitMatchSessionUC` undefined.

- [ ] **Step 4: Implement `init_match_session.go`**

```go
// internal/application/match/init_match_session.go
package match

import (
	"context"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IInitMatchSession interface {
	Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error)
}

type InitMatchSessionUC struct {
	matchRepo   IRepository
	sheetLoader ICharSheetLoader
}

func NewInitMatchSessionUC(matchRepo IRepository, sheetLoader ICharSheetLoader) *InitMatchSessionUC {
	return &InitMatchSessionUC{matchRepo: matchRepo, sheetLoader: sheetLoader}
}

func (uc *InitMatchSessionUC) Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error) {
	participants, err := uc.matchRepo.ListParticipantsByMatchUUID(ctx, matchUUID)
	if err != nil {
		return nil, err
	}

	charSheets := make(map[uuid.UUID]*csSheet.CharacterSheet, len(participants))
	for _, p := range participants {
		if p.Sheet.PlayerUUID == nil {
			continue
		}
		sheet, found, err := uc.sheetLoader.GetCharacterSheetByUUID(ctx, p.Sheet.UUID)
		if err != nil {
			return nil, err
		}
		if found {
			charSheets[*p.Sheet.PlayerUUID] = sheet
		}
	}

	return matchsession.NewMatchSession(matchUUID, charSheets, participants), nil
}
```

- [ ] **Step 5: Fix the test — add `IRepository` embed**

The `mockMatchRepo` embedded `IRepository` is an interface — it needs a concrete nil value or the embed pattern needs to be `*match.IRepository`... but interfaces can't be embedded as nil directly with a type alias. Fix by making mockMatchRepo a standalone struct with only the needed method:

Actually the embed pattern works fine in Go when you embed an interface type — the nil value of the embedded interface will panic only if unimplemented methods are called, which is the desired behavior for unused methods. The test only calls `ListParticipantsByMatchUUID`, so it's safe. Run the test to confirm.

- [ ] **Step 6: Run tests**

```bash
go test ./internal/application/match/... -run TestInitMatchSession -v
```

Expected: all PASS.

- [ ] **Step 7: Full build and all tests**

```bash
go build ./...
go test ./...
```

- [ ] **Step 8: Commit**

```bash
git add internal/application/match/i_session_loader.go internal/application/match/init_match_session.go internal/application/match/init_match_session_test.go
git commit -m "$(cat <<'EOF'
feat(application/match): add InitMatchSessionUC

Loads participants and character sheets from DB and creates a MatchSession.
NPC participants (no playerUUID) are skipped in the sheet cache.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 7 — OpenNextAction + PullAction use cases (TDD)

Both are master-only operations. They call the session, optionally persist the closed turn (Phase 3 concern — skip for now), and return results for the Room to broadcast.

**Files:**
- Create: `internal/application/match/open_next_action.go`
- Create: `internal/application/match/open_next_action_test.go`
- Create: `internal/application/match/pull_action.go`
- Create: `internal/application/match/pull_action_test.go`

- [ ] **Step 1: Write failing tests for `OpenNextActionUC`**

Create `internal/application/match/open_next_action_test.go`:

```go
package match_test

import (
	"context"
	"errors"
	"testing"

	matchDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/google/uuid"
)

func TestOpenNextActionUC(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
		session := emptyMatchSession()
		uc := match.NewOpenNextActionUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("returns result with opened turn on success", func(t *testing.T) {
		pUUID := playerUUID
		session := sessionWithPlayer(masterUUID, &pUUID)
		session.EnqueueAction(playerUUID, makeTestAction(playerUUID)) //nolint:errcheck

		uc := match.NewOpenNextActionUC()
		result, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.OpenedTurn == nil {
			t.Error("expected non-nil OpenedTurn")
		}
		if result.Resolution == nil {
			t.Error("expected non-nil Resolution")
		}
	})

	t.Run("returns ErrQueueEmpty when queue is empty", func(t *testing.T) {
		session := emptyMatchSession()
		uc := match.NewOpenNextActionUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if !errors.Is(err, service.ErrQueueEmpty) {
			t.Errorf("expected ErrQueueEmpty, got %v", err)
		}
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func emptyMatchSession() *matchsession.MatchSession {
	return matchsession.NewMatchSession(uuid.New(), nil, nil)
}

func sessionWithPlayer(masterUUID uuid.UUID, playerUUID *uuid.UUID) *matchsession.MatchSession {
	matchUUID := uuid.New()
	p := &matchDomain.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet:     csEntity.Summary{UUID: uuid.New(), PlayerUUID: playerUUID},
	}
	return matchsession.NewMatchSession(matchUUID, nil, []*matchDomain.Participant{p})
}

func makeTestAction(actorID uuid.UUID) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{Result: 5}, nil, nil, nil, nil, nil, nil)
}
```

> `csEntity` is already imported from the test file written in Task 6. If this is a separate file, add `csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"` to the imports.

- [ ] **Step 2: Run to confirm failure**

```bash
go test ./internal/application/match/... -run TestOpenNextActionUC -v
```

Expected: FAIL — `OpenNextActionUC` undefined.

- [ ] **Step 3: Implement `open_next_action.go`**

```go
// internal/application/match/open_next_action.go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type OpenNextActionResult struct {
	ClosedTurn *turn.Turn
	OpenedTurn *turn.Turn
	Resolution *service.TurnResolution
}

type IOpenNextAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*OpenNextActionResult, error)
}

type OpenNextActionUC struct{}

func NewOpenNextActionUC() *OpenNextActionUC { return &OpenNextActionUC{} }

func (uc *OpenNextActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*OpenNextActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	closed, opened, err := session.OpenNextAction()
	if err != nil {
		return nil, err
	}
	resolution := service.CombatResolver{}.Resolve(opened, nil)
	return &OpenNextActionResult{ClosedTurn: closed, OpenedTurn: opened, Resolution: resolution}, nil
}
```

> `ErrNotMatchMaster` already exists in `internal/application/match/error.go`.

- [ ] **Step 4: Write and implement `pull_action.go`** (same pattern)

Create `internal/application/match/pull_action.go`:

```go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type PullActionResult struct {
	ClosedTurn *turn.Turn
	OpenedTurn *turn.Turn
	Resolution *service.TurnResolution
}

type IPullAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, actionID uuid.UUID) (*PullActionResult, error)
}

type PullActionUC struct{}

func NewPullActionUC() *PullActionUC { return &PullActionUC{} }

func (uc *PullActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	actionID uuid.UUID,
) (*PullActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	closed, opened, err := session.PullAction(actionID)
	if err != nil {
		return nil, err
	}
	resolution := service.CombatResolver{}.Resolve(opened, nil)
	return &PullActionResult{ClosedTurn: closed, OpenedTurn: opened, Resolution: resolution}, nil
}
```

Create `internal/application/match/pull_action_test.go`:

```go
package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestPullActionUC(t *testing.T) {
	masterUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster when caller is not master", func(t *testing.T) {
		session := emptyMatchSession()
		uc := match.NewPullActionUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New(), uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("returns ErrActionNotFound for unknown actionID", func(t *testing.T) {
		session := emptyMatchSession()
		uc := match.NewPullActionUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID, uuid.New())
		if !errors.Is(err, service.ErrActionNotFound) {
			t.Errorf("expected ErrActionNotFound, got %v", err)
		}
	})
}
```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/application/match/... -run "TestOpenNextActionUC|TestPullActionUC" -v
```

Expected: all PASS.

- [ ] **Step 6: Full build and all tests**

```bash
go build ./...
go test ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/application/match/open_next_action.go internal/application/match/open_next_action_test.go \
        internal/application/match/pull_action.go internal/application/match/pull_action_test.go
git commit -m "$(cat <<'EOF'
feat(application/match): add OpenNextAction and PullAction use cases

Both are master-only. Return closed turn (if any), opened turn, and
initial TurnResolution. ErrNotMatchMaster on unauthorized callers.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 8 — EnqueueAction + AttachReaction + CloseTurn + CloseRound use cases (TDD)

**Files:**
- Create: `internal/application/match/enqueue_action.go`
- Create: `internal/application/match/enqueue_action_test.go`
- Create: `internal/application/match/attach_reaction.go`
- Create: `internal/application/match/attach_reaction_test.go`
- Create: `internal/application/match/close_turn.go`
- Create: `internal/application/match/close_turn_test.go`
- Create: `internal/application/match/close_round.go`
- Create: `internal/application/match/close_round_test.go`

- [ ] **Step 1: Implement `enqueue_action.go`**

```go
// internal/application/match/enqueue_action.go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IEnqueueAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, playerUUID uuid.UUID, a *action.Action) error
}

type EnqueueActionUC struct{}

func NewEnqueueActionUC() *EnqueueActionUC { return &EnqueueActionUC{} }

func (uc *EnqueueActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	playerUUID uuid.UUID,
	a *action.Action,
) error {
	return session.EnqueueAction(playerUUID, a)
}
```

Create `internal/application/match/enqueue_action_test.go`:

```go
package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

func TestEnqueueActionUC(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	t.Run("enqueues action for enrolled player", func(t *testing.T) {
		pUUID := playerUUID
		session := sessionWithPlayer(masterUUID, &pUUID)
		a := makeTestAction(playerUUID)
		uc := match.NewEnqueueActionUC()
		if err := uc.Execute(context.Background(), session, playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrParticipantNotFound for unknown player", func(t *testing.T) {
		session := emptyMatchSession()
		a := makeTestAction(uuid.New())
		uc := match.NewEnqueueActionUC()
		err := uc.Execute(context.Background(), session, uuid.New(), a)
		if !errors.Is(err, matchsession.ErrParticipantNotFound) {
			t.Errorf("expected ErrParticipantNotFound, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Implement `attach_reaction.go`**

```go
// internal/application/match/attach_reaction.go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type IAttachReaction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, callerUUID uuid.UUID, r *action.Action) (*service.TurnResolution, error)
}

type AttachReactionUC struct{}

func NewAttachReactionUC() *AttachReactionUC { return &AttachReactionUC{} }

func (uc *AttachReactionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	callerUUID uuid.UUID,
	r *action.Action,
) (*service.TurnResolution, error) {
	return session.AttachReaction(r)
}
```

Create `internal/application/match/attach_reaction_test.go`:

```go
package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestAttachReactionUC(t *testing.T) {
	masterUUID := uuid.New()
	playerA := uuid.New()
	playerB := uuid.New()

	t.Run("returns TurnResolution on valid reaction", func(t *testing.T) {
		pA, pB := playerA, playerB
		session := sessionWithPlayer(masterUUID, &pA)
		session2 := sessionWithPlayerPair(masterUUID, &pA, &pB)
		session2.EnqueueAction(playerA, makeTestAction(playerA)) //nolint:errcheck
		_, opened, _ := session2.OpenNextAction()
		actionID := opened.GetAction().GetID()

		reaction := action.NewAction(playerB, nil, actionID, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
		uc := match.NewAttachReactionUC()
		res, err := uc.Execute(context.Background(), session2, playerB, reaction)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatal("expected non-nil resolution")
		}
		_ = session
	})

	t.Run("returns ErrReactionNotCompatible for wrong target", func(t *testing.T) {
		pA := playerA
		session := sessionWithPlayer(masterUUID, &pA)
		session.EnqueueAction(playerA, makeTestAction(playerA)) //nolint:errcheck
		session.OpenNextAction()                                  //nolint:errcheck

		reaction := action.NewAction(playerA, nil, uuid.New(), nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
		uc := match.NewAttachReactionUC()
		_, err := uc.Execute(context.Background(), session, playerA, reaction)
		if !errors.Is(err, service.ErrReactionNotCompatible) {
			t.Errorf("expected ErrReactionNotCompatible, got %v", err)
		}
	})
}

func sessionWithPlayerPair(masterUUID uuid.UUID, pA, pB *uuid.UUID) *matchsession.MatchSession {
	matchUUID := uuid.New()
	participants := []*matchDomain.Participant{
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: uuid.New(), PlayerUUID: pA}},
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: uuid.New(), PlayerUUID: pB}},
	}
	return matchsession.NewMatchSession(matchUUID, nil, participants)
}
```

> Add missing imports: `matchDomain`, `csEntity`, `matchsession`.

- [ ] **Step 3: Implement `close_turn.go` and `close_round.go`**

```go
// internal/application/match/close_turn.go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type ICloseTurn interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*turn.Turn, error)
}

type CloseTurnUC struct{}

func NewCloseTurnUC() *CloseTurnUC { return &CloseTurnUC{} }

func (uc *CloseTurnUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*turn.Turn, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	return session.CloseTurn()
}
```

```go
// internal/application/match/close_round.go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type ICloseRound interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*round.Round, error)
}

type CloseRoundUC struct{}

func NewCloseRoundUC() *CloseRoundUC { return &CloseRoundUC{} }

func (uc *CloseRoundUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*round.Round, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	return session.CloseRound()
}
```

Create `internal/application/match/close_turn_test.go`:

```go
package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestCloseTurnUC(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster for non-master caller", func(t *testing.T) {
		session := emptyMatchSession()
		uc := match.NewCloseTurnUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("closes the current open turn", func(t *testing.T) {
		pUUID := playerUUID
		session := sessionWithPlayer(masterUUID, &pUUID)
		session.EnqueueAction(playerUUID, makeTestAction(playerUUID)) //nolint:errcheck
		session.OpenNextAction()                                       //nolint:errcheck

		uc := match.NewCloseTurnUC()
		closed, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed == nil || closed.GetFinishedAt() == nil {
			t.Error("expected non-nil closed turn with finishedAt set")
		}
	})

	t.Run("returns ErrNoCurrentTurn when round has no turns", func(t *testing.T) {
		session := emptyMatchSession()
		uc := match.NewCloseTurnUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if !errors.Is(err, service.ErrNoCurrentTurn) {
			t.Errorf("expected ErrNoCurrentTurn, got %v", err)
		}
	})
}
```

Create `internal/application/match/close_round_test.go`:

```go
package match_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

func TestCloseRoundUC(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	t.Run("returns ErrNotMatchMaster for non-master caller", func(t *testing.T) {
		session := emptyMatchSession()
		uc := match.NewCloseRoundUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, uuid.New())
		if !errors.Is(err, match.ErrNotMatchMaster) {
			t.Errorf("expected ErrNotMatchMaster, got %v", err)
		}
	})

	t.Run("closes round when no open turn", func(t *testing.T) {
		pUUID := playerUUID
		session := sessionWithPlayer(masterUUID, &pUUID)
		session.EnqueueAction(playerUUID, makeTestAction(playerUUID)) //nolint:errcheck
		session.OpenNextAction()                                       //nolint:errcheck
		session.CloseTurn()                                            //nolint:errcheck

		uc := match.NewCloseRoundUC()
		closed, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if closed == nil || closed.GetFinishedAt() == nil {
			t.Error("expected non-nil closed round with finishedAt set")
		}
	})

	t.Run("returns ErrRoundHasOpenTurn when turn is still open", func(t *testing.T) {
		pUUID := playerUUID
		session := sessionWithPlayer(masterUUID, &pUUID)
		session.EnqueueAction(playerUUID, makeTestAction(playerUUID)) //nolint:errcheck
		session.OpenNextAction()                                       //nolint:errcheck // turn still open

		uc := match.NewCloseRoundUC()
		_, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)
		if !errors.Is(err, matchsession.ErrRoundHasOpenTurn) {
			t.Errorf("expected ErrRoundHasOpenTurn, got %v", err)
		}
	})
}
```

- [ ] **Step 4: Run all new use case tests**

```bash
go test ./internal/application/match/... -run "TestEnqueueActionUC|TestAttachReactionUC|TestCloseTurnUC|TestCloseRoundUC" -v
```

Expected: all PASS.

- [ ] **Step 5: Full build and all tests**

```bash
go build ./...
go test ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/application/match/enqueue_action.go internal/application/match/enqueue_action_test.go \
        internal/application/match/attach_reaction.go internal/application/match/attach_reaction_test.go \
        internal/application/match/close_turn.go internal/application/match/close_turn_test.go \
        internal/application/match/close_round.go internal/application/match/close_round_test.go
git commit -m "$(cat <<'EOF'
feat(application/match): add EnqueueAction, AttachReaction, CloseTurn, CloseRound use cases

All delegate to MatchSession. CloseTurn/CloseRound are master-only.
AttachReaction is open to any enrolled player.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 9 — New WebSocket message types (TDD)

**Files:**
- Modify: `internal/app/game/message.go`
- Modify: `internal/app/game/game_test.go` (add coverage for new constants)

- [ ] **Step 1: Add new message type constants and payload structs to `message.go`**

Open `internal/app/game/message.go` and add after the existing constants block:

```go
const (
	// existing constants above...

	// Client → Server (game actions)
	MsgTypeEnqueueAction  MessageType = "enqueue_action"
	MsgTypeOpenNextAction MessageType = "open_next_action"
	MsgTypePullAction     MessageType = "pull_action"
	MsgTypeAttachReaction MessageType = "attach_reaction"
	MsgTypeCloseTurn      MessageType = "close_turn"
	MsgTypeCloseRound     MessageType = "close_round"

	// Server → Client (game events)
	MsgTypeTurnOpened      MessageType = "turn_opened"
	MsgTypeTurnClosed      MessageType = "turn_closed"
	MsgTypeRoundClosed     MessageType = "round_closed"
	MsgTypeResolutionUpdate MessageType = "resolution_updated"
)
```

Add payload structs (after the existing payload structs in the file):

```go
type EnqueueActionPayload struct {
	ActionType string    `json:"action_type"` // "attack", "defense", "dodge", "move"
	TargetID   uuid.UUID `json:"target_id,omitempty"`
}

type PullActionPayload struct {
	ActionID uuid.UUID `json:"action_id"`
}

type AttachReactionPayload struct {
	ReactToID  uuid.UUID `json:"react_to_id"`
	ActionType string    `json:"action_type"`
}

type TurnOpenedPayload struct {
	TurnID     uuid.UUID `json:"turn_id"`
	ActorID    uuid.UUID `json:"actor_id"`
	ActionType string    `json:"action_type"`
}

type TurnClosedPayload struct {
	TurnID uuid.UUID `json:"turn_id"`
}

type RoundClosedPayload struct {
	RoundMode string `json:"round_mode"`
}

type ResolutionUpdatedPayload struct {
	TurnID    uuid.UUID `json:"turn_id"`
	IsSettled bool      `json:"is_settled"`
}
```

- [ ] **Step 2: Run tests to confirm nothing broke**

```bash
go test ./internal/app/game/... -v
```

Expected: all PASS (new constants don't break existing tests).

- [ ] **Step 3: Full build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add internal/app/game/message.go
git commit -m "$(cat <<'EOF'
feat(game): add WebSocket message types for in-session game actions

Adds enqueue_action, open_next_action, pull_action, attach_reaction,
close_turn, close_round (client→server) and turn_opened, turn_closed,
round_closed, resolution_updated (server→client).

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 10 — Room integration (TDD)

Wire `MatchSession` into the Room: add the session field, inject the six new use case interfaces, update `StartMatch` to initialize the session, and handle the six new message types.

**Files:**
- Modify: `internal/app/game/room.go`
- Modify: `internal/app/game/game_test.go`

- [ ] **Step 1: Update `room.go` — add imports, interfaces, and new Room fields**

At the top of `room.go`, replace the import block with:

```go
import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"sync"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)
```

Add new interfaces after the existing `IKickPlayer` interface:

```go
type IInitMatchSession interface {
	Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error)
}

type IOpenNextAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*appmatch.OpenNextActionResult, error)
}

type IPullAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, actionID uuid.UUID) (*appmatch.PullActionResult, error)
}

type IEnqueueAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, playerUUID uuid.UUID, a *action.Action) error
}

type IAttachReaction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, callerUUID uuid.UUID, r *action.Action) (*appmatch.AttachReactionResult, error)
}

type ICloseTurn interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*turn.Turn, error)
}

type ICloseRound interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*round.Round, error)
}
```

> `*appmatch.AttachReactionResult` needs to be defined. Add it to `attach_reaction.go`:
> ```go
> type AttachReactionResult struct {
>     Resolution *service.TurnResolution
> }
> ```
> Then update `AttachReactionUC.Execute` return type accordingly (return `&AttachReactionResult{Resolution: res}, nil`).

Expand the `Room` struct:

```go
type Room struct {
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	state      RoomState
	clients    map[uuid.UUID]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	mu         sync.RWMutex

	session *matchsession.MatchSession // nil until match starts

	startMatchUC    IStartMatch
	kickPlayerUC    IKickPlayer
	initSessionUC   IInitMatchSession
	openNextActionUC IOpenNextAction
	pullActionUC    IPullAction
	enqueueActionUC IEnqueueAction
	attachReactionUC IAttachReaction
	closeTurnUC     ICloseTurn
	closeRoundUC    ICloseRound
}
```

Update `NewRoom`:

```go
func NewRoom(
	matchUUID, masterUUID uuid.UUID,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
	initSessionUC IInitMatchSession,
	openNextActionUC IOpenNextAction,
	pullActionUC IPullAction,
	enqueueActionUC IEnqueueAction,
	attachReactionUC IAttachReaction,
	closeTurnUC ICloseTurn,
	closeRoundUC ICloseRound,
) *Room {
	return &Room{
		matchUUID:        matchUUID,
		masterUUID:       masterUUID,
		state:            RoomStateLobby,
		clients:          make(map[uuid.UUID]*Client),
		broadcast:        make(chan []byte, 256),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		stop:             make(chan struct{}),
		startMatchUC:     startMatchUC,
		kickPlayerUC:     kickPlayerUC,
		initSessionUC:    initSessionUC,
		openNextActionUC: openNextActionUC,
		pullActionUC:     pullActionUC,
		enqueueActionUC:  enqueueActionUC,
		attachReactionUC: attachReactionUC,
		closeTurnUC:      closeTurnUC,
		closeRoundUC:     closeRoundUC,
	}
}
```

- [ ] **Step 2: Update `Room.StartMatch` to initialize the session**

Replace the existing `StartMatch` method body:

```go
func (r *Room) StartMatch(userUUID uuid.UUID) error {
	if !r.IsMaster(userUUID) {
		return ErrNotMaster
	}
	r.mu.RLock()
	if r.state != RoomStateLobby {
		r.mu.RUnlock()
		return ErrAlreadyPlaying
	}
	r.mu.RUnlock()

	ctx := context.Background()
	if err := r.startMatchUC.Start(ctx, r.matchUUID, userUUID); err != nil {
		return err
	}

	session, err := r.initSessionUC.Init(ctx, r.matchUUID)
	if err != nil {
		return err
	}

	r.mu.Lock()
	r.session = session
	r.state = RoomStatePlaying
	r.mu.Unlock()

	msg := NewServerMessage(MsgTypeMatchStarted, struct{}{})
	data, _ := json.Marshal(msg)
	go func() { r.broadcast <- data }()
	return nil
}
```

- [ ] **Step 3: Add new case handlers to `handleClientMessage`**

Extend the switch in `handleClientMessage` with six new cases:

```go
case MsgTypeOpenNextAction:
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
		return
	}
	result, err := r.openNextActionUC.Execute(context.Background(), r.session, r.masterUUID, client.userUUID)
	if err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	turnID := result.OpenedTurn.GetID()
	actorID := result.OpenedTurn.GetAction().GetActorID()
	out := NewServerMessage(MsgTypeTurnOpened, TurnOpenedPayload{
		TurnID:  turnID,
		ActorID: actorID,
	})
	data, _ := json.Marshal(out)
	go func() { r.broadcast <- data }()

case MsgTypePullAction:
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
		return
	}
	var payload PullActionPayload
	if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid pull_action payload"))
		return
	}
	result, err := r.pullActionUC.Execute(context.Background(), r.session, r.masterUUID, client.userUUID, payload.ActionID)
	if err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	out := NewServerMessage(MsgTypeTurnOpened, TurnOpenedPayload{
		TurnID:  result.OpenedTurn.GetID(),
		ActorID: result.OpenedTurn.GetAction().GetActorID(),
	})
	data, _ := json.Marshal(out)
	go func() { r.broadcast <- data }()

case MsgTypeEnqueueAction:
	var payload EnqueueActionPayload
	if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid enqueue_action payload"))
		return
	}
	a := action.NewAction(client.userUUID, []uuid.UUID{payload.TargetID}, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
	if err := r.enqueueActionUC.Execute(context.Background(), r.session, client.userUUID, a); err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	// Ack to player only — queue contents are not broadcast
	client.SendMessage(NewServerMessage(MsgTypeError, ErrorPayload{Code: "ok", Message: "action enqueued"}))

case MsgTypeAttachReaction:
	var payload AttachReactionPayload
	if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid attach_reaction payload"))
		return
	}
	reaction := action.NewAction(client.userUUID, nil, payload.ReactToID, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil)
	result, err := r.attachReactionUC.Execute(context.Background(), r.session, client.userUUID, reaction)
	if err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	// Resolution is sent to master only (visibility rule: reactions are private until revealed)
	r.mu.RLock()
	masterClient, ok := r.clients[r.masterUUID]
	r.mu.RUnlock()
	if ok {
		out := NewServerMessage(MsgTypeResolutionUpdate, ResolutionUpdatedPayload{IsSettled: result.Resolution.IsSettled})
		masterClient.SendMessage(out)
	}

case MsgTypeCloseTurn:
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
		return
	}
	closed, err := r.closeTurnUC.Execute(context.Background(), r.session, r.masterUUID, client.userUUID)
	if err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	out := NewServerMessage(MsgTypeTurnClosed, TurnClosedPayload{TurnID: closed.GetID()})
	data, _ := json.Marshal(out)
	go func() { r.broadcast <- data }()

case MsgTypeCloseRound:
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
		return
	}
	closedRound, err := r.closeRoundUC.Execute(context.Background(), r.session, r.masterUUID, client.userUUID)
	if err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	out := NewServerMessage(MsgTypeRoundClosed, RoundClosedPayload{RoundMode: closedRound.GetMode().String()})
	data, _ := json.Marshal(out)
	go func() { r.broadcast <- data }()
```

> `Round.GetMode()` returns `enum.RoundMode`. Add `.String()` or use `string(closedRound.GetMode())` depending on whether `RoundMode` has a `String()` method. Check `internal/domain/entity/enum/` and adapt.

- [ ] **Step 4: Update `game_test.go` mocks for new `NewRoom` signature**

In `game_test.go`, `TestHub` and `TestRoom` currently call:
```go
hub.GetOrCreateRoom(matchUUID, masterUUID, &mockStartMatchUC{}, &mockKickPlayerUC{})
game.NewRoom(matchUUID, masterUUID, &mockStartMatchUC{}, &mockKickPlayerUC{})
```

Add mock structs for the new UCs and update all call sites:

```go
type mockInitSessionUC struct{}

func (m *mockInitSessionUC) Init(_ context.Context, _ uuid.UUID) (*matchsession.MatchSession, error) {
	return matchsession.NewMatchSession(uuid.New(), nil, nil), nil
}

type mockOpenNextActionUC struct{}

func (m *mockOpenNextActionUC) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID) (*appmatch.OpenNextActionResult, error) {
	return nil, nil
}

type mockPullActionUC struct{}

func (m *mockPullActionUC) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID, _ uuid.UUID) (*appmatch.PullActionResult, error) {
	return nil, nil
}

type mockEnqueueActionUC struct{}

func (m *mockEnqueueActionUC) Execute(_ context.Context, _ *matchsession.MatchSession, _ uuid.UUID, _ *action.Action) error {
	return nil
}

type mockAttachReactionUC struct{}

func (m *mockAttachReactionUC) Execute(_ context.Context, _ *matchsession.MatchSession, _ uuid.UUID, _ *action.Action) (*appmatch.AttachReactionResult, error) {
	return nil, nil
}

type mockCloseTurnUC struct{}

func (m *mockCloseTurnUC) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID) (*turn.Turn, error) {
	return nil, nil
}

type mockCloseRoundUC struct{}

func (m *mockCloseRoundUC) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID) (*round.Round, error) {
	return nil, nil
}
```

Define a helper to build a Room with all mocks:

```go
func newTestRoom(matchUUID, masterUUID uuid.UUID) *game.Room {
	return game.NewRoom(
		matchUUID, masterUUID,
		&mockStartMatchUC{},
		&mockKickPlayerUC{},
		&mockInitSessionUC{},
		&mockOpenNextActionUC{},
		&mockPullActionUC{},
		&mockEnqueueActionUC{},
		&mockAttachReactionUC{},
		&mockCloseTurnUC{},
		&mockCloseRoundUC{},
	)
}
```

Replace all `game.NewRoom(...)` and `hub.GetOrCreateRoom(...)` calls in the test file with `newTestRoom(...)` or with all 9 arguments explicitly.

- [ ] **Step 5: Run game tests**

```bash
go test ./internal/app/game/... -v
```

Expected: all PASS.

- [ ] **Step 6: Full build and all tests**

```bash
go build ./...
go test ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/app/game/room.go internal/app/game/game_test.go
git commit -m "$(cat <<'EOF'
feat(game/room): integrate MatchSession and wire six in-session message handlers

Room now creates a MatchSession on StartMatch via InitMatchSessionUC.
Handles: open_next_action, pull_action, enqueue_action, attach_reaction,
close_turn, close_round. Reactions are reported to master only.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 11 — Hub and Handler wiring

**Files:**
- Modify: `internal/app/game/hub.go`
- Modify: `internal/app/game/handler.go`
- Modify: `cmd/game/main.go` (or wherever `NewHandler`/`GetOrCreateRoom` is called)

- [ ] **Step 1: Update `Hub.GetOrCreateRoom` signature**

In `hub.go`, update the method signature to pass all new UCs through to `NewRoom`:

```go
func (h *Hub) GetOrCreateRoom(
	matchUUID, masterUUID uuid.UUID,
	startMatchUC IStartMatch,
	kickPlayerUC IKickPlayer,
	initSessionUC IInitMatchSession,
	openNextActionUC IOpenNextAction,
	pullActionUC IPullAction,
	enqueueActionUC IEnqueueAction,
	attachReactionUC IAttachReaction,
	closeTurnUC ICloseTurn,
	closeRoundUC ICloseRound,
) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[matchUUID]; ok {
		return room
	}

	room := NewRoom(matchUUID, masterUUID,
		startMatchUC, kickPlayerUC,
		initSessionUC, openNextActionUC, pullActionUC,
		enqueueActionUC, attachReactionUC, closeTurnUC, closeRoundUC,
	)
	h.rooms[matchUUID] = room
	go room.Run()
	return room
}
```

- [ ] **Step 2: Update `handler.go` to accept and inject the new UCs**

Find `cmd/game/` to understand how the server is wired. Read `internal/app/game/server.go` and the game command main if it exists:

```bash
find cmd/game -name "*.go" | xargs ls -la
```

Then update `Handler` to accept new UCs in its constructor, and update `HandleWebSocket` to pass them to `GetOrCreateRoom`. Follow the existing patterns exactly — `IStartMatch` and `IKickPlayer` are already there; add the six new interfaces the same way.

- [ ] **Step 3: Update the wiring in `cmd/game/`**

Instantiate the six new use cases and pass them to `NewHandler`:

```go
// In cmd/game/main.go (or server setup file):
sheetRepo := pgsheet.NewRepository(pool)
matchRepo := pgmatch.NewRepository(pool)

initSessionUC   := appmatch.NewInitMatchSessionUC(matchRepo, sheetRepo)
openNextUC      := appmatch.NewOpenNextActionUC()
pullActionUC    := appmatch.NewPullActionUC()
enqueueActionUC := appmatch.NewEnqueueActionUC()
attachReactUC   := appmatch.NewAttachReactionUC()
closeTurnUC     := appmatch.NewCloseTurnUC()
closeRoundUC    := appmatch.NewCloseRoundUC()

handler := game.NewHandler(hub, matchRepo, enrollmentRepo, startMatchUC, kickPlayerUC,
    initSessionUC, openNextUC, pullActionUC, enqueueActionUC, attachReactUC, closeTurnUC, closeRoundUC)
```

- [ ] **Step 4: Full build — fix any compilation errors**

```bash
go build ./...
```

Fix any missed call sites. `grep -r "GetOrCreateRoom\|NewHandler\|NewRoom" --include="*.go" .` to find them.

- [ ] **Step 5: Full test suite**

```bash
go test ./...
```

Expected: all PASS.

- [ ] **Step 6: Run vet with integration tags**

```bash
go vet -tags=integration ./internal/gateway/pg/...
```

Expected: no errors.

- [ ] **Step 7: Commit**

```bash
git add internal/app/game/hub.go internal/app/game/handler.go cmd/game/
git commit -m "$(cat <<'EOF'
feat(game): wire six new use cases into Hub, Handler, and cmd/game

GetOrCreateRoom and NewHandler now accept all session use cases.
cmd/game instantiates and injects them at startup.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 12 — Documentation update

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Update `AGENTS.md` Known Issues**

Remove the old known issue line and replace with Phase 2 status:

```markdown
## Known Issues

(Phase 2 complete — MatchSession and in-session use cases operational.
Turn/Round DB persistence deferred to Phase 3.)
```

- [ ] **Step 2: Final full build and test**

```bash
go build ./...
go test ./...
go vet -tags=integration ./internal/gateway/pg/...
```

- [ ] **Step 3: Commit**

```bash
git add AGENTS.md
git commit -m "$(cat <<'EOF'
docs: mark Phase 2 complete in AGENTS.md

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Self-Review

**Spec coverage check:**

| Spec section | Covered by |
|---|---|
| `MatchSession` struct and fields | Task 2 |
| `ActionPriorityQueue` in MatchSession | Task 2 (activeQueue field) |
| `EnqueueAction` | Task 3 |
| `OpenNextAction` | Task 4 |
| `PullAction` | Task 4 |
| `AttachReaction` (+ CombatResolver re-calc) | Task 5 |
| `CloseTurn` | Task 5 |
| `CloseRound` (resets activeRound) | Task 5 |
| `charSheets` cache loaded at session start | Task 6 |
| Use cases: OpenNextAction, PullAction | Task 7 |
| Use cases: EnqueueAction, AttachReaction, CloseTurn, CloseRound | Task 8 |
| Room: session field + new message types | Tasks 9, 10 |
| Room: InitSession on StartMatch | Task 10 |
| Hub/Handler wiring | Task 11 |
| Master-only authorization on game actions | Tasks 7, 8, 10 |
| Reaction visibility: master-only broadcast | Task 10 |

**Deferred (Phase 3):**
- Turn persistence (`INSERT` on CloseTurn) — no DB schema exists yet
- Scene management (`activeScene`, `ChangeScene`)
- Initiative in `ChangeMode`
- `EnqueueMasterAction` (NPC queue)
