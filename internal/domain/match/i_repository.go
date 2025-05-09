package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
)

type IRepository interface {
	CreateMatch(ctx context.Context, match *match.Match) error
}
