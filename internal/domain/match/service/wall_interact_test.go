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
		updated, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractOpen}, false)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if !updated.Open {
			t.Error("expected Open=true")
		}
	})

	t.Run("InteractClose sets Open=false", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(openDoor, &action.Interact{Kind: action.InteractClose}, false)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if updated.Open {
			t.Error("expected Open=false")
		}
	})

	t.Run("InteractToggle flips Open", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractToggle}, false)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if !updated.Open {
			t.Error("expected Open=true after toggle")
		}
	})

	t.Run("InteractLockpick returns ok=false (roll required, not yet handled)", func(t *testing.T) {
		_, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractLockpick}, false)
		if ok {
			t.Error("expected ok=false for lockpick")
		}
	})

	t.Run("InteractExamine returns ok=false (roll required, not yet handled)", func(t *testing.T) {
		_, ok := service.ApplyWallInteract(closedDoor, &action.Interact{Kind: action.InteractExamine}, false)
		if ok {
			t.Error("expected ok=false for examine")
		}
	})

	lockedDoor := mapentity.WallSegment{ID: "door-3", WallType: mapentity.WallTypeDoor, Locked: true}
	lockedOpenDoor := mapentity.WallSegment{ID: "door-4", WallType: mapentity.WallTypeDoor, Open: true, Locked: true}
	plainWall := mapentity.WallSegment{ID: "wall-1", WallType: mapentity.WallTypeWall}
	open := &action.Interact{Kind: action.InteractOpen}

	t.Run("a character cannot open a locked door: nothing changes", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(lockedDoor, open, false)
		if ok || updated.Open {
			t.Errorf("expected the locked door to stay shut, got ok=%v open=%v", ok, updated.Open)
		}
	})

	t.Run("a character cannot toggle a locked door open, but can toggle it shut", func(t *testing.T) {
		if _, ok := service.ApplyWallInteract(lockedDoor, &action.Interact{Kind: action.InteractToggle}, false); ok {
			t.Error("expected ok=false toggling a closed locked door")
		}
		updated, ok := service.ApplyWallInteract(lockedOpenDoor, &action.Interact{Kind: action.InteractToggle}, false)
		if !ok || updated.Open {
			t.Errorf("expected the open locked door to close, got ok=%v open=%v", ok, updated.Open)
		}
	})

	t.Run("a character closes a locked open door", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(lockedOpenDoor, &action.Interact{Kind: action.InteractClose}, false)
		if !ok || updated.Open {
			t.Errorf("expected close to work, got ok=%v open=%v", ok, updated.Open)
		}
	})

	t.Run("the master opens a locked door, and the lock stays", func(t *testing.T) {
		updated, ok := service.ApplyWallInteract(lockedDoor, open, true)
		if !ok || !updated.Open || !updated.Locked {
			t.Errorf("expected open and still locked, got ok=%v open=%v locked=%v", ok, updated.Open, updated.Locked)
		}
	})

	t.Run("lock and unlock are the master's, on doors and windows", func(t *testing.T) {
		lock := &action.Interact{Kind: action.InteractLock}
		unlock := &action.Interact{Kind: action.InteractUnlock}
		if _, ok := service.ApplyWallInteract(closedDoor, lock, false); ok {
			t.Error("a character must not lock")
		}
		updated, ok := service.ApplyWallInteract(closedDoor, lock, true)
		if !ok || !updated.Locked {
			t.Errorf("expected the master to lock, got ok=%v locked=%v", ok, updated.Locked)
		}
		updated, ok = service.ApplyWallInteract(lockedDoor, unlock, true)
		if !ok || updated.Locked {
			t.Errorf("expected the master to unlock, got ok=%v locked=%v", ok, updated.Locked)
		}
		window := mapentity.WallSegment{ID: "win-1", WallType: mapentity.WallTypeWindow}
		if updated, ok := service.ApplyWallInteract(window, lock, true); !ok || !updated.Locked {
			t.Error("expected a window to lock")
		}
		if _, ok := service.ApplyWallInteract(plainWall, lock, true); ok {
			t.Error("a plain wall has no lock")
		}
	})

	t.Run("only a standing passage opens", func(t *testing.T) {
		if _, ok := service.ApplyWallInteract(plainWall, open, true); ok {
			t.Error("a plain wall does not open")
		}
		broken := closedDoor
		broken.Destroyed = true
		if _, ok := service.ApplyWallInteract(broken, open, false); ok {
			t.Error("a destroyed door does not open")
		}
	})
}
