package spiritual

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/enum"
)

type Manager struct {
	principles map[enum.PrincipleName]NenPrinciple
	hatsu      *Hatsu
}

func NewPrinciplesManager(
	principles map[enum.PrincipleName]NenPrinciple,
	hatsu *Hatsu,
) *Manager {

	return &Manager{
		principles: principles,
		hatsu:      hatsu,
	}
}

func (pm *Manager) IncreaseExpByPrinciple(
	name enum.PrincipleName, exp int,
) (int, error) {

	if principle, ok := pm.principles[name]; ok {
		return principle.CascadeUpgradeTrigger(exp), nil
	}
	return 0, fmt.Errorf("principle %s not found", name.String())
}

func (pm *Manager) IncreaseExpByCategory(
	name enum.CategoryName, exp int,
) (int, error) {

	diff, err := pm.hatsu.IncreaseExp(exp, name)
	if err != nil {
		return 0, err
	}
	return diff, nil
}

func (pm *Manager) Get(name enum.PrincipleName) (IPrinciple, error) {
	if name == enum.Hatsu {
		return pm.hatsu, nil
	}
	if principle, ok := pm.principles[name]; ok {
		return &principle, nil
	}
	return nil, fmt.Errorf("principle %s not found", name.String())
}

func (pm *Manager) GetExpPointsOfPrinciple(
	name enum.PrincipleName) (int, error) {

	principle, err := pm.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get exp")
	}
	return principle.GetExpPoints(), nil
}

func (pm *Manager) GetLevelOfPrinciple(
	name enum.PrincipleName) (int, error) {

	principle, err := pm.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level")
	}
	return principle.GetLevel(), nil
}

func (pm *Manager) GetExpPointsOfCategory(
	name enum.CategoryName) (int, error) {

	principle, err := pm.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get exp")
	}
	return principle.GetExpPoints(), nil
}

func (pm *Manager) GetLevelOfCategory(
	name enum.CategoryName) (int, error) {

	principle, err := pm.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level")
	}
	return principle.GetLevel(), nil
}
