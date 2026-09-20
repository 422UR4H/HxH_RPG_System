# Match / Motor de Combate — Protocolo WebSocket

**Status:** Fase 5 implementada (back). Este documento é o **contrato de handoff para a
Fase 6 do front** — foi escrito conferindo cada payload contra a struct em
`internal/app/game/message.go`, e os exemplos JSON abaixo foram **gerados serializando
essas structs**, não transcritos de memória.

**Servidor:** `cmd/game/` · **URL:** `ws://localhost:8081/ws?match_uuid=<uuid>&token=<jwt>&nickname=<name>`

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
    "feint": { "skillName": "Deception" },
    "move": {
      "category": "Dash",
      "from": [4, 4, 0],
      "position": [6, 4, 0],
      "speed": { "skillName": "Accelerate" },
      "charge": { "skillName": "Energy" }
    },
    "attack": {
      "weapon": "sword",
      "hit": { "skillName": "Accuracy" },
      "damage": { "skillName": "Strength" },
      "charge": { "skillName": "Energy" }
    },
    "defense": { "weapon": "sword", "rollCheck": { "skillName": "Defense" } },
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
    "repel": { "weapon": "sword", "rollCheck": { "skillName": "Repel" } }
  }
}
```

**`reactionKind` é declarado, nunca inferido.** As três fugas têm exatamente a mesma forma
— uma esquiva e um movimento — e custam três coisas diferentes; nenhuma inspeção do payload
as separa, porque o que as separa é a intenção do jogador.

| `reactionKind` | Componentes obrigatórios | Barras que cobra |
|---|---|---|
| `nothing` | — | nenhuma |
| `dodge` | `dodge` | nenhuma |
| `closedDodge` | `dodge` + entrada `Evasion` em `skills` | nenhuma |
| `escape` | `dodge` + `move` | `action` + `move` |
| `escapeGuard` | `dodge` + `move` | `action` + `move` |
| `closedEscape` | `dodge` + `move` + entrada `Evasion` em `skills` | `move` |
| `repel` | `repel` | `action` |

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
| `invalid_action` | `reaction "X" must carry a dodge` / `a move` / `a repel` / `an evasion skill entry`; `reaction kind "X" is not in the catalogue`. |
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
  "payload": { "category": "Battle", "briefInitialDescription": "Arena" }
}
```

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
    "attack": { "hit": { "skillName": "Accuracy" }, "damage": { "skillName": "Strength" } },
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
{ "type": "action_enqueued", "payload": {} }
```

Ack vazio: "recebemos". Nada sobre a ação, e nenhuma notícia para a mesa.

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
          "rung": "nearMiss",
          "margin": -3,
          "difference": 3,
          "stopsAttack": false
        }
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
| `reaction.rung` / `margin` / `difference` | Valor zero **fora de um aparo** — todos os outros tipos leem contra CD plana, não contra a escada. |
| `reaction.stopsAttack` | É a contribuição **deste** aparo, não se alguém antes na corrente já parou o ataque. |
| `pendingReactions` | Reações **anexadas e ainda não abertas**. **Sempre master-only**, mesmo num payload liquidado. É a lista de tarefas do mestre, não estado de mesa. Uma reação não aberta nunca vira passo da cadeia, então o ID dela não aparece em `targets[].reaction` — esta é a única superfície que o nomeia. |
| `errors` | **Sempre master-only.** Faltas do **motor**, não do jogo — ver abaixo. Ausente numa resolução limpa. |

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
    "category": "Battle",
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

> **Nota de implementação (back).** No domínio, `ProjectResolution` também esconde os
> *payouts* de uma reação — mas **só quando o rótulo foi rebaixado**, porque a reserva da
> esquiva fechada é a outra metade do mesmo segredo. A penalidade do aparar, que nasce
> `ScopeAnyone`, vale contra todo mundo e não é escondida. Isso ainda **não tem campo no
> wire**: nenhum payload carrega `payouts` hoje.

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
| **NPC não age** | Ver §2. |
| **Payouts não têm campo no wire** | A reserva da esquiva fechada e a penalidade do aparar existem no domínio e são persistidas, mas nenhum payload as carrega. `reaction.difference` é o que o front tem. |
