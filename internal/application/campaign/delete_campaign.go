package campaign

import (
	"context"
	"errors"

	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type IDeleteCampaign interface {
	Delete(ctx context.Context, input *DeleteCampaignInput) error
}

type DeleteCampaignInput struct {
	CampaignUUID uuid.UUID
	MasterUUID   uuid.UUID
}

type DeleteCampaignUC struct {
	repo IRepository
}

func NewDeleteCampaignUC(repo IRepository) *DeleteCampaignUC {
	return &DeleteCampaignUC{repo: repo}
}

func (uc *DeleteCampaignUC) Delete(ctx context.Context, input *DeleteCampaignInput) error {
	masterUUID, err := uc.repo.GetCampaignMasterUUID(ctx, input.CampaignUUID)
	if err != nil {
		if errors.Is(err, campaignPg.ErrCampaignNotFound) {
			return ErrCampaignNotFound
		}
		return err
	}

	if masterUUID != input.MasterUUID {
		return ErrNotCampaignOwner
	}

	err = uc.repo.DeleteCampaign(ctx, input.CampaignUUID)
	if err != nil {
		if errors.Is(err, campaignPg.ErrCampaignNotFound) {
			// Race condition: a match started between GetCampaignMasterUUID and DeleteCampaign
			return ErrCampaignHasStartedMatch
		}
		return err
	}

	return nil
}
