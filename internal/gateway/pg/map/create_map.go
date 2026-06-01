// internal/gateway/pg/map/create_map.go
package pgmap

import (
	"context"
	"fmt"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func (r *Repository) CreateMap(ctx context.Context, m *entity.TacticalMap) error {
	grid, err := marshalJSON(m.Grid)
	if err != nil {
		return err
	}
	pieces, err := marshalJSON(m.Pieces)
	if err != nil {
		return err
	}
	walls, err := marshalJSON(m.Walls)
	if err != nil {
		return err
	}
	decorations, err := marshalJSON(m.Decorations)
	if err != nil {
		return err
	}
	items, err := marshalJSON(m.Items)
	if err != nil {
		return err
	}

	var bg []byte
	if m.Bg != nil {
		bg, err = marshalJSON(m.Bg)
		if err != nil {
			return err
		}
	}

	const query = `
		INSERT INTO maps (
			uuid, campaign_uuid, name, description,
			grid, bg, pieces, walls, decorations, items,
			created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
	`
	_, err = r.q.Exec(ctx, query,
		m.ID, m.CampaignID, m.Name, m.Description,
		grid, bg, pieces, walls, decorations, items,
		m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create map: %w", err)
	}
	return nil
}
