package turn

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
)

type Turn struct {
	masterActions []action.MasterAction
	action        action.Action
	reactions     action.PriorityQueue
	mode          enum.TurnMode
	finishedAt    *time.Time
}

func NewTurn(
	mode enum.TurnMode,
	act action.Action,
	react []action.Action,
) *Turn {
	return &Turn{
		mode:      mode,
		action:    act,
		reactions: react,
	}
}

func (t *Turn) AttachReaction(reaction action.Action) {
	t.reactions = append(t.reactions, reaction)
}

func (t *Turn) GetMode() enum.TurnMode {
	return t.mode
}

func (t *Turn) GetAction() action.Action {
	return t.action
}

func (t *Turn) GetReactions() []action.Action {
	return t.reactions
}

func (t *Turn) GetFinishedAt() *time.Time {
	return t.finishedAt
}
