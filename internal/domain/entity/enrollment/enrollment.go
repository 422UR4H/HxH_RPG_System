package enrollment

import (
	"time"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/google/uuid"
)

type PlayerRef struct {
	UUID uuid.UUID
	Nick string
}

type Enrollment struct {
	UUID           uuid.UUID
	Status         string
	CreatedAt      time.Time
	CharacterSheet csEntity.Summary
	Player         PlayerRef
}
