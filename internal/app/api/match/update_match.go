package match

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type UpdateMatchRequestBody struct {
	Title                   *string `json:"title,omitempty" doc:"New title (5-32 characters)"`
	BriefInitialDescription *string `json:"brief_initial_description,omitempty" doc:"New brief description (max 255 characters)"`
	Description             *string `json:"description,omitempty" doc:"New full description"`
	IsPublic                *bool   `json:"is_public,omitempty" doc:"New public/private flag"`
	GameScheduledAt         *string `json:"game_scheduled_at,omitempty" doc:"ISO 8601 date-time"`
	StoryStartAt            *string `json:"story_start_at,omitempty" doc:"YYYY-MM-DD"`
}

type UpdateMatchRequest struct {
	UUID uuid.UUID              `path:"uuid" required:"true" doc:"UUID of the match to update"`
	Body UpdateMatchRequestBody `json:"body"`
}

type UpdateMatchResponseBody struct {
	Match MatchResponse `json:"match"`
}

type UpdateMatchResponse struct {
	Body UpdateMatchResponseBody `json:"body"`
}

func UpdateMatchHandler(
	uc matchUC.IUpdateMatch,
) func(context.Context, *UpdateMatchRequest) (*UpdateMatchResponse, error) {

	return func(ctx context.Context, req *UpdateMatchRequest) (*UpdateMatchResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		input := &matchUC.UpdateMatchInput{
			MatchUUID:               req.UUID,
			MasterUUID:              userUUID,
			Title:                   req.Body.Title,
			BriefInitialDescription: req.Body.BriefInitialDescription,
			Description:             req.Body.Description,
			IsPublic:                req.Body.IsPublic,
		}

		if req.Body.GameScheduledAt != nil {
			t, err := time.Parse(time.RFC3339, *req.Body.GameScheduledAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(
					"invalid game_scheduled_at date format, use ISO 8601. E.g. 2026-06-15T19:30:00Z")
			}
			input.GameScheduledAt = &t
		}
		if req.Body.StoryStartAt != nil {
			t, err := time.Parse("2006-01-02", *req.Body.StoryStartAt)
			if err != nil {
				return nil, huma.Error422UnprocessableEntity(
					"invalid story_start_at date format, use YYYY-MM-DD")
			}
			input.StoryStartAt = &t
		}

		m, err := uc.Update(ctx, input)
		if err != nil {
			switch {
			case errors.Is(err, matchUC.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, matchUC.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, matchUC.ErrMatchAlreadyStarted),
				errors.Is(err, matchUC.ErrMatchAlreadyFinished):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			case errors.Is(err, domain.ErrValidation):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		var gameStartAtStr *string
		if m.GameStartAt != nil {
			s := m.GameStartAt.Format(time.RFC3339)
			gameStartAtStr = &s
		}
		var storyEndAtStr *string
		if m.StoryEndAt != nil {
			s := m.StoryEndAt.Format("2006-01-02")
			storyEndAtStr = &s
		}

		response := MatchResponse{
			UUID:                    m.UUID,
			MasterUUID:              m.MasterUUID,
			CampaignUUID:            m.CampaignUUID,
			Title:                   m.Title,
			BriefInitialDescription: m.BriefInitialDescription,
			BriefFinalDescription:   m.BriefFinalDescription,
			Description:             m.Description,
			IsPublic:                m.IsPublic,
			GameScheduledAt:         m.GameScheduledAt.Format(time.RFC3339),
			GameStartAt:             gameStartAtStr,
			StoryStartAt:            m.StoryStartAt.Format("2006-01-02"),
			StoryEndAt:              storyEndAtStr,
			CreatedAt:               m.CreatedAt.Format(http.TimeFormat),
			UpdatedAt:               m.UpdatedAt.Format(http.TimeFormat),
		}
		return &UpdateMatchResponse{
			Body: UpdateMatchResponseBody{Match: response},
		}, nil
	}
}
