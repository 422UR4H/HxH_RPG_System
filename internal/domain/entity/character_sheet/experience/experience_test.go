package experience_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
)

func newTestExp() *experience.Exp {
	table := experience.NewDefaultExpTable()
	exp := experience.NewExperience(table)
	return exp
}

func TestExp_InitialState(t *testing.T) {
	exp := newTestExp()

	if exp.GetPoints() != 0 {
		t.Errorf("initial points: got %d, want 0", exp.GetPoints())
	}
	if exp.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", exp.GetLevel())
	}
	if exp.GetCurrentExp() != 0 {
		t.Errorf("initial current exp: got %d, want 0", exp.GetCurrentExp())
	}
}

func TestExp_IncreasePoints_NoLevelUp(t *testing.T) {
	exp := newTestExp()

	smallExp := 1
	diff := exp.IncreasePoints(smallExp)

	if diff != 0 {
		t.Errorf("expected no level change, got diff=%d", diff)
	}
	if exp.GetPoints() != smallExp {
		t.Errorf("points after increase: got %d, want %d", exp.GetPoints(), smallExp)
	}
	if exp.GetLevel() != 0 {
		t.Errorf("level should still be 0, got %d", exp.GetLevel())
	}
}

func TestExp_IncreasePoints_WithLevelUp(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	lvl1Exp := table.GetAggregateExpByLvl(1)
	diff := exp.IncreasePoints(lvl1Exp)

	if diff != 1 {
		t.Errorf("level diff: got %d, want 1", diff)
	}
	if exp.GetLevel() != 1 {
		t.Errorf("level after increase: got %d, want 1", exp.GetLevel())
	}
	if exp.GetPoints() != lvl1Exp {
		t.Errorf("points: got %d, want %d", exp.GetPoints(), lvl1Exp)
	}
}

func TestExp_IncreasePoints_MultiLevel(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	lvl5Exp := table.GetAggregateExpByLvl(5)
	diff := exp.IncreasePoints(lvl5Exp)

	if diff != 5 {
		t.Errorf("level diff: got %d, want 5", diff)
	}
	if exp.GetLevel() != 5 {
		t.Errorf("level: got %d, want 5", exp.GetLevel())
	}
}

func TestExp_GetCurrentExp(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	lvl1Exp := table.GetAggregateExpByLvl(1)
	// use half of next level's base to ensure we stay at level 1
	extra := table.GetBaseExpByLvl(2) / 2
	exp.IncreasePoints(lvl1Exp + extra)

	current := exp.GetCurrentExp()
	if current != extra {
		t.Errorf("current exp: got %d, want %d", current, extra)
	}
}

func TestExp_GetExpToEvolve(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	evolve := exp.GetExpToEvolve()
	expectedEvolve := table.GetAggregateExpByLvl(1)

	if evolve != expectedEvolve {
		t.Errorf("exp to evolve at lvl 0: got %d, want %d", evolve, expectedEvolve)
	}
}

func TestExp_GetNextLvlBaseExp(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	got := exp.GetNextLvlBaseExp()
	want := table.GetBaseExpByLvl(1)
	if got != want {
		t.Errorf("next lvl base exp: got %d, want %d", got, want)
	}
}

func TestExp_GetNextLvlAggregateExp(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	got := exp.GetNextLvlAggregateExp()
	want := table.GetAggregateExpByLvl(1)
	if got != want {
		t.Errorf("next lvl aggregate exp: got %d, want %d", got, want)
	}
}

func TestExp_Clone(t *testing.T) {
	exp := newTestExp()
	exp.IncreasePoints(1000)

	clone := exp.Clone()

	if clone.GetPoints() != 0 {
		t.Errorf("clone should start fresh, got points=%d", clone.GetPoints())
	}
	if clone.GetLevel() != 0 {
		t.Errorf("clone should start at lvl 0, got %d", clone.GetLevel())
	}
}

func TestExp_IncrementalIncreases(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	lvl1Exp := table.GetAggregateExpByLvl(1)
	half := lvl1Exp / 2

	diff1 := exp.IncreasePoints(half)
	if diff1 != 0 {
		t.Errorf("first half should not level up, diff=%d", diff1)
	}

	diff2 := exp.IncreasePoints(lvl1Exp - half)
	if diff2 != 1 {
		t.Errorf("second half should level up, diff=%d", diff2)
	}
	if exp.GetLevel() != 1 {
		t.Errorf("level after two increases: got %d, want 1", exp.GetLevel())
	}
}
