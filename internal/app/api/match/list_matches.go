package match

import (
	"context"
	"errors"

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

func ListMatchesHandler(
	uc domainMatch.IListMatches,
) func(context.Context, *struct{}) (*ListMatchesResponse, error) {

	return func(ctx context.Context, _ *struct{}) (*ListMatchesResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, errors.New("failed to get userID in context")
		}

		matches, err := uc.ListMatches(ctx, masterUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}

		responses := make([]MatchSummaryResponse, 0, len(matches))
		for _, m := range matches {
			responses = append(responses, ToSummaryResponse(m))
		}

		return &ListMatchesResponse{
			Body: ListMatchesResponseBody{
				Matches: responses,
			},
		}, nil
	}
}
