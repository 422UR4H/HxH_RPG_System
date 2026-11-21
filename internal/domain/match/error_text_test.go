package match_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := match.ErrMatchIsNil.Error(), "Partida não encontrada."; got != want {
		t.Errorf("ErrMatchIsNil = %q, want %q", got, want)
	}
}
