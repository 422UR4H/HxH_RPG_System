package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := spiritual.ErrNenHexAlreadyInitialized.Error(), "O hexágono de Nen já foi iniciado."; got != want {
		t.Errorf("ErrNenHexAlreadyInitialized = %q, want %q", got, want)
	}
}
