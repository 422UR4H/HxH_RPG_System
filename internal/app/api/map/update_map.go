// internal/app/api/map/update_map.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type UpdateMapRequestBody struct {
	Name        string            `json:"name" required:"true"`
	Description string            `json:"description" required:"false"`
	Grid        *entity.GridShape `json:"grid" required:"false" doc:"Grid configuration; keeps existing grid if omitted"`
}

type UpdateMapRequest struct {
	MapID uuid.UUID            `path:"map_id"`
	Body  UpdateMapRequestBody `json:"body"`
}

type UpdateMapResponse struct {
	Status int
}

func UpdateMapHandler(uc mapuc.IUpdateMap) func(context.Context, *UpdateMapRequest) (*UpdateMapResponse, error) {
	return func(ctx context.Context, req *UpdateMapRequest) (*UpdateMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		err := uc.UpdateMap(ctx, &mapuc.UpdateMapInput{
			RequesterID: userID,
			MapID:       req.MapID,
			Name:        req.Body.Name,
			Description: req.Body.Description,
			Grid:        req.Body.Grid,
		})
		if err != nil {
			switch {
			case errors.Is(err, mapuc.ErrNotMapMaster):
				return nil, huma.Error403Forbidden(err.Error())
			case errors.Is(err, mapuc.ErrMapNotFound):
				return nil, huma.Error404NotFound(err.Error())
			default:
				return nil, huma.Error422UnprocessableEntity(err.Error())
			}
		}
		return &UpdateMapResponse{Status: http.StatusNoContent}, nil
	}
}
