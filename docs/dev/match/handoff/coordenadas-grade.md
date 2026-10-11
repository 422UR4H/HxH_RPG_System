# Handoff — espaços de coordenadas da grade (back × front)

Estado em 2026-10-11. Corrigido em `feat/wall-lock-reach`: só o `move_blocked` (centros no espaço
LOCAL). O resto abaixo continua divergente.

## Os espaços

- **Local** (da grade, antes do transform): onde vivem `WallSegment.P1/P2` (o editor desenha com
  `inverseTransform`; `WallsLayer` aplica `applyTransform` para desenhar).
- **World** (depois do transform): onde fica o desenho na tela.
- Comparar parede com posição de peça deve ser feito **num espaço só**. O mais simples é o local.

## Divergências encontradas

1. **Hex: o Go usa metade do tamanho.**
   - Go `baseCenter` / `WorldToSlot` / `GridLocalBounds` (`internal/domain/map/service/coords.go`):
     `size = CellSize / 2`.
   - Front `hexToPixel(…, grid.cellSize)` (`src/features/tactical-map/utils/hex.ts`, usado por
     `coords.ts`, `walls.ts`, `GridLayer.tsx`): `size = cellSize`.
   - Efeito: em mapa hex, todo cálculo do servidor que converte slot ↔ ponto (move_blocked,
     alcance da parede `WallContactSlots`, fog, `CellsInPolygon`, landing de fuga) está em outra
     escala que o desenho. Precisa decidir qual é o certo (o desenho do front é o que o usuário vê)
     e alinhar o Go — e checar os testes de hex do Go (`movement_geometry_e2e_test.go`,
     `escape_landing_test.go`, `coords_test.go`), que hoje validam o Go contra ele mesmo.
2. **Transform diferente** (já registrado em `docs/dev/fog-of-war-postmortem.md`, "Divergência
   conhecida"): o Go aplica squash antes da rotação, sem pivô e sem `originX/originY`; o front
   rotaciona em volta do centro da grade, depois squash, depois origem. Identidade com
   `rotation: 0` e `skewRatio: 1`, então nada quebra hoje.
3. **Fog mistura espaços**: `ToLOSWalls` (`internal/domain/match/service/visibility.go`) usa as
   paredes em LOCAL, mas as origens das peças vêm de `SlotCenterToWorld` (WORLD) e o contorno do
   tabuleiro de `GridWorldCorners` (WORLD). Em mapa girado/isométrico o fog sai fora de lugar.
   Também `gatePieceMoveLocked` e `slotPayloadToWorld` (`room.go`) leem WORLD contra polígonos
   de visibilidade.

## Recomendação

Fazer TODO o cálculo do servidor em espaço LOCAL (paredes, peças, polígonos de visibilidade,
contorno = `GridLocalBounds`), e o front aplicar o transform só para desenhar — inclusive os
polígonos de fog que chegam do servidor. Assim a divergência 2 deixa de importar para o
servidor. Depois alinhar o tamanho do hex (1). Exige olhar o front do fog (os polígonos hoje são
desenhados como WORLD) e o postmortem do fog antes de mexer. Complexidade: alta — toca fog,
movimento, fuga e alcance; vale esforço alto e testes com grade girada, achatada e hex.

O que já foi feito como modelo: `SlotCenterToLocal`/`LocalToSlot`/`SlotInGrid`/`WallContactSlots`
(`coords.go`, `wall_reach.go`) e o teste `TestE2E_MoveWallCheck_RotatedGridMeasuresInLocalSpace`.
