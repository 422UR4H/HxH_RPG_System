package match

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	matchEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IGetMatch interface {
	GetMatch(
		ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID,
	) (*matchEntity.Match, error)
}

type GetMatchUC struct {
	repo IRepository
}

func NewGetMatchUC(repo IRepository) *GetMatchUC {
	return &GetMatchUC{
		repo: repo,
	}
}

func (uc *GetMatchUC) GetMatch(
	ctx context.Context, uuid uuid.UUID, userUUID uuid.UUID,
) (*matchEntity.Match, error) {
	match, err := uc.repo.GetMatch(ctx, uuid)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}

	if match.MasterUUID != userUUID && !match.IsPublic {
		return nil, auth.ErrInsufficientPermissions
	}
	return match, nil
}
