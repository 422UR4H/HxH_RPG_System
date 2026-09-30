// internal/gateway/pg/matchboard/repository.go
package pgmatchboard

import (
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
)

// Repository implements persistence for the match board (spec §4.3, B3, B14): a
// per-match snapshot that starts as a copy of the campaign map it was attached to.
type Repository struct {
	q pgfs.IQuerier
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{q: q}
}
