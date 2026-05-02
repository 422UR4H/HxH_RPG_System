package match

import (
	"net/http"
	"time"

	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type MatchSummaryResponse struct {
	UUID                    uuid.UUID `json:"uuid"`
	CampaignUUID            uuid.UUID `json:"campaign_uuid"`
	Title                   string    `json:"title"`
	BriefInitialDescription string    `json:"brief_initial_description"`
	BriefFinalDescription   *string   `json:"brief_final_description,omitempty"`
	IsPublic                bool      `json:"is_public"`
	GameScheduledAt         string    `json:"game_scheduled_at"`
	GameStartAt             *string   `json:"game_start_at,omitempty"`
	StoryStartAt            string    `json:"story_start_at"`
	StoryEndAt              *string   `json:"story_end_at,omitempty"`
	CreatedAt               string    `json:"created_at"`
	UpdatedAt               string    `json:"updated_at"`
}

func ToSummaryResponse(m *domainMatch.Summary) MatchSummaryResponse {
	var storyEndAtStr *string
	if m.StoryEndAt != nil {
		formatted := m.StoryEndAt.Format("2006-01-02")
		storyEndAtStr = &formatted
	}

	var gameStartAtStr *string
	if m.GameStartAt != nil {
		formatted := m.GameStartAt.Format(time.RFC3339)
		gameStartAtStr = &formatted
	}

	return MatchSummaryResponse{
		UUID:                    m.UUID,
		CampaignUUID:            m.CampaignUUID,
		Title:                   m.Title,
		BriefInitialDescription: m.BriefInitialDescription,
		BriefFinalDescription:   m.BriefFinalDescription,
		IsPublic:                m.IsPublic,
		GameScheduledAt:         m.GameScheduledAt.Format(time.RFC3339),
		GameStartAt:             gameStartAtStr,
		StoryStartAt:            m.StoryStartAt.Format("2006-01-02"),
		StoryEndAt:              storyEndAtStr,
		CreatedAt:               m.CreatedAt.Format(http.TimeFormat),
		UpdatedAt:               m.UpdatedAt.Format(http.TimeFormat),
	}
}
