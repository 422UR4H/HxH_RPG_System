package model

import (
	"time"

	"github.com/google/uuid"
)

// CharacterSheetSummary represents a summarized version of the character sheet
// used for listing. Contains only the essential fields for display in the list.
type CharacterSheetSummary struct {
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
	TalentLvl      int
	PhysicalsLvl   int
	MentalsLvl     int
	SpiritualsLvl  int
	SkillsLvl      int
	Stamina        StatusBar
	Health         StatusBar
	Aura           StatusBar
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
