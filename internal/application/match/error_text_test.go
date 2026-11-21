package match_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/match"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := match.ErrMatchNotFound.Error(), "Partida não encontrada."; got != want {
		t.Errorf("ErrMatchNotFound = %q, want %q", got, want)
	}
}
