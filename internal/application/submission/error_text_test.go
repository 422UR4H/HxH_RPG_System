package submission_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/submission"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := submission.ErrCharacterAlreadySubmitted.Error(), "Essa ficha já foi enviada."; got != want {
		t.Errorf("ErrCharacterAlreadySubmitted = %q, want %q", got, want)
	}
}
