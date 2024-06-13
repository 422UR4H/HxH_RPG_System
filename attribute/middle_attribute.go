package attribute

import (
	"math"

	exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type MiddleAttribute struct {
	exp               exp.Experience
	primaryAttributes []PrimaryAttribute
}

func NewMiddleAttribute(exp exp.Experience, primaryAttrs ...PrimaryAttribute) *MiddleAttribute {
	return &MiddleAttribute{exp: exp, primaryAttributes: primaryAttrs}
}

func (ma *MiddleAttribute) CascadeUpgrade(exp int) {
	ma.exp.IncreasePoints(exp)
	for _, attr := range ma.primaryAttributes {
		attr.CascadeUpgrade(exp)
	}
}

func (ma *MiddleAttribute) GetHalfOfAbilityLvl() float64 {
	if len(ma.primaryAttributes) == 0 {
		return 0
	}

	value := 0.0
	for _, primaryAttr := range ma.primaryAttributes {
		value += primaryAttr.GetHalfOfAbilityLvl()
	}
	return value / float64(len(ma.primaryAttributes))
}

func (ma *MiddleAttribute) GetExpPoints() int {
	return ma.exp.GetPoints()
}

func (ma *MiddleAttribute) GetPoints() int {
	points := 0
	for _, primaryAttr := range ma.primaryAttributes {
		points += primaryAttr.points
	}
	return int(math.Round(float64(points) / float64(len(ma.primaryAttributes))))
}

func (ma *MiddleAttribute) GetLevel() int {
	return ma.exp.GetLevel()
}

func (ma *MiddleAttribute) GetPower() int {
	return ma.GetPoints() + ma.GetLevel() + int(ma.GetHalfOfAbilityLvl())
}
