package campaign

import (
	"context"
	"time"

	campaignEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type IListPublicUpcomingCampaigns interface {
	ListPublicUpcomingCampaigns(ctx context.Context, userUUID uuid.UUID) ([]*campaignEntity.PublicSummary, error)
}

type ListPublicUpcomingCampaignsUC struct {
	repo IRepository
}

func NewListPublicUpcomingCampaignsUC(repo IRepository) *ListPublicUpcomingCampaignsUC {
	return &ListPublicUpcomingCampaignsUC{repo: repo}
}

func (uc *ListPublicUpcomingCampaignsUC) ListPublicUpcomingCampaigns(
	ctx context.Context, userUUID uuid.UUID,
) ([]*campaignEntity.PublicSummary, error) {
	now := time.Now()
	return uc.repo.ListPublicUpcomingCampaigns(ctx, now, userUUID)
}
