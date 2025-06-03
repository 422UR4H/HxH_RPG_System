package match

import (
	"time"

	"github.com/google/uuid"
)

type Summary struct {
	UUID                    uuid.UUID
	CampaignUUID            uuid.UUID
	Title                   string
	BriefInitialDescription string
	BriefFinalDescription   *string
	IsPublic                bool
	GameStartAt             time.Time
	StoryStartAt            time.Time
	StoryEndAt              *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
