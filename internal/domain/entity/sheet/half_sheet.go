package sheet

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/status"
)

type HalfSheet struct {
	profile   CharacterProfile
	ability   ability.Manager
	attribute attribute.CharacterAttributes
	skill     skill.CharacterSkills
	status    status.Manager
	// equipedItems []Item
}

func NewHalfSheet(
	profile CharacterProfile,
	abilities ability.Manager,
	attributes attribute.CharacterAttributes,
	skills skill.CharacterSkills,
	status status.Manager,
) *HalfSheet {
	return &HalfSheet{
		profile:   profile,
		ability:   abilities,
		attribute: attributes,
		skill:     skills,
		status:    status,
	}
}

func (cs *HalfSheet) GetValueForTestOfSkill(name enum.SkillName) (int, error) {
	return cs.skill.GetValueForTestOf(name)
}

func (cs *HalfSheet) IncreaseExpForSkill(
	points int, name enum.SkillName,
) (int, error) {
	return cs.skill.IncreaseExp(points, name)
}

func (cs *HalfSheet) GetMaxOfStatus(name enum.StatusName) (int, error) {
	return cs.status.GetMaxOf(name)
}

func (cs *HalfSheet) GetMinOfStatus(name enum.StatusName) (int, error) {
	return cs.status.GetMinOf(name)
}

func (cs *HalfSheet) GetLevelOfAbility(name enum.AbilityName) (int, error) {
	return cs.ability.GetLevelOf(name)
}

func (cs *HalfSheet) GetLevelOfAttribute(name enum.AttributeName) (int, error) {
	return cs.attribute.GetLevelOf(name)
}

func (cs *HalfSheet) GetLevelOfSkill(name enum.SkillName) (int, error) {
	return cs.skill.GetLevelOf(name)
}

func (cs *HalfSheet) GetExpPointsOfAbility(name enum.AbilityName) (int, error) {
	return cs.ability.GetExpPointsOf(name)
}

func (cs *HalfSheet) GetExpPointsOfAttribute(name enum.AttributeName) (int, error) {
	return cs.attribute.GetExpPointsOf(name)
}

func (cs *HalfSheet) GetExpPointsOfSkill(name enum.SkillName) (int, error) {
	return cs.skill.GetExpPointsOf(name)
}

func (cs *HalfSheet) GetAggregateExpByLvlOfSkill(
	name enum.SkillName, lvl int,
) (int, error) {
	return cs.skill.GetAggregateExpByLvlOf(name, lvl)
}

func (cs *HalfSheet) GetExpPoints() int {
	return cs.ability.GetCharacterExp()
}
