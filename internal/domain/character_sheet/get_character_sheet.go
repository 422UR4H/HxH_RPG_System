// internal/domain/character_sheet/get_character_sheet.go
package charactersheet

import (
	"context"
	"fmt"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type IGetCharacterSheet interface {
	GetCharacterSheet(
		ctx context.Context, charSheetId uuid.UUID, playerId uuid.UUID,
	) (*domainSheet.CharacterSheet, error)
}

type GetCharacterSheetUC struct {
	characterSheets *sync.Map
	factory         *domainSheet.CharacterSheetFactory
	repo            IRepository
	campaignRepo    domainCampaign.IRepository
}

func NewGetCharacterSheetUC(
	charSheets *sync.Map,
	factory *domainSheet.CharacterSheetFactory,
	repo IRepository,
	campaignRepo domainCampaign.IRepository,
) *GetCharacterSheetUC {
	return &GetCharacterSheetUC{
		characterSheets: charSheets,
		factory:         factory,
		repo:            repo,
		campaignRepo:    campaignRepo,
	}
}

func (uc *GetCharacterSheetUC) GetCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID,
) (*domainSheet.CharacterSheet, error) {

	// TODO: fix, move after auth validations or remove
	// if charSheet, ok := uc.characterSheets.Load(sheetUUID); ok {
	// 	return charSheet.(*sheet.CharacterSheet), nil
	// }

	charSheet, err := uc.repo.GetCharacterSheetByUUID(ctx, sheetUUID.String())
	if err != nil {
		return nil, err
	}
	masterUUID := charSheet.GetMasterUUID()
	playerUUID := charSheet.GetPlayerUUID()

	if masterUUID != nil && *masterUUID == userUUID {
		return uc.checkAndNormalize(ctx, sheetUUID.String(), charSheet)
	}
	if playerUUID != nil && *playerUUID == userUUID {
		return uc.checkAndNormalize(ctx, sheetUUID.String(), charSheet)
	}

	campaignUUID := charSheet.GetCampaignUUID()
	if campaignUUID == nil {
		return nil, auth.ErrInsufficientPermissions
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, *campaignUUID)
	if err == pgCampaign.ErrCampaignNotFound {
		return nil, domainCampaign.ErrCampaignNotFound
	}
	if err != nil {
		return nil, err
	}
	if campaignMasterUUID == userUUID {
		return uc.checkAndNormalize(ctx, sheetUUID.String(), charSheet)
	}
	return nil, auth.ErrInsufficientPermissions
}

func (uc *GetCharacterSheetUC) checkAndNormalize(
	ctx context.Context,
	sheetUUID string,
	charSheet *domainSheet.CharacterSheet,
) (*domainSheet.CharacterSheet, error) {
	// Status normalization is handled inside the gateway's wrap function.
	// If correction was applied, persist asynchronously.
	// TODO: expose wasCorrected from gateway or move normalization trigger here
	return charSheet, nil
}

func (uc *GetCharacterSheetUC) persistNormalizedStatus( //nolint:unused
	ctx context.Context,
	sheetUUID string,
	charSheet *domainSheet.CharacterSheet,
) {
	allBars := charSheet.GetAllStatusBar()
	if err := uc.repo.UpdateStatusBars(ctx, sheetUUID,
		allBars[enum.Health],
		allBars[enum.Stamina],
		allBars[enum.Aura],
	); err != nil {
		fmt.Printf("TODO(logger): failed to persist normalized status for sheet %s: %v\n", sheetUUID, err)
	}
}
