package auth_test

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/google/uuid"
)

type mockRegister struct {
	fn func(ctx context.Context, input *auth.RegisterInput) error
}

func (m *mockRegister) Register(ctx context.Context, input *auth.RegisterInput) error {
	return m.fn(ctx, input)
}

type mockLogin struct {
	fn func(ctx context.Context, input *auth.LoginInput) (auth.LoginOutput, error)
}

func (m *mockLogin) Login(ctx context.Context, input *auth.LoginInput) (auth.LoginOutput, error) {
	return m.fn(ctx, input)
}

type mockSessionRepo struct {
	latest   func(ctx context.Context, userUUID uuid.UUID) (string, error)
	validate func(ctx context.Context, userUUID uuid.UUID, token string) (bool, error)
}

func (m *mockSessionRepo) CreateSession(context.Context, uuid.UUID, string) error { return nil }

func (m *mockSessionRepo) ValidateSession(ctx context.Context, userUUID uuid.UUID, token string) (bool, error) {
	return m.validate(ctx, userUUID, token)
}

func (m *mockSessionRepo) GetSessionTokenByUserUUID(ctx context.Context, userUUID uuid.UUID) (string, error) {
	return m.latest(ctx, userUUID)
}
