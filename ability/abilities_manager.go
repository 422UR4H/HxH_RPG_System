package ability

import (
	"errors"

	enum "github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
	exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type Manager struct {
	characterExp exp.CharacterExp
	abilities    map[enum.AbilityName]Ability
	talent       Talent
}

func (am *Manager) NewAbilitiesManager(
	characterExp exp.CharacterExp,
	abilities map[enum.AbilityName]Ability,
	talent Talent,
) *Manager {
	return &Manager{
		characterExp: characterExp,
		abilities:    abilities,
		talent:       talent,
	}
}

func (am *Manager) Get(name enum.AbilityName) (Ability, error) {
	ability, ok := am.abilities[name]
	if !ok {
		return Ability{}, errors.New("ability not found")
	}
	return ability, nil
}

func (am *Manager) GetExpPointsOf(name enum.AbilityName) (int, error) {
	ability, err := am.Get(name)
	if err != nil {
		return 0, err
	}
	return ability.GetExpPoints(), nil
}

func (am *Manager) GetLevelOf(name enum.AbilityName) (int, error) {
	ability, err := am.Get(name)
	if err != nil {
		return 0, err
	}
	return ability.GetLevel(), nil
}
