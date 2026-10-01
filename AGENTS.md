# HxH RPG System — Agent Guide

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

- **NEVER remove TODO comments** — intentional markers by the owner
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
rtk gh run list --workflow=ci.yml --limit=1   # check status
rtk gh run view <run-id> --log-failed         # failure logs

# Local (when needed):
go test ./...                                         # all tests
go test -tags=integration ./internal/gateway/pg/...   # integration tests
make build / make dev-api / make dev-game   # dev-* usa air (hot reload); run-dev sem hot reload
make migrate-up / migrate-down / migrate-create name=X
```

## Scoped Instructions

Context-specific content lives in `.github/instructions/` (loaded only when relevant):
- `domain-map.instructions.md` — entity paths and current state (when working on `internal/`)
- `docs-workflow.instructions.md` — documentation maintenance rules (when working on `docs/`)
- `glossary.instructions.md` — EN↔PT-BR terminology (when working on `docs/game/`)
- `integration-tests.instructions.md` — test patterns, helpers, DB setup (when working on `internal/gateway/pg/`)
- `gateway-conventions.instructions.md` — SQL/repository patterns (when working on `internal/gateway/`)
- `game-server.instructions.md` — MatchSession, Room/Client/Hub pattern, message routing (when working on `internal/app/game/`)

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

**Deferred to Phase 4 (reações):**
- Reaction visibility: players see reactions only when master reveals (currently master-only)
- Initiative handling in `ChangeMode`
- `Turn.createdAt` field (turns currently use `finishedAt` as approximation for `created_at` in DB)
- No Attack mapping in `buildMasterAction` — decided, not pending (spec 2026-09-27 §2, B9): the master attacks through an NPC with `enqueue_action`; `enqueue_master_action` refuses an `attack` key outright. A master attack as an environment effect (a trap) is future work. Move maps only `position` — it is the master's drag (spec 2026-09-27 §4.3)

**Fechamento da Fase 6 — pacote de back (B1–B16, spec 2026-09-27):**
- **O tabuleiro é do servidor.** `match_boards` guarda um retrato por partida (peças, paredes,
  grade, `bg` — `NULL` herda o do mapa); `player_memories` guarda o fog explorado. A `Room`
  carrega (`loadBoard`) quando nasce e, em lobby, a cada conexão do mestre; `persistBoard` é o
  único ponto de gravação, nos três verbos de fechamento de turno, no `start_match`, nas
  master actions de peça e na interação/revelação de parede, serializado por `persistMu`.
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
  leitura teria que filtrar um tipo do outro. Gravada no instante em que é aplicada, com a
  projeção (`views`) do que cada jogador viu dela ao vivo. Decisão do dono do produto, spec §4.8.
- ⚠️ **Bug conhecido, achado na Task 12 (não corrigido nesta fase):** uma condição do mestre
  (`edit_action` com `conditions[].field` = `"dodge"`, `"defense"` ou `"repel"`) é aceita,
  grava o override, e **não muda o resultado** — `deriveReflex`/`resolveRepel`
  (`internal/domain/match/service/reaction_collision.go`) montam o `RollInput` sem ler
  `Dodge.Context.Condition`/`Repel.Context.Condition`, e a defesa automática é sempre passiva
  sem ler `Condition` nenhuma. Só `hit` (e `speed`/`moveSpeed`, via `deriveSpeeds` na sessão)
  chega ao resolvedor. É anterior a esta fase; documentado em `combat-engine.md` ("O escape:
  esquiva e movimento").

**Pendente de configurações de campanha/partida:**
- `fog_mode` (`live` | `explored`) é persistido em `maps.fog_mode` e honrado por
  `FilterMapState`, mas nenhum endpoint REST o expõe e `room.go` hardcoda `explored`.
  Será uma configuração de **partida**, escolhida pelo mestre — o mecanismo de
  configurações ainda não existe no backend. **Não remover** `FogMode`: isso eliminaria
  o modo `live`. Ver spec do refactor do mapa (repo do front), §3.
