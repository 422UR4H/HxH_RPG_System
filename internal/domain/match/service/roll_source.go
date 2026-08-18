package service

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/die"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

// RollSource is where a die face comes from.
//
// It exists so a test can be reproducible. Derive is already deterministic — it receives
// dice that already fell — but everything upstream of it went through crypto/rand with no
// way in. The phases after this one have done-criteria stated as exact numbers (the round
// economy, the multi-target chain), and an economy test that depends on luck is not a test.
//
// Production passes nil or DiceRoller{}; tests pass a scripted source.
type RollSource interface {
	RollDie(sides enum.DieSides) int
}

// DiceRoller is the production source: crypto/rand, with the math/rand fallback that
// die.Die already implements.
type DiceRoller struct{}

func (DiceRoller) RollDie(sides enum.DieSides) int { return die.NewDie(sides).Roll() }

// sourceOrDefault keeps every call site free of nil checks. A nil source means production.
func sourceOrDefault(src RollSource) RollSource {
	if src == nil {
		return DiceRoller{}
	}
	return src
}
