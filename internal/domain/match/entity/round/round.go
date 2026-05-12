package round

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
)

type Round struct {
	mode          enum.RoundMode
	turns         []*turn.Turn
	masterActions []action.MasterAction //nolint:unused // WIP: match system under development // think more about this field and its usage
	events        []GameEvent
	coast         *int       //nolint:unused // WIP: match system under development // if nil, the turn is free (no race in this turn)
	finishedAt    *time.Time
}

func NewRound(mode enum.RoundMode) *Round {
	return &Round{
		mode:   mode,
		turns:  []*turn.Turn{},
		events: []GameEvent{},
	}
}

func (r *Round) AppendTurn(t *turn.Turn) {
	r.turns = append(r.turns, t)
}

func (r *Round) CurrentTurn() *turn.Turn {
	if len(r.turns) == 0 {
		return nil
	}
	return r.turns[len(r.turns)-1]
}

func (r *Round) HasOpenTurn() bool {
	t := r.CurrentTurn()
	return t != nil && t.GetFinishedAt() == nil
}

func (r *Round) Close(at time.Time) {
	r.finishedAt = &at
}

func (r *Round) GetFinishedAt() *time.Time {
	return r.finishedAt
}

func (r *Round) ToggleMode() {
	if r.mode == enum.Race {
		r.mode = enum.Free
	} else {
		r.mode = enum.Race
	}
}

func (r *Round) GetMode() enum.RoundMode {
	return r.mode
}
