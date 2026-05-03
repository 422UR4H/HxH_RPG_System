package enrollment

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) EnrollCharacterSheet(
	ctx context.Context,
	matchUUID uuid.UUID,
	characterSheetUUID uuid.UUID,
) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx)
	}()

	const query = `
        INSERT INTO enrollments (
            uuid, match_uuid, character_sheet_uuid, created_at
        ) VALUES (
            $1, $2, $3, $4
        )
    `
	_, err = tx.Exec(ctx, query,
		uuid.New(), matchUUID, characterSheetUUID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to enroll character in match: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
