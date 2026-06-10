# Maps API

## POST /campaigns/{campaign_id}/maps — Criar mapa

**Auth:** JWT (master da campanha)

### Request

```json
{
  "name": "Floresta do Norte",
  "description": "Área densa ao norte da cidade, cheia de armadilhas",
  "grid": { "kind": "square", "cols": 25, "rows": 25, "cell_size": 64, "skew_ratio": 1.0, "rotation": 0, "color": "#ffffff", "opacity": 0.5, "line_style": "solid" },
  "bg": null,
  "pieces": []
}
```

| Campo | Regra |
|---|---|
| `name` | obrigatório, não vazio |
| `description` | opcional |
| `grid` | opcional; usa grid padrão 25×25 64px se omitido |
| `bg` | opcional |
| `pieces` | opcional; lista de peças iniciais (geralmente `[]` na criação) |

### Respostas

| Status | Situação |
|---|---|
| 201 | Mapa criado, retorna `{ "map": MapResponse }` |
| 400 | Body malformado |
| 401 | Sem JWT |
| 403 | Usuário não é o master da campanha |
| 404 | Campanha não encontrada |
| 422 | `name` vazio, `cell_size ≤ 0`, `cols/rows ≤ 0`, `skew_ratio ∉ [0,1]` |
| 500 | Erro interno |

---

## GET /campaigns/{campaign_id}/maps — Listar mapas da campanha

**Auth:** JWT (master da campanha)

### Response 200

```json
{
  "maps": [
    {
      "id": "uuid",
      "campaign_id": "uuid",
      "name": "Floresta do Norte",
      "description": "Área densa ao norte da cidade, cheia de armadilhas",
      "grid": {
        "kind": "square",
        "cols": 25,
        "rows": 25,
        "cell_size": 64,
        "skew_ratio": 1.0,
        "rotation": 0,
        "color": "#ffffff",
        "opacity": 0.5,
        "line_style": "solid"
      },
      "bg": null,
      "pieces": [],
      "walls": [],
      "decorations": [],
      "items": [],
      "created_at": "2026-05-31T00:00:00Z",
      "updated_at": "2026-05-31T00:00:00Z"
    }
  ]
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Lista (pode ser vazia) |
| 401 | Sem JWT |
| 403 | Usuário não é o master da campanha |
| 404 | Campanha não encontrada |
| 500 | Erro interno |

---

## GET /maps/{id} — Obter mapa

**Auth:** JWT (qualquer participante da campanha)

### Response 200

```json
{
  "map": {
    "id": "uuid",
    "campaign_id": "uuid",
    "name": "Floresta do Norte",
    "description": "Área densa ao norte da cidade, cheia de armadilhas",
    "grid": {
      "kind": "square",
      "cols": 25,
      "rows": 25,
      "cell_size": 64,
      "skew_ratio": 1.0,
      "rotation": 0,
      "color": "#ffffff",
      "opacity": 0.5,
      "line_style": "solid"
    },
    "bg": null,
    "pieces": [],
    "walls": [],
    "decorations": [],
    "items": [],
    "created_at": "2026-05-31T00:00:00Z",
    "updated_at": "2026-05-31T00:00:00Z"
  }
}
```

### Erros

| Status | Situação |
|---|---|
| 200 | Mapa retornado |
| 401 | Sem JWT |
| 404 | Mapa não encontrado ou usuário não é participante da campanha |
| 500 | Erro interno |

### Notas

- Acessível a todos os participantes da campanha (não apenas o master) para suportar a feature de lobby futura.

---

## PUT /maps/{id} — Atualizar mapa

**Auth:** JWT (master da campanha)

### Request

```json
{
  "name": "Floresta do Norte — Revisado",
  "description": "Nova descrição do mapa",
  "grid": { "kind": "square", "cols": 25, "rows": 25, "cell_size": 64, "skew_ratio": 1.0, "rotation": 0, "color": "#ffffff", "opacity": 0.5, "line_style": "solid" },
  "bg": null,
  "pieces": [
    { "id": "uuid", "character_id": "uuid", "coord": { "slot": { "kind": "square", "col": 3, "row": 5 }, "z": 0 }, "visible": true }
  ],
  "walls": [
    {
      "id": "uuid",
      "p1": [0, 0],
      "p2": [64, 0],
      "wall_type": "wall",
      "material": "stone",
      "move": true,
      "sense": "full",
      "direction": "both",
      "open": false,
      "locked": false,
      "hp": 100,
      "max_hp": 100,
      "resistance": 5,
      "destroyed": false
    }
  ]
}
```

| Campo | Regra (se enviado) |
|---|---|
| `name` | obrigatório, não vazio |
| `description` | opcional |
| `grid` | opcional; mantém grid existente se omitido |
| `bg` | opcional; omitir mantém existente |
| `pieces` | opcional; omitir mantém existente; `[]` remove todas as peças |
| `walls` | opcional; omitir mantém existente; `[]` remove todas as paredes |

### Respostas

| Status | Situação |
|---|---|
| 204 | Mapa atualizado |
| 400 | Body malformado |
| 401 | Sem JWT |
| 403 | Usuário não é o master da campanha |
| 404 | Mapa não encontrado |
| 422 | `name` vazio, `cell_size ≤ 0`, `cols/rows ≤ 0`, `skew_ratio ∉ [0,1]` |
| 500 | Erro interno |

---

## DELETE /maps/{id} — Deletar mapa

**Auth:** JWT (master da campanha)

### Request

Sem body.

### Respostas

| Status | Situação |
|---|---|
| 204 | Mapa deletado |
| 401 | Sem JWT |
| 403 | Usuário não é o master da campanha |
| 404 | Mapa não encontrado |
| 500 | Erro interno |

---

## MapResponse — Formato do objeto mapa

```json
{
  "id": "uuid",
  "campaign_id": "uuid",
  "name": "Floresta do Norte",
  "description": "Área densa ao norte da cidade, cheia de armadilhas",
  "grid": {
    "kind": "square",
    "cols": 25,
    "rows": 25,
    "cell_size": 64,
    "skew_ratio": 1.0,
    "rotation": 0,
    "color": "#ffffff",
    "opacity": 0.5,
    "line_style": "solid"
  },
  "bg": null,
  "pieces": [],
  "walls": [
    {
      "id": "uuid",
      "p1": [0, 0],
      "p2": [64, 0],
      "wall_type": "wall",
      "material": "stone",
      "move": true,
      "sense": "full",
      "direction": "both",
      "open": false,
      "locked": false,
      "hp": 100,
      "max_hp": 100,
      "resistance": 5,
      "destroyed": false
    }
  ],
  "decorations": [],
  "items": [],
  "created_at": "2026-05-31T00:00:00Z",
  "updated_at": "2026-05-31T00:00:00Z"
}
```

### Notas gerais

- `POST /campaigns/:id/maps` (criação) aceita `name`, `description`, `grid`, `bg` e `pieces`. `walls`, `decorations` e `items` ficam `[]` na criação.
- `PUT /maps/:id` aceita adicionalmente `walls` como lista de `WallSegment`. `decorations` e `items` ainda não são suportados no request (gerenciados por fases futuras).
- Campos JSONB (`pieces`, `walls`, `decorations`, `items`) têm default `[]`; `bg` tem default `null`.
- Todos os endpoints requerem JWT Bearer token no header `Authorization`.

---

## WallSegment — Formato do segmento de parede

```json
{
  "id": "uuid-string",
  "p1": [0.0, 0.0],
  "p2": [64.0, 0.0],
  "wall_type": "wall | door | window | secret_door | terrain",
  "material": "stone | wood | iron | magical",
  "door_subtype": "basic | double | portcullis | drawbridge",
  "window_subtype": "basic | barred | shuttered",
  "move": true,
  "sense": "full | sight | none",
  "direction": "both | left | right",
  "open": false,
  "locked": false,
  "hp": 100,
  "max_hp": 100,
  "resistance": 5,
  "destroyed": false
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `id` | string (UUID) | Identificador único do segmento, gerado pelo frontend via `crypto.randomUUID()` |
| `p1`, `p2` | `[number, number]` | Endpoints em coordenadas de mundo (pré-transform); `p1 ≠ p2` |
| `wall_type` | enum | Comportamento funcional |
| `material` | enum | Propriedades físicas (HP, resistência, cor) |
| `door_subtype` | enum? | Presente apenas quando `wall_type = "door"` |
| `window_subtype` | enum? | Presente apenas quando `wall_type = "window"` |
| `move` | bool | Bloqueia movimento físico |
| `sense` | enum | O que bloqueia em termos de percepção |
| `direction` | enum | Direção de bloqueio (both = nos dois sentidos) |
| `open` | bool | Porta/janela está aberta (só relevante para door/window) |
| `locked` | bool | Porta trancada |
| `hp` | int | Pontos de vida atuais (≥ 0) |
| `max_hp` | int | Pontos de vida máximos |
| `resistance` | int | Dano absorvido por ataque |
| `destroyed` | bool | Segmento destruído (visual alterado) |

### Defaults por tipo (aplicados pelo frontend ao criar o segmento)

| `wall_type` | `move` | `sense` | `direction` | `material` padrão |
|---|---|---|---|---|
| `wall` | `true` | `full` | `both` | `stone` |
| `door` | `true` | `full` | `both` | `wood` |
| `window` | `true` | `none` | `both` | `wood` |
| `secret_door` | `true` | `full` | `both` | `stone` |
| `terrain` | `true` | `none` | `left` | — |

### Notas gerais de validação (backend)

- `PUT /maps/:id` aceita `walls` como campo opcional. `null` ou ausente = mantém as paredes existentes. `[]` = remove todas.
- Validações: `p1 ≠ p2`; `wall_type` deve ser um dos 5 valores válidos; `hp ≥ 0`.
- O backend não calcula defaults — o frontend envia o objeto completo.
