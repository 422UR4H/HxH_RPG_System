package sheet

import (
	"context"
	"fmt"
)

func (r *Repository) ExistsCharacterWithNick(ctx context.Context, nick string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM character_profiles
			WHERE nickname = $1
		)
	`
	var exists bool
	err := r.q.QueryRow(ctx, query, nick).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if character profile exists by nickname: %w", err)
	}
	return exists, nil
}
