package enum

type SkillName uint8

const (
	// PHYSICALS
	// Constitution
	Vitality SkillName = iota
	Resistance
	Breath
	Heal
	DefenseSkill

	// Strength
	Climb
	Push
	Grab
	CarryCapacity

	// Velocity
	Run
	Swim
	Jump

	// Agility
	Dodge
	Accelerate
	Brake

	// Action Speed
	AttackSpeed
	Feint

	// Flexibility
	Acrobatics
	Sneak

	// Dexterity?
	Reflex
	Accuracy
	Stealth
	SleightOfHand

	// Sense
	Vision
	Hearing
	Smell
	Tact
	Taste
	Balance

	// Instinct
	Intuition

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
	case Resistance:
		return "Resistance"
	case Breath:
		return "Breath"
	case Heal:
		return "Heal"
	case DefenseSkill:
		return "DefenseSkill"
	case Climb:
		return "Climb"
	case Push:
		return "Push"
	case Grab:
		return "Grab"
	case CarryCapacity:
		return "CarryCapacity"
	case Run:
		return "Run"
	case Swim:
		return "Swim"
	case Jump:
		return "Jump"
	case Dodge:
		return "Dodge"
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
	case SleightOfHand:
		return "SleightOfHand"
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
