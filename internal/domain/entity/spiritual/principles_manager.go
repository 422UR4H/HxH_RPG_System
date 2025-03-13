package spiritual

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
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
	name enum.PrincipleName, values *experience.UpgradeCascade,
) error {
	if principle, ok := m.principles[name]; ok {
		principle.CascadeUpgradeTrigger(values)
		return nil
	}
	return fmt.Errorf("principle %s not found", name.String())
}

func (m *Manager) IncreaseExpByCategory(
	name enum.CategoryName, values *experience.UpgradeCascade,
) error {
	err := m.hatsu.IncreaseExp(values, name)
	if err != nil {
		return err
	}
	return nil
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

func (m *Manager) GetNextLvlAggregateExpOfPrinciple(
	name enum.PrincipleName) (int, error) {

	principle, err := m.Get(name)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s %s", err, "failed to get aggregate exp of next lvl of", name.String())
	}
	return principle.GetNextLvlAggregateExp(), nil
}

func (m *Manager) GetNextLvlBaseExpOfPrinciple(
	name enum.PrincipleName) (int, error) {

	principle, err := m.Get(name)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s %s", err, "failed to get base exp of next lvl of", name.String())
	}
	return principle.GetNextLvlBaseExp(), nil
}

func (m *Manager) GetCurrentExpOfPrinciple(name enum.PrincipleName) (int, error) {
	principle, err := m.Get(name)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s %s", err, "failed to get current exp of", name.String())
	}
	return principle.GetCurrentExp(), nil
}

func (m *Manager) GetExpPointsOfPrinciple(name enum.PrincipleName) (int, error) {
	principle, err := m.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get exp")
	}
	return principle.GetExpPoints(), nil
}

func (m *Manager) GetLevelOfPrinciple(name enum.PrincipleName) (int, error) {
	principle, err := m.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level")
	}
	return principle.GetLevel(), nil
}

func (m *Manager) GetNextLvlAggregateExpOfCategory(
	name enum.CategoryName) (int, error) {

	category, err := m.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s %s", err, "failed to get aggregate exp of next lvl of", name.String())
	}
	return category.GetNextLvlAggregateExp(), nil
}

func (m *Manager) GetNextLvlBaseExpOfCategory(
	name enum.CategoryName) (int, error) {

	category, err := m.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s %s", err, "failed to get base exp of next lvl of", name.String())
	}
	return category.GetNextLvlBaseExp(), nil
}

func (m *Manager) GetCurrentExpOfCategory(name enum.CategoryName) (int, error) {
	category, err := m.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf(
			"%w: %s %s", err, "failed to get current exp of", name.String())
	}
	return category.GetCurrentExp(), nil
}

func (m *Manager) GetExpPointsOfCategory(name enum.CategoryName) (int, error) {
	category, err := m.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get exp")
	}
	return category.GetExpPoints(), nil
}

func (m *Manager) GetLevelOfCategory(name enum.CategoryName) (int, error) {
	category, err := m.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level")
	}
	return category.GetLevel(), nil
}

func (m *Manager) GetTestValueOfCategory(name enum.CategoryName) (int, error) {
	category, err := m.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level")
	}
	return category.GetValueForTest(), nil
}

func (m *Manager) GetNextLvlAggregateExpOfPrinciples() map[enum.PrincipleName]int {
	expList := make(map[enum.PrincipleName]int)
	for name, principle := range m.principles {
		expList[name] = principle.GetNextLvlAggregateExp()
	}
	return expList
}

func (m *Manager) GetNextLvlBaseExpOfPrinciples() map[enum.PrincipleName]int {
	expList := make(map[enum.PrincipleName]int)
	for name, principle := range m.principles {
		expList[name] = principle.GetNextLvlBaseExp()
	}
	return expList
}

func (m *Manager) GetCurrentExpOfPrinciples() map[enum.PrincipleName]int {
	expList := make(map[enum.PrincipleName]int)
	for name, principle := range m.principles {
		expList[name] = principle.GetCurrentExp()
	}
	return expList
}

func (m *Manager) GetExpPointsOfPrinciples() map[enum.PrincipleName]int {
	expList := make(map[enum.PrincipleName]int)
	for name, principle := range m.principles {
		expList[name] = principle.GetExpPoints()
	}
	return expList
}

func (m *Manager) GetLevelOfPrinciples() map[enum.PrincipleName]int {
	lvlList := make(map[enum.PrincipleName]int)
	for name, principle := range m.principles {
		lvlList[name] = principle.GetLevel()
	}
	return lvlList
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

func (m *Manager) ResetNenCategory() (int, error) {
	if m.nenHexagon == nil {
		return -1, fmt.Errorf("nen hexagon not initialized")
	}
	currHexValue := m.nenHexagon.ResetCategory()
	m.hatsu.SetCategoryPercents(m.nenHexagon.GetCategoryPercents())

	return currHexValue, nil
}

func (m *Manager) GetNenCategoryName() (enum.CategoryName, error) {
	if m.nenHexagon == nil {
		return -1, fmt.Errorf("nen hexagon not initialized")
	}
	return m.nenHexagon.GetCategoryName(), nil
}

func (m *Manager) GetCurrHexValue() (int, error) {
	if m.nenHexagon == nil {
		return -1, fmt.Errorf("nen hexagon not initialized")
	}
	return m.nenHexagon.GetCurrHexValue(), nil
}

func (m *Manager) GetNextLvlAggregateExpOfCategories() map[enum.CategoryName]int {
	return m.hatsu.GetCategoriesNextLvlAggregateExp()
}

func (m *Manager) GetNextLvlBaseExpOfCategories() map[enum.CategoryName]int {
	return m.hatsu.GetCategoriesNextLvlBaseExp()
}

func (m *Manager) GetCurrentExpOfCategories() map[enum.CategoryName]int {
	return m.hatsu.GetCategoriesCurrentExp()
}

func (m *Manager) GetExpPointsOfCategories() map[enum.CategoryName]int {
	return m.hatsu.GetCategoriesExpPoints()
}

func (m *Manager) GetLevelOfCategories() map[enum.CategoryName]int {
	return m.hatsu.GetCategoriesLevel()
}

func (m *Manager) GetTestValueOfCategories() map[enum.CategoryName]int {
	return m.hatsu.GetCategoriesLevel()
}

func (m *Manager) GetAllPrinciples() map[enum.PrincipleName]IPrinciple {
	principles := make(map[enum.PrincipleName]IPrinciple)
	for name, principle := range m.principles {
		principles[name] = &principle
	}
	return principles
}

func (m *Manager) GetAllCategories() map[enum.CategoryName]ICategory {
	return m.hatsu.GetAllCategories()
}
