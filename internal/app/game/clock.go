package game

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/matchsession"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/matchevent"
	"github.com/google/uuid"
)

// clockMergeWindow: ajustes seguidos dentro desta janela, sem outro evento de relógio nem
// turno/cena/regime no meio, viram uma linha só do histórico (spec §6).
const clockMergeWindow = 10 * time.Second

// defaultClockCheckpointEvery é o intervalo do checkpoint do relógio rodando (spec §3.4).
const defaultClockCheckpointEvery = 60 * time.Second

// clockAdjustMark é o último clockAdjusted gravado, para a fusão: eventUUID é a linha que o
// próximo ajuste atualiza, at o instante real dele, from o início da sequência fundida.
type clockAdjustMark struct {
	eventUUID uuid.UUID
	at        time.Time
	from      time.Time
}

// pendingClockEvent é o evento que acompanha um Save do relógio. mergeInto != uuid.Nil
// atualiza essa linha em vez de inserir uma nova.
type pendingClockEvent struct {
	uuid      uuid.UUID
	kind      matchevent.Kind
	payload   any
	mergeInto uuid.UUID
}

// clockErrorCode traduz os erros do relógio para o código do fio; o resto é game_error.
func clockErrorCode(err error) string {
	switch {
	case errors.Is(err, clock.ErrNotSet):
		return "clock_not_set"
	case errors.Is(err, clock.ErrAlreadySet):
		return "clock_already_set"
	case errors.Is(err, clock.ErrBelowLimit):
		return "clock_below_limit"
	case errors.Is(err, clock.ErrLongSkipUnconfirmed):
		return "clock_long_skip_unconfirmed"
	case errors.Is(err, clock.ErrAlreadyRunning):
		return "clock_already_running"
	case errors.Is(err, clock.ErrZeroDelta):
		return "invalid_payload"
	case errors.Is(err, appmatch.ErrClockPlayInRace):
		return "clock_play_in_race"
	case errors.Is(err, appmatch.ErrNotMatchMaster):
		return "forbidden"
	default:
		return "game_error"
	}
}

// now é o tempo real do relógio: RoomDeps.Now, ou time.Now sem ele.
func (r *Room) now() time.Time {
	if r.nowFn != nil {
		return r.nowFn()
	}
	return time.Now()
}

func storyPtr(t time.Time) *string {
	s := clock.FormatStory(t)
	return &s
}

func clockPayloadOf(s clock.Snapshot) ClockPayload {
	p := ClockPayload{Running: s.Running, LimitReason: string(s.LimitReason)}
	if s.StoryAt != nil {
		p.StoryAt = storyPtr(*s.StoryAt)
	}
	if s.LimitAt != nil {
		p.LimitAt = storyPtr(*s.LimitAt)
	}
	return p
}

// clockPayloadLocked é o estado do relógio agora. O chamador segura r.mu (leitura ou escrita).
func (r *Room) clockPayloadLocked(sess *matchsession.MatchSession) ClockPayload {
	return clockPayloadOf(sess.ClockSnapshot(r.now()))
}

// clockChangedData monta o clock_changed com o estado de agora. Toma r.mu.RLock.
func (r *Room) clockChangedData(sess *matchsession.MatchSession, change ClockChangePayload) []byte {
	r.mu.RLock()
	state := r.clockPayloadLocked(sess)
	r.mu.RUnlock()
	data, _ := json.Marshal(NewServerMessage(MsgTypeClockChanged, ClockChangedPayload{Clock: state, Change: change}))
	return data
}

// broadcastInOrder manda as mensagens à mesa, nesta ordem, por uma goroutine só (o mesmo
// desvio que os demais braços fazem para não segurar a leitura do cliente); nil é pulado.
func (r *Room) broadcastInOrder(msgs ...[]byte) {
	go func() {
		for _, m := range msgs {
			if m != nil {
				r.broadcast <- m
			}
		}
	}()
}

// clearClockMergeLocked fecha a janela de fusão dos ajustes. Chamado sob r.mu por todo
// evento de relógio que não é ajuste e por turno, cena e regime.
func (r *Room) clearClockMergeLocked() { r.lastAdjust = nil }

func pauseEvent(p *clock.Pause) *pendingClockEvent {
	return &pendingClockEvent{
		uuid: uuid.New(),
		kind: matchevent.KindClockRealTime,
		payload: map[string]any{
			"from":           clock.FormatStory(p.From),
			"to":             clock.FormatStory(p.To),
			"elapsedSeconds": int64(p.Elapsed / time.Second),
			"reason":         string(p.Reason),
		},
	}
}

func pauseChange(p *clock.Pause) ClockChangePayload {
	return ClockChangePayload{Kind: "pause", From: storyPtr(p.From), To: storyPtr(p.To), Reason: string(p.Reason)}
}

// persistClock grava o checkpoint do relógio e, se ev != nil, o evento — numa transação
// (ClockRepo.Save). O evento precisa das linhas da cena e do round ativos: elas são
// garantidas antes; sem elas o evento é perdido (logado) e só o checkpoint vai. clockPersistMu
// envolve retrato e escrita, para dois Saves de goroutines diferentes (comando do mestre,
// checkpoint, saída do mestre) chegarem na ordem dos retratos. Falha é logada e engolida: o
// relógio em memória manda e o próximo Save cura. O chamador NÃO pode segurar r.mu.
func (r *Room) persistClock(sess *matchsession.MatchSession, ev *pendingClockEvent, why string) {
	if r.deps.ClockRepo == nil || sess == nil {
		return
	}
	r.clockPersistMu.Lock()
	defer r.clockPersistMu.Unlock()

	r.mu.RLock()
	snap := sess.ClockSnapshot(r.now())
	sc, rd := snapshotSceneAndRound(sess.GetActiveScene(), sess.GetActiveRound())
	r.mu.RUnlock()

	var clockEv *appmatch.ClockEvent
	if ev != nil {
		payload, err := json.Marshal(ev.payload)
		switch {
		case err != nil:
			log.Printf("%s: clock event %s of match %s NOT recorded (payload): %v", why, ev.kind, r.matchUUID, err)
		case !r.ensureSceneAndRoundRows(sess, sc, rd, why):
			log.Printf("%s: clock event %s of match %s NOT recorded (no scene/round rows)", why, ev.kind, r.matchUUID)
		default:
			clockEv = &appmatch.ClockEvent{
				Event: matchevent.Event{
					UUID: ev.uuid, MatchUUID: r.matchUUID, SceneUUID: sc.GetID(), RoundUUID: rd.GetID(),
					Kind: ev.kind, Payload: payload, CreatedAt: time.Now().UTC(),
				},
				MergeInto: ev.mergeInto,
			}
		}
	}
	if err := r.deps.ClockRepo.Save(context.Background(), r.matchUUID, snap, clockEv); err != nil {
		log.Printf("%s FAILED — clock of match %s NOT written (kept in memory for the next save): %v", why, r.matchUUID, err)
	}
}

// sessionForClock devolve a sessão, ou responde ao cliente e devolve nil.
func (r *Room) sessionForClock(client *Client) *matchsession.MatchSession {
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
		return nil
	}
	if r.deps.ClockUC == nil {
		client.SendMessage(NewErrorMessage("game_error", "story clock unavailable"))
		return nil
	}
	r.mu.RLock()
	sess := r.session
	r.mu.RUnlock()
	if sess == nil {
		client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
		return nil
	}
	return sess
}

// parseTimeOfDay lê "HH:MM" ou "HH:MM:SS" como deslocamento desde a meia-noite.
func parseTimeOfDay(s string) (time.Duration, bool) {
	layout := "15:04:05"
	if strings.Count(s, ":") == 1 {
		layout = "15:04"
	}
	t, err := time.Parse(layout, s)
	if err != nil || t.Format(layout) != s {
		return 0, false
	}
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute +
		time.Duration(t.Second())*time.Second, true
}

func (r *Room) handleSetClockStart(client *Client, raw json.RawMessage) {
	var payload SetClockStartPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid set_clock_start payload"))
		return
	}
	sess := r.sessionForClock(client)
	if sess == nil {
		return
	}
	tod, ok := parseTimeOfDay(payload.Time)
	if !ok {
		client.SendMessage(NewErrorMessage("invalid_payload", `time must be "HH:MM" or "HH:MM:SS"`))
		return
	}
	// A data da história vem do banco: lida fora do r.mu.
	date, err := r.deps.ClockUC.StoryDate(context.Background(), r.matchUUID)
	if err != nil {
		client.SendMessage(NewErrorMessage("game_error", err.Error()))
		return
	}
	at := date.Add(tod)

	r.mu.Lock()
	err = r.deps.ClockUC.Start(sess, r.masterUUID, client.userUUID, at)
	if err == nil {
		r.clearClockMergeLocked()
	}
	r.mu.Unlock()
	if err != nil {
		client.SendMessage(NewErrorMessage(clockErrorCode(err), err.Error()))
		return
	}
	r.persistClock(sess, &pendingClockEvent{
		uuid: uuid.New(), kind: matchevent.KindClockStarted,
		payload: map[string]string{"at": clock.FormatStory(at)},
	}, "set_clock_start")
	r.broadcastInOrder(r.clockChangedData(sess, ClockChangePayload{Kind: "start", To: storyPtr(at)}))
}

func (r *Room) handleAdjustClock(client *Client, raw json.RawMessage) {
	var payload AdjustClockPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid adjust_clock payload"))
		return
	}
	sess := r.sessionForClock(client)
	if sess == nil {
		return
	}
	r.mu.Lock()
	real := r.now()
	from, to, err := r.deps.ClockUC.Adjust(sess, r.masterUUID, client.userUUID,
		time.Duration(payload.DeltaSeconds)*time.Second, payload.ConfirmLongSkip, real)
	var ev *pendingClockEvent
	if err == nil {
		ev = r.noteClockMoveLocked(from, to, real)
	}
	r.mu.Unlock()
	r.finishClockMove(client, sess, "adjust", from, to, ev, err)
}

func (r *Room) handleSetClock(client *Client, raw json.RawMessage) {
	var payload SetClockPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "invalid set_clock payload"))
		return
	}
	sess := r.sessionForClock(client)
	if sess == nil {
		return
	}
	to, perr := clock.ParseStory(payload.StoryAt)
	if perr != nil {
		client.SendMessage(NewErrorMessage("invalid_payload", "storyAt must be 2006-01-02T15:04:05"))
		return
	}
	r.mu.Lock()
	real := r.now()
	from, err := r.deps.ClockUC.Set(sess, r.masterUUID, client.userUUID, to, payload.ConfirmLongSkip, real)
	var ev *pendingClockEvent
	if err == nil {
		ev = r.noteClockMoveLocked(from, to, real)
	}
	r.mu.Unlock()
	r.finishClockMove(client, sess, "set", from, to, ev, err)
}

// noteClockMoveLocked decide a fusão de um ajuste (adjust_clock ou set_clock) e devolve o
// clockAdjusted a gravar: dentro da janela, a mesma linha, com o from da sequência e o
// delta acumulado. O chamador segura r.mu.
func (r *Room) noteClockMoveLocked(from, to, real time.Time) *pendingClockEvent {
	ev := &pendingClockEvent{uuid: uuid.New(), kind: matchevent.KindClockAdjusted}
	seqFrom := from
	if last := r.lastAdjust; last != nil && real.Sub(last.at) <= clockMergeWindow {
		ev.uuid, ev.mergeInto, seqFrom = last.eventUUID, last.eventUUID, last.from
	}
	r.lastAdjust = &clockAdjustMark{eventUUID: ev.uuid, at: real, from: seqFrom}
	ev.payload = map[string]any{
		"from":         clock.FormatStory(seqFrom),
		"to":           clock.FormatStory(to),
		"deltaSeconds": int64(to.Sub(seqFrom) / time.Second),
	}
	return ev
}

func (r *Room) finishClockMove(
	client *Client, sess *matchsession.MatchSession, kind string, from, to time.Time,
	ev *pendingClockEvent, err error,
) {
	if err != nil {
		client.SendMessage(NewErrorMessage(clockErrorCode(err), err.Error()))
		return
	}
	r.persistClock(sess, ev, "clock "+kind)
	r.broadcastInOrder(r.clockChangedData(sess, ClockChangePayload{
		Kind: kind, From: storyPtr(from), To: storyPtr(to),
		DeltaSeconds: int64(to.Sub(from) / time.Second),
	}))
}

func (r *Room) handlePlayClock(client *Client) {
	sess := r.sessionForClock(client)
	if sess == nil {
		return
	}
	r.mu.Lock()
	err := r.deps.ClockUC.Play(sess, r.masterUUID, client.userUUID, r.now())
	if err == nil {
		r.clearClockMergeLocked()
	}
	r.mu.Unlock()
	if err != nil {
		client.SendMessage(NewErrorMessage(clockErrorCode(err), err.Error()))
		return
	}
	// Sem evento: o tempo real vira clockRealTime na pausa. O checkpoint grava o "rodando".
	r.persistClock(sess, nil, "play_clock")
	r.broadcastInOrder(r.clockChangedData(sess, ClockChangePayload{Kind: "play"}))
}

func (r *Room) handlePauseClock(client *Client) {
	sess := r.sessionForClock(client)
	if sess == nil {
		return
	}
	r.mu.Lock()
	p, err := r.deps.ClockUC.Pause(sess, r.masterUUID, client.userUUID, r.now())
	if err == nil && p != nil {
		r.clearClockMergeLocked()
	}
	r.mu.Unlock()
	if err != nil {
		client.SendMessage(NewErrorMessage(clockErrorCode(err), err.Error()))
		return
	}
	if p == nil {
		return // já estava parado: nada mudou
	}
	r.persistClock(sess, pauseEvent(p), "pause_clock")
	r.broadcastInOrder(r.clockChangedData(sess, pauseChange(p)))
}

// announceClockPause grava e prepara o anúncio de uma pausa automática (regime, cena). Devolve
// a mensagem para o chamador mandar na ordem dele; nil sem pausa. O chamador NÃO segura r.mu.
func (r *Room) announceClockPause(sess *matchsession.MatchSession, p *clock.Pause, why string) []byte {
	if p == nil {
		return nil
	}
	r.persistClock(sess, pauseEvent(p), why)
	return r.clockChangedData(sess, pauseChange(p))
}

// pauseClockMasterLeft pausa o relógio quando a conexão do mestre sai (spec §3.4) — uma
// conexão substituída por outra da mesma conta não chega aqui (removeClient devolve false).
// Roda na goroutine do Run: manda direto aos clientes que restam, sem passar pelo canal
// broadcast que o próprio Run drena.
func (r *Room) pauseClockMasterLeft() {
	r.mu.Lock()
	sess := r.session
	var p *clock.Pause
	if sess != nil {
		p = sess.Clock().Pause(clock.PauseMasterLeft, r.now())
		if p != nil {
			r.clearClockMergeLocked()
		}
	}
	r.mu.Unlock()
	if p == nil {
		return
	}
	r.persistClock(sess, pauseEvent(p), "master_left")
	data := r.clockChangedData(sess, pauseChange(p))
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		select {
		case c.send <- data:
		default:
		}
	}
}

// checkpointClock grava o relógio rodando (sem evento, sem anúncio): um restart o pausa no
// último checkpoint e perde no máximo um intervalo (spec §3.4).
func (r *Room) checkpointClock() {
	if r.deps.ClockRepo == nil {
		return
	}
	r.mu.RLock()
	sess := r.session
	running := sess != nil && sess.Clock().Running()
	r.mu.RUnlock()
	if running {
		r.persistClock(sess, nil, "clock checkpoint")
	}
}
