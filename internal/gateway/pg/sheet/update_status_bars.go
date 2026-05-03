package sheet

import (
	"context"

	"github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/model"
)

func (r *Repository) UpdateStatusBars(
	ctx context.Context,
	sheetUUID string,
	health, stamina, aura model.StatusBar,
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
			aura_max_pts     = $9
		WHERE uuid = $10
	`
	_, err := r.q.Exec(ctx, query,
		health.Min, health.Curr, health.Max,
		stamina.Min, stamina.Curr, stamina.Max,
		aura.Min, aura.Curr, aura.Max,
		sheetUUID,
	)
	return err
}
