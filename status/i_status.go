package status

type IStatus interface {
	// TODO: verify if these methods are necessary
	IncreaseAt(value int) int
	DecreaseAt(value int) int
	Upgrade(level int)
	GetMin() int
	GetCurrent() int
	GetMax() int
}

type Status struct {
	min     int
	current int
	max     int
}

func (s *Status) IncreaseAt(value int) int {
	temp := s.current + value
	if temp > s.max {
		s.current = s.max
	} else {
		s.current = temp
	}
	return s.current
}

func (s *Status) DecreaseAt(value int) int {
	temp := s.current - value
	if temp < s.min {
		s.current = s.min
	} else {
		s.current = temp
	}
	return s.current
}

func (ap *Status) Upgrade(level int) {
	if ap.current == ap.max {
		ap.current = level
	}
	// TODO: Implement else case (ex.: ap.current == ap.max - 1 -> threat % case)
	ap.max = level
}

func (s *Status) GetMin() int {
	return s.min
}

func (s *Status) GetCurrent() int {
	return s.current
}

func (s *Status) GetMax() int {
	return s.max
}
