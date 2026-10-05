package status

type IStatusBar interface {
	IStatusBarReader
	IncreaseAt(value int) int
	DecreaseAt(value int) int
	Upgrade()
	SetCurrent(value int) error
}

// IStatusBarReader is what a bar reads as: the three numbers a character_sheets row stores. The
// sheet gateway's UpdateStatusBars takes only this — writing a bar never needs to move it, and a
// detached copy (ReconstructBar) has no formula to Upgrade from.
type IStatusBarReader interface {
	GetMin() int
	GetCurrent() int
	GetMax() int
}
