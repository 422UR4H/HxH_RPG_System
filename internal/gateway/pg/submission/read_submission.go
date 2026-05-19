package submission

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
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

func (r *Repository) ExistsOtherCharacterWithNickInCampaign(
	ctx context.Context,
	nick string,
	campaignUUID uuid.UUID,
	excludedSheetUUID uuid.UUID,
) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM character_profiles cp
			JOIN character_sheets cs ON cp.character_sheet_uuid = cs.uuid
			LEFT JOIN submissions s ON s.character_sheet_uuid = cs.uuid AND s.campaign_uuid = $2
			WHERE cp.nickname = $1
			  AND cs.uuid != $3
			  AND (cs.campaign_uuid = $2 OR s.character_sheet_uuid IS NOT NULL)
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, nick, campaignUUID, excludedSheetUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check nick uniqueness in campaign: %w", err)
	}
	return exists, nil
}

func (r *Repository) GetSubmissionInfoBySheetUUID(
	ctx context.Context, sheetUUID uuid.UUID,
) (*charactersheet.SubmissionInfo, error) {
	const query = `
		SELECT campaign_uuid, created_at
		FROM submissions
		WHERE character_sheet_uuid = $1
	`
	var info charactersheet.SubmissionInfo
	err := r.q.QueryRow(ctx, query, sheetUUID).Scan(&info.CampaignUUID, &info.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get submission info: %w", err)
	}
	return &info, nil
}
