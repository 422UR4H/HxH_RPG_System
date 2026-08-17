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
