# Fase 7 — Reações — pacote de back — design

> **Escopo:** as lacunas de back e de contrato que a sessão de planejamento do front da Fase 7
> levantou em 2026-10-05, já decididas pelo dono do produto (itens 1–5 e 10), mais a
> documentação que o documento mestre manda consertar junto com a Fase 7 (§12) e a decisão do
> aviso do reflexo (item 8). **Um PR, repo `System_X_System`.** O front da Fase 7 vem **depois**
> deste merge, em outro PR, e lê o contrato que este PR escreve.
>
> Documento mestre: [`2026-09-20-front-combat-phases.md`](2026-09-20-front-combat-phases.md) §7.
> Contrato: [`../../dev/api/match-combat-ws.md`](../../dev/api/match-combat-ws.md).
> Plano: [`../plans/2026-10-05-combat-phase-7-reactions-back.md`](../plans/2026-10-05-combat-phase-7-reactions-back.md).
>
> **Branch:** `feat/combat-phase-7-reactions-back`, a partir de `main` em `754366c` (depois dos
> PRs #82 do back e #69 do front).

## 0. O workflow (cópia do §0.1 do documento mestre)

1. **Uma sessão por fase por repo.** Back e front rodam em paralelo quando não tocam arquivo em
   comum — são repos diferentes, então normalmente não tocam.
2. A sessão **lê o documento mestre e o contrato** (`docs/dev/api/match-combat-ws.md`), escreve o
   **design spec** e o **plano**, e **para** para o dono do produto revisar.
3. **Lacuna ou contradição: liste e pare.** Ela volta para o autor do documento, que corrige o
   texto. Não se contorna, e não se decide regra de jogo por conta.
4. Aprovado o spec, a sessão **compacta** e implementa **lendo o próprio plano do disco**. Se
   ela não conseguir implementar a partir do plano, o plano estava incompleto — é melhor
   descobrir nessa hora.
5. Implementação por **subagent-driven-development**, uma tarefa por subagente.
6. **Verificação no browser, com três contas** (`test@`, `test2@`, `test3@mail.com`, senha
   `12345678`): um mestre e dois jogadores, jogando o caminho que o usuário faz.
7. PR aberto dizendo o que foi verificado e **o que não foi**.

> **O passo 6, neste PR.** Este pacote não tem tela: as reações só ganham botão no PR de front.
> A verificação daqui é automatizada, contra a `Room` real por websocket (o padrão dos
> `*_e2e_test.go`), mais um smoke do `GET /history` — ver §7. O caminho do usuário, com as três
> contas, é verificado no PR de front, contra este back já mergeado.

## 1. O que você precisa saber antes

- **Turno** = uma action e suas reactions. `actorId` é o **sheetUUID**, nunca o do jogador. Uma
  pessoa dirige vários personagens; o NPC é do mestre (`charToPlayer` mapeia a ficha do NPC
  para o `master_uuid`).
- **Os dados caem quando a reação chega** (`MatchSession.AttachReaction` → `rollActionDice`).
  Abrir uma reação não rola nada nem cobra nada — as barras foram cobradas no attach.
- `room.go` é dono do lock (`r.mu`). `MatchSession` não tem lock próprio. **Nada que envia a
  cliente roda com `r.mu` preso**; `dispatchPerPlayer` pega o lock sozinho.
- Duas pistas de envio: `r.broadcast` (canal, entregue pelo goroutine do `Run`) e o envio
  direto por cliente (`client.SendMessage`, usado por `dispatchPerPlayer`). **Mensagens em
  pistas diferentes não têm ordem garantida entre si**; na mesma pista, do mesmo goroutine,
  ordem de envio é ordem de chegada.
- **O corte de action já existe** (B2): `turnActionWireLocked` (`room.go`) dá ao mestre
  `actionwire.Full` e a todos os outros — o dono incluído — `ProjectAction` seguido de
  `actionwire.Opened`, com o `move.from`/`move.position` passando pelo portão de fog da peça
  (`openedMoveViewLocked`). Este pacote **reusa essa função para as reações**; não escreve uma
  segunda.
- **A cadeia de reações já é ordenada** (`buildChainOrder`, `attack_chain.go`): primeiro as
  reações abertas, na ordem em que o mestre as abriu (`Turn.OpenedReactionIDs`); depois os alvos
  sem reação aberta, na ordem de `action.TargetID`. Um personagem só conta uma vez (`covered`).
  `TurnResolver` monta `CharacterResults` nessa ordem.
- **O consumo já existe** (Fase 4): `MatchSession.consumePendingFor` tira da fila, por barra que
  a reação cobra, a ação do personagem que `scheduler.BestPendingFor` escolhe — a de melhor
  chave. Uma ação combinada (nas duas barras) sai uma vez só. Ela **não devolve** os IDs
  consumidos: devolve só um `bool`.

## 2. As decisões recebidas

Fechadas pelo dono do produto em 2026-10-05, a partir das lacunas que a sessão do front listou.

| # | Lacuna | Decisão |
|---|---|---|
| 1 | Ninguém recebe a declaração de uma reação | `reaction_opened` leva a reação projetada **como o `turn_opened.action` do B2**: ator, tipo, `move` filtrado pelo fog, sem os números do teste. O mestre recebe inteira. `closedEscape`/`closedDodge` chegam a terceiros rebaixados (`escape`/`dodge`). `match_full_state.openTurn` leva as reações já abertas, com a mesma projeção |
| 2 | Quem reagiu não tem resposta nem reconciliação | `attach_reaction` ganha resposta só para quem reagiu, com `reactionId`, `turnId` e `actorId`. O `match_full_state` traz as reações que o destinatário já anexou. Segundo attach do mesmo personagem na mesma ação: **recusado**, com `error` |
| 3 | A ação consumida some calada | O servidor nomeia os `actionId` consumidos **para o mestre e para o dono**. Com duas na mesma barra, consome a de melhor chave (regra da Fase 4, já em código) — falta só no contrato. **Na reconexão, uma ação consumida não pode aparecer como perdida** |
| 4 | Não está escrito que o mestre reage pelo NPC | O mestre anexa reação pelo NPC. Documentar |
| 5 | A ordem de abertura se perde ao recarregar | Fixar que `targets[]` vem na ordem da cadeia, ou acrescentar o campo |
| 10 | O front teria que escrever nomes de perícia | O servidor deriva os nomes de perícia das reações (`Reflex`, `Repel`, `Evasion`), como já deriva o `hit` |
| 7 | "Default" do escape | A categoria do movimento de cada escape é **fixa** por tipo (matriz do §11.4). Não há seletor |
| §12 | `reacoes.md` | A frase do Accelerate sai (matriz do §11.4); "propositalmente trabalhosas de configurar" é falso — as fechadas são segurar + um toque em Evasão |
| 8 | O aviso do reflexo | **Sai.** Com o turno aberto o resultado é só do mestre; "seu reflexo não basta" revelaria o acerto do atacante. O jogador escolhe entre a passiva, rolar ou escapar **sem saber se precisa** — é aposta, e combina com "arriscar". Reescrever o trecho nesses termos |

As decisões 6 e 9 (o clique em Escapar arma a escolha do destino; cinco botões, com Evasão a um
toque na configuração) são de front. Este PR as registra no §7 do documento mestre e no
`reacoes.md` (§4.7), porque os dois descrevem o gesto ao jogador.

## 3. A forma geral

A reação passa a ter **as mesmas três superfícies que a ação ganhou no fechamento da Fase 6**, e
pelas mesmas funções:

| Superfície | Ação (já existe) | Reação (este PR) |
|---|---|---|
| Resposta a quem enviou | `action_enqueued {actionId}` | `reaction_attached {turnId, reactionId, actorId, consumedActionIds}` |
| Mecânica pública ao abrir | `turn_opened.action` | `reaction_opened.reaction` |
| Reconexão | `openTurn.action`, `ownQueue` | `openTurn.reactions`, `ownReactions` |
| Histórico como foi visto ao vivo | `actions.move_views` | `actions.move_views` da linha da reação |

Nenhum corte novo: o da reação é `turnActionWireLocked`, o mesmo do `turn_opened`. Nenhum portão
de fog novo: é `openedMoveViewLocked`. Nenhuma regra de jogo nova: o consumo, a ordem da cadeia
e as perícias lidas pelo resolvedor já estão em código; o que muda é que o wire passa a dizê-los.

## 4. O desenho, item a item

### 4.1 Item 1 — `reaction_opened` leva a reação

**Payload:**

```json
{
  "type": "reaction_opened",
  "payload": {
    "turnId": "5555…",
    "reactionId": "4444…",
    "reaction": { "uuid": "4444…", "actorId": "2222…", "reactToId": "3333…", "reactionKind": "escape", "move": { "category": "Dash", "position": [6, 4, 0] }, "dodge": { "rollCheck": { "skillName": "Reflex" } } }
  }
}
```

- `reaction` é `actionwire.Action`, montado por `turnActionWireLocked` **por destinatário**:
  mestre `Full`; todo o resto — o dono incluído — `ProjectAction` (rebaixa `closedDodge`→`dodge`
  e `closedEscape`→`escape` para quem não é dono nem mestre, tira a entrada de `Evasion` de
  `skills` junto, tira `consumedActionIds` — §4.3) e depois `Opened`.
- **O que `Opened` corta e o que mantém é exatamente o do `turn_opened`**: cortados os dados,
  `skillValue` e `result` de `dodge`, `repel`, `skills[]`, `move.charge`; **mantidos**
  `speed.rollCheck` e `move.speed`/`move.finalSpeed`. É a leitura de "projetada como o
  `turn_opened.action` do B2" — ver a decisão D1 em §9, que é o ponto que o revisor deve
  confirmar.
- **O `move` passa pelo portão de fog da peça**, como o de uma ação. A origem julgada é **a casa
  em que a peça do reator está quando a reação abre** (`pieceSlotOf`) — a peça de uma fuga não
  anda na abertura, então é também onde ela está. Uma reação nunca tem `move.from` (o mapper não
  o deriva; `match-history.md` já diz isso), então o veredito "só a origem" de
  `openedMoveViewLocked` vira "nada" para ela (a função já faz isso quando `from == nil`):
  `position` só vai a quem vê o destino. Mestre e dono do reator recebem o `move` inteiro.
- **Pista e ordem.** Sai de `r.broadcast` para `dispatchPerPlayer` (direto), porque agora é por
  destinatário. O `resolution_updated` master-only que vem junto já é direto e é enviado pelo
  mesmo goroutine depois — a ordem `reaction_opened` → `resolution_updated` passa a ser promessa,
  e o contrato a escreve.
- **A reação é copiada sob `r.mu`.** `MatchSession.OpenReaction` devolve um ponteiro para a
  reação dentro do turno; o arm de `open_reaction` copia o valor na mesma seção crítica do
  `Execute` (`OpenReactionResult.Opened`, que já existe), e o
  `dispatchPerPlayer` projeta a cópia pegando `r.mu.RLock` dentro do `build`, como o
  `turn_opened` faz.
- **Veredito para o histórico.** Na mesma abertura, para uma reação com `move`, o servidor grava
  o que cada jogador da sessão viu do destino — `sessionPlayerViewsLocked` com o mesmo
  `openedMoveViewLocked` e a mesma origem — em `turnWrites.reactionMoveViews[reactionID]` (só
  enquanto aquele turno ainda é o aberto, como `recordOpenedMoveViews`). O fechamento o grava em
  `actions.move_views` **da linha da reação** (§4.6).

**`match_full_state.openTurn.reactions`** — as reações **abertas** do turno aberto, na ordem em
que o mestre as abriu (`Turn.OpenedReactionIDs`), cada uma com o mesmo `turnActionWireLocked`
do `reaction_opened`. Origem julgada na reconexão: a casa da peça do reator agora. `omitempty`:
ausente = nenhuma reação aberta. Uma reação anexada e **não** aberta nunca entra aqui — ela não
foi anunciada à mesa.

### 4.2 Item 2 — `reaction_attached`, `ownReactions` e o segundo attach

**`reaction_attached`** (servidor → cliente), novo:

```json
{ "type": "reaction_attached", "payload": { "turnId": "5555…", "reactionId": "4444…", "actorId": "2222…", "consumedActionIds": ["3333…"] } }
```

- **Destino: quem reagiu, e o mestre.** A resposta é de quem enviou (decisão 2); o mestre
  recebe a **mesma** mensagem porque a decisão 3 manda nomear a ação consumida também para ele —
  é uma mensagem só, com o mesmo conteúdo, em vez de duas formas para o mesmo fato (convenção do
  `game-server.instructions.md`, "menos tipos, payloads mais ricos"). Quando quem reagiu **é** o
  mestre (reação de NPC, item 4), ele recebe uma cópia só. **A mesa não recebe nada**: que
  alguém reagiu continua não sendo notícia de mesa até o mestre abrir.
- `consumedActionIds` é **sempre** uma lista — `[]` numa reação livre ou numa cobrada que não
  achou nada na fila. É montado na mesma seção crítica do attach (§4.3).
- Sai pela pista direta (`client.SendMessage`), **antes** do `resolution_updated` master-only do
  mesmo attach — os dois do mesmo goroutine.

**O segundo attach.** `MatchSession.AttachReaction` recusa, **antes de rolar ou cobrar
qualquer coisa**, um reator que já tem reação anexada no turno aberto:
`ErrReactorAlreadyReacted` — `"this character already reacted to the open action"`, como
`game_error`. A checagem vale para qualquer `reactionKind`, aberta ou não. É o que
`buildChainOrder` já assume (`covered`: um personagem conta uma vez na cadeia) — hoje um segundo
attach era aceito, cobrado, e ignorado pela cadeia.

**`match_full_state.ownReactions`** — as reações do turno aberto **cujo ator pertence ao
destinatário** (`charToPlayer`; o mestre, pelos NPCs), abertas ou não, na ordem de chegada:

```json
"ownReactions": [ { "reactionId": "4444…", "actorId": "2222…", "reactionKind": "closedEscape", "opened": false, "consumedActionIds": [] } ]
```

- `reactionKind` é o **verdadeiro** — o dono vê o próprio.
- `opened` diz se o mestre já deu a palavra; o front usa para "reação enviada, aguardando o
  mestre" × "é a sua vez de narrar".
- `omitempty`: ausente = nenhuma reação sua no turno aberto (ou nenhum turno aberto). Diferente
  de `ownQueue`, aqui ausente e vazio querem dizer a mesma coisa — não há reconciliação que
  dependa de distinguir os dois (§4.3 explica por quê).

### 4.3 Item 3 — a ação consumida

**Domínio.** `consumePendingFor` passa a devolver os IDs consumidos (`[]uuid.UUID`, na ordem das
barras de `ReactionKind.Bars()`); `AttachReaction` grava a lista na própria reação, num campo
novo `action.Action.ConsumedActionIDs`. A regra de escolha **não muda** — `BestPendingFor`, a de
melhor chave — e o `systemBias = -1` continua valendo quando a lista não é vazia.

**Projeção.** `ConsumedActionIDs` é como a fila: segredo de mestre e dono. `ProjectAction` o
zera para quem não é nem um nem outro. `actionwire.From` o copia em todos os níveis
(`consumedActionIds`, `omitempty`) — quem não deve ver já recebeu a lista zerada da projeção.

**Onde o dono e o mestre ficam sabendo** — as três janelas:

| Quando | Superfície |
|---|---|
| Ao vivo | `reaction_attached.consumedActionIds` (§4.2) — ao dono e ao mestre |
| Reconectou com o turno ainda aberto | `ownReactions[].consumedActionIds` (§4.2) |
| Reconectou depois que o turno fechou (ou o servidor reiniciou depois do fechamento) | `GET /history`: `reactions[].consumedActionIds`, só para mestre e dono (§4.6) |

**A regra de reconciliação do contrato (B12) ganha uma linha:** uma declarada cujo `actionId`
aparece em `ownReactions[].consumedActionIds` **ou** no `consumedActionIds` de uma reação do
histórico **foi consumida**, não perdida — sai da lista sem o aviso de perda e sem devolver o
rascunho. A regra do front já espera o histórico antes de dar uma declarada como perdida
(`combate-fechamento-fase-6.md`, "Reconciliação das declaradas"); a linha nova só lhe dá o que
procurar lá.

> **Por que três superfícies e não uma.** O ack cobre o caso comum; mas um ack pode não chegar
> (a conexão cai entre o attach e a resposta). Se o turno ainda estiver aberto na reconexão,
> `ownReactions` cobre. Se fechou enquanto o jogador estava fora, a reação já está no banco — e
> só o histórico sobrevive a um reinício que venha depois. Um reinício **antes** do fechamento
> perde o turno inteiro, reação e consumo juntos (§5): aí a ação consumida se perdeu de fato,
> como toda a fila, e "perdida" é a resposta certa.

**O que este PR não muda:** o attach continua sem `bars_updated`. As barras e a ordem pública
refletem o consumo no próximo `bars_updated` (o do fechamento, ou o próximo enqueue) — como
hoje. O mestre tira a linha consumida da fila pelo `reaction_attached`.

### 4.4 Item 4 — o mestre reage pelo NPC

**Já funciona**, pelo mesmo mecanismo do `enqueue_action`: `AttachReaction` checa
`charToPlayer[actorId] == playerUUID`, e o NPC está mapeado para o mestre; o arm de
`attach_reaction` não tem checagem de papel. Falta o contrato dizer, e um teste que o prove
(e2e: o mestre anexa uma esquiva pela ficha de um NPC alvo, recebe `reaction_attached`, e a
resolução a mostra pendente). A ficha de **jogador** continua negada ao mestre, como no
`enqueue_action`.

### 4.5 Item 5 — a ordem da cadeia em `targets[]`

**Já é assim** (§1): `CharacterResults` sai na ordem de `buildChainOrder` e `targets[]` é montado
dele. Este PR **fixa** isso no contrato — sem campo novo — e o prova com testes:

- **`targets[]` vem na ordem da cadeia:** primeiro os alvos cuja reação foi aberta, na ordem de
  abertura; depois os alvos sem reação aberta, na ordem de `action.targetId`. (`targets[]` só
  traz personagens — `newResolutionUpdatedPayload` monta de `CharacterResults`; parede não entra
  nele.)
- A ordem vale para o `resolution_updated` aberto, o liquidado projetado (a projeção não
  reordena), o `match_full_state.resolution` e o histórico.
- Teste: dois alvos, reações abertas na ordem inversa de `targetId` → `targets[]` sai na ordem de
  abertura, nos três lugares.

### 4.6 Persistência e histórico

**Migração** `20261005000000_actions_consumed_action_ids.sql`:
`ALTER TABLE actions ADD COLUMN IF NOT EXISTS consumed_action_ids UUID[]` — `NULL` = nada
consumido (ou linha antiga). Só uma reação cobrada a preenche.

**Gravação.** `PersistTurnClose` já insere cada reação com `insertAction(..., nil)` para
`move_views`. Passa a mandar, por reação, o veredito gravado na abertura dela
(`TurnCloseData.ReactionMoveViews[reactionID]`, drenado de `turnWrites` com o resto) e a lista
`ConsumedActionIDs`. Uma reação nunca aberta não tem veredito: `NULL`, como hoje.

**Leitura** (`find_match_history.go`): `consumed_action_ids` volta para
`Action.ConsumedActionIDs`; o `move_views` da linha da reação volta para um mapa novo
`HistoryTurn.ReactionMoveViews` (por `reactionID`).

**Projeção do histórico.**
- `consumedActionIds`: `ProjectAction` já o zera para quem não é mestre nem dono (§4.3).
- **`move.position` de uma reação** — a regra de hoje (`shownReactionMovesFor`) mostra a
  terceiros só a fuga que **escapou** e cuja chegada eles viram, com a justificativa *"a action de
  uma reação nunca vai à mesa ao vivo"*. Essa premissa cai com o item 1. A regra que vale para
  todo o histórico é **cada leitor vê como viu ao vivo**, então passa a ser: o terceiro vê o
  destino se o viu **na abertura** (`ReactionMoveViews[reactionID][leitor] == full`) **ou** viu a
  peça chegar (a regra de hoje). Linha antiga ou reação nunca aberta: sem veredito, falha fechada.
  `match-history.md` é corrigido junto. Ver D3 em §9.

### 4.6.1 O fim da partida revela tudo — o desenho não fecha essa porta

**Direção do dono do produto (2026-10-05):** quando a partida encerrar, os jogadores que
participaram devem poder ver **todos os dados** do histórico — o rótulo verdadeiro das fechadas, a
Evasão, o consumo, o destino de toda fuga, o que o fog escondeu. **Não é implementado neste PR**;
o que este PR garante é que nada impeça implementar depois.

O que torna isso possível, e que este PR preserva:

- **O banco guarda a verdade, não a projeção.** A linha de `actions` tem o `reaction_kind`
  verdadeiro, a entrada de `Evasion`, a finta, o `move` inteiro e agora o `consumed_action_ids`;
  `turns.resolution` tem a resolução liquidada inteira, com o `landing`; `master_actions.content`
  tem o conteúdo inteiro. Os vereditos de fog (`move_views`, `landingViews`, `views`) são gravados
  **ao lado** do dado, nunca no lugar dele.
- **A projeção só acontece na leitura**, em `GetMatchHistoryUC` (`ProjectAction`,
  `ProjectResolution`, os vereditos, `Record.ProjectFor`), a partir de um `Viewer`.

Então a revelação, quando vier, é **um ramo na leitura**: partida encerrada (`match.StoryEndAt !=
nil`, o mesmo critério de `ErrMatchAlreadyFinished`) e leitor que participou → o use case não
projeta (ou projeta como para o mestre). Nenhuma migração, nenhum dado a recuperar.

**Regra para este PR e para os próximos:** nada que uma projeção esconde pode ser descartado na
**gravação**. Um dado escondido de alguém é gravado inteiro e escondido na leitura. A Task 6 do
plano tem um teste que prende isso para as reações (o `closedEscape` volta do banco como
`closedEscape`, com a `Evasion`, o destino e o consumo).

Ficam para quando a revelação for desenhada, e são decisão de produto: quem conta como "participou"
(inscrito aceito? jogou ao menos um turno?), se o que é do mestre (`errors` do motor, notas de
turno, `overridden_action_values`) também se revela, e se a revelação vale para o WS ou só para o
`GET /history`.

### 4.7 Item 10 — o servidor deriva as perícias das reações

Em `buildAction` (`action_mapper.go`), como o `hit`:

- `dodge.rollCheck.skillName` → sempre `Reflex`;
- `repel.rollCheck.skillName` → sempre `Repel`;
- `closedDodge`/`closedEscape`: a entrada `{skillName: "Evasion"}` em `skills` é **acrescentada**
  pelo servidor quando falta. A recusa `reaction "X" must carry an evasion skill entry` deixa de
  ter caso e sai do contrato.

O que o payload mandar nesses campos continua **validado** (nome desconhecido é recusado na
fronteira, como sempre) e depois substituído. O resolvedor já lê exatamente esses três nomes
(`deriveReflex`, `resolveRepel`, `deriveEvasion`) — nada muda no cálculo. Os **componentes**
continuam obrigatórios por tipo (`RequiredComponents`): o front manda `dodge: {}`, `repel: {}`
(ou `repel: {weapon}`) e o `move` das fugas — só não escreve nome de perícia nenhum.

O payload mínimo de cada tipo vai para o contrato, numa tabela:

| `reactionKind` | Payload mínimo além de `actorId`, `reactToId`, `reactionKind` |
|---|---|
| `nothing` | — |
| `dodge`, `closedDodge` | `dodge: {}` |
| `escape`, `escapeGuard` | `dodge: {}`, `move: {category: "Dash", position}` |
| `closedEscape` | `dodge: {}`, `move: {category: "Shift", position}` |
| `repel` | `repel: {}` ou `repel: {weapon}` |

### 4.8 Documentação no mesmo PR

- **Documento mestre §7**: as decisões 1–5, 7, 9, 10 e o 6 (o clique de Escapar), com o que este
  PR entregou; "O default do escape é Dash; o fechado é Shift" vira "a categoria é fixa por tipo,
  matriz do §11.4 — sem seletor". §12 marcado como feito. Tabela do §0: estado da Fase 7.
- **`reacoes.md`**:
  - **§12.1** — sai "Segurando em Escapar, a tela já vem com Accelerate escolhido… trocar por
    Brake" e "Num escape, o movimento precisa ser Shift". Entra a matriz do §11.4 em linguagem de
    jogador: escape e escape defensivo usam o Dash (arranque — "no ar"); o escape fechado usa o
    Shift (deslocamento controlado, medido pelo Brake). O jogador não escolhe a categoria: ela vem
    do tipo de escape.
  - **§12.2** — "As duas esquivas difíceis" é reescrita sem "propositalmente trabalhosas": a
    fechada é **segurar** o botão (Esquivar ou Escapar) e **um toque** em Evasão.
  - **Item 8** — sai "Se o reflexo não for suficiente, o sistema avisa você". Entra: com o turno
    aberto, só o mestre vê o resultado; você escolhe entre ficar na passiva, arriscar (rolar) ou
    escapar **sem saber se precisa** — é uma aposta.
  - **"Como configurar"** — o clique em Escapar não envia direto: ele pede a casa de destino no
    mapa, e o toque na casa envia (decisão 6). Cinco botões: Não fazer nada, Esquivar, Escapar,
    Escape defensivo, Repelir; as fechadas saem de segurar + Evasão (decisão 9).
- **Contrato `match-combat-ws.md`**: §2 (NPC também reage), §3 (índice: `attach_reaction` —
  "alvo; o mestre pelo NPC"; `reaction_attached` novo), `attach_reaction` (perícias derivadas,
  payload mínimo, consumo pela melhor chave, `ownReactions`, segundo attach recusado, nova
  resposta, erros), `reaction_opened` (payload, corte, fog, pista e ordem), `reaction_attached`
  (seção nova), `resolution_updated` (`targets[]` na ordem da cadeia), `match_full_state`
  (`openTurn.reactions`, `ownReactions`, regra de reconciliação com o consumo), §8 (diagrama), §9
  (linha "Turno aberto" cita as reações e o consumo), §7 (erro novo).
- **`match-history.md`**: `consumedActionIds` em `reactions[]`; a regra do `move.position` da
  reação (§4.6).
- **`combat-engine.md`**: a seção do consumo cita a lista; a das perícias da reação cita a
  derivação.
- **`AGENTS.md`**: "Deferred to Phase 4 (reações) — Reaction visibility" deixa de ser pendência.
- **`documentation-map.yaml`**: a migração nova, se o mapa cobrir `migrations/`.

## 5. Reinício, recarga, queda (§0.2 do mestre)

| No meio de… | Recarregar / reconectar | Reiniciar o servidor |
|---|---|---|
| Reação anexada, não aberta | `ownReactions` devolve ao dono "enviada, aguardando o mestre" (`opened: false`) e os consumos; o mestre a vê em `resolution.pendingReactions` | **Perdida com o turno** (o turno só persiste ao fechar). `openTurn` ausente; o front não mostra reação nenhuma. A ação consumida se perdeu junto com a fila — "perdida" é a verdade |
| Reação aberta | `openTurn.reactions` devolve à mesa o balão e o fantasma da fuga, na ordem de abertura; `ownReactions` com `opened: true` ao dono; `resolution.targets[]` ao mestre, na ordem da cadeia | Idem — perdida com o turno |
| Turno fechou com o cliente fora | O turno está no histórico: reações, consumos (mestre e dono) e o destino da fuga como cada um viu ao vivo | O histórico sobrevive — a reconciliação do consumo também |

Nenhum caso deixa o cliente mostrando uma reação que o servidor não tem: tudo o que descreve a
reação vem do servidor a cada conexão.

## 6. Contratos que mudam

| Mensagem / endpoint | Mudança |
|---|---|
| `attach_reaction` (c→s) | perícias derivadas; `Evasion` acrescentada; erro novo do segundo attach; recusa de "evasion skill entry" sai; mestre pelo NPC documentado |
| `reaction_attached` (s→c) | **novo** — quem reagiu + mestre |
| `reaction_opened` (s→c) | ganha `reaction`, projetado por destinatário; pista direta; ordem com `resolution_updated` |
| `resolution_updated` | `targets[]` na ordem da cadeia (documentação, sem mudança de wire) |
| `match_full_state` | `openTurn.reactions`, `ownReactions` |
| `GET /matches/{uuid}/history` | `reactions[].consumedActionIds` (mestre e dono); `move.position` da reação pelo veredito da abertura |

Nada é removido do wire. Um front antigo ignora os campos novos; o `reaction_attached` cai no
`default` do switch dele.

## 7. Testes e verificação

TDD por tarefa, no padrão de cada camada:

- **Unidade**: mapper (perícias derivadas, `Evasion` acrescentada, nome inválido ainda
  recusado); `MatchSession` (segundo attach recusado sem cobrar; `ConsumedActionIDs` com uma e
  com duas barras; ação combinada contada uma vez); `ProjectAction` (`consumedActionIds` some para
  terceiro, fica para dono e mestre); `actionwire.From` (copia o campo).
- **E2E** contra a `Room` real por websocket (`combatFixture`):
  - `reaction_attached` ao reator e ao mestre, com o consumido; nada à mesa;
  - segundo attach → `error`, barras intocadas;
  - `reaction_opened`: mestre `Full`; dono `Opened`; terceiro com o rótulo rebaixado e o `move`
    pelo fog (vê / não vê o destino) — reusar `withBystander` e o tabuleiro de `escape_e2e_test`;
  - `reaction_opened` chega antes do `resolution_updated` do mestre;
  - `match_full_state` na reconexão: `openTurn.reactions` na ordem de abertura, `ownReactions`
    com `opened` e consumo, para jogador e para mestre (NPC);
  - o mestre reage pelo NPC;
  - `targets[]` na ordem da cadeia: aberto, liquidado projetado, `match_full_state`.
- **Integração** (`-tags integration`, banco real): `PersistTurnClose` grava
  `consumed_action_ids` e o `move_views` da reação; `FindMatchHistory` os lê.
- **Histórico** (use case): `consumedActionIds` só para mestre e dono; `move.position` da reação
  para o terceiro que viu na abertura, mesmo numa fuga que falhou; linha antiga falha fechada.
- **Suíte inteira** + `go test -race ./internal/app/game/` + `go vet` com as tags `smoke` e
  `integration`.

**Smoke manual.** Montar o cenário real por REST (campanha → partida → inscrição → início) e
dirigir o WS à mão é desproporcional a este pacote, que não tem tela; a regra do `CLAUDE.md` da
raiz admite trocar pela evidência automatizada contra o handler real, que é o que os e2e acima
são. **Registrado no PR.** A verificação de ponta a ponta, com as três contas, acontece no PR de
front, contra este back mergeado — e é lá que o `dev-checkout.sh` entra.

## 8. Effort e modelos

Planejamento em **high**, como o documento mestre recomenda; a descoberta está feita (§1, com
arquivo e função). Implementadores despachados com `model: sonnet`; a tarefa que mexe no arm de
`open_reaction` e no `buildMatchFullState` (pista, lock, projeção) vai com **opus** — é a de
`room.go` com ordem de envio e seção crítica, onde o erro é sutil.

## 9. Decisões desta sessão

Forma técnica, decidida aqui. **D1 e D3 são as que o revisor deve olhar.**

| # | Decisão | Por quê |
|---|---|---|
| **D1** | O corte do `reaction` no `reaction_opened` é **exatamente** o `Opened` do `turn_opened`: mantém `speed` e `move.speed`/`finalSpeed`, corta os números de `dodge`/`repel`/`skills` | "Projetada como o `turn_opened.action` do B2" é a instrução operativa, e uma segunda função de corte divergiria da primeira. As velocidades já vão a público pelo `bars_updated` (a reação cobrada grava a velocidade dela na barra). ⚠️ Ponto a confirmar: no escape, o `move.finalSpeed` é uma das duas metades do teste de fuga. Ele só não entrega o desfecho porque o acerto do atacante continua escondido. Se "sem números" quis dizer **nenhum**, a troca é cortar `speed`/`move.speed`/`finalSpeed` só no ramo da reação — uma linha |
| D2 | `reaction_attached` vai a quem reagiu **e** ao mestre, uma mensagem só | A decisão 3 manda nomear o consumo ao mestre; um segundo tipo para o mesmo fato iria contra a convenção do servidor de jogo |
| **D3** | No histórico, o terceiro vê o `move.position` da reação se o viu **na abertura** ou viu a peça chegar | A regra-mãe do histórico é "como foi visto ao vivo"; a regra antiga só escondia porque a reação não ia à mesa ao vivo. Consequência visível: o destino de uma fuga que **falhou** passa a aparecer no histórico para quem o viu na abertura. **Aprovado pelo dono do produto (2026-10-05)** "por enquanto": ao fim da partida, quem participou vê tudo — §4.6.1 |
| D4 | O consumo é persistido numa coluna da linha da reação (`actions.consumed_action_ids`) | É a única superfície que sobrevive a "o turno fechou com o jogador fora, e depois o servidor reiniciou" — e a decisão 3 diz que a consumida nunca aparece como perdida |
| D5 | `ownReactions` vai a todo destinatário que tem reação no turno aberto, **o mestre incluído** (pelos NPCs) | O mestre declara e reage pelos NPCs; a reconciliação dele (`queue`) precisa do mesmo dado |
| D6 | `ownReactions` e `openTurn.reactions` são `omitempty` (ausente = vazio) | Nenhuma reconciliação depende de distinguir "ausente" de "vazio" aqui — diferente de `ownQueue` |
| D7 | O segundo attach é recusado para **qualquer** tipo, aberto ou não | A decisão 2 não distingue, e a cadeia já conta um personagem uma vez só |
| D8 | O nome de perícia mandado pelo cliente continua validado antes de ser substituído | Mesmo comportamento do `hit` (P4): nome desconhecido é bug de cliente e é recusado na fronteira |

## 10. Fora de escopo

- **O front da Fase 7** — botões, gesto, balões, fantasma de espera, "dar a palavra" no card F7.
  Outro PR, depois deste merge, com spec e plano próprios.
- `bars_updated` depois do attach (§4.3).
- A revelação do histórico inteiro ao fim da partida (§4.6.1) — o desenho a permite; a regra
  ainda vai ser desenhada.
- Cancelar ou trocar uma reação já anexada (o segundo attach é recusado, não substitui).
- A soma do movimento à esquiva, a colisão, a resolução da finta (§11 do mestre).
- O bug conhecido das condições do mestre sobre `dodge`/`defense`/`repel` (`AGENTS.md`) — Fase 8.
