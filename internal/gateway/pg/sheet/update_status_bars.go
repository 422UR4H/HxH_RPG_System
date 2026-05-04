// internal/gateway/pg/sheet/update_status_bars.go
package sheet

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
)

func (r *Repository) UpdateStatusBars(
	ctx context.Context,
	sheetUUID string,
	health, stamina, aura status.IStatusBar,
) error {
	const query = `
		UPDATE character_sheets
		SET
			health_min_pts   = $1,
			health_curr_pts  = $2,
			health_max_pts   = $3,
			stamina_min_pts  = $4,
			stamina_curr_pts = $5,
			stamina_max_pts  = $6,
			aura_min_pts     = $7,
			aura_curr_pts    = $8,
			aura_max_pts     = $9,
			updated_at       = $10
		WHERE uuid = $11
	`
	_, err := r.q.Exec(ctx, query,
		health.GetMin(), health.GetCurrent(), health.GetMax(),
		stamina.GetMin(), stamina.GetCurrent(), stamina.GetMax(),
		aura.GetMin(), aura.GetCurrent(), aura.GetMax(),
		time.Now(),
		sheetUUID,
	)
	return err
}
