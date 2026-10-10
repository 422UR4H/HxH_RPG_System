# Cancelar ação na fila — pacote de back — design

> **Escopo:** o verbo de cancelar uma ação ainda na fila, que o documento mestre
> ([`2026-09-20-front-combat-phases.md`](2026-09-20-front-combat-phases.md) §4.3, §6 "fora de
> escopo") registrava como "fatia futura de back + contrato". Decidido pelo dono do produto em
> 2026-10-10, depois de testar a Fase 8: o "×" de "Suas ações declaradas" só ocultava a linha, e a
> ação "cancelada" abriu mesmo assim. **Um PR, repo `System_X_System`.** O front consome no PR de
> polimento da mesa (repo `System_X_System_React`, branch `feat/combat-table-polish`).
>
> Regra de jogo: [`../../game/combate/acoes.md`](../../game/combate/acoes.md) — "Uma ação
> específica pode ser cancelada e removida da fila".
> Contrato: [`../../dev/api/match-combat-ws.md`](../../dev/api/match-combat-ws.md).
> Plano: [`../plans/2026-10-10-combat-cancel-action-back.md`](../plans/2026-10-10-combat-cancel-action-back.md).
>
> **Branch:** `feat/combat-cancel-action`, a partir de `feat/combat-phase-8-regency-back`
> (`abd85d1`, PR #85 ainda aberto). Rebase em `main` depois do merge do #85.

## 1. Decisões do dono do produto (2026-10-10)

| # | Decisão |
|---|---|
| C1 | **Quem cancela:** cada um só o que declarou — o jogador as ações dos personagens dele; o mestre as dos NPCs. É o mesmo critério do `enqueue_action` (`charToPlayer[actorId] == quem pede`; o NPC mapeia para o mestre). O mestre cancelar a ação de um jogador fica de fora. |
| C2 | **Só ação ainda na fila.** A ação que já abriu é o turno — não se cancela. Uma ação consumida por uma reação já saiu da fila — também não. |
| C3 | **Nada a devolver nas barras.** A velocidade só é cobrada quando a ação abre (`recordActed`); enquanto está na fila ela não pagou nada. |
| C4 | **O mestre fica sabendo na hora**, e a barra geral (ordem pública) se refaz. |
| C5 | O front pede confirmação antes de mandar (não é assunto do back). |

## 2. O verbo

### `cancel_action` (cliente → servidor)

```json
{ "type": "cancel_action", "payload": { "actionId": "33333333-3333-4333-8333-333333333333" } }
```

**Quem:** qualquer cliente da partida; a autorização é por ação (C1).

O `actionId` é o que o cliente já conhece: o jogador pelo `action_enqueued` (ou pelo
`match_full_state.ownQueue`), o mestre pelo `action_queued` (ou pelo `match_full_state.queue`).

### `action_cancelled` (servidor → cliente)

```json
{ "type": "action_cancelled", "payload": { "actionId": "33333333-3333-4333-8333-333333333333" } }
```

**Destino:** o mestre **e** quem pediu (quando quem pediu não é o mestre). Um mestre que cancela a
ação de um NPC recebe uma só. Nunca broadcast: a fila é secreta — a mesa só vê a ordem pública
mudar, pelo `bars_updated`.

**Dispara, nesta ordem:** `action_cancelled` (mestre, depois quem pediu) → `bars_updated` (mesa).

**Erros:**

| `code` | Quando |
|---|---|
| `invalid_payload` (`"invalid cancel_action payload"`) | payload não parseia, ou `actionId` ausente/nulo |
| `match_not_started` | sala sem sessão |
| `game_error` (`action not found in queue`) | a ação não está na fila: já abriu, foi consumida por uma reação, já foi cancelada, ou nunca existiu |
| `game_error` (`action actor does not match player`) | a ação é de um personagem que não é de quem pediu |
| `game_error` (`participant not found in match session`) | quem pediu não participa da partida |

A ordem de checagem é "existe na fila" → "é de quem pediu". Um id que não está na fila responde
`action not found in queue` para qualquer um — não vaza a existência da ação de outro (a fila é
secreta). Uma ação de outro personagem que **está** na fila responde o erro de autorização; isso
diz a um jogador que adivinhou um UUID que ele existe, o que é aceitável (UUID v4 não se adivinha)
e é o mesmo comportamento do `enqueue_action`.

## 3. Onde mora

- **Domínio** — `MatchSession.CancelAction(playerUUID, actionID uuid.UUID) error`
  (`internal/domain/match/matchsession/match_session.go`, ao lado de `EnqueueAction`): acha a
  ação em `s.activeQueue.All()`; ausente → `service.ErrActionNotFound`; checa `charToPlayer` como o
  `EnqueueAction` (mesmos erros, mesma ordem: dono errado e participante → `ErrActionActorMismatch`;
  não participante → `ErrParticipantNotFound`); remove com `s.activeQueue.ExtractByID`. Não mexe em
  preço congelado, saldo nem rodada: um preço já congelado com a velocidade da ação cancelada
  continua congelado (`FreezePrices` só congela uma vez por barra — é a regra de sempre, "uma ação
  mais lenta que chega depois não re-precifica a rodada", vale também para uma que sai). Cancelar
  não fecha a rodada: só `open_next_action` fecha (contrato, `round_closed`).
- **Use case** — `internal/application/match/cancel_action.go`: `ICancelAction` /
  `CancelActionUC`, fino como o `EnqueueActionUC` (só delega à sessão).
- **Room** — `case MsgTypeCancelAction` em `room.go`, logo depois de `MsgTypeEnqueueAction`:
  parse → sessão → `r.mu.Lock()` → `Execute` → `r.mu.Unlock()` → envia. Nada enviado com `r.mu`
  preso. `RoomDeps.CancelActionUC ICancelAction` novo, ao lado de `EnqueueActionUC`; preenchido
  em `cmd/game/main.go`. Segue a convenção do arquivo ("field left nil is a capability the room
  does not have — the arms that need it check"): com ele nil, o braço responde
  `match_not_started`, como se não houvesse sessão. Os testes que montam `RoomDeps` à mão não
  precisam mudar, exceto os e2e novos, que passam `appmatch.NewCancelActionUC()`.
- **Mensagens** — `message.go`: `MsgTypeCancelAction = "cancel_action"`,
  `MsgTypeActionCancelled = "action_cancelled"`, `CancelActionPayload{ActionID uuid.UUID json:"actionId"}`,
  `ActionCancelledPayload{ActionID uuid.UUID json:"actionId"}`.

## 4. Persistência e reconexão

Nada é gravado: a fila vive só em memória (`activeQueue`) e uma ação só vira linha em `actions`
quando o turno dela fecha. Por isso cancelar não deixa rastro no histórico — o histórico é do que
aconteceu, e a ação cancelada não aconteceu.

A reconexão já sai certa sem código novo: `match_full_state.queue` (mestre) e `.ownQueue`
(jogador) leem a fila viva, e a ação cancelada não está mais nela.

## 5. Testes

- **Unidade (domínio)**, `match_session` (pacote e arquivo de teste que o `EnqueueAction` já usa):
  cancela a própria (sai de `PendingActions`); NPC pelo mestre; ação de outro jogador →
  `ErrActionActorMismatch` e a fila intacta; não participante → `ErrParticipantNotFound`; id
  ausente → `ErrActionNotFound`; ação que já abriu (`OpenNextAction` antes) → `ErrActionNotFound`;
  cancelar duas vezes → a segunda é `ErrActionNotFound`.
- **E2E contra a `Room` real** (`internal/app/game/cancel_action_e2e_test.go`, `package game_test`,
  no molde de `queue_e2e_test.go`/`own_queue_e2e_test.go`): jogador cancela a sua → ele e o mestre
  recebem `action_cancelled`, o outro jogador **não**, todos recebem `bars_updated` sem a entrada;
  o mestre que reconecta acha a `queue` sem a ação; o jogador que reconecta acha `ownQueue: []`;
  `open_next_action` depois não abre a cancelada; jogador tenta cancelar a de outro →
  `game_error` e a fila do mestre intacta; payload inválido → `invalid_payload`.

## 6. Documentação

- `match-combat-ws.md`: `cancel_action` em §4 (depois de `enqueue_action`... na ordem do índice),
  `action_cancelled` em §5 (depois de `action_queued`), as duas linhas no índice §3, os erros no
  catálogo §7 (`game_error` já cobre; acrescentar `cancel_action` às listas onde couber), e a nota
  de `action_enqueued` ("não cancelava") passa a apontar para o verbo.
- Documento mestre: §4.3 e o "fora de escopo" da §6 dizem que o verbo existe agora (data e PR).
- `docs/game/combate/acoes.md` já diz a regra; nada muda lá.
- `AGENTS.md` Known Issues: uma linha.
- `docs/documentation-map.yaml`: o arquivo novo do use case, se o mapa lista arquivos de
  `internal/application/match/` um a um (siga o padrão do mapa).

## 7. Fora de escopo

- O mestre cancelar a ação de um jogador (C1).
- Cancelar uma reação já anexada (`attach_reaction`) — reação não está na fila; é outro verbo, se
  um dia fizer falta.
- Qualquer registro do cancelamento no histórico (§4).

---

## 8. Adendo (2026-10-10, mesmo PR): cancelar custa Desvantagem, e a cancelada fica registrada

Decidido pelo dono do produto depois da revisão final, que achou a lacuna: com o verbo novo,
cancelar e depois reagir ou declarar de novo escapava da Desvantagem que `reacoes.md` cobra de
quem **troca** de ação. Supera C3 só no que diz respeito à Desvantagem (barras continuam sem
devolução nem cobrança).

### 8.1 Decisões

| # | Decisão |
|---|---|
| P1 | **Cancelar marca o personagem até o fim da rodada.** Toda ação que ele declarar dali em diante, e toda reação dele que cobra barra (repelir, os três escapes), deriva a velocidade com Desvantagem (`systemBias = -1`) — a mesma da troca de ação de hoje: `actionSpeed` (Legerity) sempre; `moveSpeed` quando a ação ou a reação move. Acerto, esquiva, defesa e dano **não** pioram (dono do produto: "o acerto, a esquiva e o dano dele não pioram"). |
| P2 | **Só o que vem depois do cancelamento, só na rodada em que ele aconteceu.** As outras ações que ele já tinha na fila já rolaram e não são re-derivadas. |
| P3 | **Só no regime Race.** No Free a velocidade é passiva e o viés não teria efeito; o cancelamento em Free não marca ninguém. |
| P4 | **Não acumula.** Marca + consumo numa mesma reação continua sendo uma Desvantagem só (`-1`), como a troca já é. Esquiva, esquiva fechada, "não fazer nada" e as passivas não rolam velocidade e continuam sem custo. |
| P5 | **O perdão do mestre** fica com as alavancas de hoje (abrir antes com `pull_action`, deixar acontecer). Nenhum verbo novo de "perdoar". |
| P6 | **A ação cancelada é gravada**, em tabela própria `cancelled_actions` (não em `actions`, que exige turno — a cancelada nunca teve; mesmo motivo de `master_actions`, spec 2026-09-27 §4.8), com referência à **rodada** (e à cena e à partida) em que ela aconteceria; `turn_uuid` = o turno aberto no instante do cancelamento, ou NULL. |
| P7 | **Quando grava:** a mesma regra das master actions — com turno aberto, junto com o fechamento desse turno (`turnWrites` → `PersistTurnClose`, mesma transação); sem turno aberto, na hora (`ensureSceneAndRoundRows` antes, como `recordMasterAction`). Um reinício no meio do turno perde o registro junto com o turno, de propósito. |
| P8 | **No histórico** (`GET /history`): um evento de rodada `kind: "actionCancelled"` em `rounds[].events`, na ordem do tempo. O **mestre** vê `actorId` + `action` (nível `Full`); o **dono** do personagem vê `actorId` + `action` (nível `Declaration`, o mesmo do `ownQueue`); **os outros** veem só `actorId` ("Fulano cancelou uma ação") — não revela nada que a barra geral pública já não tenha mostrado. |

### 8.2 Onde mora

- **A marca:** `MatchSession.swapMarks map[uuid.UUID]uuid.UUID` (personagem → id da rodada em que
  cancelou). Vale enquanto o id for o da rodada ativa — sem gancho de reset: a rodada seguinte
  tem outro id. `CancelAction` grava a marca quando o regime ativo é Race. Um helper
  `swapBias(charID) int` devolve `-1` com marca válida, senão `0`.
- **Quem lê:** `EnqueueAction` (hoje `deriveSpeeds(a, 0)` → `deriveSpeeds(a, s.swapBias(actor))`)
  e `AttachReaction` para reação não-livre (`systemBias = -1` se consumiu **ou** se marcado).
  `Action.SystemBias` guarda o viés, então a re-derivação de uma edição do mestre o preserva (já é assim).
- **O registro:** `CancelAction` passa a devolver a ação removida (`(*action.Action, error)`), para
  a `Room` montar o registro. `cancelledaction.Record` (pacote novo em
  `internal/domain/match/entity/` ou ao lado de `masteraction` — siga onde `masteraction` mora):
  `UUID` (= id da ação), `MatchUUID`, `SceneUUID`, `RoundUUID`, `TurnUUID *uuid.UUID`,
  `ActorUUID` (ficha), `CancelledBy` (usuário), `Action action.Action`, `CancelledAt`.
- **Tabela** (migração nova): `cancelled_actions (uuid PK, match_uuid FK matches ON DELETE
  CASCADE, scene_uuid FK scenes, round_uuid FK rounds, turn_uuid NULL sem FK (como em
  master_actions), actor_uuid FK character_sheets, cancelled_by FK users, action JSONB NOT NULL,
  cancelled_at TIMESTAMPTZ NOT NULL)` + índice `(match_uuid, cancelled_at)`. `action` guarda a
  ação no **formato de armazenamento do domínio** (as mesmas peças JSON que `insertAction` grava
  nas colunas de `actions` — reaproveite o mapeamento, não invente outro), para o histórico
  reconstruir uma `action.Action` e projetá-la com `actionwire.From`, como já faz com `actions`.
- **Gravação:** gateway `internal/gateway/pg/cancelledaction` (Insert) e, no `PersistTurnClose`,
  `TurnCloseData.CancelledActions` inserido na mesma transação; `turnWrites` ganha
  `cancelledActions`. `RoomDeps.CancelledActionRepo` (nil = não grava, a convenção).
- **Leitura:** um lister no use case do histórico, como `IMasterActionLister`; o `stitch` põe cada
  registro em `rounds[].events` com `Kind: HistoryEventActionCancelled`; a projeção por leitor
  (P8) segue o padrão de visibilidade do use case (mestre / dono / resto).

### 8.3 Contrato

- `cancel_action`: a Desvantagem até o fim da rodada (Race), a gravação e onde aparece.
- `match-history.md`: `events[].kind` ganha `"actionCancelled"`, com `actorId` e `action?` e a
  tabela de visibilidade de P8.
- `combat-engine.md` (tabela de custos da troca) e `docs/game/combate/acoes.md` ou
  `reacoes.md`: uma frase de regra — "cancelar uma ação também é trocar: até o fim da rodada, o
  que você fizer de novo rola a velocidade com Desvantagem".

### 8.4 Testes

- Domínio: Race, cancela e declara → velocidade com `SystemBias -1`; Free → 0; cancela, rodada
  fecha, declara → 0; cancela e reage com repelir sem nada na fila → `-1`; cancela e esquiva →
  livre, sem viés; consumo + marca → `-1` (não `-2`); outra ação que já estava na fila não muda.
- Gateway (integração, `-tags integration`): insert e leitura de `cancelled_actions`; o
  `PersistTurnClose` grava as canceladas do turno na transação.
- E2E: cancelar sem turno aberto grava na hora; com turno aberto grava só no fechamento; `GET
  /history` (handler real) projeta mestre/dono/outro.
