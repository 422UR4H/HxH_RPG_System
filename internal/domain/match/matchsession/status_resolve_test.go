package matchsession_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Os efeitos guardados na sessão chegam ao resolvedor, e um teste que eles tornam rolado
// ganha os dados pela mesma falta que um viés pede.
func TestResolveTurnReadsStatusEffects(t *testing.T) {
	t.Run("open guard default defense draws its set once", func(t *testing.T) {
		f := newDefaultFixture(t)
		// O acerto (9 + 9) passa a esquiva passiva de C (11); a defesa padrão é lida, só rolada.
		f.session.SetStatusEffects(map[uuid.UUID][]statuseffect.Effect{
			f.c: {{CharacterID: f.c, Kind: statuseffect.KindOpenGuard}},
		})
		rolls := f.source.rolls
		cr := f.resultOf(t, f.c)
		if got := f.source.rolls - rolls; got != 2 {
			t.Fatalf("the rolled-only defense drew %d dice, want 2 (the set)", got)
		}
		if cr.Defense.Passive || cr.Defense.Total != 18 {
			t.Fatalf("Defense = passive %v total %d, want rolled 9 + 9", cr.Defense.Passive, cr.Defense.Total)
		}
		after := f.source.rolls
		f.resultOf(t, f.c)
		if f.source.rolls != after {
			t.Fatalf("resolving again drew %d dice, want 0", f.source.rolls-after)
		}
	})

	t.Run("weakened attacker hit draws the extra die", func(t *testing.T) {
		f := newDefaultFixture(t)
		f.session.SetStatusEffects(map[uuid.UUID][]statuseffect.Effect{
			f.attacker: {{
				CharacterID: f.attacker, Kind: statuseffect.KindWeakened,
				Aptitude: statuseffect.AptitudePhysical, Params: statuseffect.Params{Disadvantage: true},
			}},
		})
		rolls := f.source.rolls
		cr := f.resultOf(t, f.b)
		if got := f.source.rolls - rolls; got != 1 {
			t.Fatalf("the weakened hit drew %d dice, want 1", got)
		}
		if cr.Hit.Bias != -1 || len(cr.Hit.Rolled) != 3 {
			t.Fatalf("Hit = bias %d rolled %v, want -1 and three dice", cr.Hit.Bias, cr.Hit.Rolled)
		}
	})
}
