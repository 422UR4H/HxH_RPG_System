package skill

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
	"github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type Manager struct {
	skills     map[enum.SkillName]ISkill
	exp        experience.Exp
	skillsExp  experience.ICascadeUpgrade
	abilityExp experience.ICascadeUpgrade
}

func NewSkillsManager(
	exp experience.Exp,
	skillsExp experience.ICascadeUpgrade,
	abilityExp experience.ICascadeUpgrade) *Manager {

	return &Manager{
		skills:     make(map[enum.SkillName]ISkill),
		exp:        exp,
		skillsExp:  skillsExp,
		abilityExp: abilityExp,
	}
}

func (m *Manager) Init(skills map[enum.SkillName]ISkill) {
	if len(m.skills) > 0 {
		fmt.Println("skills already initialized")
		return
	}
	m.skills = skills
}

func (m *Manager) Get(name enum.SkillName) (ISkill, error) {
	if skill, ok := m.skills[name]; ok {
		return skill, nil
	}
	return nil, errors.New("skill not found")
}

func (m *Manager) GetLevelOf(name enum.SkillName) (int, error) {
	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetLevel(), nil
}

func (m *Manager) GetValueForTestOf(name enum.SkillName) (int, error) {
	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetValueForTest(), nil
}

func (m *Manager) IncreaseExp(exp int, name enum.SkillName) (int, error) {
	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.CascadeUpgradeTrigger(exp), nil
}

func (m *Manager) CascadeUpgrade(exp int) {
	m.exp.IncreasePoints(exp)
	m.skillsExp.CascadeUpgrade(exp)
	m.abilityExp.CascadeUpgrade(exp)
}

func (m *Manager) EndCascadeUpgrade(exp int) {
	m.exp.IncreasePoints(exp)
}
