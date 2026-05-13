package match

import (
	"context"

	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

type ICharSheetLoader interface {
	GetCharacterSheetByUUID(ctx context.Context, uuid string) (*csSheet.CharacterSheet, bool, error)
}
