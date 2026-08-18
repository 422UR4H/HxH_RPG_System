package match

import (
	"context"
	"errors"
	"log"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
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
		// No PlayerUUID guard: an NPC has PlayerUUID nil and MasterUUID set, and the
		// master plays it. The sheet loader is keyed by sheet UUID either way.
		//
		// ⚠️ The loader's second return is wasCorrected — whether hydrating the sheet had to
		// repair it — NOT whether it was found. This used to read `if found { ... }`, which
		// silently dropped every intact sheet and kept only the repaired ones, leaving the
		// session with an empty charSheets map. Nothing read that map until the character
		// collision arrived, which is why it went unnoticed. A missing sheet comes back as
		// an error, not as a false.
		sheet, _, err := uc.sheetLoader.GetCharacterSheetByUUID(ctx, p.Sheet.UUID.String())
		if err != nil {
			if errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
				// One roster entry without a sheet must not take the whole match down.
				log.Printf("init match session %s: no sheet for participant %s, skipping", matchUUID, p.Sheet.UUID)
				continue
			}
			return nil, err
		}
		charSheets[p.Sheet.UUID] = sheet
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
