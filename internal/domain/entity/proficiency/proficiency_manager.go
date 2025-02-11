package proficiency

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// unlike skills, there are only hard coded proficiencies
type Manager struct {
	personProficiencies map[string]*PersonProficiency
	commonProficiencies map[enum.WeaponName]*Proficiency
	buffs               map[enum.WeaponName]int
}

func NewManager() *Manager {
	return &Manager{
		personProficiencies: make(map[string]*PersonProficiency),
		commonProficiencies: make(map[enum.WeaponName]*Proficiency),
		buffs:               make(map[enum.WeaponName]int),
	}
}

func (m *Manager) Get(name enum.WeaponName) (IProficiency, error) {
	for _, prof := range m.personProficiencies {
		if prof.ContainsWeapon(name) {
			return prof, nil
		}
	}
	if prof, ok := m.commonProficiencies[name]; ok {
		return prof, nil
	}
	return nil, errors.New("proficiency not found")
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

func (m *Manager) IncreaseExp(exp int, name enum.WeaponName) (int, error) {
	prof, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return prof.CascadeUpgradeTrigger(exp), nil
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
