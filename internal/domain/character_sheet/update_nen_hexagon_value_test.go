package charactersheet_test

import (
	"context"
	"errors"
	"testing"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	"github.com/google/uuid"
)

func TestUpdateNenHexagonValue(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - increase", func(t *testing.T) {
		factory := newTestFactory()
		playerUUID := uuid.New()
		profile := newValidProfile()
		hexValue := 3

		charSheet, err := factory.Build(
			&playerUUID, nil, nil, profile, &hexValue, nil, nil,
		)
		if err != nil {
			t.Fatalf("failed to build character sheet: %v", err)
		}
		charSheet.UUID = uuid.New()

		sheetMap := newTestSheetMap()
		mockRepo := &testutil.MockCharacterSheetRepo{
			UpdateNenHexagonValueFn: func(ctx context.Context, id string, val int) error {
				return nil
			},
		}

		uc := charactersheet.NewUpdateNenHexagonValueUC(sheetMap, mockRepo)

		result, err := uc.UpdateNenHexagonValue(ctx, charSheet, "increase")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
		if result.CurrentHexVal != 4 {
			t.Errorf("expected hex value 4, got %d", result.CurrentHexVal)
		}
	})

	t.Run("happy path - decrease", func(t *testing.T) {
		factory := newTestFactory()
		playerUUID := uuid.New()
		profile := newValidProfile()
		hexValue := 3

		charSheet, err := factory.Build(
			&playerUUID, nil, nil, profile, &hexValue, nil, nil,
		)
		if err != nil {
			t.Fatalf("failed to build character sheet: %v", err)
		}
		charSheet.UUID = uuid.New()

		sheetMap := newTestSheetMap()
		mockRepo := &testutil.MockCharacterSheetRepo{
			UpdateNenHexagonValueFn: func(ctx context.Context, id string, val int) error {
				return nil
			},
		}

		uc := charactersheet.NewUpdateNenHexagonValueUC(sheetMap, mockRepo)

		result, err := uc.UpdateNenHexagonValue(ctx, charSheet, "decrease")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected result, got nil")
		}
		if result.CurrentHexVal != 2 {
			t.Errorf("expected hex value 2, got %d", result.CurrentHexVal)
		}
	})

	t.Run("error - invalid method", func(t *testing.T) {
		factory := newTestFactory()
		playerUUID := uuid.New()
		profile := newValidProfile()
		hexValue := 3

		charSheet, err := factory.Build(
			&playerUUID, nil, nil, profile, &hexValue, nil, nil,
		)
		if err != nil {
			t.Fatalf("failed to build character sheet: %v", err)
		}
		charSheet.UUID = uuid.New()

		sheetMap := newTestSheetMap()
		mockRepo := &testutil.MockCharacterSheetRepo{}

		uc := charactersheet.NewUpdateNenHexagonValueUC(sheetMap, mockRepo)

		_, err = uc.UpdateNenHexagonValue(ctx, charSheet, "invalid")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, charactersheet.ErrInvalidUpdateHexValMethod) {
			t.Errorf("expected ErrInvalidUpdateHexValMethod, got: %v", err)
		}
	})

	t.Run("error - repo update error", func(t *testing.T) {
		factory := newTestFactory()
		playerUUID := uuid.New()
		profile := newValidProfile()
		hexValue := 3

		charSheet, err := factory.Build(
			&playerUUID, nil, nil, profile, &hexValue, nil, nil,
		)
		if err != nil {
			t.Fatalf("failed to build character sheet: %v", err)
		}
		charSheet.UUID = uuid.New()

		sheetMap := newTestSheetMap()
		repoErr := errors.New("update failed")
		mockRepo := &testutil.MockCharacterSheetRepo{
			UpdateNenHexagonValueFn: func(ctx context.Context, id string, val int) error {
				return repoErr
			},
		}

		uc := charactersheet.NewUpdateNenHexagonValueUC(sheetMap, mockRepo)

		_, err = uc.UpdateNenHexagonValue(ctx, charSheet, "increase")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
