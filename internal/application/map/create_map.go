// internal/application/map/create_map.go
package mapuc

import (
	"context"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
	"github.com/google/uuid"
)

type ICreateMap interface {
	CreateMap(ctx context.Context, input *CreateMapInput) (*entity.TacticalMap, error)
}

type CreateMapInput struct {
	RequesterID uuid.UUID
	CampaignID  uuid.UUID
	Name        string
	Description string
}

type CreateMapUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewCreateMapUC(repo IRepository, campaignRepo campaignApp.IRepository) *CreateMapUC {
	return &CreateMapUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *CreateMapUC) CreateMap(ctx context.Context, input *CreateMapInput) (*entity.TacticalMap, error) {
	if err := service.ValidateMap(input.Name, entity.DefaultGrid()); err != nil {
		return nil, err
	}

	masterID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, input.CampaignID)
	if err != nil {
		return nil, err
	}
	if masterID != input.RequesterID {
		return nil, ErrNotMapMaster
	}

	m := entity.NewTacticalMap(input.CampaignID, input.Name, input.Description)
	if err := uc.repo.CreateMap(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}
