package auth_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/application/auth"
	"github.com/422UR4H/HxH_RPG_System/internal/application/testutil"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/user"
)

// Só o nick conta caracteres; e-mail e senha continuam em bytes.
func TestRegister_NickCountsCharacters(t *testing.T) {
	run := func(nick string) error {
		in := validRegisterInput()
		in.Nick = nick
		return auth.NewRegisterUC(&testutil.MockAuthRepo{}).Register(context.Background(), in)
	}
	t.Run("nick at limit with accent passes", func(t *testing.T) {
		if err := run(strings.Repeat("ã", 20)); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
	t.Run("nick over limit fails", func(t *testing.T) {
		if err := run(strings.Repeat("ã", 21)); !errors.Is(err, user.ErrInvalidNickLength) {
			t.Fatalf("got %v", err)
		}
	})
	t.Run("nick with 3 accented characters passes the minimum", func(t *testing.T) {
		if err := run("ããã"); err != nil {
			t.Fatalf("unexpected: %v", err)
		}
	})
}
