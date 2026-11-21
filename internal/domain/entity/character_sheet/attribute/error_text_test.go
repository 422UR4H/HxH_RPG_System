package attribute_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := attribute.ErrPrimaryAttributeNotFound.Error(), "Atributo primário não encontrado."; got != want {
		t.Errorf("ErrPrimaryAttributeNotFound = %q, want %q", got, want)
	}
}
