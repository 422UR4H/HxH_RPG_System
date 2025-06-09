package submission

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetSubmissionCampaignUUIDBySheetUUID(
	ctx context.Context, sheetUUID uuid.UUID,
) (uuid.UUID, error) {
	const query = `
        SELECT campaign_uuid
        FROM submissions
        WHERE character_sheet_uuid = $1
    `
	var campaignUUID uuid.UUID
	err := r.q.QueryRow(ctx, query, sheetUUID).Scan(&campaignUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrSubmissionNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get submission campaign UUID: %w", err)
	}
	return campaignUUID, nil
}

func (r *Repository) ExistsSubmittedCharacterSheet(
	ctx context.Context, uuid uuid.UUID,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM submissions
			WHERE character_sheet_uuid = $1
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, uuid).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if character is submitted: %w", err)
	}
	return exists, nil
}
