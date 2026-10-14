package game_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// flakyBoardLoader is an ILoadMatchBoard that fails while the scripted outcomes say so: outcomes[i]
// is the result of the i-th Load call (true = fail), and every call past the script succeeds.
// board == nil on a success means "no map attached" ((nil, nil)).
type flakyBoardLoader struct {
	mu       sync.Mutex
	calls    int
	outcomes []bool
	// nilAt lists the call indexes that succeed with (nil, nil): "no map attached".
	nilAt map[int]bool
	board *matchboard.Board
}

func (f *flakyBoardLoader) Load(_ context.Context, _ uuid.UUID) (*matchboard.Board, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	i := f.calls
	f.calls++
	if i < len(f.outcomes) && f.outcomes[i] {
		return nil, errors.New("db down")
	}
	if f.nilAt[i] {
		return nil, nil
	}
	return f.board, nil
}

func (f *flakyBoardLoader) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

type boardUnavailableFixture struct {
	server     *httptest.Server
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	playerUUID uuid.UUID
	loader     *flakyBoardLoader
}

func newBoardUnavailableFixture(t *testing.T, started bool, outcomes []bool, withBoard bool) *boardUnavailableFixture {
	t.Helper()
	f := &boardUnavailableFixture{
		matchUUID: uuid.New(), masterUUID: uuid.New(), playerUUID: uuid.New(),
		loader: &flakyBoardLoader{outcomes: outcomes},
	}
	if withBoard {
		f.loader.board = &matchboard.Board{
			MatchUUID: f.matchUUID,
			MapUUID:   uuid.New(),
			Grid: mapentity.GridShape{
				Kind: mapentity.GridKindSquare, Cols: 10, Rows: 10, CellSize: 64, SkewRatio: 1,
			},
		}
	}
	sheetUUID := uuid.New()
	participant := &match.Participant{
		UUID: uuid.New(), MatchUUID: f.matchUUID,
		Sheet: csEntity.Summary{UUID: sheetUUID, PlayerUUID: &f.playerUUID},
	}
	hub := game.NewHub()
	go hub.Run()
	handler := game.NewHandler(
		hub,
		&fogMatchRepo{masterUUID: f.masterUUID, started: started},
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
			LoadBoardUC:           f.loader,
		},
	)
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)
	f.server = httptest.NewServer(mux)
	t.Cleanup(func() {
		f.server.Close()
		hub.Stop()
	})
	return f
}

// waitLoads waits for the master's register to have called Load (Dial returns
// before Register runs) so the player cannot register first.
func (f *boardUnavailableFixture) waitLoads(t *testing.T, n int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for f.loader.count() < n {
		if time.Now().After(deadline) {
			t.Fatalf("Load calls never reached %d (got %d)", n, f.loader.count())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (f *boardUnavailableFixture) connect(t *testing.T, user uuid.UUID) *websocket.Conn {
	t.Helper()
	c := connectWS(t, f.server.URL, user, f.matchUUID)
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// drain reads every message until the socket stays quiet, so a test can assert on what was
// sent AND on what was not.
func drain(conn *websocket.Conn) []game.Message {
	var out []game.Message
	for {
		_ = conn.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
		_, data, err := conn.ReadMessage()
		if err != nil {
			return out
		}
		var m game.Message
		if json.Unmarshal(data, &m) == nil {
			out = append(out, m)
		}
	}
}

func countType(msgs []game.Message, typ game.MessageType) int {
	n := 0
	for _, m := range msgs {
		if m.Type == typ {
			n++
		}
	}
	return n
}

// countBoardUnavailable counts error messages whose code is board_unavailable.
func countBoardUnavailable(t *testing.T, msgs []game.Message) int {
	t.Helper()
	n := 0
	for _, m := range msgs {
		if m.Type != game.MsgTypeError {
			continue
		}
		var p game.ErrorPayload
		if err := json.Unmarshal(m.Payload, &p); err != nil {
			t.Fatalf("unmarshal error payload: %v", err)
		}
		if p.Code == "board_unavailable" {
			if p.Message != "the match board could not be loaded; reconnect to try again" {
				t.Errorf("unexpected board_unavailable message: %q", p.Message)
			}
			n++
		}
	}
	return n
}

func TestBoardUnavailable_MasterLoadFails_GetsErrorAndNoMap(t *testing.T) {
	f := newBoardUnavailableFixture(t, true, []bool{true}, true)
	msgs := drain(f.connect(t, f.masterUUID))
	if f.loader.count() != 1 {
		t.Fatalf("Load calls = %d, want 1", f.loader.count())
	}
	if got := countBoardUnavailable(t, msgs); got != 1 {
		t.Fatalf("board_unavailable count = %d, want 1", got)
	}
	if got := countType(msgs, game.MsgTypeMapFullState); got != 0 {
		t.Fatalf("map_full_state count = %d, want 0", got)
	}
}

func TestBoardUnavailable_PlayerRetrySucceeds_PlayerGetsMapAndMasterGetsRepush(t *testing.T) {
	f := newBoardUnavailableFixture(t, true, []bool{true}, true)
	// drain ends in a read-deadline error, which is sticky on a gorilla conn: each socket is
	// drained once, at the end, and sees everything it was sent.
	master := f.connect(t, f.masterUUID)
	f.waitLoads(t, 1)
	player := f.connect(t, f.playerUUID)

	pmsgs := drain(player)
	if got := countType(pmsgs, game.MsgTypeMapFullState); got != 1 {
		t.Fatalf("player map_full_state count = %d, want 1", got)
	}
	if got := countBoardUnavailable(t, pmsgs); got != 0 {
		t.Fatalf("player board_unavailable count = %d, want 0", got)
	}
	mmsgs := drain(master)
	if got := countBoardUnavailable(t, mmsgs); got != 1 {
		t.Fatalf("master board_unavailable count = %d, want 1", got)
	}
	if got := countType(mmsgs, game.MsgTypeMapFullState); got != 1 {
		t.Fatalf("master repush count = %d, want 1", got)
	}
	// Only the master's own register repushes match_full_state; the player's retry must not,
	// or every client's armed UI resets (the front reads each one as a reconnection).
	if got := countType(mmsgs, game.MsgTypeMatchFullState); got != 1 {
		t.Fatalf("master match_full_state count = %d, want 1 (its own register only)", got)
	}
	if f.loader.count() != 2 {
		t.Fatalf("Load calls = %d, want 2", f.loader.count())
	}
}

func TestBoardUnavailable_PlayerRetryFails_PlayerGetsError(t *testing.T) {
	f := newBoardUnavailableFixture(t, true, []bool{true, true}, true)
	master := f.connect(t, f.masterUUID)
	f.waitLoads(t, 1)
	pmsgs := drain(f.connect(t, f.playerUUID))
	if f.loader.count() != 2 {
		t.Fatalf("Load calls = %d, want 2 (retried and failed)", f.loader.count())
	}
	if got := countBoardUnavailable(t, pmsgs); got != 1 {
		t.Fatalf("player board_unavailable count = %d, want 1", got)
	}
	// The failed retry changed nothing: the master must not get a (empty) map_full_state that
	// would clear its failure state while the room still has no board.
	mmsgs := drain(master)
	if got := countBoardUnavailable(t, mmsgs); got != 1 {
		t.Fatalf("master board_unavailable count = %d, want 1", got)
	}
	if got := countType(mmsgs, game.MsgTypeMapFullState); got != 0 {
		t.Fatalf("master map_full_state count = %d, want 0 after a failed retry", got)
	}
	if got := countType(mmsgs, game.MsgTypeMatchFullState); got != 1 {
		t.Fatalf("master match_full_state count = %d, want 1 (its own register only)", got)
	}
	if got := countType(pmsgs, game.MsgTypeMapFullState); got != 0 {
		t.Fatalf("player map_full_state count = %d, want 0", got)
	}
}

func TestBoardUnavailable_InstalledBoard_PlayerNeverReloadsNorGetsError(t *testing.T) {
	// Lobby: the master loads fine, reconnects, and the reload fails — the room keeps the
	// board it has. A player connecting afterwards must not trigger a load or see the error.
	f := newBoardUnavailableFixture(t, false, []bool{false, true}, true)
	f.connect(t, f.masterUUID)
	f.waitLoads(t, 1)
	master2 := f.connect(t, f.masterUUID)
	m2 := drain(master2)
	if got := countBoardUnavailable(t, m2); got != 0 {
		t.Fatalf("master reload failure with board installed sent board_unavailable %d times", got)
	}
	f.waitLoads(t, 2)
	before := f.loader.count()
	pmsgs := drain(f.connect(t, f.playerUUID))
	if f.loader.count() != before {
		t.Fatalf("player connect called Load: %d -> %d", before, f.loader.count())
	}
	if got := countBoardUnavailable(t, pmsgs); got != 0 {
		t.Fatalf("player board_unavailable count = %d, want 0", got)
	}
}

func TestBoardUnavailable_NoMapAttached_NoErrorNoPlayerLoad(t *testing.T) {
	f := newBoardUnavailableFixture(t, true, nil, false)
	mmsgs := drain(f.connect(t, f.masterUUID))
	if got := countBoardUnavailable(t, mmsgs); got != 0 {
		t.Fatalf("master board_unavailable = %d, want 0", got)
	}
	before := f.loader.count()
	pmsgs := drain(f.connect(t, f.playerUUID))
	if f.loader.count() != before {
		t.Fatalf("player connect called Load: %d -> %d", before, f.loader.count())
	}
	if got := countBoardUnavailable(t, pmsgs); got != 0 {
		t.Fatalf("player board_unavailable = %d, want 0", got)
	}
}

func TestBoardUnavailable_LobbyBoardClearedThenLoadFails_MasterGetsErrorAndPlayerRetries(t *testing.T) {
	// Lobby: call 0 installs the board; call 1 finds the map detached (nil, nil) and clears it
	// (boardLoaded stays true); call 2 (a later master reconnect) FAILS. The room has no board
	// now, so the master must be told, and a player connecting must retry the load.
	f := newBoardUnavailableFixture(t, false, []bool{false, false, true}, true)
	f.loader.nilAt = map[int]bool{1: true}
	f.connect(t, f.masterUUID)
	f.waitLoads(t, 1)
	f.connect(t, f.masterUUID)
	f.waitLoads(t, 2)
	master3 := f.connect(t, f.masterUUID)
	f.waitLoads(t, 3)
	m3 := drain(master3)
	if got := countBoardUnavailable(t, m3); got != 1 {
		t.Fatalf("master board_unavailable count = %d, want 1", got)
	}
	before := f.loader.count()
	pmsgs := drain(f.connect(t, f.playerUUID))
	if f.loader.count() != before+1 {
		t.Fatalf("player connect Load calls: %d -> %d, want one retry", before, f.loader.count())
	}
	if got := countType(pmsgs, game.MsgTypeMapFullState); got != 1 {
		t.Fatalf("player map_full_state count = %d, want 1 after a successful retry", got)
	}
	if got := countBoardUnavailable(t, pmsgs); got != 0 {
		t.Fatalf("player board_unavailable count = %d, want 0", got)
	}
}
