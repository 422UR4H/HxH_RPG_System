package matchsession

import (
	"time"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

type MatchSession struct {
	matchUUID    uuid.UUID
	activeRound  *round.Round
	activeQueue  action.PriorityQueue
	charSheets   map[uuid.UUID]*csSheet.CharacterSheet // keyed by playerUUID
	participants map[uuid.UUID]*match.Participant       // keyed by playerUUID
	roundOrch    service.RoundOrchestrator
	combatRes    service.CombatResolver
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
		activeRound:  round.NewRound(enum.Free),
		activeQueue:  action.NewActionPriorityQueue(nil),
		charSheets:   charSheets,
		participants: pMap,
		roundOrch:    service.RoundOrchestrator{},
		combatRes:    service.CombatResolver{},
	}
}

func (s *MatchSession) GetActiveRound() *round.Round { return s.activeRound }

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

// EnqueueAction adds a player's action to the priority queue.
// playerUUID must be a known participant and must match a.GetActorID().
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
