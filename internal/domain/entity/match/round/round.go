package round

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/turn"
)

type Round struct {
	mode          enum.RoundMode
	turns         []*turn.Turn
	masterActions []action.MasterAction //nolint:unused // WIP: match system under development // think more about this field and its usage
	events        []GameEvent
	coast         *int       //nolint:unused // WIP: match system under development // if nil, the turn is free (no race in this turn)
	finishedAt    *time.Time //nolint:unused // WIP: match system under development
}

func NewRound(mode enum.RoundMode) *Round {
	return &Round{
		mode:   mode,
		turns:  []*turn.Turn{},
		events: []GameEvent{},
	}
}

func (r *Round) GetMode() enum.RoundMode {
	return r.mode
}
