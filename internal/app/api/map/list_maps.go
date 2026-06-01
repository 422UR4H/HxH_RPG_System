// internal/app/api/map/list_maps.go
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

type ListMapsRequest struct {
	CampaignID uuid.UUID `path:"campaign_id"`
}

type ListMapsResponseBody struct {
	Maps []MapResponse `json:"maps"`
}

type ListMapsResponse struct {
	Body   ListMapsResponseBody
	Status int
}

func ListMapsHandler(uc mapuc.IListMaps) func(context.Context, *ListMapsRequest) (*ListMapsResponse, error) {
	return func(ctx context.Context, req *ListMapsRequest) (*ListMapsResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		maps, err := uc.ListMaps(ctx, userID, req.CampaignID)
		if err != nil {
			if errors.Is(err, mapuc.ErrNotMapMaster) {
				return nil, huma.Error403Forbidden(err.Error())
			}
			return nil, huma.Error500InternalServerError(err.Error())
		}

		result := make([]MapResponse, 0, len(maps))
		for _, m := range maps {
			result = append(result, toMapResponse(m))
		}
		return &ListMapsResponse{Body: ListMapsResponseBody{Maps: result}, Status: http.StatusOK}, nil
	}
}
