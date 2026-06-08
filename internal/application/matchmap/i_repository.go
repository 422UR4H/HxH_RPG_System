package matchmapuc

import (
	"context"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/google/uuid"
)

type IRepository interface {
	AttachMap(ctx context.Context, matchUUID, mapUUID uuid.UUID) (*entity.MatchMap, error)
	GetMatchMap(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error)
	DetachMap(ctx context.Context, matchUUID uuid.UUID) error
}
