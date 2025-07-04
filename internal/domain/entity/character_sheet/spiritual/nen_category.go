package spiritual

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type NenCategory struct {
	exp   experience.Exp
	name  enum.CategoryName
	hatsu IHatsu
}

func NewNenCategory(
	exp experience.Exp,
	name enum.CategoryName,
	hatsu IHatsu,
) *NenCategory {
	return &NenCategory{exp: exp, name: name, hatsu: hatsu}
}

func (nc *NenCategory) CascadeUpgradeTrigger(values *experience.UpgradeCascade) {
	nc.exp.IncreasePoints(values.GetExp())
	nc.hatsu.CascadeUpgrade(values)

	values.Principles[enum.Hatsu] = experience.PrincipleCascade{
		Lvl:     nc.GetLevel(),
		Exp:     nc.GetCurrentExp(),
		TestVal: nc.GetValueForTest(),
	}
}

func (nc *NenCategory) GetNextLvlAggregateExp() int {
	return nc.exp.GetNextLvlAggregateExp()
}

func (nc *NenCategory) GetNextLvlBaseExp() int {
	return nc.exp.GetNextLvlBaseExp()
}

func (nc *NenCategory) GetCurrentExp() int {
	return nc.exp.GetCurrentExp()
}

func (nc *NenCategory) GetExpPoints() int {
	return nc.exp.GetPoints()
}

func (nc *NenCategory) GetLevel() int {
	return nc.exp.GetLevel()
}

func (nc *NenCategory) GetPercent() float64 {
	return nc.hatsu.GetPercentOf(nc.name)
}

func (nc *NenCategory) GetValueForTest() int {
	return int(float64(nc.GetLevel()) * nc.GetPercent() / 100.0)
}

func (nc *NenCategory) GetName() enum.CategoryName {
	return nc.name
}

func (nc *NenCategory) Clone(name enum.CategoryName) *NenCategory {
	return NewNenCategory(*nc.exp.Clone(), name, nc.hatsu)
}
