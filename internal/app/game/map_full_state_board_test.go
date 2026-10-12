package game

import (
	"encoding/json"
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

// map_full_state carries the board's own grid and background (what the room loaded), so the
// table draws with the same grid the server measures with — not the campaign map's.
func TestBuildMapFullState_CarriesBoardGridAndBg(t *testing.T) {
	grid := mapentity.GridShape{
		Kind: mapentity.GridKindHex, Cols: 7, Rows: 9, CellSize: 48, SkewRatio: 0.5,
		Rotation: 15, Color: "#abcdef", Opacity: 0.4, LineStyle: mapentity.LineStyleDashed,
	}
	bg := &mapentity.BgImage{URL: "https://example.com/b.png", X: 1, Y: 2, Width: 3, Height: 4, Rotation: 5, Opacity: 0.6}
	wantBg := BgImagePayload{URL: bg.URL, X: 1, Y: 2, Width: 3, Height: 4, Rotation: 5, Opacity: 0.6}

	for _, isMaster := range []bool{false, true} {
		name := "player"
		if isMaster {
			name = "master"
		}
		t.Run(name, func(t *testing.T) {
			room := newFogRoom(uuid.New(), uuid.New())
			room.grid = grid
			room.bg = bg

			got := decodeMapFull(t, room.buildMapFullState(uuid.New(), isMaster))
			if got.Grid == nil || *got.Grid != toGridShapePayload(grid) {
				t.Fatalf("Grid = %+v, want the room's %+v", got.Grid, toGridShapePayload(grid))
			}
			if got.Bg == nil || *got.Bg != wantBg {
				t.Fatalf("Bg = %+v, want %+v", got.Bg, wantBg)
			}
		})
	}
}

func TestBuildMapFullState_NoBg_SendsNullAndNoBoard_OmitsGrid(t *testing.T) {
	room := newFogRoom(uuid.New(), uuid.New())
	room.grid = mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 5, Rows: 5, CellSize: 32}
	msg := room.buildMapFullState(uuid.New(), true)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(msg.Payload, &raw); err != nil {
		t.Fatal(err)
	}
	if v, ok := raw["bg"]; !ok || string(v) != "null" {
		t.Fatalf("bg = %s (present=%v), want explicit null", v, ok)
	}
	if _, ok := raw["grid"]; !ok {
		t.Fatal("grid missing for a room with a board")
	}

	room.grid = mapentity.GridShape{} // no board: CellSize 0
	msg = room.buildMapFullState(uuid.New(), true)
	raw = map[string]json.RawMessage{}
	if err := json.Unmarshal(msg.Payload, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["grid"]; ok {
		t.Fatalf("grid = %s, want absent when the room has no board", raw["grid"])
	}
}
