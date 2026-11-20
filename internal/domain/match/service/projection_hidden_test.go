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
