package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/apierr"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/google/uuid"
)

type ListMatchesResponseBody struct {
	Matches []MatchSummaryResponse `json:"matches"`
}

type ListMatchesResponse struct {
	Body ListMatchesResponseBody `json:"body"`
}

func ListMatchesHandler(
	uc matchUC.IListMatches,
) func(context.Context, *struct{}) (*ListMatchesResponse, error) {

	return func(ctx context.Context, _ *struct{}) (*ListMatchesResponse, error) {
		masterUUID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, apierr.NoUser()
		}

		matches, err := uc.ListMatches(ctx, masterUUID)
		if err != nil {
			return nil, apierr.Internal(err)
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
