package matchsession

import (
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
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
