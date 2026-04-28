package ability_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestAbility() (*ability.Ability, *experience.CharacterExp) {
	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilityTable := experience.NewExpTable(20.0)
	abilityExp := experience.NewExperience(abilityTable)

	a := ability.NewAbility(enum.Physicals, *abilityExp, charExp)
	return a, charExp
}

func TestAbility_InitialState(t *testing.T) {
	a, _ := newTestAbility()

	if a.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", a.GetLevel())
	}
	if a.GetExpPoints() != 0 {
		t.Errorf("initial exp: got %d, want 0", a.GetExpPoints())
	}
	if a.GetName() != enum.Physicals {
		t.Errorf("name: got %v, want Physicals", a.GetName())
	}
}

func TestAbility_GetBonus(t *testing.T) {
	tests := []struct {
		name      string
		charPts   int
		wantBonus float64
	}{
		{"zero points and level", 0, 0.0},
		{"with character points", 10, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, charExp := newTestAbility()

			charExp.IncreaseCharacterPoints(tt.charPts)
			got := a.GetBonus()

			if got != tt.wantBonus {
				t.Errorf("GetBonus() = %f, want %f (charPts=%d, abilityLvl=%d)",
					got, tt.wantBonus, charExp.GetCharacterPoints(), a.GetLevel())
			}
		})
	}
}

func TestAbility_CascadeUpgrade(t *testing.T) {
	a, charExp := newTestAbility()

	cascade := experience.NewUpgradeCascade(100)
	a.CascadeUpgrade(cascade)

	if a.GetExpPoints() != 100 {
		t.Errorf("ability exp after cascade: got %d, want 100", a.GetExpPoints())
	}
	if charExp.GetExpPoints() != 100 {
		t.Errorf("char exp after cascade: got %d, want 100", charExp.GetExpPoints())
	}
	if cascade.CharacterExp != charExp {
		t.Error("cascade.CharacterExp should be set after CascadeUpgrade")
	}

	abCascade, ok := cascade.Abilities[enum.Physicals]
	if !ok {
		t.Fatal("cascade.Abilities should contain Physicals entry")
	}
	if abCascade.Exp != a.GetExpPoints() {
		t.Errorf("cascade ability exp: got %d, want %d", abCascade.Exp, a.GetExpPoints())
	}
}

func TestAbility_CascadeUpgrade_LevelUp_IncreasesCharacterPoints(t *testing.T) {
	a, charExp := newTestAbility()

	abilityTable := experience.NewExpTable(20.0)
	lvl1Exp := abilityTable.GetAggregateExpByLvl(1)

	cascade := experience.NewUpgradeCascade(lvl1Exp)
	a.CascadeUpgrade(cascade)

	if a.GetLevel() < 1 {
		t.Errorf("ability should level up, got level %d", a.GetLevel())
	}
	if charExp.GetCharacterPoints() < 1 {
		t.Errorf("character points should increase on ability level up, got %d",
			charExp.GetCharacterPoints())
	}
}

func TestAbility_GetExpReference(t *testing.T) {
	a, _ := newTestAbility()

	ref := a.GetExpReference()
	if ref == nil {
		t.Fatal("GetExpReference returned nil")
	}
}

func TestAbility_DelegatesExpMethods(t *testing.T) {
	a, _ := newTestAbility()

	if a.GetNextLvlBaseExp() <= 0 {
		t.Error("GetNextLvlBaseExp should be positive")
	}
	if a.GetNextLvlAggregateExp() <= 0 {
		t.Error("GetNextLvlAggregateExp should be positive")
	}
	if a.GetCurrentExp() != 0 {
		t.Errorf("GetCurrentExp should be 0 initially, got %d", a.GetCurrentExp())
	}
}
