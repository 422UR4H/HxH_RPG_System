# Plano — consistência do mapa: tabuleiro da partida, fundo, paredes, peças e entrada no mapa

Pedido do dono do produto (2026-10-12), depois de liberar rotação/perspectiva da grade:
1. paredes são subordinadas à grade: mudar o tamanho da célula escala as paredes junto;
2. "Remover" a imagem de fundo tem que persistir;
3. a partida tem o próprio mapa (`match_boards`): editar o mapa da campanha não pode mudar a
   partida — conferir e corrigir o que ainda age como antes;
4. o mapa precisa ser consistente na tela: peças cabem na casa achatada, clique/encaixe de parede
   medem na tela, Shift+alça funciona em qualquer rotação, setas seguem a direção da tela.

Repos e branch (`fix/grid-coord-spaces` nos dois; trabalhe direto nos checkouts, sem worktree,
sem trocar de branch; commits só com `git add` dos seus arquivos + `git commit`; nada de
`stash`/`checkout`/`reset`/`restore`/`push`; nunca adicione `.superpowers/`):
- Back: `/home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System`
- Front: `/home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System_React`

## Global Constraints

- **Wire camelCase** dos dois lados. O contrato é `System_X_System/docs/dev/api/*.md`; mudou o
  payload, muda o doc no mesmo commit.
- **Espaço LOCAL** = o de `WallSegment.p1/p2` (antes de rotação/achatamento/origem). Regras de jogo
  medem em LOCAL; o que é **percepção na tela** (raio de ficha, tolerância de clique/encaixe,
  direção das setas) mede em WORLD. O transform é `applyTransform` (front, `utils/coords.ts`):
  tira o pivô (`gridPivot`), gira, achata y por `skewRatio`, devolve o pivô, soma a origem.
- Grade sem transform (rotação 0, achatamento 1): comportamento idêntico ao atual; testes
  existentes passam sem mudança, salvo onde a task diz.
- Back: `testing` padrão, table-driven; handlers com `humatest`; ao fim de cada task `go vet ./...`
  e `go test` dos pacotes tocados. Front: vitest; `verbatimModuleSyntax`; ao fim de cada task
  `npx tsc -b` e `npx vitest run` dos diretórios tocados.
- Tolerância em teste: 1e-6. Comentários no idioma do arquivo; texto de interface em PT-BR.

## Task 1 (back): `PUT /maps/{id}` com `"bg": null` remove o fundo

Hoje `UpdateMapRequest.Body.Bg` é `*BgImageRequest`: `null` e campo omitido viram `nil`, e o caso
de uso trata `nil` como "manter" — o doc do campo promete "send null to clear", mas nada limpa. O
editor do front manda `"bg": null` ao remover a imagem (o mapa salva inteiro).
- Handler (`internal/app/api/map/update_map.go`): distinguir `"bg": null` de `bg` ausente. Jeito
  sugerido: `UnmarshalJSON` no struct do body (decodifica normal via alias e, num
  `map[string]json.RawMessage`, vê se a chave `bg` veio com `null`), marcando um campo
  `ClearBg bool \`json:"-"\``. Confira com `humatest` que o huma v2.30 usa esse `UnmarshalJSON`
  e que o schema/validação aceitam `null`.
- Caso de uso (`internal/application/map/update_map.go`): `ClearBg` → `m.Bg = nil`; `Bg != nil` →
  troca; nenhum dos dois → mantém.
- Testes: handler (humatest) — `bg: null` limpa, `bg` ausente mantém, `bg` objeto troca; caso de
  uso idem.
- Doc `docs/dev/api/maps.md`, tabela de campos do PUT: `bg` — omitir mantém, `null` remove,
  objeto troca.

## Task 2 (back): o tabuleiro da partida congela o fundo também

`LoadMatchBoardUC` (`internal/application/matchboard/load_match_board.go`) monta o retrato
fresco com `Bg: nil` ("herda o fundo do mapa no momento em que é lido"), e uma linha salva com
`bg` NULL volta sem o fundo. A tela da mesa então desenha o fundo **atual** do mapa da campanha:
se o mestre realinha ou troca a imagem do mapa, a partida em andamento muda junto — contra o
"é um retrato" do próprio doc, e desalinhando o fundo da grade do tabuleiro.
- Retrato fresco: `Bg` = cópia do `Bg` do mapa (cópia de valor, não ponteiro compartilhado).
- Linha salva com `Bg` nil (legado): resolver com o fundo atual do mapa anexado na leitura (o
  próximo salvamento do tabuleiro grava esse valor e a linha fica congelada). Linha com `Bg`
  próprio: devolve como está, sem ler o mapa.
- Testes do caso de uso: retrato fresco copia o fundo (e mudar o mapa depois não muda o devolvido);
  linha com `Bg` nil devolve o fundo do mapa; linha com `Bg` próprio devolve o próprio; mapa sem
  fundo → nil.
- Doc `docs/dev/api/match-maps.md`, "O tabuleiro da partida": `bg` é parte do retrato (copiado no
  retrato fresco); linhas antigas com NULL herdam na leitura e congelam no próximo salvamento;
  editar o mapa da campanha não muda nada na partida, nem grade nem fundo. Ajuste as frases que
  dizem o contrário (inclusive "bg nulo herda… o editor de mapa da partida (trabalho futuro) é o
  único que escreveria um valor aqui").

## Task 3 (back): `map_full_state` leva a grade e o fundo do tabuleiro

A tela da mesa pega grade e fundo pelo REST do **mapa da campanha** (`GET /matches/{id}/map` →
`mapUuid` → `GET /maps/{id}`), enquanto o servidor mede tudo com a grade do **tabuleiro**. Mudou o
mapa da campanha depois do início (tamanho da célula, colunas, linhas, tipo, fundo), a mesa
desenha uma grade e o servidor usa outra: fog, peças e paredes desalinham.
- `MapFullStatePayload` (`internal/app/game/message.go`) ganha `grid` (`*GridShapePayload`,
  ausente só quando a sala não tem tabuleiro — `CellSize` 0) e `bg` (`*BgImagePayload`, `null`
  quando o tabuleiro não tem fundo). Crie `BgImagePayload` camelCase espelhando
  `BgImageResponse` do REST (`url, x, y, width, height, rotation, opacity`).
- `Room.buildMapFullState` preenche os dois com `r.grid`/`r.bg` (o que a sala carregou do
  tabuleiro) para mestre e jogador. Lobby também manda (o front do lobby ignora).
- Testes na sala: `map_full_state` do jogador e do mestre trazem a grade da sala (não a do mapa) e
  o fundo do tabuleiro; sem fundo → `bg` null.
- Doc `docs/dev/api/maps.md`, tabela do `map_full_state`: `grid` e `bg` — "a grade e o fundo do
  **tabuleiro da partida**; a mesa desenha com estes, não com o mapa da campanha".

## Task 4 (front): a mesa desenha com a grade e o fundo do tabuleiro

- `src/hooks/useMatchWs.ts`: o parse de `map_full_state` repassa `grid` e `bg` (bg `null` vira
  `null`; ausente vira `undefined`).
- `src/features/match/combat/useLiveMapSync.ts`: guarda `boardGrid`/`boardBg` do último
  `map_full_state`.
- `src/features/match/combat/useGameTable.ts`: o `map` que a mesa usa vira o mapa do REST com
  `grid` e `bg` trocados pelos do tabuleiro quando já chegaram (`boardGrid ?? map.grid`;
  `boardBg !== undefined ? boardBg : map.bg`). Todos os consumidores (`GamePlayerPage`,
  `GameMasterPage`, compositor, âncoras, `gridKind`) passam a ler esse `map`, sem outra fonte.
  Grep por `map?.grid`/`map.grid` nas páginas da mesa e confirme que vêm de `game.map`.
- Testes: parse do `map_full_state` com e sem `grid`/`bg`; `useGameTable` (ou o helper que você
  extrair para montar o mapa da mesa) prefere a grade/fundo do tabuleiro ao REST.
- O lobby (`LobbyPage`/`useLobbyWs`) fica como está.

## Task 5 (front): paredes acompanham o tamanho da célula

Paredes estão em pixels LOCAIS; células também, mas escalam com `cellSize` em torno do (0,0) LOCAL
(quadrado: cantos em `col·cs`; hex: centros em `cs·f(q,r)`). Hoje mudar o tamanho (campo Tamanho,
alças de canto/borda, "Encaixar Grade", adicionar imagem) deixa as paredes na escala antiga.
- Helper puro (em `utils/walls.ts`): `scaleWallsToGrid(walls, prev, next)` — se `next.kind !==
  prev.kind` ou `cellSize` igual, devolve `walls` como estão; senão multiplica `p1`/`p2` por
  `k = next.cellSize / prev.cellSize`.
- `editorStore.ts`: `setGrid` e `setBgWithGrid` aplicam o helper com a grade anterior, no mesmo
  `set` (um passo só de desfazer).
- Testes: helper (k 1,25; tipo diferente; cols/rows diferentes mas mesmo cellSize → iguais);
  store: `setGrid` com cellSize 64→80 escala as paredes; desfazer volta as duas coisas; uma
  parede sobre a aresta entre (2,0) e (3,0) continua na aresta depois do resize (quadrado e hex).

## Task 6 (front): clique e encaixe de parede medem na tela

`WallsLayer` converte o ponteiro para LOCAL (`inverseTransform`) e compara com tolerâncias em
pixels de tela divididas pelo zoom (`SNAP_THRESHOLD_SCREEN / vpScale`, `HIT / vpScale`). Numa grade
achatada, a distância LOCAL no eixo achatado é maior que a da tela (÷ `skewRatio`): com
achatamento 0,3, o encaixe de 15 px vira 4,5 px e o clique de 8 px vira 2,4 px nesse eixo.
- Helpers puros: `snapWallPointWorld(world, grid, thresholdWorld)` (candidatos de
  `collectGridSnapPoints`, comparados depois de `applyTransform`; devolve o ponto LOCAL) e
  `findNearestWallWorld(world, walls, grid, hitWorld)` (distância ponto–segmento com as pontas
  transformadas). Sem transform, idênticos aos atuais.
- `WallsLayer` usa os novos em todos os pontos de encaixe/seleção/toque (desenho, seleção no
  editor, `hitWallAt` do viewer). Pode memoizar os candidatos transformados por grade.
- Testes: grade 10×10 cs40 achatada 0,3: ponteiro a 10 px de tela (WORLD) acima de uma parede
  horizontal acerta com tolerância 12; a 14 px não; um ponto de encaixe a 10 px de tela no eixo
  achatado encaixa com tolerância 15. Grade sem transform: mesmos resultados das funções antigas.

## Task 7 (front): fichas cabem na casa como ela aparece na tela

As fichas (tokens das peças) são círculos de raio `slotInradius(grid)·0,9` — o raio LOCAL. Numa
grade isométrica (45°, 0,5) a casa vira um losango com metade da altura, e a ficha passa por cima
das casas vizinhas. Decisão: a ficha continua redonda (achatar distorce o avatar) e passa a caber
na casa desenhada.
- `export function slotScreenInradius(grid)` em `coords.ts`: a menor distância do centro da casa
  (0,0) transformada até cada aresta transformada (`slotCorners` da casa (0,0) e
  `slotToWorld`). Sem transform: igual a `slotInradius` (quadrado cs/2, hex cs·√3/2). Quadrado
  cs40 a 45°/0,5: 12,649110640673518 (= 0,5·1600 / (2·√1000)).
- Trocar `slotInradius` por `slotScreenInradius` onde o raio é de desenho/toque na tela:
  `PieceSprite` (raio da ficha e hitArea), `PiecesLayer` (anel de segurar), `stackOffsetToPx`,
  `screenAnchor`, `intentGeometry` (`tokenRadius`). `slotInradius` fica para quem mede em LOCAL
  (se não sobrar ninguém, mantenha-o exportado e usado pelo novo, sem apagar).
- Testes: `slotScreenInradius` (sem transform quadrado/hex; 45°/0,5; rotação 90 sem achatamento =
  cs/2); `stackOffsetToPx` numa grade iso usa o raio novo.

## Task 8 (front): Shift + alça achata em qualquer rotação

A perspectiva por Shift só vale nas alças TC/BC. Perto de 90° de rotação elas ficam na horizontal
da tela: em 90° nada acontece (`ry` 0 → `null`) e perto disso um pixel muda muito.
- `gridFromHandleDrag`, ramo Shift: aceita TC/BC **ou** ML/MR — o par cujo `|ry|` é maior (o par
  mais vertical na tela). O outro par, com Shift, faz o redimensionamento normal.
- `MapHandlesLayer`: a marca laranja de Shift (e o cursor) vai no par que de fato achata
  (calcule pelo mesmo critério; exporte o critério de `coords.ts`, ex.
  `skewHandlePair(grid): "TC/BC" | "ML/MR"`).
- Testes: rotação 90°: Shift+ML com cursor sobre a alça mantém o achatamento e, movendo, a alça
  segue o cursor em y; Shift+TC em 90° não achata (redimensiona ou `null` — documente qual);
  rotação 0: comportamento atual (TC/BC).

## Task 9 (front): setas do teclado andam na direção da tela

No editor, com uma peça selecionada, as setas mudam `col`/`row` (só grade quadrada). Numa grade
girada 45°, ↑ manda a peça para cima-direita na tela. Correção: a seta escolhe, entre os vizinhos
da casa (8 no quadrado, 6 no hex — `neighborSlots` em `utils/reach.ts`), o que fica mais na
direção da seta **na tela** (maior cosseno entre `slotToWorld(vizinho) − slotToWorld(casa)` e a
direção da seta, y para baixo); empate → o primeiro de `neighborSlots`. Libera as setas no hex.
- Helper puro `neighborTowardScreen(slot, grid, dir: "up"|"down"|"left"|"right")`.
- `TacticalMapEditor` usa o helper (mantém os checks de dentro da grade e de casa ocupada).
- Testes: quadrado sem transform (↑ → (col,row−1), → → (col+1,row)); quadrado 45°/0,5
  (↑ → (col−1,row−1), → → (col+1,row−1)… confira as quatro pela conta); hex sem transform
  (← → (q−1,r)).

## Task 10 (docs, front)

- `docs/dev/tactical-map/coordinates.md`: o que mede em LOCAL e o que mede na tela (fichas,
  tolerâncias, setas); paredes escalam com o tamanho da célula.
- `docs/dev/match/` ou `docs/dev/tactical-map/overview.md` (onde já se fala da mesa): a mesa
  desenha com a grade e o fundo do tabuleiro da partida (`map_full_state`), não com o mapa da
  campanha. Procure frases que digam o contrário e corrija.

## Fora do escopo

- Editor de mapa da partida (escrever no `match_boards` pelo front).
- Aviso no editor de que o mapa está em uso por partidas iniciadas.

## Verificação final (controlador)

- Back: `go vet ./...` (+ `-tags smoke`, `-tags integration`), `go test ./...`,
  `golangci-lint run --max-same-issues 0 --max-issues-per-linter 0`.
- Front: `npx tsc -b`, `npm run lint`, `npm run test`.
- Browser (mesa `devseed -suffix i`, isométrica): editar o mapa da campanha (tamanho da célula,
  fundo) com a partida iniciada não muda a mesa; remover o fundo persiste; paredes escalam com o
  tamanho; fichas cabem nas casas; clique em parede e setas na grade isométrica.
