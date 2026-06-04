package pgmatchmap

import (
	"context"
	"errors"
	"fmt"
	"time"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMatchMap(ctx context.Context, matchUUID uuid.UUID) (*entity.MatchMap, error) {
	const query = `
		SELECT map_uuid, attached_at
		FROM match_maps
		WHERE match_uuid = $1
	`
	var mapUUIDVal uuid.UUID
	var attachedAt time.Time
	err := r.q.QueryRow(ctx, query, matchUUID).Scan(&mapUUIDVal, &attachedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMatchMapNotFound
		}
		return nil, fmt.Errorf("get match map: %w", err)
	}
	return &entity.MatchMap{
		MatchUUID:  matchUUID.String(),
		MapUUID:    mapUUIDVal.String(),
		AttachedAt: attachedAt,
	}, nil
}
