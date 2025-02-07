package item

type IItem interface {
	GetDice() []int
	GetStaminaCost() int
	GetHeight() float64
	GetWeight() float64
	GetVolume() int
}
