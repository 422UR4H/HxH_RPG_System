package status

type Bar struct {
	min  int
	curr int
	max  int
}

func NewStatusBar() *Bar {
	return &Bar{}
}

// ReconstructBar rebuilds a bar from the three numbers it reads as. A turn's close copies the
// damaged sheets' bars with it under the room's lock, so the transaction that writes them after
// the unlock reads a value of its own, not the live bar the next message may move.
func ReconstructBar(min, curr, max int) *Bar {
	return &Bar{min: min, curr: curr, max: max}
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

func (b *Bar) SetCurrent(value int) error {
	if value < b.min || value > b.max {
		return ErrInvalidValue
	}
	b.curr = value
	return nil
}
