package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestProjectResolution(t *testing.T) {
	owner, third := uuid.New(), uuid.New()

	base := func() *service.TurnResolution {
		return &service.TurnResolution{
			IsSettled:    true,
			ActionResult: service.RollResult{SkillName: "Legerity", Total: 18, DiceRolled: []int{9, 9}},
			CharacterResults: []service.CharacterResult{{
				TargetID:        owner,
				ReactionKind:    string(action.ReactClosedDodge),
				ReactionTotal:   15,
				RawDamage:       7,
				EffectiveDamage: 0,
				Payouts: []match.Modifier{{
					Amount: 4, Applies: match.DimDodge, Source: match.SourceSystem,
					Against: match.ScopeAllBut(uuid.New()), ExpiresAt: match.LifetimeEndOfRound,
					Reason: "closed dodge reserve",
				}},
			}},
			PendingReactions: []service.PendingReaction{{ReactionID: uuid.New(), ActorID: owner, Kind: "dodge"}},
		}
	}

	t.Run("the master sees the closed dodge as a closed dodge", func(t *testing.T) {
		got := service.ProjectResolution(base(), service.Viewer{IsMaster: true})
		if got.CharacterResults[0].ReactionKind != string(action.ReactClosedDodge) {
			t.Fatalf("ReactionKind = %q, want closedDodge", got.CharacterResults[0].ReactionKind)
		}
		if len(got.CharacterResults[0].Payouts) != 1 {
			t.Fatal("the master lost the reserve")
		}
		if len(got.PendingReactions) != 1 {
			t.Fatal("the master lost their own to-do list")
		}
	})

	t.Run("the owner sees their own closed dodge", func(t *testing.T) {
		v := service.Viewer{Owns: map[uuid.UUID]bool{owner: true}}
		got := service.ProjectResolution(base(), v)
		if got.CharacterResults[0].ReactionKind != string(action.ReactClosedDodge) {
			t.Fatalf("the owner was lied to about their own reaction: %q",
				got.CharacterResults[0].ReactionKind)
		}
		if len(got.CharacterResults[0].Payouts) != 1 {
			t.Fatal("the owner lost their own reserve")
		}
	})

	t.Run("a third party sees a plain dodge, and no reserve", func(t *testing.T) {
		v := service.Viewer{Owns: map[uuid.UUID]bool{third: true}}
		got := service.ProjectResolution(base(), v)
		if got.CharacterResults[0].ReactionKind != string(action.ReactDodge) {
			t.Fatalf("ReactionKind = %q, want dodge — the label is the leak",
				got.CharacterResults[0].ReactionKind)
		}
		if len(got.CharacterResults[0].Payouts) != 0 {
			t.Fatal("a third party can see the closed dodge's reserve")
		}
		if len(got.PendingReactions) != 0 {
			t.Fatal("a third party can read the master's to-do list")
		}
	})

	t.Run("the numbers stay public — deduction needs them", func(t *testing.T) {
		v := service.Viewer{Owns: map[uuid.UUID]bool{third: true}}
		got := service.ProjectResolution(base(), v)
		if got.ActionResult.Total != 18 || len(got.ActionResult.DiceRolled) != 2 {
			t.Fatal("the attack's numbers were hidden; the opponent has nothing to deduce from")
		}
		if got.CharacterResults[0].RawDamage != 7 {
			t.Fatal("damage is public")
		}
	})

	t.Run("closed escape reaches a third party as escape", func(t *testing.T) {
		res := base()
		res.CharacterResults[0].ReactionKind = string(action.ReactClosedEscape)
		v := service.Viewer{Owns: map[uuid.UUID]bool{third: true}}
		got := service.ProjectResolution(res, v)
		if got.CharacterResults[0].ReactionKind != string(action.ReactEscape) {
			t.Fatalf("ReactionKind = %q, want escape", got.CharacterResults[0].ReactionKind)
		}
	})

	t.Run("projecting does not mutate the original", func(t *testing.T) {
		res := base()
		v := service.Viewer{Owns: map[uuid.UUID]bool{third: true}}
		_ = service.ProjectResolution(res, v)
		if res.CharacterResults[0].ReactionKind != string(action.ReactClosedDodge) {
			t.Fatal("ProjectResolution mutated its input — the master's copy is now a lie")
		}
	})
}

func TestProjectAction(t *testing.T) {
	owner, third := uuid.New(), uuid.New()
	mk := func() action.Action {
		a := action.NewAction(owner, nil, uuid.New(),
			[]action.Skill{
				{SkillName: enum.Evasion.String()},
				{SkillName: enum.Legerity.String()},
			},
			action.ActionSpeed{}, &action.RollCheck{SkillName: enum.Legerity.String()},
			nil, nil, nil, &action.Dodge{}, nil, nil)
		a.ReactionKind = action.ReactClosedDodge
		return *a
	}

	t.Run("a third party sees neither the feint nor the evasion entry", func(t *testing.T) {
		got := service.ProjectAction(mk(), service.Viewer{Owns: map[uuid.UUID]bool{third: true}})
		if got.Feint != nil {
			t.Fatal("a revealed feint is not a feint")
		}
		for _, s := range got.Skills {
			if s.SkillName == enum.Evasion.String() {
				t.Fatal("the evasion entry leaked")
			}
		}
		if got.ReactionKind != action.ReactDodge {
			t.Fatalf("ReactionKind = %q, want dodge", got.ReactionKind)
		}
	})

	t.Run("the owner keeps everything", func(t *testing.T) {
		got := service.ProjectAction(mk(), service.Viewer{Owns: map[uuid.UUID]bool{owner: true}})
		if got.Feint == nil || len(got.Skills) != 2 || got.ReactionKind != action.ReactClosedDodge {
			t.Fatal("the owner was projected away from their own action")
		}
	})
}
