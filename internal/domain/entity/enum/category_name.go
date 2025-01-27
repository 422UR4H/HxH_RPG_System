package enum

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
