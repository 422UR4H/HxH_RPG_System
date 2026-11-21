package game

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/422UR4H/HxH_RPG_System/internal/app/wire/actionwire"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// Âncoras das mudanças de status do turno aberto (Task 15b, dono do produto 2026-11-19). A
// sala guarda as mudanças do turno (turnWrites.statusEffects) até o fechamento; até lá o mestre
// pode trocar a âncora (relink_status_effect), a âncora segue a reação que substitui a sentada,
// e remover um efeito aplicado no próprio turno desfaz a aplicação. Depois do fechamento o
// histórico é definitivo.

// errStatusNoOpenTurn recusa um relink sem turno aberto.
var errStatusNoOpenTurn = fmt.Errorf("%w: relink_status_effect só vale com um turno aberto",
	statuseffect.ErrInvalidParams)

// errStatusNoChangeInTurn recusa um relink de um efeito que não mudou no turno aberto.
var errStatusNoChangeInTurn = fmt.Errorf("%w: o efeito não tem mudança neste turno",
	statuseffect.ErrNotFound)

// followAnchorsLocked aplica as saídas de reação que a sessão registrou (TakeAnchorMoves) às
// mudanças de status pendentes do turno de cada uma: a âncora que nomeava a reação que saiu
// passa a nomear o que ficou no lugar. Devolve os personagens cujas mudanças foram reancoradas
// (o mestre recebe o turnActionId novo). O chamador segura r.mu (escrita).
func (r *Room) followAnchorsLocked() []uuid.UUID {
	if r.session == nil {
		return nil
	}
	var chars []uuid.UUID
	seen := map[uuid.UUID]bool{}
	for _, mv := range r.session.TakeAnchorMoves() {
		w, ok := r.pendingTurns[mv.TurnID]
		if !ok {
			continue
		}
		for i := range w.statusEffects {
			ev := w.statusEffects[i].Event
			if ev == nil || ev.ActionID == nil || *ev.ActionID != mv.From {
				continue
			}
			to := mv.To
			ev.ActionID = &to
			if c := w.statusEffects[i].Effect.CharacterID; !seen[c] {
				seen[c] = true
				chars = append(chars, c)
			}
		}
	}
	return chars
}

// pendingAnchorLocked é a âncora da última mudança de effectID ainda pendente no turno aberto;
// nil sem turno aberto, sem mudança nele ou sem âncora. O chamador segura r.mu (leitura basta).
func (r *Room) pendingAnchorLocked(effectID uuid.UUID) *uuid.UUID {
	turnID := r.openTurnIDLocked()
	if turnID == uuid.Nil {
		return nil
	}
	w, ok := r.pendingTurns[turnID]
	if !ok {
		return nil
	}
	for i := len(w.statusEffects) - 1; i >= 0; i-- {
		op := w.statusEffects[i]
		if op.Effect.ID != effectID || op.Event == nil {
			continue
		}
		return cloneUUIDPtr(op.Event.ActionID)
	}
	return nil
}

// anchoredChars são os personagens com mudança ancorada entre ops — o turnActionId que o
// mestre vê deles some quando o turno fecha.
func anchoredChars(ops []statuseffect.Write) []uuid.UUID {
	var out []uuid.UUID
	seen := map[uuid.UUID]bool{}
	for _, op := range ops {
		if op.Event == nil || op.Event.ActionID == nil || seen[op.Effect.CharacterID] {
			continue
		}
		seen[op.Effect.CharacterID] = true
		out = append(out, op.Effect.CharacterID)
	}
	return out
}

// undoInTurnAppliesLocked desfaz, no turno aberto, a aplicação de todo efeito removido por
// writes que foi APLICADO neste mesmo turno (a gravação "applied" dele ainda está pendente):
// tira do turno toda mudança pendente desse efeito e devolve writes sem a remoção dele — nada
// desse efeito é gravado, nem evento "removed" (dono do produto, 2026-11-19) — e os efeitos
// desfeitos. Qualquer outra remoção segue normal. As ações da fila que o status cancelou só
// voltam se o mestre pedir (restoreCancelledActions, giveBackLocked); sem o pedido, continuam
// canceladas. O chamador segura r.mu (escrita).
func (r *Room) undoInTurnAppliesLocked(writes []statuseffect.Write) (kept []statuseffect.Write, undone map[uuid.UUID]bool) {
	turnID := r.openTurnIDLocked()
	if turnID == uuid.Nil {
		return writes, nil
	}
	w, ok := r.pendingTurns[turnID]
	if !ok {
		return writes, nil
	}
	kept = writes[:0:0]
	for _, op := range writes {
		if !op.Delete || !appliedIn(w.statusEffects, op.Effect.ID) {
			kept = append(kept, op)
			continue
		}
		if undone == nil {
			undone = map[uuid.UUID]bool{}
		}
		undone[op.Effect.ID] = true
		rest := w.statusEffects[:0:0]
		for _, p := range w.statusEffects {
			if p.Effect.ID != op.Effect.ID {
				rest = append(rest, p)
			}
		}
		w.statusEffects = rest
		purgeEffectFromTurnLocked(w, op.Effect.ID)
	}
	return kept, undone
}

// restoredNotice é uma ação que voltou à fila, como cada um a recebe: o action_restored do
// mestre e o action_queued dele (a entrada na fila, como no enqueue), e o action_restored do
// dono quando o dono não é o mestre (ownerMsg nil). Tudo montado sob r.mu.
type restoredNotice struct {
	master   ActionRestoredPayload
	queued   ActionQueuedPayload
	owner    uuid.UUID
	ownerMsg *ActionRestoredPayload
}

// giveBackLocked é o lado da sala do inverso de aplicar um status no turno aberto (D3). Para
// cada efeito removido: o registro do desfazer dele (turnWrites.statusCancels) é apagado, com
// pedido ou sem; e só quando o efeito foi DESFEITO agora (undone — aplicado neste mesmo turno)
// e o mestre pediu (restore), as ações que ele tirou da fila voltam por
// RestoreCancelledByStatus, que as revalida contra a sessão de agora — o status já saiu dela —
// e as põe no mesmo lugar da fila, com as edições do mestre. A devolvida apaga o próprio
// cancelamento (forgetRestored): nada dela é gravado no fechamento. O status não marca troca
// (sem swapMarks no DropRefusedByStatus), então não há marca a tirar; o fechamento da janela de
// fusão do relógio que o cancelamento fez (clearClockMergeLocked) fica — é inofensivo.
//
// Uma recusa porque outro status ainda a recusa passa o registro dela a cada efeito desse tipo
// aplicado neste mesmo turno (ainda desfazível): desfazer esse depois a devolve, em qualquer
// ordem. Fora da janela do desfazer não há o que devolver e o pedido é ignorado.
//
// Devolve os anúncios das devolvidas e as recusas (só ao mestre). O chamador segura r.mu
// (escrita), na MESMA seção crítica da remoção e do desfazer, e r.session não é nil.
func (r *Room) giveBackLocked(removed []uuid.UUID, undone map[uuid.UUID]bool, restore bool) (
	notices []restoredNotice, refusals []matchsession.RestoreRefusal,
) {
	turnID := r.openTurnIDLocked()
	if turnID == uuid.Nil {
		return nil, nil
	}
	w, ok := r.pendingTurns[turnID]
	if !ok {
		return nil, nil
	}
	var restored []*action.Action
	for _, effectID := range removed {
		rec, ok := w.statusCancels[effectID]
		delete(w.statusCancels, effectID)
		if !ok || !restore || !undone[effectID] {
			continue
		}
		back, refused := r.session.RestoreCancelledByStatus(rec.drops)
		ids := make(map[uuid.UUID]bool, len(back))
		for _, a := range back {
			ids[a.GetID()] = true
		}
		w.forgetRestored(ids)
		restored = append(restored, back...)
		for _, f := range refused {
			r.passToBlockerLocked(w, rec.charID, f)
		}
		refusals = append(refusals, refused...)
	}
	return r.restoredNoticesLocked(restored), refusals
}

// passToBlockerLocked guarda a ação recusada no registro de cada efeito do personagem, aplicado
// neste turno, cujo tipo ainda a recusa (f.BlockedBy). O chamador segura r.mu (escrita).
func (r *Room) passToBlockerLocked(w *turnWrites, charID uuid.UUID, f matchsession.RestoreRefusal) {
	if len(f.BlockedBy) == 0 {
		return
	}
	for _, e := range r.session.StatusEffectsOf(charID) {
		if slices.Contains(f.BlockedBy, e.Kind) && appliedIn(w.statusEffects, e.ID) {
			w.noteStatusCancel(e.ID, charID, []matchsession.StatusDrop{f.Drop})
		}
	}
}

// restoredNoticesLocked monta, com a fila já refeita, o que cada ação devolvida anuncia, na
// ordem da fila. O lugar de cada uma é a próxima ação da fila que o destinatário já conhece
// quando a mensagem chega (o mestre, todas; o dono, as dele) — sem contar as outras devolvidas,
// que ainda não foram anunciadas: inseridas uma a uma, nessa ordem, caem todas no lugar. O
// chamador segura r.mu.
func (r *Room) restoredNoticesLocked(restored []*action.Action) []restoredNotice {
	if len(restored) == 0 {
		return nil
	}
	owners := r.session.GetCharToPlayer()
	pending := r.session.PendingActions()
	back := make(map[*action.Action]bool, len(restored))
	for _, a := range restored {
		back[a] = true
	}
	ownerOf := func(a *action.Action) uuid.UUID { return owners[a.GetActorID().String()] }
	nextFor := func(i int, known func(*action.Action) bool) *uuid.UUID {
		for _, q := range pending[i+1:] {
			if !back[q] && known(q) {
				id := q.GetID()
				return &id
			}
		}
		return nil
	}
	out := make([]restoredNotice, 0, len(restored))
	for i, a := range pending {
		if !back[a] {
			continue
		}
		owner := ownerOf(a)
		n := restoredNotice{
			master: ActionRestoredPayload{
				ActionID: a.GetID(), Action: actionwire.From(*a, actionwire.Full),
				BeforeActionID: nextFor(i, func(*action.Action) bool { return true }),
			},
			queued: r.newActionQueuedPayloadLocked(r.session, a),
			owner:  owner,
		}
		if owner != uuid.Nil && owner != r.masterUUID {
			n.ownerMsg = &ActionRestoredPayload{
				ActionID: a.GetID(), Action: actionwire.From(*a, actionwire.Declaration),
				BeforeActionID: nextFor(i, func(q *action.Action) bool { return ownerOf(q) == owner }),
			}
		}
		out = append(out, n)
	}
	return out
}

// cancelledActionIDsLocked é o cancelledActionIds que o mestre vê em effectID: o que ele tirou
// da fila no turno aberto, enquanto um desfazer ainda pode devolver — só para um efeito aplicado
// neste turno. nil fora disso. O chamador segura r.mu (leitura basta).
func (r *Room) cancelledActionIDsLocked(effectID uuid.UUID) []uuid.UUID {
	turnID := r.openTurnIDLocked()
	if turnID == uuid.Nil {
		return nil
	}
	w, ok := r.pendingTurns[turnID]
	if !ok || !appliedIn(w.statusEffects, effectID) {
		return nil
	}
	var out []uuid.UUID
	for _, d := range w.statusCancels[effectID].drops {
		out = append(out, d.Action.GetID())
	}
	return out
}

// uniqueIDs tira as repetições de ids, mantendo a primeira ocorrência.
func uniqueIDs(ids []uuid.UUID) []uuid.UUID {
	out := ids[:0:0]
	for _, id := range ids {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

// purgeEffectFromTurnLocked tira effectID (um Escondido desfeito) de todo registro do turno
// que será gravado: o hidden_from da action e das reações e o
// HiddenFrom das mudanças pendentes. Desfazer é "como se nunca tivesse existido" (controlador):
// um id sem evento nunca poderia ser revelado. No registro das actions, a entrada do personagem
// com esse Escondido sai inteira (efeito e quem o percebia); o mapa vazio vira nil (NULL). A
// entrada de uma reação aberta fica (vazia), porque marca que ela já foi registrada. No
// HiddenFrom de um evento, o jogador que perde o id sai.
func purgeEffectFromTurnLocked(w *turnWrites, effectID uuid.UUID) {
	w.actionHiddenFrom = purgeHiddenFrom(w.actionHiddenFrom, effectID)
	for id, m := range w.reactionHiddenFrom {
		w.reactionHiddenFrom[id] = purgeHiddenFrom(m, effectID)
	}
	for i := range w.statusEffects {
		ev := w.statusEffects[i].Event
		if ev == nil {
			continue
		}
		for pid, eff := range ev.HiddenFrom {
			if eff == effectID {
				delete(ev.HiddenFrom, pid)
			}
		}
		if len(ev.HiddenFrom) == 0 {
			ev.HiddenFrom = nil
		}
	}
}

func purgeHiddenFrom(m statuseffect.HiddenFrom, effectID uuid.UUID) statuseffect.HiddenFrom {
	for c, h := range m {
		if h.Effect == effectID {
			delete(m, c)
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// appliedIn diz se o efeito foi aplicado nas mudanças pendentes do turno. O desfazer depende de
// a gravação "applied" pendente levar o seu evento (sempre, com turno aberto).
func appliedIn(ops []statuseffect.Write, effectID uuid.UUID) bool {
	for _, p := range ops {
		if p.Effect.ID == effectID && !p.Delete && p.Event != nil && p.Event.Change == statuseffect.ChangeApplied {
			return true
		}
	}
	return false
}

func (r *Room) handleRelinkStatusEffect(client *Client, raw json.RawMessage) {
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
		return
	}
	var p RelinkStatusEffectPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.StatusEffectID == uuid.Nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid relink_status_effect payload"))
		return
	}
	charID, err := r.relinkStatusEffect(p.StatusEffectID, p.ActionID)
	if err != nil {
		if errors.Is(err, errMatchNotStarted) {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		r.refuseStatusEffect(client, err)
		return
	}
	r.syncStatusViews([]uuid.UUID{charID})
}

var errMatchNotStarted = errors.New("match session not initialized")

// relinkStatusEffect troca a âncora de toda mudança pendente de effectID no turno aberto
// (nil = do turno). Esquece o que o mestre recebeu do personagem, para que o
// status_effects_updated seguinte vá a ele mesmo sem mudança (a resposta ao relink). Devolve o
// personagem do efeito.
func (r *Room) relinkStatusEffect(effectID uuid.UUID, actionID *uuid.UUID) (uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.session == nil {
		return uuid.Nil, errMatchNotStarted
	}
	turnID := r.openTurnIDLocked()
	if turnID == uuid.Nil {
		return uuid.Nil, errStatusNoOpenTurn
	}
	if actionID != nil && !r.session.IsOpenTurnAction(*actionID) {
		return uuid.Nil, errStatusActionID
	}
	w, ok := r.pendingTurns[turnID]
	var charID uuid.UUID
	found := false
	if ok {
		for i := range w.statusEffects {
			op := &w.statusEffects[i]
			if op.Effect.ID != effectID || op.Event == nil {
				continue
			}
			op.Event.ActionID = cloneUUIDPtr(actionID)
			charID, found = op.Effect.CharacterID, true
		}
	}
	if !found {
		return uuid.Nil, errStatusNoChangeInTurn
	}
	if views := r.statusViews[r.masterUUID]; views != nil {
		delete(views, charID)
	}
	return charID, nil
}
