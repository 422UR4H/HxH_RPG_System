package user

import (
	"context"
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"golang.org/x/crypto/bcrypt"
)

func (r *Repository) CreateUser(ctx context.Context, u *user.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	u.Password = string(hashedPassword)

	const query = `
        INSERT INTO users (uuid, nick, email, password, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
	_, err = r.q.Exec(
		ctx, query, u.UUID, u.Nick, u.Email, u.Password, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}
