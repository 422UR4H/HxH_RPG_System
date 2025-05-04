package campaign

import (
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
)

type Repository struct {
	q pgfs.IQuerier
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{q: q}
}
