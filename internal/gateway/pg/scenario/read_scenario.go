package scenario

import (
	"context"
	"fmt"
)

func (r *Repository) ExistsScenarioWithName(ctx context.Context, name string) (bool, error) {
	const query = `
        SELECT EXISTS(SELECT 1 FROM scenarios WHERE name = $1)
    `
	var exists bool
	err := r.q.QueryRow(ctx, query, name).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if scenario exists by name: %w", err)
	}
	return exists, nil
}
