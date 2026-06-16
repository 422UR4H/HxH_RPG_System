package game

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

type MessageType string

const (
	// Server → Client
	MsgTypeRoomState    MessageType = "room_state"
	MsgTypePlayerJoined MessageType = "player_joined"
	MsgTypePlayerLeft   MessageType = "player_left"
	MsgTypeMasterLeft   MessageType = "master_left"
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
	MsgTypePieceMoved   MessageType = "piece_moved"
	MsgTypePieceRemoved MessageType = "piece_removed"
	// Sent by master on WS connect to seed backend in-memory state from DB.
	MsgTypeMapStateSync MessageType = "map_state_sync"

	// Server → Client (lobby map sync)
	// Sent to every client that registers, so late-joiners get the current board.
	MsgTypeMapFullState MessageType = "map_full_state"

	// Server → Client (wall events)
	MsgTypeWallStateChanged MessageType = "wall_state_changed"

	// Server → Client (wall HP/structural events)
	MsgTypeWallHpChanged MessageType = "wall_hp_changed"
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

type PieceMovedPayload struct {
	PieceID     string      `json:"piece_id"`
	Slot        SlotPayload `json:"slot"`
	CharacterID string      `json:"character_id,omitempty"`
	Visible     *bool       `json:"visible,omitempty"`
}

type PieceRemovedPayload struct {
	PieceID string `json:"piece_id"`
}

// MapPiecesPayload is shared by map_state_sync (client→server) and
// map_full_state (server→client). Both carry the complete current board.
type MapPiecesPayload struct {
	Pieces []PieceMovedPayload `json:"pieces"`
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
	Interact  *InteractPayload     `json:"interact,omitempty"`
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

type InteractPayload struct {
	Kind string `json:"kind"` // "open" | "close" | "toggle" | "lockpick" | "examine"
}

type MovePayload struct {
	Category string            `json:"category"`
	From     [3]int            `json:"from,omitempty"` // source grid position [col, row, z]; zero = not provided
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
	Interact    *InteractPayload     `json:"interact,omitempty"`
}

type MasterActionEnqueuedPayload struct {
	TargetIDs   []uuid.UUID          `json:"target_ids"`
	Skills      []ActionSkillPayload `json:"skills,omitempty"`
	Move        *MovePayload         `json:"move,omitempty"`
	Attack      *AttackPayload       `json:"attack,omitempty"`
	ActionSpeed *RollCheckPayload    `json:"action_speed,omitempty"`
	Interact    *InteractPayload     `json:"interact,omitempty"`
}

// WallStateChangedPayload is broadcast to all clients when a wall's open/locked state changes.
type WallStateChangedPayload struct {
	WallID string `json:"wall_id"`
	Open   bool   `json:"open"`
	Locked bool   `json:"locked"`
}

// WallHpChangedPayload is broadcast to all clients when a wall's HP or destroyed state changes.
type WallHpChangedPayload struct {
	WallID    string `json:"wall_id"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"max_hp"`
	Destroyed bool   `json:"destroyed"`
}

// MapStateSyncPayload extends MapPiecesPayload to include walls and grid.
// Sent by the master on WS connect to seed the room's in-memory state from the DB.
// Walls are full WallSegment objects so the room can perform movement blocking without
// additional DB queries.
type MapStateSyncPayload struct {
	Pieces []PieceMovedPayload `json:"pieces"`
	Walls  []mapentity.WallSegment  `json:"walls,omitempty"`
	Grid   *GridSyncEntry           `json:"grid,omitempty"`
}

// GridSyncEntry carries the cell size used to convert grid slot coords to world coords.
type GridSyncEntry struct {
	CellSize float64 `json:"cell_size"`
}
