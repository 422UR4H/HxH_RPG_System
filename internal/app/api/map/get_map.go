// internal/app/api/map/get_map.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetMapRequest struct {
	MapID uuid.UUID `path:"map_id"`
}

type GetMapResponseBody struct{ MapResponse }
type GetMapResponse struct {
	Body   GetMapResponseBody
	Status int
}

func GetMapHandler(uc mapuc.IGetMap) func(context.Context, *GetMapRequest) (*GetMapResponse, error) {
	return func(ctx context.Context, req *GetMapRequest) (*GetMapResponse, error) {
		m, err := uc.GetMap(ctx, req.MapID)
		if err != nil {
			if errors.Is(err, mapuc.ErrMapNotFound) {
				return nil, huma.Error404NotFound(err.Error())
			}
			return nil, huma.Error500InternalServerError(err.Error())
		}
		return &GetMapResponse{Body: GetMapResponseBody{toMapResponse(m)}, Status: http.StatusOK}, nil
	}
}
