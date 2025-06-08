package submission

import (
	"context"
	"time"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	sheetPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
)

type ISubmitCharacterSheet interface {
	Submit(
		ctx context.Context,
		userUUID uuid.UUID,
		sheetUUID uuid.UUID,
		campaignUUID uuid.UUID,
	) error
}

type SubmitCharacterSheetUC struct {
	repo         IRepository
	sheetRepo    charactersheet.IRepository
	campaignRepo campaignDomain.IRepository
}

func NewSubmitCharacterSheetUC(
	repo IRepository,
	sheetRepo charactersheet.IRepository,
	campaignRepo campaignDomain.IRepository,
) *SubmitCharacterSheetUC {
	return &SubmitCharacterSheetUC{
		repo:         repo,
		sheetRepo:    sheetRepo,
		campaignRepo: campaignRepo,
	}
}

func (uc *SubmitCharacterSheetUC) Submit(
	ctx context.Context,
	userUUID uuid.UUID,
	sheetUUID uuid.UUID,
	campaignUUID uuid.UUID,
) error {
	playerUUID, err := uc.sheetRepo.GetCharacterSheetPlayerUUID(ctx, sheetUUID)
	if err == sheetPg.ErrCharacterSheetNotFound {
		return charactersheet.ErrCharacterSheetNotFound
	}
	if err != nil {
		return err
	}
	if playerUUID != userUUID {
		return charactersheet.ErrNotCharacterSheetOwner
	}

	exists, err := uc.repo.ExistsSubmittedCharacterSheet(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if exists {
		return ErrCharacterAlreadySubmitted
	}

	masterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, campaignUUID)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if playerUUID == masterUUID {
		return ErrMasterCannotSubmitOwnSheet
	}
	return uc.repo.SubmitCharacterSheet(ctx, sheetUUID, campaignUUID, time.Now())
}
