package match

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
)

// ISheetStatusWriter persists a character's status bars.
//
// Combat damage lands on the live sheet inside the session, which is in memory and dies
// with the process. The row has to follow, and it is the only way the change reaches a
// player: the match sidebar reads HP over REST, and there is no WS event for it — that is
// front work, deliberately not this slice's.
type ISheetStatusWriter interface {
	UpdateStatusBars(ctx context.Context, sheetUUID string, health, stamina, aura status.IStatusBar) error
}
