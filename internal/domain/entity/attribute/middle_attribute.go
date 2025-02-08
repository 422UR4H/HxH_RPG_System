package attribute

import (
	"math"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type MiddleAttribute struct {
	exp          experience.Exp
	buff         *int
	primaryAttrs []*PrimaryAttribute
}

func NewMiddleAttribute(
	exp experience.Exp,
	buff *int,
	primaryAttrs ...*PrimaryAttribute,
) *MiddleAttribute {
	return &MiddleAttribute{exp: exp, buff: buff, primaryAttrs: primaryAttrs}
}

// TODO: test for lenAttrs > 2 (and for lenAttrs = 2)
func (ma *MiddleAttribute) CascadeUpgrade(exp int) {
	lenAttrs := len(ma.primaryAttrs)
	remainder := ma.exp.GetPoints() % lenAttrs

	ma.exp.IncreasePoints(exp)

	exp += remainder
	exp /= lenAttrs
	for _, attr := range ma.primaryAttrs {
		attr.CascadeUpgrade(exp)
	}
}

func (ma *MiddleAttribute) GetBonus() float64 {
	lenAttrs := len(ma.primaryAttrs)
	if lenAttrs == 0 {
		return 0
	}

	value := 0.0
	for _, primaryAttr := range ma.primaryAttrs {
		value += primaryAttr.GetBonus()
	}
	return value / float64(lenAttrs)
}

func (ma *MiddleAttribute) GetExpPoints() int {
	return ma.exp.GetPoints()
}

func (ma *MiddleAttribute) GetPoints() int {
	points := 0
	for _, primaryAttr := range ma.primaryAttrs {
		points += primaryAttr.points
	}
	return int(math.Round(float64(points) / float64(len(ma.primaryAttrs))))
}

func (ma *MiddleAttribute) GetLevel() int {
	return ma.exp.GetLevel()
}

func (ma *MiddleAttribute) GetPower() int {
	return ma.GetPoints() + ma.GetLevel() + int(ma.GetBonus()) + *ma.buff
}
