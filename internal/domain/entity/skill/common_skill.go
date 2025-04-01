package skill

import (
	attr "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type CommonSkill struct {
	name             enum.SkillName
	exp              experience.Exp
	attribute        attr.IGameAttribute
	abilitySkillsExp experience.ICascadeUpgrade
}

func NewCommonSkill(
	name enum.SkillName,
	exp experience.Exp,
	attr attr.IGameAttribute,
	abilitySkillsExp experience.ICascadeUpgrade) *CommonSkill {
	return &CommonSkill{
		name: name, exp: exp, attribute: attr, abilitySkillsExp: abilitySkillsExp,
	}
}

func (cs *CommonSkill) CascadeUpgradeTrigger(values *experience.UpgradeCascade) {
	cs.exp.IncreasePoints(values.GetExp())
	cs.attribute.CascadeUpgrade(values)
	cs.abilitySkillsExp.CascadeUpgrade(values)

	values.Skills[cs.name.String()] = experience.SkillCascade{
		Lvl:     cs.GetLevel(),
		Exp:     cs.GetCurrentExp(),
		TestVal: cs.GetValueForTest(),
	}
}

func (cs *CommonSkill) GetValueForTest() int {
	return cs.exp.GetLevel() + cs.attribute.GetPower()
}

func (cs *CommonSkill) GetNextLvlAggregateExp() int {
	return cs.exp.GetNextLvlAggregateExp()
}

func (cs *CommonSkill) GetNextLvlBaseExp() int {
	return cs.exp.GetNextLvlBaseExp()
}

func (cs *CommonSkill) GetCurrentExp() int {
	return cs.exp.GetCurrentExp()
}

func (cs *CommonSkill) GetExpPoints() int {
	return cs.exp.GetPoints()
}

func (cs *CommonSkill) GetLevel() int {
	return cs.exp.GetLevel()
}

func (cs *CommonSkill) GetName() enum.SkillName {
	return cs.name
}

func (cs *CommonSkill) Clone(name enum.SkillName) *CommonSkill {
	return NewCommonSkill(name, *cs.exp.Clone(), cs.attribute, cs.abilitySkillsExp)
}
