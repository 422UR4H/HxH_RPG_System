package skill

import (
	"errors"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type Manager struct {
	skills     map[enum.SkillName]ISkill
	buffs      map[enum.SkillName]int
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
		buffs:      make(map[enum.SkillName]int),
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

func (m *Manager) GetExpPointsOf(name enum.SkillName) (int, error) {
	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetExpPoints(), nil
}

func (m *Manager) GetLevelOf(name enum.SkillName) (int, error) {
	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	lvl := skill.GetLevel()

	if buff, ok := m.buffs[name]; ok {
		lvl += buff
	}
	return lvl, nil
}

func (m *Manager) GetValueForTestOf(name enum.SkillName) (int, error) {
	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	testVal := skill.GetValueForTest()

	if buff, ok := m.buffs[name]; ok {
		testVal += buff
	}
	return testVal, nil
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

func (m *Manager) GetAggregateExpByLvlOf(
	name enum.SkillName, lvl int,
) (int, error) {

	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetAggregateExpByLvl(lvl), nil
}

func (m *Manager) SetBuff(name enum.SkillName, value int) (int, int) {
	lvl, err := m.GetLevelOf(name)
	if err != nil {
		return 0, 0
	}
	m.buffs[name] = value
	testVal, _ := m.GetValueForTestOf(name)

	return lvl + value, testVal
}

func (m *Manager) DeleteBuff(name enum.SkillName) {
	delete(m.buffs, name)
}

func (m *Manager) GetBuffs() map[enum.SkillName]int {
	return m.buffs
}
