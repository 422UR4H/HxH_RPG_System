package proficiency

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/experience"
)

// TODO: check if proficiency really has no attribute
// TODO: upgrade adding strike (GetStrike) and hit
type JointProficiency struct {
	exp  experience.Exp
	name string
	buff int
	// attr          attr.IGameAttribute
	weapons       []enum.WeaponName
	physSkillsExp experience.ICascadeUpgrade
}

func NewJointProficiency(
	exp experience.Exp,
	name string,
	// attr attr.IGameAttribute,
	weapons []enum.WeaponName,
) *JointProficiency {
	return &JointProficiency{
		exp:  exp,
		name: name,
		// attr:          attr,
		weapons: weapons,
	}
}

func (jp *JointProficiency) Init(physSkillsExp experience.ICascadeUpgrade) error {
	if jp.physSkillsExp != nil {
		return ErrProficiencyAlreadyInitialized
	}
	if physSkillsExp == nil {
		return ErrPhysSkillsCannotBeNil
	}
	jp.physSkillsExp = physSkillsExp
	return nil
}

func (jp *JointProficiency) CascadeUpgradeTrigger(values *experience.UpgradeCascade) {
	jp.exp.IncreasePoints(values.GetExp())
	// jp.attr.CascadeUpgrade(exp)
	// TODO: if uncomment, fix BuildWeaponsMaster
	jp.physSkillsExp.CascadeUpgrade(values) //* len(jp.weapons))

	values.Proficiency[jp.name] = experience.ProficiencyCascade{
		Lvl: jp.GetLevel(),
		Exp: jp.GetCurrentExp(),
	}
}

func (jp *JointProficiency) ContainsWeapon(name enum.WeaponName) bool {
	for _, weapon := range jp.weapons {
		if weapon == name {
			return true
		}
	}
	return false
}

func (jp *JointProficiency) AddWeapon(name enum.WeaponName) {
	jp.weapons = append(jp.weapons, name)
}

func (jp *JointProficiency) GetWeapons() []enum.WeaponName {
	return jp.weapons
}

func (jp *JointProficiency) SetBuff(name enum.WeaponName, value int) int { //, int) {
	jp.buff = value
	// testVal := m.GetValueForTestOf(name)
	return jp.GetLevel() + jp.buff //, testVal
}

func (jp *JointProficiency) DeleteBuff(name enum.WeaponName) {
	jp.buff = 0
}

func (jp *JointProficiency) GetBuff() int {
	return jp.buff
}

// TODO: validate this
func (jp *JointProficiency) GetValueForTest() int {
	return jp.exp.GetLevel() //+ jp.attr.GetPower()
}

func (jp *JointProficiency) GetNextLvlAggregateExp() int {
	return jp.exp.GetNextLvlAggregateExp()
}

func (jp *JointProficiency) GetNextLvlBaseExp() int {
	return jp.exp.GetNextLvlBaseExp()
}

func (jp *JointProficiency) GetCurrentExp() int {
	return jp.exp.GetCurrentExp()
}

func (jp *JointProficiency) GetExpPoints() int {
	return jp.exp.GetPoints()
}

func (jp *JointProficiency) GetLevel() int {
	return jp.exp.GetLevel()
}

func (jp *JointProficiency) GetName() string {
	return jp.name
}
