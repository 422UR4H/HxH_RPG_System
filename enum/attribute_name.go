package enum

type AttributeName uint8

const (
	// Physicals
	Constitution = iota
	Defense
	Strength
	Velocity
	Agility
	ActionSpeed
	Flexibility
	Dexterity
	Sense
	Intuition

	// Mentals
	Resilience
	Adaptability
	Weighting
	Creativity

	// Spirituals
	Spirit
)

func (an AttributeName) String() string {
	switch an {
	case Constitution:
		return "Constitution"
	case Defense:
		return "Defense"
	case Strength:
		return "Strength"
	case Velocity:
		return "Velocity"
	case Agility:
		return "Agility"
	case ActionSpeed:
		return "ActionSpeed"
	case Flexibility:
		return "Flexibility"
	case Dexterity:
		return "Dexterity"
	case Sense:
		return "Sense"
	case Intuition:
		return "Intuition"
	case Resilience:
		return "Resilience"
	case Adaptability:
		return "Adaptability"
	case Weighting:
		return "Weighting"
	case Creativity:
		return "Creativity"
	case Spirit:
		return "Spirit"
	}
	return "Unknown"
}
