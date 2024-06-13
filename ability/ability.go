package ability

import exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"

type Ability struct {
	exp          exp.Experience
	characterExp exp.CharacterExp
}

func NewAbility(exp exp.Experience, charExp exp.CharacterExp) *Ability {
	return &Ability{exp: exp, characterExp: charExp}
}

func (a *Ability) GetHalfLvl() float64 {
	return float64(a.exp.GetPoints()) / 2.0
}

func (a *Ability) CascadeUpgrade(exp int) {
	a.exp.IncreasePoints(exp)
	a.characterExp.TriggerEndUpgrade(exp)
}

func (a *Ability) GetExpPoints() int {
	return a.exp.GetPoints()
}

func (a *Ability) GetLevel() int {
	return a.exp.GetLevel()
}
