package api_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestLivenessHandler(t *testing.T) {
	_, testAPI := humatest.New(t)

	huma.Register(testAPI, huma.Operation{
		Method: http.MethodGet,
		Path:   "/liveness",
	}, api.LivenessHandler())

	resp := testAPI.Get("/liveness")
	if resp.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.Code, http.StatusOK)
	}
}

type pingerFunc func(ctx context.Context) error

func (f pingerFunc) Ping(ctx context.Context) error { return f(ctx) }

func TestReadinessHandler(t *testing.T) {
	tests := []struct {
		name        string
		ping        pingerFunc
		wantStatus  int
		wantContain string
		wantAbsent  string
		maxDuration time.Duration
	}{
		{
			name:        "ping ok",
			ping:        func(context.Context) error { return nil },
			wantStatus:  http.StatusOK,
			wantContain: "Service is ready",
		},
		{
			name:        "ping error",
			ping:        func(context.Context) error { return errors.New("secret-dsn-detail") },
			wantStatus:  http.StatusServiceUnavailable,
			wantContain: "Database unavailable",
			wantAbsent:  "secret-dsn-detail",
		},
		{
			name: "ping blocks until deadline",
			ping: func(ctx context.Context) error {
				<-ctx.Done()
				return ctx.Err()
			},
			wantStatus:  http.StatusServiceUnavailable,
			wantContain: "Database unavailable",
			maxDuration: 3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, testAPI := humatest.New(t)
			huma.Register(testAPI, huma.Operation{
				Method: http.MethodGet,
				Path:   "/readiness",
			}, api.ReadinessHandler(tt.ping))

			start := time.Now()
			resp := testAPI.Get("/readiness")
			elapsed := time.Since(start)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", resp.Code, tt.wantStatus)
			}
			body := resp.Body.String()
			if !strings.Contains(body, tt.wantContain) {
				t.Errorf("body %q does not contain %q", body, tt.wantContain)
			}
			if tt.wantAbsent != "" && strings.Contains(body, tt.wantAbsent) {
				t.Errorf("body %q leaks %q", body, tt.wantAbsent)
			}
			if tt.maxDuration > 0 && elapsed >= tt.maxDuration {
				t.Errorf("took %v, want < %v", elapsed, tt.maxDuration)
			}
		})
	}
}
