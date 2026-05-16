package sheet

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetCharacterSheetBirthInfo(
	ctx context.Context,
	sheetUUID uuid.UUID,
) (time.Time, int, error) {
	const query = `
		SELECT birthday, age
		FROM character_profiles
		WHERE character_sheet_uuid = $1
	`
	var birthday time.Time
	var age int
	err := r.q.QueryRow(ctx, query, sheetUUID).Scan(&birthday, &age)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return time.Time{}, 0, ErrCharacterSheetNotFound
		}
		return time.Time{}, 0, fmt.Errorf("failed to get character sheet birth info: %w", err)
	}
	return birthday, age, nil
}
