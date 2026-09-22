# O combate no front — Fases 6 a 9

> **Documento mestre.** Cada fase vira uma sessão que escreve o seu design spec e o seu
> plano em cima deste texto, implementa, e abre **um PR**. Como foi nas Fases 1 a 5.
>
> **Leia isto inteiro antes de planejar uma fase.** Ele carrega decisões que não existem em
> nenhum outro lugar — foram fechadas em conversa e vêm parar aqui justamente para não se
> perderem.

## 1. O que este documento é, e o que ele não é

As Fases 1 a 5 construíram o **motor de batalha** no backend: a economia de turno, a colisão,
o catálogo de reações, a regência e a visibilidade. Está tudo em `main` e tudo documentado.

As Fases 6 a 9 constroem **a interface** — no repo `System_X_System_React`, mais um pacote de
preparação no repo Go.

**A numeração continua** de propósito. "Fase 6 — Front" já é citada em `combat-engine.md`, no
spec do motor, nos flows e em várias descrições de PR. Renumerar como "front 1" tornaria todas
essas menções erradas de uma vez. É um arco só; o front são as últimas quatro fases dele.

**O que você NÃO deve fazer:** ler o Go para descobrir como o servidor se comporta. O contrato
é [`docs/dev/api/match-combat-ws.md`](../../dev/api/match-combat-ws.md) e
[`match-history.md`](../../dev/api/match-history.md). **Divergência entre o contrato e o Go é
bug do contrato** — conserta-se o documento, não se contorna indo ler o código.

## 2. Vocabulário mínimo

Para quem chega sem ter lido o motor:

- **Turno** = uma action **e** suas reactions. **Round** = sequência de turnos. **Cena** =
  sequência de rounds.
- **Regime**: `Free` (exploração, sem preço) e `Race` (batalha, com economia de barra). O
  mestre troca à mão.
- **Duas barras** por personagem: ação e movimento. Cada uma com saldo próprio, atravessando
  rounds.
- O mestre **passa o bastão**: abre a próxima action da ordem, ou dá a palavra a uma reaction
  anexada. **A ordem em que ele abre as reactions muda o desfecho.**
- Todo cálculo acontece **quando a action/reaction chega**. O mestre nunca re-rola o dado de um
  jogador.
- **Não existe verbo de confirmação.** O mestre edita, recalcula na hora, e passar o bastão é a
  confirmação.
- `actorId` é o **sheetUUID** do personagem, nunca o do jogador.
- Wire em **camelCase** dos dois lados, sem conversão.

## 3. O que o back já faz

Tudo isto existe e está no contrato: `enqueue_action`, `attach_reaction`, `open_next_action`,
`pull_action`, `open_reaction`, `edit_action`, `close_turn` (com `close_turn_refused`),
`change_round_mode`, `change_scene`, `bars_updated`, `turn_opened`, `turn_closed`,
`round_closed`, `resolution_updated`, `action_queued`, `GET /matches/{uuid}/history`.

**Dois eixos de visibilidade**, e os dois importam para o front:

| Eixo | Regra |
|---|---|
| **Tempo** (`IsSettled`) | turno aberto → `resolution_updated` é **master-only**. Turno fechado → projetado para todos |
| **Classe** | mestre (tudo) · dono (tudo o que é dele) · todo o resto (tudo menos a deny-list) |

⚠️ **O mesmo `turnId` chega com conteúdo diferente para cada pessoa.** Um cache de resolução
indexado só por `turnId` está errado por construção.

⚠️ **`bars_updated` tem `seq`.** Guarde o maior e descarte snapshot menor. Não existe evento de
autocorreção — é requisito, não otimização.

⚠️ **O rótulo é o vazamento.** `closedDodge` chega a terceiros como `dodge`, `closedEscape` como
`escape`. **Não tente "corrigir" isso no cliente.** Ele deduz pela barra pública, que mostra que
o escape fechado cobrou uma barra onde o padrão cobra duas.

---

## 4. Preparação — o pacote de backend

> ✅ **Mergeado (PR #74).** Esta seção fica como registro do que foi decidido e por quê — a
> Fase 6 depende dessas razões. O que **ainda falta** no back para a Fase 6 está em §4.10.

**PR próprio no repo Go, antes da Fase 6.** Não é uma fase: é preparação, mesmo status do
rostering de NPC. Nada da Fase 6 é testável sem isto.

### 4.1 Nada move a peça

Nenhuma mensagem servidor→cliente aplica um movimento resolvido ao tabuleiro. O `piece_moved`
que existe é cliente→servidor, do lobby. O motor só usa `move.from` para checar parede. **Hoje
uma ação de mover acontece e a peça não sai do lugar.**

**A regra, e ela é a parte importante desta seção:**

| O movimento… | A peça |
|---|---|
| **não depende de teste** | desloca **na abertura** da action |
| **depende de uma CD** (salto, passar colado, e — quando as regras de colisão existirem — entrar em slot ocupado) | **não desloca**. Mostra-se a intenção; o servidor decide onde ela para, no fechamento |

⚠️ **Hoje, na prática, nenhum movimento cai na segunda linha.** A action só aceita Dash e Shift,
e desloca na abertura qualquer que seja a categoria. **Entrar em slot ocupado desloca e
empilha** (§10.3): o teste que isso deveria exigir é regra de colisão, e as regras de colisão
ainda não foram desenhadas (§11.3). A segunda linha da tabela descreve a regra; ela passa a ter
casos alcançáveis quando os movimentos com teste chegarem.

Por que na abertura, e não no fechamento: o dano espera o fechamento porque pode ser editado; a
posição não pode esperar, porque as reactions seguintes dependem de onde a peça está.

⭐ **Invariante: o front nunca calcula onde a peça para.** Ele desenha o pedido e depois desenha
a posição que chegou do servidor — seja o slot pretendido, o meio do caminho, ou lugar nenhum.
É isso que permite que toda a complexidade da §10 chegue depois sem tocar no front.

O servidor aplica, recalcula o fog e transmite **com projeção** — quem não enxerga aquele
quadrado não recebe.

### 4.2 Não existe snapshot de combate ao conectar

`map_full_state` cobre o mapa e só. Quem entra no meio — ou **reconecta**, e o hook reconecta
até 5 vezes sozinho — fica sem barras, sem regime, sem cena, sem turno aberto e sem reações
pendentes, até alguma coisa mudar por acaso.

Precisa de um `match_full_state`, projetado como todo o resto.

### 4.3 O jogador não recebe o ID da própria ação

`action_enqueued` é `{}`. O navegador do jogador não tem como se referir à ação que ele acabou
de mandar: não consegue cancelá-la, nem destacá-la na barra geral, nem saber que a próxima da
fila é a dele. `acoes.md` diz que uma ação pode ser cancelada — e hoje ela é inendereçável.

É o mesmo buraco que `PendingReactions` fechou para o mestre na Fase 4: **uma operação cujo ID
o cliente não recebe é uma operação que o cliente não consegue invocar.**

> O ID é daqui. **O verbo de cancelar não é da Fase 6**: ele não existe no contrato, e o front
> não tem contra o que implementar. É fatia futura de back + contrato.

### 4.4 Um bug vivo que o front esconde

`GamePage.tsx` manda `skillName: "combat_strength"`, que não existe em `enum.SkillName`. O
servidor recusa, e **o front ignora `error` num catch vazio** — então o ataque de parede por
jogador falha em silêncio hoje.

O conserto do `skillName` é daqui. **Tratar `error` é a primeira tarefa da Fase 6** — sem isso
o resto do caminho é depurado às cegas.

### 4.5 Não existe catálogo de perícias e armas

O contrato usa `"Strength"`, `"Deception"` e `"sword"` nos exemplos. Os três seriam recusados:
`Strength` é atributo e não perícia, a finta é `Feint`, e `WeaponNameFrom` é **case-sensitive**
(`"Sword"`). Não existe endpoint que entregue o catálogo.

Sem isso a bottom sheet não tem o que listar. Precisa do endpoint **e** da correção dos
exemplos do contrato.

> Agravante: o front faz `lowercaseFirstKeys` em chaves de enum vindas do backend
> (`characterClassesService.ts`), então terá que recapitalizar para enviar.

### 4.6 `Push` entra no dano

`RawDamage(dados, arma, catálogo)` não recebe perícia nenhuma. A regra é que o dano é medido
por **Push** — que já existe em `enum.SkillName`, junto com `Grab`.

Passe o `Push` do personagem para dentro do `RawDamage`. **Sem seletor**: o jogador não escolhe
isso, a arma é que dá o dano. Trocar `Push` por `Grab` (ou outra) é prerrogativa do mestre e
entra na superfície de edição dele, na **Fase 8**.

### 4.7 A finta está escondida para sempre, e não deveria

Hoje `ProjectAction` tira `Feint` de todo não-dono, inclusive no histórico, meses depois.

**A regra correta é temporal, não por classe.** Se o alvo cai na finta, ele descobre **dentro da
resolução do mesmo turno**: o sucesso dele foi contra um ataque falso, e logo vem o real. Nesse
momento ele vê que caiu.

Então: **turno aberto esconde, turno fechado revela.** É o mesmo eixo `IsSettled` que a projeção
da resolução já usa.

> A **resolução** da finta — qual teste o alvo rola contra ela — não existe e não é deste
> pacote. Ver §11.

### 4.8 Miudezas

- `useMatchWs.ts` não manda `nickname` no handshake (o do lobby manda). Definir se é
  obrigatório e alinhar.
- Validar a categoria de movimento dos escapes no **servidor** (§11.4). Hoje `Displaces()` só
  exige que exista um `Move`, sem olhar a categoria — se a regra ficar só no front, o cliente
  vira dono dela.

### 4.9 O que NÃO precisa de conserto

**O HP já tem fonte única.** `applyDamage` muta `s.charSheets[targetID]` — a ficha viva — e
`UpdateStatusBars` persiste essa mesma ficha, que é o que o REST lê. A ficha do personagem já é
a verdade para os dois canais.

**A visibilidade do HP no REST já está certa.** `GET /matches/{uuid}/participants` devolve
`CharacterSheetWithVisibilityResponse`, que põe o HP dentro de um `private` **nulo** para quem
não tem direito; e `GET /campaigns/{uuid}` serve a resposta pública (sem HP) ao jogador e a
privada só ao mestre. Não mexa.

### 4.10 O que o back ainda deve à Fase 6

Achado pela sessão que foi planejar a Fase 6, ao ler este documento contra o contrato. **Sem
estes itens, tarefas específicas da Fase 6 não têm contra o que ser implementadas** — a Fase 6
pode escrever spec e plano agora, mas implementa essas tarefas só depois do merge.

| # | O quê | Destrava na Fase 6 |
|---|---|---|
| **B1** | `turn_opened` passa a carregar **`actionId`**, não só `actorId`. Com duas ações do mesmo personagem na fila, hoje o jogador não sabe qual abriu | apagar o fantasma certo; destacar "a minha" na barra geral |
| **B2** | **HP ao vivo por WS**, projetado: vai **só para o mestre e para o dono** da ficha. Emitido de onde o dano é aplicado, não de `turn_closed` — o motor já produz `DamagedCharacter{CharacterID, NewHP}` | a barra de HP do mestre e a do próprio jogador |
| **B3** | **`turn_closed` nos dois caminhos que fecham turno.** Hoje só `close_turn` o emite (`room.go`); o fechamento implícito do `open_next_action` fecha calado | a lista de eventos da mesa (§5.3) |
| **B4** | **`attack.hit` derivado pelo servidor**, como `speed` e o movimento já são — **e** o valor padrão documentado no contrato. Hoje ele vem do cliente, e o front teria que escrever um nome de perícia à mão, que é exatamente o que o catálogo existe para evitar | a bottom sheet sem campo de perícia |

**Consertos de contrato**, sem código:

- O §2 de `match-combat-ws.md` ainda diz **"NPC hoje não age"**. Desde o PR #73 o mestre age
  por NPC: `enqueue_action` do mestre com `actorId` de NPC é aceito. Corrigir, e documentar essa
  aceitação explicitamente.
- O contrato chama o pouso em slot ocupado de **"sem caso alcançável"**. Um Dash para slot
  ocupado é aceito, então o caso existe: a peça desloca e empilha.

**Por que HP por WS e não por REST.** A linha que vale para a partida inteira: **mudança de
estado compartilhado vai por WS; leitura de dado de referência pode ir por REST.** O histórico e
o catálogo de perícias são referência — grandes, estáveis, lidos sob demanda. O HP depois do dano
é **estado**: muda a cada turno, e a mesa precisa saber na hora. Buscá-lo por REST a cada turno é
N idas ao servidor para reenviar ficha inteira por causa de um número — e o gatilho óbvio para
isso, o `turn_closed`, nem é emitido em metade dos fechamentos (B3). O REST do HP fica para o
carregamento inicial.

**O que continua fora:** o verbo de cancelar ação (§4.3) e adicionar NPC com a sala já viva, que
é do PR paralelo do verbo WS do rostering. A Fase 6 **não** depende de nenhum dos dois.

---

## 5. Arquitetura do front

### 5.1 Duas páginas, um template

**Telas separadas para mestre e jogador.** Não uma página só que esconde elementos.

A razão não é "o mestre tem mais componentes". É esta: **o servidor já projeta o payload por
destinatário.** Se a página esconder elementos condicionalmente, todo componente passa a
perguntar *"sou mestre?"* — e essa pergunta já foi respondida no backend, uma vez, no lugar
certo. Perguntá-la em trinta componentes é reimplementar a política de visibilidade no cliente,
que é exatamente o que a Fase 5 construiu para não precisar.

**A rota lê o papel uma vez e monta a página certa.** Daí para baixo, nenhum componente decide
visibilidade.

```
routes            /partidas/:id  → lê o papel, monta a página
pages             GamePlayerPage          GameMasterPage
templates                  MatchStageTemplate
organisms      TacticalMapStage · GeneralBar · ActionBalloon
               ActionStream · CharactersSidebar · CharacterSheetTemplate
```

### 5.2 As cinco zonas do `MatchStageTemplate`

O template decide **onde** cada zona aparece em cada largura. As páginas decidem **o que** vai
dentro. É a mesma divisão que `DetailPageTemplate` já usa hoje.

| Zona | Jogador | Mestre |
|---|---|---|
| `rail` | ficha · ação · inventário · nen | fila · fichas |
| `panel` | o que o rail ativou | idem — a **fila** são as actions pendentes, só o mestre vê |
| `stage` | o mapa Pixi — **nunca colapsa** | idem |
| `aside` | abas **Histórico** (padrão) e Personagens | idem, com HP |
| `topbar` | cena · regime · round | idem + regência |

**O rail mostra só o que a fase entrega.** Na Fase 6, o do jogador tem **Ação**; ficha,
inventário e Nen entram quando suas fases chegarem. A zona é a mesma do começo ao fim — o que
cresce é o conteúdo.

> Uma versão anterior pôs um item **"mapa"** no rail do mestre, sem defini-lo. Saiu: as
> ferramentas de parede e fog continuam sendo o overlay no próprio canvas, como já são hoje.

⚠️ **NÃO promova o `GamePageTemplate` atual.** Ele é uma casca de duas zonas (canvas + gaveta)
que existe para provar que o mapa funciona. Ele vira **uma** das zonas do novo — o `stage`.

### 5.3 A aba padrão é Histórico, e o HP é do mestre

**Esquerda × direita, de uma vez:** a **fila** (esquerda, só mestre) é o que foi enviado e ainda
não aconteceu. O **histórico** (direita, todos) é o que já aconteceu.

O `aside` abre em **Histórico**, não na lista de personagens: em partida, o que importa é o que
está acontecendo, não quem está na sala.

> Uma versão anterior chamava essa aba de **"Ações"** — o nome servia tanto para pendentes
> quanto para resolvidas, e confundiu quem foi planejar. É **Histórico**.

**O que ela mostra muda de fase.** Na **Fase 6**, é a lista dos eventos da mesa que o servidor
já emite: turno aberto (quem age), turno fechado com o resultado, round fechado, troca de
regime. Na **Fase 8**, vira o histórico de verdade — `GET /matches/{uuid}/history`, aninhado por
cena. O componente é o mesmo; a fonte cresce. (O "turno fechado" depende de B3, §4.10.)

**A lista de personagens do jogador não mostra HP.** A vida é dado privado de cada ficha — só o
mestre e o dono veem. E o front não precisa de nenhum `isMaster` para isso: o REST do
carregamento inicial já devolve `private: null` para quem não tem direito, e o HP ao vivo (B2,
§4.10) só chega a quem tem direito. O mesmo componente renderiza o que chegou.

### 5.4 O rail e o rodapé são o mesmo componente

No desktop o rail fica em pé à esquerda; abaixo do breakpoint ele deita e vira rodapé. O
`panel` é coluna no desktop e bottom sheet no celular.

⚠️ **Se nascerem como componentes separados, vão divergir para sempre** — e aí toda mudança
custa dois lugares, que é exatamente o que a decisão de ter um template compartilhado existe
para evitar.

### 5.5 Breakpoints nomeados

Hoje existem **480, 500, 749, 767/768, 940 e 1149** espalhados em media queries. Quatro
formatos × duas páginas com número mágico é como a UI começa a divergir entre telas sem ninguém
perceber.

| Nome | Largura | O que muda |
|---|---|---|
| `phone` | < 768 | rodapé + bottom sheets |
| `tabletUp` | ≥ 768 | sheets maiores |
| `railUp` | ≥ 1024 | rail em pé + painel em coluna |
| `asideUp` | ≥ 1280 | `aside` fixa em vez de gaveta |

O tablet gira: em pé usa `tabletUp`, deitado cai em `railUp`. Os quatro formatos entram na Fase
6 — **não deixe para depois**. O custo de fazer junto é escrever as media queries uma vez a
mais; o custo de fazer depois é descobrir que o painel de compor ação assumiu altura fixa.

### 5.6 A ficha dentro da partida é um quarto `SheetMode`

`CharacterSheetTemplate` já é reusado por view, create e edit através de um `SheetMode`
composto. A ficha dentro da partida é **mais um modo**, não uma reconstrução. O requisito é que
ela seja praticamente idêntica à de fora da partida.

---

## 6. Fase 6 — A casca e o loop mínimo

**Objetivo:** um turno inteiro jogável, sem reações, nas duas telas e nos quatro formatos.

**Escopo:**
- `MatchStageTemplate` com as cinco zonas, os quatro breakpoints nomeados, rail↔rodapé e
  painel↔bottom sheet.
- `GamePlayerPage` e `GameMasterPage` como thin orchestrators.
- **Tratar `error`** — primeira tarefa, antes de qualquer outra (§4.4).
- Peça clicável no jogo: `TacticalMapViewer` não passa `piecesInteractive` nem `onPieceSelect`
  ao `TacticalMapStage`. A infra existe em `PiecesLayer`, usada pelo editor, e está **desligada
  no jogo**.
- **Seleção de ator e de alvo** — ver o bloco logo abaixo.
- **O mestre compõe ação por um NPC**, com a **mesma** bottom sheet do jogador e o NPC como
  ator. O back já aceita (PR #73): `enqueue_action` do mestre com `actorId` de NPC.
- Bottom sheet de ação: alvo, arma, movimento. **Sem campo de perícia** — ver §11.1. O `hit` é
  derivado pelo servidor (B4, §4.10), então o front não escreve nome de perícia nenhum.
- **Movimento**: as duas categorias oferecidas — **Dash** e **Shift** —, com **Dash
  pré-marcado**. Faça o default ser uma **função, não uma constante**: o doc de jogo
  (`barra-de-acao.md`) diz que no turno livre o deslocamento normalmente é Shift, e o default
  pode passar a seguir o regime. Desenhe para ser enriquecido; não decida isso agora.
- **Rascunho persistente**: fechar preserva; trocar de alvo **migra** em vez de resetar. Vive em
  `localStorage`, por personagem + partida, para sobreviver ao refresh.
- As duas barras: a própria (no painel) e a geral (flutuando sobre o mapa, com `seq`).
- **HP ao vivo**: a barra do mestre (todos) e a do próprio jogador (só a dele). Chega por WS
  (B2, §4.10); o REST fica para o carregamento inicial.
- Mestre: fila, `open_next_action`, `pull_action`, `close_turn` com o diálogo de
  `close_turn_refused`, `change_round_mode`.
- **Fantasma da intenção declarada** (§10.2): a peça mostra para onde o dono mandou ir, do
  envio até a abertura. Só o dono vê — o mestre não conhece o destino, porque `action_queued` e
  `resolution_updated` não trazem posição. Apagar o fantasma certo depende de B1.
- **Histórico**, na versão da Fase 6: a lista de eventos da mesa (§5.3).
- Consumir `match_full_state` na conexão e em toda reconexão.

#### Seleção de ator e de alvo

A UX, como o dono do produto desenhou:

| Quem | Ator | Alvo |
|---|---|---|
| **Jogador** | implícito — é o personagem dele | clicar numa peça marca o alvo |
| **Mestre** | clicar numa peça que ele controla (um NPC) a seleciona como ator, e a UI mostra que está selecionada | em seguida, clicar em **outra coisa** — campo, parede, personagem — marca o alvo |

- **Segurar** marca mais de um alvo.
- **Alvejar a si mesmo é legítimo**, para jogador e para mestre. Por isso clicar de novo na
  própria peça marca ela como alvo — **não** desfaz a seleção.
- **Remover o ator exige um botão explícito** (um X, ou equivalente). Não pode ser clicar de novo
  (isso é alvejar a si mesmo) nem segurar (isso marca alvos, podendo incluir a própria peça).
- Decida e registre: o que acontece quando o mestre clica numa peça que ele **não** controla sem
  ter ator selecionado.

⚠️ **O gesto de segurar é decidido AQUI, não na Fase 7.** Uma versão anterior deixava "clicar e
segurar no desktop" em aberto para a Fase 7, por causa dos botões de reação. A seleção de
múltiplos alvos o puxa para cá. São dois usos do mesmo gesto — segurar **sobre a peça** marca
alvos; segurar **sobre o botão de reação** (Fase 7) abre a configuração —, e eles se distinguem
porque os botões ficam ao lado da peça, não em cima dela. **O mecanismo no desktop se decide uma
vez só**, aqui: long-press com timer sobre o Pixi, botão direito, ou outro. A zona do mapa é
pixel-tuned e isso não é barato. Registre no spec.

#### Dependências

Tudo acima é implementável com o que está em `main` **exceto** o que depende de §4.10:

| Tarefa | Espera |
|---|---|
| apagar o fantasma certo; destacar "a minha" na barra geral | B1 |
| HP ao vivo | B2 |
| "turno fechado" na lista do histórico | B3 |
| bottom sheet sem campo de perícia | B4 |

**Escreva o spec e o plano agora; ordene o plano para que essas tarefas venham depois do merge
de §4.10.** O resto não espera.

**Fora de escopo:** reações; edição do mestre; o histórico completo por REST (Fase 8); ficha;
inventário; Nen; **cancelar ação** (não existe no contrato — §4.3); **adicionar NPC com a sala
já viva** (depende do PR paralelo do verbo WS do rostering); **o fantasma de teste** — o de um
movimento que depende de CD — que só tem caso alcançável quando os movimentos com teste chegarem
(§4.1).

**Pronto quando:**
- Duas pessoas em máquinas diferentes — uma mestre, uma jogador — completam um round inteiro:
  declarar, o mestre abrir, a peça se mover, o turno fechar, as barras e o HP andarem. Em
  desktop **e** em celular.
- **O mestre age por um NPC** e ele age de verdade. O NPC precisa estar na partida **antes de a
  sala nascer** (posto pelo REST do PR #73) — adicionar com a sala viva não é critério desta fase,
  porque depende de outro PR.

## 7. Fase 7 — Reações

**Objetivo:** o combate de verdade.

**Escopo:**
- Os **sete** `reactionKind`, cada um com seus componentes obrigatórios e seu custo de barra.
  ⚠️ Não é um gesto só: `nothing`, `dodge`, `closedDodge`, `escape`, `escapeGuard`,
  `closedEscape`, `repel`.
- Botões de reação ao lado do alvo. **Clicar** envia direto; **clicar e segurar** abre a
  configuração. O gesto de segurar **já foi decidido na Fase 6** — reuse o mesmo mecanismo, não
  invente um segundo.
- Mestre: `open_reaction`, com a ordem de abertura visível — ela muda o desfecho.
- Balões: mecânica ao abrir, resultado ao encerrar.
- O default do escape é **Dash**; o fechado é **Shift** (§11.4).
- **O fantasma de espera** (§10.2): hoje o **escape com Dash** é o único movimento em que a
  peça **espera o fechamento** para se deslocar (o escape com Shift e toda ação deslocam na
  abertura). É o primeiro caso alcançável de fantasma que não é a intenção declarada da Fase 6.

  ⚠️ **A regra por trás disso está sob revisão.** O código trata "o Dash rola dado" como se fosse
  "o Dash tem CD" — e rolar a velocidade de movimento não é um teste contra dificuldade. A
  consequência é que o mesmo Dash desloca na abertura numa ação e no fechamento num escape.
  **Não construa lógica no front que dependa desse momento**: pela invariante de §4.1, desenhe
  o que o servidor mandar, quando mandar.

**Pronto quando:** três alvos reagem diferente ao mesmo ataque em área, e abrir as reactions em
ordem inversa produz resultado diferente na tela.

## 8. Fase 8 — Leitura e regência

**Objetivo:** a mesa inteira.

**Escopo:**
- Action History: `GET /matches/{uuid}/history`, **aninhado por cena** — a hierarquia do domínio
  é a da resposta; não achatar. Não existe método em `matchService.ts`, nem hook, nem tipo.
- A ficha dentro da partida (§5.6).
- Edição do mestre: `edit_action` — rolagem e perícias. É aqui que entra o seletor de perícia de
  dano, `Push` → `Grab` (§4.6).
- `change_scene`.

## 9. Fase 9 — Inventário e Nen

Não existem no backend. `aura?: StatusBar` está tipado como opcional exatamente porque o
servidor não serializa. Fase reservada, sem escopo escrito.

---

## 10. O movimento e suas complexidades

Esta seção existe porque o movimento é muito mais rico do que "andar até um slot", e quem
implementar a Fase 6 precisa saber disso **antes** de desenhar a peça.

### 10.1 Movimentos que exigem teste

- **Salto** permite deslocar mais de um slot de uma vez, e exige teste. Se falhar, o personagem
  fica **"no ar"**, no meio do caminho, exposto a ataques — e **não chega ao slot pretendido**.
- **Entrar no slot de outro personagem** tem várias resoluções possíveis: os dois compartilham o
  slot de alguma forma, um segura o outro, um sobe no outro, ou um passa **colado** no outro
  (mesmo slot). Passar colado expõe e exige teste — é análogo ao ataque de oportunidade do D&D.

### 10.2 Como desenhar a intenção

A peça real fica onde está, uma cópia **translúcida** aparece no slot pretendido, e uma seta
liga as duas. Quando o servidor manda a posição, a peça vai para lá e o fantasma some.

**São dois fantasmas, com vidas diferentes** — e é o mesmo desenho para os dois:

| Fantasma | Existe entre… | Quem vê | Fase |
|---|---|---|---|
| **Intenção declarada** | o envio da action e a abertura dela | **só o dono** — o mestre não conhece o destino, porque `action_queued` e `resolution_updated` não trazem posição | **6** |
| **Espera** | a abertura e o fechamento, quando a peça não pode deslocar na abertura | a mesa | **7** em diante |

O primeiro tem caso alcançável hoje: toda action enviada antes de abrir. O segundo só passa a
ter com o escape com Dash (Fase 7) e com os movimentos que dependem de CD, quando chegarem.

É solução de protótipo. Se for ruim na prática, troca-se **só o desenho** — o contrato não muda,
porque o front não calcula posição.

### 10.3 Empilhamento

Se o slot final já estiver ocupado por outro personagem, **a peça fica empilhada** — uma em
cima da outra.

Não existe desenho definido, e **o visual fica a seu critério**. Escolha o que ficar legível no
protótipo e diga no PR o que escolheu; a gente evolui em cima disso.

### 10.4 "No ar" é z + status

`PieceMovedPayload.Z` **já existe** — altura virtual em metros, `0` = chão. O servidor nunca lê:
a linha de visão é calculada em 2D, e a elevação atravessa como dado opaco.

O front escreve `z: 0` no editor e no placer, mas **`PiecesLayer` não renderiza z nenhum**. E
**não existe conceito de status de peça** — "no ar" é campo novo.

Por que o status importa além do z: um personagem deslocado em z pode estar **em cima de alguma
coisa** ou **no ar**, e são situações diferentes. O z sozinho não distingue.

> Enquanto o motor não souber devolver posição intermediária, falha de teste = a peça não sai
> do lugar. **Não invente meio caminho no front.**

---

## 11. Regras escritas que o código não executa

Nomeadas aqui para ninguém as implementar por engano nem fingir que funcionam.

### 11.1 A corrente de testes

Cada perícia de uma action é um teste com CD própria; a margem atravessa de um teste para o
próximo; errar por 10 ou mais mata a corrente. Está em
[`docs/game/combate/acoes.md`](../../../docs/game/combate/acoes.md).

**`match_session.go` rola cada `Skill` e ninguém lê o resultado.** A única leitura de `Skills`
em toda a colisão é a `Evasion` da esquiva fechada, por nome.

⚠️ **Por isso a Fase 6 não põe seletor de perícias na bottom sheet.** Um controle que o jogador
mexe e que não muda nada é pior do que controle nenhum.

### 11.2 A resolução da finta

Qual teste o alvo rola contra uma finta ainda **não foi desenhado**. A UI não desenha resultado
de finta até a regra existir. (A *visibilidade* da finta é outra coisa e se conserta agora —
§4.7.)

### 11.3 Posição intermediária, empilhamento, status de peça — e colisão

§10.3 e §10.4.

**As regras de colisão ainda não foram desenhadas, e vão existir.** Elas decidem o que acontece
quando alguém colide: compartilhar o slot, ser bloqueado, segurar, subir no outro, passar
colado — ou **quebrar a parede no impacto**, que conecta com o dano estrutural que o motor já
tem. **Valem contra personagens e contra todo tipo de parede, incluindo terreno.**

Hoje, sem elas, **um escape atravessa parede** e entrar em slot ocupado empilha.

⭐ **Isso não trava nenhuma fase do front.** As regras de colisão mudam **qual posição o servidor
manda**; o front só desenha a posição que chegou (§4.1). Quando elas existirem no back, o front
não muda uma linha. O que o front precisa é **não** assumir, em lugar nenhum, que a peça chega
sempre ao slot pedido.

### 11.4 A categoria de movimento dos escapes

A matriz, fechada:

| Reação | Movimento |
|---|---|
| `escape` (padrão) | **Dash** — o default do botão |
| `escapeGuard` (defensivo padrão) | **Dash** — mesma lógica do padrão |
| `closedEscape` (fechado) | **Shift** |

O discriminador é **fechado × aberto**, não defensivo × padrão. Fechado usa Shift; os outros
usam Dash.

⚠️ **Nada disso está em código.** `Displaces()` só exige que exista um `Move`, sem olhar a
categoria. Validação no servidor, §4.8.

### 11.5 Outras

`ReboundDamage` calculado e nunca aplicado ao ator; armadura reduz zero; `action.Initiative`
órfão; a exceção do percept no início de batalha (bloqueada pelos subatributos mentais);
posturas.

---

## 12. Consertos de documentação

Foram escritos por uma sessão nossa que injetou a contradição sem questionar. **Conserte junto
com a fase que tocar no assunto (Fase 7).**

**Em `docs/game/combate/reacoes.md`:**

1. *"Segurando em Escapar, a tela já vem com Accelerate escolhido"* seguido, doze linhas abaixo,
   de *"Num escape, o movimento precisa ser Shift"*. Contradição direta — Accelerate governa o
   Dash. **O certo é a matriz de §11.4.**
2. *"Esquiva fechada e escape fechado são propositalmente trabalhosas de configurar"* — **é
   falso**. Não deve ser difícil de configurar. Reescreva a seção "As duas esquivas difíceis"
   sem essa premissa.

---

## 13. Armadilhas conhecidas

- **`error` é ignorado num catch vazio.** Toda recusa do servidor desaparece hoje. Trate antes
  de qualquer outra coisa, ou o resto da fase é depurado às cegas.
- **O mesmo `turnId` chega diferente para cada pessoa.** Cache por `turnId` sozinho está errado.
- **`bars_updated.seq`**: guarde o maior, descarte menor. Não há autocorreção.
- **Não "conserte" o rótulo rebaixado.** `closedDodge` chegando como `dodge` é a regra, não bug.
- **A peça não é clicável no jogo hoje** — a infra existe e está desligada.
- **`piece_moved` do lobby é cliente→servidor.** Não é o mesmo que a Fase 6 precisa.
- **O tablet gira.** Testar em pé e deitado, não só em duas larguras.
- **A zona Pixi é pixel-tuned** e não deve ser normalizada com os tokens.
- **NPC existe, mas só entra na sala quando ela nasce.** O rostering (PR #73) está mergeado: o
  mestre põe NPC pelo REST, e o `InitMatchSession` o traz na próxima vez que a sala for criada.
  O mestre age por ele. **O que não existe ainda é pôr NPC com a sala já viva** — `cmd/api` e
  `cmd/game` são processos separados, e o REST não alcança a sessão em memória. Isso é o verbo
  de WS de um PR paralelo, e **a Fase 6 não depende dele**: para testar, ponha o NPC antes de a
  sala nascer.
