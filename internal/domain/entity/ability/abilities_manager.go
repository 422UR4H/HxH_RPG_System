package ability

import (
	"errors"

	enum "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	exp "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
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

func (m *Manager) Get(name enum.AbilityName) (IAbility, error) {
	ability, ok := m.abilities[name]
	if !ok {
		return nil, errors.New("ability not found")
	}
	return ability, nil
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

func (m *Manager) GetTalentLevel() int {
	return m.talent.GetLevel()
}

func (m *Manager) GetCharacterExp() int {
	return m.characterExp.GetExpPoints()
}
