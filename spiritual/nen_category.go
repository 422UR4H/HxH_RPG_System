package spiritual

import "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"

type NenCategory struct {
	exp      experience.Exp
	hatsuExp experience.ICascadeUpgrade
}

func NewNenCategory(
	exp experience.Exp,
	hatsuExp experience.ICascadeUpgrade,
) *NenCategory {
	return &NenCategory{exp: exp, hatsuExp: hatsuExp}
}

func (np *NenCategory) CascadeUpgradeTrigger(exp int) int {
	diff := np.exp.IncreasePoints(exp)
	np.hatsuExp.CascadeUpgrade(exp)
	return diff
}

func (np *NenCategory) GetExpPoints() int {
	return np.exp.GetPoints()
}

func (np *NenCategory) GetLevel() int {
	return np.exp.GetLevel()
}

// func Clone() *NenCategory {
// 	return NewNenPrinciple(experience.Clone(), AbilityExp);
// }
