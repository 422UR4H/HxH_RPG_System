# Match / Relógio da história — Protocolo WebSocket e campos REST

**Status:** implementado (back). Contrato de handoff para o front. Cada nome, campo e código
abaixo foi conferido contra `internal/app/game/message.go`, `clock.go`, `end_match.go`,
`room.go` e `handler.go`. **Se achar divergência, o bug é do contrato**: corrija este arquivo.

Complementa [`match-combat-ws.md`](match-combat-ws.md) (envelope, `error`, `match_full_state`) e
[`match-history.md`](match-history.md). A regra de jogo está em
[`../../game/tempo.md`](../../game/tempo.md); o desenho e o porquê, em
[`../match/clock.md`](../match/clock.md); o spec, em
`docs/superpowers/specs/2026-11-08-story-clock-design.pt-br.md`.

**Servidor:** `cmd/game/` · mesmo socket e mesmo envelope de `match-combat-ws.md` §1.

---

## 1. ⚠️ O tempo da história é ingênuo — leia antes de tudo

Todo instante **da história** (`storyAt`, `limitAt`, `from`, `to`, `minimumStoryAt`,
`storyFinishedAt`…) trafega como string `"2006-01-02T15:04:05"` (ex.: `"2000-01-01T08:30:00"`):

- **sem `Z`, sem offset, sem fração de segundo.** É relógio de parede do mundo fictício.
- O servidor **recusa** (`invalid_payload`) qualquer `storyAt` que não seja exatamente esse formato.
- ❌ **Nunca** faça `new Date(str)` / `Date.parse(str)` nem `toISOString()` nessas strings: o
  navegador aplicaria o fuso do usuário e a hora da história mudaria sozinha. Quebre a string em
  partes (`YYYY`, `MM`, `DD`, `hh`, `mm`, `ss`) com um util próprio e formate a partir delas. Para
  aritmética (diferença, +1 h), converta com `Date.UTC(...)` e volte a quebrar com os getters `getUTC*`.
- Já o instante **real** é outra coisa e vem com offset (RFC 3339): `gameEndAt` e os
  `createdAt`/`finishedAt` do histórico. Esses podem ir para `new Date`.

Hora do dia de início da partida (campo REST `storyStartTime`) é `"HH:MM:SS"`, também sem fuso.

## 2. Índice

**Cliente → servidor — todas são só do mestre** (`forbidden` para os demais)

| Mensagem | O que faz |
|---|---|
| [`set_clock_start`](#set_clock_start) | define a hora inicial (só quando a partida começou sem hora) |
| [`adjust_clock`](#adjust_clock) | ajuste relativo, em segundos (positivo ou negativo) |
| [`set_clock`](#set_clock) | vai para um instante absoluto |
| [`play_clock`](#play_clock) | liga o tempo real (só no regime `Free`) |
| [`pause_clock`](#pause_clock) | desliga o tempo real |
| [`get_combat_summary`](#get_combat_summary) | pede o resumo do combate da cena ativa |
| [`change_scene`](#change_scene-campos-novos) | (campos novos) fecha a cena com resumo final e hora de fim |
| [`end_match`](#end_match) | encerra a partida |

**Servidor → cliente**

| Mensagem | Destino |
|---|---|
| [`clock_changed`](#clock_changed) | mesa inteira |
| [`combat_summary`](#combat_summary) | **só quem pediu** (o mestre) |
| [`match_ended`](#match_ended) | mesa inteira; também a quem conecta depois do fim |
| `match_full_state.clock` | quem conecta/reconecta — ver [§7](#7-reconexão-reinício-queda) |

Nenhuma dessas mensagens tem payload além do descrito. Todo comando do mestre **sem sessão viva**
(partida em lobby) responde `match_not_started`.

## 3. O estado do relógio

Presente em `match_full_state.clock` e em `clock_changed.clock`:

```json
{
  "seq": 1762600000123456,
  "storyAt": "2000-01-01T08:30:00",
  "running": false,
  "limitAt": "2000-01-01T08:12:40",
  "limitReason": "turn"
}
```

| Campo | Tipo | Observação |
|---|---|---|
| `seq` | número | Numera os **retratos** do relógio. Monotônico, **nunca recomeça** (a sala nova parte do relógio de parede em µs, abaixo de 2^53). O cliente guarda o maior que aplicou e **descarta** qualquer retrato com `seq` menor. `match_full_state` leva o `seq` corrente sem avançá-lo; cada `clock_changed` leva um novo. Existe porque as mensagens saem por duas pistas sem ordem entre si. |
| `storyAt` | string \| `null` | O instante **agora** da história. **`null` = hora não definida** (veja [`set_clock_start`](#set_clock_start)). Sempre presente (nunca omitido). |
| `running` | bool | Tempo real ligado. Com `true`, `storyAt` vale **no instante em que o cliente recebeu a mensagem**: ele conta daí em diante, 1 s de história por 1 s real. O servidor não manda tick. |
| `limitAt` | string \| `null` | O instante antes do qual o relógio não pode voltar. `null` enquanto `storyAt` é `null`. |
| `limitReason` | string | Por que o limite está ali: `start` · `turn` · `scene` · `roundMode` · `round`. Vazio (`""`) enquanto a hora não está definida. |

**Quando o limite sobe** (e com qual motivo): início da partida/hora definida (`start`); turno
que **abre** (`turn`, só em memória até o turno fechar — ver [§6](#6-o-que-persiste-e-em-qual-transação));
round contado que fecha (`round`); troca de regime (`roundMode`); troca de cena e fim da partida
(`scene`). O limite = o `storyAt` daquele momento. Tempo real correndo **não** sobe o limite.

## 4. Cliente → servidor

Os exemplos usam só o `type` e o `payload`; o envelope é o de `match-combat-ws.md` §1.

### `set_clock_start`

**Quem:** só o mestre. Define a hora do dia quando a partida começou **sem** hora (o relógio
nasceu com `storyAt: null`). A **data** é a da partida (`storyStartAt`, fixa, lida do banco).

```json
{ "type": "set_clock_start", "payload": { "time": "08:30" } }
```

`time` é `"HH:MM"` ou `"HH:MM:SS"` (24 h, zeros à esquerda). O relógio nasce **pausado**, com
limite = esse instante e `limitReason: "start"`.

**Dispara:** [`clock_changed`](#clock_changed) `kind: "start"` (`to` = o instante). Grava o evento
`clockStarted`.

**Erros:** `forbidden` · `match_not_started` · `invalid_payload` (`"invalid set_clock_start payload"`
ou `time must be "HH:MM" or "HH:MM:SS"`) · `clock_already_set` (a hora já existe — **não é
idempotente**: a segunda chamada é recusada) · `game_error`
(`"could not read the match start date"`, `"story clock unavailable"`).

Enquanto o relógio não tem hora, ver [`clock_not_set`](#6-catálogo-de-erros): o mestre recebe o
diálogo obrigatório; `match_full_state.clock.storyAt: null` é o gatilho (também ao recarregar).

### `adjust_clock`

**Quem:** só o mestre. Ajuste relativo ao instante **corrente** (com o tempo real rodando, ao
valor de agora; ele continua rodando).

```json
{ "type": "adjust_clock", "payload": { "deltaSeconds": 3600 } }
{ "type": "adjust_clock", "payload": { "deltaSeconds": 259200, "confirmLongSkip": true } }
```

| Campo | Regra |
|---|---|
| `deltaSeconds` | inteiro, positivo ou negativo, **≠ 0** |
| `confirmLongSkip` | opcional (`false` por padrão). Avanço **maior que 172 800 s (2 dias)** a partir do relógio corrente exige `true`. Estritamente maior: exatamente 172 800 passa sem confirmar. Recuos nunca pedem. |

**Dispara:** `clock_changed` `kind: "adjust"` (`from`, `to`, `deltaSeconds` = `to − from`).
Grava `clockAdjusted` (com fusão — [§5](#fusão-de-ajustes-seguidos-no-histórico)).

**Erros:** `forbidden` · `invalid_payload` (inclui `deltaSeconds: 0`:
`"story time adjustment has no effect"`) · `match_not_started` · `clock_not_set` ·
`clock_below_limit` (o resultado ficaria abaixo de `limitAt`) · `clock_long_skip_unconfirmed`.

### `set_clock`

**Quem:** só o mestre. Vai a um instante absoluto — a edição direta da UI.

```json
{ "type": "set_clock", "payload": { "storyAt": "2000-01-01T10:40:00" } }
```

Mesmos limite e confirmação do `adjust_clock` (o salto é medido de `to` contra o relógio corrente).
`confirmLongSkip` opcional, igual.

**Dispara:** `clock_changed` `kind: "set"`. Grava `clockAdjusted` (mesma fusão).

**Erros:** os do `adjust_clock`; `invalid_payload` também para `storyAt` fora do formato naïve
(`"storyAt must be 2006-01-02T15:04:05"`) e para `storyAt` **igual** ao relógio corrente
(sem efeito).

### `play_clock`

**Quem:** só o mestre. Liga o tempo real. Sem payload (`{}`).

```json
{ "type": "play_clock", "payload": {} }
```

Só existe no regime **`Free`**. O relógio **sempre começa pausado**; o mestre liga explicitamente.

**Dispara:** `clock_changed` `kind: "play"` (sem `from`/`to`). **Não grava evento** — o tempo real
vira o evento `clockRealTime` na pausa; o estado "rodando" é gravado como checkpoint.

**Erros:** `forbidden` · `match_not_started` · `clock_not_set` · `clock_play_in_race` (round
ativo no regime `Race`) · `clock_already_running`.

### `pause_clock`

**Quem:** só o mestre. Sem payload (`{}`).

**Dispara:** `clock_changed` `kind: "pause"` com `reason: "manual"`, `from` (o instante em que o
`play` ligou), `to` (agora) e `elapsedSeconds`. Grava `clockRealTime`.

⚠️ **Já parado = sem resposta nenhuma**: nem `clock_changed`, nem erro. Não espere ack.

**Erros:** `forbidden` · `match_not_started` · `clock_not_set`.

### `get_combat_summary`

**Quem:** só o mestre. Pede o resumo do combate da **cena ativa**, que precisa ser `battle`.
Sem payload (`{}`). A resposta [`combat_summary`](#combat_summary) vai **só a quem pediu**.

Não exige hora definida, mas o front só o pede dentro do fluxo "Nova cena" da cena de combate, que
já exige relógio (sem hora, `minimumStoryAt` não é significativo).

**Erros:** `forbidden` · `match_not_started` · `not_battle_scene` (a cena ativa não é combate —
também se a cena mudou durante a leitura: `"the active scene changed"`) · `game_error`
(`"combat summary unavailable"`, `"could not read the combat rounds"`).

### `change_scene` (campos novos)

O verbo é o de [`match-combat-ws.md`](match-combat-ws.md#change_scene). Campos **novos**, todos
opcionais:

```json
{
  "type": "change_scene",
  "payload": {
    "category": "roleplay",
    "briefInitialDescription": "Taverna",
    "briefFinalDescription": "A emboscada terminou; Gon desmaiou.",
    "storyAt": "2000-01-01T09:15:00",
    "confirmLongSkip": false
  }
}
```

| Campo | Regra |
|---|---|
| `briefFinalDescription` | descrição final da cena que **fecha**, até 255 caracteres (aparado; vazio vira ausente). Acima disso: `invalid_payload`. |
| `storyAt` | hora exata do **fim** da cena que fecha (naïve). Ausente = o relógio fica onde está (+1 s se o round ativo foi contado). Não pode ficar abaixo do **mínimo** = `max(limitAt, storyAt corrente + 1 s se o round ativo conta)` — o mesmo `minimumStoryAt` do [`combat_summary`](#combat_summary). |
| `confirmLongSkip` | exigido quando `storyAt` está mais de 172 800 s à frente do relógio **antes** do comando (sem o segundo automático do round). |

O servidor, nesta ordem: recusa se o relógio não tem hora ou há turno aberto; fecha o round
(**+1 s** se contado); valida `storyAt`; **pausa o tempo real** (se estava ligado); move o relógio;
grava tudo; limite = o novo relógio com `limitReason: "scene"`.

**Dispara, nesta ordem (mesma pista):** [`scene_changed`](match-combat-ws.md#scene_changed) →
`clock_changed` `kind: "pause"`, `reason: "scene"` (**só se o tempo real corria**) →
`clock_changed` `kind: "scene"` (`from`, `to`, `deltaSeconds`; `deltaSeconds` é omitido se `0`).
Em seguida `bars_updated`, como antes.

**Erros novos:** `clock_not_set` · `clock_below_limit` · `clock_long_skip_unconfirmed` ·
`invalid_payload` (`briefFinalDescription` longo; `storyAt` malformado). Os antigos continuam
(`invalid_action`, `game_error` `cannot close round: current turn is still open`, …).

### `end_match`

**Quem:** só o mestre. Encerra a partida — **irreversível**.

```json
{
  "type": "end_match",
  "payload": {
    "storyAt": "2000-01-03T18:00:00",
    "briefFinalDescription": "Os heróis deixaram a ilha.",
    "confirmLongSkip": false
  }
}
```

| Campo | Regra |
|---|---|
| `storyAt` | **obrigatório**, naïve: a hora exata do fim na história. Não pode ficar abaixo do mínimo (`max(limitAt, corrente + 1 s se o round ativo conta)`). |
| `briefFinalDescription` | opcional, até 255 caracteres (aparado; vazio vira ausente); resumo final da **partida** (`matches.brief_final_description`). |
| `confirmLongSkip` | como em `adjust_clock`. |

**Recusas** (nesta ordem de checagem do servidor): `match_already_ended` · `clock_not_set` ·
`match_in_battle` (cena ativa é `battle` — é preciso trocar de cena antes) · `game_error`
`cannot close round: current turn is still open` · `clock_below_limit` ·
`clock_long_skip_unconfirmed`.

**Sucesso:** o servidor grava tudo numa transação ([§6](#6-o-que-persiste-e-em-qual-transação)),
**e só então** aplica o fim na sessão, manda [`match_ended`](#match_ended) à mesa inteira, fecha
**toda** conexão com o código **4002** (`"match ended"`) e para a sala. Se a gravação falha:
`game_error` `"could not end the match"`, a sessão fica intacta e a sala segue — o mestre pode tentar
de novo. Fila e reações em memória são descartadas. **Não** há `clock_changed` no fim: o
`match_ended` carrega o instante final. O tempo real, se ligado, é pausado (motivo `end`) e
registrado no histórico como `clockRealTime`.

**Erros:** `forbidden` · `invalid_payload` (`"invalid end_match payload"`, `storyAt must be
2006-01-02T15:04:05`, `briefFinalDescription is too long`) · `match_not_started` · os da lista de
recusas · `game_error` (`"match end unavailable"`, `"could not end the match"`).

## 5. Servidor → cliente

### `clock_changed`

**Destino:** **mesa inteira** (mestre e jogadores). Todo `clock_changed` leva o estado novo
(`clock`, [§3](#3-o-estado-do-relógio), com `seq` novo) e o que mudou (`change`).

```json
{
  "type": "clock_changed",
  "payload": {
    "clock": {
      "seq": 1762600000123460, "storyAt": "2000-01-01T10:40:00", "running": false,
      "limitAt": "2000-01-01T08:30:00", "limitReason": "start"
    },
    "change": {
      "kind": "adjust",
      "from": "2000-01-01T08:30:00", "to": "2000-01-01T10:40:00", "deltaSeconds": 7800
    }
  }
}
```

`change`:

| Campo | Observação |
|---|---|
| `kind` | ver tabela abaixo |
| `from`, `to` | naïve; omitidos quando o `kind` não tem (`play`, `limit`). `start` só tem `to`. |
| `deltaSeconds` | `to − from` em segundos, negativo quando volta; **omitido quando 0** |
| `reason` | só em `pause` |
| `elapsedSeconds` | só em `pause`: segundos **reais** que o tempo real correu desde o `play`. `to − from` inclui ajustes feitos com o relógio rodando; `elapsedSeconds` não. **Omitido quando 0 — ausente significa 0 s.** |

| `change.kind` | Quando | Campos extras |
|---|---|---|
| `start` | `set_clock_start` | `to` |
| `adjust` | `adjust_clock` | `from`, `to`, `deltaSeconds` |
| `set` | `set_clock` | `from`, `to`, `deltaSeconds` |
| `play` | `play_clock` | — |
| `pause` | `pause_clock`, troca de regime, troca de cena, saída do mestre | `from`, `to`, `reason`, `elapsedSeconds` |
| `round` | um round **contado** fechou pelo `open_next_action` (+1 s); chega **depois** do `round_closed`, mesma pista. **Jogadores não devem mostrar aviso** (só atualizar o número). | `from` = `to − 1 s`, `to`, `deltaSeconds: 1` |
| `scene` | `change_scene` | `from`, `to`, `deltaSeconds` |
| `limit` | só o **limite** subiu (turno abriu — `open_next_action`/`pull_action` — ou troca de regime sem tempo real ligado). Sem `from`/`to`. Útil ao painel do mestre; não é aviso. | — |

`kind: "end"` **existe no vocabulário da struct mas o servidor não o emite** nesta versão: o fim
da partida é anunciado por [`match_ended`](#match_ended). Trate `kind` desconhecido como "só
atualize o estado".

`pause.reason`:

| `reason` | Causa |
|---|---|
| `manual` | `pause_clock` |
| `roundMode` | troca de regime (`change_round_mode`) com o tempo real ligado. Sem tempo real ligado, a troca emite `kind: "limit"` no lugar. |
| `scene` | `change_scene` com o tempo real ligado |
| `masterLeft` | a conexão do mestre caiu — ver abaixo |
| `end` | fim da partida (só no histórico, `clockRealTime`; não há `clock_changed`) |
| `restart` | servidor reiniciou com o relógio rodando — só log e estado; **nunca** chega como mensagem ([§7](#7-reconexão-reinício-queda)) |

**Ordens garantidas** (mesma pista, nesta sequência): `round_closed` → `clock_changed(round)`;
`scene_changed` → `clock_changed(pause)` → `clock_changed(scene)`; `round_mode_changed` →
`clock_changed(pause|limit)` (depois `bars_updated`); `turn_opened` → `clock_changed(limit)`.
O `seq` ordena os retratos mesmo assim: descarte `seq` menor que o maior aplicado.

**A mensagem de `masterLeft`:** quando a última (única — uma conta tem uma conexão só) conexão do
mestre sai, o relógio **pausa na hora**, sem carência, e os clientes que restam recebem
`clock_changed` `pause` `reason: "masterLeft"`. **Não** dispara quando a conexão do mestre é
*substituída* por outra aba da mesma conta (ver `connection_replaced`): o relógio segue como estava.
Reconectar não religa nada: o mestre clica em ▶.

#### Fusão de ajustes seguidos no histórico

Cliques seguidos do mestre viram **um** evento de histórico. Um `adjust_clock`/`set_clock` que
chega **até 10 s reais depois do ajuste anterior** (a janela desliza: conta do último ajuste
fundido), sem outro evento no meio, **atualiza a linha anterior**: o `from` do início da sequência
fica, `to` é o novo e `deltaSeconds` é a **soma dos deltas pedidos** (o tempo real que correu entre
os cliques não entra). O `uuid` do evento não muda, nem o `createdAt` (o do primeiro clique).
Fecham a janela: `play_clock`/`pause_clock`/`set_clock_start`, abrir/fechar turno
(`open_next_action`, `pull_action`, `close_turn`), troca de regime, troca de cena, fim da partida,
reinício da sala e uma gravação que falhou. **A mensagem ao vivo não é fundida**: cada ajuste gera
o seu `clock_changed` com o próprio `from`/`to`/`deltaSeconds`. Quem relê `GET /history` vê a
linha única atualizada no lugar ([§8](#8-histórico-e-campos-rest)).

### `combat_summary`

**Destino:** **só quem pediu** (o mestre), resposta a [`get_combat_summary`](#get_combat_summary).

```json
{
  "type": "combat_summary",
  "payload": {
    "sceneId": "66666666-6666-4666-8666-666666666666",
    "countedRounds": 3,
    "countedSeconds": 3,
    "activeRoundCounts": true,
    "minimumStoryAt": "2000-01-01T08:31:05",
    "hiatuses": [
      {
        "before": {
          "roundNumber": 1,
          "turnId": "11111111-1111-4111-8111-111111111111",
          "actorId": "22222222-2222-4222-8222-222222222222",
          "actionKind": "attack"
        },
        "after": {
          "roundNumber": 2,
          "turnId": "33333333-3333-4333-8333-333333333333",
          "actorId": "44444444-4444-4444-8444-444444444444",
          "actionKind": "move"
        }
      }
    ]
  }
}
```

| Campo | Significado |
|---|---|
| `sceneId` | a cena ativa (de combate) |
| `countedRounds` | rounds **fechados** da cena que foram contados (tiveram ao menos um turno aberto no `Race`). O round ativo **não** entra aqui. |
| `countedSeconds` | `countedRounds` × 1 s. **Já está somado no relógio** (cada round contado somou 1 s ao fechar). |
| `activeRoundCounts` | o round em andamento conta: o 1 s dele entra ao fechar (a UI mostra "+1 s ao encerrar") |
| `minimumStoryAt` | naïve; início do editor de relógio ao encerrar o combate = `max(limitAt, corrente + 1 s se activeRoundCounts)`. O `storyAt` do `change_scene`/`end_match` não pode ficar abaixo. |
| `hiatuses` | sempre lista (`[]`, nunca `null`), na ordem da cena |

**Hiato** = um round **contado** que fechou com a fila **vazia** (`drained`) e que tem um **sucessor
com turno** na mesma cena (o round ativo conta como sucessor se já tem turno). Um round que
esvaziou sem sucessor com turno (o mestre trocou de cena logo depois) **não** é hiato. Um round
que fechou com ações ainda na fila também não. Round sem turno é pulado ao procurar o sucessor.
`before` = o **último** turno do round que esvaziou; `after` = o **primeiro** turno do sucessor.

`roundNumber` = posição 1-based do round na cena, contando todos os rounds da cena (também os não
contados). `actionKind` = o tipo da ação do turno, o mesmo de `actions.type`:
`attack` · `move` · `defense` · `dodge` · `feint` · `interact` · `skill` · `reaction` ·
`unspecified`. O sistema não sabe o que houve no hiato: o mestre soma o tempo no editor.

### `match_ended`

**Destino:** a mesa inteira, ao fim da partida; **e** a qualquer pessoa autorizada que conecte
depois (mesa **ou lobby**, mestre ou jogador inscrito): o handler responde `match_ended` e fecha
com 4002 logo depois do upgrade, sem criar sala. (A checagem de mestre/inscrito vem antes: quem
não pode entrar continua recebendo `403` no handshake.)

```json
{
  "type": "match_ended",
  "payload": {
    "matchId": "99999999-9999-4999-8999-999999999999",
    "storyAt": "2000-01-03T18:00:00",
    "gameEndAt": "2026-11-08T22:14:05Z"
  }
}
```

| Campo | |
|---|---|
| `matchId` | |
| `storyAt` | instante final **da história**, naïve; `null` se a partida não tem relógio gravado (só possível na reconexão a uma partida antiga) |
| `gameEndAt` | instante **real** do fim, RFC 3339 em UTC (`Z`), como `gameStartAt` |

**Logo depois, o servidor fecha o WebSocket com o código `4002`, motivo `"match ended"`.** Esse é o
sinal terminal: o cliente **não** deve reconectar (o hook de reconexão precisa tratar 4002 como
4001 — fim, não queda) e deve navegar para a página da partida com `replace`. `/game` e `/lobby`
de uma partida encerrada devem redirecionar sozinhas (cobre quem estava offline: o `GET` da
partida traz `gameEndAt`).

## 6. Catálogo de erros

Formato `error` de `match-combat-ws.md` §5. Vão **só** a quem enviou. Códigos novos desta feature:

| `code` | Gatilho | Onde |
|---|---|---|
| `clock_not_set` | o relógio não tem hora (`storyAt: null`) e o comando mexe na linha do tempo | `open_next_action`, `pull_action`, `change_round_mode`, `change_scene`, `end_match`, `adjust_clock`, `set_clock`, `play_clock`, `pause_clock`. **Não** recusa: `set_clock_start`, `get_combat_summary`, declarar ações (jogador) e as master actions de tabuleiro. |
| `clock_already_set` | `set_clock_start` com a hora já definida | `set_clock_start` |
| `clock_below_limit` | o destino ficaria abaixo do limite (ou do mínimo do fechamento) | `adjust_clock`, `set_clock`, `change_scene`, `end_match` |
| `clock_long_skip_unconfirmed` | avanço > 172 800 s sem `confirmLongSkip: true` | `adjust_clock`, `set_clock`, `change_scene`, `end_match` |
| `clock_play_in_race` | `play_clock` com o round no regime `Race` | `play_clock` |
| `clock_already_running` | `play_clock` com o tempo real já ligado | `play_clock` |
| `match_in_battle` | `end_match` com a cena ativa de combate | `end_match` |
| `match_already_ended` | a partida já foi encerrada e **qualquer mensagem de qualquer cliente** chegou depois do fim (o guarda roda no topo de `handleClientMessage`: mestre, jogadores, chat, tudo) — só uma mensagem em voo na corrida o vê, pois a sala fecha as conexões em seguida. **Terminal para o front**: trate como `match_ended` (parar, não reenviar, não reconectar). | qualquer mensagem |
| `not_battle_scene` | a cena ativa não é combate | `get_combat_summary` |

Reaproveitados: `forbidden` (não é o mestre) · `invalid_payload` (JSON inválido; `deltaSeconds` 0 ou
`set_clock` para o instante corrente; `storyAt`/`time` fora do formato; `briefFinalDescription` > 255)
· `match_not_started` · `game_error` (domínio: turno aberto
`cannot close round: current turn is still open`; falhas de leitura/gravação). Os erros de relógio
chegam também em `open_next_action`, `pull_action`, `change_round_mode` e `change_scene` com o
`code` acima (antes eram todos `game_error`), e **nada muda** na sessão nessas recusas.

## 7. Reconexão, reinício, queda

- **O que a reconexão recebe:** `match_full_state.clock` (quem conecta, jogador ou mestre), com o
  estado da [§3](#3-o-estado-do-relógio) e o `seq` corrente. É o único retrato de que o cliente
  precisa; com `running: true` ele conta a partir do recebimento. Ausente apenas se não há
  sessão (lobby). Mestre que recarrega sem ter definido a hora recebe `storyAt: null` de novo —
  o diálogo volta.
- **Checkpoint:** com o relógio rodando, o servidor grava o estado a cada **60 s** (sem evento, sem
  mensagem).
- **Reinício do servidor:** a sala que renasce encontra o relógio "rodando" no banco e o **pausa no
  último checkpoint** (`reason: restart`, só log e estado; sem evento e sem `clock_changed`,
  ninguém está conectado). Perde-se no máximo 1 min de tempo real; o tempo em que o servidor
  esteve parado nunca é somado. Quem reconecta recebe `running: false`.
- **Mestre sai:** ver `masterLeft` na [§5](#clock_changed). Os jogadores saindo não pausam.
- **Partida encerrada:** qualquer conexão recebe `match_ended` + 4002 ([§5](#match_ended)).
- **Duas abas:** a antiga recebe `connection_replaced`; isso **não** pausa o relógio.

### O que persiste em qual transação

| Comando | Transação |
|---|---|
| `set_clock_start`, `adjust_clock`, `set_clock`, `pause_clock`, pausa de regime, pausa de saída do mestre | estado do relógio (upsert em `match_clocks`) + evento (`clockStarted` / `clockAdjusted` / `clockRealTime`), juntos. Falha = log; o próximo salvamento cura o estado, o evento perdido fica só no log. |
| `play_clock`, checkpoint | só o estado do relógio |
| fechamento de turno / fim de round | a do turno: `turns.story_at` (instante da **abertura**), `rounds.counted`/`drained` e o relógio, juntos (regra "um comando do mestre, uma transação"). **Abrir** um turno sobe o limite só em memória; um reinício com turno aberto o perde junto e o limite volta ao último gravado. |
| `change_scene` | fechamento do par antigo (cena com `briefFinalDescription` e `storyFinishedAt`, round com `counted`/`drained`) + relógio, numa transação; depois o par novo (com `storyStartedAt`) e o `clockRealTime` da pausa |
| `end_match` | **uma** transação: `matches.game_end_at` (real) / `story_end_at` (data) / `brief_final_description`; cena e round fechados; relógio parado no instante final; `clockRealTime` da pausa `end`; as barras de HP ainda não gravadas (`MatchEndData.StatusBars`, fichas de um fechamento de turno que falhou); `campaigns.story_current_at` = `GREATEST(atual, instante final)`; `character_sheets.story_current_at` (data) de **todas** as fichas de `match_participants` (jogadores e NPCs), mesma regra de nunca regredir. Recusa partida já encerrada. |

## 8. Histórico e campos REST

Detalhes de formato em [`match-history.md`](match-history.md) e [`match.md`](match.md).

**`GET /matches/{uuid}/history`** — campos novos:
- `scenes[].briefFinalDescription` · `scenes[].storyStartedAt` · `scenes[].storyFinishedAt`
  (naïve; omitidos quando nulos);
- `rounds[].counted` e `rounds[].drained` (bool, sempre presentes);
- `rounds[].turns[].storyAt` (instante da história na abertura do turno; omitido se nulo);
- `rounds[].events[]` com três `kind` novos: `clockStarted` (`payload: {"at"}`),
  `clockAdjusted` (`payload: {"from","to","deltaSeconds"}`; linha única fundida) e `clockRealTime`
  (`payload: {"from","to","elapsedSeconds","reason"}`, gravado ao pausar; `reason` como em
  `pause.reason`). Os instantes dentro de `payload` são naïve.

**REST da partida** (`POST /matches`, `PATCH /matches/{uuid}`, `GET /matches/{uuid}`):
- entrada `storyStartTime`: `"HH:MM"` ou `"HH:MM:SS"`, opcional; no `PATCH`, `""` limpa; recusado
  (422) em formato inválido. Edição continua recusada depois que a partida começou.
- saída `storyStartTime` (`"HH:MM:SS"`, omitido se não há), `gameEndAt` (RFC 3339; omitido enquanto
  não encerrada) e, **só encerrada**, `storyEndTime` (`"HH:MM:SS"`, hora do fim na história).
- Listas/resumos de partida (`GET /matches`, públicas, inscrições, participantes) trazem `gameEndAt`.
- **Partida encerrada = `gameEndAt` preenchido.** `storyEndAt` (data) deixou de ser o critério.
