package sheet

import (
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	p "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	s "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterClassResponse struct {
	Profile             ClassProfileResponse          `json:"profile"`
	Distribution        *DistributionResponse         `json:"distribution,omitempty"`
	SkillsExps          map[string]int                `json:"skills_exps"`
	JointSkills         map[string]s.JointSkill       `json:"joint_skills"`
	ProficienciesExps   map[string]int                `json:"proficiencies_exps"`
	JointProficiencies  map[string]p.JointProficiency `json:"joint_proficiencies"`
	AttributesExps      map[string]int                `json:"attributes_exps"`
	IndicatedCategories []string                      `json:"indicated_categories"`
}

type ClassProfileResponse struct {
	Name             string `json:"name"`
	Alignment        string `json:"alignment"`
	Description      string `json:"description"`
	BriefDescription string `json:"brief_description"`
}

type DistributionResponse struct {
	SkillPoints          []int    `json:"skill_points"`
	ProficiencyPoints    []int    `json:"proficiency_points"`
	SkillsAllowed        []string `json:"skills_allowed"`
	ProficienciesAllowed []string `json:"proficiencies_allowed"`
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
