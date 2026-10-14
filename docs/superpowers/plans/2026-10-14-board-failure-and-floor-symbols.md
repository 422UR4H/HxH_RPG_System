# Plano — falha do tabuleiro, falha do fundo e símbolos de parede no chão (2026-10-14)

Continuação dos PRs #88 (back, `System_X_System/`) e #75 (front, `System_X_System_React/`), os
dois na branch `fix/grid-coord-spaces`. Tudo vai para esses PRs, direto nos checkouts
principais (não criar worktree, não trocar de branch). Sem spec à parte: os pedidos do dono do
produto estão citados em cada task e são a autoridade.

## Contexto (o que já existe)

- Mesa (tela da partida iniciada: `GameMasterPage`/`GamePlayerPage`) e lobby (`LobbyPage`)
  desenham **só o tabuleiro da partida**, que chega pelo WS em `map_full_state` (grade `grid`,
  fundo `bg`, peças, paredes). Do REST (`useMatchMap` → `useMap`) vem só o nome do mapa.
  `tableMap(restMap, boardGrid, boardBg)` (`src/features/match/combat/tableMap.ts`) monta o
  mapa da tela e devolve `undefined` até o tabuleiro chegar; enquanto isso as páginas mostram
  "Carregando o tabuleiro…".
- O servidor (`internal/app/game/room.go`) carrega o tabuleiro em `loadBoard`: no registro do
  **mestre** (`shouldLoadBoard := isMaster && (r.session == nil || !r.boardLoaded)`) e no
  `StartMatch`. Se `LoadBoardUC.Load` devolve erro, só loga: a sala fica sem tabuleiro, nenhum
  `map_full_state` é mandado, e a tela fica em "Carregando o tabuleiro…" para sempre.
- Jogador só entra na sala que o mestre abriu. No lobby, `lobby_not_open` manda o jogador de
  volta para a página da partida. Na mesa, o jogador fica com status `waiting` ("Aguardando o
  mestre abrir a sala…" na barra do topo), e o espaço do mapa mostra "Carregando o tabuleiro…",
  que é enganoso.
- O fundo é carregado em `src/features/tactical-map/stage/BgLayer.tsx` (`Assets.load`, ou
  `Image` para `blob:`). Uma falha hoje só não desenha o fundo, em silêncio. O resto do mapa já
  funciona sem ele.
- Paredes são guardadas em coordenadas LOCAIS da grade e desenhadas em WORLD com
  `applyTransform(p, grid)` (`src/features/tactical-map/utils/coords.ts`; a inversa é
  `inverseTransform`). Em `WallsLayer.tsx`, `drawWallTypeSymbol` desenha o arco da porta e os
  tracinhos da janela **em WORLD**: um círculo de tela e tracinhos perpendiculares na tela. Numa
  grade isométrica (rotação 45°, achatamento 0,5), eles ficam "de pé", fora do chão.

## Restrições globais

- Respostas, comentários e docs em PT-BR, identificadores em inglês; siga o estilo do arquivo.
- Back: `go vet ./...` e `go vet -tags integration ./...` e `go vet -tags smoke ./...`,
  `go test ./...` e `golangci-lint run` limpos antes de cada commit que toque `internal/`.
- Front: `npx tsc -b` limpo, `npx vitest run` todo verde, `npm run lint` sem erro novo (hoje
  há 2 erros antigos em `sheet`/`Diagram`, não mexer).
- `verbatimModuleSyntax`: import de tipo com `import type`.
- Não fazer push. Commits pequenos, com mensagem em PT-BR no estilo do `git log`.
- A camada Pixi não tem teste (o `@pixi/react` é mockado): toda geometria nova vai para uma
  função pura testável, e a camada só a chama.

---

## Task 1 (back) — `board_unavailable`: o servidor avisa quando o tabuleiro não carregou, e tenta de novo

**Pedido do dono:** se o mapa falhar, mostrar ao usuário que o carregamento falhou e que ele
pode atualizar a página para tentar de novo. Hoje o servidor engole o erro, e a tela espera para
sempre.

**Arquivos:** `internal/app/game/room.go` (struct `Room`, `loadBoard`, ramo `register` de
`Run`), um teste novo em `internal/app/game/` (ex.: `board_unavailable_test.go`, no padrão dos
testes de sala que já usam `RoomDeps.LoadBoardUC` com fake), docs (abaixo).

**Comportamento:**

1. A `Room` ganha `boardLoadFailed bool` (protegido por `r.mu`, documentado no struct como os
   vizinhos). `loadBoard` marca `true` quando `LoadBoardUC.Load` devolve erro, e `false` quando
   devolve sem erro (com tabuleiro ou `nil` = sem mapa anexado).
2. No registro, **qualquer** cliente (jogador também) dispara uma nova tentativa quando a sala
   **nunca instalou** um tabuleiro e a última carga falhou:
   `shouldLoadBoard := (isMaster && (r.session == nil || !r.boardLoaded)) || (r.boardLoadFailed && !r.boardLoaded)`.
   Um tabuleiro já instalado **nunca** é recarregado pela conexão de um jogador: isso jogaria
   fora movimentos de um turno aberto, que só são gravados no fechamento.
3. Depois da (possível) carga, se `r.boardLoadFailed && !r.boardLoaded` (lido sob `r.mu`), o
   cliente que está registrando recebe
   `NewErrorMessage("board_unavailable", "the match board could not be loaded; reconnect to try again")`,
   no ponto onde o `map_full_state` iria (logo depois do `sendRoomState`). O `match_full_state`
   continua sendo mandado como hoje.
4. Quando a nova tentativa disparada por um jogador dá certo, o repush que o ramo
   `shouldLoadBoard` já faz manda `map_full_state` aos outros conectados (o mestre que tinha
   recebido `board_unavailable` passa a ver o tabuleiro sem recarregar). Não mudar esse repush.
5. Uma carga que falha **com tabuleiro já instalado** (ex.: reconexão do mestre no lobby, ou o
   `StartMatch`) não manda erro a ninguém: a sala segue com o tabuleiro que tem.

**Testes** (fake de `ILoadMatchBoard` que falha N vezes e depois devolve um tabuleiro; contar
as chamadas):
- mestre conecta, carga falha: recebe `error` com `code` `board_unavailable` e nenhum
  `map_full_state`;
- jogador conecta nessa sala, a nova tentativa dá certo: o jogador recebe `map_full_state` e o
  mestre recebe o repush;
- jogador conecta, a nova tentativa falha de novo: o jogador recebe `board_unavailable`;
- sala com tabuleiro instalado: a conexão de um jogador **não** chama o `Load` (contagem igual),
  e nenhum `board_unavailable` é mandado, mesmo depois de uma recarga do mestre que falhou;
- `Load` devolvendo `(nil, nil)` (sem mapa): nenhum erro, e o jogador não dispara carga.

**Docs:**
- `docs/dev/api/match-combat-ws.md`: linha na tabela de mensagens do servidor e uma seção
  `board_unavailable` perto de `connection_replaced`. É um `error` que não responde a nada que o
  cliente enviou. Explicar quando vem, que a conexão continua aberta, que um `map_full_state`
  posterior o substitui, e o que o cliente faz: mostrar a falha e sugerir recarregar. Recarregar
  = reconectar, e a reconexão tenta de novo.
- `docs/dev/api/game-lobby.md`: o lobby recebe a mesma mensagem (link para a seção acima).
- `.github/instructions/game-server.instructions.md`, parágrafo "The board — the server owns
  it": uma frase sobre `boardLoadFailed` e a nova tentativa em qualquer conexão.
- Se `docs/documentation-map.yaml` já mapeia `internal/app/game/` para esses docs, nada a
  acrescentar; senão, acrescentar.

---

## Task 2 (front) — a tela do mapa diz o que está acontecendo, e o tabuleiro não depende do REST

**Pedido do dono:** não deixar o usuário preso em "Carregando…" sem explicação; avisar a
falha e sugerir atualizar a página; e não impedir o jogo por algo que não é mecânica.

**Arquivos:** `src/hooks/useMatchWs.ts`, `src/hooks/useLobbyWs.ts`,
`src/features/match/combat/tableMap.ts`, `src/features/match/combat/useGameTable.ts` (e
`useLiveMapSync` se o estado do tabuleiro vive lá), `src/pages/GamePlayerPage.tsx`,
`src/pages/GameMasterPage.tsx`, `src/pages/LobbyPage.tsx`, testes ao lado.

**Comportamento:**

1. **`board_unavailable`** (contrato da Task 1): nos dois hooks de WS, um `error` com esse
   `code` vira um callback novo (`onBoardUnavailable`). Ele **não** vai para o `onWsError` /
   toast, porque não é resposta a um envio. A tela guarda `boardUnavailable = true` e o limpa
   quando chega um `map_full_state`.
2. **O tabuleiro que chegou sempre desenha.** `tableMap` passa a devolver o mapa sempre que
   `boardGrid` existe, mesmo sem o mapa do REST (falhou, ou ainda está carregando). Os campos
   que não são geometria ganham valores neutros: `id` = `mapUuid` do `useMatchMap` se houver,
   senão `""`; `name` `""`; listas vazias; datas `""`. Conferir se algo na mesa/lobby lê
   `map.name`/`map.id` e se continua certo. "Há mapa anexado" passa a vir do `useMatchMap`
   (`mapUuid` presente), não do `useMap`. "Carregando mapa..." (o REST) só aparece enquanto o
   `useMatchMap` está pendente e o tabuleiro ainda não chegou.
3. **Mensagem no lugar do mapa**, enquanto há mapa anexado e o tabuleiro não chegou, por
   ordem de prioridade (uma função pura, testada, que as três páginas usam):
   - `boardUnavailable` → "Não foi possível carregar o tabuleiro da partida. Atualize a página
     para tentar de novo." (com o estilo de erro que já existir nos tokens, ex.
     `colors.dangerText`/equivalente; não inventar hex);
   - mesa com status `waiting` → "O tabuleiro aparece quando o mestre abrir a sala.";
   - sem conexão (mesa `disconnected`; lobby `disconnected`, `error`, `throttled`,
     `replaced`, `kicked`, `lobby_closed`) → "Sem conexão com a sala. O tabuleiro aparece
     quando a conexão voltar.";
   - senão (`connecting`/`connected`) → "Carregando o tabuleiro…".
4. Nada muda quando não há mapa anexado (a mensagem de "sem mapa" que já existe continua).

**Testes:** hooks (o `board_unavailable` chama o callback novo e não o `onWsError`;
`connection_replaced` continua igual); `tableMap` (desenha só com o tabuleiro; `undefined` sem
grade); a função de mensagem (cada ramo e a prioridade); páginas (ajustar as asserções atuais
de "Carregando o tabuleiro…" e cobrir `waiting` e `board_unavailable` na mesa do jogador e no
lobby; tabuleiro chega com o REST do mapa falhando → desenha).

---

## Task 3 (front) — fundo que falha: aviso discreto, botão de tentar de novo, e o resto do mapa normal

**Pedido do dono:** "não impedir o jogador de jogar por conta disso. basta não renderizar a
imagem de bg nesse caso e renderizar peças, paredes, etc. em última instância, a imagem de bg
é apenas experiência (não é mecânica)".

**Arquivos:** `src/features/tactical-map/stage/BgLayer.tsx`,
`src/features/tactical-map/stage/ViewportInner.tsx` (só repassa),
`src/features/tactical-map/TacticalMapStage.tsx`, testes.

**Comportamento:**

1. `BgLayer` ganha `onError?: () => void`, chamado quando a imagem não carrega (os dois
   caminhos: `Assets.load` e `Image`), e `retryKey?: number`, que entra nas dependências do
   efeito de carga para forçar uma nova tentativa.
2. A nova tentativa precisa **de fato** buscar a imagem de novo. Conferir em
   `node_modules/pixi.js` (8.x) se o `Assets`/`Loader` guarda a promessa rejeitada por URL. Se
   guardar, a nova tentativa não pode passar por ele: usar o caminho `Image`
   (`crossOrigin = "anonymous"` para URL http(s)) ou limpar o cache dessa URL do jeito que a
   versão instalada suporta. **Não** acrescentar query string à URL: ela pode ser assinada.
3. `TacticalMapStage` guarda `bgFailed`. Com `bgFailed`, mostra por cima do canvas, no topo, um
   aviso pequeno que **não bloqueia** o mapa: o contêiner com `pointer-events: none` e só o
   botão com `auto`. Texto: "Não foi possível carregar a imagem de fundo. O mapa funciona
   normalmente sem ela." Botão: "Tentar de novo", que limpa `bgFailed` e incrementa o
   `retryKey`. O overlay de "Carregando imagem..." continua durante a nova tentativa. Trocar a
   URL do fundo zera `bgFailed`. Vale para editor, lobby e mesa, porque os três usam o
   `TacticalMapStage`. Cores pelos tokens.
4. Peças, paredes, grade e fog continuam desenhados com o fundo falho. Hoje já continuam, então
   não regredir.

**Testes:** `BgLayer` com `Assets.load` mockado rejeitando chama `onError`, e um `retryKey`
novo dispara nova carga (mock de `pixi.js` só no arquivo de teste). `TacticalMapStage`: o aviso
aparece quando o fundo falha e some no "Tentar de novo". Se montar o stage inteiro for
desproporcional com o mock do Pixi, testar a parte DOM (aviso + botão) num componente pequeno
extraído, e registrar isso no relatório.

---

## Task 4 (front) — arco da porta e tracinhos da janela no chão da grade

**Pedido do dono:** "notei que as portas e janelas do mapa isométrico não acompanham o ângulo
correto que deveria acompanhar."

**Arquivos:** novo `src/features/tactical-map/utils/wallSymbols.ts` +
`src/features/tactical-map/utils/__tests__/wallSymbols.test.ts`;
`src/features/tactical-map/WallsLayer.tsx` (`drawWallTypeSymbol` passa a desenhar o que a
função pura devolve).

**Comportamento:** a geometria do símbolo é feita em LOCAL (o chão da grade) e projetada
ponto a ponto com `applyTransform`, como a própria parede:

- **Porta:** quarto de círculo em LOCAL centrado em `p1` (a dobradiça), do sentido da parede
  até +90° (mesmo sentido de hoje), amostrado em pontos suficientes para parecer liso (ex.: 16).
  O raio é uma fração da parede: `t = min(0.85, (20 / vpScale) / lenWorld)` do comprimento
  LOCAL. `lenWorld` é o comprimento da parede em WORLD. Assim, ao longo da parede, o tamanho na
  tela é o de hoje: 85% da parede, no máximo 20 px de tela.
- **Janela:** dois tracinhos em 1/3 e 2/3, perpendiculares à parede **em LOCAL**. O meio
  comprimento em LOCAL é `(5 / vpScale) * lenLocal / lenWorld`, o mesmo tamanho de tela de hoje
  medido na escala da parede.
- A função devolve polilinhas em WORLD (`{x,y}[][]`). O `WallsLayer` só faz
  `moveTo`/`lineTo`/`stroke` com o estilo e o alpha de hoje, mantendo o `moveTo` inicial que
  evita a ligação torta (comentário existente).
- Ficam como estão, de propósito: o ponto da tranca e o "×" da parede destruída são
  **marcadores** de tela, como a ficha redonda. A abertura da porta aberta e os tracejados já
  seguem a parede projetada.

**Testes:**
- grade sem rotação nem achatamento: os pontos do arco ficam a distância `r` de `a1` em WORLD,
  do ângulo da parede até +90°, ou seja, o desenho de hoje;
- grade isométrica (45°, 0,5): cada ponto do arco, levado de volta com `inverseTransform`, fica
  à mesma distância LOCAL de `p1` (está no chão). O arco em WORLD **não** é um círculo;
- janela isométrica: cada tracinho, em LOCAL, é perpendicular à parede (produto escalar ≈ 0) e
  cruza a parede em 1/3 e 2/3;
- parede degenerada (comprimento ~0) devolve nada.

---

## Task 5 (front) — desfazer do editor volta ao estado de ANTES da rajada, não ao penúltimo

**Achado no navegador (2026-10-14), dentro do pedido original do dono** ("se encontrar outros
bugs de mapa desse tipo, pode corrigir"): no editor, digitar "30" no campo Rotação (que estava
em 45) e clicar Desfazer volta para **3**, não para 45. Uma predefinição clicada logo antes
também some do histórico. O mesmo vale para sliders (opacidade, perspectiva).

**Causa:** `src/features/tactical-map/store/editorStore.ts` usa o `temporal` do zundo com
`handleSet: (handleSet) => debounce(handleSet, 400)`. O debounce de cauda chama `handleSet` com
os argumentos da **última** chamada da rajada. Por isso o `pastState` gravado é o estado de antes
da última mudança, e não o de antes da primeira. Tudo o que veio antes na rajada se perde.

**Arquivos:** `src/features/tactical-map/store/editorStore.ts` (o `handleSet`, e um helper
pequeno se ajudar, ex. no próprio arquivo ou em `utils/`), `useEditorHistory.ts` se o desfazer
precisar descarregar a rajada pendente, testes em `store/__tests__/`.

**Comportamento:**
1. Continua agrupando mudanças seguidas (mesmos 400 ms de cauda), mas o passo gravado leva o
   `pastState` da **primeira** chamada da rajada, com o estado atual da última. A assinatura do
   `handleSet` do zundo instalado: conferir em `node_modules/zundo`.
2. Desfazer ou refazer com uma rajada pendente (menos de 400 ms depois da última mudança)
   primeiro grava a rajada como um passo, depois desfaz. Hoje o passo pendente chegaria depois do
   desfazer e bagunçaria a pilha. Se isso exigir expor um `flush` do debounce, fazer do jeito
   mais simples.
3. O commit de gesto (`useGestureHistory`: pause, um passo no release) não muda. Conferir que
   uma rajada pendente quando o gesto começa não se mistura com ele: descarregar antes do
   `pause()` se preciso.

**Testes** (com `vi.useFakeTimers()`): duas mudanças de rotação em < 400 ms → um desfazer
volta ao valor original; duas mudanças separadas por > 400 ms → dois passos; mudança + desfazer
imediato (< 400 ms) → volta ao original e o refazer devolve a mudança; um gesto depois de uma
rajada pendente gera passos separados e corretos.

---

## Depois das tasks (controlador)

Revisão final do branch, verificação completa nos dois repos, teste no navegador: portas e
janelas isométricas; clique e encaixe de parede na grade achatada; Shift a 90°; aviso do fundo
com a imagem indisponível e "Tentar de novo"; mesa do jogador com o mestre fora; tabuleiro
indisponível (precisa de uma linha de `match_boards` corrompida: pedir ao dono para rodar o
comando). Atualizar as descrições dos PRs e fazer push.
