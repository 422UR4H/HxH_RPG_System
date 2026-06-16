# Tactical Map — Paredes: Fase 10-D — Visão / Line of Sight / Fog of War

**Data:** 2026-06-16
**Status:** Aprovado — pronto para writing-plans
**Escopo:** Cross-repo (`System_X_System` + `System_X_System_React`)
**Spec master:** `2026-06-10-tactical-map-walls-design.md` (seção Fase 10-D)
**Spec raiz:** `2026-05-31-tactical-map-design.md` (12 fases; seção 7.6 fog of war)
**Fase anterior:** Fase 10-C (Interações como Player Action — merged)
**Audiência:** Sessões futuras de planejamento e implementação (humanos + IA). Auto-contido — legível sem o transcrito do brainstorm.

---

## 1. Visão geral

Fase 10-D entrega **Line of Sight (LOS) e Fog of War** ao mapa tático. Cada jogador
passa a ver apenas o que seus personagens conseguem perceber; o backend filtra o estado
do mapa **por jogador** antes de enviar, tanto via REST quanto via WebSocket. O mestre vê
tudo. Paredes `secret_door` continuam aparecendo como paredes comuns até que o mestre as
revele.

A entrega segue o princípio do subsistema: **FoundryVTT como referência**. O modelo final
é o equivalente funcional ao "Token Vision + Fog Exploration" do FoundryVTT, adaptado ao
nosso modelo de dados (paredes como segmentos, grid square/hex/isométrico/rotacionado).

### 1.1 O que define "pronto" desta fase

- Jogador A com peça em uma sala não recebe (nem por REST nem por WS) as peças e paredes
  do outro lado de uma parede que bloqueia visão.
- Ao mover sua peça, o jogador recebe o que passou a ser visível; ao abrir uma porta, o
  cômodo atrás é revelado; ao destruir uma parede, a visão passa por ela.
- Áreas já vistas permanecem em cinza (fog explorado); a visão atual fica limpa; o nunca
  visto fica escuro.
- `secret_door` aparece como parede comum para jogadores (mesmo UUID, material, HP);
  pode ser atacada como parede; só vira porta visível quando o mestre revela.
- O mestre vê o mapa completo, sem fog.
- O estado de exploração é computado e **persistido no início da partida** (`start_match`)
  e tem o caminho pronto para persistir a cada fechamento de turno.

### 1.2 Princípio de design — experiência completa por padrão

O modo de fog completo (`explored` + LOS ativo) é o **default**. Não há, nesta fase, UI
para o mestre configurar o modo de fog ou desligar o LOS — isso é evolução futura
(seção 12). Enquanto não existir essa configuração, todos os jogos nascem com a
experiência completa, o que permite testar todas as features de uma vez.

---

## 2. Decisões de design

| Decisão | Escolha | Motivo |
|---|---|---|
| Algoritmo de LOS | **Geometric Angular Sweep** (polígono de visibilidade) | Opera em world coords; cobre square, hex, isométrico/rotacionado e paredes free-draw (Shift) sem conversão; one-way nativo. Ver seção 3.1 |
| Por que não Recursive Shadowcasting | Rejeitado | É algoritmo de células; falha com grid isométrico, paredes off-grid (free-draw) e one-way sem extensões frágeis. Ver seção 3.1 |
| Modos de fog | `live` e `explored`; **default `explored`** | Equivalente ao FoundryVTT; explored é a experiência completa |
| Onde filtra | Backend, em REST **e** WS, via `FilterMapState` único | Invariante do spec master: backend filtra, não confia no frontend |
| Fonte de paredes/peças p/ jogador | REST devolve estado já filtrado; WS refina ao vivo | `GET /maps/:id` chega ao jogador (confirmado em `GamePage`); precisa filtrar na origem |
| secret_door para jogador | **Mascarada** como `wall` (não removida) | Jogador vê e ataca como parede; reveal a transforma de volta |
| Reveal de secret_door | Global (todos), via `MasterAction` | `RevealedTo` por jogador (via `examine`) é evolução futura |
| Paredes que não bloqueiam LOS | `sense=="none"` OU `destroyed==true` OU `open==true` | Porta aberta/janela/parede destruída deixam ver através |
| `sense="sight"` vs `"full"` | Ambos bloqueiam **visão** | Distinção (audição passa em `sight`) reservada para percepção sonora futura |
| Recompute de LOS | Incremental por evento | Ver matriz na seção 5.3 |
| Persistência de exploração | `PlayerFogState` por (player×match×map); inicial no `start_match`, demais no turn close | Garante estado inicial; demais seguem fluxo existente de persistência de turno |
| Otimização de payload | `explored_cells` enviado por **delta** (só células novas) | Conjunto é monotônico; a união já produz o delta de graça |
| Broadcast | `dispatchPerPlayer` para conteúdo sensível; `r.broadcast` preservado | Mensagens insensíveis (chat, turn, cena) sem mudança; zero regressão |

---

## 3. Algoritmo de visibilidade

### 3.1 Por que Angular Sweep (e não Shadowcasting)

O spec master sugeriu recursive shadowcasting; durante o brainstorm da 10-D, a decisão
foi **revista** para **angular sweep** por causa de três features confirmadas do nosso
sistema que quebram o shadowcasting:

1. **Grid isométrico/rotacionado** — shadowcasting opera em células cardinais; exigiria
   transformar coords para um espaço retificado, rodar, e reverter — frágil.
2. **Paredes free-draw (Shift desativa snap)** — segmentos off-grid não mapeiam a arestas
   de células sem discretização (Bresenham), que gera erro de visibilidade.
3. **One-way walls (`direction = left/right`)** — shadowcasting padrão não tem bloqueio
   direcional por aresta.

O **angular sweep** opera em coordenadas de mundo contínuas: funciona com qualquer
geometria de parede e qualquer grid, e o one-way é um teste de lado (cross product) na
interseção. É a mesma família de algoritmo do `ClockwiseSweepPolygon` do FoundryVTT.

Custo: O(N·log N) por origem (N = endpoints de paredes que bloqueiam). Com ~7.200
segmentos máximos e ~24 origens (8 jogadores × 3 peças), ~2ms em Go por recomputação
total — adequado para o modelo turn-based.

### 3.2 Tipos (`internal/domain/match/service/visibility.go`)

```go
type Point2D struct{ X, Y float64 }

// VisibilityPolygon é o resultado do sweep para uma origem (uma peça).
// Vértices em world coords (pré-transform). Usado para filtragem backend
// e como payload para o cliente renderizar o fog.
type VisibilityPolygon struct {
    Origin   Point2D
    Vertices []Point2D
}

// WallSegmentLOS é a projeção mínima de WallSegment para o algoritmo.
// Já filtrado: contém só paredes que bloqueiam visão.
type WallSegmentLOS struct {
    ID        string
    P1, P2    Point2D
    Direction mapentity.WallDirection // "both" | "left" | "right"
}
```

### 3.3 Funções

```go
// ToLOSWalls filtra e projeta []WallSegment → []WallSegmentLOS.
// EXCLUI uma parede se qualquer condição for verdadeira:
//   - w.Sense == SenseNone   (não bloqueia percepção)
//   - w.Destroyed == true     (não existe fisicamente)
//   - w.Open == true          (porta/janela aberta — vê-se através)
// Inclui sense "full" e "sight" (ambos bloqueiam visão).
func ToLOSWalls(walls []mapentity.WallSegment) []WallSegmentLOS

// ComputeVisibilityPolygon — angular sweep a partir de uma origem.
// Coleta endpoints como eventos de ângulo, ordena, varre mantendo um
// conjunto ordenado de segmentos ativos (nearest-first), constrói o polígono.
// One-way: ao testar um segmento com Direction != "both", verifica o lado
// de bloqueio via sinal do cross product (P2-P1) × (origin-P1).
func ComputeVisibilityPolygon(origin Point2D, walls []WallSegmentLOS) VisibilityPolygon

// PointInPolygon — ray casting clássico. O(V).
func PointInPolygon(p Point2D, poly []Point2D) bool

// IsVisible — OR sobre os polígonos do jogador (uma peça → um polígono).
func IsVisible(target Point2D, polys []VisibilityPolygon) bool

// CellsInPolygon — células do grid cujo centro cai dentro do polígono.
// Só usada no modo `explored` (acúmulo de ExploredCells).
// square: itera (col,row) na bounding box; hex: itera (q,r) na bbox axial.
// Usa o GridShape completo (kind, cellSize, skewRatio, rotation).
func CellsInPolygon(poly VisibilityPolygon, grid mapentity.GridShape) []fog.CellCoord
```

> **Nota de implementação (hex/isométrico):** `CellsInPolygon` precisa de math
> slot↔world server-side para hex e para o caso isométrico (aplicar/reverter
> `applyTransform`). Se o backend ainda não tiver esse math (o frontend tem em
> `utils/coords.ts` + `hex.ts`), portar as funções necessárias para Go é parte do
> escopo desta fase. O sweep em si é grid-agnóstico (world coords) — só o tracking
> de células exploradas depende do grid.

---

## 4. Modelo de dados

### 4.1 Frontend (`src/types/tacticalMap.ts`)

```ts
export type FogMode = "live" | "explored";

// WallSegment — adicionar:
export type WallSegment = {
  // ... campos existentes ...
  revealed?: boolean; // secret_door revelada a todos pelo mestre (default false/undefined)
};

// TacticalMap — adicionar:
export type TacticalMap = {
  // ... campos existentes ...
  fogMode: FogMode; // default "explored"
};

// Estado de fog do viewer (não persiste no editor; vem do WS/REST):
export type FogState = {
  fogMode: FogMode;
  visiblePolygons: Array<Array<{ x: number; y: number }>>; // um polígono por peça do jogador
  exploredCells: Set<string>;  // "a,b" acumulado localmente; só no modo explored
};
```

### 4.2 Backend — `FogMode` e `CellCoord` (`internal/domain/match/entity/fog/`)

```go
// fog/fog_mode.go
type FogMode string
const (
    FogModeLive     FogMode = "live"
    FogModeExplored FogMode = "explored"
)

// fog/cell_coord.go
// CellCoord é genérico: (A,B) = (Col,Row) para square, (Q,R) para hex.
type CellCoord struct{ A, B int }

// fog/player_fog_state.go
type PlayerFogState struct {
    PlayerID      uuid.UUID
    MatchID       uuid.UUID
    MapID         uuid.UUID
    GridKind      mapentity.GridKind         // square | hex (para interpretar A,B)
    ExploredCells map[CellCoord]struct{}     // união acumulada de tudo já visto
    UpdatedAt     time.Time
}
```

### 4.3 Backend — `WallSegment` e `GridShape`

```go
// wall_segment.go — adicionar:
Revealed bool // secret_door revelada a todos pelo mestre
              // TODO(futuro): evoluir para RevealedTo map[uuid.UUID]bool (reveal
              // por jogador via examine quando o sistema de Skills existir)

// TacticalMap (map entity) — adicionar:
FogMode fog.FogMode // default FogModeExplored
```

`GridShape` já existe completo em `internal/domain/map/entity/grid.go`
(Kind, Cols, Rows, CellSize, SkewRatio, Rotation) — reusar.

### 4.4 Banco de dados

```sql
-- maps: nova coluna, default = experiência completa
ALTER TABLE maps ADD COLUMN fog_mode VARCHAR(10) NOT NULL DEFAULT 'explored';

-- nova tabela: estado de exploração por (match × map × player)
CREATE TABLE player_fog_states (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    match_id    UUID        NOT NULL REFERENCES matches(id) ON DELETE CASCADE,
    map_id      UUID        NOT NULL REFERENCES maps(id)    ON DELETE CASCADE,
    player_id   UUID        NOT NULL,
    grid_kind   VARCHAR(10) NOT NULL,           -- 'square' | 'hex'
    explored    JSONB       NOT NULL DEFAULT '[]', -- [[a,b],...]
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(match_id, map_id, player_id)
);
CREATE INDEX idx_player_fog_states_match_map ON player_fog_states(match_id, map_id);
```

Migração goose. O `fog_mode` reusa a coluna `maps` (sem nova migração de tabela para isso,
só `ALTER`). `player_fog_states` é tabela nova.

### 4.5 Interface de repositório (definida agora; impl PG é TODO)

```go
// application/fog/i_fog_state_repository.go
type IFogStateRepository interface {
    Upsert(ctx context.Context, state fog.PlayerFogState) error
    FindByMatchMap(ctx context.Context, matchID, mapID uuid.UUID) ([]fog.PlayerFogState, error)
    DeleteByMatch(ctx context.Context, matchID uuid.UUID) error
}
```

A interface fica no `application/` nesta fase. A implementação em
`gateway/pg/fog/fog_state_repository.go` recebe um esqueleto com `// TODO: implementar`.
O wire-up no fechamento de turno (`PersistTurnClose`) fica como `// TODO` documentado
(seção 5.4). **A persistência inicial no `start_match` é concreta e implementada agora**
(seção 5.5) — usa a mesma interface; só o ponto de injeção do repo PG fica pendente, com
fallback in-memory enquanto o repo não existir.

> **Nota de pragmatismo:** como o repo PG é TODO, a persistência "real" no banco fica
> inativa até o usuário implementá-lo. O estado vive em memória na sessão e funciona para
> sessões contínuas. Toda a arquitetura (entidade, interface, pontos de chamada, payload)
> fica pronta para que ligar o banco seja trivial.

---

## 5. Backend — sessão, recompute e broadcast

### 5.1 Campos novos em `MatchSession`

```go
type MatchSession struct {
    // ... existentes ...
    fogMode      fog.FogMode
    grid         mapentity.GridShape                          // substitui gridSize float64
    fogStates    map[uuid.UUID]*fog.PlayerFogState            // keyed by playerID
    visCache     map[uuid.UUID][]visibility.VisibilityPolygon // cache LOS; nil = recomputar
    charToPlayer map[string]uuid.UUID                         // characterID → playerUUID (NPCs ausentes)
}
```

> **Mudança de `gridSize float64` → `grid GridShape`:** hex e isométrico precisam do grid
> completo. `GetGridSize()` continua existindo (`return s.grid.CellSize`) para o code path
> de movement blocking já existente; novos consumidores usam `GetGrid()`.

`charToPlayer` é construído em `NewMatchSession` a partir de `participants`. NPCs (sem
player) ficam ausentes do mapa — o que os torna "não-próprios" de ninguém (filtrados só
por LOS), correto.

### 5.2 Métodos novos em `MatchSession`

```go
func (s *MatchSession) SyncFogStates(states []fog.PlayerFogState, mode fog.FogMode)
func (s *MatchSession) GetGrid() mapentity.GridShape

// Recomputa LOS de um player; atualiza visCache e faz união em ExploredCells.
// Retorna polígonos novos e o DELTA de células recém-exploradas (para o WS).
func (s *MatchSession) RecomputeVisibility(playerID uuid.UUID) (
    polys []visibility.VisibilityPolygon, exploredDelta []fog.CellCoord, err error)

// Recomputa para todos os players (usado quando uma parede muda de estado).
func (s *MatchSession) RecomputeAllVisibility() error

func (s *MatchSession) GetVisibility(playerID uuid.UUID) []visibility.VisibilityPolygon
func (s *MatchSession) InvalidateVisibilityCache()
func (s *MatchSession) RevealSecretDoor(wallID string) // seta Revealed=true + invalida cache

// Para PersistTurnClose (TODO de wire-up):
func (s *MatchSession) GetFogState(playerID uuid.UUID) (*fog.PlayerFogState, bool)
func (s *MatchSession) GetAllFogStates() []fog.PlayerFogState
```

`RecomputeVisibility` internamente: pega as peças do player, computa um
`VisibilityPolygon` por peça (`ComputeVisibilityPolygon` sobre `ToLOSWalls(session walls)`),
guarda em `visCache`, e — no modo `explored` — calcula `CellsInPolygon`, adiciona ao
`ExploredCells` do `PlayerFogState` e retorna só as células que eram novas.

### 5.3 Matriz de recompute

| Evento | Quem recomputa | Mensagem resultante |
|---|---|---|
| Jogador move a própria peça | só esse jogador | re-envia `map_full_state` filtrado **a ele** + `visibility_updated` |
| Mestre move NPC | ninguém (não muda polígono de jogador) | `piece_moved` filtrado por receptor (5.6) |
| Parede open/close/destroyed | todos (`RecomputeAllVisibility` + invalida cache) | `visibility_updated` por jogador + evento da parede |
| `start_match` | todos (computação inicial) | `map_full_state` filtrado por jogador |
| Client registra mid-game | ninguém (usa cache) | `map_full_state` filtrado a partir do cache |
| Mestre revela secret_door | invalida cache (geometria não muda, mas tipo sim) | `wall_revealed` (broadcast) |

### 5.4 `FilterMapState` — serviço único (REST + WS)

**Acoplamento:** `FilterMapState` é política de domínio, então **não** depende de tipos da
camada de entrega (`game.PieceMovedPayload`). Recebe uma projeção de domínio das peças e
devolve as paredes filtradas (tipo de domínio) + o **conjunto de IDs de peça visíveis**. O
`room.go` (e o handler REST) convertem seus `PieceMovedPayload` ↔ projeção e usam o conjunto
de IDs para filtrar os próprios payloads.

```go
// internal/domain/match/service/filter_map_state.go

// PieceVisibility é a projeção mínima de uma peça para a filtragem (domínio puro).
type PieceVisibility struct {
    ID          string
    CharacterID string
    Pos         visibility.Point2D // world coords (centro da peça)
    Visible     bool
}

// FilterMapState aplica todas as políticas de visibilidade. Reusado pelo handler
// REST (GET /maps/:id) e pelo dispatch WS (map_full_state, piece_moved).
// Retorna paredes filtradas/mascaradas (domínio) e o conjunto de IDs de peça que
// o jogador pode ver — o caller mapeia de volta aos seus payloads.
func FilterMapState(
    allWalls     []mapentity.WallSegment,
    pieces       []PieceVisibility,
    polys        []visibility.VisibilityPolygon, // do jogador (vazio → só explored/nada)
    explored     map[fog.CellCoord]struct{},     // nil se modo live
    fogMode      fog.FogMode,
    grid         mapentity.GridShape,
    playerID     uuid.UUID,
    charToPlayer map[string]uuid.UUID,
    isMaster     bool,
) (walls []mapentity.WallSegment, visiblePieceIDs map[string]bool)
```

**Regras:**

| Objeto | Incluir para o jogador quando |
|---|---|
| Peça `visible=true` | `IsVisible(piece.Pos, polys)` OU é peça própria (`charToPlayer[cid]==playerID`) |
| Peça `visible=false` | nunca (só mestre) |
| Parede normal | `IsVisible(midpoint, polys)` OU (modo explored E célula do midpoint em `explored`) |
| `secret_door` não revelada | incluir **mascarada** (`MaskSecretDoorForPlayer`) se visível/explorada |
| `secret_door` revelada | incluir como secret_door se visível/explorada |

`isMaster=true` → paredes completas sem máscara e `visiblePieceIDs` com todas as peças.
A "célula do midpoint" é a célula que contém `(p1+p2)/2` via `worldToSlot`.

Midpoint `(p1+p2)/2` é o ponto de teste da parede (segmentos são ≤ 1 célula pós
`explodePolyline`, então o midpoint representa bem).

### 5.5 `MaskSecretDoorForPlayer`

```go
// internal/domain/match/service/mask_secret_door.go
func MaskSecretDoorForPlayer(w mapentity.WallSegment) mapentity.WallSegment {
    masked := w                       // copia tudo: id, p1, p2, material, hp, maxHp, resistance...
    masked.WallType    = mapentity.WallTypeWall
    masked.DoorSubtype = nil
    masked.Open        = false
    masked.Locked      = false
    return masked
}
```

Tudo que importa para combate (id, material, hp, maxHp, resistance, move, sense, direction)
é preservado: o jogador ataca a "parede" e ela responde igual a uma parede de verdade.

### 5.6 Persistência inicial no `start_match`

`Room.StartMatch` ganha passos novos após `SyncMapState`:

```
StartMatch():
  1..3. (existente) startMatchUC, initSessionUC, session.SyncMapState(walls, grid)
  4. [NOVO] session.SyncFogStates(loadFromRepoOrEmpty(), map.FogMode)
  5. [NOVO] para cada player com peça no mapa:
       polys, delta := session.RecomputeVisibility(playerID)   // semeia ExploredCells
       fogStateRepo.Upsert(session.GetFogState(playerID))       // persistência inicial
       // ^ TODO de wire-up do repo PG; com fallback in-memory funciona sem o banco
  6. (existente) broadcast match_started
  7. [NOVO] para cada client conectado: sendFilteredMapFullState(client)
       // master → tudo; player → FilterMapState
```

O passo 5 é a **única persistência fora do `PersistTurnClose`**, garantindo estado inicial
mesmo se o servidor reiniciar antes do primeiro turno.

### 5.7 Wire-up de persistência no turn close (TODO)

Em `application/match/open_next_action.go` (e `pull_action.go`), após `PersistTurnClose`:

```go
// TODO(10-D persistência): persistir fog states ao fechar turno.
// for _, st := range session.GetAllFogStates() {
//     fogStateRepo.Upsert(ctx, st)
// }
// Interface: application/fog/i_fog_state_repository.go
// Injetar o repo PG aqui quando o fluxo de persistência de turno for finalizado.
```

### 5.8 Broadcast por-player (`room.go`)

```go
// dispatchPerPlayer envia payload distinto por client. build retorna nil = não envia.
func (r *Room) dispatchPerPlayer(build func(playerID uuid.UUID, isMaster bool) *Message)
```

`r.broadcast` é **preservado** para mensagens insensíveis (chat, turn_opened,
scene_changed, round_closed, player joined/left/kicked) — zero regressão.

**Mensagens que passam a usar dispatch por-player ou envio direcionado:**

| Mensagem | Antes | Depois |
|---|---|---|
| `map_full_state` (register e start_match) | broadcast/MapPieces | `dispatchPerPlayer` → `FilterMapState` |
| `visibility_updated` | n/a | direcionado ao player (após move próprio / wall change) |
| `wall_revealed` | n/a | `r.broadcast` (todos passam a ver) |
| `wall_state_changed` (open/locked) de secret_door não revelada | broadcast | só mestre |
| `wall_hp_changed` de secret_door não revelada | broadcast | `dispatchPerPlayer` — players que veem a parede recebem (tratam como wall) |
| `piece_moved` `visible=true` | broadcast | `dispatchPerPlayer` filtrado por LOS do receptor (5.9) |
| `piece_moved`/`piece_removed` `visible=false` | broadcast | só mestre |

> **Importante:** `wall_hp_changed` **não** vaza o tipo (payload só tem `hp/maxHp/destroyed`).
> Então atacar uma secret_door não revelada produz, no cliente do jogador, dano numa
> "parede" — visualmente consistente. Já `wall_state_changed` carrega `open`, que revelaria
> ser porta — por isso fica restrito ao mestre até o reveal.

### 5.9 Movimento de peça com fog (core)

Quando uma peça `visible=true` move (NPC ou peça de outro jogador), `piece_moved` é
dispatchado por receptor:

- Receptor enxerga a **posição nova** (`IsVisible(newPos, polysR)`) → envia `piece_moved`
  (R vê no destino).
- Receptor **não** vê a nova mas via a **antiga** → envia `piece_removed` (R vê sumir da
  vista; a peça não é apagada do board do servidor, só some para R).
- Receptor não via nem antiga nem nova → não envia nada.

Quando o **próprio** jogador move sua peça, o LOS *dele* muda: recomputa-se a visibilidade
dele e re-envia-se um `map_full_state` filtrado **a ele** (um jogador, barato), com as
paredes/peças recém-visíveis e os polígonos novos.

> **Escopo / TODO:** o fluxo de movimento *em jogo* (Action.Move → resolução de turno →
> atualização do board) ainda tem TODOs no subsistema de turnos. Esta fase liga o fog ao
> que já funciona: computação inicial no `start_match`, posicionamento no lobby
> (`piece_moved`), mudanças de estado de parede e o caminho de move da própria peça onde já
> está conectado. Os pontos do fluxo de turno recebem `// TODO(10-D fog)` de integração, no
> mesmo padrão das fases anteriores.

---

## 6. Lobby vs Game

### 6.1 Lobby (sem LOS)

O lobby é setup — jogadores precisam ver o board para posicionar peças. Filtragem mínima,
**sem** LOS:

- Peças `visible=false` → só mestre.
- Paredes `secret_door` → mascaradas como `wall` para jogadores.
- Sem fog overlay, sem polígonos.

A Room já existe no lobby e detém `r.walls`/`r.pieces` (semeados pelo mestre via
`map_state_sync`). `sendMapFullState` no lobby passa a incluir **paredes mascaradas** e
**peças filtradas por visible**, mas com `polys` vazio e `fogMode` tratado como "sem LOS".

### 6.2 Transição para Game

No `start_match` (seção 5.6), o LOS liga: computação inicial, persistência inicial e
`map_full_state` filtrado por LOS a cada client.

### 6.3 REST `GET /maps/:id`

Confirmado que o jogador carrega o mapa por REST (`GamePage` → `useMap`). O handler passa a
aplicar `FilterMapState` **role-aware**:

- Requester é mestre da campanha → mapa completo, sem filtro.
- Requester é jogador → `secret_door` mascarada, `visible=false` removidas; e, se o mapa
  estiver numa partida ativa, LOS computado a partir das **posições do snapshot** do mapa
  (peças do próprio jogador) + `explored` persistido (quando houver). O WS refina ao vivo.

O LOS no REST usa o snapshot (posições salvas) — pode estar levemente defasado durante a
partida; o WS corrige em seguida. Sem acoplamento REST→game-hub: o handler usa só o DB
(`maps` + `player_fog_states`) e o mesmo `FilterMapState`.

---

## 7. Mensagens WS (`message.go`)

```go
MsgTypeVisibilityUpdated MessageType = "visibility_updated"
MsgTypeWallRevealed      MessageType = "wall_revealed"

type Point2DPayload struct {
    X float64 `json:"x"`
    Y float64 `json:"y"`
}

type VisibilityUpdatedPayload struct {
    VisiblePolygons [][]Point2DPayload `json:"visible_polygons"`        // um polígono por peça
    ExploredDelta   [][2]int           `json:"explored_delta,omitempty"`// só células novas (modo explored)
}

type WallRevealedPayload struct {
    Wall mapentity.WallSegment `json:"wall"` // WallSegment completo com wallType real
}

// map_full_state estendido:
type MapFullStatePayload struct {
    Pieces          []PieceMovedPayload     `json:"pieces"`
    Walls           []mapentity.WallSegment `json:"walls,omitempty"`            // filtradas/mascaradas
    VisiblePolygons [][]Point2DPayload      `json:"visible_polygons,omitempty"` // nil p/ mestre
    ExploredCells   [][2]int                `json:"explored_cells,omitempty"`   // conjunto COMPLETO (seed); nil se live
    FogMode         string                  `json:"fog_mode"`
}
```

`explored_cells` em `map_full_state` é o conjunto completo (semeia o cliente);
`explored_delta` em `visibility_updated` são só as novas. O cliente acumula.

---

## 8. Frontend

### 8.1 `FogLayer.tsx` (novo organism)

Camada Pixi dentro do `worldContainer` (herda transform iso/rotação), acima das paredes e
abaixo do overlay de interação. Renderiza três tiers:

| Região | Visual |
|---|---|
| Não explorada | escuro ~0.92 |
| Explorada fora da visão (modo explored) | cinza ~0.5 |
| Visão atual (dentro dos polígonos) | limpo |

Técnica: preenche os limites do mundo com escuro; usa `visiblePolygons` como máscara que
recorta a visão atual; no modo explored, uma camada cinza preenche `exploredCells` e é
recortada pela visão atual. No modo live, sem camada cinza.

**Mestre:** `FogLayer` desabilitado (recebe estado completo). Toggle "ver como jogador" é
futuro.

### 8.2 `useMatchWs.ts`

Passa a tratar:
- `map_full_state` estendido → semeia walls/pieces/visiblePolygons/exploredCells/fogMode.
- `visibility_updated` → seta `visiblePolygons`; acumula `explored_delta` no Set local.
- `wall_revealed` → substitui a parede mascarada (mesmo id) pela real (vira secret_door).
- Renomeações já existentes (`map_state_sync`, `map_full_state`, `piece_moved/removed`).

### 8.3 `TacticalMapViewer.tsx` e tipos

- Monta `FogLayer` com `{ fogMode, visiblePolygons, exploredCells }`; desabilitado p/ mestre.
- `tacticalMap.ts`: `FogMode`, `WallSegment.revealed?`, `TacticalMap.fogMode`, `FogState`.
- secret_door mascarada chega como `wall` → renderiza como parede, **sem código especial**;
  action picker da 10-C já oferece "Atacar". `wall_revealed` troca para secret_door.

---

## 9. Documentação

- `System_X_System_React/docs/dev/tactical-map/overview.md` — seção LOS/fog (modelo,
  camadas, mestre vê tudo).
- `System_X_System_React/docs/dev/tactical-map/sync-and-delta.md` — eventos
  `visibility_updated`/`wall_revealed`, delta de explored, dispatch por-player.
- `System_X_System/docs/dev/api/` — `walls.md`/`maps.md`: filtragem role-aware no
  `GET /maps/:id`; shape dos novos eventos WS.
- `.github/instructions/game-server.instructions.md` — regra de broadcast por-player +
  invariante "backend filtra".
- `docs/player/paredes.md` — atualizar: o que o jogador vê (fog), secret_door parece parede.
- `docs/documentation-map.yaml` — registrar `visibility.go`, `filter_map_state.go`,
  `mask_secret_door.go`, `fog/`, tabela `player_fog_states`, eventos novos.

---

## 10. Fora de escopo (10-D)

- **UI do mestre para configurar fog** (escolher live/explored, desligar LOS, "ver como
  jogador") — default `explored`, sem config nesta fase.
- **Implementação PG do `IFogStateRepository`** e wire-up no turn close — interface e pontos
  de chamada prontos; persistência inicial no `start_match` funciona com fallback in-memory.
- **Reveal por jogador via `examine`** — `Revealed` é global; evolui para `RevealedTo`
  quando o sistema de Skills existir.
- **Raio de visão por peça** — visão ilimitada por ora (até bater em parede).
- **Iluminação / luz dinâmica** — tokens emitindo luz; escuridão como dimensão extra.
- **Integração completa do fluxo de move em jogo** (turn→board) — TODOs onde o fluxo ainda
  está em definição.

---

## 11. Critério de pronto

- Jogador A (peça em uma sala) não recebe, por REST nem WS, peças/paredes do outro lado de
  uma parede `sense=full`. Ao mover, recebe o que passou a ver.
- Abrir uma porta revela o cômodo atrás; destruir uma parede abre a visão por ela.
- Modo `explored` (default): área vista permanece cinza; visão atual limpa; nunca-visto
  escuro.
- `secret_door` aparece como parede para jogadores (mesmo UUID/material/HP), pode ser
  atacada, toma dano visível; ao reveal do mestre, vira porta secreta com interações.
- Mestre vê tudo, sem fog.
- Estado de exploração é computado e persistido (via interface) no `start_match`;
  `GetAllFogStates` pronto para o wire-up do turn close.
- `r.broadcast` intacto para chat/turn/scene; nenhuma regressão nas features existentes.

---

## 12. Capacidades futuras (além da 10-D)

- **Config de fog pelo mestre:** UI para escolher `live`/`explored`, ligar/desligar LOS por
  cena, e "ver como jogador" (preview). `fogMode` já está no schema.
- **Persistência DB do fog:** implementar `gateway/pg/fog/` + wire no turn close.
- **Reveal per-player + `examine`:** `WallSegment.RevealedTo map[uuid]bool`; `examine` roll
  revela só para quem passou.
- **Raio de visão / iluminação:** peças com alcance de visão; luzes; escuridão.
- **Otimização de delta para polígonos:** hoje `visible_polygons` vai completo (correto, pois
  encolhe); se virar gargalo, considerar diff geométrico. `explored` já é delta.
- **`sense=sight` com audição:** quando houver percepção sonora, `sight` deixa som passar.
