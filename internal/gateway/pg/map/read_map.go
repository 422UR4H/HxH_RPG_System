// internal/gateway/pg/map/read_map.go
package pgmap

import (
	"context"
	"errors"
	"fmt"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMap(ctx context.Context, id uuid.UUID) (*entity.TacticalMap, error) {
	const query = `
		SELECT uuid, campaign_uuid, name, description,
		       grid, bg, pieces, walls, decorations, items,
		       created_at, updated_at
		FROM maps WHERE uuid = $1
	`
	row := r.q.QueryRow(ctx, query, id)
	m := &pgModel{}
	err := row.Scan(
		&m.ID, &m.CampaignID, &m.Name, &m.Description,
		&m.Grid, &m.Bg, &m.Pieces, &m.Walls, &m.Decorations, &m.Items,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMapNotFound
		}
		return nil, fmt.Errorf("get map: %w", err)
	}
	return toEntity(m)
}
