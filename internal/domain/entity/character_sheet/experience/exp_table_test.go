package experience_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
)

func TestNewDefaultExpTable(t *testing.T) {
	table := experience.NewDefaultExpTable()
	if table == nil {
		t.Fatal("NewDefaultExpTable returned nil")
	}
}

func TestNewExpTable_CustomCoefficient(t *testing.T) {
	table := experience.NewExpTable(2.0)
	if table == nil {
		t.Fatal("NewExpTable(2.0) returned nil")
	}
}

func TestExpTable_LevelZeroIsZero(t *testing.T) {
	table := experience.NewDefaultExpTable()

	if base := table.GetBaseExpByLvl(0); base != 0 {
		t.Errorf("base exp at level 0: got %d, want 0", base)
	}
	if agg := table.GetAggregateExpByLvl(0); agg != 0 {
		t.Errorf("aggregate exp at level 0: got %d, want 0", agg)
	}
}

func TestExpTable_BaseExpMonotonicallyIncreasing(t *testing.T) {
	table := experience.NewDefaultExpTable()

	for lvl := 2; lvl < int(experience.MAX_LVL); lvl++ {
		curr := table.GetBaseExpByLvl(lvl)
		prev := table.GetBaseExpByLvl(lvl - 1)

		if curr < prev {
			t.Errorf("base exp decreased at lvl %d: %d < %d", lvl, curr, prev)
		}
	}
}

func TestExpTable_AggregateIsCumulativeSum(t *testing.T) {
	table := experience.NewDefaultExpTable()

	for lvl := 1; lvl < int(experience.MAX_LVL); lvl++ {
		expected := table.GetAggregateExpByLvl(lvl-1) + table.GetBaseExpByLvl(lvl)
		got := table.GetAggregateExpByLvl(lvl)

		if got != expected {
			t.Errorf("aggregate at lvl %d: got %d, want %d (prev_agg=%d + base=%d)",
				lvl, got, expected,
				table.GetAggregateExpByLvl(lvl-1), table.GetBaseExpByLvl(lvl))
		}
	}
}

func TestExpTable_AggregateMonotonicallyIncreasing(t *testing.T) {
	table := experience.NewDefaultExpTable()

	for lvl := 1; lvl < int(experience.MAX_LVL); lvl++ {
		curr := table.GetAggregateExpByLvl(lvl)
		prev := table.GetAggregateExpByLvl(lvl - 1)

		if curr <= prev {
			t.Errorf("aggregate not strictly increasing at lvl %d: %d <= %d", lvl, curr, prev)
		}
	}
}

func TestExpTable_GetLvlByExp(t *testing.T) {
	table := experience.NewDefaultExpTable()

	tests := []struct {
		name string
		exp  int
		want int
	}{
		{"zero exp is level 0", 0, 0},
		{"negative exp is level 0", -1, 0},
		{"exactly at level 1 aggregate", table.GetAggregateExpByLvl(1), 1},
		{"one below level 1 aggregate", table.GetAggregateExpByLvl(1) - 1, 0},
		{"exactly at level 10 aggregate", table.GetAggregateExpByLvl(10), 10},
		{"between level 10 and 11", table.GetAggregateExpByLvl(10) + 1, 10},
		{"exactly at max level aggregate", table.GetAggregateExpByLvl(int(experience.MAX_LVL) - 1), int(experience.MAX_LVL) - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := table.GetLvlByExp(tt.exp)
			if got != tt.want {
				t.Errorf("GetLvlByExp(%d) = %d, want %d", tt.exp, got, tt.want)
			}
		})
	}
}

func TestExpTable_CoefficientScalesValues(t *testing.T) {
	table1 := experience.NewExpTable(1.0)
	table2 := experience.NewExpTable(2.0)

	for lvl := 1; lvl < int(experience.MAX_LVL); lvl++ {
		base1 := table1.GetBaseExpByLvl(lvl)
		base2 := table2.GetBaseExpByLvl(lvl)

		// allow ±1 tolerance due to float-to-int truncation:
		// int(2.0*f(lvl)) may differ from 2*int(1.0*f(lvl)) by 1
		expected := 2 * base1
		diff := base2 - expected
		if diff < -1 || diff > 1 {
			t.Errorf("coefficient scaling at lvl %d: 2x table got %d, want %d±1 (1x=%d)",
				lvl, base2, expected, base1)
		}
	}
}

func TestExpTable_GetLvlByExp_RoundTrip(t *testing.T) {
	table := experience.NewDefaultExpTable()

	for lvl := 0; lvl < int(experience.MAX_LVL); lvl++ {
		exp := table.GetAggregateExpByLvl(lvl)
		got := table.GetLvlByExp(exp)

		if got != lvl {
			t.Errorf("round trip failed at lvl %d: GetLvlByExp(GetAggregateExpByLvl(%d)) = %d",
				lvl, lvl, got)
		}
	}
}
