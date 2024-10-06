package status

type HitPoints struct {
	Status
}

func NewHitPoints() *HitPoints {
	points := 0
	return &HitPoints{
		Status: Status{
			Min:     points,
			Current: points,
			Max:     points,
		},
	}
}

func (ap *HitPoints) StatusUpgrade(level int) {
	// TODO: Implement Min
	// Min = generateStatus.GetLvl();
	ap.Status.StatusUpgrade(level)
}

func (ap *HitPoints) IncreaseAt(value int) int {
	return ap.Status.IncreaseAt(value)
}

func (ap *HitPoints) DecreaseAt(value int) int {
	return ap.Status.DecreaseAt(value)
}

func (ap *HitPoints) GetMin() int {
	return ap.Status.Min
}

func (ap *HitPoints) GetCurrent() int {
	return ap.Status.Current
}

func (ap *HitPoints) GetMax() int {
	return ap.Status.Max
}
