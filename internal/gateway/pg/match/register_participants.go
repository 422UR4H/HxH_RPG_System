package match

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) RegisterFromAcceptedEnrollments(
	ctx context.Context, matchUUID uuid.UUID, gameStartAt time.Time,
) error {
	now := time.Now()
	// INSERT ... SELECT is atomic in PostgreSQL — no explicit transaction needed.
	const query = `
		INSERT INTO match_participants
			(uuid, match_uuid, character_sheet_uuid, joined_at, created_at, updated_at)
		SELECT gen_random_uuid(), match_uuid, character_sheet_uuid, $2, $3, $3
		FROM enrollments
		WHERE match_uuid = $1 AND status = 'accepted'
		ON CONFLICT (match_uuid, character_sheet_uuid) DO NOTHING
	`
	_, err := r.q.Exec(ctx, query, matchUUID, gameStartAt, now)
	if err != nil {
		return fmt.Errorf("failed to register match participants: %w", err)
	}
	return nil
}
