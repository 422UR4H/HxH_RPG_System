package game_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/game"
	"github.com/google/uuid"
)

func TestNewServerMessage(t *testing.T) {
	payload := game.ChatPayload{Message: "hello"}
	msg := game.NewServerMessage(game.MsgTypeChatMessage, payload)

	if msg.Type != game.MsgTypeChatMessage {
		t.Errorf("expected type %s, got %s", game.MsgTypeChatMessage, msg.Type)
	}
	if msg.SenderID != uuid.Nil {
		t.Errorf("expected nil sender for server message, got %s", msg.SenderID)
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}

	var chat game.ChatPayload
	if err := json.Unmarshal(msg.Payload, &chat); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if chat.Message != "hello" {
		t.Errorf("expected message 'hello', got '%s'", chat.Message)
	}
}

func TestNewClientMessage(t *testing.T) {
	senderID := uuid.New()
	payload := game.ChatPayload{Message: "world"}
	msg := game.NewClientMessage(game.MsgTypeChatMessage, senderID, payload)

	if msg.SenderID != senderID {
		t.Errorf("expected sender %s, got %s", senderID, msg.SenderID)
	}
	if msg.Type != game.MsgTypeChatMessage {
		t.Errorf("expected type %s, got %s", game.MsgTypeChatMessage, msg.Type)
	}
}

func TestNewErrorMessage(t *testing.T) {
	msg := game.NewErrorMessage("forbidden", "only the master can do this")

	if msg.Type != game.MsgTypeError {
		t.Errorf("expected type %s, got %s", game.MsgTypeError, msg.Type)
	}

	var errPayload game.ErrorPayload
	if err := json.Unmarshal(msg.Payload, &errPayload); err != nil {
		t.Fatalf("failed to unmarshal error payload: %v", err)
	}
	if errPayload.Code != "forbidden" {
		t.Errorf("expected code 'forbidden', got '%s'", errPayload.Code)
	}
	if errPayload.Message != "only the master can do this" {
		t.Errorf("expected message 'only the master can do this', got '%s'", errPayload.Message)
	}
}

func TestMessageSerialization(t *testing.T) {
	original := game.NewServerMessage(game.MsgTypeRoomState, game.RoomStatePayload{
		MatchUUID: uuid.New(),
		State:     "lobby",
		Players: []game.PlayerInfo{
			{UUID: uuid.New(), Nickname: "player1", IsMaster: true, IsOnline: true},
		},
	})

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded game.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Type != original.Type {
		t.Errorf("type mismatch: got %s, want %s", decoded.Type, original.Type)
	}
}

func TestHub(t *testing.T) {
	hub := game.NewHub()
	go hub.Run()
	defer hub.Stop()

	matchUUID := uuid.New()
	masterUUID := uuid.New()

	if hub.RoomCount() != 0 {
		t.Errorf("expected 0 rooms, got %d", hub.RoomCount())
	}

	room := hub.GetOrCreateRoom(matchUUID, masterUUID, &mockStartMatchUC{}, &mockKickPlayerUC{})
	if room == nil {
		t.Fatal("expected room to be created")
	}
	if hub.RoomCount() != 1 {
		t.Errorf("expected 1 room, got %d", hub.RoomCount())
	}

	room2 := hub.GetOrCreateRoom(matchUUID, masterUUID, &mockStartMatchUC{}, &mockKickPlayerUC{})
	if room2 != room {
		t.Error("expected same room for same matchUUID")
	}
	if hub.RoomCount() != 1 {
		t.Errorf("still expected 1 room, got %d", hub.RoomCount())
	}

	otherMatchUUID := uuid.New()
	otherRoom := hub.GetOrCreateRoom(otherMatchUUID, masterUUID, &mockStartMatchUC{}, &mockKickPlayerUC{})
	if otherRoom == room {
		t.Error("expected different room for different matchUUID")
	}
	if hub.RoomCount() != 2 {
		t.Errorf("expected 2 rooms, got %d", hub.RoomCount())
	}

	hub.RemoveRoom(matchUUID)
	if hub.RoomCount() != 1 {
		t.Errorf("expected 1 room after removal, got %d", hub.RoomCount())
	}

	_, found := hub.GetRoom(matchUUID)
	if found {
		t.Error("expected room to not be found after removal")
	}
}

func TestRoom(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()

	room := game.NewRoom(matchUUID, masterUUID, &mockStartMatchUC{}, &mockKickPlayerUC{})
	go room.Run()
	defer room.Stop()

	if room.GetState() != game.RoomStateLobby {
		t.Errorf("expected lobby state, got %s", room.GetState())
	}
	if room.GetMatchUUID() != matchUUID {
		t.Errorf("expected matchUUID %s, got %s", matchUUID, room.GetMatchUUID())
	}
	if !room.IsMaster(masterUUID) {
		t.Error("expected masterUUID to be master")
	}
	playerUUID := uuid.New()
	if room.IsMaster(playerUUID) {
		t.Error("expected playerUUID to NOT be master")
	}
}

func TestRoomStartMatch(t *testing.T) {
	matchUUID := uuid.New()
	masterUUID := uuid.New()
	playerUUID := uuid.New()

	room := game.NewRoom(matchUUID, masterUUID, &mockStartMatchUC{}, &mockKickPlayerUC{})
	go room.Run()
	defer room.Stop()

	time.Sleep(10 * time.Millisecond)

	if err := room.StartMatch(playerUUID); err != game.ErrNotMaster {
		t.Errorf("expected ErrNotMaster, got %v", err)
	}

	if err := room.StartMatch(masterUUID); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if room.GetState() != game.RoomStatePlaying {
		t.Errorf("expected playing state, got %s", room.GetState())
	}

	if err := room.StartMatch(masterUUID); err != game.ErrAlreadyPlaying {
		t.Errorf("expected ErrAlreadyPlaying, got %v", err)
	}
}
