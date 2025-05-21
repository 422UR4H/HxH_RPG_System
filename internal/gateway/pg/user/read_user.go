package user

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	const query = `
        SELECT id, uuid, nick, email, password, created_at, updated_at
        FROM users
        WHERE email = $1
    `
	var u user.User
	err := r.q.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.UUID, &u.Nick, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrEmailNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	return &u, nil
}

func (r *Repository) ExistsUserWithNick(ctx context.Context, nick string) (bool, error) {
	const query = `
				SELECT EXISTS (
					SELECT 1
					FROM users
					WHERE nick = $1
				)
		`
	var exists bool
	err := r.q.QueryRow(ctx, query, nick).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}
	return exists, nil
}

func (r *Repository) ExistsUserWithEmail(ctx context.Context, email string) (bool, error) {
	const query = `
				SELECT EXISTS (
					SELECT 1
					FROM users
					WHERE email = $1
				) 
		`
	var exists bool
	err := r.q.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists: %w", err)
	}
	return exists, nil
}
