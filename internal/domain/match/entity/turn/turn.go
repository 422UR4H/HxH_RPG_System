package turn

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/google/uuid"
)

type Turn struct {
	id            uuid.UUID
	action        action.Action
	reactions     []action.Action
	masterActions []action.MasterAction //nolint:unused // WIP: match system under development
	finishedAt    *time.Time
}

func NewTurn(action action.Action) *Turn {
	return &Turn{
		id:     uuid.New(),
		action: action,
	}
}

func (t *Turn) GetID() uuid.UUID {
	return t.id
}

func (t *Turn) AddReaction(action *action.Action) {
	t.reactions = append(t.reactions, *action)
}

func (t *Turn) Close(finishedAt time.Time) {
	t.finishedAt = &finishedAt
}

func (t *Turn) GetAction() action.Action {
	return t.action
}

func (t *Turn) GetReactions() []action.Action {
	reactionsCp := make([]action.Action, len(t.reactions))
	copy(reactionsCp, t.reactions)
	return reactionsCp
}

func (t *Turn) GetMasterActions() []action.MasterAction {
	masterActionsCp := make([]action.MasterAction, len(t.masterActions))
	copy(masterActionsCp, t.masterActions)
	return masterActionsCp
}

func (t *Turn) GetFinishedAt() *time.Time {
	return t.finishedAt
}
