# Plano — revisão dos PRs de espaços de coordenadas (back #88 × front #75)

Origem: revisão de `docs/superpowers/plans/2026-10-11-grid-coord-spaces.md`, já implementado na
branch `fix/grid-coord-spaces` dos dois repos. Este plano corrige o que a revisão achou e põe
testes de não-regressão que amarram o back ao front.

Repos:
- Back: `/home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System`
- Front: `/home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System_React`

Os dois já estão na branch `fix/grid-coord-spaces`. Trabalhe direto nesses checkouts (não crie
worktree, não troque de branch). Commits só com `git add` dos seus arquivos + `git commit`;
nada de `git stash`, `checkout`, `reset`, `push`.

## Global Constraints

- **O espaço LOCAL é o de `WallSegment.p1/p2`** (antes de rotação/achatamento/origem). Toda
  regra de jogo (fog, bloqueio de movimento, alcance de parede, distâncias) mede em LOCAL. O
  WORLD (depois de `applyTransform`) é só para desenhar e para hit-test de ponteiro.
- **Hex é pointy-top, `size = cellSize` (circunraio)**, axial `(q, r)` no slot e no wire; a grade
  guarda `cols/rows` em offset odd-r: `offsetToAxial(col, row) = { q: col - floor(row/2), r: row }`.
- **O desenho do front é a verdade visual**: quando back e front divergem, o back se alinha ao front.
- Comparações de ponto flutuante em teste: tolerância `1e-9`.
- Back: TDD com o `testing` padrão, table-driven com `t.Run`. Ao fim de cada task de back:
  `go vet ./...` e `go test` dos pacotes tocados.
- Front: TDD com vitest. Ao fim de cada task de front: `npx tsc -b` e `npx vitest run <arquivos tocados>`.
  `verbatimModuleSyntax` está ligado: import só de tipo usa `import type`.
- Comentários e mensagens de teste seguem o idioma do arquivo que você está editando.

## Task 1: Back — paredes na borda hex não podem ser recusadas por ruído de ponto flutuante

**Contexto.** `ValidateWallSegments` (`internal/domain/map/service/map_validator.go`) compara cada
ponta de parede com `GridLocalBounds` sem tolerância. O front calcula o vértice de um hexágono
como `centro + cellSize·cos(ângulo)`, e o vértice da borda esquerda sai 1 ulp além de
`minX = -hexW/2`. Com `cellSize = 64` (o padrão): `minX = -55.42562584220407`, mas o vértice
que o front manda é `-55.42562584220408`. Resultado: toda parede desenhada sobre a borda
esquerda de um mapa hex é recusada com 422 `wall segment out of grid bounds`.

**Mudança.**
- Em `map_validator.go`, uma constante não exportada com comentário do porquê:
  `wallBoundsTolerance = 1e-6` (unidades LOCAIS: muito acima do ruído de float64 — ~1e-14 —
  e muito abaixo de um pixel). A checagem vira
  `p[0] < minX-tol || p[0] > maxX+tol || p[1] < minY-tol || p[1] > maxY+tol`.

**Testes (RED antes da mudança).** Em `map_validator_test.go`:
1. `TestValidateWallSegments_HexBorderVertexAsTheFrontComputesIt`: grade hex `Cols=10, Rows=10,
   CellSize=64` (parta de `entity.DefaultGrid()` e mude `Kind/Cols/Rows/CellSize`). Parede com
   `P1 = {-55.42562584220408, 31.999999999999996}` e `P2 = {-55.42562584220407, -32.00000000000001}`
   — a aresta esquerda do hexágono (0,0) exatamente como o JavaScript calcula
   (`64*Math.cos(150°)`, `64*Math.sin(150°)` e o par de 210°). Espera `nil`. Comentário dizendo
   de onde vêm os números.
2. No mesmo teste (ou subteste), uma parede com uma ponta em `minX - 0.5` continua recusada com
   `ErrWallOutOfBounds` (a tolerância não pode virar folga visível).

## Task 2: Back — fog numa grade hex com parede, em escala do front, na sala de verdade

**Contexto.** O PR tem teste de fog com grade girada (quadrada) e com hex sem paredes. Falta o
caso que pega a regressão de escala hex na sala: parede em coordenadas do front entre duas
peças hex. Com a escala antiga (`CellSize/2`) a peça "de trás" ficava ANTES da parede.

**Teste novo** em `internal/app/game/fog_grid_spaces_test.go` (reuse `fogRoomOn`, `hexSlot`,
`recompute`, `decodeMapFull` que já estão lá):
`TestFog_HexGrid_LocalWallHidesThePieceBehindIt`
- Grade: `mapentity.GridShape{Kind: mapentity.GridKindHex, Cols: 10, Rows: 10, CellSize: 64, SkewRatio: 1}`.
- Peças (axial): própria em `hexSlot(1, 0)`, `"near"` em `hexSlot(2, 0)`, `"behind"` em `hexSlot(4, 0)`.
  Centros LOCAIS: `x = 64·√3·q` → 110.85…, 221.70…, 443.40… (todas em y = 0).
- Parede: a aresta compartilhada pelos hexágonos (2,0) e (3,0), em números LITERAIS (não derive
  de `SlotCenterToLocal` — o teste existe para pegar se a escala do Go mudar):
  `P1 = {277.12812921102035, -32}`, `P2 = {277.12812921102035, 32}` (x = 2,5·64·√3).
  Mesmos atributos da parede do teste girado (`WallTypeWall`, `WallMaterialStone`,
  `Sense: mapentity.SenseSight`, `HP/MaxHP 100`).
- Espera: o jogador vê `"own"` e `"near"`; não vê `"behind"`. Comentário: com o hex na metade do
  tamanho, a peça (4,0) ficava em x≈221 — antes da parede — e apareceria.
- Confirme o RED trocando temporariamente `size := g.CellSize` por `size := g.CellSize / 2` em
  `baseCenter` (`internal/domain/map/service/coords.go`), rodando o teste, e desfazendo. Registre
  isso no relatório. Não commite a troca.

## Task 3: Back — golden de geometria da grade compartilhado com o front

**Contexto.** A causa-raiz dos bugs deste PR foi o back e o front calcularem a mesma geometria
de jeitos diferentes, com testes que só validavam cada lado contra ele mesmo. Este golden é UM
arquivo de casos com números literais, copiado byte a byte nos dois repos (a Task 4 faz a cópia
no front). Cada lado tem um teste que lê o arquivo e confere a própria implementação. Quem mudar
a geometria de um lado quebra o teste desse lado.

**Arquivo** `internal/domain/map/service/testdata/grid_geometry_golden.json`, com EXATAMENTE este
conteúdo (o front copia igual):

```json
{
  "_doc": "Golden de geometria da grade, em espaço LOCAL (o de WallSegment.p1/p2). Cópia idêntica em System_X_System/internal/domain/map/service/testdata/grid_geometry_golden.json e System_X_System_React/src/features/tactical-map/utils/__tests__/fixtures/gridGeometryGolden.json: mude os dois juntos. Slots são [a, b] = [col, row] no quadrado e [q, r] axial no hex. hex40Iso tem rotação e achatamento para provar que o LOCAL os ignora. Números com tolerância 1e-9.",
  "grids": {
    "square40": { "kind": "square", "cols": 10, "rows": 10, "cellSize": 40, "skewRatio": 1, "rotation": 0 },
    "hex40": { "kind": "hex", "cols": 10, "rows": 10, "cellSize": 40, "skewRatio": 1, "rotation": 0 },
    "hex40Iso": { "kind": "hex", "cols": 10, "rows": 10, "cellSize": 40, "skewRatio": 0.6, "rotation": 30 },
    "hex64x3": { "kind": "hex", "cols": 3, "rows": 3, "cellSize": 64, "skewRatio": 1, "rotation": 0 },
    "hex64Row": { "kind": "hex", "cols": 4, "rows": 1, "cellSize": 64, "skewRatio": 1, "rotation": 0 }
  },
  "slotCenters": [
    { "grid": "square40", "slot": [0, 0], "x": 20, "y": 20 },
    { "grid": "square40", "slot": [3, 7], "x": 140, "y": 300 },
    { "grid": "hex40", "slot": [0, 0], "x": 0, "y": 0 },
    { "grid": "hex40", "slot": [1, 0], "x": 69.28203230275508, "y": 0 },
    { "grid": "hex40", "slot": [0, 1], "x": 34.64101615137754, "y": 60 },
    { "grid": "hex40", "slot": [-1, 2], "x": 0, "y": 120 },
    { "grid": "hex40", "slot": [2, 3], "x": 242.4871130596428, "y": 180 },
    { "grid": "hex40Iso", "slot": [0, 1], "x": 34.64101615137754, "y": 60 },
    { "grid": "hex40Iso", "slot": [2, 3], "x": 242.4871130596428, "y": 180 }
  ],
  "localToSlot": [
    { "grid": "square40", "x": 20, "y": 20, "slot": [0, 0] },
    { "grid": "square40", "x": 139.9, "y": 300, "slot": [3, 7] },
    { "grid": "square40", "x": 160, "y": 300, "slot": [4, 7] },
    { "grid": "hex40", "x": 0, "y": 0, "slot": [0, 0] },
    { "grid": "hex40", "x": 74.28, "y": 3, "slot": [1, 0] },
    { "grid": "hex40", "x": 34.64, "y": 50, "slot": [0, 1] },
    { "grid": "hex40", "x": 0, "y": 120, "slot": [-1, 2] },
    { "grid": "hex40", "x": 252.49, "y": 170, "slot": [2, 3] },
    { "grid": "hex40Iso", "x": 252.49, "y": 170, "slot": [2, 3] }
  ],
  "bounds": [
    { "grid": "square40", "minX": 0, "minY": 0, "maxX": 400, "maxY": 400 },
    { "grid": "hex40", "minX": -34.64101615137754, "minY": -40, "maxX": 692.8203230275509, "maxY": 580 },
    { "grid": "hex40Iso", "minX": -34.64101615137754, "minY": -40, "maxX": 692.8203230275509, "maxY": 580 },
    { "grid": "hex64x3", "minX": -55.42562584220407, "minY": -64, "maxX": 332.5537550532244, "maxY": 256 },
    { "grid": "hex64Row", "minX": -55.42562584220407, "minY": -64, "maxX": 387.9793808954285, "maxY": 64 }
  ],
  "slotInGrid": [
    { "grid": "square40", "slot": [0, 0], "inGrid": true },
    { "grid": "square40", "slot": [9, 9], "inGrid": true },
    { "grid": "square40", "slot": [10, 0], "inGrid": false },
    { "grid": "square40", "slot": [0, -1], "inGrid": false },
    { "grid": "hex40", "slot": [-2, 5], "inGrid": true },
    { "grid": "hex40", "slot": [-3, 5], "inGrid": false },
    { "grid": "hex40", "slot": [9, 0], "inGrid": true },
    { "grid": "hex40", "slot": [10, 0], "inGrid": false },
    { "grid": "hex40", "slot": [0, 10], "inGrid": false },
    { "grid": "hex40", "slot": [-4, 9], "inGrid": true },
    { "grid": "hex40", "slot": [5, 9], "inGrid": true },
    { "grid": "hex40", "slot": [6, 9], "inGrid": false }
  ],
  "wallContactSlots": [
    { "grid": "square40", "p1": [120, 200], "p2": [160, 200], "slots": [[3, 4], [3, 5]] },
    { "grid": "square40", "p1": [160, 80], "p2": [160, 160], "slots": [[3, 2], [3, 3], [4, 2], [4, 3]] },
    { "grid": "square40", "p1": [0, 0], "p2": [40, 0], "slots": [[0, 0]] },
    { "grid": "hex40", "p1": [34.64101615137755, -20], "p2": [34.64101615137755, 20], "slots": [[0, 0], [1, 0]] },
    { "grid": "hex40", "p1": [34.64101615137755, 20], "p2": [0, 40], "slots": [[0, 0], [0, 1]] },
    { "grid": "hex40Iso", "p1": [34.64101615137755, -20], "p2": [34.64101615137755, 20], "slots": [[0, 0], [1, 0]] }
  ],
  "pathBlocked": [
    { "grid": "square40", "from": [4, 4], "to": [5, 4], "wall": { "p1": [200, 160], "p2": [200, 200], "direction": "both" }, "blocked": true },
    { "grid": "square40", "from": [4, 4], "to": [5, 4], "wall": { "p1": [200, 200], "p2": [200, 240], "direction": "both" }, "blocked": false },
    { "grid": "square40", "from": [4, 4], "to": [5, 4], "wall": { "p1": [200, 160], "p2": [200, 200], "direction": "left" }, "blocked": true },
    { "grid": "square40", "from": [5, 4], "to": [4, 4], "wall": { "p1": [200, 160], "p2": [200, 200], "direction": "left" }, "blocked": false },
    { "grid": "hex40", "from": [0, 0], "to": [1, 0], "wall": { "p1": [34.64101615137755, -20], "p2": [34.64101615137755, 20], "direction": "both" }, "blocked": true },
    { "grid": "hex40", "from": [0, 0], "to": [0, 1], "wall": { "p1": [34.64101615137755, -20], "p2": [34.64101615137755, 20], "direction": "both" }, "blocked": false },
    { "grid": "hex40", "from": [1, 0], "to": [2, 0], "wall": { "p1": [103.92304845413263, -20], "p2": [103.92304845413263, 20], "direction": "both" }, "blocked": true }
  ]
}
```

**Teste** `internal/domain/map/service/grid_geometry_golden_test.go` (`package service_test`):
- Lê o JSON com `os.ReadFile("testdata/grid_geometry_golden.json")` e decodifica em structs
  locais do teste com tags camelCase (o `entity.GridShape` tem tags snake_case — não decodifique
  direto nele; monte o `entity.GridShape` a partir da struct do teste: `Kind`, `Cols`, `Rows`,
  `CellSize`, `SkewRatio`, `Rotation`).
- Um `t.Run` por seção e um subteste por caso, nomeado pelo grid e pelos números do caso:
  - `slotCenters` → `service.SlotCenterToLocal(a, b, g)` ≈ `(x, y)`.
  - `localToSlot` → `service.LocalToSlot(x, y, g)` == `slot`.
  - `bounds` → `service.GridLocalBounds(g)` ≈ `(minX, minY, maxX, maxY)`.
  - `slotInGrid` → `service.SlotInGrid(a, b, g)` == `inGrid`.
  - `wallContactSlots` → `service.WallContactSlots(w, g)`, ordenado por `(a, b)`, == `slots`
    (parede com `P1/P2` do caso; os demais campos não importam ali).
  - `pathBlocked` → centros por `service.SlotCenterToLocal`, parede com `Move: true`,
    `Direction` do caso (`entity.WallDirection`), resto zero; `service.IsPathBlocked(from, to, walls)`
    == `blocked`.
- Mensagens de falha dizem o caso e o que veio.
- Se algum caso falhar contra o código atual, PARE e reporte (o golden foi derivado da lógica do
  front; uma divergência é achado, não ajuste de teste).

## Task 4: Front — o mesmo golden, do lado do front

**Arquivo** `src/features/tactical-map/utils/__tests__/fixtures/gridGeometryGolden.json`: cópia
byte a byte de `System_X_System/internal/domain/map/service/testdata/grid_geometry_golden.json`
(use `cp`; confira com `cmp`).

**Teste** `src/features/tactical-map/utils/__tests__/gridGeometryGolden.test.ts`:
- Importa o JSON (o projeto já importa fixture JSON em `fogDraw.test.ts` — siga o mesmo jeito).
- Monta cada `GridShape` do caso com os campos do golden mais os de apresentação que o tipo
  exige (`color: "#000", opacity: 1, lineStyle: "solid"`).
- Slot do golden → `SlotCoord`: quadrado `{ kind: "square", col: a, row: b }`, hex
  `{ kind: "hex", q: a, r: b }` (pelo `kind` do grid).
- Um `describe` por seção, `it.each` por caso:
  - `slotCenters` → `slotToLocal(slot, grid)` ≈ `(x, y)` (`toBeCloseTo(v, 9)`).
  - `localToSlot` → `localToSlot({x, y}, grid)` igual ao slot (normalize `-0` se preciso).
  - `bounds` → `gridLocalBounds(grid)` ≈ os quatro valores.
  - `slotInGrid` → `isSlotInBounds(slot, grid)` == `inGrid`.
  - `wallContactSlots` → `wallContactSlots(wall, grid)` (de `../wallReach`), convertido para
    `[a, b]` e ordenado por `(a, b)`, igual a `slots`. Monte a parede como o helper `wall()` de
    `wallReach.test.ts` (`direction: "both"`, `move: true`, `open: false`, `destroyed: false`).
  - `pathBlocked` → `isPathClear(from, to, [wall], grid)` (de `../wallReach`) == `!blocked`, com
    a parede do caso (`direction` do caso, `move: true`, `open: false`, `destroyed: false`).
- Se algum caso falhar, PARE e reporte (achado de divergência, não ajuste de teste).

## Task 5: Front — pontos de encaixe de parede na grade hex seguem o desenho (offset odd-r)

**Bug.** `collectGridSnapPoints` (`src/features/tactical-map/utils/walls.ts`) percorre o hex como
`for r < rows, for q < cols: hexToPixel({ q, r })` — trata `(col, row)` como axial. O
`GridLayer` desenha a mesma grade com `offsetToAxial(col, row)`. Os pontos de encaixe formam um
losango que escorrega meia célula para a direita a cada linha: na metade de baixo/esquerda da
grade desenhada não há encaixe (e `explodePolyline` não divide paredes ali), e sobram pontos
fora da grade, à direita, que o servidor recusa como fora do contorno.

**Mudança.** No ramo hex, iterar `for (let row = 0; row < rows; row++) for (let col = 0; col < cols; col++)`
e usar `hexToPixel(offsetToAxial(col, row), cellSize)` (importar `offsetToAxial` de `./coords`;
`coords.ts` não importa `walls.ts`, não há ciclo). O resto (centro + 6 vértices) fica igual.

**Testes (RED antes)** em `walls.test.ts`, `describe("collectGridSnapPoints — hex")`, grade hex
`cols: 6, rows: 6, cellSize: 40` (com os campos de apresentação):
1. Todo vértice de todo hexágono desenhado é ponto de encaixe: para cada `(col, row)`, centro
   `hexToPixel(offsetToAxial(col, row), 40)`, vértices em `60·i − 30` graus, `i = 0..5`, raio 40;
   cada um tem um ponto de encaixe a menos de `1e-9`.
2. Todo ponto de encaixe está dentro de `gridLocalBounds(grid)` (tolerância `1e-9`).
3. Explícito: o centro do hexágono da coluna 0, linha 4 — axial `(-2, 4)`, ponto `(0, 240)` —
   é ponto de encaixe (no código antigo não é).

## Task 6: Front — o mapa aceita clique em toda a grade (hit area das peças) e o clamp usa o contorno real

**Bug.** `PiecesLayer` (`src/features/tactical-map/stage/PiecesLayer.tsx`) dá ao container das
peças `hitArea = new Rectangle(0, 0, cols·cellSize, rows·cellSize)`. No Pixi v8 o
`EventBoundary.hitPruneFn` poda o container E TODOS OS FILHOS quando o ponteiro cai fora da
`hitArea`. Numa grade hex (≈ √3 mais larga, começa em x < 0 e y < 0), girada, achatada ou com
origem, parte das peças não recebe clique nem arrasto, e o toque em slot vazio
(`onEmptySlotClick`) não dispara ali. Jogador e mestre (`GamePlayerPage`, `GameMasterPage`)
passam `piecesInteractive`.

**Mudança.**
1. `src/features/tactical-map/utils/coords.ts`: nova
   `export function gridWorldCorners(grid: GridShape): XY[]` — os 4 cantos de `gridLocalBounds`
   na ordem TL, TR, BR, BL, cada um por `applyTransform`. `gridWorldBounds` passa a usá-la (mesmo
   resultado de hoje). Comentário curto: é o contorno da grade em WORLD; a caixa de
   `gridWorldBounds` o envolve.
2. `PiecesLayer.tsx`: `gridHitArea` vira
   `new Polygon(gridWorldCorners(map.grid).flatMap((p) => [p.x, p.y]))`, memoizado em `[map.grid]`.
   Troque o import de `Rectangle` por `Polygon` (de `"pixi.js"`) se `Rectangle` não for usado em
   outro lugar do arquivo. Atualize o comentário acima do `useMemo` dizendo por que o contorno
   real (o Pixi poda os filhos fora da hitArea).
3. `src/features/tactical-map/stage/ViewportInner.tsx`, efeito do `clampToGrid`: os limites vêm
   de `gridWorldBounds(map.grid)` (`left: minX, right: maxX, top: minY, bottom: maxY`), dependências
   `[clampToGrid, map.grid]`.

**Testes (RED antes do item 1)** em `coords.test.ts`, `describe("gridWorldCorners")`:
1. Numa grade `{ ...squareGrid(40), rotation: 30, skewRatio: 0.6, originX: 15, originY: -20 }`:
   os 4 pontos são `applyTransform` dos cantos de `gridLocalBounds`, na ordem TL, TR, BR, BL.
2. Grade hex `cols: 10, rows: 10, cellSize: 64`, `rotation: 30, skewRatio: 0.6, originX: 15, originY: -20`:
   o centro WORLD (`slotToWorld`) de TODO slot da grade (`offsetToAxial(col, row)` para todo
   `col < 10, row < 10`) está dentro do quadrilátero de `gridWorldCorners` (teste de ponto em
   polígono convexo por sinal do produto vetorial, helper local no teste).
3. Premissa do bug, na grade hex sem transform (`rotation: 0, skewRatio: 1`): algum centro WORLD
   tem `x > cols·cellSize` ou `x < 0` ou `y < 0` — ou seja, o retângulo antigo não o cobria.

A camada Pixi não tem teste (o `@pixi/react` é mockado); a ligação em `PiecesLayer`/`ViewportInner`
é verificada no browser pelo controlador.

## Task 7: Front — a aproximação desempata pela distância no tabuleiro (LOCAL), não na tela

**Bug.** `approachReach` (`src/features/tactical-map/utils/reach.ts`) desempata candidatos com o
mesmo número de passos pela distância entre centros em WORLD (`slotToWorld`). Numa grade
isométrica (girada 45° e achatada) a diagonal fica mais curta NA TELA que o vizinho reto, e a
proposta escolhe a diagonal — que é mais longe no tabuleiro. O comentário da função diz
"distância real entre centros": a real é a LOCAL.

**Mudança.** Trocar `slotToWorld` por `slotToLocal` nas duas chamadas de `approachReach` (e no
import). Nada mais.

**Teste (RED antes)** em `reach.test.ts` (ou onde `approachReach` já é testado):
grade quadrada `cols: 10, rows: 10, cellSize: 40, rotation: 45, skewRatio: 0.5`; ator `(0,0)`;
`reach = [sq(2, 2), sq(2, 0)]` (nessa ordem); `isFree: () => true`; sem `isPathClear`. Espera
`{ kind: "approach", slot: sq(2, 0) }`. Comentário com os números: em LOCAL, (2,0) está a 80 e
(2,2) a 113; na tela isométrica, (2,0) fica a ≈63 e (2,2) a ≈57.

## Task 8: Docs (back e front)

Back (`System_X_System`):
- `docs/dev/api/maps.md`, tabela de respostas do `PUT /maps/{id}`: a linha 422 passa a citar
  também as paredes: `p1 = p2`, `wallType` inválido, `hp < 0`, ponta fora do contorno LOCAL da
  grade (`GridLocalBounds`: num hex vai de `-hexW/2` a …, começa antes de 0; tolerância de 1e-6
  para ruído de ponto flutuante) → `wall segment out of grid bounds`. Confira os nomes exatos dos
  erros em `internal/domain/map/service/map_validator.go`.

Front (`System_X_System_React`):
- `src/types/tacticalMap.ts`: o comentário de `VisibilityPolygon` diz "FogLayer applies
  applyTransform per vertex" → "o ViewportInner converte uma vez com polygonsToWorld".
- `docs/dev/tactical-map/pixi-stack.md`, seção "Fog de guerra": um parágrafo curto — o polígono
  chega do servidor em LOCAL e o `ViewportInner` o converte uma vez (`polygonsToWorld`) para o
  `FogLayer` e o `LosSplit`; o retângulo do fog é `gridWorldBounds` + `FOG_PADDING`. E uma nota
  nas armadilhas: a `hitArea` de um container poda os filhos fora dela — a do `PiecesLayer` é o
  contorno WORLD da grade (`gridWorldCorners`), não `cols·cellSize × rows·cellSize`.
- Commits separados por repo.

## Fora do escopo (achado na revisão, registrado no PR)

- `TacticalMapEditor.handleSave` ("Truncation check"): conta colunas/linhas não cobertas pelo
  fundo com `cols·cellSize`/`cellSize` e ignora hex e rotação; para hex sugere cortar colunas na
  unidade errada. Corrigir exige decidir o que é "coluna descoberta" num hex com fileiras
  deslocadas — decisão de produto, não deste PR.

## Verificação final (controlador)

- Back: `go vet ./...`, `go vet -tags smoke ./...`, `go vet -tags integration ./...`,
  `go test ./...`, `golangci-lint run --max-same-issues 0 --max-issues-per-linter 0`.
- Front: `npx tsc -b`, `npm run lint` (só os 2 erros antigos de sheet/Diagram), `npm run test`.
- Browser, com o `devseed` e a grade hex na mesa: peça na direita do hex recebe clique;
  fog continua certo.
