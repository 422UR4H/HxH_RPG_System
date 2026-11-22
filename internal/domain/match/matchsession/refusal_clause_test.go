package matchsession

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
)

func TestRefusalClause(t *testing.T) {
	if got := refusalClause(errtext.Sentence("O personagem está atordoado.")); got != "O personagem está atordoado" {
		t.Errorf("sentence: got %q", got)
	}
	if got := refusalClause(errors.New("pq: boom")); got != "algo impediu a volta" {
		t.Errorf("english error: got %q", got)
	}
}
