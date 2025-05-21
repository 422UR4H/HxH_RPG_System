package campaign

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	campaignPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

type IGetCampaign interface {
	GetCampaign(
		ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID,
	) (*campaign.Campaign, error)
}

type GetCampaignUC struct {
	repo IRepository
}

func NewGetCampaignUC(repo IRepository) *GetCampaignUC {
	return &GetCampaignUC{
		repo: repo,
	}
}

func (uc *GetCampaignUC) GetCampaign(
	ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID,
) (*campaign.Campaign, error) {
	campaign, err := uc.repo.GetCampaign(ctx, uuid)
	if err != nil {
		if errors.Is(err, campaignPg.ErrCampaignNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}

	if campaign.UserUUID != userUUID {
		return nil, auth.ErrInsufficientPermissions
	}
	return campaign, nil
}
