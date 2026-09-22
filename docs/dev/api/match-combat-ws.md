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

> **NPC é do mestre.** `indexParticipants` mapeia toda ficha de NPC (`player_uuid == null`,
> `master_uuid` preenchido) para o `master_uuid` dela em `charToPlayer` — o mesmo mapa que
> autoriza jogadores comuns, só que a chave que bate é a do mestre. `enqueue_action` com
> `actorId` = a ficha do NPC passa pela mesma checagem `charToPlayer[actorId] == playerUUID`
> de sempre, e quem casa é o mestre. Vale tanto para o NPC que já estava no roster quando a
> sessão nasceu (`InitMatchSessionUC`) quanto para o que entrou depois, ao vivo, por
> [`add_npc`](#add_npc) — os dois caem no mesmo `charToPlayer`, pelo mesmo mecanismo (Decisão
> 6 do plano). **Ficha de jogador continua negada ao mestre:** o dono ali é o jogador, e isso
> não muda por quem enviou a mensagem.

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
| [`add_npc`](#add_npc) | mestre |

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
| [`npc_added`](#npc_added) | mesa inteira |
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

**Um `move` fora das três fugas também é recusado — checagem de presença, não de categoria.**
`ReactionKind.Displaces()` (`reaction_kind.go`) é o único lugar que sabe quais `reactionKind`
deslocam; `action_mapper.go` a consulta para recusar um `move` anexado a `dodge`, `closedDodge`
ou `nothing` com `reaction "X" must not carry a move`, **antes** de a reação ser anexada ao
turno. Sem essa checagem um cliente podia mandar uma esquiva **livre** carregando um `move` e
o servidor não recusava nada — o `move` ficava inerte no `Action` até [`open_reaction`](#open_reaction)
lê-lo, que então deslocava a peça de uma reação que não custa nada. `open_reaction` também
consulta `Displaces()` no momento de aplicar o `move`, como segunda linha de defesa — não
depende só da recusa do mapper.

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
| `invalid_action` | `reaction "X" must carry a dodge` / `a move` / `a repel` / `an evasion skill entry`; `reaction kind "X" is not in the catalogue`; `reaction "X" must move with Y, not Z` (categoria de `move` errada — ver a matriz acima); `reaction "X" must not carry a move` (`move` presente numa reação que **não** desloca — `dodge`, `closedDodge` ou `nothing` — checagem de presença, distinta da de categoria acima). |
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

**Dispara:** [`reaction_opened`](#reaction_opened) para a **mesa** (de quem é a vez de narrar
é público) e [`resolution_updated`](#resolution_updated) **master-only** (o cálculo continua
sendo do mestre — o turno ainda está aberto). E, **só para a fuga que não testa** (ver a
tabela abaixo), [`piece_moved`](#piece_moved-servidor) (servidor), **com o mesmo gate de campo
de visão** do resto do tabuleiro e **antes** de [`reaction_opened`](#reaction_opened), pela
mesma razão de ordem que vale para a ação do turno (ver
[`piece_moved`](#piece_moved-servidor)): a mesa não pode ver a narração abrir com a peça ainda
no slot velho.

#### O deslocamento de uma fuga tem CD, e a CD é o acerto do atacante

As três fugas (`escape`, `escapeGuard`, `closedEscape`) são as únicas reações com `Move` —
[`attach_reaction`](#attach_reaction) recusa qualquer outro `reactionKind` que tente carregar
um. **Toda reação tem uma CD, e ela é a ação que está sendo movida contra quem reage: o teste
de acerto do atacante** — o mesmo número contra o qual todo teste defensivo do turno já é
lido, e que chega ao mestre como `action.total` em
[`resolution_updated`](#resolution_updated). **Nada é rolado a mais para isso:** o acerto já
existe, e o lado de quem foge é a velocidade que já foi derivada quando a reação chegou (a
partir de `move.speed`). **Esse total não é projetado como tal em nenhum
[`resolution_updated`](#resolution_updated)** — o `reaction.total` de lá é a leitura da
ESQUIVA, não a do deslocamento —, então o cliente **não tem como prever o desfecho do passo**:
quem decide é o servidor, no fechamento, e o `piece_moved` (ou a ausência dele) é a resposta.
O número em si até chega ao fio depois, diluído em `moveSpeeds` de
[`bars_updated`](#bars_updated), junto com todas as outras velocidades que já gastaram a barra
de movimento na rodada — não dá para separar de lá qual era o desta fuga.

O que decide **quando** a peça anda é se o movimento da reação **rola** ou não:

| Reação | Movimento | Rola? | Quando a peça anda |
|---|---|---|---|
| `closedEscape` | **Shift** | não — `Brake` é lido passivo, não cai dado nenhum | **na abertura** da reação |
| `escape` | **Dash** | sim — `Accelerate` | **no fechamento do turno**, se passar |
| `escapeGuard` | **Dash** | sim — `Accelerate` | **no fechamento do turno**, se passar |

- **`Shift` não tem o que ser lido contra o acerto**, então o `closedEscape` pisa no instante
  em que ganha a palavra — é o comportamento que sempre existiu, e ele não mudou.
- **`Dash` rola**, e essa rolagem É o teste. Na abertura, `open_reaction` **mostra só a
  intenção**: nenhum `piece_moved` sai. O desfecho é decidido no fechamento do turno — ver
  [`close_turn`](#close_turn), que documenta os três verbos que fecham.
- **Passa** quando *a velocidade derivada do movimento* `>=` *o acerto do atacante* — o mesmo
  `>=` que decide `avoided`, só que com outro número do lado de quem foge (ver o alerta
  abaixo). Aí sai um `piece_moved`, com o mesmo gate de fog, direto para o slot pedido — nunca
  um slot intermediário.
- **Falhar é a peça NÃO sair do lugar.** Não existe posição intermediária: enquanto o motor
  não souber devolvê-la, uma falha de teste é imobilidade, não meio caminho.
- ⚠️ **Dano e deslocamento são desfechos INDEPENDENTES**, lidos contra a MESMA CD por números
  DIFERENTES: a esquiva pelo `reaction.total`, o passo pela velocidade do movimento. As quatro
  combinações existem — dá para escapar do golpe e não sair do lugar, e dá para tomar o dano
  cheio e ainda assim andar. `avoided` não prevê o `piece_moved`, nem o contrário.

`open_reaction` consulta `ReactionKind.Displaces()` (não só `Move != nil`) antes de aplicar —
uma segunda checagem, redundante com a recusa do `attach_reaction`, para o caso de um `Move`
chegar até aqui por qualquer outro caminho.

- **Um ator sem peça no tabuleiro é no-op silencioso**, como no lado da ação: nada é emitido,
  nenhum erro sai. Vale nos dois momentos (abertura e fechamento).
- **`Z` e o `Kind` do slot** (quadrado/hex) são **preservados**, pela mesma razão documentada em
  [`piece_moved`](#piece_moved-servidor) para o movimento de ação. É o mesmo `applyMove` nos
  dois momentos, não dois caminhos.
- O deslocamento de uma reação **nunca passa pela checagem de parede** — a colisão contra
  parede ainda não foi desenhada como regra; ver a última linha de §9.

⚠️ **O movimento de uma AÇÃO não mudou.** Uma ação não tem CD vindo contra ela; a peça dela
desloca **na abertura do turno**, qualquer que seja a categoria — a posição não pode esperar,
porque as reações seguintes dependem de onde a peça está.

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

1. [`piece_moved`](#piece_moved-servidor), **um por fuga de `Dash` que passou** — ver abaixo.
2. Persistência do turno (ação, reações, overrides, resolução liquidada).
3. [`turn_closed`](#turn_closed) — mesa.
4. [`resolution_updated`](#resolution_updated) **settled e projetado por destinatário**.
5. [`bars_updated`](#bars_updated) — mesa.

#### O fechamento é onde as fugas de `Dash` são decididas

Uma fuga com `Dash` (`escape`, `escapeGuard`) **não** deslocou na abertura: ela tem uma CD a
vencer, e a CD é **o teste de acerto do atacante** (ver [`open_reaction`](#open_reaction)).
É aqui que o servidor diz o desfecho:

- **passou** (velocidade derivada do movimento `>=` acerto) → sai um
  [`piece_moved`](#piece_moved-servidor) para o
  slot pedido, com o mesmo gate de campo de visão, **antes** do
  [`turn_closed`](#turn_closed) — a mesa não pode ver o turno acabar com a peça no slot velho;
- **falhou** → **nada sai**. A peça não sai do lugar, e não existe posição intermediária.

Uma fuga com `Shift` (`closedEscape`) já andou lá atrás, na abertura da reação, e **não** é
deslocada de novo aqui.

⚠️ **Os três verbos que fecham um turno decidem isso igual.** `close_turn` é o explícito;
[`open_next_action`](#open_next_action) e [`pull_action`](#pull_action) também fecham o turno
aberto a caminho de abrir o próximo, e **os três** aplicam o desfecho das fugas. Um escape
cujo resultado dependesse de qual verbo o mestre usou seria bug, não regra. (Nesses dois o
`piece_moved` sai antes da persistência e antes do [`turn_opened`](#turn_opened) do próximo
turno; não há `turn_closed` nesse caminho — ver [`turn_closed`](#turn_closed).)

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
    "bars": ["action", "move"]
  }
}
```

**É master-only porque a fila é secreta** (`combat-engine.md` § *As barras são públicas*: "a
fila é secreta; a barra e a ordem são públicas"). Um jogador que aprendesse o que está
pendente leria as intenções da mesa no wire.

**E é o jeito de aprender o `actionId` que [`pull_action`](#pull_action) exige, no instante do
enfileiramento.** Sem esta mensagem, `pull_action` é inalcançável a partir de um cliente real
para o que acabou de entrar na fila.

Nada aqui descreve o **conteúdo** da ação: arma, alvo, perícia e dados continuam do jogador
até o mestre abrir o turno. `bars` (`action` e/ou `move`) já é dedutível da ordem pública em
`bars_updated` — nomear aqui não revela nada novo.

**Disparado por:** `enqueue_action` aceito, logo após o ack de quem enviou.

> **Esta mensagem dispara UMA VEZ, no instante do enfileiramento** — um mestre que estava
> desconectado nesse momento não a recebe depois. [`match_full_state`.`queue`](#match_full_state)
> é a versão do MESMO fato que **sobrevive à reconexão**: um payload `action_queued` inteiro
> por ação ainda pendente, na ordem de inserção da fila. Ver a seção de `match_full_state`.

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
| `targets[].reaction` | **Ausente** quando nada foi aberto e as passivas (esquiva por reflexo, depois defesa) se aplicaram em silêncio — `Reaction *ReactionResultPayload` com `omitempty` **omite a chave**, não emite `null`; em TypeScript o campo é `reaction?: ReactionResultPayload`, não `reaction: ReactionResultPayload \| null`. Uma passiva silenciosa não é resposta a reportar. |
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
      "actorId": "11111111-1111-4111-8111-111111111111"
    },
    "resolution": {
      "turnId": "55555555-5555-4555-8555-555555555555",
      "isSettled": false,
      "action": { "skillName": "Accuracy", "skillValue": 14, "diceRolled": [6, 8], "total": 20, "isCritical": false, "isCriticalFailure": false },
      "targets": []
    },
    "queue": [
      { "actionId": "33333333-3333-4333-8333-333333333333", "actorId": "11111111-1111-4111-8111-111111111111", "bars": ["action"] }
    ]
  }
}
```

| Campo | Notas |
|---|---|
| `scene` | O payload de [`scene_changed`](#scene_changed) **inteiro** — `sceneId`/`category`/`briefInitialDescription`, os mesmos nomes, a mesma struct. Não é uma segunda forma para os mesmos três valores. **Ausente** (`omitempty`) quando a partida não tem cena ativa — `Scene *SceneChangedPayload` com `omitempty` **omite a chave inteira**, não emite `null`; em TypeScript o campo é `scene?: SceneChangedPayload`, não `scene: SceneChangedPayload \| null`. Leia `scene === undefined` como "sem cena", não como "cena sem nome". `category` é minúscula (`"battle"`/`"roleplay"`) e é validada contra o enum — ver `change_scene`. |
| `roundMode` | O regime do round ativo — `"Free"` ou `"Race"`, os mesmos valores de [`round_mode_changed`](#round_mode_changed). Vem `""` quando não há round ativo. Público: vai para todo mundo que conecta. |
| `bars` | O `bars_updated` **inteiro**, reaproveitado — não é uma segunda forma para manter em sincronia com a primeira. |
| `bars.seq` | ⚠️ **É o contador CORRENTE, não um novo.** O cliente guarda o maior `seq` já aplicado e descarta qualquer coisa menor; estampar um número novo aqui zeraria essa guarda numa reconexão — o primeiro `bars_updated` atrasado a chegar depois seria aplicado por cima de um estado mais novo. É por isso que a proteção do cliente contra snapshot atrasado atravessa a reconexão: o contador nunca reinicia. |
| `openTurn` | Ausente (`omitempty`) **para todo destinatário** — jogador ou mestre — quando a mesa está em "fechado e nada aberto", estado em que ela pode legitimamente estar. Quando presente, vai para **todo mundo** que conecta: quem é o ator da vez não é segredo. |
| `resolution` | O cálculo do turno aberto, **master-only**. Ausente para qualquer outro destinatário, e também ausente para o próprio mestre quando não há turno aberto. Mesmos dois eixos de `resolution_updated` (§6) — aqui só o eixo do TEMPO se manifesta, porque um snapshot de conexão sempre reflete um turno em aberto (`isSettled: false`); não existe um `match_full_state` de turno fechado. |
| `queue` | A fila do mestre, **master-only pelo mesmo eixo de `resolution`** — ausente para qualquer outro destinatário. Um payload de [`action_queued`](#action_queued) **inteiro** por ação ainda pendente, na **ordem de inserção** da fila (não confundir com `bars.order`, que carrega a ordem *projetada* de execução — public, sem identidade de ação). `omitempty`: **ausente** significa fila vazia, não erro. Existe **com ou sem turno aberto** — o estado mais comum de reconectar é justamente "nada aberto ainda, três coisas esperando". É a versão de `action_queued` que **sobrevive à reconexão**; ver a nota na seção de `action_queued`. |

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
  `isMaster` do despacho (ou nenhum eco, se foi ele quem arrastou — o navegador dele já
  aplicou). Vale para os três caminhos que passam por `relayPieceMove`: arrasto do cliente,
  o movimento de uma ação de turno e a fuga de reação. Uma peça cujo `characterId` não está
  na partida também não tem dono, e ninguém recebe o extra.
- O dono **offline** tem o cache recalculado mas não recebe nada (ele reconectaria num
  polígono velho).
- Se o recálculo falhar, o `map_full_state` não sai — o `piece_moved` do par acima sai do
  mesmo jeito.

**Disparado por** três momentos, e é o **mesmo** `applyMove` nos três:

| Momento | Quando |
|---|---|
| Abertura do turno (`open_next_action`, `pull_action`) | a ação que abre carrega um `Move`. **Qualquer categoria** — uma ação não tem CD vindo contra ela, e a posição não pode esperar, porque as reações seguintes dependem de onde a peça está. |
| Abertura da reação ([`open_reaction`](#open_reaction)) | a fuga que ganha a palavra anda de **`Shift`** (`closedEscape`). O `Shift` não rola nada, então não há teste a ser lido contra o acerto do atacante. |
| Fechamento do turno (`close_turn`, e também `open_next_action`/`pull_action`) | a fuga anda de **`Dash`** (`escape`, `escapeGuard`) **e passou** contra o acerto do atacante. Falhou → nada sai, a peça não sai do lugar. |

As regras completas do lado da reação — de onde vem a CD, qual categoria decide quando, e o
que significa falhar — estão em [`open_reaction`](#open_reaction) e em
[`close_turn`](#close_turn). Um movimento de **ação** que dependesse de teste (um salto, um
aperto, um pouso em slot ocupado) continua **sem caso alcançável**: `move.category` só aceita
`Dash` e `Shift`, e as outras cinco são recusadas no mapeamento — então não existe código para
esse ramo.

Um ator **sem peça no tabuleiro** não é erro, em nenhum dos três momentos: não há o que mover,
nada é emitido e nenhuma mensagem de erro sai. O turno (ou a reação) abre normalmente.

⚠️ **A semântica de `Z` está em aberto.** `PieceMovedPayload.Z` é documentado como altura
virtual em metros; `Move.Position[2]` é o índice `z` da grade. São grandezas possivelmente
diferentes, e por isso o servidor **preserva o `Z` que a peça já tinha** em vez de
sobrescrevê-lo com `Move.Position[2]`. Escrever um horizontal sobre um vertical derrubaria
uma peça elevada ao chão a cada passo horizontal cujo `z` de grade for `0`. A pergunta
"`Move.Position[2]` é metro ou índice de grade?" precisa de resposta antes de qualquer
cliente escrever `Z`. Ver §9.

⚠️ **Colisão contra parede é uma fatia de regra ainda não desenhada — o efeito hoje é que a
peça atravessa.** A única checagem existente contra paredes com `move=true` e `open=false`
acontece no **enfileiramento** (`enqueue_action`, quando `move.from` é não-zero — ver a
tabela em `enqueue_action`), não de novo aqui na abertura. O caminho de reação **nunca passa
por essa checagem, nem uma vez**: quando `reactToId` é não-zero, `enqueue_action` roteia para
o mesmo tratamento de `attach_reaction` e retorna **antes** de alcançar o código que valida a
parede — esse código só existe no ramo de ação comum. `attach_reaction`, enviado direto,
também não tem checagem nenhuma no caminho. Não é validação esquecida: **ainda não existe a
regra que decide o que acontece quando um personagem colide com uma parede** — compartilhar
o slot, ser bloqueado, ou quebrar a parede no impacto são desfechos possíveis, e nenhum foi
escolhido ainda. Ver §9.

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
| `forbidden` | `"only the master can perform this action"` | `open_next_action`, `pull_action`, `open_reaction`, `edit_action`, `close_turn`, `change_round_mode`, `change_scene`, `enqueue_master_action`, `add_npc`. |
| `match_not_started` | `"match session not initialized"` — a partida não foi iniciada. | Todas as de partida (exceto `add_npc`) — na sala sem sessão, `add_npc` é caminho de sucesso (Decisão 3), não erro. |
| `invalid_action` | Payload bem formado, conteúdo inválido: perícia/arma/categoria de Nen desconhecida, reação sem componente obrigatório, `actorId` ausente, `reactToId`/`reactionKind` desemparelhados, **categoria de cena** fora de `"battle"`/`"roleplay"`. | `enqueue_action`, `attach_reaction`, `edit_action`, `change_scene`. |
| `move_blocked` | `"movement blocked by a wall"` | `enqueue_action` com `move.from` não-zero. |
| `not_found` | Partida ou ficha de personagem não encontrada — mapeia `ErrMatchNotFound`/`ErrCharacterSheetNotFound` de `AddMatchNPCUC`. | `add_npc`. |
| `invalid_npc` | Ficha não é NPC, não pertence ao mestre nem à campanha, ou a partida já encerrou — mapeia `ErrSheetNotNPC`/`ErrSheetNotOwnedByMaster`/`ErrMatchAlreadyFinished`. | `add_npc`. |
| `npc_already_in_match` | O NPC já está na SESSÃO viva (`ErrCharacterAlreadyInSession`) — **não confundir com a duplicata do banco**, que este verbo tolera de propósito (ver [`add_npc`](#add_npc)). | `add_npc`. |
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
    │◄─ piece_moved (fog-gated) ─┤  (SÓ p/ fuga de Dash que PASSOU contra o acerto)   │
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
igual (com persistência, `resolution_updated` liquidado e o `piece_moved` das fugas de `Dash`
que passaram), mas **não sai `turn_closed`** — o próximo `turn_opened` é o que anuncia a
virada. E se nada pendente ainda puder pagar, sai
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
| **Remoção de NPC ao vivo não existe** | Tirar um NPC de uma sessão VIVA esbarra em ação dele na fila, turno aberto com ele como ator/alvo, reação pendente — regras que ninguém decidiu ainda. O REST `DELETE /matches/{uuid}/npcs/{sheet_uuid}` (ver [`match-npcs.md`](match-npcs.md)) continua funcionando, mas só vale para a próxima vez que a sala nascer: uma partida em andamento não some com o NPC removido, e não existe verbo de WS equivalente a `add_npc` no sentido contrário. |
| **A semântica de `Z` está em aberto** | `PieceMovedPayload.Z` é altura virtual em metros; `Move.Position[2]` é o índice `z` da grade — grandezas possivelmente diferentes, nunca reconciliadas. Por isso o servidor preserva o `Z` que a peça já tinha em vez de escrever `Move.Position[2]` sobre ele. Bloqueia qualquer cliente que queira escrever elevação até a pergunta "`Move.Position[2]` é metro ou índice de grade?" ser respondida. Vale para todo caminho que aplica movimento (ação de turno e reação, na abertura ou no fechamento) — é o mesmo `applyMove`. |
| **Colisão contra parede ainda não foi desenhada** | Não é omissão de validação: ainda não existe a regra que decide o que acontece quando um personagem colide com uma parede — compartilhar o slot, ser bloqueado, ou quebrar a parede no impacto são desfechos possíveis, e nenhum foi escolhido ainda. Até essa regra existir, o comportamento observável é a peça atravessando: a única checagem existente (`move=true`, `open=false`) roda no `enqueue_action`, quando `move.from` é não-zero — não de novo quando o movimento é de fato aplicado, na abertura do turno ou da reação. Vale para os TRÊS momentos que deslocam peça — a ação do turno na abertura, a fuga de `Shift` em `open_reaction`, e a fuga de `Dash` que passou, no fechamento: o deslocamento de uma reação nunca passa por essa checagem, porque `enqueue_action` roteia para reação (quando `reactToId` é não-zero) antes de alcançar o código que valida, e `attach_reaction`, enviado direto, entra sem essa checagem também. O front não deve tratar isso como bug a reportar — é regra de jogo que falta ser escrita. |
