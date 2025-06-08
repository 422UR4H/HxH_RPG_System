package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// GetSessionTokenByUserUUID retrieves the most recent session token for a given user UUID.
func (r *Repository) GetSessionTokenByUserUUID(
	ctx context.Context, userUUID uuid.UUID) (string, error) {

	const query = `
        SELECT token
        FROM sessions
        WHERE user_uuid = $1
        ORDER BY created_at DESC
        LIMIT 1
    `
	var token string
	err := r.q.QueryRow(ctx, query, userUUID).Scan(&token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrSessionNotFound
		}
		return "", fmt.Errorf("failed to get session: %w", err)
	}
	return token, nil
}

// ValidateSession checks if a token exists for a given user
func (r *Repository) ValidateSession(
	ctx context.Context, userUUID uuid.UUID, token string) (bool, error) {

	const query = `
        SELECT EXISTS(
            SELECT 1
            FROM sessions
            WHERE user_uuid = $1
            AND token = $2
        )
    `
	var exists bool
	err := r.q.QueryRow(ctx, query, userUUID, token).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to validate session: %w", err)
	}
	return exists, nil
}
