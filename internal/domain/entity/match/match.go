package match

import (
	"time"

	"github.com/google/uuid"
)

type Match struct {
	UUID                    uuid.UUID
	MasterUUID              uuid.UUID
	CampaignUUID            uuid.UUID
	Title                   string
	BriefInitialDescription string
	BriefFinalDescription   *string
	Description             string
	IsPublic                bool
	GameStartAt             time.Time
	StoryStartAt            time.Time
	StoryEndAt              *time.Time
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func NewMatch(
	masterUUID uuid.UUID,
	campaignUUID uuid.UUID,
	title string,
	briefInitialDescription string,
	description string,
	isPublic bool,
	gameStartAt time.Time,
	storyStartAt time.Time,
) (*Match, error) {
	now := time.Now()
	return &Match{
		UUID:                    uuid.New(),
		MasterUUID:              masterUUID,
		CampaignUUID:            campaignUUID,
		Title:                   title,
		BriefInitialDescription: briefInitialDescription,
		Description:             description,
		IsPublic:                isPublic,
		GameStartAt:             gameStartAt,
		StoryStartAt:            storyStartAt,
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}
