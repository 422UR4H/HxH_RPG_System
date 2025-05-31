package sheet

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

type CharacterSummaryResponse struct {
	UUID           uuid.UUID  `json:"uuid"`
	PlayerUUID     *uuid.UUID `json:"player_uuid,omitempty"`
	MasterUUID     *uuid.UUID `json:"master_uuid,omitempty"`
	CampaignUUID   *uuid.UUID `json:"campaign_uuid,omitempty"`
	NickName       string     `json:"nick_name"`
	FullName       string     `json:"full_name"`
	Alignment      string     `json:"alignment"`
	CharacterClass string     `json:"character_class"`
	Birthday       string     `json:"birthday"`
	CategoryName   string     `json:"category_name"`
	CurrHexValue   *int       `json:"curr_hex_value,omitempty"`
	Level          int        `json:"level"`
	Points         int        `json:"points"`
	TalentLvl      int        `json:"talent_lvl"`
	PhysicalsLvl   int        `json:"physicals_lvl"`
	MentalsLvl     int        `json:"mentals_lvl"`
	SpiritualsLvl  int        `json:"spirituals_lvl"`
	SkillsLvl      int        `json:"skills_lvl"`
	Stamina        StatusBar  `json:"stamina"`
	Health         StatusBar  `json:"health"`
	Aura           StatusBar  `json:"aura"`
	CreatedAt      string     `json:"created_at"`
	UpdatedAt      string     `json:"updated_at"`
}

type StatusBar struct {
	Min  int `json:"min"`
	Curr int `json:"curr"`
	Max  int `json:"max"`
}

func ToSummaryResponse(
	sheet model.CharacterSheetSummary,
) CharacterSummaryResponse {
	stamina := sheet.Stamina
	health := sheet.Health
	// aura := sheet.Aura

	return CharacterSummaryResponse{
		UUID:           sheet.UUID,
		PlayerUUID:     sheet.PlayerUUID,
		MasterUUID:     sheet.MasterUUID,
		CampaignUUID:   sheet.CampaignUUID,
		NickName:       sheet.NickName,
		FullName:       sheet.FullName,
		Alignment:      sheet.Alignment,
		CharacterClass: sheet.CharacterClass,
		Birthday:       sheet.Birthday.Format("2006-01-02"),
		CategoryName:   sheet.CategoryName,
		CurrHexValue:   sheet.CurrHexValue,
		Level:          sheet.Level,
		Points:         sheet.Points,
		TalentLvl:      sheet.TalentLvl,
		PhysicalsLvl:   sheet.PhysicalsLvl,
		MentalsLvl:     sheet.MentalsLvl,
		SpiritualsLvl:  sheet.SpiritualsLvl,
		SkillsLvl:      sheet.SkillsLvl,
		Stamina: StatusBar{
			Min:  stamina.Min,
			Curr: stamina.Curr,
			Max:  stamina.Max,
		},
		Health: StatusBar{
			Min:  health.Min,
			Curr: health.Curr,
			Max:  health.Max,
		},
		// Aura: StatusBar{
		// 	Min:  aura.Min,
		// 	Curr: aura.Curr,
		// 	Max:  aura.Max,
		// },
		CreatedAt: sheet.CreatedAt.Format(time.RFC3339),
		UpdatedAt: sheet.UpdatedAt.Format(time.RFC3339),
	}
}
