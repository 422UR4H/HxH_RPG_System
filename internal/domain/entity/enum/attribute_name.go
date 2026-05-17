package enum

import "fmt"

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
	Flame      AttributeName = "Flame"
	Conscience AttributeName = "Conscience"
)

func (an AttributeName) String() string {
	return string(an)
}

func AllAttributeNames() []AttributeName {
	return []AttributeName{
		Resistance, Strength, Agility, Celerity, Flexibility, Dexterity, Sense, Constitution,
		Resilience, Adaptability, Weighting, Creativity,
		Flame, Conscience,
	}
}

func AttributeNameFrom(s string) (AttributeName, error) {
	for _, name := range AllAttributeNames() {
		if s == name.String() {
			return name, nil
		}
	}
	return "", fmt.Errorf("%w%s: %s", ErrInvalidNameOf, "attribute", s)
}
