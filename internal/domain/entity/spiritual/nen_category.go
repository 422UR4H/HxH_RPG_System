package spiritual

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/experience"
)

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

func (nc *NenCategory) CascadeUpgradeTrigger(exp int) int {
	diff := nc.exp.IncreasePoints(exp)
	nc.hatsuExp.CascadeUpgrade(exp)
	return diff
}

func (nc *NenCategory) GetExpPoints() int {
	return nc.exp.GetPoints()
}

func (nc *NenCategory) GetLevel() int {
	return nc.exp.GetLevel()
}

func (nc *NenCategory) Clone() *NenCategory {
	return NewNenCategory(*nc.exp.Clone(), nc.hatsuExp)
}
