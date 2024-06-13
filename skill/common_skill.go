package skill

import (
	attr "github.com/422UR4H/HxH_RPG_Environment.Domain/attribute"
	exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type CommonSkill struct {
	exp              exp.Experience
	attribute        attr.IGameAttribute
	abilitySkillsExp exp.IEndCascadeUpgrade
}

func NewCommonSkill(
	exp exp.Experience,
	attr attr.IGameAttribute,
	abilitySkillsExp exp.IEndCascadeUpgrade) *CommonSkill {

	return &CommonSkill{exp: exp, attribute: attr, abilitySkillsExp: abilitySkillsExp}
}

func (cs *CommonSkill) TriggerEndUpgrade(exp int) {
	cs.exp.IncreasePoints(exp)
	cs.attribute.CascadeUpgrade(exp)
	cs.abilitySkillsExp.TriggerEndUpgrade(exp)
}

func (cs *CommonSkill) IncreaseExp(points int) int {
	return cs.exp.IncreasePoints(points)
}

func (cs *CommonSkill) GetValueForTest() int {
	return cs.exp.GetLevel() + cs.attribute.GetPower()
}

func (cs *CommonSkill) GetExpPoints() int {
	return cs.exp.GetPoints()
}

func (cs *CommonSkill) GetLvl() int {
	return cs.exp.GetLevel()
}

func (cs *CommonSkill) Clone(points int) *CommonSkill {
	return NewCommonSkill(*cs.exp.Clone(), cs.attribute, cs.abilitySkillsExp)
}
