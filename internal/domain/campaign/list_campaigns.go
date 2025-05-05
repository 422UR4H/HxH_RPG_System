package campaign

import (
	"context"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type IListCampaigns interface {
	ListCampaigns(userUUID uuid.UUID) ([]*campaignEntity.Summary, error)
}

type ListCampaignsUC struct {
	repo IRepository
}

func NewListCampaignsUC(repo IRepository) *ListCampaignsUC {
	return &ListCampaignsUC{
		repo: repo,
	}
}

func (uc *ListCampaignsUC) ListCampaigns(userUUID uuid.UUID) ([]*campaignEntity.Summary, error) {
	campaigns, err := uc.repo.ListCampaignsByUserUUID(context.Background(), userUUID)
	if err != nil {
		return nil, err
	}
	return campaigns, nil
}
