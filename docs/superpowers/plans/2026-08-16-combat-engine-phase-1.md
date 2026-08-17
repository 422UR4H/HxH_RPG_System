# Combat Engine — Phase 1 (Foundation) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the pure, I/O-free foundation of the combat engine — per-character combat state (`CharacterStatus`, `ResourceBar`, `ModifierLedger`), the match rule value object (`MatchRules`), a real dice engine (`RollCalculator`), and re-key the session by character instead of by player.

**Architecture:** Everything lands in the `match` bounded context. New value objects and entities go in `internal/domain/match/` (package `match`). The dice engine is a stateless domain service in `internal/domain/match/service/`, which receives entities and returns results — it holds no state and does no I/O. `MatchSession` switches its character-indexed maps from `playerUUID` to `sheetUUID`, keeping `participants` on `playerUUID` because authorization stays a per-player concern. Nothing calls `RollCalculator` yet; Phase 2 wires it into `TurnResolver`.

**Tech Stack:** Go 1.23, standard `testing` only (table-driven with `t.Run()`), external test packages (`package foo_test`), `github.com/google/uuid`.

**Spec:** `docs/superpowers/specs/2026-08-16-combat-engine-design.md` (revision `e492e3b`). Game rules: `docs/dev/match/combat-engine.md`, `docs/game/dados.md`.

**Branch:** `feat/combat-engine-phase-1`

## Global Constraints

- **Never delete TODO comments or the owner's design-rationale comment blocks.** They are intentional markers. Task 3 rewrites `character_status.go`; its ~150-line comment block must survive **verbatim**, relocated below the type declarations.
- **No I/O in this phase.** No migrations, no repository changes, no REST, no WS payload changes. `MatchRules` persistence and the `fog_mode` unblock are a separate slice (spec §4.6).
- **Nothing calls `RollCalculator` in this phase.** `TurnResolver`'s `TargetKindCharacter` branch stays empty until Phase 2.
- **`PassiveValue` is never a stored field** — it is always derived from `DiceSet` (spec §4.6). Storing it lets a dice-set change silently decalibrate every passive test.
- **`Modifier.Bias` and `Modifier.Source` are mandatory fields**, not optional (spec §4.3).
- Test packages are external: `package match_test`, `package service_test`, `package matchsession_test`.
- Standard library `testing` only. No testify, no gomock.
- After every task that touches `internal/`: `go vet -tags=integration ./internal/...` must pass before the commit.
- Commit trailers on every commit:
  ```
  Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
  Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>
  ```

---

## File Structure

| File | Status | Responsibility |
|---|---|---|
| `internal/domain/match/match_rules.go` | create | `DiceSet` + `MatchRules` value object with embedded defaults |
| `internal/domain/match/match_rules_test.go` | create | defaults, derived passive value, fog-mode resolution |
| `internal/domain/match/modifier.go` | create | `Modifier`, `Source`, `Scope`, `ModifierLedger` |
| `internal/domain/match/modifier_test.go` | create | scoping, expiry, bias vs amount totals |
| `internal/domain/match/resource_bar.go` | create | `ResourceBar` — shape only; the closing arithmetic is Phase 3 |
| `internal/domain/match/character_status.go` | rewrite | `Stance` + `CharacterStatus` per spec §4.1, keeping the owner's comment block |
| `internal/domain/match/character_status_test.go` | create | constructor wiring, modifier expiry through the status |
| `internal/domain/match/service/roll_calculator.go` | rewrite | `Roll` (once) + `Derive` (many times) → `RollOutcome` |
| `internal/domain/match/service/roll_calculator_test.go` | rewrite | critical, critical failure, passive, advantage, accumulated bias, margin |
| `internal/domain/match/matchsession/match_session.go` | modify | re-key `charSheets`, add `statuses`, fix `CategorizeTarget` |
| `internal/domain/match/matchsession/error.go` | modify | `ErrCharacterStatusNotFound` |
| `internal/domain/match/matchsession/match_session_test.go` | modify | re-keyed lookups, NPC presence |
| `internal/application/match/init_match_session.go` | modify | load NPC sheets, key by `sheetUUID` |
| `internal/application/match/init_match_session_test.go` | modify | assert NPC sheet reaches the session |
| `docs/dev/match/combat-engine.md` | modify | mark the Phase 1 gaps closed |
| `docs/dev/match/flows/05-lacunas.md` | modify | same |
| `docs/documentation-map.yaml` | modify | map the new code paths to their docs |

Tasks 1, 2 and 3 are independent of each other. Task 4 depends on 1 and 2. Task 5 depends on 3. Task 6 depends on everything.

---

## Task 1: `MatchRules` value object

**Files:**
- Create: `internal/domain/match/match_rules.go`
- Test: `internal/domain/match/match_rules_test.go`

**Interfaces:**
- Consumes: `enum.DieSides` (`internal/domain/entity/enum`), `fog.FogMode` (`internal/domain/match/entity/fog`).
- Produces:
  - `match.DiceSet` (string type) with consts `match.DiceSet2D10`, `match.DiceSetD20`
  - `func (DiceSet) Dice() []enum.DieSides`
  - `func (DiceSet) PassiveValue() int`
  - `func (DiceSet) MaxFace() int`
  - `match.MatchRules` struct: `DiceSet DiceSet`, `LadderStep int`, `ReactionTimer *time.Duration`, `DefaultReactions bool`, `FogMode *fog.FogMode`
  - `func NewDefaultMatchRules() MatchRules`
  - `func (MatchRules) PassiveValue() int`
  - `func (MatchRules) ResolveFogMode(mapMode fog.FogMode) fog.FogMode`

**Design note for the implementer:** spec §4.6 lists `PassiveValue` inside the struct but then says it is *derived* from `DiceSet` and never hand-typed. A stored field cannot be both. It is a **method** here, which is the only shape that keeps the guarantee.

- [ ] **Step 1: Write the failing test**

Create `internal/domain/match/match_rules_test.go`:

```go
package match_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

func TestNewDefaultMatchRules(t *testing.T) {
	r := match.NewDefaultMatchRules()

	if r.DiceSet != match.DiceSet2D10 {
		t.Errorf("expected default dice set 2d10, got %q", r.DiceSet)
	}
	if r.LadderStep != 10 {
		t.Errorf("expected ladder step 10, got %d", r.LadderStep)
	}
	if r.ReactionTimer != nil {
		t.Error("expected reaction timer off by default")
	}
	if !r.DefaultReactions {
		t.Error("expected default reactions on")
	}
	if r.FogMode != nil {
		t.Error("expected nil FogMode by default (inherits from the map)")
	}
}

func TestDiceSet(t *testing.T) {
	tests := []struct {
		name    string
		set     match.DiceSet
		dice    []enum.DieSides
		passive int
		maxFace int
	}{
		{"2d10", match.DiceSet2D10, []enum.DieSides{enum.D10, enum.D10}, 11, 10},
		{"d20", match.DiceSetD20, []enum.DieSides{enum.D20}, 10, 20},
		{"unknown falls back to 2d10", match.DiceSet("nonsense"), []enum.DieSides{enum.D10, enum.D10}, 11, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.set.Dice()
			if len(got) != len(tt.dice) {
				t.Fatalf("expected %d dice, got %d", len(tt.dice), len(got))
			}
			for i := range got {
				if got[i] != tt.dice[i] {
					t.Errorf("die %d: expected %v, got %v", i, tt.dice[i], got[i])
				}
			}
			if tt.set.PassiveValue() != tt.passive {
				t.Errorf("expected passive %d, got %d", tt.passive, tt.set.PassiveValue())
			}
			if tt.set.MaxFace() != tt.maxFace {
				t.Errorf("expected max face %d, got %d", tt.maxFace, tt.set.MaxFace())
			}
		})
	}
}

func TestMatchRules_PassiveValueFollowsDiceSet(t *testing.T) {
	r := match.NewDefaultMatchRules()
	if r.PassiveValue() != 11 {
		t.Errorf("expected 11 for 2d10, got %d", r.PassiveValue())
	}

	// Swapping the dice set must move the passive value with it — that is the whole
	// reason PassiveValue is derived instead of stored.
	r.DiceSet = match.DiceSetD20
	if r.PassiveValue() != 10 {
		t.Errorf("expected 10 for d20, got %d", r.PassiveValue())
	}
}

func TestMatchRules_ResolveFogMode(t *testing.T) {
	live := fog.FogModeLive

	t.Run("match rule wins when set", func(t *testing.T) {
		r := match.NewDefaultMatchRules()
		r.FogMode = &live
		if got := r.ResolveFogMode(fog.FogModeExplored); got != fog.FogModeLive {
			t.Errorf("expected live, got %q", got)
		}
	})

	t.Run("inherits the map when unset", func(t *testing.T) {
		r := match.NewDefaultMatchRules()
		if got := r.ResolveFogMode(fog.FogModeLive); got != fog.FogModeLive {
			t.Errorf("expected live from the map, got %q", got)
		}
	})

	t.Run("falls back to explored when neither is valid", func(t *testing.T) {
		r := match.NewDefaultMatchRules()
		if got := r.ResolveFogMode(fog.FogMode("")); got != fog.FogModeExplored {
			t.Errorf("expected explored, got %q", got)
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/match/ -run 'MatchRules|DiceSet' -v`
Expected: FAIL — `undefined: match.NewDefaultMatchRules`, `undefined: match.DiceSet2D10`.

- [ ] **Step 3: Write the implementation**

Create `internal/domain/match/match_rules.go`:

```go
package match

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// DiceSet names the roll shape a match uses for every test: skill, hit, actionSpeed.
// 2D10 is the system default; D20 is an alternative match rule that flattens the
// distribution and makes luck weigh more than skill.
type DiceSet string

const (
	DiceSet2D10 DiceSet = "2d10"
	DiceSetD20  DiceSet = "d20"
)

// Dice returns the dice this set rolls, in order.
// Unknown values fall back to the system default rather than rolling nothing.
func (d DiceSet) Dice() []enum.DieSides {
	if d == DiceSetD20 {
		return []enum.DieSides{enum.D20}
	}
	return []enum.DieSides{enum.D10, enum.D10}
}

// PassiveValue is the average of the set. A passive test takes it instead of rolling,
// so rolling has exactly zero expected gain — the player only gambles when they need
// luck above the average. 11 for 2D10, 10 for D20.
func (d DiceSet) PassiveValue() int {
	if d == DiceSetD20 {
		return 10
	}
	return 11
}

// MaxFace is the top face of a single die in the set. Rolling it on every die of the
// set is a critical; rolling 1 on every die is a critical failure. The reading is on
// the individual dice, never on the sum.
func (d DiceSet) MaxFace() int {
	if d == DiceSetD20 {
		return 20
	}
	return 10
}

// MatchRules is the per-match rule configuration: the numbers a table is allowed to
// change. The shape of the result ladder — how many steps and what each one does —
// stays in code, because changing it changes the game.
//
// Phase 1 keeps this a pure value object with embedded defaults, passed by parameter
// to whoever needs it. It is not global, not read from anywhere, and not persisted.
// Persistence, the REST surface for the master to choose, and the fog_mode unblock in
// room.go are a separate slice — see the design spec §4.6.
type MatchRules struct {
	DiceSet          DiceSet
	LadderStep       int
	ReactionTimer    *time.Duration // nil = off
	DefaultReactions bool           // apply the default reaction when the target sends nothing
	FogMode          *fog.FogMode   // nil = inherit from the map
}

// NewDefaultMatchRules returns the MVP defaults.
func NewDefaultMatchRules() MatchRules {
	return MatchRules{
		DiceSet:          DiceSet2D10,
		LadderStep:       10,
		ReactionTimer:    nil,
		DefaultReactions: true,
		FogMode:          nil,
	}
}

// PassiveValue is derived from DiceSet and never stored. A stored copy would let a
// dice-set change decalibrate every passive test silently.
func (r MatchRules) PassiveValue() int { return r.DiceSet.PassiveValue() }

// ResolveFogMode implements: match rule ?? map ?? explored.
//
// A map is reusable across matches, so the fog style belongs to how *this table* wants
// to play — but the values already stored in maps.fog_mode stay meaningful as the
// default instead of being orphaned.
func (r MatchRules) ResolveFogMode(mapMode fog.FogMode) fog.FogMode {
	if r.FogMode != nil && r.FogMode.IsValid() {
		return *r.FogMode
	}
	if mapMode.IsValid() {
		return mapMode
	}
	return fog.FogModeExplored
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/domain/match/ -run 'MatchRules|DiceSet' -v`
Expected: PASS — all sub-tests green.

- [ ] **Step 5: Vet and commit**

```bash
go vet -tags=integration ./internal/...
git add internal/domain/match/match_rules.go internal/domain/match/match_rules_test.go
git commit -m "feat(match): add MatchRules value object with derived passive value

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 2: `Modifier` and `ModifierLedger`

**Files:**
- Create: `internal/domain/match/modifier.go`
- Test: `internal/domain/match/modifier_test.go`

**Interfaces:**
- Consumes: `github.com/google/uuid`.
- Produces:
  - `match.Source` with consts `match.SourceSystem`, `match.SourceMaster`
  - `match.Scope` with consts `match.ScopeEndOfTurn`, `match.ScopeEndOfRound`
  - `match.Modifier` struct: `Amount int`, `Bias int`, `Source Source`, `AgainstID *uuid.UUID`, `ExpiresAt Scope`, `Reason string`
  - `func (Modifier) AppliesTo(targetID *uuid.UUID) bool`
  - `match.ModifierLedger` struct (unexported slice inside)
  - `func NewModifierLedger() ModifierLedger`
  - `func (*ModifierLedger) Add(Modifier)`
  - `func (*ModifierLedger) All() []Modifier`
  - `func (*ModifierLedger) TotalAmount(targetID *uuid.UUID) int`
  - `func (*ModifierLedger) TotalBias(targetID *uuid.UUID) int`
  - `func (*ModifierLedger) Expire(scope Scope)`

**Why both `Amount` and `Bias`:** the accumulated difference from the repel ladder is a flat number, but the disadvantage the system generates (swapping a declared action, converting an action into a reaction) is a change in *how you roll*, not a number you can add. A ledger with only `Amount` cannot carry it. `Source` exists so the audit trail (`SystemData`, Phase 5) can tell what the system put there from what the master put there — that is the entire point of the table.

**Why bonuses are targeted and penalties are general:** you read *that* opponent, so the bonus is against them specifically (`AgainstID` set); you got off balance, so the penalty applies to everyone (`AgainstID` nil). The ledger enforces nothing about which is which — it just honors the scope it is given.

- [ ] **Step 1: Write the failing test**

Create `internal/domain/match/modifier_test.go`:

```go
package match_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
)

func TestModifier_AppliesTo(t *testing.T) {
	enemy := uuid.New()
	other := uuid.New()

	t.Run("general modifier applies to anyone", func(t *testing.T) {
		m := match.Modifier{Amount: -3, AgainstID: nil}
		if !m.AppliesTo(&enemy) {
			t.Error("expected a general modifier to apply against a named target")
		}
		if !m.AppliesTo(nil) {
			t.Error("expected a general modifier to apply with no target")
		}
	})

	t.Run("targeted modifier applies only to its target", func(t *testing.T) {
		m := match.Modifier{Amount: 4, AgainstID: &enemy}
		if !m.AppliesTo(&enemy) {
			t.Error("expected the modifier to apply against its own target")
		}
		if m.AppliesTo(&other) {
			t.Error("expected the modifier not to apply against a different target")
		}
		if m.AppliesTo(nil) {
			t.Error("expected a targeted modifier not to apply with no target")
		}
	})
}

func TestModifierLedger_Totals(t *testing.T) {
	enemy := uuid.New()
	other := uuid.New()

	l := match.NewModifierLedger()
	l.Add(match.Modifier{
		Amount: -2, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfRound,
		Reason: "off balance after a parry",
	})
	l.Add(match.Modifier{
		Amount: 5, AgainstID: &enemy, Source: match.SourceSystem,
		ExpiresAt: match.ScopeEndOfTurn, Reason: "read the opponent",
	})
	l.Add(match.Modifier{
		Bias: -1, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfTurn,
		Reason: "swapped a declared action",
	})
	l.Add(match.Modifier{
		Bias: -1, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfTurn,
		Reason: "converted the action into a reaction",
	})

	t.Run("amounts against the named target include the targeted bonus", func(t *testing.T) {
		if got := l.TotalAmount(&enemy); got != 3 { // -2 general + 5 targeted
			t.Errorf("expected 3, got %d", got)
		}
	})

	t.Run("amounts against another target exclude it", func(t *testing.T) {
		if got := l.TotalAmount(&other); got != -2 {
			t.Errorf("expected -2, got %d", got)
		}
	})

	t.Run("bias accumulates independently of amount", func(t *testing.T) {
		if got := l.TotalBias(&enemy); got != -2 {
			t.Errorf("expected -2, got %d", got)
		}
	})
}

func TestModifierLedger_Expire(t *testing.T) {
	l := match.NewModifierLedger()
	l.Add(match.Modifier{Amount: 1, ExpiresAt: match.ScopeEndOfTurn, Reason: "turn scoped"})
	l.Add(match.Modifier{Amount: 10, ExpiresAt: match.ScopeEndOfRound, Reason: "round scoped"})

	l.Expire(match.ScopeEndOfTurn)

	if got := len(l.All()); got != 1 {
		t.Fatalf("expected 1 modifier left, got %d", got)
	}
	if got := l.TotalAmount(nil); got != 10 {
		t.Errorf("expected only the round-scoped modifier to survive, got total %d", got)
	}
}

func TestModifierLedger_AllIsACopy(t *testing.T) {
	l := match.NewModifierLedger()
	l.Add(match.Modifier{Amount: 1, ExpiresAt: match.ScopeEndOfTurn})

	got := l.All()
	got[0].Amount = 999

	if l.TotalAmount(nil) != 1 {
		t.Error("expected All() to hand back a copy, not the ledger's own slice")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/match/ -run Modifier -v`
Expected: FAIL — `undefined: match.Modifier`, `undefined: match.NewModifierLedger`.

- [ ] **Step 3: Write the implementation**

Create `internal/domain/match/modifier.go`:

```go
package match

import "github.com/google/uuid"

// Source records who created a Modifier. The system generates disadvantage on its own
// (swapping a declared action, converting an action into a reaction); the master grants
// or cancels advantage by hand. Keeping them apart is what lets the master cancel the
// system's disadvantage without either one overwriting the other — and it is what the
// audit trail reads to tell the two apart.
type Source string

const (
	SourceSystem Source = "system"
	SourceMaster Source = "master"
)

// Scope is when a Modifier stops applying.
type Scope string

const (
	ScopeEndOfTurn  Scope = "end_of_turn"
	ScopeEndOfRound Scope = "end_of_round"
)

// Modifier is one accumulated bonus or penalty carried by a character.
//
// Amount and Bias are different currencies and never substitute for each other:
// Amount is a flat adjustment to the total; Bias is advantage/disadvantage on the dice
// (−1/0/+1, accumulating), which is a change in how the roll is made, not a number that
// can be added to it.
type Modifier struct {
	Amount    int
	Bias      int
	Source    Source
	AgainstID *uuid.UUID // nil = applies against anyone
	ExpiresAt Scope
	Reason    string // surfaced in the Action History and in the audit trail
}

// AppliesTo reports whether m counts for a roll made against targetID.
// A targeted modifier never applies to a roll with no named target.
func (m Modifier) AppliesTo(targetID *uuid.UUID) bool {
	if m.AgainstID == nil {
		return true
	}
	if targetID == nil {
		return false
	}
	return *m.AgainstID == *targetID
}

// ModifierLedger is the accumulated difference a character carries: the bonuses and
// penalties that the repel ladder, the system, and the master have piled on.
//
// This is deliberately NOT RollCondition. RollCondition is the master's struct for one
// roll — their dice bias and their manual adjustment. The ledger is the character's
// standing state across turns and rounds.
type ModifierLedger struct {
	modifiers []Modifier
}

func NewModifierLedger() ModifierLedger { return ModifierLedger{} }

func (l *ModifierLedger) Add(m Modifier) { l.modifiers = append(l.modifiers, m) }

// All returns a copy, so a caller iterating the ledger cannot mutate it.
func (l *ModifierLedger) All() []Modifier {
	out := make([]Modifier, len(l.modifiers))
	copy(out, l.modifiers)
	return out
}

// TotalAmount sums the flat adjustments that apply against targetID.
func (l *ModifierLedger) TotalAmount(targetID *uuid.UUID) int {
	total := 0
	for _, m := range l.modifiers {
		if m.AppliesTo(targetID) {
			total += m.Amount
		}
	}
	return total
}

// TotalBias sums the dice biases that apply against targetID. Advantage and
// disadvantage accumulate and can cancel each other out.
func (l *ModifierLedger) TotalBias(targetID *uuid.UUID) int {
	total := 0
	for _, m := range l.modifiers {
		if m.AppliesTo(targetID) {
			total += m.Bias
		}
	}
	return total
}

// Expire drops every modifier whose validity ended at scope.
func (l *ModifierLedger) Expire(scope Scope) {
	kept := l.modifiers[:0]
	for _, m := range l.modifiers {
		if m.ExpiresAt != scope {
			kept = append(kept, m)
		}
	}
	l.modifiers = kept
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/domain/match/ -run Modifier -v`
Expected: PASS — all sub-tests green.

- [ ] **Step 5: Vet and commit**

```bash
go vet -tags=integration ./internal/...
git add internal/domain/match/modifier.go internal/domain/match/modifier_test.go
git commit -m "feat(match): add ModifierLedger with amount, bias and source

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 3: `ResourceBar`, `Stance` and the `CharacterStatus` rewrite

**Files:**
- Create: `internal/domain/match/resource_bar.go`
- Rewrite: `internal/domain/match/character_status.go`
- Test: `internal/domain/match/character_status_test.go`

**Interfaces:**
- Consumes: `match.ModifierLedger`, `match.Scope` (Task 2); `action.Velocity` (`internal/domain/match/entity/action`).
- Produces:
  - `match.ResourceBar` struct: `Balance int`, `Speeds []int`
  - `func (*ResourceBar) RecordSpeed(int)`
  - `func (*ResourceBar) ResetRound()`
  - `match.Stance` (string type) with const `match.StanceNone`
  - `match.CharacterStatus` struct: `ActionBar ResourceBar`, `MoveBar ResourceBar`, `Ledger ModifierLedger`, `Stance Stance`, `Velocity action.Velocity`
  - `func NewCharacterStatus() *CharacterStatus`
  - `func (*CharacterStatus) ExpireModifiers(Scope)`

### ⚠️ Read before touching `character_status.go`

The current file is ~150 lines of the owner's design rationale (movement, the two bars, clash, footwork, charge, the quickness↔acceleration approximation) below a small struct. `05-lacunas.md` calls it *"the richest source of product intent in the repository"*, and `AGENTS.md` forbids removing the owner's comment markers.

**The comment block moves; it does not shrink.** Copy lines 5–163 of the current file verbatim to the bottom of the new file, under a header that says what it is. Only the `struct` declaration on lines 15–23 is replaced.

The three fields being dropped from the struct:
- `Position [3]int` — deliberate. Positions live in `Room` and the session already reaches them through `matchsession.PiecePositionSource`. Duplicating them would create a second source of truth while the map keeps drawing the `Room`'s copy.
- `MoveBar int` / `ActionBar int` — widened into `ResourceBar`, which also has to carry the round's rolled speeds for the Phase 3 average.

Nothing in the repository instantiates `CharacterStatus` today, so this rewrite breaks no caller.

**Also deliberately deferred:** spec §4.1 sketches a `CharacterPosition(charID)` method on `PiecePositionSource`. Nothing in Phase 1 needs a position, and adding it now forces `room.go` plus every test fake to implement a method with no caller. It lands in Phase 3, with lunges and reach.

- [ ] **Step 1: Write the failing test**

Create `internal/domain/match/character_status_test.go`:

```go
package match_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
)

func TestNewCharacterStatus(t *testing.T) {
	s := match.NewCharacterStatus()

	if s == nil {
		t.Fatal("expected non-nil CharacterStatus")
	}
	if s.ActionBar.Balance != 0 || s.MoveBar.Balance != 0 {
		t.Error("expected both bars to start at zero balance")
	}
	if len(s.ActionBar.Speeds) != 0 || len(s.MoveBar.Speeds) != 0 {
		t.Error("expected both bars to start with no recorded speeds")
	}
	if s.Stance != match.StanceNone {
		t.Errorf("expected StanceNone while posture rules do not exist, got %q", s.Stance)
	}
	if len(s.Ledger.All()) != 0 {
		t.Error("expected an empty ledger")
	}
	if s.Velocity.Speed != 0 {
		t.Errorf("expected zero velocity, got %v", s.Velocity.Speed)
	}
}

func TestResourceBar_RecordAndReset(t *testing.T) {
	s := match.NewCharacterStatus()

	s.ActionBar.RecordSpeed(20)
	s.ActionBar.RecordSpeed(14)
	s.ActionBar.Balance = 9

	if got := len(s.ActionBar.Speeds); got != 2 {
		t.Fatalf("expected 2 recorded speeds, got %d", got)
	}

	s.ActionBar.ResetRound()

	if got := len(s.ActionBar.Speeds); got != 0 {
		t.Errorf("expected the speed history cleared, got %d entries", got)
	}
	if s.ActionBar.Balance != 9 {
		t.Errorf("expected the carry-over balance to survive the round reset, got %d", s.ActionBar.Balance)
	}
}

func TestResourceBar_BarsAreIndependent(t *testing.T) {
	s := match.NewCharacterStatus()

	s.ActionBar.RecordSpeed(20)

	if len(s.MoveBar.Speeds) != 0 {
		t.Error("expected the move bar to be untouched by an action-bar record")
	}
}

func TestCharacterStatus_ExpireModifiers(t *testing.T) {
	s := match.NewCharacterStatus()
	s.Ledger.Add(match.Modifier{Amount: 3, ExpiresAt: match.ScopeEndOfTurn, Source: match.SourceSystem})
	s.Ledger.Add(match.Modifier{Amount: 7, ExpiresAt: match.ScopeEndOfRound, Source: match.SourceMaster})

	s.ExpireModifiers(match.ScopeEndOfTurn)

	if got := s.Ledger.TotalAmount(nil); got != 7 {
		t.Errorf("expected only the round-scoped modifier to survive, got %d", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/match/ -run 'CharacterStatus|ResourceBar' -v`
Expected: FAIL — `undefined: match.NewCharacterStatus`, `undefined: match.StanceNone`, and `s.ActionBar.RecordSpeed undefined (type int has no field or method RecordSpeed)`.

- [ ] **Step 3: Create `ResourceBar`**

Create `internal/domain/match/resource_bar.go`:

```go
package match

// ResourceBar is one of the two clocks a character runs on: actionSpeed (attack, item,
// ability) and moveSpeed (shift, dash, leap, roll). They have independent prices but
// share a single clock, and both live on the same scale — skill + the dice set — so the
// engine compares value against value with no conversion.
//
// Balance is the standing credit or debt, which carries across rounds. Speeds is the
// history of the speeds rolled on this bar during the current round; the round-closing
// formula averages it.
//
// Phase 1 defines only the shape. The arithmetic — average, round price, ceiling,
// carry-over — is Phase 3.
type ResourceBar struct {
	Balance int
	Speeds  []int
}

// RecordSpeed appends a speed rolled on this bar during the current round.
func (b *ResourceBar) RecordSpeed(speed int) {
	b.Speeds = append(b.Speeds, speed)
}

// ResetRound clears the round's speed history. The balance is deliberately untouched:
// it is the carry-over into the next round, as credit or as debt.
func (b *ResourceBar) ResetRound() {
	b.Speeds = nil
}
```

- [ ] **Step 4: Rewrite `character_status.go`**

Replace `internal/domain/match/character_status.go` with the content below. **The block under "Design rationale" is the current file's lines 5–163, verbatim** — copy it from git (`git show HEAD:internal/domain/match/character_status.go`), do not retype or summarize it. Only the header and the type declarations are new.

```go
package match

import "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"

// Stance is the character's combat posture: on guard, offensive, defensive, evasive.
//
// Reserved. The posture rules do not exist yet, so every character is StanceNone. It is
// declared now because CharacterStatus touches the bars, the ledger, the posture and the
// velocity all at once — getting its shape wrong cascades, so the reserved fields are
// planned up front even without a use. When postures arrive, the closed-escape discount
// (spending only the move bar) starts requiring the evasive stance; until then it applies
// unconditionally.
type Stance string

const StanceNone Stance = ""

// CharacterStatus is a character's live combat state inside a match.
//
// Position is deliberately absent: positions live in the Room (they arrive in the WS
// payloads) and the session already reaches them through matchsession.PiecePositionSource.
// Duplicating them here would create a second source of truth while the map kept drawing
// the Room's copy.
type CharacterStatus struct {
	ActionBar ResourceBar    // balance + this round's rolled speeds
	MoveBar   ResourceBar    // same, for movement
	Ledger    ModifierLedger // accumulated bonuses and penalties
	Stance    Stance         // reserved; the posture rules do not exist yet
	Velocity  action.Velocity
}

func NewCharacterStatus() *CharacterStatus {
	return &CharacterStatus{
		ActionBar: ResourceBar{},
		MoveBar:   ResourceBar{},
		Ledger:    NewModifierLedger(),
		Stance:    StanceNone,
	}
}

// ExpireModifiers drops every ledger entry whose validity ended at scope. Called at the
// end of a turn and at the end of a round.
func (s *CharacterStatus) ExpireModifiers(scope Scope) {
	s.Ledger.Expire(scope)
}

// ─────────────────────────────────────────────────────────────────────────────
// Design rationale — the product owner's notes. Kept verbatim: this is the richest
// record of intent for movement, the two bars, clash, footwork and charge that exists
// in the repository. Do not trim it.
// ─────────────────────────────────────────────────────────────────────────────

// <<< PASTE lines 5–163 of the previous character_status.go here, unchanged >>>
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./internal/domain/match/ -v`
Expected: PASS — Tasks 1, 2 and 3 tests all green.

- [ ] **Step 6: Verify the comment block survived**

Run: `git show HEAD:internal/domain/match/character_status.go | wc -l && wc -l internal/domain/match/character_status.go`
Expected: the new file is **longer** than the old one (163 lines). If it is shorter, the rationale block was trimmed — restore it before committing.

- [ ] **Step 7: Vet and commit**

```bash
go vet -tags=integration ./internal/...
git add internal/domain/match/resource_bar.go internal/domain/match/character_status.go internal/domain/match/character_status_test.go
git commit -m "feat(match): turn CharacterStatus into code with ResourceBar and Stance

Keeps the owner's design-rationale comment block verbatim.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 4: `RollCalculator` — roll once, derive many times

**Files:**
- Rewrite: `internal/domain/match/service/roll_calculator.go`
- Rewrite: `internal/domain/match/service/roll_calculator_test.go`

**Interfaces:**
- Consumes: `match.MatchRules`, `match.DiceSet`, `match.ModifierLedger` (Tasks 1–2); `action.RollCondition` (`internal/domain/match/entity/action`); `die.NewDie` (`internal/domain/entity/die`).
- Produces:
  - `service.RollAttempts` struct: `Primary []int`, `Secondary []int`
  - `service.RollInput` struct: `SkillName string`, `SkillValue int`, `Passive bool`, `Condition *action.RollCondition`, `Ledger *match.ModifierLedger`, `AgainstID *uuid.UUID`
  - `service.RollOutcome` struct: `SkillName string`, `SkillValue int`, `Dice []int`, `DiceTotal int`, `Bias int`, `Modifier int`, `Passive bool`, `Total int`, `IsCritical bool`, `IsCriticalFailure bool`
  - `func (RollOutcome) Margin(cd int) int`
  - `func (RollCalculator) Roll(rules match.MatchRules) RollAttempts`
  - `func (RollCalculator) Derive(rules match.MatchRules, attempts RollAttempts, in RollInput) RollOutcome`

### The two-set decision

Principle 3 of the spec says the dice fall **once** and the result is re-derived as many times as needed; principle 6 says every master edit forces a recalculation **with no new roll**. But advantage means "roll the set twice, take the better one" — and the master can add advantage *after* the dice have already fallen.

`Roll` therefore rolls **both sets up front, always**, and `Derive` picks between them by the accumulated bias. That is the only shape where a master edit can turn a neutral roll into an advantaged one without ever re-rolling a player's die. When the bias is 0 the secondary set is simply never read.

Ties fall back to `Primary`. With 2D10 a tie can never involve a critical: 20 only comes from two tens and 2 only from two ones, so there is no second combination to tie with.

**Deliberately not done here:** `RollOutcome` is not written back into `action.RollCheck.Result` or `RollContext`, and no character sheet is read. Phase 2 owns that wiring, along with two frictions `05-lacunas.md` already records — `RollCheck.SkillName` is a `string` while the sheet indexes by `enum.SkillName`, and `RollContext.GetDiceResult` takes a `die.Die` parameter it ignores.

**The margin is derived, not stored.** `Margin(cd)` is a method because the CD comes from the opposed roll, which does not exist until Phase 2.

- [ ] **Step 1: Write the failing test**

Replace `internal/domain/match/service/roll_calculator_test.go` entirely:

```go
package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// attempts builds a RollAttempts with both sets, so tests are deterministic:
// Roll() uses crypto/rand and has no seam, but Derive() is pure data in, data out.
func attempts(primary, secondary []int) service.RollAttempts {
	return service.RollAttempts{Primary: primary, Secondary: secondary}
}

func TestRollCalculator_Roll(t *testing.T) {
	calc := service.RollCalculator{}

	t.Run("2d10 rolls two dice in both sets", func(t *testing.T) {
		got := calc.Roll(match.NewDefaultMatchRules())

		for name, set := range map[string][]int{"primary": got.Primary, "secondary": got.Secondary} {
			if len(set) != 2 {
				t.Fatalf("%s: expected 2 dice, got %d", name, len(set))
			}
			for i, d := range set {
				if d < 1 || d > 10 {
					t.Errorf("%s die %d out of range: %d", name, i, d)
				}
			}
		}
	})

	t.Run("d20 rolls one die in both sets", func(t *testing.T) {
		rules := match.NewDefaultMatchRules()
		rules.DiceSet = match.DiceSetD20

		got := calc.Roll(rules)

		if len(got.Primary) != 1 || len(got.Secondary) != 1 {
			t.Fatalf("expected 1 die per set, got %d and %d", len(got.Primary), len(got.Secondary))
		}
		if got.Primary[0] < 1 || got.Primary[0] > 20 {
			t.Errorf("die out of range: %d", got.Primary[0])
		}
	})
}

func TestRollCalculator_Derive_Criticals(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()

	tests := []struct {
		name        string
		dice        []int
		wantTotal   int
		wantCrit    bool
		wantCritErr bool
	}{
		{"double ten is a critical", []int{10, 10}, 25, true, false},
		{"double one is a critical failure", []int{1, 1}, 7, false, true},
		{"nine and ten is neither", []int{9, 10}, 24, false, false},
		{"one and ten is neither", []int{1, 10}, 16, false, false},
		{"middling roll is neither", []int{4, 6}, 15, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := calc.Derive(rules, attempts(tt.dice, nil), service.RollInput{
				SkillName:  "Combate Desarmado",
				SkillValue: 5,
			})

			if out.Total != tt.wantTotal {
				t.Errorf("expected total %d, got %d", tt.wantTotal, out.Total)
			}
			if out.IsCritical != tt.wantCrit {
				t.Errorf("expected IsCritical %v, got %v", tt.wantCrit, out.IsCritical)
			}
			if out.IsCriticalFailure != tt.wantCritErr {
				t.Errorf("expected IsCriticalFailure %v, got %v", tt.wantCritErr, out.IsCriticalFailure)
			}
			if len(out.Dice) != 2 {
				t.Errorf("expected the individual dice to survive, got %v", out.Dice)
			}
		})
	}
}

func TestRollCalculator_Derive_D20Criticals(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()
	rules.DiceSet = match.DiceSetD20

	t.Run("natural 20 is a critical", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{20}, nil), service.RollInput{SkillValue: 3})
		if !out.IsCritical || out.Total != 23 {
			t.Errorf("expected critical with total 23, got %v / %d", out.IsCritical, out.Total)
		}
	})

	t.Run("natural 1 is a critical failure", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{1}, nil), service.RollInput{SkillValue: 3})
		if !out.IsCriticalFailure {
			t.Error("expected a critical failure on a natural 1")
		}
	})
}

func TestRollCalculator_Derive_Passive(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()

	// A passive test ignores the dice entirely and takes the average of the set.
	out := calc.Derive(rules, attempts([]int{10, 10}, []int{1, 1}), service.RollInput{
		SkillName:  "Reflexo",
		SkillValue: 6,
		Passive:    true,
	})

	if out.Total != 17 { // 11 (average of 2d10) + 6
		t.Errorf("expected 17, got %d", out.Total)
	}
	if out.DiceTotal != 11 {
		t.Errorf("expected the derived average 11, got %d", out.DiceTotal)
	}
	if len(out.Dice) != 0 {
		t.Errorf("expected no dice on a passive test, got %v", out.Dice)
	}
	if out.IsCritical || out.IsCriticalFailure {
		t.Error("expected a passive test never to crit")
	}
	if !out.Passive {
		t.Error("expected the outcome to be flagged passive")
	}
}

func TestRollCalculator_Derive_Bias(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()
	both := attempts([]int{3, 4}, []int{8, 9}) // 7 vs 17

	t.Run("advantage takes the better set", func(t *testing.T) {
		out := calc.Derive(rules, both, service.RollInput{
			SkillValue: 0,
			Condition:  &action.RollCondition{Bias: 1},
		})
		if out.DiceTotal != 17 {
			t.Errorf("expected 17, got %d", out.DiceTotal)
		}
	})

	t.Run("disadvantage takes the worse set", func(t *testing.T) {
		out := calc.Derive(rules, both, service.RollInput{
			SkillValue: 0,
			Condition:  &action.RollCondition{Bias: -1},
		})
		if out.DiceTotal != 7 {
			t.Errorf("expected 7, got %d", out.DiceTotal)
		}
	})

	t.Run("neutral takes the primary set", func(t *testing.T) {
		out := calc.Derive(rules, both, service.RollInput{SkillValue: 0})
		if out.DiceTotal != 7 {
			t.Errorf("expected the primary set, got %d", out.DiceTotal)
		}
	})

	t.Run("system disadvantage accumulates and the master can cancel it", func(t *testing.T) {
		ledger := match.NewModifierLedger()
		ledger.Add(match.Modifier{Bias: -1, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfTurn})

		// Master grants advantage: +1 from the master, −1 from the system → neutral.
		out := calc.Derive(rules, both, service.RollInput{
			Condition: &action.RollCondition{Bias: 1},
			Ledger:    &ledger,
		})
		if out.Bias != 0 {
			t.Errorf("expected the biases to cancel out, got %d", out.Bias)
		}
		if out.DiceTotal != 7 {
			t.Errorf("expected the primary set on a neutral bias, got %d", out.DiceTotal)
		}
	})

	t.Run("two system disadvantages outweigh one master advantage", func(t *testing.T) {
		ledger := match.NewModifierLedger()
		ledger.Add(match.Modifier{Bias: -1, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfTurn})
		ledger.Add(match.Modifier{Bias: -1, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfTurn})

		out := calc.Derive(rules, both, service.RollInput{
			Condition: &action.RollCondition{Bias: 1},
			Ledger:    &ledger,
		})
		if out.Bias != -1 {
			t.Errorf("expected net bias -1, got %d", out.Bias)
		}
		if out.DiceTotal != 7 {
			t.Errorf("expected the worse set, got %d", out.DiceTotal)
		}
	})

	t.Run("falls back to the primary set when no secondary was rolled", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{5, 5}, nil), service.RollInput{
			Condition: &action.RollCondition{Bias: 1},
		})
		if out.DiceTotal != 10 {
			t.Errorf("expected 10, got %d", out.DiceTotal)
		}
	})
}

func TestRollCalculator_Derive_Modifiers(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()
	enemy := uuid.New()
	other := uuid.New()

	ledger := match.NewModifierLedger()
	ledger.Add(match.Modifier{
		Amount: 5, AgainstID: &enemy, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfTurn,
	})
	ledger.Add(match.Modifier{
		Amount: -2, Source: match.SourceSystem, ExpiresAt: match.ScopeEndOfRound,
	})

	t.Run("master modifier and ledger stack against the read opponent", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{4, 6}, nil), service.RollInput{
			SkillValue: 10,
			Condition:  &action.RollCondition{Modifier: 3, Description: "creative move"},
			Ledger:     &ledger,
			AgainstID:  &enemy,
		})
		if out.Modifier != 6 { // 3 master + 5 targeted − 2 general
			t.Errorf("expected modifier 6, got %d", out.Modifier)
		}
		if out.Total != 26 { // 10 dice + 10 skill + 6
			t.Errorf("expected total 26, got %d", out.Total)
		}
	})

	t.Run("the targeted bonus does not follow to another opponent", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{4, 6}, nil), service.RollInput{
			SkillValue: 10,
			Ledger:     &ledger,
			AgainstID:  &other,
		})
		if out.Modifier != -2 {
			t.Errorf("expected only the general penalty, got %d", out.Modifier)
		}
	})

	t.Run("nil condition and nil ledger are neutral", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{4, 6}, nil), service.RollInput{SkillValue: 10})
		if out.Modifier != 0 || out.Bias != 0 || out.Total != 20 {
			t.Errorf("expected a neutral derivation, got %+v", out)
		}
	})
}

func TestRollOutcome_Margin(t *testing.T) {
	calc := service.RollCalculator{}
	out := calc.Derive(match.NewDefaultMatchRules(), attempts([]int{7, 5}, nil), service.RollInput{
		SkillValue: 5,
	}) // total 17

	tests := []struct {
		name string
		cd   int
		want int
	}{
		{"beats the CD", 15, 2},
		{"exactly meets the CD", 17, 0},
		{"misses the CD", 20, -3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := out.Margin(tt.cd); got != tt.want {
				t.Errorf("expected margin %d, got %d", tt.want, got)
			}
		})
	}
}

func TestRollCalculator_Derive_IsPure(t *testing.T) {
	// Principle 3: the dice fall once, the result is derived as many times as needed.
	// Deriving twice from the same attempts must give the same numbers.
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()
	a := attempts([]int{8, 2}, []int{9, 9})
	in := service.RollInput{SkillValue: 4, Condition: &action.RollCondition{Bias: 1}}

	first := calc.Derive(rules, a, in)
	second := calc.Derive(rules, a, in)

	if first.Total != second.Total || first.DiceTotal != second.DiceTotal {
		t.Errorf("expected a stable derivation, got %d then %d", first.Total, second.Total)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run Roll -v`
Expected: FAIL — `undefined: service.RollAttempts`, `undefined: service.RollInput`, `calc.Roll undefined`.

- [ ] **Step 3: Write the implementation**

Replace `internal/domain/match/service/roll_calculator.go` entirely:

```go
package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/die"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// RollAttempts holds BOTH dice sets, rolled together the moment the action or reaction
// arrives.
//
// Advantage means rolling the set twice and keeping the better one — but the master can
// grant advantage *after* the dice have already fallen, and the master never re-rolls a
// player's die. Rolling both sets up front is the only shape that satisfies both: a later
// edit changes which set is read, never what was rolled. On a neutral bias, Secondary is
// simply never read.
type RollAttempts struct {
	Primary   []int
	Secondary []int
}

// RollInput is everything Derive needs besides the dice themselves.
//
// Condition is the master's struct for this one roll: their dice bias and their manual
// adjustment. Ledger is the character's standing accumulated difference. They are summed
// here, never merged upstream, so the master can cancel the system's disadvantage without
// either one overwriting the other.
type RollInput struct {
	SkillName  string
	SkillValue int                   // already resolved from the sheet by the caller
	Passive    bool                  // take the set's average instead of rolling
	Condition  *action.RollCondition // master-owned; nil = neutral
	Ledger     *match.ModifierLedger // character-owned; nil = empty
	AgainstID  *uuid.UUID            // whom the roll is against; nil = nobody in particular
}

// RollOutcome is the derived result of one test. The individual dice survive because a
// critical is the combination, not the sum.
type RollOutcome struct {
	SkillName         string
	SkillValue        int
	Dice              []int // empty on a passive test
	DiceTotal         int
	Bias              int // net accumulated bias, master + system
	Modifier          int // net accumulated flat adjustment, master + ledger
	Passive           bool
	Total             int
	IsCritical        bool
	IsCriticalFailure bool
}

// Margin is the reading of this outcome against a difficulty class. The result of a test
// is the margin, not a boolean — success and failure are readings of the margin against
// thresholds.
//
// It is a method rather than a field because the CD comes from the opposed roll, which
// does not exist until the collision is implemented.
func (o RollOutcome) Margin(cd int) int { return o.Total - cd }

// RollCalculator is a stateless domain service that turns dice plus a character's numbers
// into a result. It rolls once (Roll) and derives as many times as the master edits
// (Derive).
type RollCalculator struct{}

// Roll rolls both sets for the given rules. Called once, when the action or reaction
// arrives.
func (rc RollCalculator) Roll(rules match.MatchRules) RollAttempts {
	return RollAttempts{
		Primary:   rollSet(rules.DiceSet),
		Secondary: rollSet(rules.DiceSet),
	}
}

func rollSet(set match.DiceSet) []int {
	sides := set.Dice()
	out := make([]int, 0, len(sides))
	for _, s := range sides {
		out = append(out, die.NewDie(s).Roll())
	}
	return out
}

// Derive computes the outcome from dice already rolled. Pure: same inputs, same output,
// no new dice. Every master edit goes through here.
func (rc RollCalculator) Derive(
	rules match.MatchRules, attempts RollAttempts, in RollInput,
) RollOutcome {
	bias, modifier := 0, 0
	if in.Condition != nil {
		bias += in.Condition.Bias
		modifier += in.Condition.Modifier
	}
	if in.Ledger != nil {
		bias += in.Ledger.TotalBias(in.AgainstID)
		modifier += in.Ledger.TotalAmount(in.AgainstID)
	}

	out := RollOutcome{
		SkillName:  in.SkillName,
		SkillValue: in.SkillValue,
		Bias:       bias,
		Modifier:   modifier,
		Passive:    in.Passive,
	}

	if in.Passive {
		// A passive test takes the average of the set instead of rolling, so rolling has
		// exactly zero expected gain. No dice means no critical either way.
		out.DiceTotal = rules.PassiveValue()
		out.Total = out.DiceTotal + in.SkillValue + modifier
		return out
	}

	dice := pickAttempt(attempts, bias)
	out.Dice = dice
	out.DiceTotal = sumDice(dice)
	out.IsCritical = allDiceShow(dice, rules.DiceSet.MaxFace())
	out.IsCriticalFailure = allDiceShow(dice, 1)
	out.Total = out.DiceTotal + in.SkillValue + modifier
	return out
}

// pickAttempt applies advantage and disadvantage: the better set on a positive bias, the
// worse one on a negative bias, the primary set when neutral. Magnitude beyond ±1 only
// settles the sign — accumulated advantage and disadvantage cancel each other out first.
//
// Ties fall back to Primary. With 2D10 a tie can never involve a critical: 20 only comes
// from two tens and 2 only from two ones, so there is no other combination to tie with.
func pickAttempt(a RollAttempts, bias int) []int {
	if bias == 0 || len(a.Secondary) == 0 {
		return a.Primary
	}
	if len(a.Primary) == 0 {
		return a.Secondary
	}
	if bias > 0 {
		if sumDice(a.Secondary) > sumDice(a.Primary) {
			return a.Secondary
		}
		return a.Primary
	}
	if sumDice(a.Secondary) < sumDice(a.Primary) {
		return a.Secondary
	}
	return a.Primary
}

func sumDice(dice []int) int {
	total := 0
	for _, d := range dice {
		total += d
	}
	return total
}

// allDiceShow reports whether every die of the set landed on face. A critical is read on
// the individual dice, never on the sum.
func allDiceShow(dice []int, face int) bool {
	if len(dice) == 0 {
		return false
	}
	for _, d := range dice {
		if d != face {
			return false
		}
	}
	return true
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/domain/match/service/ -v`
Expected: PASS — the new roll tests plus the pre-existing wall/visibility/turn-resolver tests.

If the build fails with an import cycle, stop: it means something under `internal/domain/match/entity/` gained an import of `internal/domain/match/service`. `service` importing its parent `match` package is intentional and cycle-free as of this branch (`match` imports only `domain`, `enum`, `entity/action`, `entity/scene`, `entity/fog`).

- [ ] **Step 5: Vet and commit**

```bash
go vet -tags=integration ./internal/...
git add internal/domain/match/service/roll_calculator.go internal/domain/match/service/roll_calculator_test.go
git commit -m "feat(match): implement RollCalculator with roll-once, derive-many

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 5: Re-key the session by character, hold NPCs, fix `CategorizeTarget`

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go:27-111` (struct + both constructors), `:153-159` (`GetCharSheet`), `:244-254` (`CategorizeTarget`)
- Modify: `internal/domain/match/matchsession/error.go`
- Modify: `internal/domain/match/matchsession/match_session_test.go:37-97`
- Modify: `internal/application/match/init_match_session.go:34-46`
- Modify: `internal/application/match/init_match_session_test.go`

**Interfaces:**
- Consumes: `match.CharacterStatus`, `match.NewCharacterStatus` (Task 3).
- Produces:
  - `MatchSession.charSheets` and `MatchSession.statuses`, both keyed by `sheetUUID`
  - `func (*MatchSession) GetCharSheet(charID uuid.UUID) (*csSheet.CharacterSheet, error)` — key changes from `playerUUID` to `sheetUUID`
  - `func (*MatchSession) GetCharacterStatus(charID uuid.UUID) (*match.CharacterStatus, error)`
  - `matchsession.ErrCharacterStatusNotFound`

### Why the key changes

The combat entity is the character, not the player. The master sends the NPCs' actions and drives several characters at once, so a `playerUUID` key cannot address them — and today the constructor's `if p.Sheet.PlayerUUID != nil` guard drops NPCs on the floor without a word.

`sheetUUID` is already the `CharacterID` the board pieces carry, so the new key matches what `Room` sends without any translation.

**What does *not* move:** `participants` stays keyed by `playerUUID`, because authorization is a per-player question, and `charToPlayer` stays as the bridge between the two axes — it is what the fog of war uses. The fog suite (`internal/app/game/fog_*_test.go`) is the safety net for that claim; it must stay green.

### The `CategorizeTarget` bug this fixes

`CategorizeTarget` compares its `id` against `participants`, which is keyed by `playerUUID` — but the `id` it receives is an `Action.TargetID`, which is the piece's `CharacterID`, i.e. a `sheetUUID`. The two key spaces never intersect, so `TargetKindCharacter` is currently unreachable and every character target falls through to wall lookup and then to `TargetKindUnknown`.

This is a pre-existing bug, not debt introduced by the re-keying — the re-keying just gives it the right map to consult. Its only caller today is `TurnResolver.Resolve`, whose character branch is empty, which is why nothing visibly broke.

- [ ] **Step 1: Write the failing tests**

In `internal/domain/match/matchsession/match_session_test.go`, replace `TestMatchSession_GetCharSheet` and `TestNewMatchSession_NPCParticipantSkipped` (lines 58–97) with:

```go
func TestMatchSession_GetCharSheet(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	sheetUUID := participant.Sheet.UUID
	sheet := &csSheet.CharacterSheet{}
	// Keyed by sheet UUID: the combat entity is the character, not the player.
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{sheetUUID: sheet}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})

	t.Run("returns the sheet for a known character", func(t *testing.T) {
		got, err := s.GetCharSheet(sheetUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != sheet {
			t.Error("expected the same sheet pointer")
		}
	})

	t.Run("the player UUID is no longer a valid key", func(t *testing.T) {
		if _, err := s.GetCharSheet(playerUUID); !errors.Is(err, matchsession.ErrCharSheetNotFound) {
			t.Errorf("expected ErrCharSheetNotFound, got %v", err)
		}
	})

	t.Run("returns ErrCharSheetNotFound for an unknown character", func(t *testing.T) {
		if _, err := s.GetCharSheet(uuid.New()); !errors.Is(err, matchsession.ErrCharSheetNotFound) {
			t.Errorf("expected ErrCharSheetNotFound, got %v", err)
		}
	})
}

func TestNewMatchSession_HoldsNPCs(t *testing.T) {
	matchUUID := uuid.New()
	// An NPC: PlayerUUID nil, MasterUUID set. It used to be dropped silently.
	npc := makeParticipant(matchUUID, nil)
	masterUUID := uuid.New()
	npc.Sheet.MasterUUID = &masterUUID
	npcSheetUUID := npc.Sheet.UUID

	npcSheet := &csSheet.CharacterSheet{}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{npcSheetUUID: npcSheet}

	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{npc})

	t.Run("the NPC sheet is reachable by sheet UUID", func(t *testing.T) {
		got, err := s.GetCharSheet(npcSheetUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != npcSheet {
			t.Error("expected the NPC sheet pointer")
		}
	})

	t.Run("the NPC has a CharacterStatus", func(t *testing.T) {
		st, err := s.GetCharacterStatus(npcSheetUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if st == nil {
			t.Fatal("expected a non-nil status")
		}
		if st.Stance != match.StanceNone {
			t.Errorf("expected StanceNone, got %q", st.Stance)
		}
	})

	t.Run("the NPC has no authorization entry", func(t *testing.T) {
		// Authorization stays per player; an NPC has no player to authorize.
		if got := len(s.PlayerIDs()); got != 0 {
			t.Errorf("expected no player IDs, got %d", got)
		}
		if got := len(s.GetCharToPlayer()); got != 0 {
			t.Errorf("expected no charToPlayer entries, got %d", got)
		}
	})
}

func TestMatchSession_GetCharacterStatus(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	t.Run("every participant gets a status", func(t *testing.T) {
		if _, err := s.GetCharacterStatus(participant.Sheet.UUID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrCharacterStatusNotFound for an unknown character", func(t *testing.T) {
		if _, err := s.GetCharacterStatus(uuid.New()); !errors.Is(err, matchsession.ErrCharacterStatusNotFound) {
			t.Errorf("expected ErrCharacterStatusNotFound, got %v", err)
		}
	})

	t.Run("the status is mutable through the session", func(t *testing.T) {
		st, _ := s.GetCharacterStatus(participant.Sheet.UUID)
		st.ActionBar.RecordSpeed(20)

		again, _ := s.GetCharacterStatus(participant.Sheet.UUID)
		if len(again.ActionBar.Speeds) != 1 {
			t.Error("expected the session to hand back the same status pointer")
		}
	})
}

func TestMatchSession_CategorizeTarget(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	t.Run("a piece CharacterID is a character", func(t *testing.T) {
		// Action.TargetID carries the piece's CharacterID, which is the sheet UUID.
		got := s.CategorizeTarget(participant.Sheet.UUID)
		if got != service.TargetKindCharacter {
			t.Errorf("expected TargetKindCharacter, got %q", got)
		}
	})

	t.Run("a wall ID is a wall segment", func(t *testing.T) {
		wallID := uuid.New()
		s.SyncMapState([]mapentity.WallSegment{{ID: wallID.String()}}, mapentity.GridShape{})

		if got := s.CategorizeTarget(wallID); got != service.TargetKindWallSegment {
			t.Errorf("expected TargetKindWallSegment, got %q", got)
		}
	})

	t.Run("anything else is unknown", func(t *testing.T) {
		if got := s.CategorizeTarget(uuid.New()); got != service.TargetKindUnknown {
			t.Errorf("expected TargetKindUnknown, got %q", got)
		}
	})
}
```

Also update `TestNewMatchSession` (line 43) so its `sheets` map is keyed by the sheet UUID:

```go
	participant := makeParticipant(matchUUID, &playerUUID)
	sheet := &csSheet.CharacterSheet{}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{participant.Sheet.UUID: sheet}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/domain/match/matchsession/ -v`
Expected: FAIL — `s.GetCharacterStatus undefined`, `undefined: matchsession.ErrCharacterStatusNotFound`, and `TestMatchSession_CategorizeTarget/a_piece_CharacterID_is_a_character` returning `unknown`.

- [ ] **Step 3: Add the error**

In `internal/domain/match/matchsession/error.go`, add to the `var` block:

```go
	ErrCharacterStatusNotFound = errors.New("character status not found in session")
```

- [ ] **Step 4: Re-key the session struct and both constructors**

In `internal/domain/match/matchsession/match_session.go`, change the two map fields (lines 32–33) and add `statuses`:

```go
	// charSheets and statuses are keyed by sheetUUID — the same ID the board pieces
	// carry as CharacterID. The combat entity is the character, not the player: the
	// master drives several characters at once, and NPCs have no player at all.
	charSheets map[uuid.UUID]*csSheet.CharacterSheet
	statuses   map[uuid.UUID]*match.CharacterStatus
	// participants stays keyed by playerUUID: authorization is a per-player question.
	// charToPlayer is the bridge between the two axes, and what the fog of war reads.
	participants map[uuid.UUID]*match.Participant
```

Replace the duplicated indexing loop in both constructors with one helper. Add it below `NewMatchSessionWithState`:

```go
// indexParticipants splits the roster along its two axes: every character gets a
// combat status (NPCs included), and only player-owned characters get an
// authorization entry and a fog bridge.
func indexParticipants(participants []*match.Participant) (
	map[uuid.UUID]*match.Participant,
	map[string]uuid.UUID,
	map[uuid.UUID]*match.CharacterStatus,
) {
	pMap := make(map[uuid.UUID]*match.Participant, len(participants))
	charToPlayer := make(map[string]uuid.UUID)
	statuses := make(map[uuid.UUID]*match.CharacterStatus, len(participants))

	for _, p := range participants {
		if p.Sheet.UUID != uuid.Nil {
			statuses[p.Sheet.UUID] = match.NewCharacterStatus()
		}
		if p.Sheet.PlayerUUID == nil {
			continue // NPC: no player to authorize, no per-player fog memory
		}
		pMap[*p.Sheet.PlayerUUID] = p
		if p.Sheet.UUID != uuid.Nil {
			charToPlayer[p.Sheet.UUID.String()] = *p.Sheet.PlayerUUID
		}
	}
	return pMap, charToPlayer, statuses
}
```

Then in `NewMatchSession`, replace the loop at lines 52–62 with:

```go
	pMap, charToPlayer, statuses := indexParticipants(participants)
```

and add `statuses: statuses,` to the returned struct literal. Do exactly the same in `NewMatchSessionWithState` (loop at lines 85–95).

- [ ] **Step 5: Update `GetCharSheet` and add `GetCharacterStatus`**

Replace `GetCharSheet` (lines 153–159) with:

```go
// GetCharSheet returns a character's sheet. charID is the sheet UUID — the same ID the
// board pieces carry as CharacterID — not the player UUID.
func (s *MatchSession) GetCharSheet(charID uuid.UUID) (*csSheet.CharacterSheet, error) {
	sheet, ok := s.charSheets[charID]
	if !ok {
		return nil, ErrCharSheetNotFound
	}
	return sheet, nil
}

// GetCharacterStatus returns a character's live combat state. The pointer is the
// session's own: mutating it mutates the session. Callers must hold room.mu.
func (s *MatchSession) GetCharacterStatus(charID uuid.UUID) (*match.CharacterStatus, error) {
	status, ok := s.statuses[charID]
	if !ok {
		return nil, ErrCharacterStatusNotFound
	}
	return status, nil
}
```

- [ ] **Step 6: Fix `CategorizeTarget`**

Replace lines 244–254 with:

```go
// CategorizeTarget returns the kind of entity the given UUID identifies.
//
// Characters are looked up in statuses, keyed by sheetUUID — which is what an
// Action.TargetID actually carries, since it comes from a board piece's CharacterID.
// It used to consult participants, keyed by playerUUID; the two key spaces never
// intersect, so TargetKindCharacter was unreachable.
//
// Characters are checked first so a character UUID is never mis-routed as a wall.
func (s *MatchSession) CategorizeTarget(id uuid.UUID) service.TargetKind {
	if _, ok := s.statuses[id]; ok {
		return service.TargetKindCharacter
	}
	if _, ok := s.walls[id.String()]; ok {
		return service.TargetKindWallSegment
	}
	return service.TargetKindUnknown
}
```

- [ ] **Step 7: Run the session tests**

Run: `go test ./internal/domain/match/matchsession/ -v`
Expected: PASS. If the test file is missing imports, add `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"` and `mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"` — both are already imported at the top of that file.

- [ ] **Step 8: Load NPCs in `InitMatchSessionUC`**

In `internal/application/match/init_match_session.go`, replace lines 34–46 with:

```go
	charSheets := make(map[uuid.UUID]*csSheet.CharacterSheet, len(participants))
	for _, p := range participants {
		// No PlayerUUID guard: an NPC has PlayerUUID nil and MasterUUID set, and the
		// master plays it. The sheet loader is keyed by sheet UUID either way.
		sheet, found, err := uc.sheetLoader.GetCharacterSheetByUUID(ctx, p.Sheet.UUID.String())
		if err != nil {
			return nil, err
		}
		if found {
			charSheets[p.Sheet.UUID] = sheet
		}
	}
```

- [ ] **Step 9: Add the UC test**

In `internal/application/match/init_match_session_test.go`, replace the `"creates session even when sheet not found (NPC case)"` sub-test (lines 93–113) with:

```go
	t.Run("loads the sheet of an NPC participant", func(t *testing.T) {
		npcSheetUUID := uuid.New()
		repo := &mockMatchRepo{
			participants: []*matchDomain.Participant{
				{
					UUID:      uuid.New(),
					MatchUUID: matchUUID,
					// NPC: no PlayerUUID. It used to be skipped before the loader ran.
					Sheet: csEntity.Summary{UUID: npcSheetUUID},
				},
			},
		}
		loader := &mockSheetLoader{sheet: &csSheet.CharacterSheet{}, found: true}

		uc := match.NewInitMatchSessionUC(repo, loader, noop)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := session.GetCharSheet(npcSheetUUID); err != nil {
			t.Errorf("expected the NPC sheet in the session, got %v", err)
		}
		if _, err := session.GetCharacterStatus(npcSheetUUID); err != nil {
			t.Errorf("expected a CharacterStatus for the NPC, got %v", err)
		}
	})

	t.Run("creates a session when the sheet is not found", func(t *testing.T) {
		repo := &mockMatchRepo{
			participants: []*matchDomain.Participant{
				{
					UUID:      uuid.New(),
					MatchUUID: matchUUID,
					Sheet:     csEntity.Summary{UUID: uuid.New()},
				},
			},
		}
		loader := &mockSheetLoader{found: false}

		uc := match.NewInitMatchSessionUC(repo, loader, noop)
		session, err := uc.Init(context.Background(), matchUUID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if session == nil {
			t.Fatal("expected a non-nil session")
		}
	})
```

- [ ] **Step 10: Run the full unit suite, including the fog safety net**

```bash
go test ./internal/... 
go vet -tags=integration ./internal/...
```

Expected: PASS everywhere. The fog suite (`internal/app/game/fog_dispatch_test.go`, `fog_regression_test.go`) is the safety net for the claim that `charToPlayer` still bridges the two axes — if it went red, the re-keying broke the fog of war and must be fixed before committing, not after.

- [ ] **Step 11: Commit**

```bash
git add internal/domain/match/matchsession/ internal/application/match/init_match_session.go internal/application/match/init_match_session_test.go
git commit -m "refactor(match): key session state by character, hold NPCs, fix CategorizeTarget

charSheets and the new statuses map move from playerUUID to sheetUUID, which is
what board pieces carry as CharacterID. participants stays on playerUUID because
authorization is per player; charToPlayer remains the bridge.

CategorizeTarget was comparing an Action.TargetID (a sheetUUID) against the
playerUUID-keyed participants map, so TargetKindCharacter was unreachable.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Task 6: Documentation

**Files:**
- Modify: `docs/dev/match/combat-engine.md`
- Modify: `docs/dev/match/flows/05-lacunas.md`
- Modify: `docs/documentation-map.yaml`

Dev docs are PT-BR prose with English code references. This task changes no code.

- [ ] **Step 1: Update `docs/dev/match/combat-engine.md`**

Change the blockquote at lines 7–8, which currently says nothing is implemented:

```markdown
> **Fase 1 implementada** (`RollCalculator`, `CharacterStatus`, `MatchRules`, chaveamento
> por personagem). O restante — colisão, barras, reações, regência — ainda não existe.
> Ver [`flows/05-lacunas.md`](flows/05-lacunas.md).
```

In the **Pendências estruturais** table (lines 292–303), replace the rows that Phase 1 closed:

```markdown
| `RollCalculator` | ✅ Fase 1 — `Roll` sorteia os dois conjuntos uma vez, `Derive` recalcula quantas vezes o mestre editar. Ninguém o chama ainda |
| `CharacterStatus` | ✅ Fase 1 — `ResourceBar` (duas barras), `ModifierLedger`, `Stance` reservado |
| Onde mora a diferença acumulada | ✅ Fase 1 — `ModifierLedger` no `CharacterStatus`, com `AgainstID`, `ExpiresAt` e `Source` |
| Conflito no `Bias` | ✅ Fase 1 — `RollCondition.Bias` é do mestre; o viés do sistema é um `Modifier` de `Source: system`, e o `RollCalculator` soma os dois em `Derive` |
```

In the **Configuração de partida** section, replace the convergence note at lines 286–288:

```markdown
> **Fase 1:** `MatchRules` existe como value object, com os padrões da tabela acima,
> recebido por parâmetro. Persistência, REST e o desbloqueio do `fog_mode` em `room.go`
> são fatia própria — `MatchRules.FogMode` é ponteiro, e a resolução é
> `partida ?? mapa ?? explored` em `MatchRules.ResolveFogMode`.
```

- [ ] **Step 2: Update `docs/dev/match/flows/05-lacunas.md`**

Four claims in this file are now false. Fix each in place, keeping the surrounding prose:

1. **§1, line 30** — *"`RollCalculator.Calculate` retorna `0` e ninguém o chama"*. The method is gone; `Roll`/`Derive` exist and still have no caller. Rewrite the paragraph to say the engine exists and that what is missing is the wiring into `TurnResolver` (Phase 2).
2. **§1, "Fricção conhecida"** — keep it. Both frictions are real and still open: `RollCheck.SkillName` is a `string` while the sheet indexes by `enum.SkillName`, and `RollContext.GetDiceResult` ignores its `die.Die` parameter. Add that Phase 1 sidestepped them by taking `SkillValue int` already resolved, and that Phase 2 has to face them.
3. **§6, lines 86–96** — *"`CharacterStatus` é só um documento de design"*. It is code now. Rewrite the section: the struct carries `ActionBar`/`MoveBar` (`ResourceBar`), `Ledger`, `Stance` and `Velocity`; `Position` was deliberately left out because it lives in the `Room`; the rationale comment block is preserved in the file. Keep the note that "`CharacterStatus` não será persistido" is still a pending architectural choice.
4. **§ line 117** — the `AttachReaction` row mentioning `s.charSheets`. Add that the map is now keyed by `sheetUUID`.

- [ ] **Step 3: Update `docs/documentation-map.yaml`**

Add these entries next to the existing `internal/domain/match/service/` block (around line 326):

```yaml
  - code_path: internal/domain/match/service/roll_calculator.go
    dev_docs:
      - path: docs/dev/match/combat-engine.md
        confidence: directly_affected
    game_docs:
      - path: docs/game/dados.md
        confidence: directly_affected
    notes: RollCalculator — Roll (once) + Derive (many); crítico pela combinação, passivo, viés acumulado

  - code_path: internal/domain/match/character_status.go
    dev_docs:
      - path: docs/dev/match/combat-engine.md
        confidence: directly_affected
    game_docs:
      - path: docs/game/combate/barra-de-acao.md
        confidence: possibly_affected
    notes: CharacterStatus — duas ResourceBar, ModifierLedger, Stance reservado

  - code_path: internal/domain/match/modifier.go
    dev_docs:
      - path: docs/dev/match/combat-engine.md
        confidence: directly_affected
    notes: ModifierLedger — bônus/penalidade acumulados, com escopo, alvo e origem (system|master)

  - code_path: internal/domain/match/match_rules.go
    dev_docs:
      - path: docs/dev/match/combat-engine.md
        confidence: directly_affected
    notes: MatchRules — conjunto de dados, degrau da escada, timer de reação, fog_mode (partida ?? mapa)
```

- [ ] **Step 4: Verify the game docs need no change**

`docs/game/` holds game rules only. Phase 1 implements rules that `docs/game/dados.md` already describes correctly (2 D10, critical by combination, advantage/disadvantage) and adds no player-visible behavior, since nothing calls the engine yet.

Read `docs/game/dados.md` and confirm it matches the implementation. If it does, no edit — record "no player-visible behaviour change; the engine has no caller yet" as the skip justification for the PR description. If it disagrees with the implementation, **stop**: `combat-engine.md:52` warns that `dados.md` is the corrected reference and the implementation must match it, so a mismatch means the code is wrong, not the doc.

- [ ] **Step 5: Commit**

```bash
git add docs/
git commit -m "docs(match): record phase 1 of the combat engine as implemented

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Delivery

Phase 1 is pure domain code with no caller, so there is no browser flow and no endpoint to curl. The delivery rule in the root `CLAUDE.md` allows substituting equivalent automated evidence when building the real scenario would take reverse-engineering out of proportion to the change — here there is no scenario at all to build, because the engine is deliberately unwired.

- [ ] **Step 1: Full verification**

```bash
go build ./...
go test ./...
go vet -tags=integration ./internal/...
go vet -tags=smoke ./internal/...
```

All four must pass. Paste the real output into the PR — do not claim a pass you did not see.

- [ ] **Step 2: Confirm every "done when" from the spec**

| Spec criterion | Where it is proved |
|---|---|
| Critical, critical failure, passive, advantage, accumulated disadvantage, `Margin(cd)` | `service/roll_calculator_test.go` |
| A participant with `PlayerUUID == nil` has a reachable sheet and `CharacterStatus`, by `sheetUUID` | `matchsession/match_session_test.go` → `TestNewMatchSession_HoldsNPCs` |
| `CategorizeTarget` returns `TargetKindCharacter` for a piece's `sheetUUID` | `matchsession/match_session_test.go` → `TestMatchSession_CategorizeTarget` |
| `go vet -tags=integration ./internal/...` passes | Step 1 output |

- [ ] **Step 3: Open the PR**

Title: `feat(match): combat engine phase 1 — state and roll engine`

The body must say, explicitly:
- what was verified (the four commands above, with output);
- that **nothing calls `RollCalculator` yet** — that is Phase 2, by design;
- that **no NPC can exist in a real match yet**: `start_match.go` only populates `match_participants` from accepted enrollments. Phase 1 makes the session *able* to hold one; creating one is a separate slice that blocks Phase 4 and has a product question inside it (how does the master add an NPC?). See spec §7;
- that `MatchRules` is not persisted and no endpoint exposes it, so `fog_mode` in `room.go` is **still hardcoded** — the resolution function exists (`ResolveFogMode`) but has no caller. Separate slice, spec §4.6;
- the docs skip justification from Task 6 Step 4.

`./dev-checkout.sh` is **not** needed: there is nothing to validate by hand, since no behavior reaches the browser or the API. Say that in the PR instead of running it.
