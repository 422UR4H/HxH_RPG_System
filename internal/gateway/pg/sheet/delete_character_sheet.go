package sheet

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) DeleteCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, playerUUID uuid.UUID,
) error {
	const query = `DELETE FROM character_sheets WHERE uuid = $1 AND player_uuid = $2`
	tag, err := r.q.Exec(ctx, query, sheetUUID, playerUUID)
	if err != nil {
		return fmt.Errorf("failed to delete character sheet: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCharacterSheetNotFound
	}
	return nil
}

func (r *Repository) DeleteNPCCharacterSheet(
	ctx context.Context, sheetUUID uuid.UUID, masterUUID uuid.UUID,
) error {
	const query = `DELETE FROM character_sheets WHERE uuid = $1 AND master_uuid = $2`
	tag, err := r.q.Exec(ctx, query, sheetUUID, masterUUID)
	if err != nil {
		return fmt.Errorf("failed to delete NPC character sheet: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCharacterSheetNotFound
	}
	return nil
}
