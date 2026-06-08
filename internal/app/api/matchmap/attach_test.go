package matchmapapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestAttachMatchMapHandler_Success(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	mapUUID := uuid.New()

	mockFn := func(_ context.Context, input *matchmapuc.AttachMatchMapInput) (*entity.MatchMap, error) {
		if input.RequesterUUID != userUUID {
			t.Errorf("requester uuid not forwarded: got %v", input.RequesterUUID)
		}
		if input.MatchUUID != matchUUID {
			t.Errorf("match uuid not forwarded: got %v", input.MatchUUID)
		}
		if input.MapUUID != mapUUID {
			t.Errorf("map uuid not forwarded: got %v", input.MapUUID)
		}
		return &entity.MatchMap{
			MatchUUID:  matchUUID.String(),
			MapUUID:    mapUUID.String(),
			AttachedAt: time.Now(),
		}, nil
	}

	_, api := humatest.New(t)
	handler := matchmapapi.AttachMatchMapHandler(&mockAttachMatchMap{fn: mockFn})

	huma.Register(api, huma.Operation{
		Method: http.MethodPost,
		Path:   "/matches/{match_uuid}/map",
		Errors: []int{
			http.StatusBadRequest, http.StatusForbidden,
			http.StatusNotFound, http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
	body := map[string]any{"map_uuid": mapUUID.String()}
	resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", body)

	if resp.Code != http.StatusOK {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	mm, ok := result["match_map"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'match_map' field")
	}
	if mm["map_uuid"] != mapUUID.String() {
		t.Errorf("got map_uuid %v, want %v", mm["map_uuid"], mapUUID.String())
	}
}

func TestAttachMatchMapHandler_NotMaster_Returns403(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	mapUUID := uuid.New()

	mockFn := func(_ context.Context, _ *matchmapuc.AttachMatchMapInput) (*entity.MatchMap, error) {
		return nil, matchmapuc.ErrNotMatchMaster
	}

	_, api := humatest.New(t)
	handler := matchmapapi.AttachMatchMapHandler(&mockAttachMatchMap{fn: mockFn})

	huma.Register(api, huma.Operation{
		Method: http.MethodPost,
		Path:   "/matches/{match_uuid}/map",
		Errors: []int{
			http.StatusBadRequest, http.StatusForbidden,
			http.StatusNotFound, http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
	body := map[string]any{"map_uuid": mapUUID.String()}
	resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", body)

	if resp.Code != http.StatusForbidden {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusForbidden, resp.Body.String())
	}
}

func TestAttachMatchMapHandler_AlreadyStarted_Returns422(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	mapUUID := uuid.New()

	mockFn := func(_ context.Context, _ *matchmapuc.AttachMatchMapInput) (*entity.MatchMap, error) {
		return nil, matchmapuc.ErrMatchAlreadyStarted
	}

	_, api := humatest.New(t)
	handler := matchmapapi.AttachMatchMapHandler(&mockAttachMatchMap{fn: mockFn})

	huma.Register(api, huma.Operation{
		Method: http.MethodPost,
		Path:   "/matches/{match_uuid}/map",
		Errors: []int{
			http.StatusBadRequest, http.StatusForbidden,
			http.StatusNotFound, http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
	body := map[string]any{"map_uuid": mapUUID.String()}
	resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", body)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusUnprocessableEntity, resp.Body.String())
	}
}
