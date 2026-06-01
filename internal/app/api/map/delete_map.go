// internal/app/api/map/delete_map.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type DeleteMapRequest struct {
	MapID uuid.UUID `path:"map_id"`
}

type DeleteMapResponse struct {
	Status int
}

func DeleteMapHandler(uc mapuc.IDeleteMap) func(context.Context, *DeleteMapRequest) (*DeleteMapResponse, error) {
	return func(ctx context.Context, req *DeleteMapRequest) (*DeleteMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		err := uc.DeleteMap(ctx, userID, req.MapID)
		if err != nil {
			switch {
			case errors.Is(err, mapuc.ErrNotMapMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, mapuc.ErrMapNotFound):
				return nil, huma.Error404NotFound(err.Error())
			default:
				return nil, huma.Error500InternalServerError(err.Error())
			}
		}
		return &DeleteMapResponse{Status: http.StatusNoContent}, nil
	}
}
