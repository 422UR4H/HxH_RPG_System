# Combat Engine — Phase 4 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** The full reaction catalogue and the chain that resolves it. A target answers an
attack with one of seven declared reaction kinds, the master opens each reaction one at a time,
and the order they are opened changes the outcome — because the attack that leaves one
resolution is the attack that enters the next.

**Architecture:** The reaction declares its **kind** on the wire; nothing is inferred from
which components the payload filled in, because the three escapes have identical shape and
different prices. `action.ReactionKind` answers the bar cost, `service.TurnResolver` grows a
reaction branch keyed on that same kind, and the multi-target case becomes a sequential walk
carrying a **residual attack** rather than a batch of independent collisions. Everything the
resolver reads has already been rolled — `Resolve` stays pure, as Phase 2 made it — so a
reaction landing mid-turn just recomputes.

**Tech Stack:** Go 1.23, standard `testing` (table-driven, `t.Run`), external test packages,
gorilla-style WebSocket delivery in `internal/app/game/`.

**Spec:** `docs/superpowers/specs/2026-08-16-combat-engine-design.md` — §5 "Fase 4" is the
binding scope. The **rules** are in `docs/dev/match/combat-engine.md` §§ "Modificadores: o que
cada um modifica, e contra quem", "O bônus da esquiva fechada", "A cadeia em área — o que passa
de um alvo para o outro", "Reações", "A escada de resultados", "O tipo da reação é declarado,
não inferido", "O custo da reação na economia de barra", "O que a Fase 4 precisa consertar em
código já escrito". Player-facing narrative: `docs/game/combate/reacoes.md`.

**Branch:** `feat/combat-engine-phase-4`, from `main`.

> **Phase 4 runs on player characters.** The NPC rostering slice does not exist and is not a
> prerequisite: the done-criterion (three targets reacting differently) is reachable with three
> PCs. What rostering blocks is *the master sending an NPC's action*, which is not this phase's
> objective. It lands before Phase 5. Do not build it here.

---

## Global Constraints

- **Go 1.23**, module `github.com/422UR4H/HxH_RPG_System`. No test frameworks — standard
  `testing` only, table-driven with `t.Run()`, external test packages (`package foo_test`).
- **NEVER remove a TODO comment.** They are intentional markers left by the repo owner.
  `RoundOrchestrator.ChangeMode`'s `// TODO: create and finish Initiative to continue here`
  stays exactly where it is.
- **Layering:** `entity ← domain ← app`. `service` imports `action`, `round`, `turn`;
  **`action` must never import `service`**, and `action` must never import `match` (the
  `Modifier` package) — the reaction kind carries no ledger knowledge.
- **Domain services are stateless structs.** `service.TurnResolver{}`, `service.RoundScheduler{}`,
  `service.BarEconomy{}` and `service.RoundOrchestrator{}` must all keep working as zero values.
  Dependencies travel as parameters, never as fields.
- **`MatchSession` has no lock of its own.** `room.go`'s `r.mu` is the only serialization. Every
  new call into the session inherits the obligation to hold it **write-locked** whenever the
  session mutates — which now includes attaching a reaction (it charges bars and pulls from the
  queue) and opening one.
- **Wire format is camelCase** on both sides, via manual struct tags.
- **The master never re-rolls a player's die.** `Resolve` derives; it never rolls. A reaction's
  dice fall once, in `rollActionDice`, at attach.
- **A passive test must not consume dice.** The reflex dodge and the default defense take the
  set's average and must not call the roll source — a phantom roll drains a scripted
  `RollSource` and shifts every number after it (`combat-engine.md` § "Um gotcha de teste").
- **Reaction is never denied for lack of balance.** The two gates schedule actions inside a
  round; a reaction is not scheduled. It debits and the character starts the next round further
  behind. Never return an error that means "you may not defend yourself".
- **The bar economy is Race-only.** A reaction charges bars under the same gate `recordActed`
  already applies, and for the same reason: `settleBars` skips a bar that never priced, so a
  speed recorded under `Free` would be charged by nothing and reset by nothing.
- Commits include `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`.
- After every task touching `internal/`: `go vet -tags=integration ./internal/...`.

---

## The rules, restated

### The catalogue and what it costs

| `ReactionKind` | Player-facing | actionSpeed | moveSpeed |
|---|---|---|---|
| `nothing` | Não fazer nada — refuses even the passives | — | — |
| `dodge` | Esquivar ativamente — gamble the roll instead of the average | — | — |
| `closedDodge` | Esquiva fechada — Evasion folded in | — | — |
| `escape` | Escape padrão — gives up the default defense | ✔ | ✔ |
| `escapeGuard` | Escape defensivo — keeps the safety net | ✔ | ✔ |
| `closedEscape` | Escape fechado | — | ✔ |
| `repel` | Repelir — attacks the attack | ✔ | — |

The reflex dodge and the default defense are **not** kinds. Nobody sends them; they are what the
engine applies when nothing arrives. `nothing` exists precisely because *refusing* them is a
send, and the only way to tell "sent nothing" from "sent `nothing`" is to have received
something.

### The repel ladder

CD is the attack's hit total; `margin = repelTotal − CD`; the step is `MatchRules.LadderStep`.
`service.ClimbLadder` already computes the rung and the difference — Phase 2 wrote it pure and
left it unwired. This phase is its first consumer.

| Rung | Damage to the repeller | Ledger payout | What travels to the next target |
|---|---|---|---|
| `great_success` (`margin ≥ step`) | 0 | **bonus** `+Difference`, `actionSpeed`, **only against the attacker**, next turn | nothing — the attack stops |
| `success` (`0 ≤ margin < step`) | 0 | — | nothing — the attack stops |
| `near_miss` (`−step ≤ margin < 0`) | 0 — parried, not reduced | **penalty** `−Difference`, `actionSpeed`, **against anyone**, next turn | reduced by the repelling weapon's defense |
| `failure` (`margin < −step`) | full — **the passives do not apply** | — | reduced by the hit target's armour |

Ties favour the defender: `margin == 0` is `success`, because `ClimbLadder` reads `margin >= 0`
as cleared.

### The chain — what leaves one target and enters the next

The collision is **not** `f(action, reactions[])`. It is a walk:

```
ataque₀ → resolve(alvo A) → ataque₁ → resolve(alvo B) → ataque₂ → …
```

What travels is the **residual attack** — how much of the blow is left:

| The target… | What reaches the next |
|---|---|
| **Dodged** (any dodge or escape that worked) | the **full** attack — dodging does not spend the blow |
| **Repelled successfully** | **nothing — the attack stops here** |
| **Was hit** | reduced by the hit target's **armour** |
| **Defended** | reduced by the **defense of the weapon** they defended with |

**Stopping is not cancelling.** A target whose reaction was already attached still has its
reaction resolved and still gets to narrate — it simply can no longer be hit. Mechanically
wasted, narratively not.

> ⚠️ **There is no rigid rule here — it is contextual, and the master may override at any
> point.** What this phase encodes is the *default per reaction type*. Master override is a
> Phase 5 surface (`SystemData` audit); do not invent an override path now.

**Armour does not exist in this codebase.** There is no armour entity and no sheet field for
it, so the "was hit" row reduces by **zero** today. Encode the row, make it read a value that
is currently always 0, and comment it the way `ApplicableDefense` comments the missing damage
types — the shape is what matters. Do not invent an armour model.

#### Sequential × simultaneous

The table above is a **sequential** attack, one that travels through the targets. The other kind
hits everyone at once: there the attack **does not diminish** — everyone takes the same — while
the master still opens one target at a time so each can narrate.

That axis is **reserved, not implemented**: it will be a configuration of the ability type, and
special abilities do not exist until post-MVP. Give `Attack` the field, default it to
sequential, branch on it in the chain, and leave the simultaneous branch covered by a unit test
only. Building it as an `if` retrofitted later is exactly what the reservation avoids.

### The conversion cost

| Situation | What happens |
|---|---|
| Active reaction, **no** pending action | the reaction **becomes** the action. No penalty. |
| Active reaction, **with** a pending action | the pending action leaves the queue, and the reaction rolls at **Disadvantage** |
| Free reaction (`nothing`, `dodge`, `closedDodge`) and the passives | consumes nothing |

Disadvantage is a **mode of reading the dice**, never an `Amount`: `RollAttempts` already holds
both sets, and the disadvantage only decides which one `pickAttempt` reads. Which pending action
leaves: the one that **would have opened first for that character on that bar** — the best key.
The scheduler already knows how to rank; this invents no new criterion.

| Reaction | Consumes |
|---|---|
| `repel` | the next pending on the **action** bar |
| `closedEscape` | the next pending on the **move** bar |
| `escape`, `escapeGuard` | the next pending on **each** bar (a combined action counts once, and goes) |
| `nothing`, `dodge`, `closedDodge`, the passives | nothing — the discount paying for itself |

### Where a bias lives — three origins, three homes

| Origin | Home | Reach |
|---|---|---|
| Master | `action.RollCondition.Bias` | that roll |
| System, situational | **`service.RollInput.SystemBias`** ← new | that roll |
| System, accumulated | `match.ModifierLedger` | until it expires |

`RollCalculator.Derive` sums all three. None overwrites another, and the audit trail can still
tell them apart — which is the whole purpose of `Source`.

---

## Decisions taken while planning (technical form — not in the docs)

Recorded so the implementer never has to invent one, and so a later phase can read this plan as
history.

1. **`ReactionKind` and `Repel` are exported fields on `Action`, not constructor parameters.**
   `action.NewAction` already takes twelve positional arguments and is called from ~40 test
   sites; growing it to fourteen buys nothing. `TargetID` and `ReactToID` are already exported
   fields *and* parameters — that is history, not a pattern to extend. The mapper assigns the
   two new fields after construction.

2. **`Turn` records the order reactions were opened.** The chain walks targets in the order the
   master opened their reactions, which is not the order they arrived. `Turn.reactions` is
   append-ordered by *attach*, so opening order needs its own list. Targets whose reaction was
   never opened — or who never sent one — are walked afterwards, in `Action.TargetID` order,
   so the resolution is total and deterministic.

3. **"Next turn" is implemented by demotion, not by a timestamp.** At turn close, a
   `LifetimeNextTurn` modifier is demoted to `LifetimeEndOfTurn`; a `LifetimeEndOfTurn` one is
   dropped. One pass, no clock, and a bonus created during turn N survives exactly turn N+1.

4. **The old `match.Scope` (expiry) is renamed to `match.Lifetime`, and `Scope` is reused for
   the targeting axis** — as `combat-engine.md` § Modificadores specifies. The rename is safe
   under the compiler: every existing use is of the constants `ScopeEndOfTurn` /
   `ScopeEndOfRound`, which cease to exist, so nothing stale compiles. Do the rename in one
   commit, before anything reads the new field.

5. **An escape must carry a `Move`.** The three escape kinds displace by definition, and
   `SpeedOn(BarMove)` reads `Move.FinalSpeed`. A payload declaring an escape without a move is a
   client bug and is refused at the WS boundary, not defaulted.

6. **A reaction never enters `activeQueue`.** It lives on the turn. Nothing in `RoundScheduler`
   ever sees it, which is why `Bars()` returning empty for a free reaction breaks no caller.

---

## File structure

**Created**

| File | Responsibility |
|---|---|
| `internal/domain/match/entity/action/reaction_kind.go` | `ReactionKind`, its seven values, `Bars()`, `IsFree()`, `Displaces()` |
| `internal/domain/match/entity/action/reaction_kind_test.go` | the cost table, value by value |
| `internal/domain/match/entity/action/repel.go` | `Repel{Weapon, RollCheck}` |
| `internal/domain/match/service/reaction_collision.go` | one reaction against one attack — the per-kind branch |
| `internal/domain/match/service/reaction_collision_test.go` | each kind's collision, in isolation |
| `internal/domain/match/service/attack_chain.go` | the residual attack walking the targets |
| `internal/domain/match/service/attack_chain_test.go` | the four rows of the chain table, plus the simultaneous axis |
| `internal/application/match/open_reaction.go` | `OpenReactionUC` — the master passes the microphone to a reaction |
| `internal/app/game/reaction_chain_e2e_test.go` | the phase's done-criterion, over a real WebSocket |

**Modified**

| File | Change |
|---|---|
| `internal/domain/match/modifier.go` | `Dimension`; `Scope` becomes the targeting axis with three forms; old `Scope` → `Lifetime` with the missing rung; ledger queries filter by dimension; `AdvanceTurn` |
| `internal/domain/match/character_status.go` | `ExpireModifiers` gains a turn-advance sibling |
| `internal/domain/match/entity/action/action.go` | `ReactionKind` and `Repel` fields |
| `internal/domain/match/entity/action/bar.go` | `Bars()` consults the reaction kind first |
| `internal/domain/match/entity/action/dodge.go` | loses `Category` |
| `internal/domain/match/entity/action/attack.go` | gains the sequential/simultaneous axis |
| `internal/domain/entity/enum/dodge_category.go` | **deleted** — absorbed by `ReactionKind` |
| `internal/domain/match/entity/turn/turn.go` | records the order reactions were opened |
| `internal/domain/match/service/roll_calculator.go` | `RollInput.SystemBias`, `RollInput.Dimension` |
| `internal/domain/match/service/turn_resolver.go` | the chain replaces the per-target loop; reaction results become real |
| `internal/domain/match/service/round_scheduler.go` | `BestPendingFor` — which pending action a reaction consumes |
| `internal/domain/match/matchsession/match_session.go` | reaction authorization, target validation, bar charge, queue consumption, disadvantage, modifier expiry |
| `internal/app/game/message.go` | `reactionKind` and `repel` on the payload; reactions on `resolution_updated`; `open_reaction` |
| `internal/app/game/action_mapper.go` | maps the kind and the repel; refuses an escape with no move |
| `internal/app/game/room.go` | the paired `ReactToID` ⟺ `ReactionKind` validation; the `open_reaction` route |
| `docs/dev/match/combat-engine.md` | § "O que a Fase 4 fixou no motor" |
| `docs/documentation-map.yaml` | the new code paths |

---

## Task 1: Reshape `Modifier` — what it modifies, against whom, for how long

**Files:**
- Modify: `internal/domain/match/modifier.go`
- Modify: `internal/domain/match/character_status.go:40-44`
- Test: `internal/domain/match/modifier_test.go`

**Interfaces:**
- Consumes: nothing — this is the foundation task.
- Produces: `match.Dimension` (`DimActionSpeed`, `DimDodge`); `match.Lifetime`
  (`LifetimeEndOfTurn`, `LifetimeNextTurn`, `LifetimeEndOfRound`); `match.Scope`
  (`ScopeAnyone()`, `ScopeOnly(id)`, `ScopeAllBut(id)`) with `AppliesTo(targetID *uuid.UUID) bool`;
  `Modifier{Amount, Bias int; Applies Dimension; Source Source; Against Scope; ExpiresAt Lifetime; Reason string}`;
  `ModifierLedger.TotalAmount(dim Dimension, targetID *uuid.UUID) int` and
  `TotalBias(dim Dimension, targetID *uuid.UUID) int`.

> ⚠️ The identifier `Scope` changes meaning in this package: it stops being the expiry axis and
> becomes the targeting axis. The old constants `ScopeEndOfTurn` / `ScopeEndOfRound` are gone, so
> nothing stale can compile — but read every compile error rather than pattern-matching it.

- [ ] **Step 1: Write the failing tests**

```go
// internal/domain/match/modifier_test.go — add to the existing file
func TestScope_AppliesTo(t *testing.T) {
	a, b := uuid.New(), uuid.New()

	t.Run("anyone applies to everybody, including a roll with no named target", func(t *testing.T) {
		s := match.ScopeAnyone()
		if !s.AppliesTo(nil) || !s.AppliesTo(&a) {
			t.Fatal("ScopeAnyone must apply to every roll")
		}
	})

	t.Run("only X applies to X and to nobody else", func(t *testing.T) {
		s := match.ScopeOnly(a)
		if !s.AppliesTo(&a) {
			t.Error("ScopeOnly must apply to its own target")
		}
		if s.AppliesTo(&b) {
			t.Error("ScopeOnly must not apply to a different target")
		}
		// A targeted modifier never applies to an untargeted roll.
		if s.AppliesTo(nil) {
			t.Error("ScopeOnly must not apply to a roll with no target")
		}
	})

	t.Run("all but X is the closed dodge's reserve — everyone except the duel opponent", func(t *testing.T) {
		s := match.ScopeAllBut(a)
		if s.AppliesTo(&a) {
			t.Error("ScopeAllBut must not apply to the excluded target")
		}
		if !s.AppliesTo(&b) {
			t.Error("ScopeAllBut must apply to a third party")
		}
		// No named target means nobody in particular, which is not the excluded one.
		if !s.AppliesTo(nil) {
			t.Error("ScopeAllBut must apply to an untargeted roll")
		}
	})
}

func TestModifierLedger_TotalsAreScopedByDimension(t *testing.T) {
	attacker := uuid.New()
	l := match.NewModifierLedger()
	// The duel reserve: actionSpeed, only against the attacker.
	l.Add(match.Modifier{
		Amount: 7, Applies: match.DimActionSpeed, Source: match.SourceSystem,
		Against: match.ScopeOnly(attacker), ExpiresAt: match.LifetimeNextTurn,
	})
	// The closed dodge reserve: dodge, against everyone but the attacker.
	l.Add(match.Modifier{
		Amount: 4, Applies: match.DimDodge, Source: match.SourceSystem,
		Against: match.ScopeAllBut(attacker), ExpiresAt: match.LifetimeNextTurn,
	})

	if got := l.TotalAmount(match.DimActionSpeed, &attacker); got != 7 {
		t.Errorf("actionSpeed against the attacker = %d, want 7", got)
	}
	if got := l.TotalAmount(match.DimDodge, &attacker); got != 0 {
		t.Errorf("dodge against the attacker = %d, want 0 — the reserve is for third parties", got)
	}
	third := uuid.New()
	if got := l.TotalAmount(match.DimDodge, &third); got != 4 {
		t.Errorf("dodge against a third party = %d, want 4", got)
	}
	if got := l.TotalAmount(match.DimActionSpeed, &third); got != 0 {
		t.Errorf("actionSpeed against a third party = %d, want 0 — the duel bonus is targeted", got)
	}
}

func TestModifierLedger_AdvanceTurn(t *testing.T) {
	l := match.NewModifierLedger()
	l.Add(match.Modifier{Amount: 1, Applies: match.DimActionSpeed, ExpiresAt: match.LifetimeEndOfTurn, Reason: "this turn"})
	l.Add(match.Modifier{Amount: 2, Applies: match.DimActionSpeed, ExpiresAt: match.LifetimeNextTurn, Reason: "next turn"})
	l.Add(match.Modifier{Amount: 4, Applies: match.DimActionSpeed, ExpiresAt: match.LifetimeEndOfRound, Reason: "the round"})

	l.AdvanceTurn()

	// end_of_turn died with its own turn; next_turn was demoted and is now live for one turn.
	if got := l.TotalAmount(match.DimActionSpeed, nil); got != 6 {
		t.Fatalf("after one turn = %d, want 6 (2 demoted + 4 round-scoped)", got)
	}
	l.AdvanceTurn()
	if got := l.TotalAmount(match.DimActionSpeed, nil); got != 4 {
		t.Fatalf("after two turns = %d, want 4 — the demoted bonus lasted exactly one turn", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/domain/match/ -run 'TestScope_AppliesTo|TestModifierLedger' -v`
Expected: FAIL — `undefined: match.ScopeAnyone`, `undefined: match.DimActionSpeed`.

- [ ] **Step 3: Rewrite `modifier.go`**

```go
package match

import "github.com/google/uuid"

// Dimension is WHAT a Modifier modifies.
//
// The system has more than one kind of accumulated reserve, and they are not
// interchangeable: the duel reserve (repel/parry) moves actionSpeed, while the closed
// dodge's reserve moves the dodge itself. Without this field the ledger cannot hold both,
// and a caller would have to know which entries it was allowed to read — which is exactly
// how the "always an actionSpeed adjustment" over-generalization happened.
type Dimension string

const (
	DimActionSpeed Dimension = "action_speed"
	DimDodge       Dimension = "dodge"
)

// Lifetime is when a Modifier stops applying.
//
// LifetimeNextTurn is the rung the repel bonus needs. A bonus created during turn N with
// LifetimeEndOfTurn would die at the close of N itself — but the bonus is earned in N and
// spent in N+1. It is implemented by demotion in AdvanceTurn, not by a clock.
type Lifetime string

const (
	LifetimeEndOfTurn  Lifetime = "end_of_turn"
	LifetimeNextTurn   Lifetime = "next_turn"
	LifetimeEndOfRound Lifetime = "end_of_round"
)

// Source records who created a Modifier. The system generates disadvantage on its own; the
// master grants or cancels advantage by hand. Keeping them apart is what lets the master
// cancel the system's disadvantage without either one overwriting the other, and it is what
// the audit trail reads to tell the two apart.
type Source string

const (
	SourceSystem Source = "system"
	SourceMaster Source = "master"
)

// ScopeKind is the shape of a Scope. It is not used directly — build a Scope through one of
// the three constructors, so an unset ID can never be mistaken for "anyone".
type ScopeKind string

const (
	scopeAnyone ScopeKind = "anyone"
	scopeOnly   ScopeKind = "only"
	scopeAllBut ScopeKind = "all_but"
)

// Scope is WHOM a Modifier counts against, and it has three forms rather than two.
//
// "All but X" is not a convenience: it is the only shape that expresses the closed dodge's
// reserve — the dodge a character did not need to spend, kept for whoever comes at them from
// outside the duel. A nil-or-one-target pointer cannot say it.
type Scope struct {
	kind ScopeKind
	id   uuid.UUID
}

func ScopeAnyone() Scope          { return Scope{kind: scopeAnyone} }
func ScopeOnly(id uuid.UUID) Scope   { return Scope{kind: scopeOnly, id: id} }
func ScopeAllBut(id uuid.UUID) Scope { return Scope{kind: scopeAllBut, id: id} }

// AppliesTo reports whether this scope counts for a roll made against targetID.
//
// A roll with no named target is "nobody in particular": a targeted bonus does not reach it,
// and an all-but exclusion does not exclude it.
func (s Scope) AppliesTo(targetID *uuid.UUID) bool {
	switch s.kind {
	case scopeOnly:
		return targetID != nil && *targetID == s.id
	case scopeAllBut:
		return targetID == nil || *targetID != s.id
	default:
		return true
	}
}

// Modifier is one accumulated bonus or penalty carried by a character.
//
// Amount and Bias are different currencies and never substitute for each other:
// Amount is a flat adjustment to the total; Bias is advantage/disadvantage on the dice
// (−1/0/+1, accumulating), which is a change in how the roll is made, not a number that
// can be added to it.
type Modifier struct {
	Amount    int
	Bias      int
	Applies   Dimension
	Source    Source
	Against   Scope
	ExpiresAt Lifetime
	Reason    string // surfaced in the Action History and in the audit trail
}

// ModifierLedger is the accumulated difference a character carries: the bonuses and
// penalties that the repel ladder, the closed dodge, the system and the master have piled on.
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

// TotalAmount sums the flat adjustments on one dimension that apply against targetID.
func (l *ModifierLedger) TotalAmount(dim Dimension, targetID *uuid.UUID) int {
	total := 0
	for _, m := range l.modifiers {
		if m.Applies == dim && m.Against.AppliesTo(targetID) {
			total += m.Amount
		}
	}
	return total
}

// TotalBias sums the dice biases on one dimension that apply against targetID. Advantage and
// disadvantage accumulate and can cancel each other out.
func (l *ModifierLedger) TotalBias(dim Dimension, targetID *uuid.UUID) int {
	total := 0
	for _, m := range l.modifiers {
		if m.Applies == dim && m.Against.AppliesTo(targetID) {
			total += m.Bias
		}
	}
	return total
}

// Expire drops every modifier whose validity ended at lifetime.
func (l *ModifierLedger) Expire(lifetime Lifetime) {
	kept := l.modifiers[:0]
	for _, m := range l.modifiers {
		if m.ExpiresAt != lifetime {
			kept = append(kept, m)
		}
	}
	l.modifiers = kept
}

// AdvanceTurn moves the ledger one turn forward: what was scoped to this turn dies, and what
// was earned FOR the next turn becomes this-turn's.
//
// This is how "next turn" is implemented — a demotion, in one pass, with no clock. A bonus
// created during turn N is demoted at N's close and dropped at N+1's, so it is live for
// exactly one turn.
func (l *ModifierLedger) AdvanceTurn() {
	kept := l.modifiers[:0]
	for _, m := range l.modifiers {
		switch m.ExpiresAt {
		case LifetimeEndOfTurn:
			continue
		case LifetimeNextTurn:
			m.ExpiresAt = LifetimeEndOfTurn
		}
		kept = append(kept, m)
	}
	l.modifiers = kept
}
```

In `character_status.go`, rename the parameter type and add the sibling:

```go
// ExpireModifiers drops every ledger entry whose validity ended at lifetime. Called when a
// round closes.
func (s *CharacterStatus) ExpireModifiers(lifetime Lifetime) {
	s.Ledger.Expire(lifetime)
}

// AdvanceTurn moves this character's ledger one turn forward. Called when a turn closes.
func (s *CharacterStatus) AdvanceTurn() {
	s.Ledger.AdvanceTurn()
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/domain/match/... ./internal/domain/match/service/... -v`
Expected: PASS. Existing callers of `TotalAmount`/`TotalBias` in `roll_calculator.go` will not
compile yet — Task 3 fixes them. To keep this task green on its own, pass `DimActionSpeed` and
the existing `in.AgainstID` at both call sites now; Task 3 replaces that with a real parameter.

- [ ] **Step 5: Run the full suite and vet**

Run: `go test ./internal/... && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/modifier.go internal/domain/match/character_status.go \
        internal/domain/match/modifier_test.go internal/domain/match/service/roll_calculator.go
git commit -m "feat(match): a modifier says what it changes, against whom, and for how long

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2: Wire the expiry — nothing expires today

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go:373-381` (`closeOpenTurn`), `:444-453` (`CloseRound`)
- Test: `internal/domain/match/matchsession/match_session_test.go`

**Interfaces:**
- Consumes: `match.Lifetime`, `CharacterStatus.AdvanceTurn`, `CharacterStatus.ExpireModifiers` (Task 1).
- Produces: the guarantee that a ledger entry written by any later task actually goes away.

`CharacterStatus.ExpireModifiers` has **no caller anywhere in the repository**. Phase 4 is the
first phase that writes into the ledger, so it is also the first that must clean it up. Without
this task every bonus this phase grants is permanent.

- [ ] **Step 1: Write the failing test**

```go
// internal/domain/match/matchsession/match_session_test.go
func TestMatchSession_ModifiersExpireOnClose(t *testing.T) {
	t.Run("a next-turn bonus survives its own turn and dies with the following one", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)

		status, err := s.GetCharacterStatus(chars[0])
		if err != nil {
			t.Fatalf("GetCharacterStatus: %v", err)
		}
		status.Ledger.Add(match.Modifier{
			Amount: 5, Applies: match.DimActionSpeed, Source: match.SourceSystem,
			Against: match.ScopeAnyone(), ExpiresAt: match.LifetimeNextTurn, Reason: "repel bonus",
		})

		// Turn 1 closes: the bonus is demoted, not dropped — it is earned for the NEXT turn.
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 10)) //nolint:errcheck
		mustOpen(t, s)
		s.EnqueueAction(playerB, makeActionWithSpeed(chars[1], 9)) //nolint:errcheck
		mustOpen(t, s) // opening the next turn closes the first

		if got := status.Ledger.TotalAmount(match.DimActionSpeed, nil); got != 5 {
			t.Fatalf("after one close = %d, want 5 — the bonus is for the next turn", got)
		}

		// Turn 2 closes: now it is spent.
		s.EnqueueAction(playerA, makeActionWithSpeed(chars[0], 8)) //nolint:errcheck
		mustOpen(t, s)
		if got := status.Ledger.TotalAmount(match.DimActionSpeed, nil); got != 0 {
			t.Fatalf("after two closes = %d, want 0 — the bonus lasted exactly one turn", got)
		}
	})

	t.Run("a round-scoped modifier dies when the round closes", func(t *testing.T) {
		playerA := uuid.New()
		s, chars := sessionWithParticipants(playerA)
		status, err := s.GetCharacterStatus(chars[0])
		if err != nil {
			t.Fatalf("GetCharacterStatus: %v", err)
		}
		status.Ledger.Add(match.Modifier{
			Amount: 3, Applies: match.DimActionSpeed, Against: match.ScopeAnyone(),
			ExpiresAt: match.LifetimeEndOfRound, Reason: "round penalty",
		})

		closeExhaustedRound(t, s)

		if got := status.Ledger.TotalAmount(match.DimActionSpeed, nil); got != 0 {
			t.Fatalf("after the round closed = %d, want 0", got)
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/match/matchsession/ -run TestMatchSession_ModifiersExpireOnClose -v`
Expected: FAIL — the bonus is still 5 after the second close, because nothing expires.

- [ ] **Step 3: Wire the two callers**

In `closeOpenTurn`, after the resolution is applied:

```go
func (s *MatchSession) closeOpenTurn() *TurnTransition {
	tr := &TurnTransition{}
	if !s.activeRound.HasOpenTurn() {
		return tr
	}
	tr.Closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
	tr.ClosedResolution = s.ResolveTurn(tr.Closed)
	tr.Damaged = s.applyResolution(tr.ClosedResolution)
	// The ledger moves one turn forward with the turn: what was scoped to this turn dies, and
	// what was earned FOR the next one becomes live. It happens AFTER applyResolution, so a
	// modifier that was meant to count in this turn still counted.
	s.advanceLedgers()
	return tr
}

// advanceLedgers moves every character's ledger one turn forward. Every character, not just
// the ones who acted: a penalty is carried by whoever earned it, and they may not have moved.
func (s *MatchSession) advanceLedgers() {
	for _, status := range s.statuses {
		status.AdvanceTurn()
	}
}
```

In `CloseRound`, beside `settleBars`:

```go
func (s *MatchSession) CloseRound() (*round.Round, error) {
	if s.activeRound.HasOpenTurn() {
		return nil, ErrRoundHasOpenTurn
	}
	s.settleBars()
	for _, status := range s.statuses {
		status.ExpireModifiers(match.LifetimeEndOfRound)
	}
	mode := s.activeRound.GetMode()
	closed := s.roundOrch.CloseRound(s.activeRound, time.Now())
	s.activeRound = round.NewRound(mode)
	s.roundPersisted = false
	return closed, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/domain/match/matchsession/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/matchsession/match_session.go \
        internal/domain/match/matchsession/match_session_test.go
git commit -m "feat(match): the ledger moves forward with the turn, and empties with the round

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3: A bias for one roll only

**Files:**
- Modify: `internal/domain/match/service/roll_calculator.go`
- Modify: `internal/domain/match/matchsession/match_session.go:515-570` (`deriveSpeeds`)
- Modify: `internal/domain/match/service/turn_resolver.go:200-260` (`resolveCharacter`)
- Test: `internal/domain/match/service/roll_calculator_test.go`

**Interfaces:**
- Consumes: `match.Dimension`, `ModifierLedger.TotalAmount/TotalBias` with a dimension (Task 1).
- Produces: `service.RollInput` gains `SystemBias int` and `Dimension match.Dimension`.
  `Derive` sums master bias + system bias + ledger bias.

The disadvantage the system generates on an action→reaction conversion cannot live in the
ledger: the ledger is the character's and lasts until it expires, so it would apply to *every*
actionSpeed roll they make, not to the one conversion. `RollCondition.Bias` is the master's by
decision (§4.4). The third home is the roll itself.

- [ ] **Step 1: Write the failing test**

```go
// internal/domain/match/service/roll_calculator_test.go
func TestDerive_ThreeBiasOrigins(t *testing.T) {
	rules := match.NewDefaultMatchRules()
	// Primary sums 6, Secondary sums 14. Advantage reads Secondary, disadvantage Primary.
	attempts := action.RollAttempts{Primary: []int{4, 2}, Secondary: []int{9, 5}}

	t.Run("the system's situational bias reads the worse set on its own", func(t *testing.T) {
		out := service.RollCalculator{}.Derive(rules, attempts, service.RollInput{
			SkillName: "Legerity", SkillValue: 3, SystemBias: -1, Dimension: match.DimActionSpeed,
		})
		if out.DiceTotal != 6 {
			t.Errorf("DiceTotal = %d, want 6 — disadvantage reads Primary", out.DiceTotal)
		}
		if out.Bias != -1 {
			t.Errorf("Bias = %d, want -1", out.Bias)
		}
	})

	t.Run("the master can cancel the system's disadvantage without overwriting it", func(t *testing.T) {
		out := service.RollCalculator{}.Derive(rules, attempts, service.RollInput{
			SkillName: "Legerity", SkillValue: 3,
			SystemBias: -1,
			Condition:  &action.RollCondition{Bias: 1},
			Dimension:  match.DimActionSpeed,
		})
		if out.Bias != 0 {
			t.Errorf("Bias = %d, want 0 — the two cancel, neither is lost", out.Bias)
		}
		if out.DiceTotal != 6 {
			t.Errorf("DiceTotal = %d, want 6 — a neutral bias reads Primary", out.DiceTotal)
		}
	})

	t.Run("all three origins sum", func(t *testing.T) {
		ledger := match.NewModifierLedger()
		ledger.Add(match.Modifier{Bias: 1, Applies: match.DimActionSpeed, Against: match.ScopeAnyone()})
		out := service.RollCalculator{}.Derive(rules, attempts, service.RollInput{
			SkillName: "Legerity", SkillValue: 3,
			SystemBias: -1,
			Condition:  &action.RollCondition{Bias: 1},
			Ledger:     &ledger,
			Dimension:  match.DimActionSpeed,
		})
		if out.Bias != 1 {
			t.Errorf("Bias = %d, want 1 (master +1, system -1, ledger +1)", out.Bias)
		}
		if out.DiceTotal != 14 {
			t.Errorf("DiceTotal = %d, want 14 — a net advantage reads Secondary", out.DiceTotal)
		}
	})

	t.Run("the ledger is read on the caller's dimension, not on everything it holds", func(t *testing.T) {
		ledger := match.NewModifierLedger()
		ledger.Add(match.Modifier{Amount: 5, Applies: match.DimDodge, Against: match.ScopeAnyone()})
		out := service.RollCalculator{}.Derive(rules, attempts, service.RollInput{
			SkillName: "Legerity", SkillValue: 3, Ledger: &ledger, Dimension: match.DimActionSpeed,
		})
		if out.Modifier != 0 {
			t.Errorf("Modifier = %d, want 0 — a dodge reserve does not move actionSpeed", out.Modifier)
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestDerive_ThreeBiasOrigins -v`
Expected: FAIL — `unknown field SystemBias in struct literal`.

- [ ] **Step 3: Add the two fields and sum three origins**

In `roll_calculator.go`, on `RollInput`, replacing the `Ledger` comment that still carries the
over-generalized invariant:

```go
	// Ledger is character-owned; nil = empty. WHAT each entry modifies is now the entry's own
	// business (match.Modifier.Applies), read against Dimension below — the caller no longer
	// decides by passing nil. An earlier comment here claimed the accumulated difference was
	// "always an actionSpeed adjustment, never a hit adjustment": that was true of the duel
	// reserve and false as a general law. The closed dodge's reserve modifies the dodge.
	Ledger *match.ModifierLedger
	// Dimension is which reserve this roll reads out of the ledger. The zero value reads
	// nothing, which is the right default for a test no reserve applies to.
	Dimension match.Dimension
	// SystemBias is advantage/disadvantage the ENGINE imposes on this one roll — the
	// action→reaction conversion is the first one. It is not the ledger (that is the
	// character's, and lasts until it expires) and not RollCondition (that is the master's).
	// Three origins, three homes, none overwriting another.
	SystemBias int
	AgainstID  *uuid.UUID // whom the roll is against; nil = nobody in particular
```

And in `Derive`:

```go
	bias, modifier := in.SystemBias, 0
	if in.Condition != nil {
		bias += in.Condition.Bias
		modifier += in.Condition.Modifier
	}
	if in.Ledger != nil {
		bias += in.Ledger.TotalBias(in.Dimension, in.AgainstID)
		modifier += in.Ledger.TotalAmount(in.Dimension, in.AgainstID)
	}
```

Then set `Dimension: match.DimActionSpeed` at the two `deriveSpeeds` call sites in
`match_session.go` that pass a ledger, and leave the hit roll in `resolveCharacter` passing no
ledger and no dimension — the duel reserve moves actionSpeed, never the hit.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/domain/match/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/roll_calculator.go \
        internal/domain/match/matchsession/match_session.go \
        internal/domain/match/service/roll_calculator_test.go
git commit -m "feat(match): a bias for one roll only, beside the master's and the ledger's

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4: `ReactionKind` — the type is declared, never inferred

**Files:**
- Create: `internal/domain/match/entity/action/reaction_kind.go`
- Create: `internal/domain/match/entity/action/reaction_kind_test.go`
- Create: `internal/domain/match/entity/action/repel.go`
- Modify: `internal/domain/match/entity/action/action.go`
- Modify: `internal/domain/match/entity/action/bar.go`
- Modify: `internal/domain/match/entity/action/dodge.go`
- Delete: `internal/domain/entity/enum/dodge_category.go`
- Modify: `internal/app/game/action_mapper.go:100-110`

**Interfaces:**
- Consumes: nothing.
- Produces: `action.ReactionKind` with seven values; `ReactionKind.Bars() []Bar`;
  `ReactionKind.IsFree() bool`; `ReactionKind.Displaces() bool`; `ReactionKind.IsValid() bool`;
  `action.ReactionKindFrom(string) (ReactionKind, error)`; `action.Repel{Weapon *enum.WeaponName; RollCheck}`;
  `Action.ReactionKind ReactionKind` and `Action.Repel *Repel` as exported fields.

`Action.Bars()` derives cost from **which components are filled in**. That works for actions —
move, act, or both — and cannot express the catalogue: the three escapes have identical shape
and three different prices, an `Evasive` closed dodge costs nothing while `Bars()` never returns
empty, and repel has no component at all. The missing information is not in the shape; it is in
the player's intent, and intent is declared.

- [ ] **Step 1: Write the failing tests**

```go
// internal/domain/match/entity/action/reaction_kind_test.go
package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

func TestReactionKind_Bars(t *testing.T) {
	tests := []struct {
		kind action.ReactionKind
		want []action.Bar
		why  string
	}{
		{action.ReactRepel, []action.Bar{action.BarAction}, "repel spends the action, never the feet"},
		{action.ReactClosedEscape, []action.Bar{action.BarMove}, "done in the exact instant, without opening the guard — the action comes back"},
		{action.ReactEscape, []action.Bar{action.BarAction, action.BarMove}, "forcing the dodge by displacing costs both"},
		{action.ReactEscapeGuard, []action.Bar{action.BarAction, action.BarMove}, "same price as the standard escape; it only keeps the safety net"},
		{action.ReactClosedDodge, nil, "free — that is what makes it worth the trouble to configure"},
		{action.ReactDodge, nil, "gambling the roll instead of the average costs nothing"},
		{action.ReactNothing, nil, "taking the blow raw costs no bar; it costs HP"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := tt.kind.Bars()
			if len(got) != len(tt.want) {
				t.Fatalf("Bars() = %v, want %v — %s", got, tt.want, tt.why)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("Bars() = %v, want %v — %s", got, tt.want, tt.why)
				}
			}
		})
	}
}

func TestAction_BarsAsksTheReactionKindFirst(t *testing.T) {
	t.Run("a free reaction charges nothing even though it carries a dodge", func(t *testing.T) {
		a := action.NewAction(uuid.New(), nil, uuid.New(), nil, action.ActionSpeed{},
			nil, nil, nil, nil, &action.Dodge{}, nil, nil)
		a.ReactionKind = action.ReactClosedDodge
		if got := a.Bars(); len(got) != 0 {
			t.Fatalf("Bars() = %v, want empty — the kind answers, not the shape", got)
		}
	})

	t.Run("the three escapes have the same shape and different prices", func(t *testing.T) {
		build := func(k action.ReactionKind) *action.Action {
			a := action.NewAction(uuid.New(), nil, uuid.New(), nil, action.ActionSpeed{},
				nil, &action.Move{}, nil, nil, &action.Dodge{}, nil, nil)
			a.ReactionKind = k
			return a
		}
		if got := build(action.ReactClosedEscape).Bars(); len(got) != 1 || got[0] != action.BarMove {
			t.Fatalf("closed escape Bars() = %v, want [move]", got)
		}
		if got := build(action.ReactEscape).Bars(); len(got) != 2 {
			t.Fatalf("standard escape Bars() = %v, want [action move]", got)
		}
	})

	t.Run("an action without a reaction kind keeps deriving from its shape", func(t *testing.T) {
		a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		got := a.Bars()
		if len(got) != 1 || got[0] != action.BarAction {
			t.Fatalf("Bars() = %v, want [action] — scheduled actions are unchanged", got)
		}
	})
}

func TestReactionKindFrom(t *testing.T) {
	if _, err := action.ReactionKindFrom("closedEscape"); err != nil {
		t.Errorf("closedEscape must be accepted: %v", err)
	}
	if _, err := action.ReactionKindFrom("parry"); err == nil {
		t.Error("an unknown kind must be refused at the boundary, not defaulted")
	}
	if _, err := action.ReactionKindFrom(""); err == nil {
		t.Error("an empty kind must be refused — the server never infers cost from shape")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/domain/match/entity/action/ -run 'TestReactionKind|TestAction_Bars' -v`
Expected: FAIL — `undefined: action.ReactRepel`.

- [ ] **Step 3: Write `reaction_kind.go` and `repel.go`**

```go
// internal/domain/match/entity/action/reaction_kind.go
package action

import "fmt"

// ReactionKind is what the target chose to do about an attack.
//
// It is DECLARED by the sender and never inferred from which components the payload filled
// in. The three escapes carry exactly the same shape — a Dodge and a Move — and cost three
// different things; no inspection of the shape can tell them apart, because the information
// that separates them is the player's intent.
//
// The reflex dodge and the default defense are NOT kinds. Nobody sends them: they are what
// the engine applies when nothing arrives. ReactNothing exists precisely because refusing
// them is a send, and the only way to tell "sent nothing" from "sent nothing on purpose" is
// to have received something.
type ReactionKind string

const (
	ReactNothing      ReactionKind = "nothing"      // refuses even the passives — takes the blow raw
	ReactDodge        ReactionKind = "dodge"        // active dodge: gamble the roll instead of the average
	ReactClosedDodge  ReactionKind = "closedDodge"  // closed dodge, with Evasion folded in
	ReactEscape       ReactionKind = "escape"       // standard escape: gives up the default defense
	ReactEscapeGuard  ReactionKind = "escapeGuard"  // defensive escape: keeps the safety net
	ReactClosedEscape ReactionKind = "closedEscape" // closed escape
	ReactRepel        ReactionKind = "repel"        // attacks the attack
)

// Bars is what this reaction costs, by the table in combat-engine.md § Reações.
//
// nil is a real answer, not an omission: the closed variants pay for themselves. Done in the
// exact instant, without opening the guard, they give the action back.
//
// ⚠️ Note the asymmetry with Action.Bars(), which is never empty. That invariant exists
// because the scheduler has to price every action it schedules by some bar — and a reaction is
// NOT scheduled: it never enters the queue and is never chosen by RoundScheduler. Empty is the
// correct answer for a free reaction.
func (k ReactionKind) Bars() []Bar {
	switch k {
	case ReactRepel:
		return []Bar{BarAction}
	case ReactClosedEscape:
		return []Bar{BarMove}
	case ReactEscape, ReactEscapeGuard:
		return []Bar{BarAction, BarMove}
	default:
		return nil
	}
}

// IsFree reports whether this reaction charges no bar — and therefore consumes no pending
// action and rolls at no disadvantage.
func (k ReactionKind) IsFree() bool { return len(k.Bars()) == 0 }

// Displaces reports whether this reaction moves the character. All three escapes do, by
// definition — an escape that does not displace is just a dodge — so the WS boundary refuses
// one that arrives without a Move.
func (k ReactionKind) Displaces() bool {
	switch k {
	case ReactEscape, ReactEscapeGuard, ReactClosedEscape:
		return true
	default:
		return false
	}
}

// KeepsDefault reports whether the default defense still stands behind this reaction when it
// fails.
//
// Escaping gives up the safety net: force the dodge and miss, and the automatic defense does
// not come in — full damage. Only the defensive escape keeps it. Repelling gives it up too:
// you committed the weapon to the incoming blow, you were not also ducking.
func (k ReactionKind) KeepsDefault() bool {
	switch k {
	case ReactEscape, ReactClosedEscape, ReactRepel, ReactNothing:
		return false
	default:
		return true
	}
}

func (k ReactionKind) IsValid() bool {
	switch k {
	case ReactNothing, ReactDodge, ReactClosedDodge,
		ReactEscape, ReactEscapeGuard, ReactClosedEscape, ReactRepel:
		return true
	}
	return false
}

func (k ReactionKind) String() string { return string(k) }

// ReactionKindFrom crosses the string→enum boundary. An unknown or empty kind is an error
// here, where it can still be answered with a WS error: the server never infers what a
// reaction costs from the shape of what arrived.
func ReactionKindFrom(s string) (ReactionKind, error) {
	k := ReactionKind(s)
	if !k.IsValid() {
		return "", fmt.Errorf("reaction kind %q is not in the catalogue", s)
	}
	return k, nil
}
```

```go
// internal/domain/match/entity/action/repel.go
package action

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

// Repel is the component of the hardest reaction in the catalogue: instead of dodging or
// parrying, the character hits the incoming blow so it does not reach them.
//
// It is shaped exactly like Defense — a weapon and a test — because it is the same gesture
// read against a different ladder. The weapon matters twice: it is what the repel is made
// with, and on a near miss (a parry) its defense is what reduces the blow travelling on to
// the next target.
type Repel struct {
	Weapon *enum.WeaponName
	RollCheck
}
```

- [ ] **Step 4: Add the fields and the first gate**

In `action.go`, beside the existing components:

```go
	Defense  *Defense
	Dodge    *Dodge
	Repel    *Repel
	Interact *Interact

	// ReactionKind is set only on a reaction, and it is what decides the cost — never the
	// shape. It is deliberately a field rather than a constructor parameter: NewAction already
	// takes twelve positional arguments and is called from dozens of sites, and growing it
	// buys nothing. The discriminator for "this is a reaction" is still ReactToID.
	ReactionKind ReactionKind
```

In `bar.go`, in front of the existing derivation:

```go
func (a *Action) Bars() []Bar {
	// A reaction answers with its declared kind. This gate comes first because the three
	// escapes are shape-identical and priced differently — no inspection below could tell
	// them apart. It is also the only path that may answer empty.
	if a.ReactionKind != "" {
		return a.ReactionKind.Bars()
	}
	if a.Move == nil {
		return []Bar{BarAction}
	}
	if a.chargesActionBar() {
		return []Bar{BarAction, BarMove}
	}
	return []Bar{BarMove}
}
```

and teach `chargesActionBar` about the new component so a master action carrying a repel is
still priced: add `a.Repel != nil ||` to its condition.

In `dodge.go`, drop the absorbed category:

```go
package action

// Dodge is the roll behind a dodge reaction. WHICH dodge — active, closed, escape, closed
// escape — is ReactionKind's business, on the Action itself.
//
// It used to carry an enum.DodgeCategory of {Evasive, Close, Scape}. That was the same axis as
// ReactionKind and strictly less expressive: Scape alone covered all THREE escapes, which is
// exactly the distinction the cost table needs. Keeping both would be redundant state that can
// disagree with itself, and state like that always does, sooner or later.
type Dodge struct {
	RollCheck
}
```

Delete `internal/domain/entity/enum/dodge_category.go`, and in `action_mapper.go` replace the
passthrough with the roll check alone (the kind is mapped in Task 5):

```go
	var dodge *action.Dodge
	if p.Dodge != nil {
		rc, err := buildRollCheck(p.Dodge.RollCheck)
		if err != nil {
			return nil, err
		}
		dodge = &action.Dodge{}
		if rc != nil {
			dodge.RollCheck = *rc
		}
	}
```

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/domain/match/... ./internal/domain/entity/... ./internal/app/game/ -v`
Expected: PASS. `grep -rn "DodgeCategory" internal/` must come back empty.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/entity/action/ internal/domain/entity/enum/dodge_category.go \
        internal/app/game/action_mapper.go
git commit -m "feat(action): the reaction declares its kind, and the kind is what it costs

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5: The wire — the kind is mandatory, and it travels with `reactToId`

**Files:**
- Modify: `internal/app/game/message.go:162-200`
- Modify: `internal/app/game/action_mapper.go`
- Modify: `internal/app/game/room.go:667-760`
- Test: `internal/app/game/action_mapper_test.go`

**Interfaces:**
- Consumes: `action.ReactionKindFrom`, `action.Repel`, `ReactionKind.Displaces` (Task 4).
- Produces: `ActionPayload.ReactionKind string` (`json:"reactionKind,omitempty"`);
  `RepelPayload{Weapon *string; RollCheck RollCheckPayload}`; `ActionPayload.Repel *RepelPayload`;
  `buildAction` populating both; the paired validation
  `ReactToID != uuid.Nil ⟺ ReactionKind != ""`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/app/game/action_mapper_test.go
func TestBuildAction_Reactions(t *testing.T) {
	actorID := uuid.New()

	t.Run("maps the declared kind onto the action", func(t *testing.T) {
		a, err := buildAction(actorID, ActionPayload{
			ActorID:      actorID,
			ReactToID:    uuid.New(),
			ReactionKind: "repel",
			Repel:        &RepelPayload{RollCheck: RollCheckPayload{SkillName: "Repel"}},
		})
		if err != nil {
			t.Fatalf("buildAction: %v", err)
		}
		if a.ReactionKind != action.ReactRepel {
			t.Errorf("ReactionKind = %q, want repel", a.ReactionKind)
		}
		if a.Repel == nil || a.Repel.SkillName != "Repel" {
			t.Fatal("the repel component did not survive the mapping")
		}
	})

	t.Run("an unknown kind is refused at the boundary", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "parry",
		})
		if err == nil {
			t.Fatal("an unknown kind must not reach the domain")
		}
	})

	t.Run("an escape without a move is refused — an escape that does not displace is a dodge", func(t *testing.T) {
		_, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, ReactToID: uuid.New(), ReactionKind: "closedEscape",
		})
		if err == nil {
			t.Fatal("a displacing reaction with no Move must be refused, never defaulted")
		}
	})

	t.Run("an action carries no reaction kind", func(t *testing.T) {
		a, err := buildAction(actorID, ActionPayload{
			ActorID: actorID, Attack: &AttackPayload{},
		})
		if err != nil {
			t.Fatalf("buildAction: %v", err)
		}
		if a.ReactionKind != "" {
			t.Errorf("ReactionKind = %q, want empty on a plain action", a.ReactionKind)
		}
	})
}
```

> `action_mapper_test.go` is `package game` — an **internal** test package — so it calls
> `buildAction` directly. Do not add an exported seam; the existing tests in that file already
> do this.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/app/game/ -run TestBuildAction_Reactions -v`
Expected: FAIL — `unknown field ReactionKind in struct literal of type game.ActionPayload`.

- [ ] **Step 3: Extend the payload and the mapper**

In `message.go`, on `ActionPayload`:

```go
	// ReactionKind names what the target chose to do, and it is MANDATORY on a reaction. The
	// server never infers the cost from the shape of what arrived — the three escapes are
	// shape-identical and priced differently. Empty on a plain action.
	ReactionKind string        `json:"reactionKind,omitempty"`
	Repel        *RepelPayload `json:"repel,omitempty"`
```

```go
// RepelPayload is the hardest reaction in the catalogue, shaped like the defense: a weapon and
// a test.
type RepelPayload struct {
	Weapon    *string          `json:"weapon,omitempty"`
	RollCheck RollCheckPayload `json:"rollCheck"`
}
```

In `action_mapper.go`, before the `NewAction` call:

```go
	var repel *action.Repel
	if p.Repel != nil {
		rc, err := buildRollCheck(&p.Repel.RollCheck)
		if err != nil {
			return nil, err
		}
		weapon, err := buildWeaponName(p.Repel.Weapon)
		if err != nil {
			return nil, err
		}
		repel = &action.Repel{Weapon: weapon, RollCheck: *rc}
	}

	var kind action.ReactionKind
	if p.ReactionKind != "" {
		kind, err = action.ReactionKindFrom(p.ReactionKind)
		if err != nil {
			return nil, err
		}
		// All three escapes displace by definition; SpeedOn(BarMove) reads Move.FinalSpeed, so
		// one without a Move would charge the move bar zero. A client bug, refused rather than
		// defaulted — the same rule as an unsupported move category.
		if kind.Displaces() && p.Move == nil {
			return nil, fmt.Errorf("reaction %q must carry a move", p.ReactionKind)
		}
	}
```

and after construction:

```go
	a := action.NewAction(
		actorCharID, p.TargetID, p.ReactToID,
		skills, speed,
		feint, move, attack, defense, dodge, nil, interact,
	)
	a.Repel = repel
	a.ReactionKind = kind
	return a, nil
```

In `room.go`, both in the `MsgTypeEnqueueAction` reaction branch and in `MsgTypeAttachReaction`,
add the paired check beside the existing `ReactToID` guard:

```go
		// The two travel together or not at all. reactToId says "this is a reaction"; the kind
		// says what it costs. One without the other is a client that is going to be surprised.
		if (payload.ReactToID != uuid.Nil) != (payload.ReactionKind != "") {
			client.SendMessage(NewErrorMessage("invalid_action",
				"a reaction needs both reactToId and reactionKind; an action needs neither"))
			return
		}
```

and delete the now-dead `payload.Dodge != nil && payload.ReactToID == uuid.Nil` guard at
`room.go:673` — the paired check subsumes it and says more.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/app/game/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/
git commit -m "feat(game): a reaction arrives with its kind, or it does not arrive

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6: Only a target may react, and only through their own character

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go:432-438` (`AttachReaction`)
- Modify: `internal/domain/match/matchsession/error.go`
- Modify: `internal/application/match/attach_reaction.go`
- Modify: `internal/app/game/room.go:952-981` (`handleReaction`)
- Test: `internal/domain/match/matchsession/match_session_test.go`

**Interfaces:**
- Consumes: `charToPlayer` (already in the session), `ReactToID` validation in
  `RoundOrchestrator.AttachReaction` (already there).
- Produces: `MatchSession.AttachReaction(playerUUID uuid.UUID, r *action.Action) (*service.TurnResolution, error)`
  — the signature grows the caller; `ErrReactionActorMismatch`, `ErrReactorNotTargeted`.

`AttachReaction` today takes no player at all: any connected client can attach a reaction in any
character's name. `EnqueueAction` has checked ownership since Phase 2 — this closes the same
hole on the other route, and adds the check the spec asks for on top of it.

- [ ] **Step 1: Write the failing tests**

```go
// internal/domain/match/matchsession/match_session_test.go — inside TestMatchSession_AttachReaction
	t.Run("refuses a reaction sent through someone else's character", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)
		a := makeActionWithSpeed(chars[0], 10)
		a.TargetID = []uuid.UUID{chars[1]}
		s.EnqueueAction(playerA, a) //nolint:errcheck
		opened := mustOpen(t, s)
		act := opened.GetAction()

		// playerA drives chars[0]; chars[1] is not theirs to answer with.
		r := makeReactionTo(chars[1], act.GetID())
		r.ReactionKind = action.ReactDodge
		if _, err := s.AttachReaction(playerA, r); err == nil {
			t.Fatal("a player must not react through a character that is not theirs")
		}
	})

	t.Run("refuses a reaction from someone the action never targeted", func(t *testing.T) {
		playerA, playerB, playerC := uuid.New(), uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB, playerC)
		a := makeActionWithSpeed(chars[0], 10)
		a.TargetID = []uuid.UUID{chars[1]} // C is a bystander
		s.EnqueueAction(playerA, a)        //nolint:errcheck
		opened := mustOpen(t, s)
		act := opened.GetAction()

		r := makeReactionTo(chars[2], act.GetID())
		r.ReactionKind = action.ReactDodge
		if _, err := s.AttachReaction(playerC, r); err == nil {
			t.Fatal("a bystander must not be able to react")
		}
	})

	t.Run("accepts a reaction from a target, through their own character", func(t *testing.T) {
		playerA, playerB := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)
		a := makeActionWithSpeed(chars[0], 10)
		a.TargetID = []uuid.UUID{chars[1]}
		s.EnqueueAction(playerA, a) //nolint:errcheck
		opened := mustOpen(t, s)
		act := opened.GetAction()

		r := makeReactionTo(chars[1], act.GetID())
		r.ReactionKind = action.ReactDodge
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("a target reacting through their own character: %v", err)
		}
	})
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/domain/match/matchsession/ -run TestMatchSession_AttachReaction -v`
Expected: FAIL — `too many arguments in call to s.AttachReaction`.

- [ ] **Step 3: Add the two checks**

```go
// AttachReaction validates that the caller may answer this attack, then attaches the reaction
// to the open turn and re-resolves it.
//
// Two axes, exactly as EnqueueAction: authorization is a per-PLAYER question and combat is a
// per-CHARACTER one, bridged by charToPlayer. On top of that, a reaction has a third
// constraint an action does not — only someone the attack is AIMED AT may answer it.
func (s *MatchSession) AttachReaction(playerUUID uuid.UUID, r *action.Action) (*service.TurnResolution, error) {
	owner, ok := s.charToPlayer[r.GetActorID().String()]
	if !ok || owner != playerUUID {
		return nil, ErrReactionActorMismatch
	}
	t := s.activeRound.CurrentTurn()
	if t == nil {
		return nil, service.ErrNoCurrentTurn
	}
	act := t.GetAction()
	if !slices.Contains(act.TargetID, r.GetActorID()) {
		return nil, ErrReactorNotTargeted
	}

	s.rollActionDice(r)
	if err := s.roundOrch.AttachReaction(s.activeRound, r); err != nil {
		return nil, err
	}
	return s.ResolveTurn(s.activeRound.CurrentTurn()), nil
}
```

with, in `error.go`:

```go
var (
	// ErrReactionActorMismatch means the caller tried to react through a character that is
	// not theirs. The same rule EnqueueAction enforces, on the other route.
	ErrReactionActorMismatch = errors.New("the reacting character does not belong to this player")
	// ErrReactorNotTargeted means the caller was not aimed at. Bystanders watch.
	ErrReactorNotTargeted = errors.New("only a target of the open action may react to it")
)
```

Thread `callerUUID` through `AttachReactionUC.Execute` — it already receives it and drops it on
the floor — and leave `handleReaction` in `room.go` unchanged apart from the compile: it already
passes `client.userUUID`.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/matchsession/ internal/application/match/attach_reaction.go
git commit -m "fix(match): a reaction needs a target, and a character of one's own

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 7: What a reaction costs, and what it consumes

**Files:**
- Modify: `internal/domain/match/service/round_scheduler.go`
- Modify: `internal/domain/match/matchsession/match_session.go` (`AttachReaction`, `deriveSpeeds`)
- Test: `internal/domain/match/service/round_scheduler_test.go` — reuse the `attackAt`,
  `moveAt`, `chargeAt` and `newFakeBars()` helpers already in that file
- Test: `internal/domain/match/matchsession/match_session_test.go`

**Interfaces:**
- Consumes: `ReactionKind.Bars`, `ReactionKind.IsFree` (Task 4); `RollInput.SystemBias` (Task 3);
  `BarEconomy`, `RoundScheduler` (Phase 3).
- Produces: `RoundScheduler.BestPendingFor(in ScheduleInput, actorID uuid.UUID, bar action.Bar) *action.Action`;
  `MatchSession.deriveSpeeds(a *action.Action, systemBias int)` — the existing signature grows a
  parameter, and `EnqueueAction` passes `0`.

Four decisions, all in `combat-engine.md` § "O custo da reação na economia de barra":

| | |
|---|---|
| **Which speed** | a reaction that charges a bar goes through `deriveSpeeds` and records the speed **it** rolled, on each bar it charges; a free reaction derives nothing and records nothing |
| **The gate** | **does not apply.** A reaction is never refused for lack of balance — it debits and the character starts the next round further behind |
| **When** | at **attach**, not at open. That is where the collision is already computed, and opening a reaction is a narration event — narration must not move a number |
| **The pending action** | leaves `activeQueue`: the one with the best key for that character on that bar |

- [ ] **Step 1: Write the failing tests**

```go
// internal/domain/match/service/round_scheduler_test.go
func TestRoundScheduler_BestPendingFor(t *testing.T) {
	t.Run("picks the pending action that would have opened first for that character", func(t *testing.T) {
		actor, other := uuid.New(), uuid.New()
		slow := attackAt(actor, 8)
		fast := attackAt(actor, 20)
		theirs := attackAt(other, 30)
		q := action.NewActionPriorityQueue(&[]*action.Action{slow, theirs, fast})
		r := round.NewRound(enum.Race)
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: newFakeBars()}

		got := service.RoundScheduler{}.BestPendingFor(in, actor, action.BarAction)
		if got == nil || got.GetID() != fast.GetID() {
			t.Fatal("the reaction consumes the moment that was about to be spent, not the leftovers")
		}
	})

	t.Run("answers nil when that character has nothing pending on that bar", func(t *testing.T) {
		actor := uuid.New()
		q := action.NewActionPriorityQueue(nil)
		r := round.NewRound(enum.Race)
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: newFakeBars()}
		if got := service.RoundScheduler{}.BestPendingFor(in, actor, action.BarAction); got != nil {
			t.Fatal("nothing pending means the reaction simply becomes the action")
		}
	})
}
```

```go
// internal/domain/match/matchsession/match_session_test.go
func TestMatchSession_ReactionCost(t *testing.T) {
	setup := func(t *testing.T) (*matchsession.MatchSession, []uuid.UUID, uuid.UUID, uuid.UUID, *action.Action) {
		t.Helper()
		playerA, playerB := uuid.New(), uuid.New()
		s, chars := sessionWithParticipants(playerA, playerB)
		s.SetRoundMode(enum.Race)
		a := makeActionWithSpeed(chars[0], 10)
		a.TargetID = []uuid.UUID{chars[1]}
		s.EnqueueAction(playerA, a) //nolint:errcheck
		opened := mustOpen(t, s)
		act := opened.GetAction()
		return s, chars, playerA, playerB, &act
	}

	t.Run("a repel debits the action bar at attach, not at open", func(t *testing.T) {
		s, chars, _, playerB, act := setup(t)
		r := makeReactionTo(chars[1], act.GetID())
		r.ReactionKind = action.ReactRepel
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		_, speeds := s.BarState(chars[1], action.BarAction)
		if len(speeds) != 1 {
			t.Fatalf("action-bar speeds = %v, want exactly one — the repel charged on arrival", speeds)
		}
		_, moveSpeeds := s.BarState(chars[1], action.BarMove)
		if len(moveSpeeds) != 0 {
			t.Fatalf("move-bar speeds = %v, want none — a repel spends the action, not the feet", moveSpeeds)
		}
	})

	t.Run("a closed dodge charges nothing", func(t *testing.T) {
		s, chars, _, playerB, act := setup(t)
		r := makeReactionTo(chars[1], act.GetID())
		r.ReactionKind = action.ReactClosedDodge
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if _, speeds := s.BarState(chars[1], action.BarAction); len(speeds) != 0 {
			t.Fatalf("action-bar speeds = %v, want none — the closed dodge pays for itself", speeds)
		}
	})

	t.Run("a reaction is never refused for lack of balance", func(t *testing.T) {
		s, chars, _, playerB, act := setup(t)
		status, err := s.GetCharacterStatus(chars[1])
		if err != nil {
			t.Fatalf("GetCharacterStatus: %v", err)
		}
		status.ActionBar.Balance = -100 // deep in debt

		r := makeReactionTo(chars[1], act.GetID())
		r.ReactionKind = action.ReactRepel
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("a reaction must never be denied for lack of balance: %v", err)
		}
	})

	t.Run("an active reaction consumes the pending action and rolls at disadvantage", func(t *testing.T) {
		s, chars, _, playerB, act := setup(t)
		pending := makeActionWithSpeed(chars[1], 30)
		s.EnqueueAction(playerB, pending) //nolint:errcheck

		r := makeReactionTo(chars[1], act.GetID())
		r.ReactionKind = action.ReactRepel
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		for _, p := range s.PendingActions() {
			if p.GetID() == pending.GetID() {
				t.Fatal("reacting actively consumes the action that was in the queue")
			}
		}
	})

	t.Run("with nothing pending the reaction simply becomes the action", func(t *testing.T) {
		s, chars, _, playerB, act := setup(t)
		r := makeReactionTo(chars[1], act.GetID())
		r.ReactionKind = action.ReactRepel
		if _, err := s.AttachReaction(playerB, r); err != nil {
			t.Fatalf("AttachReaction: %v", err)
		}
		if len(s.PendingActions()) != 0 {
			t.Fatal("there was nothing to consume")
		}
	})
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/domain/match/... -run 'BestPendingFor|ReactionCost' -v`
Expected: FAIL — `undefined: BestPendingFor`; the bars stay empty.

- [ ] **Step 3: Add `BestPendingFor`**

```go
// BestPendingFor returns the pending action that would open first for one character on one
// bar, or nil when they have none there.
//
// It exists for the reaction: reacting actively consumes the action you had queued, and the
// one it consumes is the moment that was about to be spent — the best key. Scoring reuses
// keyOnBar, so this invents no criterion the round did not already have.
//
// The gate is deliberately NOT consulted. An action that could not have opened this round is
// still the one whose moment the reaction takes; refusing to consume it would let a character
// react AND keep a queued action, which is the trade the Disadvantage is paying for.
func (rs RoundScheduler) BestPendingFor(in ScheduleInput, actorID uuid.UUID, bar action.Bar) *action.Action {
	if in.Queue == nil {
		return nil
	}
	var best *action.Action
	bestKey := 0.0
	for _, a := range in.Queue.All() {
		if a.GetActorID() != actorID || !slices.Contains(a.Bars(), bar) {
			continue
		}
		key, _ := rs.keyOnBar(in, a, bar)
		if best == nil || key > bestKey {
			best, bestKey = a, key
		}
	}
	return best
}
```

> ⚠️ `keyOnBar` returns `(0, false)` for an action the gate refuses. Read only the key here and
> ignore the boolean — that is the point of the paragraph above.

- [ ] **Step 4: Charge, consume and derive in `AttachReaction`**

```go
func (s *MatchSession) AttachReaction(playerUUID uuid.UUID, r *action.Action) (*service.TurnResolution, error) {
	owner, ok := s.charToPlayer[r.GetActorID().String()]
	if !ok || owner != playerUUID {
		return nil, ErrReactionActorMismatch
	}
	t := s.activeRound.CurrentTurn()
	if t == nil {
		return nil, service.ErrNoCurrentTurn
	}
	act := t.GetAction()
	if !slices.Contains(act.TargetID, r.GetActorID()) {
		return nil, ErrReactorNotTargeted
	}

	s.rollActionDice(r)

	// A free reaction derives nothing, records nothing and consumes nothing. That IS the
	// discount: done in the exact instant, without opening the guard, it gives the action back.
	if !r.ReactionKind.IsFree() {
		consumed := s.consumePendingFor(r)
		// Swapping what you were going to do costs Disadvantage — the engine rolls again and
		// keeps the worse of the two speeds. It is a MODE of reading the dice, never an
		// Amount: RollAttempts already holds both sets and the bias only picks which one.
		// With nothing queued there was no swap, so there is no penalty.
		systemBias := 0
		if consumed {
			systemBias = -1
		}
		s.deriveSpeeds(r, systemBias)
		s.chargeReactionBars(r)
	}

	if err := s.roundOrch.AttachReaction(s.activeRound, r); err != nil {
		return nil, err
	}
	return s.ResolveTurn(s.activeRound.CurrentTurn()), nil
}

// consumePendingFor pulls this character's about-to-open action off the queue, once per bar the
// reaction charges, and reports whether anything was taken.
//
// A combined action sits on both bars and is counted once — it leaves on the first bar that
// finds it and is simply not there for the second.
func (s *MatchSession) consumePendingFor(r *action.Action) bool {
	consumed := false
	for _, bar := range r.ReactionKind.Bars() {
		victim := s.scheduler.BestPendingFor(s.scheduleInput(), r.GetActorID(), bar)
		if victim == nil {
			continue
		}
		s.activeQueue.ExtractByID(victim.GetID())
		consumed = true
	}
	return consumed
}

// chargeReactionBars debits the reaction's own speed on every bar its kind charges.
//
// At ATTACH, not at open, and the difference matters: an action records on open because one
// that never reaches the price rolls into the next round untouched, so Speeds has to mean "acted
// for real". A reaction has nowhere to roll to — it lives inside the turn it answered — and
// opening it is a narration event. Narration must not move a number. The consequence lines up
// with Phase 5: a reaction attached and never opened has already paid, which is exactly what
// the close-turn dialogue assumes when it says such a reaction "enters the calculation but
// loses its moment to narrate".
//
// Race-only, for the same reason recordActed is: settleBars skips a bar that never priced, so a
// speed recorded under Free would be charged by nothing and reset by nothing — and would then
// make IsEligible read the character as having already acted the moment the master switches to
// Race, denying them their first action of the disputed round.
func (s *MatchSession) chargeReactionBars(r *action.Action) {
	if s.activeRound.GetMode() != enum.Race {
		return
	}
	status, ok := s.statuses[r.GetActorID()]
	if !ok {
		return
	}
	for _, bar := range r.ReactionKind.Bars() {
		status.BarFor(bar).RecordSpeed(r.SpeedOn(bar))
	}
}
```

Give `deriveSpeeds` the parameter and thread the bias into both `Derive` calls
(`SystemBias: systemBias`, `Dimension: match.DimActionSpeed`); `EnqueueAction` passes `0`.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/domain/match/... -v && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/service/round_scheduler.go internal/domain/match/matchsession/ \
        internal/domain/match/service/round_scheduler_test.go
git commit -m "feat(match): a reaction pays on arrival, and takes the moment it swapped for

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 8: Opening a reaction is a first-class operation

**Files:**
- Modify: `internal/domain/match/entity/turn/turn.go`
- Modify: `internal/domain/match/matchsession/match_session.go`
- Create: `internal/application/match/open_reaction.go`
- Modify: `internal/domain/match/matchsession/error.go` — `ErrTurnAlreadyClosed`, `ErrReactionNotFound`
- Modify: `internal/app/game/message.go`, `internal/app/game/room.go`, `internal/app/game/hub.go`, `internal/app/game/handler.go`
- Test: `internal/domain/match/entity/turn/turn_test.go` (exists, `package turn_test`), `internal/app/game/handler_test.go`

**Interfaces:**
- Consumes: `AttachReaction` (Task 6/7).
- Produces: `Turn.OpenReaction(id uuid.UUID) bool`; `Turn.OpenedReactionIDs() []uuid.UUID`;
  `MatchSession.OpenReaction(reactionID uuid.UUID) (*service.TurnResolution, error)`;
  `appmatch.OpenReactionUC` with
  `Execute(ctx, session, callerUUID uuid.UUID, reactionID uuid.UUID) (*OpenReactionResult, error)`;
  `MsgTypeOpenReaction MessageType = "open_reaction"`; `OpenReactionPayload{ReactionID uuid.UUID}`.

This is not deferred to Phase 5, and the reason is that the chain **is** the master opening one
reaction at a time. Without the operation there is no opening order, and without an opening
order the phase cannot produce its own done-criterion. "Opening" is passing the microphone; the
calculation is the side effect.

- [ ] **Step 1: Write the failing tests**

```go
// internal/domain/match/entity/turn/turn_test.go
package turn_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

func TestTurn_OpenReaction(t *testing.T) {
	base := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	tn := turn.NewTurn(*base)

	first := action.NewAction(uuid.New(), nil, base.GetID(), nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	second := action.NewAction(uuid.New(), nil, base.GetID(), nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
	tn.AddReaction(first)
	tn.AddReaction(second)

	t.Run("records the order the master opened, not the order they arrived", func(t *testing.T) {
		if !tn.OpenReaction(second.GetID()) {
			t.Fatal("OpenReaction must find an attached reaction")
		}
		if !tn.OpenReaction(first.GetID()) {
			t.Fatal("OpenReaction must find an attached reaction")
		}
		got := tn.OpenedReactionIDs()
		if len(got) != 2 || got[0] != second.GetID() || got[1] != first.GetID() {
			t.Fatal("the opening order is the master's, and it is what the chain walks")
		}
	})

	t.Run("opening the same reaction twice does not duplicate it", func(t *testing.T) {
		tn.OpenReaction(first.GetID()) //nolint:errcheck
		if len(tn.OpenedReactionIDs()) != 2 {
			t.Fatal("a reaction is opened once; re-opening is a no-op, not a second slot")
		}
	})

	t.Run("refuses an id that is not attached to this turn", func(t *testing.T) {
		if tn.OpenReaction(uuid.New()) {
			t.Fatal("only a reaction attached to this turn can be opened")
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/match/entity/turn/ -v`
Expected: FAIL — `tn.OpenReaction undefined`.

- [ ] **Step 3: Record the opening order on the turn**

```go
type Turn struct {
	id            uuid.UUID
	action        action.Action
	reactions     []action.Action
	// openedReactions is the order the MASTER opened the reactions, which is not the order
	// they arrived — reactions land whenever their players send them, and the master picks.
	// The chain walks this order, and walking it in reverse produces a different outcome; that
	// is game power, not a matter of pacing.
	openedReactions []uuid.UUID
	masterActions   []action.MasterAction
	finishedAt      *time.Time
}

// OpenReaction records that the master passed the microphone to one of the attached
// reactions, and reports whether it found it. Idempotent: opening the same one twice does not
// give it a second slot in the order.
func (t *Turn) OpenReaction(id uuid.UUID) bool {
	found := false
	for i := range t.reactions {
		if t.reactions[i].GetID() == id {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	if slices.Contains(t.openedReactions, id) {
		return true
	}
	t.openedReactions = append(t.openedReactions, id)
	return true
}

// OpenedReactionIDs returns a copy of the opening order.
func (t *Turn) OpenedReactionIDs() []uuid.UUID {
	out := make([]uuid.UUID, len(t.openedReactions))
	copy(out, t.openedReactions)
	return out
}
```

- [ ] **Step 4: Add the session method and the use case**

```go
// OpenReaction passes the microphone to one reaction on the open turn and re-resolves it.
//
// Opening is table conduct — "now it is this person's turn to narrate". The recomputation is
// the side effect, and it recomputes rather than re-rolling: every die fell at attach.
//
// It does NOT charge anything. The bars were debited when the reaction arrived (see
// chargeReactionBars) precisely so that narrating cannot move a number.
func (s *MatchSession) OpenReaction(reactionID uuid.UUID) (*service.TurnResolution, error) {
	t := s.activeRound.CurrentTurn()
	if t == nil {
		return nil, service.ErrNoCurrentTurn
	}
	if t.GetFinishedAt() != nil {
		return nil, ErrTurnAlreadyClosed
	}
	if !t.OpenReaction(reactionID) {
		return nil, ErrReactionNotFound
	}
	return s.ResolveTurn(t), nil
}
```

```go
// internal/application/match/open_reaction.go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type OpenReactionResult struct {
	Resolution *service.TurnResolution
}

type IOpenReaction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, callerUUID, reactionID uuid.UUID) (*OpenReactionResult, error)
}

type OpenReactionUC struct{}

func NewOpenReactionUC() *OpenReactionUC { return &OpenReactionUC{} }

// Execute opens one reaction on the open turn. Master-only — the caller check lives in the
// delivery layer, exactly as it does for open_next_action.
func (uc *OpenReactionUC) Execute(
	ctx context.Context, session *matchsession.MatchSession, callerUUID, reactionID uuid.UUID,
) (*OpenReactionResult, error) {
	resolution, err := session.OpenReaction(reactionID)
	if err != nil {
		return nil, err
	}
	return &OpenReactionResult{Resolution: resolution}, nil
}
```

- [ ] **Step 5: Add the WS route**

In `message.go`:

```go
	MsgTypeOpenReaction MessageType = "open_reaction"
```

```go
// OpenReactionPayload names which attached reaction the master is giving the floor to. The
// order is theirs, and it changes the result.
type OpenReactionPayload struct {
	ReactionID uuid.UUID `json:"reactionId"`
}
```

In `room.go`, following the `MsgTypeOpenNextAction` shape — master-only, **write lock across
`Execute`** (the session mutates), and the projection stays master-only until Phase 5:

```go
	case MsgTypeOpenReaction:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload OpenReactionPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid open_reaction payload"))
			return
		}
		r.mu.RLock()
		session := r.session
		r.mu.RUnlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		r.mu.Lock()
		result, err := r.openReactionUC.Execute(context.Background(), session, client.userUUID, payload.ReactionID)
		turnID := currentTurnID(session)
		r.mu.Unlock()
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		// Whose turn it is to narrate is public; the calculation is not, until Phase 5.
		out := NewServerMessage(MsgTypeReactionOpened, ReactionOpenedPayload{
			TurnID: turnID, ReactionID: payload.ReactionID,
		})
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()
		r.sendToMaster(NewServerMessage(
			MsgTypeResolutionUpdate,
			newResolutionUpdatedPayload(turnID, result.Resolution),
		))
```

with `MsgTypeReactionOpened MessageType = "reaction_opened"` and
`ReactionOpenedPayload{TurnID, ReactionID uuid.UUID}` beside it, and `openReactionUC` threaded
through `NewRoom`, `hub.go` and `handler.go` the way `attachReactionUC` already is.

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/... -v && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/domain/match/entity/turn/ internal/domain/match/matchsession/ \
        internal/application/match/open_reaction.go internal/app/game/
git commit -m "feat(match): the master passes the microphone to a reaction, and the order is theirs

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 9: One reaction against one attack

**Files:**
- Create: `internal/domain/match/service/reaction_collision.go`
- Create: `internal/domain/match/service/reaction_collision_test.go`

**Interfaces:**
- Consumes: `action.ReactionKind` and `KeepsDefault` (Task 4); `RollCalculator.Derive`,
  `RollInput.Dimension`, `match.ModifierLedger` (Tasks 1 and 3); `ClimbLadder` (Phase 2).
- Produces:
```go
type ReactionInput struct {
	Kind        action.ReactionKind
	Reaction    *action.Action        // nil = nothing was sent; the passives apply
	Target      *csSheet.CharacterSheet
	Ledger      *match.ModifierLedger // the target's; nil = empty
	AttackerID  uuid.UUID
	HitTotal    int                   // the CD every defensive test is read against
	Rules       match.MatchRules
}

type ReactionOutcome struct {
	Kind        action.ReactionKind
	Dodge       RollOutcome
	Defense     RollOutcome
	Evasion     RollOutcome // closed variants only
	Repel       RollOutcome // repel only
	Ladder      LadderOutcome
	Avoided     bool        // the blow did not land on this target
	Defended    bool
	StopsAttack bool        // a successful repel — nothing travels on
	Payouts     []match.Modifier // what this reaction wrote into the target's ledger
}

func ResolveReaction(in ReactionInput) ReactionOutcome
```

This is the per-kind branch, written as a pure function so every row of the catalogue can be
tested on its own before the chain walks them.

**The rules, kind by kind:**

| Kind | Test | If it works | If it fails |
|---|---|---|---|
| *(nothing sent)* | reflex dodge, passive (`Reflex + PassiveValue`) | avoided | default defense, CD one step lower |
| `nothing` | none — refuses the passives | — | takes the blow raw |
| `dodge` | `Reflex` **rolled** instead of passive | avoided | default defense |
| `closedDodge` | `Reflex` and `Evasion` both rolled, the **worse** counts | avoided; bonus `|Reflex − Evasion|` on `DimDodge`, `ScopeAllBut(attacker)`, `LifetimeNextTurn` | default defense |
| `escape` | `Reflex` rolled | avoided | **no default defense** — full damage |
| `escapeGuard` | `Reflex` rolled | avoided | default defense |
| `closedEscape` | as `closedDodge`, and it displaces | avoided, same bonus | **no default defense** |
| `repel` | `Repel` rolled, read against the ladder | see the ladder table | see the ladder table |

Ties favour the defender everywhere: `>=` on both the dodge and the defense, matching how
`ClimbLadder` reads `margin == 0` as cleared.

- [ ] **Step 1: Write the failing tests**

```go
// internal/domain/match/service/reaction_collision_test.go
package service_test

func reactionInput(t *testing.T, kind action.ReactionKind, r *action.Action, hit int) service.ReactionInput {
	t.Helper()
	ledger := match.NewModifierLedger()
	return service.ReactionInput{
		Kind: kind, Reaction: r, Target: plainSheet(t), Ledger: &ledger,
		AttackerID: uuid.New(), HitTotal: hit, Rules: match.NewDefaultMatchRules(),
	}
}

// reactionWith builds a reaction whose dice have already fallen, which is the state the
// session hands the resolver.
func reactionWith(kind action.ReactionKind, dodge, evasion, repel []int) *action.Action {
	a := action.NewAction(uuid.New(), nil, uuid.New(), nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil)
	a.ReactionKind = kind
	if dodge != nil {
		a.Dodge = &action.Dodge{RollCheck: action.RollCheck{
			SkillName: enum.Reflex.String(), Attempts: action.RollAttempts{Primary: dodge},
		}}
	}
	if evasion != nil {
		a.Skills = []action.Skill{{SkillName: enum.Evasion.String(), RollCheck: action.RollCheck{
			SkillName: enum.Evasion.String(), Attempts: action.RollAttempts{Primary: evasion},
		}}}
	}
	if repel != nil {
		a.Repel = &action.Repel{RollCheck: action.RollCheck{
			SkillName: enum.Repel.String(), Attempts: action.RollAttempts{Primary: repel},
		}}
	}
	return a
}

func TestResolveReaction(t *testing.T) {
	// A fresh sheet has every skill at 0, so a passive test is 0 + 11 = 11 and a rolled test
	// is just the dice.
	t.Run("nothing sent: the passives apply, in order", func(t *testing.T) {
		out := service.ResolveReaction(reactionInput(t, "", nil, 9))
		if !out.Avoided {
			t.Fatal("a passive reflex dodge of 11 stops a hit of 9")
		}
		out = service.ResolveReaction(reactionInput(t, "", nil, 15))
		if out.Avoided {
			t.Fatal("11 does not reach 15")
		}
		// The defense is one ladder step easier: CD 15 − 10 = 5, and 11 clears it.
		if !out.Defended {
			t.Fatal("the default defense comes in when the dodge fails")
		}
	})

	t.Run("nothing: refuses even the passives", func(t *testing.T) {
		out := service.ResolveReaction(reactionInput(t, action.ReactNothing, reactionWith(action.ReactNothing, nil, nil, nil), 9))
		if out.Avoided || out.Defended {
			t.Fatal("sending 'nothing' takes the blow raw — that is the whole point")
		}
	})

	t.Run("dodge: gambles the roll instead of the average", func(t *testing.T) {
		r := reactionWith(action.ReactDodge, []int{10, 8}, nil, nil) // 18, above the passive 11
		out := service.ResolveReaction(reactionInput(t, action.ReactDodge, r, 15))
		if !out.Avoided {
			t.Fatal("a rolled 18 clears a hit of 15 where the passive 11 would not")
		}
	})

	t.Run("closed dodge: the worse of the two counts, and the gap is the reserve", func(t *testing.T) {
		// Reflex 18, Evasion 13 → the dodge is 13, and 5 is banked against third parties.
		r := reactionWith(action.ReactClosedDodge, []int{10, 8}, []int{9, 4}, nil)
		out := service.ResolveReaction(reactionInput(t, action.ReactClosedDodge, r, 12))
		if out.Dodge.Total != 13 {
			t.Fatalf("dodge total = %d, want 13 — Evasion enters as Disadvantage, it does not add", out.Dodge.Total)
		}
		if !out.Avoided {
			t.Fatal("13 clears a hit of 12")
		}
		if len(out.Payouts) != 1 || out.Payouts[0].Amount != 5 {
			t.Fatalf("payouts = %+v, want a single +5 reserve", out.Payouts)
		}
		p := out.Payouts[0]
		if p.Applies != match.DimDodge {
			t.Error("the closed dodge's reserve is dodge, not actionSpeed — that law is the duel's, not the system's")
		}
		if p.ExpiresAt != match.LifetimeNextTurn {
			t.Error("the reserve is kept for whoever comes next turn")
		}
	})

	t.Run("escape abandons the safety net; escapeGuard keeps it", func(t *testing.T) {
		fail := []int{1, 2} // 3, nowhere near
		out := service.ResolveReaction(reactionInput(t, action.ReactEscape,
			reactionWith(action.ReactEscape, fail, nil, nil), 15))
		if out.Defended {
			t.Fatal("forcing the dodge by displacing gives up the automatic defense")
		}
		out = service.ResolveReaction(reactionInput(t, action.ReactEscapeGuard,
			reactionWith(action.ReactEscapeGuard, fail, nil, nil), 15))
		if !out.Defended {
			t.Fatal("the defensive escape is the one that keeps the net")
		}
	})

	t.Run("repel: the four rungs", func(t *testing.T) {
		cases := []struct {
			name     string
			dice     []int
			hit      int
			rung     service.LadderRung
			avoided  bool
			payout   int
		}{
			{"cleared by a full step", []int{10, 10}, 10, service.RungGreatSuccess, true, 10},
			{"cleared", []int{8, 6}, 12, service.RungSuccess, true, 0},
			{"parried", []int{5, 3}, 12, service.RungNearMiss, true, -4},
			{"missed by a full step", []int{1, 1}, 20, service.RungFailure, false, 0},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				r := reactionWith(action.ReactRepel, nil, nil, tc.dice)
				out := service.ResolveReaction(reactionInput(t, action.ReactRepel, r, tc.hit))
				if out.Ladder.Rung != tc.rung {
					t.Fatalf("rung = %q, want %q", out.Ladder.Rung, tc.rung)
				}
				if out.Avoided != tc.avoided {
					t.Fatalf("Avoided = %v, want %v — parrying is zero damage, not reduced", out.Avoided, tc.avoided)
				}
				if tc.rung == service.RungFailure && out.Defended {
					t.Fatal("repelling abandons the passives too: you committed the weapon to the incoming blow")
				}
				if tc.payout == 0 {
					if len(out.Payouts) != 0 {
						t.Fatalf("payouts = %+v, want none", out.Payouts)
					}
					return
				}
				if len(out.Payouts) != 1 || out.Payouts[0].Amount != tc.payout {
					t.Fatalf("payouts = %+v, want a single %d", out.Payouts, tc.payout)
				}
				if out.Payouts[0].Applies != match.DimActionSpeed {
					t.Error("the duel reserve moves actionSpeed — that is what makes two fighters speed up against each other")
				}
			})
		}
	})

	t.Run("the repel bonus is targeted and the parry penalty is general", func(t *testing.T) {
		attacker := uuid.New()
		in := reactionInput(t, action.ReactRepel, reactionWith(action.ReactRepel, nil, nil, []int{10, 10}), 10)
		in.AttackerID = attacker
		bonus := service.ResolveReaction(in).Payouts[0]
		third := uuid.New()
		if !bonus.Against.AppliesTo(&attacker) || bonus.Against.AppliesTo(&third) {
			t.Fatal("you learned to read THAT opponent — the bonus does not generalize")
		}

		in = reactionInput(t, action.ReactRepel, reactionWith(action.ReactRepel, nil, nil, []int{5, 3}), 12)
		in.AttackerID = attacker
		penalty := service.ResolveReaction(in).Payouts[0]
		if !penalty.Against.AppliesTo(&third) {
			t.Fatal("you were left off balance, and anyone can take advantage")
		}
	})
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/domain/match/service/ -run TestResolveReaction -v`
Expected: FAIL — `undefined: service.ResolveReaction`.

- [ ] **Step 3: Write `reaction_collision.go`**

Implement `ResolveReaction` following the table above. The shape to respect:

```go
// ResolveReaction is one target's answer to one attack, as a pure function.
//
// Pure in the strong sense Phase 2 established: every die it reads has already fallen and lives
// in the reaction's RollCheck.Attempts. It derives, it never rolls — which is what lets the
// master recompute on every new reaction and on every edit without re-rolling a player's die.
//
// A nil Reaction means nothing was sent, and the engine applies the defaults: reflex dodge,
// then — only if that fails — the defense, one ladder step easier. Sending ReactNothing is a
// different thing entirely: it refuses even those.
func ResolveReaction(in ReactionInput) ReactionOutcome {
	calc := RollCalculator{}
	out := ReactionOutcome{Kind: in.Kind}

	if in.Kind == action.ReactNothing {
		// Refusing the safety net is a choice, and the engine honours it exactly.
		return out
	}

	if in.Kind == action.ReactRepel {
		return resolveRepel(in, calc)
	}

	out.Dodge = deriveDodge(in, calc)      // passive when Reaction is nil, rolled otherwise
	if bonus, ok := closedReserve(in, calc, &out); ok {
		out.Payouts = append(out.Payouts, bonus)
	}
	out.Avoided = out.Dodge.Total >= in.HitTotal
	if out.Avoided {
		return out
	}
	if !in.Kind.KeepsDefault() {
		// Escaping and repelling give up the automatic defense. Miss, and the blow lands whole.
		return out
	}
	out.Defense = calc.Derive(in.Rules, action.RollAttempts{}, RollInput{
		SkillName:  enum.Defense.String(),
		SkillValue: skillValueOf(in.Target, enum.Defense.String()),
		Passive:    true,
	})
	out.Defended = out.Defense.Total >= in.HitTotal-in.Rules.LadderStep
	return out
}
```

Two details the tests pin down and the implementation must not blur:

- **The closed reserve is `|Reflex − Evasion|`, and the dodge is the smaller of the two.**
  Evasion does not add to the dodge; it enters the Disadvantage logic — roll both, the worse
  counts — and the gap is the dodge that was not spent. Bank it as
  `Modifier{Amount: gap, Applies: DimDodge, Source: SourceSystem, Against: ScopeAllBut(in.AttackerID), ExpiresAt: LifetimeNextTurn, Reason: "closed dodge reserve"}`.
- **The ladder is read with `ClimbLadder(repelTotal − in.HitTotal, in.Rules.LadderStep)`**, and
  `Avoided` is true on every rung but `RungFailure` — including the near miss, because parrying
  is **zero damage, not reduced damage**. The price of a parry is the penalty, not a scratch.
- **A near miss also sets `Defended = true`, and only the two clearing rungs set `StopsAttack`.**
  The chain reads those two flags and nothing else: a parry is the "defended" row of the chain
  table, so the blow travels on reduced by the repelling weapon's defense, while a cleared
  repel ends the attack outright. Getting this wrong is invisible in this task's tests and
  wrong in Task 10's.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/domain/match/service/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/reaction_collision.go \
        internal/domain/match/service/reaction_collision_test.go
git commit -m "feat(match): the whole reaction catalogue, one collision at a time

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 10: The chain — the residual attack walks the targets

**Files:**
- Create: `internal/domain/match/service/attack_chain.go`
- Create: `internal/domain/match/service/attack_chain_test.go`
- Modify: `internal/domain/match/entity/action/attack.go`
- Modify: `internal/domain/match/service/turn_resolver.go`
- Modify: `internal/domain/match/matchsession/match_session.go` (`applyResolution` writes payouts)

**Interfaces:**
- Consumes: `ResolveReaction` (Task 9); `Turn.OpenedReactionIDs` (Task 8); `RawDamage`,
  `ApplicableDefense`, `EffectiveDamage`, `WeaponDice` (Phase 2).
- Produces: `action.AttackSpread` (`SpreadSequential`, `SpreadSimultaneous`) with
  `Attack.Spread AttackSpread`; `service.CharacterResult` gains `ReactionKind`, `Ladder`,
  `Payouts`, `AttackStopped`; `TurnResolution.Payouts []CharacterPayout`.

- [ ] **Step 1: Write the failing test**

The chain has **two** testable surfaces, and they need different tools:

- `ChainState.Reduce` is pure arithmetic and is tested with explicit numbers. It has to be,
  because **every weapon in the catalogue carries defense 0 and armour does not exist**, so the
  two reducing rows are numerically inert when driven through the real catalogue. Testing them
  through the resolver would assert `12 → 12` and prove nothing.
- The **order** effect is tested through `TurnResolver.Resolve`, where it is real: a successful
  repel stops the attack, so whoever is walked *after* it takes nothing and whoever is walked
  *before* it takes the blow.

```go
// internal/domain/match/service/attack_chain_test.go
package service_test

func TestChainState_Reduce(t *testing.T) {
	start := service.ChainState{Residual: 12}

	t.Run("a dodge passes the full attack on — dodging does not spend the blow", func(t *testing.T) {
		got := start.Reduce(service.ReactionOutcome{Avoided: true}, 3, 5)
		if got.Residual != 12 || got.Stopped {
			t.Fatalf("got %+v, want the blow untouched", got)
		}
	})

	t.Run("a successful repel stops the attack dead", func(t *testing.T) {
		got := start.Reduce(service.ReactionOutcome{Avoided: true, StopsAttack: true}, 3, 5)
		if !got.Stopped || got.Residual != 0 {
			t.Fatalf("got %+v, want a stopped chain", got)
		}
	})

	t.Run("once stopped it stays stopped, whatever the next target does", func(t *testing.T) {
		stopped := service.ChainState{Stopped: true}
		got := stopped.Reduce(service.ReactionOutcome{}, 3, 5)
		if !got.Stopped || got.Residual != 0 {
			t.Fatalf("got %+v — stopping is not cancelling, but it is not undone either", got)
		}
	})

	t.Run("a defended blow is reduced by the weapon that defended it", func(t *testing.T) {
		got := start.Reduce(service.ReactionOutcome{Avoided: true, Defended: true}, 3, 5)
		if got.Residual != 9 {
			t.Fatalf("residual = %d, want 9 — this is the ONLY place Weapon.defense has a job", got.Residual)
		}
	})

	t.Run("a parry is the defended row: zero damage to them, reduced for the next", func(t *testing.T) {
		// A repel near miss avoids the damage AND counts as having defended.
		parry := service.ReactionOutcome{Avoided: true, Defended: true,
			Ladder: service.LadderOutcome{Rung: service.RungNearMiss, Difference: 4}}
		if got := start.Reduce(parry, 3, 5); got.Residual != 9 {
			t.Fatalf("residual = %d, want 9", got.Residual)
		}
	})

	t.Run("a blow that lands is reduced by the hit target's armour", func(t *testing.T) {
		got := start.Reduce(service.ReactionOutcome{}, 3, 5)
		if got.Residual != 7 {
			t.Fatalf("residual = %d, want 7", got.Residual)
		}
	})

	t.Run("the residual never goes negative", func(t *testing.T) {
		if got := (service.ChainState{Residual: 2}).Reduce(service.ReactionOutcome{}, 0, 5); got.Residual != 0 {
			t.Fatalf("residual = %d, want 0", got.Residual)
		}
	})
}

// areaTurn builds one attack against several targets with the dice already fallen, attaches the
// reactions their owners sent, and opens them in the order given. That order is the master's,
// and it is what the chain walks.
func areaTurn(
	t *testing.T, actorID uuid.UUID, targets []uuid.UUID,
	hitDice, damageDice []int, weapon *enum.WeaponName,
	reactions map[int]*action.Action, openOrder []int,
) *turn.Turn {
	t.Helper()
	atk := &action.Attack{
		Weapon: weapon,
		Hit: action.RollCheck{
			SkillName: enum.Accuracy.String(),
			Attempts:  action.RollAttempts{Primary: hitDice},
		},
		Damage: action.RollCheck{Attempts: action.RollAttempts{Primary: damageDice}},
	}
	a := action.NewAction(actorID, targets, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, atk, nil, nil, nil, nil)
	tn := turn.NewTurn(*a)
	opened := tn.GetAction()
	for _, i := range openOrder {
		r := reactions[i]
		r.ReactToID = opened.GetID()
		tn.AddReaction(r)
	}
	for _, i := range openOrder {
		if !tn.OpenReaction(reactions[i].GetID()) {
			t.Fatalf("reaction for target %d was not attached", i)
		}
	}
	return tn
}

func TestChain_OpeningOrderChangesTheOutcome(t *testing.T) {
	// Plain sheets: every skill 0, so a passive test is 0 + 11 = 11 and a rolled one is the
	// dice alone.
	//
	//   hit    [6, 4]   → 10
	//   damage [7, 3]   → 7 + 3 + 2 (Sword's flat damage) = 12 raw
	//   repel  [7, 4]   → 11, margin +1 over the CD of 10 → RungSuccess → stops the attack
	//
	// A: repels and stops it.  B: sends "nothing" and refuses even the passives.
	// C: sends no answer at all, so the passive reflex dodge of 11 clears the hit of 10.
	sword := enum.Sword
	actorID := uuid.New()
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	targets := []uuid.UUID{a, b, c}

	build := func(t *testing.T, openOrder []int) *service.TurnResolution {
		t.Helper()
		repel := action.NewAction(a, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		repel.ReactionKind = action.ReactRepel
		repel.Repel = &action.Repel{RollCheck: action.RollCheck{
			SkillName: enum.Repel.String(), Attempts: action.RollAttempts{Primary: []int{7, 4}},
		}}
		nothing := action.NewAction(b, nil, uuid.Nil, nil, action.ActionSpeed{}, nil, nil, nil, nil, nil, nil, nil)
		nothing.ReactionKind = action.ReactNothing

		tn := areaTurn(t, actorID, targets, []int{6, 4}, []int{7, 3}, &sword,
			map[int]*action.Action{0: repel, 1: nothing}, openOrder)

		sheets := map[uuid.UUID]*csSheet.CharacterSheet{
			actorID: plainSheet(t), a: plainSheet(t), b: plainSheet(t), c: plainSheet(t),
		}
		return service.TurnResolver{}.Resolve(service.ResolveInput{
			Turn:    tn,
			Sheets:  sheets,
			Targets: charTargets{chars: map[uuid.UUID]bool{a: true, b: true, c: true}},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})
	}

	damageFor := func(res *service.TurnResolution, id uuid.UUID) int {
		for _, cr := range res.CharacterResults {
			if cr.TargetID == id {
				return cr.EffectiveDamage
			}
		}
		return -1
	}

	t.Run("the repel opened first spares the one who refused the passives", func(t *testing.T) {
		res := build(t, []int{0, 1}) // A then B
		if got := damageFor(res, b); got != 0 {
			t.Fatalf("B took %d, want 0 — the attack was stopped before reaching them", got)
		}
		if got := damageFor(res, c); got != 0 {
			t.Fatalf("C took %d, want 0", got)
		}
	})

	t.Run("the repel opened second arrives too late for them", func(t *testing.T) {
		res := build(t, []int{1, 0}) // B then A
		if got := damageFor(res, b); got != 12 {
			t.Fatalf("B took %d, want 12 — refusing the passives takes the blow raw", got)
		}
		if got := damageFor(res, c); got != 0 {
			t.Fatalf("C took %d, want 0 — the attack was stopped, and their passive dodge cleared it anyway", got)
		}
	})

	t.Run("stopping is not cancelling: a later reaction still resolves", func(t *testing.T) {
		res := build(t, []int{0, 1})
		var found bool
		for _, cr := range res.CharacterResults {
			if cr.TargetID == b {
				found = true
				if cr.ReactionKind != string(action.ReactNothing) {
					t.Errorf("B's answer = %q, want it recorded even though it could not be hit", cr.ReactionKind)
				}
			}
		}
		if !found {
			t.Fatal("a target whose reaction was wasted mechanically still narrates — it must appear")
		}
	})

	t.Run("a simultaneous attack does not diminish", func(t *testing.T) {
		// Reserved axis, unit-tested only: nothing sets SpreadSimultaneous today.
		start := service.ChainState{Residual: 12}
		got := start.ReduceSpread(action.SpreadSimultaneous, service.ReactionOutcome{}, 3, 5)
		if got.Residual != 12 {
			t.Fatalf("residual = %d, want 12 — everyone takes the same blow", got.Residual)
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestChain_ResidualAttack -v`
Expected: FAIL.

- [ ] **Step 3: Add the spread axis to `Attack`**

```go
// AttackSpread is how an attack reaches several targets.
//
// SpreadSequential travels through them: what leaves one target is what enters the next, so
// the order the master opens the reactions changes the outcome. SpreadSimultaneous hits
// everyone at once and does NOT diminish — the master still opens one at a time, because that
// is the table gesture, but only the narration is sequential; the arithmetic is not.
//
// Reserved, not exercised: this will be a configuration of the ability type, and special
// abilities do not exist until post-MVP. The axis is here now because retrofitting it later
// would mean threading an `if` through a chain already written to assume one shape.
type AttackSpread string

const (
	SpreadSequential   AttackSpread = "" // the zero value: today's only reachable behaviour
	SpreadSimultaneous AttackSpread = "simultaneous"
)
```

- [ ] **Step 4: Write the chain and plug it into the resolver**

```go
// ChainState is what one resolution leaves for the next.
//
// The collision with several targets is NOT f(action, reactions[]). It is a walk:
//
//	ataque₀ → resolve(alvo A) → ataque₁ → resolve(alvo B) → ataque₂ → …
//
// Residual is how much of the blow is left. Stopped means a repel ended it — and note that
// stopping is not cancelling: the reactions after it still resolve and their owners still
// narrate. They simply cannot be hit any more.
type ChainState struct {
	Residual int
	Stopped  bool
}

// Reduce applies one target's outcome to the blow travelling onward.
//
//	dodged            → unchanged: dodging does not spend the blow
//	repelled          → stopped
//	defended          → minus the defense of the weapon they defended with
//	hit               → minus the hit target's armour
//
// ⚠️ There is NO rigid rule here — combat-engine.md is explicit that this is contextual and the
// master may override at any point. What lives in code is the DEFAULT per reaction type. The
// override surface is Phase 5's (SystemData); do not build one here.
//
// ⚠️ Armour does not exist in this codebase — there is no armour entity and no sheet field —
// so the hit row currently subtracts zero. The row is encoded because the shape is what
// matters, exactly as ApplicableDefense encodes the damage-type rows it cannot yet read. Do
// not invent an armour model to fill it.
func (c ChainState) Reduce(out ReactionOutcome, defenseWeaponBonus, armour int) ChainState {
	if c.Stopped {
		return ChainState{Stopped: true}
	}
	if out.StopsAttack {
		return ChainState{Stopped: true}
	}
	// The defended row is checked BEFORE the avoided one, because a parry is both: a repel
	// near miss takes zero damage AND counts as having defended. Reading Avoided first would
	// pass the blow on whole and lose the only job Weapon.defense has.
	if out.Defended {
		return ChainState{Residual: floorZero(c.Residual - defenseWeaponBonus)}
	}
	if out.Avoided {
		return c
	}
	return ChainState{Residual: floorZero(c.Residual - armour)}
}

// ReduceSpread is Reduce with the attack's spread taken into account. A simultaneous attack
// does not diminish — everyone takes the same — while the master still opens one target at a
// time so each can narrate. Reserved: nothing sets SpreadSimultaneous until abilities exist.
func (c ChainState) ReduceSpread(spread action.AttackSpread, out ReactionOutcome, defenseWeaponBonus, armour int) ChainState {
	if spread == action.SpreadSimultaneous {
		if out.StopsAttack {
			// A repel still protects the one who made it; it just does not shield the rest.
			return c
		}
		return c
	}
	return c.Reduce(out, defenseWeaponBonus, armour)
}

func floorZero(v int) int {
	if v < 0 {
		return 0
	}
	return v
}
```

In `TurnResolver.Resolve`, replace the flat per-target loop with the walk:

1. Build the target order: every opened reaction, in `Turn.OpenedReactionIDs()` order, then the
   remaining `Action.TargetID` entries in their own order.
2. For each, call `ResolveReaction` with the attack's hit total as the CD, then compute the
   damage from the chain's current residual, then `Reduce`.
3. Collect the payouts on the resolution so the session can write them at turn close.

And in `applyResolution`, write the payouts into the targets' ledgers **in the same place the
damage is applied** — once, at turn close, for the same reason: everything before it is a dry
run, and a bonus granted on every recomputation would multiply.

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./internal/domain/match/... -v && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/domain/match/
git commit -m "feat(match): the attack that leaves one target is the one that enters the next

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 11: `resolution_updated` carries the reactions

**Files:**
- Modify: `internal/app/game/message.go:286-353`
- Test: `internal/app/game/resolution_payload_test.go`

**Interfaces:**
- Consumes: `service.TurnResolution` with the reaction fields (Task 10).
- Produces: `ResolutionUpdatedPayload.Targets[].Reaction *ReactionResultPayload`.

Still **master-only**. Broadcasting to the table and projecting per recipient is Phase 5, and
anticipating it here would leak the calculation the master is entitled to hold until the turn
closes. The payload stays a slice of the resolution, not the whole of it.

- [ ] **Step 1: Write the failing test**

```go
// internal/app/game/resolution_payload_test.go
func TestResolutionPayload_CarriesTheReaction(t *testing.T) {
	targetID := uuid.New()
	res := &service.TurnResolution{
		CharacterResults: []service.CharacterResult{{
			TargetID:        targetID,
			ReactionKind:    string(action.ReactRepel),
			Ladder:          service.LadderOutcome{Rung: service.RungNearMiss, Margin: -4, Difference: 4},
			Dodged:          true,
			RawDamage:       12,
			EffectiveDamage: 0,
		}},
	}
	p := game.NewResolutionUpdatedPayloadForTest(uuid.New(), res)
	if len(p.Targets) != 1 || p.Targets[0].Reaction == nil {
		t.Fatal("the master has to see what the target answered with")
	}
	got := p.Targets[0].Reaction
	if got.Kind != "repel" || got.Rung != "near_miss" || got.Difference != 4 {
		t.Fatalf("reaction payload = %+v, want the kind, the rung and the difference", got)
	}
	if p.Targets[0].ProjectedDamage != 0 {
		t.Error("a parry is zero damage, not reduced damage")
	}
}
```

Add the seam beside `newResolutionUpdatedPayload` if the file does not already have one:

```go
// NewResolutionUpdatedPayloadForTest exposes the projection to the package's external tests.
func NewResolutionUpdatedPayloadForTest(turnID uuid.UUID, res *service.TurnResolution) ResolutionUpdatedPayload {
	return newResolutionUpdatedPayload(turnID, res)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/app/game/ -run TestResolutionPayload_CarriesTheReaction -v`
Expected: FAIL — `unknown field Reaction`.

- [ ] **Step 3: Extend the payload**

```go
// ReactionResultPayload is what one target answered with, as the master reads it.
//
// The rung and the difference travel because they are the master's whole decision surface on a
// repel: whether to let the attack continue past it, and how big the reserve it just created
// was. The individual dice do not — the reaction's own roll is a field-visibility question, and
// per-recipient projection is Phase 5.
type ReactionResultPayload struct {
	Kind       string `json:"kind"`
	Total      int    `json:"total"`
	Rung       string `json:"rung,omitempty"`
	Margin     int    `json:"margin,omitempty"`
	Difference int    `json:"difference,omitempty"`
	StopsAttack bool  `json:"stopsAttack"`
}
```

with `Reaction *ReactionResultPayload \`json:"reaction,omitempty"\`` on
`CharacterResultPayload`, populated from the `CharacterResult` fields Task 10 added.

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/app/game/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/message.go internal/app/game/resolution_payload_test.go
git commit -m "feat(game): the master's projection says what each target answered with

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 12: The done-criterion, over a real WebSocket

**Files:**
- Create: `internal/app/game/reaction_chain_e2e_test.go`

**Interfaces:**
- Consumes: everything above.
- Produces: nothing — this task is the phase's proof.

`combat_e2e_test.go` already lives in `package game_test` and carries everything this needs:
`combatSessionUC`, `recordingStatusWriter`, `scriptedFaces` (the variant that **fails loudly on
overrun** instead of repeating its last face) and a fixture that stands a real `httptest` server
with a real `Room` in front of a session the test owns. Extend that shape to four characters.

> ⚠️ Before scripting a single face, re-read `combat-engine.md` § "Um gotcha de teste:
> `rollActionDice` rola quase tudo, mas não é uniforme". How many dice fall depends on the round
> regime and the move category — and now on the reaction kind too. For this test, in `Free`:
>
> | What arrives | Faces consumed |
> |---|---|
> | the attack (`Attack.Hit` + Sword damage `D10 + D4`) | 2 + 2 = **4** — `Speed` is passive in `Free` and rolls nothing |
> | A's repel (`Repel.RollCheck`) | **2** |
> | B's `nothing` | **0** — it carries no check at all |
> | C's silence | **0** — the passives take the average and must never touch the source |
>
> A phantom roll drains the script and shifts every number after it. Assert
> `!source.overran` at the end of every run.

- [ ] **Step 1: Build the fixture**

Extend the existing one rather than writing a second: one attacker driven by its own player, and
three targets driven by three more, so authorization is exercised for real.

```go
// areaFixture is combatFixture with three targets instead of one. Each character belongs to a
// different player, so the ownership check in AttachReaction is genuinely crossed.
type areaFixture struct {
	server     *httptest.Server
	session    *matchsession.MatchSession
	source     *scriptedFaces
	master     *wsConn
	attacker   *wsConn // and the three below, one connection per player
	pA, pB, pC *wsConn
	attackerID uuid.UUID // sheet UUIDs
	a, b, c    uuid.UUID
	sheets     map[uuid.UUID]*csSheet.CharacterSheet
}

// newAreaFixture stands the server, seeds the session with four participants and installs the
// scripted source. faces is handed to the session through SetRollSource, which exists for
// exactly this.
func newAreaFixture(t *testing.T, faces []int) *areaFixture

// send marshals a client message and writes it, failing the test on any error.
func (c *wsConn) send(t *testing.T, mt game.MessageType, payload any)

// awaitTurnOpened blocks until turn_opened arrives and returns the turn ID and the action ID
// the reactions must point at.
func (f *areaFixture) awaitTurnOpened(t *testing.T) (turnID, actionID uuid.UUID)

// attachReaction sends one reaction and returns the master's resulting projection, which is
// how the test learns the reaction's own ID for the later open_reaction.
func (f *areaFixture) attachReaction(t *testing.T, c *wsConn, p game.ActionPayload) uuid.UUID

// projectedDamage reads the master's most recent resolution_updated for turnID and returns
// the projected damage per target.
func (f *areaFixture) projectedDamage(t *testing.T, turnID uuid.UUID) map[uuid.UUID]int

// sawAny reports whether this connection ever received a message of that type.
func (c *wsConn) sawAny(mt game.MessageType) bool
```

- [ ] **Step 2: Write the failing test**

```go
// TestE2E_AreaAttackWithThreeTargetsReactingDifferently is the phase's done-criterion.
//
//	target A sends an ACTIVE reaction (repel) and clears the CD
//	target B sends "nothing"        — refusing even the passives
//	target C sends no answer at all — the engine applies the defaults
//
// The same three answers and the same scripted faces run twice, with the master opening A→B and
// then B→A. What each target takes differs between the two runs, and that difference IS the
// phase's objective.
//
// Faces, in the order rollActionDice consumes them (Free round, so actionSpeed is passive):
//
//	6, 4      → the hit: 0 + 10, which the passive reflex dodge of 11 clears
//	7, 3      → the Sword's damage: 7 + 3 + 2 = 12 raw
//	7, 4      → A's repel: 11, margin +1 over the CD of 10 → RungSuccess → stops the attack
func TestE2E_AreaAttackWithThreeTargetsReactingDifferently(t *testing.T) {
	faces := []int{6, 4, 7, 3, 7, 4}

	// reverse decides whether the master opens the repel first or second.
	run := func(t *testing.T, repelFirst bool) map[uuid.UUID]int {
		t.Helper()
		f := newAreaFixture(t, faces)
		defer f.server.Close()

		sword := "Sword"
		f.attacker.send(t, game.MsgTypeEnqueueAction, game.ActionPayload{
			ActorID:  f.attackerID,
			TargetID: []uuid.UUID{f.a, f.b, f.c},
			Attack: &game.AttackPayload{
				Weapon: &sword,
				Hit:    game.RollCheckPayload{SkillName: "Accuracy"},
				Damage: game.RollCheckPayload{},
			},
		})
		f.master.send(t, game.MsgTypeOpenNextAction, struct{}{})
		turnID, actionID := f.awaitTurnOpened(t)

		repelID := f.attachReaction(t, f.pA, game.ActionPayload{
			ActorID: f.a, ReactToID: actionID, ReactionKind: "repel",
			Repel: &game.RepelPayload{RollCheck: game.RollCheckPayload{SkillName: "Repel"}},
		})
		nothingID := f.attachReaction(t, f.pB, game.ActionPayload{
			ActorID: f.b, ReactToID: actionID, ReactionKind: "nothing",
		})
		// C sends nothing at all. That silence is the third answer, and it is not the same
		// thing as B's "nothing".

		order := []uuid.UUID{nothingID, repelID}
		if repelFirst {
			order = []uuid.UUID{repelID, nothingID}
		}
		for _, id := range order {
			f.master.send(t, game.MsgTypeOpenReaction, game.OpenReactionPayload{ReactionID: id})
		}

		got := f.projectedDamage(t, turnID)

		if f.source.overran {
			t.Fatal("the scripted source ran out: a roll happened that this test did not account for")
		}
		// The calculation belongs to the master until the turn closes.
		for _, c := range []*wsConn{f.attacker, f.pA, f.pB, f.pC} {
			if c.sawAny(game.MsgTypeResolutionUpdate) {
				t.Fatal("resolution_updated must stay master-only until Phase 5")
			}
		}
		return got
	}

	t.Run("the repel opened first stops the attack before it reaches the others", func(t *testing.T) {
		f := newAreaFixture(t, faces)
		defer f.server.Close()
		got := run(t, true)
		if got[f.b] != 0 {
			t.Fatalf("B took %d, want 0 — the attack was stopped before reaching them", got[f.b])
		}
	})

	t.Run("opened second, the repel arrives too late for whoever was walked first", func(t *testing.T) {
		f := newAreaFixture(t, faces)
		defer f.server.Close()
		got := run(t, false)
		if got[f.b] != 12 {
			t.Fatalf("B took %d, want 12 — refusing the passives takes the blow raw", got[f.b])
		}
	})

	t.Run("C, who answered nothing at all, is covered by the passives either way", func(t *testing.T) {
		f := newAreaFixture(t, faces)
		defer f.server.Close()
		for _, repelFirst := range []bool{true, false} {
			if got := run(t, repelFirst)[f.c]; got != 0 {
				t.Fatalf("C took %d, want 0 — the reflex dodge is free and automatic", got)
			}
		}
	})
}
```

> ⚠️ **Do not assert that B takes more than C *because* B refused the defense.** Against an
> *armed* attack a bare-handed defense subtracts nothing — `ApplicableDefense` has no damage
> types to read — so the passive **defense** is numerically inert and the two would tie. What
> separates them here is the passive **dodge**, which is why the scripted hit lands below C's
> passive 11. The distinction is real; it just comes from the dodge, not from the defense.

- [ ] **Step 3: Run it to verify it fails**

Run: `go test ./internal/app/game/ -run TestE2E_AreaAttack -v`
Expected: FAIL.

- [ ] **Step 4: Make it pass, then run under the race detector**

Run: `go test ./internal/app/game/ -run TestE2E -race -v`
Expected: PASS. `attach_reaction` and `open_reaction` both mutate the session, so both must hold
the write lock across `Execute` — this is the test that proves it, the way
`TestE2E_AttackAgainstACharacterProducesDamage` proved it for Phase 2.

- [ ] **Step 5: Run the whole suite**

Run: `go test ./internal/... && go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/reaction_chain_e2e_test.go
git commit -m "test(game): three targets, three answers, and an order that changes the outcome

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 13: Record what Phase 4 fixed, and update the map

**Files:**
- Modify: `docs/dev/match/combat-engine.md`
- Modify: `docs/documentation-map.yaml`
- Modify: `docs/superpowers/specs/2026-08-16-combat-engine-design.md`

**Interfaces:**
- Consumes: the finished implementation.
- Produces: the source of truth for Phase 5.

Phase 3 set the precedent: the phase's section in `combat-engine.md` becomes the source of
truth from then on, *including the points that diverged from the plan*. Follow it.

- [ ] **Step 1: Add § "O que a Fase 4 fixou no motor"**

Cover, at minimum: where `ReactionKind` lives and why `Bars()` may now be empty; that a reaction
is charged at attach and never gated; how "next turn" is implemented by demotion; the chain's
default table and the fact that master override is Phase 5's; that armour reduces by zero
because armour does not exist; and — as its own subsection, the way Phase 3 documented its
`rollActionDice` gotcha — **how many dice each reaction kind consumes**, because the next
person writing a scripted test will need exactly that and will otherwise find it by
instrumenting the source.

- [ ] **Step 2: Register the new code paths in `documentation-map.yaml`**

`reaction_kind.go`, `reaction_collision.go`, `attack_chain.go`, `open_reaction.go`,
`reaction_chain_e2e_test.go` — each with the docs it affects, following the entries already
there for the Phase 2 and 3 files.

- [ ] **Step 3: Mark Phase 4 done in the spec**

Add the ✅ banner above § "Fase 4" pointing at the new `combat-engine.md` section, exactly as
Phase 3's does. Leave the ⛔ note about NPC rostering intact under Phase 5, where it now belongs.

- [ ] **Step 4: Commit**

```bash
git add docs/
git commit -m "docs(match): record what phase 4 fixed in the engine

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Verification before the PR

Per the project's delivery discipline, in this order:

1. **`go test ./internal/... -race`** and **`go vet -tags=integration ./internal/...`** clean.
2. **Prove it running.** The front has no reaction UI until Phase 6, so a browser flow is not
   available for this phase. The substitute is the automated evidence the project's rule
   explicitly allows: `TestE2E_AreaAttackWithThreeTargetsReactingDifferently` drives the real
   `Room` over a real WebSocket, through the real use cases, with scripted dice. **Say so in the
   PR, and say that no browser verification was possible and why.**
3. **`./dev-checkout.sh feat/combat-engine-phase-4`** from `System_X_System_Project/`, because
   step 2 leaves the whole master-facing surface unverified by hand. Point at what to look at:
   `attach_reaction` with each of the seven kinds, and `open_reaction` in two different orders.
4. Open the PR.

---

## Out of scope — do not build these

- **NPC rostering.** Phase 4 runs on player characters. The slice lands before Phase 5.
- **Postures.** The closed-escape discount applies unconditionally until they exist.
- **Explicit turn close, and per-recipient projection.** Phase 5. `resolution_updated` stays
  master-only.
- **Master override of the chain**, and the `SystemData` audit table. Phase 5.
- **Any front work.** Phase 6.
- **Armour, damage types, Nen.** The chain encodes the rows; the entities do not exist.
- **The simultaneous spread.** The axis is reserved and unit-tested; nothing reaches it.
- **A reaction timer clock.** The number lives in `MatchRules.ReactionTimer` (it already does);
  with the default off, closing the turn *is* the expiry. The visible countdown is Phase 6.
- **Initiative.** `action.Initiative` stays orphaned and `ChangeMode` keeps ignoring its
  parameter, with its TODO untouched.
