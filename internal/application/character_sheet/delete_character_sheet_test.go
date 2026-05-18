package charactersheet_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/google/uuid"
)

type mockFreeStateChecker struct {
	exists bool
	err    error
}

func (m *mockFreeStateChecker) ExistsSubmittedCharacterSheet(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.exists, m.err
}

func TestDeleteCharacterSheet(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - free sheet deleted", func(t *testing.T) {
		playerUUID := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{CampaignUUID: nil}, nil
			},
			DeleteCharacterSheetFn: func(ctx context.Context, sUUID uuid.UUID, pUUID uuid.UUID) error {
				return nil
			},
		}
		mockChecker := &mockFreeStateChecker{exists: false, err: nil}

		uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
		err := uc.DeleteCharacterSheet(ctx, sheetUUID, playerUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("error - not owner", func(t *testing.T) {
		playerUUID := uuid.New()
		otherUser := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
		}
		mockChecker := &mockFreeStateChecker{}

		uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
		err := uc.DeleteCharacterSheet(ctx, sheetUUID, otherUser)
		if !errors.Is(err, auth.ErrInsufficientPermissions) {
			t.Fatalf("expected ErrInsufficientPermissions, got: %v", err)
		}
	})

	t.Run("error - sheet has submission (not free)", func(t *testing.T) {
		playerUUID := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{CampaignUUID: nil}, nil
			},
		}
		mockChecker := &mockFreeStateChecker{exists: true, err: nil}

		uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
		err := uc.DeleteCharacterSheet(ctx, sheetUUID, playerUUID)
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFreeToManage) {
			t.Fatalf("expected ErrCharacterSheetNotFreeToManage, got: %v", err)
		}
	})

	t.Run("error - sheet not found", func(t *testing.T) {
		sheetUUID := uuid.New()
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, charactersheet.ErrCharacterSheetNotFound
			},
		}
		mockChecker := &mockFreeStateChecker{}

		uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
		err := uc.DeleteCharacterSheet(ctx, sheetUUID, uuid.New())
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
			t.Fatalf("expected ErrCharacterSheetNotFound, got: %v", err)
		}
	})

	t.Run("error - sheet has campaign (not free)", func(t *testing.T) {
		playerUUID := uuid.New()
		campaignUUID := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetPlayerUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return playerUUID, nil
			},
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{CampaignUUID: &campaignUUID}, nil
			},
		}
		mockChecker := &mockFreeStateChecker{}

		uc := charactersheet.NewDeleteCharacterSheetUC(mockRepo, mockChecker)
		err := uc.DeleteCharacterSheet(ctx, sheetUUID, playerUUID)
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFreeToManage) {
			t.Fatalf("expected ErrCharacterSheetNotFreeToManage, got: %v", err)
		}
	})
}
