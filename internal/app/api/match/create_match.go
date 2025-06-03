package match

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type CreateMatchRequestBody struct {
	CampaignUUID            uuid.UUID `json:"campaign_uuid" required:"true" doc:"UUID of the campaign this match is based on"`
	Title                   string    `json:"title" required:"true" maxLength:"32" doc:"Title of the match"`
	BriefInitialDescription string    `json:"brief_initial_description" maxLength:"64" doc:"Brief description of the match"`
	Description             string    `json:"description" doc:"Full description of the match"`
	IsPublic                bool      `json:"is_public" default:"true" doc:"If the match is public or private"`
	GameStartAt             string    `json:"game_start_at" required:"true" doc:"Date and time when the game starts (ISO 8601)"`
	StoryStartAt            string    `json:"story_start_at" required:"true" doc:"Date when the match story starts (YYYY-MM-DD)"`
}

type CreateMatchRequest struct {
	Body CreateMatchRequestBody `json:"body"`
}

type CreateMatchResponseBody struct {
	Match MatchResponse `json:"match"`
}

type CreateMatchResponse struct {
	Body   CreateMatchResponseBody `json:"body"`
	Status int                     `json:"status"`
}

type MatchResponse struct {
	UUID                    uuid.UUID `json:"uuid"`
	CampaignUUID            uuid.UUID `json:"campaign_uuid"`
	Title                   string    `json:"title"`
	BriefInitialDescription string    `json:"brief_initial_description"`
	BriefFinalDescription   *string   `json:"brief_final_description,omitempty"`
	Description             string    `json:"description"`
	IsPublic                bool      `json:"is_public"`
	GameStartAt             string    `json:"game_start_at"`
	StoryStartAt            string    `json:"story_start_at"`
	StoryEndAt              *string   `json:"story_end_at,omitempty"`
	CreatedAt               string    `json:"created_at"`
	UpdatedAt               string    `json:"updated_at"`
}

func CreateMatchHandler(
	uc domainMatch.ICreateMatch,
) func(context.Context, *CreateMatchRequest) (*CreateMatchResponse, error) {

	return func(ctx context.Context, req *CreateMatchRequest) (*CreateMatchResponse, error) {
		userUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, errors.New("failed to get userID in context")
		}

		storyStartAt, err := time.Parse("2006-01-02", req.Body.StoryStartAt)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(
				"invalid story_start_at date format, use YYYY-MM-DD")
		}

		gameStartAt, err := time.Parse(time.RFC3339, req.Body.GameStartAt)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity(
				"invalid game_start_at date format, use ISO 8601. E.g. 2026-06-15T19:30:00Z")
		}

		input := &domainMatch.CreateMatchInput{
			MasterUUID:              userUUID,
			CampaignUUID:            req.Body.CampaignUUID,
			Title:                   req.Body.Title,
			BriefInitialDescription: req.Body.BriefInitialDescription,
			Description:             req.Body.Description,
			IsPublic:                req.Body.IsPublic,
			GameStartAt:             gameStartAt,
			StoryStartAt:            storyStartAt,
		}
		match, err := uc.CreateMatch(ctx, input)
		if err != nil {
			switch {
			case errors.Is(err, campaign.ErrCampaignNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainMatch.ErrNotCampaignOwner):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, domain.ErrValidation):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		response := MatchResponse{
			UUID:                    match.UUID,
			CampaignUUID:            match.CampaignUUID,
			Title:                   match.Title,
			BriefInitialDescription: match.BriefInitialDescription,
			BriefFinalDescription:   match.BriefFinalDescription,
			Description:             match.Description,
			IsPublic:                match.IsPublic,
			GameStartAt:             match.GameStartAt.Format(time.RFC3339),
			StoryStartAt:            match.StoryStartAt.Format("2006-01-02"),
			CreatedAt:               match.CreatedAt.Format(http.TimeFormat),
			UpdatedAt:               match.UpdatedAt.Format(http.TimeFormat),
		}
		return &CreateMatchResponse{
			Body: CreateMatchResponseBody{
				Match: response,
			},
			Status: http.StatusCreated,
		}, nil
	}
}
