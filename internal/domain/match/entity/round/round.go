package round

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/google/uuid"
)

type Round struct {
	id            uuid.UUID
	mode          enum.RoundMode
	turns         []*turn.Turn
	masterActions []action.MasterAction //nolint:unused // WIP: match system under development // think more about this field and its usage
	events        []GameEvent
	coast         *int //nolint:unused // WIP: match system under development // if nil, the turn is free (no race in this turn)
	createdAt     time.Time
	finishedAt    *time.Time
}

func NewRound(mode enum.RoundMode) *Round {
	return &Round{
		id:        uuid.New(),
		mode:      mode,
		turns:     []*turn.Turn{},
		events:    []GameEvent{},
		createdAt: time.Now(),
	}
}

func ReconstructRound(id uuid.UUID, mode enum.RoundMode, createdAt time.Time) *Round {
	return &Round{
		id:        id,
		mode:      mode,
		turns:     []*turn.Turn{},
		events:    []GameEvent{},
		createdAt: createdAt,
	}
}

func (r *Round) GetID() uuid.UUID {
	return r.id
}

func (r *Round) GetCreatedAt() time.Time {
	return r.createdAt
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
