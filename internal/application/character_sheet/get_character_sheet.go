// internal/domain/character_sheet/get_character_sheet.go
package charactersheet

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
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

// ISubmissionLookup lets the GET use case check pending submissions for authorization.
type ISubmissionLookup interface {
	GetSubmissionCampaignUUIDBySheetUUID(ctx context.Context, sheetUUID uuid.UUID) (uuid.UUID, error)
}

// SubmissionInfo is returned by ISubmissionFetcher for the optional ?include=submission.
type SubmissionInfo struct {
	CampaignUUID uuid.UUID
	CreatedAt    time.Time
}

// ISubmissionFetcher is satisfied by the submission gateway; used by the HTTP handler only.
type ISubmissionFetcher interface {
	GetSubmissionInfoBySheetUUID(ctx context.Context, sheetUUID uuid.UUID) (*SubmissionInfo, error)
}

type GetCharacterSheetUC struct {
	characterSheets  *sync.Map
	factory          *domainSheet.CharacterSheetFactory
	repo             IRepository
	campaignRepo     domainCampaign.IRepository
	submissionLookup ISubmissionLookup
}

func NewGetCharacterSheetUC(
	charSheets *sync.Map,
	factory *domainSheet.CharacterSheetFactory,
	repo IRepository,
	campaignRepo domainCampaign.IRepository,
	submissionLookup ISubmissionLookup,
) *GetCharacterSheetUC {
	return &GetCharacterSheetUC{
		characterSheets:  charSheets,
		factory:          factory,
		repo:             repo,
		campaignRepo:     campaignRepo,
		submissionLookup: submissionLookup,
	}
}

func (uc *GetCharacterSheetUC) GetCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, userUUID uuid.UUID,
) (*domainSheet.CharacterSheet, error) {

	// TODO: fix, move after auth validations or remove
	// if charSheet, ok := uc.characterSheets.Load(sheetUUID); ok {
	// 	return charSheet.(*sheet.CharacterSheet), nil
	// }

	sheetUUIDStr := sheetUUID.String()
	charSheet, wasCorrected, err := uc.repo.GetCharacterSheetByUUID(ctx, sheetUUIDStr)
	if err != nil {
		return nil, err
	}
	masterUUID := charSheet.GetMasterUUID()
	playerUUID := charSheet.GetPlayerUUID()

	if masterUUID != nil && *masterUUID == userUUID {
		return uc.checkAndNormalize(sheetUUIDStr, charSheet, wasCorrected)
	}
	if playerUUID != nil && *playerUUID == userUUID {
		return uc.checkAndNormalize(sheetUUIDStr, charSheet, wasCorrected)
	}

	campaignUUID := charSheet.GetCampaignUUID()
	if campaignUUID != nil {
		campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, *campaignUUID)
		if err == pgCampaign.ErrCampaignNotFound {
			return nil, domainCampaign.ErrCampaignNotFound
		}
		if err != nil {
			return nil, err
		}
		if campaignMasterUUID == userUUID {
			return uc.checkAndNormalize(sheetUUIDStr, charSheet, wasCorrected)
		}
		return nil, auth.ErrInsufficientPermissions
	}

	// campaignUUID is nil: sheet not yet accepted. Check pending submission.
	subCampUUID, err := uc.submissionLookup.GetSubmissionCampaignUUIDBySheetUUID(ctx, sheetUUID)
	if err != nil {
		return nil, auth.ErrInsufficientPermissions
	}
	subCampMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, subCampUUID)
	if err == pgCampaign.ErrCampaignNotFound {
		return nil, domainCampaign.ErrCampaignNotFound
	}
	if err != nil {
		return nil, err
	}
	if subCampMasterUUID == userUUID {
		return uc.checkAndNormalize(sheetUUIDStr, charSheet, wasCorrected)
	}
	return nil, auth.ErrInsufficientPermissions
}

func (uc *GetCharacterSheetUC) checkAndNormalize(
	sheetUUID string,
	charSheet *domainSheet.CharacterSheet,
	wasCorrected bool,
) (*domainSheet.CharacterSheet, error) {
	expPoints := charSheet.GetExpPoints()
	if !wasCorrected && expPoints == 0 {
		return charSheet, nil
	}
	// Both status-bar normalization and char_exp refresh are best-effort; run async.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("TODO(logger): panic in background normalization for sheet %s: %v\n", sheetUUID, r)
			}
		}()
		ctx := context.Background()
		if wasCorrected {
			uc.persistNormalizedStatus(ctx, sheetUUID, charSheet)
		}
		if expPoints > 0 {
			if err := uc.repo.UpdateCharExp(ctx, sheetUUID, expPoints); err != nil {
				fmt.Printf("TODO(logger): failed to refresh char_exp for sheet %s: %v\n", sheetUUID, err)
			}
		}
	}()
	return charSheet, nil
}

func (uc *GetCharacterSheetUC) persistNormalizedStatus(
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
