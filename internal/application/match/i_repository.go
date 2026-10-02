package match

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchboard"
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
	// Board is the match's board as this close left it — the turn's move, its escapes and the
	// master's changes inside it — and Memories every player's fog memory with it, written in
	// this same transaction: the board on disk and the turns on disk never disagree. Board nil
	// means the match has no map attached (nothing to write a match_boards row for).
	Board    *matchboard.Board
	Memories []fogentity.PlayerMemory
	// MoveViews is what each player of the session saw of the action's move when the turn
	// opened — turn_opened's own move gate, run for every session player, connected or not
	// (owner decision, 2026-10-01): full, left, or no entry for "saw nothing". The master and
	// the actor's owner are not in it. Recorded the instant the move was shown and held with
	// the turn (turnWrites) like MasterActions; written to actions.move_views. nil when the
	// action has no move.
	MoveViews map[uuid.UUID]masteraction.View
	// LandingViews is, per escaping character (the CharacterResult's TargetID), who saw where
	// that escape's piece ended — a failed escape's landing, or an escaped one's own
	// destination — by the settled resolution_updated's landing gate, run at this close for
	// every session player. Written inside the resolution's escape entry. nil when no escape of
	// the turn moved its piece.
	LandingViews map[uuid.UUID]map[uuid.UUID]masteraction.View
	// StatusBars is every sheet this close damaged, with its bars as the session holds them
	// after the close (copies taken under r.mu), written to character_sheets in this same
	// transaction: one master command, one transaction (owner decision, 2026-10-02). A turn that
	// is not written leaves the rows as the last close did — the HP lives on in memory and the
	// next close that touches the sheet writes it whole. Empty when the close damaged nobody.
	StatusBars []SheetStatusBars
	// NextRound is the round born when this same close also ended Round — an open_next_action
	// that closes the last turn and finds nothing that can still pay its price. Round then
	// carries its finished_at, and both are written in this same transaction: the round's end,
	// the successor's birth and the turn are one master command, one transaction (owner
	// decision, 2026-10-02) — and the scene never holds two open rounds, nor none. nil when the
	// round goes on.
	NextRound *roundentity.Round
	// UnwrittenRoundEnds are rounds that ended in an EARLIER command whose write failed — still
	// open rows on disk, or not rows at all. They are written closed in this same transaction,
	// before anything here can give birth to the round the session is in now: written alone, that
	// round would be a second open round next to a stale one. Empty in the normal case.
	UnwrittenRoundEnds []RoundEnd
}

// RoundEnd is a round that ended, with the scene it ended in — both snapshots, the round carrying
// its finished_at. Written with the same idempotent upsert as EnsureSceneAndRound, so a round that
// was never a row lands closed and a finish already on the row never moves.
type RoundEnd struct {
	Scene *sceneentity.Scene
	Round *roundentity.Round
}

// SheetStatusBars is one sheet's three bars as a turn's close left them — detached copies
// (status.ReconstructBar), never the live bars of the session's sheet.
type SheetStatusBars struct {
	CharacterID           uuid.UUID
	Health, Stamina, Aura status.IStatusBarReader
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
	// PersistRoundClose writes round ends and the round the session is in now in one transaction:
	// every end in ends closed (each carries its finished_at and its own scene), then sc/next. It is
	// what a round that ends with no turn closing in the same command writes (open_next_action with
	// nothing open and no action that can still pay its price: ends = that round, next = its
	// successor), and what any write of a round goes through while an earlier round end is still
	// unwritten (ends = those, next = the round being written) — so no round is ever born open next
	// to a predecessor still open on disk. All rows go through the same idempotent upsert as
	// EnsureSceneAndRound. When a turn closes in the same command, PersistTurnClose writes them
	// instead (TurnCloseData.NextRound, UnwrittenRoundEnds).
	PersistRoundClose(ctx context.Context, matchUUID uuid.UUID, ends []RoundEnd, sc *sceneentity.Scene, next *roundentity.Round) error
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

	// MoveViews is what each player of the session saw LIVE of the action's move — the
	// verdict turn_opened's move gate reached for them at the opening (actions.move_views):
	// full (from and position), left (from only), no entry (neither). The master and the
	// actor's owner have no entry: they always see it all. nil on a row written before the
	// column existed — read as "nobody saw it" (fails closed). Read side only: the use case
	// turns it into MoveSight and clears it, so who saw what never reaches the wire.
	MoveViews map[uuid.UUID]masteraction.View
	// LandingViews is, per escaping character (the CharacterResult's TargetID), who saw
	// where that escape's piece ENDED live — a failed escape's landing, or an escape that
	// escaped's own destination (its reaction's move.position) — judged at the close by the
	// same gate as the settled resolution_updated's landing (the escape entry's landingViews in
	// turns.resolution). Same rules as MoveViews: master and the escaper's owner not stored,
	// absent fails closed, cleared by the use case.
	LandingViews map[uuid.UUID]map[uuid.UUID]masteraction.View

	// The three fields below are what this reader may see of WHERE pieces went — set by the
	// use case from the views above, applied by the REST mapping after actionwire.From, exactly
	// as room.go applies the live gate after From (the domain Move cannot say "no position", and
	// a nil Landing would change awaitsMaster). Their zero values hide: a HistoryTurn that never
	// went through the use case reveals nothing.

	// MoveSight is how much of the action's move WHERE this reader may see.
	MoveSight MoveSight
	// ShownLandings names the escaping characters whose escape.landing this reader may see.
	ShownLandings map[uuid.UUID]bool
	// ShownReactionMoves names the reactions (by ID) whose move.position this reader may see.
	ShownReactionMoves map[uuid.UUID]bool
}

// MoveSight is how much of a history action's move one reader saw live. The zero value is the
// category alone — fail closed: only the use case grants more.
type MoveSight string

const (
	// MoveSightNone keeps neither from nor position: the category alone (that the actor moved
	// is public). The zero value.
	MoveSightNone MoveSight = ""
	// MoveSightOrigin keeps move.from only — the reader saw the piece leave, not arrive.
	MoveSightOrigin MoveSight = "origin"
	// MoveSightWhole keeps move.from and move.position — the master, the actor's owner, a
	// reader who saw the destination, an action with no move.
	MoveSightWhole MoveSight = "whole"
)
