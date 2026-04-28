package characterclass_test

import (
	"errors"
	"testing"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func buildClassWithDistribution() *cc.CharacterClass {
	profile := *cc.NewClassProfile(enum.Hunter, "", "Test class", "")
	distribution := &cc.Distribution{
		SkillPoints:          []int{210, 127},
		ProficiencyPoints:    []int{127, 127},
		SkillsAllowed:        []enum.SkillName{enum.Velocity, enum.Reflex, enum.Accuracy},
		ProficienciesAllowed: []enum.WeaponName{enum.Bow, enum.Longbow, enum.Dagger},
	}
	return cc.NewCharacterClass(profile, distribution, nil, nil, nil, nil, nil, nil, nil)
}

func buildClassWithoutDistribution() *cc.CharacterClass {
	profile := *cc.NewClassProfile(enum.Swordsman, "", "No distribution", "")
	return cc.NewCharacterClass(profile, nil, nil, nil, nil, nil, nil, nil, nil)
}

func TestCharacterClass_GetName(t *testing.T) {
	class := buildClassWithDistribution()
	if class.GetName() != enum.Hunter {
		t.Errorf("GetName() = %v, want Hunter", class.GetName())
	}
	if class.GetNameString() != "Hunter" {
		t.Errorf("GetNameString() = %s, want Hunter", class.GetNameString())
	}
}

func TestCharacterClass_ValidateSkills(t *testing.T) {
	t.Run("valid skills", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Velocity: 210,
			enum.Reflex:   127,
		}
		if err := class.ValidateSkills(skills); err != nil {
			t.Errorf("ValidateSkills() error = %v, want nil", err)
		}
	})

	t.Run("no distribution and no skills is valid", func(t *testing.T) {
		class := buildClassWithoutDistribution()
		if err := class.ValidateSkills(map[enum.SkillName]int{}); err != nil {
			t.Errorf("ValidateSkills() error = %v, want nil", err)
		}
	})

	t.Run("no distribution but skills provided", func(t *testing.T) {
		class := buildClassWithoutDistribution()
		skills := map[enum.SkillName]int{enum.Velocity: 100}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrNoSkillDistribution) {
			t.Errorf("error = %v, want ErrNoSkillDistribution", err)
		}
	})

	t.Run("wrong count of skills", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Velocity: 210,
		}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrSkillsCountMismatch) {
			t.Errorf("error = %v, want ErrSkillsCountMismatch", err)
		}
	})

	t.Run("skill not allowed", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Velocity: 210,
			enum.Heal:     127,
		}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrSkillNotAllowed) {
			t.Errorf("error = %v, want ErrSkillNotAllowed", err)
		}
	})

	t.Run("points mismatch", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Velocity: 210,
			enum.Reflex:   999,
		}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrSkillsPointsMismatch) {
			t.Errorf("error = %v, want ErrSkillsPointsMismatch", err)
		}
	})
}

func TestCharacterClass_ValidateProficiencies(t *testing.T) {
	t.Run("valid proficiencies", func(t *testing.T) {
		class := buildClassWithDistribution()
		profs := map[enum.WeaponName]int{
			enum.Bow:     127,
			enum.Longbow: 127,
		}
		if err := class.ValidateProficiencies(profs); err != nil {
			t.Errorf("ValidateProficiencies() error = %v, want nil", err)
		}
	})

	t.Run("no distribution and no profs is valid", func(t *testing.T) {
		class := buildClassWithoutDistribution()
		if err := class.ValidateProficiencies(map[enum.WeaponName]int{}); err != nil {
			t.Errorf("ValidateProficiencies() error = %v, want nil", err)
		}
	})

	t.Run("no distribution but profs provided", func(t *testing.T) {
		class := buildClassWithoutDistribution()
		profs := map[enum.WeaponName]int{enum.Dagger: 100}
		err := class.ValidateProficiencies(profs)
		if !errors.Is(err, cc.ErrNoProficiencyDistribution) {
			t.Errorf("error = %v, want ErrNoProficiencyDistribution", err)
		}
	})

	t.Run("wrong count", func(t *testing.T) {
		class := buildClassWithDistribution()
		profs := map[enum.WeaponName]int{enum.Bow: 127}
		err := class.ValidateProficiencies(profs)
		if !errors.Is(err, cc.ErrProficienciesCountMismatch) {
			t.Errorf("error = %v, want ErrProficienciesCountMismatch", err)
		}
	})

	t.Run("proficiency not allowed", func(t *testing.T) {
		class := buildClassWithDistribution()
		profs := map[enum.WeaponName]int{
			enum.Bow:   127,
			enum.Rifle: 127,
		}
		err := class.ValidateProficiencies(profs)
		if !errors.Is(err, cc.ErrProficiencyNotAllowed) {
			t.Errorf("error = %v, want ErrProficiencyNotAllowed", err)
		}
	})

	t.Run("points mismatch", func(t *testing.T) {
		class := buildClassWithDistribution()
		profs := map[enum.WeaponName]int{
			enum.Bow:     127,
			enum.Longbow: 999,
		}
		err := class.ValidateProficiencies(profs)
		if !errors.Is(err, cc.ErrProficienciesPointsMismatch) {
			t.Errorf("error = %v, want ErrProficienciesPointsMismatch", err)
		}
	})
}

func TestCharacterClass_ApplySkills(t *testing.T) {
	class := buildClassWithDistribution()
	skills := map[enum.SkillName]int{
		enum.Velocity: 210,
		enum.Reflex:   127,
	}
	class.ApplySkills(skills)

	if class.SkillsExps[enum.Velocity] != 210 {
		t.Errorf("SkillsExps[Velocity] = %d, want 210", class.SkillsExps[enum.Velocity])
	}
	if class.SkillsExps[enum.Reflex] != 127 {
		t.Errorf("SkillsExps[Reflex] = %d, want 127", class.SkillsExps[enum.Reflex])
	}
}

func TestCharacterClass_ApplyProficiencies(t *testing.T) {
	class := buildClassWithDistribution()
	profs := map[enum.WeaponName]int{
		enum.Bow: 127,
	}
	class.ApplyProficiencies(profs)

	if class.ProficienciesExps[enum.Bow] != 127 {
		t.Errorf("ProficienciesExps[Bow] = %d, want 127", class.ProficienciesExps[enum.Bow])
	}
}

func TestDistribution_AllowSkill(t *testing.T) {
	d := &cc.Distribution{
		SkillsAllowed: []enum.SkillName{enum.Velocity, enum.Reflex},
	}
	if !d.AllowSkill(enum.Velocity) {
		t.Error("AllowSkill(Velocity) = false, want true")
	}
	if d.AllowSkill(enum.Heal) {
		t.Error("AllowSkill(Heal) = true, want false")
	}
}

func TestDistribution_AllowProficiency(t *testing.T) {
	d := &cc.Distribution{
		ProficienciesAllowed: []enum.WeaponName{enum.Bow, enum.Longbow},
	}
	if !d.AllowProficiency(enum.Bow) {
		t.Error("AllowProficiency(Bow) = false, want true")
	}
	if d.AllowProficiency(enum.Rifle) {
		t.Error("AllowProficiency(Rifle) = true, want false")
	}
}
