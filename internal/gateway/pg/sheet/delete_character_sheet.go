package sheet

import (
	"context"

	"github.com/google/uuid"
)

func (r *Repository) DeleteCharacterSheet(ctx context.Context, sheetUUID uuid.UUID, playerUUID uuid.UUID) error {
	sql := `
		DELETE FROM character_sheets
		WHERE uuid = $1 AND player_uuid = $2
	`

	_, err := r.q.Exec(ctx, sql, sheetUUID, playerUUID)
	if err != nil {
		return err
	}

	return nil
}
