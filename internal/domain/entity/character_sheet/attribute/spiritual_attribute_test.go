package attribute_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestSpiritualAttribute(name enum.AttributeName, buff *int) *attribute.SpiritualAttribute {
	table := experience.NewExpTable(1.0)
	exp := experience.NewExperience(table)

	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilityTable := experience.NewExpTable(5.0)
	abilityExp := experience.NewExperience(abilityTable)
	ab := ability.NewAbility(enum.Spirituals, *abilityExp, charExp)

	return attribute.NewSpiritualAttribute(name, *exp, ab, buff)
}

func TestSpiritualAttribute_InitialState(t *testing.T) {
	buff := 0
	sa := newTestSpiritualAttribute(enum.Flame, &buff)

	if sa.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", sa.GetLevel())
	}
	if sa.GetName() != enum.Flame {
		t.Errorf("name: got %v, want Flame", sa.GetName())
	}
	if sa.GetExpPoints() != 0 {
		t.Errorf("initial exp: got %d, want 0", sa.GetExpPoints())
	}
}

func TestSpiritualAttribute_GetPower(t *testing.T) {
	buff := 3
	sa := newTestSpiritualAttribute(enum.Flame, &buff)

	// Power = level(0) + int(AbilityBonus(0)) + buff(3) = 3
	got := sa.GetPower()
	if got != 3 {
		t.Errorf("GetPower with buff=3: got %d, want 3", got)
	}
}

func TestSpiritualAttribute_CascadeUpgrade(t *testing.T) {
	buff := 0
	sa := newTestSpiritualAttribute(enum.Conscience, &buff)

	cascade := experience.NewUpgradeCascade(100)
	sa.CascadeUpgrade(cascade)

	if sa.GetExpPoints() != 100 {
		t.Errorf("exp after cascade: got %d, want 100", sa.GetExpPoints())
	}

	entry, ok := cascade.Attributes[enum.Conscience]
	if !ok {
		t.Fatal("cascade.Attributes should contain Conscience entry")
	}
	if entry.Exp != sa.GetExpPoints() {
		t.Errorf("cascade exp: got %d, want %d", entry.Exp, sa.GetExpPoints())
	}
	if entry.Lvl != sa.GetLevel() {
		t.Errorf("cascade lvl: got %d, want %d", entry.Lvl, sa.GetLevel())
	}
	if entry.Power != sa.GetPower() {
		t.Errorf("cascade power: got %d, want %d", entry.Power, sa.GetPower())
	}
}

func TestSpiritualAttribute_Clone(t *testing.T) {
	buff := 0
	sa := newTestSpiritualAttribute(enum.Flame, &buff)

	cascade := experience.NewUpgradeCascade(100)
	sa.CascadeUpgrade(cascade)

	newBuff := 0
	clone := sa.Clone(enum.Conscience, &newBuff)

	if clone.GetName() != enum.Conscience {
		t.Errorf("clone name: got %v, want Conscience", clone.GetName())
	}
	if clone.GetExpPoints() != 0 {
		t.Errorf("clone should start fresh, got exp=%d", clone.GetExpPoints())
	}
}

func TestSpiritualAttribute_BuffAffectsPower(t *testing.T) {
	buff := 0
	sa := newTestSpiritualAttribute(enum.Flame, &buff)

	powerBefore := sa.GetPower()
	buff = 5
	powerAfter := sa.GetPower()

	if powerAfter != powerBefore+5 {
		t.Errorf("buff change: power before=%d, after=%d, want diff=5", powerBefore, powerAfter)
	}
}

func TestSpiritualAttribute_DelegatesExpMethods(t *testing.T) {
	buff := 0
	sa := newTestSpiritualAttribute(enum.Flame, &buff)

	if sa.GetNextLvlBaseExp() <= 0 {
		t.Error("GetNextLvlBaseExp should be positive")
	}
	if sa.GetNextLvlAggregateExp() <= 0 {
		t.Error("GetNextLvlAggregateExp should be positive")
	}
	if sa.GetCurrentExp() != 0 {
		t.Errorf("GetCurrentExp should be 0, got %d", sa.GetCurrentExp())
	}
}
