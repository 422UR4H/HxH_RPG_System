package user_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := user.ErrMissingNick.Error(), "Informe o nick (apelido)."; got != want {
		t.Errorf("ErrMissingNick = %q, want %q", got, want)
	}
}
