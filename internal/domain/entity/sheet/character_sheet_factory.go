package sheet

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/status"
)

const (
	CHARACTER_COEFF           = 10.0
	TALENT_COEFF              = 2.0
	PHYSICAL_COEFF            = 20.0
	MENTAL_COEFF              = 15.0
	SPIRITUAL_COEFF           = 5.0
	SKILLS_COEFF              = 5.0
	PHYSICAL_ATTRIBUTE_COEFF  = 5.0
	MENTAL_ATTRIBUTE_COEFF    = 3.0
	SPIRITUAL_ATTRIBUTE_COEFF = 1.0
	PHYSICAL_SKILLS_COEFF     = 1.0
	MENTAL_SKILLS_COEFF       = 2.0
	SPIRITUAL_SKILLS_COEFF    = 3.0
	SPIRITUAL_PRINCIPLE_COEFF = 1.0
)

type CharacterSheetFactory struct{}

func NewCharacterSheetFactory() *CharacterSheetFactory {
	return &CharacterSheetFactory{}
}

func (csf *CharacterSheetFactory) Build(
	profile CharacterProfile,
	categorySet *TalentByCategorySet,
	charClass *cc.CharacterClass,
) *CharacterSheet {

	characterExp := csf.BuildCharacterExp()

	talentLvl := categorySet.GetTalentLvl()
	abilities := csf.BuildPersonAbilities(characterExp, talentLvl)

	physAbility, _ := abilities.Get(enum.Physicals)
	physAttrs := csf.BuildPhysAttrs(physAbility)

	mentalAbility, _ := abilities.Get(enum.Mentals)
	mentalAttrs := csf.BuildMentalAttrs(mentalAbility)

	spiritualAbility, _ := abilities.Get(enum.Spirituals)
	spiritAttrs := csf.BuildSpiritualAttrs(spiritualAbility)

	characterAttrs := attribute.NewCharacterAttributes(
		physAttrs, mentalAttrs, spiritAttrs,
	)

	status := csf.BuildStatusManager()

	skills, _ := abilities.Get(enum.Skills)
	physSkills := csf.BuildPhysSkills(
		*status, skills, physAbility, physAttrs,
	)
	mentalSkills := csf.BuildMentalSkills(
		skills, mentalAbility, mentalAttrs,
	)
	spiritSkills := csf.BuildSpiritualSkills(
		skills, spiritualAbility, spiritAttrs,
	)
	characterSkills := skill.NewCharacterSkills(
		physSkills, mentalSkills, spiritSkills,
	)

	var nenHexagon *spiritual.NenHexagon
	var categoryPercents map[enum.CategoryName]float64
	if categorySet.initialHexValue != nil {
		nenHexagon = spiritual.NewNenHexagon(*categorySet.initialHexValue)
		categoryPercents = nenHexagon.GetCategoryPercents()
	}
	hatsu := csf.BuildHatsu(spiritualAbility, categoryPercents)
	aura, _ := status.Get(enum.Aura)

	spiritPrinciples := csf.BuildSpiritPrinciples(
		aura, spiritualAbility, nenHexagon, hatsu,
	)

	proficiency := proficiency.NewManager()

	var className enum.CharacterClassName
	if charClass != nil {
		className = charClass.GetName()
	}
	charSheet := NewCharacterSheet(
		profile,
		*abilities,
		*characterAttrs,
		*spiritPrinciples,
		*characterSkills,
		*proficiency,
		*status,
		&className,
	)
	if charClass != nil {
		charSheet = csf.Wrap(charSheet, charClass)
	}
	return charSheet
}

func (csf *CharacterSheetFactory) BuildCharacterExp() *experience.CharacterExp {
	expTable := experience.NewExpTable(CHARACTER_COEFF)
	exp := experience.NewExperience(expTable)
	return experience.NewCharacterExp(*exp)
}

func (csf *CharacterSheetFactory) BuildPersonAbilities(
	characterExp *experience.CharacterExp, talentLvl int,
) *ability.Manager {

	abilities := make(map[enum.AbilityName]ability.IAbility)

	talentExp := experience.NewExperience(experience.NewExpTable(TALENT_COEFF))
	talent := ability.NewTalent(*talentExp, talentLvl)

	physicalExp := experience.NewExperience(experience.NewExpTable(PHYSICAL_COEFF))
	abilities[enum.Physicals] = ability.NewAbility(*physicalExp, characterExp)

	mentalExp := experience.NewExperience(experience.NewExpTable(MENTAL_COEFF))
	abilities[enum.Mentals] = ability.NewAbility(*mentalExp, characterExp)

	spiritualExp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_COEFF))
	abilities[enum.Spirituals] = ability.NewAbility(*spiritualExp, characterExp)

	skillsExp := experience.NewExperience(experience.NewExpTable(SKILLS_COEFF))
	abilities[enum.Skills] = ability.NewAbility(*skillsExp, characterExp)

	return ability.NewAbilitiesManager(characterExp, abilities, *talent)
}

func (csf *CharacterSheetFactory) BuildPhysAttrs(
	physAbility ability.IAbility,
) *attribute.Manager {

	primaryAttrs := make(map[enum.AttributeName]*attribute.PrimaryAttribute)
	middleAttrs := make(map[enum.AttributeName]*attribute.MiddleAttribute)
	buffs := csf.BuildPhysAttrBuffs()

	exp := experience.NewExperience(experience.NewExpTable(PHYSICAL_ATTRIBUTE_COEFF))
	primAttr := attribute.NewPrimaryAttribute(*exp, physAbility, buffs[0])

	res := primAttr.Clone(buffs[enum.Resistance])
	agi := primAttr.Clone(buffs[enum.Agility])
	str := attribute.NewMiddleAttribute(*exp.Clone(), buffs[enum.Strength], res, agi)
	primaryAttrs[enum.Resistance] = res
	primaryAttrs[enum.Agility] = agi
	middleAttrs[enum.Strength] = str

	flx := primAttr.Clone(buffs[enum.Flexibility])
	ats := attribute.NewMiddleAttribute(*exp.Clone(), buffs[enum.ActionSpeed], agi, flx)
	primaryAttrs[enum.Flexibility] = flx
	middleAttrs[enum.ActionSpeed] = ats

	sen := primAttr.Clone(buffs[enum.Sense])
	dex := attribute.NewMiddleAttribute(*exp.Clone(), buffs[enum.Dexterity], flx, sen)
	primaryAttrs[enum.Sense] = sen
	middleAttrs[enum.Dexterity] = dex

	con := attribute.NewMiddleAttribute(*exp.Clone(), buffs[enum.Constitution], sen, res)
	middleAttrs[enum.Constitution] = con

	return attribute.NewAttributeManager(primaryAttrs, middleAttrs, buffs)
}

func (csf *CharacterSheetFactory) BuildMentalAttrs(
	mentalAbility ability.IAbility,
) *attribute.Manager {

	attrs := make(map[enum.AttributeName]*attribute.PrimaryAttribute)
	buffs := csf.BuildMentalAttrBuffs()

	exp := experience.NewExperience(experience.NewExpTable(MENTAL_ATTRIBUTE_COEFF))
	attr := attribute.NewPrimaryAttribute(*exp, mentalAbility, buffs[0])

	attrs[enum.Resilience] = attr.Clone(buffs[enum.Resilience])
	attrs[enum.Adaptability] = attr.Clone(buffs[enum.Adaptability])
	attrs[enum.Weighting] = attr.Clone(buffs[enum.Weighting])
	attrs[enum.Creativity] = attr.Clone(buffs[enum.Creativity])

	// TODO: add middle attributes which primary attributes above
	return attribute.NewAttributeManager(
		attrs, make(map[enum.AttributeName]*attribute.MiddleAttribute), buffs,
	)
}

func (csf *CharacterSheetFactory) BuildSpiritualAttrs(
	spiritualAbility ability.IAbility,
) *attribute.Manager {

	attrs := make(map[enum.AttributeName]*attribute.PrimaryAttribute)
	buffs := csf.BuildSpiritAttrsBuffs()

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_ATTRIBUTE_COEFF))
	attr := attribute.NewPrimaryAttribute(*exp, spiritualAbility, buffs[enum.Spirit])

	attrs[enum.Spirit] = attr

	// TODO: maybe add middle attributes which primary attributes above
	return attribute.NewAttributeManager(
		attrs, make(map[enum.AttributeName]*attribute.MiddleAttribute), buffs,
	)
}

func (csf *CharacterSheetFactory) BuildStatusManager() *status.Manager {
	status_bars := make(map[enum.StatusName]status.IStatusBar)

	status_bars[enum.Stamina] = status.NewStatusBar()
	status_bars[enum.Health] = status.NewStatusBar()
	status_bars[enum.Aura] = status.NewStatusBar()

	return status.NewStatusManager(status_bars)
}

func (csf *CharacterSheetFactory) BuildPhysSkills(
	status status.Manager,
	skillsExp experience.ICascadeUpgrade,
	physAbilityExp experience.ICascadeUpgrade,
	physAttrs *attribute.Manager,
) *skill.Manager {

	skills := make(map[enum.SkillName]skill.ISkill)

	exp := experience.NewExperience(experience.NewExpTable(PHYSICAL_SKILLS_COEFF))
	physSkills := skill.NewSkillsManager(*exp, skillsExp, physAbilityExp)

	res, err := physAttrs.Get(enum.Resistance)
	if err != nil {
		panic(errors.New("attribute not found"))
	}
	health, _ := status.Get(enum.Health)
	vitSkill := skill.NewPassiveSkill(health, *exp.Clone(), res, physSkills)
	skills[enum.Vitality] = vitSkill

	stamina, _ := status.Get(enum.Stamina)
	engSkill := skill.NewPassiveSkill(stamina, *exp.Clone(), res, physSkills)
	skills[enum.Energy] = engSkill

	skills[enum.Defense] = skill.NewCommonSkill(*exp.Clone(), res, physSkills)

	str, err := physAttrs.Get(enum.Strength)
	if err != nil {
		panic(errors.New("attribute not found"))
	}
	strSkill := skill.NewCommonSkill(*exp.Clone(), str, physSkills)
	skills[enum.Push] = strSkill.Clone()
	skills[enum.Grab] = strSkill.Clone()
	skills[enum.CarryCapacity] = strSkill.Clone()

	agi, err := physAttrs.Get(enum.Agility)
	if err != nil {
		panic(errors.New("attribute not found"))
	}
	velSkill := skill.NewCommonSkill(*exp.Clone(), agi, physSkills)
	skills[enum.Velocity] = velSkill.Clone()
	skills[enum.Accelerate] = velSkill.Clone()
	skills[enum.Brake] = velSkill.Clone()

	ats, err := physAttrs.Get(enum.ActionSpeed)
	if err != nil {
		panic(errors.New("attribute not found"))
	}
	atsSkill := skill.NewCommonSkill(*exp.Clone(), ats, physSkills)
	skills[enum.AttackSpeed] = atsSkill.Clone()
	skills[enum.Feint] = atsSkill.Clone()

	flx, err := physAttrs.Get(enum.Flexibility)
	if err != nil {
		panic(errors.New("attribute not found"))
	}
	flxSkill := skill.NewCommonSkill(*exp.Clone(), flx, physSkills)
	skills[enum.Acrobatics] = flxSkill.Clone()
	skills[enum.Sneak] = flxSkill.Clone()

	dex, err := physAttrs.Get(enum.Dexterity)
	if err != nil {
		panic(errors.New("attribute not found"))
	}
	dexSkill := skill.NewCommonSkill(*exp.Clone(), dex, physSkills)
	skills[enum.Reflex] = dexSkill.Clone()
	skills[enum.Accuracy] = dexSkill.Clone()
	skills[enum.Stealth] = dexSkill.Clone()

	sen, err := physAttrs.Get(enum.Sense)
	if err != nil {
		panic(errors.New("attribute not found"))
	}
	senSkill := skill.NewCommonSkill(*exp.Clone(), sen, physSkills)
	skills[enum.Vision] = senSkill.Clone()
	skills[enum.Hearing] = senSkill.Clone()
	skills[enum.Smell] = senSkill.Clone()
	skills[enum.Tact] = senSkill.Clone()
	skills[enum.Taste] = senSkill.Clone()
	skills[enum.Balance] = senSkill.Clone()

	con, err := physAttrs.Get(enum.Constitution)
	if err != nil {
		panic(errors.New("attribute not found"))
	}
	conSkill := skill.NewCommonSkill(*exp.Clone(), con, physSkills)
	skills[enum.Heal] = conSkill.Clone()
	skills[enum.Breath] = conSkill.Clone()
	skills[enum.Tenacity] = conSkill.Clone()

	physSkills.Init(skills)
	return physSkills
}

func (csf *CharacterSheetFactory) BuildMentalSkills(
	skillsExp experience.ICascadeUpgrade,
	mentalAbilityExp experience.ICascadeUpgrade,
	mentalsAttrs *attribute.Manager,
) *skill.Manager {
	// skills := make(map[enum.SkillName]skill.ISkill)

	exp := experience.NewExperience(experience.NewExpTable(MENTAL_SKILLS_COEFF))
	mentalSkills := skill.NewSkillsManager(*exp, skillsExp, mentalAbilityExp)

	return mentalSkills
}

func (csf *CharacterSheetFactory) BuildSpiritualSkills(
	skillsExp experience.ICascadeUpgrade,
	spiritualAbilityExp experience.ICascadeUpgrade,
	spiritualsAttrs *attribute.Manager,
) *skill.Manager {

	skills := make(map[enum.SkillName]skill.ISkill)

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_SKILLS_COEFF))
	spiritualSkills := skill.NewSkillsManager(*exp, skillsExp, spiritualAbilityExp)

	spr, err := spiritualsAttrs.Get(enum.Spirit)
	if err != nil {
		panic(errors.New("attribute not found"))
	}

	skill := skill.NewCommonSkill(*exp.Clone(), spr, spiritualSkills)
	skills[enum.Nen] = skill.Clone()
	skills[enum.Focus] = skill.Clone()
	skills[enum.WillPower] = skill.Clone()

	spiritualSkills.Init(skills)
	return spiritualSkills
}

func (csf *CharacterSheetFactory) BuildHatsu(
	ability ability.IAbility,
	categoryPercents map[enum.CategoryName]float64,
) *spiritual.Hatsu {

	categories := make(map[enum.CategoryName]spiritual.NenCategory)

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_PRINCIPLE_COEFF))
	hatsu := spiritual.NewHatsu(*exp, ability, categories, categoryPercents)

	category := spiritual.NewNenCategory(*exp.Clone(), enum.Reinforcement, hatsu)
	for _, name := range enum.AllNenCategoryNames() {
		categories[name] = *category.Clone(name)
	}

	hatsu.Init(categories)
	return hatsu
}

func (csf *CharacterSheetFactory) BuildSpiritPrinciples(
	aura status.IStatusBar,
	spiritAbility ability.IAbility,
	nenHexagon *spiritual.NenHexagon,
	hatsu *spiritual.Hatsu,
) *spiritual.Manager {

	principles := make(map[enum.PrincipleName]spiritual.NenPrinciple)

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_PRINCIPLE_COEFF))
	principle := spiritual.NewNenPrinciple(*exp, spiritAbility)

	for _, name := range enum.AllNenPrincipleNames() {
		if name == enum.Hatsu {
			continue
		}
		// TODO: resolve aura\mop
		// if name == enum.Mop {
		// 	principles[name] = *spiritual.NewNenStatus(aura, *exp.Clone(), spiritAbility)
		// 	continue
		// }
		principles[name] = *principle.Clone()
	}
	return spiritual.NewPrinciplesManager(principles, nenHexagon, hatsu)
}

func (csf *CharacterSheetFactory) BuildPhysAttrBuffs() map[enum.AttributeName]*int {
	buffs := make(map[enum.AttributeName]*int)

	buffs[enum.Resistance] = new(int)
	buffs[enum.Strength] = new(int)
	buffs[enum.Agility] = new(int)
	buffs[enum.ActionSpeed] = new(int)
	buffs[enum.Flexibility] = new(int)
	buffs[enum.Dexterity] = new(int)
	buffs[enum.Sense] = new(int)
	buffs[enum.Constitution] = new(int)

	return buffs
}

func (csf *CharacterSheetFactory) BuildMentalAttrBuffs() map[enum.AttributeName]*int {
	buffs := make(map[enum.AttributeName]*int)

	buffs[enum.Resilience] = new(int)
	buffs[enum.Adaptability] = new(int)
	buffs[enum.Weighting] = new(int)
	buffs[enum.Creativity] = new(int)

	return buffs
}

func (csf *CharacterSheetFactory) BuildSpiritAttrsBuffs() map[enum.AttributeName]*int {
	buffs := make(map[enum.AttributeName]*int)

	buffs[enum.Spirit] = new(int)

	return buffs
}

func (csf *CharacterSheetFactory) Wrap(
	charSheet *CharacterSheet, charClass *cc.CharacterClass,
) *CharacterSheet {
	for name, exp := range charClass.SkillsExps {
		charSheet.IncreaseExpForSkill(exp, name)
	}
	for _, skill := range charClass.JointSkills {
		charSheet.AddJointSkill(&skill)
	}
	for name, exp := range charClass.ProficienciesExps {
		charSheet.IncreaseExpForProficiency(exp, name)
	}
	for _, prof := range charClass.JointProficiencies {
		charSheet.AddJointProficiency(&prof)
	}
	for name, exp := range charClass.AttributesExps {
		charSheet.IncreaseExpForMentals(exp, name)
	}
	return charSheet
}

func (csf *CharacterSheetFactory) BuildHalfSheet(
	profile CharacterProfile,
	categorySet *TalentByCategorySet,
	charClass *cc.CharacterClass,
) *HalfSheet {
	expTable := experience.NewExpTable(CHARACTER_COEFF)
	exp := experience.NewExperience(expTable)
	characterExp := experience.NewCharacterExp(*exp)

	var talentLvl int
	if categorySet == nil {
		talentLvl = BASE_TALENT_LVL
	} else {
		talentLvl = categorySet.GetTalentLvl()
	}
	abilities := csf.BuildPersonAbilitiesHalf(characterExp, talentLvl)

	physAbility, _ := abilities.Get(enum.Physicals)
	physAttrs := csf.BuildPhysAttrs(physAbility)

	mentalAbility, _ := abilities.Get(enum.Mentals)
	mentalAttrs := csf.BuildMentalAttrs(mentalAbility)

	characterAttrs := attribute.NewCharacterAttributes(
		physAttrs, mentalAttrs, nil,
	)

	status := csf.BuildStatusManager()

	skills, _ := abilities.Get(enum.Skills)
	physSkills := csf.BuildPhysSkills(
		*status, skills, physAbility, physAttrs,
	)
	mentalSkills := csf.BuildMentalSkills(
		skills, mentalAbility, mentalAttrs,
	)
	characterSkills := skill.NewCharacterSkills(
		physSkills, mentalSkills, nil,
	)

	proficiency := proficiency.NewManager()

	var className enum.CharacterClassName
	if charClass != nil {
		className = charClass.GetName()
	}
	sheet := NewHalfSheet(
		profile,
		*abilities,
		*characterAttrs,
		*characterSkills,
		*proficiency,
		*status,
		&className,
	)
	if charClass != nil {
		sheet = csf.WrapHalf(sheet, charClass)
	}
	return sheet

}

func (csf *CharacterSheetFactory) BuildPersonAbilitiesHalf(
	characterExp *experience.CharacterExp, talentLvl int,
) *ability.Manager {

	abilities := make(map[enum.AbilityName]ability.IAbility)

	talentExp := experience.NewExperience(experience.NewExpTable(TALENT_COEFF))
	talent := ability.NewTalent(*talentExp, talentLvl)

	physicalExp := experience.NewExperience(experience.NewExpTable(PHYSICAL_COEFF))
	abilities[enum.Physicals] = ability.NewAbility(*physicalExp, characterExp)

	mentalExp := experience.NewExperience(experience.NewExpTable(MENTAL_COEFF))
	abilities[enum.Mentals] = ability.NewAbility(*mentalExp, characterExp)

	skillsExp := experience.NewExperience(experience.NewExpTable(SKILLS_COEFF))
	abilities[enum.Skills] = ability.NewAbility(*skillsExp, characterExp)

	return ability.NewAbilitiesManager(characterExp, abilities, *talent)
}

func (csf *CharacterSheetFactory) WrapHalf(
	sheet *HalfSheet, charClass *cc.CharacterClass,
) *HalfSheet {
	for name, exp := range charClass.SkillsExps {
		sheet.IncreaseExpForSkill(exp, name)
	}
	for _, skill := range charClass.JointSkills {
		sheet.AddJointSkill(&skill)
	}
	for name, exp := range charClass.ProficienciesExps {
		sheet.IncreaseExpForProficiency(exp, name)
	}
	for _, prof := range charClass.JointProficiencies {
		sheet.AddJointProficiency(&prof)
	}
	for name, exp := range charClass.AttributesExps {
		sheet.IncreaseExpForMentals(exp, name)
	}
	return sheet
}
