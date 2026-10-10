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
  - `move.from` **não é mais afetado por isso** (B6, spec §4.3 "B5, B6 e B10"): virou
    `*[3]int`, e `omitempty` funciona normalmente em ponteiro. Ausente = o ator não tinha
    peça no tabuleiro — não "zero não enviado".
- **O servidor nunca confia no cliente para** qual barra a ação paga (`speed.bar` é
  descartado), qual perícia mede a velocidade (é sempre Legerity), a perícia de velocidade
  de um movimento (vem da categoria), **qual perícia mede o acerto de um ataque (é sempre
  `Accuracy`)** ou os dados. **Os dados caem no servidor**, no instante em que a ação é
  aceita, e nunca são rolados de novo.

## 2. Quem é quem

| Termo | O que é |
|---|---|
| **Jogador / mestre** | `User` autenticado. É quem a conexão WS identifica. |
| **Personagem** | `character_sheets.uuid`. É a **entidade de combate**, e é o que vai em `actorId` e em `targetId`. |
| Ponte | Uma pessoa dirige vários personagens. O servidor checa que o personagem de `actorId` pertence a quem enviou. |

> **NPC é do mestre.** `indexParticipants` mapeia toda ficha de NPC (`player_uuid == null`,
> `master_uuid` preenchido) para o `master_uuid` dela em `charToPlayer` — o mesmo mapa que
> autoriza jogadores comuns, só que a chave que bate é a do mestre. `enqueue_action` com
> `actorId` = a ficha do NPC passa pela mesma checagem `charToPlayer[actorId] == playerUUID`
> de sempre, e quem casa é o mestre. Vale tanto para o NPC que já estava no roster quando a
> sessão nasceu (`InitMatchSessionUC`) quanto para o que entrou depois, ao vivo, por
> [`add_npc`](#add_npc) — os dois caem no mesmo `charToPlayer`, pelo mesmo mecanismo (Decisão
> 6 do plano). **Ficha de jogador continua negada ao mestre:** o dono ali é o jogador, e isso
> não muda por quem enviou a mensagem.
>
> O mesmo vale para [`attach_reaction`](#attach_reaction): o mestre reage **pelo NPC** que é
> alvo, pela mesma checagem `charToPlayer`. A ficha de jogador continua negada a ele.

## 3. Índice

**Cliente → servidor**

| Mensagem | Quem pode enviar |
|---|---|
| [`enqueue_action`](#enqueue_action) | jogador (pelo próprio personagem) |
| [`attach_reaction`](#attach_reaction) | jogador **alvo** da ação aberta, pelo próprio personagem; o mestre, pelo NPC alvo |
| [`open_next_action`](#open_next_action) | mestre |
| [`pull_action`](#pull_action) | mestre |
| [`open_reaction`](#open_reaction) | mestre |
| [`edit_action`](#edit_action) | mestre |
| [`close_turn`](#close_turn) | mestre |
| [`change_round_mode`](#change_round_mode) | mestre |
| [`change_scene`](#change_scene) | mestre |
| [`enqueue_master_action`](#enqueue_master_action) | mestre |
| [`add_npc`](#add_npc) | mestre |

**Servidor → cliente**

| Mensagem | Destino |
|---|---|
| [`action_enqueued`](#action_enqueued) | só quem enviou |
| [`action_queued`](#action_queued) | **só o mestre** |
| [`bars_updated`](#bars_updated) | mesa inteira |
| [`turn_opened`](#turn_opened) | mesa inteira |
| [`reaction_attached`](#reaction_attached) | quem reagiu **+ mestre** |
| [`reaction_opened`](#reaction_opened) | mesa inteira (projetado por destinatário) |
| [`resolution_updated`](#resolution_updated) | **mestre, ou mesa projetada** — ver §5 |
| [`action_edited`](#action_edited) | só o mestre |
| [`close_turn_refused`](#close_turn_refused) | só o mestre |
| [`turn_closed`](#turn_closed) | mesa inteira |
| [`character_hp_changed`](#character_hp_changed) | **mestre + dono da ficha** |
| [`round_closed`](#round_closed) | mesa inteira |
| [`round_mode_changed`](#round_mode_changed) | mesa inteira |
| [`scene_changed`](#scene_changed) | mesa inteira |
| [`master_action_enqueued`](#master_action_enqueued) | mesa inteira; **só o mestre** nas ações de peça |
| [`npc_added`](#npc_added) | mesa inteira |
| [`match_full_state`](#match_full_state) | quem conecta/reconecta, enquanto há sessão viva |
| [`piece_moved`](#piece_moved-servidor) (também servidor) | fog-gated, por destinatário |
| [`error`](#error) | só quem enviou |
| [`connection_replaced`](#connection_replaced) | a conexão antiga, quando a mesma conta conecta de novo |

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
| `move.from` | **Ignorado** (B6, spec §4.3 "B5, B6 e B10"). O servidor é dono do tabuleiro: a origem que ele checa é a **posição da PRÓPRIA peça do ator**, lida do board no instante do enfileiramento — nunca o que o payload manda aqui. Sem peça no tabuleiro, não há origem e não há checagem nenhuma. |
| `move.position` | Convenção de coordenada, igual em `move.from`: `[a, b, z]`, com `(a, b) = (col, row)` numa grade quadrada ou `(q, r)` axial numa hexagonal. `z` **não é lido** pelo servidor (ver a nota de `Z` mais abaixo, em `piece_moved`). |
| `interact.kind` | `open` · `close` · `toggle` · `lockpick` · `examine`. (`reveal` é master-only, por `enqueue_master_action`.) |
| `dodge.category` | **Descartado pelo mapper** — o campo existe no payload e nada o lê. Só `dodge.rollCheck` importa. |
| `attack.weapon`, `defense.weapon` | Nome do catálogo (`enum.WeaponName`). Ausente = desarmado. |
| `attack.hit.skillName` | **Derivado pelo servidor: sempre `Accuracy`.** O que o payload mandar é **validado** (nome desconhecido é recusado na fronteira, como sempre foi) e depois **substituído** — o jogador escolhe arma e alvo, nunca a perícia que lê o acerto. O front **não precisa mandar** nome de perícia aqui; mandar um não muda nada. A arma escolhida entra pela **proficiência**, não pela perícia — ver abaixo. |
| `attack.damage.skillName` | **Descartado.** O dano soma o **`Push`** do atacante, lido direto da ficha (`TurnResolver.actorDamageSkill`) — nunca a perícia que o payload manda. O campo continua **validado quando não-vazio** (`buildRollCheck` só chama `SkillNameFrom` se a string não for `""`, então `"damage": {}` passa em branco) mas não decide mais nada, o mesmo estado de `speed`. Trocar `Push` por outra perícia é prerrogativa do mestre: [`edit_action`](#edit_action) `damageSkill`. A escolha dele **não** aparece aqui — este campo continua sendo o do jogador, descartado. |

**Como o acerto é montado.** A perícia é `Accuracy`, sempre, e a arma entra pela
**proficiência**: o acerto soma o `proficiencyLevel` que o personagem tem **com a arma que
está empunhando** — o mesmo número que o
[catálogo de combate](character-sheet.md) publica por arma. Sem arma (`attack.weapon` ausente),
a proficiência lida é a de **`Fist`**, porque o golpe corporal é uma arma do catálogo como
qualquer outra. Quem não tem proficiência nenhuma com aquela arma soma **zero** — não existe
penalidade de arma destreinada; se um dia existir, é regra nova, não este caminho.

É o espelho do dano, que já funcionava assim: o dano rola os dados **da arma** e soma o `Push`
do atacante; o acerto rola o conjunto da partida sobre `Accuracy` e soma a **proficiência**
daquela arma.

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
    "dodge": {}
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
    "repel": { "weapon": "Sword" }
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
| `closedDodge` | `dodge` (+ `Evasion`, acrescentada pelo servidor) | nenhuma | — |
| `escape` | `dodge` + `move` | `action` + `move` | **Dash** |
| `escapeGuard` | `dodge` + `move` | `action` + `move` | **Dash** |
| `closedEscape` | `dodge` + `move` (+ `Evasion`, acrescentada pelo servidor) | `move` | **Shift** |
| `repel` | `repel` | `action` | — |

**As perícias da reação são derivadas pelo servidor.** `dodge.rollCheck.skillName` é sempre
`Reflex`, `repel.rollCheck.skillName` é sempre `Repel`, e `closedDodge`/`closedEscape` ganham a
entrada `{ "skillName": "Evasion" }` em `skills` quando ela falta (uma entrada `Evasion` já
presente não é duplicada). O que o payload mandar nesses campos é validado — nome de perícia
desconhecido continua recusado com `invalid_action` — e substituído: o mesmo estado do
`attack.hit`, que é sempre `Accuracy`. O front não escreve nome de perícia nenhum para reagir.

Payload mínimo de cada tipo, além de `actorId`, `reactToId` e `reactionKind`:

| `reactionKind` | Payload mínimo |
|---|---|
| `nothing` | — |
| `dodge`, `closedDodge` | `dodge: {}` |
| `escape`, `escapeGuard` | `dodge: {}`, `move: {category: "Dash", position}` |
| `closedEscape` | `dodge: {}`, `move: {category: "Shift", position}` |
| `repel` | `repel: {}` ou `repel: {weapon}` |

**A categoria de `move` das três fugas é validada no SERVIDOR, não sugerida.**
`ReactionKind.RequiredMoveCategory()` (`reaction_kind.go`) fixa o par; `action_mapper.go`
recusa o resto — um `move.category` diferente do exigido devolve `invalid_action` com
`reaction "X" must move with Dash, not Shift` (ou o par correspondente). O discriminador é
**fechado × aberto**, não defensivo × padrão: durante o `Dash` o personagem está "no ar" e
não consegue esquivar — exatamente o que a fechada existe para não fazer, e por isso ela
pisa com `Shift`, que `Brake` mede.

**Um `move` fora das três fugas também é recusado — checagem de presença, não de categoria.**
`ReactionKind.Displaces()` (`reaction_kind.go`) é o único lugar que sabe quais `reactionKind`
deslocam; `action_mapper.go` a consulta para recusar um `move` anexado a `dodge`, `closedDodge`
ou `nothing` com `reaction "X" must not carry a move`, **antes** de a reação ser anexada ao
turno. Sem essa checagem um cliente podia mandar uma esquiva **livre** carregando um `move` e
o servidor não recusava nada — o `move` ficava inerte no `Action` até alguém lê-lo e deslocar
a peça de uma reação que não custa nada. O fechamento do turno, único momento em que uma fuga
desloca, também não depende só da recusa do mapper: ele lê o veredito `escape` da resolução, e
esse veredito só existe para os `reactionKind` com `Displaces()`.

Uma reação **livre** (as que não cobram barra) não consome a ação que o personagem tinha na
fila e não rola em Desvantagem. Uma reação **cobrada** consome a ação enfileirada daquele
personagem em cada barra que cobra — e, se consumiu algo, a troca custa **Desvantagem**
(`SystemBias = -1`). Por barra que cobra, a ação consumida é a de **melhor chave** naquela barra
(`BestPendingFor`, a mesma escolha do escalonador); uma ação combinada, que está nas duas
barras, sai uma vez só. Os IDs consumidos vão ao dono e ao mestre em
[`reaction_attached`](#reaction_attached).

**Dispara:** [`reaction_attached`](#reaction_attached) (a quem reagiu e ao mestre; uma cópia só
quando quem reagiu é o mestre), depois [`resolution_updated`](#resolution_updated) **só para o
mestre** — o turno está aberto, logo `isSettled: false`. A reação aparece em
`pendingReactions`. **Não há broadcast**: a mesa não é avisada de que alguém reagiu.

**Erros**

| `code` | Quando |
|---|---|
| `invalid_payload` | `"invalid action payload"` |
| `invalid_action` | `"reaction requires react_to_id"` |
| `invalid_action` | `"a reaction needs both reactToId and reactionKind; an action needs neither"` |
| `invalid_action` | `"actorId is required: …"` |
| `invalid_action` | `reaction "X" must carry a dodge` / `a move` / `a repel`; `reaction kind "X" is not in the catalogue`; `reaction "X" must move with Y, not Z` (categoria de `move` errada — ver a matriz acima); `reaction "X" must not carry a move` (`move` presente numa reação que **não** desloca — `dodge`, `closedDodge` ou `nothing` — checagem de presença, distinta da de categoria acima). |
| `match_not_started` | Sessão inexistente. |
| `game_error` | `the reacting character does not belong to this player` · `no current turn in round` · `cannot open a reaction: turn already closed` · `only a target of the open action may react to it` · `reaction does not target the current action` · `this character already reacted to the open action` (segundo `attach_reaction` do mesmo personagem na mesma ação, de qualquer tipo, aberta ou não. Recusado antes de rolar ou cobrar: a fila e as barras não mudam). |

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
2. Se um turno fechou: persistência + [`turn_closed`](#turn_closed) — mesa — +
   [`resolution_updated`](#resolution_updated) **settled e projetado** do turno que acabou
   (é essa a resolução cujo dano foi aplicado de verdade).
3. Se nenhuma ação na fila consegue mais pagar o preço, a rodada acaba:
   [`round_closed`](#round_closed) — mesa — e **para por aí**. O fim da rodada e a rodada que
   nasce no lugar dela são gravados **antes** do aviso, na mesma transação do turno que este
   comando fechou (ou numa só deles, se nenhum turno fechou) — um comando do mestre, uma
   transação.
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

**Dispara:** [`reaction_opened`](#reaction_opened) à **mesa**, **projetado por destinatário**
(de quem é a vez de narrar, e com o quê, é público; os números não) e [`resolution_updated`](#resolution_updated) **master-only** (o cálculo continua
sendo do mestre — o turno ainda está aberto). **Nunca** um
[`piece_moved`](#piece_moved-servidor): abrir uma fuga mostra a intenção, e a peça dela só anda
no fechamento do turno — ver abaixo.

#### Uma fuga só escapa se a esquiva E o movimento passarem — e a peça espera o fechamento

As três fugas (`escape`, `escapeGuard`, `closedEscape`) são as únicas reações com `Move` —
[`attach_reaction`](#attach_reaction) recusa qualquer outro `reactionKind` que tente carregar
um. **Toda reação tem uma CD, e ela é a ação que está sendo movida contra quem reage: o teste
de acerto do atacante** — o mesmo número contra o qual todo teste defensivo do turno já é
lido, e que chega ao mestre como `action.total` em
[`resolution_updated`](#resolution_updated). **Nada é rolado a mais para isso:** o acerto já
existe, e o lado de quem foge é a esquiva e a velocidade que já foram derivadas quando a reação
chegou.

Uma fuga é uma esquiva que se move, e o movimento é um teste **próprio** contra a MESMA CD. A
regra é **uma só**, para as três fugas e para as duas categorias:

- **esquiva passa** quando `reaction.total` `>=` acerto do atacante;
- **movimento passa** quando a velocidade derivada do movimento — o `Accelerate` rolado no
  `Dash`, o `Brake` lido passivo no `Shift` — `>=` acerto do atacante;
- **escapou** = **as duas** passaram. Só então o golpe não pega (`avoided: true`). Falhar em
  qualquer uma é não escapar, e o golpe é lido como se o alvo tivesse ficado: o `escapeGuard`
  ainda cai para a defesa; `escape` e `closedEscape` tomam o golpe inteiro.

O veredito viaja em [`resolution_updated`](#resolution_updated), em `targets[].escape`
(`escaped`, `movePassed`, `dodgePassed`, `awaitsMaster`, `landing`) — master-only enquanto o
turno está aberto, como o resto da resolução; projetado para todos depois que fecha.

**Como qualquer fuga pode falhar, nenhuma desloca na abertura** — nem a de `Shift`, que antes
pisava ali por não rolar nada. `open_reaction` mostra a intenção; a peça anda no
**fechamento do turno**, igual pelos três verbos que fecham (ver [`close_turn`](#close_turn)):

| No fechamento, a fuga... | A peça |
|---|---|
| **escapou** | vai ao **destino** pedido (`move.position`) |
| **falhou**, e o mestre escolheu onde ela caiu ([`edit_action`](#edit_action) `escapeLanding`) | vai para **onde o mestre escolheu** |
| **falhou**, sem escolha | **fica** onde estava |

Onde uma fuga que falhou vai parar **ainda não é regra do motor** — é parte do desenho da
colisão, que não existe. Até existir, quem decide é o mestre, como parte de resolver o turno
(não é um arrasto). O servidor não inventa posição intermediária.

> **Regra conhecida, ainda não implementada:** o movimento **soma** à esquiva, o que torna
> escapar mais fácil que esquivar parado. Entra com o desenho da colisão, antes das duas
> comparações acima.

O fechamento lê o veredito `escape` da própria resolução, que só existe para os `reactionKind`
com `Displaces()` — uma segunda checagem, redundante com a recusa do `attach_reaction`, para o
caso de um `Move` chegar a outra reação por qualquer outro caminho.

- **Um ator sem peça no tabuleiro é no-op silencioso**, como no lado da ação: nada é emitido,
  nenhum erro sai.
- **`Z` e o `Kind` do slot** (quadrado/hex) são **preservados**, pela mesma razão documentada em
  [`piece_moved`](#piece_moved-servidor) para o movimento de ação — no destino e na queda
  escolhida pelo mestre. É o mesmo `applyMove` da ação do turno, não um segundo caminho.
- O deslocamento de uma reação **nunca passa pela checagem de parede** — a colisão contra
  parede ainda não foi desenhada como regra; ver a última linha de §10.

⚠️ **O movimento de uma AÇÃO não mudou.** Uma ação não tem CD vindo contra ela; a peça dela
desloca **na abertura do turno**, qualquer que seja a categoria — a posição não pode esperar,
porque as reações seguintes dependem de onde a peça está.

**Erros:** `forbidden` · `invalid_payload` (`"invalid open_reaction payload"`) ·
`match_not_started` · `game_error` (`no current turn in round`,
`cannot open a reaction: turn already closed`, `reaction not found on the current turn`).

### `edit_action`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

Edita a ação do turno aberto, ou uma das reações dela. Toda seção é opcional e independente
— mande só o que muda.

- **`skills` e `targetIds` substituem a lista inteira**: não há merge parcial, porque o wire não
  tem identidade por entrada.
- **`conditions` se aplica por rolagem.** Cada entrada nomeia uma rolagem (`field` ou
  `skillName`) e substitui a condição **daquela** rolagem; as rolagens que não aparecem ficam
  como estão.
- **Uma entrada zerada limpa.** `bias`, `modifier` e `description` ausentes ou zero/vazios
  (`{ "field": "hit" }`) devolvem a rolagem a "sem condição" — e, se a edição anterior tinha sido
  capturada, apagam a captura: é assim que o mestre **cancela** uma edição. Não há verbo de
  confirmação nem de cancelamento.

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
| `bias` | Vantagem/desvantagem nos dados (−1 / 0 / +1). **Não** é somado ao total: escolhe qual conjunto de dados é lido. **Recusado no dano** (`field: "damage"`): o dano rola um conjunto só de dados, não há o que escolher — `game_error` `damage has no advantage: a damage condition takes no bias`. O `modifier` no dano soma ao dano bruto (piso zero) e vale para toda a cadeia de alvos. |
| `modifier` | Ajuste plano no total. |
| `damageSkill` | A perícia que mede o dano do ataque da ação (padrão **`Push`**). Qualquer perícia válida do enum (perícia desconhecida → `invalid_action`). Ausente = não mexe; `"Push"` devolve ao padrão e apaga a captura. Só numa ação com ataque — senão `game_error` `damageSkill edit targets an action with no attack`. O original vai para `overridden_action_values` com origem `system`. Exemplo: `{ "type": "edit_action", "payload": { "damageSkill": "Grab" } }`. |
| `escapeLanding.position` | Onde a peça de uma **fuga que falhar** vai parar — ver abaixo. `null` limpa. |

**`escapeLanding` — a queda da fuga que falha é do mestre.** O motor não tem regra para onde
uma fuga que falhou vai parar (ver [`open_reaction`](#open_reaction)); o mestre decide, por esta
mesma superfície de edição. `actionId` nomeia a **reação de fuga**:

```json
{ "type": "edit_action", "payload": { "actionId": "44444444-4444-4444-8444-444444444444", "escapeLanding": { "position": [5, 2, 0] } } }
```

```json
{ "type": "edit_action", "payload": { "actionId": "44444444-4444-4444-8444-444444444444", "escapeLanding": { "position": null } } }
```

- `position` é uma posição de grade, na mesma forma de `move.position` (`[col, row, z]`; em
  hex, axial `[q, r, z]`). O `z` não muda a altura da peça, como em todo `piece_moved`.
  **`null` limpa** a escolha (chave ausente dentro de `escapeLanding` vale o mesmo): a fuga que
  falhar volta a ficar onde estava.
- Vale só para uma **reação de fuga do turno aberto** (`escape`, `escapeGuard`,
  `closedEscape`) — anexada, aberta ou não — e para uma posição **dentro da grade** (quando o
  tabuleiro tem dimensões; em hex, a coluna é `q + floor(r/2)`, como no front).
- Pode ser escolhida **a qualquer momento com o turno aberto**, esteja a fuga passando ou
  falhando agora: ela pode passar a falhar (ou a passar) quando o mestre abre outras reações ou
  edita uma leitura. **Só é usada se, no fechamento, a fuga falhou** — uma fuga que escapa vai
  ao destino, qualquer que seja a escolha guardada, e a resolução só mostra `landing` enquanto a
  fuga está falhando.
- É uma seção como as outras: pode vir **sozinha** ou junto de `conditions`/`skills`/`targetIds`,
  e aí valem as duas. **Tudo é validado antes de qualquer mutação**: uma `escapeLanding` recusada
  não deixa a condição do mesmo payload aplicada.
- Volta [`action_edited`](#action_edited) e o [`resolution_updated`](#resolution_updated)
  recomputado, com `targets[].escape.landing` preenchido (e `awaitsMaster: false`).

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
`condition edit must set either field or skillName, not both`,
`escapeLanding only applies to an escape reaction on the open turn`,
`escapeLanding position is outside the grid`, `damage has no advantage: a damage condition takes no bias`,
`damageSkill edit targets an action with no attack`).

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

Fechar um turno **não fecha a rodada**. Só `open_next_action` detecta que nenhuma ação na fila
consegue mais pagar o preço — o que acaba a rodada.

**Dispara** (no caminho que de fato fecha):

1. [`piece_moved`](#piece_moved-servidor), **um por fuga cuja peça sai do lugar** — a que
   escapou, ou a que falhou com queda escolhida pelo mestre. Ver abaixo.
2. Persistência do turno (ação, reações, overrides, resolução liquidada).
3. [`turn_closed`](#turn_closed) — mesa.
4. [`resolution_updated`](#resolution_updated) **settled e projetado por destinatário**.
5. [`bars_updated`](#bars_updated) — mesa.

#### O fechamento é onde a peça de toda fuga é decidida

Nenhuma fuga deslocou na abertura (ver [`open_reaction`](#open_reaction)). É aqui que o
servidor aplica o veredito da resolução liquidada — o mesmo `targets[].escape` que a mesa lê:

- **escapou** (esquiva e movimento `>=` acerto) → sai um
  [`piece_moved`](#piece_moved-servidor) para o slot pedido;
- **falhou, com [`escapeLanding`](#edit_action)** → sai um `piece_moved` para onde o mestre
  escolheu;
- **falhou, sem escolha** (`awaitsMaster: true`) → **nada sai**. A peça fica onde estava.

O `piece_moved` tem o mesmo gate de campo de visão e sai **antes** do
[`turn_closed`](#turn_closed) — a mesa não pode ver o turno acabar com a peça no slot velho.

⚠️ **Os três verbos que fecham um turno decidem isso igual.** `close_turn` é o explícito;
[`open_next_action`](#open_next_action) e [`pull_action`](#pull_action) também fecham o turno
aberto a caminho de abrir o próximo, e **os três** aplicam o desfecho das fugas. Um escape
cujo resultado dependesse de qual verbo o mestre usou seria bug, não regra. (Nesses dois o
`piece_moved` sai antes da persistência e antes do [`turn_opened`](#turn_opened) do próximo
turno, e o [`turn_closed`](#turn_closed) sai igual ao do `close_turn` — pela mesma razão.)

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

⚠️ **`category` é minúscula.** `enum.SceneCategory` vale `"battle"` ou `"roleplay"`.
**O servidor valida**: `enum.SceneCategoryFrom` compara a string contra a lista exata do enum
(`internal/domain/entity/enum/scene_category.go`), no mesmo padrão de `SkillNameFrom`/
`WeaponNameFrom` — comparação exata, sem normalização de caixa. Mandar `"Battle"` devolve
`invalid_action` em vez de criar uma cena cuja categoria não bate com nenhum dos dois valores.
(`roundMode`, ao contrário, é capitalizado: `"Free"`/`"Race"`.)

Fecha cena e rodada correntes e abre uma cena nova com a primeira rodada dentro.

**Dispara:** [`scene_changed`](#scene_changed) — mesa.

**Erros:** `forbidden` · `invalid_payload` (`"invalid change_scene payload"`) ·
`invalid_action` (`"invalid name of scene category: X"` — `category` fora de
`"battle"`/`"roleplay"`) · `match_not_started` ·
`game_error` (`cannot close round: current turn is still open`).

### `enqueue_master_action`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

```json
{
  "type": "enqueue_master_action",
  "payload": {
    "targetIds": ["22222222-2222-4222-8222-222222222222"],
    "skills": [ { "skillName": "Accuracy" } ],
    "actionSpeed": { "skillName": "Legerity" }
  }
}
```

Os campos são `targetIds`, `skills`, `actionSpeed`, `move`, `remove`, `interact`. **Não há
`attack`** (B9, spec §2, decisão fechada com o dono do produto): o mestre ataca pelo NPC, com
[`enqueue_action`](#enqueue_action) — um `attack` aqui é recusado, sempre, com `invalid_action`
`"the master attacks through an NPC with enqueue_action"`, decidido a partir do JSON bruto
antes de qualquer outro caminho abaixo (um cliente que ainda manda essa chave não pode ser
ignorado em silêncio). Um "ataque do mestre" só faria sentido como efeito de ambiente — uma
armadilha —, o que é **futuro**, não pendência. **Quais dos campos restantes vêm decide o
caminho** (lista abaixo) — em especial, `move` ou `remove` fazem dela uma ação de peça, antes
de qualquer `interact`.

As duas formas **de peça** (spec §4.3, "Master action de peça" — B14 e o `move` de B9):

```json
{ "type": "enqueue_master_action", "payload": { "targetIds": ["<sheetUuid>"], "move": { "position": [3, 4, 0] } } }
{ "type": "enqueue_master_action", "payload": { "targetIds": ["<sheetUuid>"], "remove": {} } }
```

Esta mensagem tem **quatro destinos possíveis**, decididos nesta ordem:

0. **`move` ou `remove`** → ação de **peça**, só com partida em andamento (no lobby o mestre
   usa `piece_moved`/`piece_removed`, ver [`match-maps.md`](match-maps.md)). `targetIds` tem
   **exatamente um** id: o **personagem** (UUID da ficha). **Aplica na hora, com ou sem turno
   aberto**; com turno aberto, a ação também é pendurada nele. Três casos:
   - **Arrastar** — o personagem tem peça: ela vai para `move.position` (`[col, row, z]`, ou
     `[q, r, z]` em grade hex), mantendo a forma do slot e o `z` que já tinha (ver a nota de
     `Z` em [`piece_moved`](#piece_moved-servidor)). `move.category`, `speed`, `charge` e
     `from` são **ignorados**: o arrastar não é movimento de jogo, não rola nem cobra barra.
   - **Pôr** — o personagem não tem peça: ganha uma nova (`pieceId` novo, forma do slot pela
     grade, `visible: true`, `z` 0). Se é **NPC do mestre** que ainda não participa, é
     inscrito na partida antes — o mesmo caminho de [`add_npc`](#add_npc): a mesa recebe
     [`npc_added`](#npc_added) e um `bars_updated` novo. Se é personagem de **jogador** que
     não participa → `error` `not_participant`, nada é criado.
   - **Tirar** (`remove`) — a peça sai do tabuleiro. Só isso: não desinscreve, não mata, não
     apaga histórico; o personagem continua nas barras. O dono da peça (jogador) recebe um
     `map_full_state` com a linha de visão refeita — a peça era um dos pontos de onde ele via.

   Sai `piece_moved` (arrastar, pôr) ou `piece_removed` (tirar) com a projeção de fog de
   [`piece_moved`](#piece_moved-servidor) **para todos, inclusive o mestre** — a tela dele
   espera essa confirmação; o `senderId` vem zero (autor servidor). Depois, o mestre recebe
   [`master_action_enqueued`](#master_action_enqueued) — **só ele**: o eco carrega a posição,
   e para peça oculta isso vazaria para a mesa. O tabuleiro é salvo — na hora sem turno
   aberto; **com turno aberto, só no fechamento dele** (ver abaixo).
1. **`interact.kind == "reveal"` com `targetIds`** → revela portas secretas. As paredes
   viram `revealed` na sessão e o `WallSegment` real é transmitido à mesa
   (`wall_revealed`, ver [`maps.md`](maps.md)). **Retorna sem ack nenhum**, exceto pelo
   `unknown_wall` abaixo.
2. **Qualquer outro `interact` com `targetIds`** → interação com parede, resolvida em
   memória: `wall_state_changed` (por jogador, com gate de linha de visão; uma porta secreta
   não revelada vai **só para o mestre**) + recálculo de `visibility_updated`. Também
   **retorna sem ack**. Um `targetId` que EXISTE no tabuleiro mas cujo `interact.kind` não se
   aplica a ele (`lockpick`/`examine` — exigem rolagem, ainda não implementado) é ignorado em
   silêncio; um `targetId` que o servidor não conhece de jeito nenhum responde `unknown_wall`
   (última defesa, spec §4.3) — em ambos os casos o resto do lote em `targetIds` continua
   sendo processado.
3. **Nenhum dos anteriores** → enfileira a ação do mestre no turno aberto e responde
   [`master_action_enqueued`](#master_action_enqueued) para a mesa.

**Toda master action aceita é gravada e aparece no histórico como cada jogador a viu** (spec
§4.8; ver [`match-history.md`](match-history.md)). **Quando** ela vira registro depende do
turno (decisão do dono do produto, 2026-10-01):

- **sem turno aberto** — gravada no instante em que é aplicada, e o tabuleiro salvo junto;
- **com turno aberto** — o registro é montado no instante (o que cada jogador viu, o `turnId`,
  a hora), mas só é **gravado no fechamento do turno**, na mesma transação do turno
  ([`close_turn`](#close_turn), [`open_next_action`](#open_next_action) ou
  [`pull_action`](#pull_action) — os três). O tabuleiro idem: com turno aberto, nem a master
  action de peça nem a interação/revelação de parede o salvam; o fechamento o grava, com elas
  dentro, **na mesma transação** do turno e das master actions. Tudo o que acontece dentro de
  um turno aberto fica durável junto com o fechamento dele, ou não fica: um reinício no meio
  do turno perde o turno **inteiro** — o movimento da abertura e as master actions do mestre
  dentro dele (ver [§9](#reinício-recarga-queda), que lista também as exceções). As ações do
  caminho 3 só existem com turno aberto, então são sempre gravadas no fechamento.

Ao vivo nada muda: a mesa recebe tudo na hora, com ou sem turno. O servidor grava, jogador a jogador da sessão
(conectado ou não), a mesma decisão de fog que tomou ao vivo: quem recebeu `piece_moved` (ou
a parede mudando, a remoção, a revelação) vê a ação inteira; quem só recebeu o
`piece_removed` de um arrasto a vê **sem o destino**; quem não recebeu nada não a vê. As
ações do caminho 3 não chegam à mesa como ação, então ficam só para o mestre. Uma master
action **recusada** não é gravada, e [`edit_action`](#edit_action) **não** é master action
(fica em `overridden_action_values`, como sempre).

**Erros:** `forbidden` · `invalid_payload` (`"invalid enqueue_master_action payload"`) ·
`invalid_action` (`"the master attacks through an NPC with enqueue_action"` — chave `attack`
presente, checado antes de tudo abaixo e independente de partida em andamento) ·
`match_not_started` (**caminhos 0 e 3** — os caminhos de parede retornam antes dessa
checagem) · `invalid_action` (**caminho 0**: `targetIds` sem exatamente um id; `move` e
`remove` juntos; `remove` de personagem sem peça — `"character has no piece"`; **pôr**
quando um segundo socket do mestre acabou de criar a peça entre a leitura de posse e a
gravação — `"character already has a piece"`, a mesma seção crítica que a leitura de
`applyMove` deixa como indecisa) ·
`not_participant` (**caminho 0**, pôr personagem de jogador que não participa) · os erros de
[`add_npc`](#add_npc) (**caminho 0**, pôr NPC que ainda não participa: `forbidden`,
`not_found`, `invalid_npc`, `npc_already_in_match`) · `unknown_wall` (**caminhos 1 e 2** —
um `targetId` de parede que o servidor não conhece; desde B14, quem carrega o tabuleiro é o
próprio servidor, então "não conhece" agora também cobre uma partida sem tabuleiro nenhum) ·
`game_error`.

### `add_npc`

**Direção:** cliente → servidor. **Quem:** **só o mestre** (`forbidden` para os demais).

```json
{
  "type": "add_npc",
  "payload": { "characterSheetUuid": "44444444-4444-4444-8444-444444444444" }
}
```

`characterSheetUuid` é o MESMO nome de campo do REST
(`POST /matches/{uuid}/npcs`, ver [`match-npcs.md`](match-npcs.md)) — o front manda a mesma
chave nos dois caminhos.

**Convive com o REST, não o substitui.** REST monta o roster ANTES de existir sala —
preparação de campanha, sem jogo em andamento. Este verbo é para o MEIO da partida: o game
server já tem o pool do Postgres, então roda o **mesmo** `AddMatchNPCUC` que o REST usa
(mesmas guardas, mesma escrita em `match_participants`) e, se houver sessão viva, injeta a
ficha nela em seguida — um ato só do ponto de vista do mestre, não uma chamada HTTP ao
endpoint REST seguida de uma injeção manual que o front poderia esquecer.

**Com sessão viva:** a ficha entra em `charSheets`, `statuses` e `charToPlayer` (dono =
mestre) da `MatchSession` corrente — o mestre pode enfileirar uma ação pelo NPC no mesmo
segundo (ver §2). O servidor responde [`npc_added`](#npc_added) e emite um `bars_updated`
NOVO (`seq` maior) que já lista o NPC — **não** `match_full_state` (ver
[`npc_added`](#npc_added) em §5). **As duas mensagens saem cada uma da sua própria goroutine**
(`go func() { r.broadcast <- data }()`, em `room.go`, tanto para `npc_added` quanto dentro de
`broadcastBars`), então a ORDEM DE CHEGADA **não é garantida** — `bars_updated` pode chegar
antes de `npc_added`. O front não deve exigir `npc_added` primeiro para aceitar o
`bars_updated` que já lista o NPC.

**Sem sessão viva (partida ainda no lobby):** só a escrita no banco acontece. `npc_added`
ainda sai para a mesa, mas não há barras para publicar — nenhum `bars_updated` é emitido. O
NPC entra em jogo quando a sala nascer: `InitMatchSessionUC` o carrega do banco junto com o
resto do roster.

**A duplicata que este verbo recusa é da SESSÃO, não do banco.** Todas as guardas de
`AddMatchNPCUC` (mestre da partida, partida não encerrada, ficha existe, é NPC,
elegibilidade) rodam ANTES do INSERT que devolveria `ErrNPCAlreadyInMatch`. Por isso o verbo
WS **não trata esse erro do banco como falha**: "já está no banco" quer dizer "passou em
tudo, só a sessão está atrasada" — exatamente o caso de quem pôs o NPC pelo REST no meio da
partida e precisa que a sala viva alcance o banco. O verbo segue e injeta do mesmo jeito.
Quem recusa é a SESSÃO, quando o NPC já está nela (`statuses[sheetUUID]` já existe) — aí o
mestre recebe `npc_already_in_match` e nenhum `npc_added` sai.

> ⚠️ **Na virada lobby→partida, QUALQUER ack de `add_npc` pode enganar — em dois sentidos
> opostos.** `StartMatch` chama `InitMatchSessionUC.Init` (lê `match_participants` do banco,
> SEM segurar `r.mu`) e só depois toma `r.mu.Lock()` para publicar `r.session`. Isso abre uma
> janela onde a leitura do roster e a escrita do `add_npc` podem se intercalar dos dois jeitos:
>
> - **Falso `npc_already_in_match`.** A partida começa e o `Init` já leu a linha nova ANTES de
>   a sessão publicada pegar o lock que a injeção de `add_npc` também disputa. O mestre recebe
>   `npc_already_in_match` sem nunca ter recebido `npc_added` para aquele NPC — mas o NPC ESTÁ
>   na sessão, carregado pelo próprio `Init`. O estado está correto; só o ack é que engana.
> - **Falso `npc_added` (o espelho).** O `Init` lê o roster ANTES do INSERT do `add_npc`
>   confirmar, e o braço `add_npc` pega `r.mu.Lock()` ANTES de `StartMatch` publicar `r.session`
>   — nesse instante `r.session` ainda é `nil`, então o braço aplica a semântica de lobby: grava
>   no banco, responde `npc_added`, sem `bars_updated` nenhum (parece o caminho feliz do §3 da
>   Decisão 3). Mas a sessão que nasce um instante depois foi montada a partir de uma leitura
>   ANTERIOR ao INSERT — ela nasce SEM o NPC, apesar do ack de sucesso.
>
> As duas corridas exigem DOIS sockets de mestre concorrentes (uma segunda aba, ou uma
> reconexão que se sobrepõe à sessão anterior) — um único socket processa suas próprias
> mensagens em ordem, então não colide consigo mesmo. **O front deve tratar QUALQUER ack de
> `add_npc` na virada lobby→partida como não-definitivo, e reenviar `add_npc` é sempre
> seguro:** a duplicata do banco é tolerada (Decisão 2) e a duplicata da sessão responde
> `npc_already_in_match` — que, pelo caso acima, significa "o NPC está na partida" nos dois
> sentidos em que pode aparecer.

**O mesmo caminho serve o "pôr" da master action de peça** ([`enqueue_master_action`](#enqueue_master_action)
com `move` para um NPC que ainda não participa, spec §4.3 B11): mesma inscrição, mesmo
`npc_added`, mesmo `bars_updated`, mesmos erros — e só depois a peça é criada.

**Erros:** `forbidden` (`"only the master can perform this action"`) · `invalid_payload`
(`"invalid add_npc payload"` — `characterSheetUuid` ausente/zero, ou payload que não é um
objeto) · `not_found` · `invalid_npc` · `npc_already_in_match` · `game_error` (catálogo
completo em §7).

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
    "bars": ["action", "move"],
    "action": {
      "uuid": "33333333-3333-4333-8333-333333333333",
      "actorId": "11111111-1111-4111-8111-111111111111",
      "targetId": ["22222222-2222-4222-8222-222222222222"],
      "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity", "skillValue": 0, "attempts": { "primary": [6, 8] }, "result": 14 } },
      "move": {
        "category": "Dash",
        "from": [4, 4, 0],
        "position": [6, 4, 0],
        "speed": { "skillName": "Accelerate", "skillValue": 0, "attempts": { "primary": [5, 7] }, "result": 12 },
        "finalSpeed": 12
      },
      "attack": {
        "weapon": "Sword",
        "hit": { "skillName": "Accuracy", "skillValue": 0, "attempts": { "primary": [6, 8] }, "result": 14 },
        "damage": { "skillName": "Push", "skillValue": 0, "attempts": { "primary": [4] }, "result": 4 },
        "relativeVelocity": 0
      }
    }
  }
}
```

**`action.move.from` é a posição da PEÇA do ator no tabuleiro** (B6, spec §4.3 "B5, B6 e B10"),
nunca o que o payload de `enqueue_action` mandou em `move.from` — por isso o exemplo mostra
`[4, 4, 0]` mesmo que o cliente tivesse mandado outra coisa. **Ausente** (nenhuma chave
`from`) quando o ator não tem peça no tabuleiro.

**É master-only porque a fila é secreta** (`combat-engine.md` § *As barras são públicas*: "a
fila é secreta; a barra e a ordem são públicas"). Um jogador que aprendesse o que está
pendente leria as intenções da mesa no wire.

**E é o jeito de aprender o `actionId` que [`pull_action`](#pull_action) exige, no instante do
enfileiramento.** Sem esta mensagem, `pull_action` é inalcançável a partir de um cliente real
para o que acabou de entrar na fila.

**`action` carrega a declaração inteira** (design spec §4.2, B1) — arma, alvo, perícias,
dados, totais, ambas velocidades. É o mesmo formato do histórico REST
([`match-history.md`](match-history.md)), no **nível cheio**
(`actionwire.Full`; ver [`internal/app/wire/actionwire`](../../../internal/app/wire/actionwire)).
Isso é seguro exatamente porque esta superfície já era master-only: o mestre já veria estes
mesmos números na abertura do turno — `action` só antecipa essa visibilidade para o instante
do enfileiramento, não a estende a ninguém novo. `bars` (`action` e/ou `move`) continua
dedutível da ordem pública em `bars_updated`; `action` é o que passou a não ser mais
redundante com ela.

**Disparado por:** `enqueue_action` aceito, logo após o ack de quem enviou.

> **Esta mensagem dispara UMA VEZ, no instante do enfileiramento** — um mestre que estava
> desconectado nesse momento não a recebe depois. [`match_full_state`.`queue`](#match_full_state)
> é a versão do MESMO fato que **sobrevive à reconexão**: um payload `action_queued` inteiro
> por ação ainda pendente, na ordem de inserção da fila. Ver a seção de `match_full_state`.

### `reaction_attached`

**Direção:** servidor → cliente. **Destino:** quem reagiu **+ o mestre** — uma cópia só quando
o mestre é quem reagiu.

Responde um [`attach_reaction`](#attach_reaction) aceito.

```json
{ "type": "reaction_attached", "payload": { "turnId": "5555…", "reactionId": "4444…", "actorId": "2222…", "consumedActionIds": ["3333…"] } }
```

| Campo | Tipo | Significado |
|---|---|---|
| `turnId` | uuid | O turno aberto a que a reação foi anexada. |
| `reactionId` | uuid | O ID da reação — o que [`open_reaction`](#open_reaction) pede de volta e o que `ownReactions` indexa. |
| `actorId` | uuid | O personagem que reagiu (o NPC, quando o mestre reage por ele). |
| `consumedActionIds` | uuid[] | As ações da fila que uma reação cobrada consumiu. |

**Destino:** quem reagiu **e** o mestre — a mesma mensagem, o mesmo conteúdo. Quando quem
reagiu é o mestre (reação de NPC), ele recebe **uma** cópia só. **A mesa não recebe nada**: que
alguém reagiu não é notícia de mesa até o mestre abrir a reação.

**`consumedActionIds` é sempre lista:** `[]` numa reação livre ou numa cobrada que não achou
nada na fila — nunca `null`, nunca ausente.

**Uso:** o front tira a ação consumida da lista de declaradas (dono) e da fila (mestre); ela
**não** foi perdida — ver a regra de reconciliação em [`match_full_state`](#match_full_state).

**Disparado por:** `attach_reaction` aceito, **antes** do `resolution_updated` do mesmo attach
(mesma goroutine).

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

**Direção:** servidor → cliente. **Destino:** mesa inteira — todo mundo recebe UMA cópia —
mas, desde B2 (design spec §4.2), o `action` dentro dela **é projetado por destinatário**:
não é mais broadcast no sentido de "byte idêntico para todos". Por isso saiu de
`r.broadcast` para a pista direta (`dispatchPerPlayer`), um payload construído por cliente —
ver a ⚠️ de ordem mais abaixo.

```json
{
  "type": "turn_opened",
  "payload": {
    "turnId": "55555555-5555-4555-8555-555555555555",
    "actorId": "11111111-1111-4111-8111-111111111111",
    "actionId": "33333333-3333-4333-8333-333333333333",
    "actionType": "",
    "action": {
      "uuid": "33333333-3333-4333-8333-333333333333",
      "actorId": "11111111-1111-4111-8111-111111111111",
      "targetId": ["22222222-2222-4222-8222-222222222222"],
      "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity", "skillValue": 0, "attempts": { "primary": [6, 8] }, "result": 14 } },
      "feint": { "skillName": "Feint", "skillValue": 0, "attempts": { "primary": [3, 5] }, "result": 8 },
      "trigger": {},
      "attack": {
        "weapon": "Sword",
        "hit": { "skillName": "Accuracy", "skillValue": 0, "attempts": { "primary": [6, 8] }, "result": 14 },
        "damage": { "skillName": "Push", "skillValue": 0, "attempts": { "primary": [4] }, "result": 4 },
        "relativeVelocity": 0
      }
    }
  }
}
```

**O exemplo acima é o que o MESTRE vê** (`actionwire.Full` — ver
[`internal/app/wire/actionwire`](../../../internal/app/wire/actionwire)).

**O que o DONO do ator vê** (`actionwire.Opened`, mesmo turno ainda aberto) — `hit`/`damage`
perdem os números, `speed` mantém o `result`, `feint` fica só com `skillName` (o dono nunca é
segredo dele mesmo, mas Opened corta os números de qualquer forma), e `trigger` continua
presente (o dono vê tudo do próprio ator, `Viewer.SeesAllOf`):

```json
{
  "type": "turn_opened",
  "payload": {
    "turnId": "55555555-5555-4555-8555-555555555555",
    "actorId": "11111111-1111-4111-8111-111111111111",
    "actionId": "33333333-3333-4333-8333-333333333333",
    "actionType": "",
    "action": {
      "uuid": "33333333-3333-4333-8333-333333333333",
      "actorId": "11111111-1111-4111-8111-111111111111",
      "targetId": ["22222222-2222-4222-8222-222222222222"],
      "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity", "skillValue": 14, "attempts": { "primary": [6, 8] }, "result": 14 } },
      "feint": { "skillName": "Feint" },
      "trigger": {},
      "attack": {
        "weapon": "Sword",
        "hit": { "skillName": "Accuracy" },
        "damage": { "skillName": "Push" },
        "relativeVelocity": 0
      }
    }
  }
}
```

> ⚠️ Note o `speed.rollCheck.skillValue: 14` acima — igual ao `result` neste exemplo, mas por
> coincidência dos números, não por regra: `skillValue` é cortado pelo MESMO `keep` que
> `result` (ver `rollCheck` em `internal/app/wire/actionwire/from.go`), então em `Opened` os
> dois sobrevivem juntos ou somem juntos; nunca um sem o outro.

**O que um TERCEIRO (bystander) vê** — mesmo corte de números que o dono (`Opened`), mas SEM
`feint` (deny-list temporal de `ProjectAction`, turno ainda aberto) e SEM `trigger` (deny-list
permanente, `ProjectAction` sempre zera `Trigger` para quem não é dono nem mestre):

```json
{
  "type": "turn_opened",
  "payload": {
    "turnId": "55555555-5555-4555-8555-555555555555",
    "actorId": "11111111-1111-4111-8111-111111111111",
    "actionId": "33333333-3333-4333-8333-333333333333",
    "actionType": "",
    "action": {
      "uuid": "33333333-3333-4333-8333-333333333333",
      "actorId": "11111111-1111-4111-8111-111111111111",
      "targetId": ["22222222-2222-4222-8222-222222222222"],
      "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity", "skillValue": 14, "attempts": { "primary": [6, 8] }, "result": 14 } },
      "attack": {
        "weapon": "Sword",
        "hit": { "skillName": "Accuracy" },
        "damage": { "skillName": "Push" },
        "relativeVelocity": 0
      }
    }
  }
}
```

Nenhum dos dois exemplos acima tem `skillValue`/`attempts`/`result` em `hit`/`damage` — a
diferença ENTRE eles é só `feint` e `trigger`, e nenhuma delas é o corte de nível: as duas são
o deny-list de `service.ProjectAction`, uma temporal (some enquanto o turno está aberto,
`feint`) e uma permanente (some sempre para quem não é dono nem mestre, `trigger`) — ver a
tabela dedicada a `feint` mais abaixo.

| Campo | O que é |
|---|---|
| `turnId` | O turno que abriu. É ele que aparece em [`turn_closed`](#turn_closed) e em [`resolution_updated`](#resolution_updated). |
| `actorId` | UUID da ficha de quem age — o mesmo ID que a peça do tabuleiro carrega. |
| `actionId` | **A ação que este turno abriu.** É o **mesmo** ID que [`action_enqueued`](#action_enqueued) devolveu a quem enfileirou e que [`action_queued`](#action_queued) deu ao mestre: é o que liga as três mensagens. |
| `action` | **A mecânica pública da ação, cortada por destinatário** (B2, design spec §4.1/§4.2). Ver a tabela de cortes abaixo. |

> **Para que serve `actionId`:** `actorId` sozinho é ambíguo assim que um personagem tem
> **duas** ações na fila — os dois `turn_opened` saem idênticos e nada no wire diz qual
> delas abriu. Com o `actionId`, o cliente casa o turno com a ação que ele próprio
> enfileirou (ou, no mestre, com a linha da fila que ele antecipou por
> [`pull_action`](#pull_action)).
>
> **A fila continua secreta.** O `actionId` só vira público aqui, quando a ação **saiu** da
> fila e virou turno aberto — cuja existência já era pública pelo próprio `turn_opened`.
> Nada é dito sobre o que continua pendente, e as operações que tomam um `actionId`
> ([`pull_action`](#pull_action)) seguem sendo só do mestre.

> ⚠️ **`actionType` é sempre `""` hoje.** O campo existe na struct e `room.go` nunca o
> preenche, no único call site que emite `turn_opened` (`announceOpenedTurn`, onde
> `open_next_action` e `pull_action` desembocam). O front **não deve** ramificar por ele.

#### O corte de `action` (design spec §4.1)

`action` nunca sai no nível `Declaration` aqui — só existem dois destinatários possíveis para
esta mensagem, e nenhum deles é "alguém que só viu a própria declaração sem rolar nada":

| Campo | Mestre (`actionwire.Full`) | Todo mundo mais, inclusive o dono do ator (`actionwire.Opened`) |
|---|---|---|
| Alvos, arma, `move.category`, nomes das perícias, `reactionKind`, `interact`, `spread`, `relativeVelocity`, `systemBias` | ✔ | ✔ |
| `move.from` (origem) e `move.position` (destino) | ✔ | ✔ para o **dono**; para os outros, **só o que a fog da peça deixa ver** — ver [abaixo](#onde-a-peça-vai-movefrom-e-moveposition-seguem-a-fog) |
| `speed.rollCheck` (actionSpeed) e `move.speed`/`move.finalSpeed` (moveSpeed) — as duas linhas de "velocidade" do spec | ✔ números | ✔ números — **não** são cortados neste nível |
| Dados, `skillValue` e `result` de `attack.hit`, `attack.damage`, `attack.charge`, `move.charge`, `skills[]`, `defense`, `dodge`, `repel` | ✔ | corta — só `skillName` sobrevive |

**`feint` não está na tabela acima porque não segue só o corte de NÍVEL — segue primeiro o
eixo do TEMPO de `service.ProjectAction`** (a mesma função e a mesma regra que
[`match-history.md`](match-history.md) descreve para o REST, rodando aqui pela primeira vez
no lado do WebSocket):

| Destinatário | `feint` em `turn_opened` (turno ainda ABERTO) |
|---|---|
| Mestre | objeto completo, com números (`actionwire.Full`) |
| **Dono** do ator (`Viewer.SeesAllOf` verdadeiro) | objeto presente, só `skillName` — o dono nunca é o alvo da própria finta, então nada aqui esconde nada DELE |
| Qualquer outro (terceiro/bystander) | **ausente por completo** — `ProjectAction` zera o campo enquanto `isSettled` é `false`; reaparece (com números cortados igual ao resto) só quando o turno fecha, no `resolution_updated` liquidado equivalente — a finta nunca teve superfície própria ali, ver a nota do §6 |

**`trigger` segue a mesma lógica de dois eixos, mas sem o eixo do TEMPO** — é presença-só (o
domínio ainda não tem campos em `action.Trigger`) e o deny-list dele não depende de
`isSettled`:

| Destinatário | `trigger` em `turn_opened` |
|---|---|
| Mestre | `{}` quando a ação tem gatilho |
| **Dono** do ator | `{}` quando a ação tem gatilho — mesma razão da finta: `Viewer.SeesAllOf` |
| Qualquer outro (terceiro/bystander) | **ausente por completo, SEMPRE** — `ProjectAction` zera `Trigger` incondicionalmente para quem não é dono nem mestre, aberto ou fechado; ao contrário da finta, não há revelação depois que o turno fecha |

#### Onde a peça vai: `move.from` e `move.position` seguem a fog

**Decisão do dono do produto (2026-10-01): o movimento não chega a quem não deveria ver a
peça.** `move.from` é a posição REAL da peça no servidor (B6) e `move.position` é onde ela
está agora — copiados para todo mundo, diziam a um jogador onde estava e para onde foi uma
peça que a fog dele (ou `visible: false`) esconde, enquanto o relay ao vivo do MESMO
movimento ([`piece_moved`/`piece_removed`](#piece_moved-também-servidor--cliente-na-abertura-do-turno)) não dizia nada.

Para quem **não é mestre nem dono do ator**, os dois campos passam pelo **mesmo gate** do
relay ao vivo (`pieceMoveView`, com os polígonos de visão em cache daquele jogador, o destino
em `move.position` e o `visible` da peça). **A origem que o gate julga** é, no `turn_opened`
ao vivo, **a casa em que a peça estava quando o turno abriu** — a mesma que o relay julgou;
não `move.from`, que é lido no **enfileiramento** (B6) e fica velho se a peça andou no meio
(arrasto do mestre, uma ação anterior do mesmo personagem). Numa reconexão
([`openTurn`](#match_full_state)) essa casa não está guardada, e a origem julgada é o próprio
`move.from`.

| O que o destinatário vê | `move` |
|---|---|
| O destino (o relay mandou `piece_moved`) | inteiro: `category`, `from`, `position` |
| Só a origem (o relay mandou `piece_removed`) | `category` e `from` — **sem `position`**; e `from` só se for **essa mesma origem** — se a peça andou desde o enfileiramento, `move.from` aponta para uma casa que o relay nunca mostrou, e sai também |
| Nenhum dos dois, **ou** a peça é `visible: false`, **ou** o ator não tem peça no tabuleiro (o mestre a tirou depois do enfileiramento: nada anda, o relay não manda nada) | só `category` — **sem `from` nem `position`** |

Mestre e dono do ator recebem o `move` inteiro, sempre. `category` fica para todos: **que** o
ator se move é mecânica pública; **onde** não é. Os demais campos de `move` (`speed`,
`finalSpeed`) seguem o corte de nível acima, sem mudança.

**O histórico segue esta mesma regra.** No `turn_opened` ao vivo, o servidor grava o veredito
deste gate para **todo jogador da sessão** (conectado ou não; mestre e dono do ator não são
gravados), com a mesma origem julgada, e o grava com o fechamento do turno
(`actions.move_views`). O [`GET /history`](match-history.md#o-movimento-como-foi-visto-ao-vivo--movefrom-moveposition-escapelanding)
mostra a cada leitor o `move` da action exatamente como esta tabela o deu a ele ao vivo — nada
é recalculado com o fog da hora da leitura; um turno gravado antes disso falha fechado (só
`category`).

Para um `Dash` de `[4, 4, 0]` a `[6, 4, 0]` (o do exemplo de [`action_queued`](#action_queued)), um terceiro que não enxerga nenhuma das pontas recebe:

```json
"move": {
  "category": "Dash",
  "speed": { "skillName": "Accelerate", "skillValue": 0, "attempts": { "primary": [5, 7] }, "result": 12 },
  "finalSpeed": 12
}
```

e um que via só a origem recebe o mesmo com `"from": [4, 4, 0]` — sem `position`.

> ⚠️ **O front tem que tolerar um `move` sem `from` e sem `position`.** `position` deixou de
> ser sempre presente neste protocolo (no TS: `position?: [number, number, number]`) — e
> também no [histórico REST](match-history.md), onde o `move` da action vem cortado pelo
> veredito gravado ao vivo (ver acima). Para o mestre e o dono do ator ele continua inteiro.

É este evento que abre a janela de reação: quem está em `action.targetId` pode mandar
[`attach_reaction`](#attach_reaction) a partir daqui. **`targetId` agora viaja nesta
mensagem** (dentro de `action`) — antes de B2 este era um gap real do contrato (§10); deixou
de ser.

**Disparado por:** `open_next_action` e `pull_action`.
**Dispara em seguida:** `resolution_updated` master-only do turno aberto.

> ⚠️ **Ordem garantida com `turn_closed` e `resolution_updated` (B2).** `turn_closed` →
> `resolution_updated` (liquidado) → `turn_opened` chegam nessa ordem exata a todo
> destinatário — os três viajam pela pista DIRETA (`dispatchPerPlayer`/`client.SendMessage`),
> nunca por `r.broadcast`, e são enviados pelo MESMO goroutine (o que processa
> `open_next_action`/`pull_action`/`close_turn`). Uma pista só, um remetente só: ordem de
> envio é ordem de chegada.
>
> **`piece_moved` entra na sequência em DOIS pontos possíveis, não um só:** o da fuga (do
> turno que está fechando — a que escapou, ou a que falhou com queda escolhida pelo mestre) sai
> ANTES de `turn_closed`; o da própria ação que está
> abrindo (quando ela tem `move`) sai DEPOIS do `resolution_updated` liquidado e ANTES deste
> `turn_opened` — são dois personagens e dois momentos diferentes, ver a nota equivalente em
> [`turn_closed`](#turn_closed) para o detalhe de onde cada um entra no código.
>
> Antes de B2 só `piece_moved` e `resolution_updated` (settled) estavam na pista direta;
> `turn_closed` e `turn_opened` ainda saíam por `r.broadcast` — nenhum dos dois, `turn_opened`
> incluído — e a ordem entre eles e o resto não era prometida.

### `reaction_opened`

**Direção:** servidor → cliente. **Destino:** **mesa inteira**, **uma cópia por destinatário**
— de quem é a vez de narrar, e com o quê, é estado de mesa (Fase 7, item 1; design spec §4.1).
Como `reaction` é **projetada por destinatário**, a mensagem viaja pela **pista direta**
(`dispatchPerPlayer`), um payload construído por cliente, e não por `r.broadcast`.

**O que o MESTRE vê** (`actionwire.Full`) — uma `closedEscape` (`Shift`) de `[6, 6, 0]` para
`[8, 6, 0]`, com o rótulo fechado, a entrada `Evasion` e os números:

```json
{
  "type": "reaction_opened",
  "payload": {
    "turnId": "55555555-5555-4555-8555-555555555555",
    "reactionId": "44444444-4444-4444-8444-444444444444",
    "reaction": {
      "uuid": "44444444-4444-4444-8444-444444444444",
      "actorId": "22222222-2222-4222-8222-222222222222",
      "reactToId": "33333333-3333-4333-8333-333333333333",
      "reactionKind": "closedEscape",
      "skills": [
        { "skillName": "Evasion", "rollCheck": { "skillName": "Evasion", "skillValue": 0, "attempts": { "primary": [7, 9] }, "result": 16 } }
      ],
      "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity", "skillValue": 0, "attempts": {}, "result": 11 } },
      "move": {
        "category": "Shift",
        "position": [8, 6, 0],
        "speed": { "skillName": "Brake", "skillValue": 0, "attempts": {}, "result": 11 },
        "finalSpeed": 11
      },
      "dodge": { "rollCheck": { "skillName": "Reflex", "skillValue": 0, "attempts": { "primary": [7, 9] }, "result": 16 } }
    }
  }
}
```

**O que um TERCEIRO vê** (nem mestre nem dono do reator) — a mesma reação, `Opened` depois de
`ProjectAction`: o rótulo rebaixado a `escape`, sem a entrada `Evasion`, sem `consumedActionIds`,
sem os números de `dodge`; as **velocidades ficam** (decisão D1). `move.position` está aqui
**só porque este terceiro vê o destino** — ver o `move` abaixo:

```json
"reaction": {
  "uuid": "44444444-4444-4444-8444-444444444444",
  "actorId": "22222222-2222-4222-8222-222222222222",
  "reactToId": "33333333-3333-4333-8333-333333333333",
  "reactionKind": "escape",
  "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity", "skillValue": 0, "attempts": {}, "result": 11 } },
  "move": {
    "category": "Shift",
    "position": [8, 6, 0],
    "speed": { "skillName": "Brake", "skillValue": 0, "attempts": {}, "result": 11 },
    "finalSpeed": 11
  },
  "dodge": { "rollCheck": { "skillName": "Reflex" } }
}
```

#### O corte de `reaction`

O mesmo de [`turn_opened`](#o-corte-de-action-design-spec-41) — a mesma função
(`turnActionWireLocked`), aplicada à reação:

| Destinatário | `reaction` |
|---|---|
| Mestre | `actionwire.Full` — inteira, com números |
| **Dono** do reator | `actionwire.Opened` depois de `ProjectAction`: `reactionKind` fechado (`closedDodge`/`closedEscape`) e a entrada `Evasion` **ficam**; dados, `skillValue` e `result` de `skills[]`, `dodge`, `defense`, `repel`, `move.charge` saem; velocidades ficam; `consumedActionIds` **fica** (`ProjectAction` o mantém para dono e mestre) |
| Qualquer outro (terceiro) | o mesmo `Opened`, e ainda: `closedDodge`→`dodge`, `closedEscape`→`escape`; a entrada `Evasion` de `skills[]` e `consumedActionIds` **tirados** |

> **O rebaixamento do escape fechado é cosmético.** O rótulo `closedEscape`→`escape` não esconde
> o que a mecânica já mostra: a categoria do deslocamento é fixa por tipo (`closedEscape` move
> com **Shift**, medido pelo Brake; `escape`/`escapeGuard`, com **Dash**, medido pelo
> Accelerate), e `move.category`/`move.speed.skillName` viajam a todos no `reaction_opened`; a
> barra pública também mostra que o escape fechado cobrou só a barra de movimento. É a doutrina
> de sempre — deduzir pela barra pública é legítimo, ser avisado não é: o rótulo não é
> entregue, mas a mecânica permite deduzi-lo. Escondê-lo de verdade exigiria mexer nas barras e
> no corte da categoria, o que não está decidido.

#### O `move` da reação: o destino segue a fog

Uma fuga leva um `move`, e o `move.position` dela passa pelo **mesmo portão de fog** do
[`turn_opened`](#onde-a-peça-vai-movefrom-e-moveposition-seguem-a-fog) para quem não é mestre
nem dono do reator. **A origem julgada é a casa da peça do reator na abertura** — ela não anda
ao abrir (a peça de toda fuga espera o fechamento), então é também a casa que uma reconexão lê.

**Uma reação nunca tem `move.from`** (o servidor não o deriva para reação). Por isso o veredito
"só a origem" vira o mesmo que "nenhuma das pontas":

| O que o terceiro vê | `move` da reação |
|---|---|
| O destino | `category`, `position`, velocidades |
| Só a origem, nenhuma das duas, a peça é `visible: false`, ou o reator não tem peça no tabuleiro | `category` e velocidades — **sem `position`** |

Mestre e dono do reator recebem `position` sempre.

> ⚠️ **O front tem que tolerar `reaction.move` sem `position`** — o mesmo `position?` do
> `turn_opened`.

O servidor grava, na abertura, o veredito deste portão para **todo jogador da sessão**
(conectado ou não; mestre e dono do reator não são gravados), e o grava com o fechamento do
turno, na linha da própria reação — para o histórico mostrar a cada leitor o `move` da reação
como ele o viu ao vivo.

> ⚠️ **Ordem garantida:** `reaction_opened` chega ao mestre **antes** do
> [`resolution_updated`](#resolution_updated) recomputado pela abertura — mesma pista (a
> direta), mesmo goroutine. Ordem de envio é ordem de chegada.

**O cálculo que a abertura desencadeia continua master-only** — o `resolution_updated` que vem
junto é de turno aberto.

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
| `targets` | **Na ordem da cadeia**: primeiro os alvos cuja reação foi aberta, na ordem em que o mestre as abriu; depois os alvos sem reação aberta, na ordem de `action.targetId`. Vale para o payload aberto, o liquidado projetado, o `match_full_state.resolution` e o histórico. É assim que a ordem de abertura — que muda o desfecho — sobrevive à reconexão. Só personagens: parede não entra em `targets`. |
| `targets[].avoided` | O golpe **não acertou este alvo, por qualquer meio**: esquiva, fuga, aparo, ou um aparo anterior que parou a corrente. **Não** é "esquivou" — pergunte a `reaction.kind` se a distinção importa. |
| `targets[].attackStopped` | `true` quando um aparo **anterior** na corrente já tinha parado o golpe ao chegar a este alvo (`avoided` vem `true` junto). Distingue "esquivou" de "o golpe já tinha parado" quando `reaction.kind` é `nothing`. **Ausente** quando `false`. |
| `targets[].projectedDamage` | **Projeção.** O HP só muda no fechamento do turno. |
| `targets[].reaction` | **Ausente** quando nada foi aberto e as passivas (esquiva por reflexo, depois defesa) se aplicaram em silêncio — `Reaction *ReactionResultPayload` com `omitempty` **omite a chave**, não emite `null`; em TypeScript o campo é `reaction?: ReactionResultPayload`, não `reaction: ReactionResultPayload \| null`. Uma passiva silenciosa não é resposta a reportar. |
| `reaction.rung` | `great_success` · `success` · `near_miss` · `failure` — **snake_case**, diferente de todo o resto do wire. Ausente fora de um aparo. |
| `reaction.margin` / `difference` | Valor zero **fora de um aparo** — todos os outros tipos leem contra CD plana, não contra a escada. |
| `targets[].escape` | O veredito de uma **fuga** (`escape`, `escapeGuard`, `closedEscape`) — **ausente** fora delas. `escaped` = `movePassed` **e** `dodgePassed` (os dois contra o acerto do atacante); só aí `avoided` é `true`. Ver [`open_reaction`](#open_reaction). |
| `escape.awaitsMaster` | `true` = falhou **e** o mestre não escolheu onde a peça cai: no fechamento, ela fica onde está. Num payload liquidado (e no histórico), lê-se "ficou". |
| `escape.landing` | Onde o mestre pôs a peça ([`edit_action`](#edit_action) `escapeLanding`), `[col, row, z]` — **presente só enquanto a fuga falha** e há escolha. Uma fuga que escapa não o mostra, mesmo com escolha guardada: vai ao destino. **No payload liquidado, sujeito à fog** — ausente para quem não é mestre nem dono do personagem e não enxerga a casa onde a peça caiu (ver §6, item 5). |
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
estado de mesa, e a mesma mensagem vai para todo mundo (nada aqui é projetado). Desde B2
(design spec §4.2) viaja pela pista DIRETA (`dispatchPerPlayer`), não mais por `r.broadcast`
— ver a ⚠️ de ordem abaixo para o porquê.

```json
{ "type": "turn_closed", "payload": { "turnId": "55555555-5555-4555-8555-555555555555" } }
```

**Os números viajam separado**, no `resolution_updated` projetado que vem em seguida.

**Disparado por:** [`close_turn`](#close_turn), [`open_next_action`](#open_next_action) e
[`pull_action`](#pull_action) — os **três** verbos que fecham um turno.

> **O fechamento implícito emite igual ao explícito.** `open_next_action` e `pull_action`
> fecham o turno aberto a caminho de abrir o próximo, e esse fechamento sai com o mesmo
> `turn_closed`, no mesmo ponto da sequência: depois das fugas e da persistência, **antes**
> do [`turn_opened`](#turn_opened) do turno seguinte. A mesa tem que ver o turno acabar
> antes de ver o próximo começar — qual verbo o mestre usou não muda o que a mesa ouve.

> ⚠️ **A ordem contra `resolution_updated` e `turn_opened` AGORA é promessa (B2).** Antes de
> B2, `turn_closed` E `turn_opened` saíam por `r.broadcast` enquanto `resolution_updated`
> (liquidado) já ia direto para a fila de cada cliente — dois caminhos diferentes, sem ordem
> de **chegada** garantida entre eles, mesmo enfileirando `turn_closed` primeiro no código.
> **Isso mudou:** `turn_closed` e `turn_opened` passaram para a MESMA pista direta que
> `resolution_updated` (settled, quando aplicável) já usava — `turn_opened` NÃO estava nela
> antes de B2, é o próprio B2 que o move para lá — e agora os três, e o `piece_moved` que
> cada um deles pode preceder ou seguir, são enviados pelo MESMO goroutine.
>
> A sequência exata, quando os dois `piece_moved` possíveis acontecem, é `[piece_moved da
> fuga, se a peça dela sair do lugar]` → `turn_closed` → `resolution_updated` (liquidado) →
> `[piece_moved da nova ação, se ela tiver `move`]` → `turn_opened` — são DOIS
> `piece_moved` diferentes, não um só: o da fuga é de um personagem do turno que ACABOU e sai
> antes do fechamento (`applyClosedEscapes`, antes de `persistBoard`); o da nova ação é do
> personagem do turno que ABRE e sai depois da resolução liquidada, dentro de
> `announceOpenedTurn` (`applyOpenedMove`, antes do `dispatchPerPlayer` de `turn_opened`). Uma
> pista, um remetente: ordem de envio é ordem de chegada, para qualquer destinatário — não só
> para quem tem um atalho de posse. Ver a nota equivalente em [`turn_opened`](#turn_opened).

### `character_hp_changed`

**Direção:** servidor → cliente. **Destino:** **projetado** — o **mestre** e o **dono da
ficha** que mudou, e mais ninguém.

```json
{
  "type": "character_hp_changed",
  "payload": {
    "characterId": "22222222-2222-4222-8222-222222222222",
    "hp": 84,
    "maxHp": 100,
    "damage": 16
  }
}
```

| Campo | O que é |
|---|---|
| `characterId` | UUID da ficha — o mesmo ID que a peça do tabuleiro e `bars_updated.characters[].characterId` carregam. |
| `hp` | O valor que a ficha tem **agora**, já aplicado — e já persistido, na transação do turno que o aplicou (se ela falhar, a mensagem sai igual: o dano vale na mesa, mas não sobrevive a um reinício; ver [§9](#reinício-recarga-queda)). |
| `maxHp` | O máximo da mesma barra de vida, para o cliente desenhar a barra sem um round trip REST. |
| `damage` | O quanto acabou de sair. É o que deixa uma linha de histórico dizer "−16" sem diffar dois snapshots. |

**É o número APLICADO, não a projeção.** `resolution_updated` carrega o ensaio
(`projectedDamage`) enquanto o turno está aberto; só o fechamento escreve na ficha. Quem
desenha a barra de vida escuta esta mensagem, não aquela.

**Por que mensagem própria, e não um campo de `resolution_updated`:** HP também vai se mexer
por **cura** e por **veneno**, e nenhum desses caminhos resolve turno nenhum. Um campo na
resolução teria que ser duplicado no instante em que o primeiro deles chegasse. O que esta
mensagem diz é "a barra mexeu", não "um turno calculou alguma coisa" — quando a cura e o
veneno existirem, eles emitem **esta** mensagem, com o mesmo formato.

**Por que projetado:** o HP exato de terceiro não é estado de mesa. A mesa fica sabendo que
alguém se machucou pela narração e pelo `resolution_updated` liquidado (que já é projetado,
§6), não por um número. É o mesmo eixo mestre/dono que `resolution_updated` aplica — não há
um segundo mecanismo.

**NPC não tem dono**, então o mestre recebe uma cópia só. Um jogador comum não é avisado do
HP de NPC nenhum por aqui — nem precisaria: ele não consegue nem buscar a ficha por REST
(ver a nota de permissão em [`npc_added`](#npc_added)).

**Uma mensagem por personagem que levou dano.** Um golpe em área fecha o turno com vários
alvos e sai um `character_hp_changed` para cada um — para o mestre todos, para cada jogador
só o(s) dele(s).

**Disparado por:** [`close_turn`](#close_turn), [`open_next_action`](#open_next_action) e
[`pull_action`](#pull_action) — os **três** verbos que fecham um turno, exatamente como
[`turn_closed`](#turn_closed). Sai **depois** da persistência (ninguém recebe um número que o
banco ainda não tem) e **antes** do `turn_closed` do mesmo fechamento.

> ⚠️ **Não sai quando o dano é zero.** Só o que foi de fato escrito na ficha vira mensagem:
> um ataque esquivado, ou aparado até sobrar nada, fecha o turno sem nenhum
> `character_hp_changed`. Quem quiser saber que houve um ataque que não machucou lê o
> `resolution_updated` liquidado.

### `round_closed`

**Direção:** servidor → cliente. **Destino:** **mesa inteira**.

```json
{ "type": "round_closed", "payload": { "roundMode": "Race" } }
```

A rodada acabou porque **nenhuma ação na fila conseguia mais pagar o preço** — nenhum turno foi
aberto. As ações que ficaram na fila guardam a rolagem que já fizeram e pertencem à próxima
rodada.

**Disparado por:** `open_next_action`, e só ele.

**Já gravado quando chega** (dono do produto, 2026-10-02 — um comando do mestre, uma transação):
o `finishedAt` da rodada que acabou e a linha da rodada que nasceu no lugar dela vão juntos, na
transação do turno que o mesmo `open_next_action` fechou — ou, se ele não fechou turno nenhum,
numa transação só deles. Nunca um sem o outro: o banco nunca fica com duas rodadas abertas na
cena, nem com nenhuma. Se essa gravação falhar, a sala guarda o fim da rodada e o grava, na
mesma transação, com a próxima gravação da rodada seguinte (um fechamento de turno, uma master
action, uma troca de regime, um `change_scene`) — a rodada seguinte nunca nasce aberta ao lado
da anterior ainda aberta no banco. O `change_scene` faz o mesmo com o par antigo quando ele nunca
virou linha ou quando fechá-lo no banco falha: ele vai, fechado, na transação que grava o par
novo.

### `round_mode_changed`

**Direção:** servidor → cliente. **Destino:** **mesa inteira** — o regime é público: todo
mundo precisa saber se as barras estão correndo.

```json
{ "type": "round_mode_changed", "payload": { "mode": "Race" } }
```

**Gravado antes de emitido.** A troca já está em `match_events` (e em `rounds.mode`) quando esta
mensagem sai — inclusive com turno aberto. Um `GET /history` que comece depois de recebê-la já a
traz.

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

**Gravado antes de emitido.** O par antigo (cena e round) já foi fechado e o novo já existe no
banco quando esta mensagem sai (`change_scene` recusa turno aberto). Um `GET /history` que comece
depois de recebê-la já traz a cena nova. Se a gravação falhar, o par fica marcado e vai na
próxima gravação que der certo — o histórico se atrasa, mas não perde a cena.

### `master_action_enqueued`

**Direção:** servidor → cliente. **Destino:** **mesa inteira** (broadcast) no caminho 3;
**só o mestre** nas ações de peça (caminho 0) — o eco carrega a posição, e para peça oculta
isso vazaria para a mesa.

**Gravado antes de emitido — fora de turno.** Sem turno aberto, a master action já está em
`master_actions` quando esta mensagem sai, e um `GET /history` que comece depois de recebê-la já
a traz. **Com turno aberto, não:** ela é gravada no fechamento do turno, na mesma transação
dele, e só aparece no histórico depois do `turn_closed`.

No caminho 3 é o **eco literal** do `MasterActionPayload` recebido:

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

Nas ações de peça é o que foi **aplicado**: `targetIds` e `move: { "position": [...] }`
(sem `category`/`speed`/`charge`, que são ignorados), ou `remove: {}`.

**Disparado por:** `enqueue_master_action` nos caminhos 0 (peça) e 3 (sem `interact`).

### `npc_added`

**Direção:** servidor → cliente. **Destino:** mesa inteira, em broadcast.

```json
{
  "type": "npc_added",
  "payload": { "characterId": "44444444-4444-4444-8444-444444444444" }
}
```

Só o `characterId`. É o ack do MESTRE e o sinal para ELE (re)buscar a ficha por REST — a
mesma forma de [`scene_changed`](#scene_changed) e
[`master_action_enqueued`](#master_action_enqueued): o evento anuncia que algo mudou, não
carrega a coisa que mudou.

⚠️ **Só o mestre consegue buscar essa ficha.** `GetCharacterSheetUC.GetCharacterSheet`
(`internal/application/character_sheet/get_character_sheet.go:64-119`) só devolve uma ficha
para o `master_uuid` dela, para o `player_uuid` dela, ou para o mestre da campanha — nessa
ordem de checagem. Uma ficha de NPC tem `player_uuid == null`, então um jogador comum que
tentar buscá-la (`GET /charactersheets/{uuid}`) cai em `auth.ErrInsufficientPermissions`
(403), a menos que por acaso seja o mestre da campanha. `npc_added` é broadcast para a mesa
inteira, mas o (re)fetch por REST que ele sugere só faz sentido para quem tem permissão de
lê-la: o mestre. **Os demais jogadores aprendem que o NPC existe pelo `bars_updated.characters`
que segue** (que já traz `characterId`, saldo e velocidades) **e pela peça que aparece no
tabuleiro** — não pela ficha completa, que não é deles para ver. Não é uma lacuna deste
contrato: é a mesma regra de visibilidade de ficha que já vale fora do combate.

Não vaza nada novo além disso: `bars_updated.characters` já lista todo personagem em combate,
NPC incluído.

**Com sessão viva, um `bars_updated` com `seq` maior sai junto — em QUALQUER ordem em relação
a este.** Não sai um `match_full_state`: o único estado de combate que muda ao pôr um NPC é o
conjunto de personagens nas barras, e `match_full_state.bars.seq` repete o `seq` CORRENTE por
contrato (ver a nota de `bars.seq` em [`match_full_state`](#match_full_state), abaixo) —
reenviá-lo aqui deixaria um `bars_updated` atrasado, do MESMO `seq` e sem o NPC, ser aplicado
por cima e apagar o NPC da tela. `bars_updated` é o caminho documentado para "qualquer coisa
que mexe nas barras", e é o único que incrementa `seq`. ⚠️ **A ordem de chegada entre
`npc_added` e esse `bars_updated` não é garantida:** cada um sai de uma goroutine própria
(`go func() { r.broadcast <- data }()`, tanto no braço `add_npc` de `room.go` quanto dentro de
`broadcastBars`), disparadas em sequência no código mas entregues ao canal de broadcast sem
ordem relativa assegurada. O front não pode assumir que `npc_added` sempre chega primeiro —
`bars_updated.characters` já é suficiente para desenhar o NPC nas barras, com ou sem o
`npc_added` correspondente já visto.

**Sem sessão viva (lobby), não há `bars_updated` nenhum atrás** — não existem barras para
publicar. `npc_added` ainda sai, como confirmação de que o roster no banco mudou; o NPC só
entra em combate quando a sala nascer (ver [`add_npc`](#add_npc)).

**Disparado por:** [`add_npc`](#add_npc) aceito — com sessão viva ou não.

### `match_full_state`

**Direção:** servidor → cliente. **Destino:** quem **conecta ou reconecta**, sempre que a
partida já tem sessão viva (combate em andamento). **Não sai** enquanto a partida está só no
lobby — não há combate para sincronizar, e `buildMatchFullState` devolve `nil`.

`map_full_state` cobre o tabuleiro e só. Quem chegava no meio — ou reconectava, e o hook do
front reconecta até cinco vezes sozinho — ficava sem barras, sem regime, sem cena, sem turno
aberto, sem reações pendentes e, se fosse o mestre, sem a fila de ações, até alguma coisa
mudar por acaso. É essa lacuna que esta mensagem fecha.

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
      "actorId": "11111111-1111-4111-8111-111111111111",
      "actionId": "33333333-3333-4333-8333-333333333333",
      "action": {
        "uuid": "33333333-3333-4333-8333-333333333333",
        "actorId": "11111111-1111-4111-8111-111111111111",
        "targetId": ["22222222-2222-4222-8222-222222222222"],
        "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity", "skillValue": 0, "attempts": { "primary": [6, 8] }, "result": 14 } },
        "attack": {
          "weapon": "Sword",
          "hit": { "skillName": "Accuracy", "skillValue": 0, "attempts": { "primary": [6, 8] }, "result": 14 },
          "damage": { "skillName": "Push", "skillValue": 0, "attempts": { "primary": [4] }, "result": 4 },
          "relativeVelocity": 0
        }
      },
      "reactions": [
        {
          "uuid": "44444444-4444-4444-8444-444444444444",
          "actorId": "22222222-2222-4222-8222-222222222222",
          "reactToId": "33333333-3333-4333-8333-333333333333",
          "reactionKind": "closedEscape",
          "skills": [
            { "skillName": "Evasion", "rollCheck": { "skillName": "Evasion", "skillValue": 0, "attempts": { "primary": [7, 9] }, "result": 16 } }
          ],
          "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity", "skillValue": 0, "attempts": {}, "result": 11 } },
          "move": {
            "category": "Shift",
            "position": [8, 6, 0],
            "speed": { "skillName": "Brake", "skillValue": 0, "attempts": {}, "result": 11 },
            "finalSpeed": 11
          },
          "dodge": { "rollCheck": { "skillName": "Reflex", "skillValue": 0, "attempts": { "primary": [7, 9] }, "result": 16 } }
        }
      ]
    },
    "resolution": {
      "turnId": "55555555-5555-4555-8555-555555555555",
      "isSettled": false,
      "action": { "skillName": "Accuracy", "skillValue": 14, "diceRolled": [6, 8], "total": 20, "isCritical": false, "isCriticalFailure": false },
      "targets": []
    },
    "queue": [
      {
        "actionId": "33333333-3333-4333-8333-333333333333",
        "actorId": "11111111-1111-4111-8111-111111111111",
        "bars": ["action"],
        "action": {
          "uuid": "33333333-3333-4333-8333-333333333333",
          "actorId": "11111111-1111-4111-8111-111111111111",
          "targetId": ["22222222-2222-4222-8222-222222222222"],
          "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity", "skillValue": 0, "attempts": { "primary": [6, 8] }, "result": 14 } },
          "attack": {
            "weapon": "Sword",
            "hit": { "skillName": "Accuracy", "skillValue": 0, "attempts": { "primary": [6, 8] }, "result": 14 },
            "damage": { "skillName": "Push", "skillValue": 0, "attempts": { "primary": [4] }, "result": 4 },
            "relativeVelocity": 0
          }
        }
      }
    ]
  }
}
```

**O exemplo acima é o que o MESTRE vê** (`queue` presente, `ownQueue` ausente). A reação aberta
em `openTurn.reactions` vem `actionwire.Full`, igual ao [`reaction_opened`](#reaction_opened) que o
mestre recebeu ao vivo. `ownReactions` está **ausente** porque o reator (`2222…`) é personagem
de jogador, não NPC do mestre — o mestre só recebe em `ownReactions` as reações dos NPCs dele; o dono do
`2222…`, ao reconectar, a recebe em `ownReactions` (ver a tabela abaixo). Um jogador
que reconecta com a MESMA ação ainda na fila vê o espelho — `queue` ausente, `ownQueue`
presente com a versão em `actionwire.Declaration` da mesma entrada:

```json
{
  "type": "match_full_state",
  "payload": {
    "roundMode": "Race",
    "bars": { "seq": 7, "prices": { "action": 14, "move": 12 }, "characters": [], "order": [] },
    "ownQueue": [
      {
        "actionId": "33333333-3333-4333-8333-333333333333",
        "action": {
          "uuid": "33333333-3333-4333-8333-333333333333",
          "actorId": "11111111-1111-4111-8111-111111111111",
          "targetId": ["22222222-2222-4222-8222-222222222222"],
          "speed": { "bar": 0, "rollCheck": { "skillName": "Legerity" } },
          "attack": { "weapon": "Sword", "hit": { "skillName": "Accuracy" }, "damage": { "skillName": "Push" }, "relativeVelocity": 0 }
        }
      }
    ]
  }
}
```

Note o que sumiu em `ownQueue[0].action` em relação ao `queue[0].action` do mestre:
`skillValue`/`attempts`/`result` de `speed.rollCheck` e de `attack.hit`/`attack.damage` —
Declaration não é Full nem Opened rebaixado, é só a declaração; `weapon` e `targetId`
continuam, porque isso o dono já sabia que declarou.

| Campo | Notas |
|---|---|
| `scene` | O payload de [`scene_changed`](#scene_changed) **inteiro** — `sceneId`/`category`/`briefInitialDescription`, os mesmos nomes, a mesma struct. Não é uma segunda forma para os mesmos três valores. **Ausente** (`omitempty`) quando a partida não tem cena ativa — `Scene *SceneChangedPayload` com `omitempty` **omite a chave inteira**, não emite `null`; em TypeScript o campo é `scene?: SceneChangedPayload`, não `scene: SceneChangedPayload \| null`. Leia `scene === undefined` como "sem cena", não como "cena sem nome". `category` é minúscula (`"battle"`/`"roleplay"`) e é validada contra o enum — ver `change_scene`. |
| `roundMode` | O regime do round ativo — `"Free"` ou `"Race"`, os mesmos valores de [`round_mode_changed`](#round_mode_changed). Vem `""` quando não há round ativo. Público: vai para todo mundo que conecta. |
| `bars` | O `bars_updated` **inteiro**, reaproveitado — não é uma segunda forma para manter em sincronia com a primeira. |
| `bars.seq` | ⚠️ **É o contador CORRENTE, não um novo.** O cliente guarda o maior `seq` já aplicado e descarta qualquer coisa menor; estampar um número novo aqui zeraria essa guarda numa reconexão — o primeiro `bars_updated` atrasado a chegar depois seria aplicado por cima de um estado mais novo. É por isso que a proteção do cliente contra snapshot atrasado atravessa a reconexão: o contador nunca reinicia — **nem quando a sala é recriada** (o servidor reiniciou, ou a sala esvaziou e se fechou): uma sala nova começa o contador no relógio, em microssegundos, acima de qualquer `seq` da sala que ela substitui. O número é grande (ordem de 10¹⁵) e cabe exato num `number` do JavaScript. |
| `openTurn` | Ausente (`omitempty`) **para todo destinatário** — jogador ou mestre — quando a mesa está em "fechado e nada aberto", estado em que ela pode legitimamente estar. Quando presente, vai para **todo mundo** que conecta: quem é o ator da vez não é segredo. |
| `openTurn.actionId` / `openTurn.action` | **B2 (design spec §4.2).** Os mesmos dois campos que o [`turn_opened`](#turn_opened) ao vivo já mandou para este mesmo destinatário — `action` projetado pela MESMA regra (mestre vê `actionwire.Full`; todo mundo mais, inclusive o dono, vê `actionwire.Opened` depois do deny-list de `service.ProjectAction`). O exemplo acima é o que o MESTRE vê; um jogador que reconecta recebe o corte de `Opened` aqui, exatamente como no `turn_opened` que perdeu ao cair — **inclusive o gate de fog de `move.from`/`move.position`** (ver [`turn_opened`](#onde-a-peça-vai-movefrom-e-moveposition-seguem-a-fog)), lido da visão dele no momento da reconexão, com `move.from` como origem julgada (a casa em que a peça estava na abertura não é guardada — numa janela rara em que a peça andou entre o enfileiramento e a abertura, o snapshot pode diferir do `turn_opened` ao vivo). Sem `actionId` o cliente não tinha como casar este turno com uma ação da própria fila reconciliada (`ownQueue`, B12). |
| `openTurn.reactions` | **Fase 7 (design spec §4.1).** As reações **abertas** do turno aberto, na ordem em que o mestre as abriu — cada uma cortada para este destinatário **exatamente** como o [`reaction_opened`](#reaction_opened) ao vivo a cortou (`reactionWireLocked`, o mesmo código): mestre vê `actionwire.Full`; todo mundo mais, o dono inclusive, `actionwire.Opened` depois de `service.ProjectAction` (rótulo fechado rebaixado, `Evasion` e `consumedActionIds` tirados para um terceiro), com o destino de uma fuga passando pelo gate de fog da peça do reator, julgado da casa em que a peça está agora (nenhuma fuga desloca antes do fechamento). Assim a reconexão mostra à mesa os mesmos balões e fantasmas de fuga, **na mesma ordem** — a ordem muda o resultado. Uma reação anexada e **não** aberta nunca está aqui: ela nunca foi anunciada. Ausente (`omitempty`) quando nenhuma está aberta. |
| `ownReactions` | **Fase 7 (design spec §4.2, decisões D5/D6).** As reações do turno aberto cujo ator pertence a este destinatário (`charToPlayer` — o mestre, pelos NPCs dele), **abertas ou não**, na ordem de chegada: o que o [`reaction_attached`](#reaction_attached) disse ao vivo, para um cliente que reconectou depois dele. Uma entrada é `{ reactionId, actorId, reactionKind, opened, consumedActionIds }`. `opened` diz se o mestre já deu a palavra ("reação enviada, aguardando o mestre" × "é a sua vez de narrar"). `reactionKind` é o **verdadeiro** — o dono vê o próprio (`closedEscape`, não `escape`). `consumedActionIds` são as ações da fila que ela consumiu — **sempre** lista, `[]` numa reação livre; uma declarada nomeada ali foi consumida, não perdida (regra de reconciliação abaixo). Ausente (`omitempty`) quando o destinatário não tem nenhuma, ou não há turno aberto — aqui ausente e vazio querem dizer a mesma coisa, diferente de `ownQueue`: nenhuma reconciliação depende de distinguir os dois. |
| `resolution` | O cálculo do turno aberto, **master-only**. Ausente para qualquer outro destinatário, e também ausente para o próprio mestre quando não há turno aberto. Mesmos dois eixos de `resolution_updated` (§6) — aqui só o eixo do TEMPO se manifesta, porque um snapshot de conexão sempre reflete um turno em aberto (`isSettled: false`); não existe um `match_full_state` de turno fechado. |
| `queue` | A fila do mestre, **master-only pelo mesmo eixo de `resolution`** — ausente para qualquer outro destinatário. Um payload de [`action_queued`](#action_queued) **inteiro** por ação ainda pendente, na **ordem de inserção** da fila (não confundir com `bars.order`, que carrega a ordem *projetada* de execução — public, sem identidade de ação). Cada entrada carrega `action` **igual, byte a byte**, ao que o `action_queued` daquela ação já mandou ao vivo — as duas vêm de `newActionQueuedPayload` (`room.go`), então não podem divergir. `omitempty`: **ausente** significa fila vazia, não erro. Existe **com ou sem turno aberto** — o estado mais comum de reconectar é justamente "nada aberto ainda, três coisas esperando". É a versão de `action_queued` que **sobrevive à reconexão**; ver a nota na seção de `action_queued`. |
| `ownQueue` | **B12 (design spec §4.2).** O espelho de `queue` para quem NÃO é o mestre: as ações ainda pendentes cujo ator pertence a este destinatário (`charToPlayer`), na mesma ordem de inserção da fila, uma `{ actionId, action }` por entrada, `action` em `actionwire.Declaration` (só o que o dono declarou — arma, alvos, `move.category/from/position`, nomes de perícia; **nenhum** dado, total ou velocidade, nem a de `speed`/`move` — ver a tabela de corte, §4.1 do design spec). Ausente **só para o mestre** — o eixo aqui não é tempo, é CLASSE, o oposto de `queue`. **SEMPRE presente para todo o resto**, mesmo sem nada pendente: vem `[]`, não ausente. É por isso que o tipo em Go é ponteiro (`*[]OwnQueuedActionPayload` com `omitempty`) em vez de slice nua — uma slice nua nula ainda serializa `null`, e o contrato aqui não é "nulo ou a lista", é "ausente (mestre) ou presente, vazia ou não (todo mundo mais)"; em TypeScript isso é `ownQueue?: OwnQueuedActionPayload[]`, e a chave só falta quando o destinatário é o mestre — para qualquer outro `ownQueue` está sempre lá, e `.length === 0` é a resposta "nada seu na fila", não a ausência do campo. Ver a **regra de reconciliação** logo abaixo. |

**Regra de reconciliação (B12).** Uma ação que o cliente ainda guarda como declarada (enviada,
nunca confirmada como aberta ou fechada) é **conhecida pelo servidor** se — e só se — o seu
`actionId` aparece em `ownQueue` **ou** é igual a `openTurn.actionId`. Essas duas fontes juntas
são o universo inteiro do que o servidor ainda tem: uma declarada que não está em nenhuma das
duas foi perdida (reinício — ver §9, linha "Fila") ou nunca chegou a ser aceita. Para essa, o
cliente descarta com um aviso e devolve o rascunho a quem declarou, para reenviar por conta
própria se quiser. **O cliente nunca reenvia sozinho**: um reenvio automático rola os dados de
novo (`EnqueueAction` rola na chegada), então "a mesma ação" reenviada não é mais a mesma ação —
é uma segunda tentativa com números novos, e só a pessoa que declarou decide se quer isso.

Uma declarada cujo `actionId` aparece em `ownReactions[].consumedActionIds` — ou no
`consumedActionIds` de uma reação do [histórico](match-history.md) — **foi consumida** por uma
reação cobrada, não perdida: sai da lista sem o aviso de perda e sem devolver o rascunho. Ao
vivo, o mesmo fato chega em [`reaction_attached`](#reaction_attached).

**Disparado por:** todo `register` (conexão OU reconexão) enquanto há sessão de partida —
logo depois de `room_state` e do `map_full_state` (se houver peças **ou paredes** no
tabuleiro — desde B14 um tabuleiro pode ter paredes sem nenhuma peça), e antes do
`player_joined` que avisa os demais da chegada.

Quem já estava conectado também recebe `match_full_state` (logo depois do `map_full_state` reenviado) quando a sessão é restaurada depois de um reinício do servidor, no registro do mestre — o jogador que chegou antes da reidratação foi recebido sem sessão e, sem isso, ficaria sem ele.

### `piece_moved` (também servidor → cliente, na ABERTURA do turno)

<a id="piece_moved-servidor"></a>

**Direção:** servidor → cliente. **Destino:** fog-gated, por destinatário.

O contrato completo de `piece_moved`/`piece_removed` — shape de `SlotPayload`, o campo `z`
— é do lobby/mapa e vive em [`game-lobby.md`](game-lobby.md) e
[`match-maps.md`](match-maps.md). Até aqui `piece_moved` era só **cliente → servidor**: o
navegador do jogador aplicando localmente um arraste e sincronizando o resto da mesa. O que o
motor de combate acrescenta:

> **`piece_moved`/`piece_removed` enviados pelo CLIENTE são lobby-only desde B14** (spec
> §4.3, "Quem move o quê", Task 4) — com sessão viva, ambos são recusados com `error`
> `forbidden` para mestre **e** jogador, sem excecão: o mestre passa a mover/pôr/tirar peça
> por `enqueue_master_action` (`move`/`remove`), e o jogador só move agindo. Ver o contrato
> completo (quem pode enviar o quê, na lobby, e a validação de posse) em
> [`match-maps.md`](match-maps.md#websocket-piece_moved--piece_removed-cliente--servidor). O
> restante desta seção — a tabela de fog, o `senderId`, o `map_full_state` extra do dono — é
> sobre o `piece_moved` **servidor → cliente** que os dois momentos abaixo emitem, o que
> continua acontecendo em pleno combate.

> **O tabuleiro é do servidor desde B14** (spec §4.3, "Quem carrega") — carregado do banco
> quando a sala nasce e, enquanto a partida é lobby, a cada conexão do mestre; depois que
> começa, só no nascimento. `map_state_sync` **não escreve mais nada**: é aceito, ignorado,
> e responde só ao remetente com o `map_full_state` atual — será removido do contrato
> quando a Fase 13 tirar o envio do front. Ver a seção "O tabuleiro é do servidor" em
> [`game-lobby.md`](game-lobby.md) para o detalhe completo.

**Quando o `Move` de uma ação de turno ABRE, o servidor aplica a posição sozinho e emite
`piece_moved` como AUTOR.** Antes, uma ação de mover acontecia no cálculo e a peça nunca
saía do lugar no tabuleiro — o fog nunca recalculava. Agora `applyOpenedMove` escreve a nova
posição na **abertura** do turno, não no fechamento: o dano ainda pode ser editado pelo
mestre depois de aberto, mas as reações que seguem dependem de onde a peça ESTÁ, e não podem
esperar. O caminho reaproveita o mesmo par `piece_moved`/`piece_removed` que o lobby já usa
— não um tipo novo.

**O gate de fog é o mesmo par de sempre — e tem DOIS cortes ANTES dele, nos dois regimes.**
A tabela vale para o que esta seção descreve: uma partida **com sessão viva** (combate em
andamento). O mesmo helper (`relayPieceMove`, `room.go`) serve o lobby, e nele o corte de
remetente e o corte de mestre rodam **primeiro**, antes de qualquer ramo de lobby ou de
combate — é por isso que as duas primeiras linhas da tabela abaixo valem sem exceção. O que
muda no lobby é só o que vem **depois** desses dois cortes: sem sessão viva, o ramo de lobby
devolve a mensagem a todo mundo antes do teste de peça oculta e do teste de campo de visão —
ver a linha da peça oculta.

| Quem | Recebe |
|---|---|
| Quem **enviou** o `piece_moved` (o `senderId` do envelope) | **nada** — mestre ou jogador. O browser dele já desenhou o arrasto; ninguém é ecoado para si mesmo. Este corte vem **antes** de todos os outros, inclusive do corte de mestre. Só não se aplica ao movimento de autor SERVIDOR, que não tem remetente. |
| Mestre (quando não é o remetente) | `piece_moved` — sem gate de fog, mesmo com `visible: false` |
| Jogador, peça marcada `visible: false` (oculta) | **nada**, **em combate.** Com sessão viva `relayPieceMove` corta a peça oculta antes de olhar linha de visão — nem `piece_moved` nem `piece_removed`, mesmo para quem enxergaria o destino a olho nu. **No lobby (sem sessão) é o contrário:** o ramo de lobby devolve a mensagem a todo mundo **antes** do teste de oculta, porque no lobby não há fog nenhum. A regra é do combate, não do protocolo. |
| Jogador, peça visível, enxerga o **destino** | `piece_moved`, com a posição nova |
| Jogador, peça visível, só enxergava a **origem** (a peça "saiu de vista") | `piece_removed`. Uma peça que não estava no tabuleiro antes não tem origem: aí não há metade `piece_removed`, só a metade `piece_moved` para quem enxerga o destino. |
| Jogador, peça visível, não enxergava nem origem nem destino | nada |

O **dono** da peça movida é julgado pela linha de visão **nova**, não pela antiga: o servidor
recomputa a visão dele antes do despacho. Sem isso, uma peça que anda para fora do próprio
campo de visão anterior faria o dono receber `piece_removed` da própria peça.

`senderId` vem **zero** (`00000000-…`) quando o autor é o servidor — nenhum navegador previu
esse movimento, então ninguém é pulado no dispatch. É assim que o cliente distingue "o
servidor moveu isto" (`senderId` zero) de "outro jogador moveu isto" (`senderId` = o UUID de
quem enviou).

O dono do personagem movido recebe, além disso, um `map_full_state` atualizado — a linha de
visão dele mudou, mesmo quando quem moveu a peça não foi ele (o mestre arrastando a peça de
um jogador, ou o motor aplicando um movimento resolvido). Três ressalvas, todas do código:

- **A peça de um NPC é tratada como sem dono, de propósito (Decisão 7).** O dono é resolvido
  a partir do **personagem** (`characterId` → jogador, via `charToPlayer`), e uma ficha de NPC
  TEM dono nesse mapa — o mestre (§2). Mas a visão do mestre não tem fog: ele enxerga o
  tabuleiro inteiro, e `buildMapFullState` já descarta os polígonos quando quem pede é o
  mestre. Recomputar linha de visão, criar uma `PlayerMemory` que ninguém lê e reenviar o
  tabuleiro inteiro a cada arrasto de NPC não mudaria nada que o mestre veja — então
  `relayPieceMove` zera o dono quando ele é o mestre e pula os três: recompute, `PlayerMemory`
  e o `map_full_state` extra. O mestre continua recebendo o `piece_moved` normal pelo ramo
  `isMaster` do despacho. Vale para os caminhos que passam por `relayPieceMove` em combate: o
  movimento de uma ação de turno, a fuga de reação e a master action de peça (abaixo) — nos
  três o autor é o SERVIDOR, não há `senderId` a cortar, e por isso o mestre nunca é suprimido:
  a tela dele espera a confirmação em vez de já ter desenhado o arrasto. **O arrasto do
  CLIENTE em combate não existe mais desde B14** — com sessão viva, mestre e jogador só movem
  peça por `enqueue_action` ou `enqueue_master_action`; `piece_moved`/`piece_removed` vindo do
  cliente só é aceito no lobby, onde o corte de remetente da primeira linha da tabela acima
  continua valendo. Uma peça cujo `characterId` não está na partida também não tem dono, e
  ninguém recebe o extra.
- O dono **offline** tem o cache recalculado mas não recebe nada (ele reconectaria num
  polígono velho).
- Se o recálculo falhar, o `map_full_state` não sai — o `piece_moved` do par acima sai do
  mesmo jeito.

**Também sai pela master action de peça** ([`enqueue_master_action`](#enqueue_master_action)
com `move`/`remove`, caminho 0): o mesmo `relayPieceMove` (e, para tirar, o mesmo gate na
última posição), autor servidor, o mestre incluído no despacho.

**Disparado por** dois momentos, e é o **mesmo** `applyMove` nos dois:

| Momento | Quando |
|---|---|
| Abertura do turno (`open_next_action`, `pull_action`) | a ação que abre carrega um `Move`. **Qualquer categoria** — uma ação não tem CD vindo contra ela, e a posição não pode esperar, porque as reações seguintes dependem de onde a peça está. |
| Fechamento do turno (`close_turn`, e também `open_next_action`/`pull_action`) | uma fuga (qualquer das três, `Dash` ou `Shift`) **escapou** → vai ao destino; **falhou com `escapeLanding`** → vai para onde o mestre escolheu. Falhou sem escolha → nada sai, a peça fica. **A abertura da reação nunca desloca.** |

As regras completas do lado da reação — de onde vem a CD, o que é escapar, e para onde vai a
peça que falha — estão em [`open_reaction`](#open_reaction) e em
[`close_turn`](#close_turn). Um movimento de **ação** que dependesse de teste **pela
categoria** — um salto, um aperto — continua **sem caso alcançável**: `move.category` só aceita
`Dash` e `Shift`, e as outras cinco são recusadas no mapeamento, então não existe código para
esse ramo.

**O pouso em slot ocupado não entra nessa lista**, e antes entrava: um `Dash` para um slot onde
já há peça é **aceito** hoje, e o que acontece é a peça ir para lá e **empilhar**. Nada valida
ocupação em nenhum dos dois momentos — nem na queda que o mestre escolhe. Isso é a mesma classe da colisão com parede, logo abaixo:
não é validação esquecida, é a regra que ainda não foi escrita — compartilhar o slot, ser
bloqueado antes de entrar, ou empurrar quem está lá são desfechos possíveis, e nenhum foi
escolhido. O front **não deve** tratar o empilhamento como bug a reportar, e também não deve
inventar a regra do seu lado: quem desenhar a colisão decide os dois casos juntos.

Um ator **sem peça no tabuleiro** não é erro, em nenhum dos dois momentos: não há o que mover,
nada é emitido e nenhuma mensagem de erro sai. O turno abre (ou fecha) normalmente.

⚠️ **A semântica de `Z` está em aberto.** `PieceMovedPayload.Z` é documentado como altura
virtual em metros; `Move.Position[2]` é o índice `z` da grade. São grandezas possivelmente
diferentes, e por isso o servidor **preserva o `Z` que a peça já tinha** em vez de
sobrescrevê-lo com `Move.Position[2]`. Escrever um horizontal sobre um vertical derrubaria
uma peça elevada ao chão a cada passo horizontal cujo `z` de grade for `0`. A pergunta
"`Move.Position[2]` é metro ou índice de grade?" precisa de resposta antes de qualquer
cliente escrever `Z`. Ver §10.

⚠️ **Colisão contra parede é uma fatia de regra ainda não desenhada — o efeito hoje é que a
peça atravessa.** A única checagem existente contra paredes com `move=true` e `open=false`
acontece no **enfileiramento** (`enqueue_action`, quando o ator TEM peça no tabuleiro — ver a
tabela em `enqueue_action`), não de novo aqui na abertura. A origem do caminho checado é a
**posição da própria peça** (B6), nunca o `move.from` que o cliente mandou; o destino é
`move.position`; e os dois pontos são convertidos para o mundo pelos **centros** dos slots
(`mapservice.SlotCenterToWorld`, com a grade da sessão) — B5, e também o que torna a conversão
certa numa grade hexagonal (B10), onde a fórmula de uma grade quadrada não vale. O caminho de
reação **nunca passa por essa checagem, nem uma vez**: quando `reactToId` é não-zero,
`enqueue_action` roteia para o mesmo tratamento de `attach_reaction` e retorna **antes** de
alcançar o código que valida a parede — esse código só existe no ramo de ação comum.
`attach_reaction`, enviado direto, também não tem checagem nenhuma no caminho, e a Move de uma
reação nunca recebe um `from` derivado (fica ausente) — o movimento de um escape só é aplicado
ao tabuleiro no fechamento do turno (`close_turn`), nunca checado contra parede na declaração.
Não é validação esquecida: **ainda não existe a regra que decide o que acontece quando um
personagem colide com uma parede** — compartilhar o slot, ser bloqueado, ou quebrar a parede
no impacto são desfechos possíveis, e nenhum foi escolhido ainda. Ver §10.

### `error`

**Direção:** servidor → cliente. **Destino:** **só quem enviou** a mensagem que falhou.

```json
{
  "type": "error",
  "payload": { "code": "forbidden", "message": "only the master can perform this action" }
}
```

Catálogo completo em §7.

### `connection_replaced`

**Direção:** servidor → cliente. **Destino:** a conexão ANTIGA — não quem mandou a mensagem
mais recente (B4, spec §4.6). É o único `error` deste protocolo que não é resposta a nada que
o destinatário tenha enviado: a mesma conta (mestre ou jogador) abriu uma segunda conexão —
segunda aba, um F5 cuja aba antiga não fechou a tempo — e o servidor decide pela última.

```json
{
  "type": "error",
  "payload": {
    "code": "connection_replaced",
    "message": "this account connected again elsewhere"
  }
}
```

O servidor fecha essa conexão imediatamente depois de mandar isto — nenhuma outra mensagem
chega por ela. **O que o cliente faz ao ver este código: nada além de aceitar o fim da
conexão.** Em particular, **não reconectar por conta própria** — existe uma conexão mais nova
da mesma conta em algum lugar (a outra aba, o F5 que já terminou), e essa é a que vale; a
antiga reabrir a sua própria só recriaria o mesmo conflito. A sala continua de pé e a conexão
nova recebe normalmente tudo que se passar dali em diante.

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

   Para o escape, o rebaixamento `closedEscape`→`escape` é **cosmético**: a categoria do
   movimento é fixa por tipo (§11.4 do documento mestre: `closedEscape` move com Shift,
   medido pelo Brake; `escape`/`escapeGuard`, com Dash, medido pelo Accelerate), e
   `move.category`/`move.speed.skillName` chegam a todos no `reaction_opened`; a barra pública
   também mostra que o escape fechado cobrou só a barra de movimento. O rótulo não é entregue,
   mas a mecânica deixa deduzi-lo — a mesma doutrina acima. Escondê-lo de verdade exigiria
   mudar as barras e o corte da categoria, o que não está decidido.

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

5. **`escape.landing` segue a fog da peça, não a classe só.** Onde uma fuga falha caiu é a
   mesma notícia que o destino de um movimento (decisão do dono do produto, 2026-10-01): para
   quem não é mestre nem dono do personagem que fugiu, `landing` **some** quando a casa de
   queda está fora da visão dele — ou a peça é `visible: false`, ou não está no tabuleiro —, pelo mesmo gate do relay ao
   vivo (`pieceMoveView`, sem origem). O resto do veredito (`escaped`, `movePassed`,
   `dodgePassed`, `awaitsMaster`) continua chegando: é público depois de liquidado. Note que
   `awaitsMaster: false` com `escaped: false` e sem `landing` quer dizer "o mestre escolheu
   uma queda que você não vê", não "ficou". O veredito deste gate é gravado no fechamento
   para todo jogador da sessão (`landingViews`, dentro do `escape` em `turns.resolution`), e
   o [`GET /history`](match-history.md#o-movimento-como-foi-visto-ao-vivo--movefrom-moveposition-escapelanding)
   mostra o `landing` só a quem o viu aqui — linhas antigas, a ninguém além de mestre e dono.
   O mesmo gate, sobre o destino de uma fuga que **escapou** (o `piece_moved` dela), decide
   quem vê no histórico o `move.position` da reação.

### Nota: a finta segue o mesmo eixo do TEMPO — desde B2, também neste protocolo

`Feint` não aparece em `resolution_updated` — a resolução liquidada não tem campo para ela, a
finta é da AÇÃO, não do cálculo. **Isso já não é mais "não aparece em payload nenhum deste
protocolo"** — era verdade até B2 (design spec §4.2), quando [`turn_opened`](#turn_opened)
passou a carregar `action`. Ela vive em `service.ProjectAction`, a mesma função que a Action
History REST chama, e segue exatamente este eixo do TEMPO ali e em `turn_opened`: escondida
de um terceiro enquanto `isSettled` é `false`; o mestre e o **dono** do ator sempre a veem
(`Viewer.SeesAllOf`), turno aberto ou fechado — quem caiu na finta descobre dentro da
resolução do MESMO turno (o sucesso foi contra um ataque falso, e o de verdade vem em
seguida), nunca meses depois olhando o histórico. Ver a tabela de `feint` na seção de
[`turn_opened`](#turn_opened) e [`match-history.md`](match-history.md), onde a mesma regra
vale para o REST.

`master_action_enqueued` continua sendo a exceção do lado do mestre — projeta
`action.MasterAction`, um tipo sem `Feint`, então nenhuma master action tem finta.

## 7. Catálogo de erros

| `code` | Significado | Onde aparece |
|---|---|---|
| `invalid_message` | `"malformed JSON"` — o envelope não parseou. | Qualquer mensagem. |
| `unknown_type` | `"unrecognized message type"` | `type` fora do catálogo. |
| `invalid_payload` | O `payload` não casa com a struct daquele `type`. A mensagem nomeia qual. | Todas. |
| `forbidden` | `"only the master can perform this action"` | `open_next_action`, `pull_action`, `open_reaction`, `edit_action`, `close_turn`, `change_round_mode`, `change_scene`, `enqueue_master_action`, `add_npc`. |
| `forbidden` | `"during a match the master moves pieces with enqueue_master_action"` (mestre) / `"players move by action"` (jogador) — com sessão viva, os dois papéis são recusados sem excecão (B14, spec §4.3, "Quem move o quê"). No lobby, o jogador ainda pode ser recusado por não ser dono da peça, ela não existir, ou tentar remover (só o mestre remove) — mensagens específicas, ver [`match-maps.md`](match-maps.md#websocket-piece_moved--piece_removed-cliente--servidor). | `piece_moved`, `piece_removed`. |
| `match_not_started` | `"match session not initialized"` — a partida não foi iniciada. | Todas as de partida (exceto `add_npc`) — na sala sem sessão, `add_npc` é caminho de sucesso (Decisão 3), não erro. |
| `invalid_action` | Payload bem formado, conteúdo inválido: perícia/arma/categoria de Nen desconhecida, reação sem componente obrigatório, `actorId` ausente, `reactToId`/`reactionKind` desemparelhados, **categoria de cena** fora de `"battle"`/`"roleplay"`; na master action de peça, `targetIds` sem exatamente um id, `move` e `remove` juntos, `remove` de personagem sem peça (`"character has no piece"`), ou **pôr** quando um segundo socket do mestre já criou a peça entre a leitura de posse e a gravação (`"character already has a piece"`). | `enqueue_action`, `attach_reaction`, `edit_action`, `change_scene`, `enqueue_master_action` (peça). |
| `not_participant` | Pôr no tabuleiro o personagem de um **jogador** que não participa da partida — só o NPC do mestre é inscrito ao ser posto (spec §4.3). | `enqueue_master_action` (`move` de personagem sem peça). |
| `move_blocked` | `"movement blocked by a wall"` | `enqueue_action` com `move`, quando o ator TEM peça no tabuleiro (a origem checada é a posição dessa peça, nunca `move.from` — B6). |
| `not_found` | Partida ou ficha de personagem não encontrada — mapeia `ErrMatchNotFound`/`ErrCharacterSheetNotFound` de `AddMatchNPCUC`. | `add_npc`, `enqueue_master_action` (pôr NPC). |
| `invalid_npc` | Ficha não é NPC, não pertence ao mestre nem à campanha, ou a partida já encerrou — mapeia `ErrSheetNotNPC`/`ErrSheetNotOwnedByMaster`/`ErrMatchAlreadyFinished`. | `add_npc`, `enqueue_master_action` (pôr NPC). |
| `npc_already_in_match` | O NPC já está na SESSÃO viva (`ErrCharacterAlreadyInSession`) — **não confundir com a duplicata do banco**, que este verbo tolera de propósito (ver [`add_npc`](#add_npc)). | `add_npc`. |
| `unknown_wall` | `"wall <id> is not on this match's board"` — um `targetId` de parede que o servidor não conhece (B14, spec §4.3, "Última defesa"). O resto do lote em `targetIds` ainda é processado. | `enqueue_master_action` (`interact.kind == "reveal"` e qualquer outro `interact`). |
| `game_error` | O domínio recusou. A `message` é o texto do erro de domínio (tabelas por mensagem em §4). | Todas as de partida. |
| `connection_replaced` | `"this account connected again elsewhere"` — a mesma conta abriu outra conexão; esta é a antiga, e o servidor a fecha a seguir (B4, spec §4.6). Ver [`connection_replaced`](#connection_replaced). | Não é resposta a mensagem nenhuma — dispara no `register`. |

**`error` nunca é broadcast.** Vai só para quem enviou a mensagem que falhou — **exceto
`connection_replaced`**, cujo destinatário não é quem mandou a última mensagem, mas a conexão
que ela está substituindo (ver acima).

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
    │◄──── turn_opened {turnId, actorId, actionId} (mesa) ──────►│◄──────────────────►│
    │                            ├──── resolution_updated ──────►│                    │
    │                            │     isSettled:false (MASTER-ONLY)                  │
    │                            │                               │                    │
    │                            │◄────────── attach_reaction ────────────────────────┤
    │                            ├──── reaction_attached ───────►│◄──────────────────►│
    │                            │     {turnId, reactionId, actorId,                  │
    │                            │     consumedActionIds}                             │
    │                            │     (a quem reagiu + mestre)                       │
    │                            ├──── resolution_updated ──────►│                    │
    │                            │     isSettled:false, pendingReactions[]            │
    │                            │                               │                    │
    │                            │◄──── open_reaction ───────────┤                    │
    │                            │      {reactionId de pendingReactions}              │
    │◄─ reaction_opened {turnId, reactionId, reaction} (mesa) ──►│◄──────────────────►│
    │                            │     reaction PROJETADA por destinatário            │
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
    │◄─ piece_moved (fog-gated) ─┤  (SÓ fuga que escapou, ou caiu onde o mestre quis) │
    │                            │   ┌─ persiste turno + resolução liquidada          │
    │◄──────────── turn_closed {turnId} (mesa) ─────────────────►│◄──────────────────►│
    │◄── resolution_updated ─────┤                               │                    │
    │    isSettled:TRUE          ├──── resolution_updated ──────►│                    │
    │    PROJETADO p/ A          │     isSettled:TRUE, completo  │                    │
    │                            ├──── resolution_updated ────────────────────────────►
    │                            │     isSettled:TRUE, PROJETADO p/ B (payload ≠ o de A)
    │◄─────────────── bars_updated (mesa) ──────────────────────►│◄──────────────────►│
```

**O que muda se o mestre usar `open_next_action` em vez de `close_turn`:** nada no
fechamento — persistência, `turn_closed`, `resolution_updated` liquidado e o `piece_moved`
das fugas cuja peça sai do lugar saem igual, e o `turn_closed` do turno que acabou vem
**antes** do `turn_opened` do próximo. O que muda é o que vem depois: se nenhuma ação na fila
conseguir mais pagar o preço, sai [`round_closed`](#round_closed) e nenhum turno novo abre — e o
fim da rodada vai na transação desse mesmo fechamento.

## 9. Reinício, recarga, queda

<a id="reinício-recarga-queda"></a>

Tabela completa (fila, barras, histórico, NPC, duas abas) em
`docs/superpowers/specs/2026-09-27-combat-closure-back-design.md` §5. Aqui, só as três
linhas que este contrato — o tabuleiro, a fila e o turno aberto — precisa dizer sozinho:

| Estado | Recarregar o cliente / reconectar | Reiniciar o servidor |
|---|---|---|
| Tabuleiro (posições, paredes, fog) | `map_full_state` do servidor (ver [`maps.md`](maps.md#map_full_state)) | **volta** de `match_boards` + `player_memories` (B3) |
| Fila | mestre: `queue`; dono: `ownQueue` (B12) — ambos em [`match_full_state`](#match_full_state) | **perdida** — `ownQueue` volta `[]`; o cliente descarta o rascunho com aviso e devolve para quem declarou. **Ninguém reenvia sozinho** |
| Turno aberto, reações anexadas, master actions feitas dentro dele | `openTurn` em [`match_full_state`](#match_full_state) com `action` e `reactions` (abertas); `ownReactions` com as suas; mestre recebe `resolution` | **perdido inteiro** — o turno só persiste ao FECHAR, e tudo o que aconteceu dentro dele vai junto: a peça da action volta para onde o último fechamento a deixou, as master actions do mestre feitas com o turno aberto (peça, parede, nota) não chegam a ser gravadas (ver abaixo), e as reações e o que elas consumiram vão junto |
| Duas abas (mesma conta conecta de novo) | a última vence — a antiga recebe [`connection_replaced`](#connection_replaced) e é fechada pelo servidor (B4) | — |
| `Register` numa sala que já fechou (`Run` retornou) | não trava — `ErrRoomClosed`; o mestre tenta uma vez mais contra uma sala nova (`GetOrCreateRoom`), o jogador recebe `lobby_not_open` (B7) | — |

**Por que o tabuleiro volta e o turno não** (spec §4.3, marcado com ⭐ lá). O tabuleiro é
gravado a cada `start_match`, a cada fechamento de turno (na transação do próprio turno) e —
**entre turnos** — a cada master action de peça e a cada interação/revelação de parede; nunca
na ABERTURA de uma action, nem com um turno aberto. A peça anda no `piece_moved`
que a abertura emite, mas se o servidor cair antes do próximo salvamento, essa gravação nunca
aconteceu: o tabuleiro que volta é o de ANTES da action que estava em curso, e o turno em si —
fila, reações anexadas, a escolha do mestre para um escape — some inteiro.

**Dentro de um turno aberto, nada é salvo antes do fechamento** (decisão do dono do produto,
2026-10-01). Com o turno aberto o tabuleiro já inclui a peça que a abertura moveu, e um
salvamento ali gravaria o efeito de um turno que ainda não existe no banco. Por isso a master
action de peça e a interação/revelação de parede **não** salvam o tabuleiro com turno aberto, e
as master actions feitas nele ficam guardadas em memória até o fechamento, que grava numa
**transação só** o turno, as master actions, o HP que ele aplicou (as barras de cada ficha
atingida, em `character_sheets`), o tabuleiro com o fog de cada jogador e — quando o mesmo
`open_next_action` também acaba a rodada — o fim dela e a rodada seguinte; ou nada disso, se ela
falhar (o tabuleiro e as fichas gravados continuam os do último fechamento; o fim da rodada, que
aconteceu na mesa, ainda é gravado com a seguinte numa transação só deles — e, se também essa
falhar, vai com a próxima gravação da rodada seguinte). Um comando do mestre,
uma transação (dono do produto, 2026-10-02). Se o servidor cair —
ou a sala fechar porque todos saíram, que para a persistência é o mesmo que um reinício — antes
do fechamento, o turno volta inteiro ao último fechamento: a peça da action, o arrasto do
mestre, a porta que ele abriu e as master actions correspondentes somem juntos. O histórico
nunca mostra uma master action de um turno que não foi gravado. Nenhum outro verbo encerra um
turno sem fechá-lo: `change_scene` é recusado com turno aberto, `change_round_mode` troca o
regime do MESMO round com o turno ainda aberto, e `kick_player` tira um jogador da mesa, não o
turno.

**O que fica de fora do "tudo junto"** — gravado na hora mesmo com turno aberto, ou fora da
transação do turno:

- a troca de regime ([`change_round_mode`](#change_round_mode)) — o evento em `match_events` e o
  `mode` da linha do round são gravados no instante: pertencem ao **round**, não ao turno, e
  sobrevivem a um reinício que perca o turno;
- a inscrição de um NPC — por [`add_npc`](#add_npc) ou pelo **pôr** de um NPC que ainda não
  participava — grava `match_participants` na hora; depois de um reinício que perca o turno, o
  NPC continua na partida, sem peça no tabuleiro.

O HP **não** é mais exceção: até 2026-10-02 os casos de uso o gravavam antes e fora da transação
do turno. Se a transação do turno falhar, o dano aplicado continua na ficha em memória — a mesa
já recebeu o [`character_hp_changed`](#character_hp_changed) —, e a sala guarda a ficha como
**não gravada**: o próximo fechamento bem-sucedido, qualquer que seja a ficha que ele atingir,
grava também as barras dela. Só as fichas que um fechamento atingiu são gravadas, nunca todas —
gravar uma ficha que a partida não tocou atropelaria uma edição feita nela por REST. Se o servidor
reiniciar antes desse próximo fechamento, a ficha volta com o HP do último fechamento
**gravado**.

**O fog memory do jogador volta pelo mesmo caminho.** `player_memories` é gravado junto com o
tabuleiro — no fechamento de turno, dentro da transação do `PersistTurnClose`; entre turnos, no
mesmo `persistBoard` —, então um jogador que reconecta longe de uma parede que já viu antes ainda a
recebe em `map_full_state` — a memória, não a visão atual, é o que decide (ver a seção de
fog em [`maps.md`](maps.md)).

**Por que a fila não volta** (design spec §4.2, "Por que só reconciliar"). As barras que uma
ação enfileirada cobra são cobradas na hora do enfileiramento e vivem só em memória
(`bars_updated`, sem tabela por trás). Uma fila persistida sobre barras zeradas por um
reinício seria um estado que nunca existiu — perder a fila é aceitável; fazê-la divergir do
que as barras mostram não é. Por isso `ownQueue` volta `[]` depois de um reinício, e não uma
versão "recuperada" das ações que estavam pendentes: ver a regra de reconciliação, acima, na
seção de [`match_full_state`](#match_full_state).

## 10. O que este contrato ainda não entrega

Registrado aqui para que a Fase 6 não descubra na integração. Fontes:
[`../match/flows/05-lacunas.md`](../match/flows/05-lacunas.md) e `AGENTS.md` § Known Issues.

| Lacuna | Consequência para o front |
|---|---|
| **`turn_opened.actionType` é sempre `""`** | Não ramifique por ele. |
| **A corrente de testes de `skills` não é executada** | `skills[].difficulty` é aceito e persistido, mas nenhuma margem atravessa de um teste para o próximo. A edição de perícias muda uma lista que ainda não decide nada. |
| **`ReboundDamage` nunca é aplicado ao ator** | Viaja no registro do turno, não vira dano. |
| **Armadura reduz zero** | Não existe entidade de armadura. A linha está codificada porque a forma importa. |
| **Remoção de NPC ao vivo não existe** | Tirar um NPC de uma sessão VIVA esbarra em ação dele na fila, turno aberto com ele como ator/alvo, reação pendente — regras que ninguém decidiu ainda. O REST `DELETE /matches/{uuid}/npcs/{sheet_uuid}` (ver [`match-npcs.md`](match-npcs.md)) continua funcionando, mas só vale para a próxima vez que a sala nascer: uma partida em andamento não some com o NPC removido, e não existe verbo de WS equivalente a `add_npc` no sentido contrário. |
| **A semântica de `Z` está em aberto** | `PieceMovedPayload.Z` é altura virtual em metros; `Move.Position[2]` é o índice `z` da grade — grandezas possivelmente diferentes, nunca reconciliadas. Por isso o servidor preserva o `Z` que a peça já tinha em vez de escrever `Move.Position[2]` sobre ele. Bloqueia qualquer cliente que queira escrever elevação até a pergunta "`Move.Position[2]` é metro ou índice de grade?" ser respondida. Vale para todo caminho que aplica movimento (ação de turno e reação, na abertura ou no fechamento) — é o mesmo `applyMove`. |
| **Colisão contra parede ainda não foi desenhada** | Não é omissão de validação: ainda não existe a regra que decide o que acontece quando um personagem colide com uma parede — compartilhar o slot, ser bloqueado, ou quebrar a parede no impacto são desfechos possíveis, e nenhum foi escolhido ainda. Até essa regra existir, o comportamento observável é a peça atravessando: a única checagem existente (`move=true`, `open=false`) roda no `enqueue_action`, quando `move.from` é não-zero — não de novo quando o movimento é de fato aplicado, na abertura do turno ou no fechamento. Vale para os DOIS momentos que deslocam peça — a ação do turno na abertura e a fuga no fechamento (para o destino, ou para a queda que o mestre escolheu): o deslocamento de uma reação nunca passa por essa checagem, porque `enqueue_action` roteia para reação (quando `reactToId` é não-zero) antes de alcançar o código que valida, e `attach_reaction`, enviado direto, entra sem essa checagem também. O front não deve tratar isso como bug a reportar — é regra de jogo que falta ser escrita. |
