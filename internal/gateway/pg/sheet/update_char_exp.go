package sheet

import "context"

func (r *Repository) UpdateCharExp(ctx context.Context, uuid string, charExp int) error {
	const query = `
		UPDATE character_sheets
		SET char_exp = $1
		WHERE uuid = $2
	`
	_, err := r.q.Exec(ctx, query, charExp, uuid)
	return err
}
