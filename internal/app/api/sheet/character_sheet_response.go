package sheet

import (
	"strings"

	domainSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/google/uuid"
)

type CharacterSheetResponse struct {
	UUID         uuid.UUID  `json:"uuid"`
	PlayerUUID   *uuid.UUID `json:"player_uuid,omitempty"`
	MasterUUID   *uuid.UUID `json:"master_uuid,omitempty"`
	CampaignUUID *uuid.UUID `json:"campaign_uuid,omitempty"`

	CharacterClass string                       `json:"character_class"`
	CategoryName   string                       `json:"category_name"`
	Profile        domainSheet.CharacterProfile `json:"profile"`

	CharacterExp CharacterExpResponse `json:"character_exp"`
	Talent       TalentResponse       `json:"talent"`
	NenHexValue  *int                 `json:"nen_hex_value,omitempty"`

	Abilities           map[string]AbilityResponse            `json:"abilities"`
	PhysicalAttributes  map[string]AttributeResponse          `json:"physical_attributes"`
	MentalAttributes    map[string]AttributeResponse          `json:"mental_attributes"`
	SpiritualAttributes map[string]SpiritualAttributeResponse `json:"spiritual_attributes"`
	PhysicalSkills      map[string]SkillResponse              `json:"physical_skills"`
	MentalSkills        map[string]SkillResponse              `json:"mental_skills"`
	SpiritualSkills     map[string]SkillResponse              `json:"spiritual_skills"`
	Principles          map[string]PrincipleResponse          `json:"principles"`
	Categories          map[string]CategoryResponse           `json:"categories"`
	// JointSkills         map[string]skill.JointSkill             `json:"joint_skills"`
	Proficiencies      map[string]CommonProficiencyResponse `json:"common_proficiencies"`
	JointProficiencies map[string]JointProficiencyResponse  `json:"joint_proficiencies"`
	Status             map[string]StatusResponse            `json:"status"`
}

// TODO: maybe refactor adding constructor
type ExperienceResponse struct {
	Level         int `json:"level"`
	Exp           int `json:"exp"`
	CurrentExp    int `json:"curr_exp"`
	NxtLvlBaseExp int `json:"next_lvl_base_exp"`
}

type CharacterExpResponse struct {
	ExperienceResponse
	Points int `json:"points"`
}

type TalentResponse struct {
	ExperienceResponse
}

type AbilityResponse struct {
	ExperienceResponse
	Bonus float64 `json:"bonus"`
}

type AttributeResponse struct {
	ExperienceResponse
	Points int `json:"points"`
	Value  int `json:"value"`
	Power  int `json:"power"`
}

type SpiritualAttributeResponse struct {
	ExperienceResponse
	Power int `json:"power"`
}

type PrincipleResponse struct {
	ExperienceResponse
	ValueForTest int `json:"value"`
}

type CategoryResponse struct {
	ExperienceResponse
	ValueForTest int     `json:"value"`
	Percent      float64 `json:"percent"`
}

type SkillResponse struct {
	ExperienceResponse
	ValueForTest int `json:"value"`
}

type CommonProficiencyResponse struct {
	ExperienceResponse
}

type JointProficiencyResponse struct {
	ExperienceResponse
	Weapons []string `json:"weapons"`
}

type StatusResponse struct {
	Min     int `json:"min"`
	Current int `json:"current"`
	Max     int `json:"max"`
}

func NewCharacterSheetResponse(
	charSheet *domainSheet.CharacterSheet) *CharacterSheetResponse {

	categoryName, err := charSheet.GetCategoryName()
	strCategoryName := categoryName.String()
	if err != nil {
		strCategoryName = ""
	}
	charClass := charSheet.GetCharacterClass()
	nenHexValue := charSheet.GetCurrHexValue()

	charExp := CharacterExpResponse{
		ExperienceResponse: ExperienceResponse{
			Level:         charSheet.GetLevel(),
			Exp:           charSheet.GetExpPoints(),
			CurrentExp:    charSheet.GetCurrentExp(),
			NxtLvlBaseExp: charSheet.GetNextLvlBaseExp(),
		},
		Points: charSheet.GetCharacterPoints(),
	}

	talent := TalentResponse{
		ExperienceResponse: ExperienceResponse{
			Level:         charSheet.GetTalentLevel(),
			Exp:           charSheet.GetTalentExpPoints(),
			CurrentExp:    charSheet.GetTalentCurrentExp(),
			NxtLvlBaseExp: charSheet.GetTalentNextLvlBaseExp(),
		},
	}

	abilities := make(map[string]AbilityResponse)
	for name, ability := range charSheet.GetAbilities() {
		abilities[name.String()] = AbilityResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         ability.GetLevel(),
				Exp:           ability.GetExpPoints(),
				CurrentExp:    ability.GetCurrentExp(),
				NxtLvlBaseExp: ability.GetNextLvlBaseExp(),
			},
			Bonus: ability.GetBonus(),
		}
	}

	physicAttrs := make(map[string]AttributeResponse)
	for name, attr := range charSheet.GetPhysicalAttributes() {
		physicAttrs[name.String()] = AttributeResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         attr.GetLevel(),
				Exp:           attr.GetExpPoints(),
				CurrentExp:    attr.GetCurrentExp(),
				NxtLvlBaseExp: attr.GetNextLvlBaseExp(),
			},
			Points: attr.GetPoints(),
			Value:  attr.GetValue(),
			Power:  attr.GetPower(),
		}
	}

	mentalAttrs := make(map[string]AttributeResponse)
	for name, attr := range charSheet.GetMentalAttributes() {
		mentalAttrs[name.String()] = AttributeResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         attr.GetLevel(),
				Exp:           attr.GetExpPoints(),
				CurrentExp:    attr.GetCurrentExp(),
				NxtLvlBaseExp: attr.GetNextLvlBaseExp(),
			},
			Points: attr.GetPoints(),
			Value:  attr.GetValue(),
			Power:  attr.GetPower(),
		}
	}

	spiritAttrs := make(map[string]SpiritualAttributeResponse)
	for name, attr := range charSheet.GetSpiritualAttributes() {
		spiritAttrs[name.String()] = SpiritualAttributeResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         attr.GetLevel(),
				Exp:           attr.GetExpPoints(),
				CurrentExp:    attr.GetCurrentExp(),
				NxtLvlBaseExp: attr.GetNextLvlBaseExp(),
			},
			Power: attr.GetPower(),
		}
	}

	physickills := make(map[string]SkillResponse)
	for name, skill := range charSheet.GetPhysicalSkills() {
		physickills[name.String()] = SkillResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         skill.GetLevel(),
				Exp:           skill.GetExpPoints(),
				CurrentExp:    skill.GetCurrentExp(),
				NxtLvlBaseExp: skill.GetNextLvlBaseExp(),
			},
			ValueForTest: skill.GetValueForTest(),
		}
	}

	mentalSkills := make(map[string]SkillResponse)
	for name, skill := range charSheet.GetMentalSkills() {
		mentalSkills[name.String()] = SkillResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         skill.GetLevel(),
				Exp:           skill.GetExpPoints(),
				CurrentExp:    skill.GetCurrentExp(),
				NxtLvlBaseExp: skill.GetNextLvlBaseExp(),
			},
			ValueForTest: skill.GetValueForTest(),
		}
	}

	spiritSkills := make(map[string]SkillResponse)
	for name, skill := range charSheet.GetSpiritualSkills() {
		spiritSkills[name.String()] = SkillResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         skill.GetLevel(),
				Exp:           skill.GetExpPoints(),
				CurrentExp:    skill.GetCurrentExp(),
				NxtLvlBaseExp: skill.GetNextLvlBaseExp(),
			},
			ValueForTest: skill.GetValueForTest(),
		}
	}

	principles := make(map[string]PrincipleResponse)
	for name, principle := range charSheet.GetPrinciples() {
		principles[name.String()] = PrincipleResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         principle.GetLevel(),
				Exp:           principle.GetExpPoints(),
				CurrentExp:    principle.GetCurrentExp(),
				NxtLvlBaseExp: principle.GetNextLvlBaseExp(),
			},
			ValueForTest: principle.GetValueForTest(),
		}
	}

	categories := make(map[string]CategoryResponse)
	for name, category := range charSheet.GetCategories() {
		categories[name.String()] = CategoryResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         category.GetLevel(),
				Exp:           category.GetExpPoints(),
				CurrentExp:    category.GetCurrentExp(),
				NxtLvlBaseExp: category.GetNextLvlBaseExp(),
			},
			ValueForTest: category.GetValueForTest(),
			Percent:      category.GetPercent(),
		}
	}

	commonProfs := make(map[string]CommonProficiencyResponse)
	for name, prof := range charSheet.GetCommonProficiencies() {
		commonProfs[name.String()] = CommonProficiencyResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         prof.GetLevel(),
				Exp:           prof.GetExpPoints(),
				CurrentExp:    prof.GetCurrentExp(),
				NxtLvlBaseExp: prof.GetNextLvlBaseExp(),
			},
		}
	}

	jointProfs := make(map[string]JointProficiencyResponse)
	for name, prof := range charSheet.GetJointProficiencies() {

		weapons := []string{}
		for _, weapon := range prof.GetWeapons() {
			weapons = append(weapons, weapon.String())
		}

		jointProfs[name] = JointProficiencyResponse{
			ExperienceResponse: ExperienceResponse{
				Level:         prof.GetLevel(),
				Exp:           prof.GetExpPoints(),
				CurrentExp:    prof.GetCurrentExp(),
				NxtLvlBaseExp: prof.GetNextLvlBaseExp(),
			},
			Weapons: weapons,
		}
	}

	status := make(map[string]StatusResponse)
	for name, statusBar := range charSheet.GetAllStatusBar() {
		status[strings.ToLower(name.String())] = StatusResponse{
			Min:     statusBar.GetMin(),
			Current: statusBar.GetCurrent(),
			Max:     statusBar.GetMax(),
		}
	}

	return &CharacterSheetResponse{
		UUID:                charSheet.UUID,
		PlayerUUID:          charSheet.GetPlayerUUID(),
		MasterUUID:          charSheet.GetMasterUUID(),
		CampaignUUID:        charSheet.GetCampaignUUID(),
		Profile:             charSheet.GetProfile(),
		CharacterClass:      charClass.String(),
		CategoryName:        strCategoryName,
		CharacterExp:        charExp,
		Talent:              talent,
		NenHexValue:         nenHexValue,
		Abilities:           abilities,
		PhysicalAttributes:  physicAttrs,
		MentalAttributes:    mentalAttrs,
		SpiritualAttributes: spiritAttrs,
		PhysicalSkills:      physickills,
		MentalSkills:        mentalSkills,
		SpiritualSkills:     spiritSkills,
		Principles:          principles,
		Categories:          categories,
		// JointSkills:         charSheet.GetJointSkills(),
		Proficiencies:      commonProfs,
		JointProficiencies: jointProfs,
		Status:             status,
	}
}
