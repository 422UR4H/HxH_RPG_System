package testutil

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
)

type MockAuthRepo struct {
	CreateUserFn          func(ctx context.Context, user *user.User) error
	GetUserByEmailFn      func(ctx context.Context, email string) (*user.User, error)
	ExistsUserWithNickFn  func(ctx context.Context, nick string) (bool, error)
	ExistsUserWithEmailFn func(ctx context.Context, email string) (bool, error)
}

func (m *MockAuthRepo) CreateUser(ctx context.Context, u *user.User) error {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, u)
	}
	return nil
}

func (m *MockAuthRepo) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(ctx, email)
	}
	return nil, nil
}

func (m *MockAuthRepo) ExistsUserWithNick(ctx context.Context, nick string) (bool, error) {
	if m.ExistsUserWithNickFn != nil {
		return m.ExistsUserWithNickFn(ctx, nick)
	}
	return false, nil
}

func (m *MockAuthRepo) ExistsUserWithEmail(ctx context.Context, email string) (bool, error) {
	if m.ExistsUserWithEmailFn != nil {
		return m.ExistsUserWithEmailFn(ctx, email)
	}
	return false, nil
}
