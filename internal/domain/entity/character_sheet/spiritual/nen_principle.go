package spiritual

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type NenPrinciple struct {
	name          enum.PrincipleName
	exp           experience.Exp
	flameNen      attribute.IGameAttribute
	conscienceNen attribute.IGameAttribute
}

func NewNenPrinciple(
	name enum.PrincipleName,
	exp experience.Exp,
	flameNen attribute.IGameAttribute,
	conscienceNen attribute.IGameAttribute,
) *NenPrinciple {
	return &NenPrinciple{
		name:          name,
		exp:           exp,
		flameNen:      flameNen,
		conscienceNen: conscienceNen,
	}
}

func (np *NenPrinciple) CascadeUpgradeTrigger(values *experience.UpgradeCascade) {
	np.exp.IncreasePoints(values.GetExp())
	// only upgrade conscienceNen
	np.conscienceNen.CascadeUpgrade(values)

	values.Principles[enum.Hatsu] = experience.PrincipleCascade{
		Lvl:     np.GetLevel(),
		Exp:     np.GetCurrentExp(),
		TestVal: np.GetValueForTest(),
	}
}

func (np *NenPrinciple) GetValueForTest() int {
	flameNenlvl := np.flameNen.GetLevel()
	return np.GetLevel() + int(np.conscienceNen.GetPower()) + flameNenlvl
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
	return NewNenPrinciple(name, *np.exp.Clone(), np.flameNen, np.conscienceNen)
}
