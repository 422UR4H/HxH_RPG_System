package charactersheet_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	pgSheet "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	"github.com/google/uuid"
)

func TestGetCharacterSheet(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - user is master", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		masterUUID := uuid.New()

		modelSheet := newValidModelSheet(nil, &masterUUID, nil)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return modelSheet, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo,
		)

		result, err := uc.GetCharacterSheet(ctx, modelSheet.UUID, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected character sheet, got nil")
		}
		if result.UUID != modelSheet.UUID {
			t.Errorf("expected UUID %v, got %v", modelSheet.UUID, result.UUID)
		}
	})

	t.Run("happy path - user is player", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		playerUUID := uuid.New()

		modelSheet := newValidModelSheet(&playerUUID, nil, nil)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return modelSheet, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo,
		)

		result, err := uc.GetCharacterSheet(ctx, modelSheet.UUID, playerUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected character sheet, got nil")
		}
	})

	t.Run("happy path - user is campaign master", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		playerUUID := uuid.New()
		campaignMaster := uuid.New()
		campaignUUID := uuid.New()

		modelSheet := newValidModelSheet(&playerUUID, nil, &campaignUUID)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return modelSheet, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return campaignMaster, nil
			},
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo,
		)

		result, err := uc.GetCharacterSheet(ctx, modelSheet.UUID, campaignMaster)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected character sheet, got nil")
		}
	})

	t.Run("error - sheet not found", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return nil, pgSheet.ErrCharacterSheetNotFound
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo,
		)

		_, err := uc.GetCharacterSheet(ctx, uuid.New(), uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFound) {
			t.Errorf("expected ErrCharacterSheetNotFound, got: %v", err)
		}
	})

	t.Run("error - repo error", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		repoErr := errors.New("database connection failed")
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return nil, repoErr
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo,
		)

		_, err := uc.GetCharacterSheet(ctx, uuid.New(), uuid.New())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Errorf("expected repo error, got: %v", err)
		}
	})

	t.Run("error - insufficient permissions (no campaign)", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		playerUUID := uuid.New()
		unrelatedUser := uuid.New()

		modelSheet := newValidModelSheet(&playerUUID, nil, nil)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return modelSheet, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo,
		)

		_, err := uc.GetCharacterSheet(ctx, modelSheet.UUID, unrelatedUser)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, auth.ErrInsufficientPermissions) {
			t.Errorf("expected ErrInsufficientPermissions, got: %v", err)
		}
	})

	t.Run("error - insufficient permissions (not campaign master)", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		playerUUID := uuid.New()
		campaignUUID := uuid.New()
		unrelatedUser := uuid.New()
		differentCampaignMaster := uuid.New()

		modelSheet := newValidModelSheet(&playerUUID, nil, &campaignUUID)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return modelSheet, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return differentCampaignMaster, nil
			},
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo,
		)

		_, err := uc.GetCharacterSheet(ctx, modelSheet.UUID, unrelatedUser)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, auth.ErrInsufficientPermissions) {
			t.Errorf("expected ErrInsufficientPermissions, got: %v", err)
		}
	})

	t.Run("error - campaign not found during permission check", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		playerUUID := uuid.New()
		campaignUUID := uuid.New()
		unrelatedUser := uuid.New()

		modelSheet := newValidModelSheet(&playerUUID, nil, &campaignUUID)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return modelSheet, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, pgCampaign.ErrCampaignNotFound
			},
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo,
		)

		_, err := uc.GetCharacterSheet(ctx, modelSheet.UUID, unrelatedUser)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, domainCampaign.ErrCampaignNotFound) {
			t.Errorf("expected ErrCampaignNotFound, got: %v", err)
		}
	})

	t.Run("error - campaign repo error during permission check", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		playerUUID := uuid.New()
		campaignUUID := uuid.New()
		unrelatedUser := uuid.New()
		repoErr := errors.New("campaign db error")

		modelSheet := newValidModelSheet(&playerUUID, nil, &campaignUUID)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return modelSheet, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, repoErr
			},
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo,
		)

		_, err := uc.GetCharacterSheet(ctx, modelSheet.UUID, unrelatedUser)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Errorf("expected repo error, got: %v", err)
		}
	})

	t.Run("triggers async persist when status curr exceeds new max", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		masterUUID := uuid.New()

		modelSheet := newValidModelSheet(nil, &masterUUID, nil)
		// HP_BASE_VALUE = 20 for a base sheet with no XP.
		// curr=25, max=30 → normalizeStatus computes round(20*25/30)=17
		modelSheet.Health.Curr = 25
		modelSheet.Health.Max = 30

		done := make(chan struct{})
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*model.CharacterSheet, error) {
				return modelSheet, nil
			},
			UpdateStatusBarsFn: func(ctx context.Context, uuid string, health, stamina, aura model.StatusBar) error {
				close(done)
				return nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewGetCharacterSheetUC(sheetMap, factory, mockRepo, mockCampaignRepo)
		result, err := uc.GetCharacterSheet(ctx, modelSheet.UUID, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected character sheet, got nil")
		}

		select {
		case <-done:
		case <-time.After(100 * time.Millisecond):
			t.Error("expected UpdateStatusBars to be called within 100ms")
		}
	})
}
