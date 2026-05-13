package auth

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
)

type IRepository interface {
	CreateUser(ctx context.Context, user *user.User) error
	GetUserByEmail(ctx context.Context, email string) (*user.User, error)
	ExistsUserWithNick(ctx context.Context, nick string) (bool, error)
	ExistsUserWithEmail(ctx context.Context, email string) (bool, error)
}
