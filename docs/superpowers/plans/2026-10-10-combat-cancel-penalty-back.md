# Cancelar ação — Desvantagem e registro — pacote de back — plano

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** cancelar uma ação marca o personagem até o fim da rodada (Race) — o que ele fizer de novo
rola a velocidade com Desvantagem —, e a ação cancelada fica gravada e aparece no histórico.

**Architecture:** marca por rodada na `MatchSession` lida por `EnqueueAction`/`AttachReaction`;
tabela própria `cancelled_actions` gravada com a regra de durabilidade das master actions; o
histórico a costura como evento de rodada, projetado por leitor.

**Tech Stack:** Go 1.23, pgx/v5, goose, gorilla/websocket, humatest, `testing` padrão.

**Spec:** [`docs/superpowers/specs/2026-10-10-combat-cancel-action-back-design.md`](../specs/2026-10-10-combat-cancel-action-back-design.md)
**§8 (adendo)** — leia o spec inteiro; §1–§7 descrevem o verbo que já existe nesta branch.

**Branch:** `feat/combat-cancel-action` (PR #86), HEAD `7b86008`.

## Global Constraints

- Go 1.23; `testing` padrão, table-driven; **TDD**. Pacotes de teste: os que cada arquivo já usa.
- `room.go` é dono do lock (`r.mu`). **Nada que envia a cliente ou escreve no banco roda com `r.mu` preso.** Siga `recordMasterAction` (`internal/app/game/master_action_record.go`) como molde de gravação.
- **Tags `json:` de `internal/domain/**` são formato de ARMAZENAMENTO** (vão para JSONB). Não as troque; o wire é `internal/app/**`.
- Wire em camelCase. Nunca remover `TODO`. Comente o porquê na densidade dos vizinhos.
- Migração nova: `migrations/2026101000000x_cancelled_actions.sql` (goose, `BEGIN/COMMIT` como as vizinhas), registrada em `docs/documentation-map.yaml` no mesmo commit; aplique no banco de teste antes dos testes de integração: `goose -dir migrations postgres "$TEST_DATABASE_URL" up` (padrão `postgres://postgres:postgres@localhost:5432/hxh_rpg_test?sslmode=disable`).
- Todo item que muda o wire atualiza o contrato (`docs/dev/api/match-combat-ws.md`, `match-history.md`) no mesmo commit.
- Verificação por tarefa: `go build ./...`, `go vet ./...`, `go vet -tags integration ./...`, `go vet -tags smoke ./...`, `go test ./internal/...`, `go test -race ./internal/app/game/`; tarefa que toca `internal/gateway/pg/`: `go test -tags=integration -p 1 ./internal/gateway/pg/...`. Antes do fim: `golangci-lint run ./...`.
- `git add` caminho por caminho. Commits terminam com `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

1. Marca de uma rodada vazando para a seguinte (P2) — o id da rodada é a chave.
2. Free nunca marca (P3); trocar o regime para Race no meio da rodada não ressuscita marca.
3. Reação livre (esquiva, fechada, nada) nunca ganha viés, mesmo marcado (P4).
4. Gravação com turno aberto: nada no banco antes do fechamento; depois, na mesma transação do turno.
5. Histórico: o "outro" jogador nunca recebe `action`.

---

### Task 1: a Desvantagem de quem cancelou (spec §8.1 P1–P5, §8.2 "A marca"/"Quem lê", §8.3)

**Files:** `internal/domain/match/matchsession/match_session.go` (+ teste), `internal/application/match/cancel_action.go` (+ o `ICancelAction`), `internal/app/game/room.go` (o braço `MsgTypeCancelAction` só se adapta à assinatura nova), `docs/dev/api/match-combat-ws.md` (`cancel_action`), `docs/dev/match/combat-engine.md` (tabela de custos), `docs/game/combate/reacoes.md` (uma frase, PT-BR de livro de regras, sem código).

- [ ] **Testes que falham** (domínio, `match_session_test.go`, rodada em Race — veja como os testes vizinhos montam uma sessão em Race e leem `Speed.Result`/`SystemBias`): os casos de §8.4 "Domínio". Para "velocidade com Desvantagem", afirme `a.SystemBias == -1` (é o campo que `deriveSpeeds` grava) e, onde o fixture tiver dados controlados, que o resultado lido é o pior dos dois conjuntos.
- [ ] **Implemente:** `swapMarks map[uuid.UUID]uuid.UUID` (inicialize onde os outros mapas da sessão são inicializados), `swapBias(charID) int`; `CancelAction` passa a devolver `(*action.Action, error)` e grava a marca se `s.activeRound.GetMode() == enum.Race`; `EnqueueAction` → `deriveSpeeds(a, s.swapBias(a.GetActorID()))`; `AttachReaction` → `systemBias = -1` se consumiu **ou** `s.swapBias(...) < 0`. `CancelActionUC.Execute` devolve a ação; o braço da `Room` ignora o valor por ora (Task 2 o usa).
- [ ] **Docs:** contrato `cancel_action` (Desvantagem até o fim da rodada em Race; o que não piora), `combat-engine.md`, `reacoes.md`.
- [ ] Verificação, commit `feat(combat): cancelar marca a Desvantagem de troca até o fim da rodada`.

### Task 2: gravar a ação cancelada (spec §8.1 P6–P7, §8.2 "O registro"/"Tabela"/"Gravação")

**Files:** migração nova; pacote do registro (ao lado de onde `masteraction` mora — `grep -rn "package masteraction" internal`); `internal/gateway/pg/cancelledaction/` (repository + integration test); `internal/gateway/pg/round/persist_turn_close.go` (+ integration test) e `TurnCloseData`; `internal/app/game/turn_writes.go`, um `cancelled_action_record.go` no molde de `master_action_record.go`, `room.go` (o braço chama o registro), `room_deps.go`, `cmd/game/main.go`; e2e em `internal/app/game/cancel_action_e2e_test.go`.

- [ ] **Leia antes:** `master_action_record.go` inteiro (o molde: snapshot sob lock, turno aberto → `turnWritesLocked`, senão `ensureSceneAndRoundRows` + insert fora do lock), `turn_writes.go`, e em `persist_turn_close.go` como `insertAction` monta as colunas JSON da ação e como `MasterActions` entram na transação.
- [ ] **Formato de `action` (JSONB):** extraia de `insertAction` a montagem das peças JSON da ação (speed, skills, move, attack, defense, dodge, repel, feint, trigger, interact, target_ids, system_bias, …) para um helper reutilizável e grave, em `cancelled_actions.action`, um objeto JSON com essas mesmas peças (chaves estáveis, documentadas no comentário da migração). Escreva também o inverso (`action.Action` a partir desse JSON) reaproveitando o que a leitura de `actions` do histórico já faz para reconstruir uma ação a partir das colunas — procure onde o histórico lê `actions` (`internal/gateway/pg/match` ou `round`) e reuse o decodificador; **não** duplique a lógica coluna a coluna.
- [ ] **Testes de integração que falham:** insert + leitura por partida; `PersistTurnClose` com `CancelledActions` grava na transação (e um rollback da transação não deixa a linha).
- [ ] **E2E que falham:** cancelar sem turno aberto → a linha existe logo depois (use o repo real? os e2e da `Room` usam fakes de repo — siga o padrão de `master_board_e2e_test.go`/`history_e2e_test.go` para afirmar o que a `Room` mandou gravar); cancelar com turno aberto → nada gravado até o `close_turn`, depois gravado com `TurnUUID`.
- [ ] Implemente, verificação (incl. integração com `-p 1`), commit `feat(combat): a ação cancelada é gravada em cancelled_actions`.

### Task 3: a cancelada no histórico (spec §8.1 P8, §8.2 "Leitura", §8.3)

**Files:** `internal/application/match/i_repository.go` (lister), `get_match_history.go` (+ testes de stitch/projeção), `internal/app/api/match/get_match_history.go` (+ teste com humatest), o gateway da Task 2 (método de listagem), `cmd/api` (injeção), `docs/dev/api/match-history.md`.

- [ ] **Testes que falham:** use case — um registro vira `rounds[].events` `kind actionCancelled` na posição de tempo certa; mestre recebe a ação (Full), dono recebe a ação (Declaration: sem números de velocidade), outro jogador recebe só `actorId`. Handler (humatest): o JSON tem `kind: "actionCancelled"`, `actorId`, e `action` só para quem deve.
- [ ] **Implemente** seguindo como `masterAction` atravessa as três camadas (`HistoryEventMasterAction` → mapeamento REST). Quem é "dono": o usuário dono da ficha `ActorUUID` (o histórico já resolve dono de ficha para `moveSightFor` — reaproveite).
- [ ] Contrato `match-history.md` (formato de `events[]`, visibilidade), verificação, commit `feat(history): a ação cancelada aparece no histórico, projetada por leitor`.
