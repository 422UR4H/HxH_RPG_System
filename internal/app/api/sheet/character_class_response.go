package sheet

import (
	"sort"

	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	cs "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	s "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterClassResponse struct {
	Profile       ClassProfileResponse          `json:"profile"`
	Abilities     map[string]Ability            `json:"abilities"`
	Attributes    map[string]Attribute          `json:"attributes"`
	Skills        map[string]Skill              `json:"skills"`
	JointSkills   map[string]s.JointSkill       `json:"jointSkills"`
	Proficiencies map[string]ExperienceResponse `json:"proficiencies"`
	// JointProficiencies  map[string]p.JointProficiency `json:"jointProficiencies"`
	JointProficiencies  []JointProf           `json:"jointProficiencies"`
	IndicatedCategories []string              `json:"indicatedCategories"`
	Distribution        *DistributionResponse `json:"distribution,omitempty"`
	// Definition são os dados brutos com que a classe monta uma ficha. Os campos
	// acima são a ficha-modelo já calculada; o motor de prévia do front
	// (src/features/sheet/engine) precisa da entrada, não do resultado, para
	// repetir a montagem do back (AssembleCharacterSheet) a cada escolha do jogador.
	Definition ClassDefinitionResponse `json:"definition"`
}

// ClassDefinitionResponse espelha os campos de montagem de cc.CharacterClass.
// Chaves são os valores String() dos enums Go ("Vitality", "Sword").
// JointSkills fica de fora: nenhuma classe usa, e o teste de paridade falha se
// alguma passar a usar (o motor do front ainda não as implementa).
type ClassDefinitionResponse struct {
	SkillsExps         map[string]int             `json:"skillsExps"`
	AttributesExps     map[string]int             `json:"attributesExps"`
	ProficienciesExps  map[string]int             `json:"proficienciesExps"`
	JointProficiencies []ClassJointProfDefinition `json:"jointProficiencies"`
	JointProfExps      map[string]int             `json:"jointProfExps"`
}

type ClassJointProfDefinition struct {
	Name    string   `json:"name"`
	Weapons []string `json:"weapons"`
}

func newClassDefinitionResponse(charClass cc.CharacterClass) ClassDefinitionResponse {
	skills := make(map[string]int, len(charClass.SkillsExps))
	for name, exp := range charClass.SkillsExps {
		skills[name.String()] = exp
	}
	attrs := make(map[string]int, len(charClass.AttributesExps))
	for name, exp := range charClass.AttributesExps {
		attrs[name.String()] = exp
	}
	profs := make(map[string]int, len(charClass.ProficienciesExps))
	for name, exp := range charClass.ProficienciesExps {
		profs[name.String()] = exp
	}
	jointExps := make(map[string]int, len(charClass.JointProfExps))
	for name, exp := range charClass.JointProfExps {
		jointExps[name] = exp
	}

	joints := make([]ClassJointProfDefinition, 0, len(charClass.JointProficiencies))
	for name, prof := range charClass.JointProficiencies {
		weapons := []string{}
		for _, w := range prof.GetWeapons() {
			weapons = append(weapons, w.String())
		}
		joints = append(joints, ClassJointProfDefinition{Name: name, Weapons: weapons})
	}
	// ordem estável: o mapa da classe não tem ordem e a resposta vira fixture
	sort.Slice(joints, func(i, j int) bool { return joints[i].Name < joints[j].Name })

	return ClassDefinitionResponse{
		SkillsExps:         skills,
		AttributesExps:     attrs,
		ProficienciesExps:  profs,
		JointProficiencies: joints,
		JointProfExps:      jointExps,
	}
}

type ClassProfileResponse struct {
	Name             string `json:"name"`
	Alignment        string `json:"alignment"`
	Description      string `json:"description"`
	BriefDescription string `json:"briefDescription"`
}

type DistributionResponse struct {
	SkillPoints          []int    `json:"skillPoints"`
	ProficiencyPoints    []LvlExp `json:"proficiencyPoints"`
	SkillsAllowed        []string `json:"skillsAllowed"`
	ProficienciesAllowed []string `json:"proficienciesAllowed"`
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
	classSheet *cs.CharacterSheet, charClass cc.CharacterClass,
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

	// atributos espirituais não têm pontos distribuíveis (Points = 0)
	spiritAttrs := classSheet.GetSpiritualAttributes()
	for attrName, attr := range spiritAttrs {
		if attr.GetExpPoints() > 0 {
			attributes[attrName.String()] = Attribute{
				ExperienceResponse: ExperienceResponse{
					Level:         attr.GetLevel(),
					Exp:           attr.GetExpPoints(),
					CurrentExp:    attr.GetCurrentExp(),
					NxtLvlBaseExp: attr.GetNextLvlBaseExp(),
				},
				Points: 0,
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

	spiritSkills := classSheet.GetSpiritualSkills()
	for skillName, skill := range spiritSkills {
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

	// ordem estável: vem de um mapa, e a resposta entra na fixture de paridade
	sort.Slice(jointProficiencies, func(i, j int) bool {
		return jointProficiencies[i].Name < jointProficiencies[j].Name
	})

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
		Definition:          newClassDefinitionResponse(charClass),
	}
}
