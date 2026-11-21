package statuseffect

import (
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/google/uuid"
)

// Change é o que aconteceu com o efeito num evento do histórico.
type Change string

const (
	ChangeApplied Change = "applied"
	ChangeEdited  Change = "edited"
	ChangeRemoved Change = "removed"
)

// Event é o registro de uma mudança de efeito no histórico, carimbado pela sala no instante
// da mudança e gravado junto da gravação do efeito (Write.Event).
type Event struct {
	ID                        uuid.UUID
	Change                    Change
	MatchID, SceneID, RoundID uuid.UUID
	TurnID                    *uuid.UUID
	// ActionID é a action (ou reação) do turno aberto a que esta mudança pertence; nil = do
	// turno (ou do round) como um todo. Na fase 1, só o mestre a informa.
	ActionID *uuid.UUID
	// Listing é quem a lista do efeito (ou o padrão do tipo) incluía no instante, já
	// resolvida — nunca "padrão".
	Listing Audience
	// Views são os jogadores que enxergavam o personagem logo antes ou logo depois da mudança,
	// mais o dono; o mestre nunca entra. Nunca nil depois de carimbado.
	Views map[uuid.UUID]masteraction.View
	// HiddenFrom são os jogadores (nunca o mestre nem o dono) para quem o personagem estava
	// Escondido no instante, com o id do efeito Escondido; quem está em Views não entra.
	HiddenFrom map[uuid.UUID]uuid.UUID
	HappenedAt time.Time
}

// EventRecord é um evento lido do histórico: Effect é o efeito como ficou (removido: como
// era), com o CharacterID dentro.
type EventRecord struct {
	Event  Event
	Effect Effect
}

// Lists diz se esta lista (já resolvida) inclui userID.
func (a Audience) Lists(userID uuid.UUID) bool {
	if a.Everyone {
		return true
	}
	for _, p := range a.Players {
		if p == userID {
			return true
		}
	}
	return false
}

// ResolveListing é a lista de quem vê e, com o padrão do tipo já resolvido (a mesma regra de
// Listed, sem o mestre): o que o histórico guarda para o instante. Players nunca é nil.
func ResolveListing(e Effect, s Subject, pcPublicByDefault bool) Audience {
	if e.Audience != nil {
		if e.Audience.Everyone {
			return Audience{Everyone: true}
		}
		return Audience{Players: append([]uuid.UUID{}, e.Audience.Players...)}
	}
	switch {
	case e.Kind.PublicByDefault(), !s.IsNPC && pcPublicByDefault:
		return Audience{Everyone: true}
	case s.IsNPC:
		return Audience{Players: []uuid.UUID{}}
	}
	return Audience{Players: []uuid.UUID{s.OwnerID}}
}

// RevealLookup diz até quando um efeito foi revelado no histórico (a última revelação); ok
// false = nunca. Um nil vale por "nunca revelado".
type RevealLookup func(effectID uuid.UUID) (revealedAt time.Time, ok bool)

// Covers diz se uma revelação do efeito cobre o instante at (revelado em at ou depois). Um
// lookup nil não cobre nada.
func (l RevealLookup) Covers(effectID uuid.UUID, at time.Time) bool {
	if l == nil {
		return false
	}
	t, ok := l(effectID)
	return ok && !t.Before(at)
}

// HistoryReading é o que a regra de visibilidade de um evento precisa saber de quem lê e do
// resto do histórico.
type HistoryReading struct {
	Reader   uuid.UUID
	IsMaster bool
	// Latest é a última lista gravada do efeito do evento (em qualquer partida); HasLatest
	// false = nenhuma.
	Latest    Audience
	HasLatest bool
	// RevealedAt vem das revelações do mestre; nil = nada revelado.
	RevealedAt RevealLookup
	// AnchorVisible diz se a action/reação a que o evento pertence é visível ao leitor (sem
	// âncora: true). Uma mudança presa a uma action escondida se esconde com ela.
	AnchorVisible bool
}

func (h HistoryReading) revealedBy(effectID uuid.UUID, at time.Time) bool {
	return h.RevealedAt.Covers(effectID, at)
}

// VisibleInHistory é a regra única de visibilidade de um evento de status no histórico:
//
//	mestre
//	|| ( âncora visível
//	     && ( o efeito foi revelado depois do evento
//	          || ( (leitor em Views || (leitor em HiddenFrom e o Escondido revelado depois do evento))
//	               && a última lista do efeito inclui o leitor ) ) )
func VisibleInHistory(rec EventRecord, h HistoryReading) bool {
	if h.IsMaster {
		return true
	}
	if !h.AnchorVisible {
		return false
	}
	at := rec.Event.HappenedAt
	if h.revealedBy(rec.Effect.ID, at) {
		return true
	}
	_, inViews := rec.Event.Views[h.Reader]
	if !inViews {
		hid, hidden := rec.Event.HiddenFrom[h.Reader]
		if !hidden || !h.revealedBy(hid, at) {
			return false
		}
	}
	return h.HasLatest && h.Latest.Lists(h.Reader)
}
