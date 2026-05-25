package sheet_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	sheetEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/google/uuid"
)

type mockSubmissionFetcher struct {
	info *cs.SubmissionInfo
	err  error
}

func (m *mockSubmissionFetcher) GetSubmissionInfoBySheetUUID(ctx context.Context, sheetUUID uuid.UUID) (*cs.SubmissionInfo, error) {
	return m.info, m.err
}

func TestGetCharacterSheetHandler(t *testing.T) {
	userUUID := uuid.New()
	sheetUUID := uuid.New()

	tests := []struct {
		name       string
		mockFn     func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error)
		wantStatus int
	}{
		{
			name: "success",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error) {
				cs := buildTestCharacterSheet(t)
				cs.UUID = id
				return cs, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "not_found",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error) {
				return nil, cs.ErrCharacterSheetNotFound
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "forbidden_insufficient_permissions",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error) {
				return nil, auth.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "internal_server_error",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error) {
				return nil, errors.New("unexpected error")
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "master_can_view_pending_sheet_via_submission_lookup",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error) {
				cs := buildTestCharacterSheet(t)
				cs.UUID = id
				return cs, nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "master_cannot_view_sheet_with_no_pending_submission",
			mockFn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error) {
				return nil, auth.ErrInsufficientPermissions
			},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, api := humatest.New(t)

			mock := &mockGetCharacterSheet{fn: tt.mockFn}
			submissionFetcher := &mockSubmissionFetcher{}
			handler := sheet.GetCharacterSheetHandler(mock, submissionFetcher)

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
