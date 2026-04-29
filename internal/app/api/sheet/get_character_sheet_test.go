package sheet_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	domainAuth "github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestGetCharacterSheetHandler(t *testing.T) {
	userUUID := uuid.New()
	sheetUUID := uuid.New()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*domainSheet.CharacterSheet, error)
		wantStatus int
	}{
		{
			name: "success",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*domainSheet.CharacterSheet, error) {
				cs := buildTestCharacterSheet(t)
				cs.UUID = id
				return cs, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not_found",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*domainSheet.CharacterSheet, error) {
				return nil, charactersheet.ErrCharacterSheetNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "forbidden_insufficient_permissions",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*domainSheet.CharacterSheet, error) {
				return nil, domainAuth.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*domainSheet.CharacterSheet, error) {
				return nil, errors.New("unexpected error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockGetCharacterSheet{fn: tt.mockFn}
			handler := sheet.GetCharacterSheetHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/charactersheets/{uuid}",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/charactersheets/"+sheetUUID.String())

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if _, ok := result["character_sheet"].(map[string]any); !ok {
					t.Fatal("response missing 'character_sheet' field")
				}
			}
		})
	}
}
