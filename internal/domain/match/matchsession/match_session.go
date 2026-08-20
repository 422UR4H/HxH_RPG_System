package matchsession

import (
	"math"
	"slices"
	"time"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// PiecePositionSource yields the current world positions of a player's pieces.
// Implemented by room.go (which owns the live piece payloads), keeping the session
// decoupled from delivery types.
type PiecePositionSource interface {
	PlayerPiecePositions(playerID uuid.UUID) []service.Point2D
}

type MatchSession struct {
	matchUUID   uuid.UUID
	activeScene *scene.Scene
	activeRound *round.Round
	activeQueue action.PriorityQueue
	// charSheets and statuses are keyed by sheetUUID — the same ID the board pieces
	// carry as CharacterID. The combat entity is the character, not the player: the
	// master drives several characters at once, and NPCs have no player at all.
	charSheets map[uuid.UUID]*csSheet.CharacterSheet
	statuses   map[uuid.UUID]*match.CharacterStatus
	// participants stays keyed by playerUUID: authorization is a per-player question.
	// charToPlayer is the bridge between the two axes, and what the fog of war reads.
	participants   map[uuid.UUID]*match.Participant
	roundOrch      service.RoundOrchestrator
	scheduler      service.RoundScheduler
	turnResolver   service.TurnResolver
	scenePersisted bool
	roundPersisted bool
	walls          map[string]mapentity.WallSegment // keyed by wall ID; nil until SyncMapState
	grid           mapentity.GridShape              // full grid shape; CellSize 0 until SyncMapState
	fogMode        fog.FogMode
	memories       map[uuid.UUID]*fog.PlayerMemory
	visCache       map[uuid.UUID][]service.VisibilityPolygon
	charToPlayer   map[string]uuid.UUID
	pieceSource    PiecePositionSource
	// rules is the per-match configuration. The embedded defaults are used until the REST
	// surface for the master to choose them exists — that is a slice of its own.
	rules match.MatchRules
	// weapons is the static weapon catalogue, the source of the damage dice.
	weapons *item.WeaponsManager
	// rollSource is where the dice come from. nil means production. Tests set it so a
	// phase whose done-criteria name exact numbers never depends on luck.
	rollSource service.RollSource
}

func NewMatchSession(
	matchUUID uuid.UUID,
	charSheets map[uuid.UUID]*csSheet.CharacterSheet,
	participants []*match.Participant,
) *MatchSession {
	pMap, charToPlayer, statuses := indexParticipants(participants)
	return &MatchSession{
		matchUUID:    matchUUID,
		activeScene:  scene.NewScene(enum.Roleplay, ""),
		activeRound:  round.NewRound(enum.Free),
		activeQueue:  action.NewActionPriorityQueue(nil),
		charSheets:   charSheets,
		statuses:     statuses,
		participants: pMap,
		roundOrch:    service.RoundOrchestrator{},
		scheduler:    service.RoundScheduler{},
		rules:        match.NewDefaultMatchRules(),
		weapons:      item.NewWeaponsManagerFactory().Build(),
		turnResolver: service.TurnResolver{},
		charToPlayer: charToPlayer,
		memories:     make(map[uuid.UUID]*fog.PlayerMemory),
		visCache:     make(map[uuid.UUID][]service.VisibilityPolygon),
	}
}

func NewMatchSessionWithState(
	matchUUID uuid.UUID,
	charSheets map[uuid.UUID]*csSheet.CharacterSheet,
	participants []*match.Participant,
	activeScene *scene.Scene,
	activeRound *round.Round,
) *MatchSession {
	pMap, charToPlayer, statuses := indexParticipants(participants)
	return &MatchSession{
		matchUUID:      matchUUID,
		activeScene:    activeScene,
		activeRound:    activeRound,
		activeQueue:    action.NewActionPriorityQueue(nil),
		charSheets:     charSheets,
		statuses:       statuses,
		participants:   pMap,
		roundOrch:      service.RoundOrchestrator{},
		scheduler:      service.RoundScheduler{},
		rules:          match.NewDefaultMatchRules(),
		weapons:        item.NewWeaponsManagerFactory().Build(),
		turnResolver:   service.TurnResolver{},
		scenePersisted: true,
		roundPersisted: true,
		charToPlayer:   charToPlayer,
		memories:       make(map[uuid.UUID]*fog.PlayerMemory),
		visCache:       make(map[uuid.UUID][]service.VisibilityPolygon),
	}
}

// indexParticipants splits the roster along its two axes: every character gets a
// combat status (NPCs included), and only player-owned characters get an
// authorization entry and a fog bridge.
func indexParticipants(participants []*match.Participant) (
	map[uuid.UUID]*match.Participant,
	map[string]uuid.UUID,
	map[uuid.UUID]*match.CharacterStatus,
) {
	pMap := make(map[uuid.UUID]*match.Participant, len(participants))
	charToPlayer := make(map[string]uuid.UUID)
	statuses := make(map[uuid.UUID]*match.CharacterStatus, len(participants))

	for _, p := range participants {
		if p.Sheet.UUID != uuid.Nil {
			statuses[p.Sheet.UUID] = match.NewCharacterStatus()
		}
		if p.Sheet.PlayerUUID == nil {
			continue // NPC: no player to authorize, no per-player fog memory
		}
		pMap[*p.Sheet.PlayerUUID] = p
		if p.Sheet.UUID != uuid.Nil {
			charToPlayer[p.Sheet.UUID.String()] = *p.Sheet.PlayerUUID
		}
	}
	return pMap, charToPlayer, statuses
}

func (s *MatchSession) GetMatchUUID() uuid.UUID      { return s.matchUUID }
func (s *MatchSession) GetActiveRound() *round.Round { return s.activeRound }
func (s *MatchSession) GetActiveScene() *scene.Scene { return s.activeScene }
func (s *MatchSession) IsRoundPersisted() bool       { return s.roundPersisted }
func (s *MatchSession) IsScenePersisted() bool       { return s.scenePersisted }

func (s *MatchSession) GetRules() match.MatchRules { return s.rules }

// SetRollSource replaces the dice source. Production never calls it; tests do.
func (s *MatchSession) SetRollSource(src service.RollSource) { s.rollSource = src }

func (s *MatchSession) MarkRoundPersisted() {
	s.scenePersisted = true
	s.roundPersisted = true
}

func (s *MatchSession) ChangeScene(category enum.SceneCategory, briefDesc string) (*scene.Scene, *round.Round, error) {
	if s.activeRound.HasOpenTurn() {
		return nil, nil, ErrRoundHasOpenTurn
	}
	now := time.Now()
	s.activeRound.Close(now)
	s.activeScene.Close(now)

	oldScene := s.activeScene
	oldRound := s.activeRound

	s.activeScene = scene.NewScene(category, briefDesc)
	s.activeRound = round.NewRound(enum.Free)
	s.scenePersisted = false
	s.roundPersisted = false

	return oldScene, oldRound, nil
}

// SetRoundMode puts the active round into a regime. Callers hold room.mu for writing: this
// changes how every later selection is scored.
func (s *MatchSession) SetRoundMode(mode enum.RoundMode) {
	s.roundOrch.SetMode(s.activeRound, mode)
}

func (s *MatchSession) EnqueueMasterAction(ma *action.MasterAction) error {
	t := s.activeRound.CurrentTurn()
	if t == nil || t.GetFinishedAt() != nil {
		return ErrNoActiveTurn
	}
	ma.SetHappenedAt(time.Now())
	t.AddMasterAction(*ma)
	return nil
}

// GetCharSheet returns a character's sheet. charID is the sheet UUID — the same ID the
// board pieces carry as CharacterID — not the player UUID.
func (s *MatchSession) GetCharSheet(charID uuid.UUID) (*csSheet.CharacterSheet, error) {
	sheet, ok := s.charSheets[charID]
	if !ok {
		return nil, ErrCharSheetNotFound
	}
	return sheet, nil
}

// GetCharacterStatus returns a character's live combat state. The pointer is the
// session's own, so mutating it is a write to session state: callers must hold room.mu
// for writing, not just RLock, even though this method only reads the map.
func (s *MatchSession) GetCharacterStatus(charID uuid.UUID) (*match.CharacterStatus, error) {
	status, ok := s.statuses[charID]
	if !ok {
		return nil, ErrCharacterStatusNotFound
	}
	return status, nil
}

// TurnTransition is what one act of the master's baton produces: the turn that closed, the
// turn that opened, the resolution of each, and whatever the closing actually applied.
//
// It is a struct rather than several return values because the two operations that produce
// it — open the next action, pull one out of order — are the same shape, and opening a
// reaction will join the list.
type TurnTransition struct {
	Closed           *turn.Turn
	Opened           *turn.Turn
	ClosedResolution *service.TurnResolution
	OpenedResolution *service.TurnResolution
	// Damaged is what the close actually wrote to a sheet. Empty on the first transition of
	// a round, when nothing closed.
	Damaged []DamagedCharacter
	// RoundExhausted reports that no pending action passes the gate that applies to it, which
	// is what ends a Race round. It is not an error: the actions still queued keep the roll
	// they already made and belong to the next round. The caller closes the round.
	RoundExhausted bool
}

// DamagedCharacter is one applied HP reduction. The caller persists it — the session holds
// the live sheet, the gateway holds the row.
type DamagedCharacter struct {
	CharacterID uuid.UUID
	Sheet       *csSheet.CharacterSheet
	Damage      int
	NewHP       int
}

// BarState implements service.BarStateSource: the carry that crossed into this round, and the
// speeds that have already acted on that bar. An unknown character reads as an empty bar
// rather than failing the whole scheduling pass.
func (s *MatchSession) BarState(charID uuid.UUID, bar action.Bar) (float64, []int) {
	status, ok := s.statuses[charID]
	if !ok {
		return 0, nil
	}
	b := status.BarFor(bar)
	return b.Balance, b.Speeds
}

// RoundPrices returns the frozen price of each bar that has priced this round. Read by the
// delivery layer to publish the general bar.
func (s *MatchSession) RoundPrices() map[action.Bar]int {
	out := map[action.Bar]int{}
	for _, bar := range []action.Bar{action.BarAction, action.BarMove} {
		if p, frozen := s.activeRound.Price(bar); frozen {
			out[bar] = p
		}
	}
	return out
}

// CharacterIDs returns every character the session holds combat state for, NPCs included.
func (s *MatchSession) CharacterIDs() []uuid.UUID {
	out := make([]uuid.UUID, 0, len(s.statuses))
	for id := range s.statuses {
		out = append(out, id)
	}
	return out
}

// ProjectedOrder is the general bar: the pending actions that can still pay, highest key
// first. It carries no action identity — the queue is secret, the order is public.
func (s *MatchSession) ProjectedOrder() []service.OrderSlot {
	return s.scheduler.ProjectOrder(s.scheduleInput())
}

// scheduleInput assembles what one scheduling decision reads.
func (s *MatchSession) scheduleInput() service.ScheduleInput {
	return service.ScheduleInput{Queue: &s.activeQueue, Round: s.activeRound, Bars: s}
}

func (s *MatchSession) OpenNextAction() (*TurnTransition, error) {
	// Prices freeze on the first selection that sees a bar with pending work — before any
	// gate is evaluated, because the gate is measured against the price.
	s.scheduler.FreezePrices(s.scheduleInput())

	if s.activeRound.GetMode() != enum.Race {
		// Free has no price, no average and no carry-over; the rolled speed is the order and
		// nothing gates. An empty queue is still an error there, as it always was.
		tr := s.closeOpenTurn()
		opened, err := s.roundOrch.NextAction(s.activeRound, &s.activeQueue)
		if err != nil {
			return tr, err
		}
		tr.Opened = opened
		tr.OpenedResolution = s.ResolveTurn(opened)
		return tr, nil
	}

	next := s.scheduler.SelectNext(s.scheduleInput())
	tr := s.closeOpenTurn()
	if next == nil {
		// Nothing pending can still pay. The round is over — the caller closes it.
		tr.RoundExhausted = true
		return tr, nil
	}

	// PullAction is reused deliberately: once the scheduler has chosen, "open the next one"
	// and "pull this one out of order" are the same operation. The master's explicit
	// pull_action stays ungated on purpose — anticipating an action is their prerogative.
	opened, err := s.roundOrch.PullAction(s.activeRound, &s.activeQueue, next.GetID())
	if err != nil {
		return tr, err
	}
	s.recordActed(next)
	tr.Opened = opened
	tr.OpenedResolution = s.ResolveTurn(opened)
	return tr, nil
}

// recordActed appends an action's speed to EVERY bar it was paid from, at the moment it opens.
//
// Every bar, because a combined action charges both: an investida costs a movement and a blow,
// and both averages move because of it.
//
// On OPEN, never on enqueue: ResourceBar.Speeds means "the speeds that acted", and that is
// what makes an action which never reached the price roll over to the next round untouched.
//
// And only in Race, because Race is the only regime with an economy. Free freezes no price
// (RoundScheduler.FreezePrices returns immediately) and settleBars skips every bar that never
// priced, so a speed recorded under Free would be charged by nothing and reset by nothing. It
// would survive the moment the master switches the round to Race and make BarEconomy.IsEligible
// read the character as having already acted, denying their FIRST action of the disputed round.
// Keeping the gate here rather than at the call site closes it for both callers at once:
// OpenNextAction only reaches this in Race already, PullAction is ungated by design.
func (s *MatchSession) recordActed(a *action.Action) {
	if s.activeRound.GetMode() != enum.Race {
		return
	}
	status, ok := s.statuses[a.GetActorID()]
	if !ok {
		return
	}
	for _, bar := range a.Bars() {
		status.BarFor(bar).RecordSpeed(a.SpeedOn(bar))
	}
}

func (s *MatchSession) PullAction(id uuid.UUID) (*TurnTransition, error) {
	tr := s.closeOpenTurn()
	s.scheduler.FreezePrices(s.scheduleInput())
	opened, err := s.roundOrch.PullAction(s.activeRound, &s.activeQueue, id)
	if err != nil {
		return tr, err
	}
	act := opened.GetAction()
	s.recordActed(&act)
	tr.Opened = opened
	tr.OpenedResolution = s.ResolveTurn(opened)
	return tr, nil
}

// closeOpenTurn ends the turn currently under the baton, resolves it one last time and
// applies what that resolution says. Both ways of opening the next turn go through it, so
// the damage lands in exactly one place.
func (s *MatchSession) closeOpenTurn() *TurnTransition {
	tr := &TurnTransition{}
	if !s.activeRound.HasOpenTurn() {
		return tr
	}
	tr.Closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
	tr.ClosedResolution = s.ResolveTurn(tr.Closed)
	tr.Damaged = s.applyResolution(tr.ClosedResolution)
	// The ledger moves one turn forward with the turn: what was scoped to this turn dies, and
	// what was earned FOR the next one becomes live. It happens AFTER applyResolution, so a
	// modifier that was meant to count in this turn still counted.
	s.advanceLedgers()
	return tr
}

// advanceLedgers moves every character's ledger one turn forward. Every character, not just
// the ones who acted: a penalty is carried by whoever earned it, and they may not have moved.
func (s *MatchSession) advanceLedgers() {
	for _, status := range s.statuses {
		status.AdvanceTurn()
	}
}

// ResolveTurn computes the resolution snapshot for t. Pure — it never touches a sheet.
// The master reads it as a projection, over and over, as reactions land.
func (s *MatchSession) ResolveTurn(t *turn.Turn) *service.TurnResolution {
	if t == nil {
		return nil
	}
	return s.turnResolver.Resolve(service.ResolveInput{
		Turn:    t,
		Sheets:  s.charSheets,
		Targets: s,
		Rules:   s.rules,
		Weapons: s.weapons,
	})
}

// applyResolution writes a resolution's effective damage to the target sheets, once.
//
// This is the moment the dry run stops being a dry run. Everything before it recalculated
// freely — every master edit, every colliding reaction — precisely because nothing had been
// applied. Called only from the turn-closing path.
func (s *MatchSession) applyResolution(res *service.TurnResolution) []DamagedCharacter {
	if res == nil {
		return nil
	}
	var out []DamagedCharacter
	for _, cr := range res.CharacterResults {
		if cr.EffectiveDamage <= 0 {
			continue
		}
		sheet, ok := s.charSheets[cr.TargetID]
		if !ok || sheet == nil {
			continue
		}
		bar, ok := sheet.GetAllStatusBar()[enum.Health]
		if !ok {
			continue
		}
		newHP := bar.DecreaseAt(cr.EffectiveDamage)
		out = append(out, DamagedCharacter{
			CharacterID: cr.TargetID,
			Sheet:       sheet,
			Damage:      cr.EffectiveDamage,
			NewHP:       newHP,
		})
	}
	return out
}

// AttachReaction validates that the caller may answer this attack, then attaches the reaction
// to the open turn and re-resolves it.
//
// Two axes, exactly as EnqueueAction: authorization is a per-PLAYER question and combat is a
// per-CHARACTER one, bridged by charToPlayer. On top of that, a reaction has a third
// constraint an action does not — only someone the attack is AIMED AT may answer it.
func (s *MatchSession) AttachReaction(playerUUID uuid.UUID, r *action.Action) (*service.TurnResolution, error) {
	owner, ok := s.charToPlayer[r.GetActorID().String()]
	if !ok || owner != playerUUID {
		return nil, ErrReactionActorMismatch
	}
	t := s.activeRound.CurrentTurn()
	if t == nil {
		return nil, service.ErrNoCurrentTurn
	}
	act := t.GetAction()
	if !slices.Contains(act.TargetID, r.GetActorID()) {
		return nil, ErrReactorNotTargeted
	}

	s.rollActionDice(r)

	// A free reaction derives nothing, records nothing and consumes nothing. That IS the
	// discount: done in the exact instant, without opening the guard, it gives the action back.
	if !r.ReactionKind.IsFree() {
		consumed := s.consumePendingFor(r)
		// Swapping what you were going to do costs Disadvantage — the engine rolls again and
		// keeps the worse of the two speeds. It is a MODE of reading the dice, never an
		// Amount: RollAttempts already holds both sets and the bias only picks which one.
		// With nothing queued there was no swap, so there is no penalty.
		systemBias := 0
		if consumed {
			systemBias = -1
		}
		s.deriveSpeeds(r, systemBias)
		s.chargeReactionBars(r)
	}

	if err := s.roundOrch.AttachReaction(s.activeRound, r); err != nil {
		return nil, err
	}
	return s.ResolveTurn(s.activeRound.CurrentTurn()), nil
}

// consumePendingFor pulls this character's about-to-open action off the queue, once per bar the
// reaction charges, and reports whether anything was taken.
//
// A combined action sits on both bars and is counted once — it leaves on the first bar that
// finds it and is simply not there for the second.
func (s *MatchSession) consumePendingFor(r *action.Action) bool {
	consumed := false
	for _, bar := range r.ReactionKind.Bars() {
		victim := s.scheduler.BestPendingFor(s.scheduleInput(), r.GetActorID(), bar)
		if victim == nil {
			continue
		}
		s.activeQueue.ExtractByID(victim.GetID())
		consumed = true
	}
	return consumed
}

// chargeReactionBars debits the reaction's own speed on every bar its kind charges.
//
// At ATTACH, not at open, and the difference matters: an action records on open because one
// that never reaches the price rolls into the next round untouched, so Speeds has to mean "acted
// for real". A reaction has nowhere to roll to — it lives inside the turn it answered — and
// opening it is a narration event. Narration must not move a number. The consequence lines up
// with Phase 5: a reaction attached and never opened has already paid, which is exactly what
// the close-turn dialogue assumes when it says such a reaction "enters the calculation but
// loses its moment to narrate".
//
// Race-only, for the same reason recordActed is: settleBars skips a bar that never priced, so a
// speed recorded under Free would be charged by nothing and reset by nothing — and would then
// make IsEligible read the character as having already acted the moment the master switches to
// Race, denying them their first action of the disputed round.
func (s *MatchSession) chargeReactionBars(r *action.Action) {
	if s.activeRound.GetMode() != enum.Race {
		return
	}
	status, ok := s.statuses[r.GetActorID()]
	if !ok {
		return
	}
	for _, bar := range r.ReactionKind.Bars() {
		status.BarFor(bar).RecordSpeed(r.SpeedOn(bar))
	}
}

// OpenReaction passes the microphone to one reaction on the open turn and re-resolves it.
//
// Opening is table conduct — "now it is this person's turn to narrate". The recomputation is
// the side effect, and it recomputes rather than re-rolling: every die fell at attach.
//
// It does NOT charge anything. The bars were debited when the reaction arrived (see
// chargeReactionBars) precisely so that narrating cannot move a number.
func (s *MatchSession) OpenReaction(reactionID uuid.UUID) (*service.TurnResolution, error) {
	t := s.activeRound.CurrentTurn()
	if t == nil {
		return nil, service.ErrNoCurrentTurn
	}
	if t.GetFinishedAt() != nil {
		return nil, ErrTurnAlreadyClosed
	}
	if !t.OpenReaction(reactionID) {
		return nil, ErrReactionNotFound
	}
	return s.ResolveTurn(t), nil
}

func (s *MatchSession) CloseTurn() (*turn.Turn, error) {
	return s.roundOrch.CloseTurnErr(s.activeRound, time.Now())
}

func (s *MatchSession) CloseRound() (*round.Round, error) {
	if s.activeRound.HasOpenTurn() {
		return nil, ErrRoundHasOpenTurn
	}
	s.settleBars()
	for _, status := range s.statuses {
		status.ExpireModifiers(match.LifetimeEndOfRound)
	}
	mode := s.activeRound.GetMode()
	closed := s.roundOrch.CloseRound(s.activeRound, time.Now())
	s.activeRound = round.NewRound(mode)
	s.roundPersisted = false
	return closed, nil
}

// settleBars turns each character's round into the balance they carry into the next one, then
// clears the round's speed history.
//
//	acted:  min(carry + mean(acted) − len(acted) × price, price)
//	silent: min(carry + price, price)   — standing still trades an action for time, and the
//	                                      trade is worth exactly one round's price
//
// The ceiling is the price on both branches, which is why standing still stops paying after a
// single round instead of compounding: whoever acts also reaches the ceiling in a few rounds.
//
// A bar that never priced is left untouched — nobody acted on that clock, so no round happened
// on it, and inventing a floor there would hand out free time.
//
// Nothing is done to the queue. An action that never reached the price was never recorded as
// having acted, so it simply belongs to the next round, carrying the roll it already made.
func (s *MatchSession) settleBars() {
	eco := service.BarEconomy{}
	for _, bar := range []action.Bar{action.BarAction, action.BarMove} {
		price, frozen := s.activeRound.Price(bar)
		if !frozen {
			continue
		}
		for _, status := range s.statuses {
			b := status.BarFor(bar)
			b.Balance = eco.CloseBalance(b.Balance, b.Speeds, price)
			b.ResetRound()
		}
	}
}

// EnqueueAction validates that playerUUID may act and that the character they are acting
// through is theirs, then puts the action in the queue.
//
// Two axes, deliberately: authorization is a per-PLAYER question, and combat is a
// per-CHARACTER one. charToPlayer is the bridge. a.actorID is the sheet UUID, so the
// resolver can index the actor's sheet in the same map it indexes the target's.
func (s *MatchSession) EnqueueAction(playerUUID uuid.UUID, a *action.Action) error {
	if _, ok := s.participants[playerUUID]; !ok {
		return ErrParticipantNotFound
	}
	owner, ok := s.charToPlayer[a.GetActorID().String()]
	if !ok || owner != playerUUID {
		return ErrActionActorMismatch
	}
	s.rollActionDice(a)
	s.deriveSpeeds(a, 0)
	s.activeQueue.Insert(a)
	return nil
}

// PendingActions returns the actions still waiting for the master, in insertion order. Read
// by the delivery layer to publish the general bar, and by tests.
func (s *MatchSession) PendingActions() []*action.Action { return s.activeQueue.All() }

// deriveSpeeds turns the dice that just fell into the numbers the round is ordered by:
// Action.Speed.Result for the action bar, Move.FinalSpeed for the move bar.
//
// It runs once, when the action arrives, and it is the only place a speed is produced. The
// master never re-rolls a player's die, so nothing downstream ever recomputes it.
//
// systemBias is the engine-imposed advantage/disadvantage for this one derivation — a reaction
// that swapped out a queued action passes -1, everyone else passes 0. It is a MODE of reading
// the dice that already fell, never an Amount: RollAttempts holds both attempts, and the bias
// only picks which one pickAttempt reads.
func (s *MatchSession) deriveSpeeds(a *action.Action, systemBias int) {
	if a == nil {
		return
	}
	sheet := s.charSheets[a.GetActorID()]
	if sheet == nil {
		return
	}
	calc := service.RollCalculator{}
	var ledger *match.ModifierLedger
	if status, ok := s.statuses[a.GetActorID()]; ok {
		ledger = &status.Ledger
	}

	// actionSpeed: always Legerity. Passive in Free, rolled in Race.
	//
	// The ledger applies here and nowhere else in a collision: the accumulated difference a
	// character carries is always an actionSpeed adjustment, never a hit adjustment. It is
	// what makes the repel ladder produce a duel — two characters facing each other speed up
	// against each other — without anyone programming duels.
	a.Speed.SkillName = enum.Legerity.String()
	a.Speed.SkillValue = skillValueOn(sheet, enum.Legerity)
	a.Speed.Result = calc.Derive(s.rules, a.Speed.Attempts, service.RollInput{
		SkillName:  a.Speed.SkillName,
		SkillValue: a.Speed.SkillValue,
		Passive:    s.activeRound.GetMode() != enum.Race,
		Condition:  a.Speed.Context.Condition,
		Ledger:     ledger,
		Dimension:  match.DimActionSpeed,
		SystemBias: systemBias,
	}).Total

	if a.Move == nil {
		return
	}
	// moveSpeed: the skill comes from the category, and so does whether it rolls at all.
	// Dash is an acceleration and is tested; Shift is controlled and takes the passive value.
	// Anything else never reaches here — the mapper refuses it at the WS boundary.
	skill, passive := enum.Accelerate, false
	if a.Move.Category == enum.Shift {
		skill, passive = enum.Brake, true
	}
	if a.Move.Speed == nil {
		a.Move.Speed = &action.RollCheck{}
	}
	a.Move.Speed.SkillName = skill.String()
	a.Move.Speed.SkillValue = skillValueOn(sheet, skill)
	a.Move.Speed.Result = calc.Derive(s.rules, a.Move.Speed.Attempts, service.RollInput{
		SkillName:  a.Move.Speed.SkillName,
		SkillValue: a.Move.Speed.SkillValue,
		Passive:    passive,
		Condition:  a.Move.Speed.Context.Condition,
		// No ledger on the move bar: the accumulated difference is an actionSpeed bonus.
		Dimension:  match.DimActionSpeed,
		SystemBias: systemBias,
	}).Total
	// Charge is deliberately not read. The momentum accumulating into CharacterStatus.Velocity
	// is the movement slice's, and the bar works without it (spec §5, Fase 3).
	a.Move.FinalSpeed = a.Move.Speed.Result
}

// skillValueOn reads a skill off the sheet, contributing 0 for a name the sheet does not
// know. The WS boundary already rejects unknown names, so reaching here means an internal one.
func skillValueOn(cs *csSheet.CharacterSheet, name enum.SkillName) int {
	v, err := cs.GetValueForTestOfSkill(name)
	if err != nil {
		return 0
	}
	return v
}

// rollActionDice drops the dice for every test the action carries, once, the moment it
// arrives. Derive is then free to run again on every master edit and on every colliding
// reaction without a single new die — "the master never re-rolls a player's die".
//
// A RollCheck whose dice already fell is left alone, so calling this twice is harmless.
func (s *MatchSession) rollActionDice(a *action.Action) {
	if a == nil {
		return
	}
	calc := service.RollCalculator{}

	// test is the match's dice set: hit, skill, actionSpeed, feint, defense, dodge.
	test := func(rc *action.RollCheck) {
		if rc == nil || !rc.Attempts.IsEmpty() {
			return
		}
		rc.Attempts = calc.Roll(s.rules, s.rollSource)
	}

	// A Free round takes the passive value for actionSpeed — there is no dispute over who
	// acts first, so there is nothing to roll for.
	if s.activeRound.GetMode() == enum.Race {
		test(&a.Speed.RollCheck)
	}
	test(a.Feint)
	for i := range a.Skills {
		test(&a.Skills[i].RollCheck)
	}
	if a.Move != nil {
		// A Shift takes the dice set's average and rolls NOTHING. Rolling anyway would be
		// harmless in production and poisonous in a test: a scripted RollSource would be
		// drained by the phantom roll and every number after it would shift.
		if a.Move.Category != enum.Shift {
			// A combined action (charge/investida) may arrive with Move.Speed still nil —
			// only Attack was filled in by the caller. Vivify it here so there is somewhere
			// for the dice to land; deriveSpeeds fills in the skill afterwards.
			if a.Move.Speed == nil {
				a.Move.Speed = &action.RollCheck{}
			}
			test(a.Move.Speed)
		}
		test(a.Move.Charge)
	}
	if a.Defense != nil {
		test(&a.Defense.RollCheck)
	}
	if a.Dodge != nil {
		test(&a.Dodge.RollCheck)
	}
	if a.Repel != nil {
		test(&a.Repel.RollCheck)
	}
	if a.Attack != nil {
		test(&a.Attack.Hit)
		test(a.Attack.Charge)
		// Damage is the OTHER family of roll: the weapon's own dice, not the match set.
		// Only Primary, because damage has no advantage.
		if a.Attack.Damage.Attempts.IsEmpty() {
			if sides, err := service.WeaponDice(a.Attack.Weapon, s.weapons); err == nil {
				a.Attack.Damage.Attempts = action.RollAttempts{
					Primary: calc.RollDice(sides, s.rollSource),
				}
			}
		}
	}
}

// SyncMapState seeds or replaces the session's in-memory map state.
// Called by room.go when the match starts, seeding from pre-match lobby state.
func (s *MatchSession) SyncMapState(walls []mapentity.WallSegment, grid mapentity.GridShape) {
	s.walls = make(map[string]mapentity.WallSegment, len(walls))
	for _, w := range walls {
		s.walls[w.ID] = w
	}
	s.grid = grid
}

func (s *MatchSession) GetWall(id string) (mapentity.WallSegment, bool) {
	w, ok := s.walls[id]
	return w, ok
}

func (s *MatchSession) UpdateWall(w mapentity.WallSegment) {
	if s.walls == nil {
		s.walls = make(map[string]mapentity.WallSegment)
	}
	s.walls[w.ID] = w
}

func (s *MatchSession) GetWalls() []mapentity.WallSegment {
	result := make([]mapentity.WallSegment, 0, len(s.walls))
	for _, w := range s.walls {
		result = append(result, w)
	}
	return result
}

func (s *MatchSession) GetGrid() mapentity.GridShape { return s.grid }
func (s *MatchSession) GetGridSize() float64         { return s.grid.CellSize }

// CategorizeTarget returns the kind of entity the given UUID identifies.
//
// Characters are looked up in statuses, keyed by sheetUUID — which is what an
// Action.TargetID actually carries, since it comes from a board piece's CharacterID.
// It used to consult participants, keyed by playerUUID; the two key spaces never
// intersect, so TargetKindCharacter was unreachable.
//
// Characters are checked first so a character UUID is never mis-routed as a wall.
func (s *MatchSession) CategorizeTarget(id uuid.UUID) service.TargetKind {
	if _, ok := s.statuses[id]; ok {
		return service.TargetKindCharacter
	}
	if _, ok := s.walls[id.String()]; ok {
		return service.TargetKindWallSegment
	}
	return service.TargetKindUnknown
}

func (s *MatchSession) SetPieceSource(src PiecePositionSource) { s.pieceSource = src }

// SyncPlayerMemories seeds per-player memory from persisted records (nil seeds empty
// memories). Resets the visibility cache.
func (s *MatchSession) SyncPlayerMemories(mems []fog.PlayerMemory, mode fog.FogMode) {
	s.fogMode = mode
	s.memories = make(map[uuid.UUID]*fog.PlayerMemory, len(mems))
	for i := range mems {
		m := mems[i]
		s.memories[m.PlayerID] = &m
	}
	s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
}

// memoryFor returns the existing memory for playerID, or lazily creates one.
func (s *MatchSession) memoryFor(playerID uuid.UUID) *fog.PlayerMemory {
	if s.memories == nil {
		s.memories = make(map[uuid.UUID]*fog.PlayerMemory)
	}
	m, ok := s.memories[playerID]
	if !ok {
		// TODO(persistence): MapID is uuid.Nil because MatchSession doesn't carry the
		// active map's UUID yet. Thread the real mapUUID in (constructor or
		// SyncPlayerMemories) before wiring the repository, so persisted rows don't all
		// collide on the (match_id, map_id, player_id) unique key with map_id = Nil.
		m = fog.NewPlayerMemory(playerID, s.matchUUID, uuid.Nil)
		s.memories[playerID] = m
	}
	return m
}

// RecomputeVisibility recomputes a player's LOS, caches the polygons, and (in explored
// mode) records every wall now in sight into that player's memory.
func (s *MatchSession) RecomputeVisibility(playerID uuid.UUID) ([]service.VisibilityPolygon, error) {
	walls := s.GetWalls()
	// The board edges block vision too, which keeps the polygon inside the map instead
	// of spilling out to maxRadius in every open direction.
	losWalls := append(service.ToLOSWalls(walls), service.BoundaryLOSWalls(s.grid)...)
	// Bound the polygon to the map diagonal (+20% margin) so a wall-less map still
	// produces a finite polygon. Falls back to visRadius when CellSize is 0.
	maxRadius := math.Hypot(float64(s.grid.Cols)*s.grid.CellSize, float64(s.grid.Rows)*s.grid.CellSize) * 1.2

	var positions []service.Point2D
	if s.pieceSource != nil {
		positions = s.pieceSource.PlayerPiecePositions(playerID)
	}
	polys := make([]service.VisibilityPolygon, 0, len(positions))
	for _, pos := range positions {
		polys = append(polys, service.ComputeVisibilityPolygon(pos, losWalls, maxRadius))
	}

	// One call, after the loop: wallInLOS already iterates every polygon internally.
	// `walls`, never `losWalls` — the latter carries phantom board-edge segments that
	// must never enter a player's memory.
	if s.fogMode == fog.FogModeExplored {
		s.memoryFor(playerID).Remember(service.SeenWalls(walls, polys))
	}

	if s.visCache == nil {
		s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
	}
	s.visCache[playerID] = polys
	return polys, nil
}

// RecomputeAllVisibility recomputes LOS for all participants.
func (s *MatchSession) RecomputeAllVisibility() error {
	for pid := range s.participants {
		if _, err := s.RecomputeVisibility(pid); err != nil {
			return err
		}
	}
	return nil
}

// GetVisibility returns the cached visibility polygons for a player.
func (s *MatchSession) GetVisibility(playerID uuid.UUID) []service.VisibilityPolygon {
	return s.visCache[playerID]
}

// InvalidateVisibilityCache clears all cached visibility polygons.
func (s *MatchSession) InvalidateVisibilityCache() {
	s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
}

// RevealSecretDoor marks a wall as revealed (secret door is now visible to all)
// and invalidates the visibility cache so the next recompute sees the change.
func (s *MatchSession) RevealSecretDoor(wallID string) {
	if w, ok := s.walls[wallID]; ok {
		w.Revealed = true
		s.walls[wallID] = w
	}
	s.InvalidateVisibilityCache()
}

func (s *MatchSession) GetFogMode() fog.FogMode               { return s.fogMode }
func (s *MatchSession) GetCharToPlayer() map[string]uuid.UUID { return s.charToPlayer }

// GetPlayerMemory returns the player's memory, or nil when they have none yet.
func (s *MatchSession) GetPlayerMemory(playerID uuid.UUID) (*fog.PlayerMemory, bool) {
	m, ok := s.memories[playerID]
	return m, ok
}

func (s *MatchSession) GetAllPlayerMemories() []fog.PlayerMemory {
	out := make([]fog.PlayerMemory, 0, len(s.memories))
	for _, m := range s.memories {
		out = append(out, *m)
	}
	return out
}

// PlayerIDs returns the player UUIDs of all participants.
func (s *MatchSession) PlayerIDs() []uuid.UUID {
	out := make([]uuid.UUID, 0, len(s.participants))
	for pid := range s.participants {
		out = append(out, pid)
	}
	return out
}
