package match

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// DiceSet names the roll shape a match uses for every test: skill, hit, actionSpeed.
// 2D10 is the system default; D20 is an alternative match rule that flattens the
// distribution and makes luck weigh more than skill.
type DiceSet string

const (
	DiceSet2D10 DiceSet = "2d10"
	DiceSetD20  DiceSet = "d20"
)

// Dice returns the dice this set rolls, in order.
// Unknown values fall back to the system default rather than rolling nothing.
func (d DiceSet) Dice() []enum.DieSides {
	if d == DiceSetD20 {
		return []enum.DieSides{enum.D20}
	}
	return []enum.DieSides{enum.D10, enum.D10}
}

// PassiveValue is the average of the set. A passive test takes it instead of rolling,
// so rolling has exactly zero expected gain — the player only gambles when they need
// luck above the average. 11 for 2D10, 10 for D20.
func (d DiceSet) PassiveValue() int {
	if d == DiceSetD20 {
		return 10
	}
	return 11
}

// MaxFace is the top face of a single die in the set. Rolling it on every die of the
// set is a critical; rolling 1 on every die is a critical failure. The reading is on
// the individual dice, never on the sum.
func (d DiceSet) MaxFace() int {
	if d == DiceSetD20 {
		return 20
	}
	return 10
}

// MatchRules is the per-match rule configuration: the numbers a table is allowed to
// change. The shape of the result ladder — how many steps and what each one does —
// stays in code, because changing it changes the game.
//
// Phase 1 keeps this a pure value object with embedded defaults, passed by parameter
// to whoever needs it. It is not global, not read from anywhere, and not persisted.
// Persistence, the REST surface for the master to choose, and the fog_mode unblock in
// room.go are a separate slice — see the design spec §4.6.
type MatchRules struct {
	DiceSet          DiceSet
	LadderStep       int
	ReactionTimer    *time.Duration // nil = off
	DefaultReactions bool           // apply the default reaction when the target sends nothing
	FogMode          *fog.FogMode   // nil = inherit from the map
}

// NewDefaultMatchRules returns the MVP defaults.
func NewDefaultMatchRules() MatchRules {
	return MatchRules{
		DiceSet:          DiceSet2D10,
		LadderStep:       10,
		ReactionTimer:    nil,
		DefaultReactions: true,
		FogMode:          nil,
	}
}

// PassiveValue is derived from DiceSet and never stored. A stored copy would let a
// dice-set change decalibrate every passive test silently.
func (r MatchRules) PassiveValue() int { return r.DiceSet.PassiveValue() }

// ResolveFogMode implements: match rule ?? map ?? explored.
//
// A map is reusable across matches, so the fog style belongs to how *this table* wants
// to play — but the values already stored in maps.fog_mode stay meaningful as the
// default instead of being orphaned.
func (r MatchRules) ResolveFogMode(mapMode fog.FogMode) fog.FogMode {
	if r.FogMode != nil && r.FogMode.IsValid() {
		return *r.FogMode
	}
	if mapMode.IsValid() {
		return mapMode
	}
	return fog.FogModeExplored
}
