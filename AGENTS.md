# System X System — Agent Guide

## Project Overview

Go 1.23 backend for a Hunter × Hunter tabletop RPG. Module: `github.com/422UR4H/HxH_RPG_System`. PostgreSQL (goose migrations). Entry points: REST API (`cmd/api/`) and WebSocket game server (`cmd/game/`).

## Architecture

```
internal/
├── app/         ← Delivery: HTTP handlers (api/) + WebSocket server (game/)
├── application/ ← Use Cases: orchestrate domain + I/O (one package per feature)
├── domain/      ← Domain: entities + domain services (pure, no I/O)
│   ├── match/   ← Match bounded context (entity/, service/, matchsession/)
│   └── entity/  ← Shared entities (character_sheet/, enum/, die/, ...)
├── gateway/     ← Infrastructure: PostgreSQL repositories
└── config/      ← Configuration loading
```

Dependency: entity ← domain ← app, entity ← gateway. Entities never import outer layers.

## Code Conventions

- **Comments and TODOs are the owner's markers.** Never delete one outside the scope of your task —
  sessions used to wipe other people's TODOs. What your change makes contradictory (a comment that
  now describes the code wrongly) must be corrected or removed in the same change, saying why in
  the commit.
- Go idiomatic: implicit interfaces, short var names, error wrapping `%w`
- **User vs Player vs Master:** `User` = generic auth entity. Use `Player`/`Master` for role-specific contexts.
- **Domain Services:** stateless structs in `domain/match/service/` — receive entities, apply RPG rules, return results. No I/O, no state.
- **Use Cases:** in `application/<feature>/` — orchestrate domain + gateway. No RPG rules.
- XP cascade: skill → attribute → ability → character (`CascadeUpgrade`/`CascadeUpgradeTrigger`)
- DDD-lite: value objects, entities, domain services, use cases, repository interfaces

## Testing

- Standard `testing` only, no frameworks. Table-driven with `t.Run()`.
- External test packages: `package foo_test`
- Mocks: `mocks_test.go` per handler package (Go idiomatic)
- Create documentation alongside tests during all development work
- **Every feature must have integration tests** (not just unit tests)
- TDD strategy per layer: see `integration-tests.instructions.md` (loaded for `internal/**`)
- `go vet ./...`/`go test ./...` don't see build-tag-gated files (`smoke`, `integration`, 12 files);
  broad wire-format/struct-tag migrations should also run `go vet -tags smoke ./...` and
  `go vet -tags integration ./...` (or the test equivalent) to catch drift there.

## Git Workflow

- **Always PRs** — never merge directly to `main`
- Branch: `feat/`, `fix/`, `docs/`, `refactor/`
- Specs EN + PT-BR versions in same commit

## Agent Model Strategy

Pick the cheapest model that will get the task right.

| Role | Model |
|------|-------|
| Orchestration/Planning | Opus (main) |
| Heavy / multi-file integration (e.g. rewiring a large delivery file) | Opus |
| Code with logic, branching, integration, or "match the real code shapes" judgment | Sonnet 4.6 |
| Well-specified boilerplate (data structs, enums, WS message types/payloads, repo stubs, mechanical edits where the plan gives complete code) | Haiku 4.5 |
| Code Review / Critique | Sonnet 4.6+ |
| Exploration / read-only / commands | Haiku 4.5 |

Haiku is preferred for well-specified boilerplate — it is fast, cheap, and reliable when
the spec is complete. Escalate to Sonnet the moment a task needs branching logic, multi-file
integration, correctness reasoning, or adapting to real code shapes. Use Opus for planning
and heavy integration. When genuinely unsure between Haiku and Sonnet, prefer Sonnet.

## Commands

**Prefer CI over local runs** (saves tokens). Local only for TDD iteration or debugging.

```bash
# CI (default):
gh run list --workflow=ci.yml --limit=1   # check status
gh run view <run-id> --log-failed         # failure logs

# Local (when needed):
go test ./...                                         # all tests
go test -tags=integration ./internal/gateway/pg/...   # integration tests
make build / make dev-api / make dev-game   # dev-* usa air (hot reload); run-dev sem hot reload
make migrate-up / migrate-down / migrate-create name=X
go run ./cmd/devseed   # mesa de teste local: mestre + 2 jogadores com senha conhecida (docs/dev/seed.md)
```

## Scoped Instructions

Context-specific content lives in `.github/instructions/` (loaded only when relevant):
- `domain-map.instructions.md` — entity paths and current state (when working on `internal/`)
- `docs-workflow.instructions.md` — documentation maintenance rules (when working on `docs/`)
- `glossary.instructions.md` — EN↔PT-BR terminology (when working on `docs/game/`)
- `integration-tests.instructions.md` — test patterns, helpers, DB setup (when working on `internal/gateway/pg/`)
- `gateway-conventions.instructions.md` — SQL/repository patterns (when working on `internal/gateway/`)
- `game-server.instructions.md` — MatchSession, Room/Client/Hub pattern, message routing (when working on `internal/app/game/`)

## Sheet assembly is mirrored in the front

The front (HxH_RPG_Environment_React, `src/features/sheet/engine/`) re-runs the sheet
assembly in TypeScript for the live preview on create/edit. The back stays the source of
truth. Any change to formulas, coefficients, cascades, classes or `AssembleCharacterSheet`
makes `TestSheetParityFixtures` fail until you regenerate
(`go test ./internal/app/api/sheet/ -run TestSheetParityFixtures -update`); then sync the
front (`npm run sync:sheet-fixtures`) and fix its engine if its parity test fails. See
`docs/dev/character-sheet/factory.md` ("Paridade com o front").

## Known Issues

**Motor de batalha — Fases 1 e 2 implementadas.** Ver
`docs/superpowers/specs/2026-08-16-combat-engine-design.md` e `docs/dev/match/combat-engine.md`.
Uma coisa parece lacuna e é **deliberada**:

- `resolution_updated` é **master-only**. O cálculo é do mestre até o turno encerrar;
  difusão para a mesa e projeção por destinatário são da Fase 5.

**HP ao vivo existe** desde a dívida P2 (documento mestre do front, §4.10): o fechamento de turno — pelos três verbos — emite
`character_hp_changed` projetado para o mestre e para o dono da ficha, com o HP aplicado, o
máximo da barra e o dano. Cura e veneno, quando existirem, emitem a MESMA mensagem
(`broadcastHpChanges` em `room.go`). Ver `docs/dev/api/match-combat-ws.md` §5.

**Cancelar ação existe** (spec 2026-10-10): `cancel_action` tira da fila uma ação ainda não aberta; quem declarou cancela (o NPC é do mestre); nada é devolvido, e a cancelada é gravada em `cancelled_actions` pela regra das master actions (com turno aberto, no fechamento dele). Ver `docs/dev/api/match-combat-ws.md`.

**Deferred to Phase 4 (reações):**
- Reaction visibility — **feito** (Fase 7, pacote de back): a reação vai à mesa projetada em `reaction_opened` e na reconexão; o cálculo continua do mestre até o fechamento.
- Initiative handling in `ChangeMode`
- `Turn.createdAt` — **feito** (status-effects PR 2, Task 15b): a sessão carimba a abertura real do turno (`Turn.GetOpenedAt`, relógio da sala, µs) e de cada reação (`actions.opened_at`, NULL = nunca aberta); `turns.created_at` é a abertura. Linhas gravadas antes ficam com o `finished_at` como `created_at`.
- No Attack mapping in `buildMasterAction` — decided, not pending (spec 2026-09-27 §2, B9): the master attacks through an NPC with `enqueue_action`; `enqueue_master_action` refuses an `attack` key outright. A master attack as an environment effect (a trap) is future work. Move maps only `position` — it is the master's drag (spec 2026-09-27 §4.3)

**Fechamento da Fase 6 — pacote de back (B1–B16, spec 2026-09-27):**
- **O tabuleiro é do servidor.** `match_boards` guarda um retrato por partida (peças, paredes,
  grade, `bg` — `NULL` = sem fundo, congelado como a grade); `player_memories` guarda o fog explorado. A `Room`
  carrega (`loadBoard`) quando nasce e, em lobby, a cada conexão do mestre; qualquer conexão (jogador também) tenta de novo enquanto a sala não tem tabuleiro agora e a última carga falhou, e quem ainda não encontra recebe `board_unavailable` (o jogador só repassa o mapa aos demais se der certo, e nunca o `match_full_state`). Grava em dois
  pontos, os dois serializados por `persistMu`: o fechamento de turno (`persistClosedTurn`, na
  transação do `PersistTurnClose` — ver abaixo) e `persistBoard`, no `start_match` e — entre
  turnos — nas master actions de peça e na interação/revelação de parede. **Dentro de um
  turno aberto nada é gravado antes do fechamento** (dono do produto, 2026-10-01): nem o
  tabuleiro (`persistBoardOutsideTurn`) nem as master actions nem o que cada jogador viu do
  movimento aberto, que a `Room` guarda por turno (`turnWrites`, `turn_writes.go`). O
  fechamento grava turno, master actions, vereditos de movimento/pouso (o `GET /history` mostra
  o `move` como cada um o viu ao vivo) e tabuleiro+fog numa transação só (`PersistTurnClose`;
  os três verbos não chamam `persistBoard`). Um reinício no meio do turno o perde inteiro, de propósito.
  **Um comando do mestre, uma transação** (dono do produto, 2026-10-02): o HP que o fechamento
  aplica também vai nela (`TurnCloseData.StatusBars`, copiado sob `r.mu` em `persistClosedTurn`),
  e, quando o mesmo `open_next_action` acaba o round, o fim dele e o round seguinte
  (`TurnCloseData.NextRound`; sem turno fechado, `PersistRoundClose`, uma transação só deles).
  Os casos de uso de fechamento e o `CloseRoundUC` não fazem I/O nenhum. Uma gravação que falha
  se cura na próxima que dá certo (`internal/app/game/unwritten.go`): fichas e fins de round não
  gravados vão na transação seguinte. Fora disso:
  `change_round_mode` (do round) e inscrição de NPC.
  `map_state_sync` deixou de escrever: é aceito, ignorado, e responde só ao remetente.
  `RoomDeps` (`internal/app/game/room_deps.go`) concentra toda dependência externa da `Room`.
- **Escape = esquiva E movimento.** `Avoided = dodgePassed && movePassed` (spec B13):
  `movePassed` compara `Move.FinalSpeed` ao acerto do atacante; `dodgePassed`, o `Dodge.Total`.
  Falhando qualquer um dos dois, o golpe é lido como se o alvo tivesse ficado. Nenhum escape
  desloca na abertura da reação — só no fechamento, por `applyClosedEscapes`; se falhou, o
  mestre escolhe onde a peça cai pelo `edit_action.escapeLanding`, guardado no `Turn`. Ver
  `docs/dev/match/combat-engine.md` ("O escape: esquiva e movimento").
- **Master actions têm tabela própria (`master_actions`), não `actions`.** O ator de uma master
  action é o mestre (usuário), não um `character_sheets`, e ela acontece fora de turno — as duas
  colunas de `actions` (`actor_uuid` → ficha, `turn_uuid NOT NULL`) teriam que afrouxar e toda
  leitura teria que filtrar um tipo do outro. Gravada no instante em que é aplicada (com turno
  aberto, no fechamento dele, na transação do turno), com a projeção (`views`) do que cada
  jogador viu dela ao vivo. Decisão do dono do produto, spec §4.8.
- **A condição do mestre é lida em toda rolagem que a resolução usa** (Fase 8, spec
  2026-10-09): acerto, dano (só o ajuste; viés recusado), esquiva, Evasion das fechadas, aparo,
  defesa padrão (portador `Action.DefaultDefense`, fora do wire) e o `moveSpeed` das fugas.
  `speed`, o `moveSpeed` da própria ação e `feint` são aceitos e não mudam o desfecho do turno
  aberto. Condição zerada é "sem condição". Ver `docs/dev/match/combat-engine.md` ("A edição do
  mestre").
- **A reação padrão é uma reação** (dono do produto, 2026-10-12). Ao abrir o turno, a sessão
  senta uma `ReactDefault` (esquiva por reflexo + defesa padrão, passivas) para cada personagem
  que o ataque nomeia (`seatDefaultReactions`); o mestre a edita e pode abri-la, e o que o alvo
  manda a substitui (`Turn.ReplaceSeated`; declarar `default` herda dados e edições). Não segura
  o `close_turn` nem aparece em `ownReactions`. **Reação não aberta entra no cálculo**, depois
  das abertas (`buildChainOrder`); `nothing` nunca precisa ser aberto nem segura o `close_turn`,
  mas o mestre pode abri-lo para posicioná-lo na cadeia. **Viés do mestre num teste passivo o
  torna rolado**, e o da troca do sistema também (P2: vantagem = melhor entre o dado e o valor
  base, desvantagem = pior; o dado cai na primeira leitura com viés — `deriveCheck` nas
  velocidades, `TurnResolution.DiceShort` + `ResolveTurn` nos testes da resolução; o valor base
  vai em `match_full_state.passiveValue`). **Repelir quem o ataque da fila mirava não é troca**
  (`keepsQueuedIntent`: `sameBlowAsQueued`, ao lado do `sameMoveAsQueued` das fugas). **O
  mestre edita ação na fila** (só condições; `targetIds`/`skills` recusados — ver o
  comentário em `ApplyMasterAction`); `speed` editada reordena a fila, e cancelar/consumir
  descarta as edições (`discardEdits`). Spec
  `docs/superpowers/specs/2026-10-12-default-reaction-and-queue-edits-design.md`.

**Relógio da história existe** (spec 2026-11-08): `clock_changed`/`set_clock`/`end_match` etc. Coisas que parecem lacuna e são deliberadas: tempo da história é **ingênuo** (string sem fuso; o front nunca usa `Date`); `clock_changed.kind: "end"` existe na struct mas o servidor não o emite (o fim vai em `match_ended` + fechamento 4002); `pause_clock` com o relógio já parado não responde nada; `get_combat_summary` não exige hora definida. Partida encerrada = `gameEndAt` (`Match.IsFinished()`), não `storyEndAt`. Ver `docs/dev/api/match-clock.md` e `docs/dev/match/clock.md`.

**Efeitos de status — fases 1 e 2** (spec 2026-11-15; planos 2026-11-17 e 2026-11-24). O mestre aplica/edita/remove (`apply_status_effect` etc.), `status_effects_updated` vai por destinatário e a sessão os guarda (a sala decide quando gravar: fora de turno na hora, dentro com o fechamento, falha cura na próxima gravação ou no `end_match`). Coisas que parecem lacuna e são deliberadas: **a fase 2 faz os efeitos pesarem nas rolagens** (`statuseffect.RollEffectOf`: vantagem/desvantagem, penalidade, falha automática, G1, Caído/Guarda aberta; um nível de viés = um dado extra, `RollAttempts{Primary, Extra}`; o crítico mora em `match.MatchRules`; `Derive`/`Resolve` nunca rolam — a sessão completa só os dados que faltam, uma vez, e todo comando que muda um viés resolve sob a trava de escrita) e traz a **morte** (HP abaixo do mínimo da barra → `dead`, aplicado pelo sistema no fechamento do turno, só desfeito pelo mestre com `remove_status_effect.hp`; `dead_at` só no fim da partida); **continuam fora**: teste automático (resistência, equilíbrio, Recuperação), dano/recuperação por round, comando de editar HP, asfixia/Fadiga, "não usa habilidades" (Nen); a desvantagem de escape do Caído/Guarda aberta é **provisória** até o redesenho do movimento; `skills[].rollCheck.result` é derivado mas nada o lê para desfecho; o jogador **fica com o último estado visto** quando o personagem sai da visão (o servidor não manda "deixou de ver"); `npc_added` e a lista de NPCs vão **só ao mestre** e as barras são **por jogador** (névoa + Escondido); a reação padrão sentada segue os status (`nothing` quando não pode reagir) e seus sinalizadores de passiva não são persistidos; uma ficha viva em duas salas guarda duas memórias (última gravação ganha). **Histórico de status e Escondido (PR 2)** — o que fica de fora de propósito: a revelação do mestre (`POST …/status-effects/{id}/reveal`) é **para todos** (por jogador é futuro) e **não tem push de WS** (o front relê `/history`); um efeito aplicado antes do PR 2 não tem evento e não pode ser revelado (404); desfazer no turno só devolve as ações da fila que o status cancelou e as reações não abertas que ele trocou com `restoreCancelledActions` (revalidadas; ações no mesmo lugar da fila e com as edições do mestre, em `action_restored`, nunca em `action_enqueued`; reações com o mesmo id e lugar, em `reaction_restored`, nunca em `reaction_attached`; recusa vira `restore_refused` ao mestre), e depois do fechamento não há volta; a âncora viva (`turnActionId`) é só do mestre; um jogador que só estava de fora por um Escondido desfeito no turno não vê aquele evento (falha fechada); um evento fora de turno num round que nunca virou linha (reinício antes de garantir a linha) sai do histórico com log; a revelação é carimbada pelo relógio da API e os eventos pelo do jogo (uns ms de diferença podem deixar descoberto um evento logo depois do clique); o registro do Escondido (`actions.hidden_from`) é lista de permissão — quem não foi nomeado no instante, inclusive quem não era da sessão, fica de fora até a revelação. Cadeia de trava: `persistMu` → `r.mu`; só `end_match` segura `clockPersistMu` → `persistMu` → `r.mu`. Ver `docs/dev/api/match-status-effects.md` e `docs/dev/match/status-effects.md`.

**Envios pendentes travam quando a sala para (não corrigido):** os `go func(){ r.broadcast <- … }()` (e `broadcastInOrder`) ainda sem leitor quando o `Run` sai ficam bloqueados para sempre — padrão anterior ao relógio, mais frequente com ele (`clock_changed` em quase todo comando do mestre).

**Pendente de configurações de campanha/partida:**
- `fog_mode` (`live` | `explored`) é persistido em `maps.fog_mode` e honrado por
  `FilterMapState`, mas nenhum endpoint REST o expõe e `room.go` hardcoda `explored`.
  Será uma configuração de **partida**, escolhida pelo mestre — o mecanismo de
  configurações ainda não existe no backend. **Não remover** `FogMode`: isso eliminaria
  o modo `live`. Ver spec do refactor do mapa (repo do front), §3.
