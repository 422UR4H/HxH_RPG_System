package mapuc_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/map"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := mapuc.ErrMapNotFound.Error(), "Mapa não encontrado."; got != want {
		t.Errorf("ErrMapNotFound = %q, want %q", got, want)
	}
}
