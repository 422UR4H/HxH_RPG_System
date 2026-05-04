// internal/domain/character_sheet/list_character_sheets.go
package charactersheet

import (
	"context"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/google/uuid"
)

type IListCharacterSheets interface {
	ListCharacterSheets(
		ctx context.Context, playerId uuid.UUID,
	) ([]csEntity.Summary, error)
}

type ListCharacterSheetsUC struct {
	repo IRepository
}

func NewListCharacterSheetsUC(repo IRepository) *ListCharacterSheetsUC {
	return &ListCharacterSheetsUC{repo: repo}
}

func (uc *ListCharacterSheetsUC) ListCharacterSheets(
	ctx context.Context, playerId uuid.UUID,
) ([]csEntity.Summary, error) {
	return uc.repo.ListCharacterSheetsByPlayerUUID(ctx, playerId.String())
}
