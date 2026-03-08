package round

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/turn"
)

type Round struct {
	mode  enum.TurnMode
	turns []*turn.Turn
}

func NewRound(mode enum.TurnMode) *Round {
	return &Round{
		mode:  mode,
		turns: []*turn.Turn{},
	}
}

func (r *Round) GetMode() enum.TurnMode {
	return r.mode
}
