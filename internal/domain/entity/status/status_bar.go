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

func (b *Bar) IncreaseAt(value int) int {
	temp := b.curr + value
	if temp > b.max {
		b.curr = b.max
	} else {
		b.curr = temp
	}
	return b.curr
}

func (b *Bar) DecreaseAt(value int) int {
	temp := b.curr - value
	if temp < b.min {
		b.curr = b.min
	} else {
		b.curr = temp
	}
	return b.curr
}

func (b *Bar) Upgrade(level int) {
	// TODO: Implement Min for hit_points
	// Min = generateStatus.GetLvl();
	if b.curr == b.max {
		b.curr = level
	}
	// TODO: Implement else case (ex.: b.current == b.max - 1 -> threat % case)
	b.max = level
}

func (b *Bar) GetMin() int {
	return b.min
}

func (b *Bar) GetCurrent() int {
	return b.curr
}

func (b *Bar) GetMax() int {
	return b.max
}
