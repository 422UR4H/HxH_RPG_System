# Tactical Map — Fase 1: Persistência + Listagem de Mapas

**Date:** 2026-05-31
**Status:** Approved — ready for implementation
**Scope:** Cross-stack (`System_X_System_React` + `System_X_System`)
**Parent spec:** `2026-05-31-tactical-map-design.md`

---

## 1. Escopo

CRUD REST de mapas no backend + listagem de mapas integrada à `CampaignPage` e `MatchPage`
via sistema de tabs URL-based. Critério de pronto: mestre cria um mapa vazio (nome +
descrição), ele persiste, aparece na lista e o card leva para a tela placeholder de edição.

---

## 2. Backend

### 2.1 Estrutura de arquivos

Segue o padrão de domínio aninhado (como `match/`):

```
internal/domain/map/
├── entity/
│   ├── map.go              — TacticalMap (agregado raiz)
│   ├── grid.go             — GridShape
│   ├── piece.go            — Piece, PieceCoord, SlotCoord (SquareCoord | HexCoord)
│   └── placeholders.go     — Wall, Decoration, MapItem (structs com defaults [])
└── service/
    └── map_validator.go    — cellSize > 0, cols/rows > 0, skewRatio ∈ [0,1]

internal/application/map/
├── create_map.go           — ICreateMap + CreateMapInput
├── get_map.go              — IGetMap
├── update_map.go           — IUpdateMap + UpdateMapInput
├── list_maps_by_campaign.go — IListMapsByCampaign
└── delete_map.go           — IDeleteMap

internal/gateway/pg/map/
├── repository.go           — struct Repository + IMapRepository interface
├── create_map.go
├── get_map.go
├── update_map.go
├── list_maps.go
├── delete_map.go
├── mapper.go               — pgModel{} ↔ domain entity; JSONB marshal/unmarshal aqui
└── *_test.go               — integration tests (go test -tags=integration)

internal/app/api/map/
├── create_map.go
├── get_map.go
├── list_maps.go
├── update_map.go
├── delete_map.go
├── map_response.go         — tipos de resposta compartilhados
└── mocks_test.go
```

### 2.2 Migration

```
migrations/20260531000000_create_maps_table.sql
```

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

### 2.3 Defaults do grid ao criar

```json
{
  "kind": "square",
  "cols": 25,
  "rows": 25,
  "cell_size": 64,
  "skew_ratio": 1.0,
  "rotation": 0,
  "color": "#ffffff",
  "opacity": 0.5,
  "line_style": "solid"
}
```

---

## 3. API Contract

Contrato completo escrito em `docs/dev/api/maps.md` (backend) e registrado em
`docs/documentation-map.yaml`. Esta seção é o resumo de referência.

### 3.1 Endpoints

| Método   | Rota                              | Quem acessa              | Status OK |
|----------|-----------------------------------|--------------------------|-----------|
| `POST`   | `/campaigns/:campaign_id/maps`    | Master da campanha       | `201`     |
| `GET`    | `/campaigns/:campaign_id/maps`    | **Master apenas**        | `200`     |
| `GET`    | `/maps/:id`                       | Qualquer participante¹   | `200`     |
| `PUT`    | `/maps/:id`                       | Master da campanha       | `200`     |
| `DELETE` | `/maps/:id`                       | Master da campanha       | `204`     |

¹ `GET /maps/:id` é acessível a todos os participantes da campanha porque o lobby
precisará deste endpoint (Fase 6). Em Fase 1 a resposta é idêntica para todos;
filtragem por role (peças invisíveis etc.) vem em fase futura.

### 3.2 Request — POST/PUT body (snake_case)

Fase 1 aceita apenas `name` (obrigatório) e `description` (opcional). Os demais
campos (`grid`, `bg`, `pieces`, `walls`, `decorations`, `items`) são gerenciados
pelo backend — recebem defaults na criação e são substituídos integralmente no PUT.

```json
{ "name": "Floresta do Norte", "description": "..." }
```

### 3.3 Response — shape de mapa (POST 201 / GET 200 / PUT 200)

```json
{
  "id": "uuid",
  "campaign_id": "uuid",
  "name": "Floresta do Norte",
  "description": "...",
  "grid": {
    "kind": "square", "cols": 25, "rows": 25, "cell_size": 64,
    "skew_ratio": 1.0, "rotation": 0, "color": "#ffffff",
    "opacity": 0.5, "line_style": "solid"
  },
  "bg": null,
  "pieces": [], "walls": [], "decorations": [], "items": [],
  "created_at": "2026-05-31T00:00:00Z",
  "updated_at": "2026-05-31T00:00:00Z"
}
```

GET list: `{ "maps": [ /* array do shape acima */ ] }`

### 3.4 Erros

| Status | Quando                                                        |
|--------|---------------------------------------------------------------|
| `403`  | Usuário não é master (write ops + list)                       |
| `404`  | Campanha/mapa não encontrado ou usuário não é participante    |
| `422`  | `name` vazio, `cell_size ≤ 0`, `cols`/`rows ≤ 0`, `skew_ratio ∉ [0,1]` |

---

## 4. Frontend

### 4.1 Novos arquivos

```
src/
├── components/
│   ├── organisms/
│   │   └── PageTabNav.tsx          — tab nav URL-based
│   └── molecules/
│       └── MapCard.tsx             — card de mapa
├── services/
│   └── mapsService.ts              — CRUD com snake↔camel
├── hooks/
│   ├── useMaps.ts
│   ├── useMap.ts
│   ├── useCreateMap.ts
│   ├── useUpdateMap.ts
│   └── useDeleteMap.ts
└── pages/
    ├── CreateMapPage.tsx            — form: nome + descrição
    └── EditMapPage.tsx              — placeholder
```

`src/types/tacticalMap.ts` já existe da Fase 0 — sem alterações.

### 4.2 PageTabNav

- Props: `tabs: { id: string; label: string }[]`
- Renderiza `null` quando `tabs.length <= 1` (sem chrome de tab vazio)
- Lê/escreve `?tab=` via `useSearchParams` do react-router-dom
- Estilizado com `colors` e `fonts` dos tokens do projeto

### 4.3 Lógica de tabs — CampaignPage

```
Master   → availableTabs = [{ id: 'matches', label: 'Partidas' }, { id: 'maps', label: 'Mapas' }]
Player   → availableTabs = [{ id: 'matches', label: 'Partidas' }]  → PageTabNav não renderiza
```

Tab `maps` (master only):
- Lista de `MapCard`s via `useMaps(campaignId)`
- `AdaptiveActionButton` "Criar Mapa" no rodapé → `navigate('/campaigns/:id/maps/new')`

### 4.4 Lógica de tabs — MatchPage

```
Master                     → availableTabs = [Eventos, Mapas]
Player + storyEndAt set    → availableTabs = [Eventos, Mapas]
Player + sem storyEndAt    → availableTabs = [Eventos]  → PageTabNav não renderiza
```

Se a URL tiver `?tab=maps` e essa tab não estiver disponível → `useEffect` com
`setSearchParams({ replace: true })` corrige para `?tab=events`. Sem flash.

Tab `maps` — master: lista via `useMaps(campaignId)`, sem botão criar.

Tab `maps` — player + partida encerrada: placeholder em Fase 1 (mensagem neutra +
ícone; sem fetch real). Fase 6 substituirá pelo conteúdo real dos mapas jogados.

### 4.5 Novas rotas (App.tsx)

```
/campaigns/:campaignId/maps/new         → CreateMapPage (lazy)
/campaigns/:campaignId/maps/:mapId/edit → EditMapPage   (lazy)
```

### 4.6 Testes

- `CampaignPage.test.tsx` — aba Mapas visível para master; oculta para player;
  lista renderiza `MapCard`s; "Criar Mapa" navega para `/campaigns/:id/maps/new`
- `CreateMapPage.test.tsx` — submit válido chama POST e navega de volta;
  erro 422 exibe mensagem em português

---

## 5. Critério de pronto

- Mestre vê aba "Mapas" na `CampaignPage`; jogador não vê
- Mestre cria mapa (nome + descrição), ele persiste no banco e aparece na lista
- Card leva para `EditMapPage` (placeholder)
- Mestre vê aba "Mapas" na `MatchPage` com lista da campanha
- Player vê aba "Mapas" na `MatchPage` apenas após `storyEndAt` (placeholder em Fase 1)
- `GET /maps/:id` acessível a todos os participantes (base para o lobby na Fase 6)
- Integration tests do gateway passando; testes RTL da `CampaignPage` e `CreateMapPage` passando
