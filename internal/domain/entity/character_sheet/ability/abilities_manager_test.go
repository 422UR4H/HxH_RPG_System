package ability_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestAbilitiesManager() *ability.Manager {
	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilities := make(map[enum.AbilityName]ability.IAbility)

	talentExp := experience.NewExperience(experience.NewExpTable(2.0))
	talent := ability.NewTalent(*talentExp)

	physicalExp := experience.NewExperience(experience.NewExpTable(20.0))
	abilities[enum.Physicals] = ability.NewAbility(enum.Physicals, *physicalExp, charExp)

	mentalExp := experience.NewExperience(experience.NewExpTable(20.0))
	abilities[enum.Mentals] = ability.NewAbility(enum.Mentals, *mentalExp, charExp)

	spiritualExp := experience.NewExperience(experience.NewExpTable(5.0))
	abilities[enum.Spirituals] = ability.NewAbility(enum.Spirituals, *spiritualExp, charExp)

	skillsExp := experience.NewExperience(experience.NewExpTable(20.0))
	abilities[enum.Skills] = ability.NewAbility(enum.Skills, *skillsExp, charExp)

	return ability.NewAbilitiesManager(charExp, abilities, *talent)
}

func TestAbilitiesManager_Get_Found(t *testing.T) {
	mgr := newTestAbilitiesManager()

	tests := []enum.AbilityName{enum.Physicals, enum.Mentals, enum.Spirituals, enum.Skills}
	for _, name := range tests {
		t.Run(string(name), func(t *testing.T) {
			a, err := mgr.Get(name)
			if err != nil {
				t.Fatalf("Get(%s) unexpected error: %v", name, err)
			}
			if a == nil {
				t.Fatalf("Get(%s) returned nil", name)
			}
		})
	}
}

func TestAbilitiesManager_Get_NotFound(t *testing.T) {
	mgr := newTestAbilitiesManager()

	_, err := mgr.Get("NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent ability, got nil")
	}
	if !errors.Is(err, ability.ErrAbilityNotFound) {
		t.Errorf("expected ErrAbilityNotFound, got %v", err)
	}
}

func TestAbilitiesManager_GetLevelOf(t *testing.T) {
	mgr := newTestAbilitiesManager()

	lvl, err := mgr.GetLevelOf(enum.Physicals)
	if err != nil {
		t.Fatalf("GetLevelOf unexpected error: %v", err)
	}
	if lvl != 0 {
		t.Errorf("initial level: got %d, want 0", lvl)
	}
}

func TestAbilitiesManager_GetLevelOf_NotFound(t *testing.T) {
	mgr := newTestAbilitiesManager()

	_, err := mgr.GetLevelOf("NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent ability")
	}
}

func TestAbilitiesManager_GetExpReferenceOf(t *testing.T) {
	mgr := newTestAbilitiesManager()

	ref, err := mgr.GetExpReferenceOf(enum.Physicals)
	if err != nil {
		t.Fatalf("GetExpReferenceOf unexpected error: %v", err)
	}
	if ref == nil {
		t.Fatal("GetExpReferenceOf returned nil")
	}
}

func TestAbilitiesManager_CharacterLevel(t *testing.T) {
	mgr := newTestAbilitiesManager()

	if mgr.GetCharacterLevel() != 0 {
		t.Errorf("initial character level: got %d, want 0", mgr.GetCharacterLevel())
	}
}

func TestAbilitiesManager_CharacterPoints(t *testing.T) {
	mgr := newTestAbilitiesManager()

	if mgr.GetCharacterPoints() != 0 {
		t.Errorf("initial character points: got %d, want 0", mgr.GetCharacterPoints())
	}
}

func TestAbilitiesManager_InitTalentWithLvl(t *testing.T) {
	mgr := newTestAbilitiesManager()

	mgr.InitTalentWithLvl(5)

	if mgr.GetTalentLevel() != 5 {
		t.Errorf("talent level after init: got %d, want 5", mgr.GetTalentLevel())
	}
}

func TestAbilitiesManager_IncreaseTalentExp(t *testing.T) {
	mgr := newTestAbilitiesManager()

	mgr.IncreaseTalentExp(10)
	if mgr.GetTalentExpPoints() != 10 {
		t.Errorf("talent exp after increase: got %d, want 10", mgr.GetTalentExpPoints())
	}
}

func TestAbilitiesManager_GetLevels(t *testing.T) {
	mgr := newTestAbilitiesManager()

	levels := mgr.GetLevels()
	if len(levels) != 4 {
		t.Errorf("expected 4 ability levels, got %d", len(levels))
	}
	for name, lvl := range levels {
		if lvl != 0 {
			t.Errorf("initial level for %s: got %d, want 0", name, lvl)
		}
	}
}

func TestAbilitiesManager_GetAllAbilities(t *testing.T) {
	mgr := newTestAbilitiesManager()

	all := mgr.GetAllAbilities()
	if len(all) != 4 {
		t.Errorf("expected 4 abilities, got %d", len(all))
	}
}

func TestAbilitiesManager_TalentDelegation(t *testing.T) {
	mgr := newTestAbilitiesManager()

	if mgr.GetTalentNextLvlBaseExp() <= 0 {
		t.Error("talent next lvl base exp should be positive")
	}
	if mgr.GetTalentNextLvlAggregateExp() <= 0 {
		t.Error("talent next lvl aggregate exp should be positive")
	}
	if mgr.GetTalentCurrentExp() != 0 {
		t.Errorf("initial talent current exp: got %d, want 0", mgr.GetTalentCurrentExp())
	}
	if mgr.GetTalentExpPoints() != 0 {
		t.Errorf("initial talent exp points: got %d, want 0", mgr.GetTalentExpPoints())
	}
}

func TestAbilitiesManager_CharacterExpDelegation(t *testing.T) {
	mgr := newTestAbilitiesManager()

	if mgr.GetCharacterNextLvlBaseExp() <= 0 {
		t.Error("character next lvl base exp should be positive")
	}
	if mgr.GetCharacterNextLvlAggregateExp() <= 0 {
		t.Error("character next lvl aggregate exp should be positive")
	}
	if mgr.GetCharacterCurrentExp() != 0 {
		t.Errorf("initial character current exp: got %d, want 0", mgr.GetCharacterCurrentExp())
	}
	if mgr.GetCharacterExpPoints() != 0 {
		t.Errorf("initial character exp points: got %d, want 0", mgr.GetCharacterExpPoints())
	}
}
