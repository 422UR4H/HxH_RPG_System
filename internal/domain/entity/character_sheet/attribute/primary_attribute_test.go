package attribute_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestPrimaryAttribute(name enum.AttributeName, buff *int) *attribute.PrimaryAttribute {
	table := experience.NewExpTable(5.0)
	exp := experience.NewExperience(table)

	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilityTable := experience.NewExpTable(20.0)
	abilityExp := experience.NewExperience(abilityTable)
	ab := ability.NewAbility(enum.Physicals, *abilityExp, charExp)

	return attribute.NewPrimaryAttribute(name, *exp, ab, buff)
}

func TestPrimaryAttribute_InitialState(t *testing.T) {
	buff := 0
	pa := newTestPrimaryAttribute(enum.Resistance, &buff)

	if pa.GetPoints() != 0 {
		t.Errorf("initial points: got %d, want 0", pa.GetPoints())
	}
	if pa.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", pa.GetLevel())
	}
	if pa.GetValue() != 0 {
		t.Errorf("initial value: got %d, want 0", pa.GetValue())
	}
	if pa.GetName() != enum.Resistance {
		t.Errorf("name: got %v, want Resistance", pa.GetName())
	}
}

func TestPrimaryAttribute_IncreasePoints(t *testing.T) {
	buff := 0
	pa := newTestPrimaryAttribute(enum.Resistance, &buff)

	result := pa.IncreasePoints(5)
	if result != 5 {
		t.Errorf("IncreasePoints return: got %d, want 5", result)
	}
	if pa.GetPoints() != 5 {
		t.Errorf("points after increase: got %d, want 5", pa.GetPoints())
	}
	// Value = points + level; level is still 0
	if pa.GetValue() != 5 {
		t.Errorf("value after increase: got %d, want 5", pa.GetValue())
	}
}

func TestPrimaryAttribute_GetPower(t *testing.T) {
	buff := 2
	pa := newTestPrimaryAttribute(enum.Resistance, &buff)

	pa.IncreasePoints(3)
	// Power = GetValue() + int(AbilityBonus) + buff
	// GetValue() = points(3) + level(0) = 3
	// AbilityBonus = (charPts(0) + abilityLvl(0)) / 2 = 0
	// Power = 3 + 0 + 2 = 5
	got := pa.GetPower()
	if got != 5 {
		t.Errorf("GetPower: got %d, want 5", got)
	}
}

func TestPrimaryAttribute_CascadeUpgrade(t *testing.T) {
	buff := 0
	pa := newTestPrimaryAttribute(enum.Resistance, &buff)

	cascade := experience.NewUpgradeCascade(100)
	pa.CascadeUpgrade(cascade)

	if pa.GetExpPoints() != 100 {
		t.Errorf("exp after cascade: got %d, want 100", pa.GetExpPoints())
	}
	entry, ok := cascade.Attributes[enum.Resistance]
	if !ok {
		t.Fatal("cascade.Attributes should contain Resistance entry")
	}
	if entry.Exp != pa.GetExpPoints() {
		t.Errorf("cascade exp: got %d, want %d", entry.Exp, pa.GetExpPoints())
	}
	if entry.Lvl != pa.GetLevel() {
		t.Errorf("cascade lvl: got %d, want %d", entry.Lvl, pa.GetLevel())
	}
	if entry.Power != pa.GetPower() {
		t.Errorf("cascade power: got %d, want %d", entry.Power, pa.GetPower())
	}
}

func TestPrimaryAttribute_Clone(t *testing.T) {
	buff := 0
	pa := newTestPrimaryAttribute(enum.Resistance, &buff)
	pa.IncreasePoints(5)

	newBuff := 0
	clone := pa.Clone(enum.Agility, &newBuff)

	if clone.GetName() != enum.Agility {
		t.Errorf("clone name: got %v, want Agility", clone.GetName())
	}
	if clone.GetPoints() != 0 {
		t.Errorf("clone should start with 0 points, got %d", clone.GetPoints())
	}
	if clone.GetExpPoints() != 0 {
		t.Errorf("clone should start with 0 exp, got %d", clone.GetExpPoints())
	}
}

func TestPrimaryAttribute_DelegatesExpMethods(t *testing.T) {
	buff := 0
	pa := newTestPrimaryAttribute(enum.Resistance, &buff)

	if pa.GetNextLvlBaseExp() <= 0 {
		t.Error("GetNextLvlBaseExp should be positive")
	}
	if pa.GetNextLvlAggregateExp() <= 0 {
		t.Error("GetNextLvlAggregateExp should be positive")
	}
	if pa.GetCurrentExp() != 0 {
		t.Errorf("GetCurrentExp should be 0, got %d", pa.GetCurrentExp())
	}
}

func TestPrimaryAttribute_BuffAffectsPower(t *testing.T) {
	buff := 0
	pa := newTestPrimaryAttribute(enum.Resistance, &buff)
	pa.IncreasePoints(3)

	powerBefore := pa.GetPower()
	buff = 5
	powerAfter := pa.GetPower()

	if powerAfter != powerBefore+5 {
		t.Errorf("buff change: power before=%d, after=%d, want diff=5", powerBefore, powerAfter)
	}
}
