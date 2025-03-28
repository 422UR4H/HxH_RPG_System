package charactersheet

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
)

type IRepository interface {
	CreateCharacterSheet(ctx context.Context, sheet *model.CharacterSheet) error
	ExistsCharacterWithNick(ctx context.Context, nick string) (bool, error)
	GetCharacterSheetByUUID(ctx context.Context, uuid string) (*model.CharacterSheet, error)
	UpdateNenHexagonValue(ctx context.Context, uuid string, val int) error
}
