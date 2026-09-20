package match

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
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
	FindActiveSession(ctx context.Context, matchUUID uuid.UUID) (*matchsession.ActiveSessionData, error)
	CloseSceneAndRound(ctx context.Context, sceneUUID, roundUUID uuid.UUID, at time.Time) error
	CloseRound(ctx context.Context, roundUUID uuid.UUID, at time.Time) error
	// FindMatchHistory returns the match's closed turns as the TREE the domain already is —
	// Scene -> Round -> Turn -> Action — not a flat list. See HistoryScene's own doc for why
	// flattening here would be the wrong call.
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

// HistoryRound is one round of a scene's history, with the turns closed inside it.
type HistoryRound struct {
	UUID       uuid.UUID
	Mode       string
	CreatedAt  time.Time
	FinishedAt *time.Time
	Turns      []HistoryTurn
}

// HistoryTurn is one closed turn: the action that drove it, whatever reactions answered it,
// and the settled collision that resulted — not the master's live edits, and not the master's
// own actions, which are never persisted (see FindMatchHistory's doc).
type HistoryTurn struct {
	UUID       uuid.UUID
	CreatedAt  time.Time
	FinishedAt time.Time
	Action     action.Action
	Reactions  []action.Action
	Resolution *service.TurnResolution // nil for a turn that resolved nothing
}
