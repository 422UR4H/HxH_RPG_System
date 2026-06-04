// internal/app/api/matchmap/get.go
package matchmapapi

import (
	"context"
	"net/http"

	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetMatchMapRequest struct {
	MatchUUID uuid.UUID `path:"match_uuid"`
}

type GetMatchMapResponseBody struct {
	MatchMap MatchMapResponse `json:"match_map"`
}

type GetMatchMapResponse struct {
	Body   *GetMatchMapResponseBody
	Status int
}

func GetMatchMapHandler(uc matchmapuc.IGetMatchMap) func(context.Context, *GetMatchMapRequest) (*GetMatchMapResponse, error) {
	return func(ctx context.Context, req *GetMatchMapRequest) (*GetMatchMapResponse, error) {
		mm, err := uc.Get(ctx, req.MatchUUID)
		if err != nil {
			return nil, huma.Error500InternalServerError(err.Error())
		}
		if mm == nil {
			return &GetMatchMapResponse{Status: http.StatusNoContent}, nil
		}
		body := &GetMatchMapResponseBody{MatchMap: toMatchMapResponse(mm)}
		return &GetMatchMapResponse{Body: body, Status: http.StatusOK}, nil
	}
}
