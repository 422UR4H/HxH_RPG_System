package campaign

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/match"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type CampaignBaseResponse struct {
	UUID       uuid.UUID `json:"uuid"`
	MasterUUID uuid.UUID `json:"master_uuid"`
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

	Matches []match.MatchSummaryResponse `json:"matches"`
}

type CampaignMasterResponse struct {
	CampaignBaseResponse
	CharacterSheets []sheet.CharacterPrivateSummaryResponse `json:"character_sheets"`
	PendingSheets   []sheet.CharacterPrivateSummaryResponse `json:"pending_sheets"`
}

type CampaignPlayerResponse struct {
	CampaignBaseResponse
	CharacterSheets []sheet.CharacterPublicSummaryResponse `json:"character_sheets"`
}

func ToMasterResponse(campaign *campaign.Campaign) CampaignMasterResponse {
	characterSheets := make([]sheet.CharacterPrivateSummaryResponse, 0, len(campaign.CharacterSheets))
	for _, cs := range campaign.CharacterSheets {
		characterSheets = append(characterSheets, sheet.ToPrivateSummaryResponse(&cs))
	}

	pendingSheets := make([]sheet.CharacterPrivateSummaryResponse, 0, len(campaign.PendingSheets))
	for _, ps := range campaign.PendingSheets {
		pendingSheets = append(pendingSheets, sheet.ToPrivateSummaryResponse(&ps))
	}

	return CampaignMasterResponse{
		CampaignBaseResponse: toSummaryBaseResponse(campaign),
		CharacterSheets:      characterSheets,
		PendingSheets:        pendingSheets,
	}
}

func ToPlayerResponse(campaign *campaign.Campaign) CampaignPlayerResponse {
	characterSheets := make([]sheet.CharacterPublicSummaryResponse, 0, len(campaign.CharacterSheets))
	for _, cs := range campaign.CharacterSheets {
		characterSheets = append(characterSheets, sheet.ToPublicSummaryResponse(&cs))
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
		MasterUUID:              campaign.MasterUUID,
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
