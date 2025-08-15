package sheet

import (
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	p "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	s "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterClassResponse struct {
	Profile             ClassProfileResponse          `json:"profile"`
	Distribution        *DistributionResponse         `json:"distribution,omitempty"`
	SkillsExps          map[string]LvlExp             `json:"skills_exps"`
	JointSkills         map[string]s.JointSkill       `json:"joint_skills"`
	ProficienciesExps   map[string]LvlExp             `json:"proficiencies_exps"`
	JointProficiencies  map[string]p.JointProficiency `json:"joint_proficiencies"`
	AttributesExps      map[string]LvlExp             `json:"attributes_exps"`
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

type LvlExp struct {
	Level int `json:"lvl"`
	Exp   int `json:"exp"`
}

func NewCharacterClassResponse(
	classSheet cs.HalfSheet, charClass cc.CharacterClass,
) CharacterClassResponse {
	skillsExps := make(map[string]LvlExp)
	physSkillsExp := classSheet.GetPhysicalSkillsExpPoints()
	physSkillsLvl := classSheet.GetPhysicalSkillsLevel()
	for skillName, exp := range physSkillsExp {
		if exp > 0 {
			skillsExps[skillName.String()] = LvlExp{
				Level: physSkillsLvl[skillName],
				Exp:   exp,
			}
		}
	}

	mentalSkillsExp := classSheet.GetMentalSkillsExpPoints()
	mentalSkillsLvl := classSheet.GetMentalSkillsLevel()
	for skillName, exp := range mentalSkillsExp {
		if exp > 0 {
			skillsExps[skillName.String()] = LvlExp{
				Level: mentalSkillsLvl[skillName],
				Exp:   exp,
			}
		}
	}

	proficienciesExps := make(map[string]LvlExp)
	commonProfs := classSheet.GetCommonProficiencies()
	for weaponName, prof := range commonProfs {
		proficienciesExps[weaponName.String()] = LvlExp{
			Level: prof.GetLevel(),
			Exp:   prof.GetExpPoints(),
		}
	}

	attributesExps := make(map[string]LvlExp)
	physAttrsExp := classSheet.GetPhysicalAttributesExpPoints()
	physAttrsLvl := classSheet.GetPhysicalAttributesLevels()
	for attrName, exp := range physAttrsExp {
		if exp > 0 {
			attributesExps[attrName.String()] = LvlExp{
				Level: physAttrsLvl[attrName],
				Exp:   exp,
			}
		}
	}

	mentalAttrsExp := classSheet.GetMentalAttributesExpPoints()
	mentalAttrsLvl := classSheet.GetMentalAttributesLevels()
	for attrName, exp := range mentalAttrsExp {
		if exp > 0 {
			attributesExps[attrName.String()] = LvlExp{
				Level: mentalAttrsLvl[attrName],
				Exp:   exp,
			}
		}
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

	profile := classSheet.GetProfile()
	return CharacterClassResponse{
		Profile: ClassProfileResponse{
			Name:             profile.NickName,
			Alignment:        profile.Alignment,
			Description:      profile.Description,
			BriefDescription: profile.BriefDescription,
		},
		Distribution:        distribution,
		SkillsExps:          skillsExps,
		JointSkills:         classSheet.GetPhysJointSkills(),
		ProficienciesExps:   proficienciesExps,
		JointProficiencies:  classSheet.GetJointProficiencies(),
		AttributesExps:      attributesExps,
		IndicatedCategories: indicatedCategories,
	}
}
