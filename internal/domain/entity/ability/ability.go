package ability

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"

type Ability struct {
	exp     experience.Exp
	charExp experience.CharacterExp
}

func NewAbility(exp experience.Exp, charExp experience.CharacterExp) *Ability {
	return &Ability{exp: exp, charExp: charExp}
}

func (a *Ability) GetHalfLvl() float64 {
	return float64(a.exp.GetPoints()) / 2.0
}

func (a *Ability) CascadeUpgrade(exp int) {
	a.exp.IncreasePoints(exp)
	a.charExp.EndCascadeUpgrade(exp)
}

func (a *Ability) GetExpPoints() int {
	return a.exp.GetPoints()
}

func (a *Ability) GetLevel() int {
	return a.exp.GetLevel()
}
