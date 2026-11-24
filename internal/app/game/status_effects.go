package game

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
	"log"
	"time"

	"github.com/422UR4H/HxH_RPG_System/internal/app/wire/statuswire"
	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	csSheet "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/masteraction"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/action"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	domainservice "github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
)

// errStatusPayload marca o que o fio trouxe malformado (enum desconhecido, ids vazios ou
// repetidos, modo de visibilidade inválido): vira invalid_payload. O tipo desconhecido não
// passa por aqui — é status_effect_invalid, como o domínio o diz.
var errStatusPayload = errtext.Sentence("Não entendi o pedido do efeito.")

// errStatusActionID recusa uma âncora (actionId) que não é a action do turno aberto nem uma
// reação dela — ou qualquer âncora sem turno aberto. É status_effect_invalid.
var errStatusActionID = errtext.Wrap(statuseffect.ErrInvalidParams,
	"Escolha a ação do turno aberto ou uma reação dela.")

// statusPayloadErr é uma recusa de payload com a frase que o mestre lê.
func statusPayloadErr(msg string) error {
	return errtext.Wrap(errStatusPayload, msg)
}

// statusEffectErrorCode traduz os erros dos três comandos para o código do fio.
func statusEffectErrorCode(err error) string {
	switch {
	case errors.Is(err, errStatusPayload):
		return "invalid_payload"
	case errors.Is(err, statuseffect.ErrUnknownKind),
		errors.Is(err, statuseffect.ErrInvalidParams),
		errors.Is(err, statuseffect.ErrInvalidAudience):
		return "status_effect_invalid"
	case errors.Is(err, statuseffect.ErrAlreadyApplied):
		return "status_effect_exists"
	case errors.Is(err, statuseffect.ErrNotFound):
		return "status_effect_not_found"
	case errors.Is(err, matchsession.ErrCharSheetNotFound):
		return "unknown_character"
	case errors.Is(err, appmatch.ErrNotMatchMaster):
		return "forbidden"
	case errors.Is(err, matchsession.ErrMatchEnded):
		return "match_already_ended"
	default:
		return "game_error"
	}
}

// statusEffectErrorText é a frase que o mestre lê para um erro dos comandos de status: a do
// próprio erro (todos os sentinelas que chegam aqui já são frases) ou, para o que não se conhece,
// uma genérica de status, com o motivo real no log.
func statusEffectErrorText(err error) string {
	return errorTextOr(err, "Não foi possível fazer essa mudança de status.")
}

// ─── fio → domínio ──────────────────────────────────────────────────────────

func paramsInputOf(p StatusEffectParamsPayload) (statuseffect.ParamsInput, error) {
	in := statuseffect.ParamsInput{
		Points: p.Points, HPPerRound: p.HPPerRound, Penalty: p.Penalty,
		Disadvantage: p.Disadvantage, Conduct: p.Conduct, Irresistible: p.Irresistible,
		ControllerID: p.ControllerID,
	}
	for _, s := range p.Domains {
		d, err := statuseffect.ParseDomain(s)
		if err != nil {
			return in, statusPayloadErr("Uma das ações bloqueadas não existe.")
		}
		in.Domains = append(in.Domains, d)
	}
	if p.Injuries != nil {
		in.Injuries = make(map[statuseffect.Domain]statuseffect.InjuryLevel, len(p.Injuries))
		for ds, ls := range p.Injuries {
			d, err := statuseffect.ParseDomain(ds)
			if err != nil {
				return in, statusPayloadErr("Uma das lesões fala de uma ação que não existe.")
			}
			l, err := statuseffect.ParseInjuryLevel(ls)
			if err != nil {
				return in, statusPayloadErr("A gravidade de uma lesão não existe.")
			}
			in.Injuries[d] = l
		}
	}
	if p.Level != "" {
		l, err := statuseffect.ParseSenseLevel(p.Level)
		if err != nil {
			return in, statusPayloadErr("A gravidade do sentido não existe.")
		}
		in.Level = l
	}
	for _, s := range p.Faces {
		f, err := statuseffect.ParseTouchFace(s)
		if err != nil {
			return in, statusPayloadErr("Uma das faces do Tato não existe.")
		}
		in.Faces = append(in.Faces, f)
	}
	return in, nil
}

// audienceOf traduz a visibilidade do fio: "default" = nil (padrão do tipo).
func audienceOf(v StatusEffectVisibilityPayload) (*statuseffect.Audience, error) {
	if v.Mode != "players" && len(v.PlayerIDs) > 0 {
		return nil, statusPayloadErr("Para escolher jogadores, marque a opção de jogadores específicos.")
	}
	switch v.Mode {
	case "default":
		return nil, nil
	case "public":
		return &statuseffect.Audience{Everyone: true}, nil
	case "players":
		return &statuseffect.Audience{Players: append([]uuid.UUID{}, v.PlayerIDs...)}, nil
	}
	return nil, statusPayloadErr("Essa forma de visibilidade não existe.")
}

func draftOf(p ApplyStatusEffectPayload) (statuseffect.Draft, error) {
	var d statuseffect.Draft
	k, err := statuseffect.ParseKind(p.Kind)
	if err != nil {
		return d, err
	}
	d.Kind, d.Alias = k, p.Alias
	if p.Sense != "" {
		if d.Sense, err = statuseffect.ParseSense(p.Sense); err != nil {
			return d, statusPayloadErr("Esse sentido não existe.")
		}
	}
	if p.Aptitude != "" {
		if d.Aptitude, err = statuseffect.ParseAptitude(p.Aptitude); err != nil {
			return d, statusPayloadErr("Essa aptidão não existe.")
		}
	}
	if d.Params, err = paramsInputOf(p.Params); err != nil {
		return d, err
	}
	if p.Visibility != nil {
		if d.Audience, err = audienceOf(*p.Visibility); err != nil {
			return d, err
		}
	}
	return d, nil
}

func checkCharacterIDs(ids []uuid.UUID) error {
	if len(ids) == 0 {
		return statusPayloadErr("Escolha ao menos um personagem.")
	}
	seen := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		if id == uuid.Nil {
			return statusPayloadErr("Um dos personagens escolhidos não é válido.")
		}
		if _, dup := seen[id]; dup {
			return statusPayloadErr("Um personagem foi escolhido duas vezes.")
		}
		seen[id] = struct{}{}
	}
	return nil
}

func patchOf(p EditStatusEffectPayload) (statuseffect.Patch, error) {
	patch := statuseffect.Patch{Alias: p.Alias}
	if p.Params != nil {
		in, err := paramsInputOf(*p.Params)
		if err != nil {
			return patch, err
		}
		patch.Params = &in
	}
	if p.Visibility != nil {
		a, err := audienceOf(*p.Visibility)
		if err != nil {
			return patch, err
		}
		patch.SetAudience, patch.Audience = true, a
	}
	return patch, nil
}

// ─── os três braços ─────────────────────────────────────────────────────────

func (r *Room) refuseStatusEffect(client *Client, err error) {
	client.SendMessage(NewErrorMessage(statusEffectErrorCode(err), statusEffectErrorText(err)))
}

func (r *Room) handleApplyStatusEffect(client *Client, raw json.RawMessage) {
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", "Só o mestre pode fazer isso."))
		return
	}
	var p ApplyStatusEffectPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "Não entendi o pedido para aplicar o efeito."))
		return
	}
	if err := checkCharacterIDs(p.CharacterIDs); err != nil {
		r.refuseStatusEffect(client, err)
		return
	}
	draft, err := draftOf(p)
	if err != nil {
		r.refuseStatusEffect(client, err)
		return
	}
	in := appmatch.ApplyStatusEffectInput{CharacterIDs: p.CharacterIDs, Draft: draft}
	r.runStatusEffectChange(client, p.ActionID, false, func(s *matchsession.MatchSession, now time.Time) (appmatch.StatusEffectChange, error) {
		return r.deps.StatusEffectsUC.Apply(s, r.masterUUID, client.userUUID, in, now)
	})
}

func (r *Room) handleEditStatusEffect(client *Client, raw json.RawMessage) {
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", "Só o mestre pode fazer isso."))
		return
	}
	var p EditStatusEffectPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.StatusEffectID == uuid.Nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "Não entendi o pedido para editar o efeito."))
		return
	}
	patch, err := patchOf(p)
	if err != nil {
		r.refuseStatusEffect(client, err)
		return
	}
	r.runStatusEffectChange(client, p.ActionID, false, func(s *matchsession.MatchSession, now time.Time) (appmatch.StatusEffectChange, error) {
		return r.deps.StatusEffectsUC.Edit(s, r.masterUUID, client.userUUID, p.StatusEffectID, patch, now)
	})
}

// handleRemoveStatusEffect remove um efeito. Com restoreCancelledActions, um efeito aplicado no
// próprio turno aberto é desfeito e o que ele tirou volta, revalidado (D3, giveBackLocked): as
// ações da fila que ele cancelou e as reações não abertas que ele trocou no turno; fora dessa
// janela a flag é ignorada — não há o que devolver.
//
// O inverso de aplicar um status no turno aberto é o desfazer com restoreCancelledActions
// (undoInTurnAppliesLocked + giveBackLocked, que chama RestoreCancelledByStatus e
// RestoreSwappedReactions). Um futuro ctrl+Z deve chamar este caminho, nunca escrever a sua
// própria reversão: a reversão revalida contra o estado de agora, sob r.mu, e recusa com motivo o
// que não cabe mais. Cobre: as gravações pendentes do efeito, as ações canceladas (lugar, edições,
// velocidades; o registro de cancelada some) e as reações trocadas (a mesma reação, as âncoras de
// volta; a barra que ela cobrou volta a ser coerente). Não cobre: a janela de fusão do relógio
// fechada pelo cancelamento (clearClockMergeLocked, inofensivo), o que a revalidação recusa (fica
// como o status deixou, a barra de uma reação recusada continua cobrada) e nada depois do
// fechamento.
func (r *Room) handleRemoveStatusEffect(client *Client, raw json.RawMessage) {
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", "Só o mestre pode fazer isso."))
		return
	}
	var p RemoveStatusEffectPayload
	if err := json.Unmarshal(raw, &p); err != nil || p.StatusEffectID == uuid.Nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "Não entendi o pedido para remover o efeito."))
		return
	}
	r.runStatusEffectChange(client, p.ActionID, p.RestoreCancelledActions, func(s *matchsession.MatchSession, _ time.Time) (appmatch.StatusEffectChange, error) {
		return r.deps.StatusEffectsUC.Remove(s, r.masterUUID, client.userUUID, p.StatusEffectID, p.HP)
	})
}

// runStatusEffectChange roda o caso de uso sob r.mu (a sessão não tem trava própria), carimba
// o evento de histórico de cada gravação (stampStatusEventsLocked), segura as gravações e, já
// sem a trava, manda a cada um a lista nova que ele vê. actionID (opcional) é a âncora do
// evento: precisa ser a action do turno aberto ou uma reação dela. restore é o
// restoreCancelledActions da remoção (giveBackLocked); false para aplicar e editar.
func (r *Room) runStatusEffectChange(
	client *Client, actionID *uuid.UUID, restore bool,
	op func(*matchsession.MatchSession, time.Time) (appmatch.StatusEffectChange, error),
) {
	if r.deps.StatusEffectsUC == nil {
		client.SendMessage(NewErrorMessage("game_error", "Os efeitos de status não estão disponíveis nesta partida."))
		return
	}
	r.mu.Lock()
	session := r.session
	if session == nil {
		r.mu.Unlock()
		client.SendMessage(NewErrorMessage("match_not_started", "A partida ainda não começou."))
		return
	}
	if actionID != nil && !session.IsOpenTurnAction(*actionID) {
		r.mu.Unlock()
		r.refuseStatusEffect(client, errStatusActionID)
		return
	}
	// Quem vê cada peça antes da mudança: um Escondido que entra, sai ou troca de lista tira
	// ou devolve peças (decisão do dono 6). O retrato é barato (peças × jogadores conectados) e
	// precisa ser tirado antes do caso de uso, que muda a sessão.
	before := r.pieceSightLocked()
	// O mesmo para o histórico, por jogador da sessão (conectado ou não): quem enxergava cada
	// personagem e para quem ele estava Escondido logo antes da mudança.
	instant := statusInstant{
		at: r.now().Truncate(time.Microsecond), actionID: actionID,
		sight: r.sessionSightLocked(), hidden: r.sessionHiddenFromLocked(),
	}
	change, err := op(session, instant.at)
	if err != nil {
		r.mu.Unlock()
		r.refuseStatusEffect(client, err)
		return
	}
	// Remover um efeito aplicado neste mesmo turno desfaz a aplicação: nada dele é gravado.
	removed := removedEffects(change.Writes)
	var undone map[uuid.UUID]bool
	change.Writes, undone = r.undoInTurnAppliesLocked(change.Writes)
	r.stampStatusEventsLocked(change.Writes, instant)
	// O HP que a mudança moveu (a volta do Morto) é gravado como o de um fechamento falho: fica
	// em unwrittenSheets e vai com a próxima gravação — fora de turno, o flushStatusEffects
	// logo abaixo, na mesma transação das gravações de status; dentro de um turno, o fechamento.
	hpChanged := hpChangesOf(change.HP)
	r.markSheetsUnwrittenLocked(hpChanged)
	var after map[uuid.UUID]map[string]bool
	if change.HiddenChanged {
		after = r.pieceSightLocked()
	}
	flushNow := r.holdStatusEffectWritesLocked(change.Writes)
	// O conjunto novo vale já para o que está pendente (decisão 8): na mesma seção crítica.
	cancelWrites, cancelled, turnTouched := r.dropRefusedByStatusLocked(session, change.Characters, changedEffects(change.Writes))
	// O desfazer pedido devolve, na mesma seção crítica, o que o efeito desfeito cancelou (D3);
	// a remoção já saiu da sessão, então ele não recusa mais nada.
	back := r.giveBackLocked(removed, undone, restore)
	if len(back.actions) > 0 || len(back.reactions) > 0 {
		turnTouched = true
	}
	// Um HP novo muda a prévia do turno aberto (quem morre com o golpe).
	if len(hpChanged) > 0 && r.openTurnIDLocked() != uuid.Nil {
		turnTouched = true
	}
	// Uma reação refeita pelo status leva junto as âncoras presas a ela.
	followed := r.followAnchorsLocked()
	turnID := session.CurrentTurnID()
	var res *domainservice.TurnResolution
	var resAge uint64
	if turnTouched {
		res = session.ResolveOpenTurn()
		resAge = r.stampResolutionLocked()
	}
	r.mu.Unlock()
	// Grava antes de avisar, como as master actions: o aviso é a barreira de quem espera a
	// gravação.
	if flushNow {
		r.flushStatusEffects()
	}
	// Depois da tentativa de gravação, como o dano do fechamento: o mestre e o dono.
	r.broadcastHpChanges(hpChanged)
	for _, w := range cancelWrites {
		r.writeCancelledAction(w)
	}
	if len(cancelled) > 0 {
		r.announceStatusCancelled(cancelled)
	}
	r.announceRestored(back)
	if len(cancelled) > 0 || len(back.actions) > 0 {
		r.broadcastBars(session)
	}
	// Uma reação do turno aberto pode ter sido refeita: o mestre recebe a resolução nova.
	r.publishResolution(turnID, res, resAge)
	if change.HiddenChanged {
		// A peça primeiro (piece_removed/piece_moved), depois as barras e os status de quem a
		// ganhou ou perdeu — a ordem que a névoa já segue quando uma peça anda.
		r.relayHiddenChange(before, after)
		// A peça que o mestre observa vê como o dono dela: um Escondido que muda muda a vista.
		r.pushMasterPieceView()
		r.syncSight()
	}
	// Sem Escondido, a visão não mudou: só os personagens tocados. Com ele, o syncSight acima
	// já cobriu quem ainda tem efeitos; este cobre o que acabou de ficar sem nenhum.
	r.syncStatusViews(append(append([]uuid.UUID(nil), change.Characters...), followed...))
}

// statusInstant é o instante de uma mudança de status, como o histórico o carimba: a hora, a
// âncora pedida e o retrato de antes da mudança (sight: personagem → jogador → enxergava;
// hidden: personagem → jogador → efeito Escondido que o escondia dele).
type statusInstant struct {
	at       time.Time
	actionID *uuid.UUID
	sight    map[uuid.UUID]map[uuid.UUID]bool
	hidden   map[uuid.UUID]map[uuid.UUID]uuid.UUID
	// turnID e round, quando dados, são o turno e o round do evento no lugar do turno aberto e do
	// round ativo: a morte é do turno que FECHOU (recordDeathsLocked), e o round dele pode já
	// ter dado lugar ao sucessor no mesmo comando.
	turnID *uuid.UUID
	round  *roundentity.Round
}

// stampStatusEventsLocked completa o evento que o caso de uso pôs em cada gravação (só a
// mudança) com o que só a sala sabe: partida, cena, round, turno aberto, âncora, a lista de
// quem vê já resolvida, quem enxergava o personagem logo antes OU logo depois (mais o dono;
// o mestre nunca) e para quem ele estava Escondido. Sem cena ou round ativos o evento é
// descartado (não há onde pendurá-lo) e a gravação do efeito segue sem ele.
//
// O chamador segura r.mu (escrita) e r.session não é nil.
func (r *Room) stampStatusEventsLocked(writes []statuseffect.Write, in statusInstant) {
	sc, rd := r.session.GetActiveScene(), r.session.GetActiveRound()
	if in.round != nil {
		rd = in.round
	}
	var turnID *uuid.UUID
	if in.turnID != nil {
		turnID = in.turnID
	} else if id := r.openTurnIDLocked(); id != uuid.Nil {
		turnID = &id
	}
	owners := r.session.GetCharToPlayer()
	for i := range writes {
		ev := writes[i].Event
		if ev == nil {
			continue
		}
		e := writes[i].Effect
		if sc == nil || rd == nil {
			log.Printf("evento de status do efeito %s (%s) na partida %s fica fora do histórico: "+
				"a sessão não tem cena ou round ativos", e.ID, ev.Change, r.matchUUID)
			writes[i].Event = nil
			continue
		}
		charID := e.CharacterID
		owner := owners[charID.String()]
		subject, pcPublic := r.statusSubjectLocked(charID)
		ev.ID = uuid.New()
		ev.MatchID, ev.SceneID, ev.RoundID = r.session.GetMatchUUID(), sc.GetID(), rd.GetID()
		ev.TurnID = cloneUUIDPtr(turnID)
		ev.ActionID = cloneUUIDPtr(in.actionID)
		ev.Listing = statuseffect.ResolveListing(e, subject, pcPublic)
		ev.Views = r.sessionPlayerViewsLocked(owner, func(pid uuid.UUID) (masteraction.View, bool) {
			return masteraction.ViewFull, in.sight[charID][pid] || r.seesCharacterLocked(pid, charID)
		})
		if owner != r.masterUUID && owner != uuid.Nil {
			ev.Views[owner] = masteraction.ViewFull
		}
		ev.HiddenFrom = r.statusHiddenFromLocked(charID, owner, ev.Views, in.hidden[charID])
		ev.HappenedAt = in.at
	}
}

// statusHiddenFromLocked é o HiddenFrom de um evento: os jogadores da sessão (nunca o mestre
// nem o dono) que não estão em views e para quem o personagem estava Escondido logo depois da
// mudança — ou, se não mais, logo antes (a remoção do Escondido). nil quando ninguém.
//
// O chamador segura r.mu (leitura basta) e r.session não é nil.
func (r *Room) statusHiddenFromLocked(
	charID, owner uuid.UUID, views map[uuid.UUID]masteraction.View, before map[uuid.UUID]uuid.UUID,
) map[uuid.UUID]uuid.UUID {
	after := r.hiddenFromMapLocked(charID)
	var out map[uuid.UUID]uuid.UUID
	for _, pid := range r.session.PlayerIDs() {
		if pid == r.masterUUID || pid == owner {
			continue
		}
		if _, saw := views[pid]; saw {
			continue
		}
		hid, ok := after[pid]
		if !ok {
			hid, ok = before[pid]
		}
		if !ok {
			continue
		}
		if out == nil {
			out = make(map[uuid.UUID]uuid.UUID)
		}
		out[pid] = hid
	}
	return out
}

func cloneUUIDPtr(p *uuid.UUID) *uuid.UUID {
	if p == nil {
		return nil
	}
	id := *p
	return &id
}

// holdStatusEffectWritesLocked guarda as gravações de uma mudança e diz se elas devem ser
// escritas já (regra da casa: o que acontece dentro de um turno aberto só dura com o fechamento
// dele, spec §10).
//
//   - Turno aberto: vão para turnWrites.statusEffects e o persistClosedTurn as leva no
//     PersistTurnClose, na transação do turno — com ou sem StatusEffectRepo.
//   - Fora de turno, sem repositório: nada é gravado nem acumulado.
//   - Fora de turno: entram no fim de unwrittenStatusEffects (atrás do que uma gravação
//     anterior não gravou) e o chamador, já sem r.mu, chama flushStatusEffects.
//
// O chamador segura r.mu (escrita).
func (r *Room) holdStatusEffectWritesLocked(ops []statuseffect.Write) (flushNow bool) {
	if len(ops) == 0 {
		return false
	}
	if turnID := r.openTurnIDLocked(); turnID != uuid.Nil {
		w := r.turnWritesLocked(turnID)
		w.statusEffects = append(w.statusEffects, ops...)
		return false
	}
	if r.deps.StatusEffectRepo == nil {
		return false
	}
	r.unwrittenStatusEffects = append(r.unwrittenStatusEffects, ops...)
	return true
}

// flushStatusEffects grava, fora de turno, tudo o que está em unwrittenStatusEffects, numa
// chamada só e em ordem, com as barras das fichas de unwrittenSheets na mesma transação.
// persistMu é segurado da cópia até o descarte, o mesmo que o persistClosedTurn faz com a sua:
// os dois nunca correm juntos, e cada um descarta exatamente o prefixo que copiou (o que chegou
// depois da cópia fica para a próxima gravação). Uma falha é registrada no log e as gravações e
// as fichas continuam pendentes juntas: vão na próxima gravação fora de turno ou no próximo
// fechamento de turno (unwritten.go).
//
// O chamador NÃO segura nenhuma trava.
func (r *Room) flushStatusEffects() {
	if r.deps.StatusEffectRepo == nil {
		return
	}
	r.persistMu.Lock()
	defer r.persistMu.Unlock()
	r.mu.Lock()
	pending := append([]statuseffect.Write(nil), r.unwrittenStatusEffects...)
	// As fichas cujo HP ainda não foi gravado (a volta do Morto, um fechamento falho) vão na
	// mesma transação: cópias das barras, tiradas na mesma seção crítica.
	sheets := make(map[uuid.UUID]*csSheet.CharacterSheet, len(r.unwrittenSheets))
	for id, sh := range r.unwrittenSheets {
		sheets[id] = sh
	}
	bars := statusBarsLocked(sheets)
	r.mu.Unlock()
	if len(pending) == 0 && len(bars) == 0 {
		return
	}
	if err := r.deps.StatusEffectRepo.Write(context.Background(), pending, bars); err != nil {
		log.Printf("gravação de efeitos de status FALHOU na partida %s — %d gravação(ões) e o HP de %d ficha(s) "+
			"continuam pendentes e vão na próxima gravação ou no próximo fechamento de turno: %v",
			r.matchUUID, len(pending), len(bars), err)
		return
	}
	r.mu.Lock()
	r.dropWrittenStatusEffectsLocked(len(pending))
	// Esquece por id: hoje nada remarca a ficha entre a cópia e aqui — só o mestre traz alguém do
	// Morto, na goroutine de leitura dele, atrás do próprio flush.
	r.forgetSheetsWrittenLocked(sheets)
	r.mu.Unlock()
}

// dropWrittenStatusEffectsLocked descarta as n primeiras gravações pendentes, que uma gravação
// acabou de levar. O chamador segura persistMu e r.mu (escrita).
func (r *Room) dropWrittenStatusEffectsLocked(n int) {
	rest := r.unwrittenStatusEffects[n:]
	if len(rest) == 0 {
		r.unwrittenStatusEffects = nil
		return
	}
	r.unwrittenStatusEffects = append([]statuseffect.Write(nil), rest...)
}

// keepFailedTurnStatusEffectsLocked guarda as gravações de um turno cujo fechamento falhou,
// logo depois das n pendentes que aquele fechamento copiou e antes de qualquer uma que tenha
// chegado depois da cópia: a ordem em que aconteceram. O chamador segura persistMu e r.mu
// (escrita).
func (r *Room) keepFailedTurnStatusEffectsLocked(n int, inTurn []statuseffect.Write) {
	if len(inTurn) == 0 {
		return
	}
	out := make([]statuseffect.Write, 0, len(r.unwrittenStatusEffects)+len(inTurn))
	out = append(out, r.unwrittenStatusEffects[:n]...)
	out = append(out, inTurn...)
	out = append(out, r.unwrittenStatusEffects[n:]...)
	r.unwrittenStatusEffects = out
}

// ─── projeção ───────────────────────────────────────────────────────────────

// projectStatusEffectsLocked é a lista de efeitos de charID que pid vê. O mestre vê tudo, com
// a lista de quem vê; os demais, só o que a lista (ou o padrão do tipo) lhes dá, e só enquanto
// enxergam o personagem (seesCharacterLocked — o dono sempre enxerga o seu). known=false: este
// jogador não vê o personagem agora — não se manda nada nem se guarda nada (decisão do dono 4:
// ele fica com o último estado que viu).
//
// O chamador segura r.mu (leitura basta).
func (r *Room) projectStatusEffectsLocked(charID uuid.UUID, pid uuid.UUID, isMaster bool) (list []StatusEffectPayload, known bool) {
	session := r.session
	if session == nil {
		return nil, false
	}
	if !isMaster && !r.seesCharacterLocked(pid, charID) {
		return nil, false
	}
	subject, pcPublic := r.statusSubjectLocked(charID)
	viewer := statuseffect.Viewer{UserID: pid, IsMaster: isMaster}
	list = []StatusEffectPayload{}
	for _, e := range session.StatusEffectsOf(charID) {
		if !statuseffect.Listed(e, viewer, subject, pcPublic) {
			continue
		}
		w := statuswire.From(e)
		if isMaster {
			w.TurnActionID = r.pendingAnchorLocked(e.ID)
			w.CancelledActionIDs, w.SwappedReactionIDs = r.cancelledActionIDsLocked(e.ID)
			vis := statuswire.VisibilityOf(e.Audience)
			// Público = está na lista de um jogador qualquer, que não é o dono (a visão à parte).
			public := statuseffect.Listed(e, statuseffect.Viewer{UserID: uuid.Nil}, subject, pcPublic)
			w.Visibility, w.Public = &vis, &public
		}
		list = append(list, w)
	}
	return list, true
}

// statusSubjectLocked é o personagem como statuseffect.Listed o lê (dono, se é NPC) e a regra
// da partida que abre os escondidos de um PC. Uma conta só, para a projeção e para o
// Escondido (hiddenFromLocked) nunca divergirem sobre quem está na lista. O chamador segura
// r.mu (leitura basta) e r.session não é nil.
func (r *Room) statusSubjectLocked(charID uuid.UUID) (statuseffect.Subject, bool) {
	owner := r.session.GetCharToPlayer()[charID.String()]
	return statuseffect.Subject{OwnerID: owner, IsNPC: owner == r.masterUUID},
		r.session.GetRules().PlayerStatusEffectsPublic
}

// ─── envio ──────────────────────────────────────────────────────────────────

// emptyStatusList é como uma lista vazia sai no fio; um personagem sem entrada no cache vale
// por ela (o jogador nunca recebeu nada dele, então não há o que apagar).
var emptyStatusList = []byte("[]")

// syncStatusViews manda, a cada conectado (o mestre incluído), a lista nova de cada personagem
// em chars — só quando ela difere da última que ele recebeu (statusViews) e só se ele vê o
// personagem. Um status_effects_updated por (destinatário, personagem) que mudou. O chamador
// NÃO segura r.mu: dispatchPerPlayer a toma, e a projeção de cada um a toma de novo.
func (r *Room) syncStatusViews(chars []uuid.UUID) {
	for _, charID := range chars {
		r.dispatchPerPlayer(func(pid uuid.UUID, isMaster bool) *Message {
			r.mu.Lock()
			defer r.mu.Unlock()
			list, known := r.projectStatusEffectsLocked(charID, pid, isMaster)
			if !known {
				return nil
			}
			data, _ := json.Marshal(list)
			prev := r.statusViews[pid][charID]
			if prev == nil {
				prev = emptyStatusList
			}
			if bytes.Equal(prev, data) {
				return nil
			}
			r.noteStatusViewLocked(pid, charID, data)
			msg := NewServerMessage(MsgTypeStatusEffectsUpdated, StatusEffectsUpdatedPayload{
				CharacterID: charID, StatusEffects: list,
			})
			return &msg
		})
	}
}

// noteStatusViewLocked registra a lista que pid acabou de receber de charID. O chamador segura
// r.mu (escrita).
func (r *Room) noteStatusViewLocked(pid, charID uuid.UUID, data []byte) {
	if r.statusViews == nil {
		r.statusViews = make(map[uuid.UUID]map[uuid.UUID][]byte)
	}
	views := r.statusViews[pid]
	if views == nil {
		views = make(map[uuid.UUID][]byte)
		r.statusViews[pid] = views
	}
	views[charID] = data
}

// statusEffectsForLocked monta o match_full_state.statusEffects de pid: cada personagem com
// efeitos que ele vê agora e que tem algo visível para ele. Devolve também o cache exato do
// que vai (statusViews[pid] é trocado por ele). O chamador segura r.mu (leitura basta).
func (r *Room) statusEffectsForLocked(pid uuid.UUID, isMaster bool) (map[uuid.UUID][]StatusEffectPayload, map[uuid.UUID][]byte) {
	views := make(map[uuid.UUID][]byte)
	if r.session == nil {
		return nil, views
	}
	var out map[uuid.UUID][]StatusEffectPayload
	for _, charID := range r.session.CharactersWithStatusEffects() {
		list, known := r.projectStatusEffectsLocked(charID, pid, isMaster)
		if !known || len(list) == 0 {
			continue
		}
		if out == nil {
			out = make(map[uuid.UUID][]StatusEffectPayload)
		}
		out[charID] = list
		views[charID], _ = json.Marshal(list)
	}
	return out, views
}

// ─── bloqueios (decisões 8 e 9) ─────────────────────────────────────────────

// actionErrorCode traduz o erro de enqueue_action/attach_reaction: a recusa de um status vira
// blocked_by_status; o resto continua game_error.
func actionErrorCode(err error) string {
	if errors.Is(err, matchsession.ErrBlockedByStatus) {
		return "blocked_by_status"
	}
	return "game_error"
}

// actionErrorMessageLocked é o texto do erro de enqueue_action/attach_reaction para quem
// mandou. Uma recusa por status nomeia só os tipos que o remetente pode ver — o mestre, todos;
// um jogador, os da lista dele pela mesma regra da projeção (statuseffect.Listed); nenhum
// visível vira "um efeito" (§9). O remetente é sempre o dono do personagem ou o mestre (a
// sessão confere a posse antes), e o dono sempre enxerga o seu, então a visão não entra.
// O chamador segura r.mu (leitura basta).
func (r *Room) actionErrorMessageLocked(err error, sender uuid.UUID) string {
	var be *matchsession.StatusBlockError
	if !errors.As(err, &be) || r.session == nil {
		return gameErrorText(err)
	}
	isMaster := sender == r.masterUUID
	subject, pcPublic := r.statusSubjectLocked(be.CharacterID)
	viewer := statuseffect.Viewer{UserID: sender, IsMaster: isMaster}
	effects := r.session.StatusEffectsOf(be.CharacterID)
	var visible []statuseffect.Kind
	for _, k := range be.Kinds {
		for _, e := range effects {
			if e.Kind == k && statuseffect.Listed(e, viewer, subject, pcPublic) {
				visible = append(visible, k)
				break
			}
		}
	}
	return be.Message(visible)
}

// statusNotesFor são as anotações dos status que valem para a action: um aviso de domínio só
// quando ela usa o domínio; uma conduta, sempre. nil quando não há nenhuma.
func statusNotesFor(sum statuseffect.Summary, a *action.Action) []StatusNotePayload {
	used := map[statuseffect.Domain]bool{}
	for _, d := range matchsession.ActionDomains(a) {
		used[d] = true
	}
	var out []StatusNotePayload
	for _, n := range sum.Notes {
		if n.Domain != "" && !used[n.Domain] {
			continue
		}
		out = append(out, StatusNotePayload{
			Kind: string(n.Kind), Domain: string(n.Domain), Conduct: n.Conduct, Irresistible: n.Irresistible,
		})
	}
	return out
}

// dropRefusedByStatusLocked aplica a mudança de status ao que os personagens têm pendente
// (decisão 8), na mesma seção crítica do caso de uso: cada action que o conjunto novo recusa sai
// da fila e é registrada como cancelada pelo mestre, pelo mesmo caminho do cancel_action (com
// turno aberto, retida para o fechamento dele). Com turno aberto, o que saiu também vai para o
// registro do desfazer (turnWrites.statusCancels) de cada efeito do personagem que esta mudança
// aplicou ou editou (changed), e as reações não abertas que ela trocou no turno também:
// atribuir a mais de um é seguro, porque a devolução revalida.
// Devolve o que gravar já, os avisos a mandar e se alguma reação do turno aberto pode ter mudado.
// O chamador segura r.mu (escrita).
func (r *Room) dropRefusedByStatusLocked(
	session *matchsession.MatchSession, chars []uuid.UUID, changed map[uuid.UUID][]uuid.UUID,
) (writes []*cancelledWrite, cancelled []cancelledNotice, turnTouched bool) {
	owners := session.GetCharToPlayer()
	turnID := r.openTurnIDLocked()
	for _, charID := range chars {
		turnTouched = turnTouched || session.TakesPartInOpenTurn(charID)
		drops, swapped := session.DropRefusedByStatusWithReactions(charID)
		if turnID != uuid.Nil && len(swapped) > 0 {
			w := r.turnWritesLocked(turnID)
			for _, effectID := range changed[charID] {
				w.noteReactionSwap(effectID, charID, swapped)
			}
		}
		if len(drops) == 0 {
			continue
		}
		if turnID != uuid.Nil {
			w := r.turnWritesLocked(turnID)
			for _, effectID := range changed[charID] {
				w.noteStatusCancel(effectID, charID, drops)
			}
		}
		ws, ids := r.holdCancelledActionsLocked(matchsession.DroppedActions(drops), r.masterUUID)
		writes = append(writes, ws...)
		for _, id := range ids {
			cancelled = append(cancelled, cancelledNotice{id: id, owner: owners[charID.String()]})
		}
	}
	return writes, cancelled, turnTouched
}

// removedEffects são os efeitos que writes remove, em ordem, como estavam antes da remoção.
func removedEffects(writes []statuseffect.Write) []statuseffect.Effect {
	var out []statuseffect.Effect
	for _, op := range writes {
		if op.Delete {
			out = append(out, op.Effect)
		}
	}
	return out
}

// changedEffects são, por personagem, os efeitos que writes aplica ou edita.
func changedEffects(writes []statuseffect.Write) map[uuid.UUID][]uuid.UUID {
	var out map[uuid.UUID][]uuid.UUID
	for _, op := range writes {
		if op.Delete {
			continue
		}
		if out == nil {
			out = map[uuid.UUID][]uuid.UUID{}
		}
		out[op.Effect.CharacterID] = append(out[op.Effect.CharacterID], op.Effect.ID)
	}
	return out
}

// cancelledNotice é um action_cancelled a mandar: a action e o dono do personagem dela.
type cancelledNotice struct {
	id, owner uuid.UUID
}

// announceStatusCancelled manda o action_cancelled de cada action que um status tirou da fila:
// ao mestre e ao dono dela — a mesma mensagem do cancel_action, que a manda ao mestre e a quem
// pediu (aqui quem pediu é o mestre, e quem precisa saber é o dono). Fora do lock.
func (r *Room) announceStatusCancelled(notices []cancelledNotice) {
	for _, n := range notices {
		msg := NewServerMessage(MsgTypeActionCancelled, ActionCancelledPayload{ActionID: n.id})
		r.sendToMaster(msg)
		if n.owner == r.masterUUID || n.owner == uuid.Nil {
			continue
		}
		r.mu.RLock()
		c, ok := r.clients[n.owner]
		r.mu.RUnlock()
		if ok {
			c.SendMessage(msg)
		}
	}
}

// announceRestored anuncia o que um desfazer devolveu. Cada ação de volta à fila: action_restored
// ao mestre (actionwire.Full) e ao dono (o corte do ownQueue), e o action_queued do mestre, como
// no enqueue — depois do action_restored, para que o cliente, que já a pôs no lugar
// (beforeActionId), só a redescreva. Não usa action_enqueued: ele é o ack de um envio do
// cliente, casado pela ordem dos envios, e um ack que ninguém pediu cairia no envio errado.
// Cada reação de volta ao turno: reaction_restored ao mestre e, quando o efeito desfeito era
// visível a ele, ao dono — não reaction_attached, que o cliente casa com o envio pendente do
// mesmo personagem. Cada recusa vai só ao mestre, como error restore_refused. Fora do lock.
func (r *Room) announceRestored(back givenBack) {
	for _, n := range back.actions {
		r.sendToMaster(NewServerMessage(MsgTypeActionRestored, n.master))
		r.sendToMaster(NewServerMessage(MsgTypeActionQueued, n.queued))
		if n.ownerMsg == nil {
			continue
		}
		r.mu.RLock()
		c, ok := r.clients[n.owner]
		r.mu.RUnlock()
		if ok {
			c.SendMessage(NewServerMessage(MsgTypeActionRestored, *n.ownerMsg))
		}
	}
	for _, n := range back.reactions {
		r.sendToMaster(NewServerMessage(MsgTypeReactionRestored, n.master))
		if n.ownerMsg == nil {
			continue
		}
		r.mu.RLock()
		c, ok := r.clients[n.owner]
		r.mu.RUnlock()
		if ok {
			c.SendMessage(NewServerMessage(MsgTypeReactionRestored, *n.ownerMsg))
		}
	}
	for _, f := range back.refusals {
		r.sendToMaster(NewErrorMessage("restore_refused", "A ação não voltou: "+f.Reason+"."))
	}
	for _, f := range back.reactionRefusals {
		r.sendToMaster(NewErrorMessage("restore_refused", "A reação não voltou: "+f.Reason+"."))
	}
}

// ─── morte no fechamento ────────────────────────────────────────────────────

// closedDeaths é o que os três fechamentos anunciam de uma morte, depois do HP: os personagens
// que morreram (a lista nova de status, pela névoa) e as ações da fila que o Morto cancelou.
type closedDeaths struct {
	chars     []uuid.UUID
	cancelled []cancelledNotice
}

// recordDeathsLocked registra os Mortos que o fechamento de closed aplicou (TurnTransition.
// Deaths): carimba cada evento como runStatusEffectChange carimba os seus — causa = a action do
// turno fechado, retrato de visão e Escondido da sala neste instante —, mas no turno que fechou
// (e no round dele, closedIn quando o mesmo comando acabou o round), e guarda as gravações nos
// turnWrites DESSE turno, que o persistClosedTurn leva no PersistTurnClose: o Morto entra na
// transação do turno que o causou. As ações da fila que o Morto tirou (drops, já fora da fila na
// sessão) são registradas como canceladas pelo mestre — a regra da fase 1 para todo status —
// também no turno fechado. Chamado na mesma seção crítica do Execute do fechamento, antes do
// persistClosedTurn; nada a fazer sem morte.
//
// O chamador segura r.mu (escrita) e r.session não é nil.
func (r *Room) recordDeathsLocked(
	closed *turnentity.Turn, closedIn *roundentity.Round,
	deaths []statuseffect.Write, drops []matchsession.StatusDrop,
) closedDeaths {
	if closed == nil || (len(deaths) == 0 && len(drops) == 0) {
		return closedDeaths{}
	}
	turnID := closed.GetID()
	act := closed.GetAction()
	actionID := act.GetID()
	instant := statusInstant{
		at: r.now().Truncate(time.Microsecond), actionID: &actionID,
		sight: r.sessionSightLocked(), hidden: r.sessionHiddenFromLocked(),
		turnID: &turnID, round: closedIn,
	}
	writes := append([]statuseffect.Write(nil), deaths...)
	r.stampStatusEventsLocked(writes, instant)
	w := r.turnWritesLocked(turnID)
	w.deaths = append(w.deaths, writes...)

	var out closedDeaths
	for _, op := range writes {
		out.chars = append(out.chars, op.Effect.CharacterID)
	}
	owners := r.session.GetCharToPlayer()
	acts := matchsession.DroppedActions(drops)
	if len(acts) > 0 {
		// Um cancelamento fecha a janela de fusão do relógio, como em holdCancelledActionsLocked.
		r.clearClockMergeLocked()
	}
	for _, a := range acts {
		// Com o turno dito, o registro fica retido para ele: nada a gravar agora.
		r.holdCancelledActionInLocked(a, r.masterUUID, turnID, closedIn)
		out.cancelled = append(out.cancelled, cancelledNotice{id: a.GetID(), owner: owners[a.GetActorID().String()]})
	}
	return out
}

// announceDeaths anuncia o que recordDeathsLocked registrou, já gravado: o action_cancelled de
// cada ação que o Morto tirou da fila e a lista nova de status de cada morto, a quem vê o
// personagem (syncStatusViews; quem não vê recebe quando a peça entrar na visão). Os três
// fechamentos o chamam depois do broadcastHpChanges e antes do broadcastTurnClosed: o HP que
// matou, a morte, o fim do turno. Fora do lock.
func (r *Room) announceDeaths(d closedDeaths) {
	if len(d.cancelled) > 0 {
		r.announceStatusCancelled(d.cancelled)
	}
	if len(d.chars) > 0 {
		r.syncStatusViews(d.chars)
	}
}

// hpChangesOf traduz o HP que uma mudança de status moveu para o formato do dano do fechamento:
// o mesmo caminho de gravação (markSheetsUnwrittenLocked) e de aviso (broadcastHpChanges).
func hpChangesOf(changes []appmatch.HPChange) []matchsession.DamagedCharacter {
	if len(changes) == 0 {
		return nil
	}
	out := make([]matchsession.DamagedCharacter, 0, len(changes))
	for _, c := range changes {
		out = append(out, matchsession.DamagedCharacter{
			CharacterID: c.CharacterID, Sheet: c.Sheet, Damage: c.Damage, NewHP: c.NewHP,
		})
	}
	return out
}
