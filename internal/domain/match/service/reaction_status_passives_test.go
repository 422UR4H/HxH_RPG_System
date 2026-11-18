package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

// Uma reação padrão com uma das passivas tirada por um status (decisão 9 do plano de efeitos
// de status). Ficha zerada: cada passiva vale 0 + 11 = 11.

func TestResolveReaction_DefaultWithoutReflexDodge(t *testing.T) {
	r := reactionWith(action.ReactDefault, nil, nil, nil)
	r.WithoutReflexDodge = true

	// Um golpe de 9 seria esquivado pelo reflexo (11); sem ele, vai à defesa (CD 9 − 10).
	out := service.ResolveReaction(reactionInput(t, action.ReactDefault, r, 9))
	if out.Avoided {
		t.Fatal("no reflex dodge, yet the blow was avoided")
	}
	if out.Dodge.Total != 0 {
		t.Fatalf("Dodge.Total = %d, want 0 — the reflex was not read", out.Dodge.Total)
	}
	if !out.Defended {
		t.Fatal("the default defense still stands behind it")
	}
}

func TestResolveReaction_DefaultWithoutDefaultDefense(t *testing.T) {
	r := reactionWith(action.ReactDefault, nil, nil, nil)
	r.WithoutDefaultDefense = true

	out := service.ResolveReaction(reactionInput(t, action.ReactDefault, r, 9))
	if !out.Avoided {
		t.Fatal("the reflex dodge (11) still stops a hit of 9")
	}
	// 15 passa do reflexo; a defesa (11 contra CD 5) ampararia, mas foi tirada.
	out = service.ResolveReaction(reactionInput(t, action.ReactDefault, r, 15))
	if out.Avoided || out.Defended {
		t.Fatalf("avoided %v defended %v, want neither — no defense to fall back on", out.Avoided, out.Defended)
	}
	if out.Defense.Total != 0 {
		t.Fatalf("Defense.Total = %d, want 0 — the defense was not read", out.Defense.Total)
	}
}

func TestResolveReaction_DodgeWithoutDefaultDefense(t *testing.T) {
	// Uma esquiva que falha (2 contra 30) cairia na defesa padrão; com ela tirada, não cai.
	r := reactionWith(action.ReactDodge, []int{1, 1}, nil, nil)
	r.WithoutDefaultDefense = true
	out := service.ResolveReaction(reactionInput(t, action.ReactDodge, r, 15))
	if out.Defended {
		t.Fatal("a dodge without the default defense fell back on it")
	}
}
