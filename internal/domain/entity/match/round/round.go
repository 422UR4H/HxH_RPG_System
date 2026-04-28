package round

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/turn"
)

type Round struct {
	coast      int
	turns      []*turn.Turn
	mode       enum.TurnMode
	finishedAt *time.Time
}

func NewRound(mode enum.TurnMode, coast int) *Round {
	return &Round{
		mode:  mode,
		turns: []*turn.Turn{},
		coast: coast,
	}
}

func (r *Round) GetMode() enum.TurnMode {
	return r.mode
}

func (r *Round) GetCoast() int {
	return r.coast
}
