package campaign

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
)

type IRepository interface {
	CreateCampaign(ctx context.Context, campaign *campaign.Campaign) error
}
