// internal/application/map/update_map.go
package mapuc

import (
	"context"
	"errors"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	pgmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
	"github.com/google/uuid"
)

type IUpdateMap interface {
	UpdateMap(ctx context.Context, input *UpdateMapInput) error
}

type UpdateMapInput struct {
	RequesterID uuid.UUID
	MapID       uuid.UUID
	Name        string
	Description string
}

type UpdateMapUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewUpdateMapUC(repo IRepository, campaignRepo campaignApp.IRepository) *UpdateMapUC {
	return &UpdateMapUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *UpdateMapUC) UpdateMap(ctx context.Context, input *UpdateMapInput) error {
	m, err := uc.repo.GetMap(ctx, input.MapID)
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
	if masterID != input.RequesterID {
		return ErrNotMapMaster
	}

	if err := service.ValidateMap(input.Name, m.Grid); err != nil {
		return err
	}

	m.Name = input.Name
	m.Description = input.Description
	return uc.repo.UpdateMap(ctx, m)
}
