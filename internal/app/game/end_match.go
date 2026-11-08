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
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid end_match payload"))
		return
	}
	storyAt, err := clock.ParseStory(payload.StoryAt)
	if err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "storyAt must be 2006-01-02T15:04:05"))
		return
	}
	brief, err := matchsession.NormalizeBriefFinal(payload.BriefFinalDescription)
	if err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "briefFinalDescription is too long"))
		return
	}
	if r.deps.MatchEndRepo == nil {
		client.SendMessage(NewErrorMessage("game_error", "match end unavailable"))
		return
	}

	// clockPersistMu antes de r.mu (a ordem de persistClock): nenhum checkpoint do relógio
	// rodando grava entre o retrato do fim e a aplicação dele.
	r.clockPersistMu.Lock()
	defer r.clockPersistMu.Unlock()

	r.mu.Lock()
	sess := r.session
	if sess == nil {
		r.mu.Unlock()
		client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
		return
	}
	plan, err := sess.PlanEndMatch(storyAt, payload.ConfirmLongSkip, r.now())
	if err != nil {
		r.mu.Unlock()
		client.SendMessage(NewErrorMessage(clockErrorCode(err), err.Error()))
		return
	}
	data := r.matchEndDataLocked(sess, plan, brief)
	r.mu.Unlock()

	if err := r.deps.MatchEndRepo.PersistMatchEnd(context.Background(), data); err != nil {
		log.Printf("end_match FAILED — match %s NOT ended (session untouched): %v", r.matchUUID, err)
		client.SendMessage(NewErrorMessage("game_error", "could not end the match"))
		return
	}

	r.mu.Lock()
	// Mesmo instante real do plano: o fim aplicado é o que foi gravado.
	if _, err := sess.EndMatchAt(storyAt, payload.ConfirmLongSkip, plan.ClosedAt); err != nil {
		// O banco já diz encerrada; a sala fecha do mesmo jeito.
		log.Printf("end_match: match %s written as ended but the session refused the end: %v", r.matchUUID, err)
	}
	r.forgetRoundEndsWrittenLocked(data.UnwrittenRoundEnds)
	r.clearClockMergeLocked()
	r.state = RoomStateClosed
	clients := make([]*Client, 0, len(r.clients))
	for _, c := range r.clients {
		clients = append(clients, c)
	}
	r.mu.Unlock()

	msg, _ := json.Marshal(NewServerMessage(MsgTypeMatchEnded, MatchEndedPayload{
		MatchID: r.matchUUID, StoryAt: storyPtr(plan.StoryAt), GameEndAt: plan.ClosedAt.UTC().Format(time.RFC3339),
	}))
	// Pela faixa direta, não pelo canal broadcast: a sala para logo abaixo e o Run pode sair
	// antes de drená-lo. A WritePump esvazia a fila (match_ended) e só então escreve o 4002.
	for _, c := range clients {
		select {
		case c.send <- msg:
		default:
			log.Printf("dropping match_ended for slow client %s", c.userUUID)
		}
		c.CloseWith(closeCodeMatchEnded, closeTextMatchEnded)
	}
	r.Stop()
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
	d := appmatch.MatchEndData{
		MatchUUID: r.matchUUID, GameEndAt: plan.ClosedAt.UTC(), StoryAt: plan.StoryAt,
		BriefFinalDescription: brief,
		End:                   appmatch.RoundEnd{Scene: sc, Round: rd},
		UnwrittenRoundEnds:    r.unwrittenRoundEndsLocked(),
		Clock:                 plan.Clock,
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
