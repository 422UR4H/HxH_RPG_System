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
	MsgTypePlayerKicked MessageType = "player_kicked"
	MsgTypeMatchStarted MessageType = "match_started"
	MsgTypeChatMessage  MessageType = "chat_message"
	MsgTypeError        MessageType = "error"

	// Client → Server
	MsgTypeStartMatch MessageType = "start_match"
	MsgTypeKickPlayer MessageType = "kick_player"
	MsgTypeChat       MessageType = "chat"

	// Client → Server (game actions)
	MsgTypeEnqueueAction  MessageType = "enqueue_action"
	MsgTypeOpenNextAction MessageType = "open_next_action"
	MsgTypePullAction     MessageType = "pull_action"
	MsgTypeAttachReaction MessageType = "attach_reaction"
	MsgTypeCloseTurn      MessageType = "close_turn"
	MsgTypeCloseRound     MessageType = "close_round"

	// Server → Client (game events)
	MsgTypeTurnOpened       MessageType = "turn_opened"
	MsgTypeTurnClosed       MessageType = "turn_closed"
	MsgTypeRoundClosed      MessageType = "round_closed"
	MsgTypeResolutionUpdate MessageType = "resolution_updated"
	MsgTypeActionEnqueued   MessageType = "action_enqueued"
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

type KickPlayerPayload struct {
	PlayerUUID uuid.UUID `json:"player_uuid"`
}

type PlayerKickedPayload struct {
	UUID     uuid.UUID `json:"uuid"`
	Nickname string    `json:"nickname"`
	Reason   string    `json:"reason"`
}

type EnqueueActionPayload struct {
	ActionType string    `json:"action_type"`
	TargetID   uuid.UUID `json:"target_id,omitempty"`
}

type PullActionPayload struct {
	ActionID uuid.UUID `json:"action_id"`
}

type AttachReactionPayload struct {
	ReactToID  uuid.UUID `json:"react_to_id"`
	ActionType string    `json:"action_type"`
}

type TurnOpenedPayload struct {
	TurnID     uuid.UUID `json:"turn_id"`
	ActorID    uuid.UUID `json:"actor_id"`
	ActionType string    `json:"action_type"`
}

type TurnClosedPayload struct {
	TurnID uuid.UUID `json:"turn_id"`
}

type RoundClosedPayload struct {
	RoundMode string `json:"round_mode"`
}

type ResolutionUpdatedPayload struct {
	TurnID    uuid.UUID `json:"turn_id"`
	IsSettled bool      `json:"is_settled"`
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
