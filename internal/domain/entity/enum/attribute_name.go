package enum

type AttributeName string

const (
	// Physicals
	Resistance   AttributeName = "Resistance"
	Strength     AttributeName = "Strength"
	Agility      AttributeName = "Agility"
	Celerity     AttributeName = "Celerity"
	Flexibility  AttributeName = "Flexibility"
	Dexterity    AttributeName = "Dexterity"
	Sense        AttributeName = "Sense"
	Constitution AttributeName = "Constitution"

	// Mentals
	Resilience   AttributeName = "Resilience"
	Adaptability AttributeName = "Adaptability"
	Weighting    AttributeName = "Weighting"
	Creativity   AttributeName = "Creativity"

	// Spirituals
	Spirit AttributeName = "Spirit"
)

func (an AttributeName) String() string {
	return string(an)
}
