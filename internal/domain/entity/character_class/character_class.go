package characterclass

import (
	"slices"

	prof "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterClass struct {
	Profile            ClassProfile
	Distribution       *Distribution
	SkillsExps         map[enum.SkillName]int
	JointSkills        map[string]skill.JointSkill
	ProficienciesExps  map[enum.WeaponName]int
	JointProficiencies map[string]prof.JointProficiency
	JointProfExps      map[string]int
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
	JointProfExps map[string]int,
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
		JointProfExps:       JointProfExps,
		AttributesExps:      attributesExps,
		IndicatedCategories: indicatedCategories,
	}
}

func (cc *CharacterClass) GetName() enum.CharacterClassName {
	return cc.Profile.Name
}

func (cc *CharacterClass) GetNameString() string {
	return cc.Profile.Name.String()
}

func (d *Distribution) AllowSkill(skill enum.SkillName) bool {
	return slices.Contains(d.SkillsAllowed, skill)
}

func (d *Distribution) AllowProficiency(prof enum.WeaponName) bool {
	return slices.Contains(d.ProficienciesAllowed, prof)
}

func (cc *CharacterClass) ValidateSkills(
	skills map[enum.SkillName]int,
) error {
	if cc.Distribution == nil {
		if len(skills) != 0 {
			return NewNoSkillDistributionError(cc.GetNameString())
		}
		return nil
	}
	d := *cc.Distribution
	skillPts := slices.Clone(d.SkillPoints)
	if len(skillPts) != len(skills) {
		return NewSkillsCountMismatchError(cc.GetNameString())
	}
	for name := range skills {
		if !d.AllowSkill(name) {
			return NewSkillNotAllowedError(name.String(), cc.GetNameString())
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
		return NewSkillsPointsMismatchError(cc.GetNameString())
	}
	return nil
}

func (cc *CharacterClass) ValidateProficiencies(
	proficiencies map[enum.WeaponName]int,
) error {
	if cc.Distribution == nil {
		if len(proficiencies) != 0 {
			return NewNoProficiencyDistributionError(cc.GetNameString())
		}
		return nil
	}
	d := *cc.Distribution
	profPts := slices.Clone(d.ProficiencyPoints)
	if len(profPts) != len(proficiencies) {
		return NewProficienciesCountMismatchError(cc.GetNameString())
	}
	for name := range proficiencies {
		if !d.AllowProficiency(name) {
			return NewProficiencyNotAllowedError(name.String(), cc.GetNameString())
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
		return NewProficienciesPointsMismatchError(cc.GetNameString())
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
