package attribute

import (
	"errors"

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
	return nil, errors.New("attribute not found")
}

func (m *Manager) GetPrimary(name enum.AttributeName) (PrimaryAttribute, error) {
	primaryAttribute, ok := m.primaryAttributes[name]
	if ok {
		return *primaryAttribute, nil
	}
	return PrimaryAttribute{}, errors.New("primary attribute not found")
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
