package charactersheet_test

import (
	"context"
	"errors"
	"testing"

	domainCampaign "github.com/422UR4H/HxH_RPG_System/internal/domain/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/domain/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/testutil"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
	"github.com/google/uuid"
)

func TestCreateCharacterSheet(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - player without campaign", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()

		result, err := uc.CreateCharacterSheet(ctx, input)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected character sheet, got nil")
		}
		if result.UUID == uuid.Nil {
			t.Error("expected generated UUID, got nil UUID")
		}
	})

	t.Run("happy path - master without campaign", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		masterUUID := uuid.New()
		input.PlayerUUID = nil
		input.MasterUUID = &masterUUID

		result, err := uc.CreateCharacterSheet(ctx, input)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected character sheet, got nil")
		}
	})

	t.Run("happy path - player with valid campaign", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()

		masterUUID := uuid.New()
		campaignUUID := uuid.New()
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return masterUUID, nil
			},
		}
		mockRepo := &testutil.MockCharacterSheetRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		input.CampaignUUID = &campaignUUID
		input.MasterUUID = &masterUUID
		input.PlayerUUID = nil

		result, err := uc.CreateCharacterSheet(ctx, input)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if result == nil {
			t.Fatal("expected character sheet, got nil")
		}
	})

	t.Run("error - class not found", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		input.CharacterClass = enum.CharacterClassName("NonExistent")

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, charactersheet.ErrCharacterClassNotFound) {
			t.Errorf("expected ErrCharacterClassNotFound, got: %v", err)
		}
	})

	t.Run("error - invalid skills distribution", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		// Swordsman has nil Distribution, so any non-empty SkillsExps should fail
		input.SkillsExps = map[enum.SkillName]int{
			enum.Vitality: 100,
		}

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error for invalid skills, got nil")
		}
	})

	t.Run("error - invalid proficiencies distribution", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		// Swordsman has nil Distribution, so any non-empty ProficienciesExps should fail
		input.ProficienciesExps = map[enum.WeaponName]int{
			enum.Katana: 100,
		}

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error for invalid proficiencies, got nil")
		}
	})

	t.Run("error - nickname matches class name", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		input.Profile.NickName = "Swordsman"
		input.Profile.FullName = "Swordsman Warrior"

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error for nickname matching class name, got nil")
		}
		if !errors.Is(err, charactersheet.ErrNicknameNotAllowed) {
			t.Errorf("expected ErrNicknameNotAllowed, got: %v", err)
		}
	})

	t.Run("error - player limit exceeded", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{
			CountCharactersByPlayerUUIDFn: func(ctx context.Context, playerUUID uuid.UUID) (int, error) {
				return 20, nil
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error for player limit, got nil")
		}
		if !errors.Is(err, charactersheet.ErrMaxCharacterSheetsLimit) {
			t.Errorf("expected ErrMaxCharacterSheetsLimit, got: %v", err)
		}
	})

	t.Run("error - count characters repo error", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		repoErr := errors.New("database connection failed")
		mockRepo := &testutil.MockCharacterSheetRepo{
			CountCharactersByPlayerUUIDFn: func(ctx context.Context, playerUUID uuid.UUID) (int, error) {
				return 0, repoErr
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Errorf("expected repo error, got: %v", err)
		}
	})

	t.Run("error - campaign not found", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, pgCampaign.ErrCampaignNotFound
			},
		}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		campaignUUID := uuid.New()
		masterUUID := uuid.New()
		input.CampaignUUID = &campaignUUID
		input.MasterUUID = &masterUUID
		input.PlayerUUID = nil

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, domainCampaign.ErrCampaignNotFound) {
			t.Errorf("expected ErrCampaignNotFound, got: %v", err)
		}
	})

	t.Run("error - campaign repo error", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		repoErr := errors.New("campaign db error")
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return uuid.Nil, repoErr
			},
		}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		campaignUUID := uuid.New()
		masterUUID := uuid.New()
		input.CampaignUUID = &campaignUUID
		input.MasterUUID = &masterUUID
		input.PlayerUUID = nil

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Errorf("expected campaign repo error, got: %v", err)
		}
	})

	t.Run("error - not campaign owner", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		differentMaster := uuid.New()
		mockCampaignRepo := &testutil.MockCampaignRepo{
			GetCampaignMasterUUIDFn: func(ctx context.Context, id uuid.UUID) (uuid.UUID, error) {
				return differentMaster, nil
			},
		}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		campaignUUID := uuid.New()
		masterUUID := uuid.New()
		input.CampaignUUID = &campaignUUID
		input.MasterUUID = &masterUUID
		input.PlayerUUID = nil

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, domainCampaign.ErrNotCampaignOwner) {
			t.Errorf("expected ErrNotCampaignOwner, got: %v", err)
		}
	})

	t.Run("error - create repo error", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		repoErr := errors.New("insert failed")
		mockRepo := &testutil.MockCharacterSheetRepo{
			CreateCharacterSheetFn: func(ctx context.Context, s *model.CharacterSheet) error {
				return repoErr
			},
		}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, repoErr) {
			t.Errorf("expected repo error, got: %v", err)
		}
	})
}
