package experience

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"

type UpgradeCascade struct {
	CharacterExp ICharacterExp
	Skills       map[string]SkillCascade
	Proficiency  map[string]ProficiencyCascade
	Abilities    map[enum.AbilityName]AbilityCascade
	Attributes   map[enum.AttributeName]AttributeCascade
	Principles   map[enum.PrincipleName]PrincipleCascade
	Status       map[enum.StatusName]StatusCascade
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
