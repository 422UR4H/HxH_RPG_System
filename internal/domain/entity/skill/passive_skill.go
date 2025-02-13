package skill

import (
	attr "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	exp "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
	status "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/status"
)

type PassiveSkill struct {
	exp              exp.Exp
	attribute        attr.IGameAttribute
	abilitySkillsExp exp.IEndCascadeUpgrade
	status           status.IStatusBar
}

func NewPassiveSkill(
	status status.IStatusBar,
	exp exp.Exp,
	attribute attr.IGameAttribute,
	abilitySkillsExp exp.IEndCascadeUpgrade) *PassiveSkill {

	skill := &PassiveSkill{
		exp:              exp,
		status:           status,
		attribute:        attribute,
		abilitySkillsExp: abilitySkillsExp,
	}
	skill.status.Upgrade(skill.GetLevel())
	return skill
}

func (ps *PassiveSkill) CascadeUpgradeTrigger(exp int) int {
	diff := ps.exp.IncreasePoints(exp)
	ps.attribute.CascadeUpgrade(exp)
	ps.abilitySkillsExp.EndCascadeUpgrade(exp)

	if diff != 0 {
		ps.status.Upgrade(ps.exp.GetLevel())
	}
	return diff
}

func (ps *PassiveSkill) GetValueForTest() int {
	return ps.exp.GetLevel() + ps.attribute.GetPower()
}

func (ps *PassiveSkill) GetNextLvlAggregateExp() int {
	return ps.exp.GetNextLvlAggregateExp()
}

func (ps *PassiveSkill) GetNextLvlBaseExp() int {
	return ps.exp.GetNextLvlBaseExp()
}

func (ps *PassiveSkill) GetCurrentExp() int {
	return ps.exp.GetCurrentExp()
}

func (ps *PassiveSkill) GetExpPoints() int {
	return ps.exp.GetPoints()
}

func (ps *PassiveSkill) GetLevel() int {
	return ps.exp.GetLevel()
}

// func (ps *StatusSkill) Clone(points int) *StatusSkill {
// 	return NewCommonSkill(*ps.exp.Clone(), ps.attribute, ps.abilitySkillsExp)
// }

// status skilln (passive skill) estende person (common) skill. devo resolver isso para continuar
// agora não estende mais. ele implementará a interface ISkill assim como commonSkill faz
// resolvido!!
