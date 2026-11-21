package sheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := sheet.ErrClassNotApplied.Error(), "A classe ainda não foi aplicada à ficha."; got != want {
		t.Errorf("ErrClassNotApplied = %q, want %q", got, want)
	}
}
