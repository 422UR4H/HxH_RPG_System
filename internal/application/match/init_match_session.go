package match

import (
	"context"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/google/uuid"
)

type IInitMatchSession interface {
	Init(ctx context.Context, matchUUID uuid.UUID) (*matchsession.MatchSession, error)
}

type InitMatchSessionUC struct {
	matchRepo   IRepository
	sheetLoader ICharSheetLoader
	roundRepo   IRoundRepository
}

func NewInitMatchSessionUC(matchRepo IRepository, sheetLoader ICharSheetLoader, roundRepo IRoundRepository) *InitMatchSessionUC {
	return &InitMatchSessionUC{matchRepo: matchRepo, sheetLoader: sheetLoader, roundRepo: roundRepo}
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

	data, err := uc.roundRepo.FindActiveSession(ctx, matchUUID)
	if err != nil {
		return nil, err
	}
	if data != nil {
		sc := sceneentity.ReconstructScene(data.SceneID, enum.SceneCategory(data.Category), data.BriefInitDesc, data.SceneCreatedAt)
		r := roundentity.ReconstructRound(data.RoundID, enum.RoundMode(data.Mode), data.RoundCreatedAt)
		return matchsession.NewMatchSessionWithState(matchUUID, charSheets, participants, sc, r), nil
	}
	return matchsession.NewMatchSession(matchUUID, charSheets, participants), nil
}

var _ IInitMatchSession = (*InitMatchSessionUC)(nil)
