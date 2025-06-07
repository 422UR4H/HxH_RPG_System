package proficiency

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// unlike skills, there are only hard coded proficiencies
type Manager struct {
	jointProficiencies  map[string]*JointProficiency
	commonProficiencies map[enum.WeaponName]*Proficiency
	buffs               map[enum.WeaponName]int
}

func NewManager() *Manager {
	return &Manager{
		jointProficiencies:  make(map[string]*JointProficiency),
		commonProficiencies: make(map[enum.WeaponName]*Proficiency),
		buffs:               make(map[enum.WeaponName]int),
	}
}

func (m *Manager) Get(name enum.WeaponName) (IProficiency, error) {
	for _, prof := range m.jointProficiencies {
		if prof.ContainsWeapon(name) {
			return prof, nil
		}
	}
	if prof, ok := m.commonProficiencies[name]; ok {
		return prof, nil
	}
	return nil, ErrProficiencyNotFound
}

func (m *Manager) GetJoint(name string) (IProficiency, error) {
	if prof, ok := m.jointProficiencies[name]; ok {
		return prof, nil
	}
	return nil, ErrProficiencyNotFound
}

func (m *Manager) IncreaseExp(
	values *experience.UpgradeCascade, name enum.WeaponName,
) error {
	prof, err := m.Get(name)
	if err != nil {
		return err
	}
	prof.CascadeUpgradeTrigger(values)
	return nil
}

func (m *Manager) IncreaseExpForJoint(
	values *experience.UpgradeCascade, name string,
) error {
	prof, err := m.GetJoint(name)
	if err != nil {
		return err
	}
	prof.CascadeUpgradeTrigger(values)
	return nil
}

func (m *Manager) AddJoint(
	proficiency *JointProficiency,
	physSkillsExp experience.ICascadeUpgrade,
	abilitySkillsExp experience.ICascadeUpgrade,
) error {
	name := proficiency.GetName()

	if _, ok := m.jointProficiencies[name]; ok {
		return ErrProficiencyAlreadyExists
	}
	if err := proficiency.Init(physSkillsExp, abilitySkillsExp); err != nil {
		return err
	}
	m.jointProficiencies[name] = proficiency
	return nil
}

func (m *Manager) AddCommon(
	name enum.WeaponName, proficiency *Proficiency,
) error {
	if _, ok := m.commonProficiencies[name]; ok {
		return ErrProficiencyAlreadyExists
	}
	m.commonProficiencies[name] = proficiency
	return nil
}

func (m *Manager) GetJointProficiencies() map[string]JointProficiency {
	lvlList := make(map[string]JointProficiency)
	for name, prof := range m.jointProficiencies {
		lvlList[name] = *prof
	}
	return lvlList
}

func (m *Manager) GetWeapons() []enum.WeaponName {
	var weapons []enum.WeaponName
	for name := range m.commonProficiencies {
		weapons = append(weapons, name)
	}
	return weapons
}

func (m *Manager) GetValueForTestOf(name enum.WeaponName) (int, error) {
	prof, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	// TODO: validate this
	// testVal := prof.GetValueForTest()
	testVal := prof.GetLevel()

	if buff, ok := m.buffs[name]; ok {
		testVal += buff
	}
	return testVal, nil
}

func (m *Manager) GetNextLvlAggregateExpOf(name enum.WeaponName) (int, error) {
	prof, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return prof.GetNextLvlAggregateExp(), nil
}

func (m *Manager) GetNextLvlBaseExpOf(name enum.WeaponName) (int, error) {
	prof, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return prof.GetNextLvlBaseExp(), nil
}

func (m *Manager) GetCurrentExpOf(name enum.WeaponName) (int, error) {
	prof, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return prof.GetCurrentExp(), nil
}

func (m *Manager) GetExpPointsOf(name enum.WeaponName) (int, error) {
	prof, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return prof.GetExpPoints(), nil
}

func (m *Manager) GetLevelOf(name enum.WeaponName) (int, error) {
	prof, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return prof.GetLevel(), nil
}

func (m *Manager) GetCommonsNextLvlAggregateExp() map[enum.WeaponName]int {
	expList := make(map[enum.WeaponName]int)
	for name, prof := range m.commonProficiencies {
		expList[name] = prof.GetNextLvlAggregateExp()
	}
	return expList
}

func (m *Manager) GetCommonsNextLvlBaseExp() map[enum.WeaponName]int {
	expList := make(map[enum.WeaponName]int)
	for name, prof := range m.commonProficiencies {
		expList[name] = prof.GetNextLvlBaseExp()
	}
	return expList
}

func (m *Manager) GetCommonsCurrentExp() map[enum.WeaponName]int {
	expList := make(map[enum.WeaponName]int)
	for name, prof := range m.commonProficiencies {
		expList[name] = prof.GetCurrentExp()
	}
	return expList
}

func (m *Manager) GetCommonsExpPoints() map[enum.WeaponName]int {
	expList := make(map[enum.WeaponName]int)
	for name, prof := range m.commonProficiencies {
		expList[name] = prof.GetExpPoints()
	}
	return expList

}

func (m *Manager) GetCommonsLevel() map[enum.WeaponName]int {
	expList := make(map[enum.WeaponName]int)
	for name, prof := range m.commonProficiencies {
		expList[name] = prof.GetLevel()
	}
	return expList
}

func (m *Manager) SetBuff(name enum.WeaponName, value int) int { //, int) {
	lvl, err := m.GetLevelOf(name)
	if err != nil {
		return 0 //, 0
	}
	m.buffs[name] = value
	// testVal := m.GetValueForTestOf(name)

	return lvl + value //, testVal
}

func (m *Manager) DeleteBuff(name enum.WeaponName) {
	delete(m.buffs, name)
}

func (m *Manager) GetBuffs() map[enum.WeaponName]int {
	return m.buffs
}

func (m *Manager) GetCommons() map[enum.WeaponName]IProficiency {
	proficiencies := make(map[enum.WeaponName]IProficiency)
	for name, prof := range m.commonProficiencies {
		proficiencies[name] = prof
	}
	return proficiencies
}
