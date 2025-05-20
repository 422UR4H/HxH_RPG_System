package match

import (
	"context"
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	matchPg "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/match"
	"github.com/google/uuid"
)

type IGetMatch interface {
	GetMatch(uuid uuid.UUID) (*match.Match, error)
}

type GetMatchUC struct {
	repo IRepository
}

func NewGetMatchUC(repo IRepository) *GetMatchUC {
	return &GetMatchUC{
		repo: repo,
	}
}

func (uc *GetMatchUC) GetMatch(uuid uuid.UUID) (*match.Match, error) {
	match, err := uc.repo.GetMatch(context.Background(), uuid)
	if err != nil {
		if errors.Is(err, matchPg.ErrMatchNotFound) {
			return nil, ErrMatchNotFound
		}
		return nil, err
	}
	return match, nil
}
