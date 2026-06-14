package service

import (
	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
)

// ApplyWallInteract applies an interact action to a wall segment.
// Returns the updated wall and ok=true for open/close/toggle.
// Returns ok=false for lockpick/examine — these require a skill roll (TODO).
func ApplyWallInteract(w mapentity.WallSegment, interact *action.Interact) (mapentity.WallSegment, bool) {
	switch interact.Kind {
	case action.InteractOpen:
		w.Open = true
	case action.InteractClose:
		w.Open = false
	case action.InteractToggle:
		w.Open = !w.Open
	default:
		// lockpick, examine — require roll check; not yet handled
		return w, false
	}
	return w, true
}
