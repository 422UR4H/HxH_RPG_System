# Memória de exploração por parede (backend) — Plano de Implementação

> **Para quem implementa:** execute tarefa por tarefa, na ordem. Cada passo é uma ação
> de 2–5 minutos. Não pule os passos de "rodar e ver falhar".

**Spec de referência (leia antes de começar):**
`docs/superpowers/specs/2026-08-05-tactical-map-wall-memory-design.md`

**Objetivo:** trocar a memória de exploração de "conjunto de células exploradas" para
"conjunto de features estáticas observadas", eliminando os dois defeitos estruturais do
modelo por célula, e remover `explored_cells`/`explored_delta` do contrato.

**Branch:** `feat/tactical-map-fog-polygon-10e` (já criada, já tem commits)

**Ordem obrigatória:** este plano vem **antes** do plano do frontend. O contrato é o
handoff (regra do `CLAUDE.md` da raiz).

**Arquivos:**
- Criar: `internal/domain/match/entity/fog/player_memory.go`
- Apagar: `internal/domain/match/entity/fog/player_fog_state.go`, `cell_coord.go`
- Modificar: `internal/domain/match/service/visibility.go`
- Modificar: `internal/domain/match/service/filter_map_state.go`
- Modificar: `internal/domain/match/matchsession/match_session.go`
- Modificar: `internal/app/game/room.go`, `internal/app/game/message.go`
- Renomear: `internal/gateway/pg/fog/fog_state_repository.go` → `player_memory_repository.go`
- Criar: `migrations/20260805000000_player_memories.sql`
- Criar: `internal/domain/match/service/wall_memory_test.go`
- Modificar: `docs/dev/api/game-lobby.md`, `docs/documentation-map.yaml`

---

## Contexto que você precisa antes de escrever qualquer linha

### 1. `player_fog_states` nunca foi escrita — não existe dado para migrar

`internal/gateway/pg/fog/fog_state_repository.go` tem `Upsert`, `FindByMatchMap` e
`DeleteByMatch` com `TODO` retornando `nil`/`nil, nil`. A memória vive só em RAM. Não
existe backfill, não existe compatibilidade retroativa a preservar. Se você se pegar
escrevendo código de migração de dados, parou no lugar errado.

### 2. `SeenWalls` PRECISA morar em `internal/domain/match/service/`

Ela chama `wallInLOS`, que é **não exportada** em `filter_map_state.go`. Isso é
deliberado, não acidente: é o que garante que o predicado que revela a parede é
literalmente o mesmo que a grava. Não exporte `wallInLOS` para colocar `SeenWalls` em
outro pacote — isso destrói o invariante que o `TestSeenWallsAgreesWithFilterMapState`
existe para proteger.

### 3. Não passe `losWalls` para `SeenWalls`

`RecomputeVisibility` monta `losWalls := append(service.ToLOSWalls(...), service.BoundaryLOSWalls(s.grid)...)`.
Esses `BoundaryLOSWalls` são paredes **fantasma** das bordas do tabuleiro, com
`ID == service.BoundaryWallID`, usadas só para limitar a varredura. Se você passar
`losWalls` para `SeenWalls`, o jogador vai "lembrar" de uma parede `__board_edge__` que
não existe. Passe `s.GetWalls()`.

### 4. Grave a memória UMA vez, depois do laço de posições

`wallInLOS` já itera sobre todos os polígonos internamente. Chame `SeenWalls` uma vez,
depois do laço que percorre as posições das peças, com a fatia `polys` completa. Chamar
dentro do laço funciona mas repete trabalho O(paredes × posições) sem ganho.

### 5. `PlayerMemory.Has` precisa aguentar receptor nil

`FilterMapState` recebe a memória do jogador, e `GetPlayerMemory` devolve `nil` para
jogador que ainda não tem memória (caso comum: primeiro `map_full_state` do lobby, e
todo o caminho do mestre). Um `Has` que não trata nil vai dar panic e derrubar o game
server. O método tem guarda explícita — mantenha-a.

### 6. Portas secretas entram na memória normalmente

Não crie caso especial. `MaskSecretDoorForPlayer` é aplicado na hora de enviar,
independente de a parede ter vindo de LOS ou de memória. Uma porta secreta não revelada
é lembrada como a parede mascarada que o jogador viu.

---

## Task 1: entidade `PlayerMemory`

**Arquivos:**
- Criar: `internal/domain/match/entity/fog/player_memory.go`
- Apagar: `internal/domain/match/entity/fog/player_fog_state.go`
- Apagar: `internal/domain/match/entity/fog/cell_coord.go`

- [ ] **Passo 1: criar `player_memory.go`**

```go
package fog

import (
	"time"

	"github.com/google/uuid"
)

// FeatureKind identifies which kind of static map object a memory entry refers to.
// "Static" means it does not move on its own: walls today, decorations in phase 11.
//
// Character pieces are deliberately NOT memorable. Characters move freely, so the last
// position a player saw tells them nothing reliable about where the piece is now —
// remembering it would misinform rather than inform.
type FeatureKind string

const (
	FeatureWall FeatureKind = "wall"
)

// FeatureRef is a stable reference to one static map object.
type FeatureRef struct {
	Kind FeatureKind
	ID   string
}

// PlayerMemory is everything one player has ever observed on one (match, map).
//
// It replaces the older cell-based explored set. A cell was a lossy proxy for "did I
// see this wall", and it failed in both directions: a wall in plain view whose cell
// centre was never lit got forgotten the moment the player stepped away, and an
// occluded wall stretch inside a cell whose centre WAS lit got remembered although the
// player never saw it. Recording the observed object itself removes both by
// construction.
type PlayerMemory struct {
	PlayerID  uuid.UUID
	MatchID   uuid.UUID
	MapID     uuid.UUID
	Seen      map[FeatureRef]struct{}
	UpdatedAt time.Time
}

func NewPlayerMemory(playerID, matchID, mapID uuid.UUID) *PlayerMemory {
	return &PlayerMemory{
		PlayerID:  playerID,
		MatchID:   matchID,
		MapID:     mapID,
		Seen:      make(map[FeatureRef]struct{}),
		UpdatedAt: time.Now(),
	}
}

// Remember unions refs into the seen set and reports how many were new.
func (m *PlayerMemory) Remember(refs []FeatureRef) int {
	added := 0
	for _, r := range refs {
		if _, ok := m.Seen[r]; ok {
			continue
		}
		m.Seen[r] = struct{}{}
		added++
	}
	if added > 0 {
		m.UpdatedAt = time.Now()
	}
	return added
}

// Has reports whether the player has ever observed this feature.
//
// The nil guard is load-bearing: callers legitimately hold a nil memory (a player with
// no memory yet, and every master/lobby code path), and FilterMapState calls this on
// every wall. Without the guard the game server panics.
func (m *PlayerMemory) Has(kind FeatureKind, id string) bool {
	if m == nil || m.Seen == nil {
		return false
	}
	_, ok := m.Seen[FeatureRef{Kind: kind, ID: id}]
	return ok
}
```

- [ ] **Passo 2: apagar os arquivos do modelo antigo**

```bash
rm internal/domain/match/entity/fog/player_fog_state.go \
   internal/domain/match/entity/fog/cell_coord.go
```

- [ ] **Passo 3: ver a build quebrar, e onde**

```bash
go build ./... 2>&1 | head -30
```

Esperado: erros em `visibility.go`, `filter_map_state.go`, `match_session.go`, `room.go`,
`fog_state_repository.go`. **Essa lista é o seu roteiro** — as próximas tasks a percorrem
na ordem de dependência (domínio → sessão → delivery). Não tente consertar tudo agora.

---

## Task 2: `SeenWalls` e o filtro por memória

**Arquivos:**
- Modificar: `internal/domain/match/service/filter_map_state.go`
- Modificar: `internal/domain/match/service/visibility.go`

- [ ] **Passo 1: em `filter_map_state.go`, apagar `wallInExploredCells`**

Apague a função inteira (a que começa em `// wallInExploredCells reports whether any
stretch of the wall lies in a cell the viewer has already explored`). Ela é o proxy que
estamos removendo.

- [ ] **Passo 2: acrescentar `SeenWalls` no mesmo arquivo, logo abaixo de `wallInLOS`**

```go
// SeenWalls returns a memory reference for every wall currently in the viewer's line of
// sight, so the caller can union them into the player's PlayerMemory.
//
// INVARIANT — the predicate that reveals is the predicate that records.
// This uses the exact same wallInLOS as FilterMapState, in the same package, on purpose.
// If the two ever diverge, a player can see a wall that never reaches memory, and it
// vanishes the instant they step away — the false negative that the cell-based model
// suffered from. Guarded by TestSeenWallsAgreesWithFilterMapState.
//
// Pass the REAL walls here, never the LOS wall set: RecomputeVisibility appends
// BoundaryLOSWalls (ID == BoundaryWallID) to bound the sweep, and those are phantom
// board edges that must never enter a player's memory.
func SeenWalls(allWalls []mapentity.WallSegment, polys []VisibilityPolygon) []fog.FeatureRef {
	refs := make([]fog.FeatureRef, 0, len(allWalls))
	for _, w := range allWalls {
		if wallInLOS(w, polys) {
			refs = append(refs, fog.FeatureRef{Kind: fog.FeatureWall, ID: w.ID})
		}
	}
	return refs
}
```

- [ ] **Passo 3: trocar a assinatura e o corpo de `FilterMapState`**

O parâmetro `explored map[fog.CellCoord]struct{}` vira `memory *fog.PlayerMemory`. O
parâmetro `grid mapentity.GridShape` **sai** — ele só existia para `wallInExploredCells`.

Assinatura nova:

```go
func FilterMapState(
	allWalls []mapentity.WallSegment,
	pieces []PieceVisibility,
	polys []VisibilityPolygon,
	memory *fog.PlayerMemory,
	fogMode fog.FogMode,
	playerID uuid.UUID,
	charToPlayer map[string]uuid.UUID,
	isMaster bool,
) (walls []mapentity.WallSegment, visiblePieceIDs map[string]bool) {
```

E o laço de paredes vira:

```go
	// Walls: any part in LOS now, or (explored mode) observed at some point in the past.
	for _, w := range allWalls {
		seen := wallInLOS(w, polys)
		if !seen && fogMode == fog.FogModeExplored {
			seen = memory.Has(fog.FeatureWall, w.ID)
		}
		if !seen {
			continue
		}
		if w.WallType == mapentity.WallTypeSecretDoor && !w.Revealed {
			walls = append(walls, MaskSecretDoorForPlayer(w))
		} else {
			walls = append(walls, w)
		}
	}
```

O laço de peças fica **exatamente como está**. Peças não têm memória; não encoste nele.

- [ ] **Passo 4: em `visibility.go`, apagar `CellsInPolygon` e `slotEnvelope`**

Apague as duas funções inteiras.

- [ ] **Passo 5: deixar o compilador e o linter apontarem o resto**

```bash
go build ./internal/domain/... && golangci-lint run ./internal/domain/... 2>&1 | tail -20
```

Apague o que o linter reportar como `unused` (provavelmente `minInt` e `maxInt`, que só
serviam a `slotEnvelope`) e os imports que ficarem órfãos em `visibility.go`
(provavelmente `fog` e o alias `mapservice`). **Não adivinhe** quais são — siga o que a
ferramenta reporta.

---

## Task 3: testes do invariante e dos dois defeitos

Escreva os testes **antes** de mexer na sessão e no room. Eles não dependem de nenhuma
das duas coisas.

**Arquivos:**
- Criar: `internal/domain/match/service/wall_memory_test.go`

- [ ] **Passo 1: criar o arquivo**

```go
package service_test

import (
	"testing"

	mapentity "github.com/422UR4H/HxH_RPG_System/internal/domain/map/entity"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/match/service"
	"github.com/google/uuid"
)

// A room with one interior wall the viewer faces, and a second wall hidden directly
// behind it. The viewer stands to the left of both.
func memoryFixture() (viewer service.Point2D, walls []mapentity.WallSegment, grid mapentity.GridShape) {
	grid = mapentity.GridShape{
		Kind: mapentity.GridKindSquare, Cols: 20, Rows: 20, CellSize: 64, SkewRatio: 1,
	}
	near := mapentity.WallSegment{
		ID: "near", P1: [2]float64{640, 0}, P2: [2]float64{640, 1280},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	// Far wall sits behind `near` from the viewer's point of view, so it can never be
	// in line of sight and must never be remembered.
	far := mapentity.WallSegment{
		ID: "far", P1: [2]float64{704, 0}, P2: [2]float64{704, 1280},
		WallType: mapentity.WallTypeWall, Material: mapentity.WallMaterialStone,
		Sense: mapentity.SenseSight, HP: 100, MaxHP: 100,
	}
	return service.Point2D{X: 160, Y: 640}, []mapentity.WallSegment{near, far}, grid
}

func polysFor(viewer service.Point2D, walls []mapentity.WallSegment, grid mapentity.GridShape) []service.VisibilityPolygon {
	losWalls := append(service.ToLOSWalls(walls), service.BoundaryLOSWalls(grid)...)
	return []service.VisibilityPolygon{
		service.ComputeVisibilityPolygon(viewer, losWalls, 4000),
	}
}

func idsOf(walls []mapentity.WallSegment) map[string]bool {
	out := make(map[string]bool, len(walls))
	for _, w := range walls {
		out[w.ID] = true
	}
	return out
}

// The invariant from the spec: whatever FilterMapState reveals through line of sight is
// exactly what SeenWalls records. If these two predicates ever drift apart, a player
// sees a wall that never reaches memory and it disappears when they walk away.
func TestSeenWallsAgreesWithFilterMapState(t *testing.T) {
	viewer, walls, grid := memoryFixture()
	polys := polysFor(viewer, walls, grid)

	// Empty memory: everything FilterMapState returns came from line of sight alone.
	revealed, _ := service.FilterMapState(
		walls, nil, polys, nil, fog.FogModeExplored, uuid.New(), nil, false,
	)
	recorded := service.SeenWalls(walls, polys)

	revealedIDs := idsOf(revealed)
	recordedIDs := make(map[string]bool, len(recorded))
	for _, r := range recorded {
		if r.Kind != fog.FeatureWall {
			t.Fatalf("SeenWalls returned a non-wall kind: %q", r.Kind)
		}
		recordedIDs[r.ID] = true
	}

	if len(revealedIDs) != len(recordedIDs) {
		t.Fatalf("revealed %d walls but recorded %d", len(revealedIDs), len(recordedIDs))
	}
	for id := range revealedIDs {
		if !recordedIDs[id] {
			t.Fatalf("wall %q is shown to the player but never recorded in memory", id)
		}
	}
}

// The false negative of the cell model: the player sees a wall, walks away, and the
// wall must still be there.
func TestRememberedWallSurvivesLeavingLineOfSight(t *testing.T) {
	viewer, walls, grid := memoryFixture()
	polys := polysFor(viewer, walls, grid)

	memory := fog.NewPlayerMemory(uuid.New(), uuid.New(), uuid.New())
	memory.Remember(service.SeenWalls(walls, polys))

	// The player is gone: no polygons at all.
	got, _ := service.FilterMapState(
		walls, nil, nil, memory, fog.FogModeExplored, uuid.New(), nil, false,
	)
	if !idsOf(got)["near"] {
		t.Fatal("a wall the player already saw disappeared once they left line of sight")
	}
}

// The false positive of the cell model: a wall the player could never see must never be
// remembered, no matter how close the lit area gets.
func TestOccludedWallIsNeverRemembered(t *testing.T) {
	viewer, walls, grid := memoryFixture()
	polys := polysFor(viewer, walls, grid)

	for _, r := range service.SeenWalls(walls, polys) {
		if r.ID == "far" {
			t.Fatal("a wall hidden behind another wall was written to the player's memory")
		}
	}

	memory := fog.NewPlayerMemory(uuid.New(), uuid.New(), uuid.New())
	memory.Remember(service.SeenWalls(walls, polys))
	got, _ := service.FilterMapState(
		walls, nil, polys, memory, fog.FogModeExplored, uuid.New(), nil, false,
	)
	if idsOf(got)["far"] {
		t.Fatal("a wall the player never saw was sent to them")
	}
}

// live mode has no memory at all: walls exist only while in view.
func TestLiveModeIgnoresMemory(t *testing.T) {
	_, walls, _ := memoryFixture()

	memory := fog.NewPlayerMemory(uuid.New(), uuid.New(), uuid.New())
	memory.Remember([]fog.FeatureRef{{Kind: fog.FeatureWall, ID: "near"}})

	got, _ := service.FilterMapState(
		walls, nil, nil, memory, fog.FogModeLive, uuid.New(), nil, false,
	)
	if len(got) != 0 {
		t.Fatalf("live mode leaked %d remembered walls", len(got))
	}
}

// A nil memory is a real, common state (lobby, master, first push). It must not panic.
func TestNilMemoryIsSafe(t *testing.T) {
	_, walls, _ := memoryFixture()
	_, _ = service.FilterMapState(
		walls, nil, nil, nil, fog.FogModeExplored, uuid.New(), nil, false,
	)
}
```

- [ ] **Passo 2: rodar**

```bash
go test ./internal/domain/match/service/... 2>&1 | tail -20
```

Esperado: PASS. Se `TestOccludedWallIsNeverRemembered` falhar, **não relaxe o teste** —
a geometria da fixture pode precisar de ajuste (afastar mais a parede `far`), mas o
comportamento afirmado está certo.

- [ ] **Passo 3: commit**

```bash
git add internal/domain/match/entity/fog/ internal/domain/match/service/
git commit -m "feat(fog): memória de exploração por parede em vez de por célula

A célula era proxy com perda para 'eu vi esta parede': gerava falso negativo
(parede vista cujo centro de célula nunca acendeu era esquecida) e falso
positivo (trecho ocluído em célula iluminada era lembrado). Registrar o
objeto observado remove os dois por construção.

SeenWalls e FilterMapState compartilham o mesmo wallInLOS não exportado de
propósito — o predicado que revela é o que grava."
```

---

## Task 4: sessão de partida

**Arquivos:**
- Modificar: `internal/domain/match/matchsession/match_session.go`

- [ ] **Passo 1: trocar o campo e o construtor**

`fogStates map[uuid.UUID]*fog.PlayerFogState` → `memories map[uuid.UUID]*fog.PlayerMemory`.
Atualize as duas inicializações (`make(...)`) que existem no arquivo.

- [ ] **Passo 2: `SyncFogStates` → `SyncPlayerMemories`**

```go
// SyncPlayerMemories seeds per-player memory from persisted records (nil seeds empty
// memories). Resets the visibility cache.
func (s *MatchSession) SyncPlayerMemories(mems []fog.PlayerMemory, mode fog.FogMode) {
	s.fogMode = mode
	s.memories = make(map[uuid.UUID]*fog.PlayerMemory, len(mems))
	for i := range mems {
		m := mems[i]
		s.memories[m.PlayerID] = &m
	}
	s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
}
```

> O `m := mems[i]` seguido de `&m` é intencional e já era assim: sem a cópia local, todas
> as entradas do mapa apontariam para a mesma variável de laço.

- [ ] **Passo 3: `fogStateFor` → `memoryFor`**

```go
// memoryFor returns the existing memory for playerID, or lazily creates one.
func (s *MatchSession) memoryFor(playerID uuid.UUID) *fog.PlayerMemory {
	if s.memories == nil {
		s.memories = make(map[uuid.UUID]*fog.PlayerMemory)
	}
	m, ok := s.memories[playerID]
	if !ok {
		// TODO(persistence): MapID is uuid.Nil because MatchSession doesn't carry the
		// active map's UUID yet. Thread the real mapUUID in (constructor or
		// SyncPlayerMemories) before wiring the repository, so persisted rows don't all
		// collide on the (match_id, map_id, player_id) unique key with map_id = Nil.
		m = fog.NewPlayerMemory(playerID, s.matchUUID, uuid.Nil)
		s.memories[playerID] = m
	}
	return m
}
```

- [ ] **Passo 4: `RecomputeVisibility` — assinatura e gravação da memória**

A função perde o retorno de delta. Novo corpo:

```go
// RecomputeVisibility recomputes a player's LOS, caches the polygons, and (in explored
// mode) records every wall now in sight into that player's memory.
func (s *MatchSession) RecomputeVisibility(playerID uuid.UUID) ([]service.VisibilityPolygon, error) {
	walls := s.GetWalls()
	// The board edges block vision too, which keeps the polygon inside the map instead
	// of spilling out to maxRadius in every open direction.
	losWalls := append(service.ToLOSWalls(walls), service.BoundaryLOSWalls(s.grid)...)
	// Bound the polygon to the map diagonal (+20% margin) so a wall-less map still
	// produces a finite polygon. Falls back to visRadius when CellSize is 0.
	maxRadius := math.Hypot(float64(s.grid.Cols)*s.grid.CellSize, float64(s.grid.Rows)*s.grid.CellSize) * 1.2

	var positions []service.Point2D
	if s.pieceSource != nil {
		positions = s.pieceSource.PlayerPiecePositions(playerID)
	}
	polys := make([]service.VisibilityPolygon, 0, len(positions))
	for _, pos := range positions {
		polys = append(polys, service.ComputeVisibilityPolygon(pos, losWalls, maxRadius))
	}

	// One call, after the loop: wallInLOS already iterates every polygon internally.
	// `walls`, never `losWalls` — the latter carries phantom board-edge segments that
	// must never enter a player's memory.
	if s.fogMode == fog.FogModeExplored {
		s.memoryFor(playerID).Remember(service.SeenWalls(walls, polys))
	}

	if s.visCache == nil {
		s.visCache = make(map[uuid.UUID][]service.VisibilityPolygon)
	}
	s.visCache[playerID] = polys
	return polys, nil
}
```

- [ ] **Passo 5: ajustar `RecomputeAllVisibility` e os getters**

```go
func (s *MatchSession) RecomputeAllVisibility() error {
	for pid := range s.participants {
		if _, err := s.RecomputeVisibility(pid); err != nil {
			return err
		}
	}
	return nil
}

// GetPlayerMemory returns the player's memory, or nil when they have none yet.
func (s *MatchSession) GetPlayerMemory(playerID uuid.UUID) (*fog.PlayerMemory, bool) {
	m, ok := s.memories[playerID]
	return m, ok
}

func (s *MatchSession) GetAllPlayerMemories() []fog.PlayerMemory {
	out := make([]fog.PlayerMemory, 0, len(s.memories))
	for _, m := range s.memories {
		out = append(out, *m)
	}
	return out
}
```

> Preserve o formato exato dos getters antigos (`GetFogState`/`GetAllFogStates`) — só o
> tipo e o nome mudam. Se o original tinha comentário ou ordenação, mantenha.

- [ ] **Passo 6: build**

```bash
go build ./internal/domain/... ./internal/gateway/... 2>&1 | head -20
```

Sobra só `fog_state_repository.go` (Task 6) e `internal/app/game` (Task 5).

---

## Task 5: delivery — room e contrato WS

**Arquivos:**
- Modificar: `internal/app/game/message.go`
- Modificar: `internal/app/game/room.go`

- [ ] **Passo 1: encolher os payloads em `message.go`**

```go
type VisibilityUpdatedPayload struct {
	VisiblePolygons [][]Point2DPayload `json:"visible_polygons"`
}
```

```go
// MapFullStatePayload is the per-player map_full_state (server→client).
type MapFullStatePayload struct {
	Pieces          []PieceMovedPayload     `json:"pieces"`
	Walls           []mapentity.WallSegment `json:"walls,omitempty"`
	VisiblePolygons [][]Point2DPayload      `json:"visible_polygons,omitempty"`
	FogMode         string                  `json:"fog_mode"`
}
```

`ExploredDelta` e `ExploredCells` saem. `fog_mode` **fica** — ele ainda decide se a
memória se aplica.

- [ ] **Passo 2: `room.go` — chamadas de `RecomputeVisibility`**

Há 5 pontos de chamada (aprox. linhas 178, 325, 728, 974, 1236). Todos passam de
`_, _, err :=` / `polys, delta, err :=` para dois valores. Encontre-os com:

```bash
grep -n "RecomputeVisibility" internal/app/game/room.go
```

- [ ] **Passo 3: `room.go` — `pushVisibilityUpdates`**

O closure interno perde o `[]fogentity.CellCoord` do retorno, e o payload perde o laço
do delta:

```go
		polys, err := func() ([]domainservice.VisibilityPolygon, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			if r.session == nil {
				return nil, nil
			}
			return r.session.RecomputeVisibility(pid)
		}()
		if err != nil || polys == nil {
			return nil
		}
		payload := VisibilityUpdatedPayload{VisiblePolygons: polysToPayload(polys)}
		msg := NewServerMessage(MsgTypeVisibilityUpdated, payload)
		return &msg
```

> Não mexa no `r.mu.Lock()` dentro do closure. Ele é write lock de propósito:
> `RecomputeVisibility` chama de volta `PlayerPiecePositions`, e o contrato de locking
> está documentado em `room.go`. Trocar por `RLock` reintroduz o deadlock da 10-D,
> coberto por `TestRecomputeVisibilityUnderRoomWriteLock_DoesNotDeadlock`.

- [ ] **Passo 4: `room.go` — `buildMapFullState`**

Troque a variável `explored` por `memory`:

```go
	var polys []domainservice.VisibilityPolygon
	fogMode := fogentity.FogModeLive
	var memory *fogentity.PlayerMemory
	charToPlayer := map[string]uuid.UUID{}
	isLobby := r.session == nil
	if r.session != nil {
		polys = r.session.GetVisibility(playerID)
		fogMode = r.session.GetFogMode()
		charToPlayer = r.session.GetCharToPlayer()
		if m, ok := r.session.GetPlayerMemory(playerID); ok {
			memory = m
		}
	}
	r.mu.RUnlock()
```

As duas chamadas de `FilterMapState` (mestre e jogador) perdem o argumento `grid` e
passam `memory`:

```go
		walls, visIDs = domainservice.FilterMapState(
			allWalls, pieceProj, polys, memory, fogMode, playerID, charToPlayer, true,
		)
```

```go
		walls, visIDs = domainservice.FilterMapState(
			allWalls, pieceProj, polys, memory, fogMode, playerID, charToPlayer, false,
		)
```

E o bloco final do payload perde as células:

```go
	payload := MapFullStatePayload{Pieces: pieces, Walls: walls, FogMode: string(fogMode)}
	if !isMaster && !isLobby {
		payload.VisiblePolygons = polysToPayload(polys)
	}
```

O ramo `case isLobby:` fica **intacto** — ele não usa memória nem LOS.

- [ ] **Passo 5: renomear `SyncFogStates` nos dois pontos de chamada**

```bash
grep -n "SyncFogStates" internal/app/game/room.go
```

Vira `r.session.SyncPlayerMemories(nil, fogentity.FogModeExplored)`. Atualize também os
comentários `TODO(10-D persistence)` ao redor, que falam em "explored sets".

- [ ] **Passo 6: apagar `cellsToPayload`**

Ficou órfã. Confirme com `grep -n "cellsToPayload" internal/app/game/` antes de apagar.

- [ ] **Passo 7: build + testes + lint**

```bash
go build ./... && go test ./internal/... 2>&1 | tail -30
golangci-lint run 2>&1 | tail -10
```

Alguns testes existentes de fog vão quebrar por assinatura (`fog_regression_test.go`,
`fog_e2e_test.go`, `fog_smoke_test.go`). Ajuste-os ao novo formato — **sem afrouxar o que
eles afirmam**. Se algum teste checava `ExploredCells` no payload, ele perdeu o objeto:
apague só essa asserção, mantendo o resto do teste.

- [ ] **Passo 8: commit**

```bash
git add internal/domain/match/matchsession/ internal/app/game/
git commit -m "feat(fog): remover explored_cells/explored_delta do contrato WS

O cliente não precisa mais de dado de memória: o backend decide quais
paredes o jogador pode conhecer, e o polígono de LOS decide o brilho de
cada pixel. fog_mode permanece — ele ainda seleciona se a memória se aplica."
```

---

## Task 6: persistência e migração

Nada aqui grava dado ainda — o repositório continua stub. O objetivo é não deixar nome
mentindo sobre conteúdo.

**Arquivos:**
- Renomear: `internal/gateway/pg/fog/fog_state_repository.go` → `player_memory_repository.go`
- Criar: `migrations/20260805000000_player_memories.sql`

- [ ] **Passo 1: renomear e atualizar o repositório**

```bash
git mv internal/gateway/pg/fog/fog_state_repository.go \
       internal/gateway/pg/fog/player_memory_repository.go
```

Conteúdo:

```go
package fog

import (
	"context"

	"github.com/google/uuid"
	fogentity "github.com/422UR4H/HxH_RPG_System/internal/domain/match/entity/fog"
)

// PlayerMemoryRepository is the Postgres-backed memory persistence.
// TODO(persistence): implement Upsert/FindByMatchMap/DeleteByMatch against
// player_memories (serialize Seen as JSONB [{"kind":"wall","id":"<uuid>"}, ...]).
// Until then the session keeps memory in RAM; start_match seeding works in-memory.
type PlayerMemoryRepository struct {
	// db *pgxpool.Pool  // wire when implementing
}

func NewPlayerMemoryRepository( /* db *pgxpool.Pool */ ) *PlayerMemoryRepository {
	return &PlayerMemoryRepository{}
}

func (r *PlayerMemoryRepository) Upsert(ctx context.Context, m fogentity.PlayerMemory) error {
	// TODO: INSERT ... ON CONFLICT (match_id, map_id, player_id) DO UPDATE.
	return nil
}

func (r *PlayerMemoryRepository) FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fogentity.PlayerMemory, error) {
	// TODO: SELECT ... WHERE match_id=$1 AND map_id=$2.
	return nil, nil
}

func (r *PlayerMemoryRepository) DeleteByMatch(ctx context.Context, matchID uuid.UUID) error {
	// TODO: DELETE FROM player_memories WHERE match_id=$1.
	return nil
}
```

Se `NewFogStateRepository` for referenciado em algum lugar, atualize:

```bash
grep -rn "FogStateRepository" --include=*.go internal/ cmd/
```

- [ ] **Passo 2: criar a migração**

`migrations/20260805000000_player_memories.sql`:

```sql
-- migrations/20260805000000_player_memories.sql
-- Renames the fog memory table to match what it now stores. The old table held a set
-- of explored grid cells; it now holds the set of static map features the player has
-- observed. No data migration: player_fog_states was never written to (the repository
-- was a stub), so the rename is safe as-is.
-- +goose Up
-- +goose StatementBegin
BEGIN;

ALTER TABLE player_fog_states RENAME TO player_memories;
ALTER TABLE player_memories RENAME COLUMN explored TO seen_features;
ALTER TABLE player_memories DROP COLUMN grid_kind;
ALTER INDEX idx_player_fog_states_match_map RENAME TO idx_player_memories_match_map;

COMMIT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
BEGIN;

ALTER INDEX idx_player_memories_match_map RENAME TO idx_player_fog_states_match_map;
ALTER TABLE player_memories ADD COLUMN grid_kind VARCHAR(10) NOT NULL DEFAULT 'square';
ALTER TABLE player_memories RENAME COLUMN seen_features TO explored;
ALTER TABLE player_memories RENAME TO player_fog_states;

COMMIT;
-- +goose StatementEnd
```

> `grid_kind` existia só para interpretar coordenadas de célula. Sem células, é coluna
> morta. O `Down` a recria com default porque a coluna original era `NOT NULL` sem
> default e a tabela pode ter linhas num rollback futuro.

- [ ] **Passo 3: verificar**

```bash
go build ./... && go vet -tags=integration ./internal/gateway/pg/... && golangci-lint run
```

- [ ] **Passo 4: commit**

```bash
git add internal/gateway/pg/fog/ migrations/
git commit -m "refactor(fog): renomear persistência de fog state para player memory"
```

---

## Task 7: contrato e mapa de documentação

**Arquivos:**
- Modificar: `docs/dev/api/game-lobby.md`
- Modificar: `docs/documentation-map.yaml`

- [ ] **Passo 1: atualizar `docs/dev/api/game-lobby.md`**

Na seção que descreve as paredes que o jogador recebe, substitua o parágrafo que fala
em modo `explored` e paredes já vistas por:

```markdown
**Paredes que o jogador recebe.** Uma parede é enviada quando qualquer trecho dela está
na linha de visão do jogador — o teste amostra ao longo do segmento e desloca cada amostra
em direção ao observador. Isso é necessário porque uma parede que **bloqueia** a visão fica
exatamente sobre a borda do polígono de visibilidade: testar o ponto médio pela regra de
contenção responde "não visível", e a parede some da tela do jogador justamente quando ele
mais precisa dela (para abrir uma porta, arrombar, atacar). Paredes atrás de outra
continuam ocultas.

Em modo `explored`, toda parede que passou nesse teste é gravada na **memória** do
jogador e continua a ser enviada mesmo depois que ele sai da linha de visão. A memória
registra a parede observada (por id), não a região do mapa: um modelo por célula gerava
falso negativo (parede vista cujo centro de célula nunca entrou no polígono era
esquecida) e falso positivo (trecho ocluído numa célula iluminada era lembrado). Em modo
`live` não há memória — a parede some ao sair da visão.

O servidor **não** envia dado de memória ao cliente. O cliente desenha todas as paredes
que recebeu e usa o polígono de visibilidade para decidir o brilho de cada pixel: nítido
dentro da linha de visão, esmaecido fora dela.
```

E remova qualquer menção a `explored_cells` / `explored_delta` no documento.

- [ ] **Passo 2: `docs/documentation-map.yaml`**

Registre a spec nova. Siga o formato já usado para as entradas existentes — leia o
arquivo antes e imite a estrutura, não invente campos.

- [ ] **Passo 3: commit**

```bash
git add docs/
git commit -m "docs(api): contrato de paredes por memória; remover explored do payload"
```

---

## Verificação final do backend

```bash
go build ./... && go test ./... 2>&1 | tail -20
golangci-lint run --build-tags=smoke,integration
```

Tudo verde antes de passar para o plano do frontend
(`System_X_System_React/docs/superpowers/plans/2026-08-05-tactical-map-fog-walls-frontend.md`).
O frontend depende do contrato novo — não comece por lá.
