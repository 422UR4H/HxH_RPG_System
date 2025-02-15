package skill

import (
	attr "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

// TODO: update to JointSkill
// do not expose to users its in v0
type JointSkill struct {
	exp  experience.Exp
	name string
	buff int
	// ladinagem (roguery), caça (hunt), atleta (athletics?), hack
	attribute        attr.IGameAttribute
	commonSkills     map[enum.SkillName]ISkill
	abilitySkillsExp experience.IEndCascadeUpgrade
}

func NewJointSkill(
	exp experience.Exp,
	name string,
	buff int,
	attr attr.IGameAttribute,
	commonSkills map[enum.SkillName]ISkill,
	abilitySkillsExp experience.IEndCascadeUpgrade) *JointSkill {

	return &JointSkill{
		exp:              exp,
		name:             name,
		buff:             buff,
		attribute:        attr,
		commonSkills:     commonSkills,
		abilitySkillsExp: abilitySkillsExp,
	}
}

func (js *JointSkill) CascadeUpgradeTrigger(exp int) int {
	diff := js.exp.IncreasePoints(exp)
	js.attribute.CascadeUpgrade(exp)
	js.abilitySkillsExp.EndCascadeUpgrade(exp * len(js.commonSkills))
	return diff
}

func (js *JointSkill) GetValueForTest() int {
	return js.exp.GetLevel() + js.attribute.GetPower() + js.buff
}

func (js *JointSkill) GetNextLvlAggregateExp() int {
	return js.exp.GetNextLvlAggregateExp()
}

func (js *JointSkill) GetNextLvlBaseExp() int {
	return js.exp.GetNextLvlBaseExp()
}

func (js *JointSkill) GetCurrentExp() int {
	return js.exp.GetCurrentExp()
}

func (js *JointSkill) GetExpPoints() int {
	return js.exp.GetPoints()
}

func (js *JointSkill) GetLevel() int {
	return js.exp.GetLevel()
}

func (js *JointSkill) Contains(name enum.SkillName) bool {
	for key := range js.commonSkills {
		if key == name {
			return true
		}
	}
	return false
}

func (js *JointSkill) GetName() string {
	return js.name
}

func (js *JointSkill) GetBuff() int {
	return js.buff
}

func (js *JointSkill) SetBuff(buff int) {
	js.buff = buff
}

func (js *JointSkill) GetCommonSkills() map[enum.SkillName]ISkill {
	return js.commonSkills
}
