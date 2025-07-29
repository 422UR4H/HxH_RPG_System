package sheet

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	prof "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type HalfSheet struct {
	profile     CharacterProfile
	ability     ability.Manager
	attribute   attribute.CharacterAttributes
	skill       skill.CharacterSkills
	proficiency prof.Manager
	status      status.Manager
	charClass   *enum.CharacterClassName
	// equipedItems []Item
}

func NewHalfSheet(
	profile CharacterProfile,
	abilities ability.Manager,
	attributes attribute.CharacterAttributes,
	skills skill.CharacterSkills,
	proficiency prof.Manager,
	status status.Manager,
	charClass *enum.CharacterClassName,
) *HalfSheet {
	return &HalfSheet{
		profile:     profile,
		ability:     abilities,
		attribute:   attributes,
		skill:       skills,
		proficiency: proficiency,
		status:      status,
		charClass:   charClass,
	}
}

func (hs *HalfSheet) GetClass() enum.CharacterClassName {
	return *hs.charClass
}

func (hs *HalfSheet) GetValueForTestOfSkill(name enum.SkillName) (int, error) {
	return hs.skill.GetValueForTestOf(name)
}

func (hs *HalfSheet) IncreaseExpForSkill(
	values *experience.UpgradeCascade, name enum.SkillName,
) error {
	return hs.skill.IncreaseExp(values, name)
}

// AddJointSkill only supports physical skills yet
func (hs *HalfSheet) AddJointSkill(
	skill *skill.JointSkill,
) error {
	physSkillsExp, err := hs.ability.GetExpReferenceOf(enum.Physicals)
	if err != nil {
		return err
	}
	if err := skill.Init(physSkillsExp); err != nil {
		return err
	}
	return hs.skill.AddPhysicalJoint(skill)
}

func (hs *HalfSheet) GetPhysJointSkills() map[string]skill.JointSkill {
	return hs.skill.GetPhysicalsJoint()
}

func (hs *HalfSheet) IncreaseExpForProficiency(
	values *experience.UpgradeCascade, name enum.WeaponName,
) error {
	return hs.proficiency.IncreaseExp(values, name)
}

// TODO: resolve this
func (hs *HalfSheet) IncreaseExpForMentals(
	values *experience.UpgradeCascade, name enum.AttributeName,
) error {
	return hs.attribute.IncreaseExpForMentals(values, name)
}

func (hs *HalfSheet) AddJointProficiency(
	proficiency *prof.JointProficiency,
) error {
	physSkillsExp, err := hs.ability.GetExpReferenceOf(enum.Physicals)
	if err != nil {
		return err
	}
	abilitySkillsExp, err := hs.ability.GetExpReferenceOf(enum.Skills)
	if err != nil {
		return err
	}
	return hs.proficiency.AddJoint(proficiency, physSkillsExp, abilitySkillsExp)
}

func (hs *HalfSheet) AddCommonProficiency(
	name enum.WeaponName, proficiency *prof.Proficiency,
) error {
	return hs.proficiency.AddCommon(name, proficiency)
}

func (hs *HalfSheet) GetMaxOfStatus(name enum.StatusName) (int, error) {
	return hs.status.GetMaxOf(name)
}

func (hs *HalfSheet) GetMinOfStatus(name enum.StatusName) (int, error) {
	return hs.status.GetMinOf(name)
}

func (hs *HalfSheet) GetLevelOfAbility(name enum.AbilityName) (int, error) {
	return hs.ability.GetLevelOf(name)
}

func (hs *HalfSheet) GetLevelOfAttribute(name enum.AttributeName) (int, error) {
	return hs.attribute.GetLevelOf(name)
}

func (hs *HalfSheet) GetLevelOfSkill(name enum.SkillName) (int, error) {
	return hs.skill.GetLevelOf(name)
}

func (hs *HalfSheet) GetExpPointsOfAbility(name enum.AbilityName) (int, error) {
	return hs.ability.GetExpPointsOf(name)
}

func (hs *HalfSheet) GetExpPointsOfAttribute(name enum.AttributeName) (int, error) {
	return hs.attribute.GetExpPointsOf(name)
}

func (hs *HalfSheet) GetExpPointsOfSkill(name enum.SkillName) (int, error) {
	return hs.skill.GetExpPointsOf(name)
}

func (hs *HalfSheet) GetExpPoints() int {
	return hs.ability.GetCharacterExpPoints()
}

func (hs *HalfSheet) ToString() string {
	const nameWidth = 14
	const valueWidth = 4

	sheet := "===========================================================\n"
	sheet += hs.profile.ToString()

	sheet += fmt.Sprintf("CHARACTER LVL: %-*d | Points: %-*d | Talent: %-*d\n",
		valueWidth, hs.ability.GetCharacterLevel(),
		valueWidth, hs.ability.GetCharacterPoints(),
		valueWidth, hs.ability.GetTalentLevel())

	sheet += fmt.Sprintf("Exp Total: %-*d | Exp: %d / %-*d\n",
		valueWidth, hs.ability.GetCharacterExpPoints(),
		hs.ability.GetCharacterCurrentExp(),
		valueWidth, hs.ability.GetCharacterNextLvlBaseExp())
	sheet += "-----------------------------------------------------------\n"

	physicals, _ := hs.ability.Get(enum.Physicals)
	sheet += fmt.Sprintf("PHYSICALS LVL: %d | Bonus: %.1f\n",
		physicals.GetLevel(),
		physicals.GetBonus())

	sheet += fmt.Sprintf("Exp Total: %-*d | Exp: %d / %-*d\n",
		valueWidth, physicals.GetExpPoints(),
		physicals.GetCurrentExp(),
		valueWidth, physicals.GetNextLvlBaseExp())

	physicalsLvl := hs.attribute.GetPhysicalsLevel()
	physicalsExp := hs.attribute.GetPhysicalsExpPoints()
	physicalsCurrExp := hs.attribute.GetPhysicalsCurrentExp()
	physicalsNextLvlExp := hs.attribute.GetPhysicalsNextLvlBaseExp()
	sortedAttrNames := []enum.AttributeName{
		enum.Resistance, enum.Strength, enum.Agility, enum.Celerity,
		enum.Flexibility, enum.Dexterity, enum.Sense, enum.Constitution,
	}
	for _, name := range sortedAttrNames {
		lvl := physicalsLvl[name]
		exp := physicalsExp[name]
		currExp := physicalsCurrExp[name]
		nextLvlExp := physicalsNextLvlExp[name]
		sheet += fmt.Sprintf("%-*s Lvl: %-*d | Exp Total: %-*d | Exp: %d / %-*d\n",
			nameWidth, name.String(),
			valueWidth, lvl,
			valueWidth, exp,
			currExp,
			valueWidth, nextLvlExp)
	}
	sheet += "-----------------------------------------------------------\n"

	skillsLvl := hs.skill.GetPhysicalsLevel()
	skillsExp := hs.skill.GetPhysicalsExpPoints()
	skillsCurrExp := hs.skill.GetPhysicalsCurrentExp()
	skillsNextLvlExp := hs.skill.GetPhysicalsNextLvlBaseExp()
	sortSkillNames := []enum.SkillName{
		enum.Vitality, enum.Energy, enum.Defense,
		enum.Push, enum.Grab, enum.Carry,
		enum.Velocity, enum.Accelerate, enum.Brake,
		enum.Legerity, enum.Repel, enum.Feint,
		enum.Acrobatics, enum.Evasion, enum.Sneak,
		enum.Reflex, enum.Accuracy, enum.Stealth,
		enum.Vision, enum.Hearing, enum.Smell, enum.Tact, enum.Taste,
		enum.Heal, enum.Breath, enum.Tenacity,
	}
	for _, name := range sortSkillNames {
		lvl := skillsLvl[name]
		exp := skillsExp[name]
		if lvl == 0 && lvl == exp {
			continue
		}
		currExp := skillsCurrExp[name]
		nextLvlExp := skillsNextLvlExp[name]

		sheet += fmt.Sprintf("%-*s Lvl: %-*d | Exp Total: %-*d | Exp: %d / %-*d\n",
			nameWidth, name.String(),
			valueWidth, lvl,
			valueWidth, exp,
			currExp,
			valueWidth, nextLvlExp)
	}
	sheet += "-----------------------------------------------------------\n"

	mentals, _ := hs.ability.Get(enum.Mentals)
	sheet += fmt.Sprintf("MENTALS LVL: %d | Bonus: %.1f\n",
		mentals.GetLevel(),
		mentals.GetBonus())

	sheet += fmt.Sprintf("Exp Total: %-*d | Exp: %d / %-*d\n",
		valueWidth, mentals.GetExpPoints(),
		mentals.GetCurrentExp(),
		valueWidth, mentals.GetNextLvlBaseExp())

	mentalsLvl := hs.attribute.GetMentalsLevel()
	mentalsExp := hs.attribute.GetMentalsExpPoints()
	mentalsCurrExp := hs.attribute.GetMentalsCurrentExp()
	mentalsNextLvlExp := hs.attribute.GetMentalsNextLvlBaseExp()
	sortedAttrNames = []enum.AttributeName{
		enum.Resilience, enum.Adaptability, enum.Weighting, enum.Creativity,
	}
	for _, name := range sortedAttrNames {
		lvl := mentalsLvl[name]
		exp := mentalsExp[name]
		currExp := mentalsCurrExp[name]
		nextLvlExp := mentalsNextLvlExp[name]
		sheet += fmt.Sprintf("%-*s Lvl: %-*d | Exp Total: %-*d | Exp: %d / %-*d\n",
			nameWidth, name.String(),
			valueWidth, lvl,
			valueWidth, exp,
			currExp,
			valueWidth, nextLvlExp)
	}
	sheet += "-----------------------------------------------------------\n"

	statusList := hs.status.GetAllStatus()
	for name, status := range statusList {
		sheet += fmt.Sprintf("%-*s Min %-*d | %d / %-*d\n",
			nameWidth, name.String(),
			valueWidth, status.GetMin(),
			status.GetCurrent(),
			valueWidth, status.GetMax())
	}
	sheet += "===========================================================\n"
	return sheet
}
