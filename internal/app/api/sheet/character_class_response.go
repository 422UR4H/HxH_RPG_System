package sheet

import (
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
)

type CharacterClassResponse struct {
	Profile             ClassProfileResponse
	Distribution        *DistributionResponse
	SkillsExps          map[string]int
	JointSkills         map[string]skill.JointSkill
	ProficienciesExps   map[string]int
	JointProficiencies  map[string]proficiency.JointProficiency
	AttributesExps      map[string]int
	IndicatedCategories []string
}

type ClassProfileResponse struct {
	Name             string
	Alignment        string
	Description      string
	BriefDescription string
}

type DistributionResponse struct {
	SkillPoints          []int
	ProficiencyPoints    []int
	SkillsAllowed        []string
	ProficienciesAllowed []string
}

func NewCharacterClassResponse(charClass cc.CharacterClass) CharacterClassResponse {
	skillsExps := make(map[string]int)
	for k, v := range charClass.SkillsExps {
		skillsExps[k.String()] = v
	}

	proficienciesExps := make(map[string]int)
	for k, v := range charClass.ProficienciesExps {
		proficienciesExps[k.String()] = v
	}

	attributesExps := make(map[string]int)
	for k, v := range charClass.AttributesExps {
		attributesExps[k.String()] = v
	}

	indicatedCategories := make([]string, len(charClass.IndicatedCategories))
	for i, v := range charClass.IndicatedCategories {
		indicatedCategories[i] = v.String()
	}
	if len(indicatedCategories) == 0 {
		for _, category := range enum.AllNenCategoryNames() {
			indicatedCategories = append(indicatedCategories, category.String())
		}
	}

	ccDistribution := charClass.Distribution
	var distribution *DistributionResponse
	if ccDistribution != nil {
		skillsAllowed := make([]string, len(ccDistribution.SkillsAllowed))
		for i, v := range ccDistribution.SkillsAllowed {
			skillsAllowed[i] = v.String()
		}

		proficienciesAllowed := make([]string, len(ccDistribution.ProficienciesAllowed))
		for i, v := range ccDistribution.ProficienciesAllowed {
			proficienciesAllowed[i] = v.String()
		}

		distribution = &DistributionResponse{
			SkillPoints:          ccDistribution.SkillPoints,
			ProficiencyPoints:    ccDistribution.ProficiencyPoints,
			SkillsAllowed:        skillsAllowed,
			ProficienciesAllowed: proficienciesAllowed,
		}
	}

	profile := charClass.Profile
	return CharacterClassResponse{
		Profile: ClassProfileResponse{
			Name:             profile.Name.String(),
			Alignment:        profile.Alignment,
			Description:      profile.Description,
			BriefDescription: profile.BriefDescription,
		},
		Distribution:        distribution,
		SkillsExps:          skillsExps,
		JointSkills:         charClass.JointSkills,
		ProficienciesExps:   proficienciesExps,
		JointProficiencies:  charClass.JointProficiencies,
		AttributesExps:      attributesExps,
		IndicatedCategories: indicatedCategories,
	}
}
