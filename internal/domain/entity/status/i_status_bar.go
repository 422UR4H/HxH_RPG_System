package status

type IStatusBar interface {
	IncreaseAt(value int) int
	DecreaseAt(value int) int
	Upgrade(level int)
	GetMin() int
	GetCurrent() int
	GetMax() int
}
