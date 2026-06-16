package matchsession

import (
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
	fogStates    map[uuid.UUID]*fog.PlayerFogState
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
		fogStates:    make(map[uuid.UUID]*fog.PlayerFogState),
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
		fogStates:      make(map[uuid.UUID]*fog.PlayerFogState),
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

// SyncFogStates seeds fog state from persisted records (nil seeds empty states).
// Resets the visibility cache.
func (s *MatchSession) SyncFogStates(states []fog.PlayerFogState, mode fog.FogMode) {
	s.fogMode = mode
	s.fogStates = make(map[uuid.UUID]*fog.PlayerFogState, len(states))
	for i := range states {
		st := states[i]
		s.fogStates[st.PlayerID] = &st
	}
	s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
}

// fogStateFor returns the existing fog state for playerID, or lazily creates one.
func (s *MatchSession) fogStateFor(playerID uuid.UUID) *fog.PlayerFogState {
	if s.fogStates == nil {
		s.fogStates = make(map[uuid.UUID]*fog.PlayerFogState)
	}
	st, ok := s.fogStates[playerID]
	if !ok {
		st = fog.NewPlayerFogState(playerID, s.matchUUID, uuid.Nil, string(s.grid.Kind))
		s.fogStates[playerID] = st
	}
	return st
}

// RecomputeVisibility recomputes a player's LOS, caches polygons, unions explored cells
// (explored mode only), and returns the polygons and the newly explored delta.
func (s *MatchSession) RecomputeVisibility(playerID uuid.UUID) ([]service.VisibilityPolygon, []fog.CellCoord, error) {
	losWalls := service.ToLOSWalls(s.GetWalls())
	var positions []service.Point2D
	if s.pieceSource != nil {
		positions = s.pieceSource.PlayerPiecePositions(playerID)
	}
	polys := make([]service.VisibilityPolygon, 0, len(positions))
	var delta []fog.CellCoord
	for _, pos := range positions {
		poly := service.ComputeVisibilityPolygon(pos, losWalls)
		polys = append(polys, poly)
		if s.fogMode == fog.FogModeExplored {
			cells := service.CellsInPolygon(poly, s.grid)
			delta = append(delta, s.fogStateFor(playerID).AddExplored(cells)...)
		}
	}
	if s.visCache == nil {
		s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
	}
	s.visCache[playerID] = polys
	return polys, delta, nil
}

// RecomputeAllVisibility recomputes LOS for all participants.
func (s *MatchSession) RecomputeAllVisibility() error {
	for pid := range s.participants {
		if _, _, err := s.RecomputeVisibility(pid); err != nil {
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

func (s *MatchSession) GetFogState(playerID uuid.UUID) (*fog.PlayerFogState, bool) {
	st, ok := s.fogStates[playerID]
	return st, ok
}

func (s *MatchSession) GetAllFogStates() []fog.PlayerFogState {
	out := make([]fog.PlayerFogState, 0, len(s.fogStates))
	for _, st := range s.fogStates {
		out = append(out, *st)
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
