# Fechamento da Fase 6 — pacote de back — plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development
> (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** entregar o §6A.5 do documento mestre (B1–B16) num PR do repo Go, com os contratos que
o PR de front consome.

**Architecture:** o servidor passa a ser o dono do tabuleiro (carrega do banco, persiste por
partida em `match_boards` + `player_memories`); um pacote `actionwire` monta o formato de action
do histórico em três níveis de corte para a fila do mestre, o `turn_opened` e a fila do dono; o
escape vira "esquiva **e** movimento", com a queda escolhida pelo mestre via `edit_action`; o
histórico ganha cena/round sem turno, uma tabela de eventos e as master actions — em tabela
própria (`master_actions`, decisão do dono do produto), projetadas pelo que cada jogador viu ao vivo.

**Tech Stack:** Go 1.23, gorilla/websocket, pgx/v5, goose, `testing` padrão.

**Spec:** [`docs/superpowers/specs/2026-09-27-combat-closure-back-design.md`](../specs/2026-09-27-combat-closure-back-design.md)
— leia o spec inteiro antes da primeira tarefa. Cada tarefa cita a seção do spec que implementa.
Documento mestre: `docs/superpowers/specs/2026-09-20-front-combat-phases.md` §6A.5 (em `main` desde o PR #80; a decisão das master actions no PR #81).

**Branch:** `feat/combat-closure-back` (a partir de `main`). Antes de abrir o PR, `git rebase origin/main` (o PR #81 muda o documento mestre).

## Global Constraints

- Go 1.23; `testing` padrão, table-driven com `t.Run`; pacotes de teste externos (`package foo_test`) onde o pacote já faz assim; **TDD** — o teste vem antes.
- `room.go` é dono do lock (`r.mu`). **Nada que envia a cliente roda com `r.mu` preso.** `dispatchPerPlayer`, `applyMove`, `relayPieceMove`, `buildMapFullState`, `buildMatchFullState` pegam o lock sozinhos — nunca chame com ele preso.
- `MatchSession` não tem lock: todo acesso à sessão é sob `r.mu`.
- Wire em **camelCase**; tipos de mensagem em snake_case; valores de enum do domínio como estão.
- **Nunca remover comentários `TODO`.** Comentários explicam o porquê, no tom e na densidade dos vizinhos.
- Todo item que muda o wire atualiza o contrato (`docs/dev/api/*.md`) **no mesmo commit**; pacote novo entra em `docs/documentation-map.yaml` no mesmo commit.
- Verificação por tarefa: `go build ./...`, `go vet ./...`, `go vet -tags integration ./...`, `go vet -tags smoke ./...`, `go test ./internal/...`. Tarefa que toca `room.go`: também `go test -race ./internal/app/game/`. Tarefa que toca `internal/gateway/pg/`: também `go test -tags=integration -p 1 ./internal/gateway/pg/...` (banco em `TEST_DATABASE_URL`, padrão `postgres://postgres:postgres@localhost:5432/hxh_rpg_test?sslmode=disable`).
- Commits terminam com a linha `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Falha de persistência é **logada dizendo o que se perdeu** e engolida (política do `persistClosedTurn`); nunca derruba a jogada.

## Review Focus

1. **Reinício no meio de um turno aberto** — o turno se perde, e a peça que andou na abertura volta com ele (o tabuleiro persiste no fechamento). Teste em T3.
2. **Dois salvamentos concorrentes do tabuleiro** (fechamento de turno + arrastar do mestre ao mesmo tempo) — o último retrato tirado é o que fica no banco. Teste em T3 (`persistMu`).
3. **Peça de jogador não inscrito no tabuleiro quando a sessão nasce** — é pulada com log, não aborta o `Init` nem vira participante. Teste em T9.
4. **Escape cuja escolha do mestre foi feita e depois o escape passou** (o mestre abriu outra reação) — a peça vai ao destino, não à escolha. Teste em T12.
5. **Player com duas abas em que a VELHA fecha depois** — a nova continua na sala e recebendo broadcast. Teste em T14.
6. **Master action pendurada num turno que se perdeu no reinício** — aparece no histórico fora de turno, na ordem do tempo, e não some. Teste em T13.

---

## Mapa de arquivos

| Arquivo | Responsabilidade | Tarefa |
|---|---|---|
| `internal/app/game/room_deps.go` (novo) | `RoomDeps`: todas as dependências de uma sala num struct | T0 |
| `migrations/20260927000000_match_boards_and_events.sql` (novo) | tabelas `match_boards`, `match_events` | T1 |
| `internal/domain/matchboard/board.go` (novo) | entidade `Board` | T1 |
| `internal/domain/matchevent/event.go` (novo) | entidade `Event` + `Kind` | T1 |
| `internal/domain/masteraction/record.go` (novo) | entidade `Record` (master action persistida) + `Kind` + `View` | T1 |
| `internal/gateway/pg/masteraction/*.go` (novo) | `Insert`, `ListByMatch` | T1 |
| `internal/gateway/pg/matchboard/*.go` (novo) | `Get`, `Save`, `Delete`, `Copy` | T1, T18 |
| `internal/gateway/pg/fog/player_memory_repository.go` | implementar o stub | T1 |
| `internal/gateway/pg/matchevent/*.go` (novo) | `Insert`, `ListByMatch` | T1 |
| `internal/application/matchboard/*.go` (novo) | `LoadMatchBoardUC`, `SaveMatchBoardUC` | T2, T3 |
| `internal/app/game/board.go` (novo) | conversão `mapentity.Piece` ↔ `PieceMovedPayload`, `loadBoard`, `persistBoard` | T2, T3 |
| `internal/app/game/master_board.go` (novo) | master actions de peça, `enrollLiveNPC` | T5 |
| `internal/app/wire/actionwire/*.go` (novo) | formato de action + `From(a, Level)` | T6 |
| `internal/domain/match/service/reaction_collision.go` | escape = esquiva **e** movimento | T11 |
| `internal/domain/match/entity/turn/turn.go` | `escapeLandings` | T12 |
| `internal/gateway/pg/round/*.go` | `EnsureSceneAndRound`, histórico com `LEFT JOIN` + eventos, escape no registro | T12, T13 |

---

### Task 0: `RoomDeps` — as dependências da sala num struct

Pura refatoração, sem mudança de comportamento. Hoje `NewHandler`, `Hub.GetOrCreateRoom` e
`NewRoom` recebem ~17 parâmetros posicionais, e este plano acrescenta mais seis. Seis lugares
constroem o handler (`cmd/game/main.go` e cinco testes).

**Files:**
- Create: `internal/app/game/room_deps.go`
- Modify: `internal/app/game/handler.go`, `hub.go`, `room.go` (struct `Room`, `NewRoom`), `cmd/game/main.go`
- Modify (testes): todo `game.NewHandler(` em `internal/app/game/*_test.go`

**Interfaces:**
- Produces: `type RoomDeps struct{...}`; `func NewHandler(hub *Hub, matchRepo MatchRepository, enrollmentRepo EnrollmentChecker, deps RoomDeps) *Handler`; `func (h *Hub) GetOrCreateRoom(matchUUID, masterUUID uuid.UUID, deps RoomDeps) *Room`; `func NewRoom(matchUUID, masterUUID uuid.UUID, deps RoomDeps) *Room`. Dentro da `Room`, os campos passam a ser `r.deps.StartMatchUC` etc.

- [ ] **Step 1: Criar o struct**

```go
package game

import appmatch "github.com/422UR4H/HxH_RPG_System/internal/application/match"

// RoomDeps is everything a Room needs from the outside, in one place.
//
// It exists because the list grew past what a positional signature can carry honestly: the
// handler, the hub and the room each took the same seventeen use cases in the same order, and
// every new dependency meant editing all three plus every test that builds a handler. A field
// left nil is a capability the room does not have — the arms that need it check.
type RoomDeps struct {
	StartMatchUC          IStartMatch
	KickPlayerUC          IKickPlayer
	InitSessionUC         IInitMatchSession
	OpenNextActionUC      IOpenNextAction
	PullActionUC          IPullAction
	EnqueueActionUC       IEnqueueAction
	AttachReactionUC      IAttachReaction
	OpenReactionUC        IOpenReaction
	CloseTurnUC           ICloseTurn
	ChangeSceneUC         IChangeScene
	RoundRepo             appmatch.IRoundRepository
	EnqueueMasterActionUC IEnqueueMasterAction
	ChangeRoundModeUC     appmatch.IChangeRoundMode
	EditActionUC          IEditAction
	AddLiveNPCUC          IAddLiveNPC
}
```

- [ ] **Step 2: Trocar as três assinaturas** e as leituras `r.startMatchUC` → `r.deps.StartMatchUC` (idem para todos). `Handler` guarda `deps RoomDeps` e repassa ao hub.
- [ ] **Step 3: Migrar os seis construtores** para `game.NewHandler(hub, matchRepo, enrollmentRepo, game.RoomDeps{StartMatchUC: ..., ...})` com campos nomeados.
- [ ] **Step 4: Rodar** `go build ./... && go vet ./... && go test -race ./internal/app/game/` — tudo PASS, nenhum teste alterado além do construtor.
- [ ] **Step 5: Commit** `refactor(game): RoomDeps reúne as dependências da sala`

---

### Task 1: Persistência do tabuleiro, da memória de fog, dos eventos e das master actions (infra de B3, B14, B15)

Spec §4.3 "Estrutura", §4.5 e §4.8. Só banco e gateway; ninguém chama ainda.

**Files:**
- Create: `migrations/20260927000000_match_boards_and_events.sql`
- Create: `internal/domain/matchboard/board.go`, `internal/domain/matchevent/event.go`
- Create: `internal/gateway/pg/matchboard/repository.go`, `board.go`, `matchboard_integration_test.go`
- Modify: `internal/gateway/pg/fog/player_memory_repository.go` (implementar), Create: `internal/gateway/pg/fog/fog_integration_test.go`
- Create: `internal/gateway/pg/matchevent/repository.go`, `matchevent_integration_test.go`
- Create: `internal/domain/masteraction/record.go`, `internal/gateway/pg/masteraction/repository.go`, `masteraction_integration_test.go`
- Modify: `internal/gateway/pg/pgtest/*` — `TruncateAll` precisa incluir as tabelas novas (confira a lista que ele trunca)
- Modify: `docs/documentation-map.yaml`

**Interfaces:**
- Produces:
  - `matchboard.Board{MatchUUID, MapUUID uuid.UUID; Grid mapentity.GridShape; Bg *mapentity.BgImage; Pieces []mapentity.Piece; Walls []mapentity.WallSegment; UpdatedAt time.Time}`
  - `pgmatchboard.Repository`: `Get(ctx, matchUUID) (*matchboard.Board, error)` (retorna `nil, nil` sem linha) · `Save(ctx, b *matchboard.Board) error` (upsert) · `Delete(ctx, matchUUID) error` · `Copy(ctx, srcMatch, dstMatch uuid.UUID) error` (T18)
  - `fog.PlayerMemoryRepository`: `NewPlayerMemoryRepository(q pgfs.IQuerier)`, `Upsert(ctx, m fogentity.PlayerMemory) error`, `FindByMatchMap(ctx, matchID, mapID uuid.UUID) ([]fogentity.PlayerMemory, error)`, `DeleteByMatch(ctx, matchID) error`
  - `matchevent.Event{UUID, MatchUUID, SceneUUID, RoundUUID uuid.UUID; Kind Kind; Payload json.RawMessage; CreatedAt time.Time}`; `Kind` = `"roundModeChanged"` (só ele: master action tem tabela própria)
  - `pgmatchevent.Repository`: `Insert(ctx, e matchevent.Event) error`, `ListByMatch(ctx, matchUUID) ([]matchevent.Event, error)` (ordem `created_at, uuid`)
  - `masteraction.Record{UUID, MatchUUID, SceneUUID, RoundUUID, MasterUUID uuid.UUID; TurnUUID *uuid.UUID; Kind Kind; Content json.RawMessage; Views map[uuid.UUID]View; HappenedAt time.Time}`; `Kind` = `"movePiece" | "placePiece" | "removePiece" | "wallInteract" | "revealWall" | "turnNote"`; `View` = `"full" | "left"` (ausente no mapa = não viu)
  - `func (r Record) ViewFor(playerID uuid.UUID) (View, bool)`
  - `type PieceContent struct{ CharacterID string \`json:"characterId"\`; PieceID string \`json:"pieceId"\`; From *[3]int \`json:"from,omitempty"\`; To *[3]int \`json:"to,omitempty"\` }` — o `Content` das três de peça
  - `func (r Record) ProjectFor(viewerIsMaster bool, viewer uuid.UUID) (Record, bool)` — a projeção do spec §4.8, **pura**: mestre → a mesma; `full` → a mesma sem `Views` (o mapa de quem viu não é dado de mesa); `left` → sem `Views` e, nas de peça, com `To` apagado do `Content`; sem entrada → `false` (a entrada não existe para ele)
  - `pgmasteraction.Repository`: `Insert(ctx, r masteraction.Record) error`, `ListByMatch(ctx, matchUUID) ([]masteraction.Record, error)` (ordem `happened_at, uuid`)

- [ ] **Step 1: Migração**

```sql
-- +goose Up
-- +goose StatementBegin
BEGIN;

-- The match's own board: a SNAPSHOT that starts as a copy of the campaign map it was attached
-- to and then belongs to the match alone. Editing the campaign map never reaches it, and two
-- matches on one map never touch each other's pieces (front-combat-phases.md §6A.5, B3).
-- bg NULL means "inherit the map's background"; the future in-match map editor writes here.
CREATE TABLE IF NOT EXISTS match_boards (
  match_uuid  UUID        PRIMARY KEY REFERENCES matches(uuid) ON DELETE CASCADE,
  map_uuid    UUID        NOT NULL    REFERENCES maps(uuid)    ON DELETE RESTRICT,
  grid        JSONB       NOT NULL,
  bg          JSONB,
  pieces      JSONB       NOT NULL DEFAULT '[]',
  walls       JSONB       NOT NULL DEFAULT '[]',
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- What happens INSIDE a round, is not a turn and is nobody's action (B15): a regime change.
CREATE TABLE IF NOT EXISTS match_events (
  uuid        UUID        PRIMARY KEY,
  match_uuid  UUID        NOT NULL REFERENCES matches(uuid) ON DELETE CASCADE,
  scene_uuid  UUID        NOT NULL REFERENCES scenes(uuid),
  round_uuid  UUID        NOT NULL REFERENCES rounds(uuid),
  kind        VARCHAR(32) NOT NULL,
  payload     JSONB       NOT NULL DEFAULT '{}',
  created_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_match_events_match ON match_events(match_uuid, created_at);

-- The master's own actions, in a table of their own and NOT in actions — a decision of the
-- product owner, and a matter of the model rather than taste (front-combat-phases.md §6A.5,
-- B14): actions.actor_uuid references character_sheets while a master action's actor is the
-- master, a USER; and actions.turn_uuid is NOT NULL while a master action happens outside any
-- turn (dragging between turns is the common case). Mixing them would loosen both columns and
-- force every history read to filter one kind out of the other.
--
-- turn_uuid is NULL outside a turn, and has no FK on purpose: a turn is only written when it
-- closes, and a restart can lose it after the master action was already recorded — which the
-- history then shows outside any turn. views is what each player saw of it LIVE
-- ({"<playerUUID>": "full"|"left"}; absent = saw nothing), so the history can show every
-- reader exactly that and no more. edit_action is NOT a master action and never lands here.
CREATE TABLE IF NOT EXISTS master_actions (
  uuid         UUID        PRIMARY KEY,
  match_uuid   UUID        NOT NULL REFERENCES matches(uuid) ON DELETE CASCADE,
  scene_uuid   UUID        NOT NULL REFERENCES scenes(uuid),
  round_uuid   UUID        NOT NULL REFERENCES rounds(uuid),
  turn_uuid    UUID,
  master_uuid  UUID        NOT NULL REFERENCES users(uuid),
  kind         VARCHAR(32) NOT NULL,
  content      JSONB       NOT NULL,
  views        JSONB       NOT NULL DEFAULT '{}',
  happened_at  TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_master_actions_match ON master_actions(match_uuid, happened_at);

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;
DROP INDEX IF EXISTS idx_master_actions_match;
DROP TABLE IF EXISTS master_actions;
DROP INDEX IF EXISTS idx_match_events_match;
DROP TABLE IF EXISTS match_events;
DROP TABLE IF EXISTS match_boards;
COMMIT;
-- +goose StatementEnd
```

(`match_events` e `master_actions` referenciam `scenes`/`rounds`: é por isso que T13 persiste
cena e round quando nascem. Até T13, quem grava master action — T5 — garante a cena/round antes;
ver T5 Step 4.)

- [ ] **Step 2: Entidades.** `matchboard.Board`, `matchevent.Event` e `masteraction.Record` como na interface. `masteraction` é domínio puro (sem I/O); `ViewFor` devolve `("", false)` para quem não está em `Views`. Teste unitário de `ProjectFor`, table-driven: mestre; `full`; `left` num `movePiece` (o `to` some, o `from` fica); `left` num `wallInteract` (igual ao `full` — não há destino); ausente → `false`; `turnNote` (Views vazio) para jogador → `false`.
- [ ] **Step 3: Testes de integração que falham** (`//go:build integration`, `package pgmatchboard_test`, padrão `pgtest.SetupTestDB` + `TruncateAll` por subteste): 
  - `TestMatchBoard`: `"get without row returns nil"`; `"save then get round-trips pieces walls grid and nil bg"`; `"save twice upserts"`; `"delete removes"`. Monte peças com `Slot: map[string]any{"kind":"square","col":3.0,"row":4.0}` — o `Slot any` volta de JSON como `map[string]any` com `float64`, e o teste deve comparar assim.
  - `TestPlayerMemory`: upsert de duas memórias do mesmo (match, map) com `Seen` diferentes → `FindByMatchMap` devolve as duas com o `Seen` certo; upsert de novo substitui; `DeleteByMatch` limpa. `seen_features` é `[{"kind":"wall","id":"<id>"}]`.
  - `TestMatchEvent`: insert de três eventos fora de ordem de inserção → `ListByMatch` em ordem de `created_at`.
  - `TestMasterAction`: insert com `turn_uuid` nulo e não nulo, `master_uuid` de um usuário real (`pgtest.InsertTestUser`), `views` com dois jogadores (`full`, `left`) → `ListByMatch` em ordem de `happened_at`, com `Views` e `Content` de volta iguais; `master_uuid` que não é usuário → erro de FK.
  - Precisa de cena/round reais para a FK: insira com SQL direto no teste (`INSERT INTO scenes ...`, `INSERT INTO rounds ...`), como `round_integration_test.go` faz.
- [ ] **Step 4: Rodar** `go test -tags=integration ./internal/gateway/pg/matchboard/... ./internal/gateway/pg/fog/... ./internal/gateway/pg/matchevent/... ./internal/gateway/pg/masteraction/...` → FAIL (não compila).
- [ ] **Step 5: Implementar.** `Save`:

```go
const q = `
	INSERT INTO match_boards (match_uuid, map_uuid, grid, bg, pieces, walls, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, now())
	ON CONFLICT (match_uuid) DO UPDATE SET
		map_uuid = EXCLUDED.map_uuid, grid = EXCLUDED.grid, bg = EXCLUDED.bg,
		pieces = EXCLUDED.pieces, walls = EXCLUDED.walls, updated_at = now()`
```

  Serialize `Grid`, `Bg`, `Pieces`, `Walls` com `json.Marshal` das entidades de `mapentity` (as tags snake_case delas são o formato de `maps.*` — **o mesmo JSON** que `maps.pieces`/`maps.walls` já guardam; confira `internal/gateway/pg/map/mapper.go` e reuse o que ele faz). `PlayerMemoryRepository` recebe `pgfs.IQuerier`; mantenha o comentário do stub reescrito para o que agora é verdade.
- [ ] **Step 6: Rodar** os testes de integração acima → PASS; `go test -tags=integration -p 1 ./internal/gateway/pg/...` → PASS.
- [ ] **Step 7: Registrar** os quatro gateways e as três entidades em `docs/documentation-map.yaml` apontando para `docs/dev/api/match-maps.md` (tabuleiro) e `docs/dev/api/match-history.md` (eventos, master actions).
- [ ] **Step 8: Commit** `feat(board): tabelas e gateways do tabuleiro, do fog, dos eventos e das master actions`

---

### Task 2: B14 — o servidor carrega o tabuleiro

Spec §4.3 "Quem carrega". Depende de T0 e T1.

**Files:**
- Create: `internal/application/matchboard/load_match_board.go`, `i_repository.go`, `load_match_board_test.go`
- Create: `internal/app/game/board.go`, `internal/app/game/board_test.go`
- Modify: `internal/app/game/room_deps.go` (campo `LoadBoardUC`), `room.go` (`Run` → carregar no register do mestre; braço `MsgTypeMapStateSync`; `revealSecretDoors`/interact → `unknown_wall`), `handler.go`, `cmd/game/main.go`
- Modify (testes): `combat_e2e_test.go` (`syncBoard` → `seedBoard`), `fog_e2e_test.go`, `fog_regression_test.go`, `fog_smoke_test.go`, `add_npc_e2e_test.go`, `mocks_test.go`/`handler_test.go` (fake de tabuleiro)
- Modify: `docs/dev/api/match-combat-ws.md` (`map_state_sync` obsoleto; `unknown_wall`), `docs/dev/api/game-lobby.md` (o servidor carrega o tabuleiro)

**Interfaces:**
- Consumes: `pgmatchboard.Repository.Get`, `pgmatchmap.Repository.GetMatchMap`, `pgmap.Repository.GetMap` (T1 e existentes).
- Produces:
  - `matchboarduc.ILoadMatchBoard{ Load(ctx, matchUUID uuid.UUID) (*matchboard.Board, error) }` — `nil, nil` quando a partida não tem mapa anexado.
  - `RoomDeps.LoadBoardUC matchboarduc.ILoadMatchBoard` (nil = sala sem tabuleiro, como nos testes antigos).
  - `func pieceToPayload(p mapentity.Piece) (PieceMovedPayload, error)` e `func payloadToPiece(p PieceMovedPayload) mapentity.Piece` em `board.go`.
  - `func (r *Room) loadBoard(ctx context.Context)` — carrega e substitui `r.pieces`, `r.walls`, `r.grid` (e `session.SyncMapState` se há sessão), sob `r.mu`.

- [ ] **Step 1: Teste do use case (falha).** `LoadMatchBoardUC` com fakes dos três repositórios, table-driven:
  - sem anexo (`GetMatchMap` → `ErrMatchMapNotFound`, confira o nome em `internal/application/matchmap/errors.go`/gateway) → `nil, nil`;
  - com anexo e linha em `match_boards` → devolve a linha, **sem** ler o mapa;
  - com anexo e sem linha → retrato do mapa: `MapUUID` do anexo, `Grid`, `Pieces`, `Walls` do mapa, `Bg: nil`.
- [ ] **Step 2: Implementar o use case** (`package matchboarduc`, interfaces próprias em `i_repository.go`, sem importar gateway além dos erros sentinela, como `matchmapuc` faz).
- [ ] **Step 3: Teste da conversão (falha)** em `board_test.go`: `Piece{ID:"p1", CharacterID:"c1", Coord{Slot: map[string]any{"kind":"hex","q":2.0,"r":-1.0}, Z:1.5}, Visible:false}` ↔ `PieceMovedPayload{PieceID:"p1", CharacterID:"c1", Slot: SlotPayload{Kind:"hex", Q:&2, R:&-1}, Z:1.5, Visible:&false}`; e o mesmo para `square` com `col`/`row`. Slot com `kind` desconhecido → erro. `payloadToPiece` devolve `Slot` como `SquareCoord`/`HexCoord` (os tipos de `mapentity`).
- [ ] **Step 4: Implementar a conversão.** Para ler `Slot any`: `json.Marshal(p.Coord.Slot)` e `json.Unmarshal` num `struct{Kind string; Col, Row, Q, R *int}` — robusto para `map[string]any` e para os structs.
- [ ] **Step 5: Fake de tabuleiro nos testes.** Em `handler_test.go` (onde vivem os mocks do pacote):

```go
// fakeBoardStore is the board repository the e2e tests share between rooms. Handing the SAME
// store to a second room is how a test simulates a server restart: memory goes, the store stays.
type fakeBoardStore struct {
	mu       sync.Mutex
	boards   map[uuid.UUID]*matchboard.Board
	saves    int
}

func (s *fakeBoardStore) Load(_ context.Context, matchUUID uuid.UUID) (*matchboard.Board, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.boards[matchUUID]
	if !ok {
		return nil, nil
	}
	cp := *b
	cp.Pieces = append([]mapentity.Piece(nil), b.Pieces...)
	cp.Walls = append([]mapentity.WallSegment(nil), b.Walls...)
	return &cp, nil
}
```

  (T3 acrescenta `Save`.) O `combatFixture` ganha `boards *fakeBoardStore` e o passa como `LoadBoardUC`.
- [ ] **Step 6: Migrar o semeio dos testes.** `syncBoard(t, master)` vira `seedBoard(t)`, chamado **antes** de `connect` (a sala carrega no register do mestre), escrevendo um `matchboard.Board` com as mesmas peças/paredes/grade de hoje (`moveBoardGrid`, `moveBoardWall`, `attackerElevation` etc.) no `fakeBoardStore`. Faça o mesmo nos outros quatro arquivos que mandam `map_state_sync` para semear. Os testes que usam `map_state_sync` para **provar** o comportamento dele (confira cada um pelo nome) passam a provar o comportamento novo (Step 7) ou são removidos com a justificativa na mensagem do commit.
- [ ] **Step 7: Testes e2e novos (falham):**
  - `TestE2E_TheRoomLoadsTheBoardWhenTheMasterConnects`: `seedBoard`, conectar mestre + jogador → os dois recebem `map_full_state` com as peças semeadas; o jogador com fog.
  - `TestE2E_MapStateSyncNoLongerWritesTheBoard`: `seedBoard`, conectar, o mestre manda `map_state_sync` com `pieces: []` → o mestre recebe `map_full_state` **com** as peças semeadas; o jogador não recebe nada novo.
  - `TestE2E_TheLobbyReloadsTheBoardOnMasterReconnect` (`inLobby`): conectar o mestre, trocar o conteúdo do `fakeBoardStore`, reconectar o mestre → o novo `map_full_state` tem o conteúdo novo.
  - `TestE2E_RevealingAnUnknownWallAnswersAnError`: `enqueue_master_action` com `interact: {kind:"reveal"}` e `targetIds` de uma parede que não existe → o mestre recebe `error` com `code: "unknown_wall"`. Idem para `kind: "open"`.
- [ ] **Step 8: Implementar na `Room`:**
  - `loadBoard(ctx)`: se `r.deps.LoadBoardUC == nil`, retorna. Senão carrega (fora do lock), converte as peças (peça que não converte é logada e pulada), e sob `r.mu.Lock` substitui `r.pieces`/`r.walls`/`r.grid`; com sessão, `SyncMapState` + `RecomputeVisibility` de cada jogador, como o braço do `map_state_sync` faz hoje.
  - No `Run`, braço `register`: **antes** de `sendRoomState`, se `client` é o mestre **e** (`r.session == nil` **ou** é o primeiro register da sala — um `boardLoaded bool` sob `r.mu`), chame `r.loadBoard`. Isso roda no goroutine do `Run`; a leitura do banco é curta e só acontece no register do mestre. Comente o porquê (é onde o `map_state_sync` chegava).
  - O `hasPieces` do register continua decidindo o `map_full_state`; mande-o também quando há paredes (`len(r.walls) > 0`), porque agora o tabuleiro pode ter paredes sem peça.
  - Braço `MsgTypeMapStateSync`: continua exigindo mestre; **não escreve nada**; responde ao remetente com `r.buildMapFullState(client.userUUID, true)`. Comentário: obsoleto até o F13 do front.
  - `applyWallInteract`/`revealSecretDoors`: parede ausente em `r.walls` → `client.SendMessage(NewErrorMessage("unknown_wall", "wall <id> is not on this match's board"))` ao mestre, no lugar do `continue` silencioso. Mantenha o resto do loop.
- [ ] **Step 9: Wiring** em `cmd/game/main.go`: `mapPg.NewRepository(pgPool)`, `matchmapPg.NewRepository(pgPool)`, `matchboardPg.NewRepository(pgPool)` → `matchboarduc.NewLoadMatchBoardUC(...)` → `RoomDeps.LoadBoardUC`.
- [ ] **Step 10: Contratos.** `match-combat-ws.md`: `map_state_sync` — "obsoleto: aceito e ignorado; responde com `map_full_state`; será removido depois do F13"; `unknown_wall` na lista de erros de `enqueue_master_action`; seção nova "O tabuleiro é do servidor" (carregado do banco quando a sala nasce e, no lobby, a cada conexão do mestre). `game-lobby.md`: o mesmo, do lado do lobby.
- [ ] **Step 11: Rodar** a verificação global + `go test -race ./internal/app/game/` → PASS.
- [ ] **Step 12: Commit** `feat(game): o servidor carrega o tabuleiro da partida (B14)`

---

### Task 3: B3 — o tabuleiro e o fog persistem por partida

Spec §4.3 "Quando persiste" e ⭐. Depende de T2.

**Files:**
- Create: `internal/application/matchboard/save_match_board.go`, `save_match_board_test.go`
- Modify: `internal/app/game/board.go` (`persistBoard`), `room.go` (pontos de salvamento; `SyncPlayerMemories` com memórias carregadas), `room_deps.go` (`SaveBoardUC`, `MemoryLoader`), `cmd/game/main.go`
- Test: `internal/app/game/board_persist_e2e_test.go` (novo)
- Modify: `docs/dev/api/match-maps.md` (seção "O tabuleiro da partida"), `match-combat-ws.md` (§ reinício)

**Interfaces:**
- Consumes: T1 (`pgmatchboard.Save`, `fog.PlayerMemoryRepository`), T2 (`loadBoard`, `payloadToPiece`).
- Produces:
  - `matchboarduc.ISaveMatchBoard{ Save(ctx, b *matchboard.Board, memories []fogentity.PlayerMemory) error }` — grava a linha e faz upsert de cada memória; erro da linha aborta, erro de uma memória é agregado (`errors.Join`).
  - `matchboarduc.IMemoryLoader{ FindByMatchMap(ctx, matchID, mapID uuid.UUID) ([]fogentity.PlayerMemory, error) }`
  - `func (r *Room) persistBoard(reason string)` — o único ponto de gravação.
  - `Room.mapUUID uuid.UUID` (vem do `Board.MapUUID` no `loadBoard`; zero = sem mapa, e aí `persistBoard` não grava nada).

- [ ] **Step 1: Teste do use case (falha)**: salva a linha e as N memórias; memória que falha não impede as outras e o erro volta agregado; linha que falha não tenta memórias.
- [ ] **Step 2: Implementar** `SaveMatchBoardUC`.
- [ ] **Step 3: Estender o fake** `fakeBoardStore` com `Save` (guarda cópia, incrementa `saves`) e um `fakeMemoryStore` com `Upsert`/`FindByMatchMap`.
- [ ] **Step 4: Testes e2e (falham)** em `board_persist_e2e_test.go`:
  - `TestE2E_AClosedTurnPersistsTheBoard`: `seedBoard`, Dash do atacante para (5,4), abrir, `close_turn` → o `fakeBoardStore` tem a peça do atacante em (5,4). Repita para os outros dois verbos que fecham (`open_next_action` com outra ação na fila; `pull_action`) — table-driven pelo verbo.
  - `TestE2E_AServerRestartBringsTheBoardBack`: depois do fechamento acima, **pare o servidor da fixture e suba outra `Room`/handler sobre o mesmo `fakeBoardStore`** (helper `f.restart(t)` que fecha `f.server`, cria hub+handler novos com os mesmos stores e o mesmo `combatSessionUC`, e troca `f.server`) → o `map_full_state` do mestre tem a peça em (5,4).
  - `TestE2E_ARestartMidTurnLosesTheTurnAndTheMove`: Dash aberto e **não** fechado, `f.restart` → a peça volta à posição semeada (review focus 1).
  - `TestE2E_PlayerMemorySurvivesARestart`: jogador anda até ver a parede `moveBoardWall`, fechar turno, `f.restart` → o `map_full_state` do jogador **longe** da parede ainda traz a parede (memória `explored`).
  - `TestPersistBoardWritesSnapshotsInOrder` (unitário no pacote, `package game` interno se precisar de acesso a `persistBoard` — se o pacote de testes é externo, exponha `export_test.go` com `func (r *Room) PersistBoardForTest(reason string)`): duas goroutines chamam `persistBoard` depois de mover a peça para A e para B em sequência; com um `Save` fake que dorme no primeiro chamado, o último `Save` gravado tem a posição B (review focus 2).
- [ ] **Step 5: Implementar `persistBoard`:**

```go
// persistBoard writes the match's board — pieces, walls, and every player's fog memory — as it
// stands NOW. It is the one place that does, and every definitive change to the board calls it:
// the lobby's moves, start_match, the three verbs that close a turn, the master's piece actions
// and wall interactions (spec §4.3).
//
// persistMu wraps the snapshot AND the write, so two saves racing from two read pumps land in
// the order their snapshots were taken; r.mu is only held for the snapshot, never across the
// round trip. A failure is logged and swallowed — the table goes on, as persistClosedTurn does.
//
// The caller must NOT hold r.mu.
func (r *Room) persistBoard(reason string) {
	if r.deps.SaveBoardUC == nil {
		return
	}
	r.persistMu.Lock()
	defer r.persistMu.Unlock()

	r.mu.RLock()
	if r.mapUUID == uuid.Nil {
		r.mu.RUnlock()
		return
	}
	b := &matchboard.Board{MatchUUID: r.matchUUID, MapUUID: r.mapUUID, Grid: r.grid, Bg: r.bg}
	if r.session != nil {
		b.Grid = r.session.GetGrid()
	}
	for _, p := range r.pieces {
		b.Pieces = append(b.Pieces, payloadToPiece(p))
	}
	for _, w := range r.walls {
		b.Walls = append(b.Walls, w)
	}
	var mems []fogentity.PlayerMemory
	if r.session != nil {
		for _, pid := range r.session.PlayerIDs() {
			if m, ok := r.session.GetPlayerMemory(pid); ok && m != nil {
				cp := *m
				cp.Seen = maps.Clone(m.Seen)
				mems = append(mems, cp)
			}
		}
	}
	r.mu.RUnlock()

	sort.Slice(b.Pieces, func(i, j int) bool { return b.Pieces[i].ID < b.Pieces[j].ID })
	sort.Slice(b.Walls, func(i, j int) bool { return b.Walls[i].ID < b.Walls[j].ID })
	if err := r.deps.SaveBoardUC.Save(context.Background(), b, mems); err != nil {
		log.Printf("persistBoard(%s) FAILED — board of match %s was NOT saved: %v", reason, r.matchUUID, err)
	}
}
```

  Confira que a memória guarda `MatchID`/`MapID` certos: ao criar memórias (`SyncPlayerMemories`/`RecomputeVisibility`), a sessão usa o `mapID` que tem — passe `r.mapUUID` para a sessão no `loadBoard` se ela não o tiver (procure onde `fog.NewPlayerMemory` é chamado em `match_session.go` e alinhe). `r.bg` é o `Board.Bg` carregado, preservado (ninguém escreve ainda).
- [ ] **Step 6: Chamar `persistBoard`** — sempre **depois** que a mudança já foi aplicada e sem `r.mu`:
  - fechamento de turno, nos três braços, **logo depois** de `r.applyClosedEscapes(...)` e antes de `persistClosedTurn` (reason `"turn_closed"`);
  - `StartMatch`, antes do `Init` (reason `"start_match"`) — B11 lê o que está ali (T9);
  - braço `MsgTypeEnqueueMasterAction`, depois do loop de interação de parede e depois de `revealSecretDoors` (reason `"wall_interact"`);
  - `handlePieceMoved`/`handlePieceRemoved` (lobby — T4 restringe ao lobby) (reason `"lobby"`).
- [ ] **Step 7: Memórias ao nascer a sessão.** Em `StartMatch` e `RehydrateSession`, troque `SyncPlayerMemories(nil, ...)` por memórias carregadas de `r.deps.MemoryLoader.FindByMatchMap(ctx, r.matchUUID, r.mapUUID)` (carregue **antes** de pegar `r.mu`; erro → log e `nil`). **Não remova** o comentário do `fogMode fixo em explored`. Remova o `// TODO(persistence): playerMemoryRepo.Upsert(...)` de `StartMatch` **somente** porque ele foi feito — nunca remova TODO que não foi resolvido; substitua pelo comentário do que acontece agora.
- [ ] **Step 8: Wiring** em `cmd/game/main.go` (`SaveBoardUC`, `MemoryLoader` = `fog.NewPlayerMemoryRepository(pgPool)`).
- [ ] **Step 9: Contratos.** `match-maps.md`: seção "O tabuleiro da partida" (retrato por partida, `bg` nulo herda, quando persiste, o que o editor de mapa da campanha **não** muda). `match-combat-ws.md`: seção de reinício com a tabela do spec §5 (linhas tabuleiro e turno aberto).
- [ ] **Step 10: Rodar** verificação global, `-race`, integração → PASS.
- [ ] **Step 11: Commit** `feat(game): o tabuleiro e o fog persistem por partida (B3)`

---

### Task 4: B14 — `piece_moved`/`piece_removed` só no lobby, validados

Spec §4.3 "Quem move o quê". Depende de T2/T3.

**Files:**
- Modify: `internal/app/game/room.go` (`handlePieceMoved`, `handlePieceRemoved`), `room_deps.go` (`SheetOwnership`), `cmd/game/main.go`
- Test: `internal/app/game/lobby_board_e2e_test.go` (novo)
- Modify: `docs/dev/api/game-lobby.md`, `match-combat-ws.md`

**Interfaces:**
- Consumes: `ISheetOwnershipReader` que já existe em `internal/application/match` (`GetCharacterSheetRelationshipUUIDs(ctx, sheetUUID) (*csEntity.RelationshipUUIDs, error)` — confira o nome exato do tipo de retorno em `add_match_npc.go`).
- Produces: `RoomDeps.SheetOwnership appmatch.ISheetOwnershipReader`.

- [ ] **Step 1: Testes e2e (falham)**, table-driven onde der:
  - lobby, mestre move qualquer peça → aplicado, relay para todos, `fakeBoardStore.saves` +1;
  - lobby, jogador move peça **existente** do próprio personagem → aplicado;
  - lobby, jogador move peça de personagem de **outro** → `error` `forbidden`, nada muda;
  - lobby, jogador manda `piece_moved` com `pieceId` que não existe (criar) → `forbidden`;
  - lobby, jogador manda `piece_removed` → `forbidden`; mestre → aplicado e persistido;
  - partida, mestre manda `piece_moved` → `forbidden` com mensagem `"during a match the master moves pieces with enqueue_master_action"`;
  - partida, jogador manda `piece_moved` → `forbidden`, `"players move by action"`.
- [ ] **Step 2: Implementar.** Em `handlePieceMoved`: com sessão → recusa pelas duas mensagens acima. Sem sessão: mestre passa; jogador precisa que a peça **já exista** em `r.pieces` e que `SheetOwnership` diga `PlayerUUID == client.userUUID` para o `CharacterID` **atual** da peça (e o `CharacterID` do payload tem que ser o mesmo — o jogador não troca o dono da peça). A leitura de posse é I/O: faça fora de `r.mu`. Depois de aplicar, `r.persistBoard("lobby")`. Troque o comentário `No server-side piece ownership validation in Phase 6` pelo que agora é verdade — **mas o `TODO: validate piece ownership per user (Phase 7+)` só sai porque foi feito**; diga isso na mensagem do commit.
- [ ] **Step 3: Contratos** (lobby: quem move o quê; partida: recusado, com o caminho certo).
- [ ] **Step 4: Rodar** verificação + `-race` → PASS.
- [ ] **Step 5: Commit** `feat(game): piece_moved só no lobby e validado no servidor (B14)`

---

### Task 5: B14 + `move` de B9 + B11 ao vivo — master actions de peça, persistidas

Spec §4.3 "Master action de peça", "B11 … Com a sala viva" e **§4.8** (tabela própria, gravada
no instante, `views`). Depende de T1–T4.

**Files:**
- Create: `internal/app/game/master_board.go`, `internal/app/game/master_action_record.go`, `internal/app/game/master_board_e2e_test.go`
- Modify: `internal/app/game/message.go` (`MasterActionPayload.Remove`), `action_mapper.go` (`buildMasterAction` mapeia `Move`), `room.go` (braço `MsgTypeEnqueueMasterAction`; `relayPieceMove`/`handlePieceRemoved`/`broadcastWallStateChangedGated`/`revealSecretDoors` passam a devolver o que cada jogador viu; braço `MsgTypeAddNPC` passa a chamar `enrollLiveNPC`), `room_deps.go` (`MasterActionRepo`)
- Modify: `internal/application/match/i_repository.go` + `internal/gateway/pg/round/` — `EnsureSceneAndRound` (ver Step 4)
- Modify: `docs/dev/api/match-combat-ws.md`, `docs/dev/api/match-npcs.md`

**Interfaces:**
- Consumes: T1 `pgmasteraction.Insert`, `masteraction.Record`/`Kind`/`View`; T3 `persistBoard`; `r.deps.AddLiveNPCUC`.
- Produces:
  - `type RemovePiecePayload struct{}`; `MasterActionPayload.Remove *RemovePiecePayload \`json:"remove,omitempty"\``
  - `func (r *Room) enrollLiveNPC(client *Client, sheetUUID uuid.UUID) bool` — o corpo atual do braço `add_npc` (use case + `session.AddNPC` + `npc_added` + `broadcastBars`), devolvendo `true` se o NPC ficou na sessão. Mesmo mapeamento de erros de hoje.
  - `func (r *Room) applyMasterPieceAction(client *Client, ma *action.MasterAction, remove bool)`
  - `func (r *Room) relayPieceMove(payload, old PieceMovedPayload, hadOld bool, origin uuid.UUID) map[uuid.UUID]masteraction.View` (antes sem retorno — os chamadores que não gravam ignoram)
  - `func (r *Room) relayPieceRemoved(old PieceMovedPayload, hadOld bool, origin uuid.UUID) map[uuid.UUID]masteraction.View` (extraído de `handlePieceRemoved`)
  - `func (r *Room) recordMasterAction(kind masteraction.Kind, content any, views map[uuid.UUID]masteraction.View)` — o único ponto que grava em `master_actions`
  - `IRoundRepository.EnsureSceneAndRound(ctx, matchUUID uuid.UUID, sc *scene.Scene, rd *round.Round) error` (idempotente; T13 também usa)
  - `RoomDeps.MasterActionRepo interface{ Insert(ctx context.Context, r masteraction.Record) error }`

- [ ] **Step 1: Teste do mapper (falha)** em `action_mapper_test.go`: `MasterActionPayload{TargetIDs: [c], Move: &MovePayload{Position: [3]int{3,4,0}}}` → `ma.Move.Position == [3]int{3,4,0}`, `ma.Move.Category` vazio. Sem `Move` → `ma.Move == nil`.
- [ ] **Step 2: `buildMasterAction` mapeia `Move`** (só `Position`; comente que categoria, speed e charge não valem para o arrastar). **Mantenha** o `TODO` de `Attack` até T17.
- [ ] **Step 3: Testes e2e (falham)** em `master_board_e2e_test.go`, com `withBystander` (o bystander fica **fora** da visão do destino) e um `fakeMasterActionStore` (guarda os `Record`s):
  - `drag, no open turn`: mestre arrasta a peça do atacante → recebem `piece_moved` o dono e **o mestre**; o bystander não; o mestre recebe `master_action_enqueued`, os jogadores **não**; `fakeBoardStore` salvo; **um `Record`** `kind: movePiece`, `TurnUUID == nil`, `MasterUUID == f.masterUUID`, `Content` = `masteraction.PieceContent{characterId, pieceId, from, to}`, `Views == {dono: full}` (bystander ausente).
  - `drag seen leaving`: o bystander vê a origem e não o destino → ele recebe `piece_removed` ao vivo, e o `Record` tem `Views[bystander] == left`.
  - `drag, open turn`: o mesmo com um turno aberto → `Record.TurnUUID` = o turno; a master action também pendurada no turno (`session.CurrentTurn().GetMasterActions()` tem 1 — leia pela `f.session` só depois que as mensagens chegaram).
  - `place an NPC of the master that is not a participant`: com `withAddLiveNPC(fake)` e um NPC fora da sessão → peça criada no slot, `npc_added` para a mesa, `bars_updated`, `Record` `placePiece`.
  - `place a player's character that is not a participant` → `error` `not_participant`, nada criado, **nenhum** `Record`.
  - `remove`: tirar a peça do atacante → `piece_removed` com fog; `Record` `removePiece`; o personagem continua participante (as barras ainda o listam no próximo `bars_updated`).
  - `wall interact`: mestre abre a porta `moveBoardWall` → `Record` `wallInteract` com `Views` = quem recebeu o `wall_state_changed` ao vivo.
  - `reveal`: revelar porta secreta → `Record` `revealWall` com `full` para todos os jogadores da sessão.
  - `generic master action on an open turn` (só `targetIds` + `skills`, o caminho que já existe) → `Record` `turnNote`, `Views` vazio.
  - `edit_action is not a master action`: um `edit_action` com turno aberto → **nenhum** `Record`.
  - `a player cannot send a master action` → `forbidden` (já é a regra do braço), nenhum `Record`.
- [ ] **Step 4: Garantir cena/round antes de gravar.** `master_actions` tem FK para `scenes`/`rounds`, e até T13 cena e round só existem no banco depois do primeiro turno fechado. Extraia de `PersistTurnClose` os dois inserts idempotentes para `EnsureSceneAndRound` (mesmo SQL `ON CONFLICT DO NOTHING`), chame-o de dentro de `PersistTurnClose` (mesma transação: aceite um `pgx.Tx`/querier) e exponha a versão pública com o pool. Teste de integração: chamar duas vezes não falha; depois `PersistTurnClose` do mesmo round não falha. O fake `mockRoundRepoHandler` ganha o método (no-op que registra chamadas).
- [ ] **Step 5: O que cada jogador viu (`views`).** Hoje o portão de fog vive dentro da closure do `dispatchPerPlayer` de `relayPieceMove` (e do `handlePieceRemoved`, e do `broadcastWallStateChangedGated`). Extraia cada portão para uma função pura sobre um jogador, e use-a **duas vezes**: no dispatch (conectados) e para montar `views` sobre **todos** os jogadores da sessão (`session.PlayerIDs()`, conectados ou não — o fog que conta é o daquele instante, lido de `r.visibilityFor`, cujo cache existe para todos):

```go
// pieceMoveView is the fog gate of one piece move for ONE player — the same decision the live
// dispatch makes, extracted so the SAME decision can be recorded with a master action and the
// history can show each reader exactly what they saw then (spec §4.8). Pure: reads the cache.
func pieceMoveView(polys []domainservice.VisibilityPolygon, newPt, oldPt domainservice.Point2D, hadOld, hidden bool) (masteraction.View, bool) {
	if hidden {
		return "", false
	}
	switch {
	case domainservice.IsVisible(newPt, polys):
		return masteraction.ViewFull, true
	case hadOld && domainservice.IsVisible(oldPt, polys):
		return masteraction.ViewLeft, true
	default:
		return "", false
	}
}
```

  O dispatch continua mandando `piece_moved` para `ViewFull`, `piece_removed` para `ViewLeft` e nada para ausente — o comportamento ao vivo **não muda** (os testes de fog existentes são a guarda). `relayPieceMove` devolve o mapa (a leitura de `PlayerIDs` sob `r.mu.RLock`, como o resto). Remoção: visível na última posição → `full`. Parede: o critério que `broadcastWallStateChangedGated` já usa → `full`. Revelar: todos `full`.
- [ ] **Step 6: Implementar** `master_board.go`:
  - O braço `MsgTypeEnqueueMasterAction` ganha, **antes** do ramo de `Interact`: `if ma.Move != nil || payload.Remove != nil { r.applyMasterPieceAction(client, ma, payload.Remove != nil); return }`.
  - `applyMasterPieceAction`: exige sessão (`match_not_started` sem ela — no lobby o mestre usa `piece_moved`); `len(ma.TargetID) != 1` → `invalid_action`. Personagem = `ma.TargetID[0]`.
  - **Tirar:** acha a peça (mesma regra do `applyMove`: menor ID do personagem); sem peça → `invalid_action` `"character has no piece"`; remove sob `r.mu`; `views := r.relayPieceRemoved(old, true, uuid.Nil)`.
  - **Mover/pôr:** com peça → mesma escrita do `applyMove` (preserve `Kind` do slot e `Z`) e `views := r.relayPieceMove(moved, old, true, uuid.Nil)`. Sem peça → antes de tudo, `SheetOwnership`: ficha de jogador (`PlayerUUID != nil`) que não está em `session.GetCharToPlayer()` → `not_participant`; NPC fora da sessão → `enrollLiveNPC` (falhou → já respondeu o erro; retorna). Cria `PieceMovedPayload{PieceID: uuid.NewString(), CharacterID, Slot: slotFor(grid, pos), Visible: ptr(true)}` e `views := r.relayPieceMove(p, PieceMovedPayload{}, false, uuid.Nil)`.
  - Depois de aplicar: se há turno aberto, `session.EnqueueMasterAction(ma)` sob `r.mu` (ignore `ErrNoActiveTurn`); `r.persistBoard("master_piece")`; `r.recordMasterAction(kind, content, views)`; `client.SendMessage(master_action_enqueued)` — **só para o mestre**, não `r.broadcast`.
  - `slotFor(g mapentity.GridShape, pos [3]int) SlotPayload`: hex → `{Kind:"hex", Q, R}`; senão `{Kind:"square", Col, Row}`.
  - Os ramos que já existem também gravam: interação de parede (`recordMasterAction(wallInteract, {wallIds, interact}, views)` com as `views` do Step 5), revelar (`revealWall`, todos `full`) e o caminho genérico pelo `enqueueMasterActionUC` quando ele **aceita** (`turnNote`, `Views` vazio, conteúdo = o payload). Só depois do sucesso: master action recusada não é gravada.
  - O braço `MsgTypeAddNPC` passa a chamar `r.enrollLiveNPC(client, payload.CharacterSheetUUID)` — comportamento idêntico; os testes de `add_npc_e2e_test.go` continuam passando sem mudança.
- [ ] **Step 7: `recordMasterAction`** em `master_action_record.go`:

```go
// recordMasterAction writes one master action the instant it was applied, with or without an
// open turn — the moment B3 saves the board (spec §4.8). It is the ONLY writer of
// master_actions, and it is called from the enqueue_master_action arm alone: edit_action also
// hangs a MasterAction on the turn, but an edit is not a master action (it lives in
// overridden_action_values), which is why this never reads Turn.GetMasterActions().
//
// views is what each player of the session saw of it live; the history shows every reader
// exactly that. The caller must NOT hold r.mu.
func (r *Room) recordMasterAction(kind masteraction.Kind, content any, views map[uuid.UUID]masteraction.View) {
	if r.deps.MasterActionRepo == nil {
		return
	}
	r.mu.RLock()
	sess := r.session
	if sess == nil {
		r.mu.RUnlock()
		return
	}
	sc, rd := sess.GetActiveScene(), sess.GetActiveRound()
	var turnID *uuid.UUID
	if id := sess.CurrentTurnID(); id != uuid.Nil && rd.HasOpenTurn() {
		turnID = &id
	}
	r.mu.RUnlock()

	raw, err := json.Marshal(content)
	if err != nil {
		log.Printf("recordMasterAction(%s): marshal: %v", kind, err)
		return
	}
	ctx := context.Background()
	if err := r.deps.RoundRepo.EnsureSceneAndRound(ctx, r.matchUUID, sc, rd); err != nil {
		log.Printf("recordMasterAction(%s) FAILED — scene/round of match %s not ensured, action NOT recorded: %v", kind, r.matchUUID, err)
		return
	}
	rec := masteraction.Record{
		UUID: uuid.New(), MatchUUID: r.matchUUID, SceneUUID: sc.GetID(), RoundUUID: rd.GetID(),
		TurnUUID: turnID, MasterUUID: r.masterUUID, Kind: kind, Content: raw, Views: views,
		HappenedAt: time.Now().UTC(),
	}
	if err := r.deps.MasterActionRepo.Insert(ctx, rec); err != nil {
		log.Printf("recordMasterAction(%s) FAILED — master action of match %s was NOT recorded: %v", kind, r.matchUUID, err)
	}
}
```

- [ ] **Step 8: Wiring** em `cmd/game/main.go`: `masteractionPg.NewRepository(pgPool)` → `RoomDeps.MasterActionRepo`.
- [ ] **Step 9: Contratos.** `match-combat-ws.md` `enqueue_master_action`: as duas formas de peça (`move`, `remove`), com ou sem turno, o que cada uma emite, `master_action_enqueued` master-only para peça, os erros (`not_participant`, `invalid_action`, `match_not_started`), e **"toda master action aceita é gravada no instante e aparece no histórico como cada jogador a viu"** (link para `match-history.md`). Atualize a tabela de destinos do §3 (`master_action_enqueued` → "mesa inteira; só o mestre nas ações de peça"). `match-npcs.md`: pôr a peça de um NPC inscreve e emite `npc_added`, igual ao `add_npc`.
- [ ] **Step 10: Rodar** verificação + `-race` + integração → PASS; os testes de fog existentes (`fog_*_test.go`, `visibility_e2e_test.go`) passam **sem mudança** — são a prova de que extrair o portão não mudou o ao vivo.
- [ ] **Step 11: Commit** `feat(game): o mestre move, põe e tira peças por master action, e toda master action é gravada (B14, B9, B11)`

---

### Task 6: `actionwire` — um formato de action, três níveis

Spec §4.1. Independente de T0–T5.

**Files:**
- Create: `internal/app/wire/actionwire/action.go`, `from.go`, `from_test.go`
- Modify: `internal/app/api/match/get_match_history.go` (usa `actionwire`), Test: `internal/app/api/match/get_match_history_golden_test.go` (novo) + `testdata/history_golden.json`
- Modify: `docs/documentation-map.yaml`

**Interfaces:**
- Produces:
  - `type Level int`; `const Full, Opened, Declaration Level`
  - `func From(a action.Action, lvl Level) Action`
  - Tipos (mesmos campos e tags JSON do `ActionResponse` de hoje): `Action`, `Skill`, `RollCheck{SkillName string; SkillValue *int \`json:"skillValue,omitempty"\`; Attempts *RollAttempts \`json:"attempts,omitempty"\`; Result *int \`json:"result,omitempty"\`}`, `RollAttempts`, `Trigger`, `ActionSpeed{Bar int; RollCheck RollCheck}`, `Move{...; FinalSpeed *int \`json:"finalSpeed,omitempty"\`}`, `Attack`, `Defense`, `Dodge`, `Repel`, `Interact`.

- [ ] **Step 1: Teste de ouro (falha antes da troca? não — ele é escrito e passa ANTES da troca).** Em `get_match_history_golden_test.go`: monte um `HistoryTurn` com uma action cheia (skills, speed, move com `From` e `Charge`, attack com charge e spread, feint, trigger, interact, systemBias) e uma reaction (dodge, defense, repel), passe por `toHistoryTurnResponse`, `json.Marshal`, e compare com `testdata/history_golden.json`. Gere o arquivo **com o código de hoje** (`-update` flag no teste) e commite-o separado: `test(history): ouro do JSON do histórico`.
- [ ] **Step 2: Teste do `From` (falha):**

```go
func TestFromCutsByLevel(t *testing.T) {
	a := fixtureAction() // attack + move + skills + feint, every RollCheck with Attempts and Result
	full := actionwire.From(a, actionwire.Full)
	opened := actionwire.From(a, actionwire.Opened)
	decl := actionwire.From(a, actionwire.Declaration)

	t.Run("full keeps every number", func(t *testing.T) {
		if full.Attack.Hit.Result == nil || full.Speed.RollCheck.Result == nil || full.Move.FinalSpeed == nil {
			t.Fatal("full must keep hit, speed and finalSpeed")
		}
	})
	t.Run("opened keeps the speeds and cuts the rest", func(t *testing.T) {
		if opened.Speed.RollCheck.Result == nil || opened.Move.Speed.Result == nil || opened.Move.FinalSpeed == nil {
			t.Fatal("opened must keep actionSpeed and moveSpeed")
		}
		for name, rc := range map[string]actionwire.RollCheck{
			"hit": opened.Attack.Hit, "damage": opened.Attack.Damage, "skill": opened.Skills[0].RollCheck, "feint": *opened.Feint,
		} {
			if rc.Result != nil || rc.Attempts != nil || rc.SkillValue != nil || rc.SkillName == "" {
				t.Errorf("%s: want only the skill name, got %+v", name, rc)
			}
		}
	})
	t.Run("declaration keeps no number at all", func(t *testing.T) {
		if decl.Speed.RollCheck.Result != nil || decl.Move.FinalSpeed != nil || decl.Move.Speed.Result != nil {
			t.Fatal("declaration must carry no speed")
		}
		if decl.Move.Position != a.Move.Position || len(decl.TargetID) != len(a.TargetID) || decl.Attack.Weapon == nil {
			t.Fatal("declaration must keep what the owner declared")
		}
	})
}
```

- [ ] **Step 3: Implementar** `action.go` (tipos, doc comments movidos do `get_match_history.go` — são o porquê de cada campo) e `from.go`: um `rollCheck(rc action.RollCheck, keep bool) RollCheck` que devolve só `SkillName` quando `!keep`; `From` decide `keep` por campo segundo a tabela do spec §4.1 (speed/move.speed/finalSpeed: `lvl != Declaration`; todo o resto: `lvl == Full`). Sem lógica de visibilidade — o doc comment de `From` diz que a deny-list é do `service.ProjectAction`, que roda antes.
- [ ] **Step 4: Trocar o REST** para `actionwire.From(a, actionwire.Full)`; `HistoryTurnResponse.Action`/`Reactions` passam a ser `actionwire.Action`. Apague os tipos duplicados de `get_match_history.go`.
- [ ] **Step 5: Rodar** `go test ./internal/app/api/match/ ./internal/app/wire/...` → o ouro continua idêntico, `From` PASS. Verificação global.
- [ ] **Step 6: Registrar** `internal/app/wire/actionwire/` no `documentation-map.yaml` → `match-history.md` e `match-combat-ws.md`.
- [ ] **Step 7: Commit** `refactor(wire): actionwire, o formato de action em três níveis`

---

### Task 7: B1 — a fila do mestre carrega a declaração inteira

Spec §4.2 B1. Depende de T6.

**Files:** Modify `internal/app/game/message.go` (`ActionQueuedPayload.Action`), `room.go` (`newActionQueuedPayload`); Test: `combat_e2e_test.go` (ou `queue_e2e_test.go` novo); Modify `match-combat-ws.md`.

**Interfaces:** Produces `ActionQueuedPayload.Action actionwire.Action \`json:"action"\``.

- [ ] **Step 1: Teste e2e (falha)** `TestActionQueuedCarriesTheWholeDeclaration`: jogador enfileira ataque com Dash → o `action_queued` do mestre tem `action.attack.weapon == "Sword"`, `action.attack.hit.result != nil`, `action.speed.rollCheck.result != nil`, `action.move.finalSpeed != nil`, `action.move.position` o enviado, `action.targetId` o alvo. E o `match_full_state.queue[0].action` de um mestre que reconecta é **igual** (compare o JSON dos dois).
- [ ] **Step 2: Teste negativo** no mesmo arquivo: o jogador que enfileirou **não** recebe `action_queued` (já é a regra; o teste fica como guarda de B1 ⚠️).
- [ ] **Step 3: Implementar**: `newActionQueuedPayload` preenche `Action: actionwire.From(*a, actionwire.Full)`. Atualize o doc comment de `ActionQueuedPayload` — a frase "Nothing here describes the action's CONTENT" deixa de ser verdade; diga que agora descreve, e por que é seguro (master-only).
- [ ] **Step 4: Contrato**: `action_queued` e `match_full_state.queue` com o campo `action`, exemplo JSON, e a linha "o formato é o de `match-history.md`, nível cheio".
- [ ] **Step 5: Rodar** verificação + `-race` → PASS. **Commit** `feat(game): a fila do mestre carrega a declaração inteira (B1)`

---

### Task 8: B2 — `turn_opened` projetado, e a ordem nova

Spec §4.2 B2. Depende de T6.

**Files:** Modify `message.go` (`TurnOpenedPayload.Action`, `OpenTurnPayload.ActionID/Action`), `room.go` (`announceOpenedTurn`, `broadcastTurnClosed`, `buildMatchFullState`, helper `viewerFor`); Test: `turn_opened_e2e_test.go` (novo); Modify `match-combat-ws.md`.

**Interfaces:**
- Produces: `TurnOpenedPayload.Action actionwire.Action \`json:"action"\``; `OpenTurnPayload{TurnID, ActorID, ActionID uuid.UUID; Action actionwire.Action}`; `func (r *Room) viewerFor(playerID uuid.UUID, isMaster bool) domainservice.Viewer` (caller holds `r.mu` — monta `Owns` de `session.GetCharToPlayer()`, o mesmo código hoje repetido em `buildMatchFullState` e `publishResolution`; troque os dois para usá-lo).

- [ ] **Step 1: Testes e2e (falham)** com `withBystander`:
  - mestre: `turn_opened.action.attack.hit.result != nil`;
  - dono (quem enfileirou): `action.attack.weapon == "Sword"`, `action.targetId` presente, `action.speed.rollCheck.result != nil`, **`action.attack.hit.result == nil`** e `action.attack.damage.result == nil`;
  - bystander: igual ao dono;
  - uma ação com `feint`: o bystander **não** recebe `feint` (deny-list com turno aberto); o dono recebe `feint` só com `skillName`;
  - `match_full_state.openTurn` de quem reconecta com turno aberto: tem `actionId` e o mesmo `action` que o `turn_opened` daquela pessoa.
- [ ] **Step 2: Teste de ordem (falha se a pista estiver errada)**: com duas ações na fila, `open_next_action` duas vezes; no coletor do bystander, o `turn_closed` do primeiro turno vem **antes** do `turn_opened` do segundo, em 50 repetições (`for i := 0; i < 50; i++` dentro do teste, fixture nova por iteração só se for barato; senão 50 pares de abertura na mesma fixture).
- [ ] **Step 3: Implementar.** `announceOpenedTurn`: troque o `go func() { r.broadcast <- data }()` por `r.dispatchPerPlayer(func(pid, isMaster) *Message { ... })`, que sob `r.mu.RLock` monta o viewer e a ação (`GetAction()` é cópia — atribua a variável antes), e fora do lock devolve a mensagem. Master: `From(act, Full)`; outros: `From(domainservice.ProjectAction(act, v, false), Opened)`. `broadcastTurnClosed` passa a usar `dispatchPerPlayer` também (mesma mensagem para todos). Reescreva os dois doc comments: a razão da pista mudou (a ordem entre as duas agora é garantida por estarem na mesma pista, no mesmo goroutine). `buildMatchFullState`: `OpenTurn` ganha `ActionID` e `Action` com a mesma regra.
- [ ] **Step 4: Contrato.** `turn_opened`: o campo `action`, a tabela de cortes do spec §4.1 (colunas mestre / todos os outros, inclusive o dono), exemplo para cada; a **ordem** garantida `piece_moved → turn_closed → resolution_updated → turn_opened`; `match_full_state.openTurn` com os dois campos. Revise o texto de `match-history.md` que diz *"nenhuma mensagem servidor→cliente projeta a declaração de uma action de jogador"* — deixou de ser verdade.
- [ ] **Step 5: Rodar** verificação + `-race` → PASS. **Commit** `feat(game): turn_opened carrega a mecânica pública da ação (B2)`

---

### Task 9: B11 — NPC do mapa é inscrito quando a sessão nasce

Spec §4.3 "B11". Depende de T2/T3 (o tabuleiro existe no banco).

**Files:** Modify `internal/application/match/init_match_session.go`, `init_match_session_test.go`; `cmd/game/main.go` (novas deps do `InitMatchSessionUC`); `docs/dev/api/match-npcs.md`.

**Interfaces:**
- Consumes: `matchboarduc.ILoadMatchBoard` (T2); `*AddMatchNPCUC.Add` (existente) por uma interface local `INPCEnroller{ Add(ctx, *AddMatchNPCInput) (*matchEntity.Participant, error) }`; `IRepository.GetMatch` para o `MasterUUID`.
- Produces: `NewInitMatchSessionUC(matchRepo IRepository, sheetLoader ICharSheetLoader, roundRepo IRoundRepository, board IBoardReader, npcs INPCEnroller) *InitMatchSessionUC` — `board`/`npcs` nil = comportamento de hoje (os testes antigos passam nil).

- [ ] **Step 1: Testes unitários (falham)**, table-driven com fakes:
  - peça de NPC do mestre que não é participante → `Add` chamado com `RequesterUUID = master`, e o NPC está nos `charSheets` da sessão devolvida;
  - `Add` → `ErrNPCAlreadyInMatch` → segue normal;
  - `Add` → `ErrSheetNotNPC` (peça de jogador não inscrito) → pulado, sessão criada, o personagem **não** entra (review focus 3);
  - `Add` → `ErrCharacterSheetNotFound` / `ErrSheetNotOwnedByMaster` → pulado com log;
  - peça de quem já é participante → `Add` **não** é chamado;
  - `CharacterID` que não é UUID → pulado;
  - sem tabuleiro (`nil, nil`) → como hoje;
  - chamar `Init` duas vezes sobre o mesmo estado → mesmo resultado (idempotente).
- [ ] **Step 2: Implementar**: antes de `ListParticipantsByMatchUUID`, se `board != nil`: `GetMatch` (mestre), `Load` do tabuleiro, lista de participantes (uma leitura) para saber quem já está, e `Add` para cada personagem de peça que não está. Erros do tabuleiro → log e segue (a sessão não pode deixar de nascer por isso). Depois, lista os participantes de novo (agora com os NPCs).
- [ ] **Step 3: Wiring** em `cmd/game/main.go` (o `addMatchNPCUC` já existe lá; passe-o).
- [ ] **Step 4: e2e** `TestE2E_StartingTheMatchEnrollsTheNPCsOnTheBoard` usando o `InitMatchSessionUC` **real** com fakes de repositório (não o `combatSessionUC`): lobby com peça de NPC → `start_match` → o NPC aparece no `bars_updated`/`match_full_state.bars.characters`. É o caminho real que a verificação da Fase 6 nunca exercitou.
- [ ] **Step 5: Contrato** `match-npcs.md`: NPC com peça no mapa entra na partida no início e ao reidratar, de forma idempotente; a peça de jogador não inscrito é ignorada.
- [ ] **Step 6: Rodar** verificação → PASS. **Commit** `feat(match): NPC no mapa da partida é NPC da partida (B11)`

---

### Task 10: B12 — o dono reconcilia a própria fila

Spec §4.2 B12. Depende de T6.

**Files:** Modify `message.go` (`OwnQueuedActionPayload`, `MatchFullStatePayload.OwnQueue`), `room.go` (`buildMatchFullState`); Test: `own_queue_e2e_test.go`; Modify `match-combat-ws.md`.

**Interfaces:** Produces `type OwnQueuedActionPayload struct{ ActionID uuid.UUID \`json:"actionId"\`; Action actionwire.Action \`json:"action"\` }`; `MatchFullStatePayload.OwnQueue []OwnQueuedActionPayload \`json:"ownQueue"\`` (**sem** `omitempty`; `nil` para o mestre vira `null` — por isso o mestre recebe o campo ausente: use um tipo ponteiro-para-slice `*[]OwnQueuedActionPayload` com `omitempty`, `nil` para o mestre e `&[]{}`/`&lista` para os outros).

- [ ] **Step 1: Testes e2e (falham)**, `withBystander`:
  - jogador enfileira duas ações e reconecta → `ownQueue` tem as duas, na ordem da fila, com `action.attack.weapon` e **sem** `action.speed.rollCheck.result` e sem `action.move.finalSpeed`;
  - o bystander reconecta → `"ownQueue": []` (presente e vazio) — confira no JSON cru;
  - o mestre reconecta → sem `ownQueue` no JSON, e com `queue`;
  - depois de o mestre abrir uma das duas, o dono reconecta → `ownQueue` com uma, e `openTurn.actionId` é a outra;
  - `f.restart(t)` (helper de T3) com ações na fila → o dono reconecta e recebe `"ownQueue": []`.
- [ ] **Step 2: Implementar** em `buildMatchFullState`: para `!isMaster`, `session.PendingActions()` filtrado por `charToPlayer[a.GetActorID().String()] == playerID`, `From(*a, Declaration)`.
- [ ] **Step 3: Contrato**: o campo, o porquê de estar sempre presente, e a **regra de reconciliação** do spec (conhecida = em `ownQueue` ou é `openTurn.actionId`; o resto sai com aviso e o rascunho volta; **nunca reenviar sozinho** — reenviar re-sorteia). Tabela de reinício (linha "fila").
- [ ] **Step 4: Rodar** verificação + `-race` → PASS. **Commit** `feat(game): match_full_state manda ao dono a fila dele (B12)`

---

### Task 11: B13 — o escape é esquiva **e** movimento (motor)

Spec §4.4 "Motor". Puro domínio.

**Files:** Modify `internal/domain/match/service/reaction_collision.go`, `turn_resolver.go` (`CharacterResult.Escape`); Test: `reaction_collision_test.go` (existente, acrescente) e `turn_resolver_test.go`.

**Interfaces:** Produces:

```go
// EscapeResult is how an escape came out. nil on every reaction that does not displace.
type EscapeResult struct {
	MovePassed  bool // Move.FinalSpeed >= the attacker's hit
	DodgePassed bool // Dodge.Total >= the attacker's hit
	Escaped     bool // both — the only way an escape avoids the blow and reaches its slot
	// Landing is where the MASTER put the piece of an escape that failed (edit_action,
	// escapeLanding). nil = not chosen; the piece stays where it stood. Filled by the
	// resolver from the turn, never by ResolveReaction.
	Landing *[3]int
}
```

`ReactionOutcome.Escape *EscapeResult`; `CharacterResult.Escape *EscapeResult`.

- [ ] **Step 1: Testes (falham)**, table-driven sobre os três escapes × (esquiva passa/falha) × (movimento passa/falha), com `HitTotal` fixo e `Move.FinalSpeed`/dados escolhidos para cada caso:

| kind | dodge | move | `Avoided` | `Defended` possível | `Escape.Escaped` |
|---|---|---|---|---|---|
| `escape` | ✔ | ✔ | true | — | true |
| `escape` | ✔ | ✘ | **false** | não (golpe inteiro) | false |
| `escape` | ✘ | ✔ | false | não | false |
| `escapeGuard` | ✔ | ✘ | false | **sim** (cai para a defesa) | false |
| `closedEscape` (Shift, Brake passivo) | ✔ | ✘ | false | não | false |
| `dodge` | ✔ | — | true | — | `Escape == nil` |

  E: escape com `Move == nil` (não deveria chegar, mas) → `MovePassed = false`.
- [ ] **Step 2: Implementar** em `ResolveReaction`, no ramo da família da esquiva:

```go
	dodgePassed := out.Dodge.Total >= in.HitTotal
	out.Avoided = dodgePassed
	if in.Kind.Displaces() {
		// An escape is a dodge that moves, and the move is its own test against the same CD:
		// the attacker's hit. Both have to clear it (front-combat-phases.md §6A.5, B13) — the
		// user cannot get out of the way if the step fails. Failing either one is not escaping,
		// and the blow is read as if they had stayed: escapeGuard still falls back on the
		// defense below, the others take it whole.
		//
		// TODO(collision): the known rule is that the movement ADDS to the dodge, which makes
		// escaping easier than dodging in place. It is part of the collision design that does
		// not exist yet — when it does, the sum goes into out.Dodge.Total right here, before
		// either comparison.
		movePassed := in.Reaction != nil && in.Reaction.Move != nil && in.Reaction.Move.FinalSpeed >= in.HitTotal
		out.Escape = &EscapeResult{MovePassed: movePassed, DodgePassed: dodgePassed, Escaped: dodgePassed && movePassed}
		out.Avoided = out.Escape.Escaped
	}
	if out.Avoided {
		return out
	}
```

  (O resto do fluxo — `KeepsDefault` e a defesa — fica como está.) Em `resolveCharacterStep` (`turn_resolver.go` ~515), copie `cr.Escape = out.Escape`.
- [ ] **Step 3: Rodar** `go test ./internal/domain/match/...` → PASS (os testes existentes de escape que esperavam `Avoided` com movimento falhado mudam: ajuste-os dizendo no nome o que a regra nova diz).
- [ ] **Step 4: Commit** `feat(match): escapar exige esquiva e movimento (B13)`

---

### Task 12: B13 — a peça do escape e a escolha do mestre

Spec §4.4 "A escolha do mestre" e "Tabuleiro". Depende de T11 (e de T3 para persistir).

**Files:** Modify `internal/domain/match/entity/turn/turn.go` (landings), `internal/domain/match/matchsession/match_session.go` (`ResolveTurn` passa as landings; `SetEscapeLanding`), `internal/domain/match/service/turn_resolver.go` (input com landings), `internal/app/game/message.go` (`EditActionPayload.EscapeLanding`, `CharacterResultPayload.Escape`), `action_mapper.go` (`buildEditAction`), `internal/application/match/edit_action.go`, `room.go` (`open_reaction` sem deslocar; `applyClosedEscapes`), `internal/gateway/pg/round/resolution_record.go`, `internal/app/api/match/get_match_history.go` (`CharacterResultResponse.Escape`); Tests: `escape_e2e_test.go` (novo), testes de turno/sessão, integração do `resolution_record`; Modify `match-combat-ws.md`, `match-history.md`, `docs/dev/match/combat-engine.md`.

**Interfaces:**
- Produces:
  - `func (t *Turn) SetEscapeLanding(reactionID uuid.UUID, pos [3]int)`, `ClearEscapeLanding(reactionID)`, `EscapeLandings() map[uuid.UUID][3]int` (cópia)
  - `func (s *MatchSession) SetEscapeLanding(reactionID uuid.UUID, pos *[3]int) (*service.TurnResolution, error)` — erros `ErrNoActiveTurn`, `ErrNotAnEscape` (novo em `matchsession/error.go`), `ErrLandingOutOfGrid` (novo)
  - `type EscapeLandingPayload struct{ Position *[3]int \`json:"position"\` }`; `EditActionPayload.EscapeLanding *EscapeLandingPayload \`json:"escapeLanding,omitempty"\``
  - `type EscapeResultPayload struct{ Escaped, MovePassed, DodgePassed, AwaitsMaster bool; Landing *[3]int }` (tags camelCase, `landing` com `omitempty`); `CharacterResultPayload.Escape *EscapeResultPayload \`json:"escape,omitempty"\``

- [ ] **Step 1: Testes de domínio (falham):** `Turn` guarda/limpa/copia landings; `SetEscapeLanding` recusa reação que não desloca (`ErrNotAnEscape`), recusa posição fora de `grid.Cols/Rows` quando a grade tem dimensões (`ErrLandingOutOfGrid`), aceita `nil` para limpar, e devolve a resolução com `Escape.Landing` preenchido para aquele alvo; `ResolveTurn` preenche `Landing` **só** quando `!Escaped`.
- [ ] **Step 2: Implementar o domínio.** O `TurnResolver` recebe as landings no input que já recebe o turno (procure a struct de input de `ResolveTurn` em `turn_resolver.go`) e, ao montar cada `CharacterResult` com `Escape != nil && !Escape.Escaped`, copia `landings[cr.ReactionID]`.
- [ ] **Step 3: `edit_action`.** `buildEditAction` passa a aceitar um payload só com `escapeLanding` (hoje ele pode exigir condições — confira e ajuste para "pelo menos uma seção"); `EditActionUC` chama `session.SetEscapeLanding` quando `EscapeLanding != nil` (antes ou depois de `ApplyMasterAction`, e se o payload tiver as duas coisas, as duas valem). Para levar o dado até o UC sem inventar campo em `MasterAction`, acrescente ao `EditActionUC.Execute` um parâmetro `landing *EscapeLandingEdit` (`struct{ ReactionID uuid.UUID; Position *[3]int }`) — atualize `IEditAction` e o mock.
- [ ] **Step 4: Testes e2e (falham)** em `escape_e2e_test.go`, com `withVictimPiece` e dados roteirizados (`scriptedFaces`):
  - `open_reaction` de **qualquer** escape (Dash e Shift) não emite `piece_moved`;
  - escape que passa nos dois → no fechamento, `piece_moved` para o destino;
  - escape que falha no movimento, sem escolha → no fechamento, **nenhum** `piece_moved`; a `resolution_updated` do mestre com turno aberto tem `escape.awaitsMaster: true`;
  - mesmo caso, com `edit_action {actionId: <reactionId>, escapeLanding: {position: [7,6,0]}}` → o mestre recebe `action_edited` e uma `resolution_updated` com `escape.landing = [7,6,0]`, `awaitsMaster: false`; no fechamento, `piece_moved` para (7,6);
  - escolha feita e depois o escape **passa** (o mestre muda a leitura — use um segundo alvo/reação ou `edit_action` de condição sobre `moveSpeed` que faça o movimento passar) → a peça vai ao **destino** (review focus 4);
  - `edit_action` com `escapeLanding` sobre reação que não é escape → `game_error`;
  - os três verbos que fecham decidem igual (table-driven pelo verbo, como T3).
  - O `resolution_updated` settled chega a todos com `escape` (números públicos depois do fechamento).
- [ ] **Step 5: Implementar na `Room`.** No braço `open_reaction`, remova o ramo que desloca o Shift (`reactionMove`/`applyMove` do ~843–860) e reescreva o comentário: nenhum escape desloca na abertura, porque pode falhar (§6A.5 B13). `applyClosedEscapes`:

```go
		switch {
		case cr.Escape != nil && cr.Escape.Escaped:
			r.applyMove(reaction.GetActorID(), reaction.Move)
		case cr.Escape != nil && cr.Escape.Landing != nil:
			// The escape failed and the master decided where the piece ended up, as part of
			// resolving this turn — not a drag (front-combat-phases.md §6A.5, B13). This is the
			// branch the definitive collision design will replace: where a failed escape lands
			// is a rule that does not exist yet.
			landing := *reaction.Move
			landing.Position = *cr.Escape.Landing
			r.applyMove(reaction.GetActorID(), &landing)
		default:
			// Failed with no choice made: the piece stays where it stood. Same pointer as above
			// for the definitive design.
		}
```

  Remova o filtro `RollsSpeed()` (agora todo escape espera). Reescreva os doc comments de `applyClosedEscapes` e `applyMove` que descrevem a regra antiga (Shift na abertura, Dash no fechamento).
- [ ] **Step 6: Wire e persistência.** `newResolutionUpdatedPayload` mapeia `Escape` (com `AwaitsMaster = !Escaped && Landing == nil`). `resolution_record.go`: `characterResultRecord` ganha `Escape *escapeRecord \`json:"escape,omitempty"\`` (ida e volta); teste de integração do registro com escape. REST do histórico: `CharacterResultResponse.Escape` com o mesmo formato do WS.
- [ ] **Step 7: Contratos.** `match-combat-ws.md`: reescrever "O deslocamento de uma fuga tem CD…" e "O fechamento é onde as fugas de Dash são decididas" para a regra única (esquiva **e** movimento; tabela da peça: passou → destino; falhou → escolha do mestre ou fica); **tirar** *"Dano e deslocamento são desfechos INDEPENDENTES"*; `edit_action.escapeLanding` (definir, limpar, quando vale); `escape` em `resolution_updated`. `match-history.md`: `escape` em `targets[]`. `combat-engine.md`: a regra do escape.
- [ ] **Step 8: Rodar** verificação + `-race` + integração → PASS. **Commit** `feat(game): o escape espera o fechamento e a queda é do mestre (B13)`

---

### Task 13: B15 — o histórico guarda o que não é turno, e as master actions

Spec §4.5 e **§4.8**. Depende de T1 e T5 (`EnsureSceneAndRound`, gravação de master actions).

**Files:** Modify `room.go` (`StartMatch`, `RehydrateSession`, braço `change_scene`, braço `change_round_mode`, round fechado por exaustão), `internal/gateway/pg/round/find_match_history.go`, `internal/application/match/i_repository.go` (`HistoryRound.Events`, `HistoryTurn.MasterActions`), `internal/application/match/get_match_history.go` (UC: costura e projeção), `internal/app/api/match/get_match_history.go` (DTOs), `room_deps.go` (`EventRepo`), `cmd/game/main.go`, `cmd/api/main.go` (o UC do histórico ganha os dois repositórios); Tests: integração do histórico, UC, e2e; Modify `match-history.md`.

**Interfaces:**
- Consumes: `EnsureSceneAndRound` (T5), `pgmatchevent.Insert/ListByMatch` (T1), `pgmasteraction.ListByMatch` (T1), `masteraction.Record.ProjectFor` (T1).
- Produces:
  - `HistoryTurn.MasterActions []masteraction.Record` (já projetadas, ordem de `HappenedAt`)
  - `HistoryRound.Events []HistoryEvent` com `HistoryEvent{Kind string /* "roundModeChanged" | "masterAction" */; At time.Time; RoundModeChange *matchevent.Event; MasterAction *masteraction.Record}`, em ordem de `At`
  - DTOs: `HistoryMasterActionResponse{UUID uuid.UUID; Kind string; TurnID *uuid.UUID \`json:"turnId,omitempty"\`; HappenedAt string; Content json.RawMessage}`; `HistoryEventResponse{UUID uuid.UUID; Kind string; CreatedAt string; Payload json.RawMessage \`json:"payload,omitempty"\`; MasterAction *HistoryMasterActionResponse \`json:"masterAction,omitempty"\`}`; `HistoryTurnResponse.MasterActions []HistoryMasterActionResponse \`json:"masterActions"\``; `HistoryRoundResponse.Events []HistoryEventResponse \`json:"events"\`` — os dois **sempre** `[]`, nunca `null`
  - `NewGetMatchHistoryUC(matchRepo, roundRepo, participationChecker, events IEventLister, masterActions IMasterActionLister)`
  - `RoomDeps.EventRepo interface{ Insert(ctx context.Context, e matchevent.Event) error }`

- [ ] **Step 1: Teste de integração (falha)** em `round_integration_test.go`: partida com uma cena sem turno, um round fechado sem turno e um round com turno → `FindMatchHistory` devolve as três coisas (hoje o `JOIN` interno some com as duas primeiras). Um round com um `turn` sem `actions` não pode quebrar a montagem.
- [ ] **Step 2: Implementar** o `LEFT JOIN` (`rounds`, `turns`, `actions`), com as colunas de turno/ação nulas lidas em ponteiros, e a montagem que só cria turno/ação quando o UUID não é nulo. Mantenha a ordenação e os desempates (o comentário do topo explica por quê). **Reescreva** o parágrafo do topo que diz *"The master's own actions … are deliberately never persisted"*: agora são, em tabela própria, e a costura é do UC (Step 5).
- [ ] **Step 3: Persistir cena e round quando nascem.** Chame `EnsureSceneAndRound` (fora do lock, com os valores lidos sob o lock) em: `StartMatch` e `RehydrateSession` (os ativos); depois do `change_scene` (a cena e o round novos); depois de um `round_closed` por exaustão, para o round novo que a sessão abriu (confira em `OpenNextActionResult` como o round novo fica acessível — `session.GetActiveRound()` sob `r.mu`). Com isso, o `if sceneWasPersisted` do `change_scene` passa a ser sempre verdadeiro — mantenha o `if` (é defensivo) mas atualize o comentário.
- [ ] **Step 4: Evento de regime.** No braço `change_round_mode`, com sucesso: `EnsureSceneAndRound` + `EventRepo.Insert(Event{Kind: roundModeChanged, Payload: {"from": old, "to": new}})` — leia o `old` antes do `Execute`, sob o mesmo lock.
- [ ] **Step 5: Costura e projeção no UC (testes primeiro, falham).** `GetMatchHistoryUC` lê a árvore, os eventos e as master actions da partida. Para cada master action: `ProjectFor(viewerIsMaster, userUUID)`; `false` → descartada. Se `TurnUUID` aponta um turno **presente na árvore** → `HistoryTurn.MasterActions`; senão (sem turno, ou turno que não foi gravado) → `HistoryEvent{Kind: "masterAction"}` do round (`RoundUUID`). Eventos de regime → `HistoryEvent{Kind: "roundModeChanged"}` do round. Round que não está na árvore → log e descarte. Ordene `Events` por `At` e `MasterActions` por `HappenedAt`. Testes do UC, table-driven:
  - o mestre vê todas as master actions, dentro e fora do turno, inteiras; o DTO nunca serializa `Views`, para ninguém (confira no teste do handler);
  - jogador com `full` vê a entrada inteira; com `left` vê o `movePiece` sem `to`; sem entrada não vê nada;
  - master action com `TurnUUID` de um turno que **não** está na árvore (perdido num reinício) → aparece em `events` do round, com `kind: "masterAction"` (review focus 6);
  - `roundModeChanged` e `masterAction` fora de turno do mesmo round saem intercalados na ordem do tempo;
  - `edit_action` não aparece em lugar nenhum (não há `Record` para ele — o teste é de regressão: montar um turno com override e conferir que `masterActions` é `[]`).
- [ ] **Step 6: e2e** `TestE2E_TheHistoryKeepsWhatIsNotATurn` (com os repositórios fake de eventos e de master actions): trocar o regime e trocar a cena sem nenhum turno → os fakes registraram `EnsureSceneAndRound` para as duas cenas e o evento de regime; arrastar entre turnos → um `Record` sem `TurnUUID`.
- [ ] **Step 7: Contrato `match-history.md`:** `events` por round (formato, os dois `kind`, ordem do tempo); `turns[].masterActions`; a regra de onde a master action entra (com turno gravado → dentro dele; senão → `events`); a **projeção** (cada leitor vê como viu ao vivo — tabela `full`/`left`/ausente do spec §4.8; o mestre vê tudo; `views` nunca sai no wire); cena/round sem turno aparecem; `edit_action` não é master action (fica em `overridden_action_values`); **tirar** a frase de que as master actions nunca são persistidas e a de que *"nenhuma mensagem servidor→cliente projeta a declaração de uma action de jogador"* se T8 ainda não o fez; **consertar os exemplos** (`category: "battle"`, `mode: "Race"`) e **mostrar um `move`** completo (`category`, `from`, `position`, `speed`, `finalSpeed`) e uma master action de cada lugar.
- [ ] **Step 8: Rodar** verificação + integração → PASS. **Commit** `feat(history): cena, round, eventos e master actions no histórico (B15)`

---

### Task 14: B4 e B7 — a conexão

Spec §4.6. Depende de T0.

**Files:** Modify `room.go` (`Run` register; `done`; `Register`), `client.go` (`ReadPump` unregister com `select`), `handler.go` (retry), `hub.go` se preciso; Test: `connection_e2e_test.go` (novo); Modify `match-combat-ws.md` (`connection_replaced`), `game-lobby.md`.

**Interfaces:** Produces `func (r *Room) Register(client *Client) error` (antes sem retorno); `ErrRoomClosed` (já existe) devolvido quando `done` fechou.

- [ ] **Step 1: Testes (falham):**
  - o mesmo jogador conecta duas vezes → a primeira conexão recebe `error` `connection_replaced` e é fechada pelo servidor; a segunda recebe o próximo broadcast (o mestre manda `chat`);
  - a velha fecha **depois** → a nova continua na sala e recebe outro `chat` (review focus 5); a sala não fecha;
  - `Register` numa sala cujo `Run` já retornou → volta `ErrRoomClosed` em menos de 1s (use `time.After` no teste; hoje trava);
  - o `ReadPump` de um cliente cuja sala fechou termina sem travar (o goroutine sai — use um canal de "saiu" no teste ou `goleak`-like simples contando goroutines antes/depois com tolerância; prefira o canal: exponha em `export_test.go` se preciso).
- [ ] **Step 2: Implementar.** `Room.done chan struct{}` criado no `NewRoom`, fechado com `defer close(r.done)` no topo do `Run`. `Register`: `select { case r.register <- c: return nil; case <-r.done: return ErrRoomClosed }`. `ReadPump`: `select { case c.room.unregister <- c: case <-c.room.done: }` (o cliente precisa enxergar `done` — `c.room` já é `*Room`). No braço `register` do `Run`: se `old, ok := r.clients[client.userUUID]; ok && old != client`, mande `NewErrorMessage("connection_replaced", "this account connected again elsewhere")` ao velho e `old.Close()`, **antes** de gravar o novo. O guarda de ponteiro do `unregister` fica. `handler.go`: `if err := room.Register(client); errors.Is(err, ErrRoomClosed)` → mestre: `GetOrCreateRoom` de novo e `Register` uma vez mais; jogador: o caminho de `lobby_not_open`.
- [ ] **Step 3: Contrato:** `connection_replaced` (quem recebe, o que o cliente faz: não reconectar sozinho — a outra aba é a que vale).
- [ ] **Step 4: Rodar** verificação + `-race` → PASS. **Commit** `fix(game): a última conexão vence e sala fechada não trava (B4, B7)`

---

### Task 15: B5, B6 e B10 — a geometria do movimento

Spec §4.3 "B5, B6 e B10". Depende de T2 (servidor sabe a posição da peça).

**Files:** Modify `internal/domain/match/entity/action/move.go` (`From *[3]int`), `action_mapper.go` (ignora `from` do payload), `room.go` (braço `enqueue_action`: origem da peça, centros, grade), `actionwire/from.go` e `action.go` (`Move.From *[3]int`, `omitempty`), `internal/gateway/pg/round/*` onde `Move` é serializado (confira o formato do JSON de `actions.move` — se é `json.Marshal(action.Move)` direto, o campo `From` passa a sair `null` quando nil; garanta que linhas antigas `[0,0,0]` continuam decodificando); Tests: `movement_geometry_e2e_test.go` (novo), `action_mapper_test.go`, ouro do histórico (T6) atualizado **só** no campo `from` se mudar; Modify `match-combat-ws.md`, `match-history.md`.

**Interfaces:** Produces `action.Move.From *[3]int`; `func (r *Room) pieceSlotOf(characterID string) (*[3]int, bool)` (caller holds `r.mu`; `[a, b, 0]`, com `a,b` = col,row ou q,r).

- [ ] **Step 1: Testes (falham):**
  - grade quadrada, parede vertical entre as colunas 4 e 5 **passando pelos cantos** dos slots: Dash de (4,4) para (5,4) é **bloqueado** (os centros cruzam a parede) e Dash que só roça o canto não é (monte a parede para o caso — use `mapservice.SlotCenterToWorld` no teste para posicioná-la);
  - peça no slot (0,0) andando para (0,2) com parede entre elas → **bloqueado** (hoje o sentinela `[0,0,0]` pula a checagem — B6);
  - o `from` do payload é ignorado: payload diz `from: [9,9,0]`, a peça está em (4,4) → a checagem usa (4,4) e o `action_queued.action.move.from` é `[4,4,0]`;
  - ator sem peça → nenhuma checagem, `move.from` ausente;
  - grade hexagonal (`GridKindHex`): parede entre `(q,r)=(2,1)` e `(3,1)` bloqueia o passo entre os dois; e o `applyMove` de um hex escreve `q`/`r` (B10 — já existe um caso? confira `combat_e2e_test.go`; se não, acrescente).
- [ ] **Step 2: Implementar.** No braço `enqueue_action`, sob `r.mu.RLock`: `from, hasPiece := r.pieceSlotOf(a.GetActorID().String())`; `a.Move.From = from` (nil sem peça); se `hasPiece`, `fromWorld := SlotCenterToWorld(from[0], from[1], grid)` e idem para `to`, com `grid := session.GetGrid()`; `IsPathBlocked(fromWorld, toWorld, walls)`. Comente a convenção `[a, b, z]` no `MovePayload` e em `action.Move`.
- [ ] **Step 3: Contratos:** a convenção de coordenada (`(col,row)` quadrada, `(q,r)` axial hex, `z` não lido); `move.from` **ignorado** na entrada e **derivado** na saída; checagem de parede pelos centros.
- [ ] **Step 4: Rodar** verificação + `-race` + integração → PASS. **Commit** `fix(game): o movimento sai da peça e é checado pelos centros (B5, B6, B10)`

---

### Task 16: B8 — o dono recebe o próprio `private`

Spec §4.7. Independente.

**Files:** Modify `internal/app/api/match/get_match_participants.go`, `internal/application/match/get_match_participants.go` (resultado carrega `ViewerUUID`); Tests: `get_match_participants_test.go` (UC e handler); Modify `docs/dev/api/match.md`.

- [ ] **Step 1: Teste do handler (falha)**, table-driven: mestre → `private` em todos; jogador dono → `private` só na ficha dele; outro jogador → nenhum; NPC → só o mestre.
- [ ] **Step 2: Implementar**: `GetMatchParticipantsResult.ViewerUUID`; `toParticipantResponse(p, viewerIsMaster, viewerUUID)` com `viewerIsMaster || (p.Sheet.PlayerUUID != nil && *p.Sheet.PlayerUUID == viewerUUID)`.
- [ ] **Step 3: Contrato** `match.md` (participantes): quem recebe `private`.
- [ ] **Step 4: Rodar** verificação → PASS. **Commit** `fix(match): o dono recebe o próprio private nos participantes (B8)`

---

### Task 17: B9 — o `attack` da master action (L1)

Spec §2 L1. **Pendente da resposta do autor do documento mestre.** Com a recomendação:

**Files:** Modify `message.go` (`MasterActionPayload`, `MasterActionEnqueuedPayload` sem `Attack`), `action_mapper.go` (sai o ramo `Attack` e seu `TODO` — o TODO sai porque a decisão foi tomada: diga isso no commit), `action/master_action.go` (o campo `Attack` do domínio **fica**: ele é usado pelo `edit_action`? confira; se não for usado por nada, fica também — é domínio, não wire); Test: `action_mapper_test.go`, e2e; Modify `match-combat-ws.md`.

- [ ] **Step 1: Teste (falha):** `enqueue_master_action` com `attack` → `error` `invalid_action` `"the master attacks through an NPC with enqueue_action"` (o JSON desconhecido não pode passar calado: decodifique para detectar a chave `attack` e recuse).
- [ ] **Step 2: Implementar.**
- [ ] **Step 3: Contrato:** o mestre ataca pelo NPC com `enqueue_action`; ataque "do ambiente" não existe ainda.
- [ ] **Step 4: Rodar** verificação → PASS. **Commit** `fix(game): o mestre ataca pelo NPC, não por master action (B9)`

Se a resposta do autor for outra, **pare** e peça o desenho antes desta tarefa.

---

### Task 18: B16 — começar de onde outra partida terminou

Spec §4.3 "Estrutura" (B16). Depende de T1/T3.

**Files:** Modify `internal/gateway/pg/matchboard/` (`Copy`), `internal/gateway/pg/fog/` (`CopyMatch(ctx, src, dst)`), `internal/application/matchmap/attach_match_map.go` (+ input `InheritBoardFromMatchUUID *uuid.UUID`; apagar tabuleiro velho ao trocar de mapa), `internal/app/api/matchmap/*` (request), `cmd/api/main.go` (deps); Tests: integração de `Copy`, UC, handler (`humatest`); Modify `docs/dev/api/match-maps.md`.

**Interfaces:** Produces `pgmatchboard.Repository.Copy(ctx, src, dst uuid.UUID) error` (transação: apaga o `dst`, `INSERT … SELECT` da linha com `match_uuid = dst`, `INSERT … SELECT` de `player_memories` com `match_id = dst` e `id = gen_random_uuid()`); erros novos em `matchmapuc`: `ErrSourceMatchNotInCampaign`, `ErrSourceMatchOnAnotherMap`, `ErrSourceMatchHasNoBoard`.

- [ ] **Step 1: Teste de integração (falha)** de `Copy`: origem com peças e duas memórias → destino idêntico (menos `match_uuid`/ids); destino pré-existente é substituído.
- [ ] **Step 2: Implementar `Copy`.**
- [ ] **Step 3: Testes do UC (falham)**, table-driven: herança válida (mesma campanha, mesmo mapa, origem com tabuleiro) → `Copy` chamado; origem de outra campanha / outro mapa / sem tabuleiro → os três erros; partida já iniciada → `ErrMatchAlreadyStarted` (já existe); anexar **outro** mapa sem herança → `Delete` do tabuleiro velho; anexar o **mesmo** mapa sem herança → tabuleiro fica.
- [ ] **Step 4: Implementar o UC e o handler** (campo opcional `inheritBoardFromMatchUuid` no body; erros → 409/422 no padrão de `match-maps.md` — confira os códigos que o arquivo já usa).
- [ ] **Step 5: Contrato** `match-maps.md`: o campo, as regras, os erros, exemplo; "anexar outro mapa apaga o tabuleiro da partida".
- [ ] **Step 6: Rodar** verificação + integração → PASS. **Commit** `feat(matchmap): uma partida pode herdar o tabuleiro de outra (B16)`

---

### Task 19: Documentação transversal e entrega

**Files:** `AGENTS.md` (Known Issues), `docs/dev/match/combat-engine.md`, `docs/documentation-map.yaml` (revisão final), `.github/instructions/game-server.instructions.md` (o tabuleiro agora é carregado do banco; `RoomDeps`; `persistBoard`).

- [ ] **Step 1: Docs.** `AGENTS.md`: tirar/atualizar o que deixou de ser verdade (o `fog_mode` continua pendente — **não** remover aquele bloco); acrescentar o tabuleiro por partida, o escape e as master actions persistidas em tabela própria (e por que não em `actions`). `game-server.instructions.md`: `RoomDeps`, o tabuleiro vem do banco, `persistBoard` é o único ponto de gravação, as duas pistas de envio e a ordem nova. Rode o mapa de documentação contra `git diff --name-only origin/main` e classifique (covered/missing/unmapped).
- [ ] **Step 2: Verificação completa**: `go build ./... && go vet ./... && go vet -tags integration ./... && go vet -tags smoke ./... && go test ./... && go test -race ./internal/app/game/ && go test -tags=integration -p 1 ./internal/gateway/pg/...` → tudo PASS. Cole a saída resumida no PR.
- [ ] **Step 3: Ponta a ponta** (`CLAUDE.md` da raiz, "Entrega"): `make migrate-up`, `make run-dev`; `curl` em `GET /matches/{uuid}/history` (cena sem turno, `events`), `GET /matches/{uuid}/participants` como dono, `PUT` do anexo com `inheritBoardFromMatchUuid`. WS com um cliente de linha de comando (`websocat` se instalado; senão um `go run` descartável no scratchpad — **não** commitar): mestre + dois jogadores, uma ação com Dash, abrir, fechar; **reiniciar o `cmd/game` no meio de uma fila e no meio de um turno** e conferir `map_full_state`, `ownQueue: []` e as posições. Registre o que foi feito e o que não foi.
- [ ] **Step 4: PR** (depois de `git rebase origin/main`): título `feat: fechamento da Fase 6 — pacote de back (B1–B16)`; descrição com o que cada B entrega, os contratos mudados (lista do spec §6), o que foi verificado, **o que não foi** (browser com três contas é do PR de front), o link para o PR de front, e L1–L2 se ainda abertos. Termina com `🤖 Generated with [Claude Code](https://claude.com/claude-code)`.
- [ ] **Step 5: Ambiente para validação manual:** `./dev-checkout.sh feat/combat-closure-back` a partir de `System_X_System_Project/` (só no fim; dizer no PR o que olhar).
