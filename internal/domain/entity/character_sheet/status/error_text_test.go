package status_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := status.ErrStatusNotFound.Error(), "Barra de status não encontrada."; got != want {
		t.Errorf("ErrStatusNotFound = %q, want %q", got, want)
	}
}
