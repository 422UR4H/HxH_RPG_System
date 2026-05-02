package submission

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) SubmitCharacterSheet(
	ctx context.Context,
	sheetUUID uuid.UUID,
	campaignUUID uuid.UUID,
	createdAt time.Time,
) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	const query = `
		INSERT INTO submissions (
			uuid, character_sheet_uuid, campaign_uuid, created_at
		) VALUES (
			$1, $2, $3, $4
		)
	`
	_, err = tx.Exec(ctx, query,
		uuid.New(), sheetUUID, campaignUUID, createdAt,
	)
	if err != nil {
		return fmt.Errorf("failed to submit character in campaign: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
