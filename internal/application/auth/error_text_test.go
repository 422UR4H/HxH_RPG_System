package auth_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
)

func TestSentinelErrors_ReadInPortuguese(t *testing.T) {
	if got, want := auth.ErrUnauthorized.Error(), "E-mail ou senha incorretos."; got != want {
		t.Errorf("ErrUnauthorized = %q, want %q", got, want)
	}
}
