# Movimentos empilhados, velocidade só de movimento e a tira de barras — desenho

**Data:** 2026-11-06 · **Branches:** `fix/live-bars` (PR #101 do back, PR #102 do front) · **Parte de:**
`2026-11-04-live-bars-design.md` · EN: `2026-11-06-stacked-moves-design.md`

## Por quê

O dono do produto jogou uma mesa e viu quatro problemas:

1. Um movimento puro mostrou a **Legerity** do personagem no card. Movimento não tem actionSpeed: só
   rola o moveSpeed (Accelerate no Dash, Brake no Shift — manter assim).
2. No **Livre** a fila é ordenada por `Action.Speed.Result` (Legerity passiva) para toda ação, inclusive
   movimento puro. Tem que ordenar pela mesma velocidade do combate.
3. **Movimento empilhado**: o segundo movimento sai da peça, não de onde o primeiro termina; e um
   segundo mais rápido abriu ANTES do primeiro (a peça pulou para o destino final e a seta do primeiro
   ficou apontando para o primeiro destino). O segundo movimento não pode acontecer antes do primeiro.
4. A tira de barras: cores erradas, barras que andam para o lado quando aparece uma velocidade, o
   `title` nativo como dica e o saldo ao vivo desenhado "antecipado".

## Decisões (dono do produto, 2026-11-06)

### D1 — Movimento não tem actionSpeed
A ação rola e deriva `speed` (Legerity) **só quando cobra a barra de ação**
(`slices.Contains(a.Bars(), action.BarAction)`): ataque, interação, finta, perícias, defesa, esquiva,
repelir, ou movimento+ataque combinados. O movimento puro (e a reação de fuga, cujas barras são só de
movimento) leva `speed.rollCheck` **vazio** (sem perícia, sem dado, resultado 0). Consequências:
- O mapper deixa de carimbar Legerity em ação que não cobra a barra de ação.
- `rollActionDice` não rola `Speed` para ela; `deriveSpeeds` não a deriva.
- A condição do mestre no campo `speed` dessa ação é recusada com o `ErrConditionTargetMissing` que já
  existe (como `feint` em ação sem finta).
- O front só mostra "Velocidade de ação" para ação cujo `bars` inclui `action`.

### D2 — O Livre ordena pela mesma velocidade do combate
O Livre continua sem preço, média e sobra. A chave da ordem é a **velocidade rolada nas barras da
ação, a menor quando são duas** — a regra do `reactionKey` (`min SpeedOn(bar)` sobre `Bars()`):
movimento puro pelo moveSpeed, ataque pelo actionSpeed, combinado pela menor das duas.
`PriorityQueue.ExtractMax` usa essa chave (empate: quem entrou antes, como hoje). A ordem da fila no
front (`queueOrder.ts`) tem que concordar.

### D3 — O movimento empilha no último movimento pendente do personagem
Quando o personagem enfileira uma ação com `move` e já tem na fila uma ação pendente com `move`, o
novo movimento fica **empilhado** na ÚLTIMA delas (o fim da pilha):
- `Move.StackedOn` = o id dessa ação (campo novo; `uuid.Nil` = não empilhado).
- `Move.From` = o `Move.Position` dessa ação (não o slot da peça). A checagem de parede no enfileirar
  parte dali. O movimento sem movimento pendente antes dele mantém a origem de hoje (a peça).
- Só movimento empilha (puro ou combinado). Ação sem movimento nunca empilha nem recebe empilhado.
- Wire: `move.stackedOn` (uuid, omitempty) em toda superfície que já leva o `move` da ação ao dono
  e ao mestre. O jogador nunca vê as pendentes de outro personagem (sem mudança).

### D4 — O empilhado nunca abre antes daquele em que está empilhado (e o atraso é o preço)
A ação cujo `StackedOn` **ainda está na fila** fica **travada**: não é candidata.
- `SelectNext`/`best`, `ProjectRound` (na simulação: travada enquanto a anterior não foi escolhida),
  `AnyEligible` e o `ExtractMax` do Livre pulam as travadas. Se a anterior fica para o próximo round,
  a empilhada vai junto.
- `pull_action` de ação travada é recusado: erro de domínio novo `ErrStackedOnPending`
  (`game_error`, mensagem `"action is stacked on a pending move"`), nada muda.
- **Nenhuma penalidade nova.** Quando o primeiro abre, vira velocidade que agiu e paga o preço; o
  segundo passa a ter a chave com esse preço já contado (`BarEconomy.Key`, a regra que toda segunda
  ação do round já segue). Exemplo: preço 9, movimentos 12 e depois 20 → abre o 12; o 20 fica com
  chave `(12+20)/2 − 9 = 7`.

### D5 — Cancelar um movimento cancela o que está empilhado nele
`cancel_action` de ação com empilhados cancela a pilha inteira depois dela (transitivamente), na ordem
da pilha. Cada cancelada passa pelo caminho de cancelar de hoje: sai da fila, edições descartadas,
gravada em `cancelled_actions`, um `action_cancelled` por ação (o mestre e quem pediu), depois um
`bars_updated`. A marca de Desvantagem por troca é a de hoje (uma marca, o mesmo personagem). O front
**avisa antes de confirmar**: o diálogo de cancelar lista os movimentos empilhados que vão junto e
pede a confirmação.

### D6 — A reação gasta o primeiro movimento da pilha
`BestPendingFor` (a ação que uma reação cobrada consome) só considera ação **não travada** — a cabeça
da pilha é "o momento prestes a ser gasto". (Reserva: sem candidata não travada naquela barra, qualquer
pendente nela, como hoje.)
Depois que a reação consome uma ação que tem empilhados:
- Se a reação **pousa onde o movimento consumido ia** (o `Move.Position` da reação igual ao da ação
  consumida — o mesmo teste do `sameMoveAsQueued`), os empilhados **sobrevivem**: o primeiro deles
  passa a apontar para o `StackedOn` da consumida (em geral `uuid.Nil`, virando a cabeça). Sem
  penalidade (a fuga manteve a intenção, como hoje).
- Senão, os empilhados são **cancelados** pelo caminho do D5 (sem penalidade além da Desvantagem de
  troca de hoje).
O diálogo de reação do front avisa, numa fuga, quando o destino dela não é o destino da cabeça e a
cabeça tem empilhados: esses movimentos serão cancelados.

### D7 — A tira de barras
- **Cores:** barra de ação = o vermelho de ataque desta tela (`mapAttackIntent`), de movimento = o
  azul de movimento (`mapMoveIntent`), por tokens semânticos (`src/styles/CLAUDE.md`). Trilho vazio = o
  mesmo tom com alfa baixo (a barra mantém a identidade vazia e fica discreta no painel escuro). Nada
  de verde.
- **O que a barra mostra (o dono escolheu o protótipo):** o **total** da PRÓXIMA ação —
  `total = sobra + média(agiram ++ [próxima])`, onde `próxima` é a primeira velocidade pendente (a
  lista `pending` do mestre já vem na ordem projetada, então `pending[0]`); para o jogador (sem
  pendentes) `total = sobra + média(agiram)`. Um tracinho em cada múltiplo do preço. O número ao lado
  da barra é esse total. A escala é comum por tipo de barra na tira inteira: largura cheia = o maior
  `max(total, preço)` entre as linhas, para comparar os personagens de relance.
- **Hover / foco:** a parte gasta (`gasto = total − saldo`) escurece, o resto fica aceso, e o número
  vira o saldo, em destaque (negrito, texto primário) — "saldo 6". Saldo negativo: o preenchimento
  todo escurece e o número fica, por exemplo, "saldo −3".
- **Servidor:** `BarView.Balance` conta `agiram ++ pending[:1]` (não todas as pendentes). O jogador não
  muda. Contrato: `actionBalance`/`moveBalance` = o saldo depois que a próxima ação pagar.
- **Dica:** o `Tooltip` do projeto (`src/components/ions/Tooltip.tsx`), com o texto de hoje
  (`"Ação: saldo 6 de 9"`).
- **Alinhamento:** a tira é um grid CSS só; toda linha desenha toda coluna (célula vazia quando a
  barra não tem nada), então a barra de movimento nunca anda porque a de ação ganhou uma velocidade.
  Números com algarismos tabulares e espaço reservado para 3 dígitos por padrão (o total também para
  uma casa decimal); um valor mais largo alarga a coluna em **todas** as linhas de uma vez.
- O texto `x̄` sai: o total ao lado da barra o substitui.

## Fora do escopo / limites conhecidos
- Uma fuga que falha (ou o mestre arrastando a peça) deixa o `From` do empilhado na origem declarada
  da pilha. Os destinos seguem absolutos, como hoje.
- A linha de ataque a partir de uma posição empilhada não muda.
