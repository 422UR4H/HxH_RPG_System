package skill

import (
	attr "github.com/422UR4H/HxH_RPG_Environment.Domain/attribute"
	exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
	status "github.com/422UR4H/HxH_RPG_Environment.Domain/status"
)

type StatusSkill struct {
	commonSkill CommonSkill
	status      status.IStatus
}

func NewStatusSkill(
	status status.IStatus,
	exp exp.Experience,
	attribute attr.IGameAttribute,
	abilitySkillsExp exp.IEndCascadeUpgrade) *StatusSkill {

	skill := &StatusSkill{
		commonSkill: *NewCommonSkill(exp, attribute, abilitySkillsExp),
		status:      status,
	}
	skill.status.Upgrade(skill.GetLvl())
	return skill
}

func (ss *StatusSkill) TriggerCascadeUpgrade(exp int) {
	diff := ss.commonSkill.IncreaseExp(exp)
	ss.commonSkill.CascadeUpgrade(exp)
	ss.AbilitySkillsExp.TriggerEndUpgrade(exp)

	if diff != 0 {
		ss.Status.Upgrade(ss.Exp.GetLvl())
	}
}

func (ss *StatusSkill) IncreaseExp(points int) int {
	return ss.IncreaseExp(points)
}

func (ss *StatusSkill) GetValueForTest() int {
	return ss.GetValueForTest()
}

func (ss *StatusSkill) GetExpPoints() int {
	return ss.GetExpPoints()
}

func (ss *StatusSkill) GetLvl() int {
	return ss.GetLvl()
}

// func (ss *StatusSkill) Clone(points int) *StatusSkill {
// 	return NewCommonSkill(*ss.exp.Clone(), ss.attribute, ss.abilitySkillsExp)
// }

// status skill estende person (common) skill. devo resolver isso para continuar
