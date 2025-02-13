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

func (pa *PrimaryAttribute) GetPower() int {
	return pa.points + pa.GetLevel() + int(pa.GetBonus()) + *pa.buff
}

func (pa *PrimaryAttribute) GetPoints() int {
	return pa.points
}

func (pa *PrimaryAttribute) GetBonus() float64 {
	return pa.ability.GetBonus()
}

func (pa *PrimaryAttribute) GetNextLvlAggregateExp() int {
	return pa.exp.GetNextLvlAggregateExp()
}

func (pa *PrimaryAttribute) GetNextLvlBaseExp() int {
	return pa.exp.GetNextLvlBaseExp()
}

func (pa *PrimaryAttribute) GetCurrentExp() int {
	return pa.exp.GetCurrentExp()
}

func (pa *PrimaryAttribute) GetExpPoints() int {
	return pa.exp.GetPoints()
}

func (pa *PrimaryAttribute) GetLevel() int {
	return pa.exp.GetLevel()
}

func (pa *PrimaryAttribute) Clone(buff *int) *PrimaryAttribute {
	return NewPrimaryAttribute(*pa.exp.Clone(), pa.ability, buff)
}
