package skill

import (
	attributes "github.com/422UR4H/HxH_RPG_Environment.Domain/attribute"
	experiences "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
	status "github.com/422UR4H/HxH_RPG_Environment.Domain/status"
)

type StatusSkill struct {
	commonSkill CommonSkill
	status      status.IStatus
}

func NewStatusSkill(
	status status.IStatus,
	exp experiences.Experience,
	attribute attributes.IGameAttribute,
	abilitySkillsExp IEndCascadeUpgrade) *StatusSkill {

	skill := &StatusSkill{
		commonSkill: *NewCommonSkill(exp, attribute, abilitySkillsExp),
		status:      status,
	}
	skill.status.StatusUpgrade(skill.GetLvl())
	return skill
}

func (ss *StatusSkill) TriggerCascadeUpgrade(exp int) {
	diff := ss.Exp.IncreasePoints(exp)
	ss.Attribute.CascadeUpgrade(exp)
	ss.AbilitySkillsExp.TriggerEndUpgrade(exp)

	if diff != 0 {
		ss.Status.StatusUpgrade(ss.Exp.GetLvl())
	}
}
