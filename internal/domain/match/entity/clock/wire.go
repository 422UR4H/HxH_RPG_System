package clock

import (
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
	"time"
)

// WireLayout é o formato do tempo da história no fio: relógio de parede
// ingênuo, sem Z e sem offset.
const WireLayout = "2006-01-02T15:04:05"

func FormatStory(t time.Time) string {
	return t.UTC().Format(WireLayout)
}

// ParseStory rejeita qualquer valor com offset ou Z, pois o layout exige
// que a string termine nos segundos. O ano vai de 1 a 9999 (o layout já
// exige 4 dígitos; o ano 0 é recusado aqui).
func ParseStory(s string) (time.Time, error) {
	t, err := time.ParseInLocation(WireLayout, s, time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	// o parser aceita fração de segundo após os segundos; o fio não
	if t.Format(WireLayout) != s {
		return time.Time{}, errtext.Sentence("A hora da história deve ter segundos inteiros.")
	}
	if t.Year() < 1 {
		return time.Time{}, errtext.Sentence("O ano da hora da história deve ficar entre 1 e 9999.")
	}
	return t, nil
}
