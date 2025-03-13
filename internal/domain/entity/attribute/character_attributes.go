package attribute

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

type CharacterAttributes struct {
	physicals  *Manager
	mentals    *Manager
	spirituals *Manager
}

func NewCharacterAttributes(physicals, mentals, spirituals *Manager) *CharacterAttributes {
	return &CharacterAttributes{
		physicals:  physicals,
		mentals:    mentals,
		spirituals: spirituals,
	}
}

// TODO: resolve this
func (ca *CharacterAttributes) IncreaseExpForMentals(
	values *experience.UpgradeCascade,
	name enum.AttributeName,
) error {
	attr, err := ca.mentals.Get(name)
	if err != nil {
		return err
	}
	// TODO: resolve CascadeUpgrade return quickly
	attr.CascadeUpgrade(values)
	// TODO: after this, return diff here
	return nil
}

func (ca *CharacterAttributes) Get(name enum.AttributeName) (IGameAttribute, error) {
	if ca.spirituals != nil {
		if attr, _ := ca.spirituals.Get(name); attr != nil {
			return attr, nil
		}
	}
	if attr, _ := ca.physicals.Get(name); attr != nil {
		return attr, nil
	}
	if attr, _ := ca.mentals.Get(name); attr != nil {
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

func (ca *CharacterAttributes) GetNextLvlAggregateExpOf(name enum.AttributeName) (int, error) {
	attr, err := ca.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetNextLvlAggregateExp(), nil
}

func (ca *CharacterAttributes) GetNextLvlBaseExpOf(name enum.AttributeName) (int, error) {
	attr, err := ca.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetNextLvlBaseExp(), nil
}

func (ca *CharacterAttributes) GetCurrentExpOf(name enum.AttributeName) (int, error) {
	attr, err := ca.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetCurrentExp(), nil
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
// 	attr, err := ca.physicals.GetPrimary(name)
// 	if err != nil {
// 		attr, err = ca.mentals.GetPrimary(name)
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

func (ca *CharacterAttributes) GetPhysicalsNextLvlAggregateExp() map[enum.AttributeName]int {
	return ca.physicals.GetAttributesNextLvlAggregateExp()
}

func (ca *CharacterAttributes) GetMentalsNextLvlAggregateExp() map[enum.AttributeName]int {
	return ca.mentals.GetAttributesNextLvlAggregateExp()
}

func (ca *CharacterAttributes) GetSpiritualsNextLvlAggregateExp() map[enum.AttributeName]int {
	return ca.spirituals.GetAttributesNextLvlAggregateExp()
}

func (ca *CharacterAttributes) GetPhysicalsNextLvlBaseExp() map[enum.AttributeName]int {
	return ca.physicals.GetAttributesNextLvlBaseExp()
}

func (ca *CharacterAttributes) GetMentalsNextLvlBaseExp() map[enum.AttributeName]int {
	return ca.mentals.GetAttributesNextLvlBaseExp()
}

func (ca *CharacterAttributes) GetSpiritualsNextLvlBaseExp() map[enum.AttributeName]int {
	return ca.spirituals.GetAttributesNextLvlBaseExp()
}

func (ca *CharacterAttributes) GetPhysicalsCurrentExp() map[enum.AttributeName]int {
	return ca.physicals.GetAttributesCurrentExp()
}

func (ca *CharacterAttributes) GetMentalsCurrentExp() map[enum.AttributeName]int {
	return ca.mentals.GetAttributesCurrentExp()
}

func (ca *CharacterAttributes) GetSpiritualsCurrentExp() map[enum.AttributeName]int {
	return ca.spirituals.GetAttributesCurrentExp()
}

func (ca *CharacterAttributes) GetPhysicalsExpPoints() map[enum.AttributeName]int {
	return ca.physicals.GetAttributesExpPoints()
}

func (ca *CharacterAttributes) GetMentalsExpPoints() map[enum.AttributeName]int {
	return ca.mentals.GetAttributesExpPoints()
}

func (ca *CharacterAttributes) GetSpiritualsExpPoints() map[enum.AttributeName]int {
	return ca.spirituals.GetAttributesExpPoints()
}

func (ca *CharacterAttributes) GetPhysicalsLevel() map[enum.AttributeName]int {
	return ca.physicals.GetAttributesLevel()
}

func (ca *CharacterAttributes) GetMentalsLevel() map[enum.AttributeName]int {
	return ca.mentals.GetAttributesLevel()
}

func (ca *CharacterAttributes) GetSpiritualsLevel() map[enum.AttributeName]int {
	return ca.spirituals.GetAttributesLevel()
}

func (ca *CharacterAttributes) GetPhysicalAttributes() map[enum.AttributeName]IGameAttribute {
	return ca.physicals.GetAllAttributes()
}

func (ca *CharacterAttributes) GetMentalAttributes() map[enum.AttributeName]IGameAttribute {
	return ca.mentals.GetAllAttributes()
}

func (ca *CharacterAttributes) GetSpiritualAttributes() map[enum.AttributeName]IGameAttribute {
	return ca.spirituals.GetAllAttributes()
}
