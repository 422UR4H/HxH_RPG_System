package enrollment

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/google/uuid"
)

func (r *Repository) ExistsEnrolledCharacterSheet(
	ctx context.Context, characterSheetUUID uuid.UUID, matchUUID uuid.UUID,
) (bool, error) {
	const query = `
        SELECT EXISTS (
            SELECT 1
            FROM enrollments
            WHERE character_sheet_uuid = $1 AND match_uuid = $2
        )
    `
	var exists bool
	err := r.q.QueryRow(ctx, query, characterSheetUUID, matchUUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if character is enrolled in this match: %w", err)
	}
	return exists, nil
}

func (r *Repository) GetEnrollmentByUUID(
	ctx context.Context,
	enrollmentUUID uuid.UUID,
) (string, uuid.UUID, error) {
	const query = `
		SELECT status, match_uuid
		FROM enrollments
		WHERE uuid = $1
	`
	var status string
	var matchUUID uuid.UUID
	err := r.q.QueryRow(ctx, query, enrollmentUUID).Scan(&status, &matchUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", uuid.Nil, ErrEnrollmentNotFound
		}
		return "", uuid.Nil, fmt.Errorf("failed to get enrollment: %w", err)
	}
	return status, matchUUID, nil
}
