package enum

type AttributeName uint8

const (
	// Physicals
	Resistance = iota
	Strength
	Agility
	ActionSpeed
	Flexibility
	Dexterity
	Sense
	Constitution
	Instinct // ?

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
	case Resistance:
		return "Resistance"
	case Strength:
		return "Strength"
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
	case Constitution:
		return "Constitution"
	case Instinct:
		return "Instinct"
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
