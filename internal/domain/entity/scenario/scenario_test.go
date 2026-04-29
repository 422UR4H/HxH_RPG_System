package scenario_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/scenario"
	"github.com/google/uuid"
)

func TestNewScenario(t *testing.T) {
	userUUID := uuid.New()

	s, err := scenario.NewScenario(userUUID, "Test Scenario", "Brief", "Full description")

	if err != nil {
		t.Fatalf("NewScenario() error = %v", err)
	}
	if s.UUID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if s.UserUUID != userUUID {
		t.Errorf("got UserUUID %v, want %v", s.UserUUID, userUUID)
	}
	if s.Name != "Test Scenario" {
		t.Errorf("got Name %q, want %q", s.Name, "Test Scenario")
	}
	if s.BriefDescription != "Brief" {
		t.Errorf("got BriefDescription %q, want %q", s.BriefDescription, "Brief")
	}
	if s.Description != "Full description" {
		t.Errorf("got Description %q, want %q", s.Description, "Full description")
	}
	if s.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if s.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	if s.Campaigns != nil {
		t.Error("expected Campaigns to be nil for new scenario")
	}
}

func TestNewScenario_GeneratesUniqueUUIDs(t *testing.T) {
	s1, _ := scenario.NewScenario(uuid.New(), "A", "", "")
	s2, _ := scenario.NewScenario(uuid.New(), "B", "", "")

	if s1.UUID == s2.UUID {
		t.Error("expected different UUIDs for different scenarios")
	}
}
