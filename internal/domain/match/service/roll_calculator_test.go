package service_test

import (
	"slices"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// attempts monta um RollAttempts com o conjunto e os extras, para um teste de Derive nomear
// números exatos sem passar por Roll. Roll tem sua própria costura — ver RollSource.
func attempts(primary, extra []int) action.RollAttempts {
	return action.RollAttempts{Primary: primary, Extra: extra}
}

func TestRollCalculator_Roll(t *testing.T) {
	calc := service.RollCalculator{}

	t.Run("2d10 rolls only the primary pair", func(t *testing.T) {
		got := calc.Roll(match.NewDefaultMatchRules(), nil)

		if len(got.Primary) != 2 || len(got.Extra) != 0 {
			t.Fatalf("expected 2 primary dice and no extra, got %v / %v", got.Primary, got.Extra)
		}
		for i, d := range got.Primary {
			if d < 1 || d > 10 {
				t.Errorf("die %d out of range: %d", i, d)
			}
		}
	})

	t.Run("d20 rolls one die", func(t *testing.T) {
		rules := match.NewDefaultMatchRules()
		rules.DiceSet = match.DiceSetD20

		got := calc.Roll(rules, nil)

		if len(got.Primary) != 1 || len(got.Extra) != 0 {
			t.Fatalf("expected 1 primary die and no extra, got %v / %v", got.Primary, got.Extra)
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
		{"double ten is a critical", []int{10, 10}, 50, true, false},
		{"double one is a critical failure", []int{1, 1}, 3, false, true},
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
		if !out.IsCritical || out.Total != 46 {
			t.Errorf("expected critical with total 46, got %v / %d", out.IsCritical, out.Total)
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

// TestRollCalculator_Derive_PassiveWithTheMastersBias pins P2 (owner decision, 2026-10-12): a
// bias — the master's, and since 2026-10-13 the system's too — makes a passive test a rolled one, with the base value (11 on 2D10) as the
// other set — advantage keeps the better of the dice and the base, disadvantage the worse.
func TestRollCalculator_Derive_PassiveWithTheMastersBias(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()
	derive := func(primary []int, cond *action.RollCondition, systemBias int) service.RollOutcome {
		return calc.Derive(rules, attempts(primary, nil), service.RollInput{
			SkillValue: 6, Passive: true, Condition: cond, SystemBias: systemBias,
		})
	}

	tests := []struct {
		name      string
		primary   []int
		bias      int
		wantDice  int // DiceTotal
		wantShown bool
	}{
		{"advantage, the dice beat the base: the dice count", []int{9, 8}, 1, 17, true},
		{"advantage, the base beats the dice: the base stands", []int{2, 3}, 1, 11, true},
		{"disadvantage, the dice fall under the base: the dice count", []int{2, 3}, -1, 5, true},
		{"disadvantage, the base is under the dice: the base stands", []int{9, 8}, -1, 11, true},
		{"no dice yet: the base stands, nothing to show", nil, 1, 11, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := derive(tt.primary, &action.RollCondition{Bias: tt.bias}, 0)
			if out.DiceTotal != tt.wantDice || out.Total != tt.wantDice+6 {
				t.Fatalf("DiceTotal = %d, Total = %d; want %d and %d", out.DiceTotal, out.Total, tt.wantDice, tt.wantDice+6)
			}
			if (len(out.Dice) > 0) != tt.wantShown {
				t.Fatalf("Dice = %v, want shown = %v", out.Dice, tt.wantShown)
			}
			if !out.Passive {
				t.Fatal("still a passive reading — the bias only adds the dice to it")
			}
		})
	}

	t.Run("a critical counts only when the dice are what counts", func(t *testing.T) {
		if out := derive([]int{10, 10}, &action.RollCondition{Bias: 1}, 0); !out.IsCritical {
			t.Fatal("advantage on 10+10: the dice count, and they are a critical")
		}
		if out := derive([]int{10, 10}, &action.RollCondition{Bias: -1}, 0); out.IsCritical {
			t.Fatal("disadvantage on 10+10: the base counts, and a base is never a critical")
		}
	})

	t.Run("the system's bias reads the same way (2026-10-13): its disadvantage keeps the worse", func(t *testing.T) {
		if out := derive([]int{2, 3}, nil, -1); out.DiceTotal != 5 {
			t.Fatalf("got %+v, want the dice 2+3 under the base", out)
		}
	})

	t.Run("the master's advantage cancels the system's disadvantage: the plain base value", func(t *testing.T) {
		if out := derive([]int{2, 3}, &action.RollCondition{Bias: 1}, -1); out.DiceTotal != 11 || len(out.Dice) != 0 {
			t.Fatalf("got %+v, want the base value — net bias zero", out)
		}
	})
}

func TestRollCalculator_Derive_Bias(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()
	both := attempts([]int{3, 4}, []int{8, 9}) // um nível lê [3 4 8]

	t.Run("advantage keeps the best pair of the pool", func(t *testing.T) {
		out := calc.Derive(rules, both, service.RollInput{
			SkillValue: 0,
			Condition:  &action.RollCondition{Bias: 1},
		})
		if out.DiceTotal != 12 {
			t.Errorf("expected 12 (4+8), got %d", out.DiceTotal)
		}
	})

	t.Run("disadvantage keeps the worst pair of the pool", func(t *testing.T) {
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
			Dimension: match.DimActionSpeed,
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
			Dimension: match.DimActionSpeed,
		})
		if out.Bias != -1 {
			t.Errorf("expected net bias -1, got %d", out.Bias)
		}
		if out.DiceTotal != 7 {
			t.Errorf("expected the worse pair, got %d", out.DiceTotal)
		}
	})

	t.Run("reads the primary set alone when no extra fell yet", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{5, 5}, nil), service.RollInput{
			Condition: &action.RollCondition{Bias: 1},
		})
		if out.DiceTotal != 10 || out.Short != 1 {
			t.Errorf("expected 10 with one die short, got %d / short %d", out.DiceTotal, out.Short)
		}
	})
}

func TestRollCalculator_Derive_DicePool(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()
	d20 := match.NewDefaultMatchRules()
	d20.DiceSet = match.DiceSetD20
	bias := func(b int) *action.RollCondition { return &action.RollCondition{Bias: b} }

	t.Run("advantage two keeps best pair of four", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{3, 4}, []int{9, 10}), service.RollInput{Condition: bias(2)})
		if out.DiceTotal != 19 || !slices.Equal(out.Dice, []int{9, 10}) {
			t.Fatalf("DiceTotal = %d, Dice = %v; want 19 and [9 10]", out.DiceTotal, out.Dice)
		}
		if !slices.Equal(out.Rolled, []int{3, 4, 9, 10}) {
			t.Errorf("Rolled = %v, want the whole pool [3 4 9 10]", out.Rolled)
		}
		if out.Short != 0 {
			t.Errorf("Short = %d, want 0", out.Short)
		}
	})

	t.Run("disadvantage keeps worst pair", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{5, 6}, []int{2, 9}), service.RollInput{Condition: bias(-2)})
		if out.DiceTotal != 7 || !slices.Equal(out.Dice, []int{5, 2}) {
			t.Fatalf("DiceTotal = %d, Dice = %v; want 7 and [5 2] (pool order)", out.DiceTotal, out.Dice)
		}
	})

	t.Run("levels cancel one by one", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{3, 4}, []int{9, 10}), service.RollInput{
			Condition: bias(2), SystemBias: -1,
		})
		if out.Bias != 1 {
			t.Fatalf("Bias = %d, want 1", out.Bias)
		}
		if out.DiceTotal != 13 || !slices.Equal(out.Rolled, []int{3, 4, 9}) {
			t.Fatalf("DiceTotal = %d, Rolled = %v; want 13 from [3 4 9]", out.DiceTotal, out.Rolled)
		}
	})

	t.Run("d20 keeps one of three", func(t *testing.T) {
		a := attempts([]int{7}, []int{15, 3})
		if out := calc.Derive(d20, a, service.RollInput{Condition: bias(2)}); out.DiceTotal != 15 {
			t.Errorf("advantage: DiceTotal = %d, want 15", out.DiceTotal)
		}
		if out := calc.Derive(d20, a, service.RollInput{Condition: bias(-2)}); out.DiceTotal != 3 {
			t.Errorf("disadvantage: DiceTotal = %d, want 3", out.DiceTotal)
		}
	})

	t.Run("short reports missing extras", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{3, 4}, []int{9}), service.RollInput{Condition: bias(3)})
		if out.Short != 2 {
			t.Fatalf("Short = %d, want 2", out.Short)
		}
		if out.DiceTotal != 13 {
			t.Errorf("DiceTotal = %d, want 13 — derives with the dice that exist", out.DiceTotal)
		}
	})

	t.Run("short counts the base set on an empty passive", func(t *testing.T) {
		out := calc.Derive(rules, action.RollAttempts{}, service.RollInput{Passive: true, Condition: bias(2)})
		if out.Short != 4 {
			t.Fatalf("Short = %d, want 4 (2 of the set + 2 levels)", out.Short)
		}
		if out.DiceTotal != 11 {
			t.Errorf("DiceTotal = %d, want the base value 11 while no die fell", out.DiceTotal)
		}
		if neutral := calc.Derive(rules, action.RollAttempts{}, service.RollInput{Passive: true}); neutral.Short != 0 {
			t.Errorf("a neutral passive needs no dice, got Short %d", neutral.Short)
		}
	})

	t.Run("passive with two levels compares kept pair with base", func(t *testing.T) {
		passive := func(primary, extra []int, b int) service.RollOutcome {
			return calc.Derive(rules, attempts(primary, extra), service.RollInput{
				SkillValue: 6, Passive: true, Condition: bias(b),
			})
		}
		if out := passive([]int{2, 3}, []int{9, 8}, 2); out.DiceTotal != 17 || out.Total != 23 {
			t.Errorf("advantage: got %d / %d, want the kept pair 17 over the base", out.DiceTotal, out.Total)
		}
		if out := passive([]int{2, 3}, []int{4, 1}, 2); out.DiceTotal != 11 {
			t.Errorf("advantage under the base: DiceTotal = %d, want the base 11", out.DiceTotal)
		}
		if out := passive([]int{9, 8}, []int{2, 3}, -2); out.DiceTotal != 5 {
			t.Errorf("disadvantage: DiceTotal = %d, want the kept pair 5 under the base", out.DiceTotal)
		}
		if out := passive([]int{9, 8}, []int{7, 6}, -2); out.DiceTotal != 11 {
			t.Errorf("disadvantage over the base: DiceTotal = %d, want the base 11", out.DiceTotal)
		}
		tie := passive([]int{5, 6}, []int{1, 2}, 2)
		if tie.DiceTotal != 11 || tie.IsCritical || tie.IsCriticalFailure {
			t.Errorf("tie: got %+v, want the base 11", tie)
		}
		if !slices.Equal(tie.Dice, []int{5, 6}) {
			t.Errorf("tie: Dice = %v, want the kept pair still shown", tie.Dice)
		}
		if !tie.Passive {
			t.Error("still a passive reading — the bias only adds the dice to it")
		}
	})

	t.Run("bias capped at ten levels", func(t *testing.T) {
		extra := []int{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 9, 9}
		out := calc.Derive(rules, attempts([]int{1, 1}, extra), service.RollInput{Condition: bias(15)})
		if out.Bias != match.MaxBiasLevels {
			t.Fatalf("Bias = %d, want %d", out.Bias, match.MaxBiasLevels)
		}
		if len(out.Rolled) != 12 || out.DiceTotal != 2 {
			t.Errorf("Rolled = %v, DiceTotal = %d; want 12 dice read and the 9s past the cap ignored", out.Rolled, out.DiceTotal)
		}
		if short := calc.Derive(rules, attempts([]int{1, 1}, nil), service.RollInput{Condition: bias(-15)}); short.Short != 10 {
			t.Errorf("Short = %d, want 10", short.Short)
		}
	})

	t.Run("status bias and penalty add to the master's numbers", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{3, 4}, []int{9, 10}), service.RollInput{
			SkillValue: 5,
			Condition:  &action.RollCondition{Bias: 1, Modifier: 1},
			Status:     service.StatusRoll{Bias: 1, Penalty: -2},
		})
		if out.Bias != 2 || out.Modifier != -1 {
			t.Fatalf("Bias = %d, Modifier = %d; want 2 and -1", out.Bias, out.Modifier)
		}
		if out.Total != 23 { // 19 + 5 − 1
			t.Errorf("Total = %d, want 23", out.Total)
		}
	})
}

func TestRollCalculator_Derive_CriticalRule(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()

	t.Run("critical doubles total with modifiers", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{10, 10}, nil), service.RollInput{
			SkillValue: 8, Condition: &action.RollCondition{Modifier: -3},
		})
		if !out.IsCritical || out.Total != 50 {
			t.Fatalf("got critical %v total %d, want true and 50", out.IsCritical, out.Total)
		}
	})

	t.Run("critical failure halves", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{1, 1}, nil), service.RollInput{SkillValue: 8})
		if !out.IsCriticalFailure || out.Total != 5 {
			t.Fatalf("got failure %v total %d, want true and 5", out.IsCriticalFailure, out.Total)
		}
	})

	t.Run("critical read on kept dice only", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{1, 1}, []int{6}), service.RollInput{
			SkillValue: 8, Condition: &action.RollCondition{Bias: 1},
		})
		if out.IsCriticalFailure || out.Total != 15 {
			t.Fatalf("advantage dropped a 1: got failure %v total %d, want false and 15", out.IsCriticalFailure, out.Total)
		}
		out = calc.Derive(rules, attempts([]int{10, 10}, []int{3}), service.RollInput{
			SkillValue: 8, Condition: &action.RollCondition{Bias: -1},
		})
		if out.IsCritical || out.Total != 21 {
			t.Fatalf("disadvantage dropped a 10: got critical %v total %d, want false and 21", out.IsCritical, out.Total)
		}
	})

	t.Run("passive base value never critical", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{10, 10}, []int{10}), service.RollInput{
			SkillValue: 4, Passive: true, Condition: &action.RollCondition{Bias: -1},
		})
		if out.IsCritical || out.Total != 15 {
			t.Fatalf("got critical %v total %d, want the base 11 + 4 = 15 with no critical", out.IsCritical, out.Total)
		}
		out = calc.Derive(rules, attempts([]int{10, 10}, nil), service.RollInput{SkillValue: 4, Passive: true})
		if out.IsCritical || out.Total != 15 {
			t.Fatalf("neutral passive: got critical %v total %d, want 15", out.IsCritical, out.Total)
		}
	})
}

func TestRollCalculator_Derive_StatusRoll(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()

	t.Run("auto fail skips dice", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{10, 10}, []int{10}), service.RollInput{
			SkillValue: 8,
			Condition:  &action.RollCondition{Bias: 3, Modifier: 4},
			Status:     service.StatusRoll{AutoFail: true},
		})
		if !out.AutoFailed || out.Total != 0 {
			t.Fatalf("got AutoFailed %v total %d, want true and 0", out.AutoFailed, out.Total)
		}
		if len(out.Dice) != 0 || len(out.Rolled) != 0 || out.IsCritical || out.IsCriticalFailure || out.Short != 0 {
			t.Errorf("an auto-fail reads no dice and asks for none, got %+v", out)
		}
	})

	t.Run("rolled only rolls a passive", func(t *testing.T) {
		out := calc.Derive(rules, attempts([]int{9, 9}, nil), service.RollInput{
			SkillValue: 3, Passive: true, Status: service.StatusRoll{RolledOnly: true},
		})
		if out.Passive || out.DiceTotal != 18 || out.Total != 21 {
			t.Fatalf("got passive %v dice %d total %d, want a rolled 18 + 3", out.Passive, out.DiceTotal, out.Total)
		}
		empty := calc.Derive(rules, action.RollAttempts{}, service.RollInput{
			Passive: true, Status: service.StatusRoll{RolledOnly: true},
		})
		if empty.Short != 2 {
			t.Errorf("Short = %d, want the whole set (2) on a passive that must roll", empty.Short)
		}
	})
}

func TestRollCalculator_TopUp(t *testing.T) {
	calc := service.RollCalculator{}
	rules := match.NewDefaultMatchRules()

	t.Run("fills the primary set first, then the extras", func(t *testing.T) {
		var a action.RollAttempts
		calc.TopUp(rules, &a, 4, &scriptedSource{faces: []int{3, 4, 9, 10}})
		if !slices.Equal(a.Primary, []int{3, 4}) || !slices.Equal(a.Extra, []int{9, 10}) {
			t.Fatalf("got %+v, want Primary [3 4] and Extra [9 10]", a)
		}
	})

	t.Run("appends only the missing extras", func(t *testing.T) {
		a := attempts([]int{3, 4}, []int{9})
		calc.TopUp(rules, &a, 2, &scriptedSource{faces: []int{5, 6}})
		if !slices.Equal(a.Primary, []int{3, 4}) || !slices.Equal(a.Extra, []int{9, 5, 6}) {
			t.Fatalf("got %+v, want the fallen dice kept and [5 6] appended", a)
		}
	})

	t.Run("nothing needed rolls nothing", func(t *testing.T) {
		src := &scriptedSource{faces: []int{5}}
		a := attempts([]int{3, 4}, nil)
		calc.TopUp(rules, &a, 0, src)
		if len(a.Extra) != 0 || src.i != 0 {
			t.Fatalf("got %+v after %d rolls, want nothing drawn", a, src.i)
		}
	})

	t.Run("d20 set is one die", func(t *testing.T) {
		d20 := match.NewDefaultMatchRules()
		d20.DiceSet = match.DiceSetD20
		var a action.RollAttempts
		calc.TopUp(d20, &a, 3, &scriptedSource{faces: []int{7, 15, 3}})
		if !slices.Equal(a.Primary, []int{7}) || !slices.Equal(a.Extra, []int{15, 3}) {
			t.Fatalf("got %+v, want Primary [7] and Extra [15 3]", a)
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
			Dimension:  match.DimActionSpeed,
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
			Dimension:  match.DimActionSpeed,
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

func TestDerive_ThreeBiasOrigins(t *testing.T) {
	rules := match.NewDefaultMatchRules()
	// Um nível lê [4 2 9]: a vantagem mantém 4+9 = 13, a desvantagem 4+2 = 6.
	attempts := action.RollAttempts{Primary: []int{4, 2}, Extra: []int{9, 5}}

	t.Run("the system's situational bias reads the worst pair on its own", func(t *testing.T) {
		out := service.RollCalculator{}.Derive(rules, attempts, service.RollInput{
			SkillName: "Legerity", SkillValue: 3, SystemBias: -1, Dimension: match.DimActionSpeed,
		})
		if out.DiceTotal != 6 {
			t.Errorf("DiceTotal = %d, want 6 — disadvantage keeps the worst pair", out.DiceTotal)
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
		if out.DiceTotal != 13 {
			t.Errorf("DiceTotal = %d, want 13 — a net advantage keeps the best pair of [4 2 9]", out.DiceTotal)
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

	// The two subtests below pin that in.Dimension is the field actually read by Derive, not
	// just a value every other subtest happens to set to match.DimActionSpeed. A regression to
	// a hardcoded match.DimActionSpeed literal (the Task 1 placeholder this task removed) would
	// pass every subtest above unchanged, because they all target DimActionSpeed already.

	t.Run("pins that a non-actionSpeed dimension is actually plumbed through", func(t *testing.T) {
		// Would read 0 under a hardcoded match.DimActionSpeed Derive, since the entry is DimDodge.
		ledger := match.NewModifierLedger()
		ledger.Add(match.Modifier{Amount: 5, Applies: match.DimDodge, Against: match.ScopeAnyone()})
		out := service.RollCalculator{}.Derive(rules, attempts, service.RollInput{
			SkillName: "Legerity", SkillValue: 3, Ledger: &ledger, Dimension: match.DimDodge,
		})
		if out.Modifier != 5 {
			t.Errorf("Modifier = %d, want 5 — the caller asked for DimDodge and the ledger holds it", out.Modifier)
		}
	})

	t.Run("pins that the zero-value Dimension reads nothing", func(t *testing.T) {
		// Would read 5 under a hardcoded match.DimActionSpeed Derive, since the entry is
		// DimActionSpeed and Dimension is left at its zero value here.
		ledger := match.NewModifierLedger()
		ledger.Add(match.Modifier{Amount: 5, Applies: match.DimActionSpeed, Against: match.ScopeAnyone()})
		out := service.RollCalculator{}.Derive(rules, attempts, service.RollInput{
			SkillName: "Legerity", SkillValue: 3, Ledger: &ledger,
		})
		if out.Modifier != 0 {
			t.Errorf("Modifier = %d, want 0 — no Dimension was named, so no reserve applies", out.Modifier)
		}
	})
}
