package spiritual

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type NenPrinciple struct {
	name    enum.PrincipleName
	exp     experience.Exp
	ability ability.IAbility
}

func NewNenPrinciple(
	name enum.PrincipleName,
	exp experience.Exp,
	ability ability.IAbility,
) *NenPrinciple {
	return &NenPrinciple{name: name, exp: exp, ability: ability}
}

func (np *NenPrinciple) CascadeUpgradeTrigger(values *experience.UpgradeCascade) {
	np.exp.IncreasePoints(values.GetExp())
	np.ability.CascadeUpgrade(values)

	values.Principles[enum.Hatsu] = experience.PrincipleCascade{
		Lvl:     np.GetLevel(),
		Exp:     np.GetCurrentExp(),
		TestVal: np.GetValueForTest(),
	}
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

func (np *NenPrinciple) GetName() enum.PrincipleName {
	return np.name
}

func (np *NenPrinciple) Clone(name enum.PrincipleName) *NenPrinciple {
	return NewNenPrinciple(name, *np.exp.Clone(), np.ability)
}
