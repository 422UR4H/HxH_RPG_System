package sheet

import (
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	s "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterClassResponse struct {
	Profile       ClassProfileResponse         `json:"profile"`
	Abilities     map[string]Ability           `json:"abilities"`
	Attributes    map[string]Attribute         `json:"attributes"`
	Skills        map[string]Skill             `json:"skills"`
	JointSkills   map[string]s.JointSkill      `json:"joint_skills"`
	Proficiencies map[string]ExperienceResponse `json:"proficiencies"`
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
	ProficiencyPoints    []LvlExp `json:"proficiency_points"`
	SkillsAllowed        []string `json:"skills_allowed"`
	ProficienciesAllowed []string `json:"proficiencies_allowed"`
}

type LvlExp struct {
	Level int `json:"level"`
	Exp   int `json:"exp"`
}

type Ability struct {
	ExperienceResponse
	Bonus float64 `json:"bonus"`
}

type Attribute struct {
	ExperienceResponse
	Points int `json:"points"`
	Power  int `json:"power"`
}

type Skill struct {
	ExperienceResponse
	Value int `json:"value"`
}

type JointProf struct {
	ExperienceResponse
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
				ExperienceResponse: ExperienceResponse{
					Level:         ability.GetLevel(),
					Exp:           ability.GetExpPoints(),
					CurrentExp:    ability.GetCurrentExp(),
					NxtLvlBaseExp: ability.GetNextLvlBaseExp(),
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
				ExperienceResponse: ExperienceResponse{
					Level:         attr.GetLevel(),
					Exp:           attr.GetExpPoints(),
					CurrentExp:    attr.GetCurrentExp(),
					NxtLvlBaseExp: attr.GetNextLvlBaseExp(),
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
				ExperienceResponse: ExperienceResponse{
					Level:         attr.GetLevel(),
					Exp:           attr.GetExpPoints(),
					CurrentExp:    attr.GetCurrentExp(),
					NxtLvlBaseExp: attr.GetNextLvlBaseExp(),
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
				ExperienceResponse: ExperienceResponse{
					Level:         skill.GetLevel(),
					Exp:           skill.GetExpPoints(),
					CurrentExp:    skill.GetCurrentExp(),
					NxtLvlBaseExp: skill.GetNextLvlBaseExp(),
				},
				Value: skill.GetValueForTest(),
			}
		}
	}
	mentalSkills := classSheet.GetMentalSkills()
	for skillName, skill := range mentalSkills {
		if skill.GetExpPoints() > 0 {
			skills[skillName.String()] = Skill{
				ExperienceResponse: ExperienceResponse{
					Level:         skill.GetLevel(),
					Exp:           skill.GetExpPoints(),
					CurrentExp:    skill.GetCurrentExp(),
					NxtLvlBaseExp: skill.GetNextLvlBaseExp(),
				},
				Value: skill.GetValueForTest(),
			}
		}
	}

	proficiencies := make(map[string]ExperienceResponse)
	commonProfs := classSheet.GetCommonProficiencies()
	for weaponName, prof := range commonProfs {
		proficiencies[weaponName.String()] = ExperienceResponse{
			Level:         prof.GetLevel(),
			Exp:           prof.GetExpPoints(),
			CurrentExp:    prof.GetCurrentExp(),
			NxtLvlBaseExp: prof.GetNextLvlBaseExp(),
		}
	}

	jointProficiencies := []JointProf{}
	classJointProfs := classSheet.GetJointProficiencies()
	for profName, prof := range classJointProfs {
		jointProficiencies = append(jointProficiencies, JointProf{
			ExperienceResponse: ExperienceResponse{
				Level:         prof.GetLevel(),
				Exp:           prof.GetExpPoints(),
				CurrentExp:    prof.GetCurrentExp(),
				NxtLvlBaseExp: prof.GetNextLvlBaseExp(),
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

		expTable := experience.NewDefaultExpTable()
		profPoints := make([]LvlExp, len(ccDistribution.ProficiencyPoints))
		for i, xp := range ccDistribution.ProficiencyPoints {
			profPoints[i] = LvlExp{Level: expTable.GetLvlByExp(xp), Exp: xp}
		}
		distribution = &DistributionResponse{
			SkillPoints:          ccDistribution.SkillPoints,
			ProficiencyPoints:    profPoints,
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
