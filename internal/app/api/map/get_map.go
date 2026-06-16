// internal/app/api/map/get_map.go
package mapapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	matchservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
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
// Non-master player (lobby path — no live match session at REST):
//   - Unrevealed secret doors are masked as plain walls (type="wall", door subtype/open/locked cleared).
//   - Pieces with visible=false are removed.
//
// LOS-at-REST is deferred to the WS layer (live match) which has exact per-player polygons.
// TODO(10-D): wire live fog state here once a /maps/:id REST endpoint is called mid-match.
func applyRoleFilter(m *mapentity.TacticalMap, isMaster bool) {
	if isMaster {
		return
	}

	// Mask unrevealed secret doors as plain walls.
	for i := range m.Walls {
		if m.Walls[i].WallType == mapentity.WallTypeSecretDoor && !m.Walls[i].Revealed {
			m.Walls[i] = matchservice.MaskSecretDoorForPlayer(m.Walls[i])
		}
	}

	// Remove invisible pieces (visible=false pieces are master-only).
	visible := m.Pieces[:0]
	for _, p := range m.Pieces {
		if p.Visible {
			visible = append(visible, p)
		}
	}
	m.Pieces = visible
}
