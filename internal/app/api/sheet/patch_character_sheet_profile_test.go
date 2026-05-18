package sheet_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestPatchCharacterSheetProfile(t *testing.T) {
	userUUID := uuid.New()
	sheetUUID := uuid.New()
	avatarURL := "https://pub.r2.dev/avatar/abc.webp"

	validBody := map[string]any{
		"avatar_url": avatarURL,
	}

	tests := []struct {
		name       string
		pathUUID   string
		body       any
		ctx        context.Context
		mockFn     func(ctx context.Context, su, pu uuid.UUID, av, cv, desc *string) error
		wantStatus int
	}{
		{
			name:     "success",
			pathUUID: sheetUUID.String(),
			body:     validBody,
			ctx:      context.WithValue(context.Background(), auth.UserIDKey, userUUID),
			mockFn: func(ctx context.Context, su, pu uuid.UUID, av, cv, desc *string) error {
				return nil
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:     "sheet_not_found",
			pathUUID: sheetUUID.String(),
			body:     validBody,
			ctx:      context.WithValue(context.Background(), auth.UserIDKey, userUUID),
			mockFn: func(ctx context.Context, su, pu uuid.UUID, av, cv, desc *string) error {
				return charactersheet.ErrCharacterSheetNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:     "internal_server_error",
			pathUUID: sheetUUID.String(),
			body:     validBody,
			ctx:      context.WithValue(context.Background(), auth.UserIDKey, userUUID),
			mockFn: func(ctx context.Context, su, pu uuid.UUID, av, cv, desc *string) error {
				return errors.New("unexpected db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:     "invalid_uuid",
			pathUUID: "not-a-valid-uuid",
			body:     validBody,
			ctx:      context.WithValue(context.Background(), auth.UserIDKey, userUUID),
			mockFn: func(ctx context.Context, su, pu uuid.UUID, av, cv, desc *string) error {
				t.Fatal("UpdateCharacterSheetProfile should not be called with invalid UUID")
				return nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:     "missing_user_id_in_context",
			pathUUID: sheetUUID.String(),
			body:     validBody,
			ctx:      context.Background(),
			mockFn: func(ctx context.Context, su, pu uuid.UUID, av, cv, desc *string) error {
				t.Fatal("UpdateCharacterSheetProfile should not be called when userID is missing")
				return nil
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockProfileUpdater{fn: tt.mockFn}
			handler := sheet.PatchCharacterSheetProfileHandler(mock)

			huma.Register(api, huma.Operation{
				Method:        http.MethodPatch,
				Path:          "/charactersheets/{uuid}/profile",
				DefaultStatus: http.StatusNoContent,
			}, handler)

			resp := api.PatchCtx(tt.ctx, "/charactersheets/"+tt.pathUUID+"/profile", tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
