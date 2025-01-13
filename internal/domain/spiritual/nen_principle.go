package spiritual

import "github.com/422UR4H/HxH_RPG_System/internal/domain/experience"

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

func (np *NenPrinciple) GetPrinciplePower() int {
	return np.GetLevel() + np.abilityExp.GetLevel()/2
}

func (np *NenPrinciple) GetExpPoints() int {
	return np.exp.GetPoints()
}

func (np *NenPrinciple) GetLevel() int {
	return np.exp.GetLevel()
}

func (np *NenPrinciple) Clone() *NenPrinciple {
	return NewNenPrinciple(*np.exp.Clone(), np.abilityExp)
}
