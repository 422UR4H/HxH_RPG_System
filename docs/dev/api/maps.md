# Maps API

## POST /campaigns/{campaign_id}/maps — Criar mapa

**Auth:** JWT (master da campanha)

### Request

```json
{
  "name": "Floresta do Norte",
  "description": "Área densa ao norte da cidade, cheia de armadilhas"
}
```

| Campo | Regra |
|---|---|
| `name` | obrigatório, não vazio |
| `description` | opcional |

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

### Request — todos os campos opcionais

```json
{
  "name": "Floresta do Norte — Revisado",
  "description": "Nova descrição do mapa"
}
```

| Campo | Regra (se enviado) |
|---|---|
| `name` | não vazio |
| `description` | opcional |

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
  "walls": [],
  "decorations": [],
  "items": [],
  "created_at": "2026-05-31T00:00:00Z",
  "updated_at": "2026-05-31T00:00:00Z"
}
```

### Notas gerais

- Fase 1: body de criação/atualização aceita apenas `name` e `description`. Campos de grid, bg, pieces, walls, decorations e items são gerenciados por endpoints futuros.
- Campos JSONB (`pieces`, `walls`, `decorations`, `items`) têm default `[]`; `bg` tem default `null`.
- Todos os endpoints requerem JWT Bearer token no header `Authorization`.
