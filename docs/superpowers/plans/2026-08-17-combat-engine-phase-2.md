# Combat Engine — Phase 2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** One attack against one character resolves end to end — dice fall once when the
action arrives, the resolver derives hit / passive dodge / passive defense / damage as a
dry-run the master can see, and the HP only changes when the turn closes.

**Architecture:** The dice are rolled **once**, at enqueue time, by `MatchSession`, and stored
in `action.RollCheck.Attempts`. `TurnResolver` therefore becomes a **pure function of the
turn** — it derives, it never rolls. That is what makes re-resolving on every reaction (Phase
4) and every master edit (Phase 5) free of re-rolls, and what makes Phase 3's and Phase 4's
exact-number done-criteria reachable: a test injects a scripted `RollSource` and the whole
pipeline downstream is deterministic. Damage is a *projection* carried in the resolution and
applied exactly once, inside the implicit turn close that already happens in
`OpenNextAction`/`PullAction`.

**Tech Stack:** Go 1.23, standard `testing` (table-driven, `t.Run`), PostgreSQL via pgx,
gorilla-style WebSocket delivery in `internal/app/game/`.

**Spec:** `docs/superpowers/specs/2026-08-16-combat-engine-design.md` — §4.7 (damage) and
§5 "Fase 2" are the binding sections. Supporting rules:
`docs/dev/match/combat-engine.md`, `docs/game/combate/reacoes.md`,
`docs/dev/match/flows/05-lacunas.md` §3 and §9.

**Branch:** `feat/combat-engine-phase-2`, based on `feat/combat-engine-phase-1` with `main`
merged in (Phase 1 is not on `main` yet — a GitHub incident blocked that merge).

---

## Global Constraints

- **Go 1.23**, module `github.com/422UR4H/HxH_RPG_System`. No test frameworks — standard
  `testing` only, table-driven with `t.Run()`, external test packages (`package foo_test`).
- **NEVER remove a TODO comment.** They are intentional markers left by the repo owner.
- **Layering:** `entity ← domain ← app`, `entity ← gateway`. Entities never import outer
  layers. Concretely here: `service` imports `action`; **`action` must never import
  `service`**, and `battle` must never import `service`.
- **Domain services are stateless structs** in `internal/domain/match/service/`. Dependencies
  travel as parameters, not as fields. `service.TurnResolver{}` and
  `service.RollCalculator{}` must keep working as zero values.
- **`MatchSession` has no lock of its own.** `room.go`'s `r.mu` is the only serialization —
  every new call into the session inherits the obligation to hold it, write-lock whenever the
  session mutates.
- **Wire format is camelCase** on both sides, via manual struct tags.
- **The master never re-rolls a player's die.** Every number is derived from dice that fell
  once, when the action arrived.
- **The hit margin never feeds damage.** Product decision, deliberate — must be recorded as a
  code comment where the damage is computed (spec §4.7).
- **Critical and critical failure must pass through untouched.** The engine flags them; no
  rule consumes them yet, and Phase 2 must not invent one (spec §4.7 "Em aberto").
- **Ledger never applies to a hit roll.** `RollInput.Ledger` is `nil` for hit; the accumulated
  difference is always an actionSpeed adjustment (spec §3).
- Commits include `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`.
- After every task touching `internal/`: `go vet -tags=integration ./internal/...`.

---

## Decisions taken while planning (not in the spec — these are technical form)

These were decided here so the implementer does not have to. They are recorded because a
later phase will read this plan as history.

1. **Ties favour the defender.** A passive dodge succeeds when `dodgeTotal >= hitTotal`;
   a passive defense succeeds when `defenseTotal >= hitTotal - LadderStep`. This mirrors the
   repel table in `combat-engine.md`, where landing exactly on `CD` is a defender success.
2. **`RollSource` is a parameter, not a struct field**, so `RollCalculator{}` stays a stateless
   zero value. `nil` source means `DiceRoller{}` (production, crypto/rand).
3. **The dice are rolled in `MatchSession`, at enqueue/attach time** — not in the resolver, not
   in the mapper. The session is the only place that holds both `MatchRules` and the weapon
   catalogue, and "when the action arrives" is exactly `EnqueueAction`/`AttachReaction`.
4. **Weapon damage dice live in `Attack.Damage.Attempts.Primary`.** Same roll-once rule as any
   other test; `Secondary` stays empty because damage has no advantage.
5. **`TurnResolver.Resolve` takes a `ResolveInput` struct** instead of growing its positional
   parameter list a fourth and fifth time. After this phase only `MatchSession` calls it.
6. **`MatchSession.OpenNextAction`/`PullAction` return a `TurnTransition` struct** rather than
   five values.
7. **Persistence of the damaged sheets lives in the use case**, not in `Room` — `Room`'s
   constructor already takes eleven dependencies, and use cases are the layer that orchestrates
   gateways. `cmd/game/main.go` already has `sheetRepository` in scope.
8. **Unarmed means `enum.Fist`.** The weapon catalogue already carries it (`D6+D6+D4`, flat
   damage 0), so "no weapon" resolves to a real catalogue entry instead of a special case.

---

## File Structure

**Created**

| File | Responsibility |
|---|---|
| `internal/domain/match/entity/action/roll_attempts.go` | `RollAttempts` — the two dice sets, rolled once. Moved here from `service` so `RollCheck` can hold it. |
| `internal/domain/match/service/roll_source.go` | `RollSource` seam + production `DiceRoller`. |
| `internal/domain/match/service/margin_ladder.go` | `ClimbLadder` — the margin ladder as a pure function. |
| `internal/domain/match/service/margin_ladder_test.go` | Boundary tests for every rung. |
| `internal/domain/match/service/damage.go` | Weapon damage arithmetic and the "applicable defense" rules of spec §4.7. |
| `internal/domain/match/service/damage_test.go` | Damage arithmetic tests. |
| `internal/domain/match/service/character_collision_test.go` | The character branch of the resolver, end to end, with a scripted roll source. |
| `internal/application/match/i_sheet_status_writer.go` | `ISheetStatusWriter` — the gateway port the use cases persist damage through. |

**Modified**

| File | Change |
|---|---|
| `internal/domain/match/entity/action/roll_check.go` | `RollCheck` gains `Attempts RollAttempts`. |
| `internal/domain/match/entity/battle/blow.go` | Constructor + accessors; `defense` becomes a pointer. |
| `internal/domain/match/service/roll_calculator.go` | Drops its local `RollAttempts`; `Roll` takes a `RollSource`; gains `RollDice`. |
| `internal/domain/match/service/roll_calculator_test.go` | Follows the type move and the new `Roll` signature. |
| `internal/domain/match/service/turn_resolver.go` | `ResolveInput`; the `TargetKindCharacter` branch; `CharacterResult`; richer `RollResult`. |
| `internal/domain/match/service/turn_resolver_test.go` | Follows the new `Resolve` signature. |
| `internal/domain/match/matchsession/match_session.go` | Holds `MatchRules`, the weapon catalogue and the roll source; rolls dice on enqueue; resolves; applies damage on turn close. |
| `internal/domain/match/matchsession/match_session_test.go` | Actor axis is now the sheet UUID; new tests for the rules, the roll-once behaviour and the damage application. |
| `internal/domain/match/matchsession/error.go` | New sentinel errors. |
| `internal/application/match/open_next_action.go` · `pull_action.go` | Take the status writer; return the closed turn's resolution; persist damaged sheets. |
| `internal/application/match/attach_reaction.go` | Follows the session's new resolve path. |
| `internal/application/match/*_test.go` | Actor axis; new mock writer. |
| `internal/app/game/message.go` | `ActionPayload.ActorID`; the real `ResolutionUpdatedPayload`. |
| `internal/app/game/action_mapper.go` | Full payload mapping + the `string → enum.SkillName` boundary. |
| `internal/app/game/room.go` | Requires `actorId`; emits the real `resolution_updated`. |
| `cmd/game/main.go` | Wires `sheetRepository` into the two use cases. |
| `docs/dev/match/combat-engine.md` · `flows/05-lacunas.md` · `documentation-map.yaml` · `AGENTS.md` | Record what Phase 2 closed. |

---

## Task 1: `RollAttempts` moves to `action`, and `RollCheck` carries it

The dice have nowhere to live today. `service.RollAttempts` cannot be a field of
`action.RollCheck`, because `service` imports `action` and the reverse would be a cycle. The
type moves down to `action`, which is where it belongs anyway: it is the raw material of a
`RollCheck`, not a service concern.

**Files:**
- Create: `internal/domain/match/entity/action/roll_attempts.go`
- Modify: `internal/domain/match/entity/action/roll_check.go`
- Modify: `internal/domain/match/service/roll_calculator.go:10-21` (delete the type), `:72-92`
  (signatures)
- Test: `internal/domain/match/entity/action/roll_attempts_test.go`
- Modify: `internal/domain/match/service/roll_calculator_test.go`

**Interfaces:**
- Produces: `action.RollAttempts{Primary, Secondary []int}` with `IsEmpty() bool`;
  `action.RollCheck.Attempts RollAttempts`;
  `service.RollCalculator.Derive(rules match.MatchRules, attempts action.RollAttempts, in RollInput) RollOutcome`.

- [ ] **Step 1: Write the failing test**

`internal/domain/match/entity/action/roll_attempts_test.go`:

```go
package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

func TestRollAttempts_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		a    action.RollAttempts
		want bool
	}{
		{"zero value is empty", action.RollAttempts{}, true},
		{"only primary is not empty", action.RollAttempts{Primary: []int{3, 7}}, false},
		{"only secondary is not empty", action.RollAttempts{Secondary: []int{3, 7}}, false},
		{"both sets is not empty", action.RollAttempts{Primary: []int{1}, Secondary: []int{2}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRollCheck_CarriesAttempts(t *testing.T) {
	rc := action.RollCheck{SkillName: "Accuracy"}
	if !rc.Attempts.IsEmpty() {
		t.Fatal("a fresh RollCheck must carry no dice")
	}
	rc.Attempts = action.RollAttempts{Primary: []int{10, 10}}
	if rc.Attempts.IsEmpty() {
		t.Error("expected the RollCheck to hold the dice it was given")
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/domain/match/entity/action/ -run 'RollAttempts|RollCheck_Carries' -v`
Expected: FAIL — `undefined: action.RollAttempts`.

- [ ] **Step 3: Create the type**

`internal/domain/match/entity/action/roll_attempts.go`:

```go
package action

// RollAttempts holds BOTH dice sets, rolled together the moment the action or reaction
// arrives.
//
// Advantage means rolling the set twice and keeping the better one — but the master can
// grant advantage *after* the dice have already fallen, and the master never re-rolls a
// player's die. Rolling both sets up front is the only shape that satisfies both: a later
// edit changes which set is read, never what was rolled. On a neutral bias, Secondary is
// simply never read.
//
// It lives in this package, not in the domain service, so a RollCheck can hold it:
// service imports action, never the reverse.
type RollAttempts struct {
	Primary   []int
	Secondary []int
}

// IsEmpty reports whether the dice have not fallen yet. It is what keeps the roll-once
// rule honest: whoever rolls checks this first and never overwrites a set that already
// landed.
func (a RollAttempts) IsEmpty() bool {
	return len(a.Primary) == 0 && len(a.Secondary) == 0
}
```

- [ ] **Step 4: Add the field to `RollCheck`**

`internal/domain/match/entity/action/roll_check.go` — replace the whole struct:

```go
package action

type RollCheck struct {
	Context    RollContext // strategy set dice based on campaign\match rules
	SkillName  string      // skill used for the roll check (test)
	SkillValue int         // filled with ValueForTest of the character sheet
	// Attempts are the dice, rolled once when the action arrived. Derive reads them as
	// many times as the master edits, and never rolls again.
	Attempts RollAttempts
	Result   int
}
```

- [ ] **Step 5: Delete the service-local copy and re-point the calculator**

In `internal/domain/match/service/roll_calculator.go`, delete the `RollAttempts` struct and
its doc comment (lines 10–21), then change the two signatures and the internal uses.
`Roll` keeps its parameter list — only the return type changes:

```go
// Roll rolls both sets for the given rules. Called once, when the action or reaction
// arrives.
func (rc RollCalculator) Roll(rules match.MatchRules) action.RollAttempts {
	return action.RollAttempts{
		Primary:   rollSet(rules.DiceSet),
		Secondary: rollSet(rules.DiceSet),
	}
}
```

```go
func (rc RollCalculator) Derive(
	rules match.MatchRules, attempts action.RollAttempts, in RollInput,
) RollOutcome {
```

```go
func pickAttempt(a action.RollAttempts, bias int) []int {
```

`rollSet` stays exactly as it is. The `src RollSource` parameter arrives in Task 2, so this
task stays a pure type move.

- [ ] **Step 6: Update `roll_calculator_test.go`**

Replace every `service.RollAttempts{` with `action.RollAttempts{` and add the import
`"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"` if it is not already
there. Do not change any assertion.

- [ ] **Step 7: Run the tests**

Run: `go test ./internal/domain/match/... && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/domain/match/entity/action/ internal/domain/match/service/roll_calculator.go internal/domain/match/service/roll_calculator_test.go
git commit -m "refactor(match): move RollAttempts into action so RollCheck can hold the dice

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: The deterministic roll seam

`RollCalculator.Roll` calls `die.NewDie(s).Roll()` directly, so everything downstream of it is
irreproducible. Phase 3 (`p1=20, p2=23, p3=11 → +9 / 0 / −2`) and Phase 4 (opening order
changes the outcome) both have done-criteria with exact numbers. The seam has to exist before
those phases, and Phase 2 is where it is cheap.

**Files:**
- Create: `internal/domain/match/service/roll_source.go`
- Modify: `internal/domain/match/service/roll_calculator.go`
- Test: `internal/domain/match/service/roll_source_test.go`
- Modify: `internal/domain/match/service/roll_calculator_test.go`

**Interfaces:**
- Consumes: `action.RollAttempts` (Task 1).
- Produces: `service.RollSource` interface with `RollDie(sides enum.DieSides) int`;
  `service.DiceRoller{}` (production);
  `service.RollCalculator.Roll(rules match.MatchRules, src RollSource) action.RollAttempts`;
  `service.RollCalculator.RollDice(sides []enum.DieSides, src RollSource) []int`.

- [ ] **Step 1: Write the failing test**

`internal/domain/match/service/roll_source_test.go`:

```go
package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

// scriptedSource hands out faces in the order given and repeats the last one once the
// script runs out. Every test in this package that needs an exact number uses it.
type scriptedSource struct {
	faces []int
	i     int
}

func (s *scriptedSource) RollDie(_ enum.DieSides) int {
	if len(s.faces) == 0 {
		return 1
	}
	if s.i >= len(s.faces) {
		return s.faces[len(s.faces)-1]
	}
	f := s.faces[s.i]
	s.i++
	return f
}

func TestRollCalculator_Roll_UsesTheGivenSource(t *testing.T) {
	rules := match.NewDefaultMatchRules()
	src := &scriptedSource{faces: []int{10, 10, 3, 4}}

	got := service.RollCalculator{}.Roll(rules, src)

	if len(got.Primary) != 2 || got.Primary[0] != 10 || got.Primary[1] != 10 {
		t.Errorf("Primary = %v, want [10 10]", got.Primary)
	}
	if len(got.Secondary) != 2 || got.Secondary[0] != 3 || got.Secondary[1] != 4 {
		t.Errorf("Secondary = %v, want [3 4]", got.Secondary)
	}
}

func TestRollCalculator_Roll_NilSourceStillRolls(t *testing.T) {
	// A nil source means production: the real roller. The values are random, so this
	// asserts only that dice actually fell and stayed in range.
	rules := match.NewDefaultMatchRules()

	got := service.RollCalculator{}.Roll(rules, nil)

	if len(got.Primary) != 2 || len(got.Secondary) != 2 {
		t.Fatalf("expected 2 dice per set, got %v / %v", got.Primary, got.Secondary)
	}
	for _, d := range append(append([]int{}, got.Primary...), got.Secondary...) {
		if d < 1 || d > 10 {
			t.Errorf("die out of range for 2D10: %d", d)
		}
	}
}

func TestRollCalculator_RollDice_RollsAnArbitrarySet(t *testing.T) {
	// Weapon damage is not the match dice set: a Sword is D10 + D4.
	src := &scriptedSource{faces: []int{9, 2}}

	got := service.RollCalculator{}.RollDice([]enum.DieSides{enum.D10, enum.D4}, src)

	if len(got) != 2 || got[0] != 9 || got[1] != 2 {
		t.Errorf("RollDice() = %v, want [9 2]", got)
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/domain/match/service/ -run RollCalculator_Roll -v`
Expected: FAIL — too many arguments to `Roll`, and `RollDice` undefined.

- [ ] **Step 3: Create the seam**

`internal/domain/match/service/roll_source.go`:

```go
package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/die"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// RollSource is where a die face comes from.
//
// It exists so a test can be reproducible. Derive is already deterministic — it receives
// dice that already fell — but everything upstream of it went through crypto/rand with no
// way in. The phases after this one have done-criteria stated as exact numbers (the round
// economy, the multi-target chain), and an economy test that depends on luck is not a test.
//
// Production passes nil or DiceRoller{}; tests pass a scripted source.
type RollSource interface {
	RollDie(sides enum.DieSides) int
}

// DiceRoller is the production source: crypto/rand, with the math/rand fallback that
// die.Die already implements.
type DiceRoller struct{}

func (DiceRoller) RollDie(sides enum.DieSides) int { return die.NewDie(sides).Roll() }

// sourceOrDefault keeps every call site free of nil checks. A nil source means production.
func sourceOrDefault(src RollSource) RollSource {
	if src == nil {
		return DiceRoller{}
	}
	return src
}
```

- [ ] **Step 4: Thread the source through the calculator**

In `internal/domain/match/service/roll_calculator.go`, replace `Roll` and `rollSet`:

```go
// Roll rolls both sets for the given rules. Called once, when the action or reaction
// arrives. A nil src means the production roller.
func (rc RollCalculator) Roll(rules match.MatchRules, src RollSource) action.RollAttempts {
	return action.RollAttempts{
		Primary:   rc.RollDice(rules.DiceSet.Dice(), src),
		Secondary: rc.RollDice(rules.DiceSet.Dice(), src),
	}
}

// RollDice rolls an arbitrary set of dice, in order. Damage needs it: a weapon carries its
// own dice (a Sword is D10 + D4), which are not the match's test set.
func (rc RollCalculator) RollDice(sides []enum.DieSides, src RollSource) []int {
	s := sourceOrDefault(src)
	out := make([]int, 0, len(sides))
	for _, face := range sides {
		out = append(out, s.RollDie(face))
	}
	return out
}
```

Delete the old `rollSet` function and add the `enum` import:
`"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"`. The `die` import moves to
`roll_source.go` — remove it from `roll_calculator.go` if nothing else there uses it.

- [ ] **Step 5: Update existing `Roll` call sites in the tests**

In `roll_calculator_test.go`, every `.Roll(rules)` becomes `.Roll(rules, nil)`.

- [ ] **Step 6: Run the tests**

Run: `go test ./internal/domain/match/... && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/match/service/
git commit -m "feat(match): add a RollSource seam so rolls can be scripted in tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3: The margin ladder as a pure function

The ladder is the shape of every opposed reading in the system. Phase 4 wires the repel
reaction into it; Phase 2 writes it and tests it standing alone, with no reaction attached.

The rungs come from `combat-engine.md` § "A escada de resultados", read as the defender's
margin against `CD = the attack's result`:

| Table row | Margin | Rung |
|---|---|---|
| `≥ CD + 10` | `margin >= step` | `RungGreatSuccess` — zero damage **and** a bonus equal to the difference, against *that* opponent |
| `CD … CD+9` | `0 <= margin < step` | `RungSuccess` — zero damage |
| `CD−10 … CD−1` | `-step <= margin < 0` | `RungNearMiss` — parried: zero damage, and a penalty equal to the difference, against anyone |
| `< CD − 10` | `margin < -step` | `RungFailure` — the attack lands |

**Files:**
- Create: `internal/domain/match/service/margin_ladder.go`
- Test: `internal/domain/match/service/margin_ladder_test.go`

**Interfaces:**
- Produces: `service.LadderRung` (`RungGreatSuccess`/`RungSuccess`/`RungNearMiss`/`RungFailure`);
  `service.LadderOutcome{Rung, Margin, Difference}`;
  `service.ClimbLadder(margin, step int) LadderOutcome`.

- [ ] **Step 1: Write the failing test**

`internal/domain/match/service/margin_ladder_test.go`:

```go
package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func TestClimbLadder(t *testing.T) {
	const step = 10

	tests := []struct {
		name       string
		margin     int
		wantRung   service.LadderRung
		wantDiff   int
	}{
		{"a full step over is a great success", 10, service.RungGreatSuccess, 10},
		{"well over is a great success", 27, service.RungGreatSuccess, 27},
		{"one under the step is a plain success", 9, service.RungSuccess, 0},
		{"landing exactly on the CD is a success", 0, service.RungSuccess, 0},
		{"one under the CD is a near miss", -1, service.RungNearMiss, 1},
		{"a full step under is still a near miss", -10, service.RungNearMiss, 10},
		{"more than a step under is a failure", -11, service.RungFailure, 0},
		{"far under is a failure", -40, service.RungFailure, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.ClimbLadder(tt.margin, step)
			if got.Rung != tt.wantRung {
				t.Errorf("Rung = %q, want %q", got.Rung, tt.wantRung)
			}
			if got.Difference != tt.wantDiff {
				t.Errorf("Difference = %d, want %d", got.Difference, tt.wantDiff)
			}
			if got.Margin != tt.margin {
				t.Errorf("Margin = %d, want %d", got.Margin, tt.margin)
			}
		})
	}
}

func TestClimbLadder_StepIsAMatchRule(t *testing.T) {
	// The step size is configurable; the shape of the ladder is not. With a step of 5 the
	// same margin lands on a different rung.
	if got := service.ClimbLadder(6, 5); got.Rung != service.RungGreatSuccess {
		t.Errorf("with step 5, margin 6 should be a great success, got %q", got.Rung)
	}
	if got := service.ClimbLadder(6, 10); got.Rung != service.RungSuccess {
		t.Errorf("with step 10, margin 6 should be a plain success, got %q", got.Rung)
	}
}

func TestClimbLadder_NonPositiveStepDegradesToPassFail(t *testing.T) {
	// Defensive: a zero step must not collapse every margin onto one rung silently.
	if got := service.ClimbLadder(3, 0); got.Rung != service.RungGreatSuccess {
		t.Errorf("with step 0, any success is a great success, got %q", got.Rung)
	}
	if got := service.ClimbLadder(-3, 0); got.Rung != service.RungFailure {
		t.Errorf("with step 0, any miss is a failure, got %q", got.Rung)
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/domain/match/service/ -run ClimbLadder -v`
Expected: FAIL — `undefined: service.ClimbLadder`.

- [ ] **Step 3: Write the ladder**

`internal/domain/match/service/margin_ladder.go`:

```go
package service

// LadderRung is where a margin landed.
//
// The shape of the ladder — how many rungs and what each one does — stays in code, because
// changing it changes the game. Only the step size is a match rule (MatchRules.LadderStep).
//
// The asymmetry is intentional and comes from the product owner: "it should be easier to
// parry than to hit the target; this system is already very punishing." Hence the near-miss
// rung, where failing by less than a step still costs the attacker the damage.
type LadderRung string

const (
	// RungGreatSuccess: cleared the CD by a full step or more. Zero damage, plus a bonus
	// equal to the difference — and that bonus is specific to the opponent read.
	RungGreatSuccess LadderRung = "great_success"
	// RungSuccess: cleared the CD. Zero damage.
	RungSuccess LadderRung = "success"
	// RungNearMiss: missed by less than a full step. Parried, which is zero damage, not
	// reduced damage — the price is a penalty equal to the difference, and that penalty is
	// general, because being off balance is something anyone can exploit.
	RungNearMiss LadderRung = "near_miss"
	// RungFailure: missed by a full step or more. The attack lands.
	RungFailure LadderRung = "failure"
)

// LadderOutcome is a margin read against the ladder.
type LadderOutcome struct {
	Rung   LadderRung
	Margin int
	// Difference is the size of the bonus on RungGreatSuccess and of the penalty on
	// RungNearMiss — the only two rungs that pay out into the ModifierLedger. It is zero
	// on the other two.
	Difference int
}

// ClimbLadder reads margin against a ladder of the given step.
//
// margin is the defender's total minus the CD, so a margin of 0 — landing exactly on the CD
// — is a success: ties favour the defender, the same way the repel table treats CD itself as
// a defender's row.
//
// Nothing is wired into this yet. Phase 4 hands it the repel reaction; Phase 2 only needs it
// to exist and be right.
func ClimbLadder(margin, step int) LadderOutcome {
	out := LadderOutcome{Margin: margin}
	switch {
	case margin >= step:
		out.Rung = RungGreatSuccess
		out.Difference = margin
	case margin >= 0:
		out.Rung = RungSuccess
	case margin >= -step:
		out.Rung = RungNearMiss
		out.Difference = -margin
	default:
		out.Rung = RungFailure
	}
	return out
}
```

- [ ] **Step 4: Run the test and confirm it passes**

Run: `go test ./internal/domain/match/service/ -run ClimbLadder -v`
Expected: PASS, all sub-tests.

> Check the zero-step cases by hand: with `step == 0`, `margin >= 0` hits the first arm
> (`margin >= step`) and becomes a great success; `margin < 0` falls past `margin >= -step`
> and becomes a failure. That is what the third test asserts.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/margin_ladder.go internal/domain/match/service/margin_ladder_test.go
git commit -m "feat(match): add the margin ladder as a pure function

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: Damage arithmetic

Spec §4.7. Two families of roll, not one: the test uses `MatchRules.DiceSet`, the damage uses
the weapon's own dice plus its flat bonus.

Note what the weapon catalogue actually holds today: **every weapon has `defense == 0`**
(`internal/domain/entity/item/weapons_factory.go`). So the subtraction is structurally present
and numerically inert until the catalogue is filled in. That is fine and must be said in a
comment — the shape is what Phase 2 owes, not the numbers.

**Files:**
- Create: `internal/domain/match/service/damage.go`
- Test: `internal/domain/match/service/damage_test.go`

**Interfaces:**
- Consumes: `service.RollCalculator.RollDice` (Task 2), `action.RollAttempts` (Task 1).
- Produces:
  `service.WeaponDice(name *enum.WeaponName, cat *item.WeaponsManager) ([]enum.DieSides, error)`;
  `service.RawDamage(dice []int, name *enum.WeaponName, cat *item.WeaponsManager) (int, error)`;
  `service.ApplicableDefense(in DefenseInput) DefenseOutcome`;
  `service.DefenseInput{AttackWeapon, DefenseWeapon *enum.WeaponName, Defended bool, Catalogue *item.WeaponsManager}`;
  `service.DefenseOutcome{Amount int, BlocksEntirely bool}`;
  `service.EffectiveDamage(raw int, d DefenseOutcome) int`.

- [ ] **Step 1: Write the failing test**

`internal/domain/match/service/damage_test.go`:

```go
package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func catalogue() *item.WeaponsManager { return item.NewWeaponsManagerFactory().Build() }

func TestWeaponDice(t *testing.T) {
	cat := catalogue()

	t.Run("a named weapon yields its own dice", func(t *testing.T) {
		sword := enum.Sword
		got, err := service.WeaponDice(&sword, cat)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 || got[0] != enum.D10 || got[1] != enum.D4 {
			t.Errorf("Sword dice = %v, want [D10 D4]", got)
		}
	})

	t.Run("no weapon means bare hands, which is a real catalogue entry", func(t *testing.T) {
		got, err := service.WeaponDice(nil, cat)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 3 {
			t.Errorf("Fist dice = %v, want 3 dice", got)
		}
	})

	t.Run("an unknown weapon is an error, not a silent zero", func(t *testing.T) {
		bogus := enum.WeaponName("Excalibur")
		if _, err := service.WeaponDice(&bogus, cat); err == nil {
			t.Error("expected an error for a weapon outside the catalogue")
		}
	})
}

func TestRawDamage(t *testing.T) {
	cat := catalogue()

	t.Run("raw damage is the dice plus the weapon's flat bonus", func(t *testing.T) {
		sword := enum.Sword // D10 + D4, flat damage 2
		got, err := service.RawDamage([]int{7, 3}, &sword, cat)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 12 { // 7 + 3 + 2
			t.Errorf("RawDamage() = %d, want 12", got)
		}
	})

	t.Run("bare hands add nothing flat", func(t *testing.T) {
		got, err := service.RawDamage([]int{4, 5, 1}, nil, cat) // Fist, flat damage 0
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 10 {
			t.Errorf("RawDamage() = %d, want 10", got)
		}
	})
}

func TestApplicableDefense(t *testing.T) {
	cat := catalogue()
	sword, dagger := enum.Sword, enum.Dagger

	t.Run("a failed defense subtracts nothing", func(t *testing.T) {
		got := service.ApplicableDefense(service.DefenseInput{
			AttackWeapon: &sword, DefenseWeapon: &dagger, Defended: false, Catalogue: cat,
		})
		if got.Amount != 0 || got.BlocksEntirely {
			t.Errorf("got %+v, want a no-op", got)
		}
	})

	t.Run("weapon against weapon lets nothing through", func(t *testing.T) {
		got := service.ApplicableDefense(service.DefenseInput{
			AttackWeapon: &sword, DefenseWeapon: &dagger, Defended: true, Catalogue: cat,
		})
		if !got.BlocksEntirely {
			t.Error("expected an armed defense against an armed attack to block entirely")
		}
	})

	t.Run("an unarmed attack passes damage through the defense", func(t *testing.T) {
		got := service.ApplicableDefense(service.DefenseInput{
			AttackWeapon: nil, DefenseWeapon: &dagger, Defended: true, Catalogue: cat,
		})
		if got.BlocksEntirely {
			t.Error("an unarmed attack must not be blocked entirely")
		}
		// Every weapon in the catalogue currently carries defense 0, so this is 0 today.
		// The assertion is on the shape, not on the number.
		if got.Amount != 0 {
			t.Errorf("Amount = %d, want the dagger's defense bonus (0 in today's catalogue)", got.Amount)
		}
	})

	t.Run("an armed attack against a bare-handed defense is not reduced", func(t *testing.T) {
		got := service.ApplicableDefense(service.DefenseInput{
			AttackWeapon: &sword, DefenseWeapon: nil, Defended: true, Catalogue: cat,
		})
		if got.BlocksEntirely || got.Amount != 0 {
			t.Errorf("got %+v, want no reduction while damage types do not exist", got)
		}
	})
}

func TestEffectiveDamage(t *testing.T) {
	tests := []struct {
		name string
		raw  int
		def  service.DefenseOutcome
		want int
	}{
		{"undefended damage passes whole", 12, service.DefenseOutcome{}, 12},
		{"a blocking defense zeroes it", 12, service.DefenseOutcome{BlocksEntirely: true}, 0},
		{"a reducing defense subtracts", 12, service.DefenseOutcome{Amount: 5}, 7},
		{"the reduction never goes below zero", 3, service.DefenseOutcome{Amount: 9}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.EffectiveDamage(tt.raw, tt.def); got != tt.want {
				t.Errorf("EffectiveDamage() = %d, want %d", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/domain/match/service/ -run 'WeaponDice|RawDamage|ApplicableDefense|EffectiveDamage' -v`
Expected: FAIL — undefined symbols.

- [ ] **Step 3: Write the damage service**

`internal/domain/match/service/damage.go`:

```go
package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

// unarmed is what "no weapon" resolves to. The catalogue already carries a Fist entry
// (D6 + D6 + D4, flat damage 0), so bare hands are a weapon like any other instead of a
// branch everything downstream has to remember.
const unarmed = enum.Fist

// WeaponDice returns the dice a weapon rolls for damage.
//
// This is the second family of roll in the system, and it is NOT MatchRules.DiceSet. The
// test set (hit, skill, actionSpeed) is 2 D10 for everyone; damage comes from the weapon —
// a Sword is D10 + D4, a Rifle is D12 + D10. Spec §4.7.
func WeaponDice(name *enum.WeaponName, cat *item.WeaponsManager) ([]enum.DieSides, error) {
	w, err := lookup(name, cat)
	if err != nil {
		return nil, err
	}
	raw := w.GetDice()
	sides := make([]enum.DieSides, 0, len(raw))
	for _, d := range raw {
		sides = append(sides, enum.DieSides(d))
	}
	return sides, nil
}

// RawDamage is the weapon's rolled dice plus its flat damage bonus.
//
// The hit margin deliberately does NOT enter here. Product owner: "it will not add into the
// damage, at least not for now, because this system is already very punishing." It is the
// one place in the system where a margin does not circulate, and that is on purpose —
// revisiting it is a post-MVP playtest question, not a TODO.
func RawDamage(dice []int, name *enum.WeaponName, cat *item.WeaponsManager) (int, error) {
	w, err := lookup(name, cat)
	if err != nil {
		return 0, err
	}
	total := w.GetDamage()
	for _, d := range dice {
		total += d
	}
	return total, nil
}

// DefenseInput is everything the applicable-defense rules read.
type DefenseInput struct {
	AttackWeapon  *enum.WeaponName // nil = unarmed attack
	DefenseWeapon *enum.WeaponName // nil = bare-handed defense
	Defended      bool             // whether the target's defense actually succeeded
	Catalogue     *item.WeaponsManager
}

// DefenseOutcome is what a defense does to the raw damage.
type DefenseOutcome struct {
	Amount         int  // subtracted from the raw damage
	BlocksEntirely bool // nothing passes through at all
}

// ApplicableDefense encodes the table in spec §4.7.
//
// The subtraction is CONDITIONAL: defense only counts when the target actually managed to
// defend. It is not automatic damage reduction.
//
//	weapon against weapon   → nothing passes through
//	unarmed attack          → the defending weapon's defense bonus subtracts
//	armed attack, bare hands → the defense has no efficacy against piercing or cutting, and
//	                           works only against concussive
//
// This is the INITIAL form of parrying, not a temporary one. The mechanic exists from the
// start; what comes after the MVP is its complexity and detail — damage types (concussive,
// cutting, piercing, ultra-piercing), armour (which subtracts too, with the master deciding
// whether a blow lands on it at all), and Nen (which will reduce the final damage). None of
// those entities exist yet, so the armed-attack-versus-bare-hands row subtracts nothing
// rather than guessing which damage type a weapon deals. No rungs are invented here.
//
// Worth knowing while reading the numbers: every weapon in the catalogue today carries
// defense 0, so the subtraction is currently inert. The shape is what matters.
func ApplicableDefense(in DefenseInput) DefenseOutcome {
	if !in.Defended {
		return DefenseOutcome{}
	}
	attackArmed := in.AttackWeapon != nil
	defenseArmed := in.DefenseWeapon != nil

	if attackArmed && defenseArmed {
		return DefenseOutcome{BlocksEntirely: true}
	}
	if attackArmed && !defenseArmed {
		// No damage types yet: cannot tell concussive from cutting, so nothing is subtracted.
		return DefenseOutcome{}
	}
	// Unarmed attack: the only row that passes reduced damage through.
	w, err := lookup(in.DefenseWeapon, in.Catalogue)
	if err != nil {
		return DefenseOutcome{}
	}
	return DefenseOutcome{Amount: w.GetDefense()}
}

// EffectiveDamage applies a defense outcome to raw damage. Never negative.
func EffectiveDamage(raw int, d DefenseOutcome) int {
	if d.BlocksEntirely {
		return 0
	}
	if v := raw - d.Amount; v > 0 {
		return v
	}
	return 0
}

func lookup(name *enum.WeaponName, cat *item.WeaponsManager) (item.Weapon, error) {
	n := unarmed
	if name != nil {
		n = *name
	}
	return cat.Get(n)
}
```

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `go test ./internal/domain/match/service/ -run 'WeaponDice|RawDamage|ApplicableDefense|EffectiveDamage' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/damage.go internal/domain/match/service/damage_test.go
git commit -m "feat(match): add weapon damage and the applicable-defense rules

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5: `battle.Blow` gets a constructor and accessors

Six private fields, no constructor, nothing can build one. In Phase 2 the target's defense is
always the passive, bare-handed one, so there is no `action.Defense` to store — the field
becomes a pointer, nil meaning "the defense was passive".

`Blow` must not carry the numeric outcome: `service` imports `battle`, so `battle` importing
`service.RollOutcome` would be a cycle. The numbers live in `service.CharacterResult`
(Task 6), and `Blow` stays what its fields already say it is — the pairing of an attack with
the defense it met.

**Files:**
- Modify: `internal/domain/match/entity/battle/blow.go`
- Test: `internal/domain/match/entity/battle/blow_test.go`

**Interfaces:**
- Produces:
  `battle.NewBlow(actorID, targetID uuid.UUID, attack action.Attack, attackSkill *action.Skill, defense *action.Defense, defenseSkill *action.Skill) *Blow`
  and the accessors `GetActorID`, `GetTargetID`, `GetAttack`, `GetAttackSkill`,
  `GetDefense`, `GetDefenseSkill`.

- [ ] **Step 1: Write the failing test**

`internal/domain/match/entity/battle/blow_test.go`:

```go
package battle_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/google/uuid"
)

func TestNewBlow(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	sword := enum.Sword
	attack := action.Attack{
		Weapon: &sword,
		Hit:    action.RollCheck{SkillName: "Accuracy", SkillValue: 4},
		Damage: action.RollCheck{SkillName: "Strike"},
	}

	t.Run("carries the pairing it was built with", func(t *testing.T) {
		b := battle.NewBlow(actorID, targetID, attack, nil, nil, nil)

		if b.GetActorID() != actorID {
			t.Errorf("GetActorID() = %v, want %v", b.GetActorID(), actorID)
		}
		if b.GetTargetID() != targetID {
			t.Errorf("GetTargetID() = %v, want %v", b.GetTargetID(), targetID)
		}
		if b.GetAttack().Hit.SkillName != "Accuracy" {
			t.Errorf("GetAttack().Hit.SkillName = %q, want Accuracy", b.GetAttack().Hit.SkillName)
		}
	})

	t.Run("a passive defense leaves the defense nil", func(t *testing.T) {
		b := battle.NewBlow(actorID, targetID, attack, nil, nil, nil)
		if b.GetDefense() != nil {
			t.Error("expected a nil defense when the target defended passively")
		}
	})

	t.Run("an explicit defense is carried through", func(t *testing.T) {
		dagger := enum.Dagger
		def := &action.Defense{Weapon: &dagger}
		b := battle.NewBlow(actorID, targetID, attack, nil, def, nil)
		if b.GetDefense() == nil || b.GetDefense().Weapon == nil || *b.GetDefense().Weapon != enum.Dagger {
			t.Errorf("GetDefense() = %+v, want the dagger defense", b.GetDefense())
		}
	})
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/domain/match/entity/battle/ -v`
Expected: FAIL — `undefined: battle.NewBlow`.

- [ ] **Step 3: Rewrite `blow.go`**

```go
package battle

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

// Blow is one attack meeting one target: the pairing of the attack with the defense that
// answered it, and the skills each side put behind them.
//
// It carries no numbers on purpose. The rolled outcome of a collision lives in
// service.CharacterResult, because the domain service imports this package and the reverse
// would be an import cycle. Blow is the *shape* of the exchange; the service holds the
// arithmetic.
type Blow struct {
	actorID       uuid.UUID
	targetID      uuid.UUID
	attack        action.Attack
	attackSkill   *action.Skill
	defense       *action.Defense // nil when the target defended passively
	defenseSkill  *action.Skill
}

func NewBlow(
	actorID, targetID uuid.UUID,
	attack action.Attack,
	attackSkill *action.Skill,
	defense *action.Defense,
	defenseSkill *action.Skill,
) *Blow {
	return &Blow{
		actorID:      actorID,
		targetID:     targetID,
		attack:       attack,
		attackSkill:  attackSkill,
		defense:      defense,
		defenseSkill: defenseSkill,
	}
}

func (b *Blow) GetActorID() uuid.UUID          { return b.actorID }
func (b *Blow) GetTargetID() uuid.UUID         { return b.targetID }
func (b *Blow) GetAttack() action.Attack       { return b.attack }
func (b *Blow) GetAttackSkill() *action.Skill  { return b.attackSkill }
func (b *Blow) GetDefense() *action.Defense    { return b.defense }
func (b *Blow) GetDefenseSkill() *action.Skill { return b.defenseSkill }
```

Note the renames: `attackSkills`/`defenseSkills` become singular, and every
`//nolint:unused` marker goes away — the fields are read now.

- [ ] **Step 4: Run the test and confirm it passes**

Run: `go test ./internal/domain/match/entity/battle/ -v && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/entity/battle/
git commit -m "feat(match): give battle.Blow a constructor and accessors

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6: The `TargetKindCharacter` branch of `TurnResolver`

The heart of the phase. `Resolve` becomes a **pure function of the turn**: every die it needs
has already fallen and lives in the action's `RollCheck.Attempts` (Task 8 puts them there).
That purity is the whole point — Phase 4 re-resolves on every reaction and Phase 5 on every
master edit, and neither may re-roll.

Collision order, from `docs/game/combate/reacoes.md`:

1. **Hit** — the attacker's active test, derived from the dice that fell on arrival.
2. **Reflex dodge** — passive (`Reflex + PassiveValue`), free, automatic. Succeeds when
   `dodgeTotal >= hitTotal`.
3. **Defense** — only if the dodge failed. Passive (`Defense + PassiveValue`), bare-handed.
   Its CD is one ladder step lower than the attack: succeeds when
   `defenseTotal >= hitTotal - LadderStep`.
4. **Damage** — §4.7, using the dice already in `Attack.Damage.Attempts.Primary`.

**Files:**
- Modify: `internal/domain/match/service/turn_resolver.go`
- Modify: `internal/domain/match/service/turn_resolver_test.go` (signature only)
- Test: `internal/domain/match/service/character_collision_test.go`

**Interfaces:**
- Consumes: `ClimbLadder` (Task 3), `WeaponDice`/`RawDamage`/`ApplicableDefense`/`EffectiveDamage`
  (Task 4), `battle.NewBlow` (Task 5), `action.RollAttempts` (Task 1).
- Produces:
  `service.ResolveInput{Turn *turn.Turn; Sheets map[uuid.UUID]*csSheet.CharacterSheet; Targets TargetReader; Rules match.MatchRules; Weapons *item.WeaponsManager}`;
  `service.TurnResolver.Resolve(in ResolveInput) *TurnResolution`;
  `service.CharacterResult{TargetID, Hit, Dodge, Defense, Dodged, Defended, DamageDice, RawDamage, DefenseApplied, EffectiveDamage, Blow}`;
  `TurnResolution.CharacterResults []CharacterResult`;
  `RollResult` gains `IsCritical`, `IsCriticalFailure`, `Margin *int`.

- [ ] **Step 1: Write the failing test**

`internal/domain/match/service/character_collision_test.go`:

```go
package service_test

import (
	"testing"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// charTargets routes every id it knows to TargetKindCharacter.
type charTargets struct{ chars map[uuid.UUID]bool }

func (c charTargets) CategorizeTarget(id uuid.UUID) service.TargetKind {
	if c.chars[id] {
		return service.TargetKindCharacter
	}
	return service.TargetKindUnknown
}
func (c charTargets) GetWall(string) (mapentity.WallSegment, bool) {
	return mapentity.WallSegment{}, false
}

func plainSheet(t *testing.T) *csSheet.CharacterSheet {
	t.Helper()
	playerUUID := uuid.New()
	cs, err := csSheet.NewCharacterSheetFactory().Build(
		&playerUUID, nil, nil,
		csSheet.CharacterProfile{NickName: "Test", FullName: "Test Subject"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

// attackTurn builds a turn holding one attack from actorID against targetID, with the
// dice already fallen — exactly the state the session hands the resolver.
func attackTurn(actorID, targetID uuid.UUID, hitDice, damageDice []int, weapon *enum.WeaponName) *turn.Turn {
	atk := &action.Attack{
		Weapon: weapon,
		Hit: action.RollCheck{
			SkillName: enum.Accuracy.String(),
			Attempts:  action.RollAttempts{Primary: hitDice},
		},
		Damage: action.RollCheck{
			Attempts: action.RollAttempts{Primary: damageDice},
		},
	}
	a := action.NewAction(
		actorID, []uuid.UUID{targetID}, uuid.Nil,
		nil, action.ActionSpeed{},
		nil, nil, atk, nil, nil, nil, nil,
	)
	return turn.NewTurn(*a)
}

func resolveInput(t *testing.T, actorID, targetID uuid.UUID, tn *turn.Turn) service.ResolveInput {
	t.Helper()
	return service.ResolveInput{
		Turn: tn,
		Sheets: map[uuid.UUID]*csSheet.CharacterSheet{
			actorID:  plainSheet(t),
			targetID: plainSheet(t),
		},
		Targets: charTargets{chars: map[uuid.UUID]bool{targetID: true}},
		Rules:   match.NewDefaultMatchRules(),
		Weapons: item.NewWeaponsManagerFactory().Build(),
	}
}

func TestResolve_CharacterBranch(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	sword := enum.Sword // D10 + D4, flat damage 2

	t.Run("a weak hit is stopped by the passive reflex dodge", func(t *testing.T) {
		// A fresh sheet has every skill at 0, so the passive dodge is 0 + 11 = 11.
		// A hit of 3 + 2 = 5 never reaches it.
		tn := attackTurn(actorID, targetID, []int{3, 2}, []int{9, 3}, &sword)
		res := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, tn))

		if len(res.CharacterResults) != 1 {
			t.Fatalf("expected 1 character result, got %d", len(res.CharacterResults))
		}
		cr := res.CharacterResults[0]
		if !cr.Dodged {
			t.Errorf("expected the dodge to stop a hit of %d against a passive 11", cr.Hit.Total)
		}
		if cr.EffectiveDamage != 0 {
			t.Errorf("EffectiveDamage = %d, want 0 when the attack was dodged", cr.EffectiveDamage)
		}
	})

	t.Run("a hit past the dodge is defended, and an armed attack is not reduced", func(t *testing.T) {
		// Hit 10 + 8 = 18 beats the passive dodge of 11. The passive defense is also 11,
		// and its CD is 18 - 10 = 8, so it succeeds. Armed attack against a bare-handed
		// defense subtracts nothing while damage types do not exist, so the sword's
		// 9 + 3 + 2 = 14 passes whole.
		tn := attackTurn(actorID, targetID, []int{10, 8}, []int{9, 3}, &sword)
		res := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, tn))

		cr := res.CharacterResults[0]
		if cr.Dodged {
			t.Fatalf("a hit of %d should have beaten the passive dodge", cr.Hit.Total)
		}
		if !cr.Defended {
			t.Error("the passive defense should succeed at a CD one ladder step lower")
		}
		if cr.RawDamage != 14 {
			t.Errorf("RawDamage = %d, want 14 (9 + 3 dice + 2 flat)", cr.RawDamage)
		}
		if cr.EffectiveDamage != 14 {
			t.Errorf("EffectiveDamage = %d, want 14", cr.EffectiveDamage)
		}
	})

	t.Run("the individual dice and the critical flags survive", func(t *testing.T) {
		tn := attackTurn(actorID, targetID, []int{10, 10}, []int{1, 1}, &sword)
		res := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, tn))

		if !res.ActionResult.IsCritical {
			t.Error("two tens is a critical and the flag must reach the resolution")
		}
		if len(res.ActionResult.DiceRolled) != 2 {
			t.Errorf("DiceRolled = %v, want the two individual dice", res.ActionResult.DiceRolled)
		}
		if res.ActionResult.Margin == nil {
			t.Fatal("the margin must be derived once a CD exists")
		}
		// CD is the passive dodge: 0 + 11.
		if *res.ActionResult.Margin != 20-11 {
			t.Errorf("Margin = %d, want %d", *res.ActionResult.Margin, 20-11)
		}
	})

	t.Run("a critical does not change the damage", func(t *testing.T) {
		// The flag passes through untouched — no multiplier exists, and Phase 2 must not
		// invent one.
		crit := attackTurn(actorID, targetID, []int{10, 10}, []int{9, 3}, &sword)
		plain := attackTurn(actorID, targetID, []int{10, 8}, []int{9, 3}, &sword)

		critRes := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, crit))
		plainRes := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, plain))

		if critRes.CharacterResults[0].EffectiveDamage != plainRes.CharacterResults[0].EffectiveDamage {
			t.Error("a critical must not change the damage while no rule consumes it")
		}
	})

	t.Run("a Blow is produced for every character target", func(t *testing.T) {
		tn := attackTurn(actorID, targetID, []int{10, 8}, []int{9, 3}, &sword)
		res := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, tn))

		if len(res.Blows) != 1 {
			t.Fatalf("expected 1 blow, got %d", len(res.Blows))
		}
		if res.Blows[0].GetTargetID() != targetID {
			t.Errorf("blow target = %v, want %v", res.Blows[0].GetTargetID(), targetID)
		}
	})

	t.Run("resolving twice yields the same numbers", func(t *testing.T) {
		// The purity that lets Phase 4 re-resolve on every reaction without re-rolling.
		tn := attackTurn(actorID, targetID, []int{10, 8}, []int{9, 3}, &sword)
		in := resolveInput(t, actorID, targetID, tn)

		first := service.TurnResolver{}.Resolve(in)
		second := service.TurnResolver{}.Resolve(in)

		if first.CharacterResults[0].EffectiveDamage != second.CharacterResults[0].EffectiveDamage {
			t.Error("Resolve must be pure: same turn in, same numbers out")
		}
	})

	t.Run("an action with no attack produces no character result", func(t *testing.T) {
		a := action.NewAction(actorID, []uuid.UUID{targetID}, uuid.Nil,
			nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		tn := turn.NewTurn(*a)
		res := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, tn))

		if len(res.CharacterResults) != 0 {
			t.Errorf("expected no character results, got %d", len(res.CharacterResults))
		}
	})
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/domain/match/service/ -run Resolve_CharacterBranch -v`
Expected: FAIL — `undefined: service.ResolveInput`.

- [ ] **Step 3: Reshape `Resolve`'s inputs and outputs**

In `internal/domain/match/service/turn_resolver.go`, add the imports
`csSheet`, `enum`, `item`, `match`, `action` as needed, then replace `TurnResolution`,
`RollResult` and the `Resolve` signature:

```go
// ResolveInput is everything a resolution reads. It is a struct rather than a parameter
// list because the list has already grown twice and will grow again — the round rules and
// the weapon catalogue arrived in Phase 2, and reactions arrive in Phase 4.
type ResolveInput struct {
	Turn *turn.Turn
	// Sheets is keyed by sheet UUID — the same ID the board pieces carry as CharacterID,
	// and the same ID Action.actorID and Action.TargetID carry since Phase 2.
	Sheets  map[uuid.UUID]*csSheet.CharacterSheet
	Targets TargetReader // nil disables wall routing
	Rules   match.MatchRules
	Weapons *item.WeaponsManager
}

// TurnResolution is the snapshot of a Turn's result — character combat, wall
// interactions, or any mix thereof.
//
// It is a DRY RUN. Nothing here has been applied to a sheet. The master sees the projected
// HP reduction from the first resolution onward, and the real application happens once, at
// turn close. That is what lets the collision be recomputed on every reaction without
// applying damage several times.
type TurnResolution struct {
	ActionResult     RollResult
	ReactionResults  []ReactionResult
	CharacterResults []CharacterResult
	Blows            []*battle.Blow
	WallResults      []WallResult
	IsSettled        bool
}

// RollResult holds the outcome of a single dice roll check.
type RollResult struct {
	SkillName  string
	SkillValue int
	DiceRolled []int
	Total      int
	// The critical flags travel untouched. No rule consumes them yet — the design notes
	// point at a narrative consequence rather than a multiplier — and the resolver must not
	// invent one.
	IsCritical        bool
	IsCriticalFailure bool
	// Margin is nil until an opposed roll gives this test a CD.
	Margin *int
}

// CharacterResult is the computed outcome of one attack against one character.
type CharacterResult struct {
	TargetID uuid.UUID
	Hit      RollOutcome
	Dodge    RollOutcome
	// Defense is the zero value when the dodge already stopped the attack.
	Defense  RollOutcome
	Dodged   bool
	Defended bool

	DamageDice      []int
	RawDamage       int
	DefenseApplied  int
	EffectiveDamage int

	Blow *battle.Blow
}
```

- [ ] **Step 4: Write the resolution body**

Replace the `Resolve` method:

```go
// Resolve calculates the current resolution snapshot for the given Turn.
//
// It is PURE: every die it needs has already fallen and lives in the action's RollCheck
// Attempts, put there by MatchSession the moment the action arrived. Resolve derives, it
// never rolls. That is what makes it safe to call again on every reaction (Phase 4) and on
// every master edit (Phase 5) — the master never re-rolls a player's die.
func (tr TurnResolver) Resolve(in ResolveInput) *TurnResolution {
	res := &TurnResolution{}
	if in.Turn == nil {
		return res
	}
	res.IsSettled = in.Turn.GetFinishedAt() != nil
	a := in.Turn.GetAction()

	if in.Targets != nil {
		for _, targetID := range a.TargetID {
			switch in.Targets.CategorizeTarget(targetID) {
			case TargetKindCharacter:
				cr, ok := tr.resolveCharacter(in, a, targetID)
				if !ok {
					continue
				}
				res.CharacterResults = append(res.CharacterResults, cr)
				res.Blows = append(res.Blows, cr.Blow)
				res.ActionResult = rollResultOf(cr.Hit, &cr.Dodge.Total)

			case TargetKindWallSegment:
				wall, found := in.Targets.GetWall(targetID.String())
				if !found {
					continue
				}
				if a.Attack != nil {
					raw, err := RawDamage(a.Attack.Damage.Attempts.Primary, a.Attack.Weapon, in.Weapons)
					if err != nil {
						raw = 0
					}
					sdr := ApplyStructuralDamage(wall, raw)
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

	reactions := in.Turn.GetReactions()
	res.ReactionResults = make([]ReactionResult, len(reactions))
	for i, r := range reactions {
		// TODO: implement per-reaction resolution — Phase 4
		res.ReactionResults[i] = ReactionResult{ReactorID: r.ReactToID}
	}
	return res
}

// resolveCharacter runs one attack against one character: hit, then the two passive
// reactions in the order the rules give them, then damage.
//
// The passives are free and automatic. The reflex dodge takes the dice set's average
// instead of rolling — rolling has exactly zero expected gain, so the player only gambles
// when they need luck above the average — and the defense is one ladder step easier than
// the attack, because it should be easier to parry than to land a blow.
func (tr TurnResolver) resolveCharacter(
	in ResolveInput, a action.Action, targetID uuid.UUID,
) (CharacterResult, bool) {
	if a.Attack == nil {
		return CharacterResult{}, false
	}
	actorSheet, okActor := in.Sheets[a.GetActorID()]
	targetSheet, okTarget := in.Sheets[targetID]
	if !okActor || !okTarget || actorSheet == nil || targetSheet == nil {
		// TODO: surface a missing-sheet error in the resolution — Phase 5, with the rest of
		// the error reporting the caller needs.
		return CharacterResult{}, false
	}

	cr := CharacterResult{TargetID: targetID}

	// 1. The hit — the attacker's active test.
	//    Ledger is deliberately nil: the accumulated difference a character carries is
	//    always an actionSpeed adjustment, never a hit adjustment.
	cr.Hit = tr.Calc().Derive(in.Rules, a.Attack.Hit.Attempts, RollInput{
		SkillName:  a.Attack.Hit.SkillName,
		SkillValue: skillValueOf(actorSheet, a.Attack.Hit.SkillName),
		Condition:  a.Attack.Hit.Context.Condition,
		AgainstID:  &targetID,
	})

	// 2. Reflex dodge — passive, free, automatic. Ties favour the defender.
	cr.Dodge = tr.Calc().Derive(in.Rules, action.RollAttempts{}, RollInput{
		SkillName:  enum.Reflex.String(),
		SkillValue: skillValueOf(targetSheet, enum.Reflex.String()),
		Passive:    true,
	})
	cr.Dodged = cr.Dodge.Total >= cr.Hit.Total

	// 3. Defense — only if the dodge failed, at a CD one ladder step lower.
	if !cr.Dodged {
		cr.Defense = tr.Calc().Derive(in.Rules, action.RollAttempts{}, RollInput{
			SkillName:  enum.Defense.String(),
			SkillValue: skillValueOf(targetSheet, enum.Defense.String()),
			Passive:    true,
		})
		cr.Defended = cr.Defense.Total >= cr.Hit.Total-in.Rules.LadderStep
	}

	// 4. Damage — the weapon's own dice, already rolled on arrival.
	if !cr.Dodged {
		cr.DamageDice = append([]int(nil), a.Attack.Damage.Attempts.Primary...)
		raw, err := RawDamage(cr.DamageDice, a.Attack.Weapon, in.Weapons)
		if err == nil {
			cr.RawDamage = raw
			def := ApplicableDefense(DefenseInput{
				AttackWeapon:  a.Attack.Weapon,
				DefenseWeapon: nil, // Phase 2: the passive defense is always bare-handed
				Defended:      cr.Defended,
				Catalogue:     in.Weapons,
			})
			cr.DefenseApplied = def.Amount
			cr.EffectiveDamage = EffectiveDamage(raw, def)
		}
	}

	cr.Blow = battle.NewBlow(a.GetActorID(), targetID, *a.Attack, nil, nil, nil)
	return cr, true
}

// Calc returns the calculator this resolver derives with. A method rather than a field so
// TurnResolver stays usable as a stateless zero value.
func (tr TurnResolver) Calc() RollCalculator { return RollCalculator{} }

// skillValueOf reads a skill off the sheet, crossing the string→enum boundary. A name the
// sheet does not know contributes 0 rather than failing the whole resolution: the mapper at
// the WS boundary already rejects unknown skill names, so reaching here means an internal
// name, and a resolution that silently drops one test is better than one that drops the turn.
func skillValueOf(cs *csSheet.CharacterSheet, name string) int {
	skillName, err := enum.SkillNameFrom(name)
	if err != nil {
		return 0
	}
	v, err := cs.GetValueForTestOfSkill(skillName)
	if err != nil {
		return 0
	}
	return v
}

// rollResultOf flattens an outcome into the wire-facing shape, deriving the margin now that
// a CD exists.
func rollResultOf(o RollOutcome, cd *int) RollResult {
	r := RollResult{
		SkillName:         o.SkillName,
		SkillValue:        o.SkillValue,
		DiceRolled:        append([]int(nil), o.Dice...),
		Total:             o.Total,
		IsCritical:        o.IsCritical,
		IsCriticalFailure: o.IsCriticalFailure,
	}
	if cd != nil {
		m := o.Margin(*cd)
		r.Margin = &m
	}
	return r
}
```

Note the wall branch now rolls real damage instead of the `rawDamage := 0` literal —
`05-lacunas.md` §3 lists that as a gap and the dice are available here for free.

- [ ] **Step 5: Update `turn_resolver_test.go` to the new signature**

Every `TurnResolver{}.Resolve(t, sheets, targets)` becomes
`TurnResolver{}.Resolve(service.ResolveInput{Turn: t, Sheets: sheets, Targets: targets, Rules: match.NewDefaultMatchRules(), Weapons: item.NewWeaponsManagerFactory().Build()})`.
Keep every existing assertion — the wall tests must keep passing unchanged except where they
asserted `EffectiveDamage: 0` from the literal; those now receive real damage, so give those
turns an `Attack.Damage.Attempts` with known dice and assert the computed number.

- [ ] **Step 6: Run the tests**

Run: `go test ./internal/domain/match/... -v && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/match/service/
git commit -m "feat(match): resolve an attack against a character end to end

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 7: The actor axis becomes the sheet UUID

`buildAction` puts the authenticated **player** UUID into `Action.actorID`, and
`EnqueueAction` enforces it. But `TargetID` carries **sheet** UUIDs, and since Phase 1 so do
`charSheets` and `statuses`. The resolver can index the target and not the actor.
`05-lacunas.md` §3 assigns the reconciliation to this phase.

Authorization stays per player: the session reads `charToPlayer` to check that the player
sending the action actually owns the character acting.

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go` (`EnqueueAction`)
- Modify: `internal/domain/match/matchsession/match_session_test.go`
- Modify: `internal/application/match/*_test.go` (helpers)
- Modify: `internal/app/game/message.go` (`ActionPayload`)
- Modify: `internal/app/game/action_mapper.go`
- Modify: `internal/app/game/room.go`

**Interfaces:**
- Produces: `ActionPayload.ActorID uuid.UUID` (`json:"actorId"`);
  `buildAction(actorCharID uuid.UUID, p ActionPayload) (*action.Action, error)`;
  `MatchSession.EnqueueAction` authorizing through `charToPlayer`.

- [ ] **Step 1: Write the failing test**

Replace `TestMatchSession_EnqueueAction` in
`internal/domain/match/matchsession/match_session_test.go`:

```go
func TestMatchSession_EnqueueAction(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	t.Run("enqueues an action whose actor is the player's character", func(t *testing.T) {
		// The combat entity is the character: actorID is the sheet UUID, the same ID the
		// board piece carries and the same ID a TargetID carries.
		a := makeAction(charID)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns ErrParticipantNotFound for an unknown player", func(t *testing.T) {
		a := makeAction(charID)
		err := s.EnqueueAction(uuid.New(), a)
		if !errors.Is(err, matchsession.ErrParticipantNotFound) {
			t.Errorf("expected ErrParticipantNotFound, got %v", err)
		}
	})

	t.Run("a player cannot act through a character they do not own", func(t *testing.T) {
		a := makeAction(uuid.New()) // some other character
		err := s.EnqueueAction(playerUUID, a)
		if !errors.Is(err, matchsession.ErrActionActorMismatch) {
			t.Errorf("expected ErrActionActorMismatch, got %v", err)
		}
	})

	t.Run("the player UUID is no longer a valid actor", func(t *testing.T) {
		// It used to be exactly this, and it is what left the resolver unable to find the
		// actor's sheet.
		a := makeAction(playerUUID)
		err := s.EnqueueAction(playerUUID, a)
		if !errors.Is(err, matchsession.ErrActionActorMismatch) {
			t.Errorf("expected ErrActionActorMismatch, got %v", err)
		}
	})
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/domain/match/matchsession/ -run EnqueueAction -v`
Expected: FAIL — the first sub-test errors with `ErrActionActorMismatch`.

- [ ] **Step 3: Re-point the authorization**

In `internal/domain/match/matchsession/match_session.go`:

```go
// EnqueueAction validates that playerUUID may act, that the character they are acting
// through is theirs, and drops the action's dice before it enters the queue.
//
// Two axes, deliberately: authorization is a per-PLAYER question, and combat is a
// per-CHARACTER one. charToPlayer is the bridge. a.actorID is the sheet UUID, so the
// resolver can index the actor's sheet in the same map it indexes the target's.
func (s *MatchSession) EnqueueAction(playerUUID uuid.UUID, a *action.Action) error {
	if _, ok := s.participants[playerUUID]; !ok {
		return ErrParticipantNotFound
	}
	owner, ok := s.charToPlayer[a.GetActorID().String()]
	if !ok || owner != playerUUID {
		return ErrActionActorMismatch
	}
	s.activeQueue.Insert(a)
	return nil
}
```

(The dice drop is added in Task 8; leave the method as above for now.)

- [ ] **Step 4: Update the other test helpers**

In `match_session_test.go`, `sessionWithParticipants` must hand back the sheet UUIDs:

```go
// sessionWithParticipants builds a session and returns it together with the sheet UUID of
// each player's character, in the order the players were given. The tests need the sheet
// UUIDs because that is what an action's actorID carries.
func sessionWithParticipants(playerUUIDs ...uuid.UUID) (*matchsession.MatchSession, []uuid.UUID) {
	matchUUID := uuid.New()
	participants := make([]*match.Participant, len(playerUUIDs))
	charIDs := make([]uuid.UUID, len(playerUUIDs))
	for i, id := range playerUUIDs {
		pID := id
		participants[i] = makeParticipant(matchUUID, &pID)
		charIDs[i] = participants[i].Sheet.UUID
	}
	return matchsession.NewMatchSession(matchUUID, nil, participants), charIDs
}
```

Then every call site changes shape, e.g. in `TestMatchSession_OpenNextAction`:

```go
		s, chars := sessionWithParticipants(playerA, playerB)

		aHigh := makeActionWithSpeed(chars[0], 10)
		aLow := makeActionWithSpeed(chars[1], 3)
		s.EnqueueAction(playerA, aHigh) //nolint:errcheck
		s.EnqueueAction(playerB, aLow)  //nolint:errcheck
```

Apply the same mechanical change to every `sessionWithParticipants` caller in the file
(`OpenNextAction`, `AttachReaction`, `CloseTurn`, `CloseRound`, `PullAction`,
`PersistenceFlags`): take the second return value, and pass `chars[i]` wherever a player UUID
was being passed as an actor.

Do the same in `internal/application/match/`: change `sessionWithPlayers` in
`attach_reaction_test.go` to return `(*matchsession.MatchSession, []uuid.UUID)` and update
`attach_reaction_test.go`, `open_next_action_test.go`, `pull_action_test.go` and
`close_round_test.go`. In `enqueue_action_test.go`, the inline participant already exposes
`p.Sheet.UUID` — pass that as the actor.

- [ ] **Step 5: Run the domain and application tests**

Run: `go test ./internal/domain/match/... ./internal/application/match/...`
Expected: PASS.

- [ ] **Step 6: Put `actorId` on the wire**

`internal/app/game/message.go` — add the field as the first member of `ActionPayload`:

```go
type ActionPayload struct {
	// ActorID is the acting character's sheet UUID — the same ID the board piece carries
	// as CharacterID. It is NOT the player UUID: one person drives several characters (the
	// master drives every NPC), so the actor has to be named explicitly. The server still
	// checks that the authenticated player owns that character.
	ActorID   uuid.UUID            `json:"actorId"`
	ReactToID uuid.UUID            `json:"reactToId,omitempty"`
	TargetID  []uuid.UUID          `json:"targetId,omitempty"`
	// ... rest unchanged
}
```

- [ ] **Step 7: Require it at the boundary**

In `internal/app/game/room.go`, in both `case MsgTypeEnqueueAction` and
`case MsgTypeAttachReaction`, right after the unmarshal and the existing validations:

```go
		if payload.ActorID == uuid.Nil {
			client.SendMessage(NewErrorMessage("invalid_action", "actorId is required: the acting character's sheet UUID"))
			return
		}
```

and change the two construction sites from `buildAction(client.userUUID, payload)` to
`buildAction(payload.ActorID, payload)` — including the one inside `handleReaction`, whose
signature keeps `client` for the error path but must use the payload's actor.

- [ ] **Step 8: Run everything and commit**

Run: `go build ./... && go test ./internal/... && go vet -tags=integration ./internal/...`
Expected: PASS.

```bash
git add internal/domain/match/matchsession/ internal/application/match/ internal/app/game/
git commit -m "refactor(match): the actor of an action is the character, not the player

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 8: The session rolls the dice once, on arrival

The dice must fall when the action arrives, not when it resolves — otherwise every reaction
and every master edit would re-roll, and the master would be re-rolling a player's die. The
session is the only place holding both the match rules and the weapon catalogue.

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Modify: `internal/domain/match/matchsession/match_session_test.go`

**Interfaces:**
- Consumes: `RollCalculator.Roll`/`RollDice` (Task 2), `WeaponDice` (Task 4),
  `action.RollAttempts` (Task 1).
- Produces: `MatchSession.GetRules() match.MatchRules`;
  `MatchSession.SetRollSource(src service.RollSource)`;
  dice populated in `RollCheck.Attempts` for `Speed`, `Skills`, `Feint`, `Attack.Hit`,
  `Attack.Damage`, `Attack.Charge`, `Defense`, `Dodge`, `Move.Speed`, `Move.Charge`.

- [ ] **Step 1: Write the failing test**

Append to `internal/domain/match/matchsession/match_session_test.go`:

```go
// fixedSource makes every die land on the same face, so a test can name exact numbers.
type fixedSource struct{ face int }

func (f fixedSource) RollDie(_ enum.DieSides) int { return f.face }

func TestMatchSession_GetRules(t *testing.T) {
	s := matchsession.NewMatchSession(uuid.New(), nil, nil)
	rules := s.GetRules()

	if rules.DiceSet != match.DiceSet2D10 {
		t.Errorf("DiceSet = %q, want 2d10", rules.DiceSet)
	}
	if rules.LadderStep != 10 {
		t.Errorf("LadderStep = %d, want 10", rules.LadderStep)
	}
	if rules.PassiveValue() != 11 {
		t.Errorf("PassiveValue() = %d, want 11", rules.PassiveValue())
	}
}

func TestMatchSession_EnqueueAction_RollsTheDiceOnce(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})
	s.SetRollSource(fixedSource{face: 7})

	sword := enum.Sword
	atk := &action.Attack{
		Weapon: &sword,
		Hit:    action.RollCheck{SkillName: enum.Accuracy.String()},
	}
	a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, atk, nil, nil, nil, nil)

	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("the hit dice fell, from the match dice set", func(t *testing.T) {
		if len(a.Attack.Hit.Attempts.Primary) != 2 {
			t.Fatalf("hit Primary = %v, want 2 dice", a.Attack.Hit.Attempts.Primary)
		}
		if a.Attack.Hit.Attempts.Primary[0] != 7 {
			t.Errorf("hit die = %d, want 7 from the scripted source", a.Attack.Hit.Attempts.Primary[0])
		}
		if len(a.Attack.Hit.Attempts.Secondary) != 2 {
			t.Error("both sets must fall up front, so a later advantage never re-rolls")
		}
	})

	t.Run("the damage dice fell, from the weapon's own set", func(t *testing.T) {
		// A Sword is D10 + D4 — two dice, not the match's 2 D10 by coincidence but by the
		// weapon's own definition.
		if len(a.Attack.Damage.Attempts.Primary) != 2 {
			t.Errorf("damage Primary = %v, want the sword's 2 dice", a.Attack.Damage.Attempts.Primary)
		}
		if len(a.Attack.Damage.Attempts.Secondary) != 0 {
			t.Error("damage has no advantage, so there is no second set")
		}
	})

	t.Run("the action speed dice fell", func(t *testing.T) {
		if a.Speed.Attempts.IsEmpty() {
			t.Error("expected the actionSpeed dice to fall on arrival too")
		}
	})
}

func TestMatchSession_EnqueueAction_NeverRerolls(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})
	s.SetRollSource(fixedSource{face: 3})

	a := action.NewAction(participant.Sheet.UUID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{Hit: action.RollCheck{}}, nil, nil, nil, nil)
	// Dice that already fell — the master edited, the action came back around, whatever.
	a.Attack.Hit.Attempts = action.RollAttempts{Primary: []int{10, 10}, Secondary: []int{1, 1}}

	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Attack.Hit.Attempts.Primary[0] != 10 {
		t.Error("dice that already fell must never be rolled again")
	}
}
```

Add the imports `enum` and `action` to the test file if they are not already there.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/domain/match/matchsession/ -run 'GetRules|RollsTheDice|NeverRerolls' -v`
Expected: FAIL — `undefined: SetRollSource`, `GetRules`.

- [ ] **Step 3: Give the session its rules, catalogue and source**

In `internal/domain/match/matchsession/match_session.go`, add to the struct:

```go
	// rules is the per-match configuration. Phase 2 uses the embedded defaults; the REST
	// surface for the master to choose them is a slice of its own.
	rules match.MatchRules
	// weapons is the static weapon catalogue, the source of the damage dice.
	weapons *item.WeaponsManager
	// rollSource is where the dice come from. nil means production. Tests set it so a
	// phase whose done-criteria name exact numbers never depends on luck.
	rollSource service.RollSource
```

and in **both** constructors:

```go
		rules:   match.NewDefaultMatchRules(),
		weapons: item.NewWeaponsManagerFactory().Build(),
```

plus the accessors:

```go
func (s *MatchSession) GetRules() match.MatchRules { return s.rules }

// SetRollSource replaces the dice source. Production never calls it; tests do.
func (s *MatchSession) SetRollSource(src service.RollSource) { s.rollSource = src }
```

Add the import `"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"`.

- [ ] **Step 4: Drop the dice on arrival**

Add to `match_session.go`:

```go
// rollActionDice drops the dice for every test the action carries, once, the moment it
// arrives. Derive is then free to run again on every master edit and on every colliding
// reaction without a single new die — "the master never re-rolls a player's die".
//
// A RollCheck whose dice already fell is left alone, so calling this twice is harmless.
func (s *MatchSession) rollActionDice(a *action.Action) {
	if a == nil {
		return
	}
	calc := service.RollCalculator{}

	// test is the match's dice set: hit, skill, actionSpeed, feint, defense, dodge.
	test := func(rc *action.RollCheck) {
		if rc == nil || !rc.Attempts.IsEmpty() {
			return
		}
		rc.Attempts = calc.Roll(s.rules, s.rollSource)
	}

	test(&a.Speed.RollCheck)
	test(a.Feint)
	for i := range a.Skills {
		test(&a.Skills[i].RollCheck)
	}
	if a.Move != nil {
		test(a.Move.Speed)
		test(a.Move.Charge)
	}
	if a.Defense != nil {
		test(&a.Defense.RollCheck)
	}
	if a.Dodge != nil {
		test(&a.Dodge.RollCheck)
	}
	if a.Attack != nil {
		test(&a.Attack.Hit)
		test(a.Attack.Charge)
		// Damage is the OTHER family of roll: the weapon's own dice, not the match set.
		// Only Primary, because damage has no advantage.
		if a.Attack.Damage.Attempts.IsEmpty() {
			if sides, err := service.WeaponDice(a.Attack.Weapon, s.weapons); err == nil {
				a.Attack.Damage.Attempts = action.RollAttempts{
					Primary: calc.RollDice(sides, s.rollSource),
				}
			}
		}
	}
}
```

Then call it from `EnqueueAction`, after the two authorization checks and before
`s.activeQueue.Insert(a)`:

```go
	s.rollActionDice(a)
	s.activeQueue.Insert(a)
```

and from `AttachReaction`, before handing the reaction to the orchestrator:

```go
func (s *MatchSession) AttachReaction(r *action.Action) (*service.TurnResolution, error) {
	s.rollActionDice(r)
	if err := s.roundOrch.AttachReaction(s.activeRound, r); err != nil {
		return nil, err
	}
	...
}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/domain/match/... && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/matchsession/
git commit -m "feat(match): drop an action's dice once, the moment it arrives

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 9: The session resolves and applies the damage at turn close

Two things at once, because they are the same seam: `OpenNextAction`/`PullAction` stop
resolving with `nil` sheets, and the turn they close is where the projected damage stops being
a projection.

`05-lacunas.md` §9 names the first half; spec §4.7 "Dry-run" names the second.

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Modify: `internal/domain/match/matchsession/error.go`
- Modify: `internal/domain/match/matchsession/match_session_test.go`

**Interfaces:**
- Consumes: `service.ResolveInput`, `service.TurnResolution`, `service.CharacterResult` (Task 6).
- Produces:
  `matchsession.TurnTransition{Closed, Opened *turn.Turn; ClosedResolution, OpenedResolution *service.TurnResolution; Damaged []DamagedCharacter}`;
  `matchsession.DamagedCharacter{CharacterID uuid.UUID; Sheet *csSheet.CharacterSheet; Damage, NewHP int}`;
  `MatchSession.OpenNextAction() (*TurnTransition, error)`;
  `MatchSession.PullAction(id uuid.UUID) (*TurnTransition, error)`;
  `MatchSession.ResolveTurn(t *turn.Turn) *service.TurnResolution`.

- [ ] **Step 1: Write the failing test**

Append to `internal/domain/match/matchsession/match_session_test.go`:

```go
func TestMatchSession_DamageIsAppliedOnlyOnTurnClose(t *testing.T) {
	matchUUID := uuid.New()
	playerA, playerB := uuid.New(), uuid.New()
	pA := makeParticipant(matchUUID, &playerA)
	pB := makeParticipant(matchUUID, &playerB)
	attacker, victim := pA.Sheet.UUID, pB.Sheet.UUID

	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		attacker: buildPlainSheet(t),
		victim:   buildPlainSheet(t),
	}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{pA, pB})
	// Every die lands on its top face, so the hit clears the passive dodge of 11 for sure.
	s.SetRollSource(fixedTopFaceSource{})

	sword := enum.Sword
	atk := &action.Attack{Weapon: &sword, Hit: action.RollCheck{SkillName: enum.Accuracy.String()}}
	a := action.NewAction(attacker, []uuid.UUID{victim}, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: 10}},
		nil, nil, atk, nil, nil, nil, nil)

	if err := s.EnqueueAction(playerA, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	hpBefore := currentHP(t, sheets[victim])

	// Opening the turn resolves it as a dry run.
	tr, err := s.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}

	t.Run("the master sees the projection before anything is applied", func(t *testing.T) {
		if tr.OpenedResolution == nil || len(tr.OpenedResolution.CharacterResults) != 1 {
			t.Fatalf("expected one character result, got %+v", tr.OpenedResolution)
		}
		if tr.OpenedResolution.CharacterResults[0].EffectiveDamage <= 0 {
			t.Fatal("expected projected damage from a maximum-roll attack")
		}
		if got := currentHP(t, sheets[victim]); got != hpBefore {
			t.Errorf("HP = %d, want %d — a projection must not touch the sheet", got, hpBefore)
		}
	})

	projected := tr.OpenedResolution.CharacterResults[0].EffectiveDamage

	t.Run("closing the turn applies it exactly once", func(t *testing.T) {
		// Enqueue a second action so there is something to open, which closes the first.
		a2 := action.NewAction(attacker, nil, uuid.Nil, nil,
			action.ActionSpeed{RollCheck: action.RollCheck{Result: 1}},
			nil, nil, nil, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerA, a2); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
		tr2, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("OpenNextAction: %v", err)
		}
		if tr2.Closed == nil {
			t.Fatal("expected the first turn to close")
		}
		if len(tr2.Damaged) != 1 {
			t.Fatalf("expected 1 damaged character, got %d", len(tr2.Damaged))
		}
		if tr2.Damaged[0].CharacterID != victim {
			t.Errorf("damaged = %v, want %v", tr2.Damaged[0].CharacterID, victim)
		}
		if tr2.Damaged[0].Damage != projected {
			t.Errorf("applied %d, projected %d — they must agree", tr2.Damaged[0].Damage, projected)
		}
		if got := currentHP(t, sheets[victim]); got != hpBefore-projected {
			t.Errorf("HP = %d, want %d", got, hpBefore-projected)
		}
	})
}

// fixedTopFaceSource lands every die on its top face.
type fixedTopFaceSource struct{}

func (fixedTopFaceSource) RollDie(sides enum.DieSides) int { return sides.GetSides() }

func buildPlainSheet(t *testing.T) *csSheet.CharacterSheet {
	t.Helper()
	playerUUID := uuid.New()
	cs, err := csSheet.NewCharacterSheetFactory().Build(
		&playerUUID, nil, nil,
		csSheet.CharacterProfile{NickName: "Test", FullName: "Test Subject"},
		nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

func currentHP(t *testing.T, cs *csSheet.CharacterSheet) int {
	t.Helper()
	bar, ok := cs.GetAllStatusBar()[enum.Health]
	if !ok {
		t.Fatal("the sheet has no health bar")
	}
	return bar.GetCurrent()
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/domain/match/matchsession/ -run DamageIsApplied -v`
Expected: FAIL — `OpenNextAction` returns three values, not two.

- [ ] **Step 3: Add the transition types and the errors**

In `internal/domain/match/matchsession/match_session.go`:

```go
// TurnTransition is what one act of the master's baton produces: the turn that closed, the
// turn that opened, the resolution of each, and whatever the closing actually applied.
//
// It is a struct rather than five return values because the two operations that produce it
// — open the next action, pull one out of order — are the same shape, and Phase 4 adds
// opening a reaction to the list.
type TurnTransition struct {
	Closed           *turn.Turn
	Opened           *turn.Turn
	ClosedResolution *service.TurnResolution
	OpenedResolution *service.TurnResolution
	// Damaged is what the close actually wrote to a sheet. Empty on the first transition of
	// a round, when nothing closed.
	Damaged []DamagedCharacter
}

// DamagedCharacter is one applied HP reduction. The caller persists it — the session holds
// the live sheet, the gateway holds the row.
type DamagedCharacter struct {
	CharacterID uuid.UUID
	Sheet       *csSheet.CharacterSheet
	Damage      int
	NewHP       int
}
```

- [ ] **Step 4: Resolve and apply**

Add to `match_session.go`:

```go
// ResolveTurn computes the resolution snapshot for t. Pure — it never touches a sheet.
// The master reads it as a projection, over and over, as reactions land.
func (s *MatchSession) ResolveTurn(t *turn.Turn) *service.TurnResolution {
	if t == nil {
		return nil
	}
	return s.turnResolver.Resolve(service.ResolveInput{
		Turn:    t,
		Sheets:  s.charSheets,
		Targets: s,
		Rules:   s.rules,
		Weapons: s.weapons,
	})
}

// applyResolution writes a resolution's effective damage to the target sheets, once.
//
// This is the moment the dry run stops being a dry run. Everything before it recalculated
// freely — every master edit, every colliding reaction — precisely because nothing had been
// applied. Called only from the turn-closing path.
func (s *MatchSession) applyResolution(res *service.TurnResolution) []DamagedCharacter {
	if res == nil {
		return nil
	}
	var out []DamagedCharacter
	for _, cr := range res.CharacterResults {
		if cr.EffectiveDamage <= 0 {
			continue
		}
		sheet, ok := s.charSheets[cr.TargetID]
		if !ok || sheet == nil {
			continue
		}
		bar, ok := sheet.GetAllStatusBar()[enum.Health]
		if !ok {
			continue
		}
		newHP := bar.DecreaseAt(cr.EffectiveDamage)
		out = append(out, DamagedCharacter{
			CharacterID: cr.TargetID,
			Sheet:       sheet,
			Damage:      cr.EffectiveDamage,
			NewHP:       newHP,
		})
	}
	return out
}
```

Add the import `csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"`
if it is not already present (it is — the struct already holds `charSheets`).

- [ ] **Step 5: Rewrite the two transitions**

Replace `OpenNextAction` and `PullAction`:

```go
func (s *MatchSession) OpenNextAction() (*TurnTransition, error) {
	tr := &TurnTransition{}
	if s.activeRound.HasOpenTurn() {
		tr.Closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
		tr.ClosedResolution = s.ResolveTurn(tr.Closed)
		tr.Damaged = s.applyResolution(tr.ClosedResolution)
	}
	opened, err := s.roundOrch.NextAction(s.activeRound, &s.activeQueue)
	if err != nil {
		return tr, err
	}
	tr.Opened = opened
	tr.OpenedResolution = s.ResolveTurn(opened)
	return tr, nil
}

func (s *MatchSession) PullAction(id uuid.UUID) (*TurnTransition, error) {
	tr := &TurnTransition{}
	if s.activeRound.HasOpenTurn() {
		tr.Closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
		tr.ClosedResolution = s.ResolveTurn(tr.Closed)
		tr.Damaged = s.applyResolution(tr.ClosedResolution)
	}
	opened, err := s.roundOrch.PullAction(s.activeRound, &s.activeQueue, id)
	if err != nil {
		return tr, err
	}
	tr.Opened = opened
	tr.OpenedResolution = s.ResolveTurn(opened)
	return tr, nil
}
```

and point `AttachReaction` at the shared path:

```go
func (s *MatchSession) AttachReaction(r *action.Action) (*service.TurnResolution, error) {
	s.rollActionDice(r)
	if err := s.roundOrch.AttachReaction(s.activeRound, r); err != nil {
		return nil, err
	}
	return s.ResolveTurn(s.activeRound.CurrentTurn()), nil
}
```

- [ ] **Step 6: Update the existing session tests to the new shape**

Every `closed, opened, err := s.OpenNextAction()` becomes `tr, err := s.OpenNextAction()`
with `tr.Closed` / `tr.Opened`; same for `PullAction`. Keep every assertion.

- [ ] **Step 7: Run the tests**

Run: `go test ./internal/domain/match/... -v`
Expected: PASS, including the new damage test.

- [ ] **Step 8: Commit**

```bash
git add internal/domain/match/matchsession/
git commit -m "feat(match): resolve with the sheets and apply damage once, at turn close

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 10: The use cases persist the damaged sheets

The session changed the live sheet; the row in `character_sheets` has to follow, or the
sidebar will not show the new HP after a reload — which is one of the phase's done-criteria.

`sheet.Repository.UpdateStatusBars` already exists and `cmd/game/main.go` already has
`sheetRepository` in scope.

**Files:**
- Create: `internal/application/match/i_sheet_status_writer.go`
- Modify: `internal/application/match/open_next_action.go`, `pull_action.go`
- Modify: `internal/application/match/open_next_action_test.go`, `pull_action_test.go`
- Modify: `internal/app/game/room.go` (call sites), `internal/app/game/game_test.go`,
  `fog_dispatch_test.go` (constructor stubs, if they build these UCs)
- Modify: `cmd/game/main.go`

**Interfaces:**
- Consumes: `matchsession.TurnTransition`, `matchsession.DamagedCharacter` (Task 9).
- Produces: `match.ISheetStatusWriter`;
  `match.NewOpenNextActionUC(statusWriter ISheetStatusWriter) *OpenNextActionUC`;
  `match.NewPullActionUC(statusWriter ISheetStatusWriter) *PullActionUC`;
  `OpenNextActionResult{ClosedTurn, OpenedTurn *turn.Turn; Resolution, ClosedResolution *service.TurnResolution; Damaged []matchsession.DamagedCharacter}` (same fields on `PullActionResult`).

- [ ] **Step 1: Write the failing test**

Add to `internal/application/match/open_next_action_test.go`:

```go
// fakeStatusWriter records what was persisted.
type fakeStatusWriter struct {
	calls []string
	err   error
}

func (f *fakeStatusWriter) UpdateStatusBars(
	_ context.Context, sheetUUID string, _, _, _ status.IStatusBar,
) error {
	f.calls = append(f.calls, sheetUUID)
	return f.err
}

func TestOpenNextActionUC_PersistsDamagedSheets(t *testing.T) {
	// A transition that damaged nobody must not write.
	writer := &fakeStatusWriter{}
	playerA := uuid.New()
	session, chars := sessionWithPlayers(playerA)
	masterUUID := uuid.New()

	a := action.NewAction(chars[0], nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: 5}},
		nil, nil, nil, nil, nil, nil, nil)
	session.EnqueueAction(playerA, a) //nolint:errcheck

	uc := match.NewOpenNextActionUC(writer)
	if _, err := uc.Execute(context.Background(), session, masterUUID, masterUUID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(writer.calls) != 0 {
		t.Errorf("expected no persistence when nothing took damage, got %v", writer.calls)
	}
}

func TestOpenNextActionUC_StillRefusesANonMaster(t *testing.T) {
	uc := match.NewOpenNextActionUC(&fakeStatusWriter{})
	session, _ := sessionWithPlayers(uuid.New())
	_, err := uc.Execute(context.Background(), session, uuid.New(), uuid.New())
	if !errors.Is(err, match.ErrNotMatchMaster) {
		t.Errorf("expected ErrNotMatchMaster, got %v", err)
	}
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/application/match/ -run OpenNextActionUC -v`
Expected: FAIL — `NewOpenNextActionUC` takes no arguments.

- [ ] **Step 3: Declare the port**

`internal/application/match/i_sheet_status_writer.go`:

```go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
)

// ISheetStatusWriter persists a character's status bars.
//
// Combat damage lands on the live sheet inside the session, which is in memory and dies
// with the process. The row has to follow, and it is the only way the change reaches a
// player: the match sidebar reads HP over REST, and there is no WS event for it — that is
// Phase 6's work, deliberately not this one's.
type ISheetStatusWriter interface {
	UpdateStatusBars(ctx context.Context, sheetUUID string, health, stamina, aura status.IStatusBar) error
}
```

- [ ] **Step 4: Rewrite the two use cases**

`internal/application/match/open_next_action.go`:

```go
package match

import (
	"context"
	"log"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type OpenNextActionResult struct {
	ClosedTurn *turn.Turn
	OpenedTurn *turn.Turn
	// Resolution is the newly opened turn's projection — a dry run, nothing applied.
	Resolution *service.TurnResolution
	// ClosedResolution is the resolution that was actually applied when the previous turn
	// closed. Nil on the first open of a round.
	ClosedResolution *service.TurnResolution
	Damaged          []matchsession.DamagedCharacter
}

type IOpenNextAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID) (*OpenNextActionResult, error)
}

type OpenNextActionUC struct {
	statusWriter ISheetStatusWriter
}

func NewOpenNextActionUC(statusWriter ISheetStatusWriter) *OpenNextActionUC {
	return &OpenNextActionUC{statusWriter: statusWriter}
}

func (uc *OpenNextActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
) (*OpenNextActionResult, error) {
	if callerUUID != masterUUID {
		return nil, ErrNotMatchMaster
	}
	tr, err := session.OpenNextAction()
	if err != nil {
		return nil, err
	}
	persistDamage(ctx, uc.statusWriter, tr.Damaged)
	return &OpenNextActionResult{
		ClosedTurn:       tr.Closed,
		OpenedTurn:       tr.Opened,
		Resolution:       tr.OpenedResolution,
		ClosedResolution: tr.ClosedResolution,
		Damaged:          tr.Damaged,
	}, nil
}

// persistDamage writes the applied HP reductions through.
//
// A failure is logged, not returned: the damage is already applied in memory and the turn is
// already closed, so refusing the whole operation would leave the table stuck with a turn
// that will not open. Losing a row is recoverable; losing the baton is not.
func persistDamage(ctx context.Context, w ISheetStatusWriter, damaged []matchsession.DamagedCharacter) {
	if w == nil {
		return
	}
	for _, d := range damaged {
		bars := d.Sheet.GetAllStatusBar()
		if err := w.UpdateStatusBars(
			ctx, d.CharacterID.String(),
			bars[enum.Health], bars[enum.Stamina], bars[enum.Aura],
		); err != nil {
			log.Printf("UpdateStatusBars for %s: %v", d.CharacterID, err)
		}
	}
}
```

Apply the identical shape to `pull_action.go` (`PullActionResult` gains the same two fields;
`NewPullActionUC(statusWriter ISheetStatusWriter)`; reuse `persistDamage`).

- [ ] **Step 5: Follow the constructor change through**

- `cmd/game/main.go`:
  ```go
	openNextActionUC := match.NewOpenNextActionUC(sheetRepository)
	pullActionUC := match.NewPullActionUC(sheetRepository)
  ```
- `internal/app/game/room.go`: the two `IOpenNextAction`/`IPullAction` handlers now read
  `result.OpenedTurn` where they read `result.OpenedTurn` already — no change — but guard the
  nil case, since a queue-empty error now returns before `OpenedTurn` is set. The existing
  `if err != nil { return }` already covers it; add `if result.OpenedTurn == nil { return }`
  right after, so a nil turn can never reach `GetAction()`.
- `internal/app/game/game_test.go` and `fog_dispatch_test.go`: if either constructs these use
  cases, pass `nil` — `persistDamage` no-ops on a nil writer, which is exactly what a delivery
  test wants.

- [ ] **Step 6: Run everything**

Run: `go build ./... && go test ./internal/... && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/application/match/ internal/app/game/ cmd/game/main.go
git commit -m "feat(match): persist the status bars of everyone a closing turn damaged

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 11: `buildAction` maps the whole payload

The WS contract already carries `Skills`, `Speed`, `Feint`, `Attack`, `Defense` and
`Move.Speed/Charge`. The mapper builds `Move`, `Dodge` and `Interact` and drops the rest —
which is why nothing downstream has ever had a number to work with.

The `string → enum.SkillName` boundary lives here: an unknown skill name is a client bug and
must come back as a WS error, not as a silent zero deep in the resolver.

**Files:**
- Modify: `internal/app/game/action_mapper.go`
- Modify: `internal/app/game/room.go` (handle the new error return)
- Test: `internal/app/game/action_mapper_test.go`

**Interfaces:**
- Consumes: `action.RollCheck.Attempts` (Task 1), `ActionPayload.ActorID` (Task 7).
- Produces: `buildAction(actorCharID uuid.UUID, p ActionPayload) (*action.Action, error)`.

- [ ] **Step 1: Write the failing test**

`internal/app/game/action_mapper_test.go`:

```go
package game

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

func TestBuildAction_MapsTheWholePayload(t *testing.T) {
	actorCharID := uuid.New()
	targetID := uuid.New()
	weapon := "Sword"

	p := ActionPayload{
		ActorID:  actorCharID,
		TargetID: []uuid.UUID{targetID},
		Skills: []ActionSkillPayload{
			{SkillName: enum.Acrobatics.String()},
		},
		Speed: &ActionSpeedPayload{
			Bar:       1,
			RollCheck: &RollCheckPayload{SkillName: enum.Legerity.String()},
		},
		Feint: &RollCheckPayload{SkillName: enum.Feint.String()},
		Attack: &AttackPayload{
			Weapon: &weapon,
			Hit:    RollCheckPayload{SkillName: enum.Accuracy.String()},
			Damage: RollCheckPayload{SkillName: enum.Push.String()},
			Charge: &RollCheckPayload{SkillName: enum.Brake.String()},
		},
		Defense: &DefensePayload{
			RollCheck: RollCheckPayload{SkillName: enum.Defense.String()},
		},
	}

	a, err := buildAction(actorCharID, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("the actor is the character, not the player", func(t *testing.T) {
		if a.GetActorID() != actorCharID {
			t.Errorf("actorID = %v, want %v", a.GetActorID(), actorCharID)
		}
	})

	t.Run("the attack survives", func(t *testing.T) {
		if a.Attack == nil {
			t.Fatal("Attack was dropped")
		}
		if a.Attack.Weapon == nil || *a.Attack.Weapon != enum.Sword {
			t.Errorf("Weapon = %v, want Sword", a.Attack.Weapon)
		}
		if a.Attack.Hit.SkillName != enum.Accuracy.String() {
			t.Errorf("Hit.SkillName = %q, want Accuracy", a.Attack.Hit.SkillName)
		}
		if a.Attack.Charge == nil {
			t.Error("Charge was dropped")
		}
	})

	t.Run("speed, skills, feint and defense survive", func(t *testing.T) {
		if a.Speed.Bar != 1 || a.Speed.SkillName != enum.Legerity.String() {
			t.Errorf("Speed = %+v, want bar 1 and Legerity", a.Speed)
		}
		if len(a.Skills) != 1 || a.Skills[0].SkillName != enum.Acrobatics.String() {
			t.Errorf("Skills = %+v, want one Acrobatics", a.Skills)
		}
		if a.Feint == nil {
			t.Error("Feint was dropped")
		}
		if a.Defense == nil || a.Defense.SkillName != enum.Defense.String() {
			t.Errorf("Defense = %+v, want a Defense check", a.Defense)
		}
	})

	t.Run("no dice have fallen yet — that is the session's job", func(t *testing.T) {
		if !a.Attack.Hit.Attempts.IsEmpty() {
			t.Error("the mapper must not roll; the session rolls on arrival")
		}
	})
}

func TestBuildAction_RejectsAnUnknownSkillName(t *testing.T) {
	p := ActionPayload{
		ActorID: uuid.New(),
		Attack: &AttackPayload{
			Hit:    RollCheckPayload{SkillName: "Kamehameha"},
			Damage: RollCheckPayload{SkillName: enum.Push.String()},
		},
	}
	if _, err := buildAction(p.ActorID, p); err == nil {
		t.Error("expected an unknown skill name to be rejected at the boundary")
	}
}

func TestBuildAction_KeepsMappingWhatItAlreadyMapped(t *testing.T) {
	actorCharID := uuid.New()
	p := ActionPayload{
		ActorID: actorCharID,
		Move: &MovePayload{
			Category: string(enum.Dash),
			From:     [3]int{1, 1, 0},
			Position: [3]int{2, 1, 0},
			Speed:    &RollCheckPayload{SkillName: enum.Accelerate.String()},
		},
		Dodge:     &DodgePayload{Category: string(enum.Evasive), RollCheck: &RollCheckPayload{SkillName: enum.Reflex.String()}},
		ReactToID: uuid.New(),
	}
	a, err := buildAction(actorCharID, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Move == nil || a.Move.Position != [3]int{2, 1, 0} {
		t.Errorf("Move = %+v, want the mapped position", a.Move)
	}
	if a.Move.Speed == nil || a.Move.Speed.SkillName != enum.Accelerate.String() {
		t.Error("Move.Speed was dropped")
	}
	if a.Dodge == nil {
		t.Error("Dodge was dropped")
	}
}
```

The enum values used above are real: `enum.MoveCategory` carries `Shift`, `Dash`, `Back`,
`Roll`, `Slide`, `Jump`, `FlatJump`; `enum.DodgeCategory` carries `Evasive`, `Close`, `Scape`.

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/app/game/ -run BuildAction -v`
Expected: FAIL — `buildAction` returns one value, not two.

- [ ] **Step 3: Rewrite the mapper**

`internal/app/game/action_mapper.go` — replace `buildAction` entirely:

```go
// buildAction maps an ActionPayload received from the WebSocket client to an Action domain
// entity.
//
// actorCharID is the acting CHARACTER's sheet UUID, resolved by the caller from the payload
// and checked against the authenticated player by the session. The combat entity is the
// character: one person drives several of them.
//
// The mapper never rolls. The dice fall in MatchSession the moment the action is accepted,
// so a payload that is rejected costs nothing and an accepted one is rolled exactly once.
func buildAction(actorCharID uuid.UUID, p ActionPayload) (*action.Action, error) {
	skills, err := buildSkills(p.Skills)
	if err != nil {
		return nil, err
	}

	speed := action.ActionSpeed{}
	if p.Speed != nil {
		speed.Bar = p.Speed.Bar
		rc, err := buildRollCheck(p.Speed.RollCheck)
		if err != nil {
			return nil, err
		}
		if rc != nil {
			speed.RollCheck = *rc
		}
	}

	feint, err := buildRollCheck(p.Feint)
	if err != nil {
		return nil, err
	}

	var move *action.Move
	if p.Move != nil {
		moveSpeed, err := buildRollCheck(p.Move.Speed)
		if err != nil {
			return nil, err
		}
		moveCharge, err := buildRollCheck(p.Move.Charge)
		if err != nil {
			return nil, err
		}
		move = &action.Move{
			Category: enum.MoveCategory(p.Move.Category),
			From:     p.Move.From,
			Position: p.Move.Position,
			Speed:    moveSpeed,
			Charge:   moveCharge,
		}
		// FinalSpeed is computed by the engine, never taken from the client.
	}

	var attack *action.Attack
	if p.Attack != nil {
		hit, err := buildRollCheck(&p.Attack.Hit)
		if err != nil {
			return nil, err
		}
		damage, err := buildRollCheck(&p.Attack.Damage)
		if err != nil {
			return nil, err
		}
		charge, err := buildRollCheck(p.Attack.Charge)
		if err != nil {
			return nil, err
		}
		weapon, err := buildWeaponName(p.Attack.Weapon)
		if err != nil {
			return nil, err
		}
		attack = &action.Attack{
			Weapon: weapon,
			Hit:    *hit,
			Damage: *damage,
			Charge: charge,
		}
	}

	var defense *action.Defense
	if p.Defense != nil {
		rc, err := buildRollCheck(&p.Defense.RollCheck)
		if err != nil {
			return nil, err
		}
		weapon, err := buildWeaponName(p.Defense.Weapon)
		if err != nil {
			return nil, err
		}
		defense = &action.Defense{Weapon: weapon, RollCheck: *rc}
	}

	var dodge *action.Dodge
	if p.Dodge != nil {
		rc, err := buildRollCheck(p.Dodge.RollCheck)
		if err != nil {
			return nil, err
		}
		dodge = &action.Dodge{Category: enum.DodgeCategory(p.Dodge.Category)}
		if rc != nil {
			dodge.RollCheck = *rc
		}
	}

	var interact *action.Interact
	if p.Interact != nil {
		interact = &action.Interact{Kind: action.InteractKind(p.Interact.Kind)}
	}

	return action.NewAction(
		actorCharID, p.TargetID, p.ReactToID,
		skills, speed,
		feint, move, attack, defense, dodge, nil, interact,
	), nil
}

// buildRollCheck crosses the string→enum boundary for a skill name. An unknown name is a
// client bug and comes back as an error here, where it can still be answered with a WS
// error, instead of contributing a silent zero deep inside the resolver.
func buildRollCheck(p *RollCheckPayload) (*action.RollCheck, error) {
	if p == nil {
		return nil, nil
	}
	if p.SkillName != "" {
		if _, err := enum.SkillNameFrom(p.SkillName); err != nil {
			return nil, err
		}
	}
	return &action.RollCheck{SkillName: p.SkillName}, nil
}

func buildSkills(ps []ActionSkillPayload) ([]action.Skill, error) {
	if len(ps) == 0 {
		return nil, nil
	}
	out := make([]action.Skill, 0, len(ps))
	for _, s := range ps {
		if _, err := enum.SkillNameFrom(s.SkillName); err != nil {
			return nil, err
		}
		out = append(out, action.Skill{
			SkillName:  s.SkillName,
			Difficulty: s.Difficulty,
			RollCheck:  action.RollCheck{SkillName: s.SkillName},
		})
	}
	return out, nil
}

func buildWeaponName(s *string) (*enum.WeaponName, error) {
	if s == nil {
		return nil, nil
	}
	name, err := enum.WeaponNameFrom(*s)
	if err != nil {
		return nil, err
	}
	return &name, nil
}
```

Leave `buildMasterAction` untouched — its full mapping is deferred to Phase 4 and the TODOs in
it are the owner's markers.

- [ ] **Step 4: Handle the error at the two call sites**

In `internal/app/game/room.go`, both `case MsgTypeEnqueueAction` and `handleReaction`:

```go
		a, err := buildAction(payload.ActorID, payload)
		if err != nil {
			client.SendMessage(NewErrorMessage("invalid_action", err.Error()))
			return
		}
```

- [ ] **Step 5: Run the tests**

Run: `go test ./internal/app/game/ ./internal/... && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/
git commit -m "feat(game): map the whole action payload and validate skill names at the boundary

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 12: `resolution_updated` carries a real payload

Today it carries `IsSettled` and a `uuid.Nil` turn ID. It stays **master-only** — the
calculation belongs to the master until the turn closes, and per-recipient projection is
Phase 5's. The payload is a **slice** of the resolution, not the whole thing.

**Files:**
- Modify: `internal/app/game/message.go`
- Modify: `internal/app/game/room.go`
- Test: `internal/app/game/resolution_payload_test.go`

**Interfaces:**
- Consumes: `service.TurnResolution`, `service.CharacterResult`, `service.RollResult` (Task 6).
- Produces: `ResolutionUpdatedPayload{TurnID, IsSettled, Action, Targets}`;
  `RollResultPayload`; `CharacterResultPayload`;
  `newResolutionUpdatedPayload(turnID uuid.UUID, res *service.TurnResolution) ResolutionUpdatedPayload`.

- [ ] **Step 1: Write the failing test**

`internal/app/game/resolution_payload_test.go`:

```go
package game

import (
	"encoding/json"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestNewResolutionUpdatedPayload(t *testing.T) {
	turnID, targetID := uuid.New(), uuid.New()
	margin := 7
	res := &service.TurnResolution{
		IsSettled: false,
		ActionResult: service.RollResult{
			SkillName:  "Accuracy",
			SkillValue: 4,
			DiceRolled: []int{10, 8},
			Total:      22,
			IsCritical: false,
			Margin:     &margin,
		},
		CharacterResults: []service.CharacterResult{{
			TargetID:        targetID,
			Dodged:          false,
			Defended:        true,
			RawDamage:       14,
			EffectiveDamage: 14,
		}},
	}

	p := newResolutionUpdatedPayload(turnID, res)

	t.Run("carries the turn and the action roll", func(t *testing.T) {
		if p.TurnID != turnID {
			t.Errorf("TurnID = %v, want %v", p.TurnID, turnID)
		}
		if p.Action.Total != 22 || len(p.Action.DiceRolled) != 2 {
			t.Errorf("Action = %+v, want total 22 and the two dice", p.Action)
		}
		if p.Action.Margin == nil || *p.Action.Margin != 7 {
			t.Errorf("Margin = %v, want 7", p.Action.Margin)
		}
	})

	t.Run("carries the projected damage per target", func(t *testing.T) {
		if len(p.Targets) != 1 {
			t.Fatalf("Targets = %+v, want one entry", p.Targets)
		}
		if p.Targets[0].ProjectedDamage != 14 || !p.Targets[0].Defended {
			t.Errorf("Targets[0] = %+v", p.Targets[0])
		}
	})

	t.Run("serializes as camelCase", func(t *testing.T) {
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		s := string(b)
		for _, key := range []string{`"turnId"`, `"isSettled"`, `"diceRolled"`, `"projectedDamage"`} {
			if !contains(s, key) {
				t.Errorf("payload is missing %s: %s", key, s)
			}
		}
	})
}

func TestNewResolutionUpdatedPayload_NilResolution(t *testing.T) {
	turnID := uuid.New()
	p := newResolutionUpdatedPayload(turnID, nil)
	if p.TurnID != turnID {
		t.Errorf("TurnID = %v, want %v", p.TurnID, turnID)
	}
	if len(p.Targets) != 0 {
		t.Errorf("Targets = %+v, want empty", p.Targets)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
```

- [ ] **Step 2: Run the test and confirm it fails**

Run: `go test ./internal/app/game/ -run ResolutionUpdatedPayload -v`
Expected: FAIL — `undefined: newResolutionUpdatedPayload`.

- [ ] **Step 3: Write the payload**

In `internal/app/game/message.go`, replace `ResolutionUpdatedPayload` and add the two new
types plus the builder:

```go
// ResolutionUpdatedPayload is the master's view of a turn's current resolution.
//
// It is a SLICE of service.TurnResolution, not the whole thing: it carries the mechanics and
// the projection, and nothing that would let a client reconstruct state it is not entitled
// to. Master-only for now — the calculation belongs to the master until the turn closes, and
// per-recipient projection is a later slice.
//
// Damage is projected, not applied. The HP only moves when the turn closes.
type ResolutionUpdatedPayload struct {
	TurnID    uuid.UUID                `json:"turnId"`
	IsSettled bool                     `json:"isSettled"`
	Action    RollResultPayload        `json:"action"`
	Targets   []CharacterResultPayload `json:"targets"`
}

// RollResultPayload is one test as the master reads it. The individual dice travel because
// a critical is the combination, not the sum.
type RollResultPayload struct {
	SkillName         string `json:"skillName"`
	SkillValue        int    `json:"skillValue"`
	DiceRolled        []int  `json:"diceRolled"`
	Total             int    `json:"total"`
	IsCritical        bool   `json:"isCritical"`
	IsCriticalFailure bool   `json:"isCriticalFailure"`
	Margin            *int   `json:"margin,omitempty"`
}

// CharacterResultPayload is what one attack did to one target.
type CharacterResultPayload struct {
	TargetID        uuid.UUID `json:"targetId"`
	Dodged          bool      `json:"dodged"`
	Defended        bool      `json:"defended"`
	DodgeTotal      int       `json:"dodgeTotal"`
	DefenseTotal    int       `json:"defenseTotal"`
	RawDamage       int       `json:"rawDamage"`
	DefenseApplied  int       `json:"defenseApplied"`
	ProjectedDamage int       `json:"projectedDamage"`
}

func newResolutionUpdatedPayload(turnID uuid.UUID, res *service.TurnResolution) ResolutionUpdatedPayload {
	p := ResolutionUpdatedPayload{TurnID: turnID, Targets: []CharacterResultPayload{}}
	if res == nil {
		return p
	}
	p.IsSettled = res.IsSettled
	p.Action = RollResultPayload{
		SkillName:         res.ActionResult.SkillName,
		SkillValue:        res.ActionResult.SkillValue,
		DiceRolled:        res.ActionResult.DiceRolled,
		Total:             res.ActionResult.Total,
		IsCritical:        res.ActionResult.IsCritical,
		IsCriticalFailure: res.ActionResult.IsCriticalFailure,
		Margin:            res.ActionResult.Margin,
	}
	for _, cr := range res.CharacterResults {
		p.Targets = append(p.Targets, CharacterResultPayload{
			TargetID:        cr.TargetID,
			Dodged:          cr.Dodged,
			Defended:        cr.Defended,
			DodgeTotal:      cr.Dodge.Total,
			DefenseTotal:    cr.Defense.Total,
			RawDamage:       cr.RawDamage,
			DefenseApplied:  cr.DefenseApplied,
			ProjectedDamage: cr.EffectiveDamage,
		})
	}
	return p
}
```

Add the import
`"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"` to `message.go`.

- [ ] **Step 4: Send it from the three places a resolution is produced**

In `internal/app/game/room.go`:

`handleReaction` — replace the payload construction:

```go
	if hasMaster {
		masterClient.SendMessage(NewServerMessage(
			MsgTypeResolutionUpdate,
			newResolutionUpdatedPayload(currentTurnID(session), result.Resolution),
		))
	}
```

with a small helper next to it:

```go
// currentTurnID reads the open turn's ID, or uuid.Nil when there is none. The reaction path
// resolves the turn it attached to, and the master needs to know which one.
func currentTurnID(session *matchsession.MatchSession) uuid.UUID {
	r := session.GetActiveRound()
	if r == nil {
		return uuid.Nil
	}
	t := r.CurrentTurn()
	if t == nil {
		return uuid.Nil
	}
	return t.GetID()
}
```

`case MsgTypeOpenNextAction` and `case MsgTypePullAction` — after the existing
`broadcastWallResults` call, add the master-only send for the newly opened turn:

```go
		if result.Resolution != nil {
			r.sendToMaster(NewServerMessage(
				MsgTypeResolutionUpdate,
				newResolutionUpdatedPayload(result.OpenedTurn.GetID(), result.Resolution),
			))
		}
```

and, when a turn closed, the settled resolution of the one that just ended:

```go
		if result.ClosedTurn != nil && result.ClosedResolution != nil {
			r.sendToMaster(NewServerMessage(
				MsgTypeResolutionUpdate,
				newResolutionUpdatedPayload(result.ClosedTurn.GetID(), result.ClosedResolution),
			))
		}
```

Place the closed-turn send inside the existing `if result.ClosedTurn != nil { ... }` block,
after the `PersistTurnClose` call.

- [ ] **Step 5: Run the tests**

Run: `go build ./... && go test ./internal/... && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/
git commit -m "feat(game): give resolution_updated a real, master-only payload

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 13: Browser verification and documentation

The phase's done-criteria are stated in terms of a real browser and a real reload. There is no
UI to compose an attack — that is Phase 6 — so the action goes out over the live WebSocket of
a real, logged-in match page, from the browser's own console. That is still the browser, the
real server, and the real database.

**Files:**
- Modify: `docs/dev/match/combat-engine.md`
- Modify: `docs/dev/match/flows/05-lacunas.md`
- Modify: `docs/documentation-map.yaml`
- Modify: `AGENTS.md`

- [ ] **Step 1: Bring the stack up**

```bash
make migrate-up
make run-dev          # REST on :5000
# in another shell, from the same worktree:
go run ./cmd/game     # WS on :8081
```

and the front from `System_X_System_React/`: `npm run dev` (port 5173).

Log in as `test@mail.com` / `12345678`, and as a second account (`test2@mail.com`) in another
profile so there is a master and a player. Start a match with two enrolled characters.

- [ ] **Step 2: Send an attack from the browser console**

On the player's match page, in the DevTools console — the page already holds an open socket;
find it or open a second one against the same match with the same token:

```js
ws.send(JSON.stringify({
  type: "enqueue_action",
  payload: {
    actorId: "<attacker sheet uuid>",
    targetId: ["<victim sheet uuid>"],
    speed: { bar: 0, rollCheck: { skillName: "Legerity" } },
    attack: {
      weapon: "Sword",
      hit: { skillName: "Accuracy" },
      damage: { skillName: "Push" }
    }
  }
}));
```

Then, on the master's page, open the action:

```js
ws.send(JSON.stringify({ type: "open_next_action", payload: {} }));
```

- [ ] **Step 3: Confirm the four criteria, and write down what you saw**

1. The master's console receives `resolution_updated` with real dice, a total, the critical
   flags and a margin, plus a `targets[0].projectedDamage`.
2. The player's console receives **no** `resolution_updated` — it is master-only.
3. The victim's HP has **not** changed yet. Reload the victim's sheet and check.
4. Open a second action (`open_next_action` again, after enqueueing another one). The first
   turn closes, and **now** reloading the match page shows the victim's HP reduced in the
   sidebar by exactly the `projectedDamage` reported in step 1.

Record the actual numbers seen — dice, total, projected damage, HP before and after — for the
PR description. If the attack was dodged (a fresh sheet's passive dodge is 11, so a low roll
loses), send it again; that is the engine working, not a failure.

- [ ] **Step 4: Update the dev docs**

`docs/dev/match/combat-engine.md` — in "Pendências estruturais", flip the rows this phase
closed:

| Item | New state |
|---|---|
| `TurnResolver` — ramo `character` | ✅ Fase 2 — acerto, esquiva por reflexo e defesa passivas, dano, `CharacterResult` |
| `battle.Blow` | ✅ Fase 2 — construtor e acessores |
| `buildAction` | ✅ Fase 2 — payload inteiro, com fronteira `string → enum.SkillName` |

and add a short section recording: the roll-once seam (`RollSource`, dice living in
`RollCheck.Attempts`, rolled by the session on arrival), the dry-run/apply-at-close split, and
that `actorID` is now the sheet UUID.

`docs/dev/match/flows/05-lacunas.md` — §1, §3, §4 and §9 are closed by this phase; rewrite
them to say what is now true and what genuinely remains (§2 the queue still has no priority —
Phase 3; §5 initiative; §7 the round cycle; §8 visibility; §10 the open design points).

`docs/documentation-map.yaml` — make sure `internal/domain/match/service/` and
`internal/app/game/action_mapper.go` map to `combat-engine.md` and `05-lacunas.md`; add the
new files if the map lists files rather than directories.

`AGENTS.md` — under "Known Issues", record that Phase 2 of the combat engine is complete and
that the master-only `resolution_updated` and the missing character-HP WS event are deliberate
(Phase 5 and Phase 6 respectively), not oversights.

- [ ] **Step 5: Final verification**

```bash
go build ./... && go test ./internal/... && go vet -tags=integration ./internal/...
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add docs/ AGENTS.md
git commit -m "docs(match): record phase 2 of the combat engine as implemented

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

- [ ] **Step 7: Open the PR**

Say explicitly what was verified in the browser (with the numbers from Step 3) and what was
not. Cross-link nothing on the front — this phase touches no front code by design.

Per `CLAUDE.md`, if anything from Step 3 could not be verified end to end, run
`./dev-checkout.sh feat/combat-engine-phase-2` from `System_X_System_Project/` and say in the
PR what the owner should look at. If everything in Step 3 was verified, say that instead of
running the script.

---

## Self-review notes

**Spec coverage.** Every bullet of §5 "Fase 2" maps to a task: full `buildAction` → Task 11;
`actorID` as sheet UUID → Task 7; passive dodge and defense → Task 6; the character branch of
`TurnResolver` and `Blow` → Tasks 5 and 6; the margin ladder as a pure function → Task 3;
damage per §4.7 → Tasks 4, 6, 9, 10; resolving with the sheets → Task 9; `RollAttempts` on
`RollCheck` → Task 1; the deterministic seam → Task 2 and Task 8; `resolution_updated` → Task
12. The four done-criteria are checked in Task 13.

**Out of scope, honoured.** No bar arithmetic (Phase 3). No active reactions and no
multi-target chaining — the resolver loops targets, but nothing chains state between them, and
that is exactly what Phase 4 adds. No explicit turn close and no per-recipient projection
(Phase 5). No character-HP WS event and no front work (Phase 6). No critical multiplier.

**Known consequence worth stating in the PR.** `Action.Speed.Result` is still never filled, so
the priority queue still has no priority — Phase 3 owns that. The dice for actionSpeed *do*
now fall on arrival, so Phase 3 only has to derive them.
