package skill

import (
	attr "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	exp "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
	status "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/status"
)

type PassiveSkill struct {
	name             enum.SkillName
	exp              exp.Exp
	attribute        attr.IGameAttribute
	abilitySkillsExp exp.IEndCascadeUpgrade
	status           status.IStatusBar
}

func NewPassiveSkill(
	name enum.SkillName,
	status status.IStatusBar,
	exp exp.Exp,
	attribute attr.IGameAttribute,
	abilitySkillsExp exp.IEndCascadeUpgrade) *PassiveSkill {

	skill := &PassiveSkill{
		name:             name,
		exp:              exp,
		status:           status,
		attribute:        attribute,
		abilitySkillsExp: abilitySkillsExp,
	}
	skill.status.Upgrade(skill.GetLevel())
	return skill
}

func (ps *PassiveSkill) CascadeUpgradeTrigger(
	values *exp.UpgradeCascade,
) {
	diff := ps.exp.IncreasePoints(values.GetExp())
	ps.attribute.CascadeUpgrade(values)
	ps.abilitySkillsExp.EndCascadeUpgrade(values)

	if diff != 0 {
		ps.status.Upgrade(ps.exp.GetLevel())
	}
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

func (ps *PassiveSkill) GetName() enum.SkillName {
	return ps.name
}

// func (ps *StatusSkill) Clone(points int) *StatusSkill {
// 	return NewCommonSkill(*ps.exp.Clone(), ps.attribute, ps.abilitySkillsExp)
// }

// status skilln (passive skill) estende person (common) skill. devo resolver isso para continuar
// agora não estende mais. ele implementará a interface ISkill assim como commonSkill faz
// resolvido!!
