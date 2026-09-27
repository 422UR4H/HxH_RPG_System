# Fechamento da Fase 6 e a Fase 7 — o que o back já faz e o front não mostra

> **Documento de handoff.** Escrito a partir de uma auditoria dos dois repos em
> 2026-09-27, depois de o dono do produto jogar a Fase 6 e relatar o que faltava. Quem pegar
> este documento **não precisa redescobrir nada**: cada relato abaixo já tem a causa achada,
> com arquivo e linha. O trabalho aqui é transformar isto em design spec e plano.
>
> **Leia antes:** o documento mestre
> [`2026-09-20-front-combat-phases.md`](2026-09-20-front-combat-phases.md) (arquitetura §5,
> invariantes §4.1 e §13) e o contrato [`match-combat-ws.md`](../../dev/api/match-combat-ws.md).
> Este documento **não repete** o que está lá — ele diz o que falta.

## 1. O diagnóstico em um parágrafo

A Fase 6 entregou o loop mínimo, e entregou bem: declarar, abrir, mover, fechar, barras e HP
ao vivo. Mas o front consome uma fração do que o back produz. **Dez tipos de mensagem do
servidor nunca são tratados nem enviados pelo front** (§4). **As reações não existem no
front** — a Fase 7 nunca foi implementada, só a 6 e a revisão dela. E há **um buraco no
back** que nenhum front conseguiria contornar: **nenhuma mensagem servidor→cliente carrega a
declaração de uma ação de jogador** — nem para o mestre, nem para o alvo. O próprio contrato
admite isso (`match-combat-ws.md`, tabela de lacunas: *"Nenhuma mensagem servidor→cliente
projeta a declaração de uma action de JOGADOR"*). É por isso que o mestre não vê arma nem
alvo na fila, e é por isso que o alvo de um ataque não tem como saber que é alvo.

## 2. Os relatos, com a causa

Cada relato é do dono do produto, jogando. A causa foi verificada no código.

### R1 — O mestre não vê o que foi declarado

> *"Na fila só aparece o nome do actor, um ícone de batalha e um de movimento. Não vejo a
> arma, quem é o alvo, o attack speed, a speed, o teste de aceleração — até a action ser
> aberta. O mestre precisa ter toda a visibilidade possível."*

**Causa — back.** `ActionQueuedPayload` (`internal/app/game/message.go`) carrega só
`actionId`, `actorId` e `bars`. O servidor tem a action inteira na memória — com as
velocidades já derivadas, porque o cálculo acontece na chegada — e não manda. A fila do
mestre dentro do `match_full_state` tem o mesmo formato.

**Conserto:** B1 (§5) e F1 (§6).

### R2 — O mestre não consegue agir por um NPC

> *"Ao selecionar o NPC aparece 'Este personagem não está inscrito na partida — é um NPC do
> mapa, e não age.'"*

**Causa — front.** O back tem os dois caminhos para pôr um NPC na partida: `add_npc`/
`npc_added` por WS (com a sala viva) e `POST /matches/{uuid}/npcs` por REST. **O front não
chama nenhum dos dois.** O `NpcPicker` lista só NPC que já está em `participants`
(`GameMasterPage.tsx`, o `useMemo` de `npcs`: `participants.filter(p =>
!p.characterSheet.playerUuid)`). Um NPC da campanha posto no mapa aparece como peça (vem de
`campaign.characterSheets`, via `npcMap`), mas não é participante — e clicar nele cai na
mensagem acima, sem saída.

> A verificação no browser da Fase 6 registrou "ação pelo NPC do mestre" como feita. Passou
> porque o NPC do teste foi inscrito por fora, antes de a sala nascer.

**Conserto:** F2 (§6). Nenhuma mudança de back.

### R3 — A ficha abre em outra tela

> *"A ficha está sendo aberta em outra tela, e deve abrir na própria tela de game."*

**Causa — front.** As duas páginas fazem `navigate('/charactersheet/${uuid}')`
(`GameMasterPage.tsx` e `GamePlayerPage.tsx`, prop `onSelectCharacterSheet`). A ficha dentro
da partida estava prevista para a Fase 8 (documento mestre §5.6: *"um quarto `SheetMode`"*);
o dono do produto a quer agora.

**Conserto:** F3 (§6).

### R4 — O histórico some

> *"Ao ir para a ficha e voltar, o histórico some. Ao recarregar a página, mestre e jogadores
> perdem o histórico. Nada deve ser perdido durante a partida."*

**Causa — front.** O histórico da tela é `state.events` do `combatReducer` — só memória.
Nada o persiste e nada o reconstrói. O back **já tem** a fonte certa: `GET
/matches/{uuid}/history` devolve os turnos fechados, aninhados por cena, projetados por
leitor (`docs/dev/api/match-history.md`). **O front nunca chama esse endpoint** — não há
método em service, nem hook, nem tipo.

Consertar R3 esconde o sintoma da navegação, mas o recarregar continua perdendo tudo.

**Conserto:** F4 (§6). Nenhuma mudança de back.

### R5 — Os cards de personagem não seguem a campanha

> *"Na campanha cada card tem a cor de NPC, morto ou jogador. No game não. E o avatar e a
> capa dos NPCs não aparecem para o mestre. O componente deveria ser reutilizado."*

**Causa — front, e não é o componente.** O game **reusa** o `CharacterSidebarItem`, o mesmo
da campanha, e ele já sabe se renderizar: a cor de NPC vem de `!playerUuid`, a de morto de
`deadAt`, e avatar e capa de `avatarUrl`/`coverUrl`. O problema está no **wrapper**
`src/features/match/MatchCharactersSidebar.tsx`: quando o participante não tem `private`, ele
**não chama o card** — cai num `BasicParticipantItem` que mostra só o nome, com borda laranja
fixa.

E `private` falta em dois casos comuns:

- **Para o jogador, olhando qualquer outro personagem** — o que está certo: HP é privado.
- **Para o NPC do mapa que não está inscrito** — ele entra na lista (`everyone`, em
  `GameMasterPage.tsx`) com o formato plano da campanha (`CharacterPrivateSummary`), sem o
  envelope `{...base, private}`.

Só que **`playerUuid`, `deadAt`, `avatarUrl`, `coverUrl` e `nickName` são públicos** —
estão na parte base da resposta (`CharacterBaseSummaryResponse`, no back), não no `private`.
O wrapper joga fora dado público que o card precisa.

**Conserto:** F5 (§6).

### R6 — O mestre não vê a barra de cada personagem

> *"Vejo a ordem de execução, mas o mestre deve ver literalmente a barra de ação dos
> personagens: os valores numéricos e a progress-bar da action speed e da move speed de
> cada um. No topo do canvas."*

**Causa — front.** O dado **já chega**. `bars_updated` traz, por personagem,
`actionBalance`, `moveBalance`, `actionSpeeds` e `moveSpeeds`, mais os preços do round e a
ordem projetada com a chave de cada slot. O `GeneralBar` desenha só a ordem.

> Detalhe que decide o desenho: `bars_updated` traz as velocidades **que já agiram**. As
> velocidades das ações **ainda na fila** não estão lá, de propósito — antes de abrir, elas
> são do mestre. Elas chegam ao mestre por B1.

**A ordenação está correta?** Ela é calculada no servidor (`RoundScheduler`, Fase 3, com
testes do exemplo canônico) e o front só desenha. Não foi verificada **na tela** — e sem os
números à vista, nem dá para verificar. É exatamente para isso que F6 existe: com saldo, preço
e chave visíveis, o mestre confere a ordem sozinho. A verificação está em §9.

**Conserto:** F6 (§6).

### R7 — As reações não aparecem

> *"A 52R4H atacou o Luiz e as possibilidades de reação não aparecem no client dele."*

**Causa — as duas pontas.**

- **Front:** a Fase 7 nunca foi implementada. O front não envia `attach_reaction` nem
  `open_reaction`, e não trata `reaction_opened`.
- **Back:** mesmo que o front tivesse os botões, **o cliente do Luiz não tem como saber que é
  alvo**. `TurnOpenedPayload` carrega `turnId`, `actorId`, `actionId` e `actionType` — nenhum
  alvo, nenhuma arma. E `resolution_updated`, que tem os alvos, é master-only enquanto o turno
  está aberto. A regra de jogo diz o contrário: *"Todos veem a mecânica da ação (alvos, arma,
  perícia); só o mestre vê o resultado"* (`docs/game/combate/acoes.md`, fluxo de uma rodada,
  passo 4).

**Conserto:** B2 (§5) e a Fase 7 (§7).

## 3. O que ninguém relatou, mas a auditoria achou

| # | O quê | Onde vai |
|---|---|---|
| A1 | **A resolução do turno aberto não é desenhada para o mestre.** Ele recebe `resolution_updated` com acerto, esquiva, defesa, reação, escada, dano e reações pendentes — e a tela mostra só uma linha no histórico **depois** que o turno fecha. Enquanto o turno está aberto, que é quando ele decide, não vê nada | F7 |
| A2 | **Trocar de cena não tem UI.** O back aceita `change_scene`; nenhum botão manda | F8 |
| A3 | **`master_action_enqueued` é ignorado.** O mestre manda `enqueue_master_action` (revelar porta, interagir com parede) e o front descarta a confirmação | F9 |
| A4 | **Edição do mestre** (`edit_action`/`action_edited`) não tem UI | Fase 8, como previsto |
| A5 | **Chat** (`chat`) não tem UI em lugar nenhum do front | fora — não pedido |
| A6 | As **oito pendências de back** que a revisão da Fase 6 anotou (`System_X_System_React/docs/dev/match/combate-fase-6.md`, "Pendências para o back") | B3 a B10 |

## 4. Mensagens do servidor que o front nunca usa

Levantado comparando todo `MessageType` de `message.go` com o `src/` do front (testes fora):

| Mensagem | Sentido | Situação |
|---|---|---|
| `add_npc`, `npc_added` | c→s, s→c | R2 — F2 |
| `attach_reaction` | c→s | R7 — Fase 7 |
| `open_reaction`, `reaction_opened` | c→s, s→c | R7 — Fase 7 |
| `change_scene` | c→s | A2 — F8 |
| `master_action_enqueued` | s→c | A3 — F9 |
| `edit_action`, `action_edited` | c→s, s→c | A4 — Fase 8 |
| `chat` | c→s | A5 — fora |

E dois endpoints REST que o front não consome: `GET /matches/{uuid}/history` (R4) e
`POST`/`DELETE /matches/{uuid}/npcs` (R2 — o front usa o `add_npc` por WS, que alcança a sala
viva; o REST fica para montar o roster antes da partida, se um dia houver tela para isso).

---

## 5. Pacote de back — um PR, repo `System_X_System`

### B1 — A fila do mestre carrega a declaração inteira

`action_queued` e a fila do mestre dentro de `match_full_state` passam a carregar a action
**inteira**, sem projeção (as duas já são master-only): alvos, arma, movimento (categoria,
origem, destino), perícias, o `hit` derivado, e **as velocidades já derivadas** —
`actionSpeed` e `moveSpeed`, cada uma com a perícia usada, os dados e o total. O
`systemBias` junto, quando houver.

⭐ **Reuse o formato da action do histórico REST** (`match-history.md`). O front já vai
precisar parsear esse formato por F4; um terceiro formato de action no protocolo seria um a
mais para divergir.

### B2 — `turn_opened` carrega a mecânica pública da ação

`turn_opened` passa a carregar a declaração da action aberta, **projetada por destinatário**
com o mesmo `ProjectAction` que o histórico já usa: dono e mestre veem tudo; o resto vê a
mecânica — alvos, arma, movimento, perícias — sem a deny-list (finta e gatilho escondidos
**enquanto o turno está aberto**; a regra temporal da finta já existe). O mesmo vale para o
`openTurn` do `match_full_state`, senão quem reconecta no meio do turno perde a informação.

Isso destrava três coisas de uma vez: o **alvo saber que é alvo** (Fase 7), o **balão de
mecânica** ao abrir (Fase 7), e o jogador **ver o que está acontecendo** na mesa.

> Não confunda com a resolução. `resolution_updated` continua master-only enquanto o turno
> está aberto — **a mecânica é pública ao abrir, o cálculo não**.

### B3 a B10 — as pendências que a revisão da Fase 6 anotou

Listadas pela sessão que implementou a Fase 6. **Todas entram**: o dono do produto não quer
bug conhecido aberto. Em ordem de gravidade:

| # | Pendência | Por que importa |
|---|---|---|
| **B3** | **As posições das peças só existem em memória.** Quando a sala esvazia ou o serviço reinicia, tudo volta ao REST — e o REST não é atualizado pelos movimentos do jogo | é perda de estado de partida, exatamente o que o dono pediu para não acontecer |
| **B4** | **O mesmo usuário conectado duas vezes** deixa um socket mudo, e fechar esse socket **fecha a sala para todos** | um jogador com duas abas derruba a mesa |
| **B5** | **A checagem de parede usa os cantos dos slots** (`from × gridSize`), não os centros | movimento rente a uma parede é bloqueado ou liberado errado |
| **B6** | **`move.from = [0,0,0]` é sentinela** de "sem origem" e colide com o slot (0,0) de verdade | um movimento que sai do canto do mapa é tratado como sem origem |
| **B7** | **`Register` numa sala fechada** bloqueia para sempre | o front contorna com watchdog; o back deveria recusar |
| **B8** | **O dono do personagem não recebe o próprio `private`** em `GET .../participants` — só o mestre recebe | o dono tem direito à própria vida; o front hoje busca a ficha à parte |
| **B9** | **`enqueue_master_action` com `attack` não é mapeado** (`buildMasterAction` deixa em `TODO`) | o mestre ataca parede por NPC via `enqueue_action`, contornando |
| **B10** | **Grade hexagonal:** confirmar a convenção das triplas `[col,row,z]` | verificar e documentar; consertar se estiver errada |

## 6. Pacote de front — fechamento da Fase 6, repo `System_X_System_React`

Tudo aqui é da Fase 6, que ficou incompleta. **F2 a F9 não dependem do back** e começam já.
F1 espera B1.

### F1 — A fila do mestre mostra a ação inteira *(espera B1)*

O card da fila abre em detalhe: atacante, alvos (por nome), arma, movimento (categoria e
destino), perícias, `actionSpeed` (perícia, dados, total), `moveSpeed` (Accelerate ou Brake,
dados, total), a chave na ordem geral, e as barras que ela cobra. Recolhido, o card fica como
hoje; o detalhe é um toque.

### F2 — O mestre põe NPC na partida e age por ele

O `NpcPicker` passa a listar **todos os NPCs da campanha**, separando os que já estão na
partida dos que não estão. Escolher um que não está manda `add_npc`; ao chegar `npc_added`, ele
vira participante e já fica selecionado como ator. **Um gesto só** — escolher o NPC é pôr e
selecionar.

Clicar num NPC do mapa que não está inscrito deixa de ser beco sem saída: em vez da mensagem,
aparece **"Colocar na partida"**, que faz o mesmo.

### F3 — A ficha abre dentro da partida

Um quarto `SheetMode` do `CharacterSheetTemplate` (documento mestre §5.6), **somente
leitura**, aberto na zona `panel` do `MatchStageTemplate` — coluna no desktop, bottom sheet no
celular. O jogador abre a própria pelo item **Ficha** do rail; o mestre abre qualquer uma
clicando no card. **Nenhum `navigate`.** O HP mostrado é o ao vivo (`character_hp_changed`),
não o do REST, que pode estar um turno atrasado.

### F4 — O histórico vem do servidor

A fonte do histórico passa a ser `GET /matches/{uuid}/history`: buscado ao montar a página e
**rebuscado a cada `turn_closed`** — o WS avisa, o REST busca. Os eventos que só existem ao
vivo (turno aberto, HP mudou) entram por cima, a partir do WS, e somem quando o REST
equivalente chega.

**Nada do histórico vive em `localStorage`.** A fonte é o servidor; guardar no navegador seria
uma segunda verdade que diverge — e não sobrevive a trocar de máquina.

> O que o REST não reconstrói: "round fechado" e "troca de regime" não são turnos, então não
> estão no histórico persistido. A árvore do REST (cena → round, com o regime de cada round)
> permite deduzir os dois. Deduza; não peça endpoint novo por isso.

### F5 — Os cards usam o dado público

`MatchCharactersSidebar` **sempre** renderiza o `CharacterSidebarItem`, montando o
`character` a partir da parte base (`playerUuid`, `deadAt`, `avatarUrl`, `coverUrl`,
`nickName`) e mesclando `private` quando ele existe. O `BasicParticipantItem` sai. O card já
trata a ausência de vida e de experiência — ele faz `if (character.health && ...)`.

Normalize os dois formatos que chegam (`{...base, private}` dos participantes e o plano
`CharacterPrivateSummary` do NPC do mapa) num adaptador só, antes do card.

### F6 — A barra de cada personagem, no topo do canvas

Uma faixa sobre o mapa com **cada personagem**: duas barras (ação e movimento), o saldo em
número, as velocidades que já agiram e a média. A escala da barra é o **preço do round** — o
saldo nunca passa do preço, pelo teto do carry-over, então saldo ÷ preço é a proporção
natural. A ordem projetada continua, com a chave de cada slot visível.

Quem vê o quê, seguindo `docs/game/combate/barra-de-acao.md` ("Enxergando a barra"):

| | Mestre | Jogador |
|---|---|---|
| barras de todos, com números | ✔ | — |
| a própria barra, com números | — | ✔ |
| a ordem geral, com chaves | ✔ | ✔ |

> O dado de `bars_updated` é público, então isto é escolha de interface, não de sigilo. Se um
> dia o jogador precisar ver as barras dos outros, é mudança só de front.

Em telas estreitas a faixa não pode cobrir o mapa: recolhe para a ordem geral, e expande num
toque.

### F7 — O painel de resolução do mestre

Enquanto o turno está aberto, o mestre vê a resolução que já recebe: o acerto (dados e total)
e, **por alvo**, esquiva, defesa, o tipo de reação, a escada do repelir, o dano projetado e os
payouts — mais as reações anexadas e ainda não abertas. É onde ele decide, e hoje ele decide no
escuro.

Os botões de **dar a palavra** às reações pendentes são da Fase 7. Os de **editar** são da Fase
8. Neste pacote o painel é **só leitura** — nada de botão que ainda não faz nada.

### F8 — Trocar de cena

Na topbar do mestre, **Trocar cena** — categoria e descrição inicial — mandando `change_scene`.
A categoria é **validada no servidor** desde o PR #74: mande o valor do enum (`battle`,
`roleplay`), minúsculo.

### F9 — O mestre recebe a confirmação das próprias ações

Tratar `master_action_enqueued`, com o mesmo cuidado que `action_enqueued` já tem: sucesso
confirma, `error` aparece. Hoje uma recusa do servidor a um "revelar porta" some.

---

## 7. Fase 7 — Reações, repo `System_X_System_React` *(espera B2)*

O escopo é o do documento mestre §7, sem mudança: os **sete** `reactionKind` (`nothing`,
`dodge`, `closedDodge`, `escape`, `escapeGuard`, `closedEscape`, `repel`), botões ao lado da
peça do alvo, clicar envia e segurar configura (**reusando o mecanismo de segurar que a Fase 6
já decidiu**), o mestre dando a palavra na ordem que escolher, e os balões.

O que muda com este documento: **o cliente do alvo só sabe que é alvo por B2.** Quando
`turn_opened` chega com ele entre os alvos, os botões aparecem ao lado da peça dele. Depois de
enviar, ele vê "reação enviada, aguardando o mestre".

Os botões de **dar a palavra** entram no painel de resolução do mestre (F7), que já lista as
reações pendentes.

---

## 8. Ordem, dependências e PRs

```
Back   ──  B1 · B2 · B3–B10   ─────────────────┐
                                               ├──►  F1  ──►  Fase 7
Front  ──  F2 · F3 · F4 · F5 · F6 · F7 · F8 · F9 ┘
```

- **O back e o front rodam em paralelo** — repos diferentes, nenhum arquivo em comum.
- F2 a F9 começam já. **F1 espera B1.** A Fase 7 espera **B2 e** o PR de front deste pacote,
  porque os dois tocam o painel do mestre.
- **Três PRs:** o de back; o de front (F1 a F9 — F1 entra depois do merge do back); a Fase 7.

## 9. Verificação

**No browser, com três contas** (`test@`, `test2@`, `test3@mail.com`, senha `12345678`): um
mestre e dois jogadores, jogando de verdade. O jogo é tempo real entre máquinas, e é
exatamente o que teste unitário não pega — a Fase 6 passou nos testes com o NPC inscrito por
fora.

**A ordem, conferida na tela.** Três personagens rolando 20, 23 e 11 (o exemplo canônico de
`barra-de-acao.md`), com `RollSource` injetado no teste ou dados arranjados: a barra de cada
um mostra os saldos, e a ordem tem que sair **p2, p1, p3, p2**. Com F6 no ar, isso vira algo
que o mestre confere olhando os números.

**Recarregar a página** no meio de um turno, nas duas telas: histórico, barras, fila, turno
aberto e reações pendentes voltam. As **posições das peças** também — isso só passa depois de
B3.

## 10. Decisões tomadas aqui — vetáveis

Tomadas para a sessão não parar; o dono do produto pode vetar qualquer uma.

1. **Escolher o NPC é pô-lo na partida** (F2): um gesto, não dois.
2. **Jogador vê a própria barra com números, não a dos outros** (F6), seguindo
   `barra-de-acao.md`.
3. **O histórico não vai para `localStorage`** (F4): o servidor é a fonte.
4. **O painel de resolução nasce só leitura** (F7): sem botão que ainda não funciona.
5. **A ficha dentro da partida é só leitura** (F3).

## 11. Em aberto — do dono do produto

1. **O escape com Dash deveria deslocar a peça na abertura, como a ação com Dash?** Hoje a ação
   com Dash desloca na abertura e o escape com Dash espera o fechamento — o mesmo movimento em
   dois momentos. A origem é um erro de um prompt anterior, que tratou "o Dash rola dado" como
   "o Dash tem CD" (o Accelerate do Dash é velocidade de movimento, não teste contra
   dificuldade). O documento mestre marca isso como sob revisão (§7). **Não trava nada**: o
   front desenha a posição que o servidor mandar, quando mandar. Mas precisa de resposta antes
   de a Fase 7 fechar.
2. **A edição do mestre (A4) continua na Fase 8, ou entra agora?** O painel de F7 é o
   pré-requisito dela; com ele no ar, editar vira o passo natural seguinte.
