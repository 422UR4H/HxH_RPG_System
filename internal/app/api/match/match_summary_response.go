package match

import (
	"net/http"
	"time"

	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/google/uuid"
)

type MatchSummaryResponse struct {
	UUID                    uuid.UUID `json:"uuid"`
	CampaignUUID            uuid.UUID `json:"campaignUuid"`
	Title                   string    `json:"title"`
	BriefInitialDescription string    `json:"briefInitialDescription"`
	BriefFinalDescription   *string   `json:"briefFinalDescription,omitempty"`
	IsPublic                bool      `json:"isPublic"`
	GameScheduledAt         string    `json:"gameScheduledAt"`
	GameStartAt             *string   `json:"gameStartAt,omitempty"`
	StoryStartAt            string    `json:"storyStartAt"`
	StoryEndAt              *string   `json:"storyEndAt,omitempty"`
	MyEnrollmentStatus      *string   `json:"myEnrollmentStatus,omitempty"`
	CreatedAt               string    `json:"createdAt"`
	UpdatedAt               string    `json:"updatedAt"`
}

func ToSummaryResponse(m *matchEntity.Summary) MatchSummaryResponse {
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
