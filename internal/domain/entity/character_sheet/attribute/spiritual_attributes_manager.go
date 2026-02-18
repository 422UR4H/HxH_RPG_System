package attribute

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type SpiritualManager struct {
	attributes map[enum.AttributeName]*SpiritualAttribute
	buffs      map[enum.AttributeName]*int
}

func NewSpiritualAttributeManager(
	attr map[enum.AttributeName]*SpiritualAttribute,
	buffs map[enum.AttributeName]*int,
) *SpiritualManager {
	return &SpiritualManager{
		attributes: attr,
		buffs:      buffs,
	}
}

func (m *SpiritualManager) Get(name enum.AttributeName) (IGameAttribute, error) {
	primaryAttribute, ok := m.attributes[name]
	if ok {
		return primaryAttribute, nil
	}
	return nil, ErrAttributeNotFound
}

func (m *SpiritualManager) GetNextLvlAggregateExpOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetNextLvlAggregateExp(), nil
}

func (m *SpiritualManager) GetNextLvlBaseExpOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetNextLvlBaseExp(), nil
}

func (m *SpiritualManager) GetCurrentExpOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetCurrentExp(), nil
}

func (m *SpiritualManager) GetExpPointsOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetExpPoints(), nil
}

func (m *SpiritualManager) GetLevelOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetLevel(), nil
}

func (m *SpiritualManager) GetAttributesNextLvlAggregateExp() map[enum.AttributeName]int {
	expList := make(map[enum.AttributeName]int)
	for name, attr := range m.attributes {
		expList[name] = attr.GetNextLvlAggregateExp()
	}
	return expList
}

func (m *SpiritualManager) GetAttributesNextLvlBaseExp() map[enum.AttributeName]int {
	expList := make(map[enum.AttributeName]int)
	for name, attr := range m.attributes {
		expList[name] = attr.GetNextLvlBaseExp()
	}
	return expList
}

func (m *SpiritualManager) GetAttributesCurrentExp() map[enum.AttributeName]int {
	expList := make(map[enum.AttributeName]int)
	for name, attr := range m.attributes {
		expList[name] = attr.GetCurrentExp()
	}
	return expList
}

func (m *SpiritualManager) GetAttributesExpPoints() map[enum.AttributeName]int {
	expList := make(map[enum.AttributeName]int)
	for name, attr := range m.attributes {
		expList[name] = attr.GetExpPoints()
	}
	return expList
}

func (m *SpiritualManager) GetAttributesLevel() map[enum.AttributeName]int {
	lvlList := make(map[enum.AttributeName]int)
	for name, attr := range m.attributes {
		lvlList[name] = attr.GetLevel()
	}
	return lvlList
}

func (m *SpiritualManager) GetBuffs() map[enum.AttributeName]*int {
	return m.buffs
}

func (m *SpiritualManager) GetBuff(name enum.AttributeName) *int {
	return m.buffs[name]
}

func (m *SpiritualManager) SetBuff(
	name enum.AttributeName, buff int,
) (map[enum.AttributeName]*int, error) {

	_, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	*m.buffs[name] = buff
	return m.buffs, nil
}

func (m *SpiritualManager) RemoveBuff(
	name enum.AttributeName,
) (map[enum.AttributeName]*int, error) {

	_, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	*m.buffs[name] = 0
	return m.buffs, nil
}

func (m *SpiritualManager) GetAllAttributes() map[enum.AttributeName]IGameAttribute {
	attributes := make(map[enum.AttributeName]IGameAttribute)
	for name, attr := range m.attributes {
		attributes[name] = attr
	}
	return attributes
}
