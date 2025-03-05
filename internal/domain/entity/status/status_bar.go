package status

type Bar struct {
	min  int
	curr int
	max  int
}

func NewStatusBar() *Bar {
	return &Bar{}
}

func (b *Bar) IncreaseAt(value int) int {
	temp := b.curr + value
	b.curr = min(temp, b.max)
	return b.curr
}

func (b *Bar) DecreaseAt(value int) int {
	temp := b.curr - value
	b.curr = max(temp, b.min)
	return b.curr
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
