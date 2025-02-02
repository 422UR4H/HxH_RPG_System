package attribute

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type PrimaryAttribute struct {
	points  int
	exp     experience.Exp
	ability ability.IAbility
	buff    *int
}

func NewPrimaryAttribute(
	exp experience.Exp,
	ability ability.IAbility,
	buff *int,
) *PrimaryAttribute {
	return &PrimaryAttribute{exp: exp, ability: ability, points: 0, buff: buff}
}

func (pa *PrimaryAttribute) CascadeUpgrade(exp int) {
	pa.exp.IncreasePoints(exp)
	pa.ability.CascadeUpgrade(exp)
}

func (pa *PrimaryAttribute) GetBonus() float64 {
	return pa.ability.GetBonus()
}

func (pa *PrimaryAttribute) GetExpPoints() int {
	return pa.exp.GetPoints()
}

func (pa *PrimaryAttribute) GetPoints() int {
	return pa.points
}

func (pa *PrimaryAttribute) GetLevel() int {
	return pa.exp.GetLevel()
}

func (pa *PrimaryAttribute) GetPower() int {
	return pa.points + pa.GetLevel() + int(pa.GetBonus()) + *pa.buff
}

func (pa *PrimaryAttribute) Clone(buff *int) *PrimaryAttribute {
	return NewPrimaryAttribute(*pa.exp.Clone(), pa.ability, buff)
}
