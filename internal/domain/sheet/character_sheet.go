package charactersheet

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/status"
)

type CharacterSheet struct {
	profile   CharacterProfile
	ability   ability.Manager
	attribute attribute.CharacterAttributes
	skill     skill.CharacterSkills
	principle spiritual.Manager
	status    status.Manager
}

func NewCharacterSheet(
	profile CharacterProfile,
	abilities ability.Manager,
	attributes attribute.CharacterAttributes,
	principles spiritual.Manager,
	skills skill.CharacterSkills,
	status status.Manager,
) *CharacterSheet {
	return &CharacterSheet{
		profile:   profile,
		ability:   abilities,
		attribute: attributes,
		skill:     skills,
		principle: principles,
		status:    status,
	}
}

func (cs *CharacterSheet) GetValueForTestOfSkill(name enum.SkillName) (int, error) {
	return cs.skill.GetValueForTestOf(name)
}

// func (cs *CharacterSheet) GetValueForTestOfAttribute(name enum.AttributeName) (int, error) {
// 	return cs.attribute.GetPowerOf(name)
// }

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
