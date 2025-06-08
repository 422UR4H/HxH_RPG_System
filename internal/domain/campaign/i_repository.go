package campaign

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateCampaign(ctx context.Context, campaign *campaign.Campaign) error
	GetCampaign(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	GetCampaignMasterUUID(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCampaignStoryDates(ctx context.Context, uuid uuid.UUID) (*campaign.Campaign, error)
	CountCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) (int, error)
	ListCampaignsByMasterUUID(ctx context.Context, masterUUID uuid.UUID) ([]*campaign.Summary, error)
}
