package match

import (
	"context"
	"errors"
	"log"
	"sort"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

// GetMatchHistoryResult is the match's history, ALREADY projected per viewer — Scene -> Round
// -> Turn -> Action, the same tree FindMatchHistory reads, with service.ProjectAction /
// service.ProjectResolution run over every action, reaction and resolution in it, and every
// master action run through masteraction.Record.ProjectFor. The one cut the domain cannot
// carry — where pieces went (a move's from/position, a reaction's position, an escape's
// landing) as far as this reader saw it live — is decided here too (HistoryTurn.MoveSight,
// ShownReactionMoves, ShownLandings) and only APPLIED by the REST mapping after
// actionwire.From; otherwise the handler serializes this straight to the wire.
type GetMatchHistoryResult struct {
	Scenes []HistoryScene
}

type IGetMatchHistory interface {
	Get(ctx context.Context, matchUUID, userUUID uuid.UUID) (*GetMatchHistoryResult, error)
}

// IEventLister reads the match's events (match_events, spec §4.5) — what happened inside a
// round that is not a turn and is nobody's action. The one real implementation is
// pgmatchevent.Repository.
type IEventLister interface {
	ListByMatch(ctx context.Context, matchUUID uuid.UUID) ([]matchevent.Event, error)
}

// IMasterActionLister reads the match's master actions (master_actions, spec §4.8). The one
// real implementation is pgmasteraction.Repository.
type IMasterActionLister interface {
	ListByMatch(ctx context.Context, matchUUID uuid.UUID) ([]masteraction.Record, error)
}

// GetMatchHistoryUC is the read side of the Action History — a game surface with per-field
// visibility, not a log. Authorization mirrors GetMatchParticipantsUC exactly (public match, or
// master, or a campaign participant); the only thing this use case adds on top is running the
// tree through the SAME projection functions the WebSocket path runs, so a field can never end
// up public on one surface and hidden on the other.
//
// It also stitches in what is not a turn (B15, spec §4.5, §4.8): the round's regime changes and
// the master's own actions come from their own tables, read in separate queries — joining them
// into FindMatchHistory's already-large join would multiply its rows — and are hung on the tree
// here by round and turn UUID.
type GetMatchHistoryUC struct {
	matchRepo            IRepository
	roundRepo            IRoundRepository
	participationChecker CampaignParticipationChecker
	// events and masterActions may be nil: nothing of that kind is stitched in. Production
	// wires both (cmd/api/main.go).
	events        IEventLister
	masterActions IMasterActionLister
}

func NewGetMatchHistoryUC(
	matchRepo IRepository,
	roundRepo IRoundRepository,
	participationChecker CampaignParticipationChecker,
	events IEventLister,
	masterActions IMasterActionLister,
) *GetMatchHistoryUC {
	return &GetMatchHistoryUC{
		matchRepo:            matchRepo,
		roundRepo:            roundRepo,
		participationChecker: participationChecker,
		events:               events,
		masterActions:        masterActions,
	}
}

func (uc *GetMatchHistoryUC) Get(
	ctx context.Context, matchUUID, userUUID uuid.UUID,
) (*GetMatchHistoryResult, error) {
	match, err := uc.matchRepo.GetMatch(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	viewerIsMaster := match.MasterUUID == userUUID
	if !match.IsPublic && !viewerIsMaster {
		ok, err := uc.participationChecker.ExistsSheetInCampaign(
			ctx, userUUID, match.CampaignUUID,
		)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, auth.ErrInsufficientPermissions
		}
	}

	// Owns is built from the match's PARTICIPANTS, not a live MatchSession: the history is
	// read for a match that may have no session running right now, so there is no
	// MatchSession.GetCharToPlayer() to borrow the way room.go's publishResolution does. A
	// participant's Sheet.PlayerUUID is the caller's own sheets — nil for an NPC, which by
	// construction nobody but the master can own.
	participants, err := uc.matchRepo.ListParticipantsByMatchUUID(ctx, matchUUID)
	if err != nil {
		return nil, err
	}
	owns := make(map[uuid.UUID]bool, len(participants))
	for _, p := range participants {
		if p.Sheet.PlayerUUID != nil && *p.Sheet.PlayerUUID == userUUID {
			owns[p.Sheet.UUID] = true
		}
	}
	viewer := service.Viewer{IsMaster: viewerIsMaster, Owns: owns}

	scenes, err := uc.roundRepo.FindMatchHistory(ctx, matchUUID)
	if err != nil {
		return nil, err
	}

	// Never nil, even when scenes itself already is not (FindMatchHistory's own doc): an
	// explicit make keeps this true regardless of what the gateway returns, so an empty
	// history marshals as [] on the wire, not null.
	projected := make([]HistoryScene, len(scenes))
	for i, sc := range scenes {
		projected[i] = sc
		projected[i].Rounds = make([]HistoryRound, len(sc.Rounds))
		for j, ro := range sc.Rounds {
			projected[i].Rounds[j] = ro
			projected[i].Rounds[j].Turns = make([]HistoryTurn, len(ro.Turns))
			for k, tu := range ro.Turns {
				pt := tu
				// Read the fact instead of assuming it: HistoryTurn.FinishedAt is a plain
				// time.Time (not a pointer), and PersistTurnClose is the only write path today
				// so it is never the zero value in practice — but reading it costs the same as
				// hardcoding true, and it does not break the day an open turn crosses this path.
				settled := !tu.FinishedAt.IsZero()
				pt.Action = service.ProjectAction(tu.Action, viewer, settled)
				pt.Reactions = make([]action.Action, len(tu.Reactions))
				for l, react := range tu.Reactions {
					pt.Reactions[l] = service.ProjectAction(react, viewer, settled)
				}
				pt.Resolution = service.ProjectResolution(tu.Resolution, viewer)
				pt.MoveSight = moveSightFor(tu, viewer, userUUID)
				pt.ShownLandings = shownLandingsFor(tu, viewer, userUUID)
				pt.ShownReactionMoves = shownReactionMovesFor(tu, viewer, userUUID)
				// Who saw what is not table data — the verdicts above are all this reader gets.
				pt.MoveViews, pt.LandingViews, pt.ReactionMoveViews = nil, nil, nil
				pt.MasterActions = make([]masteraction.Record, 0)
				projected[i].Rounds[j].Turns[k] = pt
			}
			projected[i].Rounds[j].Events = make([]HistoryEvent, 0)
		}
	}

	if err := uc.stitch(ctx, matchUUID, projected, viewerIsMaster, userUUID); err != nil {
		return nil, err
	}
	return &GetMatchHistoryResult{Scenes: projected}, nil
}

// moveSightFor is how much of tu's move WHERE this reader saw live (owner decision,
// 2026-10-01): the master and the actor's owner the whole move, anyone else what the verdict
// recorded for them at the opening says (HistoryTurn.MoveViews) — never recomputed here: the
// fog at read time is not the fog of that moment. No entry, or a row from before the verdicts
// were recorded, is the category alone: fail closed, like Record.ProjectFor.
func moveSightFor(tu HistoryTurn, viewer service.Viewer, userUUID uuid.UUID) MoveSight {
	if tu.Action.Move == nil || viewer.SeesAllOf(tu.Action.GetActorID()) {
		return MoveSightWhole
	}
	switch tu.MoveViews[userUUID] {
	case masteraction.ViewFull:
		return MoveSightWhole
	case masteraction.ViewLeft:
		return MoveSightOrigin
	default:
		return MoveSightNone
	}
}

// sawWhereItEnded reports whether this reader may see where the escape of cr put its piece:
// the master and the escaping character's owner always; anyone else only when the verdict
// recorded at the close (HistoryTurn.LandingViews) has an entry for them. A row from before
// the verdicts has none: fail closed.
func sawWhereItEnded(tu HistoryTurn, cr service.CharacterResult, viewer service.Viewer, userUUID uuid.UUID) bool {
	if viewer.SeesAllOf(cr.TargetID) {
		return true
	}
	_, saw := tu.LandingViews[cr.TargetID][userUUID]
	return saw
}

// shownLandingsFor names the escapes whose landing this reader may see (sawWhereItEnded). nil
// when none.
func shownLandingsFor(tu HistoryTurn, viewer service.Viewer, userUUID uuid.UUID) map[uuid.UUID]bool {
	if tu.Resolution == nil {
		return nil
	}
	var shown map[uuid.UUID]bool
	for _, cr := range tu.Resolution.CharacterResults {
		if cr.Escape == nil || cr.Escape.Landing == nil || !sawWhereItEnded(tu, cr, viewer, userUUID) {
			continue
		}
		if shown == nil {
			shown = map[uuid.UUID]bool{}
		}
		shown[cr.TargetID] = true
	}
	return shown
}

// shownReactionMovesFor names the reactions whose move.position this reader may see — as it was
// seen LIVE, the same rule as the turn's own move. A reaction reaches the table when the master
// opens it (reaction_opened), and what each player was shown of its destination then is the
// verdict recorded with the turn (HistoryTurn.ReactionMoveViews): whoever saw it there sees it
// here, even when the escape failed. The other way to have seen it is the piece itself: an
// escape that ESCAPED walks its piece there at the close, and the piece_moved reached whoever
// saw that cell (sawWhereItEnded). A reaction never opened and never arrived (an old row, or a
// move that moved nothing): shown to nobody but the master and the reactor's owner. nil when
// none.
func shownReactionMovesFor(tu HistoryTurn, viewer service.Viewer, userUUID uuid.UUID) map[uuid.UUID]bool {
	escaped := map[uuid.UUID]service.CharacterResult{}
	if tu.Resolution != nil {
		for _, cr := range tu.Resolution.CharacterResults {
			if cr.Escape != nil && cr.Escape.Escaped {
				escaped[cr.ReactionID] = cr
			}
		}
	}
	var shown map[uuid.UUID]bool
	for _, react := range tu.Reactions {
		if react.Move == nil {
			continue
		}
		cr, ok := escaped[react.GetID()]
		sawAtOpening := tu.ReactionMoveViews[react.GetID()][userUUID] == masteraction.ViewFull
		if !viewer.SeesAllOf(react.GetActorID()) && !sawAtOpening &&
			(!ok || !sawWhereItEnded(tu, cr, viewer, userUUID)) {
			continue
		}
		if shown == nil {
			shown = map[uuid.UUID]bool{}
		}
		shown[react.GetID()] = true
	}
	return shown
}

// stitch hangs the match's events and master actions on the projected tree, in place.
//
// A master action goes INSIDE the turn it was recorded with when that turn is in the tree;
// otherwise — recorded with no turn, or with a turn that was never written because a restart
// lost it before it closed — it goes in its round's events. The second case is not an error to
// hide: the master action happened, the saved board confirms it, and dropping it because its
// turn did not survive would erase something the table saw (spec §4.8).
//
// Each master action is projected for this reader first (masteraction.Record.ProjectFor — the
// same decision the live dispatch made, recorded in Views): an entry the reader did not see
// does not exist for them. A regime change is public, like the regime itself.
//
// An entry whose round is not in the tree is logged and dropped: every round is a row from
// birth (EnsureSceneAndRound), so this is drift, and there is no place in the tree to show it.
func (uc *GetMatchHistoryUC) stitch(
	ctx context.Context, matchUUID uuid.UUID, scenes []HistoryScene,
	viewerIsMaster bool, userUUID uuid.UUID,
) error {
	var events []matchevent.Event
	if uc.events != nil {
		evs, err := uc.events.ListByMatch(ctx, matchUUID)
		if err != nil {
			return err
		}
		events = evs
	}
	var records []masteraction.Record
	if uc.masterActions != nil {
		recs, err := uc.masterActions.ListByMatch(ctx, matchUUID)
		if err != nil {
			return err
		}
		records = recs
	}
	if len(events) == 0 && len(records) == 0 {
		return nil
	}

	// Pointers into the slices Get already allocated at their final length — nothing appends
	// to Rounds or Turns from here on, so they stay valid.
	rounds := map[uuid.UUID]*HistoryRound{}
	turns := map[uuid.UUID]*HistoryTurn{}
	for i := range scenes {
		for j := range scenes[i].Rounds {
			ro := &scenes[i].Rounds[j]
			rounds[ro.UUID] = ro
			for k := range ro.Turns {
				turns[ro.Turns[k].UUID] = &ro.Turns[k]
			}
		}
	}

	for _, e := range events {
		ro, ok := rounds[e.RoundUUID]
		if !ok {
			log.Printf("GetMatchHistory: match %s — event %s (%s) points at round %s, which is not in the history; dropped",
				matchUUID, e.UUID, e.Kind, e.RoundUUID)
			continue
		}
		ev := e
		ro.Events = append(ro.Events, HistoryEvent{
			Kind: HistoryEventRoundModeChanged, At: e.CreatedAt, RoundModeChange: &ev,
		})
	}

	for _, r := range records {
		proj, ok := r.ProjectFor(viewerIsMaster, userUUID)
		if !ok {
			continue
		}
		if r.TurnUUID != nil {
			if tu, ok := turns[*r.TurnUUID]; ok {
				tu.MasterActions = append(tu.MasterActions, proj)
				continue
			}
		}
		ro, ok := rounds[r.RoundUUID]
		if !ok {
			log.Printf("GetMatchHistory: match %s — master action %s (%s) points at round %s, which is not in the history; dropped",
				matchUUID, r.UUID, r.Kind, r.RoundUUID)
			continue
		}
		rec := proj
		ro.Events = append(ro.Events, HistoryEvent{
			Kind: HistoryEventMasterAction, At: r.HappenedAt, MasterAction: &rec,
		})
	}

	// Stable: the two lists arrive ordered by (time, uuid) from their tables, so a tie keeps
	// that order — and between a regime change and a master action tied to the instant, the
	// regime change (appended first) comes first.
	for _, ro := range rounds {
		sort.SliceStable(ro.Events, func(a, b int) bool { return ro.Events[a].At.Before(ro.Events[b].At) })
	}
	for _, tu := range turns {
		sort.SliceStable(tu.MasterActions, func(a, b int) bool {
			return tu.MasterActions[a].HappenedAt.Before(tu.MasterActions[b].HappenedAt)
		})
	}
	return nil
}
