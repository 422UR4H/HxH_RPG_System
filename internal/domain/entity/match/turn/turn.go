package turn

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
)

type Turn struct {
	masterActions []action.MasterAction //nolint:unused // WIP: match system under development
	actions       []action.Action
	events        []GameEvent
	coast         *int       //nolint:unused // WIP: match system under development // if nil, the turn is free (no race in this turn)
	finishedAt    *time.Time //nolint:unused // WIP: match system under development
}

func NewTurn() *Turn {
	return &Turn{
		actions: []action.Action{},
		events:  []GameEvent{},
	}
}

func (t *Turn) AddAction(action *action.Action) {
	t.actions = append(t.actions, *action)
}

func (t *Turn) GetActions() []action.Action {
	return t.actions
}

func (t *Turn) GetLastAction() (action.Action, error) {
	if len(t.actions) == 0 {
		return action.Action{}, ErrTurnIsEmpty
	}
	return t.actions[len(t.actions)-1], nil
}
