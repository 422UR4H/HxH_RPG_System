// internal/app/api/map/get_map_test.go
package mapapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/map"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

var _ mapuc.IGetMap = (*mockGetMap)(nil) // compile-time interface check

// mockGetMap satisfies the updated IGetMap interface.
type mockGetMap struct {
	result   *entity.TacticalMap
	isMaster bool
	err      error
}

func (m *mockGetMap) GetMap(_ context.Context, _ uuid.UUID, _ uuid.UUID) (*entity.TacticalMap, bool, error) {
	return m.result, m.isMaster, m.err
}

// newTestMapWithWalls returns a TacticalMap that has one secret_door wall and one
// invisible piece — useful for asserting the filtering invariants.
func newTestMapWithWalls(campaignID uuid.UUID) *entity.TacticalMap {
	now := time.Now().UTC()
	sub := entity.DoorSubtypeBasic
	return &entity.TacticalMap{
		ID:          uuid.New(),
		CampaignID:  campaignID,
		Name:        "Test Map",
		Description: "desc",
		Grid:        entity.DefaultGrid(),
		Walls: []entity.WallSegment{
			{
				ID:          "secret1",
				WallType:    entity.WallTypeSecretDoor,
				DoorSubtype: &sub,
				Material:    entity.WallMaterialStone,
				P1:          [2]float64{0, 0},
				P2:          [2]float64{64, 0},
				Open:        true,
				Locked:      true,
				HP:          80,
				MaxHP:       100,
				Sense:       entity.SenseFull,
				Direction:   entity.WallDirectionBoth,
				Revealed:    false,
			},
			{
				ID:        "wall1",
				WallType:  entity.WallTypeWall,
				Material:  entity.WallMaterialStone,
				P1:        [2]float64{0, 64},
				P2:        [2]float64{64, 64},
				Sense:     entity.SenseFull,
				Direction: entity.WallDirectionBoth,
			},
		},
		Pieces: []entity.Piece{
			{ID: "p1", CharacterID: "c1", Visible: false}, // invisible — must be hidden from player
			{ID: "p2", CharacterID: "c2", Visible: true},  // visible — shown to player
		},
		Decorations: []entity.Decoration{},
		Items:       []entity.MapItem{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func registerGetMapHandler(t *testing.T, mock mapuc.IGetMap) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/maps/{map_id}",
	}, mapapi.GetMapHandler(mock))
	return api
}

// TestGetMap_PlayerGetsMaskedSecretDoor asserts the two non-negotiable guarantees:
//   - A non-master player never receives the real secret_door wall_type.
//   - A non-master player never receives invisible pieces.
func TestGetMap_PlayerGetsMaskedSecretDoor(t *testing.T) {
	campaignID := uuid.New()
	userID := uuid.New()
	m := newTestMapWithWalls(campaignID)

	mock := &mockGetMap{result: m, isMaster: false, err: nil}
	api := registerGetMapHandler(t, mock)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.GetCtx(ctx, "/maps/"+m.ID.String())

	if resp.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	mapObj, ok := body["map"].(map[string]any)
	if !ok {
		t.Fatalf("response missing 'map' key, got: %v", body)
	}

	// Walls: secret_door must be masked as "wall".
	walls, ok := mapObj["walls"].([]any)
	if !ok {
		t.Fatalf("walls missing from response: %v", mapObj)
	}
	for _, w := range walls {
		wall, _ := w.(map[string]any)
		if wall["id"] == "secret1" {
			if wall["wall_type"] != "wall" {
				t.Errorf("secret door must be masked as 'wall' for non-master, got %q", wall["wall_type"])
			}
			// door_subtype must not leak
			if _, hasSubtype := wall["door_subtype"]; hasSubtype && wall["door_subtype"] != nil {
				t.Errorf("masked secret door must not expose door_subtype, got %v", wall["door_subtype"])
			}
			// open and locked must be false
			if open, _ := wall["open"].(bool); open {
				t.Errorf("masked secret door must not expose open=true")
			}
			if locked, _ := wall["locked"].(bool); locked {
				t.Errorf("masked secret door must not expose locked=true")
			}
		}
	}

	// Pieces: invisible piece (p1) must not appear; visible piece (p2) must appear.
	pieces, ok := mapObj["pieces"].([]any)
	if !ok {
		t.Fatalf("pieces missing from response: %v", mapObj)
	}
	pieceIDs := make(map[string]bool)
	for _, p := range pieces {
		if pm, ok := p.(map[string]any); ok {
			if id, ok := pm["id"].(string); ok {
				pieceIDs[id] = true
			}
		}
	}
	if pieceIDs["p1"] {
		t.Error("invisible piece p1 must not appear for non-master player")
	}
	if !pieceIDs["p2"] {
		t.Error("visible piece p2 must appear for non-master player")
	}
}

// TestGetMap_MasterSeesRealSecretDoor asserts the master receives the real secret_door
// type unmasked, and all pieces including invisible ones.
func TestGetMap_MasterSeesRealSecretDoor(t *testing.T) {
	campaignID := uuid.New()
	masterID := uuid.New()
	m := newTestMapWithWalls(campaignID)

	mock := &mockGetMap{result: m, isMaster: true, err: nil}
	api := registerGetMapHandler(t, mock)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, masterID)
	resp := api.GetCtx(ctx, "/maps/"+m.ID.String())

	if resp.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	mapObj := body["map"].(map[string]any)

	// Walls: secret_door must remain secret_door.
	walls := mapObj["walls"].([]any)
	foundSecret := false
	for _, w := range walls {
		wall, _ := w.(map[string]any)
		if wall["id"] == "secret1" {
			foundSecret = true
			if wall["wall_type"] != "secret_door" {
				t.Errorf("master must see real 'secret_door', got %q", wall["wall_type"])
			}
		}
	}
	if !foundSecret {
		t.Error("master response must include the secret_door wall")
	}

	// Pieces: master sees all pieces including invisible ones.
	pieces := mapObj["pieces"].([]any)
	pieceIDs := make(map[string]bool)
	for _, p := range pieces {
		if pm, ok := p.(map[string]any); ok {
			if id, ok := pm["id"].(string); ok {
				pieceIDs[id] = true
			}
		}
	}
	if !pieceIDs["p1"] {
		t.Error("master must see invisible piece p1")
	}
	if !pieceIDs["p2"] {
		t.Error("master must see visible piece p2")
	}
}

// TestGetMap_NotFound returns 404 when the use case returns ErrMapNotFound.
func TestGetMap_NotFound(t *testing.T) {
	userID := uuid.New()
	mock := &mockGetMap{result: nil, isMaster: false, err: mapuc.ErrMapNotFound}
	api := registerGetMapHandler(t, mock)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.GetCtx(ctx, "/maps/"+uuid.New().String())

	if resp.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", resp.Code, resp.Body.String())
	}
}
