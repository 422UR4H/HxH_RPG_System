package clock_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := clock.ErrNotSet.Error(), "Defina a hora da partida antes."; got != want {
		t.Errorf("ErrNotSet = %q, want %q", got, want)
	}
}
