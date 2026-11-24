// internal/gateway/pg/statuseffect/repository.go
package pgstatuseffect

import (
	"context"
	"encoding/json"
	"fmt"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	pgsheet "github.com/422UR4H/HxH_RPG_System/internal/gateway/pg/sheet"
	pgfs "github.com/422UR4H/HxH_RPG_System/pkg"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Repository persiste os efeitos de status das fichas (tabela character_status_effects).
//
// q é o pool ou uma transação já aberta: Write abre a própria transação sobre q, que vira
// savepoint quando q é uma pgx.Tx — assim o mesmo código roda sozinho ou dentro de outra
// escrita atômica.
type Repository struct {
	q pgfs.IQuerier
}

func NewRepository(q pgfs.IQuerier) *Repository {
	return &Repository{q: q}
}

// paramsRow e audienceRow são o formato de armazenamento (JSON camelCase); ficam aqui para o
// domínio não precisar de tags. Campos zerados são omitidos.
type paramsRow struct {
	Domains      []string          `json:"domains,omitempty"`
	Injuries     map[string]string `json:"injuries,omitempty"`
	Level        string            `json:"level,omitempty"`
	Faces        []string          `json:"faces,omitempty"`
	Points       int               `json:"points,omitempty"`
	HPPerRound   int               `json:"hpPerRound,omitempty"`
	Penalty      int               `json:"penalty,omitempty"`
	Disadvantage bool              `json:"disadvantage,omitempty"`
	Conduct      string            `json:"conduct,omitempty"`
	Irresistible bool              `json:"irresistible,omitempty"`
	ControllerID *uuid.UUID        `json:"controllerId,omitempty"`
}

// audienceRow: Players é ponteiro para que a lista vazia (só o mestre) seja gravada como [].
type audienceRow struct {
	Everyone bool         `json:"everyone,omitempty"`
	Players  *[]uuid.UUID `json:"players,omitempty"`
}

func encodeParams(p statuseffect.Params) ([]byte, error) {
	row := paramsRow{
		Level: string(p.Level), Points: p.Points, HPPerRound: p.HPPerRound, Penalty: p.Penalty,
		Disadvantage: p.Disadvantage, Conduct: p.Conduct, Irresistible: p.Irresistible,
		ControllerID: p.ControllerID,
	}
	for _, d := range p.Domains {
		row.Domains = append(row.Domains, string(d))
	}
	for _, f := range p.Faces {
		row.Faces = append(row.Faces, string(f))
	}
	if len(p.Injuries) > 0 {
		row.Injuries = make(map[string]string, len(p.Injuries))
		for d, l := range p.Injuries {
			row.Injuries[string(d)] = string(l)
		}
	}
	return json.Marshal(row)
}

func decodeParams(raw []byte) (statuseffect.Params, error) {
	var row paramsRow
	if err := json.Unmarshal(raw, &row); err != nil {
		return statuseffect.Params{}, err
	}
	p := statuseffect.Params{
		Level: statuseffect.SenseLevel(row.Level), Points: row.Points, HPPerRound: row.HPPerRound,
		Penalty: row.Penalty, Disadvantage: row.Disadvantage, Conduct: row.Conduct,
		Irresistible: row.Irresistible, ControllerID: row.ControllerID,
	}
	for _, d := range row.Domains {
		p.Domains = append(p.Domains, statuseffect.Domain(d))
	}
	for _, f := range row.Faces {
		p.Faces = append(p.Faces, statuseffect.TouchFace(f))
	}
	if len(row.Injuries) > 0 {
		p.Injuries = make(map[statuseffect.Domain]statuseffect.InjuryLevel, len(row.Injuries))
		for d, l := range row.Injuries {
			p.Injuries[statuseffect.Domain(d)] = statuseffect.InjuryLevel(l)
		}
	}
	return p, nil
}

// encodeAudience devolve nil (SQL NULL) para o padrão do tipo.
func encodeAudience(a *statuseffect.Audience) ([]byte, error) {
	if a == nil {
		return nil, nil
	}
	row := audienceRow{Everyone: a.Everyone}
	if !a.Everyone {
		players := a.Players
		if players == nil {
			players = []uuid.UUID{}
		}
		row.Players = &players
	}
	return json.Marshal(row)
}

func decodeAudience(raw []byte) (*statuseffect.Audience, error) {
	if raw == nil {
		return nil, nil
	}
	var row audienceRow
	if err := json.Unmarshal(raw, &row); err != nil {
		return nil, err
	}
	a := &statuseffect.Audience{Everyone: row.Everyone}
	if !row.Everyone {
		a.Players = []uuid.UUID{}
		if row.Players != nil {
			a.Players = *row.Players
		}
	}
	return a, nil
}

// Write aplica as operações na ordem, numa transação só: ou todas ou nenhuma.
// Inclusão/edição é upsert por uuid; ficha, tipo, variante e applied_at nunca mudam depois
// de gravados. Remoção usa só o ID e remover o que não existe não é erro. A última gravação
// ganha: uma linha de outro uuid com a mesma identidade é substituída, e uma ficha que já não
// existe é pulada sem erro. Uma operação com Event grava também o evento do histórico
// (status_effect_events) logo depois dela, na mesma transação (insertEvent).
//
// bars são as barras das fichas que a mesma mudança moveu (o HP da volta do Morto, ou o que
// um fechamento falho não gravou), gravadas depois dos efeitos na mesma transação — um comando
// do mestre, uma transação. Uma barra que falha desfaz também os efeitos.
func (r *Repository) Write(ctx context.Context, ops []statuseffect.Write, bars []appmatch.SheetStatusBars) error {
	if len(ops) == 0 && len(bars) == 0 {
		return nil
	}
	tx, err := r.q.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin status effect write: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
		_ = tx.Rollback(ctx) // no-op depois do Commit
	}()

	for _, op := range ops {
		if err := writeOp(ctx, tx, op); err != nil {
			return err
		}
		if op.Event != nil {
			if err := insertEvent(ctx, tx, op); err != nil {
				return err
			}
		}
	}
	// O UpdateStatusBars do gateway de fichas, nesta transação: o SQL mora num lugar só.
	if len(bars) > 0 {
		sheets := pgsheet.NewRepository(tx)
		for _, sb := range bars {
			if err := sheets.UpdateStatusBars(ctx, sb.CharacterID.String(), sb.Health, sb.Stamina, sb.Aura); err != nil {
				return fmt.Errorf("status bars of %s with the status write: %w", sb.CharacterID, err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit status effect write: %w", err)
	}
	return nil
}

// writeOp grava (upsert) ou apaga um efeito, dentro da transação de Write.
func writeOp(ctx context.Context, tx pgx.Tx, op statuseffect.Write) error {
	if op.Delete {
		if _, err := tx.Exec(ctx,
			`DELETE FROM character_status_effects WHERE uuid = $1`, op.Effect.ID); err != nil {
			return fmt.Errorf("delete status effect %s: %w", op.Effect.ID, err)
		}
		return nil
	}
	e := op.Effect
	params, err := encodeParams(e.Params)
	if err != nil {
		return fmt.Errorf("marshal status effect params: %w", err)
	}
	audience, err := encodeAudience(e.Audience)
	if err != nil {
		return fmt.Errorf("marshal status effect audience: %w", err)
	}
	// Última gravação ganha: outra linha com a mesma identidade (ficha viva em duas salas)
	// sai antes, senão o índice único recusaria para sempre.
	if _, err := tx.Exec(ctx, `
		DELETE FROM character_status_effects
		WHERE character_sheet_uuid = $1 AND kind = $2 AND variant = $3 AND uuid <> $4`,
		e.CharacterID, string(e.Kind), e.Identity().Variant, e.ID); err != nil {
		return fmt.Errorf("replace status effect identity %s: %w", e.ID, err)
	}
	// A ficha apagada com a sessão ainda a segurando: o SELECT ... WHERE EXISTS faz o
	// insert não gravar nada, em vez de a FK recusar a transação inteira.
	const q = `
		INSERT INTO character_status_effects
			(uuid, character_sheet_uuid, kind, variant, params, alias, audience, applied_at, updated_at)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9
		WHERE EXISTS (SELECT 1 FROM character_sheets WHERE uuid = $2)
		ON CONFLICT (uuid) DO UPDATE SET
			params = EXCLUDED.params, alias = EXCLUDED.alias,
			audience = EXCLUDED.audience, updated_at = EXCLUDED.updated_at
	`
	if _, err := tx.Exec(ctx, q,
		e.ID, e.CharacterID, string(e.Kind), e.Identity().Variant, params, e.Alias, audience,
		e.AppliedAt, e.UpdatedAt,
	); err != nil {
		return fmt.Errorf("upsert status effect %s: %w", e.ID, err)
	}
	return nil
}

// ListBySheets agrupa os efeitos por ficha, em ordem applied_at, uuid. Nunca devolve nil;
// entrada vazia não consulta o banco.
func (r *Repository) ListBySheets(
	ctx context.Context, sheetUUIDs []uuid.UUID,
) (map[uuid.UUID][]statuseffect.Effect, error) {
	out := map[uuid.UUID][]statuseffect.Effect{}
	if len(sheetUUIDs) == 0 {
		return out, nil
	}
	const q = `
		SELECT uuid, character_sheet_uuid, kind, variant, params, alias, audience, applied_at, updated_at
		FROM character_status_effects
		WHERE character_sheet_uuid = ANY($1)
		ORDER BY applied_at, uuid
	`
	rows, err := r.q.Query(ctx, q, sheetUUIDs)
	if err != nil {
		return nil, fmt.Errorf("list status effects: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var e statuseffect.Effect
		var kind, variant string
		var params, audience []byte
		if err := rows.Scan(&e.ID, &e.CharacterID, &kind, &variant, &params, &e.Alias, &audience,
			&e.AppliedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan status effect: %w", err)
		}
		e.Kind = statuseffect.Kind(kind)
		switch e.Kind {
		case statuseffect.KindSense:
			e.Sense = statuseffect.Sense(variant)
		case statuseffect.KindWeakened:
			e.Aptitude = statuseffect.Aptitude(variant)
		}
		if e.Params, err = decodeParams(params); err != nil {
			return nil, fmt.Errorf("decode params of status effect %s: %w", e.ID, err)
		}
		if e.Audience, err = decodeAudience(audience); err != nil {
			return nil, fmt.Errorf("decode audience of status effect %s: %w", e.ID, err)
		}
		e.AppliedAt, e.UpdatedAt = e.AppliedAt.UTC(), e.UpdatedAt.UTC()
		out[e.CharacterID] = append(out[e.CharacterID], e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate status effects: %w", err)
	}
	return out, nil
}
