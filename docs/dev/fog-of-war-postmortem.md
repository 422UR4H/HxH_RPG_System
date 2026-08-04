# Post-mortem — fog de guerra não aparecia para o jogador

Registro do que realmente quebrava o fog na fase 10-D, e de onde os diagnósticos
erraram. Serve para não repetir o caminho.

## Sintoma

O jogador via o mapa inteiro coberto de fog, sem enxergar nem a própria peça. O mestre
via o mapa normalmente. Em algumas tentativas o cliente do mestre entrava em loop de
reconexão e a tela ficava em "Carregando imagem...".

## Causas reais (cinco, independentes)

| # | Onde | O que acontecia |
|---|------|-----------------|
| 1 | `room.go` — `PlayerPiecePositions` | Pegava `r.mu.RLock()` enquanto o chamador já segurava `r.mu.Lock()`. Em Go isso trava a goroutine para sempre. Afetava todo caminho que recalcula fog: `StartMatch`, `RehydrateSession`, `pushVisibilityUpdates`, `handlePieceMoved`. |
| 2 | `useMatchWs.ts` + `room.go` | O front mandava `map_state_sync` com `pieces: []` e o backend substituía o tabuleiro por vazio. Sem peças não há origem de linha de visão → zero polígonos → fog cobre tudo. |
| 3 | `room.go` — `map_state_sync` | Semeava o tabuleiro mas não recalculava a visibilidade. Como `buildMapFullState` serve o **cache**, quem sincronizasse depois do jogador conectar deixava o jogador com polígonos calculados sobre tabuleiro vazio. |
| 4 | `client.go` — `maxMessageSize` | Era 4096. O `map_state_sync` de um mapa real (35×35, 17 paredes, 5 peças) tem ~5.8 KB. O servidor fechava a conexão do mestre no meio do sync (`read limit exceeded`), o cliente reconectava e reenviava — o "loop infinito" do mestre. |
| 5 | `FogLayer.tsx` | **A causa que fazia o fog nunca funcionar, mesmo com os dados certos.** Ver abaixo. |

### A causa 5, em detalhe

O `FogLayer` desenhava a escuridão e depois "apagava" a área visível com
`blendMode="erase"` dentro de um `<pixiContainer isRenderGroup>`.

`isRenderGroup` **não** cria um alvo de render isolado. Apagar direto no framebuffer
principal não revela o mapa: remove o que está embaixo e expõe a cor de limpeza do
canvas, que é preta. A área "visível" saía preta — indistinguível de fog.

Isso explica dois relatos com uma causa só: com o polígono grande demais (ver abaixo), a
área preta vazava para fora da imagem de fundo e parecia um "borrão"; com o polígono
contido no tabuleiro, ficava parecendo "fog cobrindo tudo".

### Achado relacionado: polígono maior que o mapa

`ComputeVisibilityPolygon` só era limitado por `maxRadius` (diagonal × 1.2). Medido num
mapa 3360×3360, o polígono ocupava 7505×9734 e o conjunto explorado tinha 4275 células
num grid de 1225. Corrigido adicionando as bordas do tabuleiro como bloqueadores de
visão (`BoundaryLOSWalls`). Depois: polígono dentro de `[0..2400] × [0..3360]`,
586 células exploradas.

### Bug encontrado de brinde

O `-race` acusou `Room.Run` fechando `client.send` enquanto outra goroutine fazia
`send <- data`: `panic: send on closed channel`, que derruba o processo inteiro.
Passou a sinalizar por `done`, drenando o backlog antes de fechar.

## Onde os diagnósticos erraram

| Hipótese | Por que estava errada |
|----------|----------------------|
| "`map_state_sync` zerar as peças é o bug" | Era real, mas só uma das cinco causas. Corrigir só isso não muda nada na tela. |
| "`onBgLoadingChange` vs `handleBgLoadingChange` causa o 'Carregando imagem...'" | Errado. O "Carregando imagem..." vinha da árvore Pixi derrubada por um `TypeError`, não do callback. |
| "O backend não computa o fog" | Errado o tempo todo. O backend enviava `polys=1 pieces=3 walls=6` corretamente; o problema era a renderização. |
| "Reiniciar o servidor para aplicar mudanças" | Cada restart gerava conexões zumbi e `text file busy`, e foi o que tornou o deadlock visível. Instrumentar e reiniciar **uma vez** resolve mais rápido. |
| "Testes unitários verdes significam que funciona" | Os testes usavam 1 parede. A causa 4 só aparece com o mapa real (~5.8 KB) e a causa 5 só aparece na GPU. |

Padrão comum aos erros: concluir a partir de leitura de código em vez de medir. As três
causas mais difíceis (4, 5 e o tamanho do polígono) só apareceram quando o sistema real
foi medido com dados reais.

## O que passou a proteger

- `internal/app/game/fog_regression_test.go` — deadlock, semântica de `pieces`, recálculo
  após sync, polígono e explorado dentro do tabuleiro, `send` durante shutdown.
- `internal/app/game/fog_e2e_test.go` — servidor WS real com mestre e jogador reais, nas
  duas ordens de conexão, e board grande (guarda o `maxMessageSize`).
- `internal/app/game/fog_smoke_test.go` (tag `smoke`) — contra os servidores locais e uma
  partida real. Instruções de uso no cabeçalho do arquivo.
- `src/features/tactical-map/utils/__tests__/fogTiers.test.ts` — usa como fixture o
  payload real exportado pelo smoke (`SMOKE_DUMP`), não números inventados.

Cada correção foi validada por mutation test: o defeito foi reintroduzido e confirmou-se
que o teste correspondente quebra.

## Decisão de render em aberto

O fog passou a ser desenhado como regiões que **não se sobrepõem** (anel externo + uma
quad por célula, cada área pintada uma única vez). Sem blend mode, sem alvo de render.
O custo é que a borda da área iluminada fica alinhada à grade em vez de seguir o contorno
do polígono. A camada de "explorado" já era por célula, então fica coerente. A borda suave
é possível com `Graphics.cut()`, mas exige repensar o tom intermediário de explorado.

## Divergência conhecida (não corrigida)

`applyTransform` do Go e o do TypeScript não são equivalentes: o Go aplica o squash antes
da rotação e ignora `originX`/`originY`; o front rotaciona em torno do centro da grade e
depois aplica o squash. Com `rotation: 0` e `skewRatio: 1` ambos são identidade, então
nada quebra hoje. Em mapa rotacionado ou isométrico o fog sairia fora de lugar.
