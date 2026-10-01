package match

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
)

// TurnCloseData is everything PersistTurnClose needs to write one closed turn atomically.
// It replaced a five-argument parameter list that had already grown once (Resolution, here)
// and is about to grow again (Task 9's Overrides) — a struct absorbs that growth without
// reshuffling every call site again.
type TurnCloseData struct {
	Scene     *sceneentity.Scene
	Round     *roundentity.Round
	Turn      *turnentity.Turn
	Action    *action.Action
	MatchUUID uuid.UUID
	// Resolution is the SETTLED collision — margin, damage, ladder rung, chain state — the
	// one whose damage was actually applied. nil is allowed and writes SQL NULL: a turn that
	// resolved nothing (e.g. no character or wall target) still closes, and NULL says "no
	// collision" rather than a zero-value record's "a collision that produced zero".
	Resolution *service.TurnResolution
	// Overrides is what the master's edits DISPLACED while the turn was open — never the
	// edit itself, which the Action/Resolution above already carry. Drained from the session
	// (TakeOverridesFor), not peeked: a closed turn cannot be edited again, so nothing is
	// left behind to leak into a later turn's close.
	Overrides []match.OverriddenValue
	// MasterActions is what the master did INSIDE this turn while it was open (piece and wall
	// actions, turn notes), each already built the instant it was applied — views, TurnUUID,
	// HappenedAt — but written only now, in this same transaction: everything that happens
	// inside an open turn becomes durable together with that turn's close, or not at all
	// (owner decision, 2026-10-01). Drained from the room, like Overrides from the session.
	MasterActions []masteraction.Record
}

type IRepository interface {
	CreateMatch(ctx context.Context, match *match.Match) error
	UpdateMatch(ctx context.Context, match *match.Match) error
	DeleteMatch(ctx context.Context, matchUUID uuid.UUID) error
	GetMatch(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
	GetMatchCampaignUUID(ctx context.Context, matchUUID uuid.UUID) (uuid.UUID, error)
	StartMatch(ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time) error
	ListParticipantsByMatchUUID(ctx context.Context, matchUUID uuid.UUID) ([]*match.Participant, error)
	ListMatchesByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*match.Summary, error)
	ListPublicUpcomingMatches(ctx context.Context, after time.Time, masterUUID uuid.UUID) ([]*match.Summary, error)
}

// IRoundRepository handles persistence of scene/round/turn/action lifecycle.
type IRoundRepository interface {
	PersistTurnClose(ctx context.Context, d TurnCloseData) error
	// EnsureSceneAndRound writes the scene and round as rows if they are not rows yet —
	// idempotent, the same two inserts PersistTurnClose runs in its own transaction. Whatever
	// references the ACTIVE scene/round before the first turn of that round closes (a master
	// action recorded the instant it happens, spec §4.8) calls this first.
	EnsureSceneAndRound(ctx context.Context, matchUUID uuid.UUID, sc *sceneentity.Scene, rd *roundentity.Round) error
	FindActiveSession(ctx context.Context, matchUUID uuid.UUID) (*matchsession.ActiveSessionData, error)
	CloseSceneAndRound(ctx context.Context, sceneUUID, roundUUID uuid.UUID, at time.Time) error
	CloseRound(ctx context.Context, roundUUID uuid.UUID, at time.Time) error
	// FindMatchHistory returns the match's scenes, rounds and closed turns as the TREE the
	// domain already is — Scene -> Round -> Turn -> Action — not a flat list. See
	// HistoryScene's own doc for why flattening here would be the wrong call. A scene or round
	// with no turn is in it (B15); Events and MasterActions are NOT filled here — the use case
	// stitches them in from their own tables.
	FindMatchHistory(ctx context.Context, matchUUID uuid.UUID) ([]HistoryScene, error)
}

// HistoryScene is one logical block a match is organised into, with the rounds it contains.
//
// The tree shape (Scene -> Round -> Turn -> Action) is not decoration: the front renders
// action cards INSIDE the scope of each scene, so a flat list of turns would push the
// regrouping onto every consumer of this read path instead of doing it once, here.
type HistoryScene struct {
	UUID       uuid.UUID
	Category   string
	BriefDesc  string
	CreatedAt  time.Time
	FinishedAt *time.Time
	Rounds     []HistoryRound
}

// HistoryRound is one round of a scene's history, with the turns closed inside it and what
// happened inside it that is not a turn.
type HistoryRound struct {
	UUID       uuid.UUID
	Mode       string
	CreatedAt  time.Time
	FinishedAt *time.Time
	Turns      []HistoryTurn
	// Events is what happened inside this round outside any turn the tree holds — the regime
	// changes and the master actions with no turn (or whose turn was never written), already
	// projected for the reader, in the order they happened (spec §4.5, §4.8). Never nil once
	// the use case has run.
	Events []HistoryEvent
}

// HistoryEventKind discriminates a HistoryEvent.
type HistoryEventKind string

const (
	HistoryEventRoundModeChanged HistoryEventKind = "roundModeChanged"
	HistoryEventMasterAction     HistoryEventKind = "masterAction"
)

// HistoryEvent is one entry of HistoryRound.Events: exactly one of RoundModeChange and
// MasterAction is set, the one Kind names. At is when it happened — what Events is ordered by.
type HistoryEvent struct {
	Kind            HistoryEventKind
	At              time.Time
	RoundModeChange *matchevent.Event
	MasterAction    *masteraction.Record
}

// HistoryTurn is one closed turn: the action that drove it, whatever reactions answered it,
// the settled collision that resulted, and the master's own actions applied while it was open
// — not the master's live edits, which are not master actions (they live in
// overridden_action_values and the history shows the edited action).
type HistoryTurn struct {
	UUID       uuid.UUID
	CreatedAt  time.Time
	FinishedAt time.Time
	Action     action.Action
	Reactions  []action.Action
	Resolution *service.TurnResolution // nil for a turn that resolved nothing
	// MasterActions are the master actions recorded with this turn's UUID, already projected
	// for the reader (masteraction.Record.ProjectFor), in the order they happened. Never nil
	// once the use case has run.
	MasterActions []masteraction.Record
}
