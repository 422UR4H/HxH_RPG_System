package turn

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// GameEvent represents an event that occurs during a turn/match, such as:
// - a change in the game date
// - a character's death during the turn
// - an breaking news that affects the match
type GameEvent struct {
	category     []enum.GameEventCategory
	description  *string
	changeDateTo *time.Time
	happenedAt   time.Time
}

func NewGameEvent(
	category []enum.GameEventCategory,
	description *string,
	changeDateTo *time.Time,
) *GameEvent {
	if category == nil {
		category = []enum.GameEventCategory{enum.Other}
	}

	return &GameEvent{
		category:     category,
		description:  description,
		changeDateTo: changeDateTo,
		happenedAt:   time.Now(),
	}
}
