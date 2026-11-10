package round

import (
	"context"
	"fmt"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

// ListSceneRoundsForSummary devolve os rounds FECHADOS da cena, na ordem dela (created_at,
// uuid — o mesmo desempate do histórico), cada um com o primeiro e o último turno: o ator e o
// tipo da ação que o conduziu (actions.type, a linha da ação, nunca a de uma reação). O round
// aberto — o ativo — fica de fora: o resumo o recebe da memória, com os turnos que já tem.
//
// Os turnos são ordenados como no histórico (finished_at, uuid). Um turno sem linha de ação
// (só por deriva) não entra: sem a ação não há o que mostrar dele.
func (r *Repository) ListSceneRoundsForSummary(ctx context.Context, sceneUUID uuid.UUID) ([]appmatch.SummaryRound, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT ro.uuid, ro.mode, ro.counted, ro.drained,
		        f.uuid, f.actor_uuid, f.type,
		        l.uuid, l.actor_uuid, l.type
		 FROM rounds ro
		 LEFT JOIN LATERAL (
		   SELECT t.uuid, a.actor_uuid, a.type
		   FROM turns t JOIN actions a ON a.turn_uuid = t.uuid AND a.react_to_uuid IS NULL
		   WHERE t.round_uuid = ro.uuid
		   ORDER BY t.finished_at, t.uuid LIMIT 1
		 ) f ON true
		 LEFT JOIN LATERAL (
		   SELECT t.uuid, a.actor_uuid, a.type
		   FROM turns t JOIN actions a ON a.turn_uuid = t.uuid AND a.react_to_uuid IS NULL
		   WHERE t.round_uuid = ro.uuid
		   ORDER BY t.finished_at DESC, t.uuid DESC LIMIT 1
		 ) l ON true
		 WHERE ro.scene_uuid = $1 AND ro.finished_at IS NOT NULL
		 ORDER BY ro.created_at, ro.uuid`,
		sceneUUID,
	)
	if err != nil {
		return nil, fmt.Errorf("ListSceneRoundsForSummary: %w", err)
	}
	defer rows.Close()

	out := make([]appmatch.SummaryRound, 0)
	for rows.Next() {
		var (
			sr                  appmatch.SummaryRound
			mode                string
			firstID, firstActor *uuid.UUID
			lastID, lastActor   *uuid.UUID
			firstKind, lastKind *string
		)
		if err := rows.Scan(&sr.UUID, &mode, &sr.Counted, &sr.Drained,
			&firstID, &firstActor, &firstKind, &lastID, &lastActor, &lastKind); err != nil {
			return nil, fmt.Errorf("ListSceneRoundsForSummary scan: %w", err)
		}
		sr.Mode = enum.RoundMode(mode)
		sr.First = summaryTurnOf(firstID, firstActor, firstKind)
		sr.Last = summaryTurnOf(lastID, lastActor, lastKind)
		out = append(out, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListSceneRoundsForSummary rows: %w", err)
	}
	return out, nil
}

func summaryTurnOf(id, actor *uuid.UUID, kind *string) *appmatch.SummaryTurn {
	if id == nil {
		return nil
	}
	st := &appmatch.SummaryTurn{TurnUUID: *id, ActorUUID: derefUUID(actor)}
	if kind != nil {
		st.ActionKind = *kind
	}
	return st
}
