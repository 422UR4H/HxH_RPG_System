package pgstatuseffect

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// snapshotRow é o efeito como o evento o guarda (status_effect_events.snapshot): o mesmo
// formato de params e audience de character_status_effects, mais a identidade do efeito.
type snapshotRow struct {
	ID          uuid.UUID       `json:"id"`
	CharacterID uuid.UUID       `json:"characterId"`
	Kind        string          `json:"kind"`
	Sense       string          `json:"sense,omitempty"`
	Aptitude    string          `json:"aptitude,omitempty"`
	Params      json.RawMessage `json:"params"`
	Alias       string          `json:"alias,omitempty"`
	Audience    json.RawMessage `json:"audience,omitempty"`
	AppliedAt   time.Time       `json:"appliedAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

func encodeSnapshot(e statuseffect.Effect) ([]byte, error) {
	params, err := encodeParams(e.Params)
	if err != nil {
		return nil, err
	}
	audience, err := encodeAudience(e.Audience)
	if err != nil {
		return nil, err
	}
	return json.Marshal(snapshotRow{
		ID: e.ID, CharacterID: e.CharacterID, Kind: string(e.Kind), Sense: string(e.Sense),
		Aptitude: string(e.Aptitude), Params: params, Alias: e.Alias, Audience: audience,
		AppliedAt: e.AppliedAt, UpdatedAt: e.UpdatedAt,
	})
}

func decodeSnapshot(raw []byte) (statuseffect.Effect, error) {
	var row snapshotRow
	if err := json.Unmarshal(raw, &row); err != nil {
		return statuseffect.Effect{}, err
	}
	e := statuseffect.Effect{
		ID: row.ID, CharacterID: row.CharacterID, Kind: statuseffect.Kind(row.Kind),
		Sense: statuseffect.Sense(row.Sense), Aptitude: statuseffect.Aptitude(row.Aptitude),
		Alias: row.Alias, AppliedAt: row.AppliedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(),
	}
	var err error
	if e.Params, err = decodeParams(row.Params); err != nil {
		return statuseffect.Effect{}, fmt.Errorf("params: %w", err)
	}
	if len(row.Audience) > 0 {
		if e.Audience, err = decodeAudience(row.Audience); err != nil {
			return statuseffect.Effect{}, fmt.Errorf("audience: %w", err)
		}
	}
	return e, nil
}

// encodeListing grava a lista resolvida; nunca NULL (Players vazio vira []).
func encodeListing(a statuseffect.Audience) ([]byte, error) {
	return encodeAudience(&a)
}

func decodeListing(raw []byte) (statuseffect.Audience, error) {
	a, err := decodeAudience(raw)
	if err != nil {
		return statuseffect.Audience{}, err
	}
	if a == nil {
		return statuseffect.Audience{}, fmt.Errorf("listing NULL")
	}
	return *a, nil
}

// insertEvent grava o evento de op na mesma transação da operação. Uma ficha ou partida que já
// não existe pula o evento (como a operação pula a ficha), e um id repetido — a mesma gravação
// tentada de novo depois de uma falha — é ignorado.
func insertEvent(ctx context.Context, tx pgx.Tx, op statuseffect.Write) error {
	ev := op.Event
	snapshot, err := encodeSnapshot(op.Effect)
	if err != nil {
		return fmt.Errorf("marshal status effect event snapshot: %w", err)
	}
	listing, err := encodeListing(ev.Listing)
	if err != nil {
		return fmt.Errorf("marshal status effect event listing: %w", err)
	}
	views := ev.Views
	if views == nil {
		views = map[uuid.UUID]masteraction.View{}
	}
	viewsJSON, err := json.Marshal(views)
	if err != nil {
		return fmt.Errorf("marshal status effect event views: %w", err)
	}
	var hiddenFrom []byte
	if len(ev.HiddenFrom) > 0 {
		if hiddenFrom, err = json.Marshal(ev.HiddenFrom); err != nil {
			return fmt.Errorf("marshal status effect event hidden_from: %w", err)
		}
	}
	const q = `
		INSERT INTO status_effect_events
			(uuid, effect_uuid, character_sheet_uuid, match_uuid, scene_uuid, round_uuid, turn_uuid,
			 action_uuid, change, snapshot, listing, views, hidden_from, happened_at)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		WHERE EXISTS (SELECT 1 FROM character_sheets WHERE uuid = $3)
		  AND EXISTS (SELECT 1 FROM matches WHERE uuid = $4)
		ON CONFLICT (uuid) DO NOTHING
	`
	if _, err := tx.Exec(ctx, q,
		ev.ID, op.Effect.ID, op.Effect.CharacterID, ev.MatchID, ev.SceneID, ev.RoundID, ev.TurnID,
		ev.ActionID, string(ev.Change), snapshot, listing, viewsJSON, hiddenFrom, ev.HappenedAt,
	); err != nil {
		return fmt.Errorf("insert status effect event %s: %w", ev.ID, err)
	}
	return nil
}

// ListEventsByMatch devolve os eventos de status da partida em ordem happened_at, seq. Nunca
// devolve nil. Uma linha que não decodifica é registrada no log e pulada: o histórico segue
// sem ela.
func (r *Repository) ListEventsByMatch(ctx context.Context, matchUUID uuid.UUID) ([]statuseffect.EventRecord, error) {
	const q = `
		SELECT uuid, effect_uuid, character_sheet_uuid, match_uuid, scene_uuid, round_uuid, turn_uuid,
		       action_uuid, change, snapshot, listing, views, hidden_from, happened_at
		FROM status_effect_events
		WHERE match_uuid = $1
		ORDER BY happened_at, seq
	`
	rows, err := r.q.Query(ctx, q, matchUUID)
	if err != nil {
		return nil, fmt.Errorf("list status effect events: %w", err)
	}
	defer rows.Close()

	out := []statuseffect.EventRecord{}
	for rows.Next() {
		var ev statuseffect.Event
		var effectID, sheetID uuid.UUID
		var change string
		var snapshot, listing, views, hiddenFrom []byte
		if err := rows.Scan(&ev.ID, &effectID, &sheetID, &ev.MatchID, &ev.SceneID, &ev.RoundID,
			&ev.TurnID, &ev.ActionID, &change, &snapshot, &listing, &views, &hiddenFrom, &ev.HappenedAt); err != nil {
			return nil, fmt.Errorf("scan status effect event: %w", err)
		}
		ev.Change = statuseffect.Change(change)
		ev.HappenedAt = ev.HappenedAt.UTC()
		rec, err := decodeEvent(ev, effectID, sheetID, snapshot, listing, views, hiddenFrom)
		if err != nil {
			log.Printf("evento de status %s da partida %s não decodifica e fica fora do histórico: %v",
				ev.ID, matchUUID, err)
			continue
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status effect events: %w", err)
	}
	return out, nil
}

func decodeEvent(
	ev statuseffect.Event, effectID, sheetID uuid.UUID, snapshot, listing, views, hiddenFrom []byte,
) (statuseffect.EventRecord, error) {
	effect, err := decodeSnapshot(snapshot)
	if err != nil {
		return statuseffect.EventRecord{}, fmt.Errorf("snapshot: %w", err)
	}
	// As colunas são a verdade sobre de quem é o evento.
	effect.ID, effect.CharacterID = effectID, sheetID
	if ev.Listing, err = decodeListing(listing); err != nil {
		return statuseffect.EventRecord{}, fmt.Errorf("listing: %w", err)
	}
	ev.Views = map[uuid.UUID]masteraction.View{}
	if err := json.Unmarshal(views, &ev.Views); err != nil {
		return statuseffect.EventRecord{}, fmt.Errorf("views: %w", err)
	}
	if ev.Views == nil { // "null" gravado à mão
		return statuseffect.EventRecord{}, fmt.Errorf("views: null")
	}
	if hiddenFrom != nil {
		if err := json.Unmarshal(hiddenFrom, &ev.HiddenFrom); err != nil {
			return statuseffect.EventRecord{}, fmt.Errorf("hidden_from: %w", err)
		}
	}
	return statuseffect.EventRecord{Event: ev, Effect: effect}, nil
}

// LatestListings é, por efeito, a lista do seu último evento (maior seq), em qualquer partida.
// Nunca devolve nil; entrada vazia não consulta o banco. Um efeito sem evento não aparece; um
// cuja lista não decodifica também não (falha fechada: o histórico o esconde dos jogadores).
func (r *Repository) LatestListings(ctx context.Context, effectIDs []uuid.UUID) (map[uuid.UUID]statuseffect.Audience, error) {
	out := map[uuid.UUID]statuseffect.Audience{}
	if len(effectIDs) == 0 {
		return out, nil
	}
	const q = `
		SELECT DISTINCT ON (effect_uuid) effect_uuid, listing
		FROM status_effect_events
		WHERE effect_uuid = ANY($1)
		ORDER BY effect_uuid, seq DESC
	`
	rows, err := r.q.Query(ctx, q, effectIDs)
	if err != nil {
		return nil, fmt.Errorf("latest status effect listings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var raw []byte
		if err := rows.Scan(&id, &raw); err != nil {
			return nil, fmt.Errorf("scan status effect listing: %w", err)
		}
		a, err := decodeListing(raw)
		if err != nil {
			log.Printf("última lista do efeito de status %s não decodifica e conta como nenhuma: %v", id, err)
			continue
		}
		out[id] = a
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status effect listings: %w", err)
	}
	return out, nil
}
