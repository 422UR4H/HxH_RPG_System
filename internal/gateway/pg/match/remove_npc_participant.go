package match

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// RemoveNPCParticipant hard-deletes a master-controlled NPC (character sheet
// with no player_uuid) from a match's roster. The NPC guard lives in the SQL
// (player_uuid IS NULL) so this can never delete a player's roster row, even
// if an upper layer passes the wrong sheet UUID.
func (r *Repository) RemoveNPCParticipant(
	ctx context.Context, matchUUID, sheetUUID uuid.UUID,
) error {
	const query = `
		DELETE FROM match_participants mp
		USING character_sheets cs
		WHERE mp.character_sheet_uuid = cs.uuid
		  AND mp.match_uuid = $1
		  AND mp.character_sheet_uuid = $2
		  AND cs.player_uuid IS NULL
	`
	result, err := r.q.Exec(ctx, query, matchUUID, sheetUUID)
	if err != nil {
		return fmt.Errorf("failed to remove npc participant: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNPCNotInMatch
	}
	return nil
}
