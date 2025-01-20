package spiritual

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type Manager struct {
	principles map[enum.PrincipleName]NenPrinciple
	nenHexagon *NenHexagon
	hatsu      *Hatsu
}

func NewPrinciplesManager(
	principles map[enum.PrincipleName]NenPrinciple,
	nenHexagon *NenHexagon,
	hatsu *Hatsu,
) *Manager {

	return &Manager{
		principles: principles,
		nenHexagon: nenHexagon,
		hatsu:      hatsu,
	}
}

func (m *Manager) InitNenHexagon(nenHexagon *NenHexagon) error {
	if nenHexagon != nil {
		return fmt.Errorf("nen hexagon already initialized")
	}
	m.nenHexagon = nenHexagon
	m.hatsu.SetCategoryPercents(nenHexagon.GetCategoryPercents())
	return nil
}

func (m *Manager) IncreaseExpByPrinciple(
	name enum.PrincipleName, exp int,
) (int, error) {

	if principle, ok := m.principles[name]; ok {
		return principle.CascadeUpgradeTrigger(exp), nil
	}
	return 0, fmt.Errorf("principle %s not found", name.String())
}

func (m *Manager) IncreaseExpByCategory(
	name enum.CategoryName, exp int,
) (int, error) {

	diff, err := m.hatsu.IncreaseExp(exp, name)
	if err != nil {
		return 0, err
	}
	return diff, nil
}

func (m *Manager) Get(name enum.PrincipleName) (IPrinciple, error) {
	if name == enum.Hatsu {
		return m.hatsu, nil
	}
	if principle, ok := m.principles[name]; ok {
		return &principle, nil
	}
	return nil, fmt.Errorf("principle %s not found", name.String())
}

func (m *Manager) GetExpPointsOfPrinciple(
	name enum.PrincipleName) (int, error) {

	principle, err := m.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get exp")
	}
	return principle.GetExpPoints(), nil
}

func (m *Manager) GetLevelOfPrinciple(
	name enum.PrincipleName) (int, error) {

	principle, err := m.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level")
	}
	return principle.GetLevel(), nil
}

func (m *Manager) GetExpPointsOfCategory(
	name enum.CategoryName) (int, error) {

	principle, err := m.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get exp")
	}
	return principle.GetExpPoints(), nil
}

func (m *Manager) GetLevelOfCategory(
	name enum.CategoryName) (int, error) {

	principle, err := m.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level")
	}
	return principle.GetLevel(), nil
}

// TODO: handle errors below, case nenHexagon is nil
func (m *Manager) IncreaseCurrHexValue() (
	map[enum.CategoryName]float64, enum.CategoryName) {

	percents, name := m.nenHexagon.IncreaseCurrHexValue()
	m.hatsu.SetCategoryPercents(percents)

	return percents, name
}

func (m *Manager) DecreaseCurrHexValue() (
	map[enum.CategoryName]float64, enum.CategoryName) {
	percents, name := m.nenHexagon.DecreaseCurrHexValue()
	m.hatsu.SetCategoryPercents(percents)

	return percents, name
}

func (m *Manager) ResetNenCategory() (int, enum.CategoryName) {
	currHexValue, name := m.nenHexagon.ResetCategory()
	m.hatsu.SetCategoryPercents(m.nenHexagon.GetCategoryPercents())

	return currHexValue, name
}

func (m *Manager) GetNenCategoryName() enum.CategoryName {
	return m.nenHexagon.GetCategoryName()
}

func (m *Manager) GetCurrHexValue() int {
	return m.nenHexagon.GetCurrHexValue()
}
