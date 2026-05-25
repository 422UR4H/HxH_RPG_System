package match

import (
	"context"
	"errors"
	"net/http"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchUC "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type DeleteMatchRequest struct {
	UUID string `path:"uuid" required:"true"`
}

type DeleteMatchResponse struct {
	Status int
}

func DeleteMatchHandler(
	uc matchUC.IDeleteMatch,
) func(context.Context, *DeleteMatchRequest) (*DeleteMatchResponse, error) {
	return func(ctx context.Context, req *DeleteMatchRequest) (*DeleteMatchResponse, error) {
		userUUID, ok := ctx.Value(apiAuth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID in context")
		}

		matchUUID, err := uuid.Parse(req.UUID)
		if err != nil {
			return nil, huma.Error400BadRequest("invalid uuid")
		}

		err = uc.Delete(ctx, &matchUC.DeleteMatchInput{
			MatchUUID:  matchUUID,
			MasterUUID: userUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, matchUC.ErrMatchNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, matchUC.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, matchUC.ErrMatchAlreadyStarted):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}

		return &DeleteMatchResponse{Status: http.StatusNoContent}, nil
	}
}
