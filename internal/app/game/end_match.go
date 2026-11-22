package game

import (
	"context"
	"encoding/json"
	"log"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/statuseffect"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// closeCodeMatchEnded é o código de fechamento de uma partida encerrada (spec 2026-11-08 §5).
const closeCodeMatchEnded = 4002

const closeTextMatchEnded = "match ended"

// handleEndMatch encerra a partida (spec 2026-11-08 §5, Ruling R2): valida na sessão e monta o
// que a escrita precisa sob r.mu; grava tudo numa transação fora dele; só se a escrita deu certo
// aplica o fim na sessão. Depois, match_ended à mesa inteira, toda conexão fecha com 4002 e a
// sala para (RoomStateClosed: a próxima conexão cria outra sala, que o handler recusa porque a
// partida está encerrada). Se a escrita falha, o mestre recebe game_error, a sessão fica intacta
// e a sala segue.
func (r *Room) handleEndMatch(client *Client, raw json.RawMessage) {
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
		return
	}
	var payload EndMatchPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "Não entendi o pedido para encerrar a partida."))
		return
	}
	storyAt, err := clock.ParseStory(payload.StoryAt)
	if err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "Hora da história inválida. Use o formato AAAA-MM-DDTHH:MM:SS."))
		return
	}
	brief, err := matchsession.NormalizeBriefFinal(payload.BriefFinalDescription)
	if err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "O resumo final deve ter no máximo 255 caracteres."))
		return
	}
	if r.deps.MatchEndRepo == nil {
		client.SendMessage(NewErrorMessage("game_error", "Não dá para encerrar a partida agora."))
		return
	}

	// clockPersistMu antes de r.mu (a ordem de persistClock): nenhum checkpoint do relógio
	// rodando grava entre o retrato do fim e a aplicação dele.
	r.clockPersistMu.Lock()
	defer r.clockPersistMu.Unlock()
	// persistMu em seguida (clockPersistMu → persistMu → r.mu; nada que segura persistMu toma
	// clockPersistMu): nenhum flushStatusEffects ou fechamento de turno grava ou descarta as
	// mudanças de status pendentes entre a cópia que o fim leva e o descarte dela.
	r.persistMu.Lock()
	defer r.persistMu.Unlock()

	r.mu.Lock()
	sess := r.session
	if sess == nil {
		r.mu.Unlock()
		client.SendMessage(NewErrorMessage("match_not_started", "A partida ainda não começou."))
		return
	}
	plan, err := sess.PlanEndMatch(storyAt, payload.ConfirmLongSkip, r.now())
	if err != nil {
		r.mu.Unlock()
		client.SendMessage(NewErrorMessage(clockErrorCode(err), gameErrorText(err)))
		return
	}
	data := r.matchEndDataLocked(sess, plan, brief)
	r.mu.Unlock()

	if err := r.deps.MatchEndRepo.PersistMatchEnd(context.Background(), data); err != nil {
		log.Printf("end_match FAILED — match %s NOT ended (session untouched): %v", r.matchUUID, err)
		client.SendMessage(NewErrorMessage("game_error", "Não foi possível encerrar a partida."))
		return
	}

	r.mu.Lock()
	// Mesmo instante real do plano: o fim aplicado é o que foi gravado.
	if _, err := sess.EndMatchAt(storyAt, payload.ConfirmLongSkip, plan.ClosedAt); err != nil {
		// O banco já diz encerrada; a sala fecha do mesmo jeito.
		log.Printf("end_match: match %s written as ended but the session refused the end: %v", r.matchUUID, err)
	}
	r.forgetRoundEndsWrittenLocked(data.UnwrittenRoundEnds)
	r.unwrittenSheets = nil
	r.dropWrittenStatusEffectsLocked(len(data.StatusEffects))
	r.clearClockMergeLocked()
	r.state = RoomStateClosed
	msg, _ := json.Marshal(NewServerMessage(MsgTypeMatchEnded, MatchEndedPayload{
		MatchID: r.matchUUID, StoryAt: storyPtr(plan.StoryAt), GameEndAt: plan.ClosedAt.UTC().Format(time.RFC3339),
	}))
	r.endedMsg = msg
	clients := make([]*Client, 0, len(r.clients))
	for _, c := range r.clients {
		clients = append(clients, c)
	}
	r.mu.Unlock()

	for _, c := range clients {
		sendMatchEnded(c, msg)
	}
	r.Stop()
}

// isMatchEnded diz se esta sala já encerrou a partida. Toma r.mu.
func (r *Room) isMatchEnded() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.endedMsg != nil || (r.session != nil && r.session.IsEnded())
}

// sendMatchEnded põe o match_ended na fila do cliente e o fecha com 4002. Pela faixa direta, não
// pelo canal broadcast: a sala para logo depois e o Run pode sair antes de drená-lo. A WritePump
// esvazia a fila (match_ended) e só então escreve o 4002.
func sendMatchEnded(c *Client, msg []byte) {
	select {
	case c.send <- msg:
	default:
		log.Printf("dropping match_ended for slow client %s", c.userUUID)
	}
	c.CloseWith(closeCodeMatchEnded, closeTextMatchEnded)
}

// matchEndDataLocked monta a escrita do fim a partir do plano: o par ativo fechado (cópias), os
// fins de round ainda não gravados, o relógio parado e o clockRealTime da pausa, quando houve.
// O chamador segura r.mu.
func (r *Room) matchEndDataLocked(
	sess *matchsession.MatchSession, plan *matchsession.EndMatchPlan, brief *string,
) appmatch.MatchEndData {
	sc, rd := snapshotSceneAndRound(sess.GetActiveScene(), sess.GetActiveRound())
	rd.SetDrained(plan.Drained)
	rd.Close(plan.ClosedAt)
	sc.Close(plan.ClosedAt)
	sc.SetStoryFinishedAt(plan.StoryAt)
	// O relógio do fim é um retrato que vai a uma gravação: um checkpoint anterior não o sobrescreve.
	r.clockWriteGen.Add(1)
	d := appmatch.MatchEndData{
		MatchUUID: r.matchUUID, GameEndAt: plan.ClosedAt.UTC(), StoryAt: plan.StoryAt,
		BriefFinalDescription: brief,
		End:                   appmatch.RoundEnd{Scene: sc, Round: rd},
		UnwrittenRoundEnds:    r.unwrittenRoundEndsLocked(),
		Clock:                 plan.Clock,
		// O HP que um fechamento não gravou: o fim é a última escrita da partida.
		StatusBars: statusBarsLocked(r.unwrittenSheets),
		// E, pelo mesmo motivo, as mudanças de status que uma gravação anterior não gravou.
		StatusEffects: append([]statuseffect.Write(nil), r.unwrittenStatusEffects...),
	}
	if plan.Pause != nil {
		payload, err := json.Marshal(pauseEvent(plan.Pause).payload)
		if err != nil {
			log.Printf("end_match: clockRealTime of match %s NOT recorded (payload): %v", r.matchUUID, err)
		} else {
			d.RealTimeEvent = &matchevent.Event{
				UUID: uuid.New(), MatchUUID: r.matchUUID, SceneUUID: sc.GetID(), RoundUUID: rd.GetID(),
				Kind: matchevent.KindClockRealTime, Payload: payload, CreatedAt: plan.ClosedAt.UTC(),
			}
		}
	}
	return d
}

// rejectMatchEnded diz a quem conecta que a partida acabou: match_ended (lido do banco) e o
// fechamento 4002, como o 4001 do lobby (rejectWithClose).
func rejectMatchEnded(conn *websocket.Conn, matchUUID uuid.UUID, end *appmatch.MatchEndState) {
	p := MatchEndedPayload{MatchID: matchUUID, GameEndAt: end.GameEndAt.UTC().Format(time.RFC3339)}
	if end.StoryAt != nil {
		p.StoryAt = storyPtr(*end.StoryAt)
	}
	rejectWithClose(conn, NewServerMessage(MsgTypeMatchEnded, p), closeCodeMatchEnded, closeTextMatchEnded)
}
