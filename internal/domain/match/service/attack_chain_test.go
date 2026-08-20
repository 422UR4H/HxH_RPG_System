package service_test

import (
	"testing"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

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
