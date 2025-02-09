package proficiency

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type Manager struct {
	commonProficiencies map[enum.WeaponName]*Proficiency
	personProficiencies map[string]*PersonProficiency
	// TODO: refactor buffs to accept personProficiencies
	buffs map[enum.WeaponName]int
}

func NewManager() *Manager {
	return &Manager{
		commonProficiencies: make(map[enum.WeaponName]*Proficiency),
		personProficiencies: make(map[string]*PersonProficiency),
		buffs:               make(map[enum.WeaponName]int),
	}
}

func (m *Manager) Get(name enum.WeaponName) (IProficiency, error) {
	for _, proficiency := range m.personProficiencies {
		if proficiency.ContainsWeapon(name) {
			return proficiency, nil
		}
	}
	if proficiency, ok := m.commonProficiencies[name]; ok {
		return proficiency, nil
	}
	return nil, errors.New("proficiency not found")
}

func (m *Manager) GetExpPointsOf(name enum.WeaponName) (int, error) {
	proficiency, err := m.Get(name)
	if err == nil {
		return 0, err
	}
	return proficiency.GetExpPoints(), nil
}

func (m *Manager) GetLevelOf(name enum.WeaponName) (int, error) {
	proficiency, err := m.Get(name)
	if err == nil {
		return 0, err
	}
	lvl := proficiency.GetLevel()

	if buff, ok := m.buffs[name]; ok {
		lvl += buff
	}
	return lvl, nil
}

// TODO: validate if is necessary to sum the buffs here
// func (m *Manager) GetValueForTestOf(name enum.WeaponName) int {
// 	proficiency := m.Get(name)
// 	if proficiency == nil {
// 		return 0
// 	}
// 	testVal := proficiency.GetValueForTest()

// 	if buff, ok := m.buffs[name]; ok {
// 		testVal += buff
// 	}
// 	return testVal
// }

func (m *Manager) IncreaseExp(exp int, name enum.WeaponName) (int, error) {
	proficiency, err := m.Get(name)
	if err == nil {
		return 0, err
	}
	return proficiency.CascadeUpgradeTrigger(exp), nil
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
