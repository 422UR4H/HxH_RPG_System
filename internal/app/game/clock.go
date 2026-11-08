package game

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/clock"
	roundentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/round"
	sceneentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/scene"
	turnentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/turn"
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
// próximo ajuste atualiza, at o instante real dele, from o início da sequência fundida e
// deltaSum a soma dos deltas que o mestre pediu (o tempo real que correu entre eles não entra).
type clockAdjustMark struct {
	eventUUID uuid.UUID
	at        time.Time
	from      time.Time
	deltaSum  int64
}

// pendingClockEvent é o evento que acompanha um Save do relógio. mergeInto != uuid.Nil
// atualiza essa linha em vez de inserir uma nova.
type pendingClockEvent struct {
	uuid      uuid.UUID
	kind      matchevent.Kind
	payload   any
	mergeInto uuid.UUID
	// pair, quando dado, é o par (cópias) a que o evento pertence em vez do par ativo — a pausa
	// da troca de cena é do par que estava rodando, já fechado.
	pair *appmatch.RoundEnd
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
	case errors.Is(err, matchsession.ErrMatchInBattle):
		return "match_in_battle"
	case errors.Is(err, matchsession.ErrMatchEnded):
		return "match_already_ended"
	case errors.Is(err, matchsession.ErrBriefFinalTooLong):
		return "invalid_payload"
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
	p := clockPayloadOf(sess.ClockSnapshot(r.now()))
	p.Seq = r.clockSeq
	return p
}

// clockPayloadLocked é o estado do relógio agora, com o seq corrente (sem avançá-lo).
// O chamador segura r.mu (leitura ou escrita).
//
// clockChangedData monta o clock_changed com o estado de agora e um seq novo: o retrato e o
// número saem da mesma seção crítica, então o seq ordena os RETRATOS — o cliente descarta um
// seq menor que o maior que aplicou (as duas faixas de envio não têm ordem entre si). Toma
// r.mu. O retrato é de uma seção crítica posterior à da mudança; é seguro porque os comandos
// do mestre são serializados pela única leitura dele, e o que corre em paralelo (checkpoint,
// saída do mestre) só pode deixar o retrato mais novo, nunca mais velho.
func (r *Room) clockChangedData(sess *matchsession.MatchSession, change ClockChangePayload) []byte {
	r.mu.Lock()
	r.clockSeq++
	state := r.clockPayloadLocked(sess)
	r.mu.Unlock()
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
// mergeGen conta os fechamentos, para um ajuste cuja gravação terminou depois de um deles
// não reabrir a janela.
func (r *Room) clearClockMergeLocked() {
	r.lastAdjust = nil
	r.mergeGen++
}

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
	return ClockChangePayload{
		Kind: "pause", From: storyPtr(p.From), To: storyPtr(p.To), Reason: string(p.Reason),
		ElapsedSeconds: int64(p.Elapsed / time.Second),
	}
}

// clockSnapshotForWriteLocked é o retrato do relógio que vai a uma gravação de match_clocks, e
// conta essa gravação em clockWriteGen: um checkpoint tirado antes dele não grava mais
// (writeClockCheckpoint). Todo caminho que entrega o relógio a uma escrita passa por aqui. O
// chamador segura r.mu (leitura ou escrita).
func (r *Room) clockSnapshotForWriteLocked(sess *matchsession.MatchSession) clock.Snapshot {
	r.clockWriteGen.Add(1)
	return sess.ClockSnapshot(r.now())
}

// persistClock grava o relógio e, se ev != nil, o evento — numa transação (ClockRepo.Save). O
// evento precisa das linhas da cena e do round ativos: elas são garantidas antes; sem elas o
// evento é perdido (logado) e só o relógio vai. Falha é logada e engolida: o relógio em memória
// manda e o próximo Save cura. O chamador NÃO pode segurar r.mu.
//
// O que é ordenado: clockPersistMu envolve retrato e escrita, então os Saves desta função (comando
// do mestre, saída do mestre) e o checkpoint (writeClockCheckpoint) chegam ao banco na ordem dos
// retratos. As transações que gravam o relógio junto de outra coisa (PersistTurnClose,
// PersistRoundClose, CloseSceneAndRound) não tomam clockPersistMu — vêm do laço de leitura do
// mestre, como os comandos do relógio, e entre si já são serializadas por ele; contra o checkpoint
// valem pelo clockWriteGen (o checkpoint mais velho que elas desiste). PersistMatchEnd segura
// clockPersistMu (end_match.go).
//
// Devolve true quando o evento (ev != nil) foi gravado — a fusão só aponta para linha que existe.
// O retrato é tirado aqui, numa seção crítica posterior à da mudança: seguro pelo mesmo motivo
// de clockChangedData (comandos do mestre serializados pela leitura única dele).
func (r *Room) persistClock(sess *matchsession.MatchSession, ev *pendingClockEvent, why string) bool {
	if r.deps.ClockRepo == nil || sess == nil {
		return false
	}
	r.clockPersistMu.Lock()
	defer r.clockPersistMu.Unlock()

	r.mu.RLock()
	snap := r.clockSnapshotForWriteLocked(sess)
	sc, rd := snapshotSceneAndRound(sess.GetActiveScene(), sess.GetActiveRound())
	r.mu.RUnlock()

	var clockEv *appmatch.ClockEvent
	if ev != nil {
		payload, err := json.Marshal(ev.payload)
		pairRows := r.ensureClockEventPair
		if ev.pair != nil {
			sc, rd = ev.pair.Scene, ev.pair.Round
		} else {
			pairRows = r.ensureSceneAndRoundRows
		}
		switch {
		case err != nil:
			log.Printf("%s: clock event %s of match %s NOT recorded (payload): %v", why, ev.kind, r.matchUUID, err)
		case !pairRows(sess, sc, rd, why):
			log.Printf("%s: clock event %s of match %s NOT recorded (no scene/round rows)", why, ev.kind, r.matchUUID)
		default:
			clockEv = &appmatch.ClockEvent{
				Event: matchevent.Event{
					UUID: ev.uuid, MatchUUID: r.matchUUID, SceneUUID: sc.GetID(), RoundUUID: rd.GetID(),
					Kind: ev.kind, Payload: payload, CreatedAt: r.now().UTC(),
				},
				MergeInto: ev.mergeInto,
			}
		}
	}
	if err := r.deps.ClockRepo.Save(context.Background(), r.matchUUID, snap, clockEv); err != nil {
		log.Printf("%s FAILED — clock of match %s NOT written (kept in memory for the next save): %v", why, r.matchUUID, err)
		return false
	}
	return clockEv != nil
}

// ensureClockEventPair garante as linhas de um par que já não é o ativo (o evento é dele): o
// mesmo upsert idempotente, direto, sem tocar as marcas da sessão nem os fins não gravados —
// um par fechado entra fechado. O chamador NÃO pode segurar r.mu.
func (r *Room) ensureClockEventPair(
	_ *matchsession.MatchSession, sc *sceneentity.Scene, rd *roundentity.Round, why string,
) bool {
	if r.deps.RoundRepo == nil || sc == nil || rd == nil {
		return false
	}
	if err := r.deps.RoundRepo.EnsureSceneAndRound(context.Background(), r.matchUUID, sc, rd); err != nil {
		log.Printf("%s FAILED — scene %s / round %s of match %s NOT written: %v", why, sc.GetID(), rd.GetID(), r.matchUUID, err)
		return false
	}
	return true
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
		log.Printf("set_clock_start: read story start date of match %s: %v", r.matchUUID, err)
		client.SendMessage(NewErrorMessage("game_error", "could not read the match start date"))
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
	var mv clockMove
	if err == nil {
		mv = r.noteClockMoveLocked(from, to, real)
	}
	r.mu.Unlock()
	r.finishClockMove(client, sess, "adjust", from, to, mv, err)
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
	var mv clockMove
	if err == nil {
		mv = r.noteClockMoveLocked(from, to, real)
	}
	r.mu.Unlock()
	r.finishClockMove(client, sess, "set", from, to, mv, err)
}

// clockMove é o clockAdjusted a gravar e a marca de fusão que ele deixa se for gravado. gen é
// o mergeGen do instante da mudança.
type clockMove struct {
	ev   *pendingClockEvent
	mark clockAdjustMark
	gen  uint64
}

// noteClockMoveLocked decide a fusão de um ajuste (adjust_clock ou set_clock): dentro da janela,
// a mesma linha, com o from da sequência e a soma dos deltas pedidos (set_clock entra com
// to − from no próprio instante). Não abre a janela: isso é de finishClockMove, depois que o
// evento foi gravado. O chamador segura r.mu.
func (r *Room) noteClockMoveLocked(from, to, real time.Time) clockMove {
	ev := &pendingClockEvent{uuid: uuid.New(), kind: matchevent.KindClockAdjusted}
	delta := int64(to.Sub(from) / time.Second)
	seqFrom, sum := from, delta
	if last := r.lastAdjust; last != nil && real.Sub(last.at) <= clockMergeWindow {
		ev.uuid, ev.mergeInto, seqFrom, sum = last.eventUUID, last.eventUUID, last.from, last.deltaSum+delta
	}
	ev.payload = map[string]any{
		"from":         clock.FormatStory(seqFrom),
		"to":           clock.FormatStory(to),
		"deltaSeconds": sum,
	}
	return clockMove{
		ev:   ev,
		mark: clockAdjustMark{eventUUID: ev.uuid, at: real, from: seqFrom, deltaSum: sum},
		gen:  r.mergeGen,
	}
}

func (r *Room) finishClockMove(
	client *Client, sess *matchsession.MatchSession, kind string, from, to time.Time,
	mv clockMove, err error,
) {
	if err != nil {
		client.SendMessage(NewErrorMessage(clockErrorCode(err), err.Error()))
		return
	}
	recorded := r.persistClock(sess, mv.ev, "clock "+kind)
	r.mu.Lock()
	switch {
	case !recorded:
		// Evento não gravado: o próximo ajuste não pode fundir numa linha que não existe.
		r.lastAdjust = nil
	case mv.gen == r.mergeGen:
		mark := mv.mark
		r.lastAdjust = &mark
	}
	r.mu.Unlock()
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
// a mensagem para o chamador mandar na ordem dele; nil sem pausa. pair != nil atribui o
// clockRealTime a esse par (o que estava rodando) em vez do ativo. O chamador NÃO segura r.mu.
func (r *Room) announceClockPause(sess *matchsession.MatchSession, p *clock.Pause, pair *appmatch.RoundEnd, why string) []byte {
	if p == nil {
		return nil
	}
	ev := pauseEvent(p)
	ev.pair = pair
	r.persistClock(sess, ev, why)
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
// último checkpoint e perde no máximo um intervalo (spec §3.4). O retrato e a geração saem aqui,
// na goroutine do Run; a escrita vai para outra goroutine (não segura a sala).
func (r *Room) checkpointClock() {
	if r.deps.ClockRepo == nil {
		return
	}
	r.mu.RLock()
	sess := r.session
	running := sess != nil && sess.Clock().Running()
	var snap clock.Snapshot
	gen := r.clockWriteGen.Load()
	if running {
		snap = sess.ClockSnapshot(r.now())
	}
	r.mu.RUnlock()
	if running {
		go r.writeClockCheckpoint(snap, gen)
	}
}

// writeClockCheckpoint grava o retrato snap, tirado na geração gen — a menos que outro retrato
// tenha ido a uma gravação depois dele (clockWriteGen mudou): um fechamento de turno, uma troca
// de cena ou uma pausa já levou um relógio mais novo, e o checkpoint mais velho o sobrescreveria.
// Devolve true se gravou. O chamador NÃO pode segurar r.mu.
//
// Sobra uma janela: um fechamento cujo retrato sai depois da comparação e cuja transação termina
// antes deste Save. Fechá-la exigiria clockPersistMu em PersistTurnClose/PersistRoundClose/
// CloseSceneAndRound; a próxima gravação do relógio cura.
func (r *Room) writeClockCheckpoint(snap clock.Snapshot, gen uint64) bool {
	r.clockPersistMu.Lock()
	defer r.clockPersistMu.Unlock()
	if r.clockWriteGen.Load() != gen {
		return false
	}
	if err := r.deps.ClockRepo.Save(context.Background(), r.matchUUID, snap, nil); err != nil {
		log.Printf("clock checkpoint FAILED — clock of match %s NOT written (kept in memory for the next save): %v", r.matchUUID, err)
		return false
	}
	return true
}

// handleGetCombatSummary responde ao mestre, só a ele, o resumo do combate da cena ativa (spec
// §4): os rounds fechados vêm do banco (lidos fora do r.mu), o round ativo da memória, com os
// turnos que já tem.
func (r *Room) handleGetCombatSummary(client *Client) {
	if !r.IsMaster(client.userUUID) {
		client.SendMessage(NewErrorMessage("forbidden", ErrNotMaster.Error()))
		return
	}
	r.mu.RLock()
	sess := r.session
	var sceneID uuid.UUID
	battle := false
	if sess != nil {
		sc := sess.GetActiveScene()
		sceneID, battle = sc.GetID(), sc.GetCategory() == enum.Battle
	}
	r.mu.RUnlock()
	switch {
	case sess == nil:
		client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
		return
	case !battle:
		client.SendMessage(NewErrorMessage("not_battle_scene", "the active scene is not a battle"))
		return
	case r.deps.CombatSummaryRepo == nil:
		client.SendMessage(NewErrorMessage("game_error", "combat summary unavailable"))
		return
	}

	rounds, err := r.deps.CombatSummaryRepo.ListSceneRoundsForSummary(context.Background(), sceneID)
	if err != nil {
		log.Printf("get_combat_summary: read rounds of scene %s (match %s): %v", sceneID, r.matchUUID, err)
		client.SendMessage(NewErrorMessage("game_error", "could not read the combat rounds"))
		return
	}

	r.mu.RLock()
	sc := sess.GetActiveScene()
	if sc.GetID() != sceneID {
		// a cena mudou durante a leitura: o resumo pedido não existe mais
		r.mu.RUnlock()
		client.SendMessage(NewErrorMessage("not_battle_scene", "the active scene changed"))
		return
	}
	active := summaryRoundOf(sess.GetActiveRound())
	minimum := sess.SceneCloseMinimum()
	r.mu.RUnlock()

	// O round ativo já é linha desde que nasceu, mas aberto: a consulta só traz fechados. Um
	// round que fechou em memória e cuja gravação falhou ainda aparece aberto no banco e falta.
	closed := rounds[:0:0]
	for _, rd := range rounds {
		if rd.UUID != active.UUID {
			closed = append(closed, rd)
		}
	}
	sum := appmatch.BuildCombatSummary(closed, active, minimum)
	out := CombatSummaryPayload{
		SceneID:           sceneID,
		CountedRounds:     sum.CountedRounds,
		CountedSeconds:    sum.CountedSeconds,
		ActiveRoundCounts: sum.ActiveRoundCounts,
		MinimumStoryAt:    clock.FormatStory(sum.MinimumStoryAt),
		Hiatuses:          make([]HiatusPayload, 0, len(sum.Hiatuses)),
	}
	for _, h := range sum.Hiatuses {
		out.Hiatuses = append(out.Hiatuses, HiatusPayload{Before: hiatusTurnOf(h.Before), After: hiatusTurnOf(h.After)})
	}
	client.SendMessage(NewServerMessage(MsgTypeCombatSummary, out))
}

// summaryRoundOf lê o round ativo para o resumo. O chamador segura r.mu.
func summaryRoundOf(rd *roundentity.Round) appmatch.SummaryRound {
	out := appmatch.SummaryRound{UUID: rd.GetID(), Mode: rd.GetMode(), Counted: rd.IsCounted(), Drained: rd.IsDrained()}
	turns := rd.Turns()
	if len(turns) == 0 {
		return out
	}
	out.First, out.Last = summaryTurnOfTurn(turns[0]), summaryTurnOfTurn(turns[len(turns)-1])
	return out
}

func summaryTurnOfTurn(t *turnentity.Turn) *appmatch.SummaryTurn {
	act := t.ActionRef()
	return &appmatch.SummaryTurn{TurnUUID: t.GetID(), ActorUUID: act.GetActorID(), ActionKind: act.TypeName()}
}

func hiatusTurnOf(h appmatch.HiatusTurn) HiatusTurn {
	return HiatusTurn{RoundNumber: h.RoundNumber, TurnID: h.TurnUUID, ActorID: h.ActorUUID, ActionKind: h.ActionKind}
}
