package matchsession

import (
	"time"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

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
}

func NewMatchSession(
	matchUUID uuid.UUID,
	charSheets map[uuid.UUID]*csSheet.CharacterSheet,
	participants []*match.Participant,
) *MatchSession {
	pMap := make(map[uuid.UUID]*match.Participant, len(participants))
	for _, p := range participants {
		if p.Sheet.PlayerUUID != nil {
			pMap[*p.Sheet.PlayerUUID] = p
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
	for _, p := range participants {
		if p.Sheet.PlayerUUID != nil {
			pMap[*p.Sheet.PlayerUUID] = p
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
	return s.turnResolver.Resolve(t, s.charSheets), nil
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
