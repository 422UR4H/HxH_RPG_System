// internal/application/map/update_map.go
package mapuc

import (
	"context"
	"errors"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
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
	Name        *string // nil = keep existing name
	Description string
	Grid        *entity.GridShape
	Bg          *entity.BgImage
	Pieces      *[]entity.Piece
	Walls       *[]entity.WallSegment // nil = keep existing; empty slice = clear all
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

	grid := m.Grid
	if input.Grid != nil {
		grid = *input.Grid
	}
	name := m.Name
	if input.Name != nil {
		name = *input.Name
	}
	if err := service.ValidateMap(name, grid); err != nil {
		return err
	}

	m.Name = name
	m.Description = input.Description
	m.Grid = grid
	if input.Bg != nil {
		m.Bg = input.Bg
	}
	if input.Pieces != nil {
		m.Pieces = *input.Pieces
	}
	if input.Walls != nil {
		if err := service.ValidateWallSegments(*input.Walls); err != nil {
			return err
		}
		m.Walls = *input.Walls
	}
	return uc.repo.UpdateMap(ctx, m)
}
