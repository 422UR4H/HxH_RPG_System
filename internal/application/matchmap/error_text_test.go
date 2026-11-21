package matchmapuc_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := matchmapuc.ErrMatchMapNotFound.Error(), "Esta partida não tem mapa."; got != want {
		t.Errorf("ErrMatchMapNotFound = %q, want %q", got, want)
	}
}
