package status

type AuraPoints struct {
	Min     int
	Current int
	Max     int
}

func NewAuraPoints() *AuraPoints {
	points := 0
	return &AuraPoints{
		Min:     0,
		Current: points,
		Max:     points,
	}
}

func (ap *AuraPoints) StatusUpgrade(level int) {
	if ap.Current == ap.Max {
		ap.Current = level
	}
	// TODO: Implement else case (ex.: ap.Current == ap.Max - 1 -> threat % case)
	ap.Max = level
}
