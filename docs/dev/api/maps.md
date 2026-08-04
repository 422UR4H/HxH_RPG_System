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

---

## Fog of War / Visibilidade (Fase 10-D)

### `GET /maps/:id` — Filtragem por papel

O endpoint é **role-aware**: o resultado varia conforme o papel do usuário autenticado na campanha.

| Papel | Comportamento |
|---|---|
| Master | Resposta completa e sem máscara. Todas as paredes (incluindo `secret_door` não reveladas) são retornadas com seus campos reais. |
| Jogador (não-master) | `secret_door` não reveladas (`revealed=false`) são mascaradas como parede comum. Peças com `visible=false` são removidas da resposta. Sem filtragem LOS — LOS é delegado ao WS. |

#### Máscara de porta secreta não revelada

Quando um jogador recebe um `WallSegment` com `wall_type = "secret_door"` e `revealed = false`, o backend substitui os campos identificadores da porta pelo equivalente de uma parede comum, preservando todos os campos de combate:

| Campo | Valor mascarado |
|---|---|
| `wall_type` | `"wall"` (em vez de `"secret_door"`) |
| `door_subtype` | ausente (`null` / omitido) |
| `window_subtype` | ausente (`null` / omitido) |
| `open` | `false` |
| `locked` | `false` |
| `id`, `p1`, `p2`, `material`, `hp`, `max_hp`, `resistance`, `move`, `sense`, `direction`, `destroyed` | preservados sem alteração |

> **Nota LOS-at-REST:** O `GET /maps/:id` não aplica filtragem de linha de visão (LOS). Ele só remove o que visivelmente não existe para jogadores (peças invisíveis e identidade de portas secretas não reveladas). LOS exato por jogador é entregue pelo WebSocket via `map_full_state` e `visibility_updated` quando a partida está em andamento.

---

### Campo `fog_mode` no mapa

O `TacticalMap` possui o campo `fog_mode`, persistido no banco, que controla o comportamento de exploração durante a partida:

| Valor | Comportamento |
|---|---|
| `"explored"` | Paredes já vistas em turnos anteriores continuam visíveis (mesmo fora do LOS atual). **Default** quando o campo está vazio no banco. |
| `"live"` | Apenas o LOS do turno atual é visível; nenhuma memória de exploração. |

O campo `fog_mode` é enviado pelo servidor no payload WS `map_full_state` (ver abaixo). Ele não é exposto na resposta REST de `GET /maps/:id` atualmente — apenas o WebSocket o carrega para o cliente.

---

### Eventos WebSocket — Fog of War (servidor → cliente)

Todos os payloads usam snake_case. Esses eventos são enviados durante uma partida em andamento (após `start_match`).

---

#### `map_full_state` (estendido — Fase 10-D)

Enviado a cada cliente que se conecta, e re-enviado após operações que alteram o estado do tabuleiro. Cada cliente recebe uma versão filtrada por papel.

**Payload (jogador não-master):**

```json
{
  "pieces": [
    {
      "piece_id": "uuid-string",
      "slot": { "kind": "square", "col": 3, "row": 5 },
      "character_id": "uuid-string",
      "visible": true
    }
  ],
  "walls": [ /* WallSegment[] filtrados e mascarados por LOS + masking de secret_door */ ],
  "visible_polygons": [
    [ { "x": 0.0, "y": 0.0 }, { "x": 64.0, "y": 0.0 }, { "x": 64.0, "y": 64.0 } ]
  ],
  "explored_cells": [ [0, 0], [1, 0], [0, 1] ],
  "fog_mode": "explored"
}
```

**Payload (master):**

```json
{
  "pieces": [ /* todas as peças, sem filtro */ ],
  "walls": [ /* todas as paredes, sem máscara */ ],
  "fog_mode": "explored"
}
```

> O master não recebe `visible_polygons` nem `explored_cells` — campos omitidos.

| Campo | Tipo | Descrição |
|---|---|---|
| `pieces` | `PieceMovedPayload[]` | Peças visíveis pelo receptor. Para o master, todas. Para jogador, apenas as em LOS ou próprias com `visible=true`. |
| `walls` | `WallSegment[]` | Paredes visíveis. Para jogador, filtradas por LOS/exploração; portas secretas não reveladas mascaradas como `"wall"`. |
| `visible_polygons` | `[{x,y}][]` | Polígonos de visibilidade atuais do jogador em coordenadas de mundo. Omitido para o master e no lobby (sem partida ativa). |
| `explored_cells` | `[int, int][]` | Células já exploradas: `[A, B]` = `[col, row]` (square) ou `[q, r]` (hex axial). Presente somente quando `fog_mode = "explored"` e há estado de exploração. Omitido para o master. |
| `fog_mode` | `string` | `"live"` ou `"explored"`. Sempre presente. No lobby (sem sessão ativa), o valor enviado é `"live"` como placeholder. |

---

#### `visibility_updated`

Enviado a cada jogador individual quando seu LOS muda (ex.: após movimento de peça ou revelação de porta secreta). Nunca enviado ao master.

```json
{
  "visible_polygons": [
    [ { "x": 0.0, "y": 0.0 }, { "x": 128.0, "y": 0.0 } ]
  ],
  "explored_delta": [ [2, 0], [3, 1] ]
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `visible_polygons` | `[{x,y}][]` | Novo conjunto completo de polígonos de visibilidade do jogador. |
| `explored_delta` | `[int, int][]` | Células **recém-exploradas** neste update (delta, não o total). `[A, B]` = `[col, row]` / `[q, r]`. Omitido se vazio. |

---

#### `wall_revealed`

Broadcast para **todos os clientes** quando o master revela uma porta secreta. Após este evento, todos os clientes devem substituir a entrada correspondente em seu estado local pelo `WallSegment` completo e real recebido.

```json
{
  "wall": {
    "id": "uuid-string",
    "p1": [0.0, 0.0],
    "p2": [64.0, 0.0],
    "wall_type": "secret_door",
    "material": "wood",
    "door_subtype": "basic",
    "move": true,
    "sense": "full",
    "direction": "both",
    "open": false,
    "locked": false,
    "hp": 80,
    "max_hp": 80,
    "resistance": 3,
    "destroyed": false,
    "revealed": true
  }
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `wall` | `WallSegment` | O `WallSegment` completo e real da porta secreta, agora com `revealed=true`. |

> O master aciona a revelação via `enqueue_master_action` com `interact.kind = "reveal"` e `target_ids` contendo o(s) ID(s) da(s) parede(s).

---

### Comportamento de eventos existentes com fog of war

#### `wall_hp_changed`

Enviado apenas aos clientes que conseguem **ver** o midpoint da parede afetada (LOS ativo). O master sempre recebe. O payload não carrega `wall_type`, portanto nunca vaza a identidade de uma porta secreta não revelada para jogadores.

#### `wall_state_changed`

Para uma `secret_door` **não revelada**, o evento é enviado **apenas ao master**. Para paredes reveladas ou outros tipos, é broadcast normal a todos.

---

### Fluxo de reveal de porta secreta

1. Master envia `enqueue_master_action` com `interact.kind = "reveal"` e `target_ids: ["<wall-uuid>"]`.
2. O servidor marca a parede como `revealed = true` no `MatchSession` e em memória.
3. Servidor emite `wall_revealed` (broadcast a todos) com o `WallSegment` completo e real.
4. Servidor recalcula o LOS de cada jogador e emite `visibility_updated` por jogador.
