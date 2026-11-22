package charactersheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/character_sheet"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := charactersheet.ErrCharacterSheetNotFound.Error(), "Ficha não encontrada."; got != want {
		t.Errorf("ErrCharacterSheetNotFound = %q, want %q", got, want)
	}
}
