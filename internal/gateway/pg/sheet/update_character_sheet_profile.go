package sheet

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) UpdateCharacterSheetProfile(
	ctx context.Context,
	sheetUUID uuid.UUID,
	playerUUID uuid.UUID,
	avatarURL *string,
	coverURL *string,
	description *string,
) error {
	const query = `
		UPDATE character_profiles cp
		SET avatar_url = $1, cover_url = $2, brief_description = $3, updated_at = $4
		FROM character_sheets cs
		WHERE cp.character_sheet_uuid = cs.uuid
		  AND cs.uuid = $5
		  AND cs.player_uuid = $6
	`
	tag, err := r.q.Exec(ctx, query, avatarURL, coverURL, description, time.Now(), sheetUUID, playerUUID)
	if err != nil {
		return fmt.Errorf("failed to update character sheet profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCharacterSheetNotFound
	}
	return nil
}
