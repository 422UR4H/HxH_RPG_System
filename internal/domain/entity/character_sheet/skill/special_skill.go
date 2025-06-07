package skill

import (
	attr "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type SpecialSkill struct {
	exp              experience.Exp
	name             string
	weapon           enum.WeaponName
	fixDamage        int
	fixHit           int
	volume           int
	attributes       map[enum.AttributeName]attr.IGameAttribute
	principles       map[enum.PrincipleName]spiritual.IPrinciple
	categories       map[enum.CategoryName]spiritual.ICategory
	restriction      Restriction
	Description      string
	BriefDescription string
}

func NewSpecialSkill(
	exp experience.Exp,
	name string,
	weapon enum.WeaponName,
	fixDamage int,
	fixHit int,
	volume int,
	attributes map[enum.AttributeName]attr.IGameAttribute,
	principles map[enum.PrincipleName]spiritual.IPrinciple,
	categories map[enum.CategoryName]spiritual.ICategory,
	restriction Restriction,
	description string,
	briefDescription string,
) *SpecialSkill {

	for principleName := range principles {
		if principleName == enum.Hatsu {
			delete(principles, principleName)
		}
	}

	return &SpecialSkill{
		exp:              exp,
		name:             name,
		weapon:           weapon,
		fixDamage:        fixDamage,
		fixHit:           fixHit,
		volume:           volume,
		attributes:       attributes,
		principles:       principles,
		categories:       categories,
		restriction:      restriction,
		Description:      description,
		BriefDescription: briefDescription,
	}
}

// TODO: decide how to train special skills
func (ss *SpecialSkill) CascadeUpgradeTrigger(exp int) int {
	diff := ss.exp.IncreasePoints(exp)
	// ss.attribute.CascadeUpgrade(exp)
	return diff
}

// TODO: decide how to calculate damage
func (ss *SpecialSkill) GetValueForTest() int {
	return ss.exp.GetLevel() // + ss.attribute.GetPower()
}

func (ss *SpecialSkill) GetExpPoints() int {
	return ss.exp.GetPoints()
}

func (ss *SpecialSkill) GetLevel() int {
	return ss.exp.GetLevel()
}

func (ss *SpecialSkill) GetAggregateExpByLvl(lvl int) int {
	return ss.exp.GetAggregateExpByLvl(lvl)
}

func (ss *SpecialSkill) GetName() string {
	return ss.name
}

func (ss *SpecialSkill) GetWeapon() enum.WeaponName {
	return ss.weapon
}

func (ss *SpecialSkill) GetFixDamage() int {
	return ss.fixDamage
}

func (ss *SpecialSkill) GetFixHit() int {
	return ss.fixHit
}

func (ss *SpecialSkill) GetVolume() int {
	return ss.volume
}

func (ss *SpecialSkill) GetAttributes() map[enum.AttributeName]attr.IGameAttribute {
	return ss.attributes
}

func (ss *SpecialSkill) GetPrinciples() map[enum.PrincipleName]spiritual.IPrinciple {
	return ss.principles
}

func (ss *SpecialSkill) GetCategories() map[enum.CategoryName]spiritual.ICategory {
	return ss.categories
}

func (ss *SpecialSkill) GetRestriction() Restriction {
	return ss.restriction
}
