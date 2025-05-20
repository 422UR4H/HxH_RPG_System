package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateMatch(ctx context.Context, match *match.Match) error
	GetMatch(ctx context.Context, uuid uuid.UUID) (*match.Match, error)
}
