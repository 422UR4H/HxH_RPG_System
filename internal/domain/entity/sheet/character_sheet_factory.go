package charactersheet

import (
	"errors"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/status"
)

const (
	CHARACTER_COEFF           = 10.0
	TALENT_COEFF              = 10.0
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

func (csf *CharacterSheetFactory) Build(profile CharacterProfile) *CharacterSheet {
	exp := experience.NewExperience(experience.NewExpTable(CHARACTER_COEFF))
	characterExp := experience.NewCharacterExp(*exp)

	abilities := csf.BuildPersonAbilities(*characterExp)

	physAbility, _ := abilities.Get(enum.Physicals)
	physAttrs := csf.BuildPhysAttrs(&physAbility)

	mentalAbility, _ := abilities.Get(enum.Mentals)
	mentalAttrs := csf.BuildMentalAttrs(&mentalAbility)

	spiritualAbility, _ := abilities.Get(enum.Spirituals)
	spiritAttrs := csf.BuildSpiritualAttrs(&spiritualAbility)

	characterAttrs := attribute.NewCharacterAttributes(
		*physAttrs, *mentalAttrs, *spiritAttrs,
	)

	status := csf.BuildStatusManager()

	skills, _ := abilities.Get(enum.Skills)
	physSkills := csf.BuildPhysSkills(
		*status, &skills, &physAbility, physAttrs,
	)
	mentalSkills := csf.BuildMentalSkills(
		&skills, &mentalAbility, mentalAttrs,
	)
	spiritSkills := csf.BuildSpiritualSkills(
		&skills, &spiritualAbility, spiritAttrs,
	)
	characterSkills := skill.NewCharacterSkills(
		*physSkills, *mentalSkills, *spiritSkills,
	)

	aura, _ := status.Get(enum.Aura)
	hatsu := csf.BuildHatsu(&spiritualAbility)
	spiritPrinciples := csf.BuildSpiritPrinciples(aura, &spiritualAbility, hatsu)

	return NewCharacterSheet(
		profile,
		*abilities,
		*characterAttrs,
		*spiritPrinciples,
		*characterSkills,
		*status,
	)
}

func (csf *CharacterSheetFactory) BuildPersonAbilities(
	characterExp experience.CharacterExp,
) *ability.Manager {

	abilities := make(map[enum.AbilityName]ability.Ability)

	talentExp := experience.NewExperience(experience.NewExpTable(TALENT_COEFF))
	talent := ability.NewTalent(*talentExp)

	physicalExp := experience.NewExperience(experience.NewExpTable(PHYSICAL_COEFF))
	abilities[enum.Physicals] = *ability.NewAbility(*physicalExp, characterExp)

	mentalExp := experience.NewExperience(experience.NewExpTable(MENTAL_COEFF))
	abilities[enum.Mentals] = *ability.NewAbility(*mentalExp, characterExp)

	spiritualExp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_COEFF))
	abilities[enum.Spirituals] = *ability.NewAbility(*spiritualExp, characterExp)

	skillsExp := experience.NewExperience(experience.NewExpTable(SKILLS_COEFF))
	abilities[enum.Skills] = *ability.NewAbility(*skillsExp, characterExp)

	return ability.NewAbilitiesManager(characterExp, abilities, *talent)
}

func (csf *CharacterSheetFactory) BuildPhysAttrs(
	physAbility ability.IAbility,
) *attribute.Manager {

	primaryAttributes := make(map[enum.AttributeName]attribute.PrimaryAttribute)
	middleAttributes := make(map[enum.AttributeName]attribute.MiddleAttribute)

	exp := experience.NewExperience(experience.NewExpTable(PHYSICAL_ATTRIBUTE_COEFF))
	primaryAttribute := attribute.NewPrimaryAttribute(*exp, physAbility)

	constitution := primaryAttribute.Clone()
	strength := primaryAttribute.Clone()
	defense := attribute.NewMiddleAttribute(*exp.Clone(), *constitution, *strength)
	primaryAttributes[enum.Constitution] = *constitution
	primaryAttributes[enum.Strength] = *strength
	middleAttributes[enum.Defense] = *defense

	agility := primaryAttribute.Clone()
	velocity := attribute.NewMiddleAttribute(*exp.Clone(), *strength, *agility)
	primaryAttributes[enum.Agility] = *agility
	middleAttributes[enum.Velocity] = *velocity

	flexibility := primaryAttribute.Clone()
	actionSpeed := attribute.NewMiddleAttribute(*exp.Clone(), *agility, *flexibility)
	primaryAttributes[enum.Flexibility] = *flexibility
	middleAttributes[enum.ActionSpeed] = *actionSpeed

	sense := primaryAttribute.Clone()
	dexterity := attribute.NewMiddleAttribute(*exp.Clone(), *flexibility, *sense)
	primaryAttributes[enum.Sense] = *sense
	middleAttributes[enum.Dexterity] = *dexterity

	return attribute.NewAttributeManager(primaryAttributes, middleAttributes)
}

func (csf *CharacterSheetFactory) BuildMentalAttrs(
	mentalAbility ability.IAbility,
) *attribute.Manager {

	attrs := make(map[enum.AttributeName]attribute.PrimaryAttribute)

	exp := experience.NewExperience(experience.NewExpTable(MENTAL_ATTRIBUTE_COEFF))
	attr := attribute.NewPrimaryAttribute(*exp, mentalAbility)

	attrs[enum.Resilience] = *attr.Clone()
	attrs[enum.Adaptability] = *attr.Clone()
	attrs[enum.Weighting] = *attr.Clone()
	attrs[enum.Creativity] = *attr.Clone()

	// TODO: add middle attributes which primary attributes above
	return attribute.NewAttributeManager(
		attrs, make(map[enum.AttributeName]attribute.MiddleAttribute),
	)
}

func (csf *CharacterSheetFactory) BuildSpiritualAttrs(
	spiritualAbility ability.IAbility,
) *attribute.Manager {

	attrs := make(map[enum.AttributeName]attribute.PrimaryAttribute)

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_ATTRIBUTE_COEFF))
	attr := attribute.NewPrimaryAttribute(*exp, spiritualAbility)

	attrs[enum.Spirit] = *attr

	// TODO: maybe add middle attributes which primary attributes above
	return attribute.NewAttributeManager(
		attrs, make(map[enum.AttributeName]attribute.MiddleAttribute),
	)
}

func (csf *CharacterSheetFactory) BuildStatusManager() *status.Manager {
	status_bars := make(map[enum.StatusName]status.Bar)

	status_bars[enum.Stamina] = *status.NewStatusBar()
	status_bars[enum.Health] = *status.NewStatusBar()
	status_bars[enum.Aura] = *status.NewStatusBar()

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

	con, err := physAttrs.Get(enum.Constitution)
	if err != nil {
		panic(errors.New("attribute not found"))
	}

	health, _ := status.Get(enum.Health)
	vitSkill := skill.NewPassiveSkill(health, *exp.Clone(), con, physSkills)
	skills[enum.Vitality] = vitSkill

	stamina, _ := status.Get(enum.Stamina)
	resSkill := skill.NewPassiveSkill(stamina, *exp.Clone(), con, physSkills)
	skills[enum.Resistance] = resSkill

	conSkill := skill.NewCommonSkill(*exp.Clone(), con, physSkills)
	skills[enum.Breath] = conSkill.Clone()
	skills[enum.Heal] = conSkill.Clone()

	def, err := physAttrs.Get(enum.Defense)
	if err != nil {
		panic(errors.New("attribute not found"))
	}

	defSkill := skill.NewCommonSkill(*exp.Clone(), def, physSkills)
	skills[enum.Defense] = defSkill.Clone()

	str, err := physAttrs.Get(enum.Strength)
	if err != nil {
		panic(errors.New("attribute not found"))
	}

	strSkill := skill.NewCommonSkill(*exp.Clone(), str, physSkills)
	skills[enum.Climb] = strSkill.Clone()
	skills[enum.Push] = strSkill.Clone()
	skills[enum.Grab] = strSkill.Clone()
	skills[enum.CarryCapacity] = strSkill.Clone()

	vel, err := physAttrs.Get(enum.Velocity)
	if err != nil {
		panic(errors.New("attribute not found"))
	}

	velSkill := skill.NewCommonSkill(*exp.Clone(), vel, physSkills)
	skills[enum.Run] = velSkill.Clone()
	skills[enum.Swim] = velSkill.Clone()
	skills[enum.Jump] = velSkill.Clone()

	agi, err := physAttrs.Get(enum.Agility)
	if err != nil {
		panic(errors.New("attribute not found"))
	}

	agiSkill := skill.NewCommonSkill(*exp.Clone(), agi, physSkills)
	skills[enum.Dodge] = agiSkill.Clone()
	skills[enum.Accelerate] = agiSkill.Clone()
	skills[enum.Brake] = agiSkill.Clone()

	ats, err := physAttrs.Get(enum.ActionSpeed)
	if err != nil {
		panic(errors.New("attribute not found"))
	}

	atsSkill := skill.NewCommonSkill(*exp.Clone(), ats, physSkills)
	skills[enum.ActionSpeed] = atsSkill.Clone()
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
	skills[enum.SleightOfHand] = dexSkill.Clone()

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
	abilityExp experience.ICascadeUpgrade,
) *spiritual.Hatsu {

	categories := make(map[enum.CategoryName]spiritual.NenCategory)

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_PRINCIPLE_COEFF))
	hatsu := spiritual.NewHatsu(*exp, abilityExp, categories)

	category := spiritual.NewNenCategory(*exp.Clone(), hatsu)
	for _, name := range enum.AllNenCategoryNames() {
		categories[name] = *category.Clone()
	}

	hatsu.Init(categories)
	return hatsu
}

func (csf *CharacterSheetFactory) BuildSpiritPrinciples(
	aura status.Bar,
	spiritAbilityExp experience.ICascadeUpgrade,
	hatsu *spiritual.Hatsu,
) *spiritual.Manager {

	principles := make(map[enum.PrincipleName]spiritual.NenPrinciple)

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_PRINCIPLE_COEFF))
	principle := spiritual.NewNenPrinciple(*exp, spiritAbilityExp)

	for _, name := range enum.AllNenPrincipleNames() {
		if name == enum.Hatsu {
			continue
		}
		// TODO: resolve aura\mop
		// if name == enum.Mop {
		// 	principles[name] = *spiritual.NewNenStatus(aura, *exp.Clone(), spiritAbilityExp)
		// 	continue
		// }
		principles[name] = *principle.Clone()
	}
	return spiritual.NewPrinciplesManager(principles, hatsu)
}
