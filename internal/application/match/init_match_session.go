package match

import (
	"context"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IInitMatchSession interface {
	Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error)
}

type InitMatchSessionUC struct {
	matchRepo   IRepository
	sheetLoader ICharSheetLoader
}

func NewInitMatchSessionUC(matchRepo IRepository, sheetLoader ICharSheetLoader) *InitMatchSessionUC {
	return &InitMatchSessionUC{matchRepo: matchRepo, sheetLoader: sheetLoader}
}

func (uc *InitMatchSessionUC) Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error) {
	participants, err := uc.matchRepo.ListParticipantsByMatchUUID(ctx, matchUUID)
	if err != nil {
		return nil, err
	}

	charSheets := make(map[uuid.UUID]*csSheet.CharacterSheet, len(participants))
	for _, p := range participants {
		if p.Sheet.PlayerUUID == nil {
			continue
		}
		sheet, found, err := uc.sheetLoader.GetCharacterSheetByUUID(ctx, p.Sheet.UUID.String())
		if err != nil {
			return nil, err
		}
		if found {
			charSheets[*p.Sheet.PlayerUUID] = sheet
		}
	}

	return matchsession.NewMatchSession(matchUUID, charSheets, participants), nil
}

var _ IInitMatchSession = (*InitMatchSessionUC)(nil)
