package auth_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	apiAuth "github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	jwtAuth "github.com/422UR4H/HxH_RPG_System/pkg/auth"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

// pingAPI mounts one protected route behind the middleware, with the session cache it is given.
func pingAPI(t *testing.T, sessions *sync.Map, repo *mockSessionRepo) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	api.UseMiddleware(apiAuth.AuthMiddlewareProvider(sessions, repo))
	huma.Register(api, huma.Operation{OperationID: "ping", Method: http.MethodGet, Path: "/ping"},
		func(context.Context, *struct{}) (*struct{}, error) { return &struct{}{}, nil })
	return api
}

func TestAuthMiddleware(t *testing.T) {
	userID := uuid.New()
	older, err := jwtAuth.GenerateToken(userID)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	newer := older + "-a-later-login" // only ever compared as a string, never parsed

	// One session row per login: every login the user ever made is a valid session, and the
	// latest is just the most recent of them.
	repo := &mockSessionRepo{
		latest: func(context.Context, uuid.UUID) (string, error) { return newer, nil },
		validate: func(_ context.Context, id uuid.UUID, token string) (bool, error) {
			return id == userID && (token == older || token == newer), nil
		},
	}

	tests := []struct {
		name     string
		sessions func() *sync.Map
		repo     *mockSessionRepo
		want     int
	}{
		{
			// The API process just restarted (air rebuild, deploy): the in-memory cache is empty.
			// A token from another tab or device is still a valid session — it must not be
			// refused just because it is not the user's latest login.
			name:     "empty cache, valid session that is not the latest login",
			sessions: func() *sync.Map { return &sync.Map{} },
			repo:     repo,
			want:     http.StatusNoContent,
		},
		{
			name: "warm cache holding another token, valid session",
			sessions: func() *sync.Map {
				m := &sync.Map{}
				m.Store(userID, newer)
				return m
			},
			repo: repo,
			want: http.StatusNoContent,
		},
		{
			name:     "empty cache, token with no session",
			sessions: func() *sync.Map { return &sync.Map{} },
			repo: &mockSessionRepo{
				latest:   repo.latest,
				validate: func(context.Context, uuid.UUID, string) (bool, error) { return false, nil },
			},
			want: http.StatusUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := pingAPI(t, tt.sessions(), tt.repo)
			resp := api.Get("/ping", "Authorization: Bearer "+older)
			if resp.Code != tt.want {
				t.Errorf("status = %d, want %d (body %s)", resp.Code, tt.want, resp.Body.String())
			}
		})
	}
}
