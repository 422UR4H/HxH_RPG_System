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
	authUC "github.com/422UR4H/HxH_RPG_System/internal/application/auth"
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
				return nil, authUC.ErrInsufficientPermissions
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
				return nil, authUC.ErrInsufficientPermissions
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
				charSheet, ok := result["characterSheet"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'characterSheet' field")
				}

				profile, ok := charSheet["profile"].(map[string]any)
				if !ok {
					t.Fatal("response missing 'characterSheet.profile' field")
				}
				if _, exists := profile["brief_description"]; exists {
					t.Error("response profile leaked snake_case key 'brief_description'")
				}
				if _, exists := profile["avatar_url"]; exists {
					t.Error("response profile leaked snake_case key 'avatar_url'")
				}
				if got := profile["briefDescription"]; got != "Hunter boy" {
					t.Errorf("profile.briefDescription = %v, want %q", got, "Hunter boy")
				}
				if got := profile["avatarUrl"]; got != "https://example.com/avatar.png" {
					t.Errorf("profile.avatarUrl = %v, want %q", got, "https://example.com/avatar.png")
				}
				if got := profile["coverUrl"]; got != "https://example.com/cover.png" {
					t.Errorf("profile.coverUrl = %v, want %q", got, "https://example.com/cover.png")
				}
			}
		})
	}
}

// TestGetCharacterSheetHandler_ProfileUsesCamelCase guards against the
// entity-leak regression where CharacterSheetResponse.Profile embedded
// sheetEntity.CharacterProfile directly, shipping snake_case field names
// (profile.brief_description, profile.avatar_url, profile.cover_url) in
// every character-sheet response.
func TestGetCharacterSheetHandler_ProfileUsesCamelCase(t *testing.T) {
	userUUID := uuid.New()
	sheetUUID := uuid.New()

	_, api := humatest.New(t)

	mock := &mockGetCharacterSheet{
		fn: func(ctx context.Context, id uuid.UUID, uid uuid.UUID) (*sheetEntity.CharacterSheet, error) {
			charSheet := buildTestCharacterSheet(t)
			charSheet.UUID = id
			return charSheet, nil
		},
	}
	submissionFetcher := &mockSubmissionFetcher{}
	handler := sheet.GetCharacterSheetHandler(mock, submissionFetcher)

	huma.Register(api, huma.Operation{
		Method: http.MethodGet,
		Path:   "/charactersheets/{uuid}",
	}, handler)

	ctx := context.WithValue(context.Background(), auth.UserIDKey, userUUID)
	resp := api.GetCtx(ctx, "/charactersheets/"+sheetUUID.String())

	if resp.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d. Body: %s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	charSheetBody, ok := result["characterSheet"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'characterSheet' field")
	}

	profile, ok := charSheetBody["profile"].(map[string]any)
	if !ok {
		t.Fatal("response missing 'characterSheet.profile' field")
	}

	if _, exists := profile["brief_description"]; exists {
		t.Error("response profile leaked snake_case key 'brief_description'")
	}
	if _, exists := profile["avatar_url"]; exists {
		t.Error("response profile leaked snake_case key 'avatar_url'")
	}
	if _, exists := profile["cover_url"]; exists {
		t.Error("response profile leaked snake_case key 'cover_url'")
	}

	if got := profile["briefDescription"]; got != "Hunter boy" {
		t.Errorf("profile.briefDescription = %v, want %q", got, "Hunter boy")
	}
	if got := profile["avatarUrl"]; got != "https://example.com/avatar.png" {
		t.Errorf("profile.avatarUrl = %v, want %q", got, "https://example.com/avatar.png")
	}
	if got := profile["coverUrl"]; got != "https://example.com/cover.png" {
		t.Errorf("profile.coverUrl = %v, want %q", got, "https://example.com/cover.png")
	}
}
