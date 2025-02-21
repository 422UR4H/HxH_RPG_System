package skill

import (
	"fmt"

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
	abilitySkillsExp experience.ICascadeUpgrade
}

func NewJointSkill(
	exp experience.Exp,
	name string,
	attr attr.IGameAttribute,
	commonSkills map[enum.SkillName]ISkill) *JointSkill {

	return &JointSkill{
		exp:          exp,
		name:         name,
		attribute:    attr,
		commonSkills: commonSkills,
	}
}

func (js *JointSkill) Init(abilitySkillsExp experience.ICascadeUpgrade) error {
	if js.abilitySkillsExp != nil {
		return fmt.Errorf("abilitySkillsExp already initialized")
	}
	if abilitySkillsExp == nil {
		return fmt.Errorf("abilitySkillsExp cannot be nil")
	}
	js.abilitySkillsExp = abilitySkillsExp
	return nil
}

func (js *JointSkill) IsInitialized() bool {
	return js.abilitySkillsExp != nil
}

func (js *JointSkill) CascadeUpgradeTrigger(values *experience.UpgradeCascade) {
	exp := values.GetExp()
	js.exp.IncreasePoints(exp)
	js.attribute.CascadeUpgrade(values)

	values.SetExp(exp * len(js.commonSkills))
	js.abilitySkillsExp.CascadeUpgrade(values)
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
