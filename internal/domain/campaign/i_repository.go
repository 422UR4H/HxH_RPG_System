package campaign

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateCampaign(ctx context.Context, campaign *campaign.Campaign) error
	GetCampaign(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	GetCampaignUserUUID(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCampaignStoryDates(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	CountCampaignsByUserUUID(ctx context.Context, userUUID uuid.UUID) (int, error)
	ListCampaignsByUserUUID(ctx context.Context, userUUID uuid.UUID) ([]*campaign.Summary, error)
}
