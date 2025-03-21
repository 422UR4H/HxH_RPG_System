package characterclass

import (
	"fmt"
	"slices"

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
	if cc.Distribution == nil {
		if len(skills) != 0 {
			return fmt.Errorf(
				"character class %s has no skill distribution", cc.GetName(),
			)
		}
		return nil
	}
	d := *cc.Distribution
	skillPts := slices.Clone(d.SkillPoints)
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
	for _, exp := range skills {
		for i, points := range skillPts {
			if exp == points {
				skillPts = slices.Delete(skillPts, i, i+1)
				break
			}
		}
	}
	if len(skillPts) != 0 {
		return fmt.Errorf(
			"skills is not equals in character class %s", cc.GetName(),
		)
	}
	return nil
}

func (cc *CharacterClass) ValidateProficiencies(
	proficiencies map[enum.WeaponName]int,
) error {
	if cc.Distribution == nil {
		if len(proficiencies) != 0 {
			return fmt.Errorf(
				"character class %s has no proficiency distribution", cc.GetName(),
			)
		}
		return nil
	}
	d := *cc.Distribution
	profPts := slices.Clone(d.ProficiencyPoints)
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
	for _, exp := range proficiencies {
		for i, points := range profPts {
			if exp == points {
				profPts = slices.Delete(profPts, i, i+1)
				break
			}
		}
	}
	if len(profPts) != 0 {
		return fmt.Errorf(
			"proficiencies is not equals in character class %s", cc.GetName(),
		)
	}
	return nil
}

func (cc *CharacterClass) ApplySkills(
	skills map[enum.SkillName]int,
) {
	if cc.SkillsExps == nil {
		cc.SkillsExps = make(map[enum.SkillName]int)
	}
	for name, exp := range skills {
		cc.SkillsExps[name] = exp
	}
}

func (cc *CharacterClass) ApplyProficiencies(
	profs map[enum.WeaponName]int,
) {
	if cc.ProficienciesExps == nil {
		cc.ProficienciesExps = make(map[enum.WeaponName]int)
	}
	for name, exp := range profs {
		cc.ProficienciesExps[name] = exp
	}
}
