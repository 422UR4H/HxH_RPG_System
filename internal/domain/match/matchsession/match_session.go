package matchsession

import (
	"math"
	"time"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

// PiecePositionSource yields the current world positions of a player's pieces.
// Implemented by room.go (which owns the live piece payloads), keeping the session
// decoupled from delivery types.
type PiecePositionSource interface {
	PlayerPiecePositions(playerID uuid.UUID) []service.Point2D
}

type MatchSession struct {
	matchUUID      uuid.UUID
	activeScene    *scene.Scene
	activeRound    *round.Round
	activeQueue    action.PriorityQueue
	charSheets     map[uuid.UUID]*csSheet.CharacterSheet // keyed by playerUUID
	participants   map[uuid.UUID]*match.Participant       // keyed by playerUUID
	roundOrch      service.RoundOrchestrator
	turnResolver   service.TurnResolver
	scenePersisted bool
	roundPersisted bool
	walls        map[string]mapentity.WallSegment         // keyed by wall ID; nil until SyncMapState
	grid         mapentity.GridShape                      // full grid shape; CellSize 0 until SyncMapState
	fogMode      fog.FogMode
	memories     map[uuid.UUID]*fog.PlayerMemory
	visCache     map[uuid.UUID][]service.VisibilityPolygon
	charToPlayer map[string]uuid.UUID
	pieceSource  PiecePositionSource
}

func NewMatchSession(
	matchUUID uuid.UUID,
	charSheets map[uuid.UUID]*csSheet.CharacterSheet,
	participants []*match.Participant,
) *MatchSession {
	pMap := make(map[uuid.UUID]*match.Participant, len(participants))
	charToPlayer := make(map[string]uuid.UUID)
	for _, p := range participants {
		if p.Sheet.PlayerUUID != nil {
			pMap[*p.Sheet.PlayerUUID] = p
			// Map sheet UUID (used as CharacterID on board pieces) → player UUID.
			if p.Sheet.UUID != uuid.Nil {
				charToPlayer[p.Sheet.UUID.String()] = *p.Sheet.PlayerUUID
			}
		}
	}
	return &MatchSession{
		matchUUID:    matchUUID,
		activeScene:  scene.NewScene(enum.Roleplay, ""),
		activeRound:  round.NewRound(enum.Free),
		activeQueue:  action.NewActionPriorityQueue(nil),
		charSheets:   charSheets,
		participants: pMap,
		roundOrch:    service.RoundOrchestrator{},
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
	pMap := make(map[uuid.UUID]*match.Participant, len(participants))
	charToPlayer := make(map[string]uuid.UUID)
	for _, p := range participants {
		if p.Sheet.PlayerUUID != nil {
			pMap[*p.Sheet.PlayerUUID] = p
			// Map sheet UUID (used as CharacterID on board pieces) → player UUID.
			if p.Sheet.UUID != uuid.Nil {
				charToPlayer[p.Sheet.UUID.String()] = *p.Sheet.PlayerUUID
			}
		}
	}
	return &MatchSession{
		matchUUID:      matchUUID,
		activeScene:    activeScene,
		activeRound:    activeRound,
		activeQueue:    action.NewActionPriorityQueue(nil),
		charSheets:     charSheets,
		participants:   pMap,
		roundOrch:      service.RoundOrchestrator{},
		turnResolver:   service.TurnResolver{},
		scenePersisted: true,
		roundPersisted: true,
		charToPlayer:   charToPlayer,
		memories:       make(map[uuid.UUID]*fog.PlayerMemory),
		visCache:       make(map[uuid.UUID][]service.VisibilityPolygon),
	}
}

func (s *MatchSession) GetMatchUUID() uuid.UUID      { return s.matchUUID }
func (s *MatchSession) GetActiveRound() *round.Round { return s.activeRound }
func (s *MatchSession) GetActiveScene() *scene.Scene { return s.activeScene }
func (s *MatchSession) IsRoundPersisted() bool       { return s.roundPersisted }
func (s *MatchSession) IsScenePersisted() bool       { return s.scenePersisted }

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

func (s *MatchSession) GetCharSheet(playerUUID uuid.UUID) (*csSheet.CharacterSheet, error) {
	sheet, ok := s.charSheets[playerUUID]
	if !ok {
		return nil, ErrCharSheetNotFound
	}
	return sheet, nil
}

func (s *MatchSession) OpenNextAction() (closed *turn.Turn, opened *turn.Turn, err error) {
	if s.activeRound.HasOpenTurn() {
		closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
	}
	opened, err = s.roundOrch.NextAction(s.activeRound, &s.activeQueue)
	return
}

func (s *MatchSession) PullAction(id uuid.UUID) (closed *turn.Turn, opened *turn.Turn, err error) {
	if s.activeRound.HasOpenTurn() {
		closed = s.roundOrch.CloseTurn(s.activeRound, time.Now())
	}
	opened, err = s.roundOrch.PullAction(s.activeRound, &s.activeQueue, id)
	return
}

func (s *MatchSession) AttachReaction(r *action.Action) (*service.TurnResolution, error) {
	if err := s.roundOrch.AttachReaction(s.activeRound, r); err != nil {
		return nil, err
	}
	t := s.activeRound.CurrentTurn()
	return s.turnResolver.Resolve(t, s.charSheets, s), nil
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

func (s *MatchSession) EnqueueAction(playerUUID uuid.UUID, a *action.Action) error {
	if _, ok := s.participants[playerUUID]; !ok {
		return ErrParticipantNotFound
	}
	if a.GetActorID() != playerUUID {
		return ErrActionActorMismatch
	}
	s.activeQueue.Insert(a)
	return nil
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
// Participants are checked first so character UUIDs are never mis-routed as walls.
func (s *MatchSession) CategorizeTarget(id uuid.UUID) service.TargetKind {
	if _, ok := s.participants[id]; ok {
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

func (s *MatchSession) GetFogMode() fog.FogMode                { return s.fogMode }
func (s *MatchSession) GetCharToPlayer() map[string]uuid.UUID  { return s.charToPlayer }

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
