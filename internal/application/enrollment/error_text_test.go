package enrollment_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/enrollment"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := enrollment.ErrCharacterNotInCampaign.Error(), "Esse personagem não pertence à campanha da partida."; got != want {
		t.Errorf("ErrCharacterNotInCampaign = %q, want %q", got, want)
	}
}
