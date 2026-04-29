package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func validRegisterBody() map[string]any {
	return map[string]any{
		"nick":         "testuser",
		"email":        "testuser@example.com",
		"password":     "securepassword",
		"confirm_pass": "securepassword",
	}
}

func validLoginBody() map[string]any {
	return map[string]any{
		"email":    "testuser@example.com",
		"password": "securepassword",
	}
}

func TestRegisterHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *domainAuth.RegisterInput) error
		wantStatus int
	}{
		{
			name: "success",
			body: validRegisterBody(),
			mockFn: func(ctx context.Context, input *domainAuth.RegisterInput) error {
				return nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "bad_request_missing_nick",
			body: validRegisterBody(),
			mockFn: func(ctx context.Context, input *domainAuth.RegisterInput) error {
				return user.ErrMissingNick
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "conflict_nick_already_exists",
			body: validRegisterBody(),
			mockFn: func(ctx context.Context, input *domainAuth.RegisterInput) error {
				return user.ErrNickAlreadyExists
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "unprocessable_entity_invalid_nick_length",
			body: validRegisterBody(),
			mockFn: func(ctx context.Context, input *domainAuth.RegisterInput) error {
				return user.ErrInvalidNickLength
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_server_error",
			body: validRegisterBody(),
			mockFn: func(ctx context.Context, input *domainAuth.RegisterInput) error {
				return errors.New("unexpected db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockRegister{fn: tt.mockFn}
			handler := apiAuth.NewAuthHandler(mock, &mockLogin{
				fn: func(ctx context.Context, input *domainAuth.LoginInput) (domainAuth.LoginOutput, error) {
					return domainAuth.LoginOutput{}, nil
				},
			})

			huma.Register(api, huma.Operation{
				Method:        http.MethodPost,
				Path:          "/auth/register",
				DefaultStatus: http.StatusCreated,
			}, handler.Register)

			resp := api.Post("/auth/register", tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s",
					resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}

func TestLoginHandler(t *testing.T) {
	userUUID := uuid.New()

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *domainAuth.LoginInput) (domainAuth.LoginOutput, error)
		wantStatus int
	}{
		{
			name: "success",
			body: validLoginBody(),
			mockFn: func(ctx context.Context, input *domainAuth.LoginInput) (domainAuth.LoginOutput, error) {
				return domainAuth.LoginOutput{
					Token: "tok",
					User:  &user.User{UUID: userUUID, Nick: "test", Email: "test@test.com"},
				}, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "bad_request_missing_email",
			body: validLoginBody(),
			mockFn: func(ctx context.Context, input *domainAuth.LoginInput) (domainAuth.LoginOutput, error) {
				return domainAuth.LoginOutput{}, user.ErrMissingEmail
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "unauthorized",
			body: validLoginBody(),
			mockFn: func(ctx context.Context, input *domainAuth.LoginInput) (domainAuth.LoginOutput, error) {
				return domainAuth.LoginOutput{}, domainAuth.ErrUnauthorized
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "unprocessable_entity_invalid_email_length",
			body: validLoginBody(),
			mockFn: func(ctx context.Context, input *domainAuth.LoginInput) (domainAuth.LoginOutput, error) {
				return domainAuth.LoginOutput{}, user.ErrInvalidEmailLength
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_server_error",
			body: validLoginBody(),
			mockFn: func(ctx context.Context, input *domainAuth.LoginInput) (domainAuth.LoginOutput, error) {
				return domainAuth.LoginOutput{}, errors.New("unexpected db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockLogin{fn: tt.mockFn}
			handler := apiAuth.NewAuthHandler(&mockRegister{
				fn: func(ctx context.Context, input *domainAuth.RegisterInput) error {
					return nil
				},
			}, mock)

			huma.Register(api, huma.Operation{
				Method:        http.MethodPost,
				Path:          "/auth/login",
				DefaultStatus: http.StatusOK,
			}, handler.Login)

			resp := api.Post("/auth/login", tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s",
					resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if result["token"] != "tok" {
					t.Errorf("got token %v, want 'tok'", result["token"])
				}
				userData, ok := result["user"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'user' field")
				}
				if userData["nick"] != "test" {
					t.Errorf("got nick %v, want 'test'", userData["nick"])
				}
			}
		})
	}
}
