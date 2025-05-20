package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/google/uuid"
)

type IListMatches interface {
	ListMatches(masterUUID uuid.UUID) ([]*match.Summary, error)
}

type ListMatchesUC struct {
	repo IRepository
}

func NewListMatchesUC(repo IRepository) *ListMatchesUC {
	return &ListMatchesUC{
		repo: repo,
	}
}

func (uc *ListMatchesUC) ListMatches(masterUUID uuid.UUID) ([]*match.Summary, error) {
	matches, err := uc.repo.ListMatchesByMasterUUID(context.Background(), masterUUID)
	if err != nil {
		return nil, err
	}
	return matches, nil
}
