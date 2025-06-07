package charactersheet

import (
	"context"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
)

type IUpdateNenHexagonValue interface {
	UpdateNenHexagonValue(
		ctx context.Context, charSheet *sheet.CharacterSheet, method string,
	) (*spiritual.NenHexagonUpdateResult, error)
}

type UpdateNenHexagonValueUC struct {
	characterSheets *sync.Map
	repo            IRepository
}

func NewUpdateNenHexagonValueUC(
	charSheets *sync.Map,
	repo IRepository,
) *UpdateNenHexagonValueUC {
	return &UpdateNenHexagonValueUC{
		characterSheets: charSheets,
		repo:            repo,
	}
}

func (uc *UpdateNenHexagonValueUC) UpdateNenHexagonValue(
	ctx context.Context, charSheet *sheet.CharacterSheet, method string,
) (*spiritual.NenHexagonUpdateResult, error) {

	var result *spiritual.NenHexagonUpdateResult
	var err error

	switch method {
	case "increase":
		result, err = charSheet.IncreaseNenHexValue()
	case "decrease":
		result, err = charSheet.DecreaseNenHexValue()
	default:
		err = ErrInvalidUpdateHexValMethod
	}
	if err != nil {
		return nil, err
	}

	uuid := charSheet.UUID.String()
	dbErr := uc.repo.UpdateNenHexagonValue(ctx, uuid, result.CurrentHexVal)
	if dbErr != nil {
		return nil, domain.NewDBError(dbErr)
	}
	return result, nil
}
