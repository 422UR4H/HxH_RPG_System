package pgmatchmap

import (
	"context"
	"fmt"
	"time"

	entity "github.com/422UR4H/HxH_RPG_System/internal/domain/matchmap/entity"
	"github.com/google/uuid"
)

func (r *Repository) AttachMap(ctx context.Context, matchUUID, mapUUID uuid.UUID) (*entity.MatchMap, error) {
	const query = `
		INSERT INTO match_maps (match_uuid, map_uuid, attached_at)
		VALUES ($1, $2, now())
		ON CONFLICT (match_uuid) DO UPDATE
			SET map_uuid = EXCLUDED.map_uuid,
			    attached_at = now()
		RETURNING attached_at
	`
	var attachedAt time.Time
	err := r.q.QueryRow(ctx, query, matchUUID, mapUUID).Scan(&attachedAt)
	if err != nil {
		return nil, fmt.Errorf("attach match map: %w", err)
	}
	return &entity.MatchMap{
		MatchUUID:  matchUUID.String(),
		MapUUID:    mapUUID.String(),
		AttachedAt: attachedAt,
	}, nil
}
