// internal/gateway/pg/map/list_maps.go
package pgmap

import (
	"context"
	"fmt"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
)

func (r *Repository) ListMapsByCampaign(
	ctx context.Context, campaignID uuid.UUID,
) ([]*entity.TacticalMap, error) {
	const query = `
		SELECT uuid, campaign_uuid, name, description,
		       grid, bg, pieces, walls, decorations, items,
		       created_at, updated_at
		FROM maps WHERE campaign_uuid = $1
		ORDER BY created_at ASC
	`
	rows, err := r.q.Query(ctx, query, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list maps: %w", err)
	}
	defer rows.Close()

	var result []*entity.TacticalMap
	for rows.Next() {
		m := &pgModel{}
		if err := rows.Scan(
			&m.ID, &m.CampaignID, &m.Name, &m.Description,
			&m.Grid, &m.Bg, &m.Pieces, &m.Walls, &m.Decorations, &m.Items,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan map row: %w", err)
		}
		e, err := toEntity(m)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
