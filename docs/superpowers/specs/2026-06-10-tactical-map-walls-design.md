# Tactical Map — Paredes & Obstáculos — Design Spec

**Data:** 2026-06-10
**Status:** Aprovado — pronto para implementação em fases
**Escopo:** Cross-repo (`System_X_System_React` + `System_X_System`)
**Referência base:** `docs/superpowers/specs/2026-05-31-tactical-map-design.md` (spec master das 12 fases)
**Audiência:** Sessões futuras de planejamento e implementação (humanos + IA). Auto-contido — legível sem o transcrito do brainstorm.

---

## 1. Visão geral

Este documento especifica o **subsistema de paredes e obstáculos** do mapa tático. O subsistema é entregue em **4 sub-fases sequenciais** (10, 10-B, 10-C, 10-D), cada uma com seu próprio PR. Paredes são segmentos de reta entre dois pontos, com atributos que controlam bloqueio de movimento, bloqueio de percepção, estado destrutível e interatividade.

### Princípio de design — FoundryVTT como referência

Paredes em VTTs de referência (FoundryVTT, Owlbear Rodeo) são segmentos individuais com atributos por segmento. FoundryVTT usa `PIXI.Graphics` para renderizar paredes — a mesma stack do projeto. Nenhum asset externo, nenhum SVG. O design deste subsistema segue essa abordagem e vai além: paredes têm HP, resistência por material e estado destrutível, permitindo que personagens as danifiquem e destruam em combate futuro.

---

## 2. Modelo de dados

### 2.1 WallSegment (frontend — `src/types/tacticalMap.ts`)

```ts
export type WallType =
  | "stone"       // pedra — espessa, alta resistência
  | "wood"        // madeira — destruível por força
  | "iron"        // ferro/aço — quase indestrutível
  | "magical"     // barreira mágica — vulnerável a magia, passável por não-magos (futuro)
  | "terrain"     // terreno — borda de penhasco, obstáculo natural
  | "door"        // porta — abre/fecha, pode ser trancada
  | "window"      // janela — bloqueia movimento, permite visão
  | "secret_door" // porta secreta — invisível para jogadores até ser encontrada

export type SenseKind =
  | "full"        // bloqueia toda percepção (sight, hearing, etc.)
  | "sight"       // bloqueia só visão (audição passa)
  | "none"        // não bloqueia percepção (apenas visual/decorativo)

export type WallSegment = {
  id:        string           // uuid próprio do segmento
  p1:        [number, number] // coords de mundo (antes de applyTransform)
  p2:        [number, number] // coords de mundo
  wallType:  WallType

  // Atributos de bloqueio (consumidos por 10-B e 10-D)
  move:      boolean          // bloqueia movimento físico?
  sense:     SenseKind        // o que bloqueia em termos de percepção
  oneWay:    boolean          // só bloqueia de um lado (ex: borda de penhasco)
  // Nota sobre oneWay: a direção "bloqueada" é determinada pela normal do segmento.
  // p1→p2 define a direção; bloqueio ocorre pelo lado esquerdo do vetor.

  // Estado interativo (consumido por 10-B)
  open:      boolean          // porta/janela está aberta?
  locked:    boolean          // porta está trancada?

  // Estado destrutível (consumido por mecânica de combate, 10-C+)
  hp:        number           // 0 = indestrutível (terrain, iron por padrão)
  maxHp:     number
  destroyed: boolean          // muda o visual (linha tracejada + opacidade reduzida)
}
```

> **Nota:** O tipo `Wall` existente em `tacticalMap.ts` (`{ id, points, thickness }`) será **substituído** por `WallSegment`. O campo `walls: Wall[]` em `TacticalMap` passa a ser `walls: WallSegment[]`.

### 2.2 Valores padrão por tipo

| wallType    | move  | sense  | oneWay | locked | hp padrão | observação |
|-------------|-------|--------|--------|--------|-----------|------------|
| stone       | true  | full   | false  | false  | 100       | |
| wood        | true  | full   | false  | false  | 40        | |
| iron        | true  | full   | false  | false  | 0 (∞)     | hp=0 → indestrutível |
| magical     | true  | full   | false  | false  | variável  | passabilidade por não-magos = futuro |
| terrain     | true  | none   | true   | false  | 0 (∞)     | borda de penhasco — não bloqueia visão |
| door        | true  | full   | false  | false  | 30        | open=false por padrão |
| window      | true  | none   | false  | false  | 20        | bloqueia movimento, não bloqueia visão |
| secret_door | true  | full   | false  | false  | 30        | invisível para players até reveal |

### 2.3 Backend — entidade Go (`System_X_System`)

```go
// internal/domain/map/entity/wall_segment.go

type WallType string
const (
    WallTypeStone      WallType = "stone"
    WallTypeWood       WallType = "wood"
    WallTypeIron       WallType = "iron"
    WallTypeMagical    WallType = "magical"
    WallTypeTerrain    WallType = "terrain"
    WallTypeDoor       WallType = "door"
    WallTypeWindow     WallType = "window"
    WallTypeSecretDoor WallType = "secret_door"
)

type SenseKind string
const (
    SenseFull  SenseKind = "full"
    SenseSight SenseKind = "sight"
    SenseNone  SenseKind = "none"
)

type WallSegment struct {
    ID       uuid.UUID
    P1       [2]float64
    P2       [2]float64
    WallType WallType
    Move     bool
    Sense    SenseKind
    OneWay   bool
    Open     bool
    Locked   bool
    HP       int     // 0 = indestrutível
    MaxHP    int
    Destroyed bool
}
```

O campo `walls JSONB` da tabela `maps` já existe. Sem migração — apenas o shape do objeto muda. O mapper Go (`maps_mapper.go`) precisa ser atualizado para serializar/deserializar `[]WallSegment`.

### 2.4 Visual por tipo e estado

Renderizado com `PIXI.Graphics.moveTo/lineTo`. Sem assets externos.

| wallType    | cor hex   | espessura | dash |
|-------------|-----------|-----------|------|
| stone       | `#94a3b8` | 4px       | sólida |
| wood        | `#a16207` | 3px       | sólida |
| iron        | `#64748b` | 5px       | sólida |
| magical     | `#a855f7` | 3px       | tracejada 8,4 |
| terrain     | `#16a34a` | 2.5px     | pontilhada 3,3 |
| door        | `#f59e0b` | 3px       | sólida (fechada) / gap central (aberta) |
| window      | `#38bdf8` | 2px       | tracejada 4,2 |
| secret_door | `#94a3b8` | 4px       | sólida — **não renderizada para players** |

**Estado destrutível:**

| estado    | modificação visual |
|-----------|--------------------|
| intacta   | cor cheia, opacidade 1.0 |
| danificada (hp < 50%) | tracejada, opacidade 0.8 |
| destruída | pontilhada fina, opacidade 0.4, marcas × nos endpoints |

**Handles de edição** (visíveis só no editor, modo "paredes" ativo):
- Endpoint: círculo preenchido na cor do tipo, bordas brancas
- Segmento selecionado: linha com highlight + handles maiores
- Preview durante desenho: linha semitransparente da cor do tipo ativo

---

## 3. Sistema de snap grid-aware

Função: `snapWallPoint(worldPos: XY, grid: GridShape, threshold?: number): XY`

Itera os slots visíveis na viewport e coleta candidatos de snap. Retorna o candidato mais próximo dentro do `threshold` (padrão: 15px em screen space). Se nenhum candidato estiver dentro do threshold, retorna `worldPos` (posição livre).

**Pontos de snap por tipo de grid:**

| Grid | Candidatos |
|------|------------|
| square | Vértices (cruzamentos) + meios de cada aresta |
| hex (pointy-top) | Vértices hex + meios de aresta + **centros de hex** (necessário para paredes que atravessam o hex) |
| isométrico | Idêntico ao square. `applyTransform` (rotation + skewRatio) já deforma visualmente — o snap opera em coords de mundo (pré-transform) e o resultado é passado por `applyTransform` para renderizar. |

`Shift` mantido pressionado: desativa snap (posição livre do cursor).

---

## 4. Modelo de interação — desenho e edição

### 4.1 Modo de desenho (Drawing mode)

Ferramenta "Paredes" ativa na toolbar do editor. Botão toggle — ao ativar, o cursor muda para crosshair.

**Fluxo (híbrido polilinha → segmentos):**

1. Primeiro clique: define `p1` do primeiro segmento. Uma linha "ghost" acompanha o cursor (preview).
2. Cliques seguintes: definem `p2` do segmento atual e imediatamente `p1` do próximo. Encadeamento automático.
3. **Encerramento:**
   - `Esc` ou botão direito: encerra a polilinha no último ponto.
   - Duplo clique: encerra no ponto clicado.
   - Clique no primeiro ponto da polilinha atual: fecha o polígono (o último segmento retorna ao ponto inicial).
4. Cada segmento da polilinha é explodido em um `WallSegment` independente com UUID próprio ao confirmar.

**Snap:** aplicado a cada clique via `snapWallPoint`.

**Tipo ativo:** definido na toolbar antes de desenhar. Todos os segmentos da polilinha atual herdam o tipo ativo. Para tipos diferentes, encerre e inicie nova polilinha.

### 4.2 Edição de segmentos existentes

Com a ferramenta "Paredes" ativa:

- **Selecionar:** clique em um segmento → highlight + handles nos endpoints + painel de propriedades.
- **Mover endpoint:** drag no handle → snap aplica-se → segmento atualiza em tempo real.
- **Mover segmento inteiro:** drag no meio do segmento (não nos handles) → ambos os endpoints se movem.
- **Mudar tipo:** chips de seleção de tipo no painel de propriedades.
- **Toggle atributos:** checkboxes `move`, `oneWay`, `locked` no painel.
- **Deletar:** tecla `Delete`/`Backspace` com segmento selecionado, ou botão no painel.
- **Multi-seleção:** `Shift+clique` em múltiplos segmentos → delete em lote ou mudança de tipo em lote.

### 4.3 Undo/redo

Operações de parede integram o `editorStore` (Zustand + zundo) existente. Cada gesto completo (polilinha confirmada, endpoint movido, tipo alterado, delete) = 1 entry no histórico. Consistente com o padrão `beginGesture/endGesture` já usado para outros elementos.

---

## 5. Integração com Action e MasterAction

### 5.1 Novo sub-objeto: `Interact`

```go
// internal/domain/match/entity/action/interact.go

type InteractKind string

const (
    // Interações não-violentas
    InteractOpen      InteractKind = "open"
    InteractClose     InteractKind = "close"
    InteractToggle    InteractKind = "toggle"    // alavancas, mecanismos
    InteractLockpick  InteractKind = "lockpick"  // arrombar fechadura (requer roll de skill)
    InteractExamine   InteractKind = "examine"   // percepção em porta secreta (requer roll)

    // NOTA: Interações violentas (atacar/destruir objeto) NÃO têm kind próprio aqui.
    // Usar Action.Attack normalm ente com TargetID apontando para o WallSegment UUID.
    // O CategorizeTarget resolve o tipo e roteia para dano estrutural.
)

type Interact struct {
    Kind InteractKind
    // Nota: o UUID do objeto alvo está em Action.TargetID — não aqui.
    // TargetID é genérico: pode conter UUIDs de personagens, WallSegments,
    // FloorTiles (futuro), itens. CategorizeTarget resolve o tipo de cada UUID.
}
```

### 5.2 Adições em `Action` e `MasterAction`

```go
// action.go — adicionar campo:
Interact *Interact

// master_action.go — adicionar campo:
Interact *Interact
// MasterAction.TargetID já existe; mesma semântica genérica se aplica.
```

**Quando usar MasterAction vs Action para interações com paredes:**

| Situação | Tipo |
|---|---|
| Jogador abre porta com seu personagem | `Action { Interact: open, TargetID: [doorUUID] }` |
| Mestre usa NPC para arrombar parede | `Action { actorID: npcUUID, Attack: ..., TargetID: [wallUUID] }` |
| Mestre destrói parede diretamente (narração) | `MasterAction { Interact: open/destroy, TargetID: [wallUUID] }` |
| Mestre abre porta sem usar NPC (narração) | `MasterAction { Interact: open, TargetID: [doorUUID] }` |

> `// TODO: implementar resolução de Interact em action_mapper.go e no resolution engine quando o fluxo de partida for finalizado. Ver seção 10-C.`

### 5.3 CategorizeTarget

```go
// internal/domain/match/entity/action/categorize.go

type TargetKind string

const (
    TargetKindCharacter   TargetKind = "character"
    TargetKindWallSegment TargetKind = "wall_segment"
    TargetKindFloorTile   TargetKind = "floor_tile"   // TODO: Fase futura
    TargetKindItem        TargetKind = "item"          // TODO: Fase futura
    TargetKindUnknown     TargetKind = "unknown"
)

// CategorizeTarget resolve o tipo de um UUID consultando as coleções do MatchSession.
// Retorna TargetKindUnknown se o UUID não for encontrado em nenhuma coleção —
// o caller deve tratar como erro de validação.
func CategorizeTarget(id uuid.UUID, session MatchSessionReader) TargetKind
```

**Fluxo de resolução no engine:**

```go
for _, targetID := range action.TargetID {
    switch CategorizeTarget(targetID, session) {
    case TargetKindCharacter:
        if action.Attack != nil {
            resolveAttackOnCharacter(action, targetID) // reações, dodge, defense...
        }
        // Interact em personagem = futuro (ex: curar, aplicar status)

    case TargetKindWallSegment:
        if action.Attack != nil {
            applyStructuralDamage(targetID, action.Attack) // wall.hp -= dano; checa destruction
        }
        if action.Interact != nil {
            resolveWallInteraction(targetID, action.Interact) // open/close/lockpick/examine
        }

    case TargetKindFloorTile:
        // TODO: Fase futura — dano estrutural ao piso
    }
}
```

---

## 6. Visibilidade e LOS (Line of Sight)

### 6.1 Invariante de segurança

**Paredes `secret_door` e peças com `visible=false` nunca chegam no payload WS de um jogador que não pode vê-las.** O backend filtra o estado do mapa antes de broadcast. O cliente do jogador não recebe dados que não deveria ver — a filtragem não é responsabilidade do frontend.

### 6.2 Atributo `sense` por segmento

Cada `WallSegment.sense` define o que o segmento bloqueia em termos de percepção:

- `full`: bloqueia visão, audição, todos os sentidos
- `sight`: bloqueia só visão (audição, por exemplo, passa)
- `none`: sem bloqueio de percepção (apenas obstáculo físico ou decorativo)

### 6.3 Arquitetura LOS — Fase 10-D

> **⚠️ Brainstorm obrigatório antes da Fase 10-D.** Ver seção 8.4.

O shadow casting roda **server-side em Go**. O cliente recebe apenas o estado visível filtrado. O fog decorativo no Pixi serve apenas para indicar ao jogador o que é "área não revelada ainda" — os dados reais nunca chegaram.

```
Para cada mensagem WS que inclui estado do mapa:
  Para cada player conectado:
    computeVisibleArea(player.pieces, map.walls, SenseKind)
    → filtra walls, pieces, decorations pelo visible area
    → envia payload filtrado APENAS para esse player
```

> `// TODO: A arquitetura atual de room.go usa broadcast para todos os clientes.`
> `// LOS requer broadcast POR PLAYER com payload filtrado individualmente.`
> `// Revisar arquitetura de broadcast quando o fluxo final da partida for definido.`
> `// Referência: internal/app/game/room.go`

---

## 7. Documentação do subsistema

### 7.1 Dev docs (já existem — atualizar)

- `System_X_System_React/docs/dev/tactical-map/overview.md` — adicionar seção de paredes
- `System_X_System_React/docs/dev/tactical-map/testing.md` — casos de teste para walls
- `System_X_System/docs/dev/api/` — novo arquivo `walls.md` com contratos REST (walls são parte do mapa, sem endpoints próprios — documentar o shape dentro de `PUT /maps/:id`)

### 7.2 Player-facing docs

> `// TODO: criar docs/player/walls.md explicando ao jogador os tipos de parede, como interagir, o que bloqueia movimento, como abrir portas, como destruir paredes etc.`
> `// Seguir o padrão de documentação para player que existir na estrutura do backend.`

---

## 8. Decomposição em 4 sub-fases

Cada sub-fase = sessão de planejamento (Sonnet/Opus) + sessão de implementação (Sonnet) + 1 PR por repo afetado.

---

### Fase 10 — Core visual + editor de paredes

> **Sessão de brainstorm:** Não necessária. Este documento é suficiente.
> **Sessão de planejamento:** Invocar `writing-plans` diretamente neste documento.

**Escopo:** Tudo o que permite ao mestre desenhar, editar e visualizar paredes no editor. Nenhuma mecânica de jogo — só visual e persistência.

**Entregáveis frontend (`System_X_System_React`):**

- `src/types/tacticalMap.ts`: substituir `Wall` por `WallSegment` com todos os campos da seção 2.1
- `src/features/tactical-map/utils/walls.ts`: `snapWallPoint(worldPos, grid, threshold?)` — grid-aware (square, hex, isométrico)
- `src/features/tactical-map/utils/walls.ts`: `explodePolyline(points, wallType, defaults)` → `WallSegment[]`
- `src/components/organisms/WallsLayer.tsx`: camada Pixi — renderiza `WallSegment[]` com cor/espessura/dash por tipo e estado
- `src/components/organisms/WallsLayer.tsx`: handles de edição (endpoint handles, segmento selecionado)
- `src/components/molecules/WallConfigPanel.tsx`: painel de propriedades do segmento selecionado (tipo, move, oneWay, locked)
- `src/components/molecules/WallTypeChips.tsx`: seleção do tipo ativo antes de desenhar
- `TacticalMapEditor.tsx`: integrar ferramenta "Paredes" na toolbar; drawing mode; undo/redo
- `editorStore.ts`: actions `addWallSegment`, `updateWallSegment`, `removeWallSegment`, `setWallSelection`
- Testes unit: `snapWallPoint` (square, hex, iso), `explodePolyline`

**Entregáveis backend (`System_X_System`):**

- `internal/domain/map/entity/wall_segment.go`: entidade `WallSegment` com todos os campos da seção 2.3 + enums `WallType`, `SenseKind`
- `internal/gateway/pg/maps_mapper.go`: serializar/deserializar `[]WallSegment` no campo `walls JSONB`
- `internal/domain/map/service/map_validator.go`: validar `WallSegment` (p1 ≠ p2, wallType válido, hp ≥ 0)
- `System_X_System/docs/dev/api/` — `maps.md` atualizado: shape de `walls` no request/response de `PUT /maps/:id`
- `docs/documentation-map.yaml` — registro da atualização

**Viewer (`TacticalMapViewer.tsx`):**

- Renderiza paredes com `WallsLayer` (read-only)
- Portas secretas (`secret_door`): não renderizadas para jogadores. O backend já filtra; o frontend simplesmente não recebe o segmento.

**Critério de pronto:** Mestre desenha uma sala completa (4 paredes stone), adiciona uma porta (door) e uma janela (window), salva, recarrega — tudo persiste com os tipos corretos. Viewer (jogador) não vê secret_doors. Editor suporta undo/redo de cada operação de parede.

---

### Fase 10-B — Estado interativo + bloqueio de movimento

> **Sessão de brainstorm:** Mínima (≤ 15 min). Antes de invocar `writing-plans`, confirmar:
> 1. Algoritmo de interseção geométrica para bloqueio de movimento (segmento vs segmento, considerando `oneWay`)
> 2. Como o estado `open/locked` de portas persiste durante a partida vs no snapshot do mapa
> 3. Integração de `MasterAction.Interact` com o WS existente (novo `MsgType`?)

**Escopo:** Portas e janelas ficam interativas. Mestre pode alterar estado via `MasterAction`. Bloqueio de movimento é validado no frontend e no backend.

**Entregáveis frontend:**

- `WallsLayer.tsx`: render de porta aberta (gap central no segmento) vs fechada; lock icon quando `locked=true`
- `TacticalMapViewer.tsx`: clique em porta → `Action.Interact{ Kind: open }` enviado via WS (intent do jogador — ver seção 5.2)
- `useMatchMapWs.ts` (já existe): processar novos `MsgType` de wall state change
- Bloqueio de movimento: `isMovementBlocked(from, to, walls)` — checa interseção do path com `wall.move=true` e `!wall.open`; usado no viewer para desabilitar slots inacessíveis

**Entregáveis backend:**

- `internal/domain/match/entity/action/interact.go`: `Interact` + `InteractKind` enum (seção 5.1)
- `internal/domain/match/entity/action/action.go`: campo `Interact *Interact`
- `internal/domain/match/entity/action/master_action.go`: campo `Interact *Interact`
- `internal/app/game/message.go`: `InteractPayload`, `MsgTypeWallStateChanged` (server→client)
- `internal/app/game/action_mapper.go`: mapear `InteractPayload` → `action.Interact`
- `enqueue_action` handler: validar `Interact` em paredes (está dentro do mapa? wallType suporta a interação?)
- `enqueue_master_action` handler: processar `MasterAction.Interact` → atualiza estado in-memory + broadcast
- Validação de bloqueio de movimento no `enqueue_action` para `action.Move`: verifica path contra walls com `move=true` e `!open`

> `// TODO: O fluxo de enqueue_action e processamento de turns mudará quando os fluxos de partida`
> `// forem implementados. Revisitar esta validação de bloqueio de movimento nessa ocasião.`
> `// Referência: internal/app/game/action_mapper.go (TODO existente)`

**Critério de pronto:** Jogador clica em porta fechada → intent enviado → mestre aprova → porta abre para todos via WS. Peça não consegue mover para slot bloqueado por parede `move=true` fechada. Mestre abre/fecha porta diretamente via `MasterAction` no editor da partida.

---

### Fase 10-C — Interações como player Action (domínio + WS)

> **Sessão de brainstorm:** Necessária, breve (30–45 min). Invocar `brainstorming` antes de `writing-plans`.
> **Motivo:** O fluxo de `enqueue_action → turn → resolution` ainda está sendo definido (vários TODOs em `action_mapper.go`). Antes de implementar `CategorizeTarget` e a resolução de dano estrutural, é preciso alinhar com o estado do fluxo de partida no momento.
> **Foco do brainstorm:** (1) Como `Action.Attack` em `WallSegment` entra no sistema de turns? (2) `CategorizeTarget` fica no domain ou no application layer? (3) Dano estrutural em parede durante turno: atualiza `Map` snapshot direto ou persiste como evento no `Turn`?

**Escopo:** Atacar uma parede, arrombar uma fechadura ou examinar uma porta secreta são `Action`s do jogador resolvidas pelo sistema de combate.

**Entregáveis backend:**

- `internal/domain/match/entity/action/categorize.go`: `TargetKind` + `CategorizeTarget()`
- `internal/domain/match/entity/action/interact.go`: `InteractKind` completo (já criado em 10-B, completar se necessário)
- `internal/domain/match/entity/action/action.go`: garantir que `Interact` (do 10-B) está corretamente integrado
- `internal/app/game/action_mapper.go`: mapear `InteractPayload` e `AttackPayload` contra `WallSegment` targets
  - `// TODO: implementar resolução completa quando fluxo de turns for finalizado`
- `internal/domain/match/service/`: `applyStructuralDamage(wallID, attack)` — reduz `wall.hp`, marca `destroyed` se hp≤0
- `internal/domain/match/service/`: `resolveWallInteraction(wallID, interact)` — open/close/lockpick/examine

**Entregáveis frontend:**

- `TacticalMapViewer.tsx`: tap em `WallSegment` → action picker ("Abrir", "Arrombar fechadura", "Examinar")
- Action picker considera `wallType` para mostrar opções relevantes (door → "Abrir"; wood/stone com `hp>0` → "Atacar")
- `useMatchMapWs.ts`: processar `wall_damaged` e `wall_destroyed` events → atualiza estado local
- Visual: transição de estado (intacta → danificada → destruída) animada ~300ms

> `// TODO: Fluxo de resolução de Action em objeto de mapa depende da arquitetura final do turn engine.`
> `// Este TODO bloqueia a implementação completa desta fase. Revisar com o dev do fluxo de partida.`

**Player docs:**
- `docs/player/walls.md` (criar): explicar tipos de parede, o que o jogador pode fazer, custo de ação

**Critério de pronto:** Jogador seleciona parede de madeira → "Atacar" → intent enviado com `Attack` + `TargetID=[wallUUID]` → dano aplicado → jogadores veem estado "danificada"; ao hp=0 → "destruída" com visual tracejado.

---

### Fase 10-D — Visão / Line of Sight / Fog of War

> **Sessão de brainstorm:** Obrigatória, completa. Invocar `brainstorming` com nível de detalhe **alto**.
> **Pré-requisito:** Aguardar estabilização do fluxo de partida (turns, rounds, broadcast WS). Esta fase depende da arquitetura de broadcast por-player que será definida com o fluxo de partida.
> **Motivo:** (1) Shadow casting em Go requer escolha de algoritmo (recursive shadowcasting, RPAS, ou variante). (2) A mudança de broadcast para all → por-player é uma mudança arquitetural significativa em `room.go`. (3) Performance: re-computar LOS para cada player a cada evento de mapa pode ser custoso — precisa de estratégia de cache/invalidação. (4) Portas secretas: o reveal via `examine` muda a visibilidade permanentemente? Ou por sessão?
> **Foco do brainstorm:** (1) Algoritmo de shadow casting escolhido. (2) Quando re-computar LOS (só no move? em qualquer mudança de mapa?). (3) Arquitetura do filtro por-player no WS. (4) Fog decorativo no Pixi: static fog texture vs dynamic rendering.

**Escopo:** Cada jogador vê apenas o que seus personagens podem perceber. O backend filtra o payload WS por-player antes de enviar. Secret doors são reveladas por roll de `examine`.

**Entregáveis backend:**

- `internal/domain/match/service/visibility.go`: `ComputeVisibleArea(pieces, walls, SenseKind) → VisibilityMask`
- Algoritmo: **recursive shadowcasting** (referência: RedBlobGames) — eficiente para grids até 60×60
- `internal/app/game/room.go`: refatorar de broadcast-all para broadcast-per-player com payload filtrado
  - `// TODO: Esta é a mudança arquitetural central. Coordenar com o dev do fluxo de partida.`
- Filter: antes de enviar `map_state` ou `wall_state_changed` para cada player, aplicar `FilterMapState(fullState, visibleArea)`
- `secret_door`: incluso no payload filtrado apenas se revelado (jogador passou em `examine` check) ou mestre forçou reveal

**Entregáveis frontend:**

- `WallsLayer.tsx`: fog overlay — `PIXI.Graphics` preenchendo área não-visível com cor escura semi-transparente
- Fog é puramente decorativo: representa "você não vê isso" para o que o backend não enviou
- `TacticalMapViewer.tsx`: distinção visual entre "fora do range" vs "bloqueado por parede"

**Critério de pronto:** Jogador A com peça em (3,3) não recebe no payload WS as peças e paredes do lado oposto de uma parede `sense=full`. Ao mover para (3,4), novo payload inclui o que agora é visível. Fog renderizado no canvas delimita visualmente a área revelada.

---

## 9. Capacidades futuras (além das 4 sub-fases)

- **FloorTile destrutível:** piso que pode ser danificado por ataques — `CategorizeTarget` já tem `TargetKindFloorTile` reservado
- **Paredes mágicas passáveis por não-magos:** `wallType=magical` + `magical_passable: bool` — verificado no bloqueio de movimento contra o atributo do personagem
- **Efeito de área no piso:** slots danificados aumentam custo de movimento (difícil terrain)
- **Iluminação:** tokens emitindo luz; interação com `sense` das paredes; escuridão como dimensão adicional de visibilidade
- **Threshold walls** (FoundryVTT v11): parede que só ativa LOS dentro de um raio — útil para janelas e venezianas
- **Audição vs visão:** `SenseKind = "sight"` já reserva espaço para audição passar por paredes `sense=sight`
