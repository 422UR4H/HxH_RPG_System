package charactersheet

import (
	"context"

	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateCharacterSheet(ctx context.Context, sheet *sheet.CharacterSheet) error
	ExistsCharacterWithNick(ctx context.Context, nick string) (bool, error)
	CountCharactersByPlayerUUID(ctx context.Context, playerUUID uuid.UUID) (int, error)
	GetCharacterSheetPlayerUUID(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCharacterSheetByUUID(ctx context.Context, uuid string) (*sheet.CharacterSheet, error)
	ListCharacterSheetsByPlayerUUID(ctx context.Context, playerUUID string) ([]csEntity.Summary, error)
	UpdateNenHexagonValue(ctx context.Context, uuid string, val int) error
	GetCharacterSheetRelationshipUUIDs(ctx context.Context, uuid uuid.UUID) (csEntity.RelationshipUUIDs, error)
	ExistsSheetInCampaign(ctx context.Context, playerUUID uuid.UUID, campaignUUID uuid.UUID) (bool, error)
	UpdateStatusBars(ctx context.Context, sheetUUID string, health, stamina, aura status.IStatusBar) error
}
