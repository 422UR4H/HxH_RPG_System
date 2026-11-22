package enum_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := enum.ErrInvalidNameOf.Error(), "Valor inválido."; got != want {
		t.Errorf("ErrInvalidNameOf = %q, want %q", got, want)
	}
}
