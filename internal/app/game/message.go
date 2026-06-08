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

	// Server → Client (game events)
	MsgTypeTurnOpened       MessageType = "turn_opened"
	MsgTypeRoundClosed      MessageType = "round_closed"
	MsgTypeResolutionUpdate MessageType = "resolution_updated"
	MsgTypeActionEnqueued   MessageType = "action_enqueued"

	// Client → Server (scene management)
	MsgTypeChangeScene MessageType = "change_scene"

	// Server → Client (scene events)
	MsgTypeSceneChanged MessageType = "scene_changed"

	// Client → Server (master actions)
	MsgTypeEnqueueMasterAction MessageType = "enqueue_master_action"

	// Server → Client
	MsgTypeMasterActionEnqueued MessageType = "master_action_enqueued"

	// Server → Client (lobby lifecycle)
	MsgTypeLobbyClosed  MessageType = "lobby_closed"   // master cancelled the lobby
	// MsgTypeLobbyNotOpen is sent by handler.go when a participant tries to connect before the master opens the lobby
	MsgTypeLobbyNotOpen MessageType = "lobby_not_open"

	// Client → Server (lobby lifecycle)
	MsgTypeCancelLobby MessageType = "cancel_lobby" // master requests lobby cancellation

	// Client → Server (lobby map sync)
	// lobby_ prefix distinguishes from future in-game events (Phase 7+).
	MsgTypeLobbyPieceMoved  MessageType = "lobby_piece_moved"
	MsgTypeLobbyPieceRemoved MessageType = "lobby_piece_removed"
	// Sent by master on WS connect to seed backend in-memory state from DB.
	MsgTypeLobbyStateSync   MessageType = "lobby_state_sync"

	// Server → Client (lobby map sync)
	// Sent to every client that registers, so late-joiners get the current board.
	MsgTypeLobbyFullState   MessageType = "lobby_full_state"
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

// SlotPayload represents a grid slot coordinate (square or hex).
type SlotPayload struct {
	Kind string `json:"kind"`          // "square" | "hex"
	Col  *int   `json:"col,omitempty"` // square only
	Row  *int   `json:"row,omitempty"` // square only
	Q    *int   `json:"q,omitempty"`   // hex only
	R    *int   `json:"r,omitempty"`   // hex only
}

type LobbyPieceMovedPayload struct {
	PieceID     string      `json:"piece_id"`
	Slot        SlotPayload `json:"slot"`
	CharacterID string      `json:"character_id,omitempty"`
	Visible     *bool       `json:"visible,omitempty"`
}

type LobbyPieceRemovedPayload struct {
	PieceID string `json:"piece_id"`
}

// LobbyPiecesPayload is shared by lobby_state_sync (client→server) and
// lobby_full_state (server→client). Both carry the complete current board.
type LobbyPiecesPayload struct {
	Pieces []LobbyPieceMovedPayload `json:"pieces"`
}

type PullActionPayload struct {
	ActionID uuid.UUID `json:"action_id"`
}

// ActionPayload is the unified shape for both enqueue_action and attach_reaction messages.
// The presence of ReactToID determines routing: non-zero means it is a reaction.
// The presence of sub-fields (Dodge, Attack, etc.) describes the action composition.
type ActionPayload struct {
	ReactToID uuid.UUID            `json:"react_to_id,omitempty"`
	TargetID  []uuid.UUID          `json:"target_id,omitempty"`
	Skills    []ActionSkillPayload `json:"skills,omitempty"`
	Speed     *ActionSpeedPayload  `json:"speed,omitempty"`
	Feint     *RollCheckPayload    `json:"feint,omitempty"`
	Move      *MovePayload         `json:"move,omitempty"`
	Attack    *AttackPayload       `json:"attack,omitempty"`
	Defense   *DefensePayload      `json:"defense,omitempty"`
	Dodge     *DodgePayload        `json:"dodge,omitempty"`
}

type RollCheckPayload struct {
	SkillName string `json:"skill_name"`
}

type DodgePayload struct {
	Category  string            `json:"category"`
	RollCheck *RollCheckPayload `json:"roll_check,omitempty"`
}

type AttackPayload struct {
	Weapon *string           `json:"weapon,omitempty"`
	Hit    RollCheckPayload  `json:"hit"`
	Damage RollCheckPayload  `json:"damage"`
	Charge *RollCheckPayload `json:"charge,omitempty"`
}

type DefensePayload struct {
	Weapon    *string          `json:"weapon,omitempty"`
	RollCheck RollCheckPayload `json:"roll_check"`
}

type MovePayload struct {
	Category string            `json:"category"`
	Position [3]int            `json:"position"`
	Speed    *RollCheckPayload `json:"speed,omitempty"`
	Charge   *RollCheckPayload `json:"charge,omitempty"`
}

type ActionSpeedPayload struct {
	Bar       int               `json:"bar"`
	RollCheck *RollCheckPayload `json:"roll_check,omitempty"`
}

type ActionSkillPayload struct {
	SkillName  string `json:"skill_name"`
	Difficulty *int   `json:"difficulty,omitempty"`
}

type TurnOpenedPayload struct {
	TurnID     uuid.UUID `json:"turn_id"`
	ActorID    uuid.UUID `json:"actor_id"`
	ActionType string    `json:"action_type"`
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

type ChangeScenePayload struct {
	Category                string `json:"category"`
	BriefInitialDescription string `json:"brief_initial_description"`
}

type SceneChangedPayload struct {
	SceneID                 uuid.UUID `json:"scene_id"`
	Category                string    `json:"category"`
	BriefInitialDescription string    `json:"brief_initial_description"`
}

type MasterActionPayload struct {
	TargetIDs   []uuid.UUID          `json:"target_ids"`
	Skills      []ActionSkillPayload `json:"skills,omitempty"`
	Move        *MovePayload         `json:"move,omitempty"`
	Attack      *AttackPayload       `json:"attack,omitempty"`
	ActionSpeed *RollCheckPayload    `json:"action_speed,omitempty"`
}

type MasterActionEnqueuedPayload struct {
	TargetIDs   []uuid.UUID          `json:"target_ids"`
	Skills      []ActionSkillPayload `json:"skills,omitempty"`
	Move        *MovePayload         `json:"move,omitempty"`
	Attack      *AttackPayload       `json:"attack,omitempty"`
	ActionSpeed *RollCheckPayload    `json:"action_speed,omitempty"`
}
