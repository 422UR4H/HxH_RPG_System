package sheet

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
	prof "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/status"
)

type CharacterSheet struct {
	profile     CharacterProfile
	ability     ability.Manager
	attribute   attribute.CharacterAttributes
	skill       skill.CharacterSkills
	principle   spiritual.Manager
	proficiency prof.Manager
	status      status.Manager
	charClass   *enum.CharacterClassName
	// equipedItems []Item
}

func NewCharacterSheet(
	profile CharacterProfile,
	abilities ability.Manager,
	attributes attribute.CharacterAttributes,
	principles spiritual.Manager,
	skills skill.CharacterSkills,
	proficiency prof.Manager,
	status status.Manager,
	charClass *enum.CharacterClassName,
) *CharacterSheet {
	return &CharacterSheet{
		profile:     profile,
		ability:     abilities,
		attribute:   attributes,
		skill:       skills,
		principle:   principles,
		proficiency: proficiency,
		status:      status,
		charClass:   charClass,
	}
}

func (cs *CharacterSheet) GetClass() enum.CharacterClassName {
	return *cs.charClass
}

func (cs *CharacterSheet) GetValueForTestOfSkill(name enum.SkillName) (int, error) {
	return cs.skill.GetValueForTestOf(name)
}

func (cs *CharacterSheet) GetValueForTestOfAttribute(name enum.AttributeName) (int, error) {
	return cs.attribute.GetPowerOf(name)
}

func (cs *CharacterSheet) IncreaseExpForSkill(
	values *experience.UpgradeCascade, name enum.SkillName,
) error {
	err := cs.skill.IncreaseExp(values, name)
	cs.status.Upgrade()
	return err
}

// AddJointSkill only supports physical skills yet
func (cs *CharacterSheet) AddJointSkill(
	skill *skill.JointSkill,
) error {
	physSkillsExp, err := cs.ability.GetExpReferenceOf(enum.Physicals)
	if err != nil {
		return err
	}
	if err := skill.Init(physSkillsExp); err != nil {
		return err
	}
	return cs.skill.AddPhysicalJoint(skill)
}

func (cs *CharacterSheet) GetPhysJointSkills() map[string]skill.JointSkill {
	return cs.skill.GetPhysicalsJoint()
}

func (cs *CharacterSheet) IncreaseExpForPrinciple(
	values *experience.UpgradeCascade, name enum.PrincipleName,
) error {
	err := cs.principle.IncreaseExpByPrinciple(name, values)
	cs.status.Upgrade()
	return err
}

func (cs *CharacterSheet) IncreaseExpForCategory(
	values *experience.UpgradeCascade, name enum.CategoryName,
) error {
	err := cs.principle.IncreaseExpByCategory(name, values)
	cs.status.Upgrade()
	return err
}

func (cs *CharacterSheet) IncreaseExpForProficiency(
	values *experience.UpgradeCascade, name enum.WeaponName,
) error {
	err := cs.proficiency.IncreaseExp(values, name)
	cs.status.Upgrade()
	return err
}

// TODO: resolve this
func (cs *CharacterSheet) IncreaseExpForMentals(
	values *experience.UpgradeCascade, name enum.AttributeName,
) error {
	err := cs.attribute.IncreaseExpForMentals(values, name)
	cs.status.Upgrade()
	return err
}

func (cs *CharacterSheet) AddJointProficiency(
	proficiency *prof.JointProficiency,
) error {
	physSkillsExp, err := cs.ability.GetExpReferenceOf(enum.Physicals)
	if err != nil {
		return err
	}
	return cs.proficiency.AddJoint(proficiency, physSkillsExp)
}

func (cs *CharacterSheet) AddCommonProficiency(
	name enum.WeaponName, proficiency *prof.Proficiency,
) error {
	return cs.proficiency.AddCommon(name, proficiency)
}

func (cs *CharacterSheet) GetMaxOfStatus(name enum.StatusName) (int, error) {
	return cs.status.GetMaxOf(name)
}

func (cs *CharacterSheet) GetMinOfStatus(name enum.StatusName) (int, error) {
	return cs.status.GetMinOf(name)
}

func (cs *CharacterSheet) GetLevelOfAbility(name enum.AbilityName) (int, error) {
	return cs.ability.GetLevelOf(name)
}

func (cs *CharacterSheet) GetLevelOfAttribute(name enum.AttributeName) (int, error) {
	return cs.attribute.GetLevelOf(name)
}

func (cs *CharacterSheet) GetLevelOfSkill(name enum.SkillName) (int, error) {
	return cs.skill.GetLevelOf(name)
}

func (cs *CharacterSheet) GetLevelOfPrinciple(name enum.PrincipleName) (int, error) {
	return cs.principle.GetLevelOfPrinciple(name)
}

func (cs *CharacterSheet) GetLevelOfCategory(name enum.CategoryName) (int, error) {
	return cs.principle.GetLevelOfCategory(name)
}

func (cs *CharacterSheet) GetExpPointsOfAbility(name enum.AbilityName) (int, error) {
	return cs.ability.GetExpPointsOf(name)
}

func (cs *CharacterSheet) GetExpPointsOfAttribute(name enum.AttributeName) (int, error) {
	return cs.attribute.GetExpPointsOf(name)
}

func (cs *CharacterSheet) GetExpPointsOfSkill(name enum.SkillName) (int, error) {
	return cs.skill.GetExpPointsOf(name)
}

func (cs *CharacterSheet) GetExpPointsOfPrinciple(name enum.PrincipleName) (int, error) {
	return cs.principle.GetExpPointsOfPrinciple(name)
}

func (cs *CharacterSheet) GetExpPointsOfCategory(name enum.CategoryName) (int, error) {
	return cs.principle.GetExpPointsOfCategory(name)
}

func (cs *CharacterSheet) GetExpPoints() int {
	return cs.ability.GetCharacterExpPoints()
}

func (cs *CharacterSheet) ToString() string {
	const nameWidth = 14
	const valueWidth = 4

	sheet := "===========================================================\n"
	sheet += cs.profile.ToString()

	sheet += fmt.Sprintf("CHARACTER LVL: %-*d | Points: %-*d | Talent: %-*d\n",
		valueWidth, cs.ability.GetCharacterLevel(),
		valueWidth, cs.ability.GetCharacterPoints(),
		valueWidth, cs.ability.GetTalentLevel())

	sheet += fmt.Sprintf("Exp Total: %-*d | Exp: %d / %-*d\n",
		valueWidth, cs.ability.GetCharacterExpPoints(),
		cs.ability.GetCharacterCurrentExp(),
		valueWidth, cs.ability.GetCharacterNextLvlBaseExp())
	sheet += "-----------------------------------------------------------\n"

	physicals, _ := cs.ability.Get(enum.Physicals)
	sheet += fmt.Sprintf("PHYSICALS LVL: %d | Bonus: %.1f\n",
		physicals.GetLevel(),
		physicals.GetBonus())

	sheet += fmt.Sprintf("Exp Total: %-*d | Exp: %d / %-*d\n",
		valueWidth, physicals.GetExpPoints(),
		physicals.GetCurrentExp(),
		valueWidth, physicals.GetNextLvlBaseExp())

	physicalsLvl := cs.attribute.GetPhysicalsLevel()
	physicalsExp := cs.attribute.GetPhysicalsExpPoints()
	physicalsCurrExp := cs.attribute.GetPhysicalsCurrentExp()
	physicalsNextLvlExp := cs.attribute.GetPhysicalsNextLvlBaseExp()
	sortedAttrNames := []enum.AttributeName{
		enum.Resistance, enum.Strength, enum.Agility, enum.ActionSpeed,
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

	skillsLvl := cs.skill.GetPhysicalsLevel()
	skillsExp := cs.skill.GetPhysicalsExpPoints()
	skillsCurrExp := cs.skill.GetPhysicalsCurrentExp()
	skillsNextLvlExp := cs.skill.GetPhysicalsNextLvlBaseExp()
	sortedSkillNames := []enum.SkillName{
		enum.Vitality, enum.Energy, enum.Defense,
		enum.Push, enum.Grab, enum.CarryCapacity,
		enum.Velocity, enum.Accelerate, enum.Brake,
		enum.AttackSpeed, enum.Repel, enum.Feint,
		enum.Acrobatics, enum.Evasion, enum.Sneak,
		enum.Reflex, enum.Accuracy, enum.Stealth,
		enum.Vision, enum.Hearing, enum.Smell,
		enum.Tact, enum.Taste, enum.Balance,
		enum.Heal, enum.Breath, enum.Tenacity,
	}
	for _, name := range sortedSkillNames {
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

	mentals, _ := cs.ability.Get(enum.Mentals)
	sheet += fmt.Sprintf("MENTALS LVL: %d | Bonus: %.1f\n",
		mentals.GetLevel(),
		mentals.GetBonus())

	sheet += fmt.Sprintf("Exp Total: %-*d | Exp: %d / %-*d\n",
		valueWidth, mentals.GetExpPoints(),
		mentals.GetCurrentExp(),
		valueWidth, mentals.GetNextLvlBaseExp())

	mentalsLvl := cs.attribute.GetMentalsLevel()
	mentalsExp := cs.attribute.GetMentalsExpPoints()
	mentalsCurrExp := cs.attribute.GetMentalsCurrentExp()
	mentalsNextLvlExp := cs.attribute.GetMentalsNextLvlBaseExp()
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

	statusList := cs.status.GetAllStatus()
	for name, status := range statusList {
		sheet += fmt.Sprintf("%-*s Min %-*d | %d / %-*d\n",
			nameWidth, name.String(),
			valueWidth, status.GetMin(),
			status.GetCurrent(),
			valueWidth, status.GetMax())
	}
	sheet += "-----------------------------------------------------------\n"

	profLvl := cs.proficiency.GetCommonsLevel()
	profExp := cs.proficiency.GetCommonsExpPoints()
	profCurrExp := cs.proficiency.GetCommonsCurrentExp()
	profNextLvlExp := cs.proficiency.GetCommonsNextLvlBaseExp()
	profNames := cs.proficiency.GetWeapons()

	for _, name := range profNames {
		lvl := profLvl[name]
		exp := profExp[name]
		currExp := profCurrExp[name]
		nextLvlExp := profNextLvlExp[name]
		sheet += fmt.Sprintf("%-*s Lvl: %-*d | Exp Total: %-*d | Exp: %d / %-*d\n",
			nameWidth, name.String(),
			valueWidth, lvl,
			valueWidth, exp,
			currExp,
			valueWidth, nextLvlExp)
	}

	jointProfs := cs.proficiency.GetJointProficiencies()
	for name, prof := range jointProfs {
		sheet += fmt.Sprintf("%-*s Lvl: %-*d | Exp Total: %-*d | Exp: %d / %-*d\n",
			nameWidth, name,
			valueWidth, prof.GetLevel(),
			valueWidth, prof.GetExpPoints(),
			prof.GetCurrentExp(),
			valueWidth, prof.GetNextLvlBaseExp())
	}

	sheet += "===========================================================\n"
	return sheet
}
