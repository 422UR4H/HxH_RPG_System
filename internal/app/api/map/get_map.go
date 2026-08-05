// internal/app/api/map/get_map.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
)

type GetMapRequest struct {
	MapID uuid.UUID `path:"map_id"`
}

type GetMapResponseBody struct {
	Map MapResponse `json:"map"`
}
type GetMapResponse struct {
	Body   GetMapResponseBody
	Status int
}

func GetMapHandler(uc mapuc.IGetMap) func(context.Context, *GetMapRequest) (*GetMapResponse, error) {
	return func(ctx context.Context, req *GetMapRequest) (*GetMapResponse, error) {
		userID, ok := ctx.Value(auth.UserIDKey).(uuid.UUID)
		if !ok {
			return nil, huma.Error500InternalServerError("failed to get userID")
		}

		m, isMaster, err := uc.GetMap(ctx, req.MapID, userID)
		if err != nil {
			if errors.Is(err, mapuc.ErrMapNotFound) {
				return nil, huma.Error404NotFound(err.Error())
			}
			return nil, huma.Error500InternalServerError(err.Error())
		}

		applyRoleFilter(m, isMaster)

		return &GetMapResponse{Body: GetMapResponseBody{Map: toMapResponse(m)}, Status: http.StatusOK}, nil
	}
}

// applyRoleFilter mutates the map in-place to remove information the viewer must not see.
//
// Master: no filtering — receives the full unmasked board.
//
// Non-master: receives the map SHELL only — grid, background, fog mode — with no pieces
// and no walls at all.
//
// That is deliberate, and it is not over-caution. GET /maps/:id is served by the REST
// process (cmd/api). Everything needed to decide what a player may see — the per-player
// visibility polygons — lives in MatchSession, in the RAM of a DIFFERENT process
// (cmd/game), and is recomputed on every move. REST cannot filter by line of sight and
// never could, so it used to answer with the whole board: any player could read every
// wall and every character position straight off the endpoint with their own token.
//
// The game server already computes exactly this, per player, and pushes it over
// map_full_state — in the lobby as well as in a live match. It is the only place that
// can, so it is the only place that does.
func applyRoleFilter(m *mapentity.TacticalMap, isMaster bool) {
	if isMaster {
		return
	}
	// Empty slices, not nil: the JSON stays `[]` instead of flipping to `null`, so no
	// client has to learn a second shape for "nothing here".
	m.Pieces = []mapentity.Piece{}
	m.Walls = []mapentity.WallSegment{}
}
