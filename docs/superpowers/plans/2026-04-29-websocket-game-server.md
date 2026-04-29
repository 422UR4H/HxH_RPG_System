# WebSocket Game Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a real-time WebSocket game server with Hub/Room/Client architecture for match execution.

**Architecture:** gorilla/websocket (already in go.mod) with a Hub managing Rooms (one per match). Each Room has a state machine (lobby → playing → closed). Clients authenticate via JWT on connection, validated against match enrollment in PostgreSQL.

**Tech Stack:** Go 1.25, gorilla/websocket v1.5.3, go-chi/chi v5, pgx/v5, JWT (pkg/auth)

---

## File Structure

```
internal/app/game/
├── message.go      — Message envelope + types (no deps)
├── client.go       — Client struct + readPump/writePump
├── room.go         — Room struct + state machine + broadcast
├── hub.go          — Hub struct + room management
├── handler.go      — HTTP upgrade + JWT auth + enrollment validation
└── server.go       — NewGameServer() + chi router setup

cmd/game/main.go    — Rewritten entry point (replaces prototype)
```

**Dependency order:** message → client → room → hub → handler → server → main

---

### Task 1: Message Types (`internal/app/game/message.go`)

**Files:**
- Create: `internal/app/game/message.go`
- Test: `internal/app/game/message_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/app/game/message_test.go
package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

func TestMessage_MarshalJSON(t *testing.T) {
	senderID := uuid.New()
	now := time.Now().Truncate(time.Second)

	msg := game.Message{
		Type:      game.MsgTypeMatchStarted,
		Payload:   json.RawMessage(`{}`),
		SenderID:  senderID,
		Timestamp: now,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded game.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if decoded.Type != game.MsgTypeMatchStarted {
		t.Errorf("got type %q, want %q", decoded.Type, game.MsgTypeMatchStarted)
	}
	if decoded.SenderID != senderID {
		t.Errorf("got sender %v, want %v", decoded.SenderID, senderID)
	}
}

func TestNewServerMessage(t *testing.T) {
	payload := map[string]string{"message": "hello"}
	msg := game.NewServerMessage(game.MsgTypeChatMessage, payload)

	if msg.Type != game.MsgTypeChatMessage {
		t.Errorf("got type %q, want %q", msg.Type, game.MsgTypeChatMessage)
	}
	if msg.SenderID != uuid.Nil {
		t.Errorf("expected Nil sender for server message, got %v", msg.SenderID)
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestNewErrorMessage(t *testing.T) {
	msg := game.NewErrorMessage("forbidden", "Only master can start match")

	if msg.Type != game.MsgTypeError {
		t.Errorf("got type %q, want %q", msg.Type, game.MsgTypeError)
	}

	var payload game.ErrorPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		t.Fatalf("Unmarshal payload error = %v", err)
	}
	if payload.Code != "forbidden" {
		t.Errorf("got code %q, want %q", payload.Code, "forbidden")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run TestMessage -v`
Expected: FAIL — package does not exist yet

- [ ] **Step 3: Write implementation**

```go
// internal/app/game/message.go
package game

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type MessageType string

const (
	// Server → Client
	MsgTypeRoomState    MessageType = "room_state"
	MsgTypePlayerJoined MessageType = "player_joined"
	MsgTypePlayerLeft   MessageType = "player_left"
	MsgTypeMatchStarted MessageType = "match_started"
	MsgTypeChatMessage  MessageType = "chat_message"
	MsgTypeError        MessageType = "error"

	// Client → Server
	MsgTypeStartMatch MessageType = "start_match"
	MsgTypeChat       MessageType = "chat"
)

type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	SenderID  uuid.UUID       `json:"sender_id"`
	Timestamp time.Time       `json:"timestamp"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PlayerPayload struct {
	UUID     uuid.UUID `json:"uuid"`
	Nickname string    `json:"nickname"`
}

type RoomStatePayload struct {
	MatchUUID uuid.UUID      `json:"match_uuid"`
	State     string         `json:"state"`
	Players   []PlayerInfo   `json:"players"`
}

type PlayerInfo struct {
	UUID     uuid.UUID `json:"uuid"`
	Nickname string    `json:"nickname"`
	IsMaster bool      `json:"is_master"`
	IsOnline bool      `json:"is_online"`
}

type ChatPayload struct {
	Message string `json:"message"`
}

func NewServerMessage(msgType MessageType, payload any) Message {
	data, _ := json.Marshal(payload)
	return Message{
		Type:      msgType,
		Payload:   data,
		SenderID:  uuid.Nil,
		Timestamp: time.Now(),
	}
}

func NewClientMessage(msgType MessageType, senderID uuid.UUID, payload any) Message {
	data, _ := json.Marshal(payload)
	return Message{
		Type:      msgType,
		Payload:   data,
		SenderID:  senderID,
		Timestamp: time.Now(),
	}
}

func NewErrorMessage(code, message string) Message {
	return NewServerMessage(MsgTypeError, ErrorPayload{
		Code:    code,
		Message: message,
	})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/game/ -run TestMessage -v && go test ./internal/app/game/ -run TestNew -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/message.go internal/app/game/message_test.go
git commit -m "feat(game): message types and JSON protocol

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Client (`internal/app/game/client.go`)

**Files:**
- Create: `internal/app/game/client.go`
- Test: `internal/app/game/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/app/game/client_test.go
package game_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func TestClient_WritePump_SendsMessages(t *testing.T) {
	// Setup a WS server that creates a client and sends a message
	var serverConn *websocket.Conn
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		serverConn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade error: %v", err)
		}
	}))
	defer server.Close()

	// Connect client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer clientConn.Close()

	// Wait for server to have the connection
	time.Sleep(50 * time.Millisecond)

	userID := uuid.New()
	client := game.NewClient(userID, serverConn, "TestPlayer")
	go client.WritePump()

	// Send a message via the client's send channel
	msg := game.NewServerMessage(game.MsgTypeMatchStarted, map[string]string{})
	client.SendMessage(msg)

	// Read from the client side
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if !strings.Contains(string(data), "match_started") {
		t.Errorf("expected match_started message, got: %s", data)
	}

	client.Close()
}

func TestClient_GetUserUUID(t *testing.T) {
	userID := uuid.New()
	client := game.NewClient(userID, nil, "TestPlayer")

	if client.GetUserUUID() != userID {
		t.Errorf("got %v, want %v", client.GetUserUUID(), userID)
	}
}

func TestClient_GetNickname(t *testing.T) {
	client := game.NewClient(uuid.New(), nil, "Gon")

	if client.GetNickname() != "Gon" {
		t.Errorf("got %q, want %q", client.GetNickname(), "Gon")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run TestClient -v`
Expected: FAIL — `NewClient` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/app/game/client.go
package game

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type Client struct {
	userUUID uuid.UUID
	nickname string
	conn     *websocket.Conn
	send     chan []byte
	room     *Room
	done     chan struct{}
}

func NewClient(userUUID uuid.UUID, conn *websocket.Conn, nickname string) *Client {
	return &Client{
		userUUID: userUUID,
		nickname: nickname,
		conn:     conn,
		send:     make(chan []byte, 256),
		done:     make(chan struct{}),
	}
}

func (c *Client) GetUserUUID() uuid.UUID {
	return c.userUUID
}

func (c *Client) GetNickname() string {
	return c.nickname
}

func (c *Client) SetRoom(room *Room) {
	c.room = room
}

func (c *Client) SendMessage(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("error marshaling message: %v", err)
		return
	}
	select {
	case c.send <- data:
	default:
		log.Printf("client %s send buffer full, dropping message", c.userUUID)
	}
}

func (c *Client) Close() {
	select {
	case <-c.done:
	default:
		close(c.done)
	}
}

func (c *Client) ReadPump() {
	defer func() {
		if c.room != nil {
			c.room.unregister <- c
		}
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, rawMsg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("websocket error for user %s: %v", c.userUUID, err)
			}
			break
		}
		if c.room != nil {
			c.room.handleClientMessage(c, rawMsg)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/game/ -run TestClient -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/client.go internal/app/game/client_test.go
git commit -m "feat(game): Client with readPump/writePump

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Room (`internal/app/game/room.go`)

**Files:**
- Create: `internal/app/game/room.go`
- Test: `internal/app/game/room_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/app/game/room_test.go
package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

func TestNewRoom(t *testing.T) {
	matchID := uuid.New()
	masterID := uuid.New()

	room := game.NewRoom(matchID, masterID)

	if room.GetMatchUUID() != matchID {
		t.Errorf("got matchUUID %v, want %v", room.GetMatchUUID(), matchID)
	}
	if room.GetState() != game.RoomStateLobby {
		t.Errorf("got state %v, want %v", room.GetState(), game.RoomStateLobby)
	}
}

func TestRoom_IsMaster(t *testing.T) {
	masterID := uuid.New()
	room := game.NewRoom(uuid.New(), masterID)

	if !room.IsMaster(masterID) {
		t.Error("expected true for master UUID")
	}
	if room.IsMaster(uuid.New()) {
		t.Error("expected false for non-master UUID")
	}
}

func TestRoom_RegisterClient(t *testing.T) {
	masterID := uuid.New()
	room := game.NewRoom(uuid.New(), masterID)
	go room.Run()
	defer room.Stop()

	client := game.NewClient(masterID, nil, "Master")
	room.Register(client)

	time.Sleep(50 * time.Millisecond)

	if room.ClientCount() != 1 {
		t.Errorf("got %d clients, want 1", room.ClientCount())
	}
}

func TestRoom_StartMatch_OnlyMaster(t *testing.T) {
	masterID := uuid.New()
	playerID := uuid.New()
	room := game.NewRoom(uuid.New(), masterID)
	go room.Run()
	defer room.Stop()

	// Try to start as player — should fail
	err := room.StartMatch(playerID)
	if err == nil {
		t.Error("expected error when player tries to start match")
	}

	// Start as master — should succeed
	err = room.StartMatch(masterID)
	if err != nil {
		t.Fatalf("StartMatch() error = %v", err)
	}
	if room.GetState() != game.RoomStatePlaying {
		t.Errorf("got state %v, want %v", room.GetState(), game.RoomStatePlaying)
	}
}

func TestRoom_StartMatch_AlreadyPlaying(t *testing.T) {
	masterID := uuid.New()
	room := game.NewRoom(uuid.New(), masterID)
	go room.Run()
	defer room.Stop()

	_ = room.StartMatch(masterID)
	err := room.StartMatch(masterID)
	if err == nil {
		t.Error("expected error when match already started")
	}
}

func TestRoom_BroadcastChat(t *testing.T) {
	masterID := uuid.New()
	room := game.NewRoom(uuid.New(), masterID)
	go room.Run()
	defer room.Stop()

	// Create a mock client that captures messages
	client := game.NewClient(masterID, nil, "Master")
	room.Register(client)
	time.Sleep(50 * time.Millisecond)

	// Broadcast a chat message
	chatMsg := game.NewClientMessage(game.MsgTypeChatMessage, masterID, game.ChatPayload{Message: "Hello!"})
	data, _ := json.Marshal(chatMsg)
	room.Broadcast(data)

	// Give time for broadcast
	time.Sleep(50 * time.Millisecond)

	// Verify client received it (check send channel)
	select {
	case msg := <-client.GetSendChan():
		if msg == nil {
			t.Error("received nil message")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout waiting for broadcast message")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run TestRoom -v -timeout 10s`
Expected: FAIL — `NewRoom` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/app/game/room.go
package game

import (
	"encoding/json"
	"errors"
	"log"
	"sync"

	"github.com/google/uuid"
)

type RoomState string

const (
	RoomStateLobby   RoomState = "lobby"
	RoomStatePlaying RoomState = "playing"
	RoomStateClosed  RoomState = "closed"
)

var (
	ErrNotMaster      = errors.New("only the master can perform this action")
	ErrAlreadyPlaying = errors.New("match already started")
	ErrRoomClosed     = errors.New("room is closed")
)

type Room struct {
	matchUUID  uuid.UUID
	masterUUID uuid.UUID
	state      RoomState
	clients    map[uuid.UUID]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stop       chan struct{}
	mu         sync.RWMutex
}

func NewRoom(matchUUID, masterUUID uuid.UUID) *Room {
	return &Room{
		matchUUID:  matchUUID,
		masterUUID: masterUUID,
		state:      RoomStateLobby,
		clients:    make(map[uuid.UUID]*Client),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
	}
}

func (r *Room) GetMatchUUID() uuid.UUID {
	return r.matchUUID
}

func (r *Room) GetState() RoomState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

func (r *Room) IsMaster(userUUID uuid.UUID) bool {
	return r.masterUUID == userUUID
}

func (r *Room) ClientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

func (r *Room) Register(client *Client) {
	r.register <- client
}

func (r *Room) Broadcast(data []byte) {
	r.broadcast <- data
}

func (r *Room) GetSendChanForTest(userUUID uuid.UUID) <-chan []byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if c, ok := r.clients[userUUID]; ok {
		return c.send
	}
	return nil
}

func (r *Room) Stop() {
	select {
	case <-r.stop:
	default:
		close(r.stop)
	}
}

func (r *Room) Run() {
	for {
		select {
		case client := <-r.register:
			r.mu.Lock()
			r.clients[client.userUUID] = client
			client.SetRoom(r)
			r.mu.Unlock()

			r.sendRoomState(client)
			r.broadcastPlayerJoined(client)

		case client := <-r.unregister:
			r.mu.Lock()
			if _, ok := r.clients[client.userUUID]; ok {
				delete(r.clients, client.userUUID)
				close(client.send)
			}
			r.mu.Unlock()

			r.broadcastPlayerLeft(client)

			r.mu.RLock()
			empty := len(r.clients) == 0
			r.mu.RUnlock()
			if empty {
				r.mu.Lock()
				r.state = RoomStateClosed
				r.mu.Unlock()
				return
			}

		case message := <-r.broadcast:
			r.mu.RLock()
			for _, client := range r.clients {
				select {
				case client.send <- message:
				default:
					log.Printf("dropping message for slow client %s", client.userUUID)
				}
			}
			r.mu.RUnlock()

		case <-r.stop:
			r.mu.Lock()
			r.state = RoomStateClosed
			for _, client := range r.clients {
				close(client.send)
			}
			r.clients = make(map[uuid.UUID]*Client)
			r.mu.Unlock()
			return
		}
	}
}

func (r *Room) StartMatch(userUUID uuid.UUID) error {
	if !r.IsMaster(userUUID) {
		return ErrNotMaster
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.state != RoomStateLobby {
		return ErrAlreadyPlaying
	}
	r.state = RoomStatePlaying

	msg := NewServerMessage(MsgTypeMatchStarted, struct{}{})
	data, _ := json.Marshal(msg)
	go func() { r.broadcast <- data }()
	return nil
}

func (r *Room) handleClientMessage(client *Client, rawMsg []byte) {
	var incoming Message
	if err := json.Unmarshal(rawMsg, &incoming); err != nil {
		errMsg := NewErrorMessage("invalid_message", "malformed JSON")
		client.SendMessage(errMsg)
		return
	}

	switch incoming.Type {
	case MsgTypeStartMatch:
		if err := r.StartMatch(client.userUUID); err != nil {
			client.SendMessage(NewErrorMessage("forbidden", err.Error()))
		}

	case MsgTypeChat:
		var chatPayload ChatPayload
		if err := json.Unmarshal(incoming.Payload, &chatPayload); err != nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid chat payload"))
			return
		}
		outMsg := NewClientMessage(MsgTypeChatMessage, client.userUUID, chatPayload)
		data, _ := json.Marshal(outMsg)
		r.broadcast <- data

	default:
		client.SendMessage(NewErrorMessage("unknown_type", "unrecognized message type"))
	}
}

func (r *Room) sendRoomState(client *Client) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]PlayerInfo, 0, len(r.clients))
	for _, c := range r.clients {
		players = append(players, PlayerInfo{
			UUID:     c.userUUID,
			Nickname: c.nickname,
			IsMaster: r.masterUUID == c.userUUID,
			IsOnline: true,
		})
	}

	msg := NewServerMessage(MsgTypeRoomState, RoomStatePayload{
		MatchUUID: r.matchUUID,
		State:     string(r.state),
		Players:   players,
	})
	client.SendMessage(msg)
}

func (r *Room) broadcastPlayerJoined(client *Client) {
	msg := NewServerMessage(MsgTypePlayerJoined, PlayerPayload{
		UUID:     client.userUUID,
		Nickname: client.nickname,
	})
	data, _ := json.Marshal(msg)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		if c.userUUID != client.userUUID {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

func (r *Room) broadcastPlayerLeft(client *Client) {
	msg := NewServerMessage(MsgTypePlayerLeft, PlayerPayload{
		UUID:     client.userUUID,
		Nickname: client.nickname,
	})
	data, _ := json.Marshal(msg)

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		select {
		case c.send <- data:
		default:
		}
	}
}
```

Also add `GetSendChan()` to client.go for testing:

```go
// Add to client.go
func (c *Client) GetSendChan() <-chan []byte {
	return c.send
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/game/ -run TestRoom -v -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/room.go internal/app/game/room_test.go internal/app/game/client.go
git commit -m "feat(game): Room with state machine and broadcast

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Hub (`internal/app/game/hub.go`)

**Files:**
- Create: `internal/app/game/hub.go`
- Test: `internal/app/game/hub_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/app/game/hub_test.go
package game_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

func TestNewHub(t *testing.T) {
	hub := game.NewHub()
	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}
}

func TestHub_GetOrCreateRoom(t *testing.T) {
	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	matchID := uuid.New()
	masterID := uuid.New()

	room1 := hub.GetOrCreateRoom(matchID, masterID)
	room2 := hub.GetOrCreateRoom(matchID, masterID)

	if room1 != room2 {
		t.Error("expected same room instance for same matchUUID")
	}
	if room1.GetMatchUUID() != matchID {
		t.Errorf("got matchUUID %v, want %v", room1.GetMatchUUID(), matchID)
	}
}

func TestHub_GetOrCreateRoom_DifferentMatches(t *testing.T) {
	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	room1 := hub.GetOrCreateRoom(uuid.New(), uuid.New())
	room2 := hub.GetOrCreateRoom(uuid.New(), uuid.New())

	if room1 == room2 {
		t.Error("expected different rooms for different matches")
	}
}

func TestHub_RoomCount(t *testing.T) {
	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	hub.GetOrCreateRoom(uuid.New(), uuid.New())
	hub.GetOrCreateRoom(uuid.New(), uuid.New())

	time.Sleep(50 * time.Millisecond)

	if hub.RoomCount() != 2 {
		t.Errorf("got %d rooms, want 2", hub.RoomCount())
	}
}

func TestHub_RemoveRoom(t *testing.T) {
	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	matchID := uuid.New()
	hub.GetOrCreateRoom(matchID, uuid.New())
	hub.RemoveRoom(matchID)

	time.Sleep(50 * time.Millisecond)

	if hub.RoomCount() != 0 {
		t.Errorf("got %d rooms after remove, want 0", hub.RoomCount())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run TestHub -v -timeout 10s`
Expected: FAIL — `NewHub` not defined

- [ ] **Step 3: Write implementation**

```go
// internal/app/game/hub.go
package game

import (
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	rooms map[uuid.UUID]*Room
	mu    sync.RWMutex
	stop  chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[uuid.UUID]*Room),
		stop:  make(chan struct{}),
	}
}

func (h *Hub) Run() {
	<-h.stop
}

func (h *Hub) Stop() {
	select {
	case <-h.stop:
	default:
		close(h.stop)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for _, room := range h.rooms {
		room.Stop()
	}
}

func (h *Hub) GetOrCreateRoom(matchUUID, masterUUID uuid.UUID) *Room {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[matchUUID]; ok {
		return room
	}

	room := NewRoom(matchUUID, masterUUID)
	h.rooms[matchUUID] = room
	go room.Run()
	return room
}

func (h *Hub) GetRoom(matchUUID uuid.UUID) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, ok := h.rooms[matchUUID]
	return room, ok
}

func (h *Hub) RemoveRoom(matchUUID uuid.UUID) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if room, ok := h.rooms[matchUUID]; ok {
		room.Stop()
		delete(h.rooms, matchUUID)
	}
}

func (h *Hub) RoomCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/app/game/ -run TestHub -v -timeout 10s`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/hub.go internal/app/game/hub_test.go
git commit -m "feat(game): Hub managing rooms

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Handler + Server (`internal/app/game/handler.go` + `server.go`)

**Files:**
- Create: `internal/app/game/handler.go`
- Create: `internal/app/game/server.go`
- Test: `internal/app/game/handler_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/app/game/handler_test.go
package game_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type mockMatchRepo struct {
	getMatchMasterFn func(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
}

func (m *mockMatchRepo) GetMatchMaster(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error) {
	return m.getMatchMasterFn(ctx, matchUUID)
}

type mockEnrollmentChecker struct {
	isUserEnrolledFn func(ctx context.Context, userUUID, matchUUID uuid.UUID) (bool, error)
}

func (m *mockEnrollmentChecker) IsUserEnrolledInMatch(ctx context.Context, userUUID, matchUUID uuid.UUID) (bool, error) {
	return m.isUserEnrolledFn(ctx, userUUID, matchUUID)
}

func TestHandler_MissingToken_Returns401(t *testing.T) {
	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	srv := game.NewGameServer(hub, &mockMatchRepo{}, &mockEnrollmentChecker{})
	server := httptest.NewServer(srv.Router())
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?match_uuid=" + uuid.New().String()
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		t.Fatal("expected connection to fail")
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("got status %d, want 401", resp.StatusCode)
	}
}

func TestHandler_MissingMatchUUID_Returns400(t *testing.T) {
	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	srv := game.NewGameServer(hub, &mockMatchRepo{}, &mockEnrollmentChecker{})
	server := httptest.NewServer(srv.Router())
	defer server.Close()

	// Generate a valid token for the request
	userID := uuid.New()
	token := generateTestToken(t, userID)

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	header := http.Header{"Authorization": []string{"Bearer " + token}}
	_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected connection to fail")
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want 400", resp.StatusCode)
	}
}

func TestHandler_ValidConnection(t *testing.T) {
	masterID := uuid.New()
	matchID := uuid.New()

	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	matchRepo := &mockMatchRepo{
		getMatchMasterFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return masterID, nil
		},
	}
	enrollChecker := &mockEnrollmentChecker{
		isUserEnrolledFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	srv := game.NewGameServer(hub, matchRepo, enrollChecker)
	server := httptest.NewServer(srv.Router())
	defer server.Close()

	token := generateTestToken(t, masterID)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?match_uuid=" + matchID.String()
	header := http.Header{"Authorization": []string{"Bearer " + token}}

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial error: %v (status: %d)", err, resp.StatusCode)
	}
	defer conn.Close()

	// Should receive room_state message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if !strings.Contains(string(data), "room_state") {
		t.Errorf("expected room_state message, got: %s", data)
	}
}

func TestHandler_NotEnrolled_Returns403(t *testing.T) {
	masterID := uuid.New()
	playerID := uuid.New() // different from master
	matchID := uuid.New()

	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	matchRepo := &mockMatchRepo{
		getMatchMasterFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return masterID, nil
		},
	}
	enrollChecker := &mockEnrollmentChecker{
		isUserEnrolledFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil // not enrolled
		},
	}

	srv := game.NewGameServer(hub, matchRepo, enrollChecker)
	server := httptest.NewServer(srv.Router())
	defer server.Close()

	token := generateTestToken(t, playerID) // not master, not enrolled
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws?match_uuid=" + matchID.String()
	header := http.Header{"Authorization": []string{"Bearer " + token}}

	_, resp, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err == nil {
		t.Fatal("expected connection to fail")
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("got status %d, want 403", resp.StatusCode)
	}
}

// generateTestToken creates a valid JWT for testing
func generateTestToken(t *testing.T, userID uuid.UUID) string {
	t.Helper()
	// Uses the same pkg/auth that the handler will use
	token, err := pkgAuth.GenerateToken(userID)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	return token
}
```

**Note:** The test file needs this import for `generateTestToken`:
```go
import pkgAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/app/game/ -run TestHandler -v -timeout 10s`
Expected: FAIL — `NewGameServer` not defined

- [ ] **Step 3: Write handler implementation**

```go
// internal/app/game/handler.go
package game

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	pkgAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
)

type MatchRepository interface {
	GetMatchMaster(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
}

type EnrollmentChecker interface {
	IsUserEnrolledInMatch(ctx context.Context, userUUID, matchUUID uuid.UUID) (bool, error)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// TODO: IN PRODUCTION, IMPLEMENT ORIGIN CHECKING
		return true
	},
}

func (s *GameServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 1. Validate JWT
	tokenStr := extractBearerToken(r)
	if tokenStr == "" {
		http.Error(w, "missing or invalid authorization", http.StatusUnauthorized)
		return
	}

	claims, err := pkgAuth.ValidateToken(tokenStr)
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	userUUID := claims.UserID

	// 2. Validate match_uuid parameter
	matchUUIDStr := r.URL.Query().Get("match_uuid")
	if matchUUIDStr == "" {
		http.Error(w, "missing match_uuid parameter", http.StatusBadRequest)
		return
	}
	matchUUID, err := uuid.Parse(matchUUIDStr)
	if err != nil {
		http.Error(w, "invalid match_uuid format", http.StatusBadRequest)
		return
	}

	// 3. Validate match exists and get master UUID
	masterUUID, err := s.matchRepo.GetMatchMaster(r.Context(), matchUUID)
	if err != nil {
		http.Error(w, "match not found", http.StatusNotFound)
		return
	}

	// 4. Validate user is master or enrolled
	isMaster := masterUUID == userUUID
	if !isMaster {
		enrolled, err := s.enrollChecker.IsUserEnrolledInMatch(r.Context(), userUUID, matchUUID)
		if err != nil || !enrolled {
			http.Error(w, "not authorized for this match", http.StatusForbidden)
			return
		}
	}

	// 5. Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}

	// 6. Get or create room and register client
	room := s.hub.GetOrCreateRoom(matchUUID, masterUUID)

	nickname := userUUID.String()[:8] // TODO: fetch nickname from DB
	client := NewClient(userUUID, conn, nickname)
	room.Register(client)

	go client.WritePump()
	go client.ReadPump()
}

func extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return ""
	}
	return authHeader[len(bearerPrefix):]
}
```

- [ ] **Step 4: Write server implementation**

```go
// internal/app/game/server.go
package game

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type GameServer struct {
	hub           *Hub
	matchRepo     MatchRepository
	enrollChecker EnrollmentChecker
	router        *chi.Mux
}

func NewGameServer(hub *Hub, matchRepo MatchRepository, enrollChecker EnrollmentChecker) *GameServer {
	s := &GameServer{
		hub:           hub,
		matchRepo:     matchRepo,
		enrollChecker: enrollChecker,
	}
	s.router = s.setupRouter()
	return s
}

func (s *GameServer) Router() http.Handler {
	return s.router
}

func (s *GameServer) setupRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/ws", s.handleWebSocket)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/app/game/ -run TestHandler -v -timeout 10s`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/app/game/handler.go internal/app/game/server.go internal/app/game/handler_test.go
git commit -m "feat(game): handler with JWT auth + enrollment validation + server setup

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: Repository Methods for Game Server

**Files:**
- Create: `internal/gateway/pg/match/read_match_master.go`
- Create: `internal/gateway/pg/enrollment/is_user_enrolled.go`

- [ ] **Step 1: Write GetMatchMaster**

```go
// internal/gateway/pg/match/read_match_master.go
package match

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMatchMaster(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error) {
	const query = `SELECT master_uuid FROM matches WHERE uuid = $1`

	var masterUUID uuid.UUID
	err := r.q.QueryRow(ctx, query, matchUUID).Scan(&masterUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrMatchNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get match master: %w", err)
	}
	return masterUUID, nil
}
```

- [ ] **Step 2: Write IsUserEnrolledInMatch**

```go
// internal/gateway/pg/enrollment/is_user_enrolled.go
package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) IsUserEnrolledInMatch(ctx context.Context, userUUID, matchUUID uuid.UUID) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM enrollments e
			JOIN character_sheets cs ON e.character_sheet_uuid = cs.uuid
			WHERE e.match_uuid = $1
			AND (cs.player_uuid = $2 OR cs.master_uuid = $2)
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, matchUUID, userUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user enrollment: %w", err)
	}
	return exists, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./internal/gateway/pg/match/ && go build ./internal/gateway/pg/enrollment/`
Expected: compiles without errors

- [ ] **Step 4: Commit**

```bash
git add internal/gateway/pg/match/read_match_master.go internal/gateway/pg/enrollment/is_user_enrolled.go
git commit -m "feat(gateway): add GetMatchMaster and IsUserEnrolledInMatch for game server

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 7: Rewrite `cmd/game/main.go`

**Files:**
- Modify: `cmd/game/main.go`

- [ ] **Step 1: Rewrite main.go**

```go
// cmd/game/main.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	enrollmentPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/enrollment"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
	"github.com/ardanlabs/conf/v3"
	"github.com/joho/godotenv"
)

type config struct {
	ServerAddr         string        `conf:"env:GAME_SERVER_ADDR,default:localhost:8080"`
	ServerReadTimeout  time.Duration `conf:"default:30s"`
	ServerWriteTimeout time.Duration `conf:"default:30s"`
}

func main() {
	loadEnvFile()

	cfg, err := loadConfig()
	if err != nil {
		panic(fmt.Errorf("error loading config: %w", err))
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	pgPool, err := pgfs.New(ctx, "")
	if err != nil {
		panic(fmt.Errorf("error creating pg pool: %w", err))
	}
	defer pgPool.Close()

	matchRepo := matchPg.NewRepository(pgPool)
	enrollmentRepo := enrollmentPg.NewRepository(pgPool)

	hub := game.NewHub()
	go hub.Run()

	srv := game.NewGameServer(hub, matchRepo, enrollmentRepo)

	server := http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      srv.Router(),
		ReadTimeout:  cfg.ServerReadTimeout,
		WriteTimeout: cfg.ServerWriteTimeout,
	}

	fmt.Printf("Game server starting on %s\n", cfg.ServerAddr)
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
		os.Exit(1)
	}
}

func loadEnvFile() {
	_, err := os.Stat(".env")
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		return
	}
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
}

func loadConfig() (config, error) {
	var cfg config
	if _, err := conf.Parse("", &cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./cmd/game/`
Expected: compiles successfully

- [ ] **Step 3: Commit**

```bash
git add cmd/game/main.go
git commit -m "feat(game): rewrite cmd/game with proper DI and Hub/Room architecture

Replaces the prototype chat server with a production-ready
game server using Hub/Room/Client pattern.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 8: Full Integration Test

**Files:**
- Create: `internal/app/game/integration_test.go`

- [ ] **Step 1: Write end-to-end WebSocket test**

```go
// internal/app/game/integration_test.go
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

func TestIntegration_FullMatchFlow(t *testing.T) {
	masterID := uuid.New()
	playerID := uuid.New()
	matchID := uuid.New()

	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	matchRepo := &mockMatchRepo{
		getMatchMasterFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return masterID, nil
		},
	}
	enrollChecker := &mockEnrollmentChecker{
		isUserEnrolledFn: func(_ context.Context, userUUID, _ uuid.UUID) (bool, error) {
			return userUUID == playerID, nil
		},
	}

	srv := game.NewGameServer(hub, matchRepo, enrollChecker)
	server := httptest.NewServer(srv.Router())
	defer server.Close()

	// 1. Master connects
	masterToken, _ := pkgAuth.GenerateToken(masterID)
	masterConn := dialWS(t, server.URL, matchID, masterToken)
	defer masterConn.Close()

	// Master receives room_state
	masterMsg := readWSMessage(t, masterConn)
	if masterMsg.Type != game.MsgTypeRoomState {
		t.Fatalf("master got type %q, want room_state", masterMsg.Type)
	}

	// 2. Player connects
	playerToken, _ := pkgAuth.GenerateToken(playerID)
	playerConn := dialWS(t, server.URL, matchID, playerToken)
	defer playerConn.Close()

	// Player receives room_state
	playerMsg := readWSMessage(t, playerConn)
	if playerMsg.Type != game.MsgTypeRoomState {
		t.Fatalf("player got type %q, want room_state", playerMsg.Type)
	}

	// Master receives player_joined
	masterMsg = readWSMessage(t, masterConn)
	if masterMsg.Type != game.MsgTypePlayerJoined {
		t.Fatalf("master got type %q, want player_joined", masterMsg.Type)
	}

	// 3. Master starts match
	startMsg := game.Message{Type: game.MsgTypeStartMatch, Payload: json.RawMessage(`{}`)}
	data, _ := json.Marshal(startMsg)
	masterConn.WriteMessage(websocket.TextMessage, data)

	// Both receive match_started
	masterMsg = readWSMessage(t, masterConn)
	if masterMsg.Type != game.MsgTypeMatchStarted {
		t.Fatalf("master got type %q, want match_started", masterMsg.Type)
	}
	playerMsg = readWSMessage(t, playerConn)
	if playerMsg.Type != game.MsgTypeMatchStarted {
		t.Fatalf("player got type %q, want match_started", playerMsg.Type)
	}

	// 4. Player sends chat
	chatMsg := game.Message{
		Type:    game.MsgTypeChat,
		Payload: json.RawMessage(`{"message":"Hello!"}`),
	}
	data, _ = json.Marshal(chatMsg)
	playerConn.WriteMessage(websocket.TextMessage, data)

	// Both receive chat_message
	masterMsg = readWSMessage(t, masterConn)
	if masterMsg.Type != game.MsgTypeChatMessage {
		t.Fatalf("master got type %q, want chat_message", masterMsg.Type)
	}
	playerMsg = readWSMessage(t, playerConn)
	if playerMsg.Type != game.MsgTypeChatMessage {
		t.Fatalf("player got type %q, want chat_message", playerMsg.Type)
	}
}

func dialWS(t *testing.T, serverURL string, matchID uuid.UUID, token string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws?match_uuid=" + matchID.String()
	header := http.Header{"Authorization": []string{"Bearer " + token}}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	return conn
}

func readWSMessage(t *testing.T, conn *websocket.Conn) game.Message {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	var msg game.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal error: %v (data: %s)", err, data)
	}
	return msg
}
```

- [ ] **Step 2: Run full integration test**

Run: `go test ./internal/app/game/ -run TestIntegration -v -timeout 30s`
Expected: PASS

- [ ] **Step 3: Run all game package tests**

Run: `go test ./internal/app/game/ -v -timeout 30s`
Expected: ALL PASS

- [ ] **Step 4: Run full project tests**

Run: `go test ./... -timeout 60s`
Expected: Only pre-existing `turn/engine_test.go` failure

- [ ] **Step 5: Commit**

```bash
git add internal/app/game/integration_test.go
git commit -m "test(game): full WebSocket integration test (lobby → start → chat)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 9: Final Verification and Merge

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1 -timeout 120s`
Expected: All pass except pre-existing `turn/engine_test.go`

- [ ] **Step 2: Build both binaries**

Run: `go build ./cmd/api/ && go build ./cmd/game/`
Expected: Both compile successfully

- [ ] **Step 3: Merge to main**

```bash
git checkout main
git merge feat/websocket-game-server --no-ff -m "Merge feat/websocket-game-server: Hub/Room/Client MVP"
git branch -d feat/websocket-game-server
```
