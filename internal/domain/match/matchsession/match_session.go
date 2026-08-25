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
	// removedSkillDice parks the dice of skills the master took off an action, keyed by action
	// then skill name, so putting one back is not a free re-roll. It lives for as long as the
	// session does; a turn's entries are drained into the audit when it closes.
	removedSkillDice map[uuid.UUID]map[string]action.RollAttempts
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

// CurrentTurnID reads the open turn's ID, or uuid.Nil when there is none. Read by the
// delivery layer (which used to keep its own copy of this exact walk) and by ApplyMasterAction.
func (s *MatchSession) CurrentTurnID() uuid.UUID {
	if s.activeRound == nil {
		return uuid.Nil
	}
	t := s.activeRound.CurrentTurn()
	if t == nil {
		return uuid.Nil
	}
	return t.GetID()
}

// ApplyMasterAction lands the master's edit ON the action and recomputes.
//
// There is no parallel version to merge on read: the edited action IS the action, which is
// the shape the code already had — RollCondition lives in RollContext, inside RollCheck,
// inside Action. The price of that model is that the original is destroyed in the live
// object, which is exactly why the override capture exists (see CaptureOverride).
//
// It NEVER rolls for a condition edit: Derive reads the two sets RollAttempts has held since
// the action arrived, and a late advantage changes WHICH set is read, never what fell.
//
// It never touches the economy either. Charged bars, recorded Speeds and the order already
// played stay as they are — bars_updated has been on the wire since before the edit, and
// redoing the price would reorder what has already been played.
func (s *MatchSession) ApplyMasterAction(
	ma *action.MasterAction, masterUUID uuid.UUID,
) (*service.TurnResolution, error) {
	t := s.activeRound.CurrentTurn()
	if t == nil || t.GetFinishedAt() != nil {
		return nil, ErrNoActiveTurn
	}
	target, err := s.actionOnTurn(t, ma.ActionID)
	if err != nil {
		return nil, err
	}
	if ma.TargetID != nil {
		target.TargetID = append([]uuid.UUID(nil), ma.TargetID...)
	}
	if ma.Skills != nil {
		s.applySkillEdit(target, ma.Skills)
	}
	for _, edit := range ma.Conditions {
		rc, err := resolveRollCheck(target, edit)
		if err != nil {
			return nil, err
		}
		cond := edit.Condition
		rc.Context.Condition = &cond
	}
	// Re-derive the speeds so a condition on speed or moveSpeed reads through. target.SystemBias,
	// never a literal 0: the disadvantage of an action→reaction conversion was decided once, at
	// attach, and deriveSpeeds stored it on the action for exactly this moment — passing 0 here
	// would silently erase a reaction's swap-disadvantage the instant the master edited any
	// OTHER condition on it (see Action.SystemBias).
	s.deriveSpeeds(target, target.SystemBias)
	ma.SetHappenedAt(time.Now())
	t.AddMasterAction(*ma)
	return s.ResolveTurn(t), nil
}

// actionOnTurn finds the turn's action or one of its reactions by ID. The zero UUID means the
// turn's own action, which is what a client editing "the action" sends.
func (s *MatchSession) actionOnTurn(t *turn.Turn, id uuid.UUID) (*action.Action, error) {
	a := t.ActionRef()
	if id == uuid.Nil || id == a.GetID() {
		return a, nil
	}
	if r := t.ReactionRef(id); r != nil {
		return r, nil
	}
	return nil, ErrActionNotOnTurn
}

// resolveRollCheck maps an edit's path to the RollCheck it names. A path that names nothing
// present is an error rather than a silent no-op: the master pressed a control describing a
// test they believe exists, and answering silently leaves them believing they changed it.
func resolveRollCheck(a *action.Action, e action.ConditionEdit) (*action.RollCheck, error) {
	if e.SkillName != "" {
		if e.Field != "" {
			return nil, ErrAmbiguousConditionEdit
		}
		for i := range a.Skills {
			if a.Skills[i].SkillName == e.SkillName {
				return &a.Skills[i].RollCheck, nil
			}
		}
		return nil, ErrConditionTargetMissing
	}
	switch e.Field {
	case action.FieldSpeed:
		return &a.Speed.RollCheck, nil
	case action.FieldFeint:
		if a.Feint == nil {
			return nil, ErrConditionTargetMissing
		}
		return a.Feint, nil
	case action.FieldHit:
		if a.Attack == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Attack.Hit, nil
	case action.FieldDamage:
		if a.Attack == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Attack.Damage, nil
	case action.FieldDodge:
		if a.Dodge == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Dodge.RollCheck, nil
	case action.FieldDefense:
		if a.Defense == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Defense.RollCheck, nil
	case action.FieldRepel:
		if a.Repel == nil {
			return nil, ErrConditionTargetMissing
		}
		return &a.Repel.RollCheck, nil
	case action.FieldMoveSpeed:
		if a.Move == nil || a.Move.Speed == nil {
			return nil, ErrConditionTargetMissing
		}
		return a.Move.Speed, nil
	default:
		return nil, ErrConditionTargetMissing
	}
}

// applySkillEdit replaces the action's skill list, and it is where the two asymmetric rules of
// combat-engine.md § A edição do mestre live.
//
// ADDING rolls new dice. That is not a re-roll and does not break "the master never re-rolls a
// player's die": it is the FIRST roll of a test that did not exist a moment ago.
//
// REMOVING keeps the dice. They go into removedSkillDice, keyed by action and skill name, and
// a later re-add reads them back. Without that, taking a skill out and putting it back would
// be a free re-roll — the master would only have to dislike a number to be given another one.
//
// The list changes and the list does not yet DECIDE anything: nobody reads a Skill's result
// (combat-engine.md § A corrente de testes). That is stated, not hidden.
func (s *MatchSession) applySkillEdit(a *action.Action, want []action.Skill) {
	if s.removedSkillDice == nil {
		s.removedSkillDice = map[uuid.UUID]map[string]action.RollAttempts{}
	}
	memory := s.removedSkillDice[a.GetID()]
	if memory == nil {
		memory = map[string]action.RollAttempts{}
		s.removedSkillDice[a.GetID()] = memory
	}

	held := make(map[string]action.RollAttempts, len(a.Skills))
	for _, prev := range a.Skills {
		held[prev.SkillName] = prev.Attempts
	}

	next := make([]action.Skill, 0, len(want))
	for _, sk := range want {
		switch {
		case !held[sk.SkillName].IsEmpty():
			sk.Attempts = held[sk.SkillName] // untouched: it was already there
		case !memory[sk.SkillName].IsEmpty():
			sk.Attempts = memory[sk.SkillName] // put back: same dice as before
			delete(memory, sk.SkillName)
		}
		if sk.RollCheck.SkillName == "" {
			sk.RollCheck.SkillName = sk.SkillName
		}
		next = append(next, sk)
	}
	// Whatever left the list parks its dice, so a re-add is not a re-roll.
	for name, attempts := range held {
		stillThere := false
		for _, sk := range next {
			if sk.SkillName == name {
				stillThere = true
				break
			}
		}
		if !stillThere && !attempts.IsEmpty() {
			memory[name] = attempts
		}
	}
	a.Skills = next
	// rollActionDice leaves alone every RollCheck whose dice already fell, so this rolls for
	// exactly the entries that are genuinely new.
	s.rollActionDice(a)
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
		Turn:     t,
		Sheets:   s.charSheets,
		Statuses: s.statuses,
		Targets:  s,
		Rules:    s.rules,
		Weapons:  s.weapons,
	})
}

// applyResolution writes a resolution's effective damage to the target sheets, once, and — in
// the same place, for the same reason — writes what each target's reaction earned into their
// own ledger.
//
// This is the moment the dry run stops being a dry run. Everything before it recalculated
// freely — every master edit, every colliding reaction — precisely because nothing had been
// applied. A payout written on every recomputation would multiply the same reserve or bonus
// each time the master edited a reaction; writing it here, once, is what the damage already
// relied on. Called only from the turn-closing path.
func (s *MatchSession) applyResolution(res *service.TurnResolution) []DamagedCharacter {
	if res == nil {
		return nil
	}
	var out []DamagedCharacter
	for _, cr := range res.CharacterResults {
		if dc, ok := s.applyDamage(cr); ok {
			out = append(out, dc)
		}
		// Payouts is read straight off the CharacterResult, by its own TargetID — Task 10's
		// first cut kept a second, resolution-level list of the same data (TurnResolution.
		// Payouts / CharacterPayout), which was exactly flatMap(CharacterResults, .Payouts):
		// two lists that could disagree the moment someone edited one and not the other.
		// Removed; this is the only place a payout is written, same as the damage above.
		status, ok := s.statuses[cr.TargetID]
		if !ok || status == nil {
			continue
		}
		for _, m := range cr.Payouts {
			status.Ledger.Add(m)
		}
	}
	return out
}

// applyDamage writes one CharacterResult's effective damage to its target sheet. ok is false
// when there is nothing to apply — no damage, no sheet, or no health bar — so the caller does
// not have to repeat that guard.
func (s *MatchSession) applyDamage(cr service.CharacterResult) (DamagedCharacter, bool) {
	if cr.EffectiveDamage <= 0 {
		return DamagedCharacter{}, false
	}
	sheet, ok := s.charSheets[cr.TargetID]
	if !ok || sheet == nil {
		return DamagedCharacter{}, false
	}
	bar, ok := sheet.GetAllStatusBar()[enum.Health]
	if !ok {
		return DamagedCharacter{}, false
	}
	newHP := bar.DecreaseAt(cr.EffectiveDamage)
	return DamagedCharacter{
		CharacterID: cr.TargetID,
		Sheet:       sheet,
		Damage:      cr.EffectiveDamage,
		NewHP:       newHP,
	}, true
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
	// Validate before mutating anything. A stale ReactToID is reachable whenever the master
	// opens the next turn while a target is still composing, and it must be refused with the
	// queue and the bars untouched — not after consumePendingFor has already removed the
	// player's queued action and chargeReactionBars has already debited it. This check
	// duplicates what roundOrch.AttachReaction checks below (it still owns the actual
	// attach), but here it runs first, before any of that.
	if act.GetID() != r.ReactToID {
		return nil, service.ErrReactionNotCompatible
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

// CloseOpenTurn ends the turn under the baton explicitly, and it is the SAME path the two
// baton operations take: it resolves one last time, applies what that resolution says, and
// advances the ledgers.
//
// It exists because the old CloseTurn() did none of those three. It called
// roundOrch.CloseTurnErr directly, so the turn ended, the damage evaporated and nothing
// reported an error — the worst shape a bug can have. There is now one way to close a turn,
// and this is it.
func (s *MatchSession) CloseOpenTurn() (*TurnTransition, error) {
	if !s.activeRound.HasOpenTurn() {
		return nil, ErrNoOpenTurn
	}
	return s.closeOpenTurn(), nil
}

// UnopenedReactions is the open turn's attached-but-not-opened reactions, for the confirmation
// gate in CloseTurnUC. Empty when there is no open turn — the caller's error for that case is
// ErrNoOpenTurn, raised by CloseOpenTurn, not by this reader.
func (s *MatchSession) UnopenedReactions() []action.Action {
	t := s.activeRound.CurrentTurn()
	if t == nil || t.GetFinishedAt() != nil {
		return nil
	}
	return t.UnopenedReactions()
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
// It is the only place a speed is produced — the master never re-rolls a player's die, so
// nothing downstream ever recomputes the DICE — but it is not only called once: ApplyMasterAction
// calls it again on every condition edit, so a speed/moveSpeed edit reads through. See
// Action.SystemBias for how a re-derive stays honest about the bias the action actually
// arrived under.
//
// systemBias is the engine-imposed advantage/disadvantage for THIS derivation — a reaction
// that swapped out a queued action passes -1, everyone else passes 0, and ApplyMasterAction
// passes back whatever this action was last derived under. It is a MODE of reading the dice
// that already fell, never an Amount: RollAttempts holds both attempts, and the bias only
// picks which one pickAttempt reads.
func (s *MatchSession) deriveSpeeds(a *action.Action, systemBias int) {
	if a == nil {
		return
	}
	sheet := s.charSheets[a.GetActorID()]
	if sheet == nil {
		return
	}
	// Stored so a LATER re-derive (a master edit on an unrelated condition) can apply the same
	// engine bias this action actually arrived under, instead of ApplyMasterAction having to
	// pass a literal 0 that would silently flatten a reaction's swap-disadvantage to neutral.
	a.SystemBias = systemBias
	calc := service.RollCalculator{}
	var ledger *match.ModifierLedger
	if status, ok := s.statuses[a.GetActorID()]; ok {
		ledger = &status.Ledger
	}

	// AgainstID is who this actionSpeed reading counts against. The repel bonus is banked
	// ScopeOnly(the attacker read) — see resolveRepel — so it only ever pays out when a
	// later roll is read against that exact opponent. An action names an unambiguous
	// opponent only when it has EXACTLY one target: nil (untargeted) has nobody to read
	// the bonus against, and more than one has no single answer to "against whom" — passing
	// an arbitrary member of TargetID there would let a bonus earned against one duelist
	// leak onto, or hide from, whichever target happened to land first in the slice.
	var againstID *uuid.UUID
	if len(a.TargetID) == 1 {
		againstID = &a.TargetID[0]
	}

	// actionSpeed: always Legerity. Passive in Free, rolled in Race.
	//
	// The duel reserve (repel/parry) lives on this dimension and is read here — it is what
	// makes the repel ladder produce a duel, two characters facing each other speed up
	// against each other, without anyone programming duels. The closed dodge's reserve is a
	// different dimension (DimDodge) and is read elsewhere, inside ResolveReaction, not here.
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
		AgainstID:  againstID,
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
