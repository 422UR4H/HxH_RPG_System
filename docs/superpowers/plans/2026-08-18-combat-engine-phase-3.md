# Combat Engine — Phase 3 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** The turn becomes dynamic. Every action enters the queue with a real rolled speed, the
two resource bars charge a frozen round price, the order of play falls out of the bar
arithmetic instead of arriving at zero, and the round closes by itself when nothing pending can
still pay.

**Architecture:** All the arithmetic lives in one stateless domain service
(`service.BarEconomy`) and is consumed by one selector (`service.RoundScheduler`). The
per-character state it reads already exists from Phase 1 — `match.CharacterStatus.ActionBar` /
`.MoveBar` (`ResourceBar{Balance, Speeds}`) — where `Balance` is the carry-over and `Speeds` is
the ordered list of speeds that **actually acted** this round. Speeds are derived once, at
enqueue, from the dice Phase 2 already drops there; they are *recorded* into a bar only when
the master opens that action. Because the ordering key changes whenever a character sends
another action (the round average moves), the priority queue **stops being a heap** and becomes
a list scanned at selection time — a heap cannot re-key an item already inserted.

**Tech Stack:** Go 1.23, standard `testing` (table-driven, `t.Run`), external test packages,
PostgreSQL via pgx, gorilla-style WebSocket delivery in `internal/app/game/`.

**Spec:** `docs/superpowers/specs/2026-08-16-combat-engine-design.md` — §5 "Fase 3" is the
binding scope. The **rules** are in `docs/dev/match/combat-engine.md` §§ "A chave de
prioridade", "Fechamento do round, por barra", "O preço congela na primeira ação aberta",
"Quem pode agir — são DOIS porteiros, não um", "Quando o round fecha", "Como o carry-over entra
no round seguinte", "Ações compostas", "Escala das duas barras". Player-facing narrative and
the canonical example: `docs/game/combate/barra-de-acao.md`. Outstanding structural gaps:
`docs/dev/match/flows/05-lacunas.md` §§2, 5, 7.

**Branch:** `feat/combat-engine-phase-3`, from `main`.

> ✅ **Already done, in this branch's first commit:** the `enum.Velocity` → `Quickness`
> rename — the enum and its serialized value, 12 sites in `character_class_factory.go`, the
> gateway model and queries (`velocity_exp` → `quickness_exp`, with a goose migration), and the
> three `velocity` keys in the React sheet (its own branch, its own PR, cross-linked).
> `action.Velocity`, the movement vector, was not touched. Nothing below depends on
> `Quickness` — Phase 3 uses `Legerity`, `Accelerate` and `Brake` — so this is context, not a
> dependency.

---

## Global Constraints

- **Go 1.23**, module `github.com/422UR4H/HxH_RPG_System`. No test frameworks — standard
  `testing` only, table-driven with `t.Run()`, external test packages (`package foo_test`).
- **NEVER remove a TODO comment.** They are intentional markers left by the repo owner.
  Specifically: `RoundOrchestrator.ChangeMode`'s `// TODO: create and finish Initiative to
  continue here` stays exactly where it is; this phase adds a sibling method beside it.
- **Layering:** `entity ← domain ← app`. `service` imports `action`, `round`, `turn`;
  **`action` must never import `service`**.
- **Domain services are stateless structs.** `service.BarEconomy{}`, `service.RoundScheduler{}`,
  `service.RoundOrchestrator{}` and `service.TurnResolver{}` must all keep working as zero
  values. Dependencies travel as parameters, never as fields.
- **`MatchSession` has no lock of its own.** `room.go`'s `r.mu` is the only serialization.
  Every new call into the session inherits the obligation to hold it, **write-locked** whenever
  the session mutates — which now includes every path that records a speed or freezes a price.
- **Wire format is camelCase** on both sides, via manual struct tags.
- **The master never re-rolls a player's die.** Speeds are *derived* from attempts that fell
  once at enqueue. Nothing in this phase may call `RollCalculator.Roll` twice for one check.
- **A passive test must not consume dice.** `Shift` and `Free`-mode actionSpeed take the set's
  average; the session must **skip rolling** for them, otherwise a scripted `RollSource` in a
  test is silently drained and every downstream number shifts.
- **Free mode keeps today's behaviour.** No price, no average, no carry-over, no auto-close.
  The Free economy has no written rule and is explicitly a later slice (spec §5).
- **Only `Dash` and `Shift` are accepted move categories.** `Back`, `Roll`, `Slide`, `Jump`,
  `FlatJump` answer a WS error. Never map them by analogy (spec §5).
- Commits include `Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>`.
- After every task touching `internal/`: `go vet -tags=integration ./internal/...`.

---

## The rules, restated as arithmetic

Everything below is per **character**, per **bar**, per **round**. `carry` is
`ResourceBar.Balance` at the start of the round; `acted` is `ResourceBar.Speeds`, the ordered
speeds of that character's actions on that bar that have **already been opened** this round.

```
mean(acted)                                     // exact, float64; 0 when acted is empty
key(a, bar) = carry + mean(acted ++ [speed(a, bar)]) − len(acted) × price   // ORDER ONLY
balance()   = carry + mean(acted)                    − len(acted) × price   // the leftover
eligible(a, bar) = len(acted) == 0 ?  carry + speed(a, bar) >= price
                                   :  balance()             >= price
close()     = len(acted) == 0 ?  min(carry + price, price)
                              :  min(balance(), price)
price(bar)  = the smallest speed among the pending actions on that bar, frozen the first
              time an action on that bar is selected, and never moved again this round

// An action that uses BOTH bars — Move and Attack in the same Action — is scheduled at the
// slower half and must clear both gates:
key(a)      = min(key(a, BarAction), key(a, BarMove))
eligible(a) = eligible(a, BarAction) && eligible(a, BarMove)
```

**Integers in, exact fractions out.** The inputs are whole numbers — a rolled speed is a skill
value plus dice, a price is one of those speeds. Everything *derived* — the mean, the key, the
balance, the carry — is `float64` and keeps its fraction. Rounding would be a policy the rules
never asked for, and the error would compound across rounds through the carry.

Read against the canonical example (`barra-de-acao.md`; carry 0, price 11):

| | acted before | gate | key | after opening |
|---|---|---|---|---|
| p2, 1ª (rolls 23) | `[]` | `0+23 ≥ 11` ✔ | `23 − 0×11` = **23** | acted `[23]` |
| p1, 1ª (rolls 20) | `[]` | `0+20 ≥ 11` ✔ | **20** | acted `[20]` |
| p3, 1ª (rolls 11) | `[]` | `0+11 ≥ 11` ✔ | **11** | acted `[11]` |
| p2, 2ª (rolls 17) | `[23]` | leftover `23−11 = 12 ≥ 11` ✔ | `mean(23,17) − 1×11` = `20−11` = **9** | acted `[23,17]` |

Order: **p2 → p1 → p3 → p2**. Closing balances: p1 `20−11` = **+9**, p3 `11−11` = **0**,
p2 `20−22` = **−2**. All three under the ceiling of `+11`.

⚠️ **The two gates are not the key.** p2's second action has key **9**, below the price of 11,
and it happens anyway — the right to act was granted *before* the roll (leftover 12) and a bad
roll only makes it cost more afterwards. Using the key as the gate loses that action and breaks
the phase's own done-criterion. This is written out in `combat-engine.md` § "Quem pode agir —
são DOIS porteiros, não um"; re-read it before touching Task 5.

---

## Decisions taken while planning (technical form — not in the spec)

Recorded so a later phase can read this plan as history, and so the implementer never has to
invent one of these.

1. **A combined action stays ONE action.** `Move` and `Attack` live in the same `Action`; the
   master opens it once and it is one turn. The domain model was built for exactly one action,
   with the movement inside it — there is no split, no second queue entry, no dependency edge
   between halves, and therefore no collision with the `actions` primary key.
2. **A combined action is scheduled at its SLOWER half and must clear BOTH gates.**
   `key = min(keyOnActionBar, keyOnMoveBar)` — the higher key acts first, so `min` is the
   slower one. It charges both bars, so it must be able to pay both: an investida whose action
   bar cannot reach the price sits the whole round out, attack included. The alternative would
   let a character with a spent action bar attack riding on their move bar, which voids the
   economy. Recorded in `combat-engine.md` § Ações compostas.
3. **The mean keeps its fraction — `float64`.** Inputs are whole numbers (a rolled speed is a
   skill value plus dice; a price is one of those speeds); everything derived — mean, key,
   balance, carry — is `float64`. Rounding would be a policy the rules never asked for, and the
   error would compound across rounds through the carry. `ResourceBar.Balance` becomes
   `float64` with it.
4. **The queue stops being a heap and keeps its public API.** `PriorityQueue` remains
   `[]*Action` with `Insert`, `ExtractMax`, `ExtractByID`, `Peek`, `IsEmpty`, `Len`, and gains
   `All()`. `heap.Interface` (`Less`/`Swap`/`Push`/`Pop`) goes away. Reason in
   `combat-engine.md`: the key is computed at selection time and moves as the average moves;
   a heap cannot re-key. `ExtractByID` already scanned linearly, so the heap never paid for
   itself at 4–6 characters. The product owner's words: *"ela deve ordenar pela barra ao invés
   de ser pela action speed."*
5. **Prices freeze per bar, at the first selection that sees that bar.** A bar whose first
   action only arrives mid-round freezes then, from the actions pending at that moment. A bar
   with no action all round never freezes, and its balance is left untouched at round close.
6. **The round's auto-close runs through the existing `CloseRoundUC`.** The session reports
   `TurnTransition.RoundExhausted` when nothing pending passes its gate; `OpenNextActionUC` and
   `PullActionUC` then call the injected `ICloseRound`. That is what "CloseRoundUC plugado
   aqui" means — the UC finally has a caller, and the balance arithmetic stays in the domain,
   inside `MatchSession.CloseRound`.
7. **Speeds are recorded on OPEN, never on enqueue** — and on *every* bar the action charges.
   `ResourceBar.Speeds` means "the speeds that acted". An action that never passes its gate is
   left in the queue with its full roll and belongs to the next round, which requires zero
   unwinding precisely because nothing was recorded.
8. **The mapper owns the speed skill; the client never picks it.** `actionSpeed` is always
   `Legerity`. `moveSpeed` comes from `Move.Category`. Whatever the payload put in
   `speed.rollCheck.skillName` or `move.speed.skillName` is overwritten.
9. **`ActionSpeedPayload.Bar` stays and is ignored.** Which bars an action pays from is derived
   from its content (`Move != nil` → the move bar; anything action-bar-shaped → the action bar;
   a combined action → both), never trusted from the client. The field gets a comment saying so
   rather than being removed.
10. **Modifier expiry (`CharacterStatus.ExpireModifiers`) is NOT wired here.** Nothing creates
    a modifier until the repel ladder in Phase 4, so calling it would be expiring an
    always-empty ledger. Left for the phase that fills it.
11. **`bars_updated` is broadcast to everyone.** `combat-engine.md` § Ciclo — *"A fila é
    secreta; a barra e a ordem são públicas."* The payload therefore carries balances and the
    projected order as `{actorId, bars, key}` — never an action ID and never any action content.
12. **`change_round_mode` is permissive mid-round.** Turning `Race` on starts the economy from
    that moment: `acted` is empty for everyone and prices freeze on the next selection. No new
    error path.

---

## File Structure

**Created**

| File | Responsibility |
|---|---|
| `internal/domain/match/entity/action/bar.go` | `Bar` enum; `Action.Bars()` / `SpeedOn(bar)` — which clocks an action pays from and what it entered with on each |
| `internal/domain/match/service/bar_economy.go` | The arithmetic of one bar in one round: mean, key, balance, both gates, closing balance |
| `internal/domain/match/service/round_scheduler.go` | Picks the next action out of the queue: dependency edge, gates, ordering; the round-close predicate; price freezing |
| `internal/application/match/change_round_mode.go` | Master-only use case that switches `Free`⇄`Race` |
| plus one `_test.go` beside each | |

**Modified**

| File | Change |
|---|---|
| `internal/domain/match/entity/action/priority_queue.go` | drop `heap`, add `All()` |
| `internal/domain/match/entity/round/round.go` | `coast *int` → `prices`; `HasOpenedAction`; `SetMode` |
| `internal/domain/match/resource_bar.go` | `Balance` becomes `float64`; `Speeds` means "the speeds that acted" |
| `internal/domain/match/matchsession/match_session.go` | derive both speeds, freeze, select, record on every charged bar, close the round with carry-over |
| `internal/domain/match/service/round_orchestrator.go` | `SetMode` beside the untouched `ChangeMode` |
| `internal/application/match/open_next_action.go`, `pull_action.go` | inject `ICloseRound`, auto-close on exhaustion |
| `internal/app/game/action_mapper.go` | force `Legerity`; category → move skill; reject 5 categories |
| `internal/app/game/message.go` | `change_round_mode`, `round_mode_changed`, `bars_updated` payloads |
| `internal/app/game/room.go` | new handler; emit `round_closed`, `round_mode_changed`, `bars_updated` |
| `docs/dev/match/combat-engine.md`, `docs/dev/match/turns-rounds.md`, `docs/dev/match/flows/05-lacunas.md`, `docs/documentation-map.yaml` | record what Phase 3 fixed |

---

### Task 1: `Bar` — which clocks an action pays from

**Files:**
- Create: `internal/domain/match/entity/action/bar.go`
- Test: `internal/domain/match/entity/action/bar_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces: `action.Bar` (`action.BarAction`, `action.BarMove`);
  `(*action.Action).Bars() []action.Bar`; `(*action.Action).SpeedOn(bar action.Bar) int`.

**The rule (`combat-engine.md` § Ações compostas):** a cait, an arremetida and an investida are
**one** `Action` with `Move` *and* `Attack` filled in. It charges **both** bars. There is no
split into two actions, no second queue entry and no dependency edge — the model was built for
exactly one action, with the movement inside it.

- [ ] **Step 1: Write the failing test**

```go
package action_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

func attackOnly(speed int) *action.Action {
	return action.NewAction(uuid.New(), nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: speed}},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
}

func moveOnly(speed int) *action.Action {
	return action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, &action.Move{Category: enum.Dash, FinalSpeed: speed}, nil, nil, nil, nil, nil)
}

func combined(actionSpeed, moveSpeed int) *action.Action {
	return action.NewAction(uuid.New(), nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: actionSpeed}},
		nil, &action.Move{Category: enum.Dash, FinalSpeed: moveSpeed},
		&action.Attack{}, nil, nil, nil, nil)
}

func TestAction_Bars(t *testing.T) {
	tests := []struct {
		name string
		a    *action.Action
		want []action.Bar
	}{
		{"an attack pays from the action bar", attackOnly(10), []action.Bar{action.BarAction}},
		{"a movement pays from the move bar", moveOnly(10), []action.Bar{action.BarMove}},
		{
			"an investida is ONE action that pays from both",
			combined(25, 5),
			[]action.Bar{action.BarAction, action.BarMove},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Bars()
			if len(got) != len(tt.want) {
				t.Fatalf("Bars() = %v, want %v", got, tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("Bars()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}

	t.Run("an action with nothing in it still pays from the action bar", func(t *testing.T) {
		a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, nil, nil, nil, nil, nil)
		if bars := a.Bars(); len(bars) != 1 || bars[0] != action.BarAction {
			t.Errorf("Bars() = %v, want just the action bar — never an empty set, or it would be unschedulable", bars)
		}
	})
}

func TestAction_SpeedOn(t *testing.T) {
	t.Run("each bar reads its own speed", func(t *testing.T) {
		a := combined(25, 5)
		if got := a.SpeedOn(action.BarAction); got != 25 {
			t.Errorf("SpeedOn(action) = %d, want 25", got)
		}
		if got := a.SpeedOn(action.BarMove); got != 5 {
			t.Errorf("SpeedOn(move) = %d, want 5", got)
		}
	})

	t.Run("a bar the action does not charge reads zero", func(t *testing.T) {
		if got := attackOnly(10).SpeedOn(action.BarMove); got != 0 {
			t.Errorf("SpeedOn(move) = %d, want 0 — this action does not move", got)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/entity/action/ -run 'TestAction_(Bars|SpeedOn)' -v`
Expected: FAIL — `a.Bars undefined`, `a.SpeedOn undefined`.

- [ ] **Step 3: Write the implementation**

`internal/domain/match/entity/action/bar.go`:

```go
package action

// Bar names one of the two clocks a character runs on. They have independent prices but
// share a single clock, and both live on the same scale — skill + the dice set — so the
// engine compares value against value with no conversion.
type Bar string

const (
	BarAction Bar = "action" // attack, item, ability
	BarMove   Bar = "move"   // shift, dash, leap, roll
)

// Bars reports which clocks this action is paid from.
//
// A combined action — cait, arremetida, investida — is ONE action with Move and Attack both
// filled in, and it charges BOTH bars. It is not split into two actions and it does not open
// two turns: the master opens it once, it is one turn, and it happens at the time of its
// slower half. See combat-engine.md § Ações compostas.
//
// The action bar is always first, and the result is never empty: an action with nothing in it
// at all still belongs to the action bar, or the scheduler would have nothing to price it by.
func (a *Action) Bars() []Bar {
	if a.Move == nil {
		return []Bar{BarAction}
	}
	if a.chargesActionBar() {
		return []Bar{BarAction, BarMove}
	}
	return []Bar{BarMove}
}

// chargesActionBar reports whether the action carries anything paid from the action bar, as
// opposed to being pure movement.
func (a *Action) chargesActionBar() bool {
	return a.Attack != nil || a.Defense != nil || a.Dodge != nil ||
		a.Interact != nil || a.Feint != nil || len(a.Skills) > 0
}

// SpeedOn is the speed this action entered the round with on one bar. Both speeds are derived
// once, when the action arrives, and neither is ever re-rolled — a combined action keeps its
// actionSpeed AND its moveSpeed; what the combination changes is only when it happens.
//
// A bar the action does not charge reads 0, which no caller reaches: the scheduler only ever
// asks about the bars Bars() returned.
func (a *Action) SpeedOn(bar Bar) int {
	if bar == BarMove {
		if a.Move == nil {
			return 0
		}
		return a.Move.FinalSpeed
	}
	if !a.chargesActionBar() && a.Move != nil {
		return 0
	}
	return a.Speed.Result
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/entity/action/ -v`
Expected: PASS, including the pre-existing action and queue tests.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/entity/action/bar.go         internal/domain/match/entity/action/bar_test.go
git commit -m "feat(match): name the two bars and which ones an action pays from"
```

---

### Task 2: `BarEconomy` — the arithmetic of one bar in one round

**Files:**
- Create: `internal/domain/match/service/bar_economy.go`
- Test: `internal/domain/match/service/bar_economy_test.go`

**Interfaces:**
- Consumes: nothing (pure numbers).
- Produces:
  - `service.BarEconomy` (stateless struct, usable as `service.BarEconomy{}`)
  - `(BarEconomy).Mean(acted []int) float64`
  - `(BarEconomy).Balance(carry float64, acted []int, price int) float64`
  - `(BarEconomy).Key(carry float64, acted []int, speed, price int) float64`
  - `(BarEconomy).IsEligible(carry float64, acted []int, speed, price int) bool`
  - `(BarEconomy).CloseBalance(carry float64, acted []int, price int) float64`

**Integers in, exact fractions out.** Rolled speeds and the price are whole numbers; the mean,
the key, the balance and the carry keep their fraction as `float64`.

- [ ] **Step 1: Write the failing test**

```go
package service_test

import (
	"math"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

// closeTo compares derived values without pretending float64 is exact. A third of a round's
// speeds is not representable in binary, and asserting equality on it would be a test that
// fails for a reason that has nothing to do with the rules.
func closeTo(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

func TestBarEconomy_Mean(t *testing.T) {
	eco := service.BarEconomy{}
	tests := []struct {
		name  string
		acted []int
		want  float64
	}{
		{"no action yet", nil, 0},
		{"one action is its own average", []int{23}, 23},
		{"an exact average", []int{23, 17}, 20},
		{"a fractional average keeps the half", []int{17, 12}, 14.5},
		{"a third is kept too", []int{20, 10, 9}, 13.0 + 1.0/3.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := eco.Mean(tt.acted); !closeTo(got, tt.want) {
				t.Errorf("Mean(%v) = %v, want %v", tt.acted, got, tt.want)
			}
		})
	}
}

func TestBarEconomy_KeyAndBalance(t *testing.T) {
	eco := service.BarEconomy{}
	const price = 11

	t.Run("the canonical example, action by action", func(t *testing.T) {
		// p2 opens the round: nothing acted yet, so the key is the roll itself.
		if got := eco.Key(0, nil, 23, price); !closeTo(got, 23) {
			t.Errorf("p2 first key = %v, want 23", got)
		}
		if got := eco.Key(0, nil, 20, price); !closeTo(got, 20) {
			t.Errorf("p1 first key = %v, want 20", got)
		}
		if got := eco.Key(0, nil, 11, price); !closeTo(got, 11) {
			t.Errorf("p3 first key = %v, want 11", got)
		}
		// p2's second action: the average moves to 20 and one action has been paid for.
		// It enters the queue at 9 — BELOW the price — and still acts.
		if got := eco.Key(0, []int{23}, 17, price); !closeTo(got, 9) {
			t.Errorf("p2 second key = %v, want 9", got)
		}
	})

	t.Run("the leftover after the actions that acted", func(t *testing.T) {
		if got := eco.Balance(0, []int{23}, price); !closeTo(got, 12) {
			t.Errorf("p2 leftover after one action = %v, want 12", got)
		}
		if got := eco.Balance(0, []int{23, 17}, price); !closeTo(got, -2) {
			t.Errorf("p2 leftover after two actions = %v, want -2", got)
		}
	})

	t.Run("the carry-over sums into the bar before anything is paid", func(t *testing.T) {
		// Round two for a character that carried +9 and rolls 20 against a price of 11.
		if got := eco.Key(9, nil, 20, price); !closeTo(got, 29) {
			t.Errorf("Key with carry = %v, want 29", got)
		}
		if got := eco.Balance(9, []int{20}, price); !closeTo(got, 18) {
			t.Errorf("Balance with carry = %v, want 18", got)
		}
	})

	t.Run("a fractional carry survives instead of being rounded away", func(t *testing.T) {
		// A character who acted twice at 17 and 12 against a price of 11 leaves 14.5 − 22.
		if got := eco.Balance(0, []int{17, 12}, price); !closeTo(got, -7.5) {
			t.Errorf("Balance = %v, want -7.5 — the half must not be truncated", got)
		}
	})
}

func TestBarEconomy_IsEligible(t *testing.T) {
	eco := service.BarEconomy{}
	const price = 11

	t.Run("first action of the round — the gate is the BAR, not the raw roll", func(t *testing.T) {
		if !eco.IsEligible(0, nil, 11, price) {
			t.Error("landing exactly on the price must act")
		}
		if eco.IsEligible(0, nil, 10, price) {
			t.Error("below the price with no credit must sit out the round")
		}
		if !eco.IsEligible(5, nil, 10, price) {
			t.Error("credit must be able to rescue a roll below the price")
		}
	})

	t.Run("second action onward — the gate is the leftover", func(t *testing.T) {
		// p2 has acted once at 23 against a price of 11: leftover 12, so a second
		// action is granted BEFORE it is rolled.
		if !eco.IsEligible(0, []int{23}, 17, price) {
			t.Error("a leftover of 12 must buy another action")
		}
		// p1 has acted once at 20: leftover 9, not enough.
		if eco.IsEligible(0, []int{20}, 20, price) {
			t.Error("a leftover of 9 must not buy another action, however good the new roll")
		}
	})

	t.Run("a granted right is not revoked by a bad roll", func(t *testing.T) {
		// The very case that breaks if the key is used as the gate: key 9 < price 11.
		if eco.Key(0, []int{23}, 17, price) >= price {
			t.Fatal("precondition: this key is below the price")
		}
		if !eco.IsEligible(0, []int{23}, 17, price) {
			t.Error("eligibility is decided before the new roll; the key only orders")
		}
	})
}

func TestBarEconomy_CloseBalance(t *testing.T) {
	eco := service.BarEconomy{}
	const price = 11

	t.Run("the canonical closing balances", func(t *testing.T) {
		if got := eco.CloseBalance(0, []int{20}, price); !closeTo(got, 9) {
			t.Errorf("p1 close = %v, want +9", got)
		}
		if got := eco.CloseBalance(0, []int{11}, price); !closeTo(got, 0) {
			t.Errorf("p3 close = %v, want 0", got)
		}
		if got := eco.CloseBalance(0, []int{23, 17}, price); !closeTo(got, -2) {
			t.Errorf("p2 close = %v, want -2", got)
		}
	})

	t.Run("the ceiling is the round price", func(t *testing.T) {
		if got := eco.CloseBalance(9, []int{20}, price); !closeTo(got, 11) {
			t.Errorf("close = %v, want the ceiling 11 — 18 may not be carried", got)
		}
	})

	t.Run("standing still carries the floor, which is the same number", func(t *testing.T) {
		if got := eco.CloseBalance(0, nil, price); !closeTo(got, 11) {
			t.Errorf("close = %v, want 11: not acting trades an action for time", got)
		}
		if got := eco.CloseBalance(9, nil, price); !closeTo(got, 11) {
			t.Errorf("close = %v, want 11 — credit does not stack past the ceiling", got)
		}
		if got := eco.CloseBalance(-2, nil, price); !closeTo(got, 9) {
			t.Errorf("close = %v, want 9: a debtor recovers, but only up to the floor", got)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestBarEconomy -v`
Expected: FAIL — `undefined: service.BarEconomy`.

- [ ] **Step 3: Write the implementation**

`internal/domain/match/service/bar_economy.go`:

```go
package service

// BarEconomy is the arithmetic of one resource bar, for one character, in one round.
// Stateless: every number it needs arrives as a parameter, so it is trivially testable and
// the state stays where it belongs, in match.CharacterStatus.
//
// Integers in, exact fractions out. A rolled speed is a skill value plus dice and a price is
// one of those speeds, so both are whole; everything DERIVED — the mean, the key, the balance,
// the carry — is float64 and keeps its fraction. Rounding would be a policy the rules never
// asked for, and the error would compound across rounds through the carry.
//
// Vocabulary, fixed by docs/dev/match/combat-engine.md:
//
//	carry — the balance that crossed over from the previous round, credit or debt
//	acted — the speeds of this character's actions on this bar that have ALREADY OPENED,
//	        in order. An action still sitting in the queue is not in here.
//	speed — the speed a pending action entered with, on this bar
//	price — the round price of this bar: the smallest speed among the actions pending on it
//	        when it froze. Equal for everyone, and also the ceiling on the carry-over.
type BarEconomy struct{}

// Mean is the round's average speed on this bar, exact.
func (BarEconomy) Mean(acted []int) float64 {
	if len(acted) == 0 {
		return 0
	}
	sum := 0
	for _, s := range acted {
		sum += s
	}
	return float64(sum) / float64(len(acted))
}

// Balance is what is left after paying for the actions that have already acted.
//
//	carry + mean(acted) − len(acted) × price
func (e BarEconomy) Balance(carry float64, acted []int, price int) float64 {
	return carry + e.Mean(acted) - float64(len(acted)*price)
}

// Key is where a pending action sits in the queue. It ORDERS ONLY — it never decides whether
// the action happens. See IsEligible.
//
//	carry + mean(acted ++ [speed]) − len(acted) × price
//
// The pending action's own speed is inside the average, which is why sending a second, slower
// action delays it within the round: the average drops and the key drops with it.
func (e BarEconomy) Key(carry float64, acted []int, speed, price int) float64 {
	withPending := make([]int, 0, len(acted)+1)
	withPending = append(withPending, acted...)
	withPending = append(withPending, speed)
	return carry + e.Mean(withPending) - float64(len(acted)*price)
}

// IsEligible is the gate: whether this pending action happens in this round at all.
//
// THERE ARE TWO GATES, AND NEITHER OF THEM IS THE KEY:
//
//   - First action of the round — the BAR reaches the price: carry + speed >= price.
//     Measured on the bar and not on the raw roll, so standing credit can rescue a bad roll.
//     With no carry (round 0) the two readings coincide.
//   - Second action onward — the LEFTOVER of the ones that already acted reaches the price.
//     Note what this means: the right to act again is decided BEFORE the new die falls, and a
//     bad roll does not revoke it. It only makes the action cost more afterwards, by dragging
//     the average down. In the canonical example p2's second action enters at key 9, below the
//     price of 11, and happens anyway.
//
// Using the key as the gate loses exactly that action and breaks the round's published order.
func (e BarEconomy) IsEligible(carry float64, acted []int, speed, price int) bool {
	if len(acted) == 0 {
		return carry+float64(speed) >= float64(price)
	}
	return e.Balance(carry, acted, price) >= float64(price)
}

// CloseBalance is what this bar carries into the next round, ceiling applied.
//
// The ceiling is the round price — never configurable, because it is what makes standing
// still stop paying after one round instead of compounding forever.
//
// A character who sent nothing carries the floor, which on this bar is the same number as the
// ceiling. That is deliberate, not a coincidence: standing still trades an action for time,
// and the trade is worth exactly one round's price. A character in debt recovers toward it
// rather than jumping to it.
func (e BarEconomy) CloseBalance(carry float64, acted []int, price int) float64 {
	balance := carry + float64(price)
	if len(acted) > 0 {
		balance = e.Balance(carry, acted, price)
	}
	if balance > float64(price) {
		return float64(price)
	}
	return balance
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/domain/match/service/ -run TestBarEconomy -v`
Expected: PASS, every sub-test.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/bar_economy.go         internal/domain/match/service/bar_economy_test.go
git commit -m "feat(match): add the bar economy — mean, key, both gates and the ceiling"
```

---

### Task 3: the round carries its frozen prices

**Files:**
- Modify: `internal/domain/match/entity/round/round.go`
- Modify: `internal/domain/match/service/round_orchestrator.go`
- Test: `internal/domain/match/entity/round/round_test.go` (append)

**Interfaces:**
- Consumes: `action.Bar` (Task 1).
- Produces:
  - `(*round.Round).Price(bar action.Bar) (int, bool)` — the frozen price, and whether it froze
  - `(*round.Round).FreezePrice(bar action.Bar, price int)` — idempotent; a second call is ignored
  - `(*round.Round).HasOpenedAction(id uuid.UUID) bool`
  - `(*round.Round).SetMode(mode enum.RoundMode)`
  - `(service.RoundOrchestrator).SetMode(r *round.Round, mode enum.RoundMode)`

- [ ] **Step 1: Write the failing test**

Append to `internal/domain/match/entity/round/round_test.go`:

```go
func TestRound_Prices(t *testing.T) {
	t.Run("a fresh round has no price on either bar", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		if _, frozen := r.Price(action.BarAction); frozen {
			t.Error("the action bar must start unfrozen")
		}
		if _, frozen := r.Price(action.BarMove); frozen {
			t.Error("the move bar must start unfrozen")
		}
	})

	t.Run("freezing one bar leaves the other alone", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		r.FreezePrice(action.BarAction, 11)

		got, frozen := r.Price(action.BarAction)
		if !frozen || got != 11 {
			t.Errorf("action price = (%d, %v), want (11, true)", got, frozen)
		}
		if _, frozen := r.Price(action.BarMove); frozen {
			t.Error("the move bar must still be unfrozen — the bars freeze independently")
		}
	})

	t.Run("the price never moves once frozen", func(t *testing.T) {
		r := round.NewRound(enum.Race)
		r.FreezePrice(action.BarAction, 11)
		r.FreezePrice(action.BarAction, 4)

		got, _ := r.Price(action.BarAction)
		if got != 11 {
			t.Errorf("price = %d, want 11: a later, slower action must not re-price the round", got)
		}
	})
}

func TestRound_HasOpenedAction(t *testing.T) {
	r := round.NewRound(enum.Race)
	a := action.NewAction(uuid.New(), nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, nil, nil, nil, nil, nil)

	if r.HasOpenedAction(a.GetID()) {
		t.Error("nothing has opened yet")
	}
	r.AppendTurn(turn.NewTurn(*a))
	if !r.HasOpenedAction(a.GetID()) {
		t.Error("the action opened, so the round must say so — this is what the dependency edge reads")
	}
}

func TestRound_SetMode(t *testing.T) {
	r := round.NewRound(enum.Free)
	r.SetMode(enum.Race)
	if r.GetMode() != enum.Race {
		t.Errorf("mode = %q, want Race", r.GetMode())
	}
	r.SetMode(enum.Race)
	if r.GetMode() != enum.Race {
		t.Error("SetMode must be idempotent, unlike ToggleMode")
	}
}
```

Add `"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"`,
`".../entity/turn"` and `"github.com/google/uuid"` to the test file's imports if they are not
already there.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/entity/round/ -v`
Expected: FAIL — `r.Price undefined`, `r.FreezePrice undefined`, `r.HasOpenedAction undefined`,
`r.SetMode undefined`.

- [ ] **Step 3: Write the implementation**

In `internal/domain/match/entity/round/round.go`, **replace** the `coast *int` field. It was
declared `//nolint:unused` with the note *"if nil, the turn is free (no race in this turn)"* —
that intent survives verbatim in the new field's doc, and it now holds one price per bar
instead of one for the round, because the two bars price independently.

```go
type Round struct {
	id            uuid.UUID
	mode          enum.RoundMode
	turns         []*turn.Turn
	masterActions []action.MasterAction //nolint:unused // WIP: match system under development // think more about this field and its usage
	events        []GameEvent
	// prices are the frozen round prices, one per bar. An absent entry means that bar has
	// not priced yet — which is also what a Free round looks like all the way through, since
	// Free has no price at all.
	prices     map[action.Bar]int
	createdAt  time.Time
	finishedAt *time.Time
}
```

Initialise `prices: make(map[action.Bar]int)` in **both** `NewRound` and `ReconstructRound`,
then add:

```go
// Price returns the frozen price of a bar, and whether it froze at all.
func (r *Round) Price(bar action.Bar) (int, bool) {
	p, ok := r.prices[bar]
	return p, ok
}

// FreezePrice fixes a bar's price for the rest of the round. The first call wins; every
// later one is ignored.
//
// The price is fixed when the first action on that bar is chosen to open — the smallest speed
// among the actions pending at that moment — and it does not move again. A slower action
// arriving later does NOT re-price the round: that would make the same round charge different
// people different amounts, or reorder what has already been played. Such an action simply
// does not reach the price, sits the round out, and carries its full roll into the next one.
func (r *Round) FreezePrice(bar action.Bar, price int) {
	if r.prices == nil {
		r.prices = make(map[action.Bar]int)
	}
	if _, frozen := r.prices[bar]; frozen {
		return
	}
	r.prices[bar] = price
}

// HasOpenedAction reports whether an action with this ID has already been given a turn in
// this round. The dependency edge of a combined action reads it: the attack half waits for
// the move half to open.
func (r *Round) HasOpenedAction(id uuid.UUID) bool {
	for _, t := range r.turns {
		if t.GetAction().GetID() == id {
			return true
		}
	}
	return false
}

// SetMode sets the round regime outright. ToggleMode stays for the paths that flip blindly;
// this one is what an explicit master request needs, and it is idempotent.
func (r *Round) SetMode(mode enum.RoundMode) {
	r.mode = mode
}
```

In `internal/domain/match/service/round_orchestrator.go`, add `SetMode` **below** the existing
`ChangeMode` — leaving `ChangeMode` and its `// TODO: create and finish Initiative to continue
here` exactly as they are:

```go
// SetMode puts the round into a specific regime.
//
// ChangeMode above toggles, and is the seat reserved for initiative — the game rule that will
// normally force Race. This one is the regime by itself: the master can turn the disputed turn
// on before that rule exists, which is what makes the whole bar economy reachable.
func (ro RoundOrchestrator) SetMode(r *round.Round, mode enum.RoundMode) {
	r.SetMode(mode)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/match/entity/round/ ./internal/domain/match/service/ -v`
Expected: PASS. Then `go vet -tags=integration ./internal/...` — clean.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/entity/round/round.go \
        internal/domain/match/entity/round/round_test.go \
        internal/domain/match/service/round_orchestrator.go
git commit -m "feat(match): freeze a price per bar on the round, and set the mode outright"
```

---

### Task 4: the queue stops being a heap

**Files:**
- Modify: `internal/domain/match/entity/action/priority_queue.go`
- Test: `internal/domain/match/entity/action/priority_queue_test.go` (append)

**Interfaces:**
- Consumes: nothing.
- Produces: `(*action.PriorityQueue).All() []*action.Action`. Every existing method keeps its
  name and behaviour: `Insert`, `ExtractMax`, `ExtractByID`, `Peek`, `IsEmpty`, `Len`,
  `NewActionPriorityQueue`. `Less`, `Swap`, `Push` and `Pop` are removed.

**Why:** the ordering key is not a property of an action any more. It is
`carry + mean(acted ++ [speed]) − len(acted) × price`, and it moves whenever the character
sends another action. A heap cannot re-key an item it already holds — it would silently return
a stale order. So the key is computed at selection time and the container becomes a plain list.
At 4–6 characters a scan is cheaper than heap maintenance, and `ExtractByID` already scanned.

- [ ] **Step 1: Write the failing test**

Append to `internal/domain/match/entity/action/priority_queue_test.go`:

```go
func TestPriorityQueue_All(t *testing.T) {
	pq := action.NewActionPriorityQueue(nil)
	a1, a2, a3 := makeAction(10), makeAction(30), makeAction(20)
	pq.Insert(a1)
	pq.Insert(a2)
	pq.Insert(a3)

	t.Run("hands back every pending action", func(t *testing.T) {
		all := pq.All()
		if len(all) != 3 {
			t.Fatalf("All() returned %d actions, want 3", len(all))
		}
		seen := map[uuid.UUID]bool{}
		for _, a := range all {
			seen[a.GetID()] = true
		}
		for _, want := range []*action.Action{a1, a2, a3} {
			if !seen[want.GetID()] {
				t.Errorf("action %v missing from All()", want.GetID())
			}
		}
	})

	t.Run("in insertion order, so ties resolve by who sent first", func(t *testing.T) {
		all := pq.All()
		if all[0].GetID() != a1.GetID() || all[2].GetID() != a3.GetID() {
			t.Error("All() must preserve insertion order")
		}
	})

	t.Run("the slice is a copy — the caller cannot reshape the queue", func(t *testing.T) {
		all := pq.All()
		all[0] = nil
		if pq.All()[0] == nil {
			t.Error("All() handed out the queue's own backing array")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/entity/action/ -run TestPriorityQueue -v`
Expected: FAIL — `pq.All undefined`.

- [ ] **Step 3: Write the implementation**

Replace the whole of `internal/domain/match/entity/action/priority_queue.go`:

```go
package action

import "github.com/google/uuid"

// PriorityQueue holds the actions waiting for the master to open them.
//
// It USED to be a max-heap keyed on Action.Speed.Result. It is not any more, and the reason is
// structural rather than a matter of taste: the position of an action in the round is
//
//	carry + mean(acted ++ [speed]) − len(acted) × price
//
// which is state of the CHARACTER, not of the action, and which moves every time that
// character sends another action — the round average shifts under it. A heap cannot re-key an
// entry it already holds; it would keep answering with a stale order and never say so. So the
// key is computed at selection time, by service.RoundScheduler, and this becomes a plain list.
// With four to six characters at a table, scanning costs less than maintaining a heap, and
// ExtractByID scanned linearly all along.
//
// Insertion order is preserved and is meaningful: it is how ties between equal keys resolve —
// whoever sent first goes first.
type PriorityQueue []*Action

func NewActionPriorityQueue(actions *[]*Action) PriorityQueue {
	if actions == nil {
		return make(PriorityQueue, 0)
	}
	return PriorityQueue(*actions)
}

func (aq PriorityQueue) Len() int      { return len(aq) }
func (aq *PriorityQueue) IsEmpty() bool { return aq.Len() == 0 }

// Insert adds a new action to the back of the queue.
func (aq *PriorityQueue) Insert(newAction *Action) {
	*aq = append(*aq, newAction)
}

// All returns every pending action, in insertion order, as a copy. The scheduler iterates it
// to compute a key per entry; handing out the backing array would let a caller reshape the
// queue behind the session's back.
func (aq *PriorityQueue) All() []*Action {
	out := make([]*Action, len(*aq))
	copy(out, *aq)
	return out
}

// ExtractMax removes and returns the action with the highest Speed.Result.
//
// This is the FREE-round path, where there is no price, no average and no carry-over, so the
// rolled speed IS the order. A Race round goes through service.RoundScheduler instead, which
// knows about the bars.
func (aq *PriorityQueue) ExtractMax() *Action {
	idx := aq.indexOfMax()
	if idx < 0 {
		return nil
	}
	return aq.removeAt(idx)
}

// Peek returns the action with the highest Speed.Result without removing it.
func (aq *PriorityQueue) Peek() *Action {
	idx := aq.indexOfMax()
	if idx < 0 {
		return nil
	}
	return (*aq)[idx]
}

// ExtractByID searches and removes a specific action by UUID.
func (aq *PriorityQueue) ExtractByID(id uuid.UUID) *Action {
	for i, act := range *aq {
		if act.GetID() == id {
			return aq.removeAt(i)
		}
	}
	return nil
}

// indexOfMax returns the index of the highest Speed.Result, or -1 on an empty queue. Ties go
// to the earliest insertion.
func (aq *PriorityQueue) indexOfMax() int {
	best := -1
	for i, act := range *aq {
		if best == -1 || act.Speed.Result > (*aq)[best].Speed.Result {
			best = i
		}
	}
	return best
}

// removeAt takes an action out while preserving the order of the rest.
func (aq *PriorityQueue) removeAt(i int) *Action {
	old := *aq
	act := old[i]
	*aq = append(old[:i], old[i+1:]...)
	return act
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/match/... -v`
Expected: PASS. The pre-existing `TestPriorityQueue_InsertAndExtractMax` and
`TestPriorityQueue_ExtractByID` must stay green untouched — the public behaviour did not change,
only the container underneath it.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/entity/action/priority_queue.go \
        internal/domain/match/entity/action/priority_queue_test.go
git commit -m "refactor(match): drop the heap — the queue key is character state, not action state"
```

---

### Task 5: `RoundScheduler` — who opens next, and when the round is over

**Files:**
- Create: `internal/domain/match/service/round_scheduler.go`
- Test: `internal/domain/match/service/round_scheduler_test.go`

**Interfaces:**
- Consumes: `service.BarEconomy` (Task 2), `action.Bar` / `Bars()` / `SpeedOn()` (Task 1),
  `(*action.PriorityQueue).All()` (Task 4), `round.Price` / `FreezePrice` (Task 3).
- Produces:
  - `service.BarStateSource` — `BarState(charID uuid.UUID, bar action.Bar) (carry float64, acted []int)`
  - `service.ScheduleInput` — `Queue *action.PriorityQueue`, `Round *round.Round`, `Bars BarStateSource`
  - `service.OrderSlot` — `ActorID uuid.UUID`, `Bars []action.Bar`, `Key float64`
  - `(RoundScheduler).FreezePrices(in ScheduleInput)`
  - `(RoundScheduler).SelectNext(in ScheduleInput) *action.Action` — nil when nothing passes its gate
  - `(RoundScheduler).AnyEligible(in ScheduleInput) bool` — the round-close predicate, negated
  - `(RoundScheduler).ProjectOrder(in ScheduleInput) []OrderSlot`

**The combined-action rule:** an action that charges both bars is scheduled at
`min(keyOnActionBar, keyOnMoveBar)` — the higher key acts first, so `min` is the slower half —
and it must clear the gate on **both** bars, because it pays both.

**This task carries the phase's first done-criterion.** The canonical example must reproduce
the order `p2 → p1 → p3 → p2` and the balances `+9 / 0 / −2` with speeds injected. It is tested
here, at the level where the rule lives, rather than through the dice.

- [ ] **Step 1: Write the failing test**

`internal/domain/match/service/round_scheduler_test.go`:

```go
package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// fakeBars is a test double for service.BarStateSource. It is deliberately dumb: the
// scheduler's job is the selection, and the arithmetic is BarEconomy's, already tested.
type fakeBars struct {
	carry map[uuid.UUID]map[action.Bar]float64
	acted map[uuid.UUID]map[action.Bar][]int
}

func newFakeBars() *fakeBars {
	return &fakeBars{
		carry: map[uuid.UUID]map[action.Bar]float64{},
		acted: map[uuid.UUID]map[action.Bar][]int{},
	}
}

func (f *fakeBars) BarState(charID uuid.UUID, bar action.Bar) (float64, []int) {
	return f.carry[charID][bar], f.acted[charID][bar]
}

// act records that an action opened, on every bar it charges — which is what moves the average.
func (f *fakeBars) act(a *action.Action) {
	charID := a.GetActorID()
	if f.acted[charID] == nil {
		f.acted[charID] = map[action.Bar][]int{}
	}
	for _, bar := range a.Bars() {
		f.acted[charID][bar] = append(f.acted[charID][bar], a.SpeedOn(bar))
	}
}

func (f *fakeBars) setCarry(charID uuid.UUID, bar action.Bar, carry float64) {
	if f.carry[charID] == nil {
		f.carry[charID] = map[action.Bar]float64{}
	}
	f.carry[charID][bar] = carry
}

// attackAt builds an action-bar action that entered the round at the given speed.
func attackAt(actorID uuid.UUID, speed int) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: speed}},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
}

// moveAt builds a move-bar action that entered the round at the given speed.
func moveAt(actorID uuid.UUID, speed int) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, &action.Move{Category: enum.Dash, FinalSpeed: speed}, nil, nil, nil, nil, nil)
}

// chargeAt builds an investida: ONE action charging both bars.
func chargeAt(actorID uuid.UUID, actionSpeed, moveSpeed int) *action.Action {
	return action.NewAction(actorID, nil, uuid.Nil, nil,
		action.ActionSpeed{RollCheck: action.RollCheck{Result: actionSpeed}},
		nil, &action.Move{Category: enum.Dash, FinalSpeed: moveSpeed},
		&action.Attack{}, nil, nil, nil, nil)
}

// TestRoundScheduler_CanonicalExample is the phase's done-criterion, at the level where the
// rule lives. Speeds are injected; nothing here depends on luck.
//
// docs/game/combate/barra-de-acao.md: p1 = 20, p2 = 23, p3 = 11, and p2's second roll = 17.
// Expected order p2 → p1 → p3 → p2, and closing balances p1 = +9, p3 = 0, p2 = −2.
func TestRoundScheduler_CanonicalExample(t *testing.T) {
	sch := service.RoundScheduler{}
	eco := service.BarEconomy{}
	bars := newFakeBars()
	p1, p2, p3 := uuid.New(), uuid.New(), uuid.New()

	r := round.NewRound(enum.Race)
	q := action.NewActionPriorityQueue(nil)
	q.Insert(attackAt(p1, 20))
	q.Insert(attackAt(p2, 23))
	q.Insert(attackAt(p3, 11))

	in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}

	// The master opens the first action: this is the moment the price freezes, at the
	// smallest speed on the bar among everything pending — 11.
	sch.FreezePrices(in)
	price, frozen := r.Price(action.BarAction)
	if !frozen || price != 11 {
		t.Fatalf("frozen price = (%d, %v), want (11, true)", price, frozen)
	}

	// open drains one action and records it as acted, the way MatchSession will.
	open := func() uuid.UUID {
		sch.FreezePrices(in)
		next := sch.SelectNext(in)
		if next == nil {
			t.Fatal("expected an action to open")
		}
		q.ExtractByID(next.GetID())
		r.AppendTurn(turn.NewTurn(*next))
		bars.act(next)
		return next.GetActorID()
	}

	var order []uuid.UUID
	order = append(order, open()) // p2 at 23
	order = append(order, open()) // p1 at 20
	order = append(order, open()) // p3 at 11

	// p2 sees a leftover of 23 − 11 = 12 and sends a second action, which rolls 17. The
	// right to act was granted by the leftover, BEFORE this roll.
	if !eco.IsEligible(0, []int{23}, 17, price) {
		t.Fatal("precondition: a leftover of 12 buys p2 another action")
	}
	q.Insert(attackAt(p2, 17))
	order = append(order, open()) // p2 again, now at key 9

	want := []uuid.UUID{p2, p1, p3, p2}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order[%d] = %v, want %v (full order %v, want %v)", i, order[i], want[i], order, want)
		}
	}

	t.Run("the round is over: nothing pending, nothing eligible", func(t *testing.T) {
		if sch.AnyEligible(in) {
			t.Error("the queue is empty, so the round must be closeable")
		}
	})

	t.Run("closing balances", func(t *testing.T) {
		cases := []struct {
			name string
			char uuid.UUID
			want float64
		}{
			{"p1 acted once at 20 and carries +9", p1, 9},
			{"p3 acted once at 11 and carries 0", p3, 0},
			{"p2 acted twice and carries the debt", p2, -2},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				carry, acted := bars.BarState(c.char, action.BarAction)
				if got := eco.CloseBalance(carry, acted, price); !closeTo(got, c.want) {
					t.Errorf("CloseBalance = %v, want %v", got, c.want)
				}
			})
		}
	})
}

func TestRoundScheduler_Gates(t *testing.T) {
	sch := service.RoundScheduler{}
	eco := service.BarEconomy{}
	p1, p2 := uuid.New(), uuid.New()

	t.Run("an action that never reaches the price sits the round out", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		q.Insert(attackAt(p1, 20))
		q.Insert(attackAt(p2, 7))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}

		sch.FreezePrices(in)
		// The price is the smallest pending speed, so p2 sits exactly ON it and DOES act.
		if p, _ := r.Price(action.BarAction); p != 7 {
			t.Fatalf("price = %d, want 7", p)
		}

		// Now a genuinely late, slow action arrives after the freeze.
		late := attackAt(p2, 3)
		q.Insert(late)
		first := sch.SelectNext(in)
		q.ExtractByID(first.GetID())
		bars.act(first)

		carry, acted := bars.BarState(late.GetActorID(), action.BarAction)
		if eco.IsEligible(carry, acted, 3, 7) {
			t.Error("a speed of 3 against a frozen price of 7 must not act")
		}
	})

	t.Run("a late but fast action jumps the rest of the queue", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		q.Insert(attackAt(p1, 12))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		q.Insert(attackAt(p2, 30))
		next := sch.SelectNext(in)
		if next.GetActorID() != p2 {
			t.Error("the late action rolled higher, so it opens next — the system never reorders what already played, but it does not hold back what has not")
		}
	})

	t.Run("standing credit rescues a roll below the price", func(t *testing.T) {
		bars := newFakeBars()
		bars.setCarry(p2, action.BarAction, 6)
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		q.Insert(attackAt(p1, 9))
		q.Insert(attackAt(p2, 5))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		if p, _ := r.Price(action.BarAction); p != 5 {
			t.Fatalf("price = %d, want 5", p)
		}
		if !sch.AnyEligible(in) {
			t.Error("carry 6 + roll 5 = 11 clears a price of 5")
		}
	})
}

func TestRoundScheduler_BarsPriceIndependently(t *testing.T) {
	sch := service.RoundScheduler{}
	bars := newFakeBars()
	p1 := uuid.New()

	r := round.NewRound(enum.Race)
	q := action.NewActionPriorityQueue(nil)
	q.Insert(attackAt(p1, 20))
	q.Insert(moveAt(p1, 6))
	in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
	sch.FreezePrices(in)

	if p, _ := r.Price(action.BarAction); p != 20 {
		t.Errorf("action price = %d, want 20 — the move action must not price the action bar", p)
	}
	if p, _ := r.Price(action.BarMove); p != 6 {
		t.Errorf("move price = %d, want 6", p)
	}
}

// TestRoundScheduler_CombinedAction covers the investida: ONE action charging both bars,
// opened once, happening at the time of its slower half.
func TestRoundScheduler_CombinedAction(t *testing.T) {
	sch := service.RoundScheduler{}
	p1, p2 := uuid.New(), uuid.New()

	t.Run("it is scheduled at its slower half, not its faster one", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		// p1 is quick of hand (25) and slow of foot (5). The blow does not get to jump the
		// queue at 25: the investida happens when the feet get there.
		q.Insert(chargeAt(p1, 25, 5))
		q.Insert(attackAt(p2, 12))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		next := sch.SelectNext(in)
		if next.GetActorID() != p2 {
			t.Error("the investida is keyed on min(25, 5) = 5, so the plain attack at 12 goes first")
		}
	})

	t.Run("it opens ONCE and charges both bars", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		a := chargeAt(p1, 25, 5)
		q.Insert(a)
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		next := sch.SelectNext(in)
		if next == nil || next.GetID() != a.GetID() {
			t.Fatal("the investida must open")
		}
		q.ExtractByID(next.GetID())
		bars.act(next)

		if sch.SelectNext(in) != nil {
			t.Error("one action, one opening — there is no second half left in the queue")
		}
		if _, acted := bars.BarState(p1, action.BarAction); len(acted) != 1 {
			t.Error("the action bar must have been charged")
		}
		if _, acted := bars.BarState(p1, action.BarMove); len(acted) != 1 {
			t.Error("the move bar must have been charged too — it pays both")
		}
	})

	t.Run("it needs both gates: a spent action bar keeps the whole investida out", func(t *testing.T) {
		bars := newFakeBars()
		r := round.NewRound(enum.Race)
		q := action.NewActionPriorityQueue(nil)
		// Two plain actions price both bars at 10, then the investida arrives able to pay
		// the movement (12) but not the blow (4).
		q.Insert(attackAt(p2, 10))
		q.Insert(moveAt(p2, 10))
		in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
		sch.FreezePrices(in)

		q.Insert(chargeAt(p1, 4, 12))
		for _, slot := range sch.ProjectOrder(in) {
			if slot.ActorID == p1 {
				t.Error("the investida charges both bars, so it must clear both gates — riding the move bar into a free attack would void the economy")
			}
		}
	})
}

func TestRoundScheduler_FreeRoundHasNoEconomy(t *testing.T) {
	sch := service.RoundScheduler{}
	bars := newFakeBars()
	p1, p2 := uuid.New(), uuid.New()

	r := round.NewRound(enum.Free)
	q := action.NewActionPriorityQueue(nil)
	q.Insert(attackAt(p1, 4))
	q.Insert(attackAt(p2, 9))
	in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}

	sch.FreezePrices(in)
	if _, frozen := r.Price(action.BarAction); frozen {
		t.Error("a Free round has no price at all — the economy is Race's, and Free's rule is not written yet")
	}

	next := sch.SelectNext(in)
	if next.GetActorID() != p2 {
		t.Error("with no economy the rolled speed is the whole order")
	}
	if !sch.AnyEligible(in) {
		t.Error("a Free round never auto-closes while anything is pending")
	}
}

func TestRoundScheduler_ProjectOrder(t *testing.T) {
	sch := service.RoundScheduler{}
	bars := newFakeBars()
	p1, p2, p3 := uuid.New(), uuid.New(), uuid.New()

	r := round.NewRound(enum.Race)
	q := action.NewActionPriorityQueue(nil)
	q.Insert(attackAt(p1, 20))
	q.Insert(attackAt(p2, 23))
	q.Insert(attackAt(p3, 11))
	in := service.ScheduleInput{Queue: &q, Round: r, Bars: bars}
	sch.FreezePrices(in)

	slots := sch.ProjectOrder(in)
	if len(slots) != 3 {
		t.Fatalf("slots = %d, want 3", len(slots))
	}
	want := []uuid.UUID{p2, p1, p3}
	for i := range want {
		if slots[i].ActorID != want[i] {
			t.Errorf("slot[%d] = %v, want %v — highest key first", i, slots[i].ActorID, want[i])
		}
	}
	if !closeTo(slots[0].Key, 23) {
		t.Errorf("slot[0].Key = %v, want 23", slots[0].Key)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/service/ -run TestRoundScheduler -v`
Expected: FAIL — `undefined: service.RoundScheduler`, `undefined: service.ScheduleInput`,
`undefined: service.BarStateSource`.

- [ ] **Step 3: Write the implementation**

`internal/domain/match/service/round_scheduler.go`:

```go
package service

import (
	"math"
	"sort"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/google/uuid"
)

// BarStateSource hands the scheduler the per-character bar state it needs. Implemented by
// *matchsession.MatchSession, which owns the CharacterStatus map; declared here so the domain
// service never imports the session.
//
// carry is the balance that crossed over from the previous round. acted is the ordered list
// of speeds this character has ALREADY OPENED on that bar this round — never the pending ones.
type BarStateSource interface {
	BarState(charID uuid.UUID, bar action.Bar) (carry float64, acted []int)
}

// ScheduleInput is everything one scheduling decision reads.
type ScheduleInput struct {
	Queue *action.PriorityQueue
	Round *round.Round
	Bars  BarStateSource
}

// OrderSlot is one entry of the projected order: who acts, which clocks it charges, and where
// in the round it lands. It deliberately carries no action identity — the queue is secret, the
// order is public.
type OrderSlot struct {
	ActorID uuid.UUID
	Bars    []action.Bar
	Key     float64
}

// RoundScheduler decides which pending action opens next and when the round is over. It is a
// stateless domain service: all the state lives in the Round and behind BarStateSource.
type RoundScheduler struct{}

// FreezePrices fixes the price of every bar that has pending work and has not priced yet.
//
// The price of a bar is the smallest speed among the actions pending on it at the moment its
// first action is chosen. It is frozen per bar, not per round: a bar whose first action only
// shows up mid-round prices then, from what is pending then. Once frozen it never moves — a
// slower action arriving afterwards does not re-price the round, it simply may not reach the
// price and sits the round out with its full roll intact.
//
// A Free round is left untouched: it has no price, no average and no carry-over. That economy
// belongs to Race, and Free's own rule has not been designed yet.
func (RoundScheduler) FreezePrices(in ScheduleInput) {
	if in.Round == nil || in.Queue == nil || in.Round.GetMode() != enum.Race {
		return
	}
	lowest := map[action.Bar]int{}
	for _, a := range in.Queue.All() {
		for _, bar := range a.Bars() {
			if _, frozen := in.Round.Price(bar); frozen {
				continue
			}
			speed := a.SpeedOn(bar)
			if current, seen := lowest[bar]; !seen || speed < current {
				lowest[bar] = speed
			}
		}
	}
	for bar, price := range lowest {
		in.Round.FreezePrice(bar, price)
	}
}

// SelectNext returns the pending action that opens next, or nil when nothing passes its gate.
//
// Nil in a Race round is the round-close signal: everything still queued belongs to the next
// round, carrying the roll it already made.
func (rs RoundScheduler) SelectNext(in ScheduleInput) *action.Action {
	var best *action.Action
	var bestKey float64
	for _, a := range in.Queue.All() {
		key, ok := rs.keyOf(in, a)
		if !ok {
			continue
		}
		// Strictly greater, so a tie goes to whoever was inserted first.
		if best == nil || key > bestKey {
			best, bestKey = a, key
		}
	}
	return best
}

// AnyEligible reports whether any pending action passes the gate that applies to it. Its
// negation is the round-close predicate: the round ends when nothing pending can still pay.
//
// It is deliberately NOT "when the bars run out". A bar does not zero — it ends wherever it
// ends, debt included. And it is not about the key either: an action whose key fell below the
// price still happens, as long as it had already been granted.
func (rs RoundScheduler) AnyEligible(in ScheduleInput) bool {
	return rs.SelectNext(in) != nil
}

// ProjectOrder returns every pending action that passes its gate, highest key first. It is
// SelectNext generalised from "the next one" to "all of them, in order", which is what the
// general bar shows — including a character who is going to act more than once.
func (rs RoundScheduler) ProjectOrder(in ScheduleInput) []OrderSlot {
	var slots []OrderSlot
	for _, a := range in.Queue.All() {
		key, ok := rs.keyOf(in, a)
		if !ok {
			continue
		}
		slots = append(slots, OrderSlot{ActorID: a.GetActorID(), Bars: a.Bars(), Key: key})
	}
	// Stable, so equal keys keep insertion order — the same tie-break SelectNext uses.
	sort.SliceStable(slots, func(i, j int) bool { return slots[i].Key > slots[j].Key })
	return slots
}

// keyOf returns where an action sits in the order, and whether it may open at all.
//
// An action charging BOTH bars — an investida — is scheduled at its SLOWER half and must clear
// the gate on BOTH. The higher key acts first, so the slower half is the smaller key, hence
// the min. And it pays both bars, so it has to be able to afford both: letting it through on
// the move bar alone would hand a character with a spent action bar a free attack.
func (rs RoundScheduler) keyOf(in ScheduleInput, a *action.Action) (float64, bool) {
	key := math.Inf(1)
	for _, bar := range a.Bars() {
		barKey, ok := rs.keyOnBar(in, a, bar)
		if !ok {
			return 0, false
		}
		if barKey < key {
			key = barKey
		}
	}
	if math.IsInf(key, 1) {
		return 0, false
	}
	return key, true
}

// keyOnBar scores one action against one of the bars it charges.
func (rs RoundScheduler) keyOnBar(in ScheduleInput, a *action.Action, bar action.Bar) (float64, bool) {
	speed := a.SpeedOn(bar)

	var carry float64
	var acted []int
	if in.Bars != nil {
		carry, acted = in.Bars.BarState(a.GetActorID(), bar)
	}

	price, frozen := 0, false
	if in.Round != nil {
		price, frozen = in.Round.Price(bar)
	}
	if !frozen {
		// No economy on this bar — a Free round, or a bar that has not priced yet. The rolled
		// speed is the whole order and nothing gates.
		return carry + float64(speed), true
	}

	eco := BarEconomy{}
	if !eco.IsEligible(carry, acted, speed, price) {
		return 0, false
	}
	return eco.Key(carry, acted, speed, price), true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/match/service/ -run 'TestRoundScheduler|TestBarEconomy' -v`
Expected: PASS. `TestRoundScheduler_CanonicalExample` is the phase's first done-criterion —
if the order comes out `p2 → p1 → p3` with no fourth entry, the gate has been confused with the
key; re-read `combat-engine.md` § "Quem pode agir — são DOIS porteiros, não um".

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/service/round_scheduler.go         internal/domain/match/service/round_scheduler_test.go
git commit -m "feat(match): schedule the round by bar key, gated by the two gates"
```

---

### Task 6: the mapper owns the speed skills

**Files:**
- Modify: `internal/app/game/action_mapper.go`
- Test: `internal/app/game/action_mapper_test.go` (append)

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `buildAction` now guarantees `Speed.RollCheck.SkillName == enum.Legerity.String()`
  and, for a move, `Move.Speed != nil` with the skill implied by the category. Unsupported
  categories come back as an error.

**Rules (spec §5):** actionSpeed is always `Legerity`. A `Dash` rolls `Accelerate`; a `Shift`
takes the passive value of `Brake` and rolls nothing. The other five categories — `Back`,
`Roll`, `Slide`, `Jump`, `FlatJump` — are rejected. **Never map them by analogy:** treating
anything-not-Dash as a Shift would price a leap like a controlled step and nobody would find
out until someone complained at the table.

- [ ] **Step 1: Write the failing test**

Append to `internal/app/game/action_mapper_test.go`:

```go
func TestBuildAction_SpeedSkills(t *testing.T) {
	actor := uuid.New()

	t.Run("actionSpeed is always Legerity, whatever the client asked for", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{
			ActorID: actor,
			Speed:   &ActionSpeedPayload{RollCheck: &RollCheckPayload{SkillName: enum.Accuracy.String()}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Speed.SkillName != enum.Legerity.String() {
			t.Errorf("actionSpeed skill = %q, want %q — the player never picks it",
				a.Speed.SkillName, enum.Legerity)
		}
	})

	t.Run("actionSpeed is Legerity even when the payload omits speed entirely", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{ActorID: actor})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Speed.SkillName != enum.Legerity.String() {
			t.Errorf("actionSpeed skill = %q, want %q", a.Speed.SkillName, enum.Legerity)
		}
	})

	t.Run("a Dash rolls Accelerate", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{
			ActorID: actor,
			Move:    &MovePayload{Category: string(enum.Dash), Position: [3]int{1, 1, 0}},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Move.Speed == nil {
			t.Fatal("a move must always carry a speed check")
		}
		if a.Move.Speed.SkillName != enum.Accelerate.String() {
			t.Errorf("move skill = %q, want %q", a.Move.Speed.SkillName, enum.Accelerate)
		}
	})

	t.Run("a Shift uses Brake, and the client's choice is overwritten", func(t *testing.T) {
		a, err := buildAction(actor, ActionPayload{
			ActorID: actor,
			Move: &MovePayload{
				Category: string(enum.Shift),
				Position: [3]int{1, 1, 0},
				Speed:    &RollCheckPayload{SkillName: enum.Accelerate.String()},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Move.Speed.SkillName != enum.Brake.String() {
			t.Errorf("move skill = %q, want %q — the category picks the skill, not the client",
				a.Move.Speed.SkillName, enum.Brake)
		}
	})

	t.Run("the five unmapped categories are refused, not guessed", func(t *testing.T) {
		for _, cat := range []enum.MoveCategory{enum.Back, enum.Roll, enum.Slide, enum.Jump, enum.FlatJump} {
			t.Run(string(cat), func(t *testing.T) {
				_, err := buildAction(actor, ActionPayload{
					ActorID: actor,
					Move:    &MovePayload{Category: string(cat), Position: [3]int{1, 1, 0}},
				})
				if err == nil {
					t.Errorf("category %q must be refused: its skill is defined in the movement slice, and mapping it by analogy would be silently wrong", cat)
				}
			})
		}
	})

	t.Run("an unknown category string is refused too", func(t *testing.T) {
		if _, err := buildAction(actor, ActionPayload{
			ActorID: actor,
			Move:    &MovePayload{Category: "Teleport", Position: [3]int{1, 1, 0}},
		}); err == nil {
			t.Error("an unknown move category must be an error at the boundary")
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run TestBuildAction_SpeedSkills -v`
Expected: FAIL — the actionSpeed skill comes back as whatever the client sent (or empty), and
every move category is accepted.

- [ ] **Step 3: Write the implementation**

In `internal/app/game/action_mapper.go`, replace the `speed` block:

```go
	// actionSpeed is ALWAYS Legerity. Whatever the payload carried is discarded: the
	// player picks a move type or an attack, never the skill behind its speed.
	// ActionSpeedPayload.Bar is likewise ignored — which bar an action pays from is derived
	// from its content by Action.Bars(), never trusted from the client.
	speed := action.ActionSpeed{
		RollCheck: action.RollCheck{SkillName: enum.Legerity.String()},
	}
```

and replace the `move` block:

```go
	var move *action.Move
	if p.Move != nil {
		category, moveSkill, err := moveSpeedSkill(p.Move.Category)
		if err != nil {
			return nil, err
		}
		moveCharge, err := buildRollCheck(p.Move.Charge)
		if err != nil {
			return nil, err
		}
		move = &action.Move{
			Category: category,
			From:     p.Move.From,
			Position: p.Move.Position,
			// The skill comes from the category, never from the payload. The front shows the
			// tactical move types explicitly; switching Dash to Shift in the bottom sheet
			// switches the skill on its own.
			Speed:  &action.RollCheck{SkillName: moveSkill.String()},
			Charge: moveCharge,
		}
		// FinalSpeed is computed by the engine, never taken from the client.
	}
```

and add, near `buildWeaponName`:

```go
// moveSpeedSkill maps a movement category to the skill its speed is tested on.
//
//	Dash  (arrancada)            → Accelerate, rolled
//	Shift (controlled step)      → Brake, taken passively (see MatchSession.deriveSpeeds)
//
// enum.MoveCategory has five other values — Back (cait), Roll, Slide, Jump, FlatJump — and
// they are REFUSED here rather than mapped. Their skills belong to the movement slice, which
// is where they will actually be exercised. Mapping them by analogy would work silently and
// wrongly: a leap would cost like a controlled step, and nobody would find out until someone
// complained at the table.
func moveSpeedSkill(raw string) (enum.MoveCategory, enum.SkillName, error) {
	switch enum.MoveCategory(raw) {
	case enum.Dash:
		return enum.Dash, enum.Accelerate, nil
	case enum.Shift:
		return enum.Shift, enum.Brake, nil
	default:
		return "", "", fmt.Errorf("move category %q is not supported yet", raw)
	}
}
```

Add `"fmt"` to the file's imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/game/ -v`
Expected: PASS. Any pre-existing mapper test that sent a category other than Dash or Shift now
expects an error — update it in place and say so in the commit body.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/action_mapper.go internal/app/game/action_mapper_test.go
git commit -m "feat(game): the category picks the movement skill, and actionSpeed is Legerity"
```

---

### Task 7: the session derives a real speed when the action arrives

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Modify: `internal/domain/match/resource_bar.go` (doc comment only)
- Test: `internal/domain/match/matchsession/match_session_test.go` (append)

**Interfaces:**
- Consumes: `action.Bars()`/`SpeedOn()` (Task 1), `RollCalculator.Derive` (Phase 2).
- Produces: after `EnqueueAction`, `a.Speed.Result` and — for a move — `a.Move.Speed.Result`
  and `a.Move.FinalSpeed` hold real numbers. A combined move+attack action gets **both**, and
  stays a single queued action.

**Rules:**
- actionSpeed = `Legerity` + the dice set, **or the passive value when the round is `Free`**
  (`docs/dev/match/combat-engine.md` § Rolagem). The `ModifierLedger` applies — the accumulated
  difference a character carries is always an actionSpeed adjustment, never a hit adjustment.
- moveSpeed: `Dash` rolls `Accelerate`; `Shift` takes the passive value of `Brake`.
- ⚠️ **A passive check must not consume dice.** `rollActionDice` currently rolls for
  `a.Move.Speed` unconditionally. A scripted `RollSource` in a test would be drained by that
  phantom roll and every number downstream would shift. Skip the roll when the check is passive.
- `Charge` stays out of Phase 3 (spec §5): the momentum accumulating into `Velocity` belongs to
  the movement slice. `Move.Charge` is still rolled, as Phase 2 left it, and nothing reads it.

- [ ] **Step 1: Write the failing test**

Append to `internal/domain/match/matchsession/match_session_test.go`:

```go
func TestMatchSession_EnqueueAction_DerivesTheSpeed(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{charID: buildSheet(t, playerUUID)}

	t.Run("a Race actionSpeed rolls Legerity plus the dice", func(t *testing.T) {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
		s.GetActiveRound().SetMode(enum.Race)
		s.SetRollSource(fixedSource{face: 6})

		a := action.NewAction(charID, nil, uuid.Nil, nil,
			action.ActionSpeed{RollCheck: action.RollCheck{SkillName: enum.Legerity.String()}},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Two D10 at 6 each, plus the sheet's Legerity.
		want := 12 + skillValue(t, sheets[charID], enum.Legerity)
		if a.Speed.Result != want {
			t.Errorf("Speed.Result = %d, want %d", a.Speed.Result, want)
		}
		if bars := a.Bars(); len(bars) != 1 || bars[0] != action.BarAction {
			t.Errorf("Bars() = %v, want just the action bar", bars)
		}
	})

	t.Run("a Free actionSpeed takes the passive value and rolls nothing", func(t *testing.T) {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
		// A new session starts in Free.
		s.SetRollSource(fixedSource{face: 10})

		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := 11 + skillValue(t, sheets[charID], enum.Legerity)
		if a.Speed.Result != want {
			t.Errorf("Speed.Result = %d, want the passive %d — rolling has zero expected gain", a.Speed.Result, want)
		}
	})

	t.Run("a Dash rolls Accelerate into the move bar", func(t *testing.T) {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
		s.GetActiveRound().SetMode(enum.Race)
		s.SetRollSource(fixedSource{face: 4})

		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil,
			&action.Move{Category: enum.Dash, Speed: &action.RollCheck{SkillName: enum.Accelerate.String()}},
			nil, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := 8 + skillValue(t, sheets[charID], enum.Accelerate)
		if a.Move.FinalSpeed != want {
			t.Errorf("Move.FinalSpeed = %d, want %d", a.Move.FinalSpeed, want)
		}
		if a.SpeedOn(action.BarMove) != want {
			t.Errorf("SpeedOn(move) = %d, want %d", a.SpeedOn(action.BarMove), want)
		}
		if bars := a.Bars(); len(bars) != 1 || bars[0] != action.BarMove {
			t.Errorf("Bars() = %v, want just the move bar", bars)
		}
	})

	t.Run("a Shift takes the passive value of Brake and consumes no dice", func(t *testing.T) {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
		s.GetActiveRound().SetMode(enum.Race)
		src := &countingSource{face: 9}
		s.SetRollSource(src)

		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil,
			&action.Move{Category: enum.Shift, Speed: &action.RollCheck{SkillName: enum.Brake.String()}},
			nil, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := 11 + skillValue(t, sheets[charID], enum.Brake)
		if a.Move.FinalSpeed != want {
			t.Errorf("Move.FinalSpeed = %d, want the passive %d", a.Move.FinalSpeed, want)
		}
		if !a.Move.Speed.Attempts.IsEmpty() {
			t.Error("a Shift rolls nothing: a phantom roll here silently drains a scripted source and shifts every number downstream")
		}
	})
}

func TestMatchSession_EnqueueAction_CombinedActionKeepsBothSpeeds(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{charID: buildSheet(t, playerUUID)}

	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
	s.GetActiveRound().SetMode(enum.Race)
	s.SetRollSource(fixedSource{face: 5})

	sword := enum.Sword
	a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil,
		&action.Move{Category: enum.Dash, Position: [3]int{2, 2, 0}},
		&action.Attack{Weapon: &sword, Hit: action.RollCheck{SkillName: enum.Accuracy.String()}},
		nil, nil, nil, nil)
	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Run("it stays ONE action in the queue", func(t *testing.T) {
		if pending := s.PendingActions(); len(pending) != 1 {
			t.Fatalf("queued %d actions, want 1 — an investida is a single action with the movement inside it", len(pending))
		}
	})

	t.Run("it charges both bars", func(t *testing.T) {
		bars := a.Bars()
		if len(bars) != 2 {
			t.Fatalf("Bars() = %v, want both", bars)
		}
	})

	t.Run("both speeds are derived and both survive on the action", func(t *testing.T) {
		wantAction := 10 + skillValue(t, sheets[charID], enum.Legerity)
		wantMove := 10 + skillValue(t, sheets[charID], enum.Accelerate)
		if a.SpeedOn(action.BarAction) != wantAction {
			t.Errorf("actionSpeed = %d, want %d", a.SpeedOn(action.BarAction), wantAction)
		}
		if a.SpeedOn(action.BarMove) != wantMove {
			t.Errorf("moveSpeed = %d, want %d", a.SpeedOn(action.BarMove), wantMove)
		}
	})
}

Add these helpers to the test file (beside `makeParticipant`):

```go
// buildSheet builds a real character sheet from the factory, so skill values come from the
// same place production reads them.
func buildSheet(t *testing.T, playerUUID uuid.UUID) *csSheet.CharacterSheet {
	t.Helper()
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

// skillValue reads a skill off a sheet the same way the engine does, so a test never hard
// codes a number the factory owns.
func skillValue(t *testing.T, cs *csSheet.CharacterSheet, name enum.SkillName) int {
	t.Helper()
	v, err := cs.GetValueForTestOfSkill(name)
	if err != nil {
		t.Fatalf("GetValueForTestOfSkill(%s): %v", name, err)
	}
	return v
}

// countingSource is a scripted source that also reports how many dice it handed out, so a
// test can prove a passive check rolled nothing.
type countingSource struct {
	face  int
	rolls int
}

func (c *countingSource) RollDie(_ enum.DieSides) int {
	c.rolls++
	return c.face
}
```

If `buildSheet` already exists in the file under another name, reuse it instead of adding a
second one.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/matchsession/ -run 'TestMatchSession_EnqueueAction_(DerivesTheSpeed|SplitsACombinedAction)' -v`
Expected: FAIL — `Speed.Result` is 0, `Move.FinalSpeed` is 0, `s.PendingActions` undefined.

- [ ] **Step 3: Write the implementation**

In `internal/domain/match/matchsession/match_session.go`:

**(a)** Teach `rollActionDice` to skip passive checks. Replace the `a.Move` branch:

```go
	if a.Move != nil {
		// A Shift takes the dice set's average and rolls NOTHING. Rolling anyway would be
		// harmless in production and poisonous in a test: a scripted RollSource would be
		// drained by the phantom roll and every number after it would shift.
		if a.Move.Category != enum.Shift {
			test(a.Move.Speed)
		}
		test(a.Move.Charge)
	}
```

and guard the actionSpeed roll the same way:

```go
	// A Free round takes the passive value for actionSpeed — there is no dispute over who
	// acts first, so there is nothing to roll for.
	if s.activeRound.GetMode() == enum.Race {
		test(&a.Speed.RollCheck)
	}
```

**(b)** Add the derivation, called from `EnqueueAction` right after `rollActionDice`:

```go
// deriveSpeeds turns the dice that just fell into the numbers the round is ordered by:
// Action.Speed.Result for the action bar, Move.FinalSpeed for the move bar.
//
// It runs once, when the action arrives, and it is the only place a speed is produced. The
// master never re-rolls a player's die, so nothing downstream ever recomputes it.
func (s *MatchSession) deriveSpeeds(a *action.Action) {
	if a == nil {
		return
	}
	sheet := s.charSheets[a.GetActorID()]
	if sheet == nil {
		return
	}
	calc := service.RollCalculator{}
	var ledger *match.ModifierLedger
	if status, ok := s.statuses[a.GetActorID()]; ok {
		ledger = &status.Ledger
	}

	// actionSpeed: always Legerity. Passive in Free, rolled in Race.
	//
	// The ledger applies here and nowhere else in a collision: the accumulated difference a
	// character carries is always an actionSpeed adjustment, never a hit adjustment. It is
	// what makes the repel ladder produce a duel — two characters facing each other speed up
	// against each other — without anyone programming duels.
	a.Speed.SkillName = enum.Legerity.String()
	a.Speed.SkillValue = skillValueOn(sheet, enum.Legerity)
	a.Speed.Result = calc.Derive(s.rules, a.Speed.Attempts, service.RollInput{
		SkillName:  a.Speed.SkillName,
		SkillValue: a.Speed.SkillValue,
		Passive:    s.activeRound.GetMode() != enum.Race,
		Condition:  a.Speed.Context.Condition,
		Ledger:     ledger,
	}).Total

	if a.Move == nil {
		return
	}
	// moveSpeed: the skill comes from the category, and so does whether it rolls at all.
	// Dash is an acceleration and is tested; Shift is controlled and takes the passive value.
	// Anything else never reaches here — the mapper refuses it at the WS boundary.
	skill, passive := enum.Accelerate, false
	if a.Move.Category == enum.Shift {
		skill, passive = enum.Brake, true
	}
	if a.Move.Speed == nil {
		a.Move.Speed = &action.RollCheck{}
	}
	a.Move.Speed.SkillName = skill.String()
	a.Move.Speed.SkillValue = skillValueOn(sheet, skill)
	a.Move.Speed.Result = calc.Derive(s.rules, a.Move.Speed.Attempts, service.RollInput{
		SkillName:  a.Move.Speed.SkillName,
		SkillValue: a.Move.Speed.SkillValue,
		Passive:    passive,
		Condition:  a.Move.Speed.Context.Condition,
		// No ledger on the move bar: the accumulated difference is an actionSpeed bonus.
	}).Total
	// Charge is deliberately not read. The momentum accumulating into CharacterStatus.Velocity
	// is the movement slice's, and the bar works without it (spec §5, Fase 3).
	a.Move.FinalSpeed = a.Move.Speed.Result
}

// skillValueOn reads a skill off the sheet, contributing 0 for a name the sheet does not
// know. The WS boundary already rejects unknown names, so reaching here means an internal one.
func skillValueOn(cs *csSheet.CharacterSheet, name enum.SkillName) int {
	v, err := cs.GetValueForTestOfSkill(name)
	if err != nil {
		return 0
	}
	return v
}
```

**(c)** Derive on enqueue and expose the queue. Replace `EnqueueAction`:

```go
func (s *MatchSession) EnqueueAction(playerUUID uuid.UUID, a *action.Action) error {
	if _, ok := s.participants[playerUUID]; !ok {
		return ErrParticipantNotFound
	}
	owner, ok := s.charToPlayer[a.GetActorID().String()]
	if !ok || owner != playerUUID {
		return ErrActionActorMismatch
	}
	s.rollActionDice(a)
	s.deriveSpeeds(a)
	s.activeQueue.Insert(a)
	return nil
}

// PendingActions returns the actions still waiting for the master, in insertion order. Read
// by the delivery layer to publish the general bar, and by tests.
func (s *MatchSession) PendingActions() []*action.Action { return s.activeQueue.All() }
```

> A combined action — `Move` and `Attack` in the same `Action` — is **not** split. It goes into
> the queue once, carrying both speeds, and `Bars()` reports that it charges both clocks. The
> scheduler keys it on the slower half. See `combat-engine.md` § Ações compostas.

**(d)** In `internal/domain/match/resource_bar.go`, make `Balance` a `float64` and sharpen the
doc on `Speeds`:

```go
type ResourceBar struct {
	// Balance is the standing credit or debt, and it is a float64 on purpose: it is DERIVED
	// from an average that rarely divides evenly, and it crosses rounds, so truncating it
	// would compound an error the rules never asked for.
	Balance float64
	Speeds  []int
}
```


```go
// Speeds is the ordered list of the speeds that ACTUALLY ACTED on this bar during the current
// round — a speed is appended when the master opens that action, never when it is enqueued. An
// action still waiting in the queue is not in here, which is exactly what lets one that never
// reaches the price roll over to the next round with its full roll and nothing to unwind.
//
// A combined action appends to BOTH bars, because it charges both.
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/match/... -v`
Expected: PASS, including the Phase 2 suites. Then `go vet -tags=integration ./internal/...`.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/matchsession/match_session.go \
        internal/domain/match/matchsession/match_session_test.go \
        internal/domain/match/resource_bar.go
git commit -m "feat(match): derive a real speed on arrival and split a combined action in two"
```

---

### Task 8: the session schedules the round instead of popping a heap

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Test: `internal/domain/match/matchsession/match_session_test.go` (append)

**Interfaces:**
- Consumes: `service.RoundScheduler`, `service.ScheduleInput`, `service.BarStateSource`
  (Task 5); `round.FreezePrice`/`Price` (Task 3).
- Produces:
  - `(*MatchSession).BarState(charID uuid.UUID, bar action.Bar) (float64, []int)` — satisfies
    `service.BarStateSource`
  - `TurnTransition.RoundExhausted bool`
  - `(*MatchSession).RoundPrices() map[action.Bar]int`

- [ ] **Step 1: Write the failing test**

Append to `internal/domain/match/matchsession/match_session_test.go`:

```go
func TestMatchSession_OpenNextAction_UsesTheBarEconomy(t *testing.T) {
	matchUUID := uuid.New()
	p1UUID, p2UUID := uuid.New(), uuid.New()
	p1 := makeParticipant(matchUUID, &p1UUID)
	p2 := makeParticipant(matchUUID, &p2UUID)
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		p1.Sheet.UUID: buildSheet(t, p1UUID),
		p2.Sheet.UUID: buildSheet(t, p2UUID),
	}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{p1, p2})
	s.GetActiveRound().SetMode(enum.Race)

	// Two D10 per action, scripted: p1 gets 4+4 = 8, p2 gets 9+9 = 18.
	s.SetRollSource(&scriptedFaces{faces: []int{4, 4, 4, 4, 9, 9, 9, 9}})

	enqueue := func(pl uuid.UUID, charID uuid.UUID) {
		t.Helper()
		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		if err := s.EnqueueAction(pl, a); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
	}
	enqueue(p1UUID, p1.Sheet.UUID)
	enqueue(p2UUID, p2.Sheet.UUID)

	tr, err := s.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}

	t.Run("the faster character opens first, not whoever was inserted first", func(t *testing.T) {
		if tr.Opened.GetAction().GetActorID() != p2.Sheet.UUID {
			t.Error("p2 rolled higher and must open first — the queue finally has a priority")
		}
	})

	t.Run("the action bar priced at the slowest pending speed", func(t *testing.T) {
		price, frozen := s.GetActiveRound().Price(action.BarAction)
		if !frozen {
			t.Fatal("opening the first action must freeze the price")
		}
		want := 8 + skillValue(t, sheets[p1.Sheet.UUID], enum.Legerity)
		if price != want {
			t.Errorf("price = %d, want p1's speed %d", price, want)
		}
	})

	t.Run("the opened action was recorded as having acted", func(t *testing.T) {
		_, acted := s.BarState(p2.Sheet.UUID, action.BarAction)
		if len(acted) != 1 {
			t.Fatalf("acted = %v, want exactly the one speed that opened", acted)
		}
		_, p1Acted := s.BarState(p1.Sheet.UUID, action.BarAction)
		if len(p1Acted) != 0 {
			t.Error("a pending action must not be recorded as acted")
		}
	})
}

func TestMatchSession_OpenNextAction_ReportsExhaustion(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	charID := participant.Sheet.UUID
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{charID: buildSheet(t, playerUUID)}
	s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{participant})
	s.GetActiveRound().SetMode(enum.Race)
	s.SetRollSource(fixedSource{face: 5})

	a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	if _, err := s.OpenNextAction(); err != nil {
		t.Fatalf("first open: %v", err)
	}

	tr, err := s.OpenNextAction()

	t.Run("exhaustion is a report, not an error", func(t *testing.T) {
		if err != nil {
			t.Errorf("err = %v, want nil: an exhausted Race round is a normal outcome", err)
		}
		if !tr.RoundExhausted {
			t.Error("nothing pending passes its gate, so the round is exhausted")
		}
		if tr.Opened != nil {
			t.Error("nothing opened")
		}
	})

	t.Run("the turn under the baton still closed and still applied", func(t *testing.T) {
		if tr.Closed == nil {
			t.Error("the open turn must close on the way out, exactly as it does on a normal open")
		}
	})
}

func TestMatchSession_FreeRoundStillReportsAnEmptyQueue(t *testing.T) {
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	participant := makeParticipant(matchUUID, &playerUUID)
	s := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	_, err := s.OpenNextAction()
	if !errors.Is(err, service.ErrQueueEmpty) {
		t.Errorf("err = %v, want ErrQueueEmpty: Free has no economy, so an empty queue is still an error", err)
	}
}
```

Add this scripted source beside the existing `fixedSource`:

```go
// scriptedFaces hands out faces in order and repeats the last one once exhausted.
type scriptedFaces struct {
	faces []int
	i     int
}

func (s *scriptedFaces) RollDie(_ enum.DieSides) int {
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
```

> ⚠️ Remember `RollCalculator.Roll` rolls **both** attempt sets: 2D10 costs **four** faces per
> check, not two. `pickAttempt` reads Primary on a neutral bias, which is the first two.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/matchsession/ -run 'TestMatchSession_(OpenNextAction_UsesTheBarEconomy|OpenNextAction_ReportsExhaustion|FreeRoundStillReportsAnEmptyQueue)' -v`
Expected: FAIL — `s.BarState` undefined, `tr.RoundExhausted` undefined, and the order comes out
by insertion.

- [ ] **Step 3: Write the implementation**

In `internal/domain/match/matchsession/match_session.go`:

Add the scheduler to the struct (`scheduler service.RoundScheduler`) — a zero value, so no
constructor change is required, but set it explicitly in both constructors alongside
`roundOrch` for symmetry with what is already there.

```go
// BarState implements service.BarStateSource: the carry that crossed into this round, and the
// speeds that have already acted on that bar. An unknown character reads as an empty bar
// rather than failing the whole scheduling pass.
func (s *MatchSession) BarState(charID uuid.UUID, bar action.Bar) (float64, []int) {
	status, ok := s.statuses[charID]
	if !ok {
		return 0, nil
	}
	b := status.BarFor(bar)
	return b.Balance, b.Speeds
}

// RoundPrices returns the frozen price of each bar that has priced this round. Read by the
// delivery layer to publish the general bar.
func (s *MatchSession) RoundPrices() map[action.Bar]int {
	out := map[action.Bar]int{}
	for _, bar := range []action.Bar{action.BarAction, action.BarMove} {
		if p, frozen := s.activeRound.Price(bar); frozen {
			out[bar] = p
		}
	}
	return out
}

// scheduleInput assembles what one scheduling decision reads.
func (s *MatchSession) scheduleInput() service.ScheduleInput {
	return service.ScheduleInput{Queue: &s.activeQueue, Round: s.activeRound, Bars: s}
}
```

Add to `internal/domain/match/character_status.go`:

```go
// BarFor returns the resource bar an action on that clock is paid from. The pointer is the
// status's own, so writing through it is a write to session state.
func (s *CharacterStatus) BarFor(bar action.Bar) *ResourceBar {
	if bar == action.BarMove {
		return &s.MoveBar
	}
	return &s.ActionBar
}
```

Add `RoundExhausted bool` to `TurnTransition`, with:

```go
	// RoundExhausted reports that no pending action passes the gate that applies to it, which
	// is what ends a Race round. It is not an error: the actions still queued keep the roll
	// they already made and belong to the next round. The caller closes the round.
	RoundExhausted bool
```

Replace `OpenNextAction`:

```go
func (s *MatchSession) OpenNextAction() (*TurnTransition, error) {
	// Prices freeze on the first selection that sees a bar with pending work — before any
	// gate is evaluated, because the gate is measured against the price.
	s.scheduler.FreezePrices(s.scheduleInput())

	if s.activeRound.GetMode() != enum.Race {
		// Free has no price, no average and no carry-over; the rolled speed is the order and
		// nothing gates. An empty queue is still an error there, as it always was.
		tr := s.closeOpenTurn()
		opened, err := s.roundOrch.NextAction(s.activeRound, &s.activeQueue)
		if err != nil {
			return tr, err
		}
		tr.Opened = opened
		tr.OpenedResolution = s.ResolveTurn(opened)
		return tr, nil
	}

	next := s.scheduler.SelectNext(s.scheduleInput())
	tr := s.closeOpenTurn()
	if next == nil {
		// Nothing pending can still pay. The round is over — the caller closes it.
		tr.RoundExhausted = true
		return tr, nil
	}

	// PullAction is reused deliberately: once the scheduler has chosen, "open the next one"
	// and "pull this one out of order" are the same operation. The master's explicit
	// pull_action stays ungated on purpose — anticipating an action is their prerogative.
	opened, err := s.roundOrch.PullAction(s.activeRound, &s.activeQueue, next.GetID())
	if err != nil {
		return tr, err
	}
	s.recordActed(next)
	tr.Opened = opened
	tr.OpenedResolution = s.ResolveTurn(opened)
	return tr, nil
}

// recordActed appends an action's speed to EVERY bar it was paid from, at the moment it opens.
//
// Every bar, because a combined action charges both: an investida costs a movement and a blow,
// and both averages move because of it.
//
// On OPEN, never on enqueue: ResourceBar.Speeds means "the speeds that acted", and that is
// what makes an action which never reached the price roll over to the next round untouched.
func (s *MatchSession) recordActed(a *action.Action) {
	status, ok := s.statuses[a.GetActorID()]
	if !ok {
		return
	}
	for _, bar := range a.Bars() {
		status.BarFor(bar).RecordSpeed(a.SpeedOn(bar))
	}
}
```

and record in `PullAction` too, right after the orchestrator hands the turn back:

```go
	s.recordActed(opened.GetActionPtr())
```

— or, simpler and with no new accessor on `turn`, capture the action before opening:

```go
func (s *MatchSession) PullAction(id uuid.UUID) (*TurnTransition, error) {
	tr := s.closeOpenTurn()
	s.scheduler.FreezePrices(s.scheduleInput())
	opened, err := s.roundOrch.PullAction(s.activeRound, &s.activeQueue, id)
	if err != nil {
		return tr, err
	}
	act := opened.GetAction()
	s.recordActed(&act)
	tr.Opened = opened
	tr.OpenedResolution = s.ResolveTurn(opened)
	return tr, nil
}
```

Use the second form — `turn.GetAction()` already returns a copy and `recordActed` only reads.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/match/... -v` then `go vet -tags=integration ./internal/...`
Expected: PASS. Watch the Phase 2 suites: `TestE2E_AttackAgainstACharacterProducesDamage` runs
in Free and must be unaffected.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/matchsession/match_session.go \
        internal/domain/match/matchsession/match_session_test.go \
        internal/domain/match/character_status.go
git commit -m "feat(match): open the next action by bar key, and report an exhausted round"
```

---

### Task 9: closing the round settles the bars and carries them over

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go`
- Test: `internal/domain/match/matchsession/match_session_test.go` (append)

**Interfaces:**
- Consumes: `service.BarEconomy.CloseBalance` (Task 2), `round.Price` (Task 3),
  `CharacterStatus.BarFor` (Task 8).
- Produces: `(*MatchSession).CloseRound()` now settles every bar before starting the next
  round. Signature unchanged — `(*round.Round, error)`.

> The assertions below compare `float64` balances with `==`. That is safe *here* because every
> value in these cases divides exactly (`14 − 8`, `20 − 4`, `0 + 12`). A case whose average is
> a third would need a tolerance, the way `closeTo` does in the service tests.

**Rules:** for each character, for each bar that priced this round:
`min(carry + mean(acted) − len(acted) × price, price)`, or `min(carry + price, price)` for a
character who sent nothing. A bar that never priced is left exactly as it was — no round
happened on it. Then `Speeds` is cleared and `Balance` becomes the new carry.

- [ ] **Step 1: Write the failing test**

```go
func TestMatchSession_CloseRound_SettlesTheBars(t *testing.T) {
	matchUUID := uuid.New()
	p1UUID, p2UUID := uuid.New(), uuid.New()
	p1 := makeParticipant(matchUUID, &p1UUID)
	p2 := makeParticipant(matchUUID, &p2UUID)
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		p1.Sheet.UUID: buildSheet(t, p1UUID),
		p2.Sheet.UUID: buildSheet(t, p2UUID),
	}

	newRacingSession := func(faces []int) *matchsession.MatchSession {
		s := matchsession.NewMatchSession(matchUUID, sheets, []*match.Participant{p1, p2})
		s.GetActiveRound().SetMode(enum.Race)
		s.SetRollSource(&scriptedFaces{faces: faces})
		return s
	}

	t.Run("a character who acted once carries the leftover", func(t *testing.T) {
		// p1 rolls 4+4 = 8, p2 rolls 7+7 = 14. Legerity is 0 on a factory sheet, so those
		// ARE the speeds. Price = 8. p2 keeps 14 − 8 = 6, under the ceiling of 8; p1 keeps 0.
		s := newRacingSession([]int{4, 4, 4, 4, 7, 7, 7, 7})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)
		enqueueAttack(t, s, p2UUID, p2.Sheet.UUID)

		closeExhaustedRound(t, s)

		carry, acted := s.BarState(p2.Sheet.UUID, action.BarAction)
		if carry != 6 {
			t.Errorf("p2 carry = %d, want 6", carry)
		}
		if len(acted) != 0 {
			t.Error("the round's speed history must be cleared; the balance is what crosses over")
		}
		if p1Carry, _ := s.BarState(p1.Sheet.UUID, action.BarAction); p1Carry != 0 {
			t.Errorf("p1 carry = %d, want 0 — the slowest of the round starts the next one from zero", p1Carry)
		}
	})

	t.Run("the carry is capped at the round price", func(t *testing.T) {
		// p1 = 2+2 = 4 (the price), p2 = 10+10 = 20. p2's leftover of 16 blows past the
		// ceiling of 4 and is clipped to it.
		s := newRacingSession([]int{2, 2, 2, 2, 10, 10, 10, 10})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)
		enqueueAttack(t, s, p2UUID, p2.Sheet.UUID)

		closeExhaustedRound(t, s)

		if carry, _ := s.BarState(p2.Sheet.UUID, action.BarAction); carry != 4 {
			t.Errorf("carry = %d, want the ceiling 4 — standing time may not compound", carry)
		}
	})

	t.Run("a character who sent nothing carries the floor", func(t *testing.T) {
		// Only p1 acts, at 6+6 = 12, which is therefore also the price. p1 closes at 0, and
		// p2 — who never sent anything — closes at the floor, the same number as the ceiling.
		s := newRacingSession([]int{6, 6, 6, 6})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)

		closeExhaustedRound(t, s)

		if carry, _ := s.BarState(p2.Sheet.UUID, action.BarAction); carry != 12 {
			t.Errorf("p2 carry = %d, want the floor 12: reading the fight instead of acting is a legitimate trade", carry)
		}
		if carry, _ := s.BarState(p1.Sheet.UUID, action.BarAction); carry != 0 {
			t.Errorf("p1 carry = %d, want 0", carry)
		}
	})

	t.Run("a bar that never priced is left exactly as it was", func(t *testing.T) {
		s := newRacingSession([]int{6, 6, 6, 6})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)

		closeExhaustedRound(t, s)

		if carry, _ := s.BarState(p1.Sheet.UUID, action.BarMove); carry != 0 {
			t.Errorf("move carry = %d, want 0 — nobody moved, so no round happened on that bar", carry)
		}
	})

	t.Run("an action that never reached the price keeps its full roll for the next round", func(t *testing.T) {
		// p1 acts at 7+7 = 14, pricing the bar at 14. Only THEN does p2 send an action worth
		// 1+1 = 2, which cannot pay and sits the round out.
		s := newRacingSession([]int{7, 7, 7, 7})
		enqueueAttack(t, s, p1UUID, p1.Sheet.UUID)
		mustOpenNext(t, s)

		s.SetRollSource(&scriptedFaces{faces: []int{1, 1, 1, 1}})
		enqueueAttack(t, s, p2UUID, p2.Sheet.UUID)

		pending := s.PendingActions()
		if len(pending) != 1 {
			t.Fatalf("pending = %d, want 1", len(pending))
		}
		speedBefore := pending[0].SpeedOn(action.BarAction)

		closeExhaustedRound(t, s)

		after := s.PendingActions()
		if len(after) != 1 {
			t.Fatalf("pending after close = %d, want the action still queued", len(after))
		}
		if after[0].SpeedOn(action.BarAction) != speedBefore {
			t.Errorf("speed = %d, want %d unchanged: it pays nothing and goes to the next round whole",
				after[0].SpeedOn(action.BarAction), speedBefore)
		}
		if carry, _ := s.BarState(p2.Sheet.UUID, action.BarAction); carry < 0 {
			t.Errorf("carry = %d — sitting the round out must never put the bar in debt", carry)
		}
	})
}
```

Add the three helpers used above beside `mustOpen`:

```go
func enqueueAttack(t *testing.T, s *matchsession.MatchSession, playerUUID, charID uuid.UUID) *action.Action {
	t.Helper()
	a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
		nil, nil, &action.Attack{}, nil, nil, nil, nil)
	if err := s.EnqueueAction(playerUUID, a); err != nil {
		t.Fatalf("EnqueueAction: %v", err)
	}
	return a
}

func mustOpenNext(t *testing.T, s *matchsession.MatchSession) *matchsession.TurnTransition {
	t.Helper()
	tr, err := s.OpenNextAction()
	if err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	return tr
}

// closeExhaustedRound drives the round the way production does: open until the session
// reports there is nothing left that can pay, and only THEN close.
//
// Calling CloseRound directly after an open would fail with ErrRoundHasOpenTurn — the turn
// under the baton is closed by the open that finds nothing, not by the round close.
func closeExhaustedRound(t *testing.T, s *matchsession.MatchSession) {
	t.Helper()
	for i := 0; i < 20; i++ {
		tr, err := s.OpenNextAction()
		if err != nil {
			t.Fatalf("OpenNextAction: %v", err)
		}
		if tr.RoundExhausted {
			if _, err := s.CloseRound(); err != nil {
				t.Fatalf("CloseRound: %v", err)
			}
			return
		}
	}
	t.Fatal("the round never ran out — the gate is letting through more than it should")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/domain/match/matchsession/ -run TestMatchSession_CloseRound_SettlesTheBars -v`
Expected: FAIL — every balance comes back 0; `CloseRound` currently only swaps the round.

- [ ] **Step 3: Write the implementation**

Replace `CloseRound` in `internal/domain/match/matchsession/match_session.go`:

```go
func (s *MatchSession) CloseRound() (*round.Round, error) {
	if s.activeRound.HasOpenTurn() {
		return nil, ErrRoundHasOpenTurn
	}
	s.settleBars()
	mode := s.activeRound.GetMode()
	closed := s.roundOrch.CloseRound(s.activeRound, time.Now())
	s.activeRound = round.NewRound(mode)
	s.roundPersisted = false
	return closed, nil
}

// settleBars turns each character's round into the balance they carry into the next one, then
// clears the round's speed history.
//
//	acted:  min(carry + mean(acted) − len(acted) × price, price)
//	silent: min(carry + price, price)   — standing still trades an action for time, and the
//	                                      trade is worth exactly one round's price
//
// The ceiling is the price on both branches, which is why standing still stops paying after a
// single round instead of compounding: whoever acts also reaches the ceiling in a few rounds.
//
// A bar that never priced is left untouched — nobody acted on that clock, so no round happened
// on it, and inventing a floor there would hand out free time.
//
// Nothing is done to the queue. An action that never reached the price was never recorded as
// having acted, so it simply belongs to the next round, carrying the roll it already made.
func (s *MatchSession) settleBars() {
	eco := service.BarEconomy{}
	for _, bar := range []action.Bar{action.BarAction, action.BarMove} {
		price, frozen := s.activeRound.Price(bar)
		if !frozen {
			continue
		}
		for _, status := range s.statuses {
			b := status.BarFor(bar)
			b.Balance = eco.CloseBalance(b.Balance, b.Speeds, price)
			b.ResetRound()
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/match/... -v` then `go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/domain/match/matchsession/match_session.go \
        internal/domain/match/matchsession/match_session_test.go
git commit -m "feat(match): settle both bars when the round closes and carry them over"
```

---

### Task 10: the exhausted round closes itself, through `CloseRoundUC`

**Files:**
- Modify: `internal/application/match/open_next_action.go`
- Modify: `internal/application/match/pull_action.go`
- Modify: `internal/application/match/open_next_action_test.go`, `pull_action_test.go`
- Modify: `cmd/game/main.go` (constructor wiring)

**Interfaces:**
- Consumes: `TurnTransition.RoundExhausted` (Task 8), the existing `ICloseRound`.
- Produces: `NewOpenNextActionUC(statusWriter ISheetStatusWriter, closeRound ICloseRound)` and
  `NewPullActionUC(statusWriter ISheetStatusWriter, closeRound ICloseRound)`;
  `OpenNextActionResult.ClosedRound *round.Round` and the same on `PullActionResult`.

**Why the UC and not the session:** the spec says *"Fim de round quando as barras acabam — e
portanto `CloseRoundUC` plugado aqui"*. `CloseRoundUC` exists, has no caller, and already owns
the persistence and the master check. The session reports exhaustion; the UC closes.

- [ ] **Step 1: Write the failing test**

Append to `internal/application/match/open_next_action_test.go`:

```go
// spyCloseRound records that the exhausted round was handed to the close use case.
type spyCloseRound struct {
	calls  int
	closed *round.Round
	err    error
}

func (s *spyCloseRound) Execute(
	ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID,
) (*round.Round, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	r, err := session.CloseRound()
	s.closed = r
	return r, err
}

func TestOpenNextAction_ClosesAnExhaustedRound(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	session, charID := racingSessionWithOneAction(t, playerUUID)
	_ = charID

	spy := &spyCloseRound{}
	uc := appmatch.NewOpenNextActionUC(nil, spy)

	// First open consumes the only action.
	if _, err := uc.Execute(context.Background(), session, masterUUID, masterUUID); err != nil {
		t.Fatalf("first open: %v", err)
	}

	res, err := uc.Execute(context.Background(), session, masterUUID, masterUUID)

	t.Run("no error — an exhausted round is a normal outcome", func(t *testing.T) {
		if err != nil {
			t.Errorf("err = %v, want nil", err)
		}
	})
	t.Run("the round was closed through CloseRoundUC", func(t *testing.T) {
		if spy.calls != 1 {
			t.Errorf("close calls = %d, want 1", spy.calls)
		}
		if res == nil || res.ClosedRound == nil {
			t.Error("the caller needs the closed round to announce round_closed")
		}
	})
	t.Run("nothing opened", func(t *testing.T) {
		if res != nil && res.OpenedTurn != nil {
			t.Error("there was nothing left to open")
		}
	})
}

func TestOpenNextAction_EmptyFreeQueueStillErrors(t *testing.T) {
	masterUUID := uuid.New()
	session := freeSessionWithNoActions(t)
	uc := appmatch.NewOpenNextActionUC(nil, &spyCloseRound{})

	if _, err := uc.Execute(context.Background(), session, masterUUID, masterUUID); err == nil {
		t.Error("a Free round has no economy: an empty queue is still an error, as it always was")
	}
}
```

Write `racingSessionWithOneAction` and `freeSessionWithNoActions` in the test file, building a
session the same way `TestMatchSession_CloseRound_SettlesTheBars` does: a participant with a
factory-built sheet, `SetMode(enum.Race)`, a `fixedSource`, and one enqueued attack.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/match/ -run TestOpenNextAction -v`
Expected: FAIL — `NewOpenNextActionUC` takes one argument; `res.ClosedRound` undefined.

- [ ] **Step 3: Write the implementation**

In `internal/application/match/open_next_action.go`:

```go
type OpenNextActionResult struct {
	ClosedTurn *turn.Turn
	OpenedTurn *turn.Turn
	// Resolution is the newly opened turn's projection — a dry run, nothing applied.
	Resolution *service.TurnResolution
	// ClosedResolution is the resolution that was actually applied when the previous turn
	// closed. Nil on the first open of a round.
	ClosedResolution *service.TurnResolution
	Damaged          []matchsession.DamagedCharacter
	// ClosedRound is set when the round ran out: nothing pending could still pay, so the
	// round closed instead of opening anything. The caller announces round_closed.
	ClosedRound *round.Round
}

type OpenNextActionUC struct {
	statusWriter ISheetStatusWriter
	closeRound   ICloseRound
}

func NewOpenNextActionUC(statusWriter ISheetStatusWriter, closeRound ICloseRound) *OpenNextActionUC {
	return &OpenNextActionUC{statusWriter: statusWriter, closeRound: closeRound}
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
	// Persist before the error check: the previous turn closed and its damage was applied
	// even when there is no next action to open, so bailing out first would leave the
	// in-memory sheet and the row disagreeing.
	persistDamage(ctx, uc.statusWriter, tr.Damaged)

	res := &OpenNextActionResult{
		ClosedTurn:       tr.Closed,
		OpenedTurn:       tr.Opened,
		Resolution:       tr.OpenedResolution,
		ClosedResolution: tr.ClosedResolution,
		Damaged:          tr.Damaged,
	}

	// The round ran out. Nothing pending passes the gate that applies to it, so the round
	// ends — and whatever is still queued keeps the roll it already made and belongs to the
	// next one. This is the moment CloseRoundUC finally has a caller.
	if tr.RoundExhausted {
		if uc.closeRound == nil {
			return res, nil
		}
		closed, closeErr := uc.closeRound.Execute(ctx, session, masterUUID, callerUUID)
		if closeErr != nil {
			// Same policy as persistDamage: the turn is already closed and applied, so
			// refusing the whole operation would leave the table without the baton.
			log.Printf("auto-close round: %v", closeErr)
			return res, nil
		}
		res.ClosedRound = closed
		return res, nil
	}

	if err != nil {
		return nil, err
	}
	return res, nil
}
```

Mirror the same block in `pull_action.go`, adding `ClosedRound` to `PullActionResult` and
`closeRound ICloseRound` to `PullActionUC`. `PullAction` never reports exhaustion itself (the
master named an action explicitly), so there the field simply stays nil — but the constructor
takes the collaborator for symmetry, and it is what the room already has to hand.

In `cmd/game/main.go`, pass the existing `CloseRoundUC` into both constructors. Build it once
and share it.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/application/match/ ./internal/app/game/ -v` then
`go vet -tags=integration ./internal/...`
Expected: PASS. Every existing `NewOpenNextActionUC(...)` / `NewPullActionUC(...)` call site
needs the second argument. There are three, and none of them may be missed:

- `cmd/game/main.go` — pass the shared `CloseRoundUC`.
- `internal/app/game/combat_e2e_test.go`, inside `newCombatFixture`, where both are handed
  positionally to `game.NewHandler`.
- `internal/application/match/open_next_action_test.go` / `pull_action_test.go`.

Tests that do not exercise the close path pass `nil`, which the nil guard above tolerates on
purpose.

- [ ] **Step 5: Commit**

```bash
git add internal/application/match/ cmd/game/main.go
git commit -m "feat(match): close an exhausted round through CloseRoundUC, which finally has a caller"
```

---

### Task 11: the master can turn the disputed turn on

**Files:**
- Create: `internal/application/match/change_round_mode.go`
- Create: `internal/application/match/change_round_mode_test.go`
- Modify: `internal/app/game/message.go`, `internal/app/game/room.go`, `cmd/game/main.go`
- Test: `internal/app/game/handler_test.go` (append)

**Interfaces:**
- Consumes: `RoundOrchestrator.SetMode` (Task 3).
- Produces:
  - `IChangeRoundMode` / `ChangeRoundModeUC` / `NewChangeRoundModeUC()`
  - `MsgTypeChangeRoundMode = "change_round_mode"` (client → server, master only)
  - `MsgTypeRoundModeChanged = "round_mode_changed"` (server → all)
  - `ChangeRoundModePayload{Mode string}`, `RoundModeChangedPayload{Mode string}`

**Why:** `enum.Race` has existed as a value with no path into it (`05-lacunas.md` §5). Every
rule in this phase is Race behaviour, so without this message the phase implements an economy
nobody can reach. Initiative — the game rule that will normally force Race — is a later slice;
this is the regime by itself.

- [ ] **Step 1: Write the failing test**

`internal/application/match/change_round_mode_test.go`:

```go
package match_test

import (
	"context"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/google/uuid"
)

func TestChangeRoundMode(t *testing.T) {
	masterUUID := uuid.New()
	uc := appmatch.NewChangeRoundModeUC()

	t.Run("the master turns the disputed turn on", func(t *testing.T) {
		s := freeSession(t)
		if err := uc.Execute(context.Background(), s, masterUUID, masterUUID, enum.Race); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.GetActiveRound().GetMode() != enum.Race {
			t.Error("the round must be racing")
		}
	})

	t.Run("and back off again", func(t *testing.T) {
		s := freeSession(t)
		_ = uc.Execute(context.Background(), s, masterUUID, masterUUID, enum.Race)
		if err := uc.Execute(context.Background(), s, masterUUID, masterUUID, enum.Free); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if s.GetActiveRound().GetMode() != enum.Free {
			t.Error("the round must be free again")
		}
	})

	t.Run("only the master", func(t *testing.T) {
		s := freeSession(t)
		err := uc.Execute(context.Background(), s, masterUUID, uuid.New(), enum.Race)
		if err == nil {
			t.Error("a player must not switch the regime")
		}
	})

	t.Run("an unknown mode is refused", func(t *testing.T) {
		s := freeSession(t)
		if err := uc.Execute(context.Background(), s, masterUUID, masterUUID, "Sprint"); err == nil {
			t.Error("only Free and Race exist")
		}
	})
}
```

Write `freeSession(t)` as a small helper returning a session with one participant.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/application/match/ -run TestChangeRoundMode -v`
Expected: FAIL — `undefined: appmatch.NewChangeRoundModeUC`.

- [ ] **Step 3: Write the implementation**

`internal/application/match/change_round_mode.go`:

```go
package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IChangeRoundMode interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, mode enum.RoundMode) error
}

// ChangeRoundModeUC switches the round between the free regime and the disputed one.
//
// Every rule in the bar economy — the price, the average, acting again, the carry-over — was
// designed looking at the disputed turn, and Race is the only regime with a written rule. This
// is the regime by itself; initiative, the game rule that will normally force it, is a later
// slice.
//
// Switching mid-round is allowed on purpose. The economy simply starts from that moment:
// nobody has acted yet as far as the bars are concerned, and the prices freeze on the next
// selection.
type ChangeRoundModeUC struct{}

func NewChangeRoundModeUC() *ChangeRoundModeUC { return &ChangeRoundModeUC{} }

func (uc *ChangeRoundModeUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	masterUUID, callerUUID uuid.UUID,
	mode enum.RoundMode,
) error {
	if callerUUID != masterUUID {
		return ErrNotMatchMaster
	}
	if mode != enum.Free && mode != enum.Race {
		return ErrInvalidRoundMode
	}
	session.SetRoundMode(mode)
	return nil
}
```

Add `ErrInvalidRoundMode = errors.New("round mode must be Free or Race")` to
`internal/application/match/error.go`, and to `match_session.go`:

```go
// SetRoundMode puts the active round into a regime. Callers hold room.mu for writing: this
// changes how every later selection is scored.
func (s *MatchSession) SetRoundMode(mode enum.RoundMode) {
	s.roundOrch.SetMode(s.activeRound, mode)
}
```

In `internal/app/game/message.go`:

```go
	MsgTypeChangeRoundMode MessageType = "change_round_mode"
```
beside the other client→server match messages, and
```go
	MsgTypeRoundModeChanged MessageType = "round_mode_changed"
```
beside the server→client ones, plus:

```go
// ChangeRoundModePayload asks to switch the round regime. Master only.
type ChangeRoundModePayload struct {
	Mode string `json:"mode"` // "Free" | "Race"
}

// RoundModeChangedPayload announces the new regime to the whole table. The regime is public:
// everyone needs to know whether the bars are running.
type RoundModeChangedPayload struct {
	Mode string `json:"mode"`
}
```

In `internal/app/game/room.go`, add the handler beside `MsgTypeOpenNextAction`:

```go
	case MsgTypeChangeRoundMode:
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
			return
		}
		var payload ChangeRoundModePayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid change_round_mode payload"))
			return
		}
		// Write lock across Execute — the regime decides how every later selection is scored.
		r.mu.Lock()
		session := r.session
		var err error
		if session != nil {
			err = r.changeRoundModeUC.Execute(
				context.Background(), session, r.masterUUID, client.userUUID,
				enum.RoundMode(payload.Mode),
			)
		}
		r.mu.Unlock()
		if session == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		out := NewServerMessage(MsgTypeRoundModeChanged, RoundModeChangedPayload{Mode: payload.Mode})
		data, _ := json.Marshal(out)
		go func() { r.broadcast <- data }()
```

Add the `changeRoundModeUC appmatch.IChangeRoundMode` field to `Room`, thread it through the
Room constructor the way `openNextActionUC` is threaded, and build it in `cmd/game/main.go`.

> ⚠️ `game.NewHandler` takes its use cases as a **positional list**, and this adds one. Every
> construction site has to grow an argument in the same position: `cmd/game/main.go`,
> `newCombatFixture` in `combat_e2e_test.go`, and the fog and handler test fixtures that build
> a handler. `go build ./...` finds them all; fix them in this commit rather than leaving the
> tree red.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/application/match/ ./internal/app/game/ -v` then
`go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/application/match/change_round_mode.go \
        internal/application/match/change_round_mode_test.go \
        internal/application/match/error.go \
        internal/domain/match/matchsession/match_session.go \
        internal/app/game/message.go internal/app/game/room.go cmd/game/main.go
git commit -m "feat(game): let the master turn the disputed turn on"
```

---

### Task 12: `round_closed` finally reaches the table

**Files:**
- Modify: `internal/app/game/room.go`
- Test: `internal/app/game/combat_e2e_test.go` (append)

**Interfaces:**
- Consumes: `OpenNextActionResult.ClosedRound` (Task 10).
- Produces: `MsgTypeRoundClosed` is broadcast with the existing `RoundClosedPayload{RoundMode}`.

**Why:** `MsgTypeRoundClosed` has been declared and never sent (`05-lacunas.md` §7). The phase's
second done-criterion is *"um round fecha sozinho quando as barras acabam, e `round_closed`
chega aos clients"*.

**Harness:** `combat_e2e_test.go` already has everything this needs — `newCombatFixture(t)`,
`f.connect(t)` (master first, then player), `sendWS`, `readMessage`, `connectWS`,
`newCollector`/`collector.await`. Use them. Do not build a second harness, and do not
`handleClientMessage` directly: the room's message path is the thing under test.

> ⚠️ `newCombatFixture` calls `game.NewHandler(...)` with a positional list of use cases, and
> Tasks 10 and 11 changed two of those constructors and added one. Update the fixture in the
> task that broke it, not here.

- [ ] **Step 1: Write the failing test**

```go
// TestE2E_AnExhaustedRoundClosesItself proves the second done-criterion: the round ends on
// its own when nothing pending can still pay, and the whole table is told.
func TestE2E_AnExhaustedRoundClosesItself(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	defer master.Close()
	defer player.Close()

	masterMsgs := newCollector(master)

	// Race, so the economy is on at all.
	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{
		Mode: string(enum.Race),
	})
	if !masterMsgs.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("the regime switch was never announced")
	}

	// One action, from the attacker. topFaceSource gives 10+10 = 20 on every check, so the
	// price is 20 and the single action pays for itself exactly.
	sendWS(t, player, string(game.MsgTypeEnqueueAction), game.ActionPayload{
		ActorID:  f.attackerID,
		TargetID: []uuid.UUID{f.victimID},
		Attack:   &game.AttackPayload{Hit: game.RollCheckPayload{SkillName: enum.Accuracy.String()}},
	})

	// First open consumes it; the second finds nothing that can pay.
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("the first action never opened")
	}
	sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})

	if !masterMsgs.await(game.MsgTypeRoundClosed, 2*time.Second) {
		t.Fatal("the round ran out and nobody was told")
	}

	t.Run("the payload names the regime that just ended", func(t *testing.T) {
		payload := awaitRoundClosed(t, masterMsgs)
		if payload.RoundMode != string(enum.Race) {
			t.Errorf("roundMode = %q, want %q", payload.RoundMode, enum.Race)
		}
	})

	t.Run("the players hear it too — the round is table state, not master state", func(t *testing.T) {
		playerMsgs := newCollector(player)
		_ = playerMsgs // the player collector is started at connect time in the real test
	})
}

// awaitRoundClosed pulls the round_closed payload out of what the collector already has.
func awaitRoundClosed(t *testing.T, c *collector) game.RoundClosedPayload {
	t.Helper()
	for _, m := range c.snapshotMessages() {
		if m.Type != game.MsgTypeRoundClosed {
			continue
		}
		var p game.RoundClosedPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal round_closed: %v", err)
		}
		return p
	}
	t.Fatal("no round_closed in the collected messages")
	return game.RoundClosedPayload{}
}
```

> Two notes for whoever writes this. First: start the **player's** collector at connect time,
> before anything is sent, or the broadcast is missed — the second sub-test above sketches the
> assertion but the collector has to exist earlier; move it up and assert
> `playerMsgs.await(game.MsgTypeRoundClosed, ...)`. Second: `collector` currently exposes
> `await` and `count` but not the messages themselves — add a small `snapshotMessages()` that
> returns a copy under the mutex, the same shape as the existing `snapshot()` on
> `recordingStatusWriter`.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run TestE2E_AnExhaustedRoundClosesItself -race -v`
Expected: FAIL — no `round_closed` is ever produced, and the second `open_next_action` panics or
errors on a nil `OpenedTurn`.

- [ ] **Step 3: Write the implementation**

In `internal/app/game/room.go`, inside the `MsgTypeOpenNextAction` case: the current code
dereferences `result.OpenedTurn` unconditionally. It must not, now that a successful call can
open nothing. Right after the `ClosedTurn` block:

```go
		// The round ran out: nothing pending could still pay, so it closed instead of opening
		// anything. Everyone is told — the regime and the bars are table state.
		if result.ClosedRound != nil {
			out := NewServerMessage(MsgTypeRoundClosed, RoundClosedPayload{
				RoundMode: string(result.ClosedRound.GetMode()),
			})
			data, _ := json.Marshal(out)
			go func() { r.broadcast <- data }()
			return
		}

		// Belt and braces: a successful call that opened nothing has nothing to announce.
		if result.OpenedTurn == nil {
			return
		}
```

Mirror the `result.OpenedTurn == nil` guard in the `MsgTypePullAction` case, which shares the
same shape.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/game/ -race -v` then `go vet -tags=integration ./internal/...`
Expected: PASS, including the Phase 2 e2e suite under `-race`.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/room.go internal/app/game/combat_e2e_test.go
git commit -m "feat(game): announce round_closed when the round runs out"
```

---

### Task 13: `bars_updated` — the table can see the clock

**Files:**
- Modify: `internal/app/game/message.go`, `internal/app/game/room.go`
- Test: `internal/app/game/resolution_payload_test.go` (append)

**Interfaces:**
- Consumes: `(*MatchSession).BarState`, `.RoundPrices`, `.PendingActions` (Tasks 7–8).
- Produces:
  - `MsgTypeBarsUpdated = "bars_updated"`
  - `BarsUpdatedPayload{Prices, Characters, Order}` and the row types below
  - `newBarsUpdatedPayload(session *matchsession.MatchSession) BarsUpdatedPayload`

**Why here:** Phase 6 has to draw "a sua barra e a barra geral" and nothing carries them today.
The event belongs to the phase where the bars start existing — it is no use for Phase 6 to
discover it has nothing to read from (spec §5).

**Visibility:** `combat-engine.md` § Ciclo — *"A fila é secreta; a barra e a ordem são
públicas."* So this goes to **everyone**, and it carries balances and the projected order as
`{actorId, bar, key}` — never an action ID, never any action content. That is exactly what the
game doc promises the general bar shows: the order of everyone, including who is going to act
more than once.

- [ ] **Step 1: Write the failing test**

```go
// racingSessionWithTwoActors builds a Race session holding two characters, enqueues one
// action for each, and opens the first — so the bar has a frozen price, one character with a
// recorded speed and one still pending.
func racingSessionWithTwoActors(t *testing.T) (*matchsession.MatchSession, uuid.UUID, uuid.UUID) {
	t.Helper()
	matchUUID := uuid.New()
	playerUUID := uuid.New()
	char1, char2 := uuid.New(), uuid.New()
	participants := []*match.Participant{
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: char1, PlayerUUID: &playerUUID}},
		{UUID: uuid.New(), MatchUUID: matchUUID, Sheet: csEntity.Summary{UUID: char2, PlayerUUID: &playerUUID}},
	}
	sheets := map[uuid.UUID]*csSheet.CharacterSheet{
		char1: newCombatSheet(t),
		char2: newCombatSheet(t),
	}
	s := matchsession.NewMatchSession(matchUUID, sheets, participants)
	s.GetActiveRound().SetMode(enum.Race)
	s.SetRollSource(topFaceSource{})

	for _, charID := range []uuid.UUID{char1, char2} {
		a := action.NewAction(charID, nil, uuid.Nil, nil, action.ActionSpeed{},
			nil, nil, &action.Attack{}, nil, nil, nil, nil)
		if err := s.EnqueueAction(playerUUID, a); err != nil {
			t.Fatalf("EnqueueAction: %v", err)
		}
	}
	if _, err := s.OpenNextAction(); err != nil {
		t.Fatalf("OpenNextAction: %v", err)
	}
	return s, char1, char2
}

func TestNewBarsUpdatedPayload(t *testing.T) {
	session, p1, p2 := racingSessionWithTwoActors(t)

	payload := newBarsUpdatedPayload(session)

	t.Run("carries the frozen price of each bar that priced", func(t *testing.T) {
		if _, ok := payload.Prices[string(action.BarAction)]; !ok {
			t.Error("the action bar priced and the table must see it")
		}
	})

	t.Run("carries one row per character, both bars", func(t *testing.T) {
		if len(payload.Characters) != 2 {
			t.Fatalf("characters = %d, want 2", len(payload.Characters))
		}
		for _, c := range payload.Characters {
			if c.CharacterID != p1 && c.CharacterID != p2 {
				t.Errorf("unexpected character %v", c.CharacterID)
			}
		}
	})

	t.Run("carries the projected order, and nothing that identifies an action", func(t *testing.T) {
		if len(payload.Order) == 0 {
			t.Fatal("something is pending, so the general bar has an order to show")
		}
		// Keys descend: this IS the order the master will open them in.
		for i := 1; i < len(payload.Order); i++ {
			if payload.Order[i-1].Key < payload.Order[i].Key {
				t.Error("the order must be sorted by key, highest first")
			}
		}
		raw, err := json.Marshal(payload.Order[0])
		if err != nil {
			t.Fatal(err)
		}
		for _, leak := range []string{"actionId", "attack", "skill", "target"} {
			if bytes.Contains(bytes.ToLower(raw), []byte(strings.ToLower(leak))) {
				t.Errorf("the queue is secret — only the bar and the order are public; found %q in %s", leak, raw)
			}
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run TestNewBarsUpdatedPayload -v`
Expected: FAIL — `undefined: newBarsUpdatedPayload`.

- [ ] **Step 3: Write the implementation**

In `internal/app/game/message.go`:

```go
	MsgTypeBarsUpdated MessageType = "bars_updated"
```

```go
// BarsUpdatedPayload is the two clocks as the whole table sees them.
//
// It is BROADCAST, not projected per recipient, and that is deliberate: combat-engine.md says
// "A fila é secreta; a barra e a ordem são públicas". A player who cannot see the general bar
// only finds out it was their turn after it passed.
//
// Order therefore carries who acts next and on which bar — and NOTHING that identifies the
// action itself. No action ID, no weapon, no target, no skill. Those belong to the master
// until the turn opens.
type BarsUpdatedPayload struct {
	// Prices maps a bar name to its frozen round price. A bar that has not priced is absent.
	Prices map[string]int `json:"prices"`
	// Characters is every character's standing balance on both bars.
	Characters []CharacterBarsPayload `json:"characters"`
	// Order is the projection of who acts next, highest key first.
	Order []BarSlotPayload `json:"order"`
}

type CharacterBarsPayload struct {
	CharacterID uuid.UUID `json:"characterId"`
	// ActionBalance and MoveBalance are the standing credit or debt on each clock. They are
	// fractional: the average behind them rarely divides evenly, and the fraction is kept
	// rather than rounded away.
	ActionBalance float64 `json:"actionBalance"`
	MoveBalance   float64 `json:"moveBalance"`
	// ActionSpeeds and MoveSpeeds are the speeds that have already acted this round. They
	// are public because the average they produce is what everyone is being ordered by.
	ActionSpeeds []int `json:"actionSpeeds"`
	MoveSpeeds   []int `json:"moveSpeeds"`
}

// BarSlotPayload is one slot of the general bar: who acts, which clocks it charges, and where
// in the round it lands. A combined action reports both bars and a single key — the one from
// its slower half.
type BarSlotPayload struct {
	ActorID uuid.UUID `json:"actorId"`
	Bars    []string  `json:"bars"`
	Key     float64   `json:"key"`
}
```

In `internal/app/game/room.go`, the builder plus its broadcast:

```go
// newBarsUpdatedPayload snapshots both clocks for the whole table.
//
// The caller holds r.mu — it reads session state that every open and every enqueue mutates.
func newBarsUpdatedPayload(session *matchsession.MatchSession) BarsUpdatedPayload {
	prices := map[string]int{}
	for bar, price := range session.RoundPrices() {
		prices[string(bar)] = price
	}

	out := BarsUpdatedPayload{Prices: prices}
	for _, charID := range session.CharacterIDs() {
		actionCarry, actionSpeeds := session.BarState(charID, action.BarAction)
		moveCarry, moveSpeeds := session.BarState(charID, action.BarMove)
		out.Characters = append(out.Characters, CharacterBarsPayload{
			CharacterID:   charID,
			ActionBalance: actionCarry,
			MoveBalance:   moveCarry,
			ActionSpeeds:  append([]int(nil), actionSpeeds...),
			MoveSpeeds:    append([]int(nil), moveSpeeds...),
		})
	}

	for _, slot := range session.ProjectedOrder() {
		bars := make([]string, 0, len(slot.Bars))
		for _, b := range slot.Bars {
			bars = append(bars, string(b))
		}
		out.Order = append(out.Order, BarSlotPayload{
			ActorID: slot.ActorID,
			Bars:    bars,
			Key:     slot.Key,
		})
	}
	return out
}

// broadcastBars publishes the clocks to everyone. Called after anything that moves them:
// an action enqueued, a turn opened, a round closed, a regime switched.
func (r *Room) broadcastBars(session *matchsession.MatchSession) {
	if session == nil {
		return
	}
	r.mu.RLock()
	payload := newBarsUpdatedPayload(session)
	r.mu.RUnlock()
	data, _ := json.Marshal(NewServerMessage(MsgTypeBarsUpdated, payload))
	go func() { r.broadcast <- data }()
}
```

And in `matchsession`:

```go
// BarSlot is one entry of the projected order: who acts, on which clock, and at what key.
type BarSlot struct {
	ActorID uuid.UUID
	Bar     action.Bar
	Key     int
}

// CharacterIDs returns every character the session holds combat state for, NPCs included.
func (s *MatchSession) CharacterIDs() []uuid.UUID {
	out := make([]uuid.UUID, 0, len(s.statuses))
	for id := range s.statuses {
		out = append(out, id)
	}
	return out
}

// ProjectedOrder is the general bar: the pending actions that can still pay, highest key
// first. It carries no action identity — the queue is secret, the order is public.
func (s *MatchSession) ProjectedOrder() []service.OrderSlot {
	return s.scheduler.ProjectOrder(s.scheduleInput())
}
```

`RoundScheduler.ProjectOrder` and `service.OrderSlot` already exist — Task 5 built them.

Call `r.broadcastBars(session)` at the end of the `MsgTypeEnqueueAction`, `MsgTypeOpenNextAction`,
`MsgTypePullAction` and `MsgTypeChangeRoundMode` handlers — **after** each has released `r.mu`,
since `broadcastBars` takes the read lock itself.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/game/ ./internal/domain/match/... -race -v` then
`go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/message.go internal/app/game/room.go \
        internal/app/game/resolution_payload_test.go \
        internal/domain/match/matchsession/match_session.go \
        internal/domain/match/service/round_scheduler.go
git commit -m "feat(game): publish both clocks and the projected order to the table"
```

---

### Task 14: a whole racing round, end to end over a real WebSocket

**Files:**
- Modify: `internal/app/game/combat_e2e_test.go`

**Interfaces:**
- Consumes: everything above.
- Produces: nothing — this is the phase's evidence.

**Why:** Task 5 proves the arithmetic and Tasks 7–9 prove the session. Neither proves that a
player's message actually reaches the economy. Task 12 proves the round ends; this proves the
*order* and the *carry-over* come out of the bars over a real socket.

**Harness:** `newCombatFixture` already gives two characters — `f.attackerID` and `f.victimID`
— and both belong to `f.playerUUID`, so one player connection can enqueue for both. That is
exactly what this needs: two actors, one client.

`newCombatFixture` pins `topFaceSource{}`, which returns the top face of every die. Replace it
per test with a scripted source. Add to the fixture:

```go
// setRollSource replaces the session's dice for one test. The session pointer is the
// fixture's own, and nothing is in flight when a test calls this.
func (f *combatFixture) setRollSource(src service.RollSource) { f.session.SetRollSource(src) }
```

which needs `newCombatFixture` to keep the session on the fixture (`f.session = session`) — it
builds one and throws the reference away today.

- [ ] **Step 1: Write the failing test**

```go
// TestE2E_ARacingRoundRunsOnTheBars drives a whole Race round over real WebSockets: two
// characters enqueue, the master opens, the order comes out of the bars rather than out of
// insertion, and the balances that cross into the next round are the ones the rules say.
//
// The dice are scripted, so the numbers are exact instead of lucky.
func TestE2E_ARacingRoundRunsOnTheBars(t *testing.T) {
	f := newCombatFixture(t)
	// Legerity is 0 on a factory sheet, so the actionSpeed IS the dice total. Four faces per
	// check (both attempt sets fall up front): the attacker gets 3+3 = 6, the victim 8+8 = 16.
	f.setRollSource(&scriptedFaces{faces: []int{3, 3, 3, 3, 8, 8, 8, 8}})

	master, player := f.connect(t)
	defer master.Close()
	defer player.Close()
	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	sendWS(t, master, string(game.MsgTypeChangeRoundMode), game.ChangeRoundModePayload{
		Mode: string(enum.Race),
	})
	if !masterMsgs.await(game.MsgTypeRoundModeChanged, 2*time.Second) {
		t.Fatal("the regime switch was never announced")
	}

	enqueue := func(actor uuid.UUID) {
		t.Helper()
		sendWS(t, player, string(game.MsgTypeEnqueueAction), game.ActionPayload{
			ActorID: actor,
			Attack:  &game.AttackPayload{Hit: game.RollCheckPayload{SkillName: enum.Accuracy.String()}},
		})
	}
	enqueue(f.attackerID) // 6
	enqueue(f.victimID)   // 16

	t.Run("bars_updated announces the order before anything opens", func(t *testing.T) {
		if !playerMsgs.await(game.MsgTypeBarsUpdated, 2*time.Second) {
			t.Fatal("the general bar is public and must reach the players")
		}
		bars := lastBarsUpdated(t, playerMsgs)
		if len(bars.Order) != 2 {
			t.Fatalf("order = %d slots, want 2", len(bars.Order))
		}
		if bars.Order[0].ActorID != f.victimID {
			t.Error("the character who rolled 16 leads the general bar, not the one inserted first")
		}
	})

	t.Run("the master opens, and the faster character goes first", func(t *testing.T) {
		sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
		if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
			t.Fatal("nothing opened")
		}
		opened := lastTurnOpened(t, masterMsgs)
		if opened.ActorID != f.victimID {
			t.Errorf("actor = %v, want the faster character %v", opened.ActorID, f.victimID)
		}
	})

	t.Run("the price froze at the slowest pending speed", func(t *testing.T) {
		bars := lastBarsUpdated(t, playerMsgs)
		if got := bars.Prices[string(action.BarAction)]; got != 6 {
			t.Errorf("price = %d, want the slowest pending speed, 6", got)
		}
	})

	t.Run("the second open takes the slower character", func(t *testing.T) {
		sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
		if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
			t.Fatal("the second action never opened")
		}
		opened := lastTurnOpened(t, masterMsgs)
		if opened.ActorID != f.attackerID {
			t.Errorf("actor = %v, want %v", opened.ActorID, f.attackerID)
		}
	})

	t.Run("the third open finds nothing that can pay, and the round closes itself", func(t *testing.T) {
		sendWS(t, master, string(game.MsgTypeOpenNextAction), struct{}{})
		if !masterMsgs.await(game.MsgTypeRoundClosed, 2*time.Second) {
			t.Fatal("the round ran out and nobody was told")
		}
	})

	t.Run("the carry crossed over, ceiling applied", func(t *testing.T) {
		bars := lastBarsUpdated(t, playerMsgs)
		byChar := map[uuid.UUID]game.CharacterBarsPayload{}
		for _, c := range bars.Characters {
			byChar[c.CharacterID] = c
		}
		// The fast one kept 16 − 6 = 10, clipped to the ceiling of 6.
		if got := byChar[f.victimID].ActionBalance; got != 6 {
			t.Errorf("fast balance = %d, want 6 — a leftover of 10 is clipped to the round price", got)
		}
		// The slowest of the round starts the next one from zero.
		if got := byChar[f.attackerID].ActionBalance; got != 0 {
			t.Errorf("slow balance = %d, want 0", got)
		}
		if len(byChar[f.attackerID].ActionSpeeds) != 0 {
			t.Error("the round's speed history is cleared; only the balance crosses over")
		}
	})
}
```

`lastBarsUpdated` and `lastTurnOpened` are two-line readers over `collector.snapshotMessages()`
(added in Task 12), in the same shape as `awaitRoundClosed`.

> Check the arithmetic before implementing. Both actions are on the action bar. Price = 6.
> The fast character's leftover after acting is `16 − 6 = 10`, which is `>= 6`, so it *would*
> be eligible for a second action — but it has nothing queued, and **a character with credit
> and no pending action does not hold the round**. That is precisely why the third open finds
> nothing, and it is the clause worth having an e2e prove.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run TestE2E_ARacingRound -race -v`
Expected: FAIL on the first assertion whose plumbing is missing.

- [ ] **Step 3: Fix whatever it catches**

No new production code should be needed. If some is, it is a real gap between the domain and
the delivery layer — fix it there, and say in the commit body what the e2e caught that the unit
tests did not.

- [ ] **Step 4: Run the whole suite**

Run: `go test ./... -race` and `go vet -tags=integration ./internal/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/combat_e2e_test.go
git commit -m "test(game): drive a whole racing round over a real WebSocket"
```

---

### Task 15: write down what Phase 3 fixed

**Files:**
- Modify: `docs/dev/match/combat-engine.md`
- Modify: `docs/dev/match/turns-rounds.md`
- Modify: `docs/dev/match/flows/05-lacunas.md`
- Modify: `docs/documentation-map.yaml`
- Modify: `docs/superpowers/specs/2026-08-16-combat-engine-design.md`

**Why:** the repo's convention (`docs-workflow.instructions.md`, and what Phase 2 did) is that a
phase ends by recording what it fixed, so the next one reads the code's current shape rather
than the spec's intent.

- [ ] **Step 1: `combat-engine.md` — a "O que a Fase 3 fixou no motor" section**

Beside the existing "O que a Fase 2 fixou no motor". Cover, one short subsection each:

- **A chave não mora na action.** `PriorityQueue` stopped being a heap; the key is computed at
  selection time by `service.RoundScheduler`, because it is character state that moves with the
  average. Name the file.
- **Os dois porteiros em código.** `service.BarEconomy.IsEligible` — first action against the
  bar, second onward against the leftover — and `Key`, which orders and never gates.
- **Onde o preço mora.** `Round.prices`, one per bar, frozen by `RoundScheduler.FreezePrices` at
  the first selection that sees pending work on that bar, idempotent thereafter.
- **`Speeds` são as que agiram.** Recorded on open, never on enqueue. State what that buys: an
  action that never reached the price rolls over whole with nothing to unwind.
- **A média arredonda para baixo.** `BarEconomy.Mean`, one function, floor, and why.
- **A ação combinada é UMA action.** `Move` and `Attack` in the same `Action`, charging both
  bars, opened once, scheduled at `min` of the two keys and gated on both. Say explicitly that
  the earlier "duas resoluções com aresta de dependência" was discarded and why.
- **`Race` é alcançável.** `change_round_mode`, master only; initiative is still a later slice.
- **O round fecha sozinho.** The predicate, `TurnTransition.RoundExhausted`, and `CloseRoundUC`
  finally having a caller.
- **As barras são públicas.** `bars_updated`, broadcast, carrying balances and the projected
  order and nothing that identifies an action.

- [ ] **Step 2: `turns-rounds.md`** — bring the round/turn lifecycle up to date: prices, the
  auto-close, and the two new WS messages.

- [ ] **Step 3: `05-lacunas.md`** — mark §2 ("A fila de prioridade não tem prioridade") and §5
  ("Iniciativa e modo `Race` não estão ligados", partially: the regime is reachable, initiative
  is not) as resolved with a ✅ and the Phase-3 note, in the same style Phase 2 used. In §7
  ("Buracos no ciclo do round"), strike the first two bullets and leave the third
  (`MatchSession.CloseTurn()` still has no caller — explicit turn close is Phase 5).

- [ ] **Step 4: `documentation-map.yaml`** — entries for the new code paths:
  `internal/domain/match/service/bar_economy.go`, `.../round_scheduler.go`,
  `internal/domain/match/entity/action/bar.go`,
  `internal/application/match/change_round_mode.go`, each pointing at
  `docs/dev/match/combat-engine.md` (`directly_affected`) and
  `docs/game/combate/barra-de-acao.md` where the rule is player-facing.

- [ ] **Step 5: the spec** — in §5 Fase 3, mark the phase implemented the way Phase 2 was
  marked, and fix the two wording leftovers the corrections did not reach:
  - the scope bullet still says *"Fim de round quando as barras acabam"*; the predicate is
    "nenhuma action pendente passa no porteiro que lhe cabe", and `combat-engine.md` is
    explicit that it is **not** "quando as barras acabam";
  - the same phrase appears in the done-criteria.
  Both are wording, not substance — the rule is settled. Point at `combat-engine.md` as the
  source, exactly as the composed-actions bullet already does.

- [ ] **Step 6: Commit**

```bash
git add docs/
git commit -m "docs(match): record what phase 3 fixed in the engine"
```

---

## Done — the phase's own criteria

Both come straight from the spec §5, Fase 3.

1. **The canonical example reproduces, with rolls injected.**
   `TestRoundScheduler_CanonicalExample` (Task 5): p1 = 20, p2 = 23, p3 = 11, p2's second roll
   17 → order `p2 → p1 → p3 → p2` and balances `+9 / 0 / −2`. Nothing in it touches a die.
2. **A round closes by itself and `round_closed` reaches the clients.**
   `TestRoom_AnnouncesRoundClosed` (Task 12) and `TestE2E_ARacingRoundRunsOnTheBars` (Task 14).

**Before opening the PR**, per `CLAUDE.md` § Entrega:

- `go test ./... -race` and `go vet -tags=integration ./internal/...` clean.
- **Prove it running.** `make run-dev`, then a curl/WS smoke: connect as master, send
  `change_round_mode` with `Race`, enqueue two actions from two players, open twice, and watch
  `bars_updated` and `round_closed` come back. The front does not consume any of this yet
  (Phase 6), so the browser is not the evidence here — the socket is. Say exactly that in the PR.
- The PR states what was verified and what was not. Nothing in this phase is pixel-tuned or
  Pixi-adjacent, so there is no browser-only surface to hand off.
- `./dev-checkout.sh feat/combat-engine-phase-3` only if something was left unverified.

---

## Self-review notes

Checked against the spec §5 (Fase 3) and `combat-engine.md`, section by section:

| Spec requirement | Task |
|---|---|
| actionSpeed feeding `Speed.Result`; base skill `Legerity` | 6, 7 |
| moveSpeed from the category; Dash/Accelerate, Shift/Brake passive | 6, 7 |
| Only Dash and Shift accepted; the other five refused | 6 |
| `Charge` out; the second speed regime out | 7 (stated, not built) |
| Two bars, price per bar, average, carry-over, ceiling | 2, 3, 9 |
| Forward-only recalculation when the average moves | 4, 5 (the key is recomputed, never stored) |
| A mid-round action enters at its rolled speed, no retroactive reordering | 5 |
| `RoundMode.Race` reachable | 11 |
| Round ends when nothing pending passes its gate; `CloseRoundUC` plugged in; `round_closed` emitted | 5, 9, 10, 12 |
| Composed actions: ONE action charging both bars, at the time of the slower half | 1, 5, 7 |
| Free's third-action lock is a later slice | — (explicitly out) |
| The rename is a separate PR, before this one | — (prerequisite, stated in the header) |
| The WS event carrying the bars is born here | 13 |
| Done-criterion: canonical example, rolls injected | 5 |
| Done-criterion: a round closes itself, `round_closed` reaches the clients | 12, 14 |
