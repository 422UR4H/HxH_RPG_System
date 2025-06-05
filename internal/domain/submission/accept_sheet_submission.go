package submission

import (
	"context"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/submit"
	"github.com/google/uuid"
)

type IAcceptCharacterSheetSubmission interface {
	Accept(ctx context.Context, sheetUUID uuid.UUID, masterUUID uuid.UUID) error
}

type AcceptCharacterSheetSubmissionUC struct {
	repo         IRepository
	campaignRepo campaignDomain.IRepository
}

func NewAcceptCharacterSheetSubmissionUC(
	repo IRepository,
	campaignRepo campaignDomain.IRepository,
) *AcceptCharacterSheetSubmissionUC {
	return &AcceptCharacterSheetSubmissionUC{
		repo:         repo,
		campaignRepo: campaignRepo,
	}
}

func (uc *AcceptCharacterSheetSubmissionUC) Accept(
	ctx context.Context,
	sheetUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	campaignUUID, err := uc.repo.GetSubmissionCampaignUUIDBySheetUUID(
		ctx, sheetUUID,
	)
	if err == submit.ErrSubmissionNotFound {
		return ErrSubmissionNotFound
	}
	if err != nil {
		return err
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignUserUUID(
		ctx, campaignUUID,
	)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if campaignMasterUUID != masterUUID {
		return ErrNotCampaignMaster
	}

	err = uc.repo.AcceptCharacterSheetSubmission(
		ctx,
		sheetUUID,
		campaignUUID,
	)
	if err != nil {
		return err
	}
	return nil
}
