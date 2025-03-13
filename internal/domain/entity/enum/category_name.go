package enum

import "fmt"

type CategoryName int

const (
	Reinforcement = iota
	Transmutation
	Materialization
	Specialization
	Manipulation
	Emission
)

func (cn CategoryName) String() string {
	switch cn {
	case Reinforcement:
		return "Reinforcement"
	case Transmutation:
		return "Transmutation"
	case Materialization:
		return "Materialization"
	case Specialization:
		return "Specialization"
	case Manipulation:
		return "Manipulation"
	case Emission:
		return "Emission"
	}
	return "Unknown"
}

func AllNenCategoryNames() []CategoryName {
	return []CategoryName{
		Reinforcement,
		Emission,
		Transmutation,
		Manipulation,
		Materialization,
		Specialization,
	}
}

func CategoryNameFrom(s string) (CategoryName, error) {
	switch s {
	case "Reinforcement":
		return Reinforcement, nil
	case "Transmutation":
		return Transmutation, nil
	case "Materialization":
		return Materialization, nil
	case "Specialization":
		return Specialization, nil
	case "Manipulation":
		return Manipulation, nil
	case "Emission":
		return Emission, nil
	default:
		return 0, fmt.Errorf("invalid category name: %s", s)
	}
}
