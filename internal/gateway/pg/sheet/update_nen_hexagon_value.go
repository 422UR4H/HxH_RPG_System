package sheet

import "context"

func (r *Repository) UpdateNenHexagonValue(ctx context.Context, uuid string, val int) error {
	const query = `
		UPDATE character_sheets
		SET curr_hex_value = $1
		WHERE uuid = $2
	`
	_, err := r.q.Exec(ctx, query, val, uuid)
	if err != nil {
		return err
	}
	return nil
}
