package clock

import "time"

// WireLayout é o formato do tempo da história no fio: relógio de parede
// ingênuo, sem Z e sem offset.
const WireLayout = "2006-01-02T15:04:05"

func FormatStory(t time.Time) string {
	return t.UTC().Format(WireLayout)
}

// ParseStory rejeita qualquer valor com offset ou Z, pois o layout exige
// que a string termine nos segundos.
func ParseStory(s string) (time.Time, error) {
	return time.ParseInLocation(WireLayout, s, time.UTC)
}
