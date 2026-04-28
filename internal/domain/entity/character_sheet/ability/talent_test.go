package ability_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
)

func newTestTalent() *ability.Talent {
	table := experience.NewExpTable(2.0)
	exp := experience.NewExperience(table)
	return ability.NewTalent(*exp)
}

func TestTalent_InitialState(t *testing.T) {
	talent := newTestTalent()

	if talent.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", talent.GetLevel())
	}
	if talent.GetExpPoints() != 0 {
		t.Errorf("initial exp: got %d, want 0", talent.GetExpPoints())
	}
}

func TestTalent_InitWithLvl(t *testing.T) {
	tests := []struct {
		name    string
		lvl     int
		wantLvl int
	}{
		{"init with level 1", 1, 1},
		{"init with level 5", 5, 5},
		{"init with level 20", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			talent := newTestTalent()
			talent.InitWithLvl(tt.lvl)

			if talent.GetLevel() != tt.wantLvl {
				t.Errorf("level after InitWithLvl(%d): got %d, want %d",
					tt.lvl, talent.GetLevel(), tt.wantLvl)
			}
		})
	}
}

func TestTalent_IncreaseExp(t *testing.T) {
	talent := newTestTalent()

	diff := talent.IncreaseExp(1)
	if talent.GetExpPoints() != 1 {
		t.Errorf("exp after increase: got %d, want 1", talent.GetExpPoints())
	}
	if diff != 0 {
		t.Errorf("diff for small exp: got %d, want 0", diff)
	}
}

func TestTalent_IncreaseExp_LevelUp(t *testing.T) {
	talent := newTestTalent()
	table := experience.NewExpTable(2.0)

	lvl1Agg := table.GetAggregateExpByLvl(1)
	diff := talent.IncreaseExp(lvl1Agg)

	if diff != 1 {
		t.Errorf("diff after level up exp: got %d, want 1", diff)
	}
	if talent.GetLevel() != 1 {
		t.Errorf("level after level up: got %d, want 1", talent.GetLevel())
	}
}

func TestTalent_GetCurrentExp(t *testing.T) {
	talent := newTestTalent()
	table := experience.NewExpTable(2.0)

	lvl1Agg := table.GetAggregateExpByLvl(1)
	extra := 10
	talent.IncreaseExp(lvl1Agg + extra)

	if talent.GetCurrentExp() != extra {
		t.Errorf("current exp: got %d, want %d", talent.GetCurrentExp(), extra)
	}
}

func TestTalent_GetNextLvlAggregateExp(t *testing.T) {
	talent := newTestTalent()

	got := talent.GetNextLvlAggregateExp()
	if got <= 0 {
		t.Errorf("next lvl aggregate exp should be positive, got %d", got)
	}
}

func TestTalent_GetNextLvlBaseExp(t *testing.T) {
	talent := newTestTalent()

	got := talent.GetNextLvlBaseExp()
	if got <= 0 {
		t.Errorf("next lvl base exp should be positive, got %d", got)
	}
}
