package sheet

import (
	"context"
	"fmt"

	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
)

type Repository struct {
	q pgfs.IQuerier
	// logger
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{
		q: q,
	}
}

func (r *Repository) Test(ctx context.Context, data string) error {
	// test conn
	fmt.Println(data)
	return nil
}
