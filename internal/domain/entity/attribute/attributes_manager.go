package attribute

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type Manager struct {
	primaryAttributes map[enum.AttributeName]*PrimaryAttribute
	middleAttributes  map[enum.AttributeName]*MiddleAttribute
}

func NewAttributeManager(
	primAttr map[enum.AttributeName]*PrimaryAttribute,
	midAttr map[enum.AttributeName]*MiddleAttribute) *Manager {
	return &Manager{
		primaryAttributes: primAttr,
		middleAttributes:  midAttr,
	}
}

func (am *Manager) Get(name enum.AttributeName) (IGameAttribute, error) {
	primaryAttribute, ok := am.primaryAttributes[name]
	if ok {
		return primaryAttribute, nil
	}

	middleAttribute, ok := am.middleAttributes[name]
	if ok {
		return middleAttribute, nil
	}
	return nil, errors.New("attribute not found")
}

func (am *Manager) GetPrimary(name enum.AttributeName) (PrimaryAttribute, error) {
	primaryAttribute, ok := am.primaryAttributes[name]
	if ok {
		return *primaryAttribute, nil
	}
	return PrimaryAttribute{}, errors.New("primary attribute not found")
}

func (am *Manager) GetExpPointsOf(name enum.AttributeName) (int, error) {
	attr, err := am.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetExpPoints(), nil
}

func (am *Manager) GetLevelOf(name enum.AttributeName) (int, error) {
	attr, err := am.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetLevel(), nil
}
