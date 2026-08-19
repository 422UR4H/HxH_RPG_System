package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// attempts builds a RollAttempts with both sets, so a Derive test can name exact numbers
// without going through Roll at all. Roll itself has a seam now — see RollSource.
func attempts(primary, secondary []int) action.RollAttempts {
	return action.RollAttempts{Primary: primary, Secondary: secondary}
}

func TestRollCalculator_Roll(t *testing.T) {
	calc := service.RollCalculator{}

	t.Run("2d10 rolls two dice in both sets", func(t *testing.T) {
		got := calc.Roll(match.NewDefaultMatchRules(), nil)

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

		got := calc.Roll(rules, nil)

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
		ledger.Add(match.Modifier{Bias: -1, Applies: match.DimActionSpeed, Source: match.SourceSystem, ExpiresAt: match.LifetimeEndOfTurn})

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
		ledger.Add(match.Modifier{Bias: -1, Applies: match.DimActionSpeed, Source: match.SourceSystem, ExpiresAt: match.LifetimeEndOfTurn})
		ledger.Add(match.Modifier{Bias: -1, Applies: match.DimActionSpeed, Source: match.SourceSystem, ExpiresAt: match.LifetimeEndOfTurn})

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
		Amount: 5, Applies: match.DimActionSpeed, Against: match.ScopeOnly(enemy),
		Source: match.SourceSystem, ExpiresAt: match.LifetimeEndOfTurn,
	})
	ledger.Add(match.Modifier{
		Amount: -2, Applies: match.DimActionSpeed, Source: match.SourceSystem, ExpiresAt: match.LifetimeEndOfRound,
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

func TestRollCalculator_Derive_DiceDoesNotAliasAttempts(t *testing.T) {
	// RollAttempts is the record of what fell. Derive must hand out a copy of the picked
	// set, not a slice backed by the same array, or mutating one outcome's Dice corrupts
	// the attempts (and every other outcome derived from them).
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()
	a := attempts([]int{8, 2}, nil)
	in := service.RollInput{SkillValue: 4}

	out := calc.Derive(rules, a, in)
	out.Dice[0] = 99

	if a.Primary[0] != 8 {
		t.Errorf("expected attempts.Primary to be unaffected by mutating out.Dice, got %d", a.Primary[0])
	}
}
