package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// sheetWithProficiency is plainSheet plus one weapon proficiency at a known level. The factory
// always starts a sheet with an empty proficiency manager, so the level has to be seeded by
// pouring in exactly the aggregate exp that lands on it — the same fixture the combat
// catalogue's handler test uses.
func sheetWithProficiency(t *testing.T, weapon enum.WeaponName, level int) *csSheet.CharacterSheet {
	t.Helper()
	cs := plainSheet(t)

	table := experience.NewExpTable(csSheet.PHYSICAL_SKILLS_COEFF)
	exp := experience.NewExperience(table)
	exp.IncreasePoints(table.GetAggregateExpByLvl(level))
	// physSkillsExp nil: this test only reads GetLevel(), never triggers the cascade.
	if err := cs.AddCommonProficiency(weapon, proficiency.NewProficiency(weapon, *exp, nil)); err != nil {
		t.Fatalf("AddCommonProficiency(%s): %v", weapon, err)
	}
	if got := cs.GetCommonProficiencies()[weapon].GetLevel(); got != level {
		t.Fatalf("fixture seeded level %d, want %d — the exp table moved under this test", got, level)
	}
	return cs
}

// The hit is Accuracy PLUS the proficiency with the weapon in hand — proficiencyLevel is not
// only a number the combat catalogue displays, it is added to the swing. A fresh sheet has
// Accuracy at 0, so the whole difference between the two resolutions below IS the proficiency.
func TestResolve_WeaponProficiencyAddsToTheHit(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	sword := enum.Sword
	const level = 7

	hitOf := func(t *testing.T, actorSheet *csSheet.CharacterSheet) int {
		t.Helper()
		tn := attackTurn(actorID, targetID, []int{6, 4}, []int{9, 3}, &sword)
		res := service.TurnResolver{}.Resolve(service.ResolveInput{
			Turn: tn,
			Sheets: map[uuid.UUID]*csSheet.CharacterSheet{
				actorID:  actorSheet,
				targetID: plainSheet(t),
			},
			Targets: charTargets{chars: map[uuid.UUID]bool{targetID: true}},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})
		if len(res.CharacterResults) != 1 {
			t.Fatalf("expected 1 character result, got %d", len(res.CharacterResults))
		}
		return res.CharacterResults[0].Hit.Total
	}

	unskilled := hitOf(t, plainSheet(t))
	skilled := hitOf(t, sheetWithProficiency(t, sword, level))

	if skilled-unskilled != level {
		t.Errorf("proficiency added %d to the hit, want %d (unskilled %d, skilled %d)",
			skilled-unskilled, level, unskilled, skilled)
	}
}

// The bare-handed blow reads the Fist proficiency: a nil weapon is not "no proficiency", it is
// the fist, which is exactly what the catalogue publishes as an always-present weapon.
func TestResolve_BareHandedHitReadsTheFistProficiency(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	const level = 5

	hitOf := func(t *testing.T, actorSheet *csSheet.CharacterSheet) int {
		t.Helper()
		tn := attackTurn(actorID, targetID, []int{6, 4}, []int{9, 3}, nil)
		res := service.TurnResolver{}.Resolve(service.ResolveInput{
			Turn: tn,
			Sheets: map[uuid.UUID]*csSheet.CharacterSheet{
				actorID:  actorSheet,
				targetID: plainSheet(t),
			},
			Targets: charTargets{chars: map[uuid.UUID]bool{targetID: true}},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})
		if len(res.CharacterResults) != 1 {
			t.Fatalf("expected 1 character result, got %d", len(res.CharacterResults))
		}
		return res.CharacterResults[0].Hit.Total
	}

	unskilled := hitOf(t, plainSheet(t))
	skilled := hitOf(t, sheetWithProficiency(t, enum.Fist, level))

	if skilled-unskilled != level {
		t.Errorf("the fist proficiency added %d to the bare-handed hit, want %d", skilled-unskilled, level)
	}
}

// A character with no proficiency in the weapon they swung adds zero — not a penalty. The rule
// nobody has written yet is what an untrained swing costs; until it exists, zero is the honest
// answer, and this test is what makes a future penalty a deliberate change instead of a
// silent one.
func TestResolve_NoProficiencyAddsNothing(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	sword, dagger := enum.Sword, enum.Dagger

	hitOf := func(t *testing.T, actorSheet *csSheet.CharacterSheet) int {
		t.Helper()
		tn := attackTurn(actorID, targetID, []int{6, 4}, []int{9, 3}, &sword)
		res := service.TurnResolver{}.Resolve(service.ResolveInput{
			Turn: tn,
			Sheets: map[uuid.UUID]*csSheet.CharacterSheet{
				actorID:  actorSheet,
				targetID: plainSheet(t),
			},
			Targets: charTargets{chars: map[uuid.UUID]bool{targetID: true}},
			Rules:   match.NewDefaultMatchRules(),
			Weapons: item.NewWeaponsManagerFactory().Build(),
		})
		return res.CharacterResults[0].Hit.Total
	}

	// Proficient with the dagger, swinging a sword: the dagger's level must not travel.
	if got, want := hitOf(t, sheetWithProficiency(t, dagger, 9)), hitOf(t, plainSheet(t)); got != want {
		t.Errorf("hit with an unproficient weapon = %d, want %d — another weapon's level leaked in", got, want)
	}
}
