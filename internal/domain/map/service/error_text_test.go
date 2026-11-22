package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/map/service"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := service.ErrNameTooShort.Error(), "O nome do mapa deve ter pelo menos 3 caracteres."; got != want {
		t.Errorf("ErrNameTooShort = %q, want %q", got, want)
	}
}
