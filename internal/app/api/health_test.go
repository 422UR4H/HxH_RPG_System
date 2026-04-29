package api_test

import (
	"net/http"
	"testing"

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

func TestReadinessHandler(t *testing.T) {
	_, testAPI := humatest.New(t)

	huma.Register(testAPI, huma.Operation{
		Method: http.MethodGet,
		Path:   "/readiness",
	}, api.ReadinessHandler())

	resp := testAPI.Get("/readiness")
	if resp.Code != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.Code, http.StatusOK)
	}
}
