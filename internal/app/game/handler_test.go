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

func setupTestServer(masterUUID uuid.UUID, enrolled bool) (*httptest.Server, *game.Hub) {
	hub := game.NewHub()
	go hub.Run()

	matchRepo := &mockMatchRepo{masterUUID: masterUUID}
	enrollmentRepo := &mockEnrollmentChecker{enrolled: enrolled}
	handler := game.NewHandler(hub, matchRepo, enrollmentRepo)

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
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
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
	defer conn.Close()

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

	conn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer conn.Close()

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
	defer masterConn.Close()
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()
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
	defer masterConn.Close()
	_ = readMessage(t, masterConn) // room_state

	time.Sleep(50 * time.Millisecond)

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()
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

	playerConn := connectWS(t, server.URL, playerUUID, matchUUID)
	defer playerConn.Close()
	_ = readMessage(t, playerConn) // room_state

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
