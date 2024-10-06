package skill

import (
	"errors"
	"fmt"

	enum "github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
	exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type SkillsManager struct {
	skills     map[enum.SkillName]ISkill
	exp        exp.Experience
	skillsExp  exp.ICascadeUpgrade
	abilityExp exp.ICascadeUpgrade
}

func NewSkillsManager(
	exp exp.Experience,
	skillsExp exp.ICascadeUpgrade,
	abilityExp exp.ICascadeUpgrade) *SkillsManager {

	return &SkillsManager{
		skills:     make(map[enum.SkillName]ISkill),
		exp:        exp,
		skillsExp:  skillsExp,
		abilityExp: abilityExp,
	}
}

func (sm *SkillsManager) Init(skills map[enum.SkillName]ISkill) {
	if len(sm.skills) > 0 {
		fmt.Println("skills already initialized")
		return
	}
	sm.skills = skills
}

func (sm *SkillsManager) Get(name enum.SkillName) (ISkill, error) {
	if skill, ok := sm.skills[name]; ok {
		return skill, nil
	}
	return nil, errors.New("skill not found")
}

func (sm *SkillsManager) GetLvlOf(name enum.SkillName) (int, error) {
	skill, err := sm.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetLvl(), nil
}

func (sm *SkillsManager) GetValueForTestOf(name enum.SkillName) (int, error) {
	skill, err := sm.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetValueForTest(), nil
}

func (sm *SkillsManager) IncreaseExp(exp int, name enum.SkillName) (int, error) {
	skill, err := sm.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.IncreaseExp(exp), nil
}

func (sm *SkillsManager) CascadeUpgrade(exp int) {
	sm.exp.IncreasePoints(exp)
	sm.skillsExp.CascadeUpgrade(exp)
	sm.abilityExp.CascadeUpgrade(exp)
}

func (sm *SkillsManager) TriggerEndUpgrade(exp int) {
	sm.exp.IncreasePoints(exp)
}
