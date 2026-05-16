package sheet_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/auth"
	. "github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	cs "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/google/uuid"
)

type mockProfileUpdater struct{ err error }

func (m *mockProfileUpdater) UpdateCharacterSheetProfile(_ context.Context, _, _ uuid.UUID, _, _ *string) error {
	return m.err
}

func TestPatchCharacterSheetProfileHandler_Success(t *testing.T) {
	handler := PatchCharacterSheetProfileHandler(&mockProfileUpdater{})
	sheetID := uuid.New()
	avatarURL := "https://pub.r2.dev/avatar/abc.webp"

	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	req := &PatchCharacterSheetProfileRequest{
		UUID: sheetID.String(),
		Body: PatchCharacterSheetProfileRequestBody{AvatarURL: &avatarURL},
	}

	resp, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != http.StatusNoContent {
		t.Errorf("expected 204, got %d", resp.Status)
	}
}

func TestPatchCharacterSheetProfileHandler_NotFound(t *testing.T) {
	handler := PatchCharacterSheetProfileHandler(&mockProfileUpdater{err: cs.ErrCharacterSheetNotFound})
	ctx := context.WithValue(context.Background(), auth.UserIDKey, uuid.New())
	req := &PatchCharacterSheetProfileRequest{
		UUID: uuid.New().String(),
		Body: PatchCharacterSheetProfileRequestBody{},
	}

	_, err := handler(ctx, req)
	if err == nil {
		t.Error("expected 404 error, got nil")
	}
}
