package enrollment

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) ListPlayerEnrollmentStatusesForCampaign(
	ctx context.Context,
	playerUUID uuid.UUID,
	campaignUUID uuid.UUID,
) (map[uuid.UUID]string, error) {
	const query = `
		SELECT e.match_uuid, e.status
		FROM enrollments e
		JOIN character_sheets cs ON cs.uuid = e.character_sheet_uuid
		JOIN matches m ON m.uuid = e.match_uuid
		WHERE cs.player_uuid = $1 AND m.campaign_uuid = $2
	`
	rows, err := r.q.Query(ctx, query, playerUUID, campaignUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to list player enrollment statuses: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]string)
	for rows.Next() {
		var matchUUID uuid.UUID
		var status string
		if err := rows.Scan(&matchUUID, &status); err != nil {
			return nil, fmt.Errorf("failed to scan enrollment row: %w", err)
		}
		result[matchUUID] = status
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate enrollment rows: %w", err)
	}
	return result, nil
}
