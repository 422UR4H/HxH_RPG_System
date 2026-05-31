// internal/application/map/list_maps.go
package mapuc

import (
	"context"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type IListMaps interface {
	ListMaps(ctx context.Context, requesterID, campaignID uuid.UUID) ([]*entity.TacticalMap, error)
}

type ListMapsUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewListMapsUC(repo IRepository, campaignRepo campaignApp.IRepository) *ListMapsUC {
	return &ListMapsUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *ListMapsUC) ListMaps(ctx context.Context, requesterID, campaignID uuid.UUID) ([]*entity.TacticalMap, error) {
	masterID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	if masterID != requesterID {
		return nil, ErrNotMapMaster
	}
	return uc.repo.ListMapsByCampaign(ctx, campaignID)
}
