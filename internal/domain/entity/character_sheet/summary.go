package charactersheet

import (
	"time"

	"github.com/google/uuid"
)

type StatusBar struct {
	Min  int
	Curr int
	Max  int
}

type Summary struct {
	ID             int
	UUID           uuid.UUID
	PlayerUUID     *uuid.UUID
	MasterUUID     *uuid.UUID
	CampaignUUID   *uuid.UUID
	NickName       string
	FullName       string
	Alignment      string
	CharacterClass string
	Birthday       time.Time
	CategoryName   string
	CurrHexValue   *int
	Level          int
	Points         int
	// CharExp is a denormalized copy of the character accumulated exp, sourced from
	// char_exp in character_sheets. Only used for summary list responses; never for game logic.
	CharExp        int
	TalentLvl      int
	PhysicalsLvl   int
	MentalsLvl     int
	SpiritualsLvl  int
	SkillsLvl      int
	Stamina        StatusBar
	Health         StatusBar
	Aura           StatusBar
	StoryStartAt   *time.Time
	StoryCurrentAt *time.Time
	DeadAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AvatarURL      *string
	CoverURL       *string
}
