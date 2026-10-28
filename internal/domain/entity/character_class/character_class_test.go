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
		SkillsAllowed:        []enum.SkillName{enum.Quickness, enum.Reflex, enum.Accuracy},
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
			enum.Quickness: 210,
			enum.Reflex:    127,
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
		skills := map[enum.SkillName]int{enum.Quickness: 100}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrNoSkillDistribution) {
			t.Errorf("error = %v, want ErrNoSkillDistribution", err)
		}
	})

	t.Run("wrong count of skills", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Quickness: 210,
		}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrSkillsCountMismatch) {
			t.Errorf("error = %v, want ErrSkillsCountMismatch", err)
		}
	})

	t.Run("skill not allowed", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Quickness: 210,
			enum.Heal:      127,
		}
		err := class.ValidateSkills(skills)
		if !errors.Is(err, cc.ErrSkillNotAllowed) {
			t.Errorf("error = %v, want ErrSkillNotAllowed", err)
		}
	})

	t.Run("points mismatch", func(t *testing.T) {
		class := buildClassWithDistribution()
		skills := map[enum.SkillName]int{
			enum.Quickness: 210,
			enum.Reflex:    999,
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
		enum.Quickness: 210,
		enum.Reflex:    127,
	}
	class.ApplySkills(skills)

	if class.SkillsExps[enum.Quickness] != 210 {
		t.Errorf("SkillsExps[Quickness] = %d, want 210", class.SkillsExps[enum.Quickness])
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
		SkillsAllowed: []enum.SkillName{enum.Quickness, enum.Reflex},
	}
	if !d.AllowSkill(enum.Quickness) {
		t.Error("AllowSkill(Quickness) = false, want true")
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

func TestCharacterClass_ApplyDoesNotLeakIntoCopies(t *testing.T) {
	tests := []struct {
		name      string
		skills    map[enum.SkillName]int
		profs     map[enum.WeaponName]int
		base      func() cc.CharacterClass
		wantSkill int
		wantProf  int
	}{
		{
			name: "mapas preenchidos",
			base: func() cc.CharacterClass {
				c := cc.BuildNinja()
				return c
			},
			skills:    map[enum.SkillName]int{enum.Smell: 1},
			profs:     map[enum.WeaponName]int{enum.Katana: 127},
			wantSkill: 1,
			wantProf:  127,
		},
		{
			name: "mapas nil",
			base: func() cc.CharacterClass {
				return *cc.NewCharacterClass(
					*cc.NewClassProfile(enum.Hunter, "", "x", ""),
					nil, nil, nil, nil, nil, nil, nil, nil,
				)
			},
			skills:    map[enum.SkillName]int{enum.Smell: 1},
			profs:     map[enum.WeaponName]int{enum.Katana: 127},
			wantSkill: 1,
			wantProf:  127,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c1 := tt.base()
			origSkills := len(c1.SkillsExps)
			origProfs := len(c1.ProficienciesExps)

			c2 := c1
			c2.ApplySkills(tt.skills)
			c2.ApplyProficiencies(tt.profs)

			if len(c1.SkillsExps) != origSkills {
				t.Errorf("c1.SkillsExps mudou: %d -> %d", origSkills, len(c1.SkillsExps))
			}
			if _, ok := c1.SkillsExps[enum.Smell]; ok && origSkills == 0 {
				t.Error("c1.SkillsExps vazou a perícia aplicada")
			}
			if len(c1.ProficienciesExps) != origProfs {
				t.Errorf("c1.ProficienciesExps mudou: %d -> %d", origProfs, len(c1.ProficienciesExps))
			}
			if got := c2.SkillsExps[enum.Smell]; got != tt.wantSkill {
				t.Errorf("c2 Hearing = %d, want %d", got, tt.wantSkill)
			}
			if got := c2.ProficienciesExps[enum.Katana]; got != tt.wantProf {
				t.Errorf("c2 Katana = %d, want %d", got, tt.wantProf)
			}
			if len(c2.SkillsExps) != origSkills+1 && origSkills > 0 {
				// Hearing já existe no Ninja; apenas valida que o original foi mantido
				if len(c2.SkillsExps) != origSkills {
					t.Errorf("c2 perdeu entradas originais")
				}
			}
			if len(c2.ProficienciesExps) != origProfs+1 {
				t.Errorf("c2.ProficienciesExps len = %d, want %d", len(c2.ProficienciesExps), origProfs+1)
			}
		})
	}
}
