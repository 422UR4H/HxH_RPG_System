package status

type HitPoints struct {
	Min     int
	Current int
	Max     int
}

func NewHitPoints() *HitPoints {
	points := 0
	return &HitPoints{
		Min:     points,
		Current: points,
		Max:     points,
	}
}

func (ap *HitPoints) StatusUpgrade(level int) {
	// TODO: Implement Min
	// Min = generateStatus.GetLvl();

	if ap.Current == ap.Max {
		ap.Current = level
	}
	// TODO: Implement else case (ex.: ap.Current == ap.Max - 1 -> threat % case)
	ap.Max = level
}
