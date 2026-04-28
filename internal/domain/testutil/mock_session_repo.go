package testutil

import (
	"context"

	"github.com/google/uuid"
)

type MockSessionRepo struct {
	CreateSessionFn             func(ctx context.Context, userUUID uuid.UUID, token string) error
	ValidateSessionFn           func(ctx context.Context, userUUID uuid.UUID, token string) (bool, error)
	GetSessionTokenByUserUUIDFn func(ctx context.Context, userUUID uuid.UUID) (string, error)
}

func (m *MockSessionRepo) CreateSession(ctx context.Context, userUUID uuid.UUID, token string) error {
	if m.CreateSessionFn != nil {
		return m.CreateSessionFn(ctx, userUUID, token)
	}
	return nil
}

func (m *MockSessionRepo) ValidateSession(ctx context.Context, userUUID uuid.UUID, token string) (bool, error) {
	if m.ValidateSessionFn != nil {
		return m.ValidateSessionFn(ctx, userUUID, token)
	}
	return true, nil
}

func (m *MockSessionRepo) GetSessionTokenByUserUUID(ctx context.Context, userUUID uuid.UUID) (string, error) {
	if m.GetSessionTokenByUserUUIDFn != nil {
		return m.GetSessionTokenByUserUUIDFn(ctx, userUUID)
	}
	return "", nil
}
