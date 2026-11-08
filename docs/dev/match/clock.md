# Relógio da história — desenho

Contrato do fio: [`../api/match-clock.md`](../api/match-clock.md). Regra de jogo:
[`../../game/tempo.md`](../../game/tempo.md). Spec:
`docs/superpowers/specs/2026-11-08-story-clock-design.md` (e `.pt-br.md`).

## Onde o relógio vive

- **Domínio:** `internal/domain/match/entity/clock/` (`Clock`, `Snapshot`, `Pause`, `LimitReason`,
  `PauseReason`, `LongSkip`, `RoundEquivalents`, `FormatStory`/`ParseStory`). Puro: o tempo real
  entra como argumento (`real time.Time`), nada lê `time.Now`. O `Clock` guarda `storyAt` e, rodando,
  `runningSince`; o valor corrente é `storyAt + (real − runningSince)` em segundos inteiros.
- **Sessão:** `MatchSession` é a dona (`Clock()`, `ClockSnapshot`, `SceneCloseMinimum`,
  `PlanEndMatch`/`EndMatchAt`). `ChangeScene`, `SetRoundMode`, `OpenNextAction`/`PullAction` e o
  fechamento de round movem o relógio; todos validam **antes** de mutar (uma recusa não deixa rastro).
- **Casos de uso:** `application/match/clock_commands.go` (`ClockCommandsUC`: confere o mestre,
  delega; a única regra daqui é `ErrClockPlayInRace`), `combat_summary.go` (`BuildCombatSummary`,
  sem I/O) e `init_match_session.go` (`loadOrCreateClock`).
- **Sala:** `internal/app/game/clock.go` e `end_match.go`. A sala segura `r.mu` para mutar e
  **grava fora dele**; `clockPersistMu` ordena retrato e escrita (ordem de lock: `clockPersistMu`,
  depois `r.mu`).
- **Gateway:** `internal/gateway/pg/clock/` (`match_clocks`, upsert de retrato). O fim da partida
  vive em `internal/gateway/pg/round/persist_match_end.go` (`PersistMatchEnd`, `ReadMatchEnd`) —
  **no pacote `round` de propósito, para evitar ciclo de import**: precisa do upsert do par
  cena/round (`ensureSceneAndRound`) e recebe tipos de `application/match`, que já importa
  `gateway/pg/match`.

## Tempo ingênuo

`TIMESTAMP`/`DATE` no banco, `time.Time` em UTC tratado como relógo de parede no Go, string
`"2006-01-02T15:04:05"` no fio. O parse recusa fração, `Z` e offset. O front nunca passa isso por
`Date` (gotcha nº 1; ver contrato §1).

## Transações

Regra da casa: **um comando do mestre, uma transação**, e nada que acontece dentro de um turno
aberto é gravado antes do fechamento dele.

- O relógio vai na transação do fechamento de turno/round (`TurnCloseData`, `PersistRoundClose`) e
  do `change_scene` (`CloseSceneAndRound` recebe o retrato do relógio).
- Os comandos de relógio têm a sua: `ClockRepo.Save` grava retrato + evento juntos. O retrato é um
  upsert, então uma gravação que falha é curada pela próxima; o evento perdido fica no log.
- `match_events` exige `scene_uuid`/`round_uuid`: o evento de relógio garante antes as linhas do
  par ativo (`ensureSceneAndRoundRows`); a pausa da troca de cena pertence ao par que estava
  rodando (`pendingClockEvent.pair`).
- **`end_match`** (`PersistMatchEnd`, uma transação): campos de fim da partida
  (`game_end_at`, `story_end_at`, `brief_final_description`; recusa partida já encerrada), fins de
  round ainda não gravados + cena/round ativos fechados, relógio parado no instante final,
  `clockRealTime` da pausa, `campaigns.story_current_at` e `story_current_at` das fichas
  participantes (ambos com `GREATEST`, nunca regridem). A sala **grava antes de aplicar** o fim na
  sessão (`PlanEndMatch` calcula sem mutar; `EndMatchAt` aplica no mesmo instante real): se a
  gravação falha, a sessão fica intacta. Depois: `match_ended` → toda conexão fecha com 4002
  `"match ended"` → `RoomStateClosed`. Uma partida encerrada recusa qualquer conexão nova
  (`ReadMatchEnd` no handler, antes de criar sala).

## Fusão de ajustes

`adjust_clock`/`set_clock` guardam em `Room.lastAdjust` a marca do último `clockAdjusted`
gravado (uuid do evento, instante real, `from` da sequência, soma dos deltas). Um ajuste até
`clockMergeWindow` (10 s) depois do anterior atualiza o **payload** da mesma linha de
`match_events` (`ClockEvent.MergeInto`) em vez de inserir; `deltaSeconds` soma os deltas **pedidos**
(o tempo real entre os cliques não entra). Qualquer outro evento de relógio, turno, cena ou regime
chama `clearClockMergeLocked`, que também incrementa `mergeGen`: um ajuste cuja gravação terminou
depois de um fechamento não reabre a janela. Gravação que falhou também zera a marca (não se funde
em linha inexistente).

## Checkpoint e reinício

Com o relógio rodando, o `Run` da sala tem um ticker (`defaultClockCheckpointEvery` = 60 s) que
chama `checkpointClock` — gravação assíncrona, sem evento nem mensagem. Se o servidor cai, a sala
que renasce (`loadOrCreateClock`) encontra `running_since` no banco e **pausa em `running_since`**
(o instante do checkpoint, cujo `story_at` já é o valor naquele momento) com motivo `restart`;
perde no máximo um intervalo, nunca soma o tempo parado.

## Mestre sai

`unregister` → se a conexão removida é a do mestre → `pauseClockMasterLeft` (pausa imediata, grava
`clockRealTime`, manda `clock_changed` direto às conexões que restam, sem passar pelo canal
`broadcast` que o próprio `Run` drena). `removeClient` devolve `false` para uma conexão
**substituída** pela mesma conta, que portanto não pausa nada.

## Duas pistas e o `seq`

As mensagens de relógio saem por `broadcastInOrder` (uma goroutine, ordem dada), mas a pista de
`broadcast` e a direta não têm ordem entre si. Por isso o `seq`: `clockChangedData` incrementa
`clockSeq` e tira o retrato na **mesma** seção crítica, então o número ordena retratos. O contador
parte de `time.Now().UnixMicro()` ao nascer a sala, para nunca recuar entre salas da mesma partida
(abaixo de 2^53, exato em JS).

## Hiato e conversão

`rounds.counted` (abriu turno no disputado) e `rounds.drained` (fechou com a fila vazia) são fatos;
quem decide o hiato é `BuildCombatSummary`, olhando o round seguinte com turno. A conversão para
status é `clock.RoundEquivalents(from, to)` (= segundos). Nada de status é implementado aqui.
