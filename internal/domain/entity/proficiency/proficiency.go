package proficiency

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

// TODO: upgrade adding strike (GetStrike) and hit
type Proficiency struct {
	exp           experience.Exp
	physSkillsExp experience.ICascadeUpgrade
}

func NewProficiency(
	exp experience.Exp,
	// weapon enum.WeaponName,
	physSkExp experience.ICascadeUpgrade,
) *Proficiency {
	return &Proficiency{
		exp: exp,
		// weapon:        weapon,
		physSkillsExp: physSkExp,
	}
}

func (p *Proficiency) CascadeUpgradeTrigger(exp int) int {
	diff := p.exp.IncreasePoints(exp)
	p.physSkillsExp.CascadeUpgrade(exp)
	return diff
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
