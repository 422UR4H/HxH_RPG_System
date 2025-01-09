package attribute

import (
	"math"

	"github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type MiddleAttribute struct {
	exp          experience.Exp
	primaryAttrs []PrimaryAttribute
}

func NewMiddleAttribute(
	exp experience.Exp,
	primaryAttrs ...PrimaryAttribute,
) *MiddleAttribute {
	return &MiddleAttribute{exp: exp, primaryAttrs: primaryAttrs}
}

func (ma MiddleAttribute) CascadeUpgrade(exp int) {
	ma.exp.IncreasePoints(exp)
	for _, attr := range ma.primaryAttrs {
		attr.CascadeUpgrade(exp)
	}
}

func (ma MiddleAttribute) GetHalfOfAbilityLvl() float64 {
	if len(ma.primaryAttrs) == 0 {
		return 0
	}

	value := 0.0
	for _, primaryAttr := range ma.primaryAttrs {
		value += primaryAttr.GetHalfOfAbilityLvl()
	}
	return value / float64(len(ma.primaryAttrs))
}

func (ma MiddleAttribute) GetExpPoints() int {
	return ma.exp.GetPoints()
}

func (ma MiddleAttribute) GetPoints() int {
	points := 0
	for _, primaryAttr := range ma.primaryAttrs {
		points += primaryAttr.points
	}
	return int(math.Round(float64(points) / float64(len(ma.primaryAttrs))))
}

func (ma MiddleAttribute) GetLevel() int {
	return ma.exp.GetLevel()
}

func (ma MiddleAttribute) GetPower() int {
	return ma.GetPoints() + ma.GetLevel() + int(ma.GetHalfOfAbilityLvl())
}
