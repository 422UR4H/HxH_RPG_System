package sheet_test

// TestExpPointsFormula_* verifies that the SQL backfill formula in
// migrations/20260529000002_fix_exp_points_backfill.sql produces exactly the
// same exp_points value that the domain computes via wrap().
//
// If any of these tests fail after a domain change, the SQL migration formula
// MUST be updated to match the new cascade math.

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// expInputs mirrors the exp columns that feed CharacterExp in wrap().
// Excluded: talent_exp, nen_exp, focus_exp, will_power_exp, aura_control_exp, aop_exp.
type expInputs struct {
	// Mental attributes (PrimaryAttribute → MentalsAbility → CharacterExp): ×1
	Resilience, Adaptability, Weighting, Creativity int

	// Physical skills via PrimaryAttribute (attr→physAbility + physSkills→skillsAbility): ×2
	Vitality, Energy, Defense     int // Resistance-based
	Velocity, Accelerate, Brake   int // Agility-based
	Acrobatics, Evasion, Sneak    int // Flexibility-based
	Vision, Hearing, Smell        int // Sense-based
	Tact, Taste                   int // Sense-based (continued)

	// Physical skills via MiddleAttribute (lenAttrs=2): floor(groupSum/2)*3
	Push, Grab, Carry          int // Strength = Resistance+Agility
	Legerity, Repel, Feint     int // Celerity  = Agility+Flexibility
	Reflex, Accuracy, Stealth  int // Dexterity = Flexibility+Sense
	Heal, Breath, Tenacity     int // Constitution = Sense+Resistance

	// Spiritual principles (conscienceNen→spiritualsAbility→CharacterExp): ×1
	Ten, Zetsu, Ren, Gyo, Shu, Kou, Ken, Ryu, In, En int

	// Spiritual categories (hatsu→conscienceNen→spiritualsAbility→CharacterExp): ×1
	Reinforcement, Transmutation, Materialization int
	Specialization, Manipulation, Emission        int

	// Proficiencies (physAbility→CharacterExp): ×1
	Proficiencies int
}

// expPointsFormula is the Go equivalent of the SQL backfill in
// 20260529000002_fix_exp_points_backfill.sql.
// Keep this function and that migration in sync — if one changes, so must the other.
func expPointsFormula(e expInputs) int {
	return e.Resilience + e.Adaptability + e.Weighting + e.Creativity +
		(e.Vitality+e.Energy+e.Defense)*2 +
		(e.Velocity+e.Accelerate+e.Brake)*2 +
		(e.Acrobatics+e.Evasion+e.Sneak)*2 +
		(e.Vision+e.Hearing+e.Smell+e.Tact+e.Taste)*2 +
		((e.Push+e.Grab+e.Carry)/2)*3 +
		((e.Legerity+e.Repel+e.Feint)/2)*3 +
		((e.Reflex+e.Accuracy+e.Stealth)/2)*3 +
		((e.Heal+e.Breath+e.Tenacity)/2)*3 +
		e.Ten + e.Zetsu + e.Ren + e.Gyo + e.Shu + e.Kou + e.Ken + e.Ryu + e.In + e.En +
		e.Reinforcement + e.Transmutation + e.Materialization +
		e.Specialization + e.Manipulation + e.Emission +
		e.Proficiencies
}

// applyExpToSheet applies exp inputs to the sheet in the same order wrap() uses.
func applyExpToSheet(t *testing.T, cs *sheet.CharacterSheet, e expInputs) {
	t.Helper()

	for name, exp := range map[enum.AttributeName]int{
		enum.Resilience:   e.Resilience,
		enum.Adaptability: e.Adaptability,
		enum.Weighting:    e.Weighting,
		enum.Creativity:   e.Creativity,
	} {
		if exp == 0 {
			continue
		}
		if err := cs.IncreaseExpForMentals(experience.NewUpgradeCascade(exp), name); err != nil {
			t.Fatalf("IncreaseExpForMentals(%s, %d): %v", name, exp, err)
		}
	}

	for name, exp := range map[enum.SkillName]int{
		enum.Vitality: e.Vitality, enum.Energy: e.Energy, enum.Defense: e.Defense,
		enum.Push: e.Push, enum.Grab: e.Grab, enum.Carry: e.Carry,
		enum.Velocity: e.Velocity, enum.Accelerate: e.Accelerate, enum.Brake: e.Brake,
		enum.Legerity: e.Legerity, enum.Repel: e.Repel, enum.Feint: e.Feint,
		enum.Acrobatics: e.Acrobatics, enum.Evasion: e.Evasion, enum.Sneak: e.Sneak,
		enum.Reflex: e.Reflex, enum.Accuracy: e.Accuracy, enum.Stealth: e.Stealth,
		enum.Vision: e.Vision, enum.Hearing: e.Hearing, enum.Smell: e.Smell,
		enum.Tact: e.Tact, enum.Taste: e.Taste,
		enum.Heal: e.Heal, enum.Breath: e.Breath, enum.Tenacity: e.Tenacity,
	} {
		if exp == 0 {
			continue
		}
		if err := cs.IncreaseExpForSkill(experience.NewUpgradeCascade(exp), name); err != nil {
			t.Fatalf("IncreaseExpForSkill(%s, %d): %v", name, exp, err)
		}
	}

	for name, exp := range map[enum.PrincipleName]int{
		enum.Ten: e.Ten, enum.Zetsu: e.Zetsu, enum.Ren: e.Ren, enum.Gyo: e.Gyo,
		enum.Shu: e.Shu, enum.Kou: e.Kou, enum.Ken: e.Ken, enum.Ryu: e.Ryu,
		enum.In: e.In, enum.En: e.En,
	} {
		if exp == 0 {
			continue
		}
		if err := cs.IncreaseExpForPrinciple(experience.NewUpgradeCascade(exp), name); err != nil {
			t.Fatalf("IncreaseExpForPrinciple(%s, %d): %v", name, exp, err)
		}
	}

	for name, exp := range map[enum.CategoryName]int{
		enum.Reinforcement:   e.Reinforcement,
		enum.Transmutation:   e.Transmutation,
		enum.Materialization: e.Materialization,
		enum.Specialization:  e.Specialization,
		enum.Manipulation:    e.Manipulation,
		enum.Emission:        e.Emission,
	} {
		if exp == 0 {
			continue
		}
		if err := cs.IncreaseExpForCategory(experience.NewUpgradeCascade(exp), name); err != nil {
			t.Fatalf("IncreaseExpForCategory(%s, %d): %v", name, exp, err)
		}
	}

	if e.Proficiencies > 0 {
		physSkExp, err := cs.GetPhysSkillExpReference()
		if err != nil {
			t.Fatalf("GetPhysSkillExpReference: %v", err)
		}
		expTable := experience.NewExpTable(sheet.PHYSICAL_SKILLS_COEFF)
		newExp := experience.NewExperience(expTable)
		domainProf := proficiency.NewProficiency(enum.Dagger, *newExp, physSkExp)
		if err := cs.AddCommonProficiency(enum.Dagger, domainProf); err != nil {
			t.Fatalf("AddCommonProficiency: %v", err)
		}
		if err := cs.IncreaseExpForProficiency(experience.NewUpgradeCascade(e.Proficiencies), enum.Dagger); err != nil {
			t.Fatalf("IncreaseExpForProficiency: %v", err)
		}
	}
}

func assertExpPoints(t *testing.T, cs *sheet.CharacterSheet, e expInputs) {
	t.Helper()
	got := cs.GetExpPoints()
	want := expPointsFormula(e)
	if got != want {
		t.Errorf("GetExpPoints() = %d, formula = %d (diff = %+d)", got, want, got-want)
	}
}

// --- Individual cascade type tests ---

func TestExpPointsFormula_MentalAttr_FactorOne(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{Resilience: 100, Adaptability: 80, Weighting: 60, Creativity: 40}
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e) // want: 280 (100+80+60+40 × 1)
}

func TestExpPointsFormula_PrimaryAttrSkill_FactorTwo(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{Vitality: 100, Energy: 50, Defense: 25}
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e) // want: 350 ((100+50+25) × 2)
}

func TestExpPointsFormula_AllPrimaryAttrGroups_FactorTwo(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{
		Vitality: 100, Energy: 100, Defense: 100, // Resistance-based
		Velocity: 100, Accelerate: 100, Brake: 100, // Agility-based
		Acrobatics: 100, Evasion: 100, Sneak: 100, // Flexibility-based
		Vision: 100, Hearing: 100, Smell: 100, Tact: 100, Taste: 100, // Sense-based
	}
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e) // want: 14 skills × 100 × 2 = 2800
}

func TestExpPointsFormula_MiddleAttrGroup_EvenTotal(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{Push: 10, Grab: 20, Carry: 30} // T=60 → floor(60/2)*3 = 90
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e)
}

func TestExpPointsFormula_MiddleAttrGroup_OddTotal(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{Push: 1, Grab: 2, Carry: 0} // T=3 → floor(3/2)*3 = 3
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e)
}

func TestExpPointsFormula_MiddleAttrGroup_SingleOdd(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{Push: 5} // T=5 → floor(5/2)*3 = 6
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e)
}

func TestExpPointsFormula_AllMiddleAttrGroups(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{
		Push: 100, Grab: 100, Carry: 100, // Strength
		Legerity: 100, Repel: 100, Feint: 100, // Celerity
		Reflex: 100, Accuracy: 100, Stealth: 100, // Dexterity
		Heal: 100, Breath: 100, Tenacity: 100, // Constitution
	}
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e) // want: 4 groups × floor(300/2)*3 = 4×450 = 1800
}

func TestExpPointsFormula_SpiritualPrinciple_FactorOne(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{Ten: 100, Zetsu: 200, Ren: 150}
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e) // want: 450 (×1)
}

func TestExpPointsFormula_SpiritualCategory_FactorOne(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{Reinforcement: 300, Emission: 200}
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e) // want: 500 (×1)
}

func TestExpPointsFormula_Proficiency_FactorOne(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{Proficiencies: 250}
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e) // want: 250 (×1)
}

// TestExpPointsFormula_AllTypes is the integration test — applies every column
// type simultaneously and asserts the formula matches the domain cascade result.
// This is the canonical guard against regressions in the backfill formula.
func TestExpPointsFormula_AllTypes_MatchesDomainCascade(t *testing.T) {
	cs := buildTestSheet(t)
	e := expInputs{
		Resilience: 100, Adaptability: 80, Weighting: 60, Creativity: 40,

		Vitality: 200, Energy: 150, Defense: 100,
		Velocity: 120, Accelerate: 90, Brake: 70,
		Acrobatics: 110, Evasion: 85, Sneak: 65,
		Vision: 130, Hearing: 100, Smell: 75, Tact: 50, Taste: 40,

		Push: 180, Grab: 160, Carry: 140,
		Legerity: 170, Repel: 150, Feint: 130,
		Reflex: 190, Accuracy: 170, Stealth: 150,
		Heal: 160, Breath: 140, Tenacity: 120,

		Ten: 300, Zetsu: 250, Ren: 200, Gyo: 180,
		Shu: 160, Kou: 140, Ken: 120, Ryu: 100, In: 80, En: 60,

		Reinforcement: 400, Transmutation: 350, Materialization: 300,
		Specialization: 250, Manipulation: 200, Emission: 150,

		Proficiencies: 500,
	}
	applyExpToSheet(t, cs, e)
	assertExpPoints(t, cs, e)
}
