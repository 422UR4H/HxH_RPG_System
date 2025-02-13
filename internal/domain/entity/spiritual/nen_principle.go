package spiritual

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type NenPrinciple struct {
	exp     experience.Exp
	ability ability.IAbility
}

func NewNenPrinciple(
	exp experience.Exp,
	ability ability.IAbility,
) *NenPrinciple {
	return &NenPrinciple{exp: exp, ability: ability}
}

func (np *NenPrinciple) CascadeUpgradeTrigger(exp int) int {
	diff := np.exp.IncreasePoints(exp)
	np.ability.CascadeUpgrade(exp)
	return diff
}

func (np *NenPrinciple) GetValueForTest() int {
	return np.GetLevel() + int(np.ability.GetBonus())
}

func (np *NenPrinciple) GetNextLvlAggregateExp() int {
	return np.exp.GetNextLvlAggregateExp()
}

func (np *NenPrinciple) GetNextLvlBaseExp() int {
	return np.exp.GetNextLvlBaseExp()
}

func (np *NenPrinciple) GetCurrentExp() int {
	return np.exp.GetCurrentExp()
}

func (np *NenPrinciple) GetExpPoints() int {
	return np.exp.GetPoints()
}

func (np *NenPrinciple) GetLevel() int {
	return np.exp.GetLevel()
}

func (np *NenPrinciple) Clone() *NenPrinciple {
	return NewNenPrinciple(*np.exp.Clone(), np.ability)
}
