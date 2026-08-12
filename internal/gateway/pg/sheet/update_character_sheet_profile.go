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
	// avatarURL, coverURL and description are *string: Go can't distinguish
	// an omitted JSON field from an explicit `null` (both unmarshal to nil),
	// so COALESCE is the only correct semantics here without switching to a
	// presence-wrapper type. Trade-off: this endpoint can no longer clear a
	// field to NULL — sending nil now means "leave it as is," not "erase
	// it." Acceptable today because there's no "remove avatar" UI; if that
	// ever exists, it will need a sentinel (e.g. empty string) or its own
	// endpoint.
	const query = `
		UPDATE character_profiles cp
		SET avatar_url        = COALESCE($1, cp.avatar_url),
		    cover_url         = COALESCE($2, cp.cover_url),
		    brief_description = COALESCE($3, cp.brief_description),
		    updated_at        = $4
		FROM character_sheets cs
		WHERE cp.character_sheet_uuid = cs.uuid
		  AND cs.uuid = $5
		  AND (cs.player_uuid = $6 OR cs.master_uuid = $6)
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
