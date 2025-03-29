package sheet

import "context"

func (r *Repository) IncreaseNenHexagonValue(ctx context.Context, uuid string) error {
	const query = `
		UPDATE character_sheets
		SET nen_hexagon_value = nen_hexagon_value + 1
		WHERE character_sheet_uuid = $1
	`
	_, err := r.q.Exec(ctx, query, uuid)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) DecreaseNenHexagonValue(ctx context.Context, uuid string) error {
	const query = `
		UPDATE character_sheets
		SET nen_hexagon_value = nen_hexagon_value - 1
		WHERE character_sheet_uuid = $1
	`
	_, err := r.q.Exec(ctx, query, uuid)
	if err != nil {
		return err
	}
	return nil
}
