package game

import (
	"testing"
	"time"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	mapservice "github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func hexSlot(q, r int) SlotPayload {
	return SlotPayload{Kind: "hex", Q: &q, R: &r}
}

// fogRoomOn builds a live room on the given grid with the given walls and one piece for
// the player at ownSlot. Walls are in LOCAL grid space, like production.
func fogRoomOn(t *testing.T, grid mapentity.GridShape, walls []mapentity.WallSegment, ownSlot SlotPayload) (*Room, uuid.UUID) {
	t.Helper()
	matchUUID, masterUUID, playerUUID, sheetUUID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	room := newFogRoom(matchUUID, masterUUID)
	room.grid = grid
	participant := &match.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet:     csEntity.Summary{UUID: sheetUUID, PlayerUUID: &playerUUID},
	}
	session := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})
	for _, w := range walls {
		room.walls[w.ID] = w
	}
	room.pieces["own"] = PieceMovedPayload{PieceID: "own", CharacterID: sheetUUID.String(), Slot: ownSlot}
	room.session = session
	session.SyncMapState(walls, grid)
	session.SetPieceSource(room)
	session.SyncPlayerMemories(nil, fogentity.FogModeExplored)
	room.state = RoomStatePlaying
	return room, playerUUID
}

func recompute(t *testing.T, room *Room, playerUUID uuid.UUID) []domainservice.VisibilityPolygon {
	t.Helper()
	var polys []domainservice.VisibilityPolygon
	withinTimeout(t, 3*time.Second, "RecomputeVisibility", func() {
		room.mu.Lock()
		defer room.mu.Unlock()
		polys, _ = room.session.RecomputeVisibility(playerUUID)
	})
	return polys
}

// The server works in LOCAL space only: on a rotated, flattened grid, a LOCAL wall between
// two pieces hides the one behind it. The visibility origin used to be WORLD (transformed)
// while walls stayed LOCAL, so the ray started from the wrong place.
func TestFog_RotatedGrid_LocalWallHidesThePieceBehindIt(t *testing.T) {
	grid := mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 20, Rows: 20, CellSize: 64, SkewRatio: 0.6, Rotation: 30}
	wall := mapentity.WallSegment{
		ID: "w1", P1: [2]float64{192, 64}, P2: [2]float64{192, 128},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	room, playerUUID := fogRoomOn(t, grid, []mapentity.WallSegment{wall}, squareSlot(1, 1))
	room.pieces["near"] = PieceMovedPayload{PieceID: "near", CharacterID: uuid.NewString(), Slot: squareSlot(2, 1)}
	room.pieces["behind"] = PieceMovedPayload{PieceID: "behind", CharacterID: uuid.NewString(), Slot: squareSlot(4, 1)}
	recompute(t, room, playerUUID)

	view := decodeMapFull(t, room.buildMapFullState(playerUUID, false))
	seen := map[string]bool{}
	for _, p := range view.Pieces {
		seen[p.PieceID] = true
	}
	if !seen["own"] || !seen["near"] {
		t.Fatalf("the player must see their own piece and the one before the wall, got %v", seen)
	}
	if seen["behind"] {
		t.Fatal("the piece behind the wall is visible: the origin and the wall are in different spaces")
	}
}

// On a hex grid, the visibility polygon of a board with no walls reaches the four corners of
// GridLocalBounds. The sweep radius used to be Cols*CellSize x Rows*CellSize, which on hex
// is shorter than the board and cut the polygon before the edge.
func TestFog_HexGrid_NoWalls_PolygonReachesTheBoardCorners(t *testing.T) {
	grid := mapentity.GridShape{Kind: mapentity.GridKindHex, Cols: 6, Rows: 6, CellSize: 64, SkewRatio: 1}
	room, playerUUID := fogRoomOn(t, grid, nil, hexSlot(0, 0))
	polys := recompute(t, room, playerUUID)
	if len(polys) != 1 {
		t.Fatalf("want 1 polygon, got %d", len(polys))
	}
	minX, minY, maxX, maxY := mapservice.GridLocalBounds(grid)
	const inset = 1.0
	corners := map[string]domainservice.Point2D{
		"top-left":     {X: minX + inset, Y: minY + inset},
		"top-right":    {X: maxX - inset, Y: minY + inset},
		"bottom-right": {X: maxX - inset, Y: maxY - inset},
		"bottom-left":  {X: minX + inset, Y: maxY - inset},
	}
	for name, c := range corners {
		if !domainservice.PointInPolygon(c, polys[0].Vertices) {
			t.Errorf("the polygon does not reach the %s corner (%.1f, %.1f)", name, c.X, c.Y)
		}
	}
}

// On a hex grid with front-scale coordinates (size = CellSize), a wall on the edge shared by
// hexes (2,0) and (3,0) hides the piece at (4,0) but not the one at (2,0). The wall numbers are
// literal (x = 2.5*64*sqrt(3)) on purpose: the test must catch a change in the Go hex scale.
// With the hex at half size, (4,0) sat at x~221, BEFORE the wall, and would show up.
func TestFog_HexGrid_LocalWallHidesThePieceBehindIt(t *testing.T) {
	grid := mapentity.GridShape{Kind: mapentity.GridKindHex, Cols: 10, Rows: 10, CellSize: 64, SkewRatio: 1}
	wall := mapentity.WallSegment{
		ID: "w1", P1: [2]float64{277.12812921102035, -32}, P2: [2]float64{277.12812921102035, 32},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	room, playerUUID := fogRoomOn(t, grid, []mapentity.WallSegment{wall}, hexSlot(1, 0))
	room.pieces["near"] = PieceMovedPayload{PieceID: "near", CharacterID: uuid.NewString(), Slot: hexSlot(2, 0)}
	room.pieces["behind"] = PieceMovedPayload{PieceID: "behind", CharacterID: uuid.NewString(), Slot: hexSlot(4, 0)}
	recompute(t, room, playerUUID)

	view := decodeMapFull(t, room.buildMapFullState(playerUUID, false))
	seen := map[string]bool{}
	for _, p := range view.Pieces {
		seen[p.PieceID] = true
	}
	if !seen["own"] || !seen["near"] {
		t.Fatalf("the player must see their own piece and the one before the wall, got %v", seen)
	}
	if seen["behind"] {
		t.Fatal("the piece behind the wall is visible: the hex scale does not match the front's")
	}
}
