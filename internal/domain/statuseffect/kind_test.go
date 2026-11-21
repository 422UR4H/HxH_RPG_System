package statuseffect_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
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
