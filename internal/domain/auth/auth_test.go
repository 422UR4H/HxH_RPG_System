package auth_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgUser "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/user"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func validRegisterInput() *auth.RegisterInput {
	return &auth.RegisterInput{
		Nick:        "validnick",
		Email:       "valid@email.test",
		Password:    "strongpassword",
		ConfirmPass: "strongpassword",
	}
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name    string
		input   *auth.RegisterInput
		mock    *testutil.MockAuthRepo
		wantErr error
	}{
		{
			name:    "success",
			input:   validRegisterInput(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: nil,
		},
		{
			name: "missing nick",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.Nick = ""
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrMissingNick,
		},
		{
			name: "nick too short",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.Nick = "ab"
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrInvalidNickLength,
		},
		{
			name: "nick too long",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.Nick = "thisnickiswaytoolongforthevalidation"
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrInvalidNickLength,
		},
		{
			name: "missing email",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.Email = ""
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrMissingEmail,
		},
		{
			name: "email too short",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.Email = "a@b.co"
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrInvalidEmailLength,
		},
		{
			name: "email too long",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.Email = string(make([]byte, 65))
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrInvalidEmailLength,
		},
		{
			name: "missing password",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.Password = ""
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrMissingPassword,
		},
		{
			name: "missing confirm password",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.ConfirmPass = ""
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrMissingConfirmPass,
		},
		{
			name: "password too short",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.Password = "short"
				i.ConfirmPass = "short"
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrPasswordMinLenght,
		},
		{
			name: "password too long",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.Password = string(make([]byte, 65))
				i.ConfirmPass = string(make([]byte, 65))
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrPasswordMaxLenght,
		},
		{
			name: "password mismatch",
			input: func() *auth.RegisterInput {
				i := validRegisterInput()
				i.ConfirmPass = "different123"
				return i
			}(),
			mock:    &testutil.MockAuthRepo{},
			wantErr: user.ErrMismatchPassword,
		},
		{
			name:  "nick already exists",
			input: validRegisterInput(),
			mock: &testutil.MockAuthRepo{
				ExistsUserWithNickFn: func(ctx context.Context, nick string) (bool, error) {
					return true, nil
				},
			},
			wantErr: user.ErrNickAlreadyExists,
		},
		{
			name:  "email already exists",
			input: validRegisterInput(),
			mock: &testutil.MockAuthRepo{
				ExistsUserWithEmailFn: func(ctx context.Context, email string) (bool, error) {
					return true, nil
				},
			},
			wantErr: user.ErrEmailAlreadyExists,
		},
		{
			name:  "repo ExistsUserWithNick error",
			input: validRegisterInput(),
			mock: &testutil.MockAuthRepo{
				ExistsUserWithNickFn: func(ctx context.Context, nick string) (bool, error) {
					return false, errors.New("db error")
				},
			},
			wantErr: errors.New("db error"),
		},
		{
			name:  "repo CreateUser error",
			input: validRegisterInput(),
			mock: &testutil.MockAuthRepo{
				CreateUserFn: func(ctx context.Context, u *user.User) error {
					return errors.New("create failed")
				},
			},
			wantErr: errors.New("create failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := auth.NewRegisterUC(tt.mock)
			err := uc.Register(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("validpassword1"), bcrypt.DefaultCost)

	existingUser := &user.User{
		UUID:     uuid.New(),
		Nick:     "testuser",
		Email:    "test@user.email",
		Password: string(hashedPassword),
	}

	tests := []struct {
		name        string
		input       *auth.LoginInput
		authMock    *testutil.MockAuthRepo
		sessionMock *testutil.MockSessionRepo
		wantErr     error
	}{
		{
			name: "success",
			input: &auth.LoginInput{
				Email:    "test@user.email",
				Password: "validpassword1",
			},
			authMock: &testutil.MockAuthRepo{
				GetUserByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
					return existingUser, nil
				},
			},
			sessionMock: &testutil.MockSessionRepo{},
			wantErr:     nil,
		},
		{
			name: "missing email",
			input: &auth.LoginInput{
				Email:    "",
				Password: "validpassword1",
			},
			authMock:    &testutil.MockAuthRepo{},
			sessionMock: &testutil.MockSessionRepo{},
			wantErr:     user.ErrMissingEmail,
		},
		{
			name: "email too short",
			input: &auth.LoginInput{
				Email:    "a@b.co",
				Password: "validpassword1",
			},
			authMock:    &testutil.MockAuthRepo{},
			sessionMock: &testutil.MockSessionRepo{},
			wantErr:     user.ErrInvalidEmailLength,
		},
		{
			name: "missing password",
			input: &auth.LoginInput{
				Email:    "test@user.email",
				Password: "",
			},
			authMock:    &testutil.MockAuthRepo{},
			sessionMock: &testutil.MockSessionRepo{},
			wantErr:     user.ErrMissingPassword,
		},
		{
			name: "password too short",
			input: &auth.LoginInput{
				Email:    "test@user.email",
				Password: "short",
			},
			authMock:    &testutil.MockAuthRepo{},
			sessionMock: &testutil.MockSessionRepo{},
			wantErr:     user.ErrPasswordMinLenght,
		},
		{
			name: "email not found",
			input: &auth.LoginInput{
				Email:    "no@exist.email",
				Password: "validpassword1",
			},
			authMock: &testutil.MockAuthRepo{
				GetUserByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
					return nil, pgUser.ErrEmailNotFound
				},
			},
			sessionMock: &testutil.MockSessionRepo{},
			wantErr:     auth.ErrUnauthorized,
		},
		{
			name: "wrong password",
			input: &auth.LoginInput{
				Email:    "test@user.email",
				Password: "wrongpassword1",
			},
			authMock: &testutil.MockAuthRepo{
				GetUserByEmailFn: func(ctx context.Context, email string) (*user.User, error) {
					return existingUser, nil
				},
			},
			sessionMock: &testutil.MockSessionRepo{},
			wantErr:     auth.ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessions := &sync.Map{}
			uc := auth.NewLoginUC(sessions, tt.authMock, tt.sessionMock)
			output, err := uc.Login(context.Background(), tt.input)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if err.Error() != tt.wantErr.Error() {
					t.Fatalf("expected error %q, got %q", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if output.Token == "" {
				t.Error("expected non-empty token")
			}
			if output.User == nil {
				t.Error("expected non-nil user")
			}
		})
	}
}
