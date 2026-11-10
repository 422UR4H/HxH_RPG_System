package match

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// toPgTime converte o deslocamento desde a meia-noite para a coluna TIME (nil = NULL).
func toPgTime(d *time.Duration) pgtype.Time {
	if d == nil {
		return pgtype.Time{}
	}
	return pgtype.Time{Microseconds: d.Microseconds(), Valid: true}
}

// fromPgTime faz o caminho inverso, em segundos inteiros.
func fromPgTime(t pgtype.Time) *time.Duration {
	if !t.Valid {
		return nil
	}
	d := (time.Duration(t.Microseconds) * time.Microsecond).Truncate(time.Second)
	return &d
}
