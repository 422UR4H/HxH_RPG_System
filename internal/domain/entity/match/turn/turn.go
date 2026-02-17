package turn

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
)

type Turn struct {
	masterActions []action.MasterAction
	actions       []action.Action
	events        []GameEvent
	mode          enum.TurnMode
	coast         *int // if nil, the turn is free (no race in this turn)
	finishedAt    *time.Time
}

func NewTurn(mode enum.TurnMode) *Turn {
	return &Turn{
		mode:    mode,
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
