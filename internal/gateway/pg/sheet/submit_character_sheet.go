package sheet

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
			tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	const query = `
		INSERT INTO submit_character_sheets (
			uuid, character_sheet_uuid, campaign_uuid, created_at
		) VALUES (
			$1, $2, $3, $4
		)
	`
	_, err = tx.Exec(ctx, query,
		uuid.New(), sheetUUID, campaignUUID, createdAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save submitted character sheet: %w", err)
	}
	return nil
}

func (r *Repository) ExistsSubmittedCharacterSheet(
	ctx context.Context, uuid uuid.UUID,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM submit_character_sheets
			WHERE character_sheet_uuid = $1
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, uuid).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if submitted character sheet exists: %w", err)
	}
	return exists, nil
}
