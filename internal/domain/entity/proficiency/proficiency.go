package proficiency

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
)

// TODO: add strike and hit
type Proficiency struct {
	exp    experience.Exp
	weapon enum.WeaponName
}

func NewProficiency(exp experience.Exp, weapon enum.WeaponName) *Proficiency {
	return &Proficiency{exp: exp, weapon: weapon}
}

func (p *Proficiency) CascadeUpgradeTrigger(exp int, skillExp skill.ISkill) int {
	diff := p.exp.IncreasePoints(exp)
	skillExp.CascadeUpgradeTrigger(exp)
	return diff
}

func (p *Proficiency) GetExpPoints() int {
	return p.exp.GetPoints()
}

func (p *Proficiency) GetLevel() int {
	return p.exp.GetLevel()
}

func (p *Proficiency) GetAggregateExpByLvl(lvl int) int {
	return p.exp.GetAggregateExpByLvl(lvl)
}
