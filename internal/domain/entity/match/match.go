package match

import (
	"time"

	"github.com/google/uuid"
)

type Match struct {
	UUID             uuid.UUID
	MasterUUID       uuid.UUID
	CampaignUUID     uuid.UUID
	Title            string
	BriefDescription string
	Description      string
	StoryStartAt     time.Time
	StoryEndAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewMatch(
	masterUUID uuid.UUID,
	campaignUUID uuid.UUID,
	title string,
	briefDescription string,
	description string,
	storyStartAt time.Time,
) (*Match, error) {
	now := time.Now()
	return &Match{
		UUID:             uuid.New(),
		MasterUUID:       masterUUID,
		CampaignUUID:     campaignUUID,
		Title:            title,
		BriefDescription: briefDescription,
		Description:      description,
		StoryStartAt:     storyStartAt,
		StoryEndAt:       nil,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}
