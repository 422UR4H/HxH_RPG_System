package attribute

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type Manager struct {
	primaryAttributes map[enum.AttributeName]*PrimaryAttribute
	middleAttributes  map[enum.AttributeName]*MiddleAttribute
	buffs             map[enum.AttributeName]*int
}

func NewAttributeManager(
	primAttr map[enum.AttributeName]*PrimaryAttribute,
	midAttr map[enum.AttributeName]*MiddleAttribute,
	buffs map[enum.AttributeName]*int,
) *Manager {
	return &Manager{
		primaryAttributes: primAttr,
		middleAttributes:  midAttr,
		buffs:             buffs,
	}
}

func (m *Manager) Get(name enum.AttributeName) (IGameAttribute, error) {
	primaryAttribute, ok := m.primaryAttributes[name]
	if ok {
		return primaryAttribute, nil
	}

	middleAttribute, ok := m.middleAttributes[name]
	if ok {
		return middleAttribute, nil
	}
	return nil, ErrAttributeNotFound
}

func (m *Manager) GetPrimary(name enum.AttributeName) (PrimaryAttribute, error) {
	primaryAttribute, ok := m.primaryAttributes[name]
	if ok {
		return *primaryAttribute, nil
	}
	return PrimaryAttribute{}, ErrPrimaryAttributeNotFound
}

func (m *Manager) GetNextLvlAggregateExpOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetNextLvlAggregateExp(), nil
}

func (m *Manager) GetNextLvlBaseExpOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetNextLvlBaseExp(), nil
}

func (m *Manager) GetCurrentExpOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetCurrentExp(), nil
}

func (m *Manager) GetExpPointsOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetExpPoints(), nil
}

func (m *Manager) GetLevelOf(name enum.AttributeName) (int, error) {
	attr, err := m.Get(name)
	if err != nil {
		return 0, err
	}
	return attr.GetLevel(), nil
}

func (m *Manager) GetAttributesNextLvlAggregateExp() map[enum.AttributeName]int {
	expList := make(map[enum.AttributeName]int)
	for name, attr := range m.primaryAttributes {
		expList[name] = attr.GetNextLvlAggregateExp()
	}
	for name, attr := range m.middleAttributes {
		expList[name] = attr.GetNextLvlAggregateExp()
	}
	return expList
}

func (m *Manager) GetAttributesNextLvlBaseExp() map[enum.AttributeName]int {
	expList := make(map[enum.AttributeName]int)
	for name, attr := range m.primaryAttributes {
		expList[name] = attr.GetNextLvlBaseExp()
	}
	for name, attr := range m.middleAttributes {
		expList[name] = attr.GetNextLvlBaseExp()
	}
	return expList
}

func (m *Manager) GetAttributesCurrentExp() map[enum.AttributeName]int {
	expList := make(map[enum.AttributeName]int)
	for name, attr := range m.primaryAttributes {
		expList[name] = attr.GetCurrentExp()
	}
	for name, attr := range m.middleAttributes {
		expList[name] = attr.GetCurrentExp()
	}
	return expList
}

func (m *Manager) GetAttributesExpPoints() map[enum.AttributeName]int {
	expList := make(map[enum.AttributeName]int)
	for name, attr := range m.primaryAttributes {
		expList[name] = attr.GetExpPoints()
	}
	for name, attr := range m.middleAttributes {
		expList[name] = attr.GetExpPoints()
	}
	return expList
}

func (m *Manager) GetAttributesLevel() map[enum.AttributeName]int {
	lvlList := make(map[enum.AttributeName]int)
	for name, attr := range m.primaryAttributes {
		lvlList[name] = attr.GetLevel()
	}
	for name, attr := range m.middleAttributes {
		lvlList[name] = attr.GetLevel()
	}
	return lvlList
}

func (m *Manager) GetBuffs() map[enum.AttributeName]*int {
	return m.buffs
}

func (m *Manager) GetBuff(name enum.AttributeName) *int {
	return m.buffs[name]
}

func (m *Manager) SetBuff(
	name enum.AttributeName, buff int,
) (map[enum.AttributeName]*int, error) {

	_, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	*m.buffs[name] = buff
	return m.buffs, nil
}

func (m *Manager) RemoveBuff(
	name enum.AttributeName,
) (map[enum.AttributeName]*int, error) {

	_, err := m.Get(name)
	if err != nil {
		return nil, err
	}
	*m.buffs[name] = 0
	return m.buffs, nil
}

func (m *Manager) GetAllAttributes() map[enum.AttributeName]IGameAttribute {
	attributes := make(map[enum.AttributeName]IGameAttribute)
	for name, attr := range m.primaryAttributes {
		attributes[name] = attr
	}
	for name, attr := range m.middleAttributes {
		attributes[name] = attr
	}
	return attributes
}
