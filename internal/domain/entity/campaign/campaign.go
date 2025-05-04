package campaign

import (
	"time"

	"github.com/google/uuid"
)

type Campaign struct {
	UUID             uuid.UUID
	UserUUID         uuid.UUID
	ScenarioUUID     uuid.UUID
	Name             string
	BriefDescription string
	Description      string
	StoryStartAt     time.Time
	StoryCurrentAt   *time.Time
	StoryEndAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewCampaign(
	userUUID uuid.UUID,
	scenarioUUID uuid.UUID,
	name string,
	briefDescription string,
	description string,
	storyStartAt time.Time,
	storyCurrentAt *time.Time,
	storyEndAt *time.Time,
) (*Campaign, error) {
	if name == "" {
		return nil, ErrEmptyName
	}

	if storyStartAt.IsZero() {
		return nil, ErrInvalidStartDate
	}

	now := time.Now()
	return &Campaign{
		UUID:             uuid.New(),
		UserUUID:         userUUID,
		ScenarioUUID:     scenarioUUID,
		Name:             name,
		BriefDescription: briefDescription,
		Description:      description,
		StoryStartAt:     storyStartAt,
		StoryCurrentAt:   storyCurrentAt,
		StoryEndAt:       storyEndAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}
