package sheet

import (
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	s "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterClassResponse struct {
	Profile       ClassProfileResponse    `json:"profile"`
	Abilities     map[string]Ability      `json:"abilities"`
	Attributes    map[string]Attribute    `json:"attributes"`
	Skills        map[string]Skill        `json:"skills"`
	JointSkills   map[string]s.JointSkill `json:"joint_skills"`
	Proficiencies map[string]LvlExp       `json:"proficiencies"`
	// JointProficiencies  map[string]p.JointProficiency `json:"joint_proficiencies"`
	JointProficiencies  []JointProf           `json:"joint_proficiencies"`
	IndicatedCategories []string              `json:"indicated_categories"`
	Distribution        *DistributionResponse `json:"distribution,omitempty"`
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
	Level int `json:"level"`
	Exp   int `json:"exp"`
}

type Ability struct {
	LvlExp
	Bonus float64 `json:"bonus"`
}

type Attribute struct {
	LvlExp
	Points int `json:"points"`
	Power  int `json:"power"`
}

type Skill struct {
	LvlExp
	Value int `json:"value"`
}

type JointProf struct {
	LvlExp
	Name string `json:"name"`
}

func NewCharacterClassResponse(
	classSheet cs.HalfSheet, charClass cc.CharacterClass,
) CharacterClassResponse {
	abilities := make(map[string]Ability)
	abilitiesList := classSheet.GetAbilities()
	for abilityName, ability := range abilitiesList {
		if ability.GetExpPoints() > 0 {
			abilities[abilityName.String()] = Ability{
				LvlExp: LvlExp{
					Level: ability.GetLevel(),
					Exp:   ability.GetExpPoints(),
				},
				Bonus: ability.GetBonus(),
			}
		}
	}

	// TODO: improve to use more than one "attributes"
	attributes := make(map[string]Attribute)
	physAttrs := classSheet.GetPhysicalAttributes()
	for attrName, attr := range physAttrs {
		if attr.GetExpPoints() > 0 {
			attributes[attrName.String()] = Attribute{
				LvlExp: LvlExp{
					Level: attr.GetLevel(),
					Exp:   attr.GetExpPoints(),
				},
				Points: attr.GetPoints(),
				Power:  attr.GetPower(),
			}
		}
	}
	mentalAttrs := classSheet.GetMentalAttributes()
	for attrName, attr := range mentalAttrs {
		if attr.GetExpPoints() > 0 {
			attributes[attrName.String()] = Attribute{
				LvlExp: LvlExp{
					Level: attr.GetLevel(),
					Exp:   attr.GetExpPoints(),
				},
				Points: attr.GetPoints(),
				Power:  attr.GetPower(),
			}
		}
	}

	// TODO: improve to use more than one "skills"
	skills := make(map[string]Skill)
	physSkills := classSheet.GetPhysicalSkills()
	for skillName, skill := range physSkills {
		if skill.GetExpPoints() > 0 {
			skills[skillName.String()] = Skill{
				LvlExp: LvlExp{
					Level: skill.GetLevel(),
					Exp:   skill.GetExpPoints(),
				},
				Value: skill.GetValueForTest(),
			}
		}
	}
	mentalSkills := classSheet.GetMentalSkills()
	for skillName, skill := range mentalSkills {
		if skill.GetExpPoints() > 0 {
			skills[skillName.String()] = Skill{
				LvlExp: LvlExp{
					Level: skill.GetLevel(),
					Exp:   skill.GetExpPoints(),
				},
				Value: skill.GetValueForTest(),
			}
		}
	}

	proficiencies := make(map[string]LvlExp)
	commonProfs := classSheet.GetCommonProficiencies()
	for weaponName, prof := range commonProfs {
		proficiencies[weaponName.String()] = LvlExp{
			Level: prof.GetLevel(),
			Exp:   prof.GetExpPoints(),
		}
	}

	jointProficiencies := []JointProf{}
	classJointProfs := classSheet.GetJointProficiencies()
	for profName, prof := range classJointProfs {
		jointProficiencies = append(jointProficiencies, JointProf{
			LvlExp: LvlExp{
				Level: prof.GetLevel(),
				Exp:   prof.GetExpPoints(),
			},
			Name: profName,
		})
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
		Skills:              skills,
		JointSkills:         classSheet.GetPhysJointSkills(),
		Proficiencies:       proficiencies,
		JointProficiencies:  jointProficiencies,
		Attributes:          attributes,
		Abilities:           abilities,
		IndicatedCategories: indicatedCategories,
	}
}
