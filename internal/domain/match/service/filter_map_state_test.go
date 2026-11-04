package service

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/google/uuid"
)

func TestFilterMapState_PlayerSeesOnlyVisibleAndOwn(t *testing.T) {
	player := uuid.New()
	// Visibility polygon covering a small box near origin.
	polys := []VisibilityPolygon{{Vertices: []Point2D{{0, 0}, {100, 0}, {100, 100}, {0, 100}}}}

	pieces := []PieceVisibility{
		{ID: "own", CharacterID: "cOwn", Pos: Point2D{500, 500}, Visible: true},       // far but own → seen
		{ID: "seen", CharacterID: "cE1", Pos: Point2D{50, 50}, Visible: true},         // in poly → seen
		{ID: "hiddenGeom", CharacterID: "cE2", Pos: Point2D{500, 500}, Visible: true}, // far, not own → not seen
		{ID: "invisible", CharacterID: "cE3", Pos: Point2D{50, 50}, Visible: false},   // visible=false → never
	}
	charToPlayer := map[string]uuid.UUID{"cOwn": player}

	sd := mapentity.WallSegment{ID: "sd", WallType: mapentity.WallTypeSecretDoor, P1: [2]float64{10, 10}, P2: [2]float64{20, 10}}
	normal := mapentity.WallSegment{ID: "n", WallType: mapentity.WallTypeWall, P1: [2]float64{10, 20}, P2: [2]float64{20, 20}}
	far := mapentity.WallSegment{ID: "f", WallType: mapentity.WallTypeWall, P1: [2]float64{500, 500}, P2: [2]float64{520, 500}}

	walls, visIDs := FilterMapState(
		[]mapentity.WallSegment{sd, normal, far}, pieces, polys, nil,
		fog.FogModeLive, player, charToPlayer, false,
	)

	if !visIDs["own"] || !visIDs["seen"] {
		t.Fatal("own and in-LOS pieces must be visible")
	}
	if visIDs["hiddenGeom"] || visIDs["invisible"] {
		t.Fatal("out-of-LOS non-own and invisible pieces must be hidden")
	}
	// Walls: sd masked to wall, normal kept, far dropped.
	byID := map[string]mapentity.WallSegment{}
	for _, w := range walls {
		byID[w.ID] = w
	}
	if _, ok := byID["f"]; ok {
		t.Fatal("far wall out of LOS must be dropped")
	}
	if byID["sd"].WallType != mapentity.WallTypeWall {
		t.Fatal("secret door must be masked as wall for player")
	}
	if _, ok := byID["n"]; !ok {
		t.Fatal("visible normal wall must be kept")
	}
}

func TestFilterMapState_MasterSeesEverythingUnmasked(t *testing.T) {
	sd := mapentity.WallSegment{ID: "sd", WallType: mapentity.WallTypeSecretDoor}
	pieces := []PieceVisibility{{ID: "p", CharacterID: "c", Pos: Point2D{9, 9}, Visible: false}}
	walls, visIDs := FilterMapState([]mapentity.WallSegment{sd}, pieces, nil, nil,
		fog.FogModeLive, uuid.New(), nil, true)
	if walls[0].WallType != mapentity.WallTypeSecretDoor {
		t.Fatal("master must see real secret door type")
	}
	if !visIDs["p"] {
		t.Fatal("master must see invisible pieces")
	}
}

func TestFilterPerspective(t *testing.T) {
	poly := VisibilityPolygon{Vertices: []Point2D{{0, 0}, {100, 0}, {100, 100}, {0, 100}}}
	inWall := mapentity.WallSegment{ID: "in", WallType: mapentity.WallTypeWall, P1: [2]float64{10, 20}, P2: [2]float64{20, 20}}
	outWall := mapentity.WallSegment{ID: "out", WallType: mapentity.WallTypeWall, P1: [2]float64{500, 500}, P2: [2]float64{520, 500}}
	secret := mapentity.WallSegment{ID: "sd", WallType: mapentity.WallTypeSecretDoor, P1: [2]float64{10, 10}, P2: [2]float64{20, 10}}

	tests := []struct {
		name       string
		walls      []mapentity.WallSegment
		pieces     []PieceVisibility
		viewer     string
		wantWalls  []string
		wantMasked string
		wantPieces []string
	}{
		{name: "wall in polygon returned, wall outside not", walls: []mapentity.WallSegment{inWall, outWall}, wantWalls: []string{"in"}},
		{name: "unrevealed secret door is masked", walls: []mapentity.WallSegment{secret}, wantWalls: []string{"sd"}, wantMasked: "sd"},
		{
			name: "piece inside visible, piece outside not",
			pieces: []PieceVisibility{
				{ID: "a", Pos: Point2D{50, 50}, Visible: true},
				{ID: "b", Pos: Point2D{500, 500}, Visible: true},
			},
			wantPieces: []string{"a"},
		},
		{
			name:       "invisible piece inside polygon is hidden",
			pieces:     []PieceVisibility{{ID: "a", Pos: Point2D{50, 50}, Visible: false}},
			wantPieces: nil,
		},
		{
			name:       "viewer piece always included even when invisible",
			pieces:     []PieceVisibility{{ID: "me", Pos: Point2D{500, 500}, Visible: false}},
			viewer:     "me",
			wantPieces: []string{"me"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walls, ids := FilterPerspective(tt.walls, tt.pieces, poly, tt.viewer)
			if len(walls) != len(tt.wantWalls) {
				t.Fatalf("walls = %d, want %d", len(walls), len(tt.wantWalls))
			}
			for i, w := range walls {
				if w.ID != tt.wantWalls[i] {
					t.Errorf("wall[%d] = %s, want %s", i, w.ID, tt.wantWalls[i])
				}
				if w.ID == tt.wantMasked && w.WallType != mapentity.WallTypeWall {
					t.Errorf("wall %s type = %s, want masked as wall", w.ID, w.WallType)
				}
			}
			if len(ids) != len(tt.wantPieces) {
				t.Fatalf("pieces = %v, want %v", ids, tt.wantPieces)
			}
			for _, id := range tt.wantPieces {
				if !ids[id] {
					t.Errorf("piece %s missing", id)
				}
			}
		})
	}
}
