package match

import (
	"context"
	"errors"
	"fmt"
	"time"

	matchmapuc "github.com/422UR4H/HxH_RPG_System/internal/application/matchmap"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMatchInfo(ctx context.Context, matchUUID uuid.UUID) (*matchmapuc.MatchInfo, error) {
	const query = `
		SELECT master_uuid, game_start_at
		FROM matches
		WHERE uuid = $1
	`
	var masterUUID uuid.UUID
	var gameStartAt *time.Time
	err := r.q.QueryRow(ctx, query, matchUUID).Scan(&masterUUID, &gameStartAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMatchNotFound
		}
		return nil, fmt.Errorf("get match info: %w", err)
	}
	return &matchmapuc.MatchInfo{
		MasterUUID:  masterUUID,
		GameStartAt: gameStartAt,
	}, nil
}
