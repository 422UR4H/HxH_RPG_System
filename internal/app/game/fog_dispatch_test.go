package game

import (
	"encoding/json"
	"testing"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
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
	session.SyncFogStates(nil, "explored")
	if _, _, err := session.RecomputeVisibility(playerUUID); err != nil {
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
	var masterWall *mapentity.WallSegment
	for i := range masterView.Walls {
		if masterView.Walls[i].ID == doorID {
			masterWall = &masterView.Walls[i]
		}
	}
	if masterWall == nil {
		t.Fatal("master must see the secret door")
	}
	if masterWall.WallType != mapentity.WallTypeSecretDoor {
		t.Fatalf("master must see real type secret_door, got %s", masterWall.WallType)
	}

	// Player sees the same wall masked as a plain wall, with no leaked door fields.
	var playerWall *mapentity.WallSegment
	for i := range playerView.Walls {
		if playerView.Walls[i].ID == doorID {
			playerWall = &playerView.Walls[i]
		}
	}
	if playerWall == nil {
		t.Fatal("player should see the door's geometry, masked as a wall")
	}
	if playerWall.WallType != mapentity.WallTypeWall {
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
func TestBuildMapFullState_LobbyMasksSecretsWithoutLOS(t *testing.T) {
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

	var sd, nw *mapentity.WallSegment
	for i := range view.Walls {
		switch view.Walls[i].ID {
		case "sd1":
			sd = &view.Walls[i]
		case "w1":
			nw = &view.Walls[i]
		}
	}
	if sd == nil || sd.WallType != mapentity.WallTypeWall {
		t.Fatal("lobby: secret door must be masked as a plain wall for players")
	}
	if sd.DoorSubtype != nil || sd.Open || sd.Locked {
		t.Fatal("lobby: masked door must not leak subtype/open/locked")
	}
	if nw == nil {
		t.Fatal("lobby: normal wall must be visible (no LOS gating in lobby)")
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
