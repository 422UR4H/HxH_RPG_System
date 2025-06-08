package campaign

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type CampaignBaseResponse struct {
	UUID uuid.UUID `json:"uuid"`
	// ScenarioUUID     uuid.UUID `json:"scenario_uuid"`
	Name                    string  `json:"name"`
	BriefInitialDescription string  `json:"brief_initial_description"`
	BriefFinalDescription   *string `json:"brief_final_description,omitempty"`
	Description             string  `json:"description"`
	IsPublic                bool    `json:"is_public"`
	CallLink                string  `json:"call_link"`
	StoryStartAt            string  `json:"story_start_at"`
	StoryCurrentAt          *string `json:"story_current_at,omitempty"`
	StoryEndAt              *string `json:"story_end_at,omitempty"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`

	Matches []match.MatchSummaryResponse `json:"matches,omitempty"`
}

type CampaignMasterResponse struct {
	CampaignBaseResponse
	CharacterSheets []sheet.CharacterMasterSummaryResponse `json:"character_sheets,omitempty"`
	PendingSheets   []sheet.CharacterMasterSummaryResponse `json:"pending_sheets,omitempty"`
}

type CampaignPlayerResponse struct {
	CampaignBaseResponse
	CharacterSheets []sheet.CharacterPlayerSummaryResponse `json:"character_sheets,omitempty"`
}

func ToMasterResponse(campaign *campaign.Campaign) CampaignMasterResponse {
	characterSheets := make([]sheet.CharacterMasterSummaryResponse, 0, len(campaign.CharacterSheets))
	for _, cs := range campaign.CharacterSheets {
		characterSheets = append(characterSheets, sheet.ToSummaryMasterResponse(&cs))
	}

	pendingSheets := make([]sheet.CharacterMasterSummaryResponse, 0, len(campaign.PendingSheets))
	for _, ps := range campaign.PendingSheets {
		pendingSheets = append(pendingSheets, sheet.ToSummaryMasterResponse(&ps))
	}

	return CampaignMasterResponse{
		CampaignBaseResponse: toSummaryBaseResponse(campaign),
		CharacterSheets:      characterSheets,
		PendingSheets:        pendingSheets,
	}
}

func ToPlayerResponse(campaign *campaign.Campaign) CampaignPlayerResponse {
	characterSheets := make([]sheet.CharacterPlayerSummaryResponse, 0, len(campaign.CharacterSheets))
	for _, cs := range campaign.CharacterSheets {
		characterSheets = append(characterSheets, sheet.ToSummaryPlayerResponse(&cs))
	}
	return CampaignPlayerResponse{
		CampaignBaseResponse: toSummaryBaseResponse(campaign),
		CharacterSheets:      characterSheets,
	}
}

func toSummaryBaseResponse(campaign *campaign.Campaign) CampaignBaseResponse {
	var storyCurrentAtStr, storyEndAtStr *string
	if campaign.StoryCurrentAt != nil {
		formattedTime := campaign.StoryCurrentAt.Format(time.RFC3339)
		storyCurrentAtStr = &formattedTime
	}
	if campaign.StoryEndAt != nil {
		formattedDate := campaign.StoryEndAt.Format("2006-01-02")
		storyEndAtStr = &formattedDate
	}

	matches := make([]match.MatchSummaryResponse, 0, len(campaign.Matches))
	for _, m := range campaign.Matches {
		matches = append(matches, match.ToSummaryResponse(&m))
	}
	return CampaignBaseResponse{
		UUID:                    campaign.UUID,
		Name:                    campaign.Name,
		BriefInitialDescription: campaign.BriefInitialDescription,
		BriefFinalDescription:   campaign.BriefFinalDescription,
		Description:             campaign.Description,
		IsPublic:                campaign.IsPublic,
		CallLink:                campaign.CallLink,
		StoryStartAt:            campaign.StoryStartAt.Format("2006-01-02"),
		StoryCurrentAt:          storyCurrentAtStr,
		StoryEndAt:              storyEndAtStr,
		CreatedAt:               campaign.CreatedAt.Format(time.RFC3339),
		UpdatedAt:               campaign.UpdatedAt.Format(time.RFC3339),
		Matches:                 matches,
	}
}
