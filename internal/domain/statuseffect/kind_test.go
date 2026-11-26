package statuseffect_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// O nome de cada tipo na mesa (o mesmo da tabela de match-status-effects.md §3): é o que as
// recusas mostram ao usuário.
func TestKindLabel(t *testing.T) {
	want := map[statuseffect.Kind]string{
		statuseffect.KindParalyzed: "Paralisado", statuseffect.KindStupefied: "Atônito",
		statuseffect.KindStunned: "Atordoado", statuseffect.KindUnconscious: "Inconsciente",
		statuseffect.KindRestrained: "Preso", statuseffect.KindInjured: "Lesionado",
		statuseffect.KindSlowed: "Lento", statuseffect.KindSense: "Sentido afetado",
		statuseffect.KindWeakened: "Debilitado", statuseffect.KindBleeding: "Sangrando",
		statuseffect.KindBurning: "Queimando", statuseffect.KindFreezing: "Congelando",
		statuseffect.KindPoisoned: "Envenenado", statuseffect.KindAsphyxiating: "Asfixiando",
		statuseffect.KindProne: "Caído", statuseffect.KindAirborne: "No ar",
		statuseffect.KindOpenGuard: "Guarda aberta", statuseffect.KindCompelled: "Compelido",
		statuseffect.KindControlled: "Controlado", statuseffect.KindHidden: "Escondido",
		statuseffect.KindDead: "Morto",
	}
	for k, label := range want {
		if _, err := statuseffect.ParseKind(string(k)); err != nil {
			t.Fatalf("%s is not a kind: %v", k, err)
		}
		if got := k.Label(); got != label {
			t.Errorf("%s.Label() = %q, want %q", k, got, label)
		}
	}
	if got := statuseffect.Kind("unknownKind").Label(); got != "unknownKind" {
		t.Errorf("an unknown kind's label = %q, want the raw value", got)
	}
}

func TestErrorsReadAsSingleSentences(t *testing.T) {
	two := 0
	cases := []struct {
		name string
		err  error
		is   error
		want string
	}{
		{"missing points", func() error {
			_, err := statuseffect.Normalize(statuseffect.KindParalyzed, "", "", statuseffect.ParamsInput{})
			return err
		}(), statuseffect.ErrInvalidParams, "Informe os pontos do efeito."},
		{"foreign field", func() error {
			_, err := statuseffect.Normalize(statuseffect.KindProne, "", "", statuseffect.ParamsInput{Points: &two})
			return err
		}(), statuseffect.ErrInvalidParams, "Esse efeito não tem pontos."},
		{"unknown kind", func() error { _, err := statuseffect.ParseKind("foo"); return err }(),
			statuseffect.ErrUnknownKind, "Esse efeito não existe."},
		{"invalid player", statuseffect.Audience{Players: []uuid.UUID{uuid.New(), uuid.Nil}}.Validate(),
			statuseffect.ErrInvalidAudience, "Um dos jogadores escolhidos não é válido."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(tc.err, tc.is) || tc.err.Error() != tc.want {
				t.Errorf("err = %q (is=%v), want %q", tc.err, errors.Is(tc.err, tc.is), tc.want)
			}
		})
	}
}

func TestStatePhrase(t *testing.T) {
	if got := statuseffect.KindOpenGuard.StatePhrase(); got != "com a guarda aberta" {
		t.Errorf("StatePhrase = %q", got)
	}
}

func TestIdentityStatePhrase(t *testing.T) {
	cases := []struct {
		id   statuseffect.Identity
		want string
	}{
		{statuseffect.Identity{Kind: statuseffect.KindParalyzed}, "paralisado"},
		{statuseffect.Identity{Kind: statuseffect.KindSense, Variant: "vision"}, "com a visão afetada"},
		{statuseffect.Identity{Kind: statuseffect.KindWeakened, Variant: "mental"}, "com a mente debilitada"},
	}
	for _, tc := range cases {
		if got := tc.id.StatePhrase(); got != tc.want {
			t.Errorf("StatePhrase(%+v) = %q, want %q", tc.id, got, tc.want)
		}
	}
}
