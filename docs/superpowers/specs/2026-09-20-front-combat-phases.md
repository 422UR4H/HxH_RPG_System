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

**PR próprio no repo Go, antes da Fase 6.** Não é uma fase: é preparação, mesmo status do
rostering de NPC. Nada da Fase 6 é testável sem isto.

### 4.1 Nada move a peça

Nenhuma mensagem servidor→cliente aplica um movimento resolvido ao tabuleiro. O `piece_moved`
que existe é cliente→servidor, do lobby. O motor só usa `move.from` para checar parede. **Hoje
uma ação de mover acontece e a peça não sai do lugar.**

**A regra, e ela é a parte importante desta seção:**

| O movimento… | A peça |
|---|---|
| **não depende de teste** (shift/dash para slot livre) | desloca **na abertura** da action |
| **depende de uma CD** (salto, entrar em slot ocupado, passar colado) | **não desloca**. Mostra-se a intenção; o servidor decide onde ela para, no fechamento |

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

> O verbo de cancelar em si é da Fase 6. O ID é daqui.

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
a verdade para os dois canais. A única janela de divergência fecha no `turn_closed`.

**A visibilidade do HP no REST já está certa.** `GET /matches/{uuid}/participants` devolve
`CharacterSheetWithVisibilityResponse`, que põe o HP dentro de um `private` **nulo** para quem
não tem direito; e `GET /campaigns/{uuid}` serve a resposta pública (sem HP) ao jogador e a
privada só ao mestre. Não mexa.

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
| `rail` | ficha · ação · inventário · nen | fila · fichas · mapa |
| `panel` | o que o rail ativou | idem |
| `stage` | o mapa Pixi — **nunca colapsa** | idem |
| `aside` | abas **Ações** (padrão) e Personagens | idem, com HP |
| `topbar` | cena · regime · round | idem + regência |

⚠️ **NÃO promova o `GamePageTemplate` atual.** Ele é uma casca de duas zonas (canvas + gaveta)
que existe para provar que o mapa funciona. Ele vira **uma** das zonas do novo — o `stage`.

### 5.3 A aba padrão é Ações, e o HP é do mestre

O `aside` abre em **Ações**, não na lista de personagens: em partida, o que importa é o que está
acontecendo, não quem está na sala.

**A lista de personagens do jogador não mostra HP.** A vida é dado privado de cada ficha — só o
mestre e o dono veem. E o front não precisa de nenhum `isMaster` para isso: o REST já devolve
`private: null` para quem não tem direito, então o mesmo componente renderiza o que chegou.

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
- Bottom sheet de ação: alvo (clicando na peça), arma, movimento. **Sem perícias** — ver §11.1.
- **Rascunho persistente**: fechar preserva; trocar de alvo **migra** em vez de resetar. Vive em
  `localStorage`, por personagem + partida, para sobreviver ao refresh.
- As duas barras: a própria (no painel) e a geral (flutuando sobre o mapa, com `seq`).
- Mestre: fila, `open_next_action`, `pull_action`, `close_turn` com o diálogo de
  `close_turn_refused`, `change_round_mode`.
- Movimento com fantasma (§10).
- Consumir `match_full_state` na conexão e em toda reconexão.

**Fora de escopo:** reações, edição, histórico, ficha, inventário, Nen.

**Pronto quando:** duas pessoas em máquinas diferentes — uma mestre, uma jogador — completam um
round inteiro: declarar, o mestre abrir, a peça se mover, o turno fechar, as barras andarem. Em
desktop **e** em celular.

## 7. Fase 7 — Reações

**Objetivo:** o combate de verdade.

**Escopo:**
- Os **sete** `reactionKind`, cada um com seus componentes obrigatórios e seu custo de barra.
  ⚠️ Não é um gesto só: `nothing`, `dodge`, `closedDodge`, `escape`, `escapeGuard`,
  `closedEscape`, `repel`.
- Botões de reação ao lado do alvo. **Clicar** envia direto; **clicar e segurar** abre a
  configuração.
- Mestre: `open_reaction`, com a ordem de abertura visível — ela muda o desfecho.
- Balões: mecânica ao abrir, resultado ao encerrar.
- O default do escape é **Dash**; o fechado é **Shift** (§11.4).

**Em aberto para quem planejar:** "clicar e segurar" é gesto de toque. No desktop vira o quê —
long-press com timer sobre o Pixi, botão direito, hover? A zona do mapa é pixel-tuned e isso não
é barato. Decida e registre.

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

Enquanto o teste não resolve: a peça real fica onde está, uma cópia **translúcida** aparece no
slot pretendido, e uma seta liga as duas. No fechamento, a peça vai para onde o servidor mandar
e o fantasma some.

É solução de protótipo. Se for ruim na prática, troca-se **só o desenho** — o contrato não muda,
porque o front não calcula posição.

### 10.3 Empilhamento — não desenhado

Se o slot final já estiver ocupado por outro personagem, **a peça deve ficar empilhada**, uma
em cima da outra. **Não existe desenho para isso ainda.**

> Deixe comentado no código, explicitamente, no ponto onde o empilhamento aconteceria. Não
> invente um visual: é decisão de produto que ainda não foi tomada.

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

### 11.3 Posição intermediária, empilhamento, status de peça

§10.3 e §10.4.

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

⚠️ **O modelo não sabe expressar "escape fechado defensivo".** `ReactionKind` trata fechado e
defensivo como valores irmãos de um enum, não como eixos ortogonais. Fica registrado; **não
remodele agora** — há uma ideia maior, de compor actions por domínios, guardada para quando as
actions forem revisitadas.

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
- **Sem NPC não há inimigo.** O rostering é fatia própria, em paralelo. Sem ele o teste é
  personagem de jogador contra personagem de jogador — legítimo para o motor, mas não é uma
  mesa.
