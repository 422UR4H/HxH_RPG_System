package sheet_test

import (
	"errors"
	"testing"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

func buildSheetWithClass(t *testing.T, charClass cc.CharacterClass) *sheet.CharacterSheet {
	t.Helper()
	factory := sheet.NewCharacterSheetFactory()
	playerUUID := uuid.New()
	profile := sheet.CharacterProfile{
		NickName: "Gon", FullName: "Gon Freecss", Alignment: "Chaotic-Good",
	}
	cs, err := factory.Build(&playerUUID, nil, nil, profile, nil, nil, &charClass)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

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

	if err := cs.IncreaseExpForSkill(experience.NewUpgradeCascade(50000), enum.Vitality); err != nil {
		t.Fatal(err)
	}

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

func TestCharacterSheet_ApplyInitialAttributePoints(t *testing.T) {
	// Swordsman: physLvl=3, mentLvl=0
	// Use Swordsman for tests that need physLvl > 0.
	// Use no-class sheet for tests where level=0 suffices.

	t.Run("empty map valid when both levels are zero", func(t *testing.T) {
		cs := buildTestSheet(t)
		err := cs.ApplyInitialAttributePoints(map[enum.AttributeName]int{})
		if err != nil {
			t.Errorf("empty map should be valid when levels are 0: %v", err)
		}
	})

	t.Run("physical points exceed physical level", func(t *testing.T) {
		cs := buildTestSheet(t) // physLvl=0
		err := cs.ApplyInitialAttributePoints(map[enum.AttributeName]int{
			enum.Resistance: 1,
		})
		if !errors.Is(err, sheet.ErrInvalidDistributionPoints) {
			t.Errorf("expected ErrInvalidDistributionPoints, got: %v", err)
		}
	})

	t.Run("mental points exceed mental level", func(t *testing.T) {
		cs := buildTestSheet(t) // mentLvl=0
		err := cs.ApplyInitialAttributePoints(map[enum.AttributeName]int{
			enum.Resilience: 1,
		})
		if !errors.Is(err, sheet.ErrInvalidDistributionPoints) {
			t.Errorf("expected ErrInvalidDistributionPoints, got: %v", err)
		}
	})

	t.Run("physical points fewer than physical level (incomplete)", func(t *testing.T) {
		cs := buildSheetWithClass(t, cc.BuildSwordsman()) // physLvl=3
		err := cs.ApplyInitialAttributePoints(map[enum.AttributeName]int{
			enum.Resistance: 2,
		})
		if !errors.Is(err, sheet.ErrInvalidDistributionPoints) {
			t.Errorf("expected ErrInvalidDistributionPoints for incomplete distribution, got: %v", err)
		}
	})

	t.Run("zero physical points when level is nonzero (all incomplete)", func(t *testing.T) {
		cs := buildSheetWithClass(t, cc.BuildSwordsman()) // physLvl=3
		err := cs.ApplyInitialAttributePoints(map[enum.AttributeName]int{})
		if !errors.Is(err, sheet.ErrInvalidDistributionPoints) {
			t.Errorf("expected ErrInvalidDistributionPoints for zero points with nonzero level, got: %v", err)
		}
	})

	t.Run("unknown (spiritual) attribute name", func(t *testing.T) {
		cs := buildTestSheet(t)
		err := cs.ApplyInitialAttributePoints(map[enum.AttributeName]int{
			enum.Flame: 1,
		})
		if err == nil {
			t.Error("expected error for spiritual attribute name")
		}
	})

	t.Run("exact physical distribution succeeds", func(t *testing.T) {
		// Swordsman physLvl=3; primary physicals: Resistance, Agility, Flexibility, Sense
		cs := buildSheetWithClass(t, cc.BuildSwordsman())
		err := cs.ApplyInitialAttributePoints(map[enum.AttributeName]int{
			enum.Resistance:  1,
			enum.Agility:     1,
			enum.Flexibility: 1,
		})
		if err != nil {
			t.Errorf("exact primary distribution should succeed: %v", err)
		}

		pts, err := cs.GetPointsOfAttribute(enum.Resistance)
		if err != nil {
			t.Fatalf("GetPointsOfAttribute error: %v", err)
		}
		if pts != 1 {
			t.Errorf("expected Resistance points=1, got %d", pts)
		}
	})

	t.Run("negative points are ignored, treated as zero", func(t *testing.T) {
		cs := buildTestSheet(t) // physLvl=0, mentLvl=0
		err := cs.ApplyInitialAttributePoints(map[enum.AttributeName]int{
			enum.Resistance: -1,
		})
		if err != nil {
			t.Errorf("negative points should be filtered (zero level = zero required): %v", err)
		}
	})
}

func TestCharacterSheet_ReconstructPrimaryMentalPoints(t *testing.T) {
	t.Run("restores mental points even when mentLvl is zero (no class applied)", func(t *testing.T) {
		cs := buildTestSheet(t) // mentLvl=0

		err := cs.ReconstructPrimaryMentalPoints(enum.Resilience, 2)
		if err != nil {
			t.Errorf("expected no error restoring Resilience=2, got: %v", err)
		}

		pts, err := cs.GetPointsOfAttribute(enum.Resilience)
		if err != nil {
			t.Fatalf("GetPointsOfAttribute: %v", err)
		}
		if pts != 2 {
			t.Errorf("expected Resilience points=2, got %d", pts)
		}
	})
}

func TestCharacterSheet_ReconstructPrimaryPhysicalPoints(t *testing.T) {
	t.Run("restores primary points even when physLvl is zero (no class applied)", func(t *testing.T) {
		// Gateway reconstruction: sheet built without a class (AddDryCharacterClass
		// sets only the name, not ability levels), so physLvl=0. The capacity check
		// in IncreasePtsForPhysPrimaryAttr would reject any points > 0.
		cs := buildTestSheet(t) // physLvl=0

		err := cs.ReconstructPrimaryPhysicalPoints(enum.Resistance, 1)
		if err != nil {
			t.Errorf("expected no error restoring Resistance=1, got: %v", err)
		}

		pts, err := cs.GetPointsOfAttribute(enum.Resistance)
		if err != nil {
			t.Fatalf("GetPointsOfAttribute: %v", err)
		}
		if pts != 1 {
			t.Errorf("expected Resistance points=1, got %d", pts)
		}
	})

	t.Run("updates derived middle attributes after primary restore", func(t *testing.T) {
		cs := buildTestSheet(t)

		if err := cs.ReconstructPrimaryPhysicalPoints(enum.Agility, 2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := cs.ReconstructPrimaryPhysicalPoints(enum.Flexibility, 2); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Celerity = (Agility + Flexibility) / 2 = (2+2)/2 = 2
		pts, err := cs.GetPointsOfAttribute(enum.Celerity)
		if err != nil {
			t.Fatalf("GetPointsOfAttribute Celerity: %v", err)
		}
		if pts != 2 {
			t.Errorf("expected Celerity points=2, got %d", pts)
		}
	})
}
