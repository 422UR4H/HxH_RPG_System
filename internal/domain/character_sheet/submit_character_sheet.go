package charactersheet

import (
	"context"
	"time"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
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
	sheetRepo    IRepository
	campaignRepo campaignDomain.IRepository
}

func NewSubmitCharacterSheetUC(
	sheetRepo IRepository, campaignRepo campaignDomain.IRepository,
) *SubmitCharacterSheetUC {
	return &SubmitCharacterSheetUC{
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
	if err == sheet.ErrCharacterSheetNotFound {
		return ErrCharacterSheetNotFound
	}
	if err != nil {
		return err
	}
	if playerUUID != userUUID {
		return ErrNotCharacterSheetOwner
	}

	exists, err := uc.sheetRepo.ExistsSubmittedCharacterSheet(ctx, sheetUUID)
	if err != nil {
		return err
	}
	if exists {
		return ErrCharacterAlreadySubmitted
	}

	masterUUID, err := uc.campaignRepo.GetCampaignUserUUID(ctx, campaignUUID)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}

	if playerUUID == masterUUID {
		return ErrMasterCannotSubmitOwnSheet
	}
	err = uc.sheetRepo.SubmitCharacterSheet(
		ctx, sheetUUID, campaignUUID, time.Now(),
	)
	if err != nil {
		return err
	}
	return nil
}
