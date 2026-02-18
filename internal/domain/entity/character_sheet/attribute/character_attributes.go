package attribute

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterAttributes struct {
	physicals  *Manager
	mentals    *Manager
	spirituals *SpiritualManager
}

func NewCharacterAttributes(
	physicals *Manager,
	mentals *Manager,
	spirituals *SpiritualManager,
) *CharacterAttributes {
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
	attr.CascadeUpgrade(values)
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
	return nil, ErrAttributeNotFound
}

func (ca *CharacterAttributes) GetDistributable(name enum.AttributeName) (IDistributableAttribute, error) {
	if attr, _ := ca.physicals.Get(name); attr != nil {
		return attr, nil
	}
	if attr, _ := ca.mentals.Get(name); attr != nil {
		return attr, nil
	}
	return nil, ErrAttributeNotFound
}

func (ca *CharacterAttributes) GetPointsOf(name enum.AttributeName) (int, error) {
	attr, err := ca.GetDistributable(name)
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

func (ca *CharacterAttributes) IncreasePrimaryPhysicalPts(
	name enum.AttributeName, points int,
) (map[enum.AttributeName]int, error) {
	return ca.physicals.IncreasePointsForPrimary(name, points)
}

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

func (ca *CharacterAttributes) GetPhysicalsPrimaryPoints() map[enum.AttributeName]int {
	return ca.physicals.GetDistributedPrimaryPoints()
}

func (ca *CharacterAttributes) GetPhysicalAttributes() map[enum.AttributeName]IDistributableAttribute {
	return ca.physicals.GetAllAttributes()
}

func (ca *CharacterAttributes) GetMentalAttributes() map[enum.AttributeName]IDistributableAttribute {
	return ca.mentals.GetAllAttributes()
}

func (ca *CharacterAttributes) GetSpiritualAttributes() map[enum.AttributeName]IGameAttribute {
	return ca.spirituals.GetAllAttributes()
}
