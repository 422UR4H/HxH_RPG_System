package charactersheet

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

type IRepository interface {
	CreateCharacterSheet(ctx context.Context, sheet *model.CharacterSheet) error
	ExistsCharacterWithNick(ctx context.Context, nick string) (bool, error)
	ExistsSubmittedCharacterSheet(ctx context.Context, uuid uuid.UUID) (bool, error)
	SubmitCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, campaignUUID uuid.UUID, createdAt time.Time) error
	CountCharactersByPlayerUUID(ctx context.Context, playerUUID uuid.UUID) (int, error)
	GetCharacterSheetPlayerUUID(ctx context.Context, uuid uuid.UUID) (uuid.UUID, error)
	GetCharacterSheetByUUID(ctx context.Context, uuid string) (*model.CharacterSheet, error)
	ListCharacterSheetsByPlayerUUID(ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error)
	UpdateNenHexagonValue(ctx context.Context, uuid string, val int) error
}
