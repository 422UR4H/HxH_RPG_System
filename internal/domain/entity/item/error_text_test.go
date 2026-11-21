package item_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/item"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := item.ErrWeaponNotFound.Error(), "Arma não encontrada."; got != want {
		t.Errorf("ErrWeaponNotFound = %q, want %q", got, want)
	}
}
