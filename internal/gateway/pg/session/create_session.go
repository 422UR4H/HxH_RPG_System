package session

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateSession persists a new session for a user in the database.
func (r *Repository) CreateSession(
	ctx context.Context, userUUID uuid.UUID, token string) error {

	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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
        INSERT INTO sessions (
            uuid, user_uuid, token, created_at
        ) VALUES (
            $1, $2, $3, $4
        )
    `
	_, err = tx.Exec(ctx, query,
		uuid.New(), userUUID, token, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}
