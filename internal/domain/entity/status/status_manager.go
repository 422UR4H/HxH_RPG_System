package status

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/enum"
)

type Manager struct {
	status map[enum.StatusName]Bar
}

func NewStatusManager(status map[enum.StatusName]Bar) *Manager {
	return &Manager{
		status: status,
	}
}

func (sm *Manager) Get(name enum.StatusName) (Bar, error) {
	if status, ok := sm.status[name]; ok {
		return status, nil
	}
	return Bar{}, errors.New("status not found")
}

func (sm *Manager) GetMaxOf(name enum.StatusName) (int, error) {
	status, err := sm.Get(name)
	if err != nil {
		return 0, err
	}
	return status.GetMax(), nil
}

func (sm *Manager) GetMinOf(name enum.StatusName) (int, error) {
	status, err := sm.Get(name)
	if err != nil {
		return 0, err
	}
	return status.GetMin(), nil
}

func (sm *Manager) GetCurrentOf(name enum.StatusName) (int, error) {
	status, err := sm.Get(name)
	if err != nil {
		return 0, err
	}
	return status.GetCurrent(), nil
}
