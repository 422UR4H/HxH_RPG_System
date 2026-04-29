package auth_test

import (
	"context"

	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
)

type mockRegister struct {
	fn func(ctx context.Context, input *domainAuth.RegisterInput) error
}

func (m *mockRegister) Register(ctx context.Context, input *domainAuth.RegisterInput) error {
	return m.fn(ctx, input)
}

type mockLogin struct {
	fn func(ctx context.Context, input *domainAuth.LoginInput) (domainAuth.LoginOutput, error)
}

func (m *mockLogin) Login(ctx context.Context, input *domainAuth.LoginInput) (domainAuth.LoginOutput, error) {
	return m.fn(ctx, input)
}
