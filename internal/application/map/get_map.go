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

// IGetMap fetches a tactical map and returns whether the requester is its campaign master.
// The isMaster flag drives role-aware filtering in the handler.
type IGetMap interface {
	GetMap(ctx context.Context, mapID uuid.UUID, requesterID uuid.UUID) (*entity.TacticalMap, bool, error)
}

type GetMapUC struct {
	repo         IRepository
	campaignRepo campaignApp.IRepository
}

func NewGetMapUC(repo IRepository, campaignRepo campaignApp.IRepository) *GetMapUC {
	return &GetMapUC{repo: repo, campaignRepo: campaignRepo}
}

func (uc *GetMapUC) GetMap(ctx context.Context, mapID uuid.UUID, requesterID uuid.UUID) (*entity.TacticalMap, bool, error) {
	m, err := uc.repo.GetMap(ctx, mapID)
	if err != nil {
		if errors.Is(err, pgmap.ErrMapNotFound) {
			return nil, false, ErrMapNotFound
		}
		return nil, false, err
	}
	masterID, err := uc.campaignRepo.GetCampaignMasterUUID(ctx, m.CampaignID)
	if err != nil {
		// Best-effort: if we cannot determine master, treat as non-master (safe default).
		return m, false, nil
	}
	return m, masterID == requesterID, nil
}
