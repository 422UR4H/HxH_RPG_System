package skill

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type CharacterSkills struct {
	physicals  *Manager
	mentals    *Manager
	spirituals *Manager
}

func NewCharacterSkills(
	physicals,
	mentals,
	spirituals *Manager) *CharacterSkills {

	return &CharacterSkills{
		physicals:  physicals,
		mentals:    mentals,
		spirituals: spirituals,
	}
}

func (cs *CharacterSkills) IncreaseExp(
	values *experience.UpgradeCascade,
	name enum.SkillName,
) error {
	skill, err := cs.Get(name)
	if err != nil {
		return err
	}
	skill.CascadeUpgradeTrigger(values)
	return nil
}

func (cs *CharacterSkills) AddPhysicalJoint(skill *JointSkill) error {
	return cs.physicals.AddJointSkill(skill)
}

func (cs *CharacterSkills) GetPhysicalsJoint() map[string]JointSkill {
	return cs.physicals.GetJointSkills()
}

func (cs *CharacterSkills) Get(name enum.SkillName) (ISkill, error) {
	if cs.spirituals != nil {
		if skill, _ := cs.spirituals.Get(name); skill != nil {
			return skill, nil
		}
	}
	if skill, _ := cs.physicals.Get(name); skill != nil {
		return skill, nil
	}
	if skill, _ := cs.mentals.Get(name); skill != nil {
		return skill, nil
	}
	return nil, ErrSkillNotFound
}

func (cs *CharacterSkills) GetValueForTestOf(name enum.SkillName) (int, error) {
	skill, err := cs.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetValueForTest(), nil
}

func (cs *CharacterSkills) GetNextLvlAggregateExpOf(name enum.SkillName) (int, error) {

	skill, err := cs.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetNextLvlAggregateExp(), nil
}

func (cs *CharacterSkills) GetNextLvlBaseExpOf(name enum.SkillName) (int, error) {
	skill, err := cs.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetNextLvlBaseExp(), nil
}

func (cs *CharacterSkills) GetCurrentExpOf(name enum.SkillName) (int, error) {
	skill, err := cs.Get(name)
	if err != nil {
		return 0, err
	}
	return skill.GetCurrentExp(), nil
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
	return skill.GetLevel(), nil
}

func (cs *CharacterSkills) GetPhysicalsNextLvlAggregateExp() map[enum.SkillName]int {
	return cs.physicals.GetSkillsNextLvlAggregateExp()
}

func (cs *CharacterSkills) GetMentalsNextLvlAggregateExp() map[enum.SkillName]int {
	return cs.mentals.GetSkillsNextLvlAggregateExp()
}

func (cs *CharacterSkills) GetSpiritualsNextLvlAggregateExp() map[enum.SkillName]int {
	return cs.spirituals.GetSkillsNextLvlAggregateExp()
}

func (cs *CharacterSkills) GetPhysicalsNextLvlBaseExp() map[enum.SkillName]int {
	return cs.physicals.GetSkillsNextLvlBaseExp()
}

func (cs *CharacterSkills) GetMentalsNextLvlBaseExp() map[enum.SkillName]int {
	return cs.mentals.GetSkillsNextLvlBaseExp()
}

func (cs *CharacterSkills) GetSpiritualsNextLvlBaseExp() map[enum.SkillName]int {
	return cs.spirituals.GetSkillsNextLvlBaseExp()
}

func (cs *CharacterSkills) GetPhysicalsCurrentExp() map[enum.SkillName]int {
	return cs.physicals.GetSkillsCurrentExp()
}

func (cs *CharacterSkills) GetMentalsCurrentExp() map[enum.SkillName]int {
	return cs.mentals.GetSkillsCurrentExp()
}

func (cs *CharacterSkills) GetSpiritualsCurrentExp() map[enum.SkillName]int {
	return cs.spirituals.GetSkillsCurrentExp()
}

func (cs *CharacterSkills) GetPhysicalsExpPoints() map[enum.SkillName]int {
	return cs.physicals.GetSkillsExpPoints()
}

func (cs *CharacterSkills) GetMentalsExpPoints() map[enum.SkillName]int {
	return cs.mentals.GetSkillsExpPoints()
}

func (cs *CharacterSkills) GetSpiritualsExpPoints() map[enum.SkillName]int {
	return cs.spirituals.GetSkillsExpPoints()
}

func (cs *CharacterSkills) GetPhysicalsLevel() map[enum.SkillName]int {
	return cs.physicals.GetSkillsLevel()
}

func (cs *CharacterSkills) GetMentalsLevel() map[enum.SkillName]int {
	return cs.mentals.GetSkillsLevel()
}

func (cs *CharacterSkills) GetSpiritualsLevel() map[enum.SkillName]int {
	return cs.spirituals.GetSkillsLevel()
}

func (cs *CharacterSkills) GetPhysicalSkills() map[enum.SkillName]ISkill {
	return cs.physicals.GetCommonSkills()
}

func (cs *CharacterSkills) GetMentalSkills() map[enum.SkillName]ISkill {
	return cs.mentals.GetCommonSkills()
}

func (cs *CharacterSkills) GetSpiritualSkills() map[enum.SkillName]ISkill {
	return cs.spirituals.GetCommonSkills()
}
