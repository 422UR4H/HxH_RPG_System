package matchmapapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestDetachMatchMapHandler_Success(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()

	mockFn := func(_ context.Context, input *matchmapuc.DetachMatchMapInput) error {
		if input.RequesterUUID != userUUID {
			t.Errorf("requester uuid not forwarded: got %v", input.RequesterUUID)
		}
		if input.MatchUUID != matchUUID {
			t.Errorf("match uuid not forwarded: got %v", input.MatchUUID)
		}
		return nil
	}

	_, api := humatest.New(t)
	handler := matchmapapi.DetachMatchMapHandler(&mockDetachMatchMap{fn: mockFn})

	huma.Register(api, huma.Operation{
		Method:        http.MethodDelete,
		Path:          "/matches/{match_uuid}/map",
		DefaultStatus: http.StatusNoContent,
		Errors: []int{
			http.StatusBadRequest, http.StatusForbidden,
			http.StatusNotFound, http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
	resp := api.DeleteCtx(ctx, "/matches/"+matchUUID.String()+"/map")

	if resp.Code != http.StatusNoContent {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusNoContent, resp.Body.String())
	}
}

func TestDetachMatchMapHandler_NotMaster_Returns403(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()

	mockFn := func(_ context.Context, _ *matchmapuc.DetachMatchMapInput) error {
		return matchmapuc.ErrNotMatchMaster
	}

	_, api := humatest.New(t)
	handler := matchmapapi.DetachMatchMapHandler(&mockDetachMatchMap{fn: mockFn})

	huma.Register(api, huma.Operation{
		Method:        http.MethodDelete,
		Path:          "/matches/{match_uuid}/map",
		DefaultStatus: http.StatusNoContent,
		Errors: []int{
			http.StatusBadRequest, http.StatusForbidden,
			http.StatusNotFound, http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
	resp := api.DeleteCtx(ctx, "/matches/"+matchUUID.String()+"/map")

	if resp.Code != http.StatusForbidden {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusForbidden, resp.Body.String())
	}
}

func TestDetachMatchMapHandler_AlreadyStarted_Returns422(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()

	mockFn := func(_ context.Context, _ *matchmapuc.DetachMatchMapInput) error {
		return matchmapuc.ErrMatchAlreadyStarted
	}

	_, api := humatest.New(t)
	handler := matchmapapi.DetachMatchMapHandler(&mockDetachMatchMap{fn: mockFn})

	huma.Register(api, huma.Operation{
		Method:        http.MethodDelete,
		Path:          "/matches/{match_uuid}/map",
		DefaultStatus: http.StatusNoContent,
		Errors: []int{
			http.StatusBadRequest, http.StatusForbidden,
			http.StatusNotFound, http.StatusUnprocessableEntity,
			http.StatusInternalServerError,
		},
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
	resp := api.DeleteCtx(ctx, "/matches/"+matchUUID.String()+"/map")

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusUnprocessableEntity, resp.Body.String())
	}
}
