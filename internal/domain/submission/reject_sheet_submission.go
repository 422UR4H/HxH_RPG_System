package submission

import (
	"context"

	campaignDomain "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	submissionPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/submission"
	"github.com/google/uuid"
)

type IRejectCharacterSheetSubmission interface {
	Reject(ctx context.Context, sheetUUID uuid.UUID, masterUUID uuid.UUID) error
}

type RejectCharacterSheetSubmissionUC struct {
	repo         IRepository
	campaignRepo campaignDomain.IRepository
}

func NewRejectCharacterSheetSubmissionUC(
	repo IRepository,
	campaignRepo campaignDomain.IRepository,
) *RejectCharacterSheetSubmissionUC {
	return &RejectCharacterSheetSubmissionUC{
		repo:         repo,
		campaignRepo: campaignRepo,
	}
}

func (uc *RejectCharacterSheetSubmissionUC) Reject(
	ctx context.Context,
	sheetUUID uuid.UUID,
	masterUUID uuid.UUID,
) error {
	// TODO: optimize that 2 calls to db to only 1
	campaignUUID, err := uc.repo.GetSubmissionCampaignUUIDBySheetUUID(ctx, sheetUUID)
	if err == submissionPg.ErrSubmissionNotFound {
		return ErrSubmissionNotFound
	}
	if err != nil {
		return err
	}

	campaignMasterUUID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, campaignUUID)
	if err == campaignPg.ErrCampaignNotFound {
		return campaignDomain.ErrCampaignNotFound
	}
	if err != nil {
		return err
	}
	if campaignMasterUUID != masterUUID {
		return ErrNotCampaignMaster
	}

	err = uc.repo.RejectCharacterSheetSubmission(ctx, sheetUUID)
	if err != nil {
		return err
	}
	return nil
}
