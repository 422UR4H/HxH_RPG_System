package status

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type Manager struct {
	status map[enum.StatusName]IStatusBar
}

func NewStatusManager(status map[enum.StatusName]IStatusBar) *Manager {
	return &Manager{
		status: status,
	}
}

func (sm *Manager) Get(name enum.StatusName) (IStatusBar, error) {
	if status, ok := sm.status[name]; ok {
		return status, nil
	}
	return nil, errors.New("status not found")
}

func (sm *Manager) Upgrade() error {
	for name := range sm.status {
		status, err := sm.Get(name)
		if err != nil {
			return err
		}
		status.Upgrade()
	}
	return nil
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

func (sm *Manager) GetAllStatus() map[enum.StatusName]IStatusBar {
	return sm.status
}
