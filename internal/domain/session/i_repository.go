package session

import (
	"context"

	"github.com/google/uuid"
)

type IRepository interface {
	CreateSession(ctx context.Context, userUUID uuid.UUID, token string) error
	ValidateSession(ctx context.Context, userUUID uuid.UUID, token string) (bool, error)
	GetSessionTokenByUserUUID(ctx context.Context, userUUID uuid.UUID) (string, error)
}
