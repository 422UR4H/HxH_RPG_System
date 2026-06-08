package matchmapuc

import (
	"context"
	"errors"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	pgmatchmap "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/matchmap"
	"github.com/google/uuid"
)

type IGetMatchMap interface {
	Get(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error)
}

type GetMatchMapUC struct {
	repo IRepository
}

func NewGetMatchMapUC(repo IRepository) *GetMatchMapUC {
	return &GetMatchMapUC{repo: repo}
}

// Get returns nil if no map is attached (not an error).
func (uc *GetMatchMapUC) Get(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error) {
	mm, err := uc.repo.GetMatchMap(ctx, matchUUID)
	if err != nil {
		if errors.Is(err, pgmatchmap.ErrMatchMapNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return mm, nil
}
