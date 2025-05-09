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
	if title == "" {
		return nil, ErrEmptyTitle
	}

	if len(title) < 5 {
		return nil, ErrMinTitleLength
	}

	if len(title) > 32 {
		return nil, ErrMaxTitleLength
	}

	if len(briefDescription) > 64 {
		return nil, ErrMaxBriefDescLength
	}

	if storyStartAt.IsZero() {
		return nil, ErrInvalidStartDate
	}

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
