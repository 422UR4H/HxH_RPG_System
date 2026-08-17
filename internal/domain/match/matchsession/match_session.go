package matchsession

import (
	"math"
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
}

// DamagedCharacter is one applied HP reduction. The caller persists it — the session holds
// the live sheet, the gateway holds the row.
type DamagedCharacter struct {
	CharacterID uuid.UUID
	Sheet       *csSheet.CharacterSheet
	Damage      int
	NewHP       int
}

func (s *MatchSession) OpenNextAction() (*TurnTransition, error) {
	tr := s.closeOpenTurn()
	opened, err := s.roundOrch.NextAction(s.activeRound, &s.activeQueue)
	if err != nil {
		return tr, err
	}
	tr.Opened = opened
	tr.OpenedResolution = s.ResolveTurn(opened)
	return tr, nil
}

func (s *MatchSession) PullAction(id uuid.UUID) (*TurnTransition, error) {
	tr := s.closeOpenTurn()
	opened, err := s.roundOrch.PullAction(s.activeRound, &s.activeQueue, id)
	if err != nil {
		return tr, err
	}
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
	return tr
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

func (s *MatchSession) AttachReaction(r *action.Action) (*service.TurnResolution, error) {
	s.rollActionDice(r)
	if err := s.roundOrch.AttachReaction(s.activeRound, r); err != nil {
		return nil, err
	}
	return s.ResolveTurn(s.activeRound.CurrentTurn()), nil
}

func (s *MatchSession) CloseTurn() (*turn.Turn, error) {
	return s.roundOrch.CloseTurnErr(s.activeRound, time.Now())
}

func (s *MatchSession) CloseRound() (*round.Round, error) {
	if s.activeRound.HasOpenTurn() {
		return nil, ErrRoundHasOpenTurn
	}
	mode := s.activeRound.GetMode()
	closed := s.roundOrch.CloseRound(s.activeRound, time.Now())
	s.activeRound = round.NewRound(mode)
	s.roundPersisted = false
	return closed, nil
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
	s.activeQueue.Insert(a)
	return nil
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

	test(&a.Speed.RollCheck)
	test(a.Feint)
	for i := range a.Skills {
		test(&a.Skills[i].RollCheck)
	}
	if a.Move != nil {
		test(a.Move.Speed)
		test(a.Move.Charge)
	}
	if a.Defense != nil {
		test(&a.Defense.RollCheck)
	}
	if a.Dodge != nil {
		test(&a.Dodge.RollCheck)
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
