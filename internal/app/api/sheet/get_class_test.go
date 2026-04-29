package sheet_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func TestGetClassHandler(t *testing.T) {
	tests := []struct {
		name       string
		getClassFn func(name string) (cc.CharacterClass, error)
		getSheetFn func(name string) (domainSheet.HalfSheet, error)
		wantStatus int
	}{
		{
			name: "success",
			getClassFn: func(name string) (cc.CharacterClass, error) {
				return cc.CharacterClass{}, nil
			},
			getSheetFn: func(name string) (domainSheet.HalfSheet, error) {
				return buildTestHalfSheet(t), nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not_found_invalid_class_name",
			getClassFn: func(name string) (cc.CharacterClass, error) {
				return cc.CharacterClass{}, errors.New("invalid class name")
			},
			getSheetFn: func(name string) (domainSheet.HalfSheet, error) {
				t.Fatal("GetClassSheet should not be called when GetCharacterClass fails")
				return domainSheet.HalfSheet{}, nil
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "internal_error_class_sheet_failure",
			getClassFn: func(name string) (cc.CharacterClass, error) {
				return cc.CharacterClass{}, nil
			},
			getSheetFn: func(name string) (domainSheet.HalfSheet, error) {
				return domainSheet.HalfSheet{}, errors.New("unexpected error")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockGetCharacterClass{
				getClassFn: tt.getClassFn,
				getSheetFn: tt.getSheetFn,
			}
			handler := sheet.GetClassHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/classes/{name}",
			}, handler)

			resp := api.GetCtx(context.Background(), "/classes/Hunter")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if _, ok := result["CharacterClass"]; !ok {
					t.Fatal("response missing 'CharacterClass' field")
				}
			}
		})
	}
}
