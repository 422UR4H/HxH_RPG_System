# Cancelar ação na fila — pacote de back — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** o verbo `cancel_action` — quem declarou tira a própria ação da fila antes de ela abrir, o
mestre fica sabendo na hora (`action_cancelled`) e a ordem pública se refaz (`bars_updated`).

**Architecture:** `MatchSession.CancelAction` (domínio, mesma autorização do `EnqueueAction`) →
`CancelActionUC` (fino) → braço `MsgTypeCancelAction` em `room.go` (lock, executa, solta, envia).
Nada persiste: a fila é só memória.

**Tech Stack:** Go 1.23, gorilla/websocket, `testing` padrão.

**Spec:** [`docs/superpowers/specs/2026-10-10-combat-cancel-action-back-design.md`](../specs/2026-10-10-combat-cancel-action-back-design.md)
— leia inteiro antes da primeira tarefa.

**Branch:** `feat/combat-cancel-action` (já criada a partir de `feat/combat-phase-8-regency-back`).

## Global Constraints

- Go 1.23; `testing` padrão, table-driven com `t.Run`; **TDD**. Os `*_e2e_test.go` de `internal/app/game/` são `package game_test`; `match_session_test.go` é o pacote que ele já declara.
- `room.go` é dono do lock (`r.mu`). **Nada que envia a cliente roda com `r.mu` preso.**
- Wire em **camelCase**.
- **Nunca remover comentários `TODO`.** Comente o porquê, na densidade dos vizinhos (este repo comenta bastante).
- Todo item que muda o wire atualiza `docs/dev/api/match-combat-ws.md` **no mesmo commit** (Task 1 escreve a parte do contrato; Task 2 o resto da documentação).
- **O checkout tem 6 arquivos `_test.go` modificados só por gofmt, que NÃO são desta branch** (`internal/app/game/connection_e2e_test.go`, `rehydrate_race_e2e_test.go`, `internal/application/match/enqueue_action_test.go`, `list_match_enrollments_test.go`, `match_uc_test.go`, `start_match_test.go`). Não os commite, não os reverta, não rode `gofmt -w` em diretório: `git add` só caminho por caminho, e `gofmt -l`/`-w` só nos arquivos que você criou ou editou.
- Verificação por tarefa: `go build ./...`, `go vet ./...`, `go vet -tags integration ./...`, `go vet -tags smoke ./...`, `go test ./internal/...`, `go test -race ./internal/app/game/`. Antes do PR: `golangci-lint run ./...`.
- Commits terminam com `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Docs em PT-BR; nomes de código em inglês.

## Review Focus

1. **Ação de outro personagem** — o pedido é recusado e a fila do mestre continua com a ação (teste e2e e de domínio).
2. **Ação que já abriu** — `action not found in queue`; o turno aberto não é tocado.
3. **Bystander** — o terceiro jogador não recebe `action_cancelled` (a fila é secreta); só vê `bars_updated`.
4. **Reconexão** — `match_full_state.queue` (mestre) e `.ownQueue` (dono) sem a ação.
5. **Lock** — nenhum `SendMessage`/`sendToMaster`/`broadcastBars` dentro de `r.mu`.

---

### Task 1: o verbo, de ponta a ponta

**Files:**
- Modify: `internal/domain/match/matchsession/match_session.go` (depois de `EnqueueAction`)
- Test: `internal/domain/match/matchsession/match_session_test.go`
- Create: `internal/application/match/cancel_action.go`
- Modify: `internal/app/game/message.go`, `internal/app/game/room_deps.go`, `internal/app/game/room.go`, `cmd/game/main.go`
- Create: `internal/app/game/cancel_action_e2e_test.go`
- Modify: `docs/dev/api/match-combat-ws.md`

**Interfaces:**
- Produces: `func (s *MatchSession) CancelAction(playerUUID, actionID uuid.UUID) error`; `appmatch.ICancelAction`, `appmatch.NewCancelActionUC()`; `game.MsgTypeCancelAction`, `game.MsgTypeActionCancelled`, `game.CancelActionPayload`, `game.ActionCancelledPayload`; `RoomDeps.CancelActionUC`.

- [ ] **Step 1: testes de domínio que falham** — `TestMatchSession_CancelAction` em `match_session_test.go`, no molde de `TestMatchSession_EnqueueAction` (use `makeParticipant`, `makeAction`, `matchsession.NewMatchSession`). Casos (spec §5): cancela a própria e ela some de `s.PendingActions()`; um NPC (`makeParticipant` com `MasterUUID` e sem `PlayerUUID` — veja como `indexParticipants`/linhas 155–165 de `match_session.go` mapeiam o NPC para o mestre, e monte o participante assim) cancelado pelo mestre; ação do personagem de outro jogador → `errors.Is(err, matchsession.ErrActionActorMismatch)` e a fila ainda com ela; não participante (`uuid.New()`) → `ErrParticipantNotFound`; id inexistente → `errors.Is(err, service.ErrActionNotFound)`; ação aberta por `s.OpenNextAction()` → `ErrActionNotFound`; cancelar duas vezes → a segunda `ErrActionNotFound`.

- [ ] **Step 2: rode e veja falhar** — `go test ./internal/domain/match/matchsession/ -run TestMatchSession_CancelAction` → não compila (`CancelAction` indefinido).

- [ ] **Step 3: implemente o domínio.**

```go
// CancelAction takes one of the caller's own actions out of the queue before it opens
// (acoes.md: "Uma ação específica pode ser cancelada e removida da fila").
//
// Same authorization as EnqueueAction — whoever may declare through a character may withdraw
// what they declared through it; the master's NPCs map to the master in charToPlayer. The queue
// is looked up FIRST: an id that is not pending answers ErrActionNotFound to anyone, so the
// answer never tells a caller whether someone else's action exists (the queue is secret).
//
// Nothing is refunded and nothing else moves: a queued action has paid nothing yet (speeds are
// recorded when an action opens), and a price already frozen stays frozen — the same rule as a
// slower action arriving after the freeze. Cancelling never closes the round; only
// OpenNextAction does.
func (s *MatchSession) CancelAction(playerUUID, actionID uuid.UUID) error {
	var target *action.Action
	for _, a := range s.activeQueue.All() {
		if a.GetID() == actionID {
			target = a
			break
		}
	}
	if target == nil {
		return service.ErrActionNotFound
	}
	owner, ok := s.charToPlayer[target.GetActorID().String()]
	if !ok || owner != playerUUID {
		if _, isParticipant := s.participants[playerUUID]; !isParticipant {
			return ErrParticipantNotFound
		}
		return ErrActionActorMismatch
	}
	s.activeQueue.ExtractByID(actionID)
	return nil
}
```

  Atenção: o mestre não está em `participants` (comentário de `EnqueueAction`). O mestre pedindo
  a ação de um **jogador** cai em `ErrParticipantNotFound` por esse ramo — é o mesmo que o
  `EnqueueAction` responderia. Aceitável e coerente; documente no contrato que o mestre recebe
  `participant not found in match session` nesse caso.

- [ ] **Step 4: rode e veja passar.**

- [ ] **Step 5: use case** — `internal/application/match/cancel_action.go`, no molde exato de `enqueue_action.go`:

```go
type ICancelAction interface {
	Execute(ctx context.Context, session *matchsession.MatchSession, playerUUID, actionID uuid.UUID) error
}

type CancelActionUC struct{}

func NewCancelActionUC() *CancelActionUC { return &CancelActionUC{} }

func (uc *CancelActionUC) Execute(
	ctx context.Context,
	session *matchsession.MatchSession,
	playerUUID, actionID uuid.UUID,
) error {
	return session.CancelAction(playerUUID, actionID)
}
```

- [ ] **Step 6: mensagens** — em `message.go`, junto das constantes cliente→servidor e servidor→cliente vizinhas (`MsgTypePullAction`, `MsgTypeActionQueued`): `MsgTypeCancelAction MessageType = "cancel_action"`, `MsgTypeActionCancelled MessageType = "action_cancelled"`; e, perto de `PullActionPayload`/`ActionEnqueuedPayload`:

```go
// CancelActionPayload withdraws one of the sender's own queued actions (spec 2026-10-10).
type CancelActionPayload struct {
	ActionID uuid.UUID `json:"actionId"`
}

// ActionCancelledPayload names the action that left the queue. Master-only plus the sender:
// the queue is secret, the table only sees the public order change in bars_updated.
type ActionCancelledPayload struct {
	ActionID uuid.UUID `json:"actionId"`
}
```

  Se `message.go` tiver uma lista/registro de tipos válidos de mensagem de entrada (procure onde
  `MsgTypePullAction` aparece além da constante — ex.: validação de `unknown_type`), inclua
  `MsgTypeCancelAction` lá também.

- [ ] **Step 7: `RoomDeps` e `main.go`** — `CancelActionUC appmatch.ICancelAction` logo depois de `EnqueueActionUC` em `room_deps.go` (use o mesmo alias de import que o arquivo já usa para o pacote `internal/application/match`); `cmd/game/main.go` instancia `appmatch.NewCancelActionUC()` e preenche o campo, no padrão de `enqueueActionUC`.

- [ ] **Step 8: teste e2e que falha** — `internal/app/game/cancel_action_e2e_test.go`. Monte o cenário reusando o `combatFixture` e os helpers de `own_queue_e2e_test.go`/`queue_e2e_test.go` (`enqueueMoveAttackFrom`, `actionIDsInOrder`, `collector`, `sendWS`) — leia esses dois arquivos e copie como eles montam mestre + dois jogadores e como esperam mensagem (não invente helper de espera novo se já existir um). O fixture precisa passar `CancelActionUC: appmatch.NewCancelActionUC()` nos `RoomDeps` — veja onde o fixture monta `RoomDeps` e acrescente o campo lá (é um helper de teste; a mudança vale para todos os e2e e é inofensiva). Testes:
  1. `TestE2E_PlayerCancelsOwnQueuedAction`: jogador A declara; pega o id do `action_enqueued`; manda `cancel_action`; o mestre recebe `action_cancelled{actionId}`; A recebe `action_cancelled{actionId}`; o jogador B **não** recebe `action_cancelled` (espere um `bars_updated` em B depois do cancelamento e então confira que nenhuma mensagem `action_cancelled` chegou a B); o último `bars_updated` não tem entrada do ator de A em `order`; o mestre manda `open_next_action` e recebe `error` `game_error` com `action queue is empty` (ou, se o fixture estiver em Race, `round_closed` — leia o regime do fixture e escreva a asserção certa) — em nenhum dos dois casos chega `turn_opened`.
  2. `TestE2E_CancelledActionIsGoneAfterReconnect`: A declara e cancela; o mestre reconecta e o `match_full_state.queue` não tem o id; A reconecta e `ownQueue` é `[]`.
  3. `TestE2E_CannotCancelSomeoneElsesAction`: A declara; B manda `cancel_action` com o id de A → B recebe `error` `game_error` `action actor does not match player`; o mestre **não** recebe `action_cancelled`; o mestre reconecta e a `queue` ainda tem a ação.
  4. `TestE2E_CancelActionRejectsBadPayload`: `cancel_action` com `{"actionId": 123}` → `invalid_payload` (`invalid cancel_action payload`).

- [ ] **Step 9: rode e veja falhar** — `go test ./internal/app/game/ -run 'Cancel'` → `unknown_type` ou timeout.

- [ ] **Step 10: o braço em `room.go`**, logo depois do `case MsgTypeEnqueueAction` (antes de `MsgTypeAttachReaction`):

```go
	case MsgTypeCancelAction:
		var payload CancelActionPayload
		if err := json.Unmarshal(incoming.Payload, &payload); err != nil || payload.ActionID == uuid.Nil {
			client.SendMessage(NewErrorMessage("invalid_payload", "invalid cancel_action payload"))
			return
		}
		// Write lock across Execute: cancelling takes the action out of the same queue
		// open_next_action and pull_action read. Nothing is sent under the lock.
		r.mu.Lock()
		session := r.session
		var err error
		if session != nil && r.deps.CancelActionUC != nil {
			err = r.deps.CancelActionUC.Execute(context.Background(), session, client.userUUID, payload.ActionID)
		}
		r.mu.Unlock()
		if session == nil || r.deps.CancelActionUC == nil {
			client.SendMessage(NewErrorMessage("match_not_started", "match session not initialized"))
			return
		}
		if err != nil {
			client.SendMessage(NewErrorMessage("game_error", err.Error()))
			return
		}
		// The master decides what opens next, so they hear it first; the sender gets the same
		// news as the ack of their own request. A master cancelling an NPC's action is both —
		// one message, not two. The table only sees the public order change.
		cancelled := NewServerMessage(MsgTypeActionCancelled, ActionCancelledPayload{ActionID: payload.ActionID})
		r.sendToMaster(cancelled)
		if !r.IsMaster(client.userUUID) {
			client.SendMessage(cancelled)
		}
		r.broadcastBars(session)
```

  (Ajuste ao estilo real dos vizinhos: se `IsMaster` lê `r.masterUUID` sob lock, ele é seguro
  aqui fora do lock — confira.)

- [ ] **Step 11: rode e veja passar** — `go test ./internal/app/game/ -run 'Cancel'` e depois `go test -race ./internal/app/game/`.

- [ ] **Step 12: contrato** (`docs/dev/api/match-combat-ws.md`):
  - Índice §3: as duas linhas novas, no formato das vizinhas (direção, quem).
  - §4, depois de `pull_action`: seção `### \`cancel_action\`` com o texto da spec §2 (payload, quem, de onde vem o `actionId`, "Dispara, nesta ordem", tabela de erros — incluindo o caso do mestre pedindo a ação de um jogador → `participant not found in match session`, nota "nada é gravado; não aparece no histórico; nada é devolvido nas barras; um preço congelado fica congelado; não fecha a rodada").
  - §5, depois de `action_queued`: seção `### \`action_cancelled\`` (destino, payload, por que não é broadcast).
  - `action_enqueued`: a frase "não cancelava" ganha um link para [`cancel_action`](#cancel_action).
  - §7: em `invalid_payload`/`game_error` nada muda de texto (as linhas já dizem "Todas"); só confira.

- [ ] **Step 13: verificação completa** (Global Constraints) e commit:

```bash
git add internal/domain/match/matchsession/match_session.go internal/domain/match/matchsession/match_session_test.go \
  internal/application/match/cancel_action.go internal/app/game/message.go internal/app/game/room_deps.go \
  internal/app/game/room.go cmd/game/main.go internal/app/game/cancel_action_e2e_test.go \
  docs/dev/api/match-combat-ws.md <o arquivo do fixture, se mudou>
git commit -m "feat(combat): cancel_action — quem declarou tira a própria ação da fila"
```

### Task 2: documentação

**Files:**
- Modify: `docs/superpowers/specs/2026-09-20-front-combat-phases.md` (§4.3 e a linha "Fora de escopo" da §6 que cita "cancelar ação")
- Modify: `AGENTS.md` (Known Issues)
- Modify: `docs/documentation-map.yaml`
- Modify: `docs/dev/match/combat-engine.md` (só se ele descrever a fila e as operações sobre ela — procure "fila"/"queue"; se descrever, uma linha sobre `CancelAction`)

- [ ] **Step 1:** documento mestre §4.3: acrescente ao fim um parágrafo "**Feito em 2026-10-10** (spec `2026-10-10-combat-cancel-action-back-design.md`): `cancel_action` / `action_cancelled`; cada um cancela o que declarou; só na fila." Na §6 "Fora de escopo", acrescente após "**cancelar ação**" um "(feito depois, em 2026-10-10 — §4.3)". Não reescreva o resto.
- [ ] **Step 2:** `AGENTS.md`, em Known Issues, uma linha no estilo das vizinhas: "**Cancelar ação existe** (spec 2026-10-10): `cancel_action` tira da fila uma ação ainda não aberta; quem declarou cancela (o NPC é do mestre); nada é gravado nem devolvido. Ver `match-combat-ws.md`."
- [ ] **Step 3:** `documentation-map.yaml`: veja como `internal/application/match/` (linha ~984) e os arquivos de `internal/app/game/` estão mapeados; se o diretório já cobre o arquivo novo, nada a fazer; se a entrada de `message.go`/`room.go` lista os verbos por nome (linha ~828), acrescente `cancel_action`/`action_cancelled`.
- [ ] **Step 4:** commit `docs: cancelar ação — documento mestre, AGENTS e mapa`.
