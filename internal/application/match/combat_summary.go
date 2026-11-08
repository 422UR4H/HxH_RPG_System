package match

import (
	"context"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

// SummaryTurn é um turno visto pelo resumo do combate: quem agiu e o tipo da ação.
// ActionKind é o mesmo discriminador que actions.type grava (action.Action.TypeName).
type SummaryTurn struct {
	TurnUUID   uuid.UUID
	ActorUUID  uuid.UUID
	ActionKind string
}

// SummaryRound é um round da cena para o resumo, na ordem da cena (created_at): o regime em
// que ficou, se contou, se fechou com a fila vazia, e o primeiro e o último turno (nil sem turno).
type SummaryRound struct {
	UUID    uuid.UUID
	Mode    enum.RoundMode
	Counted bool
	Drained bool
	First   *SummaryTurn
	Last    *SummaryTurn
}

// HiatusTurn é uma ponta de um hiato: o turno e a posição (1-based) do round dele na cena.
type HiatusTurn struct {
	RoundNumber int
	SummaryTurn
}

// Hiatus é um intervalo em que ninguém disputou ações (spec §3.6): o último turno do round
// que esvaziou a fila e o primeiro turno do round seguinte que tem turno.
type Hiatus struct {
	Before HiatusTurn
	After  HiatusTurn
}

// CombatSummary é o resumo do combate da cena ativa (spec §4).
type CombatSummary struct {
	// CountedRounds/CountedSeconds: rounds fechados contados da cena — já somados ao relógio.
	CountedRounds  int
	CountedSeconds int
	// ActiveRoundCounts: o round em andamento conta (o segundo dele entra no fechamento).
	ActiveRoundCounts bool
	// MinimumStoryAt é o menor horário em que o combate pode terminar agora.
	MinimumStoryAt time.Time
	Hiatuses       []Hiatus
}

// ICombatSummaryRepository lê os rounds fechados de uma cena para o resumo do combate.
type ICombatSummaryRepository interface {
	ListSceneRoundsForSummary(ctx context.Context, sceneUUID uuid.UUID) ([]SummaryRound, error)
}

// BuildCombatSummary monta o resumo a partir dos rounds fechados da cena (na ordem da cena), do
// round ativo (com os turnos em memória) e do mínimo do relógio. Sem I/O.
//
// Hiato = round disputado (Counted: abriu turno em Race, mesmo que o regime tenha mudado depois)
// fechado com drained cujo PRÓXIMO round da cena com turno existe; o round
// ativo entra nessa busca com os turnos que já tem. Um round drenado sem sucessor com turno não é
// hiato: o combate acabou ali. Rounds sem turno são pulados na busca do "próximo" (round vazio
// não fecha, mas a leitura não depende disso).
func BuildCombatSummary(rounds []SummaryRound, active SummaryRound, minimum time.Time) CombatSummary {
	out := CombatSummary{
		ActiveRoundCounts: active.Counted,
		MinimumStoryAt:    minimum,
		Hiatuses:          []Hiatus{},
	}
	all := append(append(make([]SummaryRound, 0, len(rounds)+1), rounds...), active)
	for i, r := range rounds {
		if r.Counted {
			out.CountedRounds++
		}
		if !r.Counted || !r.Drained || r.Last == nil {
			continue
		}
		for j := i + 1; j < len(all); j++ {
			if next := all[j]; next.First != nil {
				out.Hiatuses = append(out.Hiatuses, Hiatus{
					Before: HiatusTurn{RoundNumber: i + 1, SummaryTurn: *r.Last},
					After:  HiatusTurn{RoundNumber: j + 1, SummaryTurn: *next.First},
				})
				break
			}
		}
	}
	// 1 round = 1 s (spec §3.5)
	out.CountedSeconds = out.CountedRounds
	return out
}
