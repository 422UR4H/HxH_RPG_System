package sheet

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/proficiency"
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
	proficiency proficiency.Manager
	status      status.Manager
	// equipedItems []Item
}

func NewCharacterSheet(
	profile CharacterProfile,
	abilities ability.Manager,
	attributes attribute.CharacterAttributes,
	principles spiritual.Manager,
	skills skill.CharacterSkills,
	proficiency proficiency.Manager,
	status status.Manager,
) *CharacterSheet {
	return &CharacterSheet{
		profile:     profile,
		ability:     abilities,
		attribute:   attributes,
		skill:       skills,
		principle:   principles,
		proficiency: proficiency,
		status:      status,
	}
}

func (cs *CharacterSheet) GetValueForTestOfSkill(name enum.SkillName) (int, error) {
	return cs.skill.GetValueForTestOf(name)
}

func (cs *CharacterSheet) GetValueForTestOfAttribute(name enum.AttributeName) (int, error) {
	return cs.attribute.GetPowerOf(name)
}

func (cs *CharacterSheet) IncreaseExpForSkill(
	points int, name enum.SkillName,
) (int, error) {
	return cs.skill.IncreaseExp(points, name)
}

func (cs *CharacterSheet) IncreaseExpForPrinciple(
	points int, name enum.PrincipleName,
) (int, error) {
	return cs.principle.IncreaseExpByPrinciple(name, points)
}

func (cs *CharacterSheet) IncreaseExpForCategory(
	points int, name enum.CategoryName,
) (int, error) {
	return cs.principle.IncreaseExpByCategory(name, points)
}

func (cs *CharacterSheet) IncreaseExpForProficiency(
	points int, name enum.WeaponName,
) (int, error) {
	return cs.proficiency.IncreaseExp(points, name)
}

// func (cs *CharacterSheet) AddProficiency(name enum.WeaponName) error {
// 	return cs.proficiency.AddProficiency(name)
// }

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
	sheet := "=============================\n"
	sheet += cs.profile.ToString()

	sheet += "CHARACTER LVL: " + fmt.Sprint(cs.ability.GetCharacterLevel()) +
		"\t| Points: " + fmt.Sprint(cs.ability.GetCharacterPoints()) +
		"\t| Talent: " + fmt.Sprint(cs.ability.GetTalentLevel()) + "\n"

	sheet += "Exp Total: " + fmt.Sprint(cs.ability.GetCharacterExpPoints()) +
		"\t| Exp: " + fmt.Sprint(cs.ability.GetCharacterCurrentExp()) +
		" / " + fmt.Sprint(cs.ability.GetCharacterNextLvlBaseExp()) + "\n"
	sheet += "-----------------------------\n"

	physicals, _ := cs.ability.Get(enum.Physicals)
	sheet += "PHYSICALS LVL: " + fmt.Sprint(physicals.GetLevel()) +
		"\t| Bonus: " + fmt.Sprintf("%.1f\n", physicals.GetBonus())

	sheet += "Exp Total: " + fmt.Sprint(physicals.GetExpPoints()) +
		"\t| Exp: " + fmt.Sprint(physicals.GetCurrentExp()) +
		" / " + fmt.Sprint(physicals.GetNextLvlBaseExp()) + "\n"

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
		sheet += name.String() + " Lvl: " + fmt.Sprint(lvl) +
			"\t| Exp Total: " + fmt.Sprint(exp) +
			"\t| Exp: " + fmt.Sprint(currExp) +
			" / " + fmt.Sprint(nextLvlExp) + "\n"
	}
	sheet += "-----------------------------\n"

	skillsLvl := cs.skill.GetPhysicalsLevel()
	skillsExp := cs.skill.GetPhysicalsExpPoints()
	skillsCurrExp := cs.skill.GetPhysicalsCurrentExp()
	skillsNextLvlExp := cs.skill.GetPhysicalsNextLvlBaseExp()
	sortSkillNames := []enum.SkillName{
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
	for _, name := range sortSkillNames {
		lvl := skillsLvl[name]
		exp := skillsExp[name]
		if lvl == 0 && lvl == exp {
			continue
		}
		currExp := skillsCurrExp[name]
		nextLvlExp := skillsNextLvlExp[name]

		sheet += name.String() + " Lvl: " + fmt.Sprint(lvl) +
			"\t| Exp Total: " + fmt.Sprint(exp) +
			"\t| Exp: " + fmt.Sprint(currExp) +
			" / " + fmt.Sprint(nextLvlExp) + "\n"
	}
	sheet += "-----------------------------\n"

	mentals, _ := cs.ability.Get(enum.Mentals)
	sheet += "MENTALS LVL: " + fmt.Sprint(mentals.GetLevel()) +
		"\t| Bonus: " + fmt.Sprintf("%.1f\n", mentals.GetBonus())

	sheet += "Exp Total: " + fmt.Sprint(mentals.GetExpPoints()) +
		"\t| Exp: " + fmt.Sprint(mentals.GetCurrentExp()) +
		" / " + fmt.Sprint(mentals.GetNextLvlAggregateExp()) + "\n"
	sheet += "-----------------------------\n"

	statusList := cs.status.GetAllStatus()
	for name, status := range statusList {
		sheet += name.String() + ": Min " + fmt.Sprint(status.GetMin()) +
			"\t| : " + fmt.Sprint(status.GetCurrent()) +
			" / " + fmt.Sprint(status.GetMax()) + "\n"
	}
	sheet += "=============================\n"
	return sheet
}
