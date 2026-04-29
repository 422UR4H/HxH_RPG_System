package sheet_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestCreateCharacterSheetHandler(t *testing.T) {
	userUUID := uuid.New()

	validBody := map[string]any{
		"profile": map[string]any{
			"nickname":          "Gon",
			"fullname":          "Gon Freecss",
			"alignment":         "Chaotic-Good",
			"description":       "A young hunter",
			"brief_description": "Hunter boy",
			"age":               12,
			"birthday":          nil,
		},
		"character_class":    "Hunter",
		"skills_exps":        map[string]any{},
		"proficiencies_exps": map[string]any{},
		"categories": map[string]any{
			"Reinforcement": true,
		},
		"initial_hex_value": nil,
	}

	tests := []struct {
		name       string
		body       map[string]any
		mockFn     func(ctx context.Context, input *charactersheet.CreateCharacterSheetInput) (*domainSheet.CharacterSheet, error)
		wantStatus int
	}{
		{
			name: "success",
			body: validBody,
			mockFn: func(ctx context.Context, input *charactersheet.CreateCharacterSheetInput) (*domainSheet.CharacterSheet, error) {
				return buildTestCharacterSheet(t), nil
			},
			wantStatus: http.StatusCreated,
		},
		{
			name: "conflict_nickname_already_exists",
			body: validBody,
			mockFn: func(ctx context.Context, input *charactersheet.CreateCharacterSheetInput) (*domainSheet.CharacterSheet, error) {
				return nil, charactersheet.ErrNicknameAlreadyExists
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "internal_server_error",
			body: validBody,
			mockFn: func(ctx context.Context, input *charactersheet.CreateCharacterSheetInput) (*domainSheet.CharacterSheet, error) {
				return nil, errors.New("unexpected db error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockCreateCharacterSheet{fn: tt.mockFn}
			handler := sheet.CreateCharacterSheetHandler(mock)

			huma.Register(api, huma.Operation{
				Method:        http.MethodPost,
				Path:          "/charactersheets",
				DefaultStatus: http.StatusCreated,
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.PostCtx(ctx, "/charactersheets", tt.body)

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}
