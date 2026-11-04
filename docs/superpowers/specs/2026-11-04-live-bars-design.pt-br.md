# Barras ao vivo, projetadas por destinatário — design

> Os dois repos. Back: `bars_updated` / `match_full_state.bars` (`internal/app/game/`,
> `internal/domain/match/`). Front: `src/features/match/combat/` (`GeneralBar`,
> `CharacterBarsStrip`, `OwnBars`, `barsMath`). Branch `fix/live-bars` nos dois.

## Problema

As barras do topo (a tira abaixo da barra geral) não mostram o personagem quando ele envia a
ação — só depois que o mestre a abre — e o "saldo" que mostram fica congelado o round inteiro.
Ao mesmo tempo, a barra geral vaza segredo para os jogadores: toda ação pendente de todo
personagem, com um número que revela a velocidade rolada.

## Decisões (dono do produto, 2026-11-04)

| # | Decisão |
|---|---|
| D1 | **Ação pendente é segredo do mestre.** O mestre vê cada envio na hora — os dele (NPCs) e os dos jogadores. O jogador vê só os cards do que ELE enviou e o que o mestre já abriu. Ação pendente de outro personagem é invisível ao jogador até abrir. |
| D2 | **Nenhum dado rolado chega ao jogador antes de o mestre abrir a ação — nem o da própria.** Senão ele cancelaria a ação que rolou mal ("anti game"). Vale para velocidade, acerto, dano, qualquer teste. A ação que o jogador enviou continua aparecendo como hoje: o card, sem número. |
| D3 | A barra mostra o **saldo ao vivo** (regime disputado): `carry + média(velocidades) − n × preço`, mudando a cada alteração. Para o mestre inclui as pendentes que abrem neste round; para o jogador, só o que já é público. |
| D4 | Antes de a primeira abertura congelar o preço, o mestre vê um **preço provisório** — a menor velocidade pendente, que é exatamente a que vai congelar — marcado como provisório. |
| D5 | No regime **livre** a tira de barras fica escondida (lá não há preço nem média). |
| D6 | O back mantém a conta fracionária exata (ela ordena a fila). Arredondar é só exibição, uma casa, como o resto da tela já faz. |
| D7 | **O preço congelado continua público**, como hoje: o jogador o vê quando o mestre abre a primeira ação do round. Aceito sabendo: numa mesa pequena o jogador às vezes deduz que o preço é a velocidade da própria ação pendente (ou que ela é pelo menos o preço). O jogador vê a própria velocidade de ação e de movimento só quando aquela ação é aberta. |

D1 corrige `docs/game/combate/barra-de-acao.md` ("Enxergando a barra"), que dizia que todos veem a
barra geral com a ordem de todo mundo. O texto é reescrito conforme D1.

## Achados (o que está errado hoje)

| # | Achado | Onde |
|---|---|---|
| F1 | A velocidade só é gravada quando o mestre abre a ação (`recordActed`), e o preço só congela na primeira abertura. Até lá a linha não tem velocidade nem preço, e a tira a esconde (`hasBarsData`). Nada aparece no envio. | `match_session.go`, `barsMath.ts` |
| F2 | `actionBalance`/`moveBalance` no wire é o **carry** do round anterior (`ResourceBar.Balance` só muda em `settleBars`). A barra e o painel "Saldo" do jogador não se mexem durante o round. | `room.go newBarsUpdatedPayload`, `CharacterBarsStrip`, `OwnBars` |
| F3 | `bars_updated` vai igual para todos, inclusive `order` — ator, barra e chave de toda ação pendente. Antes de o preço congelar a chave é `carry + velocidade`, e o `carry` é público: a velocidade rolada fica legível para todos. Viola D1/D2. | `broadcastBars` |
| F4 | A velocidade de uma reação entra na barra quando ela é **anexada** (`chargeReactionBars` dentro de `AttachReaction`), não quando é aberta; o próximo `bars_updated` a publica em `actionSpeeds`/`moveSpeeds` para todos. Viola D2. | `match_session.go` ~L1334 |
| F5 | Antes da primeira abertura, `ProjectOrder` pontua barra não congelada sem preço e sem portão, então a ordem publicada difere da jogada — exatamente o caso que o próprio comentário dele descreve (p2 23 e 17, p1 20, p3 11 publica p2 → p1 → p2 → p3; a mesa joga p2 → p1 → p3 → p2). | `round_scheduler.go keyOnBar` |
| F6 | O chip da fila imprime a chave crua: medido `-5.333333333333332`, `-42.333333333333336`, `-65.47619047619048` (a média divide pelo número de ações; o carry fracionário atravessa rounds; mais ruído de ponto flutuante). | `GeneralBar.tsx` `<Key>` |
| F7 | O `x̄` da tira faz a média de ação e movimento juntos, e a lista de velocidades não diz de qual barra é cada número. Cada barra tem a sua velocidade. | `CharacterBarsStrip.tsx` |
| F8 | `characters` sai da iteração de um mapa do Go (`CharacterIDs`), então a ordem das linhas muda aleatoriamente entre atualizações e as linhas pulam. | `match_session.go CharacterIDs` |
| F9 | No livre nada é gravado, então a tira nunca mostra nada lá, ao contrário do próprio comentário. Resolvido por D5. | `CharacterBarsStrip.tsx` |

## Design

### Back

1. **Preço provisório.** Extrair de `FreezePrices` a conta "menor velocidade pendente por barra
   não congelada" (só disputado) para uma função só de leitura. `FreezePrices` a usa para
   congelar; a projeção a usa como preço provisório.
2. **A projeção usa o preço provisório (F5).** Num round disputado, `ProjectOrder` pontua a
   barra não congelada contra o preço provisório — chave E portão — para a ordem publicada ser a
   que o mestre vai jogar se nada mais chegar. `SelectNext` não muda (roda depois de
   `FreezePrices`). Livre fica como está.
3. **O que a projeção reporta.** Para cada ação pendente a simulação já decide se ela abre neste
   round. Ela também precisa expor, por personagem e barra, as velocidades das ações que abre (na
   ordem em que abre) e das que deixa para o próximo round, para o payload ser montado sem um
   segundo caminho de pontuação.
4. **Saldo ao vivo.** `BarEconomy.Balance(carry, velocidades, preço)`, sem teto (o teto só vale no
   fechamento do round, `CloseBalance`). Sem preço naquela barra (congelado ou provisório) o saldo
   ao vivo é o carry. Exemplo da regra: p2 sozinho com 23, preço 11 → +12; com a segunda ação
   em 17 → −2.
5. **Velocidade pública × secreta (F4).** Uma velocidade fica pública quando a ação ou reação
   dela é aberta. O estado da barra precisa separar a parte pública do resto para a visão do
   jogador — hoje `ResourceBar.Speeds` é um `[]int` sem identidade. Como rastrear é escolha de
   quem implementa (ex.: gravar a velocidade da reação com marca de "ainda não pública" e virar no
   `open_reaction`, ou guardar as velocidades de reação à parte). A ECONOMIA continua usando toda
   velocidade, exatamente como hoje; só a projeção para o jogador filtra.
6. **Projeção por destinatário (F3).** `bars_updated` sai de `r.broadcast` para a pista por
   destinatário (`dispatchPerPlayer`), como `turn_opened`. `match_full_state.bars` monta a mesma
   visão para o seu destinatário. `seq` continua subindo uma vez por retrato sob `r.mu`, igual em
   todas as cópias. Visão do mestre × do jogador: ver a tabela do wire abaixo.
7. **Ordem determinística (F8).** `characters` ordenado por `characterId`.

### Wire (`bars_updated` / `match_full_state.bars`)

Por personagem:

| Campo | Mestre | Jogador |
|---|---|---|
| `characterId` | ✓ | ✓ |
| `actionCarry` / `moveCarry` — **novo**: o que atravessou do round anterior (crédito ou dívida) | ✓ | ✓ |
| `actionBalance` / `moveBalance` — **muda de sentido**: saldo ao vivo (D3), não o carry | abertas + pendentes que abrem neste round | só velocidades públicas |
| `actionSpeeds` / `moveSpeeds` — abertas neste round, em ordem | todas | só as públicas (F4) |
| `pendingActionSpeeds` / `pendingMoveSpeeds` — **novo**: na fila, abrem neste round, na ordem projetada | ✓ | ausente |
| `deferredActionSpeeds` / `deferredMoveSpeeds` — **novo**: na fila, não cabem neste round | ✓ | ausente |

Nível de cima:

| Campo | Mestre | Jogador |
|---|---|---|
| `seq`, `roundTurns` | ✓ | ✓ |
| `prices` — preços congelados (sem mudança) | ✓ | ✓ |
| `provisionalPrices` — **novo**: barras ainda não congeladas, só disputado | ✓ | ausente |
| `order` | ✓ (chaves agora com o preço provisório, F5) | `[]` |

Ação combinada continua cobrando as duas barras: a velocidade dela em cada barra aparece nas
listas daquela barra. Os campos novos nunca vão `null` no wire (`[]` vazio, ou omitidos quando
são só do mestre — quem implementa escolhe um e o contrato diz qual; o `normalizeBars` do front
continua se defendendo).

### Front

1. **Tipos e normalização** seguem a tabela do wire (`combatMessages.ts`, `normalizeWire.ts`).
2. **`GeneralBar`.** Chave do chip da fila com `fmt` (uma casa, F6). O trecho "Fila" só aparece
   para o mestre (o jogador recebe `order: []`; "Fila vazia" não deve aparecer para ele). Preços:
   congelados como hoje; provisórios com a marca "(provisório)". O botão "Barras" e a tira somem
   no livre (D5).
3. **`CharacterBarsStrip`.** A linha aparece quando o personagem tem alguma velocidade (aberta,
   pendente ou adiada) ou a barra tem preço — congelado, ou provisório para o mestre. A barra usa o
   `actionBalance`/`moveBalance` ao vivo contra o preço congelado ou provisório. Velocidades
   agrupadas por barra (⚔ … / ➜ …): abertas no estilo normal, pendentes mais claras, adiadas
   riscadas com a dica "próximo round". Um `x̄` por barra (F7), sobre as mesmas velocidades que
   compõem o saldo daquela barra. Linhas ordenadas pelo nome exibido (estável — F8).
4. **`OwnBars`** mostra o saldo ao vivo (mesmo campo, novo sentido).

### Docs

- `docs/dev/api/match-combat-ws.md`: `bars_updated` reescrito (destino: por destinatário; a tabela
  acima; nota do F5 sobre as chaves; sai a justificativa antiga de "broadcast, não projetado").
  `match_full_state.bars` aponta para ele.
- `docs/game/combate/barra-de-acao.md`, "Enxergando a barra": reescrito conforme D1/D2/D7 em
  linguagem de jogador (o mestre vê a fila; o jogador vê os próprios cards e o que foi aberto).
- `docs/dev/match/combat-engine.md`: preço provisório e barras por destinatário, onde descreve a
  barra geral.
- Front: o dev doc de combate que descreve a barra geral (`docs/dev/match/`).

## Fora do escopo

- Nenhuma mudança na economia das barras (média, preço, portões, carry, teto).
- Nenhum arredondamento no back.
- Nenhuma economia para o regime livre.
- Nenhuma mudança no que `turn_opened`, `action_queued`, `ownQueue` ou `resolution_updated` revelam.

## Verificação

- Back: testes de unidade do preço provisório, da projeção com preço provisório (o exemplo
  canônico p1/p2/p3 tem que publicar p2 → p1 → p3 → p2 antes de qualquer abertura), do saldo ao
  vivo, da separação pública/secreta das velocidades de reação e do payload por destinatário (o
  mestre recebe pendentes/adiadas/provisório/ordem; o jogador nada disso e nenhuma velocidade de
  reação não aberta). Teste no nível da `Room`: o `bars_updated` de um jogador depois do
  `enqueue_action` de outro não carrega rastro dele.
- Front: testes de componente da tira (estilos de pendente/adiada, média por barra, marca de
  provisório, escondida no livre) e da `GeneralBar` (chave formatada, sem "Fila" para o jogador).
- Browser: `go run ./cmd/devseed`, mestre e jogador lado a lado (`docs/dev/seed.md`): o jogador
  envia → a tira do mestre mostra na hora a linha com a velocidade pendente e o preço provisório;
  o topo do jogador não mostra nada novo. O mestre abre → o topo do jogador mostra a linha.

## Resolvido depois da revisão (decisões do autor, 2026-11-04)

- **Quando a velocidade de uma reação fica pública.** Quando a mesa conhece a reação: no
  `reaction_opened`, ou no fechamento do turno (a resolução assentada vai à mesa) para uma reação
  nunca aberta. Reação gratuita (`nothing`, a `default` sentada) não grava velocidade, então não
  importa. A reação do próprio jogador segue D2, como a ação dele na fila.
- **O saldo do jogador pode pular.** Velocidade escondida fica fora do número do jogador até ficar
  pública; o pulo quando fica é consequência esperada de D2, não bug.
- **Pendente × adiada.** Toda ação da fila que a projeção abre é pendente; o que sobra quando ela
  para é adiado. Ação combinada passa nos dois portões ou em nenhum (`keyOf`), então é pendente
  nas duas barras ou adiada nas duas.
- **Barra sem preço** (nem congelado, nem provisório) não mostra o medidor, só as velocidades se
  houver — como a tira já faz hoje. A linha aparece mesmo assim quando a outra barra tem preço ou
  existe alguma velocidade.
- **Regime livre.** O back manda o payload como no disputado (preços vazios, sem provisório); o
  front esconde a tira. D1 vale no livre também: o jogador recebe `order: []`.
- **Destinatários.** `dispatchPerPlayer` conhece dois tipos: o mestre e um jogador. O jogador vê
  todo personagem — o próprio incluso — pela mesma visão de jogador (D2); ter vários personagens
  não muda nada.

## A cargo de quem implementa

- Como o estado da barra marca uma velocidade como pública (Design 5).
- Se os campos só do mestre são omitidos ou vazios para o jogador (dizer no contrato).
- O visual exato das velocidades pendentes/adiadas e da marca de provisório, dentro dos tokens
  existentes.
