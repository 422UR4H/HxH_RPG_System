package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

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
			name    string
			dice    []int
			hit     int
			rung    service.LadderRung
			avoided bool
			payout  int
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
