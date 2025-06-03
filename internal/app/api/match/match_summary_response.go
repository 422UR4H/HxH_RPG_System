package match

import (
	"net/http"

	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type MatchSummaryResponse struct {
	UUID             uuid.UUID `json:"uuid"`
	CampaignUUID     uuid.UUID `json:"campaign_uuid"`
	Title            string    `json:"title"`
	BriefDescription string    `json:"brief_description"`
	StoryStartAt     string    `json:"story_start_at"`
	StoryEndAt       *string   `json:"story_end_at,omitempty"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
}

func ToSummaryResponse(m *domainMatch.Summary) MatchSummaryResponse {
	var storyEndAtStr *string
	if m.StoryEndAt != nil {
		formatted := m.StoryEndAt.Format("2006-01-02")
		storyEndAtStr = &formatted
	}

	return MatchSummaryResponse{
		UUID:             m.UUID,
		CampaignUUID:     m.CampaignUUID,
		Title:            m.Title,
		BriefDescription: m.BriefDescription,
		StoryStartAt:     m.StoryStartAt.Format("2006-01-02"),
		StoryEndAt:       storyEndAtStr,
		CreatedAt:        m.CreatedAt.Format(http.TimeFormat),
		UpdatedAt:        m.UpdatedAt.Format(http.TimeFormat),
	}
}
