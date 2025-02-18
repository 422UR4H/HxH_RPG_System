package characterclass

import (
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
