package campaign

import (
	"time"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type Campaign struct {
	UUID                    uuid.UUID
	MasterUUID              uuid.UUID
	ScenarioUUID            *uuid.UUID
	Name                    string
	BriefInitialDescription string
	BriefFinalDescription   *string
	Description             string
	IsPublic                bool
	CallLink                string
	StoryStartAt            time.Time
	StoryCurrentAt          *time.Time
	StoryEndAt              *time.Time
	CharacterSheets         []csEntity.Summary
	PendingSheets           []csEntity.Summary
	Matches                 []match.Summary
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

func NewCampaign(
	masterUUID uuid.UUID,
	scenarioUUID *uuid.UUID,
	name string,
	briefInitialDescription string,
	description string,
	isPublic bool,
	callLink string,
	storyStartAt time.Time,
	storyCurrentAt *time.Time,
) (*Campaign, error) {
	now := time.Now()
	return &Campaign{
		UUID:                    uuid.New(),
		MasterUUID:              masterUUID,
		ScenarioUUID:            scenarioUUID,
		Name:                    name,
		BriefInitialDescription: briefInitialDescription,
		Description:             description,
		IsPublic:                isPublic,
		CallLink:                callLink,
		StoryStartAt:            storyStartAt,
		StoryCurrentAt:          storyCurrentAt,
		CreatedAt:               now,
		UpdatedAt:               now,
	}, nil
}
