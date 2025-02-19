package characterclass

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	prof "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
)

type CharacterClass struct {
	Profile            ClassProfile
	Distribution       *Distribution
	SkillsExps         map[enum.SkillName]int
	JointSkills        map[string]skill.JointSkill
	ProficienciesExps  map[enum.WeaponName]int
	JointProficiencies map[string]prof.JointProficiency
	// TODO: resolve mental skills and update below
	AttributesExps      map[enum.AttributeName]int
	IndicatedCategories []enum.CategoryName
}

// TODO: refactor to other file
type Distribution struct {
	SkillPoints          []int
	ProficiencyPoints    []int
	SkillsAllowed        []enum.SkillName
	ProficienciesAllowed []enum.WeaponName
}

func NewCharacterClass(
	profile ClassProfile,
	distribution *Distribution,
	skillsExps map[enum.SkillName]int,
	jointSkills map[string]skill.JointSkill,
	proficienciesExps map[enum.WeaponName]int,
	jointProficiencies map[string]prof.JointProficiency,
	attributesExps map[enum.AttributeName]int,
	indicatedCategories []enum.CategoryName,
) *CharacterClass {
	return &CharacterClass{
		Profile:             profile,
		Distribution:        distribution,
		SkillsExps:          skillsExps,
		JointSkills:         jointSkills,
		ProficienciesExps:   proficienciesExps,
		JointProficiencies:  jointProficiencies,
		AttributesExps:      attributesExps,
		IndicatedCategories: indicatedCategories,
	}
}

func (cc *CharacterClass) GetName() enum.CharacterClassName {
	return cc.Profile.Name
}

func (d *Distribution) AllowSkill(skill enum.SkillName) bool {
	for _, s := range d.SkillsAllowed {
		if s == skill {
			return true
		}
	}
	return false
}

func (d *Distribution) AllowProficiency(prof enum.WeaponName) bool {
	for _, s := range d.ProficienciesAllowed {
		if s == prof {
			return true
		}
	}
	return false
}

func (cc *CharacterClass) ValidateSkills(
	skills map[enum.SkillName]int,
) error {
	d := *cc.Distribution
	skillPts := d.SkillPoints
	if len(skillPts) != len(skills) {
		return fmt.Errorf(
			"skills count is not exact in character class %s", cc.GetName(),
		)
	}
	for name := range skills {
		if !d.AllowSkill(name) {
			return fmt.Errorf(
				"skill %s not found in character class %s", name, cc.GetName(),
			)
		}
	}
	for name, exp := range skills {
		for i, points := range skillPts {
			if exp == points {
				skillPts = append(skillPts[:i], skillPts[i+1:]...)
				delete(skills, name)
				break
			}
		}
		if len(skills) != 0 || len(skillPts) != 0 {
			return fmt.Errorf(
				"skills is not equals in character class %s", cc.GetName(),
			)
		}
	}
	return nil
}

func (cc *CharacterClass) ValidateProficiencies(
	proficiencies map[enum.WeaponName]int,
) error {
	d := *cc.Distribution
	profPts := d.ProficiencyPoints
	if len(profPts) != len(proficiencies) {
		return fmt.Errorf(
			"proficiencies count is not exact in character class %s", cc.GetName(),
		)
	}
	for name := range proficiencies {
		if !d.AllowProficiency(name) {
			return fmt.Errorf(
				"proficiency %s not found in character class %s", name, cc.GetName(),
			)
		}
	}
	for name, exp := range proficiencies {
		for i, points := range profPts {
			if exp == points {
				profPts = append(profPts[:i], profPts[i+1:]...)
				delete(proficiencies, name)
				break
			}
		}
		if len(proficiencies) != 0 || len(profPts) != 0 {
			return fmt.Errorf(
				"proficiencies is not equals in character class %s", cc.GetName(),
			)
		}
	}
	return nil
}

func (cc *CharacterClass) ApplySkills(
	skills map[enum.SkillName]int,
) {
	for name, exp := range skills {
		cc.SkillsExps[name] = exp
	}
}

func (cc *CharacterClass) ApplyProficiencies(
	profs map[enum.WeaponName]int,
) {
	for name, exp := range profs {
		cc.ProficienciesExps[name] = exp
	}
}
