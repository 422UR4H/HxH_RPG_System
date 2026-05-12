package charactersheet_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/auth"
	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/google/uuid"
)

func TestGetCharacterSheet(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - user is master", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		masterUUID := uuid.New()

		domainS := newValidDomainSheet(nil, &masterUUID, nil)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}
		mockSubLookup := &mockSubmissionLookup{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
		)

		result, err := uc.GetCharacterSheet(ctx, domainS.UUID, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected character sheet, got nil")
		}
		if result.UUID != domainS.UUID {
			t.Errorf("expected UUID %v, got %v", domainS.UUID, result.UUID)
		}
	})

	t.Run("happy path - user is player", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		playerUUID := uuid.New()

		domainS := newValidDomainSheet(&playerUUID, nil, nil)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}
		mockSubLookup := &mockSubmissionLookup{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
		)

		result, err := uc.GetCharacterSheet(ctx, domainS.UUID, playerUUID)
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

		domainS := newValidDomainSheet(&playerUUID, nil, &campaignUUID)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return campaignMaster, nil
			},
		}
		mockSubLookup := &mockSubmissionLookup{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
		)

		result, err := uc.GetCharacterSheet(ctx, domainS.UUID, campaignMaster)
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return nil, false, charactersheet.ErrCharacterSheetNotFound
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}
		mockSubLookup := &mockSubmissionLookup{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return nil, false, repoErr
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}
		mockSubLookup := &mockSubmissionLookup{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
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

		domainS := newValidDomainSheet(&playerUUID, nil, nil)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}
		mockSubLookup := &mockSubmissionLookup{
			err: errors.New("not found"),
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
		)

		_, err := uc.GetCharacterSheet(ctx, domainS.UUID, unrelatedUser)
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

		domainS := newValidDomainSheet(&playerUUID, nil, &campaignUUID)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return differentCampaignMaster, nil
			},
		}
		mockSubLookup := &mockSubmissionLookup{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
		)

		_, err := uc.GetCharacterSheet(ctx, domainS.UUID, unrelatedUser)
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

		domainS := newValidDomainSheet(&playerUUID, nil, &campaignUUID)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, pgCampaign.ErrCampaignNotFound
			},
		}
		mockSubLookup := &mockSubmissionLookup{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
		)

		_, err := uc.GetCharacterSheet(ctx, domainS.UUID, unrelatedUser)
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

		domainS := newValidDomainSheet(&playerUUID, nil, &campaignUUID)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, repoErr
			},
		}
		mockSubLookup := &mockSubmissionLookup{}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
		)

		_, err := uc.GetCharacterSheet(ctx, domainS.UUID, unrelatedUser)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Errorf("expected repo error, got: %v", err)
		}
	})

	t.Run("happy path - master can view pending sheet via submission lookup", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		playerUUID := uuid.New()
		masterUUID := uuid.New()
		campaignUUID := uuid.New()

		// Sheet has no campaign_uuid (pending submission, not yet accepted)
		domainS := newValidDomainSheet(&playerUUID, nil, nil)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return masterUUID, nil
			},
		}
		mockSubLookup := &mockSubmissionLookup{
			campaignUUID: campaignUUID,
			err:          nil,
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
		)

		result, err := uc.GetCharacterSheet(ctx, domainS.UUID, masterUUID)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected character sheet, got nil")
		}
		if result.UUID != domainS.UUID {
			t.Errorf("expected UUID %v, got %v", domainS.UUID, result.UUID)
		}
	})

	t.Run("error - master cannot view sheet with no pending submission", func(t *testing.T) {
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		playerUUID := uuid.New()
		masterUUID := uuid.New()

		// Sheet has no campaign_uuid and no pending submission
		domainS := newValidDomainSheet(&playerUUID, nil, nil)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*domainSheet.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}
		mockSubLookup := &mockSubmissionLookup{
			campaignUUID: uuid.Nil,
			err:          errors.New("not found"),
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			sheetMap, factory, mockRepo, mockCampaignRepo, mockSubLookup,
		)

		_, err := uc.GetCharacterSheet(ctx, domainS.UUID, masterUUID)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, auth.ErrInsufficientPermissions) {
			t.Errorf("expected ErrInsufficientPermissions, got: %v", err)
		}
	})
}

// mockSubmissionLookup implements charactersheet.ISubmissionLookup for testing.
type mockSubmissionLookup struct {
	campaignUUID uuid.UUID
	err          error
}

func (m *mockSubmissionLookup) GetSubmissionCampaignUUIDBySheetUUID(
	ctx context.Context, sheetUUID uuid.UUID,
) (uuid.UUID, error) {
	return m.campaignUUID, m.err
}
