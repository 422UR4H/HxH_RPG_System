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
	MatchUUID uuid.UUID    `json:"match_uuid"`
	State     string       `json:"state"`
	Players   []PlayerInfo `json:"players"`
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
