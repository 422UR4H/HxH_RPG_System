package enum

type CategoryName int

const (
	Reinforcement = iota
	Emission
	Transmutation
	Manipulation
	Materialization
	Specialization
)

func (cn CategoryName) String() string {
	switch cn {
	case Reinforcement:
		return "Reinforcement"
	case Emission:
		return "Emission"
	case Transmutation:
		return "Transmutation"
	case Manipulation:
		return "Manipulation"
	case Materialization:
		return "Materialization"
	case Specialization:
		return "Specialization"
	}
	return "Unknown"
}
