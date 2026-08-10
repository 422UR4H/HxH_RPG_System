// internal/app/api/map/update_map_test.go
package mapapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	mapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/map"
	mapuc "github.com/422UR4H/HxH_RPG_System/internal/application/map"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func registerUpdateMapHandler(t *testing.T, mock *mockUpdateMap) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	huma.Register(api, huma.Operation{
		Method:        http.MethodPut,
		Path:          "/maps/{map_id}",
		DefaultStatus: http.StatusNoContent,
	}, mapapi.UpdateMapHandler(mock))
	return api
}

// TestUpdateMapHandler_ConvertsAllRequestFields sends a fully populated grid/bg/pieces/
// walls body and asserts every field survives toEntityGridShape/toEntityBgImage/
// toEntityPiece/toEntityWallSegment intact. UpdateMapHandler answers 204 with no body,
// so the only observable is what actually reached the use case — captured on
// mockUpdateMap.received — rather than the HTTP response used for Create/Get.
//
// Every field gets a distinct, non-zero, non-default value (two pieces, two walls —
// one exercising DoorSubtype, the other WindowSubtype) so a dropped-field bug can't
// hide behind a zero-value coincidence, and so the per-item loops are actually proven.
func TestUpdateMapHandler_ConvertsAllRequestFields(t *testing.T) {
	userID := uuid.New()
	mapID := uuid.New()

	mock := &mockUpdateMap{err: nil}
	api := registerUpdateMapHandler(t, mock)

	name := "Updated Map Name"
	body := map[string]any{
		"name":        name,
		"description": "updated description",
		"grid": map[string]any{
			"kind":      "hex",
			"cols":      55,
			"rows":      60,
			"cellSize":  88.5,
			"skewRatio": 0.15,
			"rotation":  270.5,
			"color":     "#fedcba",
			"opacity":   0.33,
			"lineStyle": "dashed",
		},
		"bg": map[string]any{
			"url":      "https://example.com/update-bg.png",
			"x":        100.1,
			"y":        200.2,
			"width":    300.3,
			"height":   400.4,
			"rotation": 15.15,
			"opacity":  0.85,
		},
		"pieces": []map[string]any{
			{
				"id":          "piece-a",
				"characterId": "char-a",
				"coord": map[string]any{
					"slot": map[string]any{"kind": "square", "col": 7, "row": 8},
					"z":    2.5,
				},
				"visible": true,
			},
			{
				"id":          "piece-b",
				"characterId": "char-b",
				"coord": map[string]any{
					"slot": map[string]any{"kind": "hex", "q": 1, "r": -2},
					"z":    0.5,
				},
				"visible": false,
			},
		},
		"walls": []map[string]any{
			{
				"id":          "wall-1",
				"p1":          []float64{1.1, 2.2},
				"p2":          []float64{3.3, 4.4},
				"wallType":    "door",
				"material":    "wood",
				"doorSubtype": "double",
				"move":        true,
				"sense":       "sight",
				"direction":   "left",
				"open":        true,
				"locked":      true,
				"hp":          45,
				"maxHp":       100,
				"resistance":  7,
				"destroyed":   false,
				"revealed":    true,
			},
			{
				"id":            "wall-2",
				"p1":            []float64{5.5, 6.6},
				"p2":            []float64{7.7, 8.8},
				"wallType":      "window",
				"material":      "iron",
				"windowSubtype": "barred",
				"move":          false,
				"sense":         "none",
				"direction":     "right",
				"open":          false,
				"locked":        false,
				"hp":            12,
				"maxHp":         30,
				"resistance":    3,
				"destroyed":     true,
				"revealed":      false,
			},
		},
	}

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.PutCtx(ctx, "/maps/"+mapID.String(), body)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("got status %d, want %d. Body: %s", resp.Code, http.StatusNoContent, resp.Body.String())
	}

	if mock.received == nil {
		t.Fatal("UpdateMap was never called")
	}
	in := mock.received

	if in.Name == nil || *in.Name != name {
		t.Errorf("Name = %v, want %q", in.Name, name)
	}
	if in.Description != "updated description" {
		t.Errorf("Description = %q, want %q", in.Description, "updated description")
	}

	// --- grid: every field, toEntityGridShape ---
	if in.Grid == nil {
		t.Fatal("Grid is nil")
	}
	wantGrid := entity.GridShape{
		Kind: "hex", Cols: 55, Rows: 60, CellSize: 88.5, SkewRatio: 0.15,
		Rotation: 270.5, Color: "#fedcba", Opacity: 0.33, LineStyle: "dashed",
	}
	if *in.Grid != wantGrid {
		t.Errorf("Grid = %+v, want %+v", *in.Grid, wantGrid)
	}

	// --- bg: every field, toEntityBgImage ---
	if in.Bg == nil {
		t.Fatal("Bg is nil")
	}
	wantBg := entity.BgImage{
		URL: "https://example.com/update-bg.png", X: 100.1, Y: 200.2,
		Width: 300.3, Height: 400.4, Rotation: 15.15, Opacity: 0.85,
	}
	if *in.Bg != wantBg {
		t.Errorf("Bg = %+v, want %+v", *in.Bg, wantBg)
	}

	// --- pieces: every field on both entries, toEntityPiece ---
	if in.Pieces == nil || len(*in.Pieces) != 2 {
		t.Fatalf("Pieces = %v, want exactly 2", in.Pieces)
	}
	pieces := *in.Pieces

	pA := pieces[0]
	if pA.ID != "piece-a" || pA.CharacterID != "char-a" || pA.Coord.Z != 2.5 || !pA.Visible {
		t.Errorf("pieces[0] = %+v, want id=piece-a characterId=char-a z=2.5 visible=true", pA)
	}
	slotA, ok := pA.Coord.Slot.(map[string]any)
	if !ok {
		t.Fatalf("pieces[0].Coord.Slot is not a map: %#v", pA.Coord.Slot)
	}
	if slotA["kind"] != "square" || slotA["col"] != float64(7) || slotA["row"] != float64(8) {
		t.Errorf("pieces[0].Coord.Slot = %v, want kind=square col=7 row=8", slotA)
	}

	pB := pieces[1]
	if pB.ID != "piece-b" || pB.CharacterID != "char-b" || pB.Coord.Z != 0.5 || pB.Visible {
		t.Errorf("pieces[1] = %+v, want id=piece-b characterId=char-b z=0.5 visible=false", pB)
	}
	slotB, ok := pB.Coord.Slot.(map[string]any)
	if !ok {
		t.Fatalf("pieces[1].Coord.Slot is not a map: %#v", pB.Coord.Slot)
	}
	if slotB["kind"] != "hex" || slotB["q"] != float64(1) || slotB["r"] != float64(-2) {
		t.Errorf("pieces[1].Coord.Slot = %v, want kind=hex q=1 r=-2", slotB)
	}

	// --- walls: every field on both entries, toEntityWallSegment (including the
	// DoorSubtype/WindowSubtype nilable-pointer branches, one populated in each entry) ---
	if in.Walls == nil || len(*in.Walls) != 2 {
		t.Fatalf("Walls = %v, want exactly 2", in.Walls)
	}
	walls := *in.Walls

	w1 := walls[0]
	wantW1 := entity.WallSegment{
		ID: "wall-1", P1: [2]float64{1.1, 2.2}, P2: [2]float64{3.3, 4.4},
		WallType: "door", Material: "wood",
		Move: true, Sense: "sight", Direction: "left",
		Open: true, Locked: true, HP: 45, MaxHP: 100, Resistance: 7,
		Destroyed: false, Revealed: true,
	}
	if w1.DoorSubtype == nil || *w1.DoorSubtype != "double" {
		t.Errorf("walls[0].DoorSubtype = %v, want \"double\"", w1.DoorSubtype)
	}
	if w1.WindowSubtype != nil {
		t.Errorf("walls[0].WindowSubtype = %v, want nil", *w1.WindowSubtype)
	}
	w1NoPtrs := w1
	w1NoPtrs.DoorSubtype, w1NoPtrs.WindowSubtype = nil, nil
	if w1NoPtrs != wantW1 {
		t.Errorf("walls[0] = %+v, want %+v", w1NoPtrs, wantW1)
	}

	w2 := walls[1]
	wantW2 := entity.WallSegment{
		ID: "wall-2", P1: [2]float64{5.5, 6.6}, P2: [2]float64{7.7, 8.8},
		WallType: "window", Material: "iron",
		Move: false, Sense: "none", Direction: "right",
		Open: false, Locked: false, HP: 12, MaxHP: 30, Resistance: 3,
		Destroyed: true, Revealed: false,
	}
	if w2.WindowSubtype == nil || *w2.WindowSubtype != "barred" {
		t.Errorf("walls[1].WindowSubtype = %v, want \"barred\"", w2.WindowSubtype)
	}
	if w2.DoorSubtype != nil {
		t.Errorf("walls[1].DoorSubtype = %v, want nil", *w2.DoorSubtype)
	}
	w2NoPtrs := w2
	w2NoPtrs.DoorSubtype, w2NoPtrs.WindowSubtype = nil, nil
	if w2NoPtrs != wantW2 {
		t.Errorf("walls[1] = %+v, want %+v", w2NoPtrs, wantW2)
	}
}

// TestUpdateMapHandler_NotMaster_Returns403 keeps the error-path parity with the create
// handler's equivalent test.
func TestUpdateMapHandler_NotMaster_Returns403(t *testing.T) {
	userID := uuid.New()
	mapID := uuid.New()

	mock := &mockUpdateMap{err: mapuc.ErrNotMapMaster}
	api := registerUpdateMapHandler(t, mock)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userID)
	resp := api.PutCtx(ctx, "/maps/"+mapID.String(), map[string]any{"description": ""})

	if resp.Code != http.StatusForbidden {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusForbidden, resp.Body.String())
	}
}
