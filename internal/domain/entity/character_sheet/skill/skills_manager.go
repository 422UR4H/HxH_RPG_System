package skill

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type Manager struct {
	jointSkills map[string]*JointSkill
	skills      map[enum.SkillName]ISkill
	buffs       map[enum.SkillName]int
	exp         experience.Exp
	skillsExp   experience.ICascadeUpgrade
}

func NewSkillsManager(
	exp experience.Exp,
	skillsExp experience.ICascadeUpgrade) *Manager {

	return &Manager{
		jointSkills: make(map[string]*JointSkill),
		skills:      make(map[enum.SkillName]ISkill),
		buffs:       make(map[enum.SkillName]int),
		exp:         exp,
		skillsExp:   skillsExp,
	}
}

func (m *Manager) Init(skills map[enum.SkillName]ISkill) error {
	if len(m.skills) > 0 {
		return ErrSkillsAlreadyInitialized
	}
	m.skills = skills
	return nil
}

func (m *Manager) CascadeUpgrade(values *experience.UpgradeCascade) {
	m.exp.IncreasePoints(values.GetExp())
	m.skillsExp.CascadeUpgrade(values)
}

func (m *Manager) IncreaseExp(
	values *experience.UpgradeCascade,
	name enum.SkillName,
) error {
	skill, err := m.Get(name)
	if err != nil {
		return err
	}
	skill.CascadeUpgradeTrigger(values)
	return nil
}

func (m *Manager) Get(name enum.SkillName) (ISkill, error) {
	// TODO: maybe do not get jointSkills here
	for _, jointSk := range m.jointSkills {
		if jointSk.Contains(name) {
			return jointSk, nil
		}
	}
	// TODO: study if should return sum of both joint and common skills
	if skill, ok := m.skills[name]; ok {
		return skill, nil
	}
	return nil, ErrSkillNotFound
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

func (m *Manager) GetNextLvlAggregateExpOf(name enum.SkillName) (int, error) {
	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetNextLvlAggregateExp(), nil
}

func (m *Manager) GetNextLvlBaseExpOf(name enum.SkillName) (int, error) {
	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetNextLvlBaseExp(), nil
}

func (m *Manager) GetCurrentExpOf(name enum.SkillName) (int, error) {
	skill, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetCurrentExp(), nil
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
	return skill.GetLevel(), nil
}

func (m *Manager) GetLevel() int {
	return m.exp.GetLevel()
}

func (m *Manager) GetSkillsNextLvlAggregateExp() map[enum.SkillName]int {
	expList := make(map[enum.SkillName]int)
	for name, skill := range m.skills {
		expList[name] = skill.GetNextLvlAggregateExp()
	}
	return expList
}

func (m *Manager) GetSkillsNextLvlBaseExp() map[enum.SkillName]int {
	expList := make(map[enum.SkillName]int)
	for name, skill := range m.skills {
		expList[name] = skill.GetNextLvlBaseExp()
	}
	return expList
}

func (m *Manager) GetSkillsCurrentExp() map[enum.SkillName]int {
	expList := make(map[enum.SkillName]int)
	for name, skill := range m.skills {
		expList[name] = skill.GetCurrentExp()
	}
	return expList
}

func (m *Manager) GetSkillsExpPoints() map[enum.SkillName]int {
	expList := make(map[enum.SkillName]int)
	for name, skill := range m.skills {
		expList[name] = skill.GetExpPoints()
	}
	return expList
}

func (m *Manager) GetSkillsLevel() map[enum.SkillName]int {
	lvlList := make(map[enum.SkillName]int)
	for name, skill := range m.skills {
		lvlList[name] = skill.GetLevel()
	}
	return lvlList
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

func (m *Manager) GetCommonSkills() map[enum.SkillName]ISkill {
	skills := make(map[enum.SkillName]ISkill)
	for name, skill := range m.skills {
		skills[name] = skill
	}
	return skills
}

func (m *Manager) AddJointSkill(js *JointSkill) error {
	if !js.IsInitialized() {
		return ErrJointSkillNotInitialized
	}
	name := js.GetName()
	if _, ok := m.jointSkills[name]; ok {
		return ErrJointSkillAlreadyExists
	}
	m.jointSkills[name] = js
	return nil
}

func (m *Manager) GetJointSkills() map[string]JointSkill {
	jointSkills := make(map[string]JointSkill)
	for name, value := range m.jointSkills {
		jointSkills[name] = *value
	}
	return jointSkills
}
