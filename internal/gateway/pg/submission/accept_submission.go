package submission

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func (r *Repository) AcceptCharacterSheetSubmission(
	ctx context.Context,
	sheetUUID uuid.UUID,
	campaignUUID uuid.UUID,
	birthday time.Time,
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

	now := time.Now()
	const updateSheetQuery = `
        UPDATE character_sheets
        SET campaign_uuid = $1, updated_at = $2
        WHERE uuid = $3
    `
	_, err = tx.Exec(ctx, updateSheetQuery, campaignUUID, now, sheetUUID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrNickConflict
		}
		return fmt.Errorf("failed to update character sheet with campaign: %w", err)
	}

	const updateBirthdayQuery = `
        UPDATE character_profiles
        SET birthday = $1, updated_at = $2
        WHERE character_sheet_uuid = $3
    `
	_, err = tx.Exec(ctx, updateBirthdayQuery, birthday, now, sheetUUID)
	if err != nil {
		return fmt.Errorf("failed to update character profile birthday: %w", err)
	}

	const deleteSubmissionQuery = `
        DELETE FROM submissions
        WHERE character_sheet_uuid = $1
    `
	_, err = tx.Exec(ctx, deleteSubmissionQuery, sheetUUID)
	if err != nil {
		return fmt.Errorf("failed to delete submission: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
