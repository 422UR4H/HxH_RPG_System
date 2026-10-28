package charactersheet_test

import (
	"fmt"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/campaign"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	sheetEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	pgCampaign "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/campaign"
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
		if !errors.Is(err, campaign.ErrCampaignNotFound) {
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
		if !errors.Is(err, campaign.ErrNotCampaignOwner) {
			t.Errorf("expected ErrNotCampaignOwner, got: %v", err)
		}
	})

	t.Run("error - create repo error", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		repoErr := errors.New("insert failed")
		mockRepo := &testutil.MockCharacterSheetRepo{
			CreateCharacterSheetFn: func(ctx context.Context, s *sheetEntity.CharacterSheet) error {
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

	t.Run("error - too many physical attribute points", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		// Swordsman physLvl=3; sending 4 physical pts should fail
		input.AttributePoints = map[enum.AttributeName]int{
			enum.Resistance:  2,
			enum.Agility:     2,
		}

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error for over-distributed points, got nil")
		}
	})

	t.Run("error - incomplete physical attribute points", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput()
		// Swordsman physLvl=3; only 2 pts distributed should fail
		input.AttributePoints = map[enum.AttributeName]int{
			enum.Resistance: 1,
			enum.Agility:    1,
		}

		_, err := uc.CreateCharacterSheet(ctx, input)
		if err == nil {
			t.Fatal("expected error for incomplete distribution, got nil")
		}
	})

	t.Run("attribute points are saved on character sheet", func(t *testing.T) {
		classMap := newTestClassMap()
		sheetMap := newTestSheetMap()
		factory := newTestFactory()
		mockRepo := &testutil.MockCharacterSheetRepo{}
		mockCampaignRepo := &testutil.MockCampaignRepo{}

		uc := charactersheet.NewCreateCharacterSheetUC(
			classMap, sheetMap, factory, mockRepo, mockCampaignRepo,
		)
		input := newValidCreateInput() // already has valid 3-pt distribution

		result, err := uc.CreateCharacterSheet(ctx, input)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		pts, err := result.GetPointsOfAttribute(enum.Resistance)
		if err != nil {
			t.Fatalf("GetPointsOfAttribute error: %v", err)
		}
		if pts != 1 {
			t.Errorf("expected Resistance points=1, got %d", pts)
		}
	})
}

func TestCreateCharacterSheet_DoesNotLeakChoicesIntoClass(t *testing.T) {
	ctx := context.Background()
	classMap := newTestClassMap()
	// Swordsman nativo não tem distribuição; cria uma com proficiências livres.
	swordsman := cc.BuildSwordsman()
	swordsman.Distribution = &cc.Distribution{
		ProficiencyPoints:    []int{127},
		ProficienciesAllowed: []enum.WeaponName{enum.Dagger, enum.Katana, enum.Katar},
	}
	// Mapa não-nil, como na API real, onde o map é compartilhado entre cópias.
	swordsman.ProficienciesExps = map[enum.WeaponName]int{enum.ThrowingDagger: 127}
	classMap.Store(enum.Swordsman, swordsman)
	uc := charactersheet.NewCreateCharacterSheetUC(
		classMap, newTestSheetMap(), newTestFactory(),
		&testutil.MockCharacterSheetRepo{}, &testutil.MockCampaignRepo{},
	)

	before, _ := classMap.Load(enum.Swordsman)
	profsBefore := len(before.(cc.CharacterClass).ProficienciesExps)

	choices := []enum.WeaponName{enum.Katana, enum.Katar}
	for i, w := range choices {
		input := newValidCreateInput()
		input.Profile.NickName = fmt.Sprintf("Sword%d", i)
		input.ProficienciesExps = map[enum.WeaponName]int{w: 127}

		result, err := uc.CreateCharacterSheet(ctx, input)
		if err != nil {
			t.Fatalf("sheet %d: unexpected error: %v", i, err)
		}
		profs := result.GetCommonProficiencies()
		for _, other := range choices {
			_, has := profs[other]
			if other == w && !has {
				t.Errorf("sheet %d: missing chosen %v", i, other)
			}
			if other != w && has {
				t.Errorf("sheet %d: leaked %v from another sheet", i, other)
			}
		}
	}

	after, _ := classMap.Load(enum.Swordsman)
	if got := len(after.(cc.CharacterClass).ProficienciesExps); got != profsBefore {
		t.Errorf("class ProficienciesExps changed: %d -> %d", profsBefore, got)
	}
}
