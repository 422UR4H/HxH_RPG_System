# Tactical Map — Walls Phase 10-C Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add player wall interactions (attack/interact) as queued Actions that go through master approval; rename CombatResolver → TurnResolver; migrate wall state to MatchSession; drop `lobby_` prefix from WS message types.

**Architecture:** `TurnResolver.Resolve` routes by `TargetKind` (character vs wall) using the `TargetReader` interface, which `MatchSession` implements — preventing circular imports. Wall damage is computed by `ApplyStructuralDamage` (stateless domain service). `room.go` receives `WallResults` and broadcasts `wall_hp_changed` / `wall_state_changed`. Frontend shows action picker on wall tap and applies live HP state from WS events.

**Tech Stack:** Go 1.23 (domain services, use cases, WS delivery); React 18 + PixiJS + TypeScript (WallsLayer rendering, action picker overlay).

**Spec:** `docs/superpowers/specs/2026-06-14-tactical-map-walls-10c-design.md`

**Scope note — pieces:** `pieces` (board positions) stay in `room.go` as `r.pieces` (no `lobby_` prefix). Moving them to `MatchSession` would require domain to import delivery types; walls + gridSize migrate because they have domain logic (blocking + damage). This is a pragmatic deviation from the spec; pieces can migrate in a future cleanup.

---

## File Map

**Create:**
- `internal/domain/match/service/structural_damage.go` — `ApplyStructuralDamage` + result types
- `internal/domain/match/service/structural_damage_test.go`
- `internal/domain/match/service/wall_interact.go` — `ApplyWallInteract` pure function
- `internal/domain/match/service/wall_interact_test.go`
- `docs/game/combate/paredes.md` — player-facing wall interaction rules

**Rename:**
- `internal/domain/match/service/combat_resolver.go` → `turn_resolver.go`
- `internal/domain/match/service/combat_resolver_test.go` → `turn_resolver_test.go`

**Modify (backend):**
- `internal/domain/match/matchsession/match_session.go` — add walls/gridSize + CategorizeTarget + accessors
- `internal/application/match/open_next_action.go` — TurnResolver + pass session as TargetReader
- `internal/application/match/pull_action.go` — same as above
- `internal/app/game/message.go` — rename lobby_ types + add wall_hp_changed
- `internal/app/game/room.go` — migrate lobby_ fields, wire session on StartMatch, broadcast WallResults
- `docs/documentation-map.yaml`
- `docs/superpowers/specs/2026-06-10-tactical-map-walls-design.md` — update stale refs

**Modify (frontend):**
- `src/hooks/useLobbyWs.ts` — rename 4 event strings
- `src/hooks/useMatchWs.ts` — rename map_state_sync + add wall_hp_changed handler
- `src/pages/GamePage.tsx` — wall_hp_changed + action picker
- `src/components/organisms/WallsLayer.tsx` — visual states + rename onDoorClick → onWallClick
- `src/components/organisms/TacticalMapStage.tsx` — thread onWallClick prop
- `src/features/tactical-map/TacticalMapViewer.tsx` — thread onWallClick prop

---

## Task 1: Rename CombatResolver → TurnResolver

**Files:**
- Rename: `internal/domain/match/service/combat_resolver.go` → `turn_resolver.go`
- Rename: `internal/domain/match/service/combat_resolver_test.go` → `turn_resolver_test.go`
- Modify: `internal/domain/match/matchsession/match_session.go`
- Modify: `internal/application/match/open_next_action.go`
- Modify: `internal/application/match/pull_action.go`

- [ ] **Step 1: Delete old file + create turn_resolver.go**

Delete `internal/domain/match/service/combat_resolver.go` and create `turn_resolver.go` with the same content but renamed struct and comment:

```go
package service

import (
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// TurnResolution is the snapshot of a Turn's result — character combat, wall
// interactions, or any mix thereof.
type TurnResolution struct {
	ActionResult    RollResult
	ReactionResults []ReactionResult
	Blows           []*battle.Blow
	IsSettled       bool
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

// TurnResolver is a stateless domain service that calculates Turn resolution
// for any action type: character combat, wall attacks, door interactions, etc.
type TurnResolver struct{}

// Resolve calculates the current resolution snapshot for the given Turn.
// sheets maps participant UUIDs to their character sheets; nil is valid.
func (tr TurnResolver) Resolve(t *turn.Turn, sheets map[uuid.UUID]*csSheet.CharacterSheet) *TurnResolution {
	res := &TurnResolution{
		IsSettled: t.GetFinishedAt() != nil,
	}

	// TODO: implement ActionResult calculation using RollCalculator + sheets

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

- [ ] **Step 2: Rename + update turn_resolver_test.go**

Delete `combat_resolver_test.go` and create `turn_resolver_test.go`:

```go
package service_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestTurnResolver_Resolve(t *testing.T) {
	resolver := service.TurnResolver{}

	t.Run("returns non-nil TurnResolution for a Turn with only an action", func(t *testing.T) {
		tRn := makeTurn()
		res := resolver.Resolve(tRn, nil)
		if res == nil {
			t.Fatal("expected non-nil TurnResolution")
		}
	})

	t.Run("IsSettled is false when turn has no finishedAt", func(t *testing.T) {
		tRn := makeTurn()
		res := resolver.Resolve(tRn, nil)
		if res.IsSettled {
			t.Error("expected IsSettled=false for open turn")
		}
	})

	t.Run("IsSettled is true when turn is closed", func(t *testing.T) {
		tRn := makeTurn()
		tRn.Close(time.Now())
		res := resolver.Resolve(tRn, nil)
		if !res.IsSettled {
			t.Error("expected IsSettled=true for closed turn")
		}
	})

	t.Run("ReactionResults has one entry per reaction", func(t *testing.T) {
		tRn := makeTurn()
		act := tRn.GetAction()
		reaction := makeReactionTo((&act).GetID())
		tRn.AddReaction(reaction)

		res := resolver.Resolve(tRn, nil)

		if len(res.ReactionResults) != 1 {
			t.Errorf("expected 1 ReactionResult, got %d", len(res.ReactionResults))
		}
	})
}

func makeTurn() *turn.Turn {
	a := action.NewAction(
		uuid.New(),
		[]uuid.UUID{uuid.New()},
		uuid.Nil,
		nil,
		action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
	)
	return turn.NewTurn(*a)
}

func makeReactionTo(targetID uuid.UUID) *action.Action {
	a := action.NewAction(
		uuid.New(),
		nil,
		targetID,
		nil,
		action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil,
	)
	a.ReactToID = targetID
	return a
}
```

- [ ] **Step 3: Update match_session.go — combatRes → turnResolver**

In `internal/domain/match/matchsession/match_session.go`, change:

```go
// Field (line 25):
combatRes      service.CombatResolver
// → 
turnResolver   service.TurnResolver

// NewMatchSession (line 49):
combatRes:    service.CombatResolver{},
// →
turnResolver: service.TurnResolver{},

// NewMatchSessionWithState (line 74):
combatRes:      service.CombatResolver{},
// →
turnResolver:   service.TurnResolver{},

// AttachReaction (line 149):
return s.combatRes.Resolve(t, s.charSheets), nil
// →
return s.turnResolver.Resolve(t, s.charSheets), nil
```

- [ ] **Step 4: Update open_next_action.go**

In `internal/application/match/open_next_action.go`, change line 38:
```go
resolution := service.CombatResolver{}.Resolve(opened, nil)
// →
resolution := service.TurnResolver{}.Resolve(opened, nil)
```

- [ ] **Step 5: Update pull_action.go**

In `internal/application/match/pull_action.go`, find the equivalent call and change:
```go
resolution := service.CombatResolver{}.Resolve(opened, nil)
// →
resolution := service.TurnResolver{}.Resolve(opened, nil)
```

- [ ] **Step 6: Verify build**

```bash
go vet ./internal/...
```

Expected: no output (no errors).

- [ ] **Step 7: Run existing tests**

```bash
go test ./internal/domain/match/service/...
```

Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/domain/match/service/turn_resolver.go \
        internal/domain/match/service/turn_resolver_test.go \
        internal/domain/match/matchsession/match_session.go \
        internal/application/match/open_next_action.go \
        internal/application/match/pull_action.go
git rm internal/domain/match/service/combat_resolver.go \
       internal/domain/match/service/combat_resolver_test.go
git commit -m "refactor: rename CombatResolver → TurnResolver for generalized action resolution

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: Add Wall State + CategorizeTarget to MatchSession

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`

- [ ] **Step 1: Add imports and fields**

Add the import `mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"` to the import block in `match_session.go`.

Add three fields to the `MatchSession` struct (after `roundPersisted bool`):

```go
walls    map[string]mapentity.WallSegment // keyed by wall ID; nil until SyncMapState
gridSize float64                          // cell size in world coords; 0 until SyncMapState
```

- [ ] **Step 2: Add TargetKind constants and CategorizeTarget**

Append to `match_session.go` (after existing methods):

```go
// TargetKind identifies what kind of entity a UUID refers to in a match.
// Defined here and used by service.TargetReader — MatchSession implements the interface.
type TargetKind string

const (
	TargetKindCharacter   TargetKind = "character"    // checked first
	TargetKindWallSegment TargetKind = "wall_segment"
	TargetKindUnknown     TargetKind = "unknown"
	// TODO: TargetKindFloorTile, TargetKindItem — future phases
)

// CategorizeTarget returns the kind of entity the given UUID identifies.
// Participants are checked first so character UUIDs are never mis-routed as walls.
func (s *MatchSession) CategorizeTarget(id uuid.UUID) TargetKind {
	if _, ok := s.participants[id]; ok {
		return TargetKindCharacter
	}
	if _, ok := s.walls[id.String()]; ok {
		return TargetKindWallSegment
	}
	return TargetKindUnknown
}
```

- [ ] **Step 3: Add wall + grid accessors**

Append to `match_session.go`:

```go
// SyncMapState seeds or replaces the session's in-memory map state.
// Called by room.go when the match starts, seeding from pre-match lobby state.
func (s *MatchSession) SyncMapState(walls []mapentity.WallSegment, gridSize float64) {
	s.walls = make(map[string]mapentity.WallSegment, len(walls))
	for _, w := range walls {
		s.walls[w.ID] = w
	}
	s.gridSize = gridSize
}

func (s *MatchSession) GetWall(id string) (mapentity.WallSegment, bool) {
	w, ok := s.walls[id]
	return w, ok
}

func (s *MatchSession) UpdateWall(w mapentity.WallSegment) {
	if s.walls == nil {
		s.walls = make(map[string]mapentity.WallSegment)
	}
	s.walls[w.ID] = w
}

func (s *MatchSession) GetWalls() []mapentity.WallSegment {
	result := make([]mapentity.WallSegment, 0, len(s.walls))
	for _, w := range s.walls {
		result = append(result, w)
	}
	return result
}

func (s *MatchSession) GetGridSize() float64 { return s.gridSize }
```

- [ ] **Step 4: Verify build**

```bash
go vet ./internal/domain/match/...
```

Expected: no output.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/matchsession/match_session.go
git commit -m "feat: add wall state + CategorizeTarget to MatchSession

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3: Add TargetReader Interface + WallResult to TurnResolver

**Files:**
- Modify: `internal/domain/match/service/turn_resolver.go`
- Modify: `internal/domain/match/service/turn_resolver_test.go`
- Modify: `internal/domain/match/matchsession/match_session.go` (AttachReaction)
- Modify: `internal/application/match/open_next_action.go`
- Modify: `internal/application/match/pull_action.go`

- [ ] **Step 1: Add TargetReader, WallResultKind, WallResult to turn_resolver.go**

Add the import `mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"` to `turn_resolver.go`.

Add these types BEFORE the `TurnResolution` type, and add `WallResults` to `TurnResolution`:

```go
// TargetReader allows TurnResolver to categorize and read action targets
// without importing matchsession (prevents circular imports).
// *matchsession.MatchSession implements this interface implicitly.
type TargetReader interface {
	CategorizeTarget(id uuid.UUID) TargetKind
	GetWall(id string) (mapentity.WallSegment, bool)
}

// TargetKind mirrors matchsession.TargetKind — defined in service/ so TargetReader
// can use it without importing matchsession.
type TargetKind = matchsession.TargetKind

// WallResultKind discriminates attack vs interact outcomes in WallResult.
type WallResultKind string

const (
	WallResultKindAttack   WallResultKind = "attack"
	WallResultKindInteract WallResultKind = "interact"
)

// WallResult is the computed outcome of one player action targeting a wall.
type WallResult struct {
	UpdatedWall     mapentity.WallSegment
	EffectiveDamage int
	ReboundDamage   int // melee rebound candidate; TODO: apply to actor if melee, subtract Defense
	Kind            WallResultKind
}
```

Wait — defining `TargetKind = matchsession.TargetKind` would import matchsession from service, which matchsession already imports from service. Circular import!

**Correct approach:** Move `TargetKind` constants to a new file in `matchsession/` is wrong (same issue). Instead, define `TargetKind` in `service/` and have `MatchSession.CategorizeTarget` return `service.TargetKind`. Update Task 2's CategorizeTarget accordingly.

Replace the `TargetKind` block you added in Task 2 (`match_session.go`) with just the method — the constants live in service/:

```go
// In match_session.go — CategorizeTarget uses service.TargetKind (no local constants):
import "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"

func (s *MatchSession) CategorizeTarget(id uuid.UUID) service.TargetKind {
	if _, ok := s.participants[id]; ok {
		return service.TargetKindCharacter
	}
	if _, ok := s.walls[id.String()]; ok {
		return service.TargetKindWallSegment
	}
	return service.TargetKindUnknown
}
```

And in `turn_resolver.go`, define the full `TargetKind` constants:

```go
// TargetKind identifies the entity type a UUID refers to in an active match.
type TargetKind string

const (
	TargetKindCharacter   TargetKind = "character"
	TargetKindWallSegment TargetKind = "wall_segment"
	TargetKindUnknown     TargetKind = "unknown"
)
```

Remove the local `TargetKind` type and constants from `match_session.go` (added in Task 2 Step 2) and replace them with just `CategorizeTarget` returning `service.TargetKind`.

The full updated turn_resolver.go:

```go
package service

import (
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

// TargetKind identifies the entity type a UUID refers to in an active match.
type TargetKind string

const (
	TargetKindCharacter   TargetKind = "character"   // checked first in CategorizeTarget
	TargetKindWallSegment TargetKind = "wall_segment"
	TargetKindUnknown     TargetKind = "unknown"
	// TODO: TargetKindFloorTile, TargetKindItem — future phases
)

// TargetReader allows TurnResolver to categorize and read action targets
// without importing matchsession (prevents circular imports).
// *matchsession.MatchSession implements this interface implicitly.
type TargetReader interface {
	CategorizeTarget(id uuid.UUID) TargetKind
	GetWall(id string) (mapentity.WallSegment, bool)
}

// WallResultKind discriminates attack vs interact outcomes in WallResult.
type WallResultKind string

const (
	WallResultKindAttack   WallResultKind = "attack"
	WallResultKindInteract WallResultKind = "interact"
)

// WallResult is the computed outcome of one player action targeting a wall.
type WallResult struct {
	UpdatedWall     mapentity.WallSegment
	EffectiveDamage int
	ReboundDamage   int // melee rebound candidate; TODO: apply to actor if melee, subtract actor Defense
	Kind            WallResultKind
}

// TurnResolution is the snapshot of a Turn's result — character combat, wall
// interactions, or any mix thereof.
type TurnResolution struct {
	ActionResult    RollResult
	ReactionResults []ReactionResult
	Blows           []*battle.Blow
	WallResults     []WallResult
	IsSettled       bool
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

// TurnResolver is a stateless domain service that calculates Turn resolution
// for any action type: character combat, wall attacks, door interactions, etc.
type TurnResolver struct{}

// Resolve calculates the current resolution snapshot for the given Turn.
// sheets maps participant UUIDs to their character sheets; nil is valid.
// targets is used to categorize action targets; nil disables wall routing.
func (tr TurnResolver) Resolve(
	t *turn.Turn,
	sheets map[uuid.UUID]*csSheet.CharacterSheet,
	targets TargetReader,
) *TurnResolution {
	res := &TurnResolution{
		IsSettled: t.GetFinishedAt() != nil,
	}

	// TODO: implement ActionResult calculation using RollCalculator + sheets

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

(Wall routing is added in Task 5 after structural_damage and wall_interact exist.)

- [ ] **Step 2: Fix match_session.go TargetKind — remove local constants, use service.TargetKind**

In `match_session.go`, remove the `TargetKind` type + const block added in Task 2 Step 2. Keep only the `CategorizeTarget` method, updated to return `service.TargetKind`:

```go
// CategorizeTarget returns the kind of entity the given UUID identifies.
// Participants are checked first so character UUIDs are never mis-routed as walls.
func (s *MatchSession) CategorizeTarget(id uuid.UUID) service.TargetKind {
	if _, ok := s.participants[id]; ok {
		return service.TargetKindCharacter
	}
	if _, ok := s.walls[id.String()]; ok {
		return service.TargetKindWallSegment
	}
	return service.TargetKindUnknown
}
```

Confirm `service` is already in the import block (it is — `RoundOrchestrator` and `TurnResolver` use it).

- [ ] **Step 3: Update AttachReaction in match_session.go**

Change line 149:
```go
return s.turnResolver.Resolve(t, s.charSheets), nil
// →
return s.turnResolver.Resolve(t, s.charSheets, s), nil
```

`s` (`*MatchSession`) satisfies `service.TargetReader` because it has `CategorizeTarget(uuid.UUID) service.TargetKind` and `GetWall(string) (mapentity.WallSegment, bool)` — confirmed in Task 2.

- [ ] **Step 4: Update open_next_action.go + pull_action.go**

In `open_next_action.go` line 38:
```go
resolution := service.TurnResolver{}.Resolve(opened, nil)
// →
resolution := service.TurnResolver{}.Resolve(opened, nil, session)
```

In `pull_action.go` (same pattern):
```go
resolution := service.TurnResolver{}.Resolve(opened, nil)
// →
resolution := service.TurnResolver{}.Resolve(opened, nil, session)
```

Both already receive `session *matchsession.MatchSession` as a parameter.

- [ ] **Step 5: Update turn_resolver_test.go to pass mock TargetReader**

Add a `noopTargetReader` and update all `resolver.Resolve` calls:

```go
// add at top of test file, outside test functions:
type noopTargetReader struct{}

func (noopTargetReader) CategorizeTarget(uuid.UUID) service.TargetKind {
	return service.TargetKindUnknown
}
func (noopTargetReader) GetWall(string) (mapentity.WallSegment, bool) {
	return mapentity.WallSegment{}, false
}
```

Add the import: `mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"`

Change every `resolver.Resolve(tRn, nil)` to `resolver.Resolve(tRn, nil, noopTargetReader{})`.

- [ ] **Step 6: Verify build and tests**

```bash
go vet ./internal/...
go test ./internal/domain/match/...
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/match/service/turn_resolver.go \
        internal/domain/match/service/turn_resolver_test.go \
        internal/domain/match/matchsession/match_session.go \
        internal/application/match/open_next_action.go \
        internal/application/match/pull_action.go
git commit -m "feat: add TargetReader interface + WallResult to TurnResolver

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: Create ApplyStructuralDamage + ApplyWallInteract

**Files:**
- Create: `internal/domain/match/service/structural_damage.go`
- Create: `internal/domain/match/service/structural_damage_test.go`
- Create: `internal/domain/match/service/wall_interact.go`
- Create: `internal/domain/match/service/wall_interact_test.go`

- [ ] **Step 1: Write failing tests for ApplyStructuralDamage**

Create `internal/domain/match/service/structural_damage_test.go`:

```go
package service_test

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func TestApplyStructuralDamage(t *testing.T) {
	base := mapentity.WallSegment{
		ID:         "wall-1",
		HP:         40,
		MaxHP:      40,
		Resistance: 5,
		Destroyed:  false,
	}

	t.Run("indestructible wall (MaxHP=0) returns full rebound, HP unchanged", func(t *testing.T) {
		w := base
		w.HP = 0
		w.MaxHP = 0
		result := service.ApplyStructuralDamage(w, 20)
		if result.EffectiveDamage != 0 {
			t.Errorf("expected EffectiveDamage=0, got %d", result.EffectiveDamage)
		}
		if result.ReboundDamage != 20 {
			t.Errorf("expected ReboundDamage=20, got %d", result.ReboundDamage)
		}
		if result.UpdatedWall.Destroyed {
			t.Error("indestructible wall must not be marked Destroyed")
		}
	})

	t.Run("damage <= resistance: effective=0, rebound=rawDamage", func(t *testing.T) {
		w := base
		result := service.ApplyStructuralDamage(w, 3) // 3 < resistance(5)
		if result.EffectiveDamage != 0 {
			t.Errorf("expected EffectiveDamage=0, got %d", result.EffectiveDamage)
		}
		if result.ReboundDamage != 3 {
			t.Errorf("expected ReboundDamage=3, got %d", result.ReboundDamage)
		}
		if result.UpdatedWall.HP != 40 {
			t.Errorf("expected HP=40 (no damage), got %d", result.UpdatedWall.HP)
		}
	})

	t.Run("damage > resistance: effective=raw-resistance, HP decremented", func(t *testing.T) {
		w := base
		result := service.ApplyStructuralDamage(w, 15) // effective = 15-5 = 10
		if result.EffectiveDamage != 10 {
			t.Errorf("expected EffectiveDamage=10, got %d", result.EffectiveDamage)
		}
		if result.ReboundDamage != 5 {
			t.Errorf("expected ReboundDamage=5 (=resistance), got %d", result.ReboundDamage)
		}
		if result.UpdatedWall.HP != 30 {
			t.Errorf("expected HP=30, got %d", result.UpdatedWall.HP)
		}
		if result.UpdatedWall.Destroyed {
			t.Error("wall still has HP, must not be Destroyed")
		}
	})

	t.Run("damage brings HP to 0: Destroyed=true", func(t *testing.T) {
		w := base
		result := service.ApplyStructuralDamage(w, 45) // effective = 45-5 = 40 → HP=0
		if result.UpdatedWall.HP != 0 {
			t.Errorf("expected HP=0, got %d", result.UpdatedWall.HP)
		}
		if !result.UpdatedWall.Destroyed {
			t.Error("expected Destroyed=true when HP reaches 0")
		}
	})

	t.Run("overkill damage: HP clamped to 0, Destroyed=true", func(t *testing.T) {
		w := base
		result := service.ApplyStructuralDamage(w, 200)
		if result.UpdatedWall.HP != 0 {
			t.Errorf("expected HP=0 (clamped), got %d", result.UpdatedWall.HP)
		}
		if !result.UpdatedWall.Destroyed {
			t.Error("expected Destroyed=true")
		}
	})
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/domain/match/service/... -run TestApplyStructuralDamage
```

Expected: FAIL — `service.ApplyStructuralDamage undefined`.

- [ ] **Step 3: Implement ApplyStructuralDamage**

Create `internal/domain/match/service/structural_damage.go`:

```go
package service

import mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"

// StructuralDamageResult is the outcome of one attack on a wall segment.
type StructuralDamageResult struct {
	UpdatedWall     mapentity.WallSegment
	EffectiveDamage int // damage applied to the wall (≥ 0)
	ReboundDamage   int // = min(rawDamage, Resistance) — melee rebound candidate
	// TODO: apply ReboundDamage to actor only if attack is melee (check Attack.Category)
	// TODO: subtract actor Defense from ReboundDamage before applying
	// TODO: include ReboundDamage in broadcast (enrich wall_hp_changed or separate event)
}

// ApplyStructuralDamage applies raw attack damage to a WallSegment, respecting
// material resistance. MaxHP==0 signals an indestructible wall (no HP system).
func ApplyStructuralDamage(w mapentity.WallSegment, rawDamage int) StructuralDamageResult {
	if w.MaxHP == 0 {
		// Indestructible — no HP system; full rebound regardless of attack type.
		// TODO: if range attack (Attack.Category == "range"), ReboundDamage = 0
		return StructuralDamageResult{UpdatedWall: w, EffectiveDamage: 0, ReboundDamage: rawDamage}
	}
	effective := rawDamage - w.Resistance
	if effective < 0 {
		effective = 0
	}
	rebound := rawDamage - effective // = min(rawDamage, Resistance)
	w.HP -= effective
	if w.HP < 0 {
		w.HP = 0
	}
	if w.HP == 0 && w.MaxHP > 0 {
		w.Destroyed = true
	}
	// TODO: persist new HP state in map snapshot on turn close (see PersistTurnClose).
	return StructuralDamageResult{UpdatedWall: w, EffectiveDamage: effective, ReboundDamage: rebound}
}
```

- [ ] **Step 4: Run test — expect PASS**

```bash
go test ./internal/domain/match/service/... -run TestApplyStructuralDamage -v
```

Expected: all 5 sub-tests PASS.

- [ ] **Step 5: Write failing tests for ApplyWallInteract**

Create `internal/domain/match/service/wall_interact_test.go`:

```go
package service_test

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func TestApplyWallInteract(t *testing.T) {
	closedDoor := mapentity.WallSegment{ID: "door-1", WallType: mapentity.WallTypeDoor, Open: false, Locked: false}
	openDoor := mapentity.WallSegment{ID: "door-2", WallType: mapentity.WallTypeDoor, Open: true, Locked: false}

	t.Run("InteractOpen sets Open=true", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractOpen})
		if !ok { t.Fatal("expected ok=true") }
		if !updated.Open { t.Error("expected Open=true") }
	})

	t.Run("InteractClose sets Open=false", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(openDoor, &action.Interact{Kind: action.InteractClose})
		if !ok { t.Fatal("expected ok=true") }
		if updated.Open { t.Error("expected Open=false") }
	})

	t.Run("InteractToggle flips Open", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractToggle})
		if !ok { t.Fatal("expected ok=true") }
		if !updated.Open { t.Error("expected Open=true after toggle") }
	})

	t.Run("InteractLockpick returns ok=false (roll required, not yet handled)", func(t *testing.T) {
		_, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractLockpick})
		if ok { t.Error("expected ok=false for lockpick") }
	})

	t.Run("InteractExamine returns ok=false (roll required, not yet handled)", func(t *testing.T) {
		_, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractExamine})
		if ok { t.Error("expected ok=false for examine") }
	})
}
```

- [ ] **Step 6: Run test — expect FAIL**

```bash
go test ./internal/domain/match/service/... -run TestApplyWallInteract
```

Expected: FAIL — `service.ApplyWallInteract undefined`.

- [ ] **Step 7: Implement ApplyWallInteract**

Create `internal/domain/match/service/wall_interact.go`:

```go
package service

import (
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

// ApplyWallInteract applies an interact action to a wall segment.
// Returns the updated wall and ok=true for open/close/toggle.
// Returns ok=false for lockpick/examine — these require a skill roll (TODO).
func ApplyWallInteract(w mapentity.WallSegment, interact *action.Interact) (mapentity.WallSegment, bool) {
	switch interact.Kind {
	case action.InteractOpen:
		w.Open = true
	case action.InteractClose:
		w.Open = false
	case action.InteractToggle:
		w.Open = !w.Open
	default:
		// lockpick, examine — require roll check; not yet handled
		return w, false
	}
	return w, true
}
```

- [ ] **Step 8: Run all service tests**

```bash
go test ./internal/domain/match/service/... -v
```

Expected: all PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/domain/match/service/structural_damage.go \
        internal/domain/match/service/structural_damage_test.go \
        internal/domain/match/service/wall_interact.go \
        internal/domain/match/service/wall_interact_test.go
git commit -m "feat: add ApplyStructuralDamage + ApplyWallInteract domain services

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5: Update TurnResolver.Resolve to Route Wall Targets

**Files:**
- Modify: `internal/domain/match/service/turn_resolver.go`
- Modify: `internal/domain/match/service/turn_resolver_test.go`

- [ ] **Step 1: Write failing tests for wall routing**

Add to `turn_resolver_test.go` (new test function + mock target reader):

```go
// mockWallReader implements TargetReader with a pre-configured wall.
type mockWallReader struct {
	wallID string
	wall   mapentity.WallSegment
}

func (m mockWallReader) CategorizeTarget(id uuid.UUID) service.TargetKind {
	if id.String() == m.wallID {
		return service.TargetKindWallSegment
	}
	return service.TargetKindUnknown
}
func (m mockWallReader) GetWall(id string) (mapentity.WallSegment, bool) {
	if id == m.wallID {
		return m.wall, true
	}
	return mapentity.WallSegment{}, false
}
```

Add import `mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"` if not already there.

New test function:

```go
func TestTurnResolver_Resolve_WallTargets(t *testing.T) {
	resolver := service.TurnResolver{}
	wallID := uuid.New()
	wall := mapentity.WallSegment{
		ID:         wallID.String(),
		HP:         40,
		MaxHP:      40,
		Resistance: 5,
	}
	reader := mockWallReader{wallID: wallID.String(), wall: wall}

	t.Run("Attack target categorized as wall produces WallResult with Kind=attack", func(t *testing.T) {
		a := action.NewAction(
			uuid.New(),
			[]uuid.UUID{wallID},
			uuid.Nil,
			nil,
			action.ActionSpeed{},
			nil, nil,
			&action.Attack{}, // non-nil Attack triggers wall damage path
			nil, nil, nil, nil,
		)
		tRn := turn.NewTurn(*a)

		res := resolver.Resolve(tRn, nil, reader)

		if len(res.WallResults) != 1 {
			t.Fatalf("expected 1 WallResult, got %d", len(res.WallResults))
		}
		if res.WallResults[0].Kind != service.WallResultKindAttack {
			t.Errorf("expected Kind=attack, got %s", res.WallResults[0].Kind)
		}
		if res.WallResults[0].UpdatedWall.ID != wallID.String() {
			t.Errorf("UpdatedWall.ID mismatch")
		}
	})

	t.Run("Interact (open) target categorized as wall produces WallResult with Kind=interact", func(t *testing.T) {
		a := action.NewAction(
			uuid.New(),
			[]uuid.UUID{wallID},
			uuid.Nil,
			nil,
			action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil,
			&action.Interact{Kind: action.InteractOpen},
		)
		tRn := turn.NewTurn(*a)

		res := resolver.Resolve(tRn, nil, reader)

		if len(res.WallResults) != 1 {
			t.Fatalf("expected 1 WallResult, got %d", len(res.WallResults))
		}
		if res.WallResults[0].Kind != service.WallResultKindInteract {
			t.Errorf("expected Kind=interact, got %s", res.WallResults[0].Kind)
		}
		if !res.WallResults[0].UpdatedWall.Open {
			t.Error("expected UpdatedWall.Open=true after InteractOpen")
		}
	})

	t.Run("nil targets skips wall routing", func(t *testing.T) {
		a := action.NewAction(
			uuid.New(),
			[]uuid.UUID{wallID},
			uuid.Nil,
			nil,
			action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil,
		)
		tRn := turn.NewTurn(*a)

		res := resolver.Resolve(tRn, nil, nil) // nil TargetReader

		if len(res.WallResults) != 0 {
			t.Errorf("expected no WallResults when targets=nil, got %d", len(res.WallResults))
		}
	})
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/domain/match/service/... -run TestTurnResolver_Resolve_WallTargets
```

Expected: FAIL — wall routing not implemented yet.

- [ ] **Step 3: Update Resolve to route wall targets**

In `turn_resolver.go`, update the `Resolve` method body. Insert the wall routing block right after `res := &TurnResolution{...}` and before the TODO comment:

```go
func (tr TurnResolver) Resolve(
	t *turn.Turn,
	sheets map[uuid.UUID]*csSheet.CharacterSheet,
	targets TargetReader,
) *TurnResolution {
	res := &TurnResolution{
		IsSettled: t.GetFinishedAt() != nil,
	}

	if targets != nil {
		a := t.GetAction()
		for _, targetID := range a.TargetID {
			switch targets.CategorizeTarget(targetID) {
			case TargetKindCharacter:
				// TODO: implement character combat rolls (existing path)

			case TargetKindWallSegment:
				wall, ok := targets.GetWall(targetID.String())
				if !ok {
					continue
				}
				if a.Attack != nil {
					rawDamage := 0 // TODO: extract from a.Attack.Damage roll when contrato finalizar
					sdr := ApplyStructuralDamage(wall, rawDamage)
					res.WallResults = append(res.WallResults, WallResult{
						UpdatedWall:     sdr.UpdatedWall,
						EffectiveDamage: sdr.EffectiveDamage,
						ReboundDamage:   sdr.ReboundDamage,
						Kind:            WallResultKindAttack,
					})
				}
				if a.Interact != nil {
					updated, ok := ApplyWallInteract(wall, a.Interact)
					if ok {
						res.WallResults = append(res.WallResults, WallResult{
							UpdatedWall: updated,
							Kind:        WallResultKindInteract,
						})
					}
				}

			case TargetKindUnknown:
				// TODO: record unknown-target error in resolution for caller to surface
			}
		}
	}

	// TODO: implement ActionResult calculation using RollCalculator + sheets

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

- [ ] **Step 4: Run all service tests — expect PASS**

```bash
go test ./internal/domain/match/service/... -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/turn_resolver.go \
        internal/domain/match/service/turn_resolver_test.go
git commit -m "feat: route wall targets in TurnResolver.Resolve via TargetReader

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6: Rename lobby_ MessageTypes + Add wall_hp_changed

**Files:**
- Modify: `internal/app/game/message.go`
- Modify: `internal/app/game/room.go`

- [ ] **Step 1: Rename MessageType constants in message.go**

In `message.go`, apply these renames:

| Old constant | New constant |
|---|---|
| `MsgTypeLobbyPieceMoved` | `MsgTypePieceMoved` |
| `MsgTypeLobbyPieceRemoved` | `MsgTypePieceRemoved` |
| `MsgTypeLobbyStateSync` | `MsgTypeMapStateSync` |
| `MsgTypeLobbyFullState` | `MsgTypeMapFullState` |

Old string values → new string values:

| Old string | New string |
|---|---|
| `"lobby_piece_moved"` | `"piece_moved"` |
| `"lobby_piece_removed"` | `"piece_removed"` |
| `"lobby_state_sync"` | `"map_state_sync"` |
| `"lobby_full_state"` | `"map_full_state"` |

Remove the comment `// lobby_ prefix distinguishes from future in-game events (Phase 7+).` as it's now obsolete.

Add `MsgTypeWallHpChanged`:

```go
// Server → Client (wall HP/structural events)
MsgTypeWallHpChanged MessageType = "wall_hp_changed"
```

- [ ] **Step 2: Rename payload types in message.go**

| Old Go type | New Go type |
|---|---|
| `LobbyPieceMovedPayload` | `PieceMovedPayload` |
| `LobbyPieceRemovedPayload` | `PieceRemovedPayload` |
| `LobbyPiecesPayload` | `MapPiecesPayload` |
| `LobbyStateSyncPayload` | `MapStateSyncPayload` |

Update the struct definitions and their doc comments accordingly.

- [ ] **Step 3: Add WallHpChangedPayload**

Append to `message.go` (near `WallStateChangedPayload`):

```go
// WallHpChangedPayload is broadcast to all clients when a wall's HP or destroyed state changes.
type WallHpChangedPayload struct {
	WallID    string `json:"wall_id"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"max_hp"`
	Destroyed bool   `json:"destroyed"`
}
```

- [ ] **Step 4: Update room.go references**

In `room.go`, update every occurrence:
- `MsgTypeLobbyPieceMoved` → `MsgTypePieceMoved`
- `MsgTypeLobbyPieceRemoved` → `MsgTypePieceRemoved`
- `MsgTypeLobbyStateSync` → `MsgTypeMapStateSync`
- `MsgTypeLobbyFullState` → `MsgTypeMapFullState`
- `LobbyPieceMovedPayload` → `PieceMovedPayload`
- `LobbyPieceRemovedPayload` → `PieceRemovedPayload`
- `LobbyPiecesPayload` → `MapPiecesPayload`
- `LobbyStateSyncPayload` → `MapStateSyncPayload`

- [ ] **Step 5: Verify build**

```bash
go vet ./internal/app/game/...
```

Expected: no output.

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/message.go internal/app/game/room.go
git commit -m "refactor: remove lobby_ prefix from WS MessageTypes + add wall_hp_changed

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 7: Migrate room.go — Walls to Session, Wire SyncMapState

**Files:**
- Modify: `internal/app/game/room.go`

- [ ] **Step 1: Rename lobby fields in Room struct**

In the `Room` struct, rename the three fields (keep them in room.go for pre-match lobby state; they transfer to session on StartMatch):

```go
// Before:
lobbyPieces  map[string]LobbyPieceMovedPayload
lobbyWalls   map[string]mapentity.WallSegment
lobbyGridSize float64

// After:
pieces   map[string]PieceMovedPayload  // board state before/during lobby
walls    map[string]mapentity.WallSegment  // wall state before StartMatch; seeded to session on StartMatch
gridSize float64                           // cell size; seeded to session on StartMatch
```

Also remove the now-redundant `import mapentity` if the only reference was through these fields (check — it's still needed since `applyWallInteract` references it). Keep the import.

- [ ] **Step 2: Update NewRoom initializer**

In `NewRoom`:
```go
// Before:
lobbyPieces:           make(map[string]LobbyPieceMovedPayload),
lobbyWalls:            make(map[string]mapentity.WallSegment),
lobbyGridSize:         64,

// After:
pieces:   make(map[string]PieceMovedPayload),
walls:    make(map[string]mapentity.WallSegment),
gridSize: 64,
```

- [ ] **Step 3: Wire SyncMapState in StartMatch**

In `StartMatch`, after `r.session = session`:

```go
r.mu.Lock()
r.session = session
// Seed the session with pre-match map state collected during lobby.
r.session.SyncMapState(
    func() []mapentity.WallSegment {
        ws := make([]mapentity.WallSegment, 0, len(r.walls))
        for _, w := range r.walls {
            ws = append(ws, w)
        }
        return ws
    }(),
    r.gridSize,
)
r.state = RoomStatePlaying
r.mu.Unlock()
```

- [ ] **Step 4: Update movement blocking to prefer session**

In `handleClientMessage` case `MsgTypeEnqueueAction`, find the movement blocking block and update to use session when available:

```go
if a.Move != nil {
    from := a.Move.From
    to := a.Move.Position
    if from != ([3]int{}) {
        r.mu.RLock()
        var gridSize float64
        var walls []mapentity.WallSegment
        if r.session != nil {
            gridSize = r.session.GetGridSize()
            walls = r.session.GetWalls()
        } else {
            gridSize = r.gridSize
            walls = make([]mapentity.WallSegment, 0, len(r.walls))
            for _, w := range r.walls {
                walls = append(walls, w)
            }
        }
        r.mu.RUnlock()
        fromWorld := [2]float64{float64(from[0]) * gridSize, float64(from[1]) * gridSize}
        toWorld := [2]float64{float64(to[0]) * gridSize, float64(to[1]) * gridSize}
        if mapservice.IsPathBlocked(fromWorld, toWorld, walls) {
            client.SendMessage(NewErrorMessage("move_blocked", "movement blocked by a wall"))
            return
        }
    }
}
```

- [ ] **Step 5: Update applyWallInteract to use session when available**

Replace the `applyWallInteract` method body:

```go
func (r *Room) applyWallInteract(wallID string, interact *action.Interact) (open, locked bool, ok bool) {
    r.mu.Lock()
    defer r.mu.Unlock()
    var w mapentity.WallSegment
    var exists bool
    if r.session != nil {
        w, exists = r.session.GetWall(wallID)
    } else {
        w, exists = r.walls[wallID]
    }
    if !exists {
        return false, false, false
    }
    updated, interactOK := mapservice_domain.ApplyWallInteract(w, interact)
    if !interactOK {
        return false, false, false
    }
    if r.session != nil {
        r.session.UpdateWall(updated)
    } else {
        r.walls[wallID] = updated
    }
    return updated.Open, updated.Locked, true
}
```

Add import alias for the domain service (to distinguish from mapservice):
```go
import (
    ...
    domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)
```

And use `domainservice.ApplyWallInteract(w, interact)` in the method.

- [ ] **Step 6: Update handleClientMessage — map state sync and full state handlers**

In the `MsgTypeMapStateSync` case, update `r.lobbyPieces` → `r.pieces`, `r.lobbyWalls` → `r.walls`, `r.lobbyGridSize` → `r.gridSize`. If `r.session != nil`, also call `r.session.SyncMapState(walls, gridSize)`:

```go
case MsgTypeMapStateSync:
    if !r.IsMaster(client.userUUID) {
        client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
        return
    }
    var payload MapStateSyncPayload
    if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
        client.SendMessage(NewErrorMessage("invalid_payload", "invalid map_state_sync payload"))
        return
    }
    r.mu.Lock()
    r.pieces = make(map[string]PieceMovedPayload, len(payload.Pieces))
    for _, p := range payload.Pieces {
        r.pieces[p.PieceID] = p
    }
    r.walls = make(map[string]mapentity.WallSegment, len(payload.Walls))
    for _, w := range payload.Walls {
        r.walls[w.ID] = w
    }
    if payload.Grid != nil && payload.Grid.CellSize > 0 {
        r.gridSize = payload.Grid.CellSize
    }
    if r.session != nil {
        wallSlice := make([]mapentity.WallSegment, 0, len(r.walls))
        for _, w := range r.walls {
            wallSlice = append(wallSlice, w)
        }
        r.session.SyncMapState(wallSlice, r.gridSize)
    }
    r.mu.Unlock()
```

Update `sendLobbyFullState` → `sendMapFullState` (rename the method), and update the `Run()` method's `sendLobbyFullState` call:

```go
func (r *Room) sendMapFullState(client *Client) {
    r.mu.RLock()
    pieces := make([]PieceMovedPayload, 0, len(r.pieces))
    for _, p := range r.pieces {
        pieces = append(pieces, p)
    }
    r.mu.RUnlock()
    msg := NewServerMessage(MsgTypeMapFullState, MapPiecesPayload{Pieces: pieces})
    client.SendMessage(msg)
}
```

In `Run()`, update the `hasPieces` check:
```go
r.mu.RLock()
hasPieces := len(r.pieces) > 0
r.mu.RUnlock()
if hasPieces {
    r.sendMapFullState(client)
}
```

Update `MsgTypePieceMoved` and `MsgTypePieceRemoved` handlers in `handleClientMessage` to use `r.pieces` instead of `r.lobbyPieces`.

- [ ] **Step 7: Verify build**

```bash
go vet ./internal/app/game/...
```

Expected: no output.

- [ ] **Step 8: Commit**

```bash
git add internal/app/game/room.go
git commit -m "refactor: migrate room.go lobby state → session; wire SyncMapState on StartMatch

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 8: Update OpenNextActionUC + Broadcast WallResults in room.go

**Files:**
- Modify: `internal/application/match/open_next_action.go`
- Modify: `internal/app/game/room.go`

Both open_next_action and pull_action were already updated in Task 3 (pass `session` to Resolve). This task adds the wall result processing in room.go.

- [ ] **Step 1: Add wall event broadcasting to OpenNextAction handler in room.go**

In `handleClientMessage`, case `MsgTypeOpenNextAction`, after broadcasting `turn_opened`:

```go
// Broadcast wall events for each WallResult.
for _, wr := range result.Resolution.WallResults {
    // Apply wall state change to session (Resolve computed but did not mutate session).
    r.mu.Lock()
    session.UpdateWall(wr.UpdatedWall)
    r.mu.Unlock()

    var evt Message
    switch wr.Kind {
    case domainservice.WallResultKindAttack:
        evt = NewServerMessage(MsgTypeWallHpChanged, WallHpChangedPayload{
            WallID:    wr.UpdatedWall.ID,
            HP:        wr.UpdatedWall.HP,
            MaxHP:     wr.UpdatedWall.MaxHP,
            Destroyed: wr.UpdatedWall.Destroyed,
        })
    case domainservice.WallResultKindInteract:
        evt = NewServerMessage(MsgTypeWallStateChanged, WallStateChangedPayload{
            WallID: wr.UpdatedWall.ID,
            Open:   wr.UpdatedWall.Open,
            Locked: wr.UpdatedWall.Locked,
        })
    default:
        continue
    }
    data, _ := json.Marshal(evt)
    go func(d []byte) { r.broadcast <- d }(data)
}
```

- [ ] **Step 2: Apply the same block to PullAction handler**

In case `MsgTypePullAction`, add the identical wall broadcasting block after broadcasting `turn_opened`.

- [ ] **Step 3: Verify build**

```bash
go vet ./internal/...
```

Expected: no output.

- [ ] **Step 4: Run all tests**

```bash
go test ./internal/...
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/room.go internal/application/match/open_next_action.go \
        internal/application/match/pull_action.go
git commit -m "feat: broadcast wall_hp_changed and wall_state_changed from player Actions

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 9: Update Docs

**Files:**
- Modify: `docs/documentation-map.yaml`
- Modify: `docs/superpowers/specs/2026-06-10-tactical-map-walls-design.md`
- Create: `docs/game/combate/paredes.md`

- [ ] **Step 1: Update documentation-map.yaml**

Add the following entries to the `mappings:` list:

```yaml
  # ─── Match: Domain Services (TurnResolver, structural damage) ───
  - code_path: internal/domain/match/service/turn_resolver.go
    dev_docs:
      - path: docs/dev/match/turns-rounds.md
        confidence: directly_affected
    notes: TurnResolver — routes Action targets (character vs wall) and computes resolution

  - code_path: internal/domain/match/service/structural_damage.go
    dev_docs:
      - path: docs/dev/match/turns-rounds.md
        confidence: directly_affected
    game_docs:
      - path: docs/game/combate/paredes.md
        confidence: directly_affected
    notes: ApplyStructuralDamage — wall HP, resistance, destroyed flag, rebound candidate

  - code_path: internal/domain/match/service/wall_interact.go
    dev_docs:
      - path: docs/dev/match/actions.md
        confidence: directly_affected
    game_docs:
      - path: docs/game/combate/paredes.md
        confidence: directly_affected
    notes: ApplyWallInteract — open/close/toggle; lockpick/examine are TODO (roll check)
```

Update the existing `internal/app/game/message.go` entry notes to reference `wall_hp_changed` and renamed message types.

Update the existing `internal/app/game/room.go` entry notes to reference `map_state_sync`, `map_full_state`, `piece_moved`, `piece_removed` (was `lobby_*`).

- [ ] **Step 2: Fix stale CombatResolver references in walls-design.md**

In `docs/superpowers/specs/2026-06-10-tactical-map-walls-design.md`, find and replace any occurrence of `CombatResolver` with `TurnResolver`, and remove any mention of "turn engine". Add a note at the top:

```markdown
> **Nota de atualização (Fase 10-C):** `CombatResolver` foi renomeado para `TurnResolver`.
> `TurnResolver` suporta alvos do tipo `TargetKindCharacter` e `TargetKindWallSegment`.
```

- [ ] **Step 3: Create docs/game/combate/paredes.md**

```markdown
# Paredes, Portas e Obstáculos

As paredes do mapa tático são alvos válidos para ações de jogadores. Toda ação de interação com paredes entra na **fila de prioridade** e é aberta pelo mestre — respeitando a ordem de iniciativa.

## Tipos de Parede

| Tipo | Descrição |
|---|---|
| `wall` / `terrain` | Parede sólida ou terreno. Pode ser atacada (se tiver HP). |
| `door` | Porta. Pode ser aberta, fechada ou atacada. |
| `window` | Janela. Comporta-se como porta para fins de ação. |
| `secret_door` | Porta secreta. Invisível para jogadores — não aparece no menu de ação. |

## Ações Disponíveis por Tipo

| Tipo de parede | Ações visíveis para o jogador |
|---|---|
| `wall` / `terrain` | "Atacar" (se `maxHp > 0` e não destruída) |
| `door` (fechada, destrancada) | "Abrir", "Atacar" |
| `door` (trancada) | "Arrombar fechadura", "Atacar" |
| `window` | "Abrir", "Atacar" |

## Resistência e Dano

Cada parede tem:
- **HP** — pontos de vida atuais
- **HP Máximo** — `0` significa indestrutível
- **Resistência** — subtrai do dano bruto do ataque

Fórmula: `dano efetivo = max(0, dano bruto − resistência)`

Se o dano bruto não ultrapassar a resistência, a parede não é danificada. O excesso retorna como **dano rebote** para o atacante (apenas ataques corpo-a-corpo — regra completa em implementação futura).

## Estados Visuais

| Estado | Condição | Visual |
|---|---|---|
| Intacta | `hp == maxHp` | Cor cheia, opacidade 1.0 |
| Danificada | `0 < hp < maxHp` | Tracejada, opacidade 0.8 |
| Destruída | `destroyed == true` | Pontilhada fina, opacidade 0.4, marcas × |
| Indestrutível | `maxHp == 0` | Cor cheia, opacidade 1.0 (sem barra de HP) |

> 🔧 Para Desenvolvedores: `docs/dev/match/turns-rounds.md` — fluxo de Action + TurnResolver.
```

- [ ] **Step 4: Commit**

```bash
git add docs/documentation-map.yaml \
        docs/superpowers/specs/2026-06-10-tactical-map-walls-design.md \
        docs/game/combate/paredes.md
git commit -m "docs: update map for 10-C, fix stale CombatResolver refs, add paredes.md

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 10: Frontend — Update useLobbyWs.ts Event Strings

**Files:**
- Modify: `src/hooks/useLobbyWs.ts`

- [ ] **Step 1: Rename receive-side case labels**

In `ws.onmessage` switch, rename:
- `case "lobby_piece_moved":` → `case "piece_moved":`
- `case "lobby_piece_removed":` → `case "piece_removed":`
- `case "lobby_full_state":` → `case "map_full_state":`

- [ ] **Step 2: Rename send-side strings in callbacks**

In `sendPieceMoved`:
```ts
sendMessage("lobby_piece_moved", { ... })
// →
sendMessage("piece_moved", { ... })
```

In `sendPieceRemoved`:
```ts
sendMessage("lobby_piece_removed", { piece_id: pieceId })
// →
sendMessage("piece_removed", { piece_id: pieceId })
```

In `sendLobbySync`:
```ts
sendMessage("lobby_state_sync", { ... })
// →
sendMessage("map_state_sync", { ... })
```

Also update the comment on line 22 from `// Shape of each piece entry inside lobby_full_state` to `// Shape of each piece entry inside map_full_state`.

- [ ] **Step 3: Build check**

```bash
cd System_X_System_React && npm run build
```

Expected: exits 0, no TypeScript errors.

- [ ] **Step 4: Commit**

```bash
git add src/hooks/useLobbyWs.ts
git commit -m "refactor: rename lobby_* WS event strings → piece_moved, map_state_sync, etc.

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 11: Frontend — Update useMatchWs.ts + Add wall_hp_changed

**Files:**
- Modify: `src/hooks/useMatchWs.ts`

- [ ] **Step 1: Rename lobby_state_sync → map_state_sync in sendWallSync**

In `sendWallSync` callback (line 56):
```ts
sendRaw("lobby_state_sync", { ... })
// →
sendRaw("map_state_sync", { ... })
```

- [ ] **Step 2: Add onWallHpChanged option + handler**

Add to `UseMatchWsOptions`:
```ts
/** Called when the server broadcasts a wall HP / destroyed change (attack result). */
onWallHpChanged?: (wallId: string, hp: number, maxHp: number, destroyed: boolean) => void;
```

Add to the destructure in `useMatchWs`:
```ts
const onWallHpChangedRef = useRef(onWallHpChanged);
onWallHpChangedRef.current = onWallHpChanged;
```

In `ws.onmessage`, extend the if-else chain:
```ts
if (msg.type === "wall_state_changed") {
    const p = msg.payload as WallStateChangedPayload;
    onWallStateChangedRef.current?.(p.wall_id, p.open, p.locked);
} else if (msg.type === "wall_hp_changed") {
    const p = msg.payload as { wall_id: string; hp: number; max_hp: number; destroyed: boolean };
    onWallHpChangedRef.current?.(p.wall_id, p.hp, p.max_hp, p.destroyed);
}
```

- [ ] **Step 3: Extend sendAction payload type**

Update the `sendAction` return type signature to include an optional `attack` field:

```ts
const sendAction = useCallback(
    (payload: {
        target_id?: string[];
        interact?: { kind: string };
        move?: { from: [number, number, number]; position: [number, number, number]; category: string };
        attack?: { hit: { skill_name: string }; damage: { skill_name: string } };
    }) => {
        sendRaw("enqueue_action", payload);
    },
    [sendRaw],
);
```

- [ ] **Step 4: Build check**

```bash
npm run build
```

Expected: exits 0.

- [ ] **Step 5: Commit**

```bash
git add src/hooks/useMatchWs.ts
git commit -m "feat: add wall_hp_changed handler + attack type in useMatchWs

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 12: Frontend — GamePage.tsx Wall HP Handler + Action Picker

**Files:**
- Modify: `src/pages/GamePage.tsx`
- Modify: `src/features/tactical-map/TacticalMapViewer.tsx`
- Modify: `src/components/organisms/TacticalMapStage.tsx`
- Modify: `src/components/organisms/WallsLayer.tsx`

This task threads `onWallClick` (replacing `onDoorClick`) from WallsLayer up to GamePage, and adds the action picker state + overlay in GamePage.

- [ ] **Step 1: Update WallsLayer — replace onDoorClick with onWallClick**

In `WallsLayer.tsx` Props type:
```ts
// Remove:
onDoorClick?: (wallId: string) => void;
// Add:
onWallClick?: (wall: WallSegment) => void;
```

In the component args destructure:
```ts
// Remove onDoorClick, add:
onWallClick,
```

Update `onDoorClickRef` → `onWallClickRef`:
```ts
const onWallClickRef = useRef(onWallClick);
onWallClickRef.current = onWallClick;
```

In `handleViewerClick` effect (around line 279):
```ts
// Before:
const hit = findNearestWall([rawPt.x, rawPt.y], wallsRef.current, HIT / vpScaleRef.current);
if (hit && (hit.wallType === "door" || hit.wallType === "window")) {
    onDoorClickRef.current(hit.id);
}

// After:
const hit = findNearestWall([rawPt.x, rawPt.y], wallsRef.current, HIT / vpScaleRef.current);
if (hit) {
    onWallClickRef.current?.(hit);
}
```

- [ ] **Step 2: Update TacticalMapStage — rename onDoorClick → onWallClick**

In `TacticalMapStage.tsx`, find the `onDoorClick?: (wallId: string) => void` prop definition and rename it:
```ts
onWallClick?: (wall: WallSegment) => void;
```

Update the prop destructure and pass-through to `WallsLayer`:
```ts
// In destructure:
onWallClick,
// In WallsLayer props:
onWallClick={onWallClick}
```

- [ ] **Step 3: Update TacticalMapViewer — rename onDoorClick → onWallClick**

```ts
// Props type:
type Props = {
  map: TacticalMap;
  width: number;
  height: number;
  npcMap?: Map<string, CharacterPrivateSummary>;
  onWallClick?: (wall: WallSegment) => void;
};

export default function TacticalMapViewer({ map, width, height, npcMap, onWallClick }: Props) {
  return <TacticalMapStage map={map} width={width} height={height} npcMap={npcMap} onWallClick={onWallClick} />;
}
```

Add `import type { WallSegment } from "../../types/tacticalMap";` if not already there.

- [ ] **Step 4: Update GamePage.tsx — wall HP handler + action picker**

Add `useState` to imports (already imported). Add `WallSegment` type import if not already:
```ts
import type { WallSegment } from "../types/tacticalMap";
```

Add `handleWallHpChanged` callback:
```ts
const handleWallHpChanged = useCallback((wallId: string, hp: number, maxHp: number, destroyed: boolean) => {
    setLiveWalls((prev) =>
        prev.map((w) => (w.id === wallId ? { ...w, hp, maxHp, destroyed } : w)),
    );
}, []);
```

Add to `useMatchWs` options:
```ts
onWallHpChanged: handleWallHpChanged,
```

Add action picker state:
```ts
const [wallPicker, setWallPicker] = useState<WallSegment | null>(null);
```

Replace `handleDoorClick` with `handleWallClick`:
```ts
const handleWallClick = useCallback(
    (wall: WallSegment) => {
        if (isMaster) {
            if (wall.wallType === "door" || wall.wallType === "window") {
                sendMasterAction({ target_ids: [wall.id], interact: { kind: "toggle" } });
            }
        } else {
            setWallPicker(wall);
        }
    },
    [isMaster, sendMasterAction],
);
```

Update `TacticalMapViewer` usage:
```tsx
<TacticalMapViewer
    map={{ ...map, walls: liveWalls }}
    width={width}
    height={height}
    npcMap={npcMap}
    onWallClick={handleWallClick}
/>
```

Add action picker overlay JSX (inside the component return, after `<GamePageTemplate ...>`):
```tsx
{wallPicker && (
    <WallActionOverlay onClick={() => setWallPicker(null)}>
        <WallActionMenu onClick={(e) => e.stopPropagation()}>
            <WallActionTitle>
                {wallPicker.wallType === "door" ? "Porta" : wallPicker.wallType === "window" ? "Janela" : "Parede"}
            </WallActionTitle>
            {(wallPicker.wallType === "door" || wallPicker.wallType === "window") && !wallPicker.locked && (
                <WallActionButton onClick={() => {
                    const intent = wallPicker.open ? "close" : "open";
                    sendAction({ target_id: [wallPicker.id], interact: { kind: intent } });
                    setWallPicker(null);
                }}>
                    {wallPicker.open ? "Fechar" : "Abrir"}
                </WallActionButton>
            )}
            {wallPicker.wallType === "door" && wallPicker.locked && (
                <WallActionButton onClick={() => {
                    sendAction({ target_id: [wallPicker.id], interact: { kind: "lockpick" } });
                    setWallPicker(null);
                }}>
                    Arrombar fechadura
                </WallActionButton>
            )}
            {wallPicker.maxHp > 0 && !wallPicker.destroyed && (
                <WallActionButton onClick={() => {
                    sendAction({
                        target_id: [wallPicker.id],
                        attack: {
                            hit: { skill_name: "combat_strength" },    // TODO: player skill selection
                            damage: { skill_name: "combat_strength" }, // TODO: player skill selection
                        },
                    });
                    setWallPicker(null);
                }}>
                    Atacar
                </WallActionButton>
            )}
            <WallActionCancel onClick={() => setWallPicker(null)}>Cancelar</WallActionCancel>
        </WallActionMenu>
    </WallActionOverlay>
)}
```

Add styled components at the bottom of GamePage.tsx:
```ts
const WallActionOverlay = styled.div`
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
`;

const WallActionMenu = styled.div`
    background: ${colors.surfaceSidebar};
    border: 1px solid ${colors.grayMid};
    border-radius: 8px;
    padding: 16px;
    min-width: 200px;
    display: flex;
    flex-direction: column;
    gap: 8px;
`;

const WallActionTitle = styled.h3`
    font-family: ${fonts.display};
    font-size: 14px;
    color: ${colors.textMuted};
    text-transform: uppercase;
    letter-spacing: 1px;
    margin: 0 0 4px;
`;

const WallActionButton = styled.button`
    background: ${colors.brandPrimary};
    color: ${colors.textPrimary};
    font-family: ${fonts.sans};
    font-size: 14px;
    border: none;
    border-radius: 4px;
    padding: 8px 12px;
    cursor: pointer;
    text-align: left;
    &:hover { opacity: 0.85; }
`;

const WallActionCancel = styled.button`
    background: transparent;
    color: ${colors.textMuted};
    font-family: ${fonts.sans};
    font-size: 13px;
    border: 1px solid ${colors.grayMid};
    border-radius: 4px;
    padding: 6px 12px;
    cursor: pointer;
    margin-top: 4px;
    &:hover { background: ${colors.grayMid}; }
`;
```

- [ ] **Step 5: Build check**

```bash
npm run build
```

Expected: exits 0.

- [ ] **Step 6: Smoke test in browser**

Start dev servers: `./dev-checkout.sh <current-branch>` from project root, or manually `make dev-game` + `npm run dev`.

Open http://localhost:5173, navigate to a match with a map. As a non-master player:
- Click on a door → action picker opens with "Abrir"/"Fechar" + "Atacar" options
- Click on a wall with HP → action picker shows only "Atacar"
- Click "Cancelar" or outside → picker closes
- As master: clicking door still toggles it directly (no picker)

- [ ] **Step 7: Commit**

```bash
git add src/pages/GamePage.tsx \
        src/features/tactical-map/TacticalMapViewer.tsx \
        src/components/organisms/TacticalMapStage.tsx \
        src/components/organisms/WallsLayer.tsx
git commit -m "feat: add wall action picker + wall_hp_changed live state in GamePage

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 13: Frontend — WallsLayer.tsx Visual States

**Files:**
- Modify: `src/components/organisms/WallsLayer.tsx`

- [ ] **Step 1: Add damaged + destroyed visual logic in drawMaterial**

In the `drawMaterial` callback (inside `for (const w of walls)` loop), replace the alpha calculation and rendering block:

```ts
// Before:
const alpha = w.destroyed ? 0.4 : 1.0;
if (w.wallType === "secret_door") {
    drawDashedLine(g, a1, a2, color, width, alpha);
} else if (w.wallType === "terrain") {
    drawDottedLine(g, a1, a2, color, width, alpha);
} else if (w.wallType === "door" && w.open) {
    drawOpenDoor(g, a1, a2, color, width, alpha);
} else {
    g.setStrokeStyle({ color, width, alpha });
    g.moveTo(a1.x, a1.y); g.lineTo(a2.x, a2.y); g.stroke();
}

// After:
const isDamaged = w.maxHp > 0 && w.hp > 0 && w.hp < w.maxHp;
const isDestroyed = w.destroyed;

if (isDestroyed) {
    drawDestroyedWall(g, a1, a2, color, width, vpScale);
} else if (w.wallType === "secret_door") {
    const alpha = 1.0;
    drawDashedLine(g, a1, a2, color, width, alpha);
} else if (w.wallType === "terrain") {
    const alpha = isDamaged ? 0.8 : 1.0;
    drawDottedLine(g, a1, a2, color, width, alpha);
} else if (w.wallType === "door" && w.open) {
    const alpha = isDamaged ? 0.8 : 1.0;
    drawOpenDoor(g, a1, a2, color, width, alpha);
} else if (isDamaged) {
    drawDashedLine(g, a1, a2, color, width, 0.8);
} else {
    g.setStrokeStyle({ color, width, alpha: 1.0 });
    g.moveTo(a1.x, a1.y); g.lineTo(a2.x, a2.y); g.stroke();
}
```

- [ ] **Step 2: Add drawDestroyedWall helper**

Add this function at the bottom of the file (near other drawing helpers):

```ts
function drawDestroyedWall(
    g: import("pixi.js").Graphics,
    a1: { x: number; y: number },
    a2: { x: number; y: number },
    color: number,
    width: number,
    vpScale: number,
) {
    const dx = a2.x - a1.x, dy = a2.y - a1.y;
    const totalLen = Math.hypot(dx, dy);
    if (totalLen < 0.1) return;
    const ux = dx / totalLen, uy = dy / totalLen;
    // Fine dotted line — very small dots with large gaps
    const dotLen = 1, gapLen = 7;
    let t = 0, drawing = true;
    while (t < totalLen) {
        const end = Math.min(t + (drawing ? dotLen : gapLen), totalLen);
        if (drawing) {
            g.setStrokeStyle({ color, width, alpha: 0.4 });
            g.moveTo(a1.x + t * ux, a1.y + t * uy);
            g.lineTo(a1.x + end * ux, a1.y + end * uy);
            g.stroke();
        }
        t = end;
        drawing = !drawing;
    }
    // × marks at endpoints
    const xSize = Math.max(3, 6 / vpScale);
    for (const pt of [a1, a2]) {
        g.setStrokeStyle({ color, width: Math.max(1, 1.5 / vpScale), alpha: 0.4 });
        g.moveTo(pt.x - xSize, pt.y - xSize); g.lineTo(pt.x + xSize, pt.y + xSize); g.stroke();
        g.moveTo(pt.x + xSize, pt.y - xSize); g.lineTo(pt.x - xSize, pt.y + xSize); g.stroke();
    }
}
```

- [ ] **Step 3: Add `vpScale` to drawMaterial dependency**

`vpScale` is already in the `useCallback` dependency array for `drawMaterial` (line 316: `[walls, grid, selectedWallId, vpScale]`). No change needed.

- [ ] **Step 4: Build check**

```bash
npm run build
```

Expected: exits 0.

- [ ] **Step 5: Visual smoke test**

In the browser, with the editor open:
1. Draw a wood wall segment (HP 40, Resistance 2)
2. Manually set its HP to 20 via browser devtools or by triggering an attack action
3. Verify: damaged wall shows dashed line + opacity 0.8
4. Set `destroyed: true` on a wall
5. Verify: fine dotted line + opacity 0.4 + × marks at both endpoints
6. Verify: intact wall still shows solid line at full opacity
7. Verify: indestructible wall (maxHp=0) shows same as intact

- [ ] **Step 6: Commit**

```bash
git add src/components/organisms/WallsLayer.tsx
git commit -m "feat: add damaged/destroyed visual states to WallsLayer

Co-Authored-By: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Self-Review Checklist

**Spec coverage:**
- [x] `CombatResolver` → `TurnResolver` (Tasks 1–3)
- [x] `TargetReader` interface + `TargetKind` + `WallResult` (Task 3)
- [x] `CategorizeTarget` on MatchSession (Task 2)
- [x] Wall fields + accessors on MatchSession (Task 2)
- [x] `ApplyStructuralDamage` + tests (Task 4)
- [x] `ApplyWallInteract` + tests (Task 4)
- [x] TurnResolver.Resolve routes wall targets (Task 5)
- [x] Rename lobby_ MessageTypes (Task 6)
- [x] `wall_hp_changed` event (Task 6)
- [x] Room migration (lobby_ fields → session) (Task 7)
- [x] OpenNextActionUC + PullActionUC pass session (Task 3 + 8)
- [x] room.go broadcasts WallResults (Task 8)
- [x] documentation-map.yaml updated (Task 9)
- [x] Player-facing wall doc (Task 9)
- [x] useLobbyWs event renames (Task 10)
- [x] useMatchWs wall_hp_changed + sendAction attack type (Task 11)
- [x] GamePage wall_hp_changed + action picker (Task 12)
- [x] WallsLayer visual states (Task 13)

**Intentional deviations from spec:**
- `pieces` (board positions) stay in `room.go` as `r.pieces` — not moved to MatchSession because pieces have no domain logic and moving them would require domain to import delivery types.
- TargetKind constants live in `service/` package (not `matchsession/`) to avoid circular imports.
- TurnResolver.Resolve does NOT call `UpdateWall` directly (pure computation); caller (room.go) applies state changes after receiving `WallResults`. This is cleaner and more testable than the spec's pseudocode implied.
