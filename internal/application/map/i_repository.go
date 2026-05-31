// internal/application/map/i_repository.go
package mapuc

import (
	"context"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateMap(ctx context.Context, m *entity.TacticalMap) error
	GetMap(ctx context.Context, id uuid.UUID) (*entity.TacticalMap, error)
	ListMapsByCampaign(ctx context.Context, campaignID uuid.UUID) ([]*entity.TacticalMap, error)
	UpdateMap(ctx context.Context, m *entity.TacticalMap) error
	DeleteMap(ctx context.Context, id uuid.UUID) error
}
