package sheet

import (
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

// TODO: strike the gavel about these changes
const (
	CHARACTER_COEFF           = 10.0
	TALENT_COEFF              = 2.0
	PHYSICAL_COEFF            = 20.0
	MENTAL_COEFF              = 20.0 // 15.0
	SPIRITUAL_COEFF           = 5.0
	SKILLS_COEFF              = 20.0 // 5.0
	PHYSICAL_ATTRIBUTE_COEFF  = 5.0
	MENTAL_ATTRIBUTE_COEFF    = 1.0 // 3.0
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
	playerUUID *uuid.UUID,
	masterUUID *uuid.UUID,
	campaignUUID *uuid.UUID,
	profile CharacterProfile,
	hexValue *int,
	category *enum.CategoryName,
	charClass *cc.CharacterClass,
) (*CharacterSheet, error) {

	characterExp := csf.BuildCharacterExp()
	abilities := csf.BuildPersonAbilities(characterExp)

	physAbility, _ := abilities.Get(enum.Physicals)
	physAttrs := csf.BuildPhysAttrs(physAbility)

	mentalAbility, _ := abilities.Get(enum.Mentals)
	mentalAttrs := csf.BuildMentalAttrs(mentalAbility)

	spiritAbility, _ := abilities.Get(enum.Spirituals)
	spiritAttrs := csf.BuildSpiritualAttrs(spiritAbility)

	charAttrs := attribute.NewCharacterAttributes(physAttrs, mentalAttrs, spiritAttrs)

	skills, _ := abilities.Get(enum.Skills)
	physSkills, err := csf.BuildPhysSkills(skills, physAttrs)
	if err != nil {
		return nil, err
	}

	mentalSkills := csf.BuildMentalSkills(skills, mentalAttrs)

	spiritSkills, err := csf.BuildSpiritualSkills(skills, spiritAttrs)
	if err != nil {
		return nil, err
	}
	charSkills := skill.NewCharacterSkills(physSkills, mentalSkills, spiritSkills)

	var nenHexagon *spiritual.NenHexagon
	var categoryPercents map[enum.CategoryName]float64
	if hexValue != nil {
		nenHexagon = spiritual.NewNenHexagon(*hexValue, category)
		categoryPercents = nenHexagon.GetCategoryPercents()
	}
	flame, _ := spiritAttrs.Get(enum.Flame)
	conscience, _ := spiritAttrs.Get(enum.Conscience)
	hatsu := csf.BuildHatsu(flame, conscience, categoryPercents)
	// aura, _ := status.Get(enum.Aura)
	spiritPrinciples := csf.BuildSpiritPrinciples(flame, conscience, nenHexagon, hatsu)

	proficiency := proficiency.NewManager()

	var className enum.CharacterClassName
	if charClass != nil {
		className = charClass.GetName()
	}
	status := csf.BuildStatusManager(abilities, charAttrs, charSkills)

	charSheet, err := NewCharacterSheet(
		playerUUID,
		masterUUID,
		campaignUUID,
		profile,
		*abilities,
		*charAttrs,
		*spiritPrinciples,
		*charSkills,
		*proficiency,
		*status,
		&className,
	)
	if err != nil {
		return nil, err
	}

	if charClass != nil {
		return csf.Wrap(charSheet, charClass)
	}
	return charSheet, nil
}

func (csf *CharacterSheetFactory) BuildCharacterExp() *experience.CharacterExp {
	expTable := experience.NewExpTable(CHARACTER_COEFF)
	exp := experience.NewExperience(expTable)
	return experience.NewCharacterExp(*exp)
}

func (csf *CharacterSheetFactory) BuildPersonAbilities(
	characterExp *experience.CharacterExp,
) *ability.Manager {

	abilities := make(map[enum.AbilityName]ability.IAbility)

	talentExp := experience.NewExperience(experience.NewExpTable(TALENT_COEFF))
	talent := ability.NewTalent(*talentExp)

	physicalExp := experience.NewExperience(experience.NewExpTable(PHYSICAL_COEFF))
	abilities[enum.Physicals] = ability.NewAbility(enum.Physicals, *physicalExp, characterExp)

	mentalExp := experience.NewExperience(experience.NewExpTable(MENTAL_COEFF))
	abilities[enum.Mentals] = ability.NewAbility(enum.Mentals, *mentalExp, characterExp)

	spiritualExp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_COEFF))
	abilities[enum.Spirituals] = ability.NewAbility(enum.Spirituals, *spiritualExp, characterExp)

	skillsExp := experience.NewExperience(experience.NewExpTable(SKILLS_COEFF))
	abilities[enum.Skills] = ability.NewAbility(enum.Skills, *skillsExp, characterExp)

	return ability.NewAbilitiesManager(characterExp, abilities, *talent)
}

func (csf *CharacterSheetFactory) BuildPhysAttrs(
	physAbility ability.IAbility,
) *attribute.Manager {

	primaryAttrs := make(map[enum.AttributeName]*attribute.PrimaryAttribute)
	middleAttrs := make(map[enum.AttributeName]*attribute.MiddleAttribute)
	buffs := csf.BuildPhysAttrBuffs()

	exp := experience.NewExperience(experience.NewExpTable(PHYSICAL_ATTRIBUTE_COEFF))
	primAttr := attribute.NewPrimaryAttribute(
		enum.Resistance, *exp, physAbility, buffs[enum.Resistance],
	)

	res := primAttr
	agi := primAttr.Clone(enum.Agility, buffs[enum.Agility])
	str := attribute.NewMiddleAttribute(
		enum.Strength, *exp.Clone(), buffs[enum.Strength], res, agi,
	)
	primaryAttrs[enum.Resistance] = res
	primaryAttrs[enum.Agility] = agi
	middleAttrs[enum.Strength] = str

	flx := primAttr.Clone(enum.Flexibility, buffs[enum.Flexibility])
	ats := attribute.NewMiddleAttribute(
		enum.Celerity, *exp.Clone(), buffs[enum.Celerity], agi, flx,
	)
	primaryAttrs[enum.Flexibility] = flx
	middleAttrs[enum.Celerity] = ats

	sen := primAttr.Clone(enum.Sense, buffs[enum.Sense])
	dex := attribute.NewMiddleAttribute(
		enum.Dexterity, *exp.Clone(), buffs[enum.Dexterity], flx, sen,
	)
	primaryAttrs[enum.Sense] = sen
	middleAttrs[enum.Dexterity] = dex

	con := attribute.NewMiddleAttribute(
		enum.Constitution, *exp.Clone(), buffs[enum.Constitution], sen, res,
	)
	middleAttrs[enum.Constitution] = con

	return attribute.NewAttributeManager(primaryAttrs, middleAttrs, buffs)
}

func (csf *CharacterSheetFactory) BuildMentalAttrs(
	mentalAbility ability.IAbility,
) *attribute.Manager {

	attrs := make(map[enum.AttributeName]*attribute.PrimaryAttribute)
	buffs := csf.BuildMentalAttrBuffs()

	exp := *experience.NewExperience(experience.NewExpTable(MENTAL_ATTRIBUTE_COEFF))
	attr := attribute.NewPrimaryAttribute(
		enum.Resilience, exp, mentalAbility, buffs[enum.Resilience],
	)

	attrs[enum.Resilience] = attr
	attrs[enum.Adaptability] = attr.Clone(enum.Adaptability, buffs[enum.Adaptability])
	attrs[enum.Weighting] = attr.Clone(enum.Weighting, buffs[enum.Weighting])
	attrs[enum.Creativity] = attr.Clone(enum.Creativity, buffs[enum.Creativity])

	// TODO: add middle attributes which primary attributes above
	return attribute.NewAttributeManager(
		attrs, nil, buffs,
	)
}

func (csf *CharacterSheetFactory) BuildSpiritualAttrs(
	spiritualAbility ability.IAbility,
) *attribute.SpiritualManager {

	attrs := make(map[enum.AttributeName]*attribute.SpiritualAttribute)
	buffs := csf.BuildSpiritAttrsBuffs()

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_ATTRIBUTE_COEFF))
	attr := attribute.NewSpiritualAttribute(
		enum.Flame, *exp, spiritualAbility, buffs[enum.Flame],
	)
	attrs[enum.Flame] = attr
	attrs[enum.Conscience] = attr.Clone(enum.Conscience, buffs[enum.Conscience])

	return attribute.NewSpiritualAttributeManager(attrs, buffs)
}

func (csf *CharacterSheetFactory) BuildStatusManager(
	abilities *ability.Manager,
	attrs *attribute.CharacterAttributes,
	skills *skill.CharacterSkills,
) *status.Manager {
	status_bars := make(map[enum.StatusName]status.IStatusBar)

	physAbility, _ := abilities.Get(enum.Physicals)
	resistance, _ := attrs.GetDistributable(enum.Resistance)
	vitality, _ := skills.Get(enum.Vitality)
	status_bars[enum.Health] = status.NewHealthPoints(physAbility, resistance, vitality)

	energy, _ := skills.Get(enum.Energy)
	status_bars[enum.Stamina] = status.NewStaminaPoints(physAbility, resistance, energy)

	spiritualAbility, _ := abilities.Get(enum.Spirituals)
	if spiritualAbility != nil {
		conscience, _ := attrs.Get(enum.Conscience)
		mop, _ := skills.Get(enum.Mop)
		status_bars[enum.Aura], _ = status.NewAuraPoints(spiritualAbility, conscience, mop)
	}

	return status.NewStatusManager(status_bars)
}

func (csf *CharacterSheetFactory) BuildPhysSkills(
	skillsExp experience.ICascadeUpgrade,
	physAttrs *attribute.Manager,
) (*skill.Manager, error) {

	skills := make(map[enum.SkillName]skill.ISkill)

	exp := experience.NewExperience(experience.NewExpTable(PHYSICAL_SKILLS_COEFF))
	physSkills := skill.NewSkillsManager(*exp, skillsExp)

	res, err := physAttrs.Get(enum.Resistance)
	if err != nil {
		return nil, err
	}
	resSkill := skill.NewCommonSkill(enum.Vitality, *exp.Clone(), res, physSkills)
	skills[enum.Vitality] = resSkill
	skills[enum.Energy] = resSkill.Clone(enum.Energy)
	skills[enum.Defense] = resSkill.Clone(enum.Defense)

	str, err := physAttrs.Get(enum.Strength)
	if err != nil {
		return nil, err
	}
	strSkill := skill.NewCommonSkill(enum.Push, *exp.Clone(), str, physSkills)
	skills[enum.Push] = strSkill
	skills[enum.Grab] = strSkill.Clone(enum.Grab)
	skills[enum.Carry] = strSkill.Clone(enum.Carry)

	agi, err := physAttrs.Get(enum.Agility)
	if err != nil {
		return nil, err
	}
	agiSkill := skill.NewCommonSkill(enum.Velocity, *exp.Clone(), agi, physSkills)
	skills[enum.Velocity] = agiSkill
	skills[enum.Accelerate] = agiSkill.Clone(enum.Accelerate)
	skills[enum.Brake] = agiSkill.Clone(enum.Brake)

	ats, err := physAttrs.Get(enum.Celerity)
	if err != nil {
		return nil, err
	}
	atsSkill := skill.NewCommonSkill(enum.Legerity, *exp.Clone(), ats, physSkills)
	skills[enum.Legerity] = atsSkill
	skills[enum.Repel] = atsSkill.Clone(enum.Repel)
	skills[enum.Feint] = atsSkill.Clone(enum.Feint)

	flx, err := physAttrs.Get(enum.Flexibility)
	if err != nil {
		return nil, err
	}
	flxSkill := skill.NewCommonSkill(enum.Acrobatics, *exp.Clone(), flx, physSkills)
	skills[enum.Acrobatics] = flxSkill
	skills[enum.Evasion] = flxSkill.Clone(enum.Evasion)
	skills[enum.Sneak] = flxSkill.Clone(enum.Sneak)

	dex, err := physAttrs.Get(enum.Dexterity)
	if err != nil {
		return nil, err
	}
	dexSkill := skill.NewCommonSkill(enum.Reflex, *exp.Clone(), dex, physSkills)
	skills[enum.Reflex] = dexSkill
	skills[enum.Accuracy] = dexSkill.Clone(enum.Accuracy)
	skills[enum.Stealth] = dexSkill.Clone(enum.Stealth)

	sen, err := physAttrs.Get(enum.Sense)
	if err != nil {
		return nil, err
	}
	senSkill := skill.NewCommonSkill(enum.Vision, *exp.Clone(), sen, physSkills)
	skills[enum.Vision] = senSkill
	skills[enum.Hearing] = senSkill.Clone(enum.Hearing)
	skills[enum.Smell] = senSkill.Clone(enum.Smell)
	skills[enum.Tact] = senSkill.Clone(enum.Tact)
	skills[enum.Taste] = senSkill.Clone(enum.Taste)

	con, err := physAttrs.Get(enum.Constitution)
	if err != nil {
		return nil, err
	}
	conSkill := skill.NewCommonSkill(enum.Heal, *exp.Clone(), con, physSkills)
	skills[enum.Heal] = conSkill
	skills[enum.Breath] = conSkill.Clone(enum.Breath)
	skills[enum.Tenacity] = conSkill.Clone(enum.Tenacity)

	if err := physSkills.Init(skills); err != nil {
		return nil, err
	}
	return physSkills, nil
}

func (csf *CharacterSheetFactory) BuildMentalSkills(
	skillsExp experience.ICascadeUpgrade,
	mentalsAttrs *attribute.Manager,
) *skill.Manager {
	// skills := make(map[enum.SkillName]skill.ISkill)

	exp := experience.NewExperience(experience.NewExpTable(MENTAL_SKILLS_COEFF))
	mentalSkills := skill.NewSkillsManager(*exp, skillsExp)

	return mentalSkills
}

func (csf *CharacterSheetFactory) BuildSpiritualSkills(
	skillsExp experience.ICascadeUpgrade,
	spiritualsAttrs *attribute.SpiritualManager,
) (*skill.Manager, error) {

	skills := make(map[enum.SkillName]skill.ISkill)

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_SKILLS_COEFF))
	spiritualSkills := skill.NewSkillsManager(*exp, skillsExp)

	flameNen, err := spiritualsAttrs.Get(enum.Flame)
	if err != nil {
		return nil, err
	}
	flameSkill := skill.NewCommonSkill(enum.Focus, *exp.Clone(), flameNen, spiritualSkills)
	skills[enum.Focus] = flameSkill
	skills[enum.WillPower] = flameSkill.Clone(enum.WillPower)
	skills[enum.SelfKnowledge] = flameSkill.Clone(enum.SelfKnowledge)

	conscienceNen, err := spiritualsAttrs.Get(enum.Conscience)
	if err != nil {
		return nil, err
	}
	conscienceSkill := skill.NewCommonSkill(enum.Coa, *exp.Clone(), conscienceNen, spiritualSkills)
	skills[enum.Coa] = conscienceSkill
	skills[enum.Mop] = conscienceSkill.Clone(enum.Mop)
	skills[enum.Aop] = conscienceSkill.Clone(enum.Aop)

	err = spiritualSkills.Init(skills)
	if err != nil {
		return nil, err
	}
	return spiritualSkills, nil
}

func (csf *CharacterSheetFactory) BuildHatsu(
	flameNen attribute.IGameAttribute,
	conscienceNen attribute.IGameAttribute,
	categoryPercents map[enum.CategoryName]float64,
) *spiritual.Hatsu {

	categories := make(map[enum.CategoryName]spiritual.NenCategory)

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_PRINCIPLE_COEFF))
	hatsu := spiritual.NewHatsu(*exp, flameNen, conscienceNen, categories, categoryPercents)

	category := spiritual.NewNenCategory(*exp.Clone(), enum.Reinforcement, hatsu)
	for _, name := range enum.AllNenCategoryNames() {
		categories[name] = *category.Clone(name)
	}

	hatsu.Init(categories)
	return hatsu
}

func (csf *CharacterSheetFactory) BuildSpiritPrinciples(
	flameNen attribute.IGameAttribute,
	conscienceNen attribute.IGameAttribute,
	nenHexagon *spiritual.NenHexagon,
	hatsu *spiritual.Hatsu,
) *spiritual.Manager {

	principles := make(map[enum.PrincipleName]spiritual.NenPrinciple)

	exp := experience.NewExperience(experience.NewExpTable(SPIRITUAL_PRINCIPLE_COEFF))
	principle := spiritual.NewNenPrinciple(enum.Ten, *exp, flameNen, conscienceNen)

	for _, name := range enum.AllNenPrincipleNames() {
		if name == enum.Hatsu {
			continue
		}
		principles[name] = *principle.Clone(name)
	}
	return spiritual.NewPrinciplesManager(principles, nenHexagon, hatsu)
}

func (csf *CharacterSheetFactory) BuildPhysAttrBuffs() map[enum.AttributeName]*int {
	buffs := make(map[enum.AttributeName]*int)

	buffs[enum.Resistance] = new(int)
	buffs[enum.Strength] = new(int)
	buffs[enum.Agility] = new(int)
	buffs[enum.Celerity] = new(int)
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

	buffs[enum.Flame] = new(int)
	buffs[enum.Conscience] = new(int)

	return buffs
}

func (csf *CharacterSheetFactory) Wrap(
	charSheet *CharacterSheet,
	charClass *cc.CharacterClass,
) (*CharacterSheet, error) {
	for name, exp := range charClass.SkillsExps {
		charSheet.IncreaseExpForSkill(experience.NewUpgradeCascade(exp), name)
	}
	for _, skill := range charClass.JointSkills {
		charSheet.AddJointSkill(&skill)
	}
	for _, prof := range charClass.JointProficiencies {
		charSheet.AddJointProficiency(&prof)
	}
	for name, exp := range charClass.AttributesExps {
		charSheet.IncreaseExpForMentals(experience.NewUpgradeCascade(exp), name)
	}

	physSkExp, err := charSheet.ability.GetExpReferenceOf(enum.Physicals)
	if err != nil {
		return charSheet, NewClassNotAppliedError("getting exp reference")
	}
	expTable := experience.NewExpTable(PHYSICAL_SKILLS_COEFF)
	newExp := experience.NewExperience(expTable)
	for name, exp := range charClass.ProficienciesExps {
		prof := proficiency.NewProficiency(name, *newExp, physSkExp)
		charSheet.AddCommonProficiency(name, prof)
		charSheet.IncreaseExpForProficiency(experience.NewUpgradeCascade(exp), name)
	}
	for name, exp := range charClass.JointProfExps {
		charSheet.IncreaseExpForJointProficiency(
			experience.NewUpgradeCascade(exp),
			name,
		)
	}
	return charSheet, nil
}

func (csf *CharacterSheetFactory) BuildHalfSheet(
	profile CharacterProfile,
	charClass *cc.CharacterClass,
	// categorySet *TalentByCategorySet,
) (*HalfSheet, error) {

	// TODO: like Build func above, move to client that calls this func
	// var talentLvl int
	// if categorySet == nil {
	// 	talentLvl = BASE_TALENT_LVL
	// } else {
	// 	talentLvl = categorySet.GetTalentLvl()
	// }
	characterExp := csf.BuildCharacterExp()
	abilities := csf.BuildPersonAbilitiesHalf(characterExp)

	physAbility, _ := abilities.Get(enum.Physicals)
	physAttrs := csf.BuildPhysAttrs(physAbility)

	mentalAbility, _ := abilities.Get(enum.Mentals)
	mentalAttrs := csf.BuildMentalAttrs(mentalAbility)

	characterAttrs := attribute.NewCharacterAttributes(physAttrs, mentalAttrs, nil)

	skills, _ := abilities.Get(enum.Skills)
	physSkills, err := csf.BuildPhysSkills(skills, physAttrs)
	if err != nil {
		return nil, err
	}
	mentalSkills := csf.BuildMentalSkills(skills, mentalAttrs)
	charSkills := skill.NewCharacterSkills(physSkills, mentalSkills, nil)

	proficiency := proficiency.NewManager()

	var className enum.CharacterClassName
	if charClass != nil {
		className = charClass.GetName()
	}
	// TODO: fix after add aura (MOP - spiritual status)
	status := csf.BuildStatusManager(abilities, characterAttrs, charSkills)

	sheet := NewHalfSheet(
		profile,
		*abilities,
		*characterAttrs,
		*charSkills,
		*proficiency,
		*status,
		&className,
	)
	if charClass != nil {
		return csf.WrapHalf(sheet, charClass)
	}
	return sheet, nil
}

func (csf *CharacterSheetFactory) BuildPersonAbilitiesHalf(
	characterExp *experience.CharacterExp,
) *ability.Manager {

	abilities := make(map[enum.AbilityName]ability.IAbility)

	talentExp := experience.NewExperience(experience.NewExpTable(TALENT_COEFF))
	talent := ability.NewTalent(*talentExp)

	physicalExp := experience.NewExperience(experience.NewExpTable(PHYSICAL_COEFF))
	abilities[enum.Physicals] = ability.NewAbility(enum.Physicals, *physicalExp, characterExp)

	mentalExp := experience.NewExperience(experience.NewExpTable(MENTAL_COEFF))
	abilities[enum.Mentals] = ability.NewAbility(enum.Mentals, *mentalExp, characterExp)

	skillsExp := experience.NewExperience(experience.NewExpTable(SKILLS_COEFF))
	abilities[enum.Skills] = ability.NewAbility(enum.Skills, *skillsExp, characterExp)

	return ability.NewAbilitiesManager(characterExp, abilities, *talent)
}

func (csf *CharacterSheetFactory) WrapHalf(
	charSheet *HalfSheet,
	charClass *cc.CharacterClass,
) (*HalfSheet, error) {
	for name, exp := range charClass.SkillsExps {
		charSheet.IncreaseExpForSkill(experience.NewUpgradeCascade(exp), name)
	}
	for _, skill := range charClass.JointSkills {
		charSheet.AddJointSkill(&skill)
	}
	for _, prof := range charClass.JointProficiencies {
		charSheet.AddJointProficiency(&prof)
	}
	for name, exp := range charClass.AttributesExps {
		charSheet.IncreaseExpForMentals(experience.NewUpgradeCascade(exp), name)
	}

	physSkExp, err := charSheet.ability.GetExpReferenceOf(enum.Physicals)
	if err != nil {
		return charSheet, NewClassNotAppliedError("getting exp reference")
	}
	expTable := experience.NewExpTable(PHYSICAL_SKILLS_COEFF)
	newExp := experience.NewExperience(expTable)
	for name, exp := range charClass.ProficienciesExps {
		prof := proficiency.NewProficiency(name, *newExp, physSkExp)
		charSheet.AddCommonProficiency(name, prof)
		charSheet.IncreaseExpForProficiency(experience.NewUpgradeCascade(exp), name)
	}
	for name, exp := range charClass.JointProfExps {
		charSheet.IncreaseExpForJointProficiency(
			experience.NewUpgradeCascade(exp),
			name,
		)
	}
	return charSheet, nil
}
