package user

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
	"golang.org/x/crypto/bcrypt"
)

type Repository struct {
	q pgfs.IQuerier
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{q: q}
}

func (r *Repository) CreateUser(ctx context.Context, u *user.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	u.Password = string(hashedPassword)

	const query = `
        INSERT INTO users (uuid, nick, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING uuid
    `
	err = r.q.QueryRow(ctx, query, u.UUID, u.Nick, u.Email, u.Password, u.CreatedAt, u.UpdatedAt).Scan(&u.ID)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	const query = `
        SELECT id, uuid, nick, email, password, created_at, updated_at
        FROM users
        WHERE email = $1
    `
	var u user.User
	err := r.q.QueryRow(ctx, query, email).Scan(&u.ID, &u.UUID, &u.Nick, &u.Email, &u.Password, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	return &u, nil
}
