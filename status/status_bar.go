package status

type Bar struct {
	min  int
	curr int
	max  int
}

func NewStatusBar() *Bar {
	points := 0
	return &Bar{
		min:  points,
		curr: points,
		max:  points,
	}
}

func (sb *Bar) IncreaseAt(value int) int {
	temp := sb.curr + value
	if temp > sb.max {
		sb.curr = sb.max
	} else {
		sb.curr = temp
	}
	return sb.curr
}

func (sb *Bar) DecreaseAt(value int) int {
	temp := sb.curr - value
	if temp < sb.min {
		sb.curr = sb.min
	} else {
		sb.curr = temp
	}
	return sb.curr
}

func (sb *Bar) Upgrade(level int) {
	// TODO: Implement Min for hit_points
	// Min = generateStatus.GetLvl();
	if sb.curr == sb.max {
		sb.curr = level
	}
	// TODO: Implement else case (ex.: sb.current == sb.max - 1 -> threat % case)
	sb.max = level
}

func (sb *Bar) GetMin() int {
	return sb.min
}

func (sb *Bar) GetCurrent() int {
	return sb.curr
}

func (sb *Bar) GetMax() int {
	return sb.max
}
