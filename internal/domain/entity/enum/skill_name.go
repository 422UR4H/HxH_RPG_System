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
