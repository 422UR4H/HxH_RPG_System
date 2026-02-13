package match

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/turn"
)

type Scene struct {
	Category                enum.SceneCategory
	BriefInitialDescription string // need to be sent by the master when starting the match
	BriefFinalDescription   *string

	Turns []turn.Turn

	GameStartAt  time.Time
	StoryStartAt time.Time
	StoryEndAt   *time.Time
	CreatedAt    time.Time
	FinishedAt   time.Time
}
