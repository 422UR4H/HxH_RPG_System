package charactersheet

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

type IListCharacterSheets interface {
	ListCharacterSheets(
		ctx context.Context, playerId uuid.UUID,
	) ([]model.CharacterSheetSummary, error)
}

type ListCharacterSheetsUC struct {
	repo IRepository
}

func NewListCharacterSheetsUC(repo IRepository) *ListCharacterSheetsUC {
	return &ListCharacterSheetsUC{repo: repo}
}

func (uc *ListCharacterSheetsUC) ListCharacterSheets(
	ctx context.Context, playerId uuid.UUID,
) ([]model.CharacterSheetSummary, error) {

	sheet, err := uc.repo.ListCharacterSheetsByPlayerUUID(ctx, playerId.String())
	if err != nil {
		return nil, err
	}
	return sheet, nil
}
