package match

import (
	"context"
	"errors"
	"log"
	"maps"
	"sort"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/cancelledaction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
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

// ICancelledActionLister reads the match's cancelled actions (cancelled_actions, spec
// 2026-10-10 §8.1 P6). The one real implementation is pgcancelledaction.Repository.
type ICancelledActionLister interface {
	ListByMatch(ctx context.Context, matchUUID uuid.UUID) ([]cancelledaction.Record, error)
}

// IStatusEffectEventLister lê os eventos de status da partida (status_effect_events) e a última
// lista gravada de cada efeito, em qualquer partida. A implementação real é
// pgstatuseffect.Repository.
type IStatusEffectEventLister interface {
	ListEventsByMatch(ctx context.Context, matchUUID uuid.UUID) ([]statuseffect.EventRecord, error)
	LatestListings(ctx context.Context, effectIDs []uuid.UUID) (map[uuid.UUID]statuseffect.Audience, error)
}

// IStatusEffectRevealReader lê as revelações do mestre: para cada efeito, a última (MAX em todas
// as partidas); um efeito nunca revelado não aparece. É uma interface à parte do
// IStatusEffectEventLister porque é outra tabela e outro assunto (o comando de revelar a grava),
// e assim as implementações e os fakes de eventos não mudam. A implementação real é
// pgstatuseffect.Repository.
type IStatusEffectRevealReader interface {
	RevealedAt(ctx context.Context, effectIDs []uuid.UUID) (map[uuid.UUID]time.Time, error)
}

// GetMatchHistoryUC is the read side of the Action History — a game surface with per-field
// visibility, not a log. Authorization mirrors GetMatchParticipantsUC exactly (public match, or
// master, or a campaign participant); the only thing this use case adds on top is running the
// tree through the SAME projection functions the WebSocket path runs, so a field can never end
// up public on one surface and hidden on the other.
//
// It also stitches in what is not a turn (B15, spec §4.5, §4.8; spec 2026-10-10 §8.1 P8): the
// round's regime changes, the master's own actions and the cancelled actions come from their
// own tables, read in separate queries — joining them into FindMatchHistory's already-large
// join would multiply its rows — and are hung on the tree here by round and turn UUID.
type GetMatchHistoryUC struct {
	matchRepo            IRepository
	roundRepo            IRoundRepository
	participationChecker CampaignParticipationChecker
	// events, masterActions and cancelledActions may be nil: nothing of that kind is stitched
	// in. Production wires all three (cmd/api/main.go).
	events           IEventLister
	masterActions    IMasterActionLister
	cancelledActions ICancelledActionLister
	// statusEvents pode ser nil: nenhum evento de status entra.
	statusEvents IStatusEffectEventLister
	// revealReader pode ser nil: nada é revelado (falha fechada).
	revealReader IStatusEffectRevealReader
}

func NewGetMatchHistoryUC(
	matchRepo IRepository,
	roundRepo IRoundRepository,
	participationChecker CampaignParticipationChecker,
	events IEventLister,
	masterActions IMasterActionLister,
	cancelledActions ICancelledActionLister,
) *GetMatchHistoryUC {
	return &GetMatchHistoryUC{
		matchRepo:            matchRepo,
		roundRepo:            roundRepo,
		participationChecker: participationChecker,
		events:               events,
		masterActions:        masterActions,
		cancelledActions:     cancelledActions,
	}
}

// WithStatusEffectEvents liga a leitura dos eventos de status; nil = nenhum.
func (uc *GetMatchHistoryUC) WithStatusEffectEvents(l IStatusEffectEventLister) *GetMatchHistoryUC {
	uc.statusEvents = l
	return uc
}

// WithStatusEffectReveals liga a leitura das revelações do mestre; nil = nada revelado.
func (uc *GetMatchHistoryUC) WithStatusEffectReveals(r IStatusEffectRevealReader) *GetMatchHistoryUC {
	uc.revealReader = r
	return uc
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

	// As actions canceladas são lidas antes das revelações: o registro do ator Escondido delas
	// também nomeia efeitos a consultar.
	var cancelled []cancelledaction.Record
	if uc.cancelledActions != nil {
		recs, err := uc.cancelledActions.ListByMatch(ctx, matchUUID)
		if err != nil {
			return nil, err
		}
		cancelled = recs
	}

	// Os eventos de status, a última lista e as revelações de cada efeito que o histórico
	// nomeia são lidos AQUI, uma vez por requisição, antes de projetar a árvore: as revelações
	// valem para a regra dos eventos (R2), para o escondido dos turnos (R4) e para o das
	// actions canceladas.
	sd, err := uc.readStatusData(ctx, matchUUID, scenes, cancelled)
	if err != nil {
		return nil, err
	}

	// Never nil, even when scenes itself already is not (FindMatchHistory's own doc): an
	// explicit make keeps this true regardless of what the gateway returns, so an empty
	// history marshals as [] on the wire, not null.
	//
	// Turns are built by append: a turn whose actor was Hidden from this reader (and not yet
	// revealed) is not in their history at all (Task 15, R4).
	cut := newConcealment()
	projected := make([]HistoryScene, len(scenes))
	for i, sc := range scenes {
		projected[i] = sc
		projected[i].Rounds = make([]HistoryRound, len(sc.Rounds))
		for j, ro := range sc.Rounds {
			projected[i].Rounds[j] = ro
			projected[i].Rounds[j].Turns = make([]HistoryTurn, 0, len(ro.Turns))
			for _, tu := range ro.Turns {
				pt, ok := uc.projectTurn(tu, viewer, userUUID, cut, sd.reveals)
				if !ok {
					continue
				}
				projected[i].Rounds[j].Turns = append(projected[i].Rounds[j].Turns, pt)
			}
			projected[i].Rounds[j].Events = make([]HistoryEvent, 0)
		}
	}

	if err := uc.stitch(ctx, matchUUID, projected, viewer, userUUID, cut, sd, cancelled); err != nil {
		return nil, err
	}
	return &GetMatchHistoryResult{Scenes: projected}, nil
}

// concealment é o que a projeção da árvore decidiu esconder deste leitor e o stitch precisa
// para as âncoras dos eventos de status: toda action/reação gravada (pelo turno dela) e as que
// o leitor não vê (de um turno ausente, ou uma reação cortada).
type concealment struct {
	stored map[uuid.UUID]uuid.UUID // action/reação → turno
	hidden map[uuid.UUID]bool
}

func newConcealment() concealment {
	return concealment{stored: map[uuid.UUID]uuid.UUID{}, hidden: map[uuid.UUID]bool{}}
}

// projectTurn é o turno como este leitor o vê, ou false quando o ator dele estava Escondido
// para o leitor (o turno some). Um reator escondido leva a reação e o resultado dele; um alvo
// escondido na abertura sai dos alvos e da resolução (Viewer.Hidden). O mestre recebe as marcas do que ainda está escondido
// (ConcealedBy). Os registros de quem estava Escondido nunca saem daqui.
func (uc *GetMatchHistoryUC) projectTurn(
	tu HistoryTurn, viewer service.Viewer, userUUID uuid.UUID, cut concealment,
	reveals statuseffect.RevealLookup,
) (HistoryTurn, bool) {
	cut.stored[tu.Action.GetID()] = tu.UUID
	for _, re := range tu.Reactions {
		cut.stored[re.GetID()] = tu.UUID
	}
	// O instante de cada registro é o real: a abertura do turno para a action e os alvos; a da
	// reação para o reator — a nunca aberta foi registrada no fechamento.
	concealedAt := func(m HiddenFrom, c uuid.UUID, at time.Time) bool {
		return !viewer.IsMaster && !viewer.Owns[c] && m.Conceals(c, userUUID, at, reveals)
	}
	concealed := func(m HiddenFrom, c uuid.UUID) bool { return concealedAt(m, c, tu.CreatedAt) }
	reactionAt := func(id uuid.UUID) time.Time {
		if at, ok := tu.ReactionOpenedAt[id]; ok {
			return at
		}
		return tu.FinishedAt
	}

	tv := viewer
	var hidden map[uuid.UUID]bool
	hide := func(c uuid.UUID) {
		if hidden == nil {
			hidden = map[uuid.UUID]bool{}
		}
		hidden[c] = true
	}
	for c := range tu.ActionHiddenFrom {
		if concealed(tu.ActionHiddenFrom, c) {
			hide(c)
		}
	}
	if hidden[tu.Action.GetActorID()] {
		cut.hidden[tu.Action.GetID()] = true
		for _, re := range tu.Reactions {
			cut.hidden[re.GetID()] = true
		}
		return HistoryTurn{}, false
	}
	// Os alvos saem só pelo registro da abertura (ActionHiddenFrom): o leitor viu o turno abrir
	// com eles, e nada se esconde para trás (dono do produto). O registro de uma reação corta a
	// reação e o resultado do reator, não o id dele nos alvos.
	onOpening := maps.Clone(hidden)
	reactions := make([]action.Action, 0, len(tu.Reactions))
	for _, re := range tu.Reactions {
		if m := tu.ReactionHiddenFrom[re.GetID()]; concealedAt(m, re.GetActorID(), reactionAt(re.GetID())) {
			// A reação some, e com ela o resultado do reator (ele é o alvo que a mandou).
			cut.hidden[re.GetID()] = true
			hide(re.GetActorID())
			continue
		}
		reactions = append(reactions, re)
	}
	tv.Hidden = hidden
	ta := viewer
	ta.Hidden = onOpening

	pt := tu
	// Read the fact instead of assuming it: HistoryTurn.FinishedAt is a plain time.Time (not a
	// pointer), and PersistTurnClose is the only write path today so it is never the zero value
	// in practice — but reading it costs the same as hardcoding true, and it does not break the
	// day an open turn crosses this path.
	settled := !tu.FinishedAt.IsZero()
	pt.Action = service.ProjectAction(tu.Action, ta, settled)
	pt.Reactions = make([]action.Action, len(reactions))
	for l, react := range reactions {
		pt.Reactions[l] = service.ProjectAction(react, ta, settled)
	}
	pt.Resolution = service.ProjectResolution(tu.Resolution, tv)
	pt.MoveSight = moveSightFor(tu, viewer, userUUID)
	pt.ShownLandings = shownLandingsFor(tu, viewer, userUUID)
	pt.ShownReactionMoves = shownReactionMovesFor(tu, viewer, userUUID)
	// Who saw what is not table data — the verdicts above are all this reader gets.
	pt.MoveViews, pt.LandingViews, pt.ReactionMoveViews = nil, nil, nil
	pt.ActionHiddenFrom, pt.ReactionHiddenFrom = nil, nil
	pt.ConcealedBy = nil
	if viewer.IsMaster {
		pt.ConcealedBy = concealedEffects(tu, reveals)
	}
	pt.MasterActions = make([]masteraction.Record, 0)
	pt.StatusEffects = make([]HistoryStatusEffect, 0)
	return pt, true
}

// concealedEffects são os Escondidos que ainda escondem algo do turno de alguém (não revelados
// no instante do registro), em ordem estável; nil quando nenhum. Todo registro conta: a lista
// de permissão sempre deixa alguém de fora (no mínimo quem não era da sessão). A sentinela de
// linha ilegível não é um efeito e fica de fora.
func concealedEffects(tu HistoryTurn, reveals statuseffect.RevealLookup) []uuid.UUID {
	seen := map[uuid.UUID]bool{}
	add := func(m HiddenFrom, at time.Time) {
		for _, h := range m {
			if h.Effect != uuid.Nil && !reveals.Covers(h.Effect, at) {
				seen[h.Effect] = true
			}
		}
	}
	add(tu.ActionHiddenFrom, tu.CreatedAt)
	for id, m := range tu.ReactionHiddenFrom {
		at, ok := tu.ReactionOpenedAt[id]
		if !ok {
			at = tu.FinishedAt
		}
		add(m, at)
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]uuid.UUID, 0, len(seen))
	for eff := range seen {
		out = append(out, eff)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].String() < out[b].String() })
	return out
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

// stitch hangs the match's events, master actions and cancelled actions on the projected tree,
// in place.
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
// A cancelled action always goes in its round's events, even when it was cancelled while a turn
// the tree holds was open: it never belonged to that turn — it left the queue before it could
// be one. Who cancelled and whose action it was reach every reader who perceived the actor at
// the instant; the action itself is cut per reader by cancelledActionFor. An actor Hidden at
// the cancel instant (Record.HiddenFrom, an allow-list) takes the whole entry away from every
// reader who did not perceive them — the master and the owner aside — until a reveal of that
// Hidden covers the instant (revealedAt >= cancelledAt).
//
// Um evento de status entra como uma master action (no turno dele quando o turno está na
// árvore, senão nos eventos do round), filtrado para o leitor pela regra única do domínio
// (statuseffect.VisibleInHistory) com a última lista do efeito, lida uma vez só para todos.
//
// An entry whose round is not in the tree is logged and dropped: every round is a row from
// birth (EnsureSceneAndRound), so this is drift, and there is no place in the tree to show it.
func (uc *GetMatchHistoryUC) stitch(
	ctx context.Context, matchUUID uuid.UUID, scenes []HistoryScene,
	viewer service.Viewer, userUUID uuid.UUID, cut concealment, sd statusData,
	cancelled []cancelledaction.Record,
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
	statusEvents, latest := sd.recs, sd.latest
	if len(events) == 0 && len(records) == 0 && len(cancelled) == 0 && len(statusEvents) == 0 {
		return nil
	}

	// Pointers into the slices Get already built — nothing appends to Rounds or Turns from
	// here on, so they stay valid. A turn this reader lost (its actor was Hidden from them) is
	// not here: its master actions and unanchored status events fall to the round's events.
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
		switch e.Kind {
		case matchevent.KindRoundModeChanged:
			ro.Events = append(ro.Events, HistoryEvent{
				Kind: HistoryEventRoundModeChanged, At: e.CreatedAt, RoundModeChange: &ev,
			})
		case matchevent.KindClockStarted, matchevent.KindClockAdjusted, matchevent.KindClockRealTime:
			// o relógio é público: todo leitor vê o evento, com o payload como foi gravado
			ro.Events = append(ro.Events, HistoryEvent{
				Kind: HistoryEventKind(e.Kind), At: e.CreatedAt, ClockChange: &ev,
			})
		default:
			log.Printf("GetMatchHistory: match %s — event %s has unknown kind %q; dropped", matchUUID, e.UUID, e.Kind)
		}
	}

	for _, r := range records {
		proj, ok := r.ProjectFor(viewer.IsMaster, userUUID)
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

	for _, c := range cancelled {
		if !viewer.SeesAllOf(c.ActorUUID) && c.HiddenFrom.Conceals(c.ActorUUID, userUUID, c.CancelledAt, sd.reveals) {
			continue
		}
		ro, ok := rounds[c.RoundUUID]
		if !ok {
			log.Printf("GetMatchHistory: match %s — cancelled action %s points at round %s, which is not in the history; dropped",
				matchUUID, c.UUID, c.RoundUUID)
			continue
		}
		ro.Events = append(ro.Events, HistoryEvent{
			Kind: HistoryEventActionCancelled, At: c.CancelledAt, CancelledAction: cancelledActionFor(c, viewer),
		})
	}

	for _, se := range statusEvents {
		// Uma âncora que não nomeia action/reação gravada do turno (uma reação sentada que foi
		// substituída depois) não é âncora: o evento é do turno.
		se.Event.ActionID = storedAnchor(se.Event, cut)
		h := statuseffect.HistoryReading{
			Reader: userUUID, IsMaster: viewer.IsMaster, RevealedAt: sd.reveals,
			AnchorVisible: se.Event.ActionID == nil || !cut.hidden[*se.Event.ActionID],
		}
		h.Latest, h.HasLatest = latest[se.Effect.ID]
		if !statuseffect.VisibleInHistory(se, h) {
			continue
		}
		hs := statusEventFor(se, viewer.IsMaster, sd.reveals)
		if se.Event.TurnID != nil {
			if tu, ok := turns[*se.Event.TurnID]; ok {
				tu.StatusEffects = append(tu.StatusEffects, hs)
				continue
			}
		}
		ro, ok := rounds[se.Event.RoundID]
		if !ok {
			log.Printf("GetMatchHistory: match %s — status effect event %s points at round %s, which is not in the history; dropped",
				matchUUID, se.Event.ID, se.Event.RoundID)
			continue
		}
		ro.Events = append(ro.Events, HistoryEvent{
			Kind: HistoryEventStatusEffect, At: hs.HappenedAt, StatusEffect: &hs,
		})
	}

	// Stable: the lists arrive ordered by (time, uuid) from their tables, so a tie keeps that
	// order — and between entries tied to the instant, the one appended first (regime change,
	// then master action, then cancelled action, then status event) comes first.
	for _, ro := range rounds {
		sort.SliceStable(ro.Events, func(a, b int) bool { return ro.Events[a].At.Before(ro.Events[b].At) })
	}
	for _, tu := range turns {
		sort.SliceStable(tu.MasterActions, func(a, b int) bool {
			return tu.MasterActions[a].HappenedAt.Before(tu.MasterActions[b].HappenedAt)
		})
		sort.SliceStable(tu.StatusEffects, func(a, b int) bool {
			return tu.StatusEffects[a].HappenedAt.Before(tu.StatusEffects[b].HappenedAt)
		})
	}
	return nil
}

// statusData é o que a requisição leu sobre os efeitos de status: os eventos da partida, a
// última lista de cada efeito e as revelações do mestre (nil = nada revelado).
type statusData struct {
	recs    []statuseffect.EventRecord
	latest  map[uuid.UUID]statuseffect.Audience
	reveals statuseffect.RevealLookup
}

// readStatusData lê os eventos de status da partida e, numa chamada só cada, a última lista dos
// efeitos que eles nomeiam e a última revelação de todo efeito que o histórico usa: os dos
// eventos, os Escondidos de HiddenFrom deles e os dos hidden_from das actions e reações da
// árvore e das actions canceladas. Sem o leitor de revelações, nada é revelado. A sentinela uuid.Nil (linha ilegível,
// escondida de todos) nunca entra: nunca é revelável.
func (uc *GetMatchHistoryUC) readStatusData(
	ctx context.Context, matchUUID uuid.UUID, scenes []HistoryScene, cancelled []cancelledaction.Record,
) (statusData, error) {
	var sd statusData
	if uc.statusEvents != nil {
		recs, err := uc.statusEvents.ListEventsByMatch(ctx, matchUUID)
		if err != nil {
			return statusData{}, err
		}
		if len(recs) > 0 {
			seen := make(map[uuid.UUID]bool, len(recs))
			ids := make([]uuid.UUID, 0, len(recs))
			for _, r := range recs {
				if !seen[r.Effect.ID] {
					seen[r.Effect.ID] = true
					ids = append(ids, r.Effect.ID)
				}
			}
			latest, err := uc.statusEvents.LatestListings(ctx, ids)
			if err != nil {
				return statusData{}, err
			}
			sd.recs, sd.latest = recs, latest
		}
	}
	if uc.revealReader == nil {
		return sd, nil
	}

	need := map[uuid.UUID]bool{}
	add := func(id uuid.UUID) {
		if id != uuid.Nil {
			need[id] = true
		}
	}
	addMap := func(m HiddenFrom) {
		for _, h := range m {
			add(h.Effect)
		}
	}
	for _, r := range sd.recs {
		add(r.Effect.ID)
		for _, eff := range r.Event.HiddenFrom {
			add(eff)
		}
	}
	for _, c := range cancelled {
		addMap(c.HiddenFrom)
	}
	for _, sc := range scenes {
		for _, ro := range sc.Rounds {
			for _, tu := range ro.Turns {
				addMap(tu.ActionHiddenFrom)
				for _, m := range tu.ReactionHiddenFrom {
					addMap(m)
				}
			}
		}
	}
	if len(need) == 0 {
		return sd, nil
	}
	ids := make([]uuid.UUID, 0, len(need))
	for id := range need {
		ids = append(ids, id)
	}
	at, err := uc.revealReader.RevealedAt(ctx, ids)
	if err != nil {
		return statusData{}, err
	}
	sd.reveals = func(id uuid.UUID) (time.Time, bool) {
		if id == uuid.Nil {
			return time.Time{}, false
		}
		t, ok := at[id]
		return t, ok
	}
	return sd, nil
}

// storedAnchor é a âncora do evento quando ela nomeia uma action/reação gravada do turno dele;
// nil caso contrário.
func storedAnchor(ev statuseffect.Event, cut concealment) *uuid.UUID {
	if ev.ActionID == nil || ev.TurnID == nil {
		return nil
	}
	if tu, ok := cut.stored[*ev.ActionID]; !ok || tu != *ev.TurnID {
		return nil
	}
	return ev.ActionID
}

// statusEventFor é o evento como este leitor o recebe: a lista de quem vê (Audience e Public)
// e se uma revelação o cobre (Revealed) só vão ao mestre.
func statusEventFor(se statuseffect.EventRecord, isMaster bool, reveals statuseffect.RevealLookup) HistoryStatusEffect {
	e := se.Effect.Clone()
	out := HistoryStatusEffect{
		UUID: se.Event.ID, Change: se.Event.Change, Effect: e,
		HappenedAt: se.Event.HappenedAt,
	}
	if se.Event.TurnID != nil {
		id := *se.Event.TurnID
		out.TurnUUID = &id
	}
	if se.Event.ActionID != nil {
		id := *se.Event.ActionID
		out.ActionID = &id
	}
	if isMaster {
		public := se.Event.Listing.Everyone
		// Só a revelação do PRÓPRIO efeito conta: é nele que o botão "revelar" age, e revelar o
		// Escondido não abre a lista de um status secreto.
		revealed := reveals.Covers(se.Effect.ID, se.Event.HappenedAt)
		out.Public, out.Revealed = &public, &revealed
	} else {
		out.Effect.Audience = nil
	}
	return out
}

// cancelledActionFor is c as this reader may see it (spec 2026-10-10 §8.1 P8): the master the
// whole action, the actor's owner what they declared (the cut of their own ownQueue), anyone
// else only who cancelled. Both readers who get the action see all of it under
// service.ProjectAction, so the deny-list has nothing to strip; only the cut differs.
func cancelledActionFor(c cancelledaction.Record, viewer service.Viewer) *HistoryCancelledAction {
	out := &HistoryCancelledAction{UUID: c.UUID, ActorUUID: c.ActorUUID}
	switch {
	case viewer.IsMaster:
		out.Cut = CancelledActionCutFull
	case viewer.Owns[c.ActorUUID]:
		out.Cut = CancelledActionCutDeclaration
	default:
		return out
	}
	a := c.Action
	out.Action = &a
	return out
}
