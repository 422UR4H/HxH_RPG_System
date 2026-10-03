package matchmapapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	matchmapapi "github.com/422UR4H/HxH_RPG_System/internal/app/api/matchmap"
	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
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
	body := map[string]any{"mapUuid": mapUUID.String()}
	resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", body)

	if resp.Code != http.StatusOK {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	mm, ok := result["matchMap"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'matchMap' field")
	}
	if mm["mapUuid"] != mapUUID.String() {
		t.Errorf("got mapUuid %v, want %v", mm["mapUuid"], mapUUID.String())
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
	body := map[string]any{"mapUuid": mapUUID.String()}
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
	body := map[string]any{"mapUuid": mapUUID.String()}
	resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", body)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusUnprocessableEntity, resp.Body.String())
	}
}

// TestAttachMatchMapHandler_ForwardsInheritBoardFromMatchUuid covers B16 (spec §4.3):
// the optional field must reach AttachMatchMapInput so the use case can validate and copy.
func TestAttachMatchMapHandler_ForwardsInheritBoardFromMatchUuid(t *testing.T) {
	userUUID := uuid.New()
	matchUUID := uuid.New()
	mapUUID := uuid.New()
	srcMatchUUID := uuid.New()

	mockFn := func(_ context.Context, input *matchmapuc.AttachMatchMapInput) (*entity.MatchMap, error) {
		if input.InheritBoardFromMatchUUID == nil {
			t.Fatalf("expected InheritBoardFromMatchUUID to be forwarded, got nil")
		}
		if *input.InheritBoardFromMatchUUID != srcMatchUUID {
			t.Errorf("got InheritBoardFromMatchUUID %v, want %v", *input.InheritBoardFromMatchUUID, srcMatchUUID)
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
	body := map[string]any{"mapUuid": mapUUID.String(), "inheritBoardFromMatchUuid": srcMatchUUID.String()}
	resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", body)

	if resp.Code != http.StatusOK {
		t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}
}

// TestAttachMatchMapHandler_SourceMatchErrors_Return422 covers the four B16 errors
// (spec §4.3): all map to 422, same as ErrMatchAlreadyStarted, per match-maps.md's existing
// pattern of using 422 for semantically-invalid requests rather than 409.
func TestAttachMatchMapHandler_SourceMatchErrors_Return422(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"is the same match", matchmapuc.ErrSourceMatchIsTheSameMatch},
		{"not in campaign", matchmapuc.ErrSourceMatchNotInCampaign},
		{"on another map", matchmapuc.ErrSourceMatchOnAnotherMap},
		{"has no board", matchmapuc.ErrSourceMatchHasNoBoard},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			userUUID := uuid.New()
			matchUUID := uuid.New()
			mapUUID := uuid.New()

			mockFn := func(_ context.Context, _ *matchmapuc.AttachMatchMapInput) (*entity.MatchMap, error) {
				return nil, tc.err
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
			body := map[string]any{"mapUuid": mapUUID.String(), "inheritBoardFromMatchUuid": uuid.New().String()}
			resp := api.PostCtx(ctx, "/matches/"+matchUUID.String()+"/map", body)

			if resp.Code != http.StatusUnprocessableEntity {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, http.StatusUnprocessableEntity, resp.Body.String())
			}
		})
	}
}
