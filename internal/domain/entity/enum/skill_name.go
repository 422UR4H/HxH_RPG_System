package enum

type SkillName uint8

const (
	// PHYSICALS
	// Resistance
	Vitality SkillName = iota
	Energy
	Defense

	// Strength
	Push
	Grab
	CarryCapacity

	// Agility
	Velocity
	Accelerate
	Brake

	// Action Speed => TODO: change to other name
	AttackSpeed // TODO: change to ActionSpeed
	Feint

	// Flexibility
	Acrobatics
	Sneak

	// Dexterity?
	Reflex
	Accuracy
	Stealth

	// Sense
	Vision
	Hearing
	Smell
	Tact
	Taste
	Balance

	// Constitution
	Heal
	Breath
	Tenacity

	// Instinct
	Intuition // ?

	// MENTALS
	// Resilience
	// Adaptability
	// Weighting
	// Creativity

	// SPIRITUALS
	// Spirit
	Nen
	Focus
	WillPower
)

func (sn SkillName) String() string {
	switch sn {
	case Vitality:
		return "Vitality"
	case Energy:
		return "Energy"
	case Defense:
		return "Defense"
	case Breath:
		return "Breath"
	case Heal:
		return "Heal"
	case Push:
		return "Push"
	case Grab:
		return "Grab"
	case CarryCapacity:
		return "CarryCapacity"
	// case Dodge:
	// 	return "Dodge"
	case Accelerate:
		return "Accelerate"
	case Brake:
		return "Brake"
	case AttackSpeed:
		return "AttackSpeed"
	case Feint:
		return "Feint"
	case Acrobatics:
		return "Acrobatics"
	case Sneak:
		return "Sneak"
	case Reflex:
		return "Reflex"
	case Accuracy:
		return "Accuracy"
	case Stealth:
		return "Stealth"
	case Vision:
		return "Vision"
	case Hearing:
		return "Hearing"
	case Smell:
		return "Smell"
	case Tact:
		return "Tact"
	case Taste:
		return "Taste"
	case Balance:
		return "Balance"
	case Intuition:
		return "Intuition"
	case Nen:
		return "Nen"
	case Focus:
		return "Focus"
	case WillPower:
		return "WillPower"
	}
	return "Unknown"
}
