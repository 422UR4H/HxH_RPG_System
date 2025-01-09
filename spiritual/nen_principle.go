package spiritual

import "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"

type NenPrinciple struct {
	exp        experience.Exp
	abilityExp experience.ICascadeUpgrade
}

func NewNenPrinciple(
	exp experience.Exp,
	abilityExp experience.ICascadeUpgrade,
) *NenPrinciple {
	return &NenPrinciple{exp: exp, abilityExp: abilityExp}
}

func (np *NenPrinciple) CascadeUpgradeTrigger(exp int) int {
	diff := np.exp.IncreasePoints(exp)
	np.abilityExp.CascadeUpgrade(exp)
	return diff
}

func (np *NenPrinciple) GetExpPoints() int {
	return np.exp.GetPoints()
}

func (np *NenPrinciple) GetLvl() int {
	return np.exp.GetLevel()
}

// func Clone() *NenPrinciple {
// 	return NewNenPrinciple(experience.Clone(), AbilityExp);
// }
