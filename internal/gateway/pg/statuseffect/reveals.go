package pgstatuseffect

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// InsertReveal grava uma revelação do mestre (status_effect_reveals): a partir de at, o que o
// efeito escondeu até at fica visível a todos. Cada chamada é uma linha nova.
func (r *Repository) InsertReveal(ctx context.Context, matchUUID, effectUUID uuid.UUID, at time.Time) error {
	const q = `
		INSERT INTO status_effect_reveals (uuid, effect_uuid, match_uuid, revealed_at)
		VALUES ($1, $2, $3, $4)
	`
	if _, err := r.q.Exec(ctx, q, uuid.New(), effectUUID, matchUUID, at); err != nil {
		return fmt.Errorf("insert status effect reveal: %w", err)
	}
	return nil
}

// RevealedAt é, por efeito, a última revelação (MAX em todas as partidas). Nunca devolve nil;
// entrada vazia não consulta o banco; um efeito nunca revelado não aparece.
func (r *Repository) RevealedAt(ctx context.Context, effectIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	out := map[uuid.UUID]time.Time{}
	if len(effectIDs) == 0 {
		return out, nil
	}
	const q = `
		SELECT effect_uuid, MAX(revealed_at)
		FROM status_effect_reveals
		WHERE effect_uuid = ANY($1)
		GROUP BY effect_uuid
	`
	rows, err := r.q.Query(ctx, q, effectIDs)
	if err != nil {
		return nil, fmt.Errorf("status effect reveals: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var at time.Time
		if err := rows.Scan(&id, &at); err != nil {
			return nil, fmt.Errorf("scan status effect reveal: %w", err)
		}
		out[id] = at.UTC()
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status effect reveals: %w", err)
	}
	return out, nil
}

// RevealableInMatch diz se o efeito é revelável nesta partida: tem algum evento de status, em
// QUALQUER partida, e a ficha dele (status_effect_events.character_sheet_uuid) é participante
// desta (match_participants). Um Escondido aplicado na partida A e ainda ativo na B esconde
// turnos da B — o concealedBy da B o nomeia, e o mestre da B precisa poder revelá-lo. Índices:
// idx_status_effect_events_effect (effect_uuid), idx_match_participants_character_sheet_uuid e
// idx_match_participants_match_uuid.
func (r *Repository) RevealableInMatch(ctx context.Context, matchUUID, effectUUID uuid.UUID) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1
			FROM status_effect_events e
			JOIN match_participants mp ON mp.character_sheet_uuid = e.character_sheet_uuid
			WHERE e.effect_uuid = $2 AND mp.match_uuid = $1
		)
	`
	var ok bool
	if err := r.q.QueryRow(ctx, q, matchUUID, effectUUID).Scan(&ok); err != nil {
		return false, fmt.Errorf("status effect revealable in match: %w", err)
	}
	return ok, nil
}
