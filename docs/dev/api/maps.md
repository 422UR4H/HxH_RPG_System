# Maps API

## POST /campaigns/{campaign_id}/maps — Criar mapa

**Auth:** JWT (master da campanha)

### Request

```json
{
  "name": "Floresta do Norte",
  "description": "Área densa ao norte da cidade, cheia de armadilhas",
  "grid": { "kind": "square", "cols": 25, "rows": 25, "cellSize": 64, "skewRatio": 1.0, "rotation": 0, "color": "#ffffff", "opacity": 0.5, "lineStyle": "solid" },
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
| 422 | `name` vazio, `cellSize ≤ 0`, `cols/rows ≤ 0`, `skewRatio ∉ [0,1]` |
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
      "campaignId": "uuid",
      "name": "Floresta do Norte",
      "description": "Área densa ao norte da cidade, cheia de armadilhas",
      "grid": {
        "kind": "square",
        "cols": 25,
        "rows": 25,
        "cellSize": 64,
        "skewRatio": 1.0,
        "rotation": 0,
        "color": "#ffffff",
        "opacity": 0.5,
        "lineStyle": "solid"
      },
      "bg": null,
      "pieces": [],
      "walls": [],
      "decorations": [],
      "items": [],
      "createdAt": "2026-05-31T00:00:00Z",
      "updatedAt": "2026-05-31T00:00:00Z"
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
    "campaignId": "uuid",
    "name": "Floresta do Norte",
    "description": "Área densa ao norte da cidade, cheia de armadilhas",
    "grid": {
      "kind": "square",
      "cols": 25,
      "rows": 25,
      "cellSize": 64,
      "skewRatio": 1.0,
      "rotation": 0,
      "color": "#ffffff",
      "opacity": 0.5,
      "lineStyle": "solid"
    },
    "bg": null,
    "pieces": [],
    "walls": [],
    "decorations": [],
    "items": [],
    "createdAt": "2026-05-31T00:00:00Z",
    "updatedAt": "2026-05-31T00:00:00Z"
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

**O que cada papel recebe.** O mestre recebe o mapa completo, sem máscara. Quem não é
mestre recebe apenas a **casca**: grid, background, nome — com `pieces` e
`walls` **vazios**.

Isso não é uma limitação temporária. `GET /maps/:id` é servido pelo processo REST
(`cmd/api`), e o que decide o que um jogador pode ver — os polígonos de visibilidade —
vive na memória do processo do game server (`cmd/game`), recalculado a cada movimento. O
REST não tem como filtrar por linha de visão, então não envia tabuleiro nenhum.

O tabuleiro do jogador chega **exclusivamente** por `map_full_state` no WebSocket, tanto
no lobby quanto em partida. Ver a seção de eventos WS abaixo.

---

## PUT /maps/{id} — Atualizar mapa

**Auth:** JWT (master da campanha)

### Request

```json
{
  "name": "Floresta do Norte — Revisado",
  "description": "Nova descrição do mapa",
  "grid": { "kind": "square", "cols": 25, "rows": 25, "cellSize": 64, "skewRatio": 1.0, "rotation": 0, "color": "#ffffff", "opacity": 0.5, "lineStyle": "solid" },
  "bg": null,
  "pieces": [
    { "id": "uuid", "characterId": "uuid", "coord": { "slot": { "kind": "square", "col": 3, "row": 5 }, "z": 0 }, "visible": true }
  ],
  "walls": [
    {
      "id": "uuid",
      "p1": [0, 0],
      "p2": [64, 0],
      "wallType": "wall",
      "material": "stone",
      "move": true,
      "sense": "full",
      "direction": "both",
      "open": false,
      "locked": false,
      "hp": 100,
      "maxHp": 100,
      "resistance": 5,
      "destroyed": false,
      "revealed": false
    }
  ]
}
```

| Campo | Regra (se enviado) |
|---|---|
| `name` | obrigatório, não vazio |
| `description` | opcional |
| `grid` | opcional; mantém grid existente se omitido |
| `bg` | opcional; omitir mantém existente, `null` remove o fundo, objeto troca |
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
| 422 | `name` vazio, `cellSize ≤ 0`, `cols/rows ≤ 0`, `skewRatio ∉ [0,1]`; em `walls`: `p1 = p2`, `wallType` inválido, `hp < 0`, ou ponta fora do contorno LOCAL da grade (`wall segment out of grid bounds`; ver nota abaixo) |
| 500 | Erro interno |

> **Limites das paredes.** O contorno é o `GridLocalBounds` da grade, em coordenadas LOCAL (pré-transform). Na grade quadrada vai de 0 a `cols·cellSize` × `rows·cellSize`; num hex as células são centradas nos pontos da grade, então o contorno **começa antes de 0** (a partir de `-hexW/2` em x) — uma parede na borda esquerda de um hex tem x negativo e é válida. A checagem tem tolerância de `1e-6` (`wallBoundsTolerance`) para absorver ruído de ponto flutuante do cálculo feito no front. **Bug conhecido do handler (não é contrato):** hoje `update_map.go` mapeia todo erro que não seja 403/404 para 422, inclusive falha do repositório; esta deveria ser 500, como na tabela acima. Não dependa do 422 para erros de infraestrutura.

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
  "campaignId": "uuid",
  "name": "Floresta do Norte",
  "description": "Área densa ao norte da cidade, cheia de armadilhas",
  "grid": {
    "kind": "square",
    "cols": 25,
    "rows": 25,
    "cellSize": 64,
    "skewRatio": 1.0,
    "rotation": 0,
    "color": "#ffffff",
    "opacity": 0.5,
    "lineStyle": "solid"
  },
  "bg": null,
  "pieces": [],
  "walls": [
    {
      "id": "uuid",
      "p1": [0, 0],
      "p2": [64, 0],
      "wallType": "wall",
      "material": "stone",
      "move": true,
      "sense": "full",
      "direction": "both",
      "open": false,
      "locked": false,
      "hp": 100,
      "maxHp": 100,
      "resistance": 5,
      "destroyed": false,
      "revealed": false
    }
  ],
  "decorations": [],
  "items": [],
  "createdAt": "2026-05-31T00:00:00Z",
  "updatedAt": "2026-05-31T00:00:00Z"
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
  "wallType": "wall | door | window | secret_door | terrain",
  "material": "stone | wood | iron | magical",
  "doorSubtype": "basic | double | portcullis | drawbridge",
  "windowSubtype": "basic | barred | shuttered",
  "move": true,
  "sense": "full | sight | none",
  "direction": "both | left | right",
  "open": false,
  "locked": false,
  "hp": 100,
  "maxHp": 100,
  "resistance": 5,
  "destroyed": false,
  "revealed": false
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `id` | string (UUID) | Identificador único do segmento, gerado pelo frontend via `crypto.randomUUID()` |
| `p1`, `p2` | `[number, number]` | Endpoints em coordenadas LOCAIS da grade (pré-transform); `p1 ≠ p2` |
| `wallType` | enum | Comportamento funcional |
| `material` | enum | Propriedades físicas (HP, resistência, cor) |
| `doorSubtype` | enum? | Presente apenas quando `wallType = "door"` |
| `windowSubtype` | enum? | Presente apenas quando `wallType = "window"` |
| `move` | bool | Bloqueia movimento físico |
| `sense` | enum | O que bloqueia em termos de percepção |
| `direction` | enum | Direção de bloqueio (both = nos dois sentidos) |
| `open` | bool | Porta/janela está aberta (só relevante para door/window) |
| `locked` | bool | Porta trancada |
| `hp` | int | Pontos de vida atuais (≥ 0) |
| `maxHp` | int | Pontos de vida máximos |
| `resistance` | int | Dano absorvido por ataque |
| `destroyed` | bool | Segmento destruído (visual alterado) |
| `revealed` | bool | Porta secreta já revelada pelo master (sempre `false` para paredes que não são `secret_door`) — ver seção Fog of War abaixo |

### Espaço LOCAL da grade

`p1`/`p2`, `visiblePolygons` e toda a geometria de jogo vivem no espaço LOCAL da grade: o da grade antes de rotação, achatamento e origem. `rotation`/`skewRatio` nunca entram nele; o cliente os aplica só para desenhar.

- **Quadrada:** a célula `(col, row)` ocupa `[col·cellSize, (col+1)·cellSize] × [row·cellSize, (row+1)·cellSize]`; o centro fica em `+cellSize/2` nos dois eixos.
- **Hex:** pointy-top, e `cellSize` é o circunraio (centro → vértice). Os slots são axiais `(q, r)`; o centro é `(cellSize·√3·(q + r/2), 1.5·cellSize·r)`. `cols`/`rows` contam células em offset odd-r, então `col = q + floor(r/2)`.
- **Contorno (`GridLocalBounds`):** quadrada, `[0, cols·cellSize] × [0, rows·cellSize]`. Hex, com `hexW = cellSize·√3`: x vai de `−hexW/2` a `(cols−1)·hexW + hexW/2`, mais `hexW/2` se `rows ≥ 2` (as linhas ímpares deslocam `hexW/2` à direita); y vai de `−cellSize` a `(rows−1)·1.5·cellSize + cellSize`.

A definição executável é o par de goldens `internal/domain/map/service/testdata/grid_geometry_golden.json` (back) e `src/features/tactical-map/utils/__tests__/fixtures/gridGeometryGolden.json` (front), idênticos byte a byte: mude os dois juntos.

### Defaults por tipo (aplicados pelo frontend ao criar o segmento)

| `wallType` | `move` | `sense` | `direction` | `material` padrão |
|---|---|---|---|---|
| `wall` | `true` | `full` | `both` | `stone` |
| `door` | `true` | `full` | `both` | `wood` |
| `window` | `true` | `none` | `both` | `wood` |
| `secret_door` | `true` | `full` | `both` | `stone` |
| `terrain` | `true` | `none` | `left` | — |

### Notas gerais de validação (backend)

- `PUT /maps/:id` aceita `walls` como campo opcional. `null` ou ausente = mantém as paredes existentes. `[]` = remove todas.
- Validações: `p1 ≠ p2`; `wallType` deve ser um dos 5 valores válidos; `hp ≥ 0`.
- O backend não calcula defaults — o frontend envia o objeto completo.

---

## Fog of War / Visibilidade (Fase 10-D)

### `GET /maps/:id` — Filtragem por papel

O endpoint é **role-aware**, mas de forma binária, sem máscara campo a campo: o mestre
recebe o mapa completo, sem máscara; qualquer outro participante recebe apenas a casca
(`pieces` e `walls` vazios). Ver "O que cada papel recebe" na seção
`GET /maps/{id} — Obter mapa` acima para o detalhe completo e o porquê (separação de
processos REST/game server).

A máscara de porta secreta não revelada e a filtragem por LOS/visibilidade não acontecem
mais na camada REST — são responsabilidade exclusiva do WebSocket, entregues via
`map_full_state` (ver abaixo).

---

### Campo `fog_mode` no mapa

O `TacticalMap` possui o campo `fog_mode`, persistido no banco, que controla o comportamento de exploração durante a partida:

| Valor | Comportamento |
|---|---|
| `"explored"` | Paredes já vistas continuam visíveis (mesmo fora do LOS atual), gravadas na memória do jogador por id de parede. **Default** quando o campo está vazio no banco. |
| `"live"` | Apenas o LOS do turno atual é visível; nenhuma memória de exploração. |

O campo é enviado pelo servidor no payload WS `map_full_state` como `fogMode` (ver abaixo). Ele não é exposto na resposta REST de `GET /maps/:id` atualmente — apenas o WebSocket o carrega para o cliente.

---

### Eventos WebSocket — Fog of War (servidor → cliente)

Todos os payloads usam camelCase. Esses eventos são enviados durante uma partida em andamento (após `start_match`).

---

#### `map_full_state`

Enviado a cada cliente que se conecta sempre que a sala tem tabuleiro — mesmo um tabuleiro sem peça nem parede, só com grade e fundo — ou quando o tabuleiro do lobby acaba de ser limpo (mapa desanexado); re-enviado após operações que alteram o estado do tabuleiro. Cada cliente recebe uma versão filtrada por papel.

**Payload (jogador não-master):**

```json
{
  "pieces": [
    {
      "pieceId": "uuid-string",
      "slot": { "kind": "square", "col": 3, "row": 5 },
      "characterId": "uuid-string",
      "visible": true
    }
  ],
  "walls": [ /* WallSegment[] filtrados e mascarados por LOS + masking de secret_door */ ],
  "visiblePolygons": [
    [ { "x": 0.0, "y": 0.0 }, { "x": 64.0, "y": 0.0 }, { "x": 64.0, "y": 64.0 } ]
  ],
  "fogMode": "explored",
  "grid": { "kind": "square", "cols": 20, "rows": 20, "cellSize": 64, "skewRatio": 1, "rotation": 0, "color": "#000000", "opacity": 0.5, "lineStyle": "solid" },
  "bg": { "url": "https://...", "x": 0, "y": 0, "width": 1280, "height": 1280, "rotation": 0, "opacity": 1 }
}
```

**Payload (master):**

```json
{
  "pieces": [ /* todas as peças, sem filtro */ ],
  "walls": [ /* todas as paredes, sem máscara */ ],
  "fogMode": "explored",
  "grid": { "kind": "square", "cols": 20, "rows": 20, "cellSize": 64, "skewRatio": 1, "rotation": 0, "color": "#000000", "opacity": 0.5, "lineStyle": "solid" },
  "bg": { "url": "https://...", "x": 0, "y": 0, "width": 1280, "height": 1280, "rotation": 0, "opacity": 1 }
}
```

| Campo | Descrição |
|---|---|
| `grid` | A grade do **tabuleiro da partida** (mesmo formato de `grid` no REST). A mesa desenha com esta, não com a do mapa da campanha. Ausente só quando a sala não tem tabuleiro. |
| `bg` | O fundo do **tabuleiro da partida** (mesmo formato de `bg` no REST: `url, x, y, width, height, rotation, opacity`); `null` quando o tabuleiro não tem fundo. Ausente junto com `grid`, quando a sala não tem tabuleiro. A mesa desenha com este, não com o do mapa da campanha. |

Vale para mestre e jogador, na mesa e no lobby: os dois desenham a grade e o fundo daqui, nunca os do mapa da campanha.

> O master não recebe `visiblePolygons` — campo omitido.

| Campo | Tipo | Descrição |
|---|---|---|
| `pieces` | `PieceMovedPayload[]` | Peças visíveis pelo receptor. Para o master, todas. Para jogador, apenas as em LOS ou próprias com `visible=true`. Peças nunca entram na memória do jogador — personagens se movem livremente, então a última posição vista informaria mal. Cada peça inclui `z` (elevação em metros, 0 = chão, omitido quando 0). |
| `walls` | `WallSegment[]` | Paredes visíveis. Para jogador, qualquer parede em LOS agora, mais (em modo `explored`) qualquer parede já vista antes — gravada na memória do jogador por id, não por região do mapa. No lobby (sem partida, sem LOS) o jogador recebe todas as paredes do tabuleiro, que o lobby desenha. Portas secretas não reveladas mascaradas como `"wall"`, na partida e no lobby. |
| `visiblePolygons` | `[{x,y}][]` | Polígonos de visibilidade atuais do jogador em coordenadas LOCAIS da grade (pré-transform, o mesmo espaço de `p1`/`p2` das paredes); o cliente aplica o transform da grade (rotação, achatamento, origem) a cada vértice para desenhar. Omitido para o master e no lobby (sem partida ativa). O cliente usa este polígono como máscara de stencil: nítido dentro dele, esmaecido fora — não há dado de memória separado no payload. |
| `fogMode` | `string` | `"live"` ou `"explored"`. Sempre presente. No lobby (sem sessão ativa), o valor enviado é `"live"` como placeholder. |

##### Máscara de porta secreta não revelada

Quando um jogador recebe, dentro de `walls`, um `WallSegment` com `wallType = "secret_door"`
e `revealed = false`, o game server substitui os campos identificadores da porta pelo
equivalente de uma parede comum, preservando todos os campos de combate:

| Campo | Valor mascarado |
|---|---|
| `wallType` | `"wall"` (em vez de `"secret_door"`) |
| `doorSubtype` | ausente (`null` / omitido) |
| `windowSubtype` | ausente (`null` / omitido) |
| `open` | `false` |
| `locked` | `false` |
| `id`, `p1`, `p2`, `material`, `hp`, `maxHp`, `resistance`, `move`, `sense`, `direction`, `destroyed` | preservados sem alteração |

Aplicado por `MaskSecretDoorForPlayer`, chamado pelo `FilterMapState` do processo do game
server (`cmd/game`) — nunca pelo processo REST (`cmd/api`), que não tem acesso a esse estado.

---

#### `visibility_updated`

Enviado a cada jogador individual quando seu LOS muda (ex.: após movimento de peça ou revelação de porta secreta). Nunca enviado ao master.

```json
{
  "visiblePolygons": [
    [ { "x": 0.0, "y": 0.0 }, { "x": 128.0, "y": 0.0 } ]
  ]
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `visiblePolygons` | `[{x,y}][]` | Novo conjunto completo de polígonos de visibilidade do jogador, em coordenadas LOCAIS da grade (pré-transform, o mesmo espaço de `p1`/`p2` das paredes); o cliente aplica o transform da grade para desenhar. |

---

#### `wall_revealed`

Broadcast para **todos os clientes** quando o master revela uma porta secreta. Após este evento, todos os clientes devem substituir a entrada correspondente em seu estado local pelo `WallSegment` completo e real recebido.

```json
{
  "wall": {
    "id": "uuid-string",
    "p1": [0.0, 0.0],
    "p2": [64.0, 0.0],
    "wallType": "secret_door",
    "material": "wood",
    "doorSubtype": "basic",
    "move": true,
    "sense": "full",
    "direction": "both",
    "open": false,
    "locked": false,
    "hp": 80,
    "maxHp": 80,
    "resistance": 3,
    "destroyed": false,
    "revealed": true
  }
}
```

| Campo | Tipo | Descrição |
|---|---|---|
| `wall` | `WallSegment` | O `WallSegment` completo e real da porta secreta, agora com `revealed=true`. |

> O master aciona a revelação via `enqueue_master_action` com `interact.kind = "reveal"` e `targetIds` contendo o(s) ID(s) da(s) parede(s).

---

### Comportamento de eventos existentes com fog of war

#### `wall_hp_changed`

Enviado apenas aos clientes que conseguem **ver** a parede afetada — qualquer trecho dela, pelo mesmo teste que a põe na tela do jogador no `map_full_state` (`IsWallInSight`; ver [game-lobby.md](game-lobby.md#paredes-que-o-jogador-recebe-em-map_full_state)). Antes testava só o ponto médio, que fica **sobre** a borda do polígono de visão de quem está de frente para a parede, e o jogador encarando a parede não recebia a notícia. O master sempre recebe. Sai no **fechamento** do turno que atacou a parede (o golpe cai com o dano nos personagens) e quando o mestre derruba a parede pela mão dele (`interact.kind = "destroy"`). O payload não carrega `wallType`, portanto nunca vaza a identidade de uma porta secreta não revelada para jogadores.

#### `wall_state_changed`

Para uma `secret_door` **não revelada**, o evento é enviado **apenas ao master**. Para paredes reveladas ou outros tipos, é broadcast normal a todos.

---

### Fluxo de reveal de porta secreta

1. Master envia `enqueue_master_action` com `interact.kind = "reveal"` e `targetIds: ["<wall-uuid>"]`.
2. O servidor marca a parede como `revealed = true` no `MatchSession` e em memória.
3. Servidor emite `wall_revealed` (broadcast a todos) com o `WallSegment` completo e real.
4. Servidor recalcula o LOS de cada jogador e emite `visibility_updated` por jogador.
