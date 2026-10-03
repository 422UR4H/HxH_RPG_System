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
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
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
//
// The board itself is seeded via the fixture's fakeBoardStore, not by the master sending
// map_state_sync over the socket — that write path is gone since B14 (spec §4.3, "Quem
// carrega"): the server loads the board itself, on the master's register, which for this
// fixture's already-started match happens the moment the master connects (see
// combinatFixture.seedBoard in combat_e2e_test.go for the sibling of this file's seedBoard).

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
	piece      mapentity.Piece
	boards     *fakeBoardStore
}

func newFogFixture(t *testing.T) *fogFixture {
	t.Helper()

	f := &fogFixture{
		matchUUID:  uuid.New(),
		masterUUID: uuid.New(),
		playerUUID: uuid.New(),
		sheetUUID:  uuid.New(),
		boards:     newFakeBoardStore(),
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
	f.piece = mapentity.Piece{
		ID:          "piece-1",
		CharacterID: f.sheetUUID.String(),
		Coord:       mapentity.PieceCoord{Slot: mapentity.SquareCoord{Kind: "square", Col: 2, Row: 2}},
		Visible:     true,
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
		game.RoomDeps{
			StartMatchUC:          &mockStartMatchUC{},
			KickPlayerUC:          &mockKickPlayerUC{},
			InitSessionUC:         &fogInitSessionUC{matchUUID: f.matchUUID, participant: participant},
			OpenNextActionUC:      &mockOpenNextActionUCHandler{},
			PullActionUC:          &mockPullActionUCHandler{},
			EnqueueActionUC:       &mockEnqueueActionUCHandler{},
			AttachReactionUC:      &mockAttachReactionUCHandler{},
			OpenReactionUC:        &mockOpenReactionUCHandler{},
			CloseTurnUC:           &mockCloseTurnUCHandler{},
			ChangeSceneUC:         &mockChangeSceneUCHandler{},
			RoundRepo:             &mockRoundRepoHandler{},
			EnqueueMasterActionUC: &mockEnqueueMasterActionUCHandler{},
			ChangeRoundModeUC:     &mockChangeRoundModeUCHandler{},
			EditActionUC:          &mockEditActionUCHandler{},
			LoadBoardUC:           f.boards,
		},
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

// seedBoard puts the fixture's piece and wall directly into the fakeBoardStore — the server
// loads this itself on the master's register (spec §4.3, B14), so seeding must happen BEFORE
// connectMaster, not sent over the socket the way map_state_sync used to.
func (f *fogFixture) seedBoard(t *testing.T) {
	t.Helper()
	f.seedBoardWithWalls(t, []mapentity.WallSegment{f.wall})
}

func (f *fogFixture) seedBoardWithWalls(t *testing.T, walls []mapentity.WallSegment) {
	t.Helper()
	f.boards.seed(f.matchUUID, &matchboard.Board{
		MatchUUID: f.matchUUID,
		MapUUID:   uuid.New(),
		Grid:      f.grid,
		Pieces:    []mapentity.Piece{f.piece},
		Walls:     walls,
	})
}

// connectMaster dials as the master. This is the path that rehydrates the session for an
// already-started match — the path that used to deadlock and freeze the whole room — and,
// since B14, the path whose register also loads the board (loadBoard runs before
// sendRoomState in Run's register branch).
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

// The board is seeded before the master ever connects — production has it in match_boards
// (or the attached map) long before anyone opens the game page. The master's register loads
// it; the player's own register then serves it straight from the room's now-populated board,
// with their line of sight already lit.
func TestE2E_MasterSyncsThenPlayerJoins_PlayerSeesFogLiftedAroundOwnPiece(t *testing.T) {
	f := newFogFixture(t)
	f.seedBoard(t)

	master := f.connectMaster(t)
	defer master.Close()   //nolint:errcheck
	readMessage(t, master) // room_state

	player := f.connectPlayer(t)
	defer player.Close() //nolint:errcheck

	assertPlayerCanSee(t, f, awaitMapFullState(t, player, 3*time.Second))
}

// The master must never be fogged: no polygons, and every piece on the board. Board seeded
// up front, same as above — the server loads it on the master's own register, so there is no
// sync round-trip left to await.
func TestE2E_MasterSeesWholeBoardWithoutFog(t *testing.T) {
	f := newFogFixture(t)
	f.seedBoard(t)

	master := f.connectMaster(t)
	defer master.Close() //nolint:errcheck

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
