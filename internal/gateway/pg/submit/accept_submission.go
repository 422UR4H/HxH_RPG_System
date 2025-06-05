package submit

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (r *Repository) AcceptCharacterSheetSubmission(
	ctx context.Context,
	sheetUUID uuid.UUID,
	campaignUUID uuid.UUID,
) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	now := time.Now()
	const updateSheetQuery = `
        UPDATE character_sheets
        SET campaign_uuid = $1, updated_at = $2
        WHERE uuid = $3
    `
	_, err = tx.Exec(ctx, updateSheetQuery, campaignUUID, now, sheetUUID)
	if err != nil {
		return fmt.Errorf("failed to update character sheet with campaign: %w", err)
	}

	const deleteSubmissionQuery = `
        DELETE FROM submit_character_sheets
        WHERE character_sheet_uuid = $1
    `
	_, err = tx.Exec(ctx, deleteSubmissionQuery, sheetUUID)
	if err != nil {
		return fmt.Errorf("failed to delete submission: %w", err)
	}

	return nil
}
