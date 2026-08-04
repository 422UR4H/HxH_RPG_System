package service

import (
	"testing"

	"github.com/google/uuid"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

func TestFilterMapState_PlayerSeesOnlyVisibleAndOwn(t *testing.T) {
	player := uuid.New()
	grid := mapentity.GridShape{Kind: mapentity.GridKindSquare, CellSize: 64, SkewRatio: 1}
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
		fog.FogModeLive, grid, player, charToPlayer, false,
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
	grid := mapentity.GridShape{Kind: mapentity.GridKindSquare, CellSize: 64, SkewRatio: 1}
	sd := mapentity.WallSegment{ID: "sd", WallType: mapentity.WallTypeSecretDoor}
	pieces := []PieceVisibility{{ID: "p", CharacterID: "c", Pos: Point2D{9, 9}, Visible: false}}
	walls, visIDs := FilterMapState([]mapentity.WallSegment{sd}, pieces, nil, nil,
		fog.FogModeLive, grid, uuid.New(), nil, true)
	if walls[0].WallType != mapentity.WallTypeSecretDoor {
		t.Fatal("master must see real secret door type")
	}
	if !visIDs["p"] {
		t.Fatal("master must see invisible pieces")
	}
}
