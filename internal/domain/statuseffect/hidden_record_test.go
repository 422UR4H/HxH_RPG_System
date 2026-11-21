package statuseffect_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// TestHiddenFromConceals: o registro do Escondido é uma lista de permissão — só quem o
// percebia no instante vê; todo outro leitor (um de fora da partida pública, um membro da
// campanha não inscrito, um inscrito depois) fica de fora até uma revelação cobrir o instante.
// A sentinela de linha ilegível esconde de todos e nenhuma revelação a cobre.
func TestHiddenFromConceals(t *testing.T) {
	char, eff := uuid.New(), uuid.New()
	perceiver, outsider := uuid.New(), uuid.New()
	at := time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC)
	revealedAt := func(when time.Time) statuseffect.RevealLookup {
		return func(id uuid.UUID) (time.Time, bool) { return when, id == eff || id == uuid.Nil }
	}
	rec := statuseffect.HiddenFrom{char: {Effect: eff, PerceivedBy: []uuid.UUID{perceiver}}}
	sentinel := statuseffect.HiddenFrom{char: {}}

	for _, c := range []struct {
		name    string
		m       statuseffect.HiddenFrom
		char    uuid.UUID
		reader  uuid.UUID
		reveals statuseffect.RevealLookup
		want    bool
	}{
		{"a reader who perceived it sees", rec, char, perceiver, nil, false},
		{"a reader never named is concealed", rec, char, outsider, nil, true},
		{"a reveal at or after the instant shows it", rec, char, outsider, revealedAt(at), false},
		{"a reveal before the instant does not", rec, char, outsider, revealedAt(at.Add(-time.Second)), true},
		{"a character without a record is not concealed", rec, uuid.New(), outsider, nil, false},
		{"nil map conceals nothing", nil, char, outsider, nil, false},
		{"the unreadable sentinel conceals from everyone", sentinel, char, perceiver, nil, true},
		{"no reveal covers the sentinel", sentinel, char, outsider, revealedAt(at.Add(time.Hour)), true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := c.m.Conceals(c.char, c.reader, at, c.reveals); got != c.want {
				t.Fatalf("Conceals = %v, want %v", got, c.want)
			}
		})
	}
}
