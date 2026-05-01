package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) AcceptEnrollment(
	ctx context.Context,
	enrollmentUUID uuid.UUID,
) error {
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
		UPDATE enrollments SET status = 'accepted'
		WHERE uuid = $1
	`
	result, err := tx.Exec(ctx, query, enrollmentUUID)
	if err != nil {
		return fmt.Errorf("failed to accept enrollment: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrEnrollmentNotFound
	}
	return nil
}
