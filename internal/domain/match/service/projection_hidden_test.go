package service_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/battle"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

func TestViewerPerceives(t *testing.T) {
	c := uuid.New()
	cases := []struct {
		name string
		v    service.Viewer
		want bool
	}{
		{"master", service.Viewer{IsMaster: true, Hidden: map[uuid.UUID]bool{c: true}}, true},
		{"owner of a hidden character", service.Viewer{Owns: map[uuid.UUID]bool{c: true}, Hidden: map[uuid.UUID]bool{c: true}}, true},
		{"hidden", service.Viewer{Hidden: map[uuid.UUID]bool{c: true}}, false},
		{"not hidden", service.Viewer{Hidden: map[uuid.UUID]bool{uuid.New(): true}}, true},
		{"nil map", service.Viewer{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.Perceives(c); got != tc.want {
				t.Fatalf("Perceives = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestProjectResolutionCutsAHiddenTarget(t *testing.T) {
	actor, seen, hidden := uuid.New(), uuid.New(), uuid.New()
	res := &service.TurnResolution{
		IsSettled: true,
		CharacterResults: []service.CharacterResult{
			{TargetID: seen, ReactionKind: string(action.ReactDodge)},
			{TargetID: hidden, ReactionKind: string(action.ReactEscape), Escape: &service.EscapeResult{}},
		},
		Blows: []*battle.Blow{
			battle.NewBlow(actor, seen, action.Attack{}, nil, nil, nil),
			battle.NewBlow(actor, hidden, action.Attack{}, nil, nil, nil),
		},
	}

	t.Run("a reader who does not perceive the target loses its result and blows", func(t *testing.T) {
		got := service.ProjectResolution(res, service.Viewer{Hidden: map[uuid.UUID]bool{hidden: true}})
		if len(got.CharacterResults) != 1 || got.CharacterResults[0].TargetID != seen {
			t.Fatalf("CharacterResults = %+v, want only the seen target", got.CharacterResults)
		}
		if len(got.Blows) != 1 || got.Blows[0].GetTargetID() != seen {
			t.Fatalf("Blows = %d, want only the seen target's", len(got.Blows))
		}
		if len(res.CharacterResults) != 2 || len(res.Blows) != 2 {
			t.Fatal("the projection mutated the master's copy")
		}
	})
	t.Run("the target's owner keeps it", func(t *testing.T) {
		v := service.Viewer{Owns: map[uuid.UUID]bool{hidden: true}, Hidden: map[uuid.UUID]bool{hidden: true}}
		got := service.ProjectResolution(res, v)
		if len(got.CharacterResults) != 2 || len(got.Blows) != 2 {
			t.Fatalf("results/blows = %d/%d, want 2/2", len(got.CharacterResults), len(got.Blows))
		}
	})
	t.Run("the master keeps it", func(t *testing.T) {
		got := service.ProjectResolution(res, service.Viewer{IsMaster: true, Hidden: map[uuid.UUID]bool{hidden: true}})
		if len(got.CharacterResults) != 2 || len(got.Blows) != 2 {
			t.Fatalf("results/blows = %d/%d, want 2/2", len(got.CharacterResults), len(got.Blows))
		}
	})
}

func TestProjectActionCutsAHiddenTarget(t *testing.T) {
	actor, seen, hidden := uuid.New(), uuid.New(), uuid.New()
	a := action.Action{TargetID: []uuid.UUID{seen, hidden}}
	_ = actor

	t.Run("a reader who does not perceive the target loses its id", func(t *testing.T) {
		got := service.ProjectAction(a, service.Viewer{Hidden: map[uuid.UUID]bool{hidden: true}}, true)
		if len(got.TargetID) != 1 || got.TargetID[0] != seen {
			t.Fatalf("TargetID = %v, want [%s]", got.TargetID, seen)
		}
		if len(a.TargetID) != 2 {
			t.Fatal("the projection mutated the live action")
		}
	})
	t.Run("the actor's owner who does not perceive the target loses it too", func(t *testing.T) {
		v := service.Viewer{Owns: map[uuid.UUID]bool{a.GetActorID(): true}, Hidden: map[uuid.UUID]bool{hidden: true}}
		got := service.ProjectAction(a, v, true)
		if len(got.TargetID) != 1 {
			t.Fatalf("TargetID = %v, want one", got.TargetID)
		}
	})
	t.Run("the target's owner and the master keep it", func(t *testing.T) {
		for _, v := range []service.Viewer{
			{Owns: map[uuid.UUID]bool{hidden: true}, Hidden: map[uuid.UUID]bool{hidden: true}},
			{IsMaster: true, Hidden: map[uuid.UUID]bool{hidden: true}},
		} {
			if got := service.ProjectAction(a, v, true); len(got.TargetID) != 2 {
				t.Fatalf("TargetID = %v, want both", got.TargetID)
			}
		}
	})
}

// Alvo Escondido atingido (D2): quem não o percebe não sabe nada dele, mas fica sabendo QUANTOS
// golpes acertaram alguém que não vê — só o número, para a mesa ler "desconhecido N".
func TestProjectResolution_HiddenTargetHit_CountsAnonymously(t *testing.T) {
	actor, seen, h1, h2 := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	res := &service.TurnResolution{
		IsSettled: true,
		CharacterResults: []service.CharacterResult{
			{TargetID: seen, ReactionKind: string(action.ReactDodge)},
			{TargetID: h1, RawDamage: 9, EffectiveDamage: 7},
			{TargetID: h2, RawDamage: 5, EffectiveDamage: 5},
		},
		Blows: []*battle.Blow{
			battle.NewBlow(actor, seen, action.Attack{}, nil, nil, nil),
			battle.NewBlow(actor, h1, action.Attack{}, nil, nil, nil),
			battle.NewBlow(actor, h2, action.Attack{}, nil, nil, nil),
		},
	}
	got := service.ProjectResolution(res, service.Viewer{Hidden: map[uuid.UUID]bool{h1: true, h2: true}})
	if got.HiddenTargetsHit != 2 {
		t.Fatalf("HiddenTargetsHit = %d, want 2", got.HiddenTargetsHit)
	}
	if len(got.CharacterResults) != 1 || got.CharacterResults[0].TargetID != seen {
		t.Fatalf("CharacterResults = %+v, want only the seen target", got.CharacterResults)
	}
	if len(got.Blows) != 1 || got.Blows[0].GetTargetID() != seen {
		t.Fatalf("Blows = %d, want only the seen target's", len(got.Blows))
	}
	if res.HiddenTargetsHit != 0 {
		t.Fatal("the projection mutated the source resolution")
	}
}

func TestProjectResolution_HiddenTargetAvoided_LeavesNoTrace(t *testing.T) {
	hidden := uuid.New()
	res := &service.TurnResolution{CharacterResults: []service.CharacterResult{{TargetID: hidden, Avoided: true}}}
	got := service.ProjectResolution(res, service.Viewer{Hidden: map[uuid.UUID]bool{hidden: true}})
	if got.HiddenTargetsHit != 0 || len(got.CharacterResults) != 0 {
		t.Fatalf("hit/results = %d/%d, want 0/0", got.HiddenTargetsHit, len(got.CharacterResults))
	}
}

func TestProjectResolution_HiddenTargetAttackStopped_LeavesNoTrace(t *testing.T) {
	hidden := uuid.New()
	res := &service.TurnResolution{CharacterResults: []service.CharacterResult{{TargetID: hidden, AttackStopped: true}}}
	got := service.ProjectResolution(res, service.Viewer{Hidden: map[uuid.UUID]bool{hidden: true}})
	if got.HiddenTargetsHit != 0 || len(got.CharacterResults) != 0 {
		t.Fatalf("hit/results = %d/%d, want 0/0", got.HiddenTargetsHit, len(got.CharacterResults))
	}
}

func TestProjectResolution_MasterGetsNoCount(t *testing.T) {
	hidden := uuid.New()
	res := &service.TurnResolution{CharacterResults: []service.CharacterResult{{TargetID: hidden}}}
	got := service.ProjectResolution(res, service.Viewer{IsMaster: true, Hidden: map[uuid.UUID]bool{hidden: true}})
	if got.HiddenTargetsHit != 0 || len(got.CharacterResults) != 1 {
		t.Fatalf("hit/results = %d/%d, want 0/1", got.HiddenTargetsHit, len(got.CharacterResults))
	}
}

func TestProjectResolution_OwnerOfHiddenTargetSeesOwnResult(t *testing.T) {
	hidden := uuid.New()
	res := &service.TurnResolution{CharacterResults: []service.CharacterResult{{TargetID: hidden}}}
	v := service.Viewer{Owns: map[uuid.UUID]bool{hidden: true}, Hidden: map[uuid.UUID]bool{hidden: true}}
	got := service.ProjectResolution(res, v)
	if got.HiddenTargetsHit != 0 || len(got.CharacterResults) != 1 {
		t.Fatalf("hit/results = %d/%d, want 0/1", got.HiddenTargetsHit, len(got.CharacterResults))
	}
}

// A margem do ataque (total - margem = a esquiva do alvo) só existe com um alvo e entregaria a
// esquiva do Escondido, mesmo a de quem esquivou: sai quando algum alvo é cortado.
func TestProjectResolution_HiddenTargetCut_DropsTheActionMargin(t *testing.T) {
	seen, hidden := uuid.New(), uuid.New()
	margin := 5
	mk := func(cr ...service.CharacterResult) *service.TurnResolution {
		return &service.TurnResolution{
			ActionResult:     service.RollResult{Total: 18, Margin: &margin},
			CharacterResults: cr,
		}
	}
	hiddenV := service.Viewer{Hidden: map[uuid.UUID]bool{hidden: true}}

	t.Run("a single hidden target that dodged leaves no margin", func(t *testing.T) {
		res := mk(service.CharacterResult{TargetID: hidden, Avoided: true})
		got := service.ProjectResolution(res, hiddenV)
		if got.ActionResult.Margin != nil || got.HiddenTargetsHit != 0 {
			t.Fatalf("margin/hit = %v/%d, want nil/0", got.ActionResult.Margin, got.HiddenTargetsHit)
		}
		if got.ActionResult.Total != 18 {
			t.Fatalf("total = %d, want it untouched", got.ActionResult.Total)
		}
		if res.ActionResult.Margin == nil || *res.ActionResult.Margin != 5 {
			t.Fatal("the projection mutated the source margin")
		}
	})
	t.Run("a single hidden target that was stopped or hit leaves no margin either", func(t *testing.T) {
		for _, cr := range []service.CharacterResult{
			{TargetID: hidden, AttackStopped: true}, {TargetID: hidden},
		} {
			if got := service.ProjectResolution(mk(cr), hiddenV); got.ActionResult.Margin != nil {
				t.Fatalf("margin = %v, want nil", *got.ActionResult.Margin)
			}
		}
	})
	t.Run("the master, the owner and an unrelated single visible target keep it", func(t *testing.T) {
		for name, v := range map[string]service.Viewer{
			"master": {IsMaster: true, Hidden: map[uuid.UUID]bool{hidden: true}},
			"owner":  {Owns: map[uuid.UUID]bool{hidden: true}, Hidden: map[uuid.UUID]bool{hidden: true}},
		} {
			got := service.ProjectResolution(mk(service.CharacterResult{TargetID: hidden}), v)
			if got.ActionResult.Margin == nil || *got.ActionResult.Margin != 5 {
				t.Fatalf("%s: margin = %v, want 5", name, got.ActionResult.Margin)
			}
		}
		got := service.ProjectResolution(mk(service.CharacterResult{TargetID: seen}), hiddenV)
		if got.ActionResult.Margin == nil || *got.ActionResult.Margin != 5 {
			t.Fatalf("visible single target: margin = %v, want 5", got.ActionResult.Margin)
		}
	})
}

func TestProjectResolution_MixedVisibleAndHiddenTargets(t *testing.T) {
	actor, seen, hit, dodged := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	margin := 3
	res := &service.TurnResolution{
		IsSettled:    true,
		ActionResult: service.RollResult{Total: 15, Margin: &margin},
		CharacterResults: []service.CharacterResult{
			{TargetID: seen, ReactionKind: string(action.ReactDodge), EffectiveDamage: 4},
			{TargetID: hit, EffectiveDamage: 9},
			{TargetID: dodged, Avoided: true},
		},
		Blows: []*battle.Blow{
			battle.NewBlow(actor, seen, action.Attack{}, nil, nil, nil),
			battle.NewBlow(actor, hit, action.Attack{}, nil, nil, nil),
			battle.NewBlow(actor, dodged, action.Attack{}, nil, nil, nil),
		},
	}
	got := service.ProjectResolution(res, service.Viewer{Hidden: map[uuid.UUID]bool{hit: true, dodged: true}})
	if got.HiddenTargetsHit != 1 {
		t.Fatalf("HiddenTargetsHit = %d, want exactly 1", got.HiddenTargetsHit)
	}
	if len(got.CharacterResults) != 1 || got.CharacterResults[0].TargetID != seen || got.CharacterResults[0].EffectiveDamage != 4 {
		t.Fatalf("CharacterResults = %+v, want only the visible result", got.CharacterResults)
	}
	if len(got.Blows) != 1 || got.Blows[0].GetTargetID() != seen {
		t.Fatalf("Blows = %d, want only the visible target's", len(got.Blows))
	}
	if got.ActionResult.Margin != nil {
		t.Fatalf("margin = %d, want none", *got.ActionResult.Margin)
	}
	if len(res.CharacterResults) != 3 || len(res.Blows) != 3 || res.HiddenTargetsHit != 0 ||
		res.ActionResult.Margin == nil || *res.ActionResult.Margin != 3 {
		t.Fatal("the projection mutated the source resolution")
	}
}
