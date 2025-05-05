package scenario

import (
	"context"
	"fmt"

	scenarioEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

func (r *Repository) GetScenario(ctx context.Context, uuid uuid.UUID) (*scenarioEntity.Scenario, error) {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
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
			SELECT 
					uuid, user_uuid, name, brief_description, description, created_at, updated_at
			FROM scenarios
			WHERE uuid = $1
	`
	var scenario scenarioEntity.Scenario
	err = tx.QueryRow(ctx, query, uuid).Scan(
		&scenario.UUID,
		&scenario.UserUUID,
		&scenario.Name,
		&scenario.BriefDescription,
		&scenario.Description,
		&scenario.CreatedAt,
		&scenario.UpdatedAt,
	)
	if err != nil {
		return nil, ErrScenarioNotFound
	}
	return &scenario, nil
}

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
