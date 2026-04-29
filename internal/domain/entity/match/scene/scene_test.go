package scene_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/scene"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/turn"
)

func TestNewScene(t *testing.T) {
	s := scene.NewScene(enum.Roleplay, "A dark forest")

	if s.GetCategory() != enum.Roleplay {
		t.Errorf("got category %v, want %v", s.GetCategory(), enum.Roleplay)
	}
	if s.BriefInitialDescription != "A dark forest" {
		t.Errorf("got BriefInitialDescription %q, want %q", s.BriefInitialDescription, "A dark forest")
	}
	if s.GetCreatedAt().IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if s.GetFinishedAt() != nil {
		t.Error("expected FinishedAt to be nil for new scene")
	}
	if len(s.GetTurns()) != 0 {
		t.Error("expected no turns for new scene")
	}
}

func TestNewScene_BattleCategory(t *testing.T) {
	s := scene.NewScene(enum.Battle, "Arena")

	if s.GetCategory() != enum.Battle {
		t.Errorf("got category %v, want %v", s.GetCategory(), enum.Battle)
	}
}

func TestScene_AddTurn(t *testing.T) {
	s := scene.NewScene(enum.Roleplay, "Town square")
	tn := &turn.Turn{}

	if err := s.AddTurn(tn); err != nil {
		t.Fatalf("AddTurn() error = %v", err)
	}
	if len(s.GetTurns()) != 1 {
		t.Errorf("got %d turns, want 1", len(s.GetTurns()))
	}
}

func TestScene_AddTurn_MultipleTurns(t *testing.T) {
	s := scene.NewScene(enum.Battle, "Colosseum")

	for i := 0; i < 3; i++ {
		if err := s.AddTurn(&turn.Turn{}); err != nil {
			t.Fatalf("AddTurn() iteration %d error = %v", i, err)
		}
	}
	if len(s.GetTurns()) != 3 {
		t.Errorf("got %d turns, want 3", len(s.GetTurns()))
	}
}

func TestScene_AddTurn_AfterFinished(t *testing.T) {
	s := scene.NewScene(enum.Roleplay, "Cave")
	_ = s.FinishScene("Escaped the cave")

	err := s.AddTurn(&turn.Turn{})
	if err == nil {
		t.Error("expected error when adding turn to finished scene")
	}
}

func TestScene_FinishScene(t *testing.T) {
	s := scene.NewScene(enum.Roleplay, "Inn")

	err := s.FinishScene("Left the inn")
	if err != nil {
		t.Fatalf("FinishScene() error = %v", err)
	}
	if s.BriefFinalDescription == nil || *s.BriefFinalDescription != "Left the inn" {
		t.Errorf("got BriefFinalDescription %v, want %q", s.BriefFinalDescription, "Left the inn")
	}
	if s.GetFinishedAt() == nil {
		t.Error("expected FinishedAt to be set after finishing")
	}
}

func TestScene_FinishScene_AlreadyFinished(t *testing.T) {
	s := scene.NewScene(enum.Roleplay, "Market")
	_ = s.FinishScene("Done")

	err := s.FinishScene("Done again")
	if err == nil {
		t.Error("expected error when finishing already finished scene")
	}
}
