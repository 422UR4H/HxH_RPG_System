package scenario

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
)

func (r *Repository) CreateScenario(ctx context.Context, scenario *scenario.Scenario) error {
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback(ctx)
			// TODO: refactor panic to
			// err = fmt.Errorf("internal error: recovered from panic: %v", p)
			panic(p)
			// TODO: refactor logging the error:
			// tx.Rollback(ctx)
			// logger.Error("recovered from panic in database operation",
			//     zap.Any("panic", p),
			//     zap.Stack("stack"))
			// err = errors.New("internal server error during database operation")
		} else if err != nil {
			tx.Rollback(ctx)
		} else {
			tx.Commit(ctx)
		}
	}()

	const query = `
        INSERT INTO scenarios (
            uuid, user_uuid,
						name, brief_description, description,
						created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7
        )
    `
	_, err = tx.Exec(ctx, query,
		scenario.UUID, scenario.UserUUID,
		scenario.Name, scenario.BriefDescription, scenario.Description,
		scenario.CreatedAt, scenario.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save scenario: %w", err)
	}
	return nil
}
