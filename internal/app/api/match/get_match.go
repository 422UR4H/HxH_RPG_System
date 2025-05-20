package match

import (
	"context"
	"errors"
	"net/http"

	domainMatch "github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetMatchRequest struct {
	UUID uuid.UUID `path:"uuid" required:"true" doc:"UUID of the match to retrieve"`
}

type GetMatchResponseBody struct {
	Match MatchResponse `json:"match"`
}

type GetMatchResponse struct {
	Body GetMatchResponseBody `json:"body"`
}

func GetMatchHandler(
	uc domainMatch.IGetMatch,
) func(context.Context, *GetMatchRequest) (*GetMatchResponse, error) {

	return func(ctx context.Context, req *GetMatchRequest) (*GetMatchResponse, error) {
		match, err := uc.GetMatch(req.UUID)
		if err != nil {
			switch {
			case errors.Is(err, domainMatch.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		var storyEndAtStr *string
		if match.StoryEndAt != nil {
			formattedDate := match.StoryEndAt.Format("2006-01-02")
			storyEndAtStr = &formattedDate
		}

		response := MatchResponse{
			UUID:             match.UUID,
			CampaignUUID:     match.CampaignUUID,
			Title:            match.Title,
			BriefDescription: match.BriefDescription,
			Description:      match.Description,
			StoryStartAt:     match.StoryStartAt.Format("2006-01-02"),
			StoryEndAt:       storyEndAtStr,
			CreatedAt:        match.CreatedAt.Format(http.TimeFormat),
			UpdatedAt:        match.UpdatedAt.Format(http.TimeFormat),
		}

		return &GetMatchResponse{
			Body: GetMatchResponseBody{
				Match: response,
			},
		}, nil
	}
}
