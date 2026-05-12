package enrollment

import (
	"context"

	"github.com/google/uuid"
)

type IListPlayerEnrollmentsForCampaign interface {
	ListPlayerEnrollmentsForCampaign(
		ctx context.Context,
		playerUUID uuid.UUID,
		campaignUUID uuid.UUID,
	) (map[uuid.UUID]string, error)
}

type ListPlayerEnrollmentsForCampaignUC struct {
	repo IRepository
}

func NewListPlayerEnrollmentsForCampaignUC(repo IRepository) *ListPlayerEnrollmentsForCampaignUC {
	return &ListPlayerEnrollmentsForCampaignUC{repo: repo}
}

func (uc *ListPlayerEnrollmentsForCampaignUC) ListPlayerEnrollmentsForCampaign(
	ctx context.Context,
	playerUUID uuid.UUID,
	campaignUUID uuid.UUID,
) (map[uuid.UUID]string, error) {
	return uc.repo.ListPlayerEnrollmentStatusesForCampaign(ctx, playerUUID, campaignUUID)
}
