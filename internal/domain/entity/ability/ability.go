package ability

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type Ability struct {
	name    enum.AbilityName
	exp     experience.Exp
	charExp experience.ICharacterExp
}

func NewAbility(
	name enum.AbilityName, exp experience.Exp, charExp experience.ICharacterExp,
) *Ability {
	return &Ability{name: name, exp: exp, charExp: charExp}
}

// maybe character points should only go down for training
// in this case, change this name to GetTrainingBonus
// and create another GetBonus that only lowers your level / 2
func (a *Ability) GetBonus() float64 {
	pts := float64(a.charExp.GetCharacterPoints())
	lvl := float64(a.exp.GetLevel())
	return (pts + lvl) / 2.0
}

func (a *Ability) CascadeUpgrade(values *experience.UpgradeCascade) {
	diff := a.exp.IncreasePoints(values.GetExp())
	a.charExp.EndCascadeUpgrade(values)

	if diff > 0 {
		a.charExp.IncreaseCharacterPoints(diff)
	}

	values.Abilities[a.name] = experience.AbilityCascade{
		Exp:   a.GetExpPoints(),
		Lvl:   a.GetLevel(),
		Bonus: a.GetBonus(),
	}
}

func (a *Ability) GetNextLvlBaseExp() int {
	return a.exp.GetNextLvlBaseExp()
}

func (a *Ability) GetNextLvlAggregateExp() int {
	return a.exp.GetNextLvlAggregateExp()
}

func (a *Ability) GetCurrentExp() int {
	return a.exp.GetCurrentExp()
}

func (a *Ability) GetExpPoints() int {
	return a.exp.GetPoints()
}

func (a *Ability) GetLevel() int {
	return a.exp.GetLevel()
}

func (a *Ability) GetName() enum.AbilityName {
	return a.name
}

func (a *Ability) GetExpReference() experience.ICascadeUpgrade {
	return a
}
