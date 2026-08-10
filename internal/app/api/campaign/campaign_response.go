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
	MasterUUID uuid.UUID `json:"masterUuid"`
	// ScenarioUUID     uuid.UUID `json:"scenarioUuid"`
	Name                    string  `json:"name"`
	BriefInitialDescription string  `json:"briefInitialDescription"`
	BriefFinalDescription   *string `json:"briefFinalDescription,omitempty"`
	Description             string  `json:"description"`
	IsPublic                bool    `json:"isPublic"`
	CallLink                string  `json:"callLink"`
	StoryStartAt            string  `json:"storyStartAt"`
	StoryCurrentAt          *string `json:"storyCurrentAt,omitempty"`
	StoryEndAt              *string `json:"storyEndAt,omitempty"`
	CreatedAt               string  `json:"createdAt"`
	UpdatedAt               string  `json:"updatedAt"`

	Matches []match.MatchSummaryResponse `json:"matches"`
}

type CampaignMasterResponse struct {
	CampaignBaseResponse
	CharacterSheets []sheet.CharacterPrivateSummaryResponse `json:"characterSheets"`
	PendingSheets   []sheet.CharacterPrivateSummaryResponse `json:"pendingSheets"`
}

type CampaignPlayerResponse struct {
	CampaignBaseResponse
	CharacterSheets []sheet.CharacterPublicSummaryResponse `json:"characterSheets"`
	MyPendingSheet  *sheet.CharacterPrivateSummaryResponse `json:"myPendingSheet,omitempty"`
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

func ToPlayerResponse(
	c *campaign.Campaign,
	enrollmentStatuses map[uuid.UUID]string,
	playerUUID uuid.UUID,
) CampaignPlayerResponse {
	base := toSummaryBaseResponse(c)

	for i, m := range c.Matches {
		if status, ok := enrollmentStatuses[m.UUID]; ok {
			s := status
			base.Matches[i].MyEnrollmentStatus = &s
		}
	}

	characterSheets := make([]sheet.CharacterPublicSummaryResponse, 0, len(c.CharacterSheets))
	for _, cs := range c.CharacterSheets {
		characterSheets = append(characterSheets, sheet.ToPublicSummaryResponse(&cs))
	}

	var myPendingSheet *sheet.CharacterPrivateSummaryResponse
	for _, ps := range c.PendingSheets {
		if ps.PlayerUUID != nil && *ps.PlayerUUID == playerUUID {
			r := sheet.ToPrivateSummaryResponse(&ps)
			myPendingSheet = &r
			break
		}
	}

	return CampaignPlayerResponse{
		CampaignBaseResponse: base,
		CharacterSheets:      characterSheets,
		MyPendingSheet:       myPendingSheet,
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
