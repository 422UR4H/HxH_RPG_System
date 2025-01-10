package spiritual

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
)

type PrinciplesManager struct {
	principles map[enum.PrincipleName]NenPrinciple
	hatsu      *Hatsu
}

func NewPrinciplesManager(
	principles map[enum.PrincipleName]NenPrinciple,
	hatsu *Hatsu,
) *PrinciplesManager {

	return &PrinciplesManager{
		principles: principles,
		hatsu:      hatsu,
	}
}

func (pm *PrinciplesManager) IncreaseExpByPrinciple(
	name enum.PrincipleName, exp int,
) (int, error) {

	if principle, ok := pm.principles[name]; ok {
		return principle.CascadeUpgradeTrigger(exp), nil
	}
	return 0, fmt.Errorf("principle %s not found", name.String())
}

func (pm *PrinciplesManager) IncreaseExpByCategory(
	name enum.CategoryName, exp int,
) (int, error) {

	diff, err := pm.hatsu.IncreaseExp(exp, name)
	if err != nil {
		return 0, err
	}
	return diff, nil
}

func (pm *PrinciplesManager) Get(name enum.PrincipleName) (IPrinciple, error) {
	if name == enum.Hatsu {
		return pm.hatsu, nil
	}
	if principle, ok := pm.principles[name]; ok {
		return &principle, nil
	}
	return nil, fmt.Errorf("principle %s not found", name.String())
}

func (pm *PrinciplesManager) GetExpPointsOfPrinciple(
	name enum.PrincipleName) (int, error) {

	principle, err := pm.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get exp")
	}
	return principle.GetExpPoints(), nil
}

func (pm *PrinciplesManager) GetLevelOfPrinciple(
	name enum.PrincipleName) (int, error) {

	principle, err := pm.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level")
	}
	return principle.GetLevel(), nil
}

func (pm *PrinciplesManager) GetExpPointsOfCategory(
	name enum.CategoryName) (int, error) {

	principle, err := pm.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get exp")
	}
	return principle.GetExpPoints(), nil
}

func (pm *PrinciplesManager) GetLevelOfCategory(
	name enum.CategoryName) (int, error) {

	principle, err := pm.hatsu.Get(name)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", err, "failed to get level")
	}
	return principle.GetLevel(), nil
}
