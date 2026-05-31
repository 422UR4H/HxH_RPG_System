// internal/gateway/pg/map/delete_map.go
package pgmap

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (r *Repository) DeleteMap(ctx context.Context, id uuid.UUID) error {
	const query = `DELETE FROM maps WHERE uuid = $1`
	_, err := r.q.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete map: %w", err)
	}
	return nil
}
