package enum

type AbilityName uint8

const (
	Physicals AbilityName = iota
	Mentals
	Spirituals
	Skills
)

func (an AbilityName) String() string {
	switch an {
	case Physicals:
		return "Physicals"
	case Mentals:
		return "Mentals"
	case Spirituals:
		return "Spirituals"
	case Skills:
		return "Skills"

		// knowledge
		// talent
	}
	return "Unknown"
}
