package sheet

import (
	"fmt"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	prof "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/status"
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
	points int, name enum.SkillName,
) (int, error) {
	return hs.skill.IncreaseExp(points, name)
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
	points int, name enum.WeaponName,
) (int, error) {
	return hs.proficiency.IncreaseExp(points, name)
}

// TODO: resolve this
func (hs *HalfSheet) IncreaseExpForMentals(
	points int, name enum.AttributeName,
) (int, error) {
	return hs.attribute.IncreaseExpForMentals(points, name)
}

func (hs *HalfSheet) AddJointProficiency(
	proficiency *prof.JointProficiency,
) error {
	physSkillsExp, err := hs.ability.GetExpReferenceOf(enum.Physicals)
	if err != nil {
		return err
	}
	return hs.proficiency.AddJoint(proficiency, physSkillsExp)
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
	sheet := "=============================\n"
	sheet += hs.profile.ToString()

	sheet += "CHARACTER LVL: " + fmt.Sprint(hs.ability.GetCharacterLevel()) +
		"\t| Points: " + fmt.Sprint(hs.ability.GetCharacterPoints()) +
		"\t| Talent: " + fmt.Sprint(hs.ability.GetTalentLevel()) + "\n"

	sheet += "Exp Total: " + fmt.Sprint(hs.ability.GetCharacterExpPoints()) +
		"\t| Exp: " + fmt.Sprint(hs.ability.GetCharacterCurrentExp()) +
		" / " + fmt.Sprint(hs.ability.GetCharacterNextLvlBaseExp()) + "\n"
	sheet += "-----------------------------\n"

	physicals, _ := hs.ability.Get(enum.Physicals)
	sheet += "PHYSICALS LVL: " + fmt.Sprint(physicals.GetLevel()) +
		"\t| Bonus: " + fmt.Sprintf("%.1f\n", physicals.GetBonus())

	sheet += "Exp Total: " + fmt.Sprint(physicals.GetExpPoints()) +
		"\t| Exp: " + fmt.Sprint(physicals.GetCurrentExp()) +
		" / " + fmt.Sprint(physicals.GetNextLvlBaseExp()) + "\n"

	physicalsLvl := hs.attribute.GetPhysicalsLevel()
	physicalsExp := hs.attribute.GetPhysicalsExpPoints()
	physicalsCurrExp := hs.attribute.GetPhysicalsCurrentExp()
	physicalsNextLvlExp := hs.attribute.GetPhysicalsNextLvlBaseExp()
	sortAttrNames := []enum.AttributeName{
		enum.Resistance, enum.Strength, enum.Agility, enum.ActionSpeed,
		enum.Flexibility, enum.Dexterity, enum.Sense, enum.Constitution,
	}
	for _, name := range sortAttrNames {
		lvl := physicalsLvl[name]
		exp := physicalsExp[name]
		currExp := physicalsCurrExp[name]
		nextLvlExp := physicalsNextLvlExp[name]
		sheet += name.String() + "\tLvl: " + fmt.Sprint(lvl) +
			"\t| Exp Total: " + fmt.Sprint(exp) +
			"\t| Exp: " + fmt.Sprint(currExp) +
			" / " + fmt.Sprint(nextLvlExp) + "\n"
	}
	sheet += "-----------------------------\n"

	skillsLvl := hs.skill.GetPhysicalsLevel()
	skillsExp := hs.skill.GetPhysicalsExpPoints()
	skillsCurrExp := hs.skill.GetPhysicalsCurrentExp()
	skillsNextLvlExp := hs.skill.GetPhysicalsNextLvlBaseExp()
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

		sheet += name.String() + "\tLvl: " + fmt.Sprint(lvl) +
			"\t| Exp Total: " + fmt.Sprint(exp) +
			"\t| Exp: " + fmt.Sprint(currExp) +
			" / " + fmt.Sprint(nextLvlExp) + "\n"
	}
	sheet += "-----------------------------\n"

	mentals, _ := hs.ability.Get(enum.Mentals)
	sheet += "MENTALS LVL: " + fmt.Sprint(mentals.GetLevel()) +
		"\t| Bonus: " + fmt.Sprintf("%.1f\n", mentals.GetBonus())

	sheet += "Exp Total: " + fmt.Sprint(mentals.GetExpPoints()) +
		"\t| Exp: " + fmt.Sprint(mentals.GetCurrentExp()) +
		" / " + fmt.Sprint(mentals.GetNextLvlAggregateExp()) + "\n"
	sheet += "-----------------------------\n"

	statusList := hs.status.GetAllStatus()
	for name, status := range statusList {
		sheet += name.String() + ": Min " + fmt.Sprint(status.GetMin()) +
			"\t| : " + fmt.Sprint(status.GetCurrent()) +
			" / " + fmt.Sprint(status.GetMax()) + "\n"
	}
	sheet += "=============================\n"
	return sheet
}
