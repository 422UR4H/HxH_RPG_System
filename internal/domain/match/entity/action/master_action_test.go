package action_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

func TestMasterAction_SetHappenedAt(t *testing.T) {
	ma := action.NewMasterAction()
	if !ma.GetHappenedAt().IsZero() {
		t.Error("expected zero happenedAt on new MasterAction")
	}
	now := time.Now()
	ma.SetHappenedAt(now)
	if !ma.GetHappenedAt().Equal(now) {
		t.Errorf("expected happenedAt %v, got %v", now, ma.GetHappenedAt())
	}
}

func TestMasterAction_Skills(t *testing.T) {
	ma := action.NewMasterAction()
	ma.Skills = []action.Skill{{SkillName: "Gyo"}}
	got := ma.GetSkills()
	if len(got) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(got))
	}
	if got[0].SkillName != "Gyo" {
		t.Errorf("expected SkillName 'Gyo', got %q", got[0].SkillName)
	}
}
