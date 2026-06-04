package matchmapapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	matchmapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestGetMatchMapHandler_WithMap_Returns200(t *testing.T) {
	matchUUID := uuid.New()
	mapUUID := uuid.New()

	mockFn := func(_ context.Context, mUUID uuid.UUID) (*entity.MatchMap, error) {
		if mUUID != matchUUID {
			t.Errorf("match uuid not forwarded: got %v", mUUID)
		}
		return &entity.MatchMap{
			MatchUUID:  matchUUID.String(),
			MapUUID:    mapUUID.String(),
			AttachedAt: time.Now(),
		}, nil
	}

	_, api := humatest.New(t)
	handler := matchmapapi.GetMatchMapHandler(&mockGetMatchMap{fn: mockFn})

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/matches/{match_uuid}/map",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusInternalServerError,
		},
	}, handler)

	resp := api.GetCtx(context.Background(), "/matches/"+matchUUID.String()+"/map")

	if resp.Code != http.StatusOK {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if _, ok := result["match_map"]; !ok {
		t.Fatal("response missing 'match_map' field")
	}
}

func TestGetMatchMapHandler_NoMap_Returns204(t *testing.T) {
	matchUUID := uuid.New()

	mockFn := func(_ context.Context, _ uuid.UUID) (*entity.MatchMap, error) {
		return nil, nil
	}

	_, api := humatest.New(t)
	handler := matchmapapi.GetMatchMapHandler(&mockGetMatchMap{fn: mockFn})

	huma.Register(api, huma.Operation{
		Method:        http.MethodGet,
		Path:          "/matches/{match_uuid}/map",
		DefaultStatus: http.StatusNoContent,
		Errors: []int{
			http.StatusBadRequest,
			http.StatusInternalServerError,
		},
	}, handler)

	resp := api.GetCtx(context.Background(), "/matches/"+matchUUID.String()+"/map")

	if resp.Code != http.StatusNoContent {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusNoContent, resp.Body.String())
	}
}

func TestGetMatchMapHandler_InternalError_Returns500(t *testing.T) {
	matchUUID := uuid.New()

	mockFn := func(_ context.Context, _ uuid.UUID) (*entity.MatchMap, error) {
		return nil, errors.New("db error")
	}

	_, api := humatest.New(t)
	handler := matchmapapi.GetMatchMapHandler(&mockGetMatchMap{fn: mockFn})

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/matches/{match_uuid}/map",
		Errors: []int{
			http.StatusBadRequest,
			http.StatusInternalServerError,
		},
	}, handler)

	resp := api.GetCtx(context.Background(), "/matches/"+matchUUID.String()+"/map")

	if resp.Code != http.StatusInternalServerError {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusInternalServerError, resp.Body.String())
	}
}

