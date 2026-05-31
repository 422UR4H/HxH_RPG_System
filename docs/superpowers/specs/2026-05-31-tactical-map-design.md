# Tactical Map System — Design Spec

**Date:** 2026-05-31
**Status:** Approved — ready for phased implementation
**Scope:** Cross-repo (`System_X_System_React` + `System_X_System`)
**Audience:** Future planning/implementation sessions (humans + AI). Self-contained — readable without the brainstorm transcript.

---

## 1. Visão geral

Introduzir um **sistema de mapa tático** no produto. O mestre cria mapas (campo + malha + imagem de fundo + peças posicionadas) em uma página dedicada, dentro de uma campanha, e anexa um mapa a uma partida. Durante a partida, jogadores e mestre veem o campo; o mestre opera as peças e os jogadores enviam intenções (intent) de movimento/ataque que o mestre aprova.

A entrega completa do sistema é grande. Está dividida em **12 fases sequenciais**, cada uma em sua sessão e seu PR. Esta é a primeira vez que a stack gráfica entra no projeto.

### O que define "pronto" do sistema (visão de produto)

- Mestre cria e edita mapas reutilizáveis por campanha (malha quadrada ou hexagonal, imagem de fundo, peças posicionadas)
- Mestre pode regular distorção isométrica e rotação da malha
- Jogadores e mestre veem o mapa em tempo real durante a partida; mestre publica mudanças
- Jogadores interagem por tap (slot, peça) e enviam intent de movimento
- Mestre pode desenhar paredes/obstáculos e colocar decorações (PNGs livres)
- Persistência: mapas salvos no backend; mudanças durante a partida ficam no event log existente (`Action`/`MasterAction`/`Turn`/`Round`)

### Restrições e premissas

- **Dispositivos**: jogadores rodam em mobile/tablet/desktop com paridade de visualização e intent. Edição completa (drag-and-drop, ajustes finos) é desktop; edição básica em mobile é alvo de polish.
- **Escala**: até ~60×60 slots e ~50 peças por mapa em pior caso real.
- **Sincronização**: turn-based, discreta. Não há streaming. Mestre edita localmente; ações se tornam públicas via WS quando o mestre publica.
- **Pré-produção**: sem usuários reais. Calibração é por **excelência técnica**, não MVP-cut.

---

## 2. Stack escolhida e racional

### Bibliotecas novas

| Lib | Versão alvo | Para quê | Bundle (gz) |
|---|---|---|---|
| `pixi.js` | v8 | Renderer 2D acelerado por GPU (WebGL com fallback Canvas2D) | ~180KB |
| `@pixi/react` | v8 (compat com pixi.js v8) | JSX por cima do PixiJS — declarativo, plugável no React 19 | ~30KB |
| `pixi-viewport` | v6 | Pan, pinch-zoom, wheel-zoom prontos | ~25KB |
| `zustand` | v5 | Store local do editor (separado de React Query/server state) | ~3KB |
| `zundo` | v2 | Middleware de undo/redo para Zustand | ~1KB |
| `immer` | v10 | Estado imutável escrito como mutável; gera JSON Patch automático | ~6KB |

**Total adicional ~245KB gzip**, lazy-loaded apenas nas rotas do mapa (`CreateMapPage`, `EditMapPage`, `GamePage`). Sem custo pra quem não abre o mapa.

### Por que PixiJS (e não Konva ou SVG)

| Critério | PixiJS | Konva | SVG |
|---|---|---|---|
| Renderiza 60×60 + bg + futuros PNGs em mobile | ✅ trivial (sprite batching GPU) | ⚠️ degrada acima de ~500 shapes | ❌ inviável (3600+ DOM) |
| Pan + pinch-zoom out-of-the-box | ✅ pixi-viewport | ❌ implementação manual | ❌ implementação manual |
| Distorção isométrica + rotação | ✅ transformações afins nativas | ⚠️ skew possível, rotação combinada difícil | ⚠️ CSS transform lento em escala |
| Hit-test por sprite | ✅ nativo | ✅ nativo | ✅ nativo |
| Touch em mobile | ✅ excelente | ✅ bom | ⚠️ manual em escala |
| Comprovação na vida real | FoundryVTT (líder de mercado em VTTs) | UIs interativas em geral | Diagramas, dashboards |
| React integration | `@pixi/react` v8 ativo | `react-konva` estabelecido | trivial |
| Curva de aprendizado | maior | menor | menor |

**Decisão**: PixiJS. O custo da curva inicial paga em todas as 12 fases. As alternativas levariam a refator quando entrarem isométrico + PNGs decorativos.

### Por que Zustand + zundo + immer

- React Query continua dono do estado servidor (mapa carregado, mutations).
- Zustand cobre o estado **local complexo** do editor (rascunho, ferramenta ativa, seleção) com selectors finos — sem re-render em cascata como Context.
- zundo é middleware oficial pra histórico (undo/redo), sem código custom.
- immer permite escrever "mutável" no reducer e ele gera o JSON Patch — útil pra delta WS futuro.

Alternativa rejeitada: `useReducer` + Context. Verbosidade alta pra histórico, re-render desnecessário com Context.

---

## 3. Arquitetura

### Padrão híbrido: 1 núcleo + 2 cascas

Três componentes, responsabilidades isoladas:

1. **`organisms/TacticalMapStage`** — núcleo de renderização. Recebe `map: TacticalMap` por prop. Não sabe se é editor ou viewer. Não tem toolbar nem lógica de input alto-nível. Responsabilidade única: **transformar dados em pixels**.

2. **`features/tactical-map/TacticalMapEditor`** — casca do mestre. Envolve o `Stage` e adiciona toolbar, drag-and-drop, roster lateral, salvamento, undo/redo. Usada em `CreateMapPage` e `EditMapPage`.

3. **`features/tactical-map/TacticalMapViewer`** — casca do jogador (e do mestre em modo "apresentação"). Envolve o `Stage` e adiciona tap-to-move (intent), tap-to-target, listagem read-only de peças. Usada em `GamePage`.

**Por que híbrido** (vs componente único com prop `mode`, vs dois componentes independentes): único arquivo cresce indefinidamente e mistura responsabilidades; dois independentes divergem visualmente com o tempo. Híbrido garante paridade visual (render escrito uma vez) e isolamento de comportamento.

### Camadas da cena Pixi

Dentro do `TacticalMapStage`, a cena é estruturada em camadas explícitas — cada uma é um `Container` Pixi:

```
pixi-viewport (pan / pinch-zoom / wheel-zoom)
└─ worldContainer
   ├─ Layer 0: bgImage          (Sprite — imagem de fundo)
   ├─ Layer 1: grid             (Graphics — linhas da malha)
   ├─ Layer 2: decorations      [futuro — Fase 11]
   ├─ Layer 3: pieces           (Sprite + sombra + badge Z)
   ├─ Layer 4: walls            [futuro — Fase 10]
   └─ Layer 5: overlay          (hover, seleção, highlight de slot)
```

Camadas reservadas desde a Fase 0 — `decorations[]`, `walls[]`, `items[]` existem no estado com default `[]`. Quando a fase chega, é só renderizar; sem migração.

### Estrutura de arquivos novos

#### Frontend (`System_X_System_React/`)

```
src/
├─ features/tactical-map/
│  ├─ TacticalMapEditor.tsx           ← casca mestre
│  ├─ TacticalMapViewer.tsx           ← casca jogador
│  ├─ hooks/
│  │  ├─ useTacticalMap.ts            ← orquestra store + queries
│  │  └─ useMatchMapWs.ts             ← análogo a useLobbyWs (Fase 7)
│  ├─ store/
│  │  └─ editorStore.ts               ← Zustand + zundo + immer
│  └─ utils/
│     ├─ coords.ts                    ← slot ↔ world (square + hex + isométrico)
│     ├─ hex.ts                       ← axial coords, vizinhos, distância
│     └─ patches.ts                   ← geração e aplicação de JSON Patch
├─ components/
│  ├─ organisms/
│  │  ├─ TacticalMapStage.tsx         ← núcleo Pixi
│  │  ├─ MapEditorToolbar.tsx
│  │  └─ CharacterRoster.tsx
│  └─ molecules/
│     ├─ GridConfigPanel.tsx
│     └─ BgImageAdjuster.tsx
├─ pages/
│  ├─ CreateMapPage.tsx
│  ├─ EditMapPage.tsx
│  └─ GamePage.tsx                    ← substitui placeholder existente (Fase 6)
├─ services/
│  └─ mapsService.ts                  ← REST client com snake↔camel
├─ hooks/
│  ├─ useMaps.ts                      ← lista mapas da campanha
│  ├─ useMap.ts                       ← carrega um mapa
│  ├─ useCreateMap.ts
│  └─ useUpdateMap.ts
└─ types/
   └─ tacticalMap.ts                  ← TacticalMap, GridShape, Piece, etc.
```

#### Backend (`System_X_System/`)

```
internal/
├─ domain/map/
│  ├─ entity/
│  │  ├─ map.go                       ← TacticalMap (snapshot)
│  │  ├─ grid.go                      ← GridShape (kind, dims, skew, rotation, line style)
│  │  ├─ piece.go                     ← Piece (character_id, coord, z, visible)
│  │  └─ ...                          ← walls, decorations, items (placeholders)
│  └─ service/
│     └─ map_validator.go             ← regras de domínio (slot dentro do grid, etc.)
├─ application/map/
│  ├─ create_map.go
│  ├─ get_map.go
│  ├─ update_map.go
│  ├─ list_maps_by_campaign.go
│  └─ delete_map.go
├─ gateway/pg/
│  ├─ maps_repository.go
│  └─ maps_mapper.go                  ← row → entity
└─ app/api/
   └─ maps_handler.go                 ← endpoints REST
```

Em fases posteriores (7+), `app/game/` e `domain/match/entity/action/` ganham extensões — detalhadas na seção de fases.

---

## 4. Modelo de dados

### Tipos no frontend

```ts
// src/types/tacticalMap.ts

// Coordenadas
export type SquareCoord = { kind: 'square'; col: number; row: number };
export type HexCoord    = { kind: 'hex';    q: number;   r: number };  // axial
export type SlotCoord   = SquareCoord | HexCoord;

export type PieceCoord = {
  slot: SlotCoord;
  z: number;                         // altura "virtual" em metros; 0 = chão
};

// Malha
export type GridKind  = 'square' | 'hex';
export type LineStyle = 'solid' | 'dashed';

export type GridShape = {
  kind: GridKind;
  cols: number;
  rows: number;
  cellSize: number;                  // tamanho base do slot em px do mundo
  skewRatio: number;                 // 1 = top-down; <1 = isométrico (1:2 = 0.5)
  rotation: number;                  // graus; default 0
  color: string;                     // cor da linha (token do projeto)
  opacity: number;                   // 0-1
  lineStyle: LineStyle;              // renderer respeita desde Fase 0; UI do toggle = polish futuro
};

// Imagem de fundo
export type BgImage = {
  url: string;                       // Cloudflare-hosted ou externa (qualquer URL)
  x: number; y: number;              // canto sup. esq. em coords do mundo
  width: number; height: number;
  rotation: number;
  opacity: number;                   // útil já agora (legibilidade da malha) e necessário pra multi-andar
} | null;

// Peça
export type Piece = {
  id: string;                        // uuid próprio da peça no mapa (não é character.id)
  characterId: string;               // FK pra CharacterSheet (jogador ou NPC)
  coord: PieceCoord;
  visible: boolean;                  // Fase 7. Default true. Evolui pra `visibleTo: 'all' | UserId[]`
};

// Capacidades futuras — declaradas com [] desde a Fase 0
export type Wall       = { id: string; points: Array<[number, number]>; thickness: number };          // Fase 10
export type Decoration = { id: string; url: string; x: number; y: number; width: number; height: number; rotation: number; zOrder: number; opacity: number }; // Fase 11
export type MapItem    = { id: string; itemDefId: string; coord: SlotCoord };                          // sistema de itens (futuro paralelo)

// Raiz
export type TacticalMap = {
  id: string;
  campaignId: string;
  name: string;
  description?: string;
  grid: GridShape;
  bg: BgImage;
  pieces: Piece[];
  walls: Wall[];                     // []
  decorations: Decoration[];         // []
  items: MapItem[];                  // []
  createdAt: string;                 // ISO
  updatedAt: string;                 // ISO
};
```

### Backend: tabela `maps`

```sql
CREATE TABLE maps (
  id          UUID         PRIMARY KEY,
  campaign_id UUID         NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
  name        VARCHAR(255) NOT NULL,
  description TEXT,
  grid        JSONB        NOT NULL,
  bg          JSONB,
  pieces      JSONB        NOT NULL DEFAULT '[]',
  walls       JSONB        NOT NULL DEFAULT '[]',
  decorations JSONB        NOT NULL DEFAULT '[]',
  items       JSONB        NOT NULL DEFAULT '[]',
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
  updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_maps_campaign_id ON maps(campaign_id);
```

**Por que JSONB e não tabelas relacionais separadas para pieces/walls:** o mapa é editado como agregado único (mestre salva o estado inteiro). Não há consultas relacionais úteis sobre peças isoladas fora do contexto do mapa. Mudanças incrementais durante a partida não passam pela tabela `maps` — passam pelo event log (`Action`/`MasterAction`/`Turn`).

### Backend: tabela `match_maps` (Fase 6)

```sql
CREATE TABLE match_maps (
  match_id    UUID         PRIMARY KEY REFERENCES matches(id) ON DELETE CASCADE,
  map_id      UUID         NOT NULL REFERENCES maps(id) ON DELETE RESTRICT,
  attached_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
```

Mapa não pode ser deletado enquanto anexado a uma partida — `ON DELETE RESTRICT`.

### Decisões embutidas

| Decisão | Por quê |
|---|---|
| `SlotCoord` como discriminated union | Compilador força tratamento exaustivo; nenhuma ambiguidade entre square e hex |
| Hex em coords axiais (q, r) | Padrão da literatura (RedBlobGames); matemática de distância/vizinhos trivial |
| `skewRatio` e `rotation` desde a Fase 0 com defaults | Quando a UI ligar (Fase 9), zero refator no math |
| `walls`, `decorations`, `items` sempre `[]` | Zero migração quando suas fases chegarem |
| `Piece.id` separado de `characterId` | Peça é instância no mapa; um personagem pode aparecer em zero, um ou vários mapas |
| `Piece.visible: boolean` por enquanto | Evolui pra `visibleTo: 'all' \| UserId[]` quando visibilidade por jogador entrar |
| `BgImage.opacity` mantida | Útil já agora (legibilidade da malha) e necessária pra multi-andar (transparência entre níveis) |
| `BgImage.url` aceita Cloudflare ou externa | Reuso do padrão `ImagePickerModal` existente (upload + paste URL) |
| `lineStyle: 'solid' \| 'dashed'` desde a Fase 0 | Renderer respeita; UI de seleção fica como polish futuro |

---

## 5. Modelo de interação

### Sistema de coordenadas — três espaços

| Espaço | Forma | Quem usa |
|---|---|---|
| **Slot** | `{kind, col, row}` (quad) ou `{kind, q, r}` (hex axial) | Estado canônico; regra de jogo (distância, vizinhos) |
| **World** | `{x, y}` em pixels do mundo (independente de zoom/pan) | Renderização Pixi |
| **Screen** | pixels da tela do navegador | Input do usuário (tap, click, drag) |

**Funções puras** em `features/tactical-map/utils/coords.ts`:

```ts
slotToWorld(slot: SlotCoord, grid: GridShape): { x: number; y: number };
worldToSlot(world: { x: number; y: number }, grid: GridShape): SlotCoord;
// screen ↔ world: resolvido pelo pixi-viewport
```

Ambas aplicam `skewRatio` e `rotation` desde o dia 1, com defaults `1` e `0`. A UI de distorção/rotação (Fase 9) só liga os controles; matemática já está lá.

### Estado do editor — Zustand

```ts
// features/tactical-map/store/editorStore.ts (esqueleto)

type ToolKind = 'grid' | 'bg' | 'pieces' | 'walls' | 'decorations';
type Selection = { kind: 'piece'; id: string } | { kind: 'decoration'; id: string } | null;

type EditorState = {
  map: TacticalMap;          // rascunho local
  isDirty: boolean;
  activeTool: ToolKind;
  selection: Selection;

  // actions
  setGrid: (grid: GridShape) => void;
  setBg: (bg: BgImage | null) => void;
  placePiece: (characterId: string, slot: SlotCoord) => void;
  movePiece: (pieceId: string, slot: SlotCoord) => void;
  setPieceZ: (pieceId: string, z: number) => void;
  removePiece: (pieceId: string) => void;
  setActiveTool: (tool: ToolKind) => void;
  setSelection: (sel: Selection) => void;
  // futuro: addWall, addDecoration, ...
};
```

Middleware aplicados:
- **`zundo`** envolve as actions — `undo()` / `redo()` viram disponíveis no hook
- **`immer`** envolve cada reducer — mutação aparente, imutabilidade real

Selectors:
- Hooks consumidores leem `slice` específica (ex: `usePieces()` retorna `map.pieces`) — sem re-render em cascata

### Fluxo de save (explícito)

Mestre clica **"Salvar"** (ou Ctrl/Cmd+S) → React Query mutation → backend persiste → store marca `isDirty = false`.

**Proteção contra perda**: rascunho salva em `localStorage[\`tactical-map-draft:${mapId}\`]` ao mudar; restaurado ao reabrir.

**Por que não autosave**: combina com o modelo "publicação discreta" (mestre decide quando algo fica oficial), evita spam ao backend em mapas grandes, mantém undo bem comportado.

### Sincronização durante a partida (Fase 7+)

Modelo turn-based, **discreto**, alinhado com o `MatchSession` existente em `cmd/game/`:

1. Mestre edita localmente. Nada vai via WS durante a edição.
2. Mestre executa uma **ação publicável** (mover peça oficialmente, revelar peça oculta, etc).
3. Frontend serializa essa ação como `Action` ou `MasterAction` (formato existente do backend) e envia via WS.
4. Backend valida (autoria, mapa anexado, slot válido), persiste no `Turn`/`Round`, broadcasta a mesma mensagem pros clientes da sala.
5. Receptores aplicam a ação no estado local (via `immer`, gerando o JSON Patch internamente para auditoria).
6. UI anima a transição entre estados (lerp de A→B em ~300ms).

**Decisão estratégica: Action/MasterAction == delta WS == persistência.** Uma única representação, três usos. Sem inventar protocolo de delta separado. Se algum dia detectarmos que ajustes ad-hoc do mestre (ex: trocar cor de label) ficam estranhos como `MasterAction`, aí sim consideramos um canal "delta puro". Por ora, unificado.

### Mensagens WS (Fase 7)

| Tipo | Direção | Conteúdo |
|---|---|---|
| `action_submitted` | jogador → server | `Action` serializada |
| `master_action_executed` | mestre → server | `MasterAction` serializada |
| `action_published` | server → todos | `Action` confirmada (pós-validação do mestre, Fase 8) |
| `master_action_broadcast` | server → todos | `MasterAction` aplicada |
| `map_full_state` | server → cliente que reconectou | `TacticalMap` completo |

### Hit-test

PixiJS oferece hit-test nativo por sprite (`eventMode: 'static'` + `hitArea`):

- **Tap em peça**: sprite da peça intercepta; retorna `pieceId`.
- **Tap em slot vazio**: nenhuma peça intercepta; container `worldContainer` recebe; calcula `worldToSlot(eventPos, grid)`.
- **Drag de peça (desktop)**: `pointerdown` no sprite → `pointermove` reposiciona sprite localmente (sem rerender React) → `pointerup` calcula `worldToSlot` no destino → dispara action na store.
- **Tap em decoração/parede** (futuro): mesmo padrão por camada.

Pan e pinch-zoom isolados no `pixi-viewport` — não interferem com eventos das sprites internas.

### MasterAction: `Round` vs `Turn`

Decisão de design existente no backend, **mantida intencionalmente**:

- **`Round`-level (default)**: a `MasterAction` afeta o estado geral, fora de um turno específico. Aplica-se à maioria das ações de mapa (`AddPiece`, `RemovePiece`, `SetVisibility`, `Move` de NPC fora de turno) e a eventos gerais (spawn de NPC, clima, anúncio).
- **`Turn`-level (explícito)**: a `MasterAction` acontece **dentro** de um turno e interage com ele. O frontend terá um botão pro mestre marcar isso explicitamente. **O backend não infere; confia na origem do payload.**

### "Action override" — fluxo separado, NÃO confundir com MasterAction

Existe (planejado, ainda sem implementação no backend) um fluxo distinto onde o mestre **modifica** a `Action` de um jogador (ex: ajustar roll, penalizar) após narração:

1. Jogador envia `Action`.
2. Mestre abre da fila.
3. Jogador narra.
4. Mestre, com base na narração, sobrescreve a `Action` original — gera uma **nova `Action` modificada**, vinculada por referência à original.
5. Original deve persistir em tabela de histórico (a criar; clean slate confirmado).

Isso **não é** `MasterAction`. É edição de ciclo de vida de `Action`. Documentado aqui para que ninguém conflate no futuro.

---

## 6. Decomposição em 12 fases

Cada fase = uma sessão de planejamento (Opus) + uma sessão de implementação (Sonnet) + 1 PR. PRs pequenos, revisáveis, construindo em cima do anterior sem refator.

### Fase 0 — Setup (walking skeleton) — frontend apenas

**Escopo**: terreno preparado. Nenhuma feature de usuário; todo o resto encaixa sem refator depois.

**Entregáveis**:
- Instalar `pixi.js`, `@pixi/react`, `pixi-viewport`, `zustand`, `zundo`, `immer` (pin de versão no package.json)
- `src/types/tacticalMap.ts` com **todos** os tipos da seção 4 (incluindo placeholders futuros)
- `src/features/tactical-map/utils/coords.ts` + `hex.ts` — funções puras para slot↔world (square e hex axial), com `skewRatio` e `rotation` aplicados
- `src/features/tactical-map/store/editorStore.ts` — Zustand + zundo + immer com actions vazias mas tipadas
- `src/components/organisms/TacticalMapStage.tsx` — núcleo Pixi que aceita `map: TacticalMap` por prop e renderiza grid + bg + pieces (sem interação)
- `src/features/tactical-map/TacticalMapEditor.tsx` + `TacticalMapViewer.tsx` — cascas vazias, só importam o Stage
- Rota dev `/dev/tactical-map-demo` com mapa hardcoded — smoke test visual
- Configuração de teste: mock do WebGL em `vitest.setup.ts`; testes unit das utils de coords
- Lazy-loading da rota via `React.lazy` (validar que bundle principal não inclui pixi)

**Critério de pronto**: rota `/dev/tactical-map-demo` mostra grade quadrada 10×10 + 2 peças placeholder + bg image fictícia, pan/pinch-zoom funcionando.

### Fase 1 — Persistência + listagem de mapas — backend + frontend

**Escopo**: entidade `Map` no backend, CRUD REST, listagem no frontend dentro da `CampaignPage`.

**Entregáveis backend (`System_X_System`)**:
- `internal/domain/map/entity/` — entities (Map, GridShape, Piece, BgImage, Wall, Decoration, MapItem como placeholders)
- `internal/domain/map/service/map_validator.go` — validações (slot dentro do grid, cellSize > 0, etc.)
- `internal/application/map/` — use cases (create, get, update, delete, list_by_campaign)
- `internal/gateway/pg/maps_repository.go` + `maps_mapper.go`
- `internal/app/api/maps_handler.go` — endpoints REST
- Migração goose: tabela `maps`
- `docs/dev/api/maps.md` — contrato REST completo (snake_case)
- Registro em `docs/documentation-map.yaml`
- Integration tests (`go test -tags=integration ./internal/gateway/pg/...`)

**Entregáveis frontend**:
- `src/services/mapsService.ts` — REST client com `objToSnakeCase` / `objToCamelCase`
- `src/hooks/useMaps.ts`, `useMap.ts`, `useCreateMap.ts`, `useUpdateMap.ts`, `useDeleteMap.ts`
- Seção "Mapas" na `CampaignPage` (lista mapas da campanha; botão "Criar mapa")
- Rotas: `/campaigns/:campaignId/maps/new` e `/campaigns/:campaignId/maps/:mapId/edit` (cascas vazias por ora)
- Testes RTL + msw mockando `mapsService`

**Critério de pronto**: mestre vê lista de mapas dentro da CampaignPage; consegue criar um mapa vazio (apenas nome+descrição) e ele persiste; cards levam para tela placeholder de edição.

### Fase 2 — Editor: configurar malha — frontend

**Escopo**: `CreateMapPage` funcional mínimo — mestre escolhe malha e dimensões.

**Entregáveis**:
- Layout da `CreateMapPage` — canvas central + toolbar lateral (organism `MapEditorToolbar`)
- `molecules/GridConfigPanel` — controles para `kind` (square/hex), `cols`, `rows`, `cellSize`, `color`, `opacity`
- `TacticalMapStage` renderiza a malha configurada (com `skewRatio=1`, `rotation=0`)
- Integração com pixi-viewport (pan + pinch-zoom em mobile, wheel-zoom no desktop)
- Botão "Salvar" persiste no backend
- Indicador "alterações não salvas" (`isDirty`)
- Confirmação ao sair com mudanças não salvas (`navigator.onbeforeunload`)

**Critério de pronto**: mestre cria um mapa novo, escolhe square ou hex, define 20×15, salva, recarrega — malha aparece corretamente.

### Fase 3 — Editor: imagem de fundo — frontend

**Escopo**: upload e ajuste de imagem de fundo, com malha sobreposta como guia.

**Entregáveis**:
- `molecules/BgImageAdjuster` reusando `ImagePickerModal` (upload + URL externa)
- Drag para reposicionar, sliders para escala, rotação, opacidade
- Compressão via `browser-image-compression` (já no projeto)
- Malha continua visível durante o ajuste (guia)
- Camada bg na cena Pixi (Sprite com transformações)
- Persistência (`bg` no JSONB de `maps`)

**Critério de pronto**: mestre faz upload (ou cola URL), ajusta tamanho/posição/rotação com a malha como referência, aplica, salva, recarrega — imagem aparece sob a malha.

### Fase 4 — Editor: colocar peças + altura virtual — frontend

**Escopo**: o coração visual da experiência. Mestre puxa personagens do roster e posiciona com altura Z.

**Entregáveis**:
- `organisms/CharacterRoster` — sidebar com personagens da campanha + NPCs (lista filtrada/searchable)
- Drag-and-drop (desktop): arrasta do roster → solta no slot
- Tap-to-place (mobile): tap no roster → tap no slot
- Camada Pieces na cena Pixi:
  - Sombra projetada no slot real
  - Sprite (gungi-frame + avatar) renderizado com offset Y proporcional a `z`
  - Badge "+Xm" no canto superior do sprite
- Painel de propriedade da peça selecionada: slider de altura Z, botão remover
- Persistência (`pieces[]`)

**Critério de pronto**: mestre coloca 3 peças (2 no chão, 1 a 2m de altura), salva, recarrega — todas aparecem corretamente com sombra e badge.

### Fase 5 — Editor: polish e mobile básico — frontend

**Escopo**: experiência de criação consolidada — toolbar sempre acessível (sem wizard), undo/redo, edição em mobile.

**Entregáveis**:
- Todas as ferramentas (grid/bg/peças) abertas e operáveis a qualquer momento (sem fluxo linear)
- Undo/redo via `zundo` — botões + atalhos (Cmd/Ctrl+Z / Shift+Cmd/Ctrl+Z)
- Modo mobile do mestre: toolbar colapsável; criação básica do mapa funcional em celular
- `EditMapPage` (reuso quase total do editor; só carrega o mapa por id e popula a store)
- Polish geral: toasts de feedback, validações inline

**Critério de pronto**: mestre cria um mapa completo em celular (malha + bg simples + 2 peças); desfaz e refaz mudanças no desktop; abre um mapa existente para edição.

### Fase 6 — Viewer in-match (GamePage) — backend + frontend

**Escopo**: `GamePage` para de ser placeholder. Mestre anexa um mapa à partida e todos veem.

**Entregáveis backend**:
- Tabela `match_maps` + repository + endpoints (POST `/matches/:id/map`, GET `/matches/:id/map`, DELETE)
- `docs/dev/api/match-maps.md` + registro no `documentation-map.yaml`

**Entregáveis frontend**:
- UI no `MatchPage`/lobby para mestre anexar um mapa (dropdown dos mapas da campanha)
- `GamePage` carrega o mapa anexado via React Query
- Renderiza `TacticalMapViewer` em modo viewer (read-only)
- Mestre vê em modo "presenter" (sem toolbar de edição ainda — a edição in-match é Fase 12)
- Layout responsivo (mobile, tablet, desktop)

**Critério de pronto**: mestre anexa um mapa criado a uma partida; jogadores entram na GamePage e veem o mesmo mapa (estático, read-only).

### Fase 7 — WebSocket sync (mestre publica) — backend + frontend

**Escopo**: mestre move peça → jogadores conectados veem em tempo real. Mestre revela/oculta peças.

**Entregáveis backend**:
- `internal/domain/match/entity/action/master_action.go`: adicionar variantes `AddPiece`, `RemovePiece`, `SetVisibility` (campos `*AddPieceOp`, etc.)
- `internal/app/game/`: handlers para os novos message types; routing pra sala
- `internal/app/game/action_mapper.go`: completar TODOs do `Move` com `slot_kind` (square/hex)
- `docs/dev/api/match-map-actions.md` — contrato dos novos message types WS + registro
- Tests de unidade + integration

**Entregáveis frontend**:
- `hooks/useMatchMapWs.ts` — análogo a `useLobbyWs`; recebe mensagens e aplica no store local
- Animação de movimento entre slots no viewer (lerp ~300ms)
- Visibility model no `Piece` — `visible: boolean` aplicado na renderização (oculta para jogadores quando `visible=false`)
- Mestre tem botões "publicar" para ações que afetam o estado público
- Botão "marcar como ação dentro do turno" (promove `MasterAction` para `Turn`-level via payload)

**Critério de pronto**: mestre publica um movimento de peça e o jogador conectado vê a peça mover-se em tempo real; mestre revela uma peça oculta e o jogador a vê aparecer.

### Fase 8 — Jogador: tap-to-move (intent) — backend + frontend

**Escopo**: jogador toca um slot e envia intenção de movimento para o mestre aprovar.

**Entregáveis backend**:
- Completar contrato do `Action.Move` no `action_mapper.go` (com `slot_kind`, `from`, `to`, `speed`, `final_speed`)
- WS routing: `action_submitted` (jogador) → vai pra `pendingActions` do `MatchSession`
- Mestre confirma via `confirmedAt` em `Action`; broadcast como `action_published`

**Entregáveis frontend**:
- Viewer: tap em slot vazio → "mover para aqui" → confirma com botão "Enviar movimento"
- Viewer: tap em peça → "alvejar" → vai para action queue (foundation para Fase futura de ataques)
- UI mestre: action queue mostra `Action` pendentes, permite aprovar/negar
- Animação de movimento ao publicar

**Critério de pronto**: jogador no celular toca um slot, envia intent; mestre vê na queue, aprova; a peça do jogador move-se para todos os clientes.

### Fase 9 — Distorção isométrica + rotação — frontend

**Escopo**: ligar a capacidade que já existe no math (Fase 0).

**Entregáveis**:
- Slider de `skewRatio` (1:1 → 1:3) na toolbar do mestre
- Handle de rotação no Stage
- Decisão UX: rotação aplica-se à malha (e bg sincroniza)? Apenas à malha? Toggle?
- Persistência (`grid.skewRatio`, `grid.rotation` já existem no schema)
- Coerência com Fase 11 (decorações herdam a transformação ou são independentes?)

**Critério de pronto**: mestre regula distorção isométrica 1:2; o mapa se transforma sem distorcer bg image (se assim decidido).

### Fase 10 — Paredes / obstáculos — frontend (+ possível BE)

**Escopo**: mestre desenha paredes que aparecem visualmente. Regra de bloqueio de movimento entra como fase à parte (mecânica de RPG).

**Entregáveis**:
- Ferramenta "Paredes" na toolbar
- Modo de desenho: cliques sucessivos definem vértices; tecla `Esc` ou click duplo encerra
- Camada Walls (Pixi Graphics, `lineTo`)
- Persistência (`walls[]` no JSONB)
- Edição: selecionar parede, mover vértices, apagar

**Critério de pronto**: mestre desenha 3 paredes em volta de uma sala; salva; recarrega — paredes persistem visualmente.

### Fase 11 — PNGs decorativos — frontend

**Escopo**: árvores, casas, tokens isométricos — assets livres do Google. Drag/scale/rotate/Z-order.

**Entregáveis**:
- Ferramenta "Decorações" na toolbar
- Modal de adicionar decoração (upload + URL externa, reusando `ImagePickerModal`)
- Camada Decorations entre BG e peças
- Property panel: posição, escala, rotação, opacidade, Z-order, remover
- Persistência (`decorations[]`)

**Critério de pronto**: mestre adiciona 5 árvores e 1 casa isométrica num mapa; arrasta, redimensiona, ajusta Z-order; salva; recarrega — tudo persiste.

### Fase 12 — Editar mapa durante a partida — stretch

**Escopo**: ativar o modo de edição do mapa dentro da partida ao vivo. Mestre pode ajustar o cenário em tempo real.

**Entregáveis**:
- Botão "Editar mapa" só visível ao mestre na `GamePage`
- Mudanças viram `MasterAction` específicas (não modificam `Map` snapshot direto)
- Broadcast via WS, jogadores veem mudanças
- Decisão UX: pause da partida durante edição, ou ao vivo?

**Critério de pronto**: discussão e validação UX antes da implementação (esta fase pode ser revisitada quando chegar a vez).

---

## 7. Capacidades futuras (planejadas além das 12 fases)

Documentadas aqui para que ninguém invente arquitetura contra esses caminhos.

### 7.1 Multi-andar (multi-level maps) — alta prioridade futura

Inspiração: módulo "Levels" do FoundryVTT. Um mapa pode ter vários níveis verticais (térreo, 1º andar, subsolo). Cada nível tem **sua própria imagem de fundo, paredes, decorações, iluminação**.

**Caminho de evolução**:
- Hoje: `bg: BgImage | null` (uma bg só)
- Quando vier: `bg` evolui para `floors: MapFloor[]`, cada `MapFloor` com seu `bg`, `walls`, `decorations`
- `Piece` ganha `floor: number` indicando em qual nível está
- UI: slider/dropdown de andar ativo; jogador/mestre vê apenas peças do andar visível (ou com opacidade reduzida quando em outro andar)

Migração será uma transformação em `maps.bg → maps.floors`, planejada no momento.

### 7.2 Sistema de itens (Items system) — paralelo, não-bloqueante

Mestre define itens da campanha (`Item`: nome, png, descrição, propriedades). Instâncias aparecem no mapa (`MapItem`: referência ao `Item` + `coord`). Jogadores podem larger, pegar, usar.

**Touchpoints**:
- Backend: nova entidade `Item` (campanha-level) + tabela
- Frontend: tela "Itens da campanha" (CRUD)
- Tactical map: `MapItem[]` já existe vazio; camada de renderização análoga a decorations
- Mecânica de pegar/largar: nova action type

### 7.3 In-match map editing — Fase 12 (stretch)

Já listado nas 12 fases mas merece UX dedicada. Decisão: pausa a partida durante edição, ou edição "live"? Implementação simples: edições viram `MasterAction.UpdateMap`, broadcasted como qualquer outra.

### 7.4 Action override (modificar action do jogador)

Fluxo separado, descrito na seção 5. **Não confundir com MasterAction.** Necessitará:
- Tabela de histórico de `Action` (versões originais)
- UI mestre para abrir uma `Action`, ver narração, editar campos, "sobrescrever"
- Manter cadeia de versões para auditoria

### 7.5 Pathfinding e regras de movimento

Quando paredes (Fase 10) e altura (Fase 4) tiverem integração com regras: distância considerando obstáculos, custo de movimento por terreno, line-of-sight para ataques à distância. Math de coords axial já viabiliza A* em hex.

### 7.6 Iluminação / fog of war

FoundryVTT-style: paredes opacas bloqueiam visão; cada peça projeta seu cone/raio de visão. Camadas + algoritmo de polygon clipping. Esta é a fase mais ambiciosa visualmente — beleza visível.

### 7.7 Linha tracejada (UX do `lineStyle`)

O renderer já respeita desde a Fase 0. UI de toggle (sólida/tracejada) é polish; entra quando o mestre pedir.

---

## 8. Cross-cutting

### 8.1 Estratégia de testes

| Camada | O que testar | Como |
|---|---|---|
| **Coords (puras)** | `slotToWorld`, `worldToSlot`, hex axial, skewRatio, rotation | Vitest unit — table-driven, sem render |
| **Store Zustand** | reducers, undo/redo, save flow | Vitest unit — instancia store, dispara actions, asserts |
| **React + UI puro** | toolbar, painéis, modal de bg, roster | Testing Library — interação humana, sem Pixi |
| **`TacticalMapStage`** | aceita prop `map` e instancia layers corretas | Mock do `@pixi/react` em `vitest.setup.ts`; assert "qual JSX foi renderizado pro Pixi" |
| **Integração editor** | drag drop coloca peça → store muda → save chama mutation | Testing Library + msw — mocka o Stage; foca no fluxo |
| **Sync (Fase 7+)** | recebe `MasterAction` via WS → store muda → Stage atualiza | msw com WS handler; integração simulando broadcast |
| **Visual/manual** | Pixi renderizando de fato + UX | Rota `/dev/tactical-map-demo` + browser + visual companion |
| **Backend** | Map CRUD, validações, WS message handlers | `go test` + integration (Postgres real) |

**Decisão chave**: WebGL **não é testado em Vitest** (jsdom não tem GPU). Tudo que é "Pixi de verdade" valida visualmente. Testes automatizados focam no determinístico: math, estado, contratos React, backend.

### 8.2 Performance

| Risco | Mitigação |
|---|---|
| Re-render React em cada move | Zustand seleciona slice; componente que lê `pieces[id]` re-renderiza isolado |
| Re-criar sprites a cada render | `useMemo` para refs Pixi; chave estável por `piece.id` |
| Grade gigante (60×60 = 3600 linhas) | Render em **um único `Graphics`** (não 3600 sprites); redesenhar só quando `grid` muda |
| Drag em 60fps em mobile | Sprite move localmente; commit apenas em `pointerup` (zero rerender durante drag) |
| Imagem de fundo gigante (10MB) | `browser-image-compression` (já no projeto) + limite no upload (e.g., 4MB pós-compressão) |
| Bundle pesando 250KB+ | Lazy load das rotas de mapa via `React.lazy` |
| WS spammando | Resolvido pelo modelo: ações discretas (publicação explícita); zero streaming |
| Hex math em hot path | Funções puras sem alocação; benchmark se virar gargalo |

### 8.3 Erros

- **Carregar mapa falha**: `ErrorContainer` (atom existente) + retry da React Query.
- **WebGL não suportado**: Pixi tenta fallback Canvas2D automático; se falhar, exibir mensagem + link "saber mais".
- **WS desconecta**: padrão de `useLobbyWs` — reconnect com backoff + sync via `map_full_state`.
- **Save falha**: rascunho em `localStorage`; toast "não conseguimos salvar; suas mudanças estão protegidas localmente; tentar de novo?".
- **Upload de imagem falha**: erro inline no `BgImageAdjuster`.
- **Drop em slot ocupado**: bloqueio com feedback visual (slot vermelho); empilhamento físico (montaria) entra como feature futura.
- **Validação de slot fora da grade**: rejeita no client e re-valida no server (defense-in-depth).

### 8.4 Acessibilidade

Tactical maps são inerentemente visuais. Não temos como atingir paridade total com leitores de tela, mas:

- **Teclado para mestre desktop**: setas movem peça selecionada slot a slot; `Enter` confirma; `Esc` cancela; `Tab` navega painéis da toolbar; `Cmd/Ctrl+S` salva; `Cmd/Ctrl+Z` desfaz.
- **Toolbar é HTML normal** (botões, inputs, ARIA labels) — totalmente acessível.
- **`<canvas>` recebe `role="application"` + `aria-label` descritivo + texto alternativo com resumo textual** (e.g., "Campo tático: malha quadrada 20×20, 5 peças, fundo: imagem 'Floresta da Morte'").
- **Modo de leitura textual** (futuro polish): lista textual de peças com posição — útil pra screen readers e replays.
- **Tokens de cor**: respeitar `src/styles/tokens.ts`. Cor da malha e fundo devem ter contraste mínimo (WCAG AA quando viável).

---

## 9. Integração com o sistema existente

### 9.1 Pontos de toque

| Surface | Mudança | Fase |
|---|---|---|
| `package.json` (front) | + 6 dependências | 0 |
| `src/types/tacticalMap.ts` | novo | 0 |
| `src/components/organisms/TacticalMapStage.tsx` | novo | 0 |
| `src/features/tactical-map/*` | novo | 0 |
| `src/pages/CreateMapPage.tsx` | novo | 1-2 |
| `src/pages/EditMapPage.tsx` | novo | 5 |
| `src/pages/GamePage.tsx` | substitui placeholder pelo viewer | 6 |
| `src/pages/CampaignPage.tsx` | adicionar seção "Mapas" | 1 |
| `src/services/mapsService.ts` | novo (snake↔camel) | 1 |
| `src/hooks/useMaps.ts` etc. | novos | 1 |
| `src/App.tsx` | + rotas | 0, 1, 2, 5 |
| `src/components/molecules/ImagePickerModal.tsx` | reuso para bg image | 3 |
| `internal/domain/map/*` (Go) | novo bounded context | 1 |
| `internal/application/map/*` | novo | 1 |
| `internal/gateway/pg/maps_*.go` | novo | 1 |
| `internal/app/api/maps_handler.go` | novo | 1 |
| `internal/app/game/action_mapper.go` | preencher TODOs do `Move` com `slot_kind` | 7 |
| `internal/app/game/message.go` | novos message types (`master_action_*`, `map_*`) | 7 |
| `internal/domain/match/entity/action/master_action.go` | + variants `AddPiece`, `RemovePiece`, `SetVisibility` | 7 |
| `docs/dev/api/maps.md` | novo | 1 |
| `docs/dev/api/match-maps.md` | novo | 6 |
| `docs/dev/api/match-map-actions.md` | novo | 7 |
| `docs/documentation-map.yaml` | registrar contratos | 1, 6, 7 |

### 9.2 Auth, snake↔camel, React Query

Padrões do projeto seguidos sem desvio:
- JWT em `localStorage["token"]` via `TokenContext`; interceptor 401 em `httpClient.ts` (sem mudança).
- `services/mapsService.ts` aplica `objToSnakeCase` na saída e `objToCamelCase` na entrada.
- Hooks de React Query incluem `token` (e ids) no `queryKey`; `enabled: !!token && !!campaignId`.
- `retry: 1` consistente.

### 9.3 WebSocket (cmd/game/)

Reuso do hub/room/client pattern existente. Adicionar message types específicos do mapa segue o mesmo padrão do lobby; nenhuma reformulação de infra.

---

## 10. Riscos e mitigações

| Risco | Probabilidade | Impacto | Mitigação |
|---|---|---|---|
| Curva de PixiJS atrasa Fase 0 | Média | Médio | Dev docs em `docs/dev/tactical-map/` escritas pra junior; rota `/dev/tactical-map-demo` valida cedo |
| Bundle pesa demais | Baixa | Médio | Lazy load das rotas; benchmark do bundle em Fase 0 |
| Pinch-zoom buga em algum dispositivo | Média | Médio | `pixi-viewport` é maduro; testar em browser stack matrix antes do release de Fase 2 |
| `slot_kind` no Move quebra contrato antigo | Baixa | Alto | Backend está com TODOs; nenhum cliente existente assume formato; sincronizar Fase 7 BE+FE |
| Multi-andar exigir migração futura | Certo | Baixo | `bg → floors` é transformação previsível; documentada na seção 7.1 |
| Performance em mobile com 60×60 | Média | Médio | Bench em Fase 0 + Fase 4; otimizações se necessário (sprite atlasing, etc.) |
| Visibility (Fase 7) revelar estado por engano | Média | Alto | Validação no backend antes do broadcast; testes de "estado oculto não vaza" |

---

## 11. Histórico de decisões deste brainstorm

Para futuros leitores entenderem **por que** cada escolha:

- **PixiJS escolhido sobre Konva** porque o sistema escala até 60×60 + futuros PNGs + isométrico + rotação, e FoundryVTT comprova a stack em produção.
- **Híbrido (Stage + Editor + Viewer)** porque componente único cresce indefinidamente e dois componentes independentes divergem visualmente.
- **Save explícito**, não autosave, porque combina com "publicação discreta" turn-based.
- **Action/MasterAction == delta WS** porque já é a representação canônica de mudanças no backend e evita duplicar protocolos.
- **`bg.opacity` mantido** porque é útil já agora (legibilidade) e necessário para multi-andar futuro.
- **`walls`, `decorations`, `items` declarados desde a Fase 0 com `[]`** para evitar migração quando suas fases chegarem.
- **Hex em coords axiais (q, r)** porque é padrão da literatura (RedBlobGames) e simplifica vizinhos/distância.
- **MasterAction default em `Round`** (decisão pré-existente do backend) — promovido a `Turn` apenas com flag explícita do mestre via UI. Backend não infere.
- **Action override é fluxo separado**, não MasticAction. Documentado pra ninguém conflar.
- **Zustand + zundo + immer** porque undo/redo de qualidade em editor grande exige histórico estruturado, e a tríade resolve isso com ~10KB.
- **Tipos completos desde a Fase 0** (incluindo placeholders futuros) porque o backend está esperando contratos finalizados (TODOs em `action_mapper.go`).

---

## 12. Próximos passos após este spec

1. Usuário revisa este spec.
2. Para cada fase: nova sessão Opus → invoca `superpowers:writing-plans` com este spec como contexto → gera plan detalhado da fase → nova sessão Sonnet executa o plan.
3. Dev docs em `System_X_System_React/docs/dev/tactical-map/` (escritas neste mesmo brainstorm) servem ao dev humano (junior) em paralelo ao spec.
4. PRs separados por repo (frontend e backend), cross-link nas descrições — convenção do projeto.

---

## Anexo A — Convenções

### Naming

- **Feature folder**: `tactical-map` (kebab-case) tanto no frontend (`src/features/tactical-map/`) quanto na documentação (`docs/dev/tactical-map/`).
- **Termos**: "campo" (PT) = "field"/"map" no código. Preferir **map** no código; **mapa** no copy de UI em PT-BR.
- **MasterAction variants**: `AddPiece`, `RemovePiece`, `SetVisibility` (PascalCase no Go, snake_case nos message types JSON).

### Dependências — versões alvo (Fase 0)

```json
{
  "pixi.js": "^8.0.0",
  "@pixi/react": "^8.0.0",
  "pixi-viewport": "^6.0.0",
  "zustand": "^5.0.0",
  "zundo": "^2.0.0",
  "immer": "^10.0.0"
}
```

### Padrão de teste

- Vitest com `vitest.config.ts` já configurado no projeto.
- Mocks em `vitest.setup.ts` para `@pixi/react` (stub que renderiza children).
- msw para mocks de REST e WS (já usado em `LobbyPage.test.tsx`).

---

## Anexo B — Dev docs entregues junto com este spec

```
System_X_System_React/docs/dev/tactical-map/
├─ overview.md           ← visão pra dev
├─ pixi-stack.md         ← PixiJS / @pixi/react / pixi-viewport explicados
├─ state-management.md   ← Zustand / zundo / immer
├─ coordinates.md        ← slot / world / screen + math hex
├─ sync-and-delta.md     ← WS, Action/MasterAction, JSON Patch
└─ testing.md            ← estratégia e mocks
```

Escritas para **junior dev** — exemplos curtos, modelos mentais, links pros docs originais. Documentam **o quê** e **por que**, não o **como fazer agora** (isso é responsabilidade do plan de cada fase).
