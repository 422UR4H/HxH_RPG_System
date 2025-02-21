package experience

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type UpgradeCascade struct {
	expInserted  int
	CharacterExp ICharacterExp
	Skills       map[string]SkillCascade
	Proficiency  map[string]ProficiencyCascade
	Abilities    map[enum.AbilityName]AbilityCascade
	Attributes   map[enum.AttributeName]AttributeCascade
	Principles   map[enum.PrincipleName]PrincipleCascade
	Status       map[enum.StatusName]StatusCascade
}

func NewUpgradeCascade(
	expInserted int,
) *UpgradeCascade {
	return &UpgradeCascade{
		expInserted: expInserted,
		Skills:      make(map[string]SkillCascade),
		Proficiency: make(map[string]ProficiencyCascade),
		Abilities:   make(map[enum.AbilityName]AbilityCascade),
		Attributes:  make(map[enum.AttributeName]AttributeCascade),
		Principles:  make(map[enum.PrincipleName]PrincipleCascade),
		Status:      make(map[enum.StatusName]StatusCascade),
	}
}

func (uc *UpgradeCascade) GetExp() int {
	return uc.expInserted
}

func (uc *UpgradeCascade) SetExp(exp int) {
	uc.expInserted = exp
}

type AbilityCascade struct {
	Exp   int
	Lvl   int
	Bonus float64
}

type AttributeCascade struct {
	Exp   int
	Lvl   int
	Power int
}

type SkillCascade struct {
	Exp     int
	Lvl     int
	TestVal int
}

type PrincipleCascade struct {
	Exp     int
	Lvl     int
	TestVal int
}

type ProficiencyCascade struct {
	Exp int
	Lvl int
	// TestVal int
}

type StatusCascade struct {
	Min  int
	Curr int
	Max  int
}
