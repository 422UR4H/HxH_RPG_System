// internal/application/map/delete_map.go
package mapuc

import (
	"context"
	"errors"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	pgmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
	"github.com/google/uuid"
)

type IDeleteMap interface {
	DeleteMap(ctx context.Context, requesterID, mapID uuid.UUID) error
}

type DeleteMapUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewDeleteMapUC(repo IRepository, campaignRepo campaignApp.IRepository) *DeleteMapUC {
	return &DeleteMapUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *DeleteMapUC) DeleteMap(ctx context.Context, requesterID, mapID uuid.UUID) error {
	m, err := uc.repo.GetMap(ctx, mapID)
	if err != nil {
		if errors.Is(err, pgmap.ErrMapNotFound) {
			return ErrMapNotFound
		}
		return err
	}

	masterID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, m.CampaignID)
	if err != nil {
		return err
	}
	if masterID != requesterID {
		return ErrNotMapMaster
	}

	return uc.repo.DeleteMap(ctx, mapID)
}
