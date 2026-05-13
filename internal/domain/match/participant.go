package match

import (
	"time"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/google/uuid"
)

type Participant struct {
	UUID      uuid.UUID
	MatchUUID uuid.UUID
	Sheet     csEntity.Summary
	JoinedAt  time.Time
	LeftAt    *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
