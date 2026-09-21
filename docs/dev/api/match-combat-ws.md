# Match / Motor de Combate — Protocolo WebSocket

**Status:** Fase 5 implementada (back). Este documento é o **contrato de handoff para a
Fase 6 do front** — foi escrito conferindo cada payload contra a struct em
`internal/app/game/message.go`, e os exemplos JSON abaixo foram **gerados serializando
essas structs**, não transcritos de memória.

**Servidor:** `cmd/game/` · **URL:** `ws://localhost:8081/ws?match_uuid=<uuid>&token=<jwt>&nickname=<name>`

`nickname` é **opcional**, ao contrário do que a URL acima sugere. Vazio ou ausente, o
servidor usa os 8 primeiros caracteres do UUID do usuário autenticado
(`HandleWebSocket` em `handler.go`) — não é um erro tratado, é o fallback real. Isto era
erro do contrato, não do servidor: corrigido aqui.

> ### ⚠️ Nenhum cliente real leu este contrato ainda
>
> Os exemplos de payload aqui foram **conferidos contra as structs de
> `internal/app/game/message.go`** — gerados serializando-as, não escritos de memória — e o
> despacho foi lido em `room.go`. Mas **nenhum byte deste documento passou por um cliente**:
> a Fase 6 do front é o primeiro leitor real, e construir um verificador antes dela seria
> construir o cliente duas vezes.
>
> **Se você achar uma divergência, o bug é do CONTRATO.** Corrija este arquivo — não contorne
> indo ler o Go e seguindo em frente. Um documento que a primeira pessoa aprendeu a não
> confiar deixa de ser contrato e vira decoração, e a próxima pessoa paga de novo o custo que
> ele existe para evitar. Se a divergência for do servidor, o conserto é lá, mas o registro
> vem para cá do mesmo jeito.

Este arquivo cobre as mensagens de **partida e combate**. As de lobby estão em
[`game-lobby.md`](game-lobby.md); as de mapa, peça, parede e fog em
[`maps.md`](maps.md) e [`match-maps.md`](match-maps.md). As regras de jogo por trás dos
números estão em [`../match/combat-engine.md`](../match/combat-engine.md) e
[`../../game/combate/reacoes.md`](../../game/combate/reacoes.md); os fluxos desenhados, em
[`../match/flows/03-fluxo-de-acao.md`](../match/flows/03-fluxo-de-acao.md).

---

## 1. O envelope

Toda mensagem, nos dois sentidos, é um `Message`:

```json
{
  "type": "turn_closed",
  "payload": { "turnId": "5555…" },
  "senderId": "00000000-0000-0000-0000-000000000000",
  "timestamp": "2026-09-20T16:44:00Z"
}
```

| Campo | Observação |
|---|---|
| `type` | Discriminador. Um `type` desconhecido devolve `error` com código `unknown_type`. |
| `payload` | **Objeto JSON**, não string. (`Payload` é `json.RawMessage`.) |
| `senderId` | `00000000-…` em toda mensagem **de servidor**. No sentido cliente→servidor o campo é **ignorado**: a identidade vem do JWT do handshake, nunca do corpo. |
| `timestamp` | Preenchido pelo servidor. Ignorado no que chega do cliente. |

**Regras do wire que valem para tudo abaixo:**

- **camelCase dos dois lados.** Sem conversão no front.
- **`omitempty` não funciona em UUID nem em array de tamanho fixo.** `uuid.UUID` é
  `[16]byte` e `encoding/json` nunca elide um array. Na prática:
  - `reactToId` **sempre aparece** num `enqueue_action` de ação comum, com o valor
    `"00000000-0000-0000-0000-000000000000"`. Esse zero é o sentinela de "não é reação" em
    todo o código — não é um bug de serialização, e o front deve tratá-lo como ausência.
  - `move.from` aparece como `[0, 0, 0]` quando não foi enviado. Zero significa
    "não informado" (e desliga a checagem de parede).
- **O servidor nunca confia no cliente para** qual barra a ação paga (`speed.bar` é
  descartado), qual perícia mede a velocidade (é sempre Legerity), a perícia de velocidade
  de um movimento (vem da categoria) ou os dados. **Os dados caem no servidor**, no instante
  em que a ação é aceita, e nunca são rolados de novo.

## 2. Quem é quem

| Termo | O que é |
|---|---|
| **Jogador / mestre** | `User` autenticado. É quem a conexão WS identifica. |
| **Personagem** | `character_sheets.uuid`. É a **entidade de combate**, e é o que vai em `actorId` e em `targetId`. |
| Ponte | Uma pessoa dirige vários personagens. O servidor checa que o personagem de `actorId` pertence a quem enviou. |

> **NPC hoje não age.** `indexParticipants` só mapeia personagem→jogador para fichas com
> `player_uuid`; uma ficha de NPC (dono = mestre) não entra nesse mapa, então
> `enqueue_action` por ela devolve `game_error: action actor does not match player`. Isso é
> a lacuna "Rostering de NPC" de `AGENTS.md`, não um erro deste contrato.

## 3. Índice

**Cliente → servidor**

| Mensagem | Quem pode enviar |
|---|---|
| [`enqueue_action`](#enqueue_action) | jogador (pelo próprio personagem) |
| [`attach_reaction`](#attach_reaction) | jogador **alvo** da ação aberta |
| [`open_next_action`](#open_next_action) | mestre |
| [`pull_action`](#pull_action) | mestre |
| [`open_reaction`](#open_reaction) | mestre |
| [`edit_action`](#edit_action) | mestre |
| [`close_turn`](#close_turn) | mestre |
| [`change_round_mode`](#change_round_mode) | mestre |
| [`change_scene`](#change_scene) | mestre |
| [`enqueue_master_action`](#enqueue_master_action) | mestre |

**Servidor → cliente**

| Mensagem | Destino |
|---|---|
| [`action_enqueued`](#action_enqueued) | só quem enviou |
| [`action_queued`](#action_queued) | **só o mestre** |
| [`bars_updated`](#bars_updated) | mesa inteira |
| [`turn_opened`](#turn_opened) | mesa inteira |
| [`reaction_opened`](#reaction_opened) | mesa inteira |
| [`resolution_updated`](#resolution_updated) | **mestre, ou mesa projetada** — ver §5 |
| [`action_edited`](#action_edited) | só o mestre |
| [`close_turn_refused`](#close_turn_refused) | só o mestre |
| [`turn_closed`](#turn_closed) | mesa inteira |
| [`round_closed`](#round_closed) | mesa inteira |
| [`round_mode_changed`](#round_mode_changed) | mesa inteira |
| [`scene_changed`](#scene_changed) | mesa inteira |
| [`master_action_enqueued`](#master_action_enqueued) | mesa inteira |
| [`match_full_state`](#match_full_state) | quem conecta/reconecta, enquanto há sessão viva |
| [`piece_moved`](#piece_moved-servidor) (também servidor) | fog-gated, por destinatário |
| [`error`](#error) | só quem enviou |

---

## 4. Cliente → servidor

### `enqueue_action`

**Direção:** cliente → servidor. **Quem:** qualquer jogador, pelo personagem que é dele.
Não é master-only — mas o servidor checa a posse do personagem.

Põe uma ação na fila. **Não abre turno nenhum**: quando ela abre é decisão do mestre
(`open_next_action` / `pull_action`). A fila é secreta; a barra e a ordem são públicas.

Se `reactToId` for não-zero, a mensagem é roteada internamente como
[`attach_reaction`](#attach_reaction) — mesmo payload, mesmas regras.

```json
{
  "type": "enqueue_action",
  "payload": {
    "actorId": "11111111-1111-4111-8111-111111111111",
    "reactToId": "00000000-0000-0000-0000-000000000000",
    "targetId": ["22222222-2222-4222-8222-222222222222"],
    "skills": [ { "skillName": "Accuracy", "difficulty": 15 } ],
    "speed": { "bar": 1, "rollCheck": { "skillName": "Legerity" } },
    "feint": { "skillName": "Feint" },
    "move": {
      "category": "Dash",
      "from": [4, 4, 0],
      "position": [6, 4, 0],
      "speed": { "skillName": "Accelerate" },
      "charge": { "skillName": "Energy" }
    },
    "attack": {
      "weapon": "Sword",
      "hit": { "skillName": "Accuracy" },
      "damage": { "skillName": "Push" },
      "charge": { "skillName": "Energy" }
    },
    "defense": { "weapon": "Sword", "rollCheck": { "skillName": "Defense" } },
    "dodge": { "category": "Dash", "rollCheck": { "skillName": "Reflex" } },
    "interact": { "kind": "open" }
  }
}
```

Toda sub-seção é opcional; o exemplo mostra **todas juntas** para nomear os campos, não
porque uma ação plausível carregue todas.

| Campo | Notas |
|---|---|
| `actorId` | **Obrigatório.** UUID da **ficha**, não do jogador. |
| `targetId` | Lista, apesar do nome no singular. Alvos podem ser personagens **ou paredes** (`wall_segment`). |
| `skills[].difficulty` | CD proposta pelo cliente. A corrente de testes ainda **não é executada** pelo motor (ver §8). |
| `speed` | **Descartado.** `bar` é derivado do conteúdo por `Action.Bars()`; a perícia é sempre Legerity. |
| `move.category` | Só **`Dash`** e `Shift` são aceitos. `Back`, `Roll`, `Slide`, `Jump`, `FlatJump` são **recusados** — a fatia de movimento é que vai exercê-los. |
| `move.from` | `[col, row, z]`. Quando não-zero, o servidor valida o caminho contra paredes com `move=true` e `open=false`. |
| `interact.kind` | `open` · `close` · `toggle` · `lockpick` · `examine`. (`reveal` é master-only, por `enqueue_master_action`.) |
| `dodge.category` | **Descartado pelo mapper** — o campo existe no payload e nada o lê. Só `dodge.rollCheck` importa. |
| `attack.weapon`, `defense.weapon` | Nome do catálogo (`enum.WeaponName`). Ausente = desarmado. |
| `attack.damage.skillName` | **Descartado.** O dano soma o **`Push`** do atacante, lido direto da ficha (`TurnResolver.actorPush`) — nunca a perícia que o payload manda. O campo continua **validado quando não-vazio** (`buildRollCheck` só chama `SkillNameFrom` se a string não for `""`, então `"damage": {}` passa em branco) mas não decide mais nada, o mesmo estado de `speed`. Trocar `Push` por `Grab` é prerrogativa do mestre, ainda não implementada. |

**Dispara:** [`action_enqueued`](#action_enqueued) para quem enviou,
[`action_queued`](#action_queued) **só para o mestre**, e
[`bars_updated`](#bars_updated) para a mesa.

**Erros**

| `code` | Quando |
|---|---|
| `invalid_payload` | `"invalid action payload"` — JSON não casa com a struct. |
| `invalid_action` | `"a reaction needs both reactToId and reactionKind; an action needs neither"` — os dois viajam juntos ou nenhum. |
| `invalid_action` | `"actorId is required: the acting character's sheet UUID"` |
| `invalid_action` | Erro do mapper: perícia desconhecida, arma desconhecida, `move category "X" is not supported yet`. |
| `match_not_started` | `"match session not initialized"` — sessão ainda não existe. |
| `move_blocked` | `"movement blocked by a wall"` |
| `game_error` | `participant not found in match session` · `action actor does not match player`. |

### `attach_reaction`

**Direção:** cliente → servidor. **Quem:** o jogador de um personagem que **está em
`targetId` da ação aberta**. Um espectador não reage: recebe
`game_error: only a target of the open action may react to it`.

Anexa uma reação ao turno aberto. **Anexar não é narrar** — a reação entra no cálculo na
hora, mas só ganha a palavra quando o mestre manda [`open_reaction`](#open_reaction).

O payload é o mesmo `ActionPayload`, com `reactToId` e `reactionKind` obrigatórios.

```json
{
  "type": "attach_reaction",
  "payload": {
    "actorId": "22222222-2222-4222-8222-222222222222",
    "reactToId": "33333333-3333-4333-8333-333333333333",
    "reactionKind": "closedDodge",
    "skills": [ { "skillName": "Evasion" } ],
    "dodge": { "category": "Dash", "rollCheck": { "skillName": "Reflex" } }
  }
}
```

```json
{
  "type": "attach_reaction",
  "payload": {
    "actorId": "22222222-2222-4222-8222-222222222222",
    "reactToId": "33333333-3333-4333-8333-333333333333",
    "reactionKind": "repel",
    "repel": { "weapon": "Sword", "rollCheck": { "skillName": "Repel" } }
  }
}
```

**`reactionKind` é declarado, nunca inferido.** As três fugas têm exatamente a mesma forma
— uma esquiva e um movimento — e custam três coisas diferentes; nenhuma inspeção do payload
as separa, porque o que as separa é a intenção do jogador.

| `reactionKind` | Componentes obrigatórios | Barras que cobra | Categoria de `move` exigida |
|---|---|---|---|
| `nothing` | — | nenhuma | — |
| `dodge` | `dodge` | nenhuma | — |
| `closedDodge` | `dodge` + entrada `Evasion` em `skills` | nenhuma | — |
| `escape` | `dodge` + `move` | `action` + `move` | **Dash** |
| `escapeGuard` | `dodge` + `move` | `action` + `move` | **Dash** |
| `closedEscape` | `dodge` + `move` + entrada `Evasion` em `skills` | `move` | **Shift** |
| `repel` | `repel` | `action` | — |

**A categoria de `move` das três fugas é validada no SERVIDOR, não sugerida.**
`ReactionKind.RequiredMoveCategory()` (`reaction_kind.go`) fixa o par; `action_mapper.go`
recusa o resto — um `move.category` diferente do exigido devolve `invalid_action` com
`reaction "X" must move with Dash, not Shift` (ou o par correspondente). O discriminador é
**fechado × aberto**, não defensivo × padrão: durante o `Dash` o personagem está "no ar" e
não consegue esquivar — exatamente o que a fechada existe para não fazer, e por isso ela
pisa com `Shift`, que `Brake` mede.

Uma reação **livre** (as que não cobram barra) não consome a ação que o personagem tinha na
fila e não rola em Desvantagem. Uma reação **cobrada** consome a ação enfileirada daquele
personagem em cada barra que cobra — e, se consumiu algo, a troca custa **Desvantagem**
(`SystemBias = -1`).

**Dispara:** [`resolution_updated`](#resolution_updated) **só para o mestre** — o turno está
aberto, logo `isSettled: false`. A reação aparece em `pendingReactions`.
Não há ack próprio e **não há broadcast**: a mesa não é avisada de que alguém reagiu.

**Erros**

| `code` | Quando |
|---|---|
| `invalid_payload` | `"invalid action payload"` |
| `invalid_action` | `"reaction requires react_to_id"` |
| `invalid_action` | `"a reaction needs both reactToId and reactionKind; an action needs neither"` |
| `invalid_action` | `"actorId is required: …"` |
| `invalid_action` | `reaction "X" must carry a dodge` / `a move` / `a repel` / `an evasion skill entry`; `reaction kind "X" is not in the catalogue`; `reaction "X" must move with Y, not Z` (categoria de `move` errada — ver a matriz acima). |
| `match_not_started` | Sessão inexistente. |
| `game_error` | `the reacting character does not belong to this player` · `no current turn in round` · `cannot open a reaction: turn already closed` · `only a target of the open action may react to it` · `reaction does not target the current action`. |

### `open_next_action`

**Direção:** cliente → servidor. **Quem:** **só o mestre.** Qualquer outro recebe
`error` com `code: "forbidden"` e mensagem `only the master can perform this action`.

Sem payload (o que vier é ignorado):

```json
{ "type": "open_next_action", "payload": {} }
```

Fecha o turno sob a batuta (se houver), aplica o dano dele, e abre o próximo. Em **Race** o
escalonador escolhe (ele conhece as barras e o portão de elegibilidade); em **Free** sai a
ação de **maior velocidade rolada** — sem preço, média nem carry-over, a velocidade É a
ordem. Empate entre chaves iguais resolve por ordem de chegada.

**Dispara, nesta ordem:**

1. [`bars_updated`](#bars_updated) — mesa.
2. Se um turno fechou: persistência + [`resolution_updated`](#resolution_updated) **settled
   e projetado** do turno que acabou (é essa a resolução cujo dano foi aplicado de verdade).
3. Se a rodada esgotou: [`round_closed`](#round_closed) — mesa — e **para por aí**.
4. Senão: [`turn_opened`](#turn_opened) — mesa — e
   [`resolution_updated`](#resolution_updated) **master-only** do turno recém-aberto
   (`isSettled: false`).

> O passo 2 acontece **antes** do erro ser reportado, mesmo quando o `open` falha: um turno
> que já fechou e já aplicou dano precisa chegar à mesa de qualquer jeito.

**Erros:** `forbidden` · `match_not_started` · `game_error` (`action queue is empty`,
`action not found in queue`).

> O use case também recusa um chamador que não é o mestre (`user is not the match master`),
> mas por esta porta isso é inalcançável: `room.go` já recusou antes, com `forbidden`. Vale
> para toda mensagem master-only abaixo.

### `pull_action`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

Abre uma ação específica **fora de ordem**. Antecipar é prerrogativa do mestre: diferente de
`open_next_action`, isto **não passa pelo portão de elegibilidade** do escalonador.

```json
{ "type": "pull_action", "payload": { "actionId": "33333333-3333-4333-8333-333333333333" } }
```

> **`actionId` só pode ser aprendido por [`action_queued`](#action_queued)**, que é
> master-only. Não existe outra superfície que nomeie o ID de uma ação enfileirada — a fila
> é secreta. Um ID que o cliente não consegue aprender é uma operação que o cliente não
> consegue invocar; `action_queued` existe exatamente para fechar esse buraco.

**Dispara:** o mesmo conjunto de `open_next_action` (passos 1, 2 e 4). **Nunca** fecha a
rodada — `round_closed` só sai de `open_next_action`.

**Erros:** `forbidden` · `invalid_payload` (`"invalid pull_action payload"`) ·
`match_not_started` · `game_error` (`action not found in queue`).

### `open_reaction`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

Passa o microfone para uma reação já anexada. **A ordem em que o mestre abre as reações é
poder de mestre e muda o resultado**: uma reação anexada mas não aberta deliberadamente
*não* vira passo da cadeia.

```json
{ "type": "open_reaction", "payload": { "reactionId": "44444444-4444-4444-8444-444444444444" } }
```

> **`reactionId` vem de `resolution_updated.pendingReactions[].reactionId`** (master-only),
> ou de `close_turn_refused.pendingReactions[]`. Uma reação já aberta aparece em
> `targets[].reaction.reactionId`.

Abrir **não cobra nada**: as barras foram debitadas no *attach*, justamente para que narrar
não mova número.

**Dispara:** [`reaction_opened`](#reaction_opened) para a **mesa** (de quem é a vez de
narrar é público) e [`resolution_updated`](#resolution_updated) **master-only** (o cálculo
continua sendo do mestre — o turno ainda está aberto).

**Erros:** `forbidden` · `invalid_payload` (`"invalid open_reaction payload"`) ·
`match_not_started` · `game_error` (`no current turn in round`,
`cannot open a reaction: turn already closed`, `reaction not found on the current turn`).

### `edit_action`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

Edita a ação do turno aberto, ou uma das reações dela. Toda seção é opcional e independente
— mande só o que muda. **Uma seção presente SUBSTITUI a lista inteira**: não há merge
parcial, porque o wire não tem identidade por entrada.

```json
{
  "type": "edit_action",
  "payload": {
    "actionId": "33333333-3333-4333-8333-333333333333",
    "conditions": [
      { "field": "hit", "bias": -1, "modifier": -2, "description": "escuridao" },
      { "skillName": "Accuracy", "bias": 1 }
    ],
    "skills": [ { "skillName": "Accuracy", "difficulty": 15 } ],
    "targetIds": ["22222222-2222-4222-8222-222222222222"]
  }
}
```

| Campo | Notas |
|---|---|
| `actionId` | Ausente ou zero = **a ação própria do turno**. Senão, o ID de uma reação anexada. |
| `conditions[].field` | `speed` · `hit` · `damage` · `dodge` · `defense` · `repel` · `feint` · `moveSpeed`. |
| `conditions[].skillName` | **Alternativa** a `field`, nomeando uma entrada de `skills`. Mandar os dois é erro. |
| `bias` | Vantagem/desvantagem nos dados (−1 / 0 / +1). **Não** é somado ao total: escolhe qual conjunto de dados é lido. |
| `modifier` | Ajuste plano no total. |

A edição **não re-rola nada**. Os dados já caíram; `bias` só troca qual conjunto é lido.
Toda condição é **validada antes de qualquer mutação** — uma edição que falha no meio não
deixa `targetIds` ou `skills` já alterados.

**Dispara:** [`action_edited`](#action_edited) **só para o mestre** e
[`resolution_updated`](#resolution_updated) recomputado.

**Erros:** `forbidden` · `invalid_payload` (`"invalid edit_action payload"`) ·
`invalid_action` (`condition field "X" is not recognized`, perícia desconhecida) ·
`match_not_started` · `game_error` (`no active turn in current round`,
`action is not on the open turn`,
`condition edit targets a check that is not on this action`,
`condition edit must set either field or skillName, not both`).

### `close_turn`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

Encerra o turno aberto de propósito, **sem abrir o próximo**. É o que torna "fechado e nada
aberto" um estado em que o mestre pode ficar.

```json
{ "type": "close_turn", "payload": {} }
```

```json
{ "type": "close_turn", "payload": { "confirm": true } }
```

**O diálogo de confirmação é do SERVIDOR.** Se existir alguma reação **anexada e não
aberta**, um `close_turn` sem `confirm` **não fecha nada** e devolve
[`close_turn_refused`](#close_turn_refused) com a lista de quem ficou sem narrar. O retry é
a **mesma mensagem com `confirm: true`**.

> O que se confirma não é o cálculo — a reação não aberta entra na colisão de qualquer jeito
> — é o **momento de narrar**, que ela vai perder. Um cliente que manda `confirm: true`
> sempre não burla nada: sem pendência, não há o que confirmar.

Fechar um turno **não fecha a rodada**. Só `open_next_action` detecta exaustão.

**Dispara** (no caminho que de fato fecha):

1. Persistência do turno (ação, reações, overrides, resolução liquidada).
2. [`turn_closed`](#turn_closed) — mesa.
3. [`resolution_updated`](#resolution_updated) **settled e projetado por destinatário**.
4. [`bars_updated`](#bars_updated) — mesa.

**Erros:** `forbidden` · `invalid_payload` (`"invalid close_turn payload"`) ·
`match_not_started` · `game_error` (`no open turn to close`).

### `change_round_mode`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

```json
{ "type": "change_round_mode", "payload": { "mode": "Race" } }
```

`mode` ∈ `Free` | `Race`. **Free** não tem preço, média nem carry-over: a velocidade rolada
é a ordem e nada tranca. **Race** é o regime com economia de barras.

O regime que a mesa recebe de volta é **lido da sessão**, não ecoado do pedido.

**Dispara:** [`round_mode_changed`](#round_mode_changed) e [`bars_updated`](#bars_updated),
ambos para a mesa.

**Erros:** `forbidden` · `invalid_payload` (`"invalid change_round_mode payload"`) ·
`match_not_started` · `game_error` (`round mode must be Free or Race`).

### `change_scene`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

```json
{
  "type": "change_scene",
  "payload": { "category": "battle", "briefInitialDescription": "Arena" }
}
```

⚠️ **`category` é minúscula.** `enum.SceneCategory` vale `"battle"` ou `"roleplay"` — e o
servidor **não valida**: ele faz `enum.SceneCategory(payload.Category)` e guarda a string como
veio. Mandar `"Battle"` não dá erro; cria uma cena cuja categoria não é igual a nenhum dos dois
valores, e ela sai assim em `scene_changed` e em `match_full_state`. (`roundMode`, ao
contrário, é capitalizado: `"Free"`/`"Race"`.)

Fecha cena e rodada correntes e abre uma cena nova com a primeira rodada dentro.

**Dispara:** [`scene_changed`](#scene_changed) — mesa.

**Erros:** `forbidden` · `invalid_payload` (`"invalid change_scene payload"`) ·
`match_not_started` · `game_error` (`cannot close round: current turn is still open`).

### `enqueue_master_action`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

```json
{
  "type": "enqueue_master_action",
  "payload": {
    "targetIds": ["22222222-2222-4222-8222-222222222222"],
    "skills": [ { "skillName": "Accuracy" } ],
    "move": { "category": "Dash", "from": [0, 0, 0], "position": [6, 4, 0] },
    "attack": { "hit": { "skillName": "Accuracy" }, "damage": { "skillName": "Push" } },
    "actionSpeed": { "skillName": "Legerity" },
    "interact": { "kind": "reveal" }
  }
}
```

Esta mensagem tem **três destinos possíveis**, decididos nesta ordem:

1. **`interact.kind == "reveal"` com `targetIds`** → revela portas secretas. As paredes
   viram `revealed` na sessão e o `WallSegment` real é transmitido à mesa
   (`wall_revealed`, ver [`maps.md`](maps.md)). **Retorna sem ack nenhum.**
2. **Qualquer outro `interact` com `targetIds`** → interação com parede, resolvida em
   memória: `wall_state_changed` (por jogador, com gate de linha de visão; uma porta secreta
   não revelada vai **só para o mestre**) + recálculo de `visibility_updated`. Também
   **retorna sem ack**. Paredes fora do estado em memória são ignoradas **em silêncio**.
3. **Sem `interact`** → enfileira a ação do mestre e responde
   [`master_action_enqueued`](#master_action_enqueued) para a mesa.

> **`move` e `attack` ainda não são mapeados** (`buildMasterAction` tem TODOs explícitos
> aguardando o contrato do front). Mandar essas seções hoje é no-op silencioso. `targetIds`,
> `skills`, `actionSpeed` e `interact` funcionam.

**Erros:** `forbidden` · `invalid_payload` (`"invalid enqueue_master_action payload"`) ·
`match_not_started` (**só no caminho 3** — os caminhos de parede retornam antes dessa
checagem) · `game_error`.

---

## 5. Servidor → cliente

### `action_enqueued`

**Direção:** servidor → cliente. **Destino:** **só quem enviou** o `enqueue_action`.

```json
{ "type": "action_enqueued", "payload": { "actionId": "33333333-3333-4333-8333-333333333333" } }
```

**Deixou de ser `{}`.** Nomeia a MESMA ação que `action_queued` acabou de nomear para o
mestre. Sem o ID, o navegador de quem enviou não tinha como se referir ao que acabou de
mandar: não cancelava, não destacava na barra geral, não sabia que a próxima a abrir era
dela. É o mesmo buraco que `PendingReactions` fechou para o mestre — um ID que o cliente
não aprende é uma operação que ele não consegue invocar.

Continua sem dizer nada sobre o **conteúdo** da ação (arma, alvo, perícia) e continua sem
ser notícia para a mesa — isso é `action_queued`, master-only, logo abaixo.

**Disparado por:** `enqueue_action` aceito.

### `action_queued`

**Direção:** servidor → cliente. **Destino:** **SÓ O MESTRE.** Se o mestre não estiver
conectado, a mensagem simplesmente não é entregue a ninguém.

```json
{
  "type": "action_queued",
  "payload": {
    "actionId": "33333333-3333-4333-8333-333333333333",
    "actorId": "11111111-1111-4111-8111-111111111111",
    "bars": ["action", "move"]
  }
}
```

**É master-only porque a fila é secreta** (`combat-engine.md` § *As barras são públicas*: "a
fila é secreta; a barra e a ordem são públicas"). Um jogador que aprendesse o que está
pendente leria as intenções da mesa no wire.

**E é o único jeito de aprender o `actionId` que [`pull_action`](#pull_action) exige.**
Nenhuma outra superfície nomeia uma ação enfileirada. Sem esta mensagem, `pull_action` é
inalcançável a partir de um cliente real.

Nada aqui descreve o **conteúdo** da ação: arma, alvo, perícia e dados continuam do jogador
até o mestre abrir o turno. `bars` (`action` e/ou `move`) já é dedutível da ordem pública em
`bars_updated` — nomear aqui não revela nada novo.

**Disparado por:** `enqueue_action` aceito, logo após o ack de quem enviou.

### `bars_updated`

**Direção:** servidor → cliente. **Destino:** **mesa inteira**, em broadcast — *não* é
projetado por destinatário. Um jogador que não enxergasse a barra geral só descobriria que
era a vez dele depois que passou.

```json
{
  "type": "bars_updated",
  "payload": {
    "seq": 7,
    "prices": { "action": 14, "move": 12 },
    "characters": [
      {
        "characterId": "11111111-1111-4111-8111-111111111111",
        "actionBalance": -2.5,
        "moveBalance": 0,
        "actionSpeeds": [16, 14],
        "moveSpeeds": [12]
      }
    ],
    "order": [
      { "actorId": "22222222-2222-4222-8222-222222222222", "bars": ["action"], "key": 18 }
    ]
  }
}
```

| Campo | Notas |
|---|---|
| `seq` | **Ordena os snapshots, e o cliente PRECISA usar.** Isto é estado completo, entregue de uma goroutine destacada: dois opens em sequência rápida disputam o envio e o snapshot mais velho pode chegar por último. **Guarde o maior `seq` aplicado e DESCARTE qualquer coisa menor.** Não há evento posterior de auto-correção — o que fecha a rodada é o último `bars_updated` que a mesa recebe. |
| `prices` | Preço congelado da rodada por barra. Uma barra que ainda não precificou **está ausente do mapa**. |
| `characters[].actionBalance` / `moveBalance` | Saldo (crédito ou débito) — **fracionário**, não arredondado. |
| `characters[].actionSpeeds` / `moveSpeeds` | Velocidades que **já agiram** nesta rodada. Públicas, porque a média que produzem é o que ordena todo mundo. |
| `order` | Projeção de quem age em seguida, **maior `key` primeiro**. Carrega quem e em qual barra, e **nada que identifique a ação** — nem ID, nem arma, nem alvo, nem perícia. |

**Disparado por:** qualquer coisa que mova as barras — `enqueue_action`,
`open_next_action`, `pull_action`, `close_turn`, `change_round_mode`.

### `turn_opened`

**Direção:** servidor → cliente. **Destino:** **mesa inteira**.

```json
{
  "type": "turn_opened",
  "payload": {
    "turnId": "55555555-5555-4555-8555-555555555555",
    "actorId": "11111111-1111-4111-8111-111111111111",
    "actionType": ""
  }
}
```

> ⚠️ **`actionType` é sempre `""` hoje.** O campo existe na struct e `room.go` nunca o
> preenche, nos dois call sites que emitem `turn_opened`. O front **não deve** ramificar por
> ele.

É este evento que abre a janela de reação: quem está em `targetId` da ação pode mandar
[`attach_reaction`](#attach_reaction) a partir daqui. Mas o `targetId` **não viaja nesta
mensagem** — um jogador descobre que foi alvo pelo `resolution_updated` liquidado (depois) ou
pela narração do mestre. Essa é uma lacuna real do contrato, registrada em §8.

**Disparado por:** `open_next_action` e `pull_action`.
**Dispara em seguida:** `resolution_updated` master-only do turno aberto.

### `reaction_opened`

**Direção:** servidor → cliente. **Destino:** **mesa inteira** — de quem é a vez de narrar
é estado de mesa.

```json
{
  "type": "reaction_opened",
  "payload": {
    "turnId": "55555555-5555-4555-8555-555555555555",
    "reactionId": "44444444-4444-4444-8444-444444444444"
  }
}
```

Anuncia quem narra em seguida. **O cálculo que isso desencadeia continua master-only** — o
`resolution_updated` que vem junto é de turno aberto.

**Disparado por:** `open_reaction`.

### `resolution_updated`

**Direção:** servidor → cliente. **Destino: depende de dois eixos independentes — ver §6.**

```json
{
  "type": "resolution_updated",
  "payload": {
    "turnId": "55555555-5555-4555-8555-555555555555",
    "isSettled": false,
    "action": {
      "skillName": "Accuracy",
      "skillValue": 14,
      "diceRolled": [6, 8],
      "total": 20,
      "isCritical": false,
      "isCriticalFailure": false,
      "margin": 3
    },
    "targets": [
      {
        "targetId": "22222222-2222-4222-8222-222222222222",
        "avoided": false,
        "defended": true,
        "dodgeTotal": 12,
        "defenseTotal": 15,
        "rawDamage": 10,
        "defenseApplied": 3,
        "projectedDamage": 7,
        "reaction": {
          "kind": "repel",
          "total": 17,
          "reactionId": "44444444-4444-4444-8444-444444444444",
          "rung": "near_miss",
          "margin": -3,
          "difference": 3,
          "stopsAttack": false
        },
        "payouts": [
          {
            "amount": -3,
            "bias": 0,
            "applies": "action_speed",
            "source": "system",
            "againstKind": "anyone",
            "againstId": "00000000-0000-0000-0000-000000000000",
            "expiresAt": "next_turn",
            "reason": "repel: near miss penalty"
          }
        ]
      }
    ],
    "pendingReactions": [
      {
        "reactionId": "44444444-4444-4444-8444-444444444444",
        "actorId": "22222222-2222-4222-8222-222222222222",
        "kind": "dodge"
      }
    ],
    "errors": [
      {
        "subject": "22222222-2222-4222-8222-222222222222",
        "kind": "unknown_target",
        "detail": "action target is neither a character nor a wall segment"
      }
    ]
  }
}
```

| Campo | Notas |
|---|---|
| `isSettled` | `false` = turno aberto, cálculo provisório, **master-only**. `true` = turno fechado, é **este** o cálculo cujo dano foi aplicado. |
| `action` | O teste de acerto do atacante — **um golpe só**, compartilhado por toda a cadeia. `margin` só existe quando há CD única, isto é, **num ataque de alvo único**; num ataque multi-alvo o campo é **omitido**. |
| `action.diceRolled` | Os dados **efetivamente lidos**. Um teste enviesado rolou dois conjuntos; só o lido viaja. |
| `targets[].avoided` | O golpe **não acertou este alvo, por qualquer meio**: esquiva, fuga, aparo, ou um aparo anterior que parou a corrente. **Não** é "esquivou" — pergunte a `reaction.kind` se a distinção importa. |
| `targets[].projectedDamage` | **Projeção.** O HP só muda no fechamento do turno. |
| `targets[].reaction` | `null` quando nada foi aberto e as passivas (esquiva por reflexo, depois defesa) se aplicaram em silêncio. Uma passiva silenciosa não é resposta a reportar. |
| `reaction.rung` | `great_success` · `success` · `near_miss` · `failure` — **snake_case**, diferente de todo o resto do wire. Ausente fora de um aparo. |
| `reaction.margin` / `difference` | Valor zero **fora de um aparo** — todos os outros tipos leem contra CD plana, não contra a escada. |
| `targets[].payouts` | O que a reação **deste alvo rendeu**: o bônus ou a penalidade do aparar, a reserva da esquiva fechada. Ausente quando não rendeu nada, que é a maioria. **Sujeito à projeção** — ver §6. |
| `reaction.stopsAttack` | É a contribuição **deste** aparo, não se alguém antes na corrente já parou o ataque. |
| `pendingReactions` | Reações **anexadas e ainda não abertas**. **Sempre master-only**, mesmo num payload liquidado. É a lista de tarefas do mestre, não estado de mesa. Uma reação não aberta nunca vira passo da cadeia, então o ID dela não aparece em `targets[].reaction` — esta é a única superfície que o nomeia. |
| `errors` | **Sempre master-only.** Faltas do **motor**, não do jogo — ver abaixo. Ausente numa resolução limpa. |

**Um `payout` é um modificador acumulado no personagem**, escrito na ficha dele no fechamento
do turno. Os campos:

| Campo | O que é |
|---|---|
| `amount` | Ajuste **plano** no total. |
| `bias` | Vantagem/desvantagem **nos dados** (−1/0/+1). Moeda diferente de `amount`: muda COMO a rolagem é lida, não é um número que se some a ela. |
| `applies` | Que dimensão isso move: `action_speed` ou `dodge`. Há mais de um tipo de reserva no sistema e elas não são intercambiáveis. |
| `source` | `system` ou `master`. |
| `againstKind` | `anyone` · `only` · `all_but`. **É o ponto do payout**: diz QUEM pode contá-lo. |
| `againstId` | O personagem em que `only`/`all_but` se apoiam. **Zero UUID** em `anyone` — que é exatamente o que esse caso significa, não um buraco. |
| `expiresAt` | `end_of_turn` · `next_turn` · `end_of_round`. |
| `reason` | Texto para humano. Não parseie. |

> ⚠️ `applies`, `source`, `againstKind` e `expiresAt` são **snake_case**, como `rung`: são
> valores de enum do domínio serializados como estão, não tags de struct.

**`errors` não é mensagem de erro.** A presença de uma entrada não quer dizer que a operação
falhou: o turno resolveu, os números acima são reais, e **um pedaço da colisão está faltando
neles**. `kind` é o discriminador estável; `detail` é prosa para humano e **não deve ser
parseado**.

| `kind` | O que significa |
|---|---|
| `unknown_target` | Um alvo da ação que o motor não conseguiu classificar — nem personagem, nem parede. |
| `missing_sheet` | Um personagem no tabuleiro cuja ficha não chegou ao resolvedor. **Esse alvo não produziu entrada em `targets`.** `subject` nomeia a ficha faltante (pode ser a do **ator**). |
| `no_attack` | Guarda defensiva, inalcançável pelo único call site atual. |

**Disparado por:** `open_next_action`, `pull_action`, `attach_reaction`, `open_reaction`,
`edit_action`, `close_turn`.

### `action_edited`

**Direção:** servidor → cliente. **Destino:** **só o mestre** (é ack de `edit_action`).

```json
{
  "type": "action_edited",
  "payload": {
    "turnId": "55555555-5555-4555-8555-555555555555",
    "actionId": "33333333-3333-4333-8333-333333333333"
  }
}
```

Não carrega número nenhum: a resolução recomputada viaja no `resolution_updated` seguinte,
projetada como qualquer outra.

### `close_turn_refused`

**Direção:** servidor → cliente. **Destino:** **só o mestre** — nomeia reações que a mesa
ainda não viu.

```json
{
  "type": "close_turn_refused",
  "payload": {
    "turnId": "55555555-5555-4555-8555-555555555555",
    "pendingReactions": [
      {
        "reactionId": "44444444-4444-4444-8444-444444444444",
        "actorId": "22222222-2222-4222-8222-222222222222",
        "kind": "repel"
      }
    ]
  }
}
```

**Nada foi fechado.** Este payload é o conteúdo do diálogo de confirmação, computado pelo
servidor — o front desenha, o back decide que o diálogo é necessário (e é por isso que o
critério é verificável sem front nenhum).

**O retry é `close_turn` com `confirm: true`.** Ver [`close_turn`](#close_turn).

**Disparado por:** `close_turn` sem `confirm` com reações anexadas e não abertas.

### `turn_closed`

**Direção:** servidor → cliente. **Destino:** **mesa inteira** — que um turno acabou é
estado de mesa.

```json
{ "type": "turn_closed", "payload": { "turnId": "55555555-5555-4555-8555-555555555555" } }
```

**Os números viajam separado**, no `resolution_updated` projetado que vem em seguida.

**Disparado por:** `close_turn`. (Um turno fechado por `open_next_action`/`pull_action`
**não** emite `turn_closed` — ele se anuncia pelo `resolution_updated` liquidado e pelo
`turn_opened` do próximo.)

### `round_closed`

**Direção:** servidor → cliente. **Destino:** **mesa inteira**.

```json
{ "type": "round_closed", "payload": { "roundMode": "Race" } }
```

A rodada acabou porque **nada pendente ainda conseguia pagar** — nenhum turno foi aberto.

**Disparado por:** `open_next_action`, e só ele.

### `round_mode_changed`

**Direção:** servidor → cliente. **Destino:** **mesa inteira** — o regime é público: todo
mundo precisa saber se as barras estão correndo.

```json
{ "type": "round_mode_changed", "payload": { "mode": "Race" } }
```

### `scene_changed`

**Direção:** servidor → cliente. **Destino:** **mesa inteira**.

```json
{
  "type": "scene_changed",
  "payload": {
    "sceneId": "55555555-5555-4555-8555-555555555555",
    "category": "battle",
    "briefInitialDescription": "Arena"
  }
}
```

### `master_action_enqueued`

**Direção:** servidor → cliente. **Destino:** **mesa inteira** (broadcast, não master-only).

É o **eco literal** do `MasterActionPayload` recebido:

```json
{
  "type": "master_action_enqueued",
  "payload": {
    "targetIds": ["22222222-2222-4222-8222-222222222222"],
    "skills": [ { "skillName": "Accuracy" } ],
    "actionSpeed": { "skillName": "Legerity" }
  }
}
```

**Disparado por:** `enqueue_master_action` no caminho 3 (sem `interact`).

### `match_full_state`

**Direção:** servidor → cliente. **Destino:** quem **conecta ou reconecta**, sempre que a
partida já tem sessão viva (combate em andamento). **Não sai** enquanto a partida está só no
lobby — não há combate para sincronizar, e `buildMatchFullState` devolve `nil`.

`map_full_state` cobre o tabuleiro e só. Quem chegava no meio — ou reconectava, e o hook do
front reconecta até cinco vezes sozinho — ficava sem barras, sem regime, sem cena, sem turno
aberto e sem reações pendentes, até alguma coisa mudar por acaso. É essa lacuna que esta
mensagem fecha.

```json
{
  "type": "match_full_state",
  "payload": {
    "scene": {
      "sceneId": "66666666-6666-4666-8666-666666666666",
      "category": "battle",
      "briefInitialDescription": "Arena"
    },
    "roundMode": "Race",
    "bars": {
      "seq": 7,
      "prices": { "action": 14, "move": 12 },
      "characters": [
        {
          "characterId": "11111111-1111-4111-8111-111111111111",
          "actionBalance": -2.5,
          "moveBalance": 0,
          "actionSpeeds": [16, 14],
          "moveSpeeds": [12]
        }
      ],
      "order": [
        { "actorId": "22222222-2222-4222-8222-222222222222", "bars": ["action"], "key": 18 }
      ]
    },
    "openTurn": {
      "turnId": "55555555-5555-4555-8555-555555555555",
      "actorId": "11111111-1111-4111-8111-111111111111"
    },
    "resolution": {
      "turnId": "55555555-5555-4555-8555-555555555555",
      "isSettled": false,
      "action": { "skillName": "Accuracy", "skillValue": 14, "diceRolled": [6, 8], "total": 20, "isCritical": false, "isCriticalFailure": false },
      "targets": []
    }
  }
}
```

| Campo | Notas |
|---|---|
| `scene` | O payload de [`scene_changed`](#scene_changed) **inteiro** — `sceneId`/`category`/`briefInitialDescription`, os mesmos nomes, a mesma struct. Não é uma segunda forma para os mesmos três valores. Ausente (`omitempty`) quando a partida não tem cena ativa; leia `scene == null` como "sem cena", não como "cena sem nome". `category` é minúscula (`"battle"`/`"roleplay"`) e não é validada — ver `change_scene`. |
| `roundMode` | O regime do round ativo — `"Free"` ou `"Race"`, os mesmos valores de [`round_mode_changed`](#round_mode_changed). Vem `""` quando não há round ativo. Público: vai para todo mundo que conecta. |
| `bars` | O `bars_updated` **inteiro**, reaproveitado — não é uma segunda forma para manter em sincronia com a primeira. |
| `bars.seq` | ⚠️ **É o contador CORRENTE, não um novo.** O cliente guarda o maior `seq` já aplicado e descarta qualquer coisa menor; estampar um número novo aqui zeraria essa guarda numa reconexão — o primeiro `bars_updated` atrasado a chegar depois seria aplicado por cima de um estado mais novo. É por isso que a proteção do cliente contra snapshot atrasado atravessa a reconexão: o contador nunca reinicia. |
| `openTurn` | Ausente (`omitempty`) **para todo destinatário** — jogador ou mestre — quando a mesa está em "fechado e nada aberto", estado em que ela pode legitimamente estar. Quando presente, vai para **todo mundo** que conecta: quem é o ator da vez não é segredo. |
| `resolution` | O cálculo do turno aberto, **master-only**. Ausente para qualquer outro destinatário, e também ausente para o próprio mestre quando não há turno aberto. Mesmos dois eixos de `resolution_updated` (§6) — aqui só o eixo do TEMPO se manifesta, porque um snapshot de conexão sempre reflete um turno em aberto (`isSettled: false`); não existe um `match_full_state` de turno fechado. |

**Disparado por:** todo `register` (conexão OU reconexão) enquanto há sessão de partida —
logo depois de `room_state` e do `map_full_state` (se houver peças no tabuleiro), e antes do
`player_joined` que avisa os demais da chegada.

### `piece_moved` (também servidor → cliente, na ABERTURA do turno)

<a id="piece_moved-servidor"></a>

**Direção:** servidor → cliente. **Destino:** fog-gated, por destinatário.

O contrato completo de `piece_moved`/`piece_removed` — shape de `SlotPayload`, o campo `z`,
o par origem/destino do sync de tabuleiro — é do lobby/mapa e vive em
[`game-lobby.md`](game-lobby.md) e [`maps.md`](maps.md). Até aqui `piece_moved` era só
**cliente → servidor**: o navegador do jogador aplicando localmente um arraste e
sincronizando o resto da mesa. O que o motor de combate acrescenta:

**Quando o `Move` de uma ação de turno ABRE, o servidor aplica a posição sozinho e emite
`piece_moved` como AUTOR.** Antes, uma ação de mover acontecia no cálculo e a peça nunca
saía do lugar no tabuleiro — o fog nunca recalculava. Agora `applyOpenedMove` escreve a nova
posição na **abertura** do turno, não no fechamento: o dano ainda pode ser editado pelo
mestre depois de aberto, mas as reações que seguem dependem de onde a peça ESTÁ, e não podem
esperar. O caminho reaproveita o mesmo par `piece_moved`/`piece_removed` que o lobby já usa
— não um tipo novo.

**O gate de fog é o mesmo par de sempre — e tem um corte ANTES dele:**

| Quem | Recebe |
|---|---|
| Mestre | sempre `piece_moved` — sem gate, mesmo com `visible: false` |
| Jogador, peça marcada `visible: false` (oculta) | **nada.** `relayPieceMove` corta antes de sequer olhar linha de visão — nenhum jogador recebe `piece_moved`/`piece_removed` para uma peça oculta, mesmo quem enxergaria o destino a olho nu. |
| Jogador, peça visível, enxerga o **destino** | `piece_moved`, com a posição nova |
| Jogador, peça visível, só enxergava a **origem** (a peça "saiu de vista") | `piece_removed` |
| Jogador, peça visível, não enxergava nem origem nem destino | nada |

`senderId` vem **zero** (`00000000-…`) quando o autor é o servidor — nenhum navegador previu
esse movimento, então ninguém é pulado no dispatch. É assim que o cliente distingue "o
servidor moveu isto" (`senderId` zero) de "outro jogador moveu isto" (`senderId` = o UUID de
quem enviou).

O dono do personagem movido recebe, além disso, um `map_full_state` atualizado — a linha de
visão dele mudou, mesmo quando quem moveu a peça não foi ele (o mestre arrastando a peça de
um jogador, ou o motor aplicando um movimento resolvido).

**Disparado por:** `open_next_action` e `pull_action`, quando o turno que abre carrega um
`Move`. Só o ramo que **não testa** desloca — hoje `move.category` só aceita `Dash` e
`Shift`, e nenhum dos dois rola contra CD; um ramo com teste (um salto, um aperto, um pouso
em slot ocupado) não tem caso alcançável hoje, então não existe código para ele.

⚠️ **Uma reação de escape NÃO move a peça.** Só o `Move` da própria ação do turno é aplicado
aqui. Uma reação — inclusive `escape`/`escapeGuard`/`closedEscape`, que carregam `Move`
também — se anexa a um turno já aberto, e a regra de quando a peça dela sai do lugar não
está escrita em lugar nenhum. Decisão consciente, não esquecimento — ver §9.

⚠️ **A semântica de `Z` está em aberto.** `PieceMovedPayload.Z` é documentado como altura
virtual em metros; `Move.Position[2]` é o índice `z` da grade. São grandezas possivelmente
diferentes, e por isso o servidor **preserva o `Z` que a peça já tinha** em vez de
sobrescrevê-lo com `Move.Position[2]`. Escrever um horizontal sobre um vertical derrubaria
uma peça elevada ao chão a cada passo horizontal cujo `z` de grade for `0`. A pergunta
"`Move.Position[2]` é metro ou índice de grade?" precisa de resposta antes de qualquer
cliente escrever `Z`. Ver §9.

⚠️ **O movimento aplicado não revalida parede.** A checagem contra paredes com `move=true` e
`open=false` acontece no **enfileiramento** (`enqueue_action`, quando `move.from` é
não-zero — ver a tabela em `enqueue_action`), não de novo aqui na abertura. Ver §9.

### `error`

**Direção:** servidor → cliente. **Destino:** **só quem enviou** a mensagem que falhou.

```json
{
  "type": "error",
  "payload": { "code": "forbidden", "message": "only the master can perform this action" }
}
```

Catálogo completo em §7.

---

## 6. `resolution_updated` tem DOIS eixos

Este é o ponto do contrato mais fácil de implementar errado. **Não é um eixo de visibilidade,
são dois, e eles se compõem.**

### Eixo 1 — TEMPO: `isSettled`

| `isSettled` | Quem recebe |
|---|---|
| `false` (turno aberto) | **só o mestre.** Ninguém mais recebe mensagem nenhuma. |
| `true` (turno fechado) | **todo cliente conectado**, um payload por destinatário. |

Enquanto o turno está aberto, o cálculo é do mestre. É por isso que um jogador vê
`turn_opened` e **não** vê `resolution_updated` até o turno fechar.

### Eixo 2 — CLASSE: mestre / dono / resto

Quando `isSettled` é `true`, o servidor projeta **uma cópia por destinatário**. São
**exatamente três classes, não quatro**:

| Classe | Enxerga |
|---|---|
| **Mestre** | Tudo, sem projeção. |
| **Dono** do personagem naquele `targets[]` | A verdade não projetada sobre **o próprio personagem**. |
| **Todo o resto** | A versão rebaixada. |

> **O alvo de um ataque deliberadamente NÃO é uma classe.** Uma finta contra você não te
> avisa que era finta — privilegiar o alvo apagaria a finta do jogo.

### O que isso significa na prática

**Dois clientes de jogador recebem payloads DIFERENTES para o MESMO turno, e isso não é bug.**
O `turnId` é o mesmo, o `isSettled` é o mesmo, e `targets[0]` tem conteúdo distinto conforme
quem está lendo. Um front que faça cache de resolução por `turnId` global, compartilhado
entre sessões, está errado por construção — a resolução é **por destinatário**.

O que muda de uma classe para outra, dentro de cada `targets[]` que o destinatário **não**
possui:

1. **`reaction.kind` é rebaixado:**

   | Real | Chega a terceiros como |
   |---|---|
   | `closedDodge` | **`dodge`** |
   | `closedEscape` | **`escape`** |

   **O rótulo é o vazamento.** Se `closedDodge` chegasse público, ninguém precisaria deduzir
   nada — o nome já disse que havia Evasão embutida. Deduzir da barra pública é legítimo (uma
   fuga fechada cobra uma barra onde a padrão cobra duas, e `bars_updated` é público); ser
   avisado não é.

   Todos os outros tipos — `dodge`, `escape`, `escapeGuard`, `nothing`, `repel` — chegam com
   o nome verdadeiro.

2. **`pendingReactions` some.** Sempre, para todo mundo que não é o mestre.

3. **`errors` some.** Sempre, para todo mundo que não é o mestre.

**Os números NÃO somem.** `total`, `diceRolled`, `rawDamage`, `projectedDamage`,
`dodgeTotal`, `defenseTotal`, `rung`, `margin`, `difference` viajam para todo mundo — público
por omissão. "O oponente tem que deduzir pelos números" é impossível sem eles.

4. **`payouts` é retido — mas só junto com o rótulo.** Esta é a mesma condição do item 1, não
   uma segunda regra: a reserva da esquiva fechada é a outra metade daquele segredo, porque o
   tamanho da esquiva não gasta diz quanta Evasão foi embutida, então ela sai junto com o
   nome. **A penalidade do aparar não sai.** Ela nasce `againstKind: "anyone"`, um aparo nunca
   é rebaixado, e `reacoes.md` diz que *"vale contra todo mundo — qualquer um pode
   aproveitar"*: quem pode aproveitar precisa conseguir ler.

   Na prática, para um terceiro:

   | Reação | `reaction.kind` | `payouts` |
   |---|---|---|
   | `repel` (near miss) | `repel` | **chega** — a penalidade |
   | `repel` (great success) | `repel` | **chega** — o bônus, `againstKind: "only"` |
   | `closedDodge` | `dodge` | **some** |
   | `closedEscape` | `escape` | **some** |

### Nota: a finta segue o mesmo eixo do TEMPO, mas em outro documento

`Feint` não aparece em payload nenhum deste protocolo — nenhuma mensagem servidor→cliente
projeta a **declaração** de uma `action.Action` **de jogador** (`master_action_enqueued` é a
exceção do lado do mestre, mas projeta `action.MasterAction`, um tipo sem `Feint`; a lacuna
correspondente está em §9), e é por isso que a finta não tem onde aparecer aqui. Ela vive em `service.ProjectAction`, a mesma
função que a Action History REST chama, e segue exatamente este eixo do TEMPO: escondida
enquanto `isSettled` é `false`, revelada quando o turno fecha — quem caiu na finta descobre
dentro da resolução do MESMO turno (o sucesso foi contra um ataque falso, e o de verdade vem
em seguida), nunca meses depois olhando o histórico. Ver
[`match-history.md`](match-history.md).

## 7. Catálogo de erros

| `code` | Significado | Onde aparece |
|---|---|---|
| `invalid_message` | `"malformed JSON"` — o envelope não parseou. | Qualquer mensagem. |
| `unknown_type` | `"unrecognized message type"` | `type` fora do catálogo. |
| `invalid_payload` | O `payload` não casa com a struct daquele `type`. A mensagem nomeia qual. | Todas. |
| `forbidden` | `"only the master can perform this action"` | `open_next_action`, `pull_action`, `open_reaction`, `edit_action`, `close_turn`, `change_round_mode`, `change_scene`, `enqueue_master_action`. |
| `match_not_started` | `"match session not initialized"` — a partida não foi iniciada. | Todas as de partida. |
| `invalid_action` | Payload bem formado, conteúdo inválido: perícia/arma/categoria desconhecida, reação sem componente obrigatório, `actorId` ausente, `reactToId`/`reactionKind` desemparelhados. | `enqueue_action`, `attach_reaction`, `edit_action`. |
| `move_blocked` | `"movement blocked by a wall"` | `enqueue_action` com `move.from` não-zero. |
| `game_error` | O domínio recusou. A `message` é o texto do erro de domínio (tabelas por mensagem em §4). | Todas as de partida. |

**`error` nunca é broadcast.** Vai só para quem enviou a mensagem que falhou.

## 8. A sequência típica de um turno inteiro

```
JOGADOR A                    SERVIDOR                         MESTRE            JOGADOR B (alvo)
    │                            │                               │                    │
    ├─ enqueue_action ──────────►│                               │                    │
    │◄──── action_enqueued {} ───┤                               │                    │
    │                            ├──── action_queued ───────────►│                    │
    │                            │     {actionId, actorId, bars}  │   (MASTER-ONLY:   │
    │                            │                               │    é aqui que o    │
    │                            │                               │    mestre aprende  │
    │                            │                               │    o actionId)     │
    │◄─────────────── bars_updated (mesa) ──────────────────────►│◄──────────────────►│
    │                            │                               │                    │
    │                            │◄──── open_next_action ────────┤                    │
    │                            │         (ou pull_action {actionId})                │
    │◄────────────── turn_opened {turnId, actorId} (mesa) ──────►│◄──────────────────►│
    │                            ├──── resolution_updated ──────►│                    │
    │                            │     isSettled:false (MASTER-ONLY)                  │
    │                            │                               │                    │
    │                            │◄────────── attach_reaction ────────────────────────┤
    │                            ├──── resolution_updated ──────►│                    │
    │                            │     isSettled:false, pendingReactions[]            │
    │                            │                               │                    │
    │                            │◄──── open_reaction ───────────┤                    │
    │                            │      {reactionId de pendingReactions}              │
    │◄─────────── reaction_opened {turnId, reactionId} (mesa) ──►│◄──────────────────►│
    │                            ├──── resolution_updated ──────►│                    │
    │                            │     isSettled:false (MASTER-ONLY)                  │
    │                            │                               │                    │
    │                            │◄──── edit_action ─────────────┤   (opcional,       │
    │                            ├──── action_edited ───────────►│    N vezes)        │
    │                            ├──── resolution_updated ──────►│                    │
    │                            │                               │                    │
    │                            │◄──── close_turn {} ───────────┤                    │
    │                            ├──── close_turn_refused ──────►│  (SÓ se houver     │
    │                            │     {pendingReactions[]}      │   reação anexada   │
    │                            │      NADA FOI FECHADO         │   e não aberta)    │
    │                            │                               │                    │
    │                            │◄──── close_turn {confirm:true}┤                    │
    │                            │   ┌─ persiste turno + resolução liquidada          │
    │◄──────────── turn_closed {turnId} (mesa) ─────────────────►│◄──────────────────►│
    │◄── resolution_updated ─────┤                               │                    │
    │    isSettled:TRUE          ├──── resolution_updated ──────►│                    │
    │    PROJETADO p/ A          │     isSettled:TRUE, completo  │                    │
    │                            ├──── resolution_updated ────────────────────────────►
    │                            │     isSettled:TRUE, PROJETADO p/ B (payload ≠ o de A)
    │◄─────────────── bars_updated (mesa) ──────────────────────►│◄──────────────────►│
```

**O que muda se o mestre usar `open_next_action` em vez de `close_turn`:** o turno fecha
igual (com persistência e `resolution_updated` liquidado), mas **não sai `turn_closed`** — o
próximo `turn_opened` é o que anuncia a virada. E se nada pendente ainda puder pagar, sai
[`round_closed`](#round_closed) e nenhum turno novo abre.

## 9. O que este contrato ainda não entrega

Registrado aqui para que a Fase 6 não descubra na integração. Fontes:
[`../match/flows/05-lacunas.md`](../match/flows/05-lacunas.md) e `AGENTS.md` § Known Issues.

| Lacuna | Consequência para o front |
|---|---|
| **`turn_opened` não carrega `targetId`** | Um jogador não tem como saber, pelo wire, que foi alvo — só pelo `resolution_updated` liquidado (tarde demais para reagir) ou pela narração. A janela de `attach_reaction` depende de canal humano hoje. |
| **`turn_opened.actionType` é sempre `""`** | Não ramifique por ele. |
| **Não existe evento de HP de personagem** | O dano é persistido em `character_sheets` no fechamento do turno; a sidebar lê por REST. O caminho ao vivo é trabalho da Fase 6. |
| **A corrente de testes de `skills` não é executada** | `skills[].difficulty` é aceito e persistido, mas nenhuma margem atravessa de um teste para o próximo. A edição de perícias muda uma lista que ainda não decide nada. |
| **`ReboundDamage` nunca é aplicado ao ator** | Viaja no registro do turno, não vira dano. |
| **Armadura reduz zero** | Não existe entidade de armadura. A linha está codificada porque a forma importa. |
| **`move`/`attack` de `enqueue_master_action` não são mapeados** | No-op silencioso até o contrato do front fechar. |
| **Nenhuma mensagem servidor→cliente projeta a declaração de uma action de JOGADOR** | `ActionPayload` só existe no sentido cliente→servidor; o front aprende o que um jogador declarou pelo histórico REST, não pelo WS. (`master_action_enqueued` é a exceção do lado do mestre — ver abaixo — mas não carrega `ActionPayload`, e não tem `Feint`.) É por isso que `systemBias` — exposto em `match-history.md` — **não tem equivalente aqui**: não há onde. O argumento do "já é dedutível" também não valeria, porque `resolution_updated` emite só `diceRolled`, o conjunto efetivamente lido. É também por isso que a finta (§6, nota no fim) não tem superfície neste protocolo — ela só existe em `Action.Feint`, e nenhuma ação de MESTRE tem finta. |
| **NPC não age** | Ver §2. |
| **Uma reação de escape não move a peça** | Só o `Move` da própria ação do turno é aplicado ao tabuleiro (`applyOpenedMove`, ver `piece_moved` acima). `escape`/`escapeGuard`/`closedEscape` também carregam `Move`, mas anexam a um turno já aberto, e a regra de quando a peça sai do lugar nesse caso não está escrita em lugar nenhum. Decisão consciente deste PR, não esquecimento. |
| **A semântica de `Z` está em aberto** | `PieceMovedPayload.Z` é altura virtual em metros; `Move.Position[2]` é o índice `z` da grade — grandezas possivelmente diferentes, nunca reconciliadas. Por isso o servidor preserva o `Z` que a peça já tinha em vez de escrever `Move.Position[2]` sobre ele. Bloqueia qualquer cliente que queira escrever elevação até a pergunta "`Move.Position[2]` é metro ou índice de grade?" ser respondida. |
| **O mestre perde o `actionId` da fila ao reconectar** | [`action_queued`](#action_queued), **no instante do enfileiramento**, é o único emissor de um `actionId` para o mestre. [`match_full_state`](#match_full_state) não carrega fila nenhuma, e `bars_updated.order` diz na própria linha da tabela que não leva nada que identifique a ação. Depois de reconectar, o mestre recebe barras, cena, regime, turno aberto e resolução — e **zero IDs** das ações que já estavam enfileiradas. [`open_next_action`](#open_next_action) continua funcionando (não precisa de ID); [`pull_action`](#pull_action) fica **inalcançável** para tudo que entrou na fila antes da reconexão, e volta a ser alcançável só para o que for enfileirado depois. É o mesmo buraco que `action_queued` fechou, reaberto pelo caminho da reconexão — e o contraste é direto: as **reações pendentes** sobrevivem à reconexão (viajam dentro de `resolution`), as ações enfileiradas não. **Para quem desenha a tela:** uma superfície de antecipar ação não pode depender de uma lista que o cliente acumulou de `action_queued`; depois de qualquer reconexão ela estará incompleta, sem nenhum sinal de que está. A saída completa é um campo de fila **master-only** no `match_full_state` — não implementado aqui de propósito: se a interface vai ou não ter superfície de antecipar ação é decisão de produto, tomada à parte. |
| **O movimento aplicado não revalida parede** | A checagem de parede (`move=true`, `open=false`) roda no `enqueue_action`, quando `move.from` é não-zero — não de novo quando o movimento é de fato aplicado na abertura do turno. |
