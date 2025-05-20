package match

import (
	"time"

	"github.com/google/uuid"
)

type Summary struct {
	UUID             uuid.UUID
	CampaignUUID     uuid.UUID
	Title            string
	BriefDescription string
	StoryStartAt     time.Time
	StoryEndAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
