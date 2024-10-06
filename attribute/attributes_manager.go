package attribute

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
)

type AttributeManager struct {
	primaryAttributes map[enum.AttributeName]PrimaryAttribute
	middleAttributes  map[enum.AttributeName]MiddleAttribute
}

// NewAttributeManager creates a new instance of AttributeManager.
func NewAttributeManager(
	primAttr map[enum.AttributeName]PrimaryAttribute,
	midAttr map[enum.AttributeName]MiddleAttribute) *AttributeManager {
	return &AttributeManager{
		primaryAttributes: primAttr,
		middleAttributes:  midAttr,
	}
}

func (am *AttributeManager) Get(name enum.AttributeName) (IGameAttribute, error) {
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

func (am *AttributeManager) GetPrimary(name enum.AttributeName) (PrimaryAttribute, error) {
	primaryAttribute, ok := am.primaryAttributes[name]
	if ok {
		return primaryAttribute, nil
	}
	return PrimaryAttribute{}, errors.New("primary attribute not found")
}