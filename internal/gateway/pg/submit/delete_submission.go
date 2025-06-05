package submit

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) RejectCharacterSheetSubmission(
	ctx context.Context,
	sheetUUID uuid.UUID,
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

	const deleteSubmissionQuery = `
        DELETE FROM submit_character_sheets
        WHERE character_sheet_uuid = $1
    `
	result, err := tx.Exec(ctx, deleteSubmissionQuery, sheetUUID)
	if err != nil {
		return fmt.Errorf("failed to delete submission: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrSubmissionNotFound
	}
	return nil
}
