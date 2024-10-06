package status

type StaminaPoints struct {
	Status
}

// TODO: rerify if should I private status and its fields
func NewStaminaPoints() *StaminaPoints {
	points := 0
	return &StaminaPoints{
		Status: Status{
			Min:     points,
			Current: points,
			Max:     points,
		},
	}
}

func (ap *StaminaPoints) StatusUpgrade(level int) {
	ap.Status.StatusUpgrade(level)
}

func (ap *StaminaPoints) IncreaseAt(value int) int {
	return ap.Status.IncreaseAt(value)
}

func (ap *StaminaPoints) DecreaseAt(value int) int {
	return ap.Status.DecreaseAt(value)
}

func (ap *StaminaPoints) GetMin() int {
	return ap.Status.Min
}

func (ap *StaminaPoints) GetCurrent() int {
	return ap.Status.Current
}

func (ap *StaminaPoints) GetMax() int {
	return ap.Status.Max
}
