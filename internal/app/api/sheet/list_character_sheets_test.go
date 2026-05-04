package sheet_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

func TestListCharacterSheetsHandler(t *testing.T) {
	userUUID := uuid.New()
	now := time.Now()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, uid uuid.UUID) ([]csEntity.Summary, error)
		wantStatus int
		wantCount  int
	}{
		{
			name: "success_with_sheets",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]csEntity.Summary, error) {
				playerUUID := uid
				return []csEntity.Summary{
					{
						UUID:           uuid.New(),
						PlayerUUID:     &playerUUID,
						NickName:       "Gon",
						FullName:       "Gon Freecss",
						Alignment:      "Chaotic-Good",
						CharacterClass: "Hunter",
						Birthday:       now,
						Level:          1,
						Stamina:        csEntity.StatusBar{Min: 0, Curr: 100, Max: 100},
						Health:         csEntity.StatusBar{Min: 0, Curr: 100, Max: 100},
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				}, nil
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, uid uuid.UUID) ([]csEntity.Summary, error) {
				return nil, errors.New("db connection failed")
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockListCharacterSheets{fn: tt.mockFn}
			handler := sheet.ListCharacterSheetsHandler(mock)

			huma.Register(api, huma.Operation{
				Method: http.MethodGet,
				Path:   "/charactersheets",
			}, handler)

			ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
			resp := api.GetCtx(ctx, "/charactersheets")

			if resp.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d. Body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var result map[string]any
				if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				sheets, ok := result["character_sheets"].([]any)
				if !ok {
					t.Fatal("response missing 'character_sheets' field")
				}
				if len(sheets) != tt.wantCount {
					t.Errorf("got %d sheets, want %d", len(sheets), tt.wantCount)
				}
			}
		})
	}
}
