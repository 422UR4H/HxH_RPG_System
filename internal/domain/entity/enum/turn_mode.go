package enum

type TurnMode string

const (
	Free TurnMode = "Free"
	Race TurnMode = "Race"
)

func (tm TurnMode) String() string {
	return string(tm)
}
