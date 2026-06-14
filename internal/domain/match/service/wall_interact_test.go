package service_test

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
)

func TestApplyWallInteract(t *testing.T) {
	closedDoor := mapentity.WallSegment{ID: "door-1", WallType: mapentity.WallTypeDoor, Open: false, Locked: false}
	openDoor := mapentity.WallSegment{ID: "door-2", WallType: mapentity.WallTypeDoor, Open: true, Locked: false}

	t.Run("InteractOpen sets Open=true", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractOpen})
		if !ok {
			t.Fatal("expected ok=true")
		}
		if !updated.Open {
			t.Error("expected Open=true")
		}
	})

	t.Run("InteractClose sets Open=false", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(openDoor, &action.Interact{Kind: action.InteractClose})
		if !ok {
			t.Fatal("expected ok=true")
		}
		if updated.Open {
			t.Error("expected Open=false")
		}
	})

	t.Run("InteractToggle flips Open", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractToggle})
		if !ok {
			t.Fatal("expected ok=true")
		}
		if !updated.Open {
			t.Error("expected Open=true after toggle")
		}
	})

	t.Run("InteractLockpick returns ok=false (roll required, not yet handled)", func(t *testing.T) {
		_, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractLockpick})
		if ok {
			t.Error("expected ok=false for lockpick")
		}
	})

	t.Run("InteractExamine returns ok=false (roll required, not yet handled)", func(t *testing.T) {
		_, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractExamine})
		if ok {
			t.Error("expected ok=false for examine")
		}
	})
}
