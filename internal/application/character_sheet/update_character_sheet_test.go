package charactersheet_test

import (
	"context"
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	charactersheet "github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	csEntity "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

func TestUpdateCharacterSheet(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path - owner, no campaign, no submission", func(t *testing.T) {
		playerUUID := uuid.New()
		sheetUUID := uuid.New()

		updateCalled := false
		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{PlayerUUID: &playerUUID, CampaignUUID: nil}, nil
			},
			UpdateCharacterSheetFn: func(ctx context.Context, s *sheet.CharacterSheet) error {
				updateCalled = true
				return nil
			},
		}
		mockChecker := &mockFreeStateChecker{exists: false, err: nil}

		classMap := newTestClassMap()
		factory := newTestFactory()
		uc := charactersheet.NewUpdateCharacterSheetUC(classMap, factory, mockRepo, mockChecker)

		input := newValidCreateInput()
		input.PlayerUUID = &playerUUID

		err := uc.UpdateCharacterSheet(ctx, sheetUUID, playerUUID, input)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if !updateCalled {
			t.Error("expected UpdateCharacterSheet to be called on repo")
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

		classMap := newTestClassMap()
		factory := newTestFactory()
		uc := charactersheet.NewUpdateCharacterSheetUC(classMap, factory, mockRepo, mockChecker)

		input := newValidCreateInput()
		err := uc.UpdateCharacterSheet(ctx, sheetUUID, otherUser, input)
		if !errors.Is(err, auth.ErrInsufficientPermissions) {
			t.Fatalf("expected ErrInsufficientPermissions, got: %v", err)
		}
	})

	t.Run("error - has submission (not free)", func(t *testing.T) {
		playerUUID := uuid.New()
		sheetUUID := uuid.New()

		mockRepo := &testutil.MockCharacterSheetRepo{
			GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
				return csEntity.RelationshipUUIDs{PlayerUUID: &playerUUID, CampaignUUID: nil}, nil
			},
		}
		mockChecker := &mockFreeStateChecker{exists: true, err: nil}

		classMap := newTestClassMap()
		factory := newTestFactory()
		uc := charactersheet.NewUpdateCharacterSheetUC(classMap, factory, mockRepo, mockChecker)

		input := newValidCreateInput()
		err := uc.UpdateCharacterSheet(ctx, sheetUUID, playerUUID, input)
		if !errors.Is(err, charactersheet.ErrCharacterSheetNotFreeToManage) {
			t.Fatalf("expected ErrCharacterSheetNotFreeToManage, got: %v", err)
		}
	})
}

func TestUpdateCharacterSheet_DoesNotLeakChoicesIntoClass(t *testing.T) {
	ctx := context.Background()
	playerUUID := uuid.New()
	mockRepo := &testutil.MockCharacterSheetRepo{
		GetCharacterSheetRelationshipUUIDsFn: func(ctx context.Context, id uuid.UUID) (csEntity.RelationshipUUIDs, error) {
			return csEntity.RelationshipUUIDs{PlayerUUID: &playerUUID}, nil
		},
	}
	classMap := newTestClassMap()
	swordsman := cc.BuildSwordsman()
	swordsman.Distribution = &cc.Distribution{
		ProficiencyPoints:    []int{127},
		ProficienciesAllowed: []enum.WeaponName{enum.Dagger, enum.Katana},
	}
	swordsman.ProficienciesExps = map[enum.WeaponName]int{enum.ThrowingDagger: 127}
	classMap.Store(enum.Swordsman, swordsman)
	uc := charactersheet.NewUpdateCharacterSheetUC(
		classMap, newTestFactory(), mockRepo, &mockFreeStateChecker{},
	)

	for _, w := range []enum.WeaponName{enum.Dagger, enum.Katana} {
		input := newValidCreateInput()
		input.PlayerUUID = &playerUUID
		input.ProficienciesExps = map[enum.WeaponName]int{w: 127}
		if err := uc.UpdateCharacterSheet(ctx, uuid.New(), playerUUID, input); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	stored, _ := classMap.Load(enum.Swordsman)
	got := stored.(cc.CharacterClass).ProficienciesExps
	if len(got) != 1 {
		t.Errorf("class ProficienciesExps changed: %v", got)
	}
}
