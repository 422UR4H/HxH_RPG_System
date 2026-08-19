package game

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

type MessageType string

const (
	// Server → Client
	MsgTypeRoomState    MessageType = "room_state"
	MsgTypePlayerJoined MessageType = "player_joined"
	MsgTypeMasterJoined MessageType = "master_joined"
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
	MsgTypeBarsUpdated      MessageType = "bars_updated"

	// Client → Server (scene management)
	MsgTypeChangeScene MessageType = "change_scene"

	// Server → Client (scene events)
	MsgTypeSceneChanged MessageType = "scene_changed"

	// Client → Server (master actions)
	MsgTypeEnqueueMasterAction MessageType = "enqueue_master_action"
	MsgTypeChangeRoundMode     MessageType = "change_round_mode"

	// Server → Client
	MsgTypeMasterActionEnqueued MessageType = "master_action_enqueued"
	MsgTypeRoundModeChanged     MessageType = "round_mode_changed"

	// Server → Client (lobby lifecycle)
	MsgTypeLobbyClosed MessageType = "lobby_closed" // master cancelled the lobby
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

	// Server → Client (fog of war events)
	MsgTypeVisibilityUpdated MessageType = "visibility_updated"
	MsgTypeWallRevealed      MessageType = "wall_revealed"
)

type Message struct {
	Type      MessageType     `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	SenderID  uuid.UUID       `json:"senderId"`
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
	MatchUUID uuid.UUID    `json:"matchUuid"`
	State     string       `json:"state"`
	Players   []PlayerInfo `json:"players"`
}

type PlayerInfo struct {
	UUID     uuid.UUID `json:"uuid"`
	Nickname string    `json:"nickname"`
	IsMaster bool      `json:"isMaster"`
	IsOnline bool      `json:"isOnline"`
}

type ChatPayload struct {
	Message string `json:"message"`
}

type KickPlayerPayload struct {
	PlayerUUID uuid.UUID `json:"playerUuid"`
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
	PieceID     string      `json:"pieceId"`
	Slot        SlotPayload `json:"slot"`
	CharacterID string      `json:"characterId,omitempty"`
	Visible     *bool       `json:"visible,omitempty"`
	// Z is the piece's virtual height in metres (0 = ground). The game server never
	// reads it: line of sight is computed in 2D, so elevation rides through as opaque
	// passthrough exactly like Slot does. It is on the wire because the client used to
	// recover it from GET /maps/:id, and that endpoint no longer hands a player any
	// pieces — without this field every piece would flatten to the ground for players.
	Z float64 `json:"z,omitempty"`
}

type PieceRemovedPayload struct {
	PieceID string `json:"pieceId"`
}

type PullActionPayload struct {
	ActionID uuid.UUID `json:"actionId"`
}

// ActionPayload is the unified shape for both enqueue_action and attach_reaction messages.
// The presence of ReactToID determines routing: non-zero means it is a reaction.
// The presence of sub-fields (Dodge, Attack, etc.) describes the action composition.
type ActionPayload struct {
	// ActorID is the acting character's sheet UUID — the same ID the board piece carries
	// as CharacterID. It is NOT the player UUID: one person drives several characters (the
	// master drives every NPC), so the actor has to be named explicitly. The server still
	// checks that the authenticated player owns that character.
	ActorID   uuid.UUID            `json:"actorId"`
	ReactToID uuid.UUID            `json:"reactToId,omitempty"`
	TargetID  []uuid.UUID          `json:"targetId,omitempty"`
	Skills    []ActionSkillPayload `json:"skills,omitempty"`
	Speed     *ActionSpeedPayload  `json:"speed,omitempty"`
	Feint     *RollCheckPayload    `json:"feint,omitempty"`
	Move      *MovePayload         `json:"move,omitempty"`
	Attack    *AttackPayload       `json:"attack,omitempty"`
	Defense   *DefensePayload      `json:"defense,omitempty"`
	Dodge     *DodgePayload        `json:"dodge,omitempty"`
	Interact  *InteractPayload     `json:"interact,omitempty"`
	// ReactionKind names what the target chose to do, and it is MANDATORY on a reaction. The
	// server never infers the cost from the shape of what arrived — the three escapes are
	// shape-identical and priced differently. Empty on a plain action.
	ReactionKind string        `json:"reactionKind,omitempty"`
	Repel        *RepelPayload `json:"repel,omitempty"`
}

type RollCheckPayload struct {
	SkillName string `json:"skillName"`
}

// RepelPayload is the hardest reaction in the catalogue, shaped like the defense: a weapon and
// a test.
type RepelPayload struct {
	Weapon    *string          `json:"weapon,omitempty"`
	RollCheck RollCheckPayload `json:"rollCheck"`
}

type DodgePayload struct {
	Category  string            `json:"category"`
	RollCheck *RollCheckPayload `json:"rollCheck,omitempty"`
}

type AttackPayload struct {
	Weapon *string           `json:"weapon,omitempty"`
	Hit    RollCheckPayload  `json:"hit"`
	Damage RollCheckPayload  `json:"damage"`
	Charge *RollCheckPayload `json:"charge,omitempty"`
}

type DefensePayload struct {
	Weapon    *string          `json:"weapon,omitempty"`
	RollCheck RollCheckPayload `json:"rollCheck"`
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
	RollCheck *RollCheckPayload `json:"rollCheck,omitempty"`
}

type ActionSkillPayload struct {
	SkillName  string `json:"skillName"`
	Difficulty *int   `json:"difficulty,omitempty"`
}

type TurnOpenedPayload struct {
	TurnID     uuid.UUID `json:"turnId"`
	ActorID    uuid.UUID `json:"actorId"`
	ActionType string    `json:"actionType"`
}

type RoundClosedPayload struct {
	RoundMode string `json:"roundMode"`
}

// BarsUpdatedPayload is the two clocks as the whole table sees them.
//
// It is BROADCAST, not projected per recipient, and that is deliberate: combat-engine.md says
// "A fila é secreta; a barra e a ordem são públicas". A player who cannot see the general bar
// only finds out it was their turn after it passed.
//
// Order therefore carries who acts next and on which bar — and NOTHING that identifies the
// action itself. No action ID, no weapon, no target, no skill. Those belong to the master
// until the turn opens.
type BarsUpdatedPayload struct {
	// Seq orders the snapshots, and exists because they can arrive out of order.
	//
	// This is a FULL STATE snapshot, and it is handed to the broadcast channel from a detached
	// goroutine: two opens in quick succession race on that send, and the older snapshot can be
	// delivered last. There is no later event to self-correct from either — the one that closes
	// the round is the last bars_updated the table gets.
	//
	// It is assigned at the instant the snapshot is taken, not when it is sent, and it rises by
	// one every time. A client keeps the highest it has applied and DISCARDS anything lower.
	Seq uint64 `json:"seq"`
	// Prices maps a bar name to its frozen round price. A bar that has not priced is absent.
	Prices map[string]int `json:"prices"`
	// Characters is every character's standing balance on both bars.
	Characters []CharacterBarsPayload `json:"characters"`
	// Order is the projection of who acts next, highest key first.
	Order []BarSlotPayload `json:"order"`
}

type CharacterBarsPayload struct {
	CharacterID uuid.UUID `json:"characterId"`
	// ActionBalance and MoveBalance are the standing credit or debt on each clock. They are
	// fractional: the average behind them rarely divides evenly, and the fraction is kept
	// rather than rounded away.
	ActionBalance float64 `json:"actionBalance"`
	MoveBalance   float64 `json:"moveBalance"`
	// ActionSpeeds and MoveSpeeds are the speeds that have already acted this round. They
	// are public because the average they produce is what everyone is being ordered by.
	ActionSpeeds []int `json:"actionSpeeds"`
	MoveSpeeds   []int `json:"moveSpeeds"`
}

// BarSlotPayload is one slot of the general bar: who acts, which clocks it charges, and where
// in the round it lands. A combined action reports both bars and a single key — the one from
// its slower half.
type BarSlotPayload struct {
	ActorID uuid.UUID `json:"actorId"`
	Bars    []string  `json:"bars"`
	Key     float64   `json:"key"`
}

// ResolutionUpdatedPayload is the master's view of a turn's current resolution.
//
// It is a SLICE of service.TurnResolution, not the whole thing: it carries the mechanics and
// the projection, and nothing that would let a client reconstruct state it is not entitled
// to. Master-only for now — the calculation belongs to the master until the turn closes, and
// per-recipient projection is a later slice.
//
// Damage is projected, not applied. The HP only moves when the turn closes.
type ResolutionUpdatedPayload struct {
	TurnID    uuid.UUID                `json:"turnId"`
	IsSettled bool                     `json:"isSettled"`
	Action    RollResultPayload        `json:"action"`
	Targets   []CharacterResultPayload `json:"targets"`
}

// RollResultPayload is one test as the master reads it. The individual dice travel because
// a critical is the combination, not the sum.
type RollResultPayload struct {
	SkillName         string `json:"skillName"`
	SkillValue        int    `json:"skillValue"`
	DiceRolled        []int  `json:"diceRolled"`
	Total             int    `json:"total"`
	IsCritical        bool   `json:"isCritical"`
	IsCriticalFailure bool   `json:"isCriticalFailure"`
	Margin            *int   `json:"margin,omitempty"`
}

// CharacterResultPayload is what one attack did to one target.
type CharacterResultPayload struct {
	TargetID        uuid.UUID `json:"targetId"`
	Dodged          bool      `json:"dodged"`
	Defended        bool      `json:"defended"`
	DodgeTotal      int       `json:"dodgeTotal"`
	DefenseTotal    int       `json:"defenseTotal"`
	RawDamage       int       `json:"rawDamage"`
	DefenseApplied  int       `json:"defenseApplied"`
	ProjectedDamage int       `json:"projectedDamage"`
}

func newResolutionUpdatedPayload(turnID uuid.UUID, res *service.TurnResolution) ResolutionUpdatedPayload {
	p := ResolutionUpdatedPayload{TurnID: turnID, Targets: []CharacterResultPayload{}}
	if res == nil {
		return p
	}
	p.IsSettled = res.IsSettled
	p.Action = RollResultPayload{
		SkillName:         res.ActionResult.SkillName,
		SkillValue:        res.ActionResult.SkillValue,
		DiceRolled:        res.ActionResult.DiceRolled,
		Total:             res.ActionResult.Total,
		IsCritical:        res.ActionResult.IsCritical,
		IsCriticalFailure: res.ActionResult.IsCriticalFailure,
		Margin:            res.ActionResult.Margin,
	}
	for _, cr := range res.CharacterResults {
		p.Targets = append(p.Targets, CharacterResultPayload{
			TargetID:        cr.TargetID,
			Dodged:          cr.Dodged,
			Defended:        cr.Defended,
			DodgeTotal:      cr.Dodge.Total,
			DefenseTotal:    cr.Defense.Total,
			RawDamage:       cr.RawDamage,
			DefenseApplied:  cr.DefenseApplied,
			ProjectedDamage: cr.EffectiveDamage,
		})
	}
	return p
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
	BriefInitialDescription string `json:"briefInitialDescription"`
}

type SceneChangedPayload struct {
	SceneID                 uuid.UUID `json:"sceneId"`
	Category                string    `json:"category"`
	BriefInitialDescription string    `json:"briefInitialDescription"`
}

type MasterActionPayload struct {
	TargetIDs   []uuid.UUID          `json:"targetIds"`
	Skills      []ActionSkillPayload `json:"skills,omitempty"`
	Move        *MovePayload         `json:"move,omitempty"`
	Attack      *AttackPayload       `json:"attack,omitempty"`
	ActionSpeed *RollCheckPayload    `json:"actionSpeed,omitempty"`
	Interact    *InteractPayload     `json:"interact,omitempty"`
}

type MasterActionEnqueuedPayload struct {
	TargetIDs   []uuid.UUID          `json:"targetIds"`
	Skills      []ActionSkillPayload `json:"skills,omitempty"`
	Move        *MovePayload         `json:"move,omitempty"`
	Attack      *AttackPayload       `json:"attack,omitempty"`
	ActionSpeed *RollCheckPayload    `json:"actionSpeed,omitempty"`
	Interact    *InteractPayload     `json:"interact,omitempty"`
}

// ChangeRoundModePayload asks to switch the round regime. Master only.
type ChangeRoundModePayload struct {
	Mode string `json:"mode"` // "Free" | "Race"
}

// RoundModeChangedPayload announces the new regime to the whole table. The regime is public:
// everyone needs to know whether the bars are running.
type RoundModeChangedPayload struct {
	Mode string `json:"mode"`
}

// WallStateChangedPayload is broadcast to all clients when a wall's open/locked state changes.
type WallStateChangedPayload struct {
	WallID string `json:"wallId"`
	Open   bool   `json:"open"`
	Locked bool   `json:"locked"`
}

// WallHpChangedPayload is broadcast to all clients when a wall's HP or destroyed state changes.
type WallHpChangedPayload struct {
	WallID    string `json:"wallId"`
	HP        int    `json:"hp"`
	MaxHP     int    `json:"maxHp"`
	Destroyed bool   `json:"destroyed"`
}

// MapStateSyncPayload carries pieces, walls and grid.
// Sent by the master on WS connect to seed the room's in-memory state from the DB.
// Walls are full WallSegment objects so the room can perform movement blocking without
// additional DB queries.
// Pieces is a pointer so the room can tell "this sync carries no piece information"
// (field absent → nil → keep the current board) apart from "the board is empty"
// (field present but empty → clear it). The in-match master client syncs walls only;
// treating its absent pieces as an empty board would erase every LOS origin and leave
// players with no visibility polygons at all.
type MapStateSyncPayload struct {
	Pieces *[]PieceMovedPayload `json:"pieces,omitempty"`
	Walls  []WallSegmentPayload `json:"walls,omitempty"`
	Grid   *GridShapePayload    `json:"grid,omitempty"`
}

type Point2DPayload struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type VisibilityUpdatedPayload struct {
	VisiblePolygons [][]Point2DPayload `json:"visiblePolygons"`
}

type WallRevealedPayload struct {
	Wall WallSegmentPayload `json:"wall"`
}

// MapFullStatePayload is the per-player map_full_state (server→client).
type MapFullStatePayload struct {
	Pieces          []PieceMovedPayload  `json:"pieces"`
	Walls           []WallSegmentPayload `json:"walls,omitempty"`
	VisiblePolygons [][]Point2DPayload   `json:"visiblePolygons,omitempty"`
	FogMode         string               `json:"fogMode"`
}

// WallSegmentPayload is the WS wire-format mirror of entity.WallSegment, camelCase-tagged.
// Kept local to the game package (not shared with internal/app/api/map) so this delivery
// channel stays self-contained, mirroring the REST boundary DTOs in map_request.go/map_response.go.
type WallSegmentPayload struct {
	ID            string     `json:"id"`
	P1            [2]float64 `json:"p1"`
	P2            [2]float64 `json:"p2"`
	WallType      string     `json:"wallType"`
	Material      string     `json:"material"`
	DoorSubtype   *string    `json:"doorSubtype,omitempty"`
	WindowSubtype *string    `json:"windowSubtype,omitempty"`
	Move          bool       `json:"move"`
	Sense         string     `json:"sense"`
	Direction     string     `json:"direction"`
	Open          bool       `json:"open"`
	Locked        bool       `json:"locked"`
	HP            int        `json:"hp"`
	MaxHP         int        `json:"maxHp"`
	Resistance    int        `json:"resistance"`
	Destroyed     bool       `json:"destroyed"`
	Revealed      bool       `json:"revealed"`
}

// GridShapePayload is the WS wire-format mirror of entity.GridShape, camelCase-tagged.
type GridShapePayload struct {
	Kind      string  `json:"kind"`
	Cols      int     `json:"cols"`
	Rows      int     `json:"rows"`
	CellSize  float64 `json:"cellSize"`
	SkewRatio float64 `json:"skewRatio"`
	Rotation  float64 `json:"rotation"`
	Color     string  `json:"color"`
	Opacity   float64 `json:"opacity"`
	LineStyle string  `json:"lineStyle"`
}

func toWallSegmentPayload(w mapentity.WallSegment) WallSegmentPayload {
	p := WallSegmentPayload{
		ID:         w.ID,
		P1:         w.P1,
		P2:         w.P2,
		WallType:   string(w.WallType),
		Material:   string(w.Material),
		Move:       w.Move,
		Sense:      string(w.Sense),
		Direction:  string(w.Direction),
		Open:       w.Open,
		Locked:     w.Locked,
		HP:         w.HP,
		MaxHP:      w.MaxHP,
		Resistance: w.Resistance,
		Destroyed:  w.Destroyed,
		Revealed:   w.Revealed,
	}
	if w.DoorSubtype != nil {
		s := string(*w.DoorSubtype)
		p.DoorSubtype = &s
	}
	if w.WindowSubtype != nil {
		s := string(*w.WindowSubtype)
		p.WindowSubtype = &s
	}
	return p
}

func toEntityWallSegment(w WallSegmentPayload) mapentity.WallSegment {
	seg := mapentity.WallSegment{
		ID:         w.ID,
		P1:         w.P1,
		P2:         w.P2,
		WallType:   mapentity.WallType(w.WallType),
		Material:   mapentity.WallMaterial(w.Material),
		Move:       w.Move,
		Sense:      mapentity.SenseKind(w.Sense),
		Direction:  mapentity.WallDirection(w.Direction),
		Open:       w.Open,
		Locked:     w.Locked,
		HP:         w.HP,
		MaxHP:      w.MaxHP,
		Resistance: w.Resistance,
		Destroyed:  w.Destroyed,
		Revealed:   w.Revealed,
	}
	if w.DoorSubtype != nil {
		d := mapentity.DoorSubtype(*w.DoorSubtype)
		seg.DoorSubtype = &d
	}
	if w.WindowSubtype != nil {
		wi := mapentity.WindowSubtype(*w.WindowSubtype)
		seg.WindowSubtype = &wi
	}
	return seg
}

func toEntityGridShape(g GridShapePayload) mapentity.GridShape {
	return mapentity.GridShape{
		Kind:      mapentity.GridKind(g.Kind),
		Cols:      g.Cols,
		Rows:      g.Rows,
		CellSize:  g.CellSize,
		SkewRatio: g.SkewRatio,
		Rotation:  g.Rotation,
		Color:     g.Color,
		Opacity:   g.Opacity,
		LineStyle: mapentity.LineStyle(g.LineStyle),
	}
}
