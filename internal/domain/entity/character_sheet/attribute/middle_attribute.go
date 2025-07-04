package attribute

import (
	"math"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type MiddleAttribute struct {
	name         enum.AttributeName
	exp          experience.Exp
	buff         *int
	primaryAttrs []*PrimaryAttribute
}

func NewMiddleAttribute(
	name enum.AttributeName,
	exp experience.Exp,
	buff *int,
	primaryAttrs ...*PrimaryAttribute,
) *MiddleAttribute {
	return &MiddleAttribute{
		name: name, exp: exp, buff: buff, primaryAttrs: primaryAttrs,
	}
}

// TODO: test for lenAttrs > 2 (and for lenAttrs = 2)
// maybe move or share this logic with UpgradeCascade
func (ma *MiddleAttribute) CascadeUpgrade(values *experience.UpgradeCascade) {
	lenAttrs := len(ma.primaryAttrs)
	remainder := ma.exp.GetPoints() % lenAttrs

	ma.exp.IncreasePoints(values.GetExp())

	exp := remainder + values.GetExp()
	exp /= lenAttrs
	values.SetExp(exp)
	values.Attributes[ma.name] = experience.AttributeCascade{
		Exp:   ma.GetExpPoints(),
		Lvl:   ma.GetLevel(),
		Power: ma.GetPower(),
	}

	for _, attr := range ma.primaryAttrs {
		attr.CascadeUpgrade(values)
	}
}

func (ma *MiddleAttribute) GetAbilityBonus() float64 {
	lenAttrs := len(ma.primaryAttrs)
	if lenAttrs == 0 {
		return 0
	}

	value := 0.0
	for _, primaryAttr := range ma.primaryAttrs {
		value += primaryAttr.GetAbilityBonus()
	}
	return value / float64(lenAttrs)
}

func (ma *MiddleAttribute) GetPoints() int {
	points := 0
	for _, primaryAttr := range ma.primaryAttrs {
		points += primaryAttr.points
	}
	return int(math.Round(float64(points) / float64(len(ma.primaryAttrs))))
}

func (ma *MiddleAttribute) GetValue() int {
	return ma.GetPoints() + ma.GetLevel()
}

func (ma *MiddleAttribute) GetPower() int {
	return ma.GetPoints() + ma.GetLevel() + int(ma.GetAbilityBonus()) + *ma.buff
}

func (ma *MiddleAttribute) GetNextLvlAggregateExp() int {
	return ma.exp.GetNextLvlAggregateExp()
}

func (ma *MiddleAttribute) GetNextLvlBaseExp() int {
	return ma.exp.GetNextLvlBaseExp()
}

func (ma *MiddleAttribute) GetCurrentExp() int {
	return ma.exp.GetCurrentExp()
}

func (ma *MiddleAttribute) GetExpPoints() int {
	return ma.exp.GetPoints()
}

func (ma *MiddleAttribute) GetLevel() int {
	return ma.exp.GetLevel()
}

func (ma *MiddleAttribute) GetName() enum.AttributeName {
	return ma.name
}
