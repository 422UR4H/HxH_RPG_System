package scenario

import (
	"context"
	"fmt"

	"github.com/google/uuid"
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

func (r *Repository) ExistsScenario(ctx context.Context, uuid uuid.UUID) (bool, error) {
	const query = `
        SELECT EXISTS(SELECT 1 FROM scenarios WHERE uuid = $1)
    `
	var exists bool
	err := r.q.QueryRow(ctx, query, uuid).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if scenario exists: %w", err)
	}
	return exists, nil
}
