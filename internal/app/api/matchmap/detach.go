// internal/app/api/matchmap/detach.go
package matchmapapi

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type DetachMatchMapRequest struct {
	MatchUUID uuid.UUID `path:"match_uuid"`
}

type DetachMatchMapResponse struct{}

func DetachMatchMapHandler(uc matchmapuc.IDetachMatchMap) func(context.Context, *DetachMatchMapRequest) (*DetachMatchMapResponse, error) {
	return func(ctx context.Context, req *DetachMatchMapRequest) (*DetachMatchMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		err := uc.Detach(ctx, &matchmapuc.DetachMatchMapInput{
			RequesterUUID: userID,
			MatchUUID:     req.MatchUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, matchmapuc.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, matchmapuc.ErrMatchNotFound),
				errors.Is(err, matchmapuc.ErrMatchMapNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, matchmapuc.ErrMatchAlreadyStarted):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &DetachMatchMapResponse{}, nil
	}
}
