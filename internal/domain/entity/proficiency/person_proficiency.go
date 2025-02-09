package proficiency

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

// TODO: upgrade adding strike (GetStrike) and hit
type PersonProficiency struct {
	exp           experience.Exp
	name          string
	weapons       []enum.WeaponName
	physSkillsExp experience.ICascadeUpgrade
}

func NewPersonProficiency(
	exp experience.Exp,
	name string,
	weapons []enum.WeaponName,
	physSkExp experience.ICascadeUpgrade,
) *PersonProficiency {
	return &PersonProficiency{
		exp:           exp,
		name:          name,
		weapons:       weapons,
		physSkillsExp: physSkExp,
	}
}

func (pp *PersonProficiency) CascadeUpgradeTrigger(exp int) int {
	diff := pp.exp.IncreasePoints(exp)
	pp.physSkillsExp.CascadeUpgrade(exp)
	return diff
}

// TODO: validate this
func (pp *PersonProficiency) GetValueForTest() int {
	return pp.exp.GetLevel() //+ pp.attribute.GetPower()
}

func (pp *PersonProficiency) GetExpPoints() int {
	return pp.exp.GetPoints()
}

func (pp *PersonProficiency) GetLevel() int {
	return pp.exp.GetLevel()
}

func (pp *PersonProficiency) GetName() string {
	return pp.name
}

func (pp *PersonProficiency) ContainsWeapon(name enum.WeaponName) bool {
	for _, weapon := range pp.weapons {
		if weapon == name {
			return true
		}
	}
	return false
}

func (pp *PersonProficiency) AddWeapon(name enum.WeaponName) {
	pp.weapons = append(pp.weapons, name)
}

func (pp *PersonProficiency) GetAggregateExpByLvl(lvl int) int {
	return pp.exp.GetAggregateExpByLvl(lvl)
}
