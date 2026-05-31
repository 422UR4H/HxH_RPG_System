package game_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	scene "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
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

type mockChangeSceneUCHandler struct{}

func (m *mockChangeSceneUCHandler) Execute(_ context.Context, _ *matchsession.MatchSession, _, _ uuid.UUID, _ enum.SceneCategory, _ string) (*scene.Scene, *roundentity.Round, error) {
	return scene.NewScene(enum.Roleplay, ""), roundentity.NewRound(enum.Free), nil
}

type mockRoundRepoHandler struct{}

func (m *mockRoundRepoHandler) PersistTurnClose(_ context.Context, _ *scene.Scene, _ *roundentity.Round, _ *turnentity.Turn, _ *action.Action, _ uuid.UUID) error {
	return nil
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
		&mockChangeSceneUCHandler{},
		&mockRoundRepoHandler{},
		&mockEnqueueMasterActionUCHandler{},
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
	defer masterConn.Close() //nolint:errcheck
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
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state

	_ = readMessage(t, masterConn) // player_joined

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
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // player_joined

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
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // player_joined

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
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // player_joined

	kickMsg := game.Message{
		Type:    game.MsgTypeKickPlayer,
		Payload: json.RawMessage(`{"player_uuid":"` + playerUUID.String() + `"}`),
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
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // player_joined

	kickMsg := game.Message{
		Type:    game.MsgTypeKickPlayer,
		Payload: json.RawMessage(`{"player_uuid":"` + masterUUID.String() + `"}`),
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
	defer masterConn.Close() //nolint:errcheck
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
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // player_joined

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
	defer masterConn.Close() //nolint:errcheck
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close() //nolint:errcheck
	_ = readMessage(t, playerConn) // room_state
	_ = readMessage(t, masterConn) // player_joined

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
