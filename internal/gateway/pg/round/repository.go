package round

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository implements persistence for round-scoped operations
// (scenes, rounds, turns, actions).
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}
