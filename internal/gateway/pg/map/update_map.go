// internal/gateway/pg/map/update_map.go
package pgmap

import (
	"context"
	"fmt"
	"time"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
)

func (r *Repository) UpdateMap(ctx context.Context, m *entity.TacticalMap) error {
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

	m.UpdatedAt = time.Now().UTC()

	const query = `
		UPDATE maps SET
			name=$1, description=$2, fog_mode=$3, grid=$4, bg=$5,
			pieces=$6, walls=$7, decorations=$8, items=$9, updated_at=$10
		WHERE uuid=$11
	`
	_, err = r.q.Exec(ctx, query,
		m.Name, m.Description, string(m.FogMode), grid, bg,
		pieces, walls, decorations, items, m.UpdatedAt,
		m.ID,
	)
	if err != nil {
		return fmt.Errorf("update map: %w", err)
	}
	return nil
}
