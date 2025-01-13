package skill

import (
	attr "github.com/422UR4H/HxH_RPG_System/internal/domain/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/experience"
)

type CommonSkill struct {
	exp              experience.Exp
	attribute        attr.IGameAttribute
	abilitySkillsExp experience.IEndCascadeUpgrade
}

func NewCommonSkill(
	exp experience.Exp,
	attr attr.IGameAttribute,
	abilitySkillsExp experience.IEndCascadeUpgrade) *CommonSkill {

	return &CommonSkill{exp: exp, attribute: attr, abilitySkillsExp: abilitySkillsExp}
}

func (cs *CommonSkill) CascadeUpgradeTrigger(exp int) int {
	diff := cs.exp.IncreasePoints(exp)
	cs.attribute.CascadeUpgrade(exp)
	cs.abilitySkillsExp.EndCascadeUpgrade(exp)
	return diff
}

func (cs *CommonSkill) GetValueForTest() int {
	return cs.exp.GetLevel() + cs.attribute.GetPower()
}

func (cs *CommonSkill) GetExpPoints() int {
	return cs.exp.GetPoints()
}

func (cs *CommonSkill) GetLevel() int {
	return cs.exp.GetLevel()
}

func (cs *CommonSkill) Clone() *CommonSkill {
	return NewCommonSkill(*cs.exp.Clone(), cs.attribute, cs.abilitySkillsExp)
}
