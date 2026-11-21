package matchsession_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := matchsession.ErrMatchEnded.Error(), "A partida já terminou."; got != want {
		t.Errorf("ErrMatchEnded = %q, want %q", got, want)
	}
}
