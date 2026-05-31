// internal/application/map/get_map.go
package mapuc

import (
	"context"
	"errors"

	campaignApp "github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	pgmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/map"
	"github.com/google/uuid"
)

type IGetMap interface {
	GetMap(ctx context.Context, requesterID, mapID uuid.UUID) (*entity.TacticalMap, error)
}

type GetMapUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewGetMapUC(repo IRepository, campaignRepo campaignApp.IRepository) *GetMapUC {
	return &GetMapUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *GetMapUC) GetMap(ctx context.Context, requesterID, mapID uuid.UUID) (*entity.TacticalMap, error) {
	m, err := uc.repo.GetMap(ctx, mapID)
	if err != nil {
		if errors.Is(err, pgmap.ErrMapNotFound) {
			return nil, ErrMapNotFound
		}
		return nil, err
	}
	return m, nil
}
