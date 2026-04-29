package match

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (r *Repository) GetMatchMaster(
	ctx context.Context, matchUUID uuid.UUID,
) (uuid.UUID, error) {
	const query = `
		SELECT master_uuid
		FROM matches
		WHERE uuid = $1
	`
	var masterUUID uuid.UUID
	err := r.q.QueryRow(ctx, query, matchUUID).Scan(&masterUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrMatchNotFound
		}
		return uuid.Nil, fmt.Errorf("failed to get match master: %w", err)
	}
	return masterUUID, nil
}
