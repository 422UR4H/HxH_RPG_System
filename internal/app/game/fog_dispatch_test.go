package game

import (
	"encoding/json"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// fogTestRoom builds a Room with a live session in which playerUUID owns one piece
// sitting at the world center of cell (0,0), plus a secret door near that piece so the
// player has line of sight to it. Returns the room and the secret door's wall ID.
func fogTestRoom(t *testing.T) (*Room, uuid.UUID, string) {
	t.Helper()

	matchUUID := uuid.New()
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	sheetUUID := uuid.New()

	room := newFogRoom(matchUUID, masterUUID)

	// Participant whose sheet UUID is the piece CharacterID and is owned by the player.
	participant := &match.Participant{
		UUID:      uuid.New(),
		MatchUUID: matchUUID,
		Sheet: csEntity.Summary{
			UUID:       sheetUUID,
			PlayerUUID: &playerUUID,
		},
	}
	session := matchsession.NewMatchSession(matchUUID, nil, []*match.Participant{participant})

	grid := mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 20, Rows: 20, CellSize: 64, SkewRatio: 1}

	// A secret door with sense=none so it never occludes the player's own LOS, ensuring its
	// midpoint stays visible — the masking branch is what we assert.
	sub := mapentity.DoorSubtypeBasic
	door := mapentity.WallSegment{
		ID:          "sd1",
		P1:          [2]float64{20, 20},
		P2:          [2]float64{40, 20},
		WallType:    mapentity.WallTypeSecretDoor,
		Material:    mapentity.WallMaterialStone,
		DoorSubtype: &sub,
		Sense:       mapentity.SenseNone,
		Open:        true,
		Locked:      true,
		HP:          80,
		MaxHP:       100,
	}

	piece := PieceMovedPayload{
		PieceID:     "p1",
		CharacterID: sheetUUID.String(),
		Slot:        squareSlot(0, 0),
	}

	room.session = session
	room.grid = grid
	room.walls[door.ID] = door
	room.pieces[piece.PieceID] = piece

	session.SyncMapState([]mapentity.WallSegment{door}, grid)
	session.SetPieceSource(room)
	session.SyncPlayerMemories(nil, fogentity.FogModeExplored)
	if _, err := session.RecomputeVisibility(playerUUID); err != nil {
		t.Fatalf("recompute visibility: %v", err)
	}

	return room, playerUUID, door.ID
}

func squareSlot(col, row int) SlotPayload {
	c, r := col, row
	return SlotPayload{Kind: "square", Col: &c, Row: &r}
}

func newFogRoom(matchUUID, masterUUID uuid.UUID) *Room {
	return NewRoom(
		matchUUID, masterUUID,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
}

func decodeMapFull(t *testing.T, msg *Message) MapFullStatePayload {
	t.Helper()
	if msg == nil {
		t.Fatal("expected a message, got nil")
	}
	if msg.Type != MsgTypeMapFullState {
		t.Fatalf("expected map_full_state, got %s", msg.Type)
	}
	var p MapFullStatePayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		t.Fatalf("unmarshal map_full_state: %v", err)
	}
	return p
}

func TestBuildMapFullState_MasksSecretDoorForPlayer(t *testing.T) {
	room, playerUUID, doorID := fogTestRoom(t)

	playerView := decodeMapFull(t, room.buildMapFullState(playerUUID, false))
	masterView := decodeMapFull(t, room.buildMapFullState(uuid.New(), true))

	// Master sees the real secret door.
	var masterWall *WallSegmentPayload
	for i := range masterView.Walls {
		if masterView.Walls[i].ID == doorID {
			masterWall = &masterView.Walls[i]
		}
	}
	if masterWall == nil {
		t.Fatal("master must see the secret door")
	}
	if masterWall.WallType != string(mapentity.WallTypeSecretDoor) {
		t.Fatalf("master must see real type secret_door, got %s", masterWall.WallType)
	}

	// Player sees the same wall masked as a plain wall, with no leaked door fields.
	var playerWall *WallSegmentPayload
	for i := range playerView.Walls {
		if playerView.Walls[i].ID == doorID {
			playerWall = &playerView.Walls[i]
		}
	}
	if playerWall == nil {
		t.Fatal("player should see the door's geometry, masked as a wall")
	}
	if playerWall.WallType != string(mapentity.WallTypeWall) {
		t.Fatalf("player must see masked type wall, got %s", playerWall.WallType)
	}
	if playerWall.DoorSubtype != nil || playerWall.Open || playerWall.Locked {
		t.Fatal("masked door must not leak subtype/open/locked to players")
	}
	// Combat-relevant fields are preserved on the mask.
	if playerWall.HP != 80 || playerWall.MaxHP != 100 {
		t.Fatalf("masked wall must keep hp for combat parity, got hp=%d maxhp=%d", playerWall.HP, playerWall.MaxHP)
	}
}

func TestBuildMapFullState_PlayerGetsPolygons_MasterDoesNot(t *testing.T) {
	room, playerUUID, _ := fogTestRoom(t)

	playerView := decodeMapFull(t, room.buildMapFullState(playerUUID, false))
	masterView := decodeMapFull(t, room.buildMapFullState(uuid.New(), true))

	if len(playerView.VisiblePolygons) == 0 {
		t.Fatal("player view must carry visible polygons")
	}
	if len(masterView.VisiblePolygons) != 0 {
		t.Fatal("master view must not carry visible polygons")
	}
}

// In the lobby (no live session) there is no LOS, but a player must still not see
// secret-door identity or invisible pieces — the backend masks/hides regardless of phase.
// An unrevealed secret door's open/locked change must reach the master only. Players see
// the door as a plain wall, and a plain wall has no open/locked state — broadcasting it
// would leak the door's identity. Guards the master-action interact path (C-1).
func TestBroadcastWallStateChangedGated_UnrevealedSecretDoorIsMasterOnly(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	room := newFogRoom(uuid.New(), masterUUID)

	room.walls["sd1"] = mapentity.WallSegment{
		ID: "sd1", WallType: mapentity.WallTypeSecretDoor, Revealed: false,
		P1: [2]float64{0, 0}, P2: [2]float64{10, 0},
	}

	master := NewClient(masterUUID, nil, "gm")
	player := NewClient(playerUUID, nil, "p1")
	room.clients[masterUUID] = master
	room.clients[playerUUID] = player

	room.broadcastWallStateChangedGated("sd1", true, false)

	select {
	case data := <-master.send:
		var m Message
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if m.Type != MsgTypeWallStateChanged {
			t.Fatalf("master got %s, want wall_state_changed", m.Type)
		}
	default:
		t.Fatal("master must receive the secret door's state change")
	}

	select {
	case <-player.send:
		t.Fatal("player must NOT receive wall_state_changed for an unrevealed secret door")
	default:
	}
}

func TestBuildMapFullState_LobbyHidesWallsFromPlayers(t *testing.T) {
	// Verify that non-master lobby players do not receive walls in the payload.
	// The masking computation (computeLobbyMapState) is still performed internally
	// and verified separately in TestComputeLobbyMapState_MasksSecretDoors.
	room := newFogRoom(uuid.New(), uuid.New())
	room.grid = mapentity.GridShape{Kind: mapentity.GridKindSquare, Cols: 20, Rows: 20, CellSize: 64, SkewRatio: 1}
	// No session set → lobby phase.

	sub := mapentity.DoorSubtypeBasic
	door := mapentity.WallSegment{
		ID: "sd1", P1: [2]float64{20, 20}, P2: [2]float64{40, 20},
		WallType: mapentity.WallTypeSecretDoor, Material: mapentity.WallMaterialStone,
		DoorSubtype: &sub, Open: true, Locked: true, HP: 80, MaxHP: 100,
	}
	normal := mapentity.WallSegment{
		ID: "w1", P1: [2]float64{0, 0}, P2: [2]float64{64, 0},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone, HP: 100, MaxHP: 100,
	}
	room.walls[door.ID] = door
	room.walls[normal.ID] = normal

	visiblePiece := PieceMovedPayload{PieceID: "pv", CharacterID: uuid.New().String(), Slot: squareSlot(0, 0)}
	no := false
	hiddenPiece := PieceMovedPayload{PieceID: "ph", CharacterID: uuid.New().String(), Slot: squareSlot(1, 1), Visible: &no}
	room.pieces[visiblePiece.PieceID] = visiblePiece
	room.pieces[hiddenPiece.PieceID] = hiddenPiece

	view := decodeMapFull(t, room.buildMapFullState(uuid.New(), false))

	// Non-master lobby players do not receive walls (frontend doesn't consume them yet)
	if len(view.Walls) != 0 {
		t.Fatal("lobby: walls must not be sent to non-master players")
	}

	ids := map[string]bool{}
	for _, p := range view.Pieces {
		ids[p.PieceID] = true
	}
	if !ids["pv"] {
		t.Fatal("lobby: a visible piece must be shown")
	}
	if ids["ph"] {
		t.Fatal("lobby: an invisible piece must never reach players")
	}
	if len(view.VisiblePolygons) != 0 {
		t.Fatal("lobby view must not carry visibility polygons")
	}
}

func TestComputeLobbyMapState_MasksSecretDoors(t *testing.T) {
	// Verify that computeLobbyMapState correctly masks unrevealed secret doors
	// and passes through normal walls. This tests the masking wiring that
	// buildMapFullState uses but doesn't expose to players (for now).
	sub := mapentity.DoorSubtypeBasic
	revealedSecret := mapentity.WallSegment{
		ID: "sd_revealed", P1: [2]float64{10, 10}, P2: [2]float64{20, 10},
		WallType: mapentity.WallTypeSecretDoor, Material: mapentity.WallMaterialStone,
		DoorSubtype: &sub, Open: true, Locked: true, HP: 80, MaxHP: 100, Revealed: true,
	}
	unrevealed := mapentity.WallSegment{
		ID: "sd_unrevealed", P1: [2]float64{30, 30}, P2: [2]float64{40, 30},
		WallType: mapentity.WallTypeSecretDoor, Material: mapentity.WallMaterialStone,
		DoorSubtype: &sub, Open: true, Locked: true, HP: 80, MaxHP: 100, Revealed: false,
	}
	normal := mapentity.WallSegment{
		ID: "w_normal", P1: [2]float64{0, 0}, P2: [2]float64{64, 0},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone, HP: 100, MaxHP: 100,
	}

	allWalls := []mapentity.WallSegment{revealedSecret, unrevealed, normal}
	pieceProj := []domainservice.PieceVisibility{
		{ID: "visible", Visible: true},
		{ID: "hidden", Visible: false},
	}

	walls, visIDs := computeLobbyMapState(allWalls, pieceProj)

	// Should have 3 walls: revealed secret (unchanged), masked unrevealed secret, normal wall
	if len(walls) != 3 {
		t.Fatalf("expected 3 walls, got %d", len(walls))
	}

	// Verify revealed secret is passed through unchanged
	if walls[0].ID != "sd_revealed" || walls[0].WallType != mapentity.WallTypeSecretDoor {
		t.Fatal("revealed secret door should pass through unchanged")
	}
	if walls[0].DoorSubtype == nil || walls[0].Open != true || walls[0].Locked != true {
		t.Fatal("revealed secret door metadata must be preserved")
	}

	// Verify unrevealed secret is masked as a plain wall
	if walls[1].ID != "sd_unrevealed" || walls[1].WallType != mapentity.WallTypeWall {
		t.Fatal("unrevealed secret door should be masked as a plain wall")
	}
	if walls[1].DoorSubtype != nil || walls[1].Open || walls[1].Locked {
		t.Fatal("masked secret door must not leak subtype/open/locked")
	}

	// Verify normal wall is passed through
	if walls[2].ID != "w_normal" || walls[2].WallType != mapentity.WallTypeWall {
		t.Fatal("normal wall should pass through unchanged")
	}

	// Verify visIDs contains only visible pieces
	if !visIDs["visible"] || visIDs["hidden"] {
		t.Fatal("visIDs must contain only visible pieces")
	}
}
