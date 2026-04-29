package campaign_test

import (
	"testing"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/campaign"
	"github.com/google/uuid"
)

func TestNewCampaign(t *testing.T) {
	masterUUID := uuid.New()
	scenarioUUID := uuid.New()
	storyStart := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)

	c, err := campaign.NewCampaign(
		masterUUID,
		&scenarioUUID,
		"Test Campaign",
		"Brief desc",
		"Full description",
		true,
		"https://meet.example.com",
		storyStart,
		nil,
	)

	if err != nil {
		t.Fatalf("NewCampaign() error = %v", err)
	}
	if c.UUID == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if c.MasterUUID != masterUUID {
		t.Errorf("got MasterUUID %v, want %v", c.MasterUUID, masterUUID)
	}
	if c.Name != "Test Campaign" {
		t.Errorf("got Name %q, want %q", c.Name, "Test Campaign")
	}
	if !c.IsPublic {
		t.Error("expected IsPublic to be true")
	}
	if c.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if c.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

func TestNewCampaign_WithOptionalStoryCurrentAt(t *testing.T) {
	storyCurrent := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	c, err := campaign.NewCampaign(
		uuid.New(),
		nil,
		"Campaign",
		"",
		"",
		false,
		"",
		time.Now(),
		&storyCurrent,
	)

	if err != nil {
		t.Fatalf("NewCampaign() error = %v", err)
	}
	if c.StoryCurrentAt == nil {
		t.Fatal("expected StoryCurrentAt to be set")
	}
	if !c.StoryCurrentAt.Equal(storyCurrent) {
		t.Errorf("got StoryCurrentAt %v, want %v", *c.StoryCurrentAt, storyCurrent)
	}
	if c.ScenarioUUID != nil {
		t.Error("expected ScenarioUUID to be nil")
	}
}
