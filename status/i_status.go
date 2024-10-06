package status

type IStatus interface {
	// TODO: verify if these methods are necessary
	// GetMin() int
	// GetCurrent() int
	// GetMax() int
	IncreaseAt(value int) int
	DecreaseAt(value int) int
	StatusUpgrade(level int)
	GetMin() int
	GetCurrent() int
	GetMax() int
}

type Status struct {
	Min     int
	Current int
	Max     int
}

func (s *Status) IncreaseAt(value int) int {
	temp := s.Current + value
	if temp > s.Max {
		s.Current = s.Max
	} else {
		s.Current = temp
	}
	return s.Current
}

func (s *Status) DecreaseAt(value int) int {
	temp := s.Current - value
	if temp < s.Min {
		s.Current = s.Min
	} else {
		s.Current = temp
	}
	return s.Current
}

func (ap *Status) StatusUpgrade(level int) {
	if ap.Current == ap.Max {
		ap.Current = level
	}
	// TODO: Implement else case (ex.: ap.Current == ap.Max - 1 -> threat % case)
	ap.Max = level
}
