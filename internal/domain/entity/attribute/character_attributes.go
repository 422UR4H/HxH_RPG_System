package attribute

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterAttributes struct {
	physicalAttributes  *Manager
	mentalAttributes    *Manager
	spiritualAttributes *Manager
}

func NewCharacterAttributes(physical, mental, spiritual *Manager) *CharacterAttributes {
	return &CharacterAttributes{
		physicalAttributes:  physical,
		mentalAttributes:    mental,
		spiritualAttributes: spiritual,
	}
}

func (ca *CharacterAttributes) Get(name enum.AttributeName) (IGameAttribute, error) {
	if ca.spiritualAttributes != nil {
		if attr, _ := ca.spiritualAttributes.Get(name); attr != nil {
			return attr, nil
		}
	}
	if attr, _ := ca.physicalAttributes.Get(name); attr != nil {
		return attr, nil
	}
	if attr, _ := ca.mentalAttributes.Get(name); attr != nil {
		return attr, nil
	}
	return nil, errors.New("attribute not found")
}

func (ca *CharacterAttributes) GetPointsOf(name enum.AttributeName) (int, error) {
	attr, err := ca.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetPoints(), nil
}

func (ca *CharacterAttributes) GetExpPointsOf(name enum.AttributeName) (int, error) {
	attr, err := ca.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetExpPoints(), nil
}

func (ca *CharacterAttributes) GetLevelOf(name enum.AttributeName) (int, error) {
	attr, err := ca.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetLevel(), nil
}

// TODO: verify if this is necessary
// func (ca *CharacterAttributes) SetPoints(points int, name enum.AttributeName) error {
// 	attr, err := ca.physicalAttributes.GetPrimary(name)
// 	if err != nil {
// 		attr, err = ca.mentalAttributes.GetPrimary(name)
// 	}
// 	if err != nil {
// 		return err
// 	}
// 	attr.SetPoints(points)
// 	return nil
// }

func (ca *CharacterAttributes) GetPowerOf(name enum.AttributeName) (int, error) {
	attr, err := ca.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetPower(), nil
}
