// Package statuswire é a forma de um efeito de status no fio, a mesma para a sala (WS) e para
// o histórico (REST): os dois leem o mesmo efeito, e duas cópias do mapeamento divergiriam.
package statuswire

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Params são os parâmetros de um efeito no fio. Na entrada, ponteiro nil = não informado; na
// saída, só os campos que o tipo possui, com os padrões já aplicados.
type Params struct {
	Domains      []string          `json:"domains,omitempty"`
	Injuries     map[string]string `json:"injuries,omitempty"`
	Level        string            `json:"level,omitempty"`
	Faces        []string          `json:"faces,omitempty"`
	Points       *int              `json:"points,omitempty"`
	HPPerRound   *int              `json:"hpPerRound,omitempty"`
	Penalty      *int              `json:"penalty,omitempty"`
	Disadvantage *bool             `json:"disadvantage,omitempty"`
	Conduct      string            `json:"conduct,omitempty"`
	Irresistible *bool             `json:"irresistible,omitempty"`
	ControllerID *uuid.UUID        `json:"controllerId,omitempty"`
}

// Visibility é a lista de quem vê um efeito. Mode: "default" (padrão do tipo), "public"
// (todos) ou "players" (PlayerIDs; vazio = só o mestre).
type Visibility struct {
	Mode      string      `json:"mode"`
	PlayerIDs []uuid.UUID `json:"playerIds,omitempty"`
}

// Effect é um efeito como o destinatário o vê. Visibility e Public só vão ao mestre; Public
// diz se todo jogador está na lista (a visão à parte).
type Effect struct {
	ID         uuid.UUID   `json:"id"`
	Kind       string      `json:"kind"`
	Sense      string      `json:"sense,omitempty"`
	Aptitude   string      `json:"aptitude,omitempty"`
	Params     Params      `json:"params"`
	Alias      string      `json:"alias,omitempty"`
	AppliedAt  time.Time   `json:"appliedAt"`
	Visibility *Visibility `json:"visibility,omitempty"`
	Public     *bool       `json:"public,omitempty"`
	// TurnActionID é só da sala e só do mestre: a âncora da última mudança deste efeito ainda
	// pendente no turno aberto (a action ou a reação a que ela pertence). Ausente sem turno
	// aberto, sem mudança nele ou sem âncora; o histórico nunca a põe.
	TurnActionID *uuid.UUID `json:"turnActionId,omitempty"`
	// CancelledActionIDs é só da sala e só do mestre, com o mesmo corte do TurnActionID: as
	// ações da fila que ESTE efeito tirou no turno aberto e que um remove_status_effect com
	// restoreCancelledActions ainda pode devolver (D3). Ausente fora da janela do desfazer (efeito
	// não aplicado neste turno, sem turno aberto, depois do fechamento); o histórico nunca o põe.
	CancelledActionIDs []uuid.UUID `json:"cancelledActionIds,omitempty"`
}

// From é o efeito no fio, sem Visibility nem Public (quem é mestre as põe).
func From(e statuseffect.Effect) Effect {
	return Effect{
		ID:        e.ID,
		Kind:      string(e.Kind),
		Sense:     string(e.Sense),
		Aptitude:  string(e.Aptitude),
		Params:    paramsOf(e),
		Alias:     e.Alias,
		AppliedAt: e.AppliedAt,
	}
}

// VisibilityOf traduz a lista de quem vê: nil = "default" (padrão do tipo).
func VisibilityOf(a *statuseffect.Audience) Visibility {
	switch {
	case a == nil:
		return Visibility{Mode: "default"}
	case a.Everyone:
		return Visibility{Mode: "public"}
	}
	return Visibility{Mode: "players", PlayerIDs: append([]uuid.UUID(nil), a.Players...)}
}

// paramsOf põe no fio todo campo que o tipo possui, e só esses (tabela da Task 2). Um sentido
// anulado não leva desvantagem nem penalidade.
func paramsOf(e statuseffect.Effect) Params {
	p := e.Params
	intp := func(v int) *int { return &v }
	boolp := func(v bool) *bool { return &v }
	var out Params
	switch e.Kind {
	case statuseffect.KindParalyzed:
		out.Domains = domainsOf(p.Domains)
		out.Points = intp(p.Points)
	case statuseffect.KindStupefied, statuseffect.KindStunned:
		out.Points = intp(p.Points)
	case statuseffect.KindInjured:
		out.Injuries = make(map[string]string, len(p.Injuries))
		for d, l := range p.Injuries {
			out.Injuries[string(d)] = string(l)
		}
	case statuseffect.KindSlowed:
		out.Penalty = intp(p.Penalty)
	case statuseffect.KindSense:
		out.Level = string(p.Level)
		if p.Level == statuseffect.SenseImpaired {
			out.Disadvantage = boolp(p.Disadvantage)
			out.Penalty = intp(p.Penalty)
		}
		if e.Sense == statuseffect.SenseTact {
			for _, f := range p.Faces {
				out.Faces = append(out.Faces, string(f))
			}
		}
	case statuseffect.KindWeakened:
		out.Disadvantage = boolp(p.Disadvantage)
		out.Penalty = intp(p.Penalty)
	case statuseffect.KindBleeding, statuseffect.KindBurning, statuseffect.KindFreezing, statuseffect.KindPoisoned:
		out.HPPerRound = intp(p.HPPerRound)
	case statuseffect.KindCompelled:
		out.Conduct = p.Conduct
		out.Irresistible = boolp(p.Irresistible)
	case statuseffect.KindControlled:
		if p.ControllerID != nil {
			id := *p.ControllerID
			out.ControllerID = &id
		}
	}
	return out
}

func domainsOf(ds []statuseffect.Domain) []string {
	out := make([]string, 0, len(ds))
	for _, d := range ds {
		out = append(out, string(d))
	}
	return out
}
