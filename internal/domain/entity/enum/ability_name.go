package enum

type AbilityName string

const (
	Physicals  AbilityName = "Physicals"
	Mentals    AbilityName = "Mentals"
	Spirituals AbilityName = "Spirituals"
	Skills     AbilityName = "Skills"
)

func (an AbilityName) String() string {
	return string(an)
}
