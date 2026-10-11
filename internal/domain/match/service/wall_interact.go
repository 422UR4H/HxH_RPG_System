package service

import (
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

// ApplyWallInteract applies an interact action to a wall segment and returns the updated wall,
// with ok=false when nothing changes hands: the kind does not apply to this wall, or to whoever
// is acting on it.
//
//   - open / close / toggle move a passage (door, window, secret door) that is still standing.
//     A character cannot open a LOCKED passage — the open simply fails, the passage stays shut;
//     closing one always works. The master's hand ignores the lock.
//   - lock / unlock are the master's alone, on doors and windows.
//   - lockpick / examine need a roll check that is not built yet: they change nothing. The master
//     reads the declared attempt and unlocks the door by hand when it succeeds.
func ApplyWallInteract(w mapentity.WallSegment, interact *action.Interact, byMaster bool) (mapentity.WallSegment, bool) {
	if w.Destroyed {
		return w, false
	}
	switch interact.Kind {
	case action.InteractOpen:
		if !isPassage(w) || (w.Locked && !byMaster) {
			return w, false
		}
		w.Open = true
	case action.InteractClose:
		if !isPassage(w) {
			return w, false
		}
		w.Open = false
	case action.InteractToggle:
		if !isPassage(w) || (!w.Open && w.Locked && !byMaster) {
			return w, false
		}
		w.Open = !w.Open
	case action.InteractLock, action.InteractUnlock:
		if !byMaster || !isLockable(w) {
			return w, false
		}
		w.Locked = interact.Kind == action.InteractLock
	default:
		// lockpick, examine — require roll check; not yet handled
		return w, false
	}
	return w, true
}

func isPassage(w mapentity.WallSegment) bool {
	return w.WallType == mapentity.WallTypeDoor || w.WallType == mapentity.WallTypeWindow ||
		w.WallType == mapentity.WallTypeSecretDoor
}

func isLockable(w mapentity.WallSegment) bool {
	return w.WallType == mapentity.WallTypeDoor || w.WallType == mapentity.WallTypeWindow
}
