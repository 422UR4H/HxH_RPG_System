package match

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type ListMatchesResponseBody struct {
	Matches []MatchSummaryResponse `json:"matches"`
}

type ListMatchesResponse struct {
	Body ListMatchesResponseBody `json:"body"`
}

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

func ListMatchesHandler(
	uc domainMatch.IListMatches,
) func(context.Context, *struct{}) (*ListMatchesResponse, error) {

	return func(ctx context.Context, _ *struct{}) (*ListMatchesResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, errors.New("failed to get userID in context")
		}

		matches, err := uc.ListMatches(masterUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		responses := make([]MatchSummaryResponse, 0, len(matches))
		for _, m := range matches {
			var storyEndAtStr *string
			if m.StoryEndAt != nil {
				formatted := m.StoryEndAt.Format("2006-01-02")
				storyEndAtStr = &formatted
			}

			responses = append(responses, MatchSummaryResponse{
				UUID:             m.UUID,
				CampaignUUID:     m.CampaignUUID,
				Title:            m.Title,
				BriefDescription: m.BriefDescription,
				StoryStartAt:     m.StoryStartAt.Format("2006-01-02"),
				StoryEndAt:       storyEndAtStr,
				CreatedAt:        m.CreatedAt.Format(http.TimeFormat),
				UpdatedAt:        m.UpdatedAt.Format(http.TimeFormat),
			})
		}

		return &ListMatchesResponse{
			Body: ListMatchesResponseBody{
				Matches: responses,
			},
		}, nil
	}
}
