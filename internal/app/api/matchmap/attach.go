// internal/app/api/matchmap/attach.go
package matchmapapi

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type AttachMatchMapRequestBody struct {
	MapUUID uuid.UUID `json:"map_uuid" required:"true" doc:"UUID of the map to attach"`
}

type AttachMatchMapRequest struct {
	MatchUUID uuid.UUID                 `path:"match_uuid"`
	Body      AttachMatchMapRequestBody
}

type AttachMatchMapResponseBody struct {
	MatchMap MatchMapResponse `json:"match_map"`
}

type AttachMatchMapResponse struct {
	Body AttachMatchMapResponseBody
}

func AttachMatchMapHandler(uc matchmapuc.IAttachMatchMap) func(context.Context, *AttachMatchMapRequest) (*AttachMatchMapResponse, error) {
	return func(ctx context.Context, req *AttachMatchMapRequest) (*AttachMatchMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		mm, err := uc.Attach(ctx, &matchmapuc.AttachMatchMapInput{
			RequesterUUID: userID,
			MatchUUID:     req.MatchUUID,
			MapUUID:       req.Body.MapUUID,
		})
		if err != nil {
			switch {
			case errors.Is(err, matchmapuc.ErrNotMatchMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, matchmapuc.ErrMatchNotFound),
				errors.Is(err, matchmapuc.ErrMapNotFound):
				return nil, huma.Error404NotFound(err.Error())
			case errors.Is(err, matchmapuc.ErrMatchAlreadyStarted):
				return nil, huma.Error422UnprocessableEntity(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &AttachMatchMapResponse{Body: AttachMatchMapResponseBody{MatchMap: toMatchMapResponse(mm)}}, nil
	}
}
