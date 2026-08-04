package game_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// This file drives the real WebSocket handler over a real HTTP server with two real
// clients — the master and a player — reproducing the exact production sequence for a
// match that is already started in the DB (the case the user tests: server restarted,
// master opens the game page, player opens the game page).
//
// It is the end-to-end guarantee for fog of war: the player must receive a lit area
// around their own piece, and must see that piece.

// ─── configurable mocks ─────────────────────────────────────────────────────

type fogMatchRepo struct {
	masterUUID uuid.UUID
	started    bool
}

func (m *fogMatchRepo) GetMatchMaster(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return m.masterUUID, nil
}
func (m *fogMatchRepo) IsStarted(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.started, nil
}

type fogInitSessionUC struct {
	matchUUID   uuid.UUID
	participant *match.Participant
}

func (m *fogInitSessionUC) Init(_ context.Context, _ uuid.UUID) (*matchsession.MatchSession, error) {
	return matchsession.NewMatchSession(m.matchUUID, nil, []*match.Participant{m.participant}), nil
}

// ─── fixture ────────────────────────────────────────────────────────────────

type fogFixture struct {
	server     *httptest.Server
	hub        *game.Hub
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	playerUUID uuid.UUID
	sheetUUID  uuid.UUID
	grid       mapentity.GridShape
	wall       mapentity.WallSegment
	piece      game.PieceMovedPayload
}

func newFogFixture(t *testing.T) *fogFixture {
	t.Helper()

	f := &fogFixture{
		matchUUID:  uuid.New(),
		masterUUID: uuid.New(),
		playerUUID: uuid.New(),
		sheetUUID:  uuid.New(),
	}
	f.grid = mapentity.GridShape{
		Kind: mapentity.GridKindSquare, Cols: 30, Rows: 30, CellSize: 64, SkewRatio: 1,
	}
	// A wall well away from the piece so it clips the polygon without enclosing it.
	f.wall = mapentity.WallSegment{
		ID: "w1", P1: [2]float64{512, 0}, P2: [2]float64{512, 640},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	col, row := 2, 2
	f.piece = game.PieceMovedPayload{
		PieceID:     "piece-1",
		CharacterID: f.sheetUUID.String(),
		Slot:        game.SlotPayload{Kind: "square", Col: &col, Row: &row},
	}

	hub := game.NewHub()
	go hub.Run()

	participant := &match.Participant{
		UUID:      uuid.New(),
		MatchUUID: f.matchUUID,
		Sheet:     csEntity.Summary{UUID: f.sheetUUID, PlayerUUID: &f.playerUUID},
	}

	handler := game.NewHandler(
		hub,
		&fogMatchRepo{masterUUID: f.masterUUID, started: true},
		&mockEnrollmentChecker{enrolled: true},
		&mockStartMatchUC{},
		&mockKickPlayerUC{},
		&fogInitSessionUC{matchUUID: f.matchUUID, participant: participant},
		&mockOpenNextActionUCHandler{},
		&mockPullActionUCHandler{},
		&mockEnqueueActionUCHandler{},
		&mockAttachReactionUCHandler{},
		&mockChangeSceneUCHandler{},
		&mockRoundRepoHandler{},
		&mockEnqueueMasterActionUCHandler{},
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)

	f.server = httptest.NewServer(mux)
	f.hub = hub
	t.Cleanup(func() {
		f.server.Close()
		hub.Stop()
	})
	return f
}

// connectMaster dials as the master. This is the path that rehydrates the session for an
// already-started match — the path that used to deadlock and freeze the whole room.
func (f *fogFixture) connectMaster(t *testing.T) *websocket.Conn {
	t.Helper()
	done := make(chan *websocket.Conn, 1)
	go func() { done <- connectWS(t, f.server.URL, f.masterUUID, f.matchUUID) }()
	select {
	case c := <-done:
		return c
	case <-time.After(5 * time.Second):
		t.Fatal("DEADLOCK: master could not connect — session rehydration never returned")
		return nil
	}
}

func (f *fogFixture) connectPlayer(t *testing.T) *websocket.Conn {
	t.Helper()
	return connectWS(t, f.server.URL, f.playerUUID, f.matchUUID)
}

// sendBoardSync mirrors what the master's client sends once its REST map has loaded.
func (f *fogFixture) sendBoardSync(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	f.sendBoardSyncWithWalls(t, conn, []mapentity.WallSegment{f.wall})
}

func (f *fogFixture) sendBoardSyncWithWalls(t *testing.T, conn *websocket.Conn, walls []mapentity.WallSegment) {
	t.Helper()
	pieces := []game.PieceMovedPayload{f.piece}
	payload, err := json.Marshal(game.MapStateSyncPayload{
		Pieces: &pieces,
		Walls:  walls,
		Grid:   &f.grid,
	})
	if err != nil {
		t.Fatalf("marshal sync payload: %v", err)
	}
	raw, err := json.Marshal(map[string]any{
		"type": "map_state_sync", "payload": json.RawMessage(payload),
	})
	if err != nil {
		t.Fatalf("marshal sync message: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("send map_state_sync: %v", err)
	}
}

// awaitMapFullState reads until a map_full_state arrives or the deadline passes.
func awaitMapFullState(t *testing.T, conn *websocket.Conn, d time.Duration) *game.MapFullStatePayload {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(deadline)
		_, data, err := conn.ReadMessage()
		if err != nil {
			return nil
		}
		var msg game.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type != game.MsgTypeMapFullState {
			continue
		}
		var p game.MapFullStatePayload
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			t.Fatalf("unmarshal map_full_state: %v", err)
		}
		return &p
	}
	return nil
}

func assertPlayerCanSee(t *testing.T, f *fogFixture, view *game.MapFullStatePayload) {
	t.Helper()
	if view == nil {
		t.Fatal("player never received a map_full_state — the client stays fully fogged")
	}
	if len(view.VisiblePolygons) == 0 {
		t.Fatal("player received zero visibility polygons — the fog covers the whole map")
	}
	if len(view.VisiblePolygons[0]) < 3 {
		t.Fatalf("visibility polygon is degenerate: %d vertices", len(view.VisiblePolygons[0]))
	}
	found := false
	for _, p := range view.Pieces {
		if p.CharacterID == f.sheetUUID.String() {
			found = true
		}
	}
	if !found {
		t.Fatal("player cannot see their own character's piece")
	}
}

// ─── tests ──────────────────────────────────────────────────────────────────

// Master connects and seeds the board, then the player joins. The player's first
// map_full_state must already carry their line of sight.
func TestE2E_MasterSyncsThenPlayerJoins_PlayerSeesFogLiftedAroundOwnPiece(t *testing.T) {
	f := newFogFixture(t)

	master := f.connectMaster(t)
	defer master.Close()
	readMessage(t, master) // room_state

	f.sendBoardSync(t, master)
	time.Sleep(200 * time.Millisecond) // let the server apply the sync

	player := f.connectPlayer(t)
	defer player.Close()

	assertPlayerCanSee(t, f, awaitMapFullState(t, player, 3*time.Second))
}

// The reverse order, which is what happens in practice: the player's page is already
// open when the master's REST map finishes loading and the sync goes out. The server
// must re-push the refreshed state instead of leaving the player on the empty board.
func TestE2E_PlayerJoinsBeforeSync_ServerRepushesLineOfSight(t *testing.T) {
	f := newFogFixture(t)

	master := f.connectMaster(t)
	defer master.Close()
	readMessage(t, master) // room_state

	player := f.connectPlayer(t)
	defer player.Close()
	time.Sleep(200 * time.Millisecond) // player is registered on the empty board

	f.sendBoardSync(t, master)

	assertPlayerCanSee(t, f, awaitMapFullState(t, player, 3*time.Second))
}

// A real map's board sync does not fit in a small frame. When the server's read limit
// is too low, gorilla closes the master's connection mid-sync: the board is never
// seeded, players never get a polygon, and the master reconnects and retries forever.
// This drives a board comfortably larger than one 4 KB frame.
func TestE2E_LargeBoardSyncIsNotRejected(t *testing.T) {
	f := newFogFixture(t)

	walls := make([]mapentity.WallSegment, 0, 60)
	for i := range 60 {
		x := float64(512 + i*8)
		walls = append(walls, mapentity.WallSegment{
			ID: uuid.New().String(), P1: [2]float64{x, 0}, P2: [2]float64{x, 640},
			WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
			Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
		})
	}

	master := f.connectMaster(t)
	defer master.Close()
	readMessage(t, master) // room_state

	player := f.connectPlayer(t)
	defer player.Close()
	time.Sleep(200 * time.Millisecond)

	f.sendBoardSyncWithWalls(t, master, walls)

	view := awaitMapFullState(t, player, 3*time.Second)
	if view == nil {
		t.Fatal("player got nothing — the server dropped the master's board sync")
	}
	if len(view.VisiblePolygons) == 0 {
		t.Fatal("player received zero visibility polygons after a large board sync")
	}
}

// The master must never be fogged: no polygons, and every piece on the board.
func TestE2E_MasterSeesWholeBoardWithoutFog(t *testing.T) {
	f := newFogFixture(t)

	master := f.connectMaster(t)
	defer master.Close()
	readMessage(t, master) // room_state

	f.sendBoardSync(t, master)

	view := awaitMapFullState(t, master, 3*time.Second)
	if view == nil {
		t.Fatal("master never received a map_full_state")
	}
	if len(view.VisiblePolygons) != 0 {
		t.Fatalf("master must not be fogged: got %d visibility polygons", len(view.VisiblePolygons))
	}
	if len(view.Pieces) != 1 {
		t.Fatalf("master must see every piece: got %d, want 1", len(view.Pieces))
	}
}
