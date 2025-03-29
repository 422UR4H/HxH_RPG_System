package charactersheet

import (
	"context"
	"sync"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/sheet"
)

type IUpdateNenHexagonValue interface {
	UpdateNenHexagonValue(
		ctx context.Context, charSheet *sheet.CharacterSheet, method string,
	) (map[enum.CategoryName]float64, enum.CategoryName, error)
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
) (map[enum.CategoryName]float64, enum.CategoryName, error) {

	var percentList map[enum.CategoryName]float64
	var categoryName enum.CategoryName
	var err error
	var dbErr error
	uuid := charSheet.UUID.String()

	switch method {
	case "increase":
		percentList, categoryName, err = charSheet.IncreaseNenHexValue()
		dbErr = uc.repo.IncreaseNenHexagonValue(ctx, uuid)
	case "decrease":
		percentList, categoryName, err = charSheet.DecreaseNenHexValue()
		dbErr = uc.repo.DecreaseNenHexagonValue(ctx, uuid)
	default:
		err = ErrInvalidUpdateHexValMethod
	}

	if err != nil {
		return nil, categoryName, err
	}
	if dbErr != nil {
		return nil, categoryName, domain.NewDBError(dbErr)
	}
	return percentList, categoryName, nil
}
