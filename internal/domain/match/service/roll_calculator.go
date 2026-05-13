package service

import (
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

// RollCalculator is a stateless domain service that computes the final result
// of a dice roll: dice values + character skill value + modifiers from the sheet.
// Full formula implementation is deferred pending game rule specification.
type RollCalculator struct{}

// Calculate returns the final numeric result of the given RollCheck for the
// character with the provided sheet. sheet may be nil during development.
func (rc RollCalculator) Calculate(check action.RollCheck, sheet *csSheet.CharacterSheet) int {
	// TODO: implement — roll dice in check.Context.Dice, apply check.SkillValue,
	// apply condition modifiers (check.Context.Condition), add sheet bonuses.
	return 0
}
