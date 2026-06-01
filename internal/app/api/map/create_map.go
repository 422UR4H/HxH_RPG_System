// internal/app/api/map/create_map.go
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

type CreateMapRequestBody struct {
	Name        string             `json:"name" required:"true" doc:"Name of the map"`
	Description string             `json:"description" required:"false" doc:"Description of the map"`
	Grid        *entity.GridShape  `json:"grid" required:"false" doc:"Grid configuration; defaults to 25x25 64px if omitted"`
}

type CreateMapRequest struct {
	CampaignID uuid.UUID            `path:"campaign_id"`
	Body       CreateMapRequestBody `json:"body"`
}

type CreateMapResponseBody struct {
	Map MapResponse `json:"map"`
}
type CreateMapResponse struct {
	Body   CreateMapResponseBody
	Status int
}

func CreateMapHandler(uc mapuc.ICreateMap) func(context.Context, *CreateMapRequest) (*CreateMapResponse, error) {
	return func(ctx context.Context, req *CreateMapRequest) (*CreateMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		m, err := uc.CreateMap(ctx, &mapuc.CreateMapInput{
			RequesterID: userID,
			CampaignID:  req.CampaignID,
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
		return &CreateMapResponse{Body: CreateMapResponseBody{Map: toMapResponse(m)}, Status: http.StatusCreated}, nil
	}
}
