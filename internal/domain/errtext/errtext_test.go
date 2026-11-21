package errtext_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
)

func TestWrap_ReadsAsSentenceAndKeepsSentinel(t *testing.T) {
	sentinel := errtext.Sentence("Ficha não encontrada.")
	err := errtext.Wrap(sentinel, "Essa ficha não é sua.")

	if err.Error() != "Essa ficha não é sua." {
		t.Errorf("Error() = %q, want only the sentence", err.Error())
	}
	if !errors.Is(err, sentinel) {
		t.Error("Wrap must keep the sentinel for errors.Is")
	}
}

func TestSentence_IsItsOwnSentinel(t *testing.T) {
	a := errtext.Sentence("Mesma frase.")
	b := errtext.Sentence("Mesma frase.")
	if errors.Is(a, b) {
		t.Error("two sentinels with the same text must stay distinct")
	}
}

func TestOf_FindsTheSentenceThroughWrappers(t *testing.T) {
	sentence := errtext.Sentence("Partida não encontrada.")
	wrapped := fmt.Errorf("load: %w", sentence)

	if got, ok := errtext.Of(wrapped); !ok || got != "Partida não encontrada." {
		t.Errorf("Of(wrapped) = %q, %v", got, ok)
	}
	if _, ok := errtext.Of(errors.New("pq: boom")); ok {
		t.Error("an English infrastructure error is not a sentence")
	}
}
