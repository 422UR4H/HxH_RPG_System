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
