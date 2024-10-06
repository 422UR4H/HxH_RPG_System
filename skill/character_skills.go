package skill

import (
	"errors"

	enum "github.com/422UR4H/HxH_RPG_Environment.Domain/enum"
)

type CharacterSkills struct {
	physicalSkills  SkillsManager
	mentalSkills    SkillsManager
	spiritualSkills SkillsManager
}

func NewCharacterSkills(
	physicalSkills,
	mentalSkills,
	spiritualSkills SkillsManager) *CharacterSkills {

	return &CharacterSkills{
		physicalSkills:  physicalSkills,
		mentalSkills:    mentalSkills,
		spiritualSkills: spiritualSkills,
	}
}

func (cs *CharacterSkills) Get(name enum.SkillName) (ISkill, error) {
	if skill, err := cs.spiritualSkills.Get(name); err == nil {
		return skill, nil
	}
	if skill, err := cs.physicalSkills.Get(name); err == nil {
		return skill, nil
	}
	if skill, err := cs.mentalSkills.Get(name); err == nil {
		return skill, nil
	}
	return nil, errors.New("skill not found")
}

func (cs *CharacterSkills) GetValueForTestOf(name enum.SkillName) (int, error) {
	skill, err := cs.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetValueForTest(), nil
}

func (cs *CharacterSkills) GetExpPointsOf(name enum.SkillName) (int, error) {
	skill, err := cs.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetExpPoints(), nil
}

func (cs *CharacterSkills) GetLevelOf(name enum.SkillName) (int, error) {
	skill, err := cs.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetLvl(), nil
}

func (cs *CharacterSkills) IncreaseExp(points int, name enum.SkillName) (int, error) {
	skill, err := cs.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.IncreaseExp(points), nil
}
