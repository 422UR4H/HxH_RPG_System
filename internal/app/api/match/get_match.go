package match

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
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
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, errors.New("failed to get userID in context")
		}

		match, err := uc.GetMatch(ctx, req.UUID, userUUID)
		if err != nil {
			switch {
			case errors.Is(err, domainMatch.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, domainAuth.ErrInsufficientPermissions):
				return nil, huma.Error403Forbidden(err.Error())
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
