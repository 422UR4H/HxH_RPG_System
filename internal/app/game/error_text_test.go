package game

import (
	"errors"
	"fmt"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
)

func TestGameErrorText(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"sentence is shown as is", ErrNotMaster, "Só o mestre pode fazer isso."},
		{"sentence behind a wrapper", fmt.Errorf("room: %w", errtext.Sentence("A partida já terminou.")), "A partida já terminou."},
		{"infrastructure error becomes the generic sentence", errors.New("pq: connection refused"), "Não foi possível fazer isso agora."},
		{"english domain error is not a sentence", domain.NewDomainError(errors.New("failed to build hatsu")), "Não foi possível fazer isso agora."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gameErrorText(tt.err); got != tt.want {
				t.Errorf("gameErrorText = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStatusEffectErrorText_UnknownErrorUsesTheStatusSentence(t *testing.T) {
	got := statusEffectErrorText(errors.New("pq: boom"))
	if got != "Não foi possível fazer essa mudança de status." {
		t.Errorf("got %q", got)
	}
}
