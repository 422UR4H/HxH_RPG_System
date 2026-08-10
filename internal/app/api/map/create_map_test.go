// internal/app/api/map/create_map_test.go
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

func newTestMap(campaignID uuid.UUID, name string) *entity.TacticalMap {
	now := time.Now().UTC()
	return &entity.TacticalMap{
		ID:          uuid.New(),
		CampaignID:  campaignID,
		Name:        name,
		Description: "test description",
		Grid:        entity.DefaultGrid(),
		Pieces:      []entity.Piece{},
		Walls:       []entity.WallSegment{},
		// Decorations and Items are not settable via the create/update request bodies
		// (no request DTO field for either) — they only ever come back on the response,
		// so a fixed non-empty entry here is what exercises toDecorationResponse and
		// toMapItemResponse in TestCreateMapHandler_Success below.
		Decorations: []entity.Decoration{
			{
				ID: "deco-1", URL: "https://example.com/deco.png",
				X: 3.5, Y: 4.5, Width: 32.5, Height: 48.5,
				Rotation: 90, ZOrder: 2, Opacity: 0.66,
			},
		},
		Items: []entity.MapItem{
			{ID: "item-1", ItemDefID: "sword-basic"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestCreateMapHandler_Success(t *testing.T) {
	userID := uuid.New()
	campaignID := uuid.New()
	mapName := "Test Map"

	_, api := humatest.New(t)

	mock := &mockCreateMap{
		result: newTestMap(campaignID, mapName),
		err:    nil,
	}
	handler := mapapi.CreateMapHandler(mock)

	huma.Register(api, huma.Operation{
		Method:        http.MethodPost,
		Path:          "/campaigns/{campaign_id}/maps",
		DefaultStatus: http.StatusCreated,
	}, handler)

	// Every grid/bg/pieces field gets a distinct, non-zero, non-default value so a
	// dropped-field bug in toEntityGridShape/toEntityBgImage/toEntityPiece (or in their
	// response-side counterparts) can't hide behind a zero-value coincidence.
	body := map[string]any{
		"name":        mapName,
		"description": "test description",
		"grid": map[string]any{
			"kind":      "hex",
			"cols":      42,
			"rows":      37,
			"cellSize":  77.5,
			"skewRatio": 0.63,
			"rotation":  12.25,
			"color":     "#abcdef",
			"opacity":   0.42,
			"lineStyle": "dashed",
		},
		"bg": map[string]any{
			"url":      "https://example.com/bg.png",
			"x":        11.5,
			"y":        22.5,
			"width":    800.25,
			"height":   600.75,
			"rotation": 5.5,
			"opacity":  0.77,
		},
		"pieces": []map[string]any{
			{
				"id":          "piece-1",
				"characterId": "char-uuid-1",
				"coord": map[string]any{
					"slot": map[string]any{"kind": "square", "col": 3, "row": 4},
					"z":    1.5,
				},
				"visible": true,
			},
		},
	}
	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.PostCtx(ctx, "/campaigns/"+campaignID.String()+"/maps", body)

	if resp.Code != http.StatusCreated {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusCreated, resp.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	mapObj, ok := result["map"].(map[string]any)
	if !ok {
		t.Fatalf("response missing 'map' key, got: %v", result)
	}
	if mapObj["name"] != mapName {
		t.Errorf("got name %v, want %q", mapObj["name"], mapName)
	}
	if mapObj["campaignId"] != campaignID.String() {
		t.Errorf("got campaignId %v, want %v", mapObj["campaignId"], campaignID.String())
	}

	// --- grid: every field, toEntityGridShape + GridShapeResponse round trip ---
	grid, ok := mapObj["grid"].(map[string]any)
	if !ok {
		t.Fatalf("response missing 'grid', got: %v", mapObj)
	}
	wantGrid := map[string]any{
		"kind": "hex", "cols": float64(42), "rows": float64(37),
		"cellSize": 77.5, "skewRatio": 0.63, "rotation": 12.25,
		"color": "#abcdef", "opacity": 0.42, "lineStyle": "dashed",
	}
	for field, want := range wantGrid {
		if got := grid[field]; got != want {
			t.Errorf("grid.%s = %v, want %v", field, got, want)
		}
	}

	// --- bg: every field, toEntityBgImage + toBgImageResponse round trip ---
	bg, ok := mapObj["bg"].(map[string]any)
	if !ok {
		t.Fatalf("response missing 'bg', got: %v", mapObj)
	}
	wantBg := map[string]any{
		"url": "https://example.com/bg.png", "x": 11.5, "y": 22.5,
		"width": 800.25, "height": 600.75, "rotation": 5.5, "opacity": 0.77,
	}
	for field, want := range wantBg {
		if got := bg[field]; got != want {
			t.Errorf("bg.%s = %v, want %v", field, got, want)
		}
	}

	// --- pieces: every field, toEntityPiece + toPieceResponse round trip ---
	pieces, ok := mapObj["pieces"].([]any)
	if !ok || len(pieces) != 1 {
		t.Fatalf("response pieces = %v, want exactly 1", mapObj["pieces"])
	}
	piece, ok := pieces[0].(map[string]any)
	if !ok {
		t.Fatalf("piece[0] is not an object: %v", pieces[0])
	}
	if piece["id"] != "piece-1" {
		t.Errorf("piece.id = %v, want %q", piece["id"], "piece-1")
	}
	if piece["characterId"] != "char-uuid-1" {
		t.Errorf("piece.characterId = %v, want %q", piece["characterId"], "char-uuid-1")
	}
	if piece["visible"] != true {
		t.Errorf("piece.visible = %v, want true", piece["visible"])
	}
	coord, ok := piece["coord"].(map[string]any)
	if !ok {
		t.Fatalf("piece.coord is not an object: %v", piece["coord"])
	}
	if coord["z"] != 1.5 {
		t.Errorf("piece.coord.z = %v, want 1.5", coord["z"])
	}
	slot, ok := coord["slot"].(map[string]any)
	if !ok {
		t.Fatalf("piece.coord.slot is not an object: %v", coord["slot"])
	}
	wantSlot := map[string]any{"kind": "square", "col": float64(3), "row": float64(4)}
	for field, want := range wantSlot {
		if got := slot[field]; got != want {
			t.Errorf("piece.coord.slot.%s = %v, want %v", field, got, want)
		}
	}

	// --- decorations/items: not settable via the request, only exercised on the
	// response side (toDecorationResponse/toMapItemResponse) via the fixture map
	// newTestMap seeds — see its comment. ---
	decorations, ok := mapObj["decorations"].([]any)
	if !ok || len(decorations) != 1 {
		t.Fatalf("response decorations = %v, want exactly 1", mapObj["decorations"])
	}
	deco, ok := decorations[0].(map[string]any)
	if !ok {
		t.Fatalf("decoration[0] is not an object: %v", decorations[0])
	}
	wantDeco := map[string]any{
		"id": "deco-1", "url": "https://example.com/deco.png",
		"x": 3.5, "y": 4.5, "width": 32.5, "height": 48.5,
		"rotation": float64(90), "zOrder": float64(2), "opacity": 0.66,
	}
	for field, want := range wantDeco {
		if got := deco[field]; got != want {
			t.Errorf("decoration.%s = %v, want %v", field, got, want)
		}
	}

	items, ok := mapObj["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("response items = %v, want exactly 1", mapObj["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("item[0] is not an object: %v", items[0])
	}
	if item["id"] != "item-1" {
		t.Errorf("item.id = %v, want %q", item["id"], "item-1")
	}
	if item["itemDefId"] != "sword-basic" {
		t.Errorf("item.itemDefId = %v, want %q", item["itemDefId"], "sword-basic")
	}
}

func TestCreateMapHandler_NotMaster_Returns403(t *testing.T) {
	userID := uuid.New()
	campaignID := uuid.New()

	_, api := humatest.New(t)

	mock := &mockCreateMap{
		result: nil,
		err:    mapuc.ErrNotMapMaster,
	}
	handler := mapapi.CreateMapHandler(mock)

	huma.Register(api, huma.Operation{
		Method:        http.MethodPost,
		Path:          "/campaigns/{campaign_id}/maps",
		DefaultStatus: http.StatusCreated,
	}, handler)

	body := map[string]any{
		"name":        "Forbidden Map",
		"description": "",
	}
	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.PostCtx(ctx, "/campaigns/"+campaignID.String()+"/maps", body)

	if resp.Code != http.StatusForbidden {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusForbidden, resp.Body.String())
	}
}
