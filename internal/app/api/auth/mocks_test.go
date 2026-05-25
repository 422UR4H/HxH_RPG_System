package auth_test

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
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
