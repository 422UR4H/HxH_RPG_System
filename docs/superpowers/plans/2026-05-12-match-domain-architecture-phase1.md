# Match Domain Architecture — Phase 1: Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate all existing code to the approved bounded-context architecture and deliver the three stateless domain services (RoundOrchestrator, CombatResolver, RollCalculator) with full TDD coverage.

**Architecture:** Use cases move from `internal/domain/<feature>/` to `internal/application/<feature>/`. Match entities move from `internal/domain/entity/match/` into the `internal/domain/match/` bounded context. The two broken engines (`turn/engine.go`, `round/engine.go`, `match/engine.go`) are deleted; useful logic migrates to entity methods and to the new `RoundOrchestrator` service. No new game functionality is added in this plan — only correct structure and tested services.

**Tech Stack:** Go 1.23, `testing` standard package, table-driven tests with `t.Run()`, external test packages (`package <pkg>_test`).

> **Scope:** This is Phase 1 of the spec `2026-05-11-match-domain-architecture-design.md`.
> **Phase 2 (separate plan):** `MatchSession`, new use cases (`OpenNextAction`, `AttachReaction`, etc.), and Room integration.

---

## Import Path Change Reference

Keep this table open while working. Every sed command in this plan references it.

| Old import path | New import path | Reason |
|---|---|---|
| `.../domain/auth` | `.../application/auth` | use case → application layer |
| `.../domain/campaign` | `.../application/campaign` | use case → application layer |
| `.../domain/character_sheet` | `.../application/character_sheet` | use case → application layer |
| `.../domain/enrollment` | `.../application/enrollment` | use case → application layer |
| `.../domain/match` *(use cases)* | `.../application/match` | use case → application layer |
| `.../domain/scenario` | `.../application/scenario` | use case → application layer |
| `.../domain/session` | `.../application/session` | use case → application layer |
| `.../domain/submission` | `.../application/submission` | use case → application layer |
| `.../domain/entity/match"` *(root entity)* | `.../domain/match"` | entity → bounded context |
| `.../domain/entity/match/action` | `.../domain/match/entity/action` | entity → bounded context |
| `.../domain/entity/match/round` | `.../domain/match/entity/round` | entity → bounded context |
| `.../domain/entity/match/turn` | `.../domain/match/entity/turn` | entity → bounded context |
| `.../domain/entity/match/scene` | `.../domain/match/entity/scene` | entity → bounded context |
| `.../domain/entity/match/battle` | `.../domain/match/entity/battle` | entity → bounded context |

**Module prefix (use in sed):** `github.com/422UR4H/HxH_RPG_System/internal`

---

## File Map

### Created by this plan

| File | Package | Responsibility |
|---|---|---|
| `internal/application/auth/` | `auth` | moved use cases (no change in content) |
| `internal/application/campaign/` | `campaign` | moved use cases |
| `internal/application/character_sheet/` | `character_sheet` | moved use cases |
| `internal/application/enrollment/` | `enrollment` | moved use cases |
| `internal/application/match/` | `match` | moved use cases |
| `internal/application/scenario/` | `scenario` | moved use cases |
| `internal/application/session/` | `session` | moved use cases |
| `internal/application/submission/` | `submission` | moved use cases |
| `internal/application/testutil/` | `testutil` | moved mock helpers |
| `internal/domain/match/entity/action/` | `action` | match action entities (moved) |
| `internal/domain/match/entity/round/` | `round` | Round entity (moved, engine removed) |
| `internal/domain/match/entity/turn/` | `turn` | Turn entity (moved, engine deleted) |
| `internal/domain/match/entity/scene/` | `scene` | Scene entity (moved) |
| `internal/domain/match/entity/battle/` | `battle` | Blow entity (moved) |
| `internal/domain/match/service/round_orchestrator.go` | `service` | stateless Round/Turn lifecycle |
| `internal/domain/match/service/round_orchestrator_test.go` | `service_test` | TDD for RoundOrchestrator |
| `internal/domain/match/service/combat_resolver.go` | `service` | stateless combat collision |
| `internal/domain/match/service/combat_resolver_test.go` | `service_test` | TDD for CombatResolver |
| `internal/domain/match/service/roll_calculator.go` | `service` | stateless dice + attribute calc |
| `internal/domain/match/service/roll_calculator_test.go` | `service_test` | TDD for RollCalculator |
| `internal/domain/match/service/error.go` | `service` | domain service errors |

### Modified by this plan

| File | Change |
|---|---|
| `internal/domain/match/entity/round/round.go` | Add `AppendTurn`, `CurrentTurn`, `HasOpenTurn`, `Close` |
| `internal/domain/match/entity/turn/turn.go` | Add `GetID()` method |
| `internal/app/api/**/*.go` | Update use case import paths |
| `internal/app/game/room.go` | Update use case import paths |
| `internal/gateway/pg/**/*.go` | Update entity import paths |
| `internal/domain/entity/campaign/campaign.go` | Update match entity import path |
| `.github/instructions/domain-map.instructions.md` | Update paths to new structure |
| `AGENTS.md` | Update architecture section |
| `docs/documentation-map.yaml` | Already updated (done in spec session) |

### Deleted by this plan

| File | Reason |
|---|---|
| `internal/domain/entity/match/turn/engine.go` | Old semantics — fully superseded |
| `internal/domain/entity/match/round/engine.go` | Logic moves to `RoundOrchestrator` + `Round` methods |
| `internal/domain/entity/match/engine.go` | Logic moves to `MatchSession` (Phase 2) |
| `internal/domain/entity/match/` *(dir)* | Entire dir removed after move |

---

## Task 1 — Baseline Verification

**Files:** no changes

- [ ] **Step 1: Confirm the build is clean before touching anything**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System
go build ./...
```

Expected: no errors. If there are existing errors, fix them before continuing.

- [ ] **Step 2: Confirm tests pass**

```bash
go test ./...
```

Expected: all tests pass (or only the known broken match test fails — that is documented as WIP in AGENTS.md).

---

## Task 2 — Create `internal/application/` and Migrate Use Cases

This task moves all use case packages from `domain/` to `application/`. No logic changes — only file locations and import paths.

**Files:**
- Create: `internal/application/{auth,campaign,character_sheet,enrollment,match,scenario,session,submission,testutil}/`
- Modify: all files in `internal/app/` that import use case packages
- Modify: `internal/domain/enrollment/*.go` (imports `domain/match` use cases)

- [ ] **Step 1: Create the application directory structure**

```bash
mkdir -p internal/application/{auth,campaign,character_sheet,enrollment,match,scenario,session,submission,testutil}
```

- [ ] **Step 2: Move all use case files**

```bash
# Move each package (git mv preserves history)
git mv internal/domain/auth/*.go internal/application/auth/
git mv internal/domain/campaign/*.go internal/application/campaign/
git mv internal/domain/character_sheet/*.go internal/application/character_sheet/
git mv internal/domain/enrollment/*.go internal/application/enrollment/
git mv internal/domain/match/*.go internal/application/match/
git mv internal/domain/scenario/*.go internal/application/scenario/
git mv internal/domain/session/*.go internal/application/session/
git mv internal/domain/submission/*.go internal/application/submission/
git mv internal/domain/testutil/*.go internal/application/testutil/
```

> `domain/match/` is now empty. It will receive the match entity files in Task 3.

- [ ] **Step 3: Update use-case import paths in `app/` handlers and in enrollment use cases**

Run these sed commands IN ORDER (the `domain/match` rule must run before `domain/entity/match` in Task 3):

```bash
BASE="github.com/422UR4H/HxH_RPG_System/internal"

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/auth\b|${BASE}/application/auth|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/campaign\b|${BASE}/application/campaign|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/character_sheet\b|${BASE}/application/character_sheet|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/enrollment\b|${BASE}/application/enrollment|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/match\b|${BASE}/application/match|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/scenario\b|${BASE}/application/scenario|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/session\b|${BASE}/application/session|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/submission\b|${BASE}/application/submission|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/testutil\b|${BASE}/application/testutil|g" {} \;
```

> The `\b` (word boundary) prevents partial matches. On macOS, replace `sed -i` with `sed -i ''`.

- [ ] **Step 4: Verify the build compiles**

```bash
go build ./...
```

Expected: no errors. If you see "cannot find package", the sed missed a file — find it with:
```bash
grep -r "domain/auth\|domain/campaign\|domain/character_sheet\|domain/enrollment\|domain/scenario\|domain/session\b\|domain/submission" --include="*.go" internal/ | grep -v "application/"
```
Fix manually, then re-run `go build ./...`.

- [ ] **Step 5: Run tests**

```bash
go test ./...
```

Expected: same result as Task 1 baseline.

- [ ] **Step 6: Commit**

```bash
git add internal/application/ internal/app/ internal/domain/
git commit -m "$(cat <<'EOF'
refactor: move use cases from domain/ to application/

Creates internal/application/ layer and moves all use case packages
(auth, campaign, character_sheet, enrollment, match, scenario, session,
submission, testutil) out of internal/domain/. Updates all import paths.
No logic changes.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3 — Move Match Entities to Bounded Context

This task moves match entities from `internal/domain/entity/match/` into the `internal/domain/match/` bounded context. `domain/match/` is now empty (use cases moved in Task 2), so it is safe to receive entity files.

**Files:**
- Create: `internal/domain/match/entity/{action,round,turn,scene,battle}/`
- Move: all entity files from `internal/domain/entity/match/`
- Modify: gateway, campaign entity, and any remaining files with old entity imports

- [ ] **Step 1: Create the bounded context entity directories**

```bash
mkdir -p internal/domain/match/entity/{action,round,turn,scene,battle}
```

- [ ] **Step 2: Move entity files (engines will be deleted in Task 4)**

```bash
# Move root match entity files (Match, Participant, Summary, etc.)
git mv internal/domain/entity/match/match.go internal/domain/match/match.go
git mv internal/domain/entity/match/participant.go internal/domain/match/participant.go
git mv internal/domain/entity/match/summary.go internal/domain/match/summary.go
git mv internal/domain/entity/match/game_event.go internal/domain/match/game_event.go
git mv internal/domain/entity/match/character_status.go internal/domain/match/character_status.go
git mv internal/domain/entity/match/error.go internal/domain/match/error.go
git mv internal/domain/entity/match/match_test.go internal/domain/match/match_test.go
git mv internal/domain/entity/match/game_event_test.go internal/domain/match/game_event_test.go

# Move action subpackage
git mv internal/domain/entity/match/action/* internal/domain/match/entity/action/

# Move round subpackage (engine.go stays for now — deleted in Task 4)
git mv internal/domain/entity/match/round/round.go internal/domain/match/entity/round/round.go
git mv internal/domain/entity/match/round/game_event.go internal/domain/match/entity/round/game_event.go
git mv internal/domain/entity/match/round/error.go internal/domain/match/entity/round/error.go

# Move turn subpackage (engine.go stays for now — deleted in Task 4)
git mv internal/domain/entity/match/turn/turn.go internal/domain/match/entity/turn/turn.go
git mv internal/domain/entity/match/turn/error.go internal/domain/match/entity/turn/error.go

# Move scene subpackage
git mv internal/domain/entity/match/scene/* internal/domain/match/entity/scene/

# Move battle subpackage
git mv internal/domain/entity/match/battle/* internal/domain/match/entity/battle/
```

- [ ] **Step 3: Update entity import paths everywhere**

```bash
BASE="github.com/422UR4H/HxH_RPG_System/internal"

# Sub-packages first (more specific paths before the root path)
find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/entity/match/action\b|${BASE}/domain/match/entity/action|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/entity/match/round\b|${BASE}/domain/match/entity/round|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/entity/match/turn\b|${BASE}/domain/match/entity/turn|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/entity/match/scene\b|${BASE}/domain/match/entity/scene|g" {} \;

find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/entity/match/battle\b|${BASE}/domain/match/entity/battle|g" {} \;

# Root match entity LAST (prevent partial match with sub-packages)
find internal/ -name "*.go" -exec sed -i \
  "s|${BASE}/domain/entity/match\b|${BASE}/domain/match|g" {} \;
```

- [ ] **Step 4: Verify the build compiles**

```bash
go build ./...
```

If you see import errors, find the missed files:
```bash
grep -r "domain/entity/match" --include="*.go" internal/
```

- [ ] **Step 5: Run tests**

```bash
go test ./...
```

Expected: same result as before.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/ internal/domain/entity/ internal/gateway/ internal/app/ internal/application/
git commit -m "$(cat <<'EOF'
refactor: move match entities to domain/match/ bounded context

Moves all match entity packages from domain/entity/match/ into the
domain/match/ bounded-context structure (domain/match/entity/{action,
round,turn,scene,battle}). Root match entities (Match, Participant, etc.)
move to domain/match/ root. Updates all import paths.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4 — Delete Engines and Add Entity Methods

The three engine files are deleted. Before deleting `round/engine.go`, extract the methods that belong on `Round` itself (the pure-state ones). The engine orchestration logic moves to `RoundOrchestrator` in Task 5.

**Files:**
- Delete: `internal/domain/entity/match/turn/engine.go` (already at old location — git rm)
- Delete: `internal/domain/entity/match/round/engine.go` (already at old location — git rm)
- Delete: `internal/domain/entity/match/engine.go` (already at old location — git rm)
- Modify: `internal/domain/match/entity/round/round.go` — add `AppendTurn`, `CurrentTurn`, `HasOpenTurn`, `Close`
- Modify: `internal/domain/match/entity/turn/turn.go` — add `GetID()`

- [ ] **Step 1: Write the failing tests for the new Round entity methods**

Create `internal/domain/match/entity/round/round_test.go`:

```go
package round_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

func TestRound_AppendTurn(t *testing.T) {
	r := round.NewRound(enum.Free)
	a := makeAction()
	tRn := turn.NewTurn(a)

	r.AppendTurn(tRn)

	if r.CurrentTurn() != tRn {
		t.Error("CurrentTurn should return the appended turn")
	}
}

func TestRound_HasOpenTurn(t *testing.T) {
	t.Run("false when no turns", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		if r.HasOpenTurn() {
			t.Error("expected false when Round has no turns")
		}
	})

	t.Run("true when current turn is open", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		r.AppendTurn(turn.NewTurn(makeAction()))
		if !r.HasOpenTurn() {
			t.Error("expected true when Turn has no finishedAt")
		}
	})

	t.Run("false when current turn is closed", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		tRn := turn.NewTurn(makeAction())
		r.AppendTurn(tRn)
		tRn.Close(time.Now())
		if r.HasOpenTurn() {
			t.Error("expected false when Turn is closed")
		}
	})
}

func TestRound_Close(t *testing.T) {
	r := round.NewRound(enum.Free)
	at := time.Now()
	r.Close(at)
	if r.GetFinishedAt() == nil {
		t.Error("expected finishedAt to be set after Close")
	}
}

func makeAction() action.Action {
	return action.Action{ReactToID: uuid.Nil}
}
```

- [ ] **Step 2: Run the test to see it fail**

```bash
go test ./internal/domain/match/entity/round/... -v
```

Expected: FAIL — `AppendTurn`, `CurrentTurn`, `HasOpenTurn`, `Close`, `GetFinishedAt` undefined.

- [ ] **Step 3: Add the methods to `round.go`**

Open `internal/domain/match/entity/round/round.go` and add:

```go
func (r *Round) AppendTurn(t *turn.Turn) {
	r.turns = append(r.turns, t)
}

func (r *Round) CurrentTurn() *turn.Turn {
	if len(r.turns) == 0 {
		return nil
	}
	return r.turns[len(r.turns)-1]
}

func (r *Round) HasOpenTurn() bool {
	t := r.CurrentTurn()
	return t != nil && t.GetFinishedAt() == nil
}

func (r *Round) Close(at time.Time) {
	r.finishedAt = &at
}

func (r *Round) GetFinishedAt() *time.Time {
	return r.finishedAt
}
```

Also add the missing `time` import if not already present.

- [ ] **Step 4: Write the failing test for `Turn.GetID()`**

Add to `internal/domain/match/entity/turn/turn_test.go` (create if it doesn't exist):

```go
package turn_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

func TestTurn_GetID(t *testing.T) {
	a := action.Action{}
	tRn := turn.NewTurn(a)
	id := tRn.GetID()
	if id == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
}
```

- [ ] **Step 5: Run the test to see it fail**

```bash
go test ./internal/domain/match/entity/turn/... -v
```

Expected: FAIL — `GetID` undefined.

- [ ] **Step 6: Add `GetID()` to `turn.go`**

The `Turn` struct needs an `id` field (add if missing) and `GetID()`:

```go
// In the Turn struct, add:
//   id uuid.UUID

func (t *Turn) GetID() uuid.UUID {
	return t.id
}
```

Also ensure `NewTurn` initialises `id`:

```go
func NewTurn(a action.Action) *Turn {
	return &Turn{
		id:       uuid.New(),
		action:   a,
		openedAt: time.Now(),
	}
}
```

Add `"github.com/google/uuid"` import if needed.

- [ ] **Step 7: Run all entity tests**

```bash
go test ./internal/domain/match/entity/... -v
```

Expected: all PASS.

- [ ] **Step 8: Delete the engine files**

```bash
# These are still at the old location (git mv only moved the non-engine files)
git rm internal/domain/entity/match/turn/engine.go
git rm internal/domain/entity/match/round/engine.go
git rm internal/domain/entity/match/engine.go

# Remove now-empty old entity/match dir
rmdir internal/domain/entity/match/turn 2>/dev/null || true
rmdir internal/domain/entity/match/round 2>/dev/null || true
rmdir internal/domain/entity/match/ 2>/dev/null || true
```

- [ ] **Step 9: Full build and test**

```bash
go build ./...
go test ./...
```

Expected: builds and all tests pass (the pre-existing broken turn/round test is gone since engine.go was the broken file).

- [ ] **Step 10: Commit**

```bash
git add internal/domain/match/ internal/domain/entity/
git commit -m "$(cat <<'EOF'
refactor: dissolve match engines into entity methods

Deletes turn/engine.go (obsolete), round/engine.go (orchestration moves
to RoundOrchestrator in Phase 1), and match/engine.go (moves to
MatchSession in Phase 2). Adds AppendTurn, CurrentTurn, HasOpenTurn,
Close, GetFinishedAt to Round and GetID to Turn — the pure-state methods
that belonged on the entities.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5 — `RoundOrchestrator` Domain Service (TDD)

**Files:**
- Create: `internal/domain/match/service/error.go`
- Create: `internal/domain/match/service/round_orchestrator.go`
- Create: `internal/domain/match/service/round_orchestrator_test.go`

- [ ] **Step 1: Create `error.go`**

```go
// internal/domain/match/service/error.go
package service

import "errors"

var (
	ErrQueueEmpty          = errors.New("action queue is empty")
	ErrActionNotFound      = errors.New("action not found in queue")
	ErrNoCurrentTurn       = errors.New("no current turn in round")
	ErrReactionNotCompatible = errors.New("reaction does not target the current action")
)
```

- [ ] **Step 2: Write all failing tests for `RoundOrchestrator`**

Create `internal/domain/match/service/round_orchestrator_test.go`:

```go
package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestRoundOrchestrator_NextAction(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("extracts highest-speed action and appends Turn to Round", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5), makeActionWithSpeed(10))

		tRn, err := orch.NextAction(r, &q)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tRn == nil {
			t.Fatal("expected non-nil Turn")
		}
		if tRn.GetAction().Speed.Result != 10 {
			t.Errorf("expected speed 10, got %d", tRn.GetAction().Speed.Result)
		}
		if r.CurrentTurn() != tRn {
			t.Error("Round should reference the new Turn as CurrentTurn")
		}
	})

	t.Run("closes previous open turn before opening next", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5), makeActionWithSpeed(10))

		first, _ := orch.NextAction(r, &q)
		if r.HasOpenTurn() == false {
			t.Fatal("first turn should be open")
		}

		second, err := orch.NextAction(r, &q)

		if err != nil {
			t.Fatalf("unexpected error on second call: %v", err)
		}
		if first.GetFinishedAt() == nil {
			t.Error("first turn should be closed after second NextAction")
		}
		if r.CurrentTurn() != second {
			t.Error("Round should reference the second Turn as CurrentTurn")
		}
	})

	t.Run("returns ErrQueueEmpty when queue is empty", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)

		_, err := orch.NextAction(r, &q)

		if !errors.Is(err, service.ErrQueueEmpty) {
			t.Errorf("expected ErrQueueEmpty, got %v", err)
		}
	})
}

func TestRoundOrchestrator_PullAction(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("extracts specific action by UUID and appends Turn", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		target := makeActionWithSpeed(7)
		other := makeActionWithSpeed(15)
		q := newQueue(target, other)
		targetID := target.GetID()

		tRn, err := orch.PullAction(r, &q, targetID)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tRn.GetAction().GetID() != targetID {
			t.Errorf("expected action %v, got %v", targetID, tRn.GetAction().GetID())
		}
	})

	t.Run("returns ErrActionNotFound for unknown UUID", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		q := newQueue(makeActionWithSpeed(5))

		_, err := orch.PullAction(r, &q, uuid.New())

		if !errors.Is(err, service.ErrActionNotFound) {
			t.Errorf("expected ErrActionNotFound, got %v", err)
		}
	})
}

func TestRoundOrchestrator_CloseTurn(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("sets finishedAt on current turn", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5))
		orch.NextAction(r, &q) //nolint:errcheck

		at := time.Now()
		closed := orch.CloseTurn(r, at)

		if closed == nil {
			t.Fatal("expected closed turn")
		}
		if closed.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set")
		}
		if r.HasOpenTurn() {
			t.Error("round should have no open turn after CloseTurn")
		}
	})

	t.Run("returns ErrNoCurrentTurn when round has no turns", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		_, err := orch.CloseTurnErr(r, time.Now())
		if !errors.Is(err, service.ErrNoCurrentTurn) {
			t.Errorf("expected ErrNoCurrentTurn, got %v", err)
		}
	})
}

func TestRoundOrchestrator_AttachReaction(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("attaches reaction targeting current action", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5))
		tRn, _ := orch.NextAction(r, &q)
		actionID := tRn.GetAction().GetID()

		reaction := makeReactionTo(actionID)
		err := orch.AttachReaction(r, &reaction)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(r.CurrentTurn().GetReactions()) != 1 {
			t.Error("expected 1 reaction on current turn")
		}
	})

	t.Run("returns ErrReactionNotCompatible for wrong target", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		q := newQueue(makeActionWithSpeed(5))
		orch.NextAction(r, &q) //nolint:errcheck

		reaction := makeReactionTo(uuid.New()) // wrong target
		err := orch.AttachReaction(r, &reaction)

		if !errors.Is(err, service.ErrReactionNotCompatible) {
			t.Errorf("expected ErrReactionNotCompatible, got %v", err)
		}
	})

	t.Run("returns ErrNoCurrentTurn when round has no turns", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		reaction := makeReactionTo(uuid.New())
		err := orch.AttachReaction(r, &reaction)
		if !errors.Is(err, service.ErrNoCurrentTurn) {
			t.Errorf("expected ErrNoCurrentTurn, got %v", err)
		}
	})
}

func TestRoundOrchestrator_CloseRound(t *testing.T) {
	orch := service.RoundOrchestrator{}

	t.Run("sets finishedAt on Round", func(t *testing.T) {
		r := round.NewRound(enum.Free)
		at := time.Now()

		closed := orch.CloseRound(r, at)

		if closed.GetFinishedAt() == nil {
			t.Error("expected finishedAt to be set on Round")
		}
	})
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newQueue(actions ...action.Action) action.PriorityQueue {
	q := action.NewActionPriorityQueue(nil)
	for i := range actions {
		q.Insert(&actions[i])
	}
	return q
}

func makeActionWithSpeed(speed int) action.Action {
	a := action.Action{}
	a.Speed.Result = speed
	return a
}

func makeReactionTo(targetID uuid.UUID) action.Action {
	return action.Action{ReactToID: targetID}
}
```

- [ ] **Step 3: Run tests to confirm they fail**

```bash
go test ./internal/domain/match/service/... -v
```

Expected: FAIL — `service.RoundOrchestrator` undefined.

- [ ] **Step 4: Implement `round_orchestrator.go`**

Create `internal/domain/match/service/round_orchestrator.go`:

```go
package service

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// RoundOrchestrator is a stateless domain service that manages the Round/Turn
// lifecycle: extracting actions from the queue, creating and closing Turns,
// attaching reactions, and closing Rounds.
//
// It holds no state. All state lives in the Round entity passed as a parameter.
type RoundOrchestrator struct{}

// NextAction closes any open Turn, extracts the highest-priority Action from
// the queue, creates a new Turn, appends it to the Round, and returns it.
func (ro RoundOrchestrator) NextAction(r *round.Round, q *action.PriorityQueue) (*turn.Turn, error) {
	if r.HasOpenTurn() {
		r.CurrentTurn().Close(time.Now())
	}
	next := q.ExtractMax()
	if next == nil {
		return nil, ErrQueueEmpty
	}
	t := turn.NewTurn(*next)
	r.AppendTurn(t)
	return t, nil
}

// PullAction closes any open Turn, extracts the Action with the given UUID from
// the queue, creates a new Turn, and appends it to the Round.
func (ro RoundOrchestrator) PullAction(r *round.Round, q *action.PriorityQueue, id uuid.UUID) (*turn.Turn, error) {
	if r.HasOpenTurn() {
		r.CurrentTurn().Close(time.Now())
	}
	next := q.ExtractByID(id)
	if next == nil {
		return nil, ErrActionNotFound
	}
	t := turn.NewTurn(*next)
	r.AppendTurn(t)
	return t, nil
}

// CloseTurn sets finishedAt on the current Turn and returns it.
// Panics are avoided: use CloseTurnErr when a missing Turn should return an error.
func (ro RoundOrchestrator) CloseTurn(r *round.Round, at time.Time) *turn.Turn {
	t := r.CurrentTurn()
	if t == nil {
		return nil
	}
	t.Close(at)
	return t
}

// CloseTurnErr is identical to CloseTurn but returns ErrNoCurrentTurn when the
// Round has no turns yet.
func (ro RoundOrchestrator) CloseTurnErr(r *round.Round, at time.Time) (*turn.Turn, error) {
	t := r.CurrentTurn()
	if t == nil {
		return nil, ErrNoCurrentTurn
	}
	t.Close(at)
	return t, nil
}

// CloseRound sets finishedAt on the Round and returns it.
func (ro RoundOrchestrator) CloseRound(r *round.Round, at time.Time) *round.Round {
	r.Close(at)
	return r
}

// AttachReaction validates that the reaction targets the current Turn's Action,
// then appends it.
func (ro RoundOrchestrator) AttachReaction(r *round.Round, reaction *action.Action) error {
	t := r.CurrentTurn()
	if t == nil {
		return ErrNoCurrentTurn
	}
	if t.GetAction().GetID() != reaction.ReactToID {
		return ErrReactionNotCompatible
	}
	t.AddReaction(reaction)
	return nil
}

// ChangeMode toggles the Round mode between Free and Race.
// TODO: create and finish Initiative to continue here
func (ro RoundOrchestrator) ChangeMode(r *round.Round, initiative *action.Initiative) {
	r.ToggleMode()
	if initiative != nil && r.GetMode() == 0 { // TODO: process initiative
	}
}
```

> `Round.ToggleMode()` and `Action.GetID()` are needed. `GetID()` was added in Task 4.
> Add `ToggleMode()` to `round.go`:
>
> ```go
> func (r *Round) ToggleMode() {
>     if r.mode == enum.Race {
>         r.mode = enum.Free
>     } else {
>         r.mode = enum.Race
>     }
> }
> ```

- [ ] **Step 5: Run tests**

```bash
go test ./internal/domain/match/service/... -v
```

Expected: all PASS.

- [ ] **Step 6: Full build and test**

```bash
go build ./...
go test ./...
```

- [ ] **Step 7: Commit**

```bash
git add internal/domain/match/service/ internal/domain/match/entity/
git commit -m "$(cat <<'EOF'
feat: add RoundOrchestrator domain service with full test coverage

Stateless domain service that manages Round/Turn lifecycle: NextAction,
PullAction, CloseTurn, CloseRound, AttachReaction, ChangeMode. Migrates
orchestration logic from the dissolved round/engine.go.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6 — `CombatResolver` Domain Service (Structure + TDD)

The full combat formula (how attack damage vs defense/dodge is resolved numerically) is not
fully specified yet. This task creates the correct interface and types, with tests for the
structural behavior. Formula implementation is deferred.

**Files:**
- Create: `internal/domain/match/service/combat_resolver.go`
- Create: `internal/domain/match/service/combat_resolver_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/domain/match/service/combat_resolver_test.go`:

```go
package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestCombatResolver_Resolve(t *testing.T) {
	resolver := service.CombatResolver{}

	t.Run("returns non-nil TurnResolution for a Turn with only an action", func(t *testing.T) {
		tRn := makeTurnWithAttack()
		sheets := map[uuid.UUID]interface{}{} // empty — formulas not yet implemented

		res := resolver.Resolve(tRn, nil)

		if res == nil {
			t.Fatal("expected non-nil TurnResolution")
		}
	})

	t.Run("IsSettled is false when turn has no finishedAt", func(t *testing.T) {
		tRn := makeTurnWithAttack()
		res := resolver.Resolve(tRn, nil)
		if res.IsSettled {
			t.Error("expected IsSettled=false for open turn")
		}
	})

	t.Run("IsSettled is true when turn is closed", func(t *testing.T) {
		tRn := makeTurnWithAttack()
		tRn.Close(time.Now()) // close it
		res := resolver.Resolve(tRn, nil)
		if !res.IsSettled {
			t.Error("expected IsSettled=true for closed turn")
		}
	})

	t.Run("ReactionResults has one entry per reaction", func(t *testing.T) {
		tRn := makeTurnWithAttack()
		reaction := makeReactionTo(tRn.GetAction().GetID())
		tRn.AddReaction(&reaction)

		res := resolver.Resolve(tRn, nil)

		if len(res.ReactionResults) != 1 {
			t.Errorf("expected 1 ReactionResult, got %d", len(res.ReactionResults))
		}
	})
}

func makeTurnWithAttack() *turn.Turn {
	a := action.Action{}
	a.Attack = &action.Attack{}
	return turn.NewTurn(a)
}
```

- [ ] **Step 2: Run tests to confirm fail**

```bash
go test ./internal/domain/match/service/... -run TestCombatResolver -v
```

Expected: FAIL — `CombatResolver` undefined.

- [ ] **Step 3: Implement `combat_resolver.go`**

Create `internal/domain/match/service/combat_resolver.go`:

```go
package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/google/uuid"
)

// TurnResolution is the snapshot of a Turn's combat result at a given moment.
// It is recalculated every time a Reaction is added to the Turn.
// The master sees each update; players see it only when the master reveals the reaction.
type TurnResolution struct {
	ActionResult    RollResult
	ReactionResults []ReactionResult
	Blows           []*battle.Blow
	// IsSettled is true once the Turn is closed and no more reactions can arrive.
	IsSettled bool
}

// RollResult holds the outcome of a single dice roll check.
type RollResult struct {
	SkillName  string
	SkillValue int
	DiceRolled []int
	Total      int
}

// ReactionResult holds the outcome of one reaction within the Turn.
type ReactionResult struct {
	ReactorID uuid.UUID
	Roll      RollResult
}

// CombatResolver is a stateless domain service that calculates the resolution
// of a Turn (its action and all current reactions) against the character sheets
// of the involved participants.
//
// Call Resolve every time the Turn state changes (on open and after each reaction).
// Full combat formula implementation is deferred — this is the correct interface.
type CombatResolver struct{}

// Resolve calculates the current resolution snapshot for the given Turn.
// sheets maps participant UUIDs to their character sheets.
// Passing nil sheets is valid during development — formulas return zero values.
func (cr CombatResolver) Resolve(t *turn.Turn, sheets map[uuid.UUID]*csSheet.Sheet) *TurnResolution {
	res := &TurnResolution{
		IsSettled: t.GetFinishedAt() != nil,
	}

	// TODO: implement ActionResult calculation using RollCalculator + sheets
	// res.ActionResult = rollCalc.Calculate(t.GetAction().Speed, sheets[t.GetAction().ActorID()])

	reactions := t.GetReactions()
	res.ReactionResults = make([]ReactionResult, len(reactions))
	for i, r := range reactions {
		// TODO: implement per-reaction resolution
		res.ReactionResults[i] = ReactionResult{ReactorID: r.ReactToID}
	}

	// TODO: populate Blows from attack/defense collision
	return res
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/domain/match/service/... -run TestCombatResolver -v
```

Expected: all PASS.

- [ ] **Step 5: Full build**

```bash
go build ./...
```

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/service/combat_resolver.go internal/domain/match/service/combat_resolver_test.go
git commit -m "$(cat <<'EOF'
feat: add CombatResolver domain service (structure + types)

Defines TurnResolution, RollResult, ReactionResult types and the
CombatResolver.Resolve interface. Full formula implementation deferred
pending game rule specification. IsSettled and ReactionResults count
are correctly derived from Turn state.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7 — `RollCalculator` Domain Service (Structure + TDD)

Same approach as Task 6: correct interface + structural tests now, full formula later.

**Files:**
- Create: `internal/domain/match/service/roll_calculator.go`
- Create: `internal/domain/match/service/roll_calculator_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/domain/match/service/roll_calculator_test.go`:

```go
package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func TestRollCalculator_Calculate(t *testing.T) {
	calc := service.RollCalculator{}

	t.Run("returns int result for a RollCheck with no sheet", func(t *testing.T) {
		check := action.RollCheck{
			SkillName:  "Velocidade",
			SkillValue: 10,
		}

		result := calc.Calculate(check, nil)

		// Formula not implemented — result is 0 for now.
		// This test ensures the method exists and accepts the correct types.
		_ = result
	})

	t.Run("does not panic when sheet is nil", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Calculate panicked with nil sheet: %v", r)
			}
		}()
		calc.Calculate(action.RollCheck{}, nil)
	})
}
```

- [ ] **Step 2: Run tests to confirm fail**

```bash
go test ./internal/domain/match/service/... -run TestRollCalculator -v
```

Expected: FAIL — `RollCalculator` undefined.

- [ ] **Step 3: Implement `roll_calculator.go`**

Create `internal/domain/match/service/roll_calculator.go`:

```go
package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

// RollCalculator is a stateless domain service that computes the final result
// of a dice roll: dice values + character skill value + modifiers from the sheet.
//
// Full formula implementation is deferred pending game rule specification.
type RollCalculator struct{}

// Calculate returns the final numeric result of the given RollCheck for the
// character with the provided sheet. sheet may be nil during development.
func (rc RollCalculator) Calculate(check action.RollCheck, sheet *csSheet.Sheet) int {
	// TODO: implement — roll dice in check.Context.Dice, apply check.SkillValue,
	// apply condition modifiers (check.Context.Condition), add sheet bonuses.
	return 0
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/domain/match/service/... -run TestRollCalculator -v
```

Expected: all PASS.

- [ ] **Step 5: Full build and all tests**

```bash
go build ./...
go test ./...
```

Expected: build clean, all tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/service/roll_calculator.go internal/domain/match/service/roll_calculator_test.go
git commit -m "$(cat <<'EOF'
feat: add RollCalculator domain service (structure)

Stateless service interface for dice + attribute + modifier calculation.
Full formula deferred pending game rule specification. Accepts RollCheck
and optional Sheet, returns int result.

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8 — Update Developer Documentation

**Files:**
- Modify: `.github/instructions/domain-map.instructions.md`
- Modify: `AGENTS.md`

- [ ] **Step 1: Update `domain-map.instructions.md`**

Replace the `Match/Combat` table entries:

```markdown
| Match/Combat Root | `internal/domain/match/` |
| Match Entities    | `internal/domain/match/entity/` |
| Match Services    | `internal/domain/match/service/` |
| Match Session     | `internal/domain/match/matchsession/` *(Phase 2)* |
| All Use Cases     | `internal/application/<feature>/` |
```

Remove the old `internal/domain/entity/match/` entry.

Update the **Current State** section:

```markdown
## Current State

- ✅ `character_sheet/` — Stable, fully tested
- ✅ `domain/match/` — Bounded context: entities + 3 domain services (Phase 1 complete)
- ⏳ `domain/match/matchsession/` — Pending Phase 2
- ✅ `gateway/` — PostgreSQL repositories (fully implemented)
- ✅ `app/api/` — HTTP handlers (unit tested)
- ✅ `app/game/` — WebSocket game server (Hub/Room/Client pattern)
- ✅ `application/` — Use cases migrated from domain/ (all features)
```

- [ ] **Step 2: Update `AGENTS.md`**

Update the architecture diagram and the `Engines` note:

```markdown
## Architecture

```
internal/
├── app/         ← Delivery: HTTP handlers (api/) + WebSocket server (game/)
├── application/ ← Use Cases: orchestrate domain + I/O (one package per feature)
├── domain/      ← Domain: entities + domain services (pure, no I/O)
│   ├── match/   ← Match bounded context (entity/, service/, matchsession/)
│   └── entity/  ← Shared entities (character_sheet/, enum/, die/, ...)
├── gateway/     ← Infrastructure: PostgreSQL repositories
└── config/      ← Configuration loading
```

Dependency rule: entity ← service ← matchsession ← usecase ← app
                 entity ← gateway
```

Replace the old `Engines = domain services...` line with:

```markdown
- **Domain Services:** stateless structs in `domain/match/service/` — receive entities,
  apply RPG rules, return results. No I/O, no state.
- **MatchSession:** stateful in-memory match state, lives in Room — see Phase 2 plan.
- **Use Cases:** in `application/<feature>/` — orchestrate domain + gateway. No RPG rules.
```

- [ ] **Step 3: Final full build and test**

```bash
go build ./...
go test ./...
```

- [ ] **Step 4: Commit**

```bash
git add .github/instructions/domain-map.instructions.md AGENTS.md
git commit -m "$(cat <<'EOF'
docs: update architecture docs to reflect Phase 1 structure

Updates domain-map and AGENTS.md to document the new bounded-context
layout (domain/match/, application/), domain services pattern, and
current state of MatchSession (Phase 2 pending).

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>
EOF
)"
```

---

## Self-Check Before Starting

Before handing off to an implementation agent:

- [ ] Spec section "Entity Layer — What Changes" → covered by Tasks 3 + 4
- [ ] Spec section "Domain Services" (RoundOrchestrator) → Task 5
- [ ] Spec section "Domain Services" (CombatResolver) → Task 6
- [ ] Spec section "Domain Services" (RollCalculator) → Task 7
- [ ] Spec "application/ vs app/" → Task 2
- [ ] Spec "Bounded context for match" → Task 3
- [ ] Spec "ActionPriorityQueue in MatchSession" → deferred to Phase 2 ✓
- [ ] Spec "MatchSession" → deferred to Phase 2 ✓
- [ ] Spec "New use cases + Room" → deferred to Phase 2 ✓
