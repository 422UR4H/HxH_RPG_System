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
		if !cr.Avoided {
			t.Errorf("expected the dodge to stop a hit of %d against a passive 11", cr.Hit.Total)
		}
		if cr.EffectiveDamage != 0 {
			t.Errorf("EffectiveDamage = %d, want 0 when the attack was dodged", cr.EffectiveDamage)
		}
	})

	t.Run("a hit past the dodge is defended, and an armed attack is not reduced", func(t *testing.T) {
		// Hit 10 + 8 = 18 beats the passive dodge of 11. The passive defense is also 11,
		// and its CD is 18 - 10 = 8, so it succeeds. An armed attack against a bare-handed
		// defense subtracts nothing while damage types do not exist, so the sword's
		// 9 + 3 + 2 = 14 passes whole.
		tn := attackTurn(actorID, targetID, []int{10, 8}, []int{9, 3}, &sword)
		res := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, tn))

		cr := res.CharacterResults[0]
		if cr.Avoided {
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

	t.Run("an unarmed attack is reduced by the defense instead of blocked", func(t *testing.T) {
		// Bare hands are the Fist entry: D6 + D6 + D4, flat damage 0. Its defense bonus is
		// what subtracts — 0 in today's catalogue, so the shape is what this asserts.
		tn := attackTurn(actorID, targetID, []int{10, 8}, []int{5, 4, 2}, nil)
		res := service.TurnResolver{}.Resolve(resolveInput(t, actorID, targetID, tn))

		cr := res.CharacterResults[0]
		if cr.RawDamage != 11 {
			t.Errorf("RawDamage = %d, want 11 (5 + 4 + 2 dice, 0 flat)", cr.RawDamage)
		}
		if cr.EffectiveDamage != 11-cr.DefenseApplied {
			t.Errorf("EffectiveDamage = %d, want raw minus the applied defense", cr.EffectiveDamage)
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

	t.Run("a missing sheet is skipped rather than resolved with zeros", func(t *testing.T) {
		tn := attackTurn(actorID, targetID, []int{10, 8}, []int{9, 3}, &sword)
		in := resolveInput(t, actorID, targetID, tn)
		delete(in.Sheets, targetID)

		res := service.TurnResolver{}.Resolve(in)

		if len(res.CharacterResults) != 0 {
			t.Errorf("expected no character results without the target's sheet, got %d", len(res.CharacterResults))
		}
	})
}

// TestResolve_SurfacesEngineFaults pins the two failures the resolver used to swallow whole.
//
// Both are engine faults, not game outcomes: an action whose target the engine cannot
// classify, and a collision it cannot compute because a sheet it was handed is missing. The
// old code dropped each on the floor — an empty `case TargetKindUnknown` and a bare
// `return false` — so a target that evaporated and a target that was never resolved looked
// identical to every caller, and identical to a turn where nothing was targeted at all.
func TestResolve_SurfacesEngineFaults(t *testing.T) {
	resolver := service.TurnResolver{}

	t.Run("an unclassifiable target is reported, not dropped", func(t *testing.T) {
		actorID, ghostID := uuid.New(), uuid.New()
		tn := attackTurn(actorID, ghostID, []int{9, 9}, []int{5, 5}, nil)

		res := resolver.Resolve(service.ResolveInput{
			Turn:    tn,
			Sheets:  map[uuid.UUID]*csSheet.CharacterSheet{actorID: plainSheet(t)},
			Targets: charTargets{chars: map[uuid.UUID]bool{}}, // knows nobody → Unknown
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})

		if len(res.Errors) != 1 {
			t.Fatalf("Errors = %+v, want exactly 1 unknown-target fault", res.Errors)
		}
		if res.Errors[0].Kind != service.ResolutionErrUnknownTarget {
			t.Errorf("Kind = %q, want %q", res.Errors[0].Kind, service.ResolutionErrUnknownTarget)
		}
		if res.Errors[0].Subject != ghostID {
			t.Errorf("Subject = %s, want the target the engine could not classify (%s)",
				res.Errors[0].Subject, ghostID)
		}
	})

	t.Run("a missing target sheet is reported, not dropped", func(t *testing.T) {
		actorID, targetID := uuid.New(), uuid.New()
		tn := attackTurn(actorID, targetID, []int{9, 9}, []int{5, 5}, nil)

		res := resolver.Resolve(service.ResolveInput{
			Turn:   tn,
			Sheets: map[uuid.UUID]*csSheet.CharacterSheet{actorID: plainSheet(t)}, // target absent
			// The target IS classified as a character — it is on the board. Only its sheet
			// is missing, which is why this is a different fault from the one above.
			Targets: charTargets{chars: map[uuid.UUID]bool{targetID: true}},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})

		if len(res.CharacterResults) != 0 {
			t.Fatalf("a target with no sheet produced a result: %+v", res.CharacterResults)
		}
		if len(res.Errors) != 1 {
			t.Fatalf("Errors = %+v, want exactly 1 missing-sheet fault", res.Errors)
		}
		if res.Errors[0].Kind != service.ResolutionErrMissingSheet {
			t.Errorf("Kind = %q, want %q", res.Errors[0].Kind, service.ResolutionErrMissingSheet)
		}
		if res.Errors[0].Subject != targetID {
			t.Errorf("Subject = %s, want the sheet that was missing (%s)",
				res.Errors[0].Subject, targetID)
		}
	})

	t.Run("a missing ACTOR sheet names the actor", func(t *testing.T) {
		actorID, targetID := uuid.New(), uuid.New()
		tn := attackTurn(actorID, targetID, []int{9, 9}, []int{5, 5}, nil)

		res := resolver.Resolve(service.ResolveInput{
			Turn:    tn,
			Sheets:  map[uuid.UUID]*csSheet.CharacterSheet{targetID: plainSheet(t)}, // actor absent
			Targets: charTargets{chars: map[uuid.UUID]bool{targetID: true}},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})

		if len(res.Errors) != 1 {
			t.Fatalf("Errors = %+v, want exactly 1 missing-sheet fault", res.Errors)
		}
		if res.Errors[0].Subject != actorID {
			t.Errorf("Subject = %s, want the ACTOR whose sheet was missing (%s)",
				res.Errors[0].Subject, actorID)
		}
	})

	// The actor is ONE character, shared by every target in the chain walk. Reporting their
	// missing sheet once per target would hand the master N copies of one fact.
	t.Run("a missing ACTOR sheet is reported once, not once per target", func(t *testing.T) {
		actorID := uuid.New()
		t1, t2, t3 := uuid.New(), uuid.New(), uuid.New()
		tn := attackTurn(actorID, t1, []int{9, 9}, []int{5, 5}, nil)
		act := tn.GetAction()
		act.TargetID = []uuid.UUID{t1, t2, t3}
		tn = turn.NewTurn(act)

		res := resolver.Resolve(service.ResolveInput{
			Turn: tn,
			Sheets: map[uuid.UUID]*csSheet.CharacterSheet{
				t1: plainSheet(t), t2: plainSheet(t), t3: plainSheet(t),
			}, // only the actor is absent
			Targets: charTargets{chars: map[uuid.UUID]bool{t1: true, t2: true, t3: true}},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})

		if len(res.Errors) != 1 {
			t.Fatalf("Errors = %+v, want exactly 1 — one actor, one fault", res.Errors)
		}
		if res.Errors[0].Subject != actorID || res.Errors[0].Kind != service.ResolutionErrMissingSheet {
			t.Errorf("Errors[0] = %+v, want a missing_sheet naming the actor %s",
				res.Errors[0], actorID)
		}
	})

	t.Run("a clean resolution reports no faults", func(t *testing.T) {
		actorID, targetID := uuid.New(), uuid.New()
		tn := attackTurn(actorID, targetID, []int{9, 9}, []int{5, 5}, nil)

		res := resolver.Resolve(resolveInput(t, actorID, targetID, tn))

		if len(res.Errors) != 0 {
			t.Fatalf("a clean resolution reported faults: %+v", res.Errors)
		}
	})
}
