package proficiency

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

// TODO: upgrade adding strike (GetStrike) and hit
type Proficiency struct {
	weapon        enum.WeaponName
	exp           experience.Exp
	physSkillsExp experience.ICascadeUpgrade
}

func NewProficiency(
	weapon enum.WeaponName,
	exp experience.Exp,
	physSkExp experience.ICascadeUpgrade,
) *Proficiency {
	return &Proficiency{
		weapon:        weapon,
		exp:           exp,
		physSkillsExp: physSkExp,
	}
}

func (p *Proficiency) CascadeUpgradeTrigger(values *experience.UpgradeCascade) {
	p.exp.IncreasePoints(values.GetExp())
	p.physSkillsExp.CascadeUpgrade(values)
}

// TODO: validate this
func (p *Proficiency) GetValueForTest() int {
	return p.exp.GetLevel() //+ p.attribute.GetPower()
}

func (p *Proficiency) GetNextLvlAggregateExp() int {
	return p.exp.GetNextLvlAggregateExp()
}

func (p *Proficiency) GetNextLvlBaseExp() int {
	return p.exp.GetNextLvlBaseExp()
}

func (p *Proficiency) GetCurrentExp() int {
	return p.exp.GetCurrentExp()
}

func (p *Proficiency) GetExpPoints() int {
	return p.exp.GetPoints()
}

func (p *Proficiency) GetLevel() int {
	return p.exp.GetLevel()
}

func (p *Proficiency) GetWeapon() enum.WeaponName {
	return p.weapon
}
