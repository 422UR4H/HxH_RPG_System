package status

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
)

type StatusManager struct {
	status map[enum.StatusName]Bar
}

func NewStatusManager(status map[enum.StatusName]Bar) *StatusManager {
	return &StatusManager{
		status: status,
	}
}

func (sm *StatusManager) Get(name enum.StatusName) (Bar, error) {
	if status, ok := sm.status[name]; ok {
		return status, nil
	}
	return Bar{}, errors.New("status not found")
}

func (sm *StatusManager) GetMaxOf(name enum.StatusName) (int, error) {
	status, err := sm.Get(name)
	if err != nil {
		return 0, err
	}
	return status.GetMax(), nil
}

func (sm *StatusManager) GetMinOf(name enum.StatusName) (int, error) {
	status, err := sm.Get(name)
	if err != nil {
		return 0, err
	}
	return status.GetMin(), nil
}

func (sm *StatusManager) GetCurrentOf(name enum.StatusName) (int, error) {
	status, err := sm.Get(name)
	if err != nil {
		return 0, err
	}
	return status.GetCurrent(), nil
}
