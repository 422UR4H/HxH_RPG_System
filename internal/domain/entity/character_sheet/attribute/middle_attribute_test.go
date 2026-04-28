package attribute_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestMiddleAttribute(
	name enum.AttributeName,
	buff *int,
	primaries ...*attribute.PrimaryAttribute,
) *attribute.MiddleAttribute {
	table := experience.NewExpTable(5.0)
	exp := experience.NewExperience(table)
	return attribute.NewMiddleAttribute(name, *exp, buff, primaries...)
}

func TestMiddleAttribute_InitialState(t *testing.T) {
	buff := 0
	p1 := newTestPrimaryAttribute(enum.Resistance, &buff)
	p2 := newTestPrimaryAttribute(enum.Strength, &buff)
	mid := newTestMiddleAttribute(enum.Constitution, &buff, p1, p2)

	if mid.GetPoints() != 0 {
		t.Errorf("initial points: got %d, want 0", mid.GetPoints())
	}
	if mid.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", mid.GetLevel())
	}
	if mid.GetValue() != 0 {
		t.Errorf("initial value: got %d, want 0", mid.GetValue())
	}
	if mid.GetName() != enum.Constitution {
		t.Errorf("name: got %v, want Constitution", mid.GetName())
	}
}

func TestMiddleAttribute_GetPoints_RoundsUp(t *testing.T) {
	buff := 0
	p1 := newTestPrimaryAttribute(enum.Resistance, &buff)
	p2 := newTestPrimaryAttribute(enum.Strength, &buff)
	mid := newTestMiddleAttribute(enum.Constitution, &buff, p1, p2)

	p1.IncreasePoints(3)
	p2.IncreasePoints(4)

	// avg(3, 4) = 3.5 → math.Round → 4
	got := mid.GetPoints()
	if got != 4 {
		t.Errorf("GetPoints avg(3,4): got %d, want 4", got)
	}
}

func TestMiddleAttribute_GetPoints_EvenAverage(t *testing.T) {
	buff := 0
	p1 := newTestPrimaryAttribute(enum.Resistance, &buff)
	p2 := newTestPrimaryAttribute(enum.Strength, &buff)
	mid := newTestMiddleAttribute(enum.Constitution, &buff, p1, p2)

	p1.IncreasePoints(4)
	p2.IncreasePoints(6)

	// avg(4, 6) = 5.0
	got := mid.GetPoints()
	if got != 5 {
		t.Errorf("GetPoints avg(4,6): got %d, want 5", got)
	}
}

func TestMiddleAttribute_GetAbilityBonus(t *testing.T) {
	buff := 0
	p1 := newTestPrimaryAttribute(enum.Resistance, &buff)
	p2 := newTestPrimaryAttribute(enum.Strength, &buff)
	mid := newTestMiddleAttribute(enum.Constitution, &buff, p1, p2)

	// Both primary attrs have ability bonus = 0 at start
	got := mid.GetAbilityBonus()
	if got != 0.0 {
		t.Errorf("initial ability bonus: got %f, want 0.0", got)
	}
}

func TestMiddleAttribute_GetPower(t *testing.T) {
	buff := 2
	p1 := newTestPrimaryAttribute(enum.Resistance, &buff)
	p2 := newTestPrimaryAttribute(enum.Strength, &buff)
	mid := newTestMiddleAttribute(enum.Constitution, &buff, p1, p2)

	p1.IncreasePoints(4)
	p2.IncreasePoints(6)

	// GetPower = GetValue() + int(AbilityBonus) + buff
	// GetValue() = GetPoints(5) + GetLevel(0) = 5
	// AbilityBonus = avg(0, 0) = 0
	// Power = 5 + 0 + 2 = 7
	got := mid.GetPower()
	if got != 7 {
		t.Errorf("GetPower: got %d, want 7", got)
	}
}

func TestMiddleAttribute_CascadeUpgrade(t *testing.T) {
	buff := 0
	p1 := newTestPrimaryAttribute(enum.Resistance, &buff)
	p2 := newTestPrimaryAttribute(enum.Strength, &buff)
	mid := newTestMiddleAttribute(enum.Constitution, &buff, p1, p2)

	cascade := experience.NewUpgradeCascade(100)
	mid.CascadeUpgrade(cascade)

	if mid.GetExpPoints() != 100 {
		t.Errorf("middle exp after cascade: got %d, want 100", mid.GetExpPoints())
	}

	entry, ok := cascade.Attributes[enum.Constitution]
	if !ok {
		t.Fatal("cascade.Attributes should contain Constitution entry")
	}
	if entry.Exp != mid.GetExpPoints() {
		t.Errorf("cascade exp: got %d, want %d", entry.Exp, mid.GetExpPoints())
	}

	// Primary attributes should also receive cascaded exp (100/2 = 50 each)
	if _, ok := cascade.Attributes[enum.Resistance]; !ok {
		t.Error("cascade should contain Resistance from primary cascade")
	}
	if _, ok := cascade.Attributes[enum.Strength]; !ok {
		t.Error("cascade should contain Strength from primary cascade")
	}

	// Each primary receives 50 exp
	if p1.GetExpPoints() != 50 {
		t.Errorf("p1 exp after cascade: got %d, want 50", p1.GetExpPoints())
	}
	if p2.GetExpPoints() != 50 {
		t.Errorf("p2 exp after cascade: got %d, want 50", p2.GetExpPoints())
	}
}

func TestMiddleAttribute_DelegatesExpMethods(t *testing.T) {
	buff := 0
	p1 := newTestPrimaryAttribute(enum.Resistance, &buff)
	mid := newTestMiddleAttribute(enum.Constitution, &buff, p1)

	if mid.GetNextLvlBaseExp() <= 0 {
		t.Error("GetNextLvlBaseExp should be positive")
	}
	if mid.GetNextLvlAggregateExp() <= 0 {
		t.Error("GetNextLvlAggregateExp should be positive")
	}
	if mid.GetCurrentExp() != 0 {
		t.Errorf("GetCurrentExp should be 0, got %d", mid.GetCurrentExp())
	}
}
