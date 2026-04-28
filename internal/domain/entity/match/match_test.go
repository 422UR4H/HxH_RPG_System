package match_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/scene"
	"github.com/google/uuid"
)

func TestNewMatch(t *testing.T) {
	masterUUID := uuid.New()
	campaignUUID := uuid.New()
	gameStart := time.Now().Add(24 * time.Hour)
	storyStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	m, err := match.NewMatch(
		masterUUID, campaignUUID,
		"Test Match", "A brief description",
		"Full description", true,
		gameStart, storyStart,
	)

	if err != nil {
		t.Fatalf("NewMatch() error = %v", err)
	}
	if m.UUID == uuid.Nil {
		t.Error("UUID should not be nil")
	}
	if m.MasterUUID != masterUUID {
		t.Errorf("MasterUUID = %v, want %v", m.MasterUUID, masterUUID)
	}
	if m.CampaignUUID != campaignUUID {
		t.Errorf("CampaignUUID = %v, want %v", m.CampaignUUID, campaignUUID)
	}
	if m.Title != "Test Match" {
		t.Errorf("Title = %s, want Test Match", m.Title)
	}
	if m.BriefInitialDescription != "A brief description" {
		t.Errorf("BriefInitialDescription = %s", m.BriefInitialDescription)
	}
	if m.Description != "Full description" {
		t.Errorf("Description = %s", m.Description)
	}
	if !m.IsPublic {
		t.Error("IsPublic = false, want true")
	}
	if m.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestMatch_AddScene_GetScenes(t *testing.T) {
	m, _ := match.NewMatch(
		uuid.New(), uuid.New(),
		"Test", "Brief", "Desc", false,
		time.Now(), time.Now(),
	)

	if len(m.GetScenes()) != 0 {
		t.Fatalf("initial scenes should be empty")
	}

	s1 := scene.NewScene(enum.Roleplay, "Opening scene")
	s2 := scene.NewScene(enum.Battle, "First battle")

	m.AddScene(s1)
	m.AddScene(s2)

	scenes := m.GetScenes()
	if len(scenes) != 2 {
		t.Fatalf("GetScenes() len = %d, want 2", len(scenes))
	}
	if scenes[0].GetCategory() != enum.Roleplay {
		t.Errorf("first scene category = %v, want Roleplay", scenes[0].GetCategory())
	}
	if scenes[1].GetCategory() != enum.Battle {
		t.Errorf("second scene category = %v, want Battle", scenes[1].GetCategory())
	}
}
