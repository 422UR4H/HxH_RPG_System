package charactersheet_test

import (
	"context"
	"errors"
	"testing"

	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

func TestListCharacterSheets(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - returns list", func(t *testing.T) {
		expected := []model.CharacterSheetSummary{
			{UUID: uuid.New(), NickName: "Gon"},
			{UUID: uuid.New(), NickName: "Killua"},
		}
		mockRepo := &testutil.MockCharacterSheetRepo{
			ListCharacterSheetsByPlayerUUIDFn: func(ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error) {
				return expected, nil
			},
		}

		uc := charactersheet.NewListCharacterSheetsUC(mockRepo)
		playerUUID := uuid.New()

		result, err := uc.ListCharacterSheets(ctx, playerUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 results, got %d", len(result))
		}
		if result[0].NickName != "Gon" {
			t.Errorf("expected first NickName 'Gon', got %q", result[0].NickName)
		}
	})

	t.Run("happy path - empty list", func(t *testing.T) {
		mockRepo := &testutil.MockCharacterSheetRepo{
			ListCharacterSheetsByPlayerUUIDFn: func(ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error) {
				return []model.CharacterSheetSummary{}, nil
			},
		}

		uc := charactersheet.NewListCharacterSheetsUC(mockRepo)
		playerUUID := uuid.New()

		result, err := uc.ListCharacterSheets(ctx, playerUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if len(result) != 0 {
			t.Errorf("expected empty list, got %d items", len(result))
		}
	})

	t.Run("error - repo error", func(t *testing.T) {
		repoErr := errors.New("database error")
		mockRepo := &testutil.MockCharacterSheetRepo{
			ListCharacterSheetsByPlayerUUIDFn: func(ctx context.Context, playerUUID string) ([]model.CharacterSheetSummary, error) {
				return nil, repoErr
			},
		}

		uc := charactersheet.NewListCharacterSheetsUC(mockRepo)
		playerUUID := uuid.New()

		_, err := uc.ListCharacterSheets(ctx, playerUUID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Errorf("expected repo error, got: %v", err)
		}
	})
}
