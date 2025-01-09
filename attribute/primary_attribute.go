package attribute

import (
	"github.com/422UR4H/HxH_RPG_Environment.Domain/ability"
	"github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type PrimaryAttribute struct {
	points  int
	exp     experience.Exp
	ability ability.IAbility
}

func NewPrimaryAttribute(
	exp experience.Exp,
	ability ability.IAbility,
) *PrimaryAttribute {
	return &PrimaryAttribute{exp: exp, ability: ability, points: 0}
}

func (pa PrimaryAttribute) CascadeUpgrade(exp int) {
	pa.exp.IncreasePoints(exp)
	pa.ability.CascadeUpgrade(exp)
}

func (pa PrimaryAttribute) GetHalfOfAbilityLvl() float64 {
	return pa.ability.GetHalfLvl()
}

func (pa PrimaryAttribute) GetExpPoints() int {
	return pa.exp.GetPoints()
}

func (pa PrimaryAttribute) GetPoints() int {
	return pa.points
}

func (pa PrimaryAttribute) GetLevel() int {
	return pa.exp.GetLevel()
}

func (pa PrimaryAttribute) GetPower() int {
	return pa.points + pa.GetLevel() + int(pa.GetHalfOfAbilityLvl())
}

func (pa PrimaryAttribute) Clone() *PrimaryAttribute {
	return NewPrimaryAttribute(*pa.exp.Clone(), pa.ability)
}
