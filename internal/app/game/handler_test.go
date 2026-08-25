package game_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	scene "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	pkgAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type mockMatchRepo struct {
	masterUUID uuid.UUID
	err        error
}

func (m *mockMatchRepo) GetMatchMaster(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return m.masterUUID, m.err
}

func (m *mockMatchRepo) IsStarted(_ context.Context, _ uuid.UUID) (bool, error) {
	return false, nil
}

type mockEnrollmentChecker struct {
	enrolled bool
	err      error
}

func (m *mockEnrollmentChecker) IsPlayerEnrolledInMatch(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.enrolled, m.err
}

type mockStartMatchUC struct {
	err error
}

func (m *mockStartMatchUC) Start(_ context.Context, _, _ uuid.UUID) error {
	return m.err
}

type mockKickPlayerUC struct {
	err error
}

func (m *mockKickPlayerUC) Kick(_ context.Context, _, _, _ uuid.UUID) error {
	return m.err
}

type mockInitSessionUCHandler struct{}

func (m *mockInitSessionUCHandler) Init(_ context.Context, _ uuid.UUID) (*matchsession.MatchSession, error) {
	return matchsession.NewMatchSession(uuid.New(), nil, nil), nil
}

type mockOpenNextActionUCHandler struct{}

func (m *mockOpenNextActionUCHandler) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID) (*appmatch.OpenNextActionResult, error) {
	return nil, nil
}

type mockPullActionUCHandler struct{}

func (m *mockPullActionUCHandler) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID, _ uuid.UUID) (*appmatch.PullActionResult, error) {
	return nil, nil
}

type mockEnqueueActionUCHandler struct{}

func (m *mockEnqueueActionUCHandler) Execute(_ context.Context, _ *matchsession.MatchSession, _ uuid.UUID, _ *action.Action) error {
	return nil
}

type mockAttachReactionUCHandler struct{}

func (m *mockAttachReactionUCHandler) Execute(_ context.Context, _ *matchsession.MatchSession, _ uuid.UUID, _ *action.Action) (*appmatch.AttachReactionResult, error) {
	return nil, nil
}

type mockOpenReactionUCHandler struct{}

func (m *mockOpenReactionUCHandler) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID) (*appmatch.OpenReactionResult, error) {
	return nil, nil
}

type mockCloseTurnUCHandler struct{}

func (m *mockCloseTurnUCHandler) Execute(
	_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID, _ bool,
) (*appmatch.CloseTurnResult, error) {
	return nil, nil
}

type mockChangeSceneUCHandler struct{}

func (m *mockChangeSceneUCHandler) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID, _ enum.SceneCategory, _ string) (*scene.Scene, *roundentity.Round, error) {
	return scene.NewScene(enum.Roleplay, ""), roundentity.NewRound(enum.Free), nil
}

// mockRoundRepoHandler is a no-op round repository, except it records every closed turn
// ID PersistTurnClose was called with — tests that need to assert a turn actually reached
// persistence (rather than just that Execute returned no error) read persistedTurnIDs()
// instead of adding a second test double.
type mockRoundRepoHandler struct {
	mu             sync.Mutex
	persistedTurns []uuid.UUID
}

func (m *mockRoundRepoHandler) PersistTurnClose(_ context.Context, d appmatch.TurnCloseData) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.persistedTurns = append(m.persistedTurns, d.Turn.GetID())
	return nil
}

// persistedTurnIDs returns a snapshot of every turn ID PersistTurnClose was called with.
func (m *mockRoundRepoHandler) persistedTurnIDs() []uuid.UUID {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]uuid.UUID(nil), m.persistedTurns...)
}
func (m *mockRoundRepoHandler) FindActiveSession(_ context.Context, _ uuid.UUID) (*matchsession.ActiveSessionData, error) {
	return nil, nil
}
func (m *mockRoundRepoHandler) CloseSceneAndRound(_ context.Context, _, _ uuid.UUID, _ time.Time) error {
	return nil
}
func (m *mockRoundRepoHandler) CloseRound(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

type mockEnqueueMasterActionUCHandler struct{}

func (m *mockEnqueueMasterActionUCHandler) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID, _ *action.MasterAction) error {
	return nil
}

type mockChangeRoundModeUCHandler struct{}

func (m *mockChangeRoundModeUCHandler) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID, _ enum.RoundMode) error {
	return nil
}

type mockEditActionUCHandler struct{}

func (m *mockEditActionUCHandler) Execute(
	_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID, _ *action.MasterAction,
) (*appmatch.EditActionResult, error) {
	// Non-nil: a test that actually exercises edit_action through a room built with this mock
	// must fail on an assertion, not panic on a nil-pointer dereference of the result.
	return &appmatch.EditActionResult{Resolution: &service.TurnResolution{}, TurnID: uuid.New()}, nil
}

func setupTestServer(masterUUID uuid.UUID, enrolled bool) (*httptest.Server, *game.Hub) {
	hub := game.NewHub()
	go hub.Run()

	matchRepo := &mockMatchRepo{masterUUID: masterUUID}
	enrollmentRepo := &mockEnrollmentChecker{enrolled: enrolled}
	startUC := &mockStartMatchUC{}
	kickUC := &mockKickPlayerUC{}
	handler := game.NewHandler(
		hub, matchRepo, enrollmentRepo,
		startUC, kickUC,
		&mockInitSessionUCHandler{},
		&mockOpenNextActionUCHandler{},
		&mockPullActionUCHandler{},
		&mockEnqueueActionUCHandler{},
		&mockAttachReactionUCHandler{},
		&mockOpenReactionUCHandler{},
		&mockCloseTurnUCHandler{},
		&mockChangeSceneUCHandler{},
		&mockRoundRepoHandler{},
		&mockEnqueueMasterActionUCHandler{},
		&mockChangeRoundModeUCHandler{},
		&mockEditActionUCHandler{},
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handler.HandleWebSocket)

	server := httptest.NewServer(mux)
	return server, hub
}

func connectWS(t *testing.T, serverURL string, userUUID uuid.UUID, matchUUID uuid.UUID) *websocket.Conn {
	t.Helper()

	token, err := pkgAuth.GenerateToken(userUUID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") +
		"/ws?token=" + token +
		"&match_uuid=" + matchUUID.String() +
		"&nickname=testplayer"

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		if resp != nil {
			t.Fatalf("websocket dial failed with status %d: %v", resp.StatusCode, err)
		}
		t.Fatalf("websocket dial failed: %v", err)
	}
	return conn
}

func readMessage(t *testing.T, conn *websocket.Conn) game.Message {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read message: %v", err)
	}
	var msg game.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("failed to unmarshal message: %v", err)
	}
	return msg
}

func TestHandlerRejectsNoToken(t *testing.T) {
	masterUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	matchUUID := uuid.New()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") +
		"/ws?match_uuid=" + matchUUID.String()

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected connection to fail without token")
	}
	if resp != nil && resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandlerRejectsNoMatchUUID(t *testing.T) {
	masterUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	token, _ := pkgAuth.GenerateToken(masterUUID)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?token=" + token

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected connection to fail without match_uuid")
	}
	if resp != nil && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestHandlerRejectsUnenrolledPlayer(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, false)
	defer server.Close()
	defer hub.Stop()

	matchUUID := uuid.New()
	token, _ := pkgAuth.GenerateToken(playerUUID)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") +
		"/ws?token=" + token +
		"&match_uuid=" + matchUUID.String()

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected connection to fail for unenrolled player")
	}
	if resp != nil && resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestMasterCanConnect(t *testing.T) {
	masterUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, false)
	defer server.Close()
	defer hub.Stop()

	conn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer conn.Close() //nolint:errcheck

	msg := readMessage(t, conn)
	if msg.Type != game.MsgTypeRoomState {
		t.Errorf("expected room_state message, got %s", msg.Type)
	}

	var roomState game.RoomStatePayload
	if err := json.Unmarshal(msg.Payload, &roomState); err != nil {
		t.Fatalf("failed to unmarshal room state: %v", err)
	}
	if roomState.State != "lobby" {
		t.Errorf("expected lobby state, got %s", roomState.State)
	}
}

func TestPlayerCanConnect(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	// master must open the lobby first
	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()       //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	conn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer conn.Close() //nolint:errcheck

	msg := readMessage(t, conn)
	if msg.Type != game.MsgTypeRoomState {
		t.Errorf("expected room_state, got %s", msg.Type)
	}
}

func TestChatFlow(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()       //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()       //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state

	_ = readMessage(t, masterConn) // master_joined

	chatMsg := game.Message{
		Type:    game.MsgTypeChat,
		Payload: json.RawMessage(`{"message":"hello master!"}`),
	}
	data, _ := json.Marshal(chatMsg)
	if err := playerConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send chat: %v", err)
	}

	received := readMessage(t, masterConn)
	if received.Type != game.MsgTypeChatMessage {
		t.Errorf("expected chat_message, got %s", received.Type)
	}
	if received.SenderID != playerUUID {
		t.Errorf("expected sender %s, got %s", playerUUID, received.SenderID)
	}
}

func TestStartMatchFlow(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()       //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()       //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // master_joined

	startMsg := game.Message{
		Type:    game.MsgTypeStartMatch,
		Payload: json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(startMsg)
	if err := masterConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send start_match: %v", err)
	}

	received := readMessage(t, playerConn)
	if received.Type != game.MsgTypeMatchStarted {
		t.Errorf("expected match_started, got %s", received.Type)
	}

	masterReceived := readMessage(t, masterConn)
	if masterReceived.Type != game.MsgTypeMatchStarted {
		t.Errorf("expected master to get match_started, got %s", masterReceived.Type)
	}
}

func TestPlayerCannotStartMatch(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()       //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()       //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // master_joined

	startMsg := game.Message{
		Type:    game.MsgTypeStartMatch,
		Payload: json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(startMsg)
	if err := playerConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send start_match: %v", err)
	}

	received := readMessage(t, playerConn)
	if received.Type != game.MsgTypeError {
		t.Errorf("expected error, got %s", received.Type)
	}
}

func TestKickPlayerFlow(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()       //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()       //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // master_joined

	kickMsg := game.Message{
		Type:    game.MsgTypeKickPlayer,
		Payload: json.RawMessage(`{"playerUuid":"` + playerUUID.String() + `"}`),
	}
	data, _ := json.Marshal(kickMsg)
	if err := masterConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send kick_player: %v", err)
	}

	playerReceived := readMessage(t, playerConn)
	if playerReceived.Type != game.MsgTypePlayerKicked {
		t.Errorf("expected player_kicked, got %s", playerReceived.Type)
	}

	masterReceived := readMessage(t, masterConn)
	if masterReceived.Type != game.MsgTypePlayerKicked {
		t.Errorf("expected master to get player_kicked, got %s", masterReceived.Type)
	}
}

func TestPlayerCannotKick(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()       //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()       //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // master_joined

	kickMsg := game.Message{
		Type:    game.MsgTypeKickPlayer,
		Payload: json.RawMessage(`{"playerUuid":"` + masterUUID.String() + `"}`),
	}
	data, _ := json.Marshal(kickMsg)
	if err := playerConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send kick_player: %v", err)
	}

	received := readMessage(t, playerConn)
	if received.Type != game.MsgTypeError {
		t.Errorf("expected error, got %s", received.Type)
	}
}

func TestPlayerGetsLobbyNotOpenWhenNoRoom(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	// player connects without master having opened the lobby first
	conn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer conn.Close() //nolint:errcheck

	msg := readMessage(t, conn)
	if msg.Type != game.MsgTypeLobbyNotOpen {
		t.Errorf("expected lobby_not_open, got %s", msg.Type)
	}
}

func TestLobbyNotOpenTextFrameArrivesBeforeCloseFrame(t *testing.T) {
	// Regression: conn.Close() was called immediately after the close frame,
	// which could send a TCP RST before the browser processed the text frame.
	// The goroutine drain in handler.go ensures the text frame is delivered first.
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	conn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer conn.Close() //nolint:errcheck

	// The FIRST thing received must be the lobby_not_open text frame,
	// not a close frame (which would be delivered as an error from ReadMessage).
	conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:errcheck
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("expected lobby_not_open text frame first, got read error: %v", err)
	}
	if msgType != websocket.TextMessage {
		t.Errorf("expected text frame first, got frame type %d", msgType)
	}
	var msg game.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("failed to unmarshal first message: %v", err)
	}
	if msg.Type != game.MsgTypeLobbyNotOpen {
		t.Errorf("expected lobby_not_open in text frame, got %s", msg.Type)
	}
}

func TestPlayerCanConnectAfterMasterOpensLobby(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	// master opens lobby first
	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()       //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	// now player can connect
	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck

	msg := readMessage(t, playerConn)
	if msg.Type != game.MsgTypeRoomState {
		t.Errorf("expected room_state, got %s", msg.Type)
	}
}

func TestMasterReceivesLobbyClosed(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()       //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()       //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // master_joined

	cancelMsg := game.Message{
		Type:    game.MsgTypeCancelLobby,
		Payload: json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(cancelMsg)
	if err := masterConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send cancel_lobby: %v", err)
	}

	received := readMessage(t, playerConn)
	if received.Type != game.MsgTypeLobbyClosed {
		t.Errorf("expected lobby_closed for player, got %s", received.Type)
	}

	masterReceived := readMessage(t, masterConn)
	if masterReceived.Type != game.MsgTypeLobbyClosed {
		t.Errorf("expected master to get lobby_closed, got %s", masterReceived.Type)
	}
}

func TestPlayerCannotCancelLobby(t *testing.T) {
	masterUUID := uuid.New()
	playerUUID := uuid.New()
	matchUUID := uuid.New()
	server, hub := setupTestServer(masterUUID, true)
	defer server.Close()
	defer hub.Stop()

	masterConn := connectWS(t, server.URL, masterUUID, matchUUID)
	defer masterConn.Close()       //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()       //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // master_joined

	cancelMsg := game.Message{
		Type:    game.MsgTypeCancelLobby,
		Payload: json.RawMessage(`{}`),
	}
	data, _ := json.Marshal(cancelMsg)
	if err := playerConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("failed to send cancel_lobby: %v", err)
	}

	received := readMessage(t, playerConn)
	if received.Type != game.MsgTypeError {
		t.Errorf("expected error, got %s", received.Type)
	}
}

// TestE2E_CloseTurn drives close_turn over a real socket: a non-master is refused, the master
// gets the server-computed refusal naming the one reaction that was attached and never opened,
// and the same master resending with confirm:true actually closes the turn.
//
// Reuses reaction_chain_e2e_test.go's areaFixture — the same real Room, real use cases and
// wsConn helpers the reaction-chain tests already exercise close_turn's neighbours with.
func TestE2E_CloseTurn(t *testing.T) {
	faces := []int{
		6, 4, 1, 1, // Attack.Hit: primary 6,4 (→10); secondary 1,1 (unused filler)
		7, 3, // Sword damage: D10=7, D4=3 → raw 7+3+2=12
	}
	f := newAreaFixture(t, faces)
	defer f.server.Close()

	sword := "Sword"
	f.attacker.send(t, game.MsgTypeEnqueueAction, game.ActionPayload{
		ActorID:  f.attackerID,
		TargetID: []uuid.UUID{f.a},
		Attack: &game.AttackPayload{
			Weapon: &sword,
			Hit:    game.RollCheckPayload{SkillName: "Accuracy"},
			Damage: game.RollCheckPayload{},
		},
	})
	if !f.attacker.msgs.await(game.MsgTypeActionEnqueued, 2*time.Second) {
		t.Fatal("the attack was never acknowledged as enqueued")
	}
	f.master.send(t, game.MsgTypeOpenNextAction, struct{}{})
	_, actionID := f.awaitTurnOpened(t)

	// A attaches a reaction and the master never opens it — exactly the scenario close_turn's
	// refusal exists for.
	reactionID := f.attachReaction(t, f.pA, game.ActionPayload{
		ActorID: f.a, ReactToID: actionID, ReactionKind: "nothing",
	})

	t.Run("a non-master gets forbidden", func(t *testing.T) {
		f.pA.send(t, game.MsgTypeCloseTurn, game.CloseTurnPayload{})
		if !f.pA.msgs.await(game.MsgTypeError, 2*time.Second) {
			t.Fatal("the non-master never got an error back")
		}
	})

	t.Run("the master gets close_turn_refused naming the pending reaction", func(t *testing.T) {
		before := f.master.msgs.count(game.MsgTypeCloseTurnRefused)
		f.master.send(t, game.MsgTypeCloseTurn, game.CloseTurnPayload{})
		if !awaitCount(f.master.msgs, game.MsgTypeCloseTurnRefused, before+1, 2*time.Second) {
			t.Fatal("the master never got close_turn_refused")
		}
		msgs := f.master.msgs.snapshotMessages()
		var refused *game.CloseTurnRefusedPayload
		for i := len(msgs) - 1; i >= 0; i-- {
			if msgs[i].Type != game.MsgTypeCloseTurnRefused {
				continue
			}
			var p game.CloseTurnRefusedPayload
			if err := json.Unmarshal(msgs[i].Payload, &p); err != nil {
				t.Fatalf("unmarshal close_turn_refused: %v", err)
			}
			refused = &p
			break
		}
		if refused == nil {
			t.Fatal("no close_turn_refused message found")
		}
		if len(refused.PendingReactions) != 1 || refused.PendingReactions[0].ReactionID != reactionID {
			t.Fatalf("PendingReactions = %+v, want exactly [%v]", refused.PendingReactions, reactionID)
		}
	})

	t.Run("resending with confirm:true closes it", func(t *testing.T) {
		f.master.send(t, game.MsgTypeCloseTurn, game.CloseTurnPayload{Confirm: true})
		if !f.master.msgs.await(game.MsgTypeTurnClosed, 2*time.Second) {
			t.Fatal("the master never got turn_closed")
		}
	})
}

// TestActionQueuedReachesOnlyTheMaster closes the hole pull_action left open since it was
// written: the master has to learn an action's ID from somewhere before it can send it back.
// This proves both halves — the master gets a usable ID, and a player never sees the message
// at all, because the queue is secret.
func TestActionQueuedReachesOnlyTheMaster(t *testing.T) {
	f := newCombatFixture(t)
	master, player := f.connect(t)
	masterMsgs := newCollector(master)
	playerMsgs := newCollector(player)

	f.enqueueAttack(t, player)

	if !masterMsgs.await(game.MsgTypeActionQueued, 2*time.Second) {
		t.Fatal("the master was never told an action was queued")
	}
	var p game.ActionQueuedPayload
	for _, m := range masterMsgs.snapshotMessages() {
		if m.Type == game.MsgTypeActionQueued {
			if err := json.Unmarshal(m.Payload, &p); err != nil {
				t.Fatalf("unmarshal action_queued: %v", err)
			}
		}
	}
	if p.ActionID == uuid.Nil {
		t.Fatal("action_queued carried no action ID — pull_action stays unreachable")
	}
	if p.ActorID != f.attackerID {
		t.Fatalf("ActorID = %v, want the attacking character %v", p.ActorID, f.attackerID)
	}
	if n := playerMsgs.count(game.MsgTypeActionQueued); n != 0 {
		t.Fatalf("a player received %d action_queued; the queue is secret", n)
	}

	// And the ID is usable: pull_action with it opens that exact turn.
	sendWS(t, master, "pull_action", map[string]any{"actionId": p.ActionID})
	if !masterMsgs.await(game.MsgTypeTurnOpened, 2*time.Second) {
		t.Fatal("pull_action with the advertised ID opened nothing")
	}
}
