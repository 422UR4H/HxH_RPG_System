package status

import (
	"errors"

	enum "github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
)

type StatusManager struct {
	status map[enum.StatusName]IStatus
}

func NewStatusManager(status map[enum.StatusName]IStatus) *StatusManager {
	return &StatusManager{
		status: status,
	}
}

// TODO: refactor this exception
func (sm *StatusManager) Get(name enum.StatusName) (IStatus, error) {
	if status, ok := sm.status[name]; ok {
		return status, nil
	}
	return nil, errors.New("status not found")
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
