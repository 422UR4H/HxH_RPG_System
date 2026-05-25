package charactersheet_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	sheetEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
		if !errors.Is(err, campaign.ErrCampaignNotFound) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
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

func TestGetCharacterSheet_checkAndNormalize(t *testing.T) {
	ctx := context.Background()

	t.Run("UpdateCharExp not called when sheet has no exp", func(t *testing.T) {
		playerUUID := uuid.New()
		domainS := newValidDomainSheet(&playerUUID, nil, nil)
		// fresh sheet: GetExpPoints() == 0

		updateCharExpCalled := false
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
			UpdateCharExpFn: func(ctx context.Context, sheetUUID string, charExp int) error {
				updateCharExpCalled = true
				return nil
			},
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			newTestSheetMap(), newTestFactory(), mockRepo, &testutil.MockCampaignRepo{}, &mockSubmissionLookup{},
		)

		if _, err := uc.GetCharacterSheet(ctx, domainS.UUID, playerUUID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updateCharExpCalled {
			t.Error("UpdateCharExp should not be called when GetExpPoints() == 0")
		}
	})

	t.Run("UpdateCharExp called async with correct value when sheet has exp", func(t *testing.T) {
		playerUUID := uuid.New()
		domainS := newValidDomainSheet(&playerUUID, nil, nil)
		if err := domainS.IncreaseExpForSkill(experience.NewUpgradeCascade(500), enum.Vitality); err != nil {
			t.Fatalf("failed to add exp to sheet: %v", err)
		}
		expectedExp := domainS.GetExpPoints()
		if expectedExp == 0 {
			t.Fatal("test setup error: expected GetExpPoints() > 0 after IncreaseExpForSkill")
		}

		type captureArgs struct {
			uuid string
			exp  int
		}
		captured := make(chan captureArgs, 1)
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
				return domainS, false, nil
			},
			UpdateCharExpFn: func(ctx context.Context, sheetUUID string, charExp int) error {
				captured <- captureArgs{uuid: sheetUUID, exp: charExp}
				return nil
			},
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			newTestSheetMap(), newTestFactory(), mockRepo, &testutil.MockCampaignRepo{}, &mockSubmissionLookup{},
		)

		if _, err := uc.GetCharacterSheet(ctx, domainS.UUID, playerUUID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		select {
		case args := <-captured:
			if args.exp != expectedExp {
				t.Errorf("UpdateCharExp called with exp=%d, want %d", args.exp, expectedExp)
			}
			if args.uuid != domainS.UUID.String() {
				t.Errorf("UpdateCharExp called with uuid=%s, want %s", args.uuid, domainS.UUID.String())
			}
		case <-time.After(500 * time.Millisecond):
			t.Error("UpdateCharExp was not called within timeout when GetExpPoints() > 0")
		}
	})

	t.Run("UpdateStatusBars called async when wasCorrected", func(t *testing.T) {
		playerUUID := uuid.New()
		domainS := newValidDomainSheet(&playerUUID, nil, nil)

		done := make(chan struct{})
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
				return domainS, true, nil // wasCorrected = true
			},
			UpdateStatusBarsFn: func(ctx context.Context, id string, health, stamina, aura status.IStatusBar) error {
				close(done)
				return nil
			},
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			newTestSheetMap(), newTestFactory(), mockRepo, &testutil.MockCampaignRepo{}, &mockSubmissionLookup{},
		)

		if _, err := uc.GetCharacterSheet(ctx, domainS.UUID, playerUUID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		select {
		case <-done:
		case <-time.After(500 * time.Millisecond):
			t.Error("UpdateStatusBars was not called within timeout when wasCorrected=true")
		}
	})

	t.Run("UpdateStatusBars not called when not wasCorrected", func(t *testing.T) {
		playerUUID := uuid.New()
		domainS := newValidDomainSheet(&playerUUID, nil, nil)

		updateStatusBarsCalled := false
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetByUUIDFn: func(ctx context.Context, id string) (*sheetEntity.CharacterSheet, bool, error) {
				return domainS, false, nil // wasCorrected = false
			},
			UpdateStatusBarsFn: func(ctx context.Context, id string, health, stamina, aura status.IStatusBar) error {
				updateStatusBarsCalled = true
				return nil
			},
		}

		uc := charactersheet.NewGetCharacterSheetUC(
			newTestSheetMap(), newTestFactory(), mockRepo, &testutil.MockCampaignRepo{}, &mockSubmissionLookup{},
		)

		if _, err := uc.GetCharacterSheet(ctx, domainS.UUID, playerUUID); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// No goroutine is launched when wasCorrected=false; no need to wait.
		if updateStatusBarsCalled {
			t.Error("UpdateStatusBars should not be called when wasCorrected=false")
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
