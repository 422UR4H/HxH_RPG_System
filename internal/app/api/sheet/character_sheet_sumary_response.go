package sheet

import (
	"time"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	sheetEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/google/uuid"
)

// charExpTable is used only to compute curr_exp / next_lvl_base_exp for summary responses.
// It must use CHARACTER_COEFF to match the domain entity. The authoritative values always
// come from CharacterExp.GetCurrentExp / GetNextLvlBaseExp after full sheet reconstruction.
var charExpTable = experience.NewExpTable(sheetEntity.CHARACTER_COEFF)

type CharacterBaseSummaryResponse struct {
	UUID           uuid.UUID  `json:"uuid"`
	PlayerUUID     *uuid.UUID `json:"player_uuid,omitempty"`
	MasterUUID     *uuid.UUID `json:"master_uuid,omitempty"`
	CampaignUUID   *uuid.UUID `json:"campaign_uuid,omitempty"`
	NickName       string     `json:"nick_name"`
	AvatarURL      *string    `json:"avatar_url,omitempty"`
	CoverURL       *string    `json:"cover_url,omitempty"`
	StoryStartAt   *string    `json:"story_start_at,omitempty"`
	StoryCurrentAt *string    `json:"story_current_at,omitempty"`
	DeadAt         *string    `json:"dead_at,omitempty"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
}

// CharacterPrivateOnlyResponse holds the fields that are private to the sheet
// owner (and to the master of a match the sheet is enrolled in). It does NOT
// embed the base — it is meant to be nested under a base-typed parent.
type CharacterPrivateOnlyResponse struct {
	FullName       string    `json:"full_name"`
	Alignment      string    `json:"alignment"`
	CharacterClass string    `json:"character_class"`
	Birthday       string    `json:"birthday"`
	CategoryName   string    `json:"category_name"`
	CurrHexValue   *int      `json:"curr_hex_value,omitempty"`
	Level          int       `json:"level"`
	Points         int       `json:"points"`
	// CurrExp and NxtLvlBaseExp are derived from char_exp (denormalized) + charExpTable.
	// Do NOT use for game logic — always use the full sheet build for that.
	CurrExp        int       `json:"curr_exp"`
	NxtLvlBaseExp  int       `json:"next_lvl_base_exp"`
	TalentLvl      int       `json:"talent_lvl"`
	PhysicalsLvl   int       `json:"physicals_lvl"`
	MentalsLvl     int       `json:"mentals_lvl"`
	SpiritualsLvl  int       `json:"spirituals_lvl"`
	SkillsLvl      int       `json:"skills_lvl"`
	Stamina        StatusBar `json:"stamina"`
	Health         StatusBar `json:"health"`
	// Aura           StatusBar  `json:"aura"`
}

// CharacterPrivateSummaryResponse is the flat (base + private) shape used by
// existing endpoints (kept for backward compatibility).
type CharacterPrivateSummaryResponse struct {
	CharacterBaseSummaryResponse
	CharacterPrivateOnlyResponse
}

type CharacterPublicSummaryResponse struct {
	CharacterBaseSummaryResponse
}

type StatusBar struct {
	Min     int `json:"min"`
	Current int `json:"current"`
	Max     int `json:"max"`
}

func ToPrivateOnlyResponse(sheet *csEntity.Summary) CharacterPrivateOnlyResponse {
	stamina := sheet.Stamina
	health := sheet.Health
	// aura := sheet.Aura
	return CharacterPrivateOnlyResponse{
		FullName:       sheet.FullName,
		Alignment:      sheet.Alignment,
		CharacterClass: sheet.CharacterClass,
		Birthday:       sheet.Birthday.Format("2006-01-02"),
		CategoryName:   sheet.CategoryName,
		CurrHexValue:   sheet.CurrHexValue,
		Level:         sheet.Level,
		Points:        sheet.Points,
		CurrExp:       deriveCurrExp(sheet.CharExp),
		NxtLvlBaseExp: deriveNxtLvlBaseExp(sheet.CharExp),
		TalentLvl:      sheet.TalentLvl,
		PhysicalsLvl:   sheet.PhysicalsLvl,
		MentalsLvl:     sheet.MentalsLvl,
		SpiritualsLvl:  sheet.SpiritualsLvl,
		SkillsLvl:      sheet.SkillsLvl,
		Stamina: StatusBar{
			Min:     stamina.Min,
			Current: stamina.Curr,
			Max:     stamina.Max,
		},
		Health: StatusBar{
			Min:     health.Min,
			Current: health.Curr,
			Max:     health.Max,
		},
		// Aura: StatusBar{
		// 	Min:  aura.Min,
		// 	Curr: aura.Curr,
		// 	Max:  aura.Max,
		// },
	}
}

// deriveCurrExp returns the exp accumulated within the current level.
// Level is derived from charExp itself (not from sheet.Level) because the stored level
// can be inconsistent with the accumulated exp — deriving from charExp always matches
// how the domain entity computes GetCurrentExp().
func deriveCurrExp(charExp int) int {
	level := charExpTable.GetLvlByExp(charExp)
	v := charExp - charExpTable.GetAggregateExpByLvl(level)
	if v < 0 {
		return 0
	}
	return v
}

// deriveNxtLvlBaseExp derives the base exp for the next level from charExp directly.
// Same reasoning as deriveCurrExp: uses charExp to compute level, not sheet.Level.
func deriveNxtLvlBaseExp(charExp int) int {
	level := charExpTable.GetLvlByExp(charExp)
	return charExpTable.GetBaseExpByLvl(level + 1)
}

func ToPrivateSummaryResponse(sheet *csEntity.Summary) CharacterPrivateSummaryResponse {
	return CharacterPrivateSummaryResponse{
		CharacterBaseSummaryResponse: ToBaseSummaryResponse(sheet),
		CharacterPrivateOnlyResponse: ToPrivateOnlyResponse(sheet),
	}
}

func ToPublicSummaryResponse(sheet *csEntity.Summary) CharacterPublicSummaryResponse {
	return CharacterPublicSummaryResponse{
		CharacterBaseSummaryResponse: ToBaseSummaryResponse(sheet),
	}
}

type CharacterSheetWithVisibilityResponse struct {
	CharacterBaseSummaryResponse
	Private *CharacterPrivateOnlyResponse `json:"private"`
}

func ToBaseSummaryResponse(sheet *csEntity.Summary) CharacterBaseSummaryResponse {
	var storyStartAtStr, storyCurrentAtStr, deadAtStr *string
	if sheet.StoryStartAt != nil {
		formatted := sheet.StoryStartAt.Format("2006-01-02")
		storyStartAtStr = &formatted
	}
	if sheet.StoryCurrentAt != nil {
		formatted := sheet.StoryCurrentAt.Format("2006-01-02")
		storyCurrentAtStr = &formatted
	}
	if sheet.DeadAt != nil {
		formatted := sheet.DeadAt.Format(time.RFC3339)
		deadAtStr = &formatted
	}

	return CharacterBaseSummaryResponse{
		UUID:           sheet.UUID,
		PlayerUUID:     sheet.PlayerUUID,
		MasterUUID:     sheet.MasterUUID,
		CampaignUUID:   sheet.CampaignUUID,
		NickName:       sheet.NickName,
		AvatarURL:      sheet.AvatarURL,
		CoverURL:       sheet.CoverURL,
		StoryStartAt:   storyStartAtStr,
		StoryCurrentAt: storyCurrentAtStr,
		DeadAt:         deadAtStr,
		CreatedAt:      sheet.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      sheet.UpdatedAt.Format(time.RFC3339),
	}
}
