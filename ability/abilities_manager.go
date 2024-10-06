package ability

import (
	"errors"

	enum "github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
	exp "github.com/422UR4H/HxH_RPG_Environment.Domain/experience"
)

type AbilitiesManager struct {
	characterExp exp.CharacterExp
	abilities    map[enum.AbilityName]Ability
	talent       Talent
}

func (am *AbilitiesManager) NewAbilitiesManager(
	characterExp exp.CharacterExp,
	abilities map[enum.AbilityName]Ability,
	talent Talent,
) *AbilitiesManager {
	return &AbilitiesManager{
		characterExp: characterExp,
		abilities:    abilities,
		talent:       talent,
	}
}

func (am *AbilitiesManager) Get(name enum.AbilityName) (Ability, error) {
	ability, ok := am.abilities[name]
	if !ok {
		return Ability{}, errors.New("ability not found")
	}
	return ability, nil
}

func (am *AbilitiesManager) GetExpPointsOf(name enum.AbilityName) (int, error) {
	ability, err := am.Get(name)
	if err != nil {
		return 0, err
	}
	return ability.GetExpPoints(), nil
}

func (am *AbilitiesManager) GetLevelOf(name enum.AbilityName) (int, error) {
	ability, err := am.Get(name)
	if err != nil {
		return 0, err
	}
	return ability.GetLevel(), nil
}
