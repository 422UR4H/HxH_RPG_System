# Plano — espaços de coordenadas da grade (back × front)

Origem: `docs/dev/match/handoff/coordenadas-grade.md`. Branch `fix/grid-coord-spaces` nos dois repos.

## Decisão

1. **O servidor calcula tudo em espaço LOCAL** (o espaço de `WallSegment.P1/P2`, antes do
   transform de rotação/achatamento/origem). Peças, paredes, contorno do tabuleiro e polígonos
   de visibilidade. O `applyTransform` do Go deixa de existir: a divergência de transform
   (handoff §2) some do servidor, porque só o front transforma, e só para desenhar.
   - É exato: o transform é afim (rotação + escala vertical positiva + translação). Visibilidade
     por raio comuta com transform afim e preserva orientação, então o polígono LOCAL
     transformado é o polígono WORLD — inclusive para paredes de mão única (sinal do produto
     vetorial preservado).
2. **Hex: `size = CellSize`** (circunraio, centro→vértice), como o front desenha
   (`hexToPixel(…, grid.cellSize)`, `walls.ts` encaixa vértices em raio `cellSize`). O Go usava
   `CellSize/2` sem motivo registrado (commit 1b98a6e). Paredes salvas já estão na escala do
   front, peças são slots — **não há migração de dados**.
3. **Wire:** `visiblePolygons` (em `map_full_state` e `visibility_updated`) passa a vir em
   coordenadas LOCAIS da grade. O front aplica `applyTransform(p, grid)` a cada vértice antes de
   desenhar. Mudança de semântica do contrato → atualizar `docs/dev/api/maps.md`.

## Task 1: Back — `System_X_System/` (TDD: teste falhando primeiro em cada item)

### B1. Hex na escala do front — `internal/domain/map/service/coords.go`
- `baseCenter`, `GridLocalBounds`, `LocalToSlot`: `size := g.CellSize` (era `/2`). Atualizar os
  comentários (inclusive o de `GridLocalBounds`: "one hex radius above y=0" continua certo).
- Teste novo em `coords_test.go`: centro de hex (q=1, r=0) com CellSize 64 = `(64·√3, 0)`;
  (0, 1) = `(64·√3/2, 96)` — os números de `hexToPixel` do front. `GridLocalBounds` hex
  conferido contra `gridLocalBounds` do front (`src/features/tactical-map/utils/coords.ts`)
  para Cols=Rows=3, CellSize=64: minX=-hexW/2, minY=-64, maxX=2·hexW+hexW/2+hexW/2, maxY=2·96+64,
  com hexW=64·√3.
- Consertar os testes de hex que validavam o Go contra ele mesmo:
  `internal/app/game/movement_geometry_e2e_test.go`,
  `internal/domain/matchsession/escape_landing_test.go` (caminho real:
  `internal/domain/match/matchsession/escape_landing_test.go`), `wall_reach_test.go`,
  `coords_test.go`. Onde um teste monta parede a partir de centro de slot, ele deve usar
  `SlotCenterToLocal` (não números fixos na escala antiga).

### B2. Remover o espaço WORLD do servidor
- `coords.go`: apagar `applyTransform`, `reverseTransform`, `SlotCenterToWorld`, `WorldToSlot`,
  `GridWorldCorners`. Ficam `SlotCenterToLocal`, `LocalToSlot`, `GridLocalBounds`, `SlotInGrid`.
  Os testes de round-trip viram `SlotCenterToLocal` → `LocalToSlot`.
- `internal/app/game/room.go`: `slotPayloadToWorld` vira `slotPayloadToLocal` (usa
  `SlotCenterToLocal`); `gatePieceMoveLocked` usa `SlotCenterToLocal`. Atualizar os comentários
  que dizem "world" (ex.: `PlayerPiecePositions`).
- `internal/domain/match/service/visibility.go`: `BoundaryLOSWalls` usa o retângulo de
  `GridLocalBounds` (TL→TR→BR→BL). Comentários "World coords" → LOCAL.
- `internal/domain/match/matchsession/match_session.go` `RecomputeVisibility`: `maxRadius` =
  diagonal de `GridLocalBounds` × 1.2 (hoje `hypot(Cols·CellSize, Rows·CellSize)`, que para hex
  na escala nova fica MENOR que o tabuleiro e cortaria o polígono antes da borda).
- Testes:
  - fog numa grade **girada** (ex.: rotation 30, skewRatio 0.6) com parede LOCAL entre duas
    peças: a peça de trás fica fora do polígono (`FilterMapState` não a mostra). Hoje falha,
    porque a origem vinha em WORLD.
  - fog numa grade **hex**: polígono de um tabuleiro sem paredes alcança os quatro cantos de
    `GridLocalBounds` (pega o `maxRadius`).
  - `BoundaryLOSWalls` hex/girado = retângulo LOCAL.
- `go vet ./...`, `go vet -tags smoke ./...`, `go vet -tags integration ./...`, `go test ./...`,
  `golangci-lint run` antes de entregar.

### B3. Docs
- `docs/dev/api/maps.md`: `visiblePolygons` (as duas tabelas, ~l.424 e ~l.461) — "coordenadas
  LOCAIS da grade (pré-transform, o mesmo espaço de `p1/p2` das paredes); o cliente aplica o
  transform da grade para desenhar". Corrigir também a linha ~315 (`p1`, `p2`: hoje diz "mundo
  (pré-transform)", que é contraditório → "LOCAL da grade (pré-transform)").
- `docs/dev/fog-of-war-postmortem.md` "Divergência conhecida": resolvida — o servidor não
  transforma mais nada.
- `docs/dev/match/handoff/coordenadas-grade.md`: apagar (o handoff foi cumprido) e tirar de
  `docs/documentation-map.yaml` se estiver lá.

## Task 2: Front — `System_X_System_React/` (TDD)

### F1. Polígonos LOCAL → WORLD
- `src/features/tactical-map/utils/coords.ts`: `polygonsToWorld(polys: VisibilityPolygon[], grid)`
  — aplica `applyTransform` em cada vértice. Teste: grade girada 90° → vértice transformado
  bate com `applyTransform`; identidade com rotation 0/skew 1/sem origem.
- `src/features/tactical-map/stage/ViewportInner.tsx`: `useMemo` dos polígonos em WORLD a partir
  de `fog.visiblePolygons` e `map.grid`; passar o resultado ao `FogLayer` (via `fog`) e ao
  `WallsLayer.losPolygons`. Um ponto só de conversão.
- Comentários a corrigir: `fogDraw.ts` (`drawLosMask`: "The points are written verbatim: the
  backend already produces them in world space…" → os pontos chegam já em WORLD, convertidos
  pelo chamador a partir do LOCAL do servidor) e `LosSplit.tsx` (prop `polygons`).
- `src/features/tactical-map/utils/__tests__/fogDraw.test.ts` usa uma fixture real
  (`realFogPayload.json`) com grade identidade — continua válida; não reescrever a fixture.

### F2. O retângulo de fog cobre a grade real
- `FogLayer` recebe `worldWidth/worldHeight` = `cols·cellSize`/`rows·cellSize`
  (`ViewportInner.tsx` ~l.420). Em hex (largura ≈ cols·cellSize·√3) ou girado, a grade passa
  dessa caixa; o `FOG_PADDING` (2000) só esconde isso em mapa pequeno — num hex 55×88.5 a
  borda direita fica sem fog. Trocar por uma caixa calculada dos 4 cantos de `gridLocalBounds`
  transformados (já feito do mesmo jeito em `ViewportInner.tsx` ~l.183) — mudar `drawFog` para
  receber `{minX, minY, maxX, maxY}` e testar.

### F3. Verificação
- `npm run typecheck` (ou `tsc -b`), `npm run lint`, `npm test`.

## Verificação ponta a ponta (sessão principal, não é task de subagente)
Com `go run ./cmd/devseed`: grade girada/achatada e grade hex, jogador com parede na frente —
fog e peças escondidas coincidem com o desenho; mestre move peça atrás da parede e o jogador não
a vê. Ver `docs/dev/seed.md`.
