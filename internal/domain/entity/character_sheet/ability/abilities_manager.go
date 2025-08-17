package ability

import (
	exp "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	enum "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type Manager struct {
	characterExp *exp.CharacterExp
	abilities    map[enum.AbilityName]IAbility
	talent       Talent
}

func NewAbilitiesManager(
	characterExp *exp.CharacterExp,
	abilities map[enum.AbilityName]IAbility,
	talent Talent,
) *Manager {
	return &Manager{
		characterExp: characterExp,
		abilities:    abilities,
		talent:       talent,
	}
}

func (m *Manager) InitTalentWithLvl(lvl int) {
	m.talent.InitWithLvl(lvl)
}

func (m *Manager) IncreaseTalentExp(exp int) {
	m.talent.IncreaseExp(exp)
}

func (m *Manager) GetCharacterPoints() int {
	return m.characterExp.GetCharacterPoints()
}

func (m *Manager) Get(name enum.AbilityName) (IAbility, error) {
	ability, ok := m.abilities[name]
	if !ok {
		return nil, ErrAbilityNotFound
	}
	return ability, nil
}

func (m *Manager) GetExpReferenceOf(
	name enum.AbilityName,
) (exp.ICascadeUpgrade, error) {

	ability, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	return ability.GetExpReference(), nil
}

func (m *Manager) GetNextLvlAggregateExpOf(name enum.AbilityName) (int, error) {
	ability, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return ability.GetNextLvlAggregateExp(), nil
}

func (m *Manager) GetNextLvlBaseExpOf(name enum.AbilityName) (int, error) {
	ability, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return ability.GetNextLvlBaseExp(), nil
}

func (m *Manager) GetCurrentExpOf(name enum.AbilityName) (int, error) {
	ability, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return ability.GetCurrentExp(), nil
}

func (m *Manager) GetExpPointsOf(name enum.AbilityName) (int, error) {
	ability, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return ability.GetExpPoints(), nil
}

func (m *Manager) GetLevelOf(name enum.AbilityName) (int, error) {
	ability, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return ability.GetLevel(), nil
}

func (m *Manager) GetAbilitiesNextLvlAggregateExp() map[enum.AbilityName]int {
	expList := make(map[enum.AbilityName]int)
	for name, ability := range m.abilities {
		expList[name] = ability.GetNextLvlAggregateExp()
	}
	return expList
}

func (m *Manager) GetAbilitiesNextLvlBaseExp() map[enum.AbilityName]int {
	expList := make(map[enum.AbilityName]int)
	for name, ability := range m.abilities {
		expList[name] = ability.GetNextLvlBaseExp()
	}
	return expList
}

func (m *Manager) GetAbilitiesCurrentExp() map[enum.AbilityName]int {
	expList := make(map[enum.AbilityName]int)
	for name, ability := range m.abilities {
		expList[name] = ability.GetCurrentExp()
	}
	return expList
}

func (m *Manager) GetExpPoints() map[enum.AbilityName]int {
	expList := make(map[enum.AbilityName]int)
	for name, ability := range m.abilities {
		expList[name] = ability.GetExpPoints()
	}
	return expList
}

func (m *Manager) GetLevels() map[enum.AbilityName]int {
	lvlList := make(map[enum.AbilityName]int)
	for name, ability := range m.abilities {
		lvlList[name] = ability.GetLevel()
	}
	return lvlList
}

func (m *Manager) GetPhysicalsLevel() (int, error) {
	phys, err := m.Get(enum.Physicals)
	if err != nil {
		return 0, err
	}
	return phys.GetLevel(), nil
}

func (m *Manager) GetCharacterNextLvlAggregateExp() int {
	return m.characterExp.GetNextLvlAggregateExp()
}

func (m *Manager) GetCharacterNextLvlBaseExp() int {
	return m.characterExp.GetNextLvlBaseExp()
}

func (m *Manager) GetCharacterCurrentExp() int {
	return m.characterExp.GetCurrentExp()
}

func (m *Manager) GetCharacterExpPoints() int {
	return m.characterExp.GetExpPoints()
}

func (m *Manager) GetCharacterLevel() int {
	return m.characterExp.GetLevel()
}

func (m *Manager) GetTalentNextLvlAggregateExp() int {
	return m.talent.GetNextLvlAggregateExp()
}

func (m *Manager) GetTalentNextLvlBaseExp() int {
	return m.talent.GetNextLvlBaseExp()
}

func (m *Manager) GetTalentCurrentExp() int {
	return m.talent.GetCurrentExp()
}

func (m *Manager) GetTalentExpPoints() int {
	return m.talent.GetExpPoints()
}

func (m *Manager) GetTalentLevel() int {
	return m.talent.GetLevel()
}

func (m *Manager) GetAllAbilities() map[enum.AbilityName]IAbility {
	abilities := make(map[enum.AbilityName]IAbility)
	for name, ability := range m.abilities {
		abilities[name] = ability
	}
	return abilities
}
