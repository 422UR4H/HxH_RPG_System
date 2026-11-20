package game

import (
	"log"
	"slices"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/cancelledaction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// turnWrites is what happened INSIDE one open turn and becomes durable only together with that
// turn's close, in PersistTurnClose's own transaction — or not at all (owner decision,
// 2026-10-01). The turn itself is only ever written when it closes, so anything written about
// it before that moment could outlive it: a restart mid-turn used to leave master_actions rows
// pointing at a turn that never existed, and a board row already holding the opened move and
// the master's changes. Holding them here instead makes the whole turn roll back together.
//
// Each entry is built the instant it happened — views, timestamps, ids are those of that
// instant — and only its WRITE waits. persistClosedTurn drains it (takeTurnWritesLocked) into
// TurnCloseData; a new kind of in-turn data adds a field here and its drain there.
type turnWrites struct {
	// masterActions are the turn's master actions, in the order they were applied.
	masterActions []masteraction.Record
	// cancelledActions are the actions cancelled while the turn was open, in the order they were
	// cancelled (holdCancelledActionLocked; spec 2026-10-10 §8.1 P7).
	cancelledActions []cancelledaction.Record
	// moveViews is what each session player saw of the turn's opened move, recorded at the
	// opening (recordOpenedMoveViews) and written with the action (actions.move_views): the
	// history shows each reader the move as they saw it then. nil when the action has no move.
	moveViews map[uuid.UUID]masteraction.View
	// reactionMoveViews is, per opened reaction that moves (an escape), what each session
	// player saw of its destination when the master opened it (recordOpenedReactionMoveViews) —
	// written with the reaction's own row (actions.move_views): the history shows each reader
	// a reaction's move as they saw it then (Phase 7, item 1). Absent for a reaction never
	// opened: it was never shown.
	reactionMoveViews map[uuid.UUID]map[uuid.UUID]masteraction.View
	// statusEffects são as mudanças de efeitos de status feitas com o turno aberto, na ordem em
	// que aconteceram (holdStatusEffectWritesLocked): gravadas pelo PersistTurnClose
	// (TurnCloseData.StatusEffects), atrás das que uma gravação anterior não gravou.
	statusEffects []statuseffect.Write
	// actionHiddenFrom é, para o ator e cada alvo da action Escondido na abertura do turno
	// (recordOpenedMoveViews), o efeito e quem o percebia (hiddenRecordLocked) — gravado com a
	// action (actions.hidden_from). nil = nenhum.
	actionHiddenFrom appmatch.HiddenFrom
	// reactionHiddenFrom é, por reação aberta, o registro do reator Escondido na abertura
	// dela (recordOpenedReactionMoveViews). A entrada existe (mesmo nil) para toda reação aberta:
	// é o que diz ao fechamento que ela já foi registrada; a que nunca abriu é registrada lá.
	reactionHiddenFrom map[uuid.UUID]appmatch.HiddenFrom
	// statusCancels é o registro do desfazer (D3): por efeito aplicado ou editado neste turno, as
	// ações da fila que a mudança dele tirou (dropRefusedByStatusLocked), em ordem de remoção. É
	// o que um remove_status_effect com restoreCancelledActions devolve (giveBackLocked). Morre
	// com o turno — fechamento, reinício, end_match —, então nenhum ponteiro velho sobrevive a
	// ele; nunca é gravado.
	statusCancels map[uuid.UUID]statusCancel
}

// statusCancel são as ações da fila que um efeito tirou no turno aberto, do personagem dele. Os
// ponteiros são os que saíram da fila: depois do DropRefusedByStatus só este registro os segura
// (o registro de cancelamento guarda uma cópia), então voltar com eles é seguro.
type statusCancel struct {
	charID  uuid.UUID
	actions []*action.Action
}

// noteStatusCancel registra que effectID (de charID) tirou acts da fila no turno w.
func (w *turnWrites) noteStatusCancel(effectID, charID uuid.UUID, acts []*action.Action) {
	if w.statusCancels == nil {
		w.statusCancels = map[uuid.UUID]statusCancel{}
	}
	rec := w.statusCancels[effectID]
	rec.charID = charID
	rec.actions = append(rec.actions, acts...)
	w.statusCancels[effectID] = rec
}

// forgetRestored tira do turno o que o cancelamento de cada ação devolvida deixou: o registro
// em cancelledActions (nada dela é gravado no fechamento) e a entrada dela no registro do
// desfazer de qualquer efeito (ela não pode ser devolvida duas vezes).
func (w *turnWrites) forgetRestored(ids map[uuid.UUID]bool) {
	if len(ids) == 0 {
		return
	}
	kept := w.cancelledActions[:0:0]
	for _, rec := range w.cancelledActions {
		if !ids[rec.UUID] {
			kept = append(kept, rec)
		}
	}
	w.cancelledActions = kept
	for effectID, rec := range w.statusCancels {
		acts := rec.actions[:0:0]
		for _, a := range rec.actions {
			if !ids[a.GetID()] {
				acts = append(acts, a)
			}
		}
		if len(acts) == 0 {
			delete(w.statusCancels, effectID)
			continue
		}
		rec.actions = acts
		w.statusCancels[effectID] = rec
	}
}

// statusCancelChars são os personagens com registro do desfazer — o cancelledActionIds que o
// mestre vê deles some quando o turno fecha.
func (w *turnWrites) statusCancelChars() []uuid.UUID {
	var out []uuid.UUID
	for _, rec := range w.statusCancels {
		if !slices.Contains(out, rec.charID) {
			out = append(out, rec.charID)
		}
	}
	return out
}

// openTurnIDLocked is the session's open turn, uuid.Nil when there is none (or no session).
// "Open" is HasOpenTurn, not CurrentTurnID alone: the round keeps its last turn as current
// after it closed. The caller must hold r.mu (read or write).
func (r *Room) openTurnIDLocked() uuid.UUID {
	if r.session == nil {
		return uuid.Nil
	}
	rd := r.session.GetActiveRound()
	if rd == nil || !rd.HasOpenTurn() {
		return uuid.Nil
	}
	return r.session.CurrentTurnID()
}

// turnWritesLocked returns the pending writes of turnID, creating them. The caller must hold
// r.mu for writing.
func (r *Room) turnWritesLocked(turnID uuid.UUID) *turnWrites {
	if r.pendingTurns == nil {
		r.pendingTurns = map[uuid.UUID]*turnWrites{}
	}
	w, ok := r.pendingTurns[turnID]
	if !ok {
		w = &turnWrites{}
		r.pendingTurns[turnID] = w
	}
	return w
}

// takeTurnWritesLocked hands over and forgets what is pending for turnID — the turn is closing,
// and it is written with it now or lost with it. Keyed by turn, not a single slot: open_next_action
// closes one turn and opens the next inside the same Execute, so the turn being drained is not
// necessarily the one open by then. The caller must hold r.mu for writing.
func (r *Room) takeTurnWritesLocked(turnID uuid.UUID) turnWrites {
	w, ok := r.pendingTurns[turnID]
	if !ok {
		return turnWrites{}
	}
	delete(r.pendingTurns, turnID)
	return *w
}

// droppedTurn is what a room that goes away with a turn still open loses: the turn (its opened
// move and anything else applied live inside it) and the master actions and cancellations held
// for it.
type droppedTurn struct {
	turnID           uuid.UUID
	masterActions    int
	cancelledActions int
	statusEffects    int
	// unwrittenStatusEffects são as mudanças de status de fora de turno (ou de um fechamento que
	// falhou) que ainda esperavam uma gravação: morrem com a sala.
	unwrittenStatusEffects int
}

// describeDroppedTurnLocked reads what is lost when the room itself goes away — its last client
// left, or the hub stopped it — with a turn open. That is a restart as far as persistence goes:
// the next room rehydrates from the database, where the turn was never written, so nothing of it
// is written now either (the board is not saved, the held master actions are not inserted) and
// the turn rolls back whole to the last close, exactly like a restart mid-turn. Writing the board
// here would hand the next room an opened move whose turn does not exist. The result is only for
// logDroppedTurn, after the unlock. The caller must hold r.mu.
func (r *Room) describeDroppedTurnLocked() droppedTurn {
	d := droppedTurn{turnID: r.openTurnIDLocked(), unwrittenStatusEffects: len(r.unwrittenStatusEffects)}
	for _, w := range r.pendingTurns {
		d.masterActions += len(w.masterActions)
		d.cancelledActions += len(w.cancelledActions)
		d.statusEffects += len(w.statusEffects)
	}
	return d
}

// logDroppedTurn says what a room that went away mid-turn did not write — never silently.
func logDroppedTurn(matchUUID uuid.UUID, d droppedTurn) {
	if d.unwrittenStatusEffects > 0 {
		log.Printf("sala da partida %s fechou com %d gravação(ões) de efeitos de status pendentes de uma gravação que falhou — NÃO foram gravadas",
			matchUUID, d.unwrittenStatusEffects)
	}
	if d.turnID == uuid.Nil && d.masterActions == 0 && d.cancelledActions == 0 && d.statusEffects == 0 {
		return
	}
	log.Printf("room of match %s closed with turn %s still open — the turn, its opened move, the %d master action(s) "+
		"applied, the %d action(s) cancelled and the %d gravação(ões) de efeitos de status feitas nele were NOT persisted; "+
		"the match resumes from the last closed turn",
		matchUUID, d.turnID, d.masterActions, d.cancelledActions, d.statusEffects)
}
