package sheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

func buildTestSheet(t *testing.T) *sheet.CharacterSheet {
	t.Helper()

	factory := sheet.NewCharacterSheetFactory()
	profile := sheet.CharacterProfile{
		NickName:         "Gon",
		FullName:         "Gon Freecss",
		Alignment:        "Chaotic-Good",
		BriefDescription: "A young hunter",
		Age:              12,
	}

	playerUUID := uuid.New()
	campaignUUID := uuid.New()

	cs, err := factory.Build(
		&playerUUID, nil, &campaignUUID,
		profile, nil, nil, nil,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

func TestCharacterSheet_InitialState(t *testing.T) {
	cs := buildTestSheet(t)

	if cs.GetLevel() != 0 {
		t.Errorf("initial character level = %d, want 0", cs.GetLevel())
	}
	if cs.GetCharacterPoints() != 0 {
		t.Errorf("initial character points = %d, want 0", cs.GetCharacterPoints())
	}
}

func TestCharacterSheet_IncreaseExpForSkill_Cascade(t *testing.T) {
	cs := buildTestSheet(t)

	initialCharExp := cs.GetExpPoints()

	err := cs.IncreaseExpForSkill(experience.NewUpgradeCascade(500), enum.Vitality)
	if err != nil {
		t.Fatalf("IncreaseExpForSkill error: %v", err)
	}

	if cs.GetExpPoints() <= initialCharExp {
		t.Errorf("character exp should increase after cascade: was %d, now %d",
			initialCharExp, cs.GetExpPoints())
	}
}

func TestCharacterSheet_StatusUpgradeAfterExpIncrease(t *testing.T) {
	cs := buildTestSheet(t)

	hpBefore, err := cs.GetMaxOfStatus(enum.Health)
	if err != nil {
		t.Fatalf("GetMaxOfStatus error: %v", err)
	}

	cs.IncreaseExpForSkill(experience.NewUpgradeCascade(50000), enum.Vitality)

	hpAfter, err := cs.GetMaxOfStatus(enum.Health)
	if err != nil {
		t.Fatalf("GetMaxOfStatus error: %v", err)
	}

	if hpAfter <= hpBefore {
		t.Errorf("HP should increase after massive XP gain: before=%d, after=%d",
			hpBefore, hpAfter)
	}
}

func TestCharacterSheet_GetValueForTestOfSkill(t *testing.T) {
	cs := buildTestSheet(t)

	val, err := cs.GetValueForTestOfSkill(enum.Vitality)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if val < 0 {
		t.Errorf("value for test should be >= 0, got %d", val)
	}
}

func TestCharacterSheet_SetCurrStatus(t *testing.T) {
	cs := buildTestSheet(t)

	maxHP, _ := cs.GetMaxOfStatus(enum.Health)
	if maxHP == 0 {
		t.Skip("max HP is 0, cannot test SetCurrStatus")
	}

	err := cs.SetCurrStatus(enum.Health, maxHP-1)
	if err != nil {
		t.Fatalf("SetCurrStatus error: %v", err)
	}
}

func TestCharacterSheet_NenHexagonNotInitialized(t *testing.T) {
	cs := buildTestSheet(t)

	hexVal := cs.GetCurrHexValue()
	if hexVal != nil {
		t.Errorf("hex value should be nil when not initialized, got %d", *hexVal)
	}
}

func TestCharacterSheet_OwnerValidation(t *testing.T) {
	factory := sheet.NewCharacterSheetFactory()
	profile := sheet.CharacterProfile{
		NickName: "Gon",
		FullName: "Gon Freecss",
		Age:      12,
	}

	t.Run("both nil should fail", func(t *testing.T) {
		_, err := factory.Build(nil, nil, nil, profile, nil, nil, nil)
		if err == nil {
			t.Error("should reject when both player and master are nil")
		}
	})

	t.Run("both non-nil should fail", func(t *testing.T) {
		p := uuid.New()
		m := uuid.New()
		_, err := factory.Build(&p, &m, nil, profile, nil, nil, nil)
		if err == nil {
			t.Error("should reject when both player and master are non-nil")
		}
	})

	t.Run("player only valid", func(t *testing.T) {
		p := uuid.New()
		_, err := factory.Build(&p, nil, nil, profile, nil, nil, nil)
		if err != nil {
			t.Errorf("player-only should be valid: %v", err)
		}
	})

	t.Run("master only valid", func(t *testing.T) {
		m := uuid.New()
		_, err := factory.Build(nil, &m, nil, profile, nil, nil, nil)
		if err != nil {
			t.Errorf("master-only should be valid: %v", err)
		}
	})
}
