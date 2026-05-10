package enum

type RoundMode string

const (
	Free RoundMode = "Free"
	Race RoundMode = "Race"
)

func (tm RoundMode) String() string {
	return string(tm)
}
