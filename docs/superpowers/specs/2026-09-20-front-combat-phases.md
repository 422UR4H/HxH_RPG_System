# O combate no front — Fases 6 a 8

> **Documento mestre.** Cada fase vira uma sessão que escreve o seu design spec e o seu
> plano em cima deste texto, implementa, e abre **um PR**. Como foi nas Fases 1 a 5.
>
> **Leia isto inteiro antes de planejar uma fase.** Ele carrega decisões que não existem em
> nenhum outro lugar — foram fechadas em conversa e vêm parar aqui justamente para não se
> perderem.

## 0. Estado em 2026-09-27 — leia primeiro

**Este é o único documento de fases do combate no front.** Uma auditoria de 2026-09-27 chegou a
virar um documento separado; o conteúdo dela foi consolidado aqui, em §6A, para não haver duas
fontes de verdade.

| Fase | O quê | Estado |
|---|---|---|
| **6** | A casca e o loop mínimo (§6) | ✅ feita — e revisada no front (`System_X_System_React/docs/dev/match/combate-fase-6.md`) |
| **Fechamento da 6** | Visibilidade do mestre, consistência da partida, NPC, histórico, ficha, cards, barras (§6A) | **próxima** — um PR de back e um de front, em paralelo |
| **7** | Reações (§7) | depois do fechamento |
| **8** | Regência — a edição do mestre (§8) | depois da 7 |

**Depois da Fase 8, o próximo passo é enriquecer a mecânica de combate** — regras de colisão
(§11.3), iniciativa, efeitos de ambiente (armadilha — o único caso em que um "ataque do mestre"
faria sentido, §6A.5 B9), e o que mais o dono do produto desenhar. Nada disso tem desenho ainda, e
nada disso trava as fases acima. **Não há uma "Fase 9" planejada**: inventário e Nen não existem
no back e não são o próximo passo.

**O "fechamento" não vira fase numerada** pelo mesmo motivo de sempre: "Fase 7 = reações" e
"Fase 8 = regência" já são citadas em documentos dos dois repos. Renumerar tornaria todas essas
menções erradas. É o mesmo precedente do fechamento da Fase 5 (PR #72).

As seções §4 e §6 ficam como **registro** do que foi decidido e por quê: as fases seguintes
dependem dessas razões.

### 0.1 O workflow de cada fase

> **Sessões que planejam uma fase: copiem esta seção para o design spec de vocês.** Ela é como
> este projeto trabalha, e quem implementa a partir do plano precisa dela tanto quanto quem
> planejou.

1. **Uma sessão por fase por repo.** Back e front rodam em paralelo quando não tocam arquivo em
   comum — são repos diferentes, então normalmente não tocam.
2. A sessão **lê este documento e o contrato** (`docs/dev/api/match-combat-ws.md`), escreve o
   **design spec** e o **plano**, e **para** para o dono do produto revisar.
3. **Lacuna ou contradição neste documento: liste e pare.** Ela volta para o autor deste
   documento, que corrige o texto. Não se contorna, e não se decide regra de jogo por conta.
4. Aprovado o spec, a sessão **compacta** e implementa **lendo o próprio plano do disco**. Se
   ela não conseguir implementar a partir do plano, o plano estava incompleto — é melhor
   descobrir nessa hora.
5. Implementação por **subagent-driven-development**, uma tarefa por subagente.
6. **Verificação no browser, com três contas** (`test@`, `test2@`, `test3@mail.com`, senha
   `12345678`): um mestre e dois jogadores, jogando o caminho que o usuário faz. A Fase 6 passou
   nos testes e falhou na mesa — o NPC do teste tinha sido inscrito por fora, e o caminho real
   nunca foi exercitado.
7. PR aberto dizendo o que foi verificado e **o que não foi**.

### 0.2 Regra transversal: nenhum reinício deixa cliente e servidor divergindo

**Todo desenho, de toda fase, tem que responder: o que acontece se o servidor reiniciar, se o
cliente recarregar, ou se a conexão cair e voltar — no meio de qualquer coisa?** A resposta
aceitável é sempre a mesma: **cliente e servidor voltam a concordar**, sem ação manual de
ninguém.

- **O servidor é a fonte.** Estado que o cliente guarda por conveniência (rascunho, aba aberta)
  pode existir; estado que **descreve a partida** (fila, histórico, posições, turno aberto) vem
  do servidor, e o cliente se reconcilia com ele a cada conexão.
- **Perder estado no servidor pode ser aceitável; divergir, não.** Se algo se perde num
  reinício, todos os clientes ficam sabendo — nenhum continua mostrando o que não existe mais.
- **Reconciliar nunca re-sorteia.** Um cliente não reenvia sozinho o que o servidor perdeu:
  reenviar é rolar os dados de novo.
- **O spec de cada fase tem uma seção sobre isso**, e a verificação inclui recarregar o cliente
  e reiniciar o servidor no meio do fluxo que a fase entrega.

O caso que originou esta regra é o R8 (§6A.2): depois de um reinício, o mestre perdeu a fila e
os jogadores continuaram vendo as ações que tinham declarado.

**Effort.** A recomendação é **high** para a sessão que planeja: o effort alto paga em
descoberta, e a descoberta de cada fase já está feita neste documento, com arquivo e linha.
`xhigh`/`max` gastariam reexplorando o que está escrito.

**A decisão final é da sessão.** Ela vai explorar coisas que este documento não previu, e pode
concluir que uma tarefa pede mais — ou menos. Se mudar o effort, de uma tarefa ou de um trecho,
registre no spec por quê.

Os subagentes que implementam cada tarefa bem especificada podem rodar mais baratos. O `model`
se escolhe a cada despacho (`sonnet`). O **effort** de um subagente vem da definição do tipo de
agente (frontmatter de `.claude/agents/*.md`), não do despacho: para rodar implementadores em
**medium**, crie um tipo de agente implementador com esse effort e despache por ele. Confira o
nome exato da chave de frontmatter ao criar — não a invente.

## 1. O que este documento é, e o que ele não é

As Fases 1 a 5 construíram o **motor de batalha** no backend: a economia de turno, a colisão,
o catálogo de reações, a regência e a visibilidade. Está tudo em `main` e tudo documentado.

As Fases 6 a 8 constroem **a interface** — no repo `System_X_System_React`, mais um pacote de
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

> ✅ **Mergeado.** Os itens aqui eram chamados B1–B4 e foram renomeados para **P1–P4**, porque
> colidiam com os B1–B13 do fechamento (§6A.5). O `AGENTS.md` do back chamava o P2 de "dívida B2".

Achado pela sessão que foi planejar a Fase 6, ao ler este documento contra o contrato. **Sem
estes itens, tarefas específicas da Fase 6 não têm contra o que ser implementadas** — a Fase 6
pode escrever spec e plano agora, mas implementa essas tarefas só depois do merge.

| # | O quê | Destrava na Fase 6 |
|---|---|---|
| **P1** | `turn_opened` passa a carregar **`actionId`**, não só `actorId`. Com duas ações do mesmo personagem na fila, hoje o jogador não sabe qual abriu | apagar o fantasma certo; destacar "a minha" na barra geral |
| **P2** | **HP ao vivo por WS**, projetado: vai **só para o mestre e para o dono** da ficha. Emitido de onde o dano é aplicado, não de `turn_closed` — o motor já produz `DamagedCharacter{CharacterID, NewHP}` | a barra de HP do mestre e a do próprio jogador |
| **P3** | **`turn_closed` nos dois caminhos que fecham turno.** Hoje só `close_turn` o emite (`room.go`); o fechamento implícito do `open_next_action` fecha calado | a lista de eventos da mesa (§5.3) |
| **P4** | **`attack.hit` derivado pelo servidor**, como `speed` e o movimento já são — **e** o valor padrão documentado no contrato. Hoje ele vem do cliente, e o front teria que escrever um nome de perícia à mão, que é exatamente o que o catálogo existe para evitar | a bottom sheet sem campo de perícia |

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
isso, o `turn_closed`, nem é emitido em metade dos fechamentos (P3). O REST do HP fica para o
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
regime. No **fechamento da Fase 6** (F4, §6A.6), vira o histórico de verdade — `GET /matches/{uuid}/history`, aninhado por
cena. O componente é o mesmo; a fonte cresce. (O "turno fechado" depende de P3, §4.10.)

**A lista de personagens do jogador não mostra HP.** A vida é dado privado de cada ficha — só o
mestre e o dono veem. E o front não precisa de nenhum `isMaster` para isso: o REST do
carregamento inicial já devolve `private: null` para quem não tem direito, e o HP ao vivo (P2,
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
  derivado pelo servidor (P4, §4.10), então o front não escreve nome de perícia nenhum.
- **Movimento**: as duas categorias oferecidas — **Dash** e **Shift** —, com **Dash
  pré-marcado**. Faça o default ser uma **função, não uma constante**: o doc de jogo
  (`barra-de-acao.md`) diz que no turno livre o deslocamento normalmente é Shift, e o default
  pode passar a seguir o regime. Desenhe para ser enriquecido; não decida isso agora.
- **Rascunho persistente**: fechar preserva; trocar de alvo **migra** em vez de resetar. Vive em
  `localStorage`, por personagem + partida, para sobreviver ao refresh.
- As duas barras: a própria (no painel) e a geral (flutuando sobre o mapa, com `seq`).
- **HP ao vivo**: a barra do mestre (todos) e a do próprio jogador (só a dele). Chega por WS
  (P2, §4.10); o REST fica para o carregamento inicial.
- Mestre: fila, `open_next_action`, `pull_action`, `close_turn` com o diálogo de
  `close_turn_refused`, `change_round_mode`.
- **Fantasma da intenção declarada** (§10.2): a peça mostra para onde o dono mandou ir, do
  envio até a abertura. Só o dono vê — o mestre não conhece o destino, porque `action_queued` e
  `resolution_updated` não trazem posição. Apagar o fantasma certo depende de P1.
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
| apagar o fantasma certo; destacar "a minha" na barra geral | P1 |
| HP ao vivo | P2 |
| "turno fechado" na lista do histórico | P3 |
| bottom sheet sem campo de perícia | P4 |

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

## 6A. Fechamento da Fase 6

> Auditoria dos dois repos em 2026-09-27, depois de o dono do produto jogar a Fase 6. Cada
> relato abaixo já tem a causa, com arquivo e linha — **não refaça a descoberta**.

### 6A.1 O diagnóstico

A Fase 6 entregou o loop mínimo. Mas o front consome uma fração do que o back produz: **dez
tipos de mensagem nunca são tratados nem enviados** (§6A.4). E há um buraco de back que nenhum
front contorna: **nenhuma mensagem servidor→cliente carrega o conteúdo de uma ação de
jogador** — o próprio contrato admite, na tabela de lacunas de `match-combat-ws.md`.

> Precisão, porque a frase confunde: o mestre **recebe a existência** da ação —
> `action_queued`, com o ID, é o que desenha o card e permite abri-la. O que não chega a
> ninguém, pelo WS, é o **conteúdo**: arma, alvos, movimento, velocidades. Antes de abrir, nem o
> mestre vê; depois de abrir, o jogador alvo continua sem ver.

### 6A.2 Os relatos, com a causa

**R1 — O mestre não vê o que foi declarado.** *"Só o nome do actor, um ícone de batalha e um de
movimento. Não vejo a arma, o alvo, o attack speed, a speed, o teste de aceleração."*
Causa, **back**: `ActionQueuedPayload` (`internal/app/game/message.go`) carrega só `actionId`,
`actorId` e `bars`. O servidor tem a action inteira, com as velocidades já derivadas, e não
manda. A fila do mestre no `match_full_state` tem o mesmo formato. → B1, F1.

**R2 — O NPC do mapa não age.** *"Este personagem não está inscrito na partida — é um NPC do
mapa, e não age."*
Causa, **back — e não é base velha.** Só dois lugares inserem em `match_participants`:
`start_match` (a partir de inscrições **aceitas**, ou seja, só jogadores) e o `add_npc`
explícito. **Pôr uma peça de NPC no mapa da partida nunca a inscreve.** Uma partida criada hoje
tem o mesmo problema. E o front piora: o `NpcPicker` lista só NPC que já está em `participants`
(`GameMasterPage.tsx`, o `useMemo` de `npcs`), e clicar num NPC do mapa cai numa mensagem sem
saída. → B11, F2.

> A regra do dono do produto: **NPC no mapa da partida é NPC da partida.** Não faz sentido um
> estar no tabuleiro e fora da partida.
>
> A verificação no browser da Fase 6 registrou "ação pelo NPC do mestre" como feita. Passou
> porque o NPC do teste foi inscrito por fora — o caminho real nunca foi exercitado.

**R3 — A ficha abre em outra tela.** Causa, front: as duas páginas fazem
`navigate('/charactersheet/${uuid}')` (prop `onSelectCharacterSheet`). → F3.

**R4 — O histórico some ao navegar e ao recarregar.** *"Nada deve ser perdido durante a
partida."*
Causa, front: o histórico da tela é `state.events` do `combatReducer` — só memória. O back **já
persiste** o turno a cada fechamento (`PersistTurnClose`, com a resolução em
`turns.resolution`) e **já expõe** `GET /matches/{uuid}/history`, aninhado por cena e projetado
por leitor. **O front nunca chama esse endpoint.** → F4.

**R5 — Os cards não seguem a campanha** (cor de NPC, de morto, de jogador; avatar e capa).
Causa, front — **e não é o componente.** O game reusa o `CharacterSidebarItem` da campanha, e
ele já sabe se renderizar: cor de NPC vem de `!playerUuid`, de morto de `deadAt`, e avatar e
capa de `avatarUrl`/`coverUrl`. O problema é o wrapper
`src/features/match/MatchCharactersSidebar.tsx`: **quando falta `private`, ele não chama o
card** — cai num `BasicParticipantItem` com o nome e borda laranja fixa. E `playerUuid`,
`deadAt`, `avatarUrl`, `coverUrl` e `nickName` **são públicos** (parte base da resposta,
`CharacterBaseSummaryResponse`). O wrapper joga fora dado público que o card precisa. → F5.

**R6 — O mestre não vê a barra de cada personagem.** *"Os valores numéricos e a progress-bar da
action speed e da move speed de cada um, no topo do canvas."*
Causa, front: **o dado já chega.** `bars_updated` traz por personagem `actionBalance`,
`moveBalance`, `actionSpeeds` e `moveSpeeds`, mais os preços e a ordem com a chave de cada slot.
O `GeneralBar` desenha só a ordem. → F6.

> **A ordem está certa?** É calculada no servidor (`RoundScheduler`, Fase 3, com testes do
> exemplo canônico) e o front só desenha. Nunca foi conferida **na tela** — e sem os números à
> vista, nem dá. Com F6, o mestre confere olhando. Verificação em §6A.8.

**R7 — As reações não aparecem no cliente do alvo.** Causa, **as duas pontas**: a Fase 7 nunca
foi implementada no front; e mesmo com os botões, **o cliente do alvo não tem como saber que é
alvo** — `TurnOpenedPayload` carrega `turnId`, `actorId`, `actionId` e `actionType`, nenhum alvo.
A regra diz o contrário: *"Todos veem a mecânica da ação (alvos, arma, perícia); só o mestre vê o
resultado"* (`docs/game/combate/acoes.md`, passo 4). → B2 e Fase 7.

**R8 — Servidor reiniciado: o mestre perde a fila, os jogadores não.** Depois de reiniciar o
servidor, a fila do mestre voltou vazia — o servidor a guarda só em memória —, mas os clientes
dos jogadores continuaram mostrando as ações que tinham declarado: o front guarda a lista em
`localStorage` (`declaredStorage.ts`). **Essa inconsistência não pode existir.** → B12, F10.

**R9 — Selecionar uma peça troca a aba da direita para Personagens.** Comportamento ruim;
sai. → F11.

### 6A.3 O que ninguém relatou, mas a auditoria achou

| # | O quê | Vai para |
|---|---|---|
| A1 | **A resolução do turno aberto não é desenhada para o mestre.** Ele recebe `resolution_updated` com acerto, esquiva, defesa, reação, escada, dano e reações pendentes — e a tela mostra só uma linha no histórico **depois** que o turno fecha. Enquanto o turno está aberto, que é quando ele decide, não vê nada | F7 |
| A2 | **Trocar de cena não tem UI** — o back aceita `change_scene` | F8 |
| A3 | **Revelar/interagir com parede desconhecida é ignorado em silêncio** pelo servidor | B14 |
| A4 | **Edição do mestre** (`edit_action`/`action_edited`) sem UI | Fase 8 |
| A5 | **Chat** sem UI em lugar nenhum do front | fora — não pedido |
| A6 | As **oito pendências de back** que a revisão da Fase 6 anotou (`System_X_System_React/docs/dev/match/combate-fase-6.md`, "Pendências para o back") | B3–B10 |

### 6A.4 Mensagens do servidor que o front nunca usa

| Mensagem | Sentido | Vai para |
|---|---|---|
| `add_npc`, `npc_added` | c→s, s→c | F2 |
| `attach_reaction`, `open_reaction`, `reaction_opened` | c→s, c→s, s→c | Fase 7 |
| `change_scene` | c→s | F8 |
| `master_action_enqueued` | s→c | F12 — o arrastar do mestre é master action **sem** `interact`, e é a primeira vez que o front manda assim |
| `edit_action`, `action_edited` | c→s, s→c | Fase 8 |
| `chat` | c→s | fora |

E o `GET /matches/{uuid}/history`, que o front não consome (F4).

### 6A.5 Pacote de back — um PR, repo `System_X_System`

**B1 — A fila do mestre carrega a declaração inteira.** `action_queued` e a fila do mestre no
`match_full_state` passam a carregar a action **inteira**, sem projeção (as duas já são
master-only): alvos, arma, movimento (categoria, origem, destino), perícias, o `hit` derivado,
e as velocidades já derivadas — `actionSpeed` e `moveSpeed`, cada uma com perícia, dados e
total —, mais o `systemBias`. ⭐ **Reuse o formato de action do histórico REST**
(`match-history.md`): o front já vai parseá-lo por F4, e um terceiro formato de action no
protocolo é um a mais para divergir.

⚠️ **B1 é só do mestre. O dono da ação não pode receber as velocidades dela antes de abrir** —
regra do dono do produto: *o jogador não sabe o valor que a ação dele gerou até o mestre
abri-la*. Isso também vale para o que B12 mandar ao dono.

**B2 — `turn_opened` carrega a mecânica pública da ação.** A declaração da action aberta,
**projetada por destinatário** com o `ProjectAction` que o histórico já usa: o **mestre** vê
tudo; **todos os outros — inclusive o dono** — veem a mecânica (a tabela abaixo diz exatamente
quais campos), sem a deny-list (finta e gatilho escondidos
**enquanto o turno está aberto** — a regra temporal da finta já existe). Vale também para o
`openTurn` do `match_full_state`. Destrava: o alvo saber que é alvo, o balão de mecânica, o
jogador ver o que acontece na mesa. A **resolução** continua master-only enquanto o turno está
aberto — a mecânica é pública ao abrir, o cálculo não.

**Quais números vão, ao abrir**, para todo mundo que não é o mestre — **inclusive o dono**:

| Vai ao abrir | Só no fechamento (resolução projetada e histórico) |
|---|---|
| alvos, arma, movimento, os **nomes** das perícias | os dados e o `result` do acerto |
| `actionSpeed` e `moveSpeed`, com dados e total | os dados e o `result` do dano |
| | os dados e o `result` das perícias e da finta |

O formato continua o do histórico REST; só esses campos ficam de fora. Não é regra nova: é a de
sempre — *"todos veem a mecânica da ação; só o mestre vê o resultado"* (`acoes.md`, passo 4) —,
e "todos" inclui o dono. As velocidades vão porque o `bars_updated` já as revela ao abrir (F6).

**B3 a B10 — as pendências que a revisão da Fase 6 anotou.** Todas entram: o dono do produto
não quer bug conhecido aberto.

| # | Pendência | Por que importa |
|---|---|---|
| **B3** | **O tabuleiro da partida só existe em memória.** Sala vazia ou servidor reiniciado → tudo se perde. São **três** coisas, não uma: **as posições das peças**; **o estado das paredes** — porta aberta, trancada, parede danificada (há um `TODO` para isso em `structural_damage.go:35`); e **o fog que cada jogador já explorou** (o repositório existe em `gateway/pg/fog/`, mas o servidor de jogo começa sempre com `nil`, `room.go:211` e `:366`). **Persista as três por partida**, em todo momento em que o tabuleiro muda de forma definitiva — no fechamento de turno, no arrastar do mestre, nos movimentos do lobby e no `start_match` —, numa estrutura própria que se sobrepõe ao mapa quando a sala nasce — **não** em `maps.pieces` (ver abaixo) | perda de estado de partida |
| **B4** | **Mesmo usuário conectado duas vezes** deixa um socket mudo, e fechar esse socket **fecha a sala para todos** | um jogador com duas abas derruba a mesa |
| **B5** | **Checagem de parede usa os cantos dos slots**, não os centros | movimento rente à parede bloqueado ou liberado errado |
| **B6** | **`move.from = [0,0,0]` é sentinela** de "sem origem" e colide com o slot (0,0) de verdade | movimento que sai do canto é tratado como sem origem |
| **B7** | **`Register` numa sala fechada** bloqueia para sempre | o front contorna com watchdog; o back deveria recusar |
| **B8** | **O dono não recebe o próprio `private`** em `GET .../participants` — só o mestre | o dono tem direito à própria vida |
| **B9** | **`enqueue_master_action` não mapeia `attack` nem `move`** (`buildMasterAction` deixa os dois em `TODO`). O `move` é mapeado. O **`attack` sai do payload** — ver abaixo | o **arrastar do mestre** (B14) precisa do `move` |
| **B10** | **Grade hexagonal:** confirmar a convenção `[col,row,z]` | verificar, documentar, consertar se preciso |

**O `attack` da master action sai — decisão do dono do produto.** O mestre ataca **pelo NPC**, com
`enqueue_action`, como qualquer personagem (PR #73). Mapear um `attack` para dentro de uma master
action não mudaria nada na mesa: nada lê o conteúdo dela. Um "ataque do mestre" só faria sentido
como **efeito de ambiente** — uma armadilha, por exemplo —, e isso ainda não existe: é **futuro**,
parte do enriquecimento da mecânica (§0), não pendência deste pacote. O servidor recusa uma master
action com `attack`, dizendo o caminho certo.

**Onde o tabuleiro da partida é salvo — decisão do dono do produto.** O mapa é da **campanha**:
desenhado no editor de mapas (fora da partida), e o mesmo mapa pode estar anexado a várias
partidas (`match_maps.map_uuid` não é único). Por isso **cada partida tem o seu tabuleiro**,
salvo nela, por cima do mapa:

- o **editor de mapas** mostra sempre o desenho original, e editá-lo com uma partida rolando
  **não** muda o tabuleiro dela;
- duas partidas no mesmo mapa não mexem nas peças uma da outra;
- uma partida pode **começar de onde outra terminou** (B16);
- **o tabuleiro do lobby já é o tabuleiro da partida.** Hoje, ao iniciar a partida, o front grava
  as posições do lobby **no mapa da campanha** (`mapsService.updateMap(... { pieces })`,
  `LobbyPage.tsx:167`) — exatamente o que esta decisão proíbe. Isso sai (F16); o lobby persiste
  no tabuleiro da partida;
- **trocar o mapa anexado depois do `start_match` é recusado.** O anexo diz de qual mapa da
  campanha o tabuleiro da partida **partiu**; trocá-lo por baixo de um tabuleiro vivo deixaria
  posições e estados de parede apontando para paredes que não existem mais. **Isso já é assim:**
  o `AttachMatchMapUC` recusa com `ErrMatchAlreadyStarted` (o *gateway* faz upsert, mas o use case
  não chega nele depois do início). Não há o que implementar.
- ⚠️ **Isso não impede o mestre de mudar o mapa no meio da partida** — só muda **onde** ele muda.
  Vai existir um **editor de mapa da partida** (ainda não existe), e nele o mestre poderá trocar o
  fundo, entre outras coisas. Essas edições são do **tabuleiro da partida**, não do mapa da
  campanha. **Desenhe a estrutura de B3 para comportar isso**: o tabuleiro da partida tem que
  poder sobrepor o fundo do mapa, e não só as posições. Não implemente o editor agora — só não
  feche a porta para ele.

**B11 — NPC no mapa da partida é NPC da partida.** A invariante tem **uma direção só**: toda
peça de NPC no mapa da partida corresponde a um participante. (O contrário não vale: um NPC pode
estar na partida sem peça — um reforço que ainda não entrou em cena.) Onde ela se garante:

- **ao iniciar a partida**, inscrevendo cada NPC que já tem peça no mapa;
- **ao pôr uma peça de NPC no mapa**, no lobby ou com a sala viva;
- **ao nascer a sessão** (`InitMatchSession`), de forma **idempotente** — isso conserta as
  partidas que já existem, inclusive as da base local, **sem mexer à mão no banco**.

Tirar a peça do mapa **não** desinscreve o NPC.

**Como se põe uma peça com a partida rolando:** pela master action de mover (B14) aplicada a um
personagem que ainda não tem peça — ela **cria** a peça. Se for NPC, dispara a inscrição deste
item e o `npc_added`. É o caminho do reforço que entra no meio da cena (F2).

**Tirar uma peça** também é master action, só do mestre, e vale para **qualquer** peça — NPC ou
personagem de jogador: alguém que cai num alçapão, foge da cena, sai pela porta. Tirar a peça
**não desinscreve** o personagem, **não o mata** e **não apaga** o histórico dele.

**Inscrever pela peça avisa a mesa do mesmo jeito que o `add_npc`: com `npc_added`.** O front
não pode ter de adivinhar qual dos dois caminhos inscreveu.

**B12 — A fila não pode divergir entre servidor e cliente.** Requisito: **nenhum cliente mostra
uma ação que o servidor não tem.** O servidor é a fonte.

**Reconciliar:** o `match_full_state` passa a mandar ao dono a lista das ações **dele** que
ainda estão na fila — só IDs e o que ele mesmo declarou, **sem velocidade** (ver B1) —, e o front
descarta o que o servidor não tem, avisando o jogador.

**Persistir a fila não entra.** Ela parece resolver, mas diverge por dentro: as barras são
cobradas no enqueue e vivem só em memória — saldo, carry-over, velocidades de quem já agiu — e
voltam zeradas num reinício (`NewMatchSessionWithState`). Uma fila persistida sobre barras
zeradas é um estado que nunca existiu. Persistir com coerência exigiria persistir a economia do
round inteira, que é muito maior que este item. Reconciliar cabe no §0.2: **perder pode,
divergir não.**

⚠️ **O cliente não reenvia sozinho.** Reenviar sorteia de novo: seria uma re-rolagem que o
jogador não escolheu, e *o mestre nunca re-rola o dado de um jogador*. Se a ação se perdeu, o
jogador fica sabendo e declara de novo — com o rascunho de volta, para não refazer tudo.

**B13 — Todo escape espera o fechamento, e a falha é do mestre.** Regra do dono do produto:
**o escape não desloca a peça na abertura, porque ele pode falhar.** Na abertura, mostra-se para
onde o personagem quer ir; no fechamento, decide-se.

⚠️ **Esta é uma versão de rascunho.** As colisões e as interações entre ações ainda vão ser
desenhadas "pra valer" pelo dono do produto. O que está abaixo fecha o fluxo das ações sem
antecipar esse desenho.

**O escape é uma esquiva que se desloca**, e o teste de movimento está **aninhado** nela:

- O teste de movimento é o **Accelerate** rolado, se for Dash, ou o **valor base** do Brake, se
  for Shift.
- **Para escapar, o personagem precisa passar nos dois**: no movimento e na esquiva.
- Hoje o motor decide a esquiva (`Avoided = Dodge.Total >= HitTotal`, `reaction_collision.go`) e,
  separado, move a peça do Dash por `Move.FinalSpeed` contra o acerto (`applyClosedEscapes`,
  `room.go`) — duas respostas que podem discordar. **Junte as duas numa só**: escapou = passou
  nos dois.
- ⚠️ **Isso muda o dano, não só a peça.** Nas palavras do dono do produto: *"o usuário não
  consegue esquivar se falhar no movimento"*. Então:
  - o `Avoided` de um escape passa a exigir **esquiva e movimento**;
  - passou na esquiva e falhou no movimento → **toma o golpe**;
  - no `escapeGuard`, nesse caso, **cai para a defesa** — é a rede de segurança que ele mantém.
- O contrato hoje diz o contrário — *"Dano e deslocamento são desfechos INDEPENDENTES"*
  (`match-combat-ws.md`, seção do `open_reaction`). **Corrija o contrato.**

> **Regra conhecida, não implementar agora:** na prática o movimento **se soma à esquiva**, e por
> isso escapar com movimento tende a ser mais fácil que esquivar parado. O motor hoje **não**
> soma (`Dodge.Total` não leva o movimento). Isso é parte do desenho de colisão que ainda vai
> existir — registre como pendência, com comentário no código onde a soma entraria.

**O que acontece com a peça:**

| | A peça |
|---|---|
| **Passou nos dois** | vai para o destino, no fechamento |
| **Falhou em algum** — inclusive o escape defensivo que falhou na esquiva e segurou na defesa | **o mestre decide a posição final** |

Na falha, o personagem pode ter se deslocado ou não, pode ter caído em outro slot — e **como essa
interação vai funcionar ainda não está decidido.** Por isso, no rascunho:

- a resolução **marca** o escape que falhou como "posição final a critério do mestre";
- **o mestre escolhe o slot final como parte da resolução daquele turno** — é uma decisão sobre
  a colisão que está sendo resolvida, e fica registrada com o turno (resolução e histórico);
- no fechamento, a peça vai para onde o mestre escolheu. Se ele não escolheu, fica onde estava.

⚠️ **Isto não é uma master action**, e não é o arrastar de B14. O arrastar é o mestre agindo
fora do fluxo; aqui ele está **resolvendo o turno** — o mesmo lugar onde, na Fase 8, ele vai
editar rolagens. Não implemente como um arrastar: implemente como parte da resolução.

Deixe **comentário no código**, no ramo da falha, apontando para esta seção: é ali que o desenho
definitivo vai entrar.

> **Se a intervenção do mestre crescer demais** para este pacote, o fallback do dono do produto
> vale: a peça fica onde estava, a intervenção vira pendência registrada, e o comentário no
> código fica. Diga no spec qual caminho tomou e por quê.

**B14 — O servidor é o dono do tabuleiro.** Hoje ele não é, de duas formas:

- **As paredes e as peças chegam ao servidor pelo navegador do mestre.** O `cmd/game` não tem
  repositório de mapa e não lê o mapa do banco: quem escreve o tabuleiro na sala é o
  `map_state_sync`, que só o mestre manda (`room.go`, `case MsgTypeMapStateSync`). Se o
  navegador do mestre tem uma cópia velha, ou ainda não mandou, **o servidor não conhece a
  parede** — e revelar ou interagir com ela é ignorado em silêncio (contrato,
  `enqueue_master_action`). É também por isso que o tabuleiro "ressincroniza com o REST velho"
  (B3).
- **Qualquer cliente move qualquer peça.** O `piece_moved` não tem validação no servidor — o
  código diz que *"o cliente restringe o arraste"* e tem um `TODO` para validar
  (`room.go`, `case MsgTypePieceMoved`).

**O conserto é inverter o fluxo.** O servidor carrega o tabuleiro **sozinho**, do banco, quando a
sala nasce: as paredes e as peças do mapa da partida, com as posições da partida por cima (B3).
O tabuleiro passa a correr **só do servidor para o cliente** (`map_full_state`, que já existe).
O `map_state_sync` deixa de escrever qualquer coisa no servidor.

> É a regra do §0.2 aplicada ao tabuleiro. O dono do produto: *"o servidor precisa conhecer
> tudo. Ele é a fonte da verdade. O client deve renderizar o que vem da fonte da verdade."* Um
> erro de "parede desconhecida" no meio da partida tira do mestre o poder de conduzir a cena —
> **o desenho não deve permitir que ele aconteça.**

**Arrastar peça durante a partida é uma master action.** Normalmente o mestre move um NPC por uma
ação comum, como qualquer jogador; o arrastar é o poder dele de mover **fora** desse fluxo. Passa
por `enqueue_master_action` com `move` (por isso depende do `move` de B9): só o mestre, validado,
persistido (B3) e transmitido com projeção de fog. **Jogador nunca arrasta** na partida: ele move
por ação.

- **É uma master action de verdade** — mesma mensagem (`enqueue_master_action`) e, com turno
  aberto, o mesmo registro no turno. **Toda** master action vira linha no histórico, não só a
  que acontece com turno aberto — ver "Master actions persistidas", logo abaixo.
- **Vale com ou sem turno aberto.** Hoje `EnqueueMasterAction` recusa sem turno aberto
  (`match_session.go`, `ErrNoActiveTurn`) e só pendura a master action no turno corrente — e
  arrastar sem turno é o caso comum: o mestre arrumando a cena entre turnos, ou antes do
  primeiro. A master action de mover **se aplica na hora**. Com turno aberto, fica registrada no
  turno; sem turno, entra no histórico como evento fora de turno — o mesmo mecanismo de B15, que
  já passa a guardar troca de cena, de regime e round fechado.
- A mesa recebe `piece_moved` com projeção de fog; o mestre recebe `master_action_enqueued` como
  confirmação.
- **O `piece_moved` do cliente fica só no lobby, e validado no servidor:** o mestre move qualquer
  peça; o jogador, só as dele. Com a partida rolando, ele deixa de mover peça.

> Não confunda com a posição final de um escape que falhou (B13) — aquilo é resolução de turno,
> não master action.

**Master actions persistidas — decisão do dono do produto.** Hoje as master actions vivem só em
memória, penduradas no `Turn`: o `PersistTurnClose` nunca gravou `t.GetMasterActions()`. Elas
passam a ser persistidas, e **em tabela própria — não em `actions`**. As duas aparecem no
histórico.

Por que não na mesma tabela — é o modelo, não preferência:

- `actions.actor_uuid` referencia `character_sheets` (PR #69). O ator de uma master action é o
  **mestre**, que é usuário, não ficha.
- `actions.turn_uuid` é `NOT NULL`. A master action pode acontecer **fora de turno** — o
  arrastar entre turnos é o caso comum.
- Misturar obriga a afrouxar as duas colunas e a filtrar um tipo do outro em toda leitura do
  histórico.

O que a tabela precisa (o formato é da sessão de back): a partida, o turno **opcional** (nulo =
fora de turno), o mestre (`users`), o tipo, o conteúdo (move, interact, …) e o instante.
Persista nos mesmos momentos de B3/B15.

- **No histórico:** a master action com turno entra **dentro daquele turno**; a sem turno entra
  como **evento fora de turno**, pelo mesmo mecanismo de B15, na ordem do tempo.
- **Projeção:** cada leitor vê a master action **como a viu ao vivo** — o que o fog escondeu dele
  na hora não aparece para ele depois.
- **`overridden_action_values` continua separado de tudo isso:** a edição do mestre
  (`edit_action`) não é master action.

Mantenha, como **última defesa**, a resposta com `error` para uma ação sobre parede que o
servidor não conhece. Com o servidor carregando o tabuleiro, esse caminho deveria ficar
inalcançável; se for alcançado, é bug, e o mestre fica sabendo em vez de ser enganado.

**B16 — Uma partida começa de onde outra terminou.** Ao anexar um mapa a uma partida nova, o mestre
pode escolher **herdar o tabuleiro de outra partida** da mesma campanha e no mesmo mapa: as
posições, o estado das paredes e o fog explorado com que ela terminou viram o começo da nova.
Com B3 salvando o tabuleiro por partida, isso é **copiar o tabuleiro de uma partida para outra**.
É o que dá continuidade entre sessões: *uma sessão começa onde a anterior parou.*

⭐ **O desenho de B3 tem que tornar essa cópia trivial** — é o critério para escolher a estrutura
em que o tabuleiro fica salvo.

**B15 — O histórico guarda o que não é turno.** O front precisa reconstruir, depois de
recarregar, três linhas que hoje só existem ao vivo — e o REST não guarda:

- **a troca de regime**: ela acontece **dentro** do round em andamento (`Round.SetMode`), sem
  abrir outro, e o REST guarda só o regime final de cada round;
- **a cena e o round sem nenhum turno**: a resposta pode vir `{ "scenes": [] }` numa partida
  sem turno fechado, então uma troca de cena ou um round fechado por exaustão se perdem;
- **o round fechado** em si.

Persista esses três eventos e devolva-os no `GET /matches/{uuid}/history`, na posição certa da
árvore. *"Nada deve ser perdido durante a partida"* vale para eles também.

E, no mesmo PR, **conserte o `match-history.md`**: os exemplos usam `"category": "combat"` e
`"mode": "combat"`, que não existem (o enum de cena é `battle`/`roleplay`; o de regime,
`Free`/`Race`), e o formato de `move` é só citado, nunca mostrado.

### 6A.6 Pacote de front — um PR, repo `System_X_System_React`

Começam já: F2, F3, F5, F7, F8, F11, e **a parte de F4 e de F6 que não depende do back**
(marcada em cada um). **Esperam o back:** F1 (B1), F10 (B12), F12, F13 e F16 (B14), F14 (B13),
F15 (B16), o resto de F4 (B15) e o resto de F6 (B1). F2 funciona inteiro só depois de B11, mas pode ser construído antes.

> **F9 saiu.** Ele pedia tratar `master_action_enqueued`, mas essa mensagem só existe quando o
> mestre manda `enqueue_master_action` **sem** `interact` — e o front nunca manda assim; o único
> envio é o menu de parede, sempre com `interact`, que pelo contrato volta sem ack. O `error`
> desse envio já aparece na tela. A recusa que sumia (A3) é do **servidor**, e virou B14.

**F1 — A fila do mestre mostra a ação inteira** *(espera B1)*. **E o mapa mostra o fantasma** de
cada ação da fila com movimento, lendo o destino que B1 passa a trazer, com o **mesmo desenho** que
o dono já vê para a própria declaração (§10.2) — não um segundo mecanismo. O card abre em detalhe: atacante,
alvos por nome, arma, movimento (categoria e destino), perícias, `actionSpeed` (perícia, dados,
total), `moveSpeed` (Accelerate ou Brake, dados, total), a chave na ordem geral, as barras que
cobra. Recolhido, fica como hoje; o detalhe é um toque.

**F2 — O mestre age por qualquer NPC da partida.** Com B11, todo NPC do mapa é participante, e o
`NpcPicker` os lista. Busque os participantes de novo quando chegar `npc_added` — e também quando
aparecer no tabuleiro uma peça de personagem que não é participante, como rede de segurança. A mensagem "não está inscrito" deixa de ter caso. Continua valendo pôr na
partida um NPC da campanha que **não** está no mapa, pelo `add_npc` (ao chegar `npc_added`, ele
vira participante e fica selecionável).

**F3 — A ficha abre dentro da partida.** Um quarto `SheetMode` do `CharacterSheetTemplate`
(§5.6), **somente leitura**, na zona `panel` — coluna no desktop, bottom sheet no celular (a
mesma em que a ação abre). **Ficha** é um item do rail, junto de Ação (e de Inventário e Nen,
quando existirem): o jogador abre a própria por ele; o mestre abre qualquer uma pelo card, e o
card também ativa o item Ficha — a ficha abre **ali, no lugar de Ação/Fila**. A aba da direita
(`aside`) não muda.
**Nenhum `navigate`.** O HP mostrado é o ao vivo (`character_hp_changed`), não o do REST.

**O painel alarga para a ficha.** Hoje ele tem largura fixa no desktop e a ficha não cabe. Com
a ficha aberta, a coluna do painel fica mais larga; com Ação/Fila, volta ao normal. Na bottom
sheet a largura já é a da tela.

**F4 — O histórico vem do servidor.** Buscado de `GET /matches/{uuid}/history` ao montar e
**rebuscado a cada `turn_closed`** — o WS avisa, o REST busca. O que só existe ao vivo (turno
aberto, HP mudou) entra por cima, pelo WS, e sai quando o REST equivalente chega. O servidor é a
fonte; guardar o histórico no navegador seria uma segunda verdade — exatamente o tipo de
divergência de R8. **Divide-se em dois.** Os **turnos fechados** saem do REST que já existe — começa já. As linhas
de **troca de cena, troca de regime e round fechado** não dão para deduzir do REST de hoje (a
troca de regime acontece dentro do round e o REST guarda só o regime final; cena e round sem
turno não aparecem) — elas esperam **B15**, que as persiste. Até lá, elas existem só ao vivo.
As **master actions** (B14) entram na mesma leva: dentro do turno quando têm turno, como evento
fora de turno quando não têm — e, como `master_action_enqueued` vai à mesa inteira, ele também
rebusca o histórico.

**F5 — Os cards usam o dado público.** `MatchCharactersSidebar` **sempre** renderiza o
`CharacterSidebarItem`, montando o `character` pela parte base e mesclando `private` quando
existe. O `BasicParticipantItem` sai — o card já trata ausência de vida e de experiência. Os dois
formatos que chegam (`{...base, private}` dos participantes e o plano `CharacterPrivateSummary`)
passam por um adaptador só, antes do card.

**F6 — As barras de cada personagem, no topo do canvas.** Uma faixa com cada personagem: duas
barras (ação e movimento), o saldo em número, as velocidades que agiram e a média. A escala é o
**preço do round** — o saldo nunca passa do preço, pelo teto do carry-over. A ordem projetada
continua, com a chave de cada slot.

**Quem vê o quê é revelado em sequência**, conforme o mestre abre as ações — regra do dono do
produto:

| | Mestre | Jogador |
|---|---|---|
| velocidades de ações **já abertas** — de todos | ✔ | ✔, à medida que o mestre abre |
| velocidades de ações **ainda na fila** — de todos | ✔ (B1) | ✗ — **nem as da própria ação** |
| saldo e carry-over de todos | ✔ | ✔ |

O jogador **não sabe o valor que a ação dele gerou até o mestre abri-la**. O `bars_updated` já
se comporta assim — ele só carrega velocidades que agiram —, então o jogador recebe exatamente o
que pode ver. **Não mostre ao jogador nenhuma velocidade vinda de outra fonte.** Em telas
estreitas a faixa recolhe para a ordem geral e expande num toque.

**Divide-se em dois.** A faixa pública — saldo, velocidades que já agiram, média, ordem —
**começa já**. As velocidades **da fila** são do mestre e chegam por B1: entram junto com F1, no
card da fila, no fim do plano.

**No regime Livre não há barra.** Sem preço, média nem carry-over (contrato,
`change_round_mode`), não existe escala. Mostre só as velocidades que agiram e a ordem.

**F7 — O cálculo do turno aberto, no card da ação.** Enquanto o turno está aberto, o mestre vê o
que já recebe: o acerto (dados e total) e, por alvo, esquiva, defesa, tipo de reação, escada do
repelir, dano projetado e payouts, mais as reações anexadas e não abertas.

**Onde — desenho do dono do produto:** **anexado ao card da própria ação, na fila** — não é um
item novo do rail. A ação aberta **continua na fila**, no topo, marcada como em andamento, com o
cálculo embaixo. (Hoje, ao abrir, ela some da fila.) O desenho ainda vai ser refinado.

> **Por que ele nasce só de leitura:** os botões que agem sobre ele pertencem a fases que ainda
> não chegaram — **dar a palavra** a uma reação é da Fase 7, **editar** é da Fase 8. Pôr o botão
> antes da fase seria um controle que não faz nada. O painel **fica completo ao fim da Fase 8**,
> e cada fase acrescenta o seu.

**F8 — Trocar de cena.** Na topbar do mestre, categoria e descrição inicial → `change_scene`. A
categoria é validada no servidor desde o PR #74: mande o valor do enum, minúsculo.


**F12 — O mestre arrasta, põe e tira peças: master action.** Arrastar a peça na tela do mestre
manda a master action de B14. **Pôr uma peça** de um personagem que ainda não está no tabuleiro
usa a mesma master action — é como um NPC que entrou pelo `add_npc` chega ao mapa (F2).
**Tirar uma peça** também, para qualquer peça (B11).

**O mestre confirma antes de mandar.** Soltar a peça no destino não envia nada: aparece a
confirmação, e só ela manda a master action; cancelar devolve a peça. O mesmo para pôr e tirar.
É resiliência a clique errado — uma master action muda a partida para todo mundo e fica no
histórico. O jogador **nunca** arrasta peça na partida: ele move por ação. *(Espera
B14 e o `move` de B9.)*

**F14 — O mestre escolhe onde cai o escape que falhou.** Quando a resolução marca um escape que
falhou (B13), o cálculo no card da ação (F7) destaca o caso, e o mestre escolhe o slot final — no
card ou tocando no mapa. É parte da resolução do turno, **não** o arrastar de F12: são dois
gestos diferentes para duas coisas diferentes. *(Espera B13.)*

**F15 — Começar uma partida de onde outra terminou.** Na tela em que o mestre anexa o mapa a uma
partida, a opção de herdar o tabuleiro de outra partida da mesma campanha e no mesmo mapa (B16).
*(Espera B16.)*

**F13 — O front para de escrever o tabuleiro no servidor.** Hoje a página do mestre manda o
tabuleiro ao servidor a cada conexão (`map_state_sync`). Com B14, o servidor carrega o tabuleiro
sozinho, e esse envio sai. O front passa a só desenhar o que vem em `map_full_state`.
*(Espera B14.)*

**F16 — Iniciar a partida não grava mais no mapa da campanha.** Sai o
`mapsService.updateMap(... { pieces: lobbyPieces })` de `LobbyPage.tsx`. O tabuleiro do lobby é o
tabuleiro da partida, e quem o guarda é o servidor (B3, B14). *(Espera B14.)*

**F10 — A lista de declaradas segue o servidor** *(espera B12)*. Na conexão e em toda reconexão,
a lista do jogador é reconciliada com o que o servidor diz que existe. Ação que o servidor não
tem **sai da lista, com aviso**, e o rascunho dela volta para o composer. **Nunca reenvie
sozinho** (B12).

**F11 — Selecionar uma peça não troca a aba da direita.** O `aside` fica onde o usuário o deixou.

### 6A.7 Ordem e PRs

```
Back   ──  B1 · B2 · B3–B16   ─────────────────────┐
                                                   ├──►  F1 · F10 · F12–F16 · resto de F4 e F6  ──►  Fase 7
Front  ──  F2 · F3 · F5 · F7 · F8 · F11 · parte de F4 e F6 ┘
```

- Back e front **em paralelo**: repos diferentes, nenhum arquivo em comum.
- No plano do back, **B14, B1, B2, B11 e B12 vêm primeiro** — são os que destravam o front. B14
  vem antes de todos: B3 e B11 dependem de o servidor ser dono do tabuleiro. **B16 vem por
  último**, e o desenho de B3 tem que ser escolhido pensando nele.
- No plano do front, **F1, F10, F12 a F16 e as partes de F4 e F6 que dependem do back vêm por
  último**,
  depois do merge do back.
- A Fase 7 espera **B2, B13 e** o PR de front deste fechamento (os dois tocam o painel do
  mestre).

### 6A.8 Verificação

**No browser, com três contas**, um mestre e dois jogadores (§0.1). **O NPC entra pelo caminho
real**: posto no mapa pelo mestre, numa partida criada do zero — nada de inscrever por fora.

**A ordem, conferida na tela.** Três personagens rolando 20, 23 e 11 (o exemplo canônico de
`barra-de-acao.md`): a ordem tem que sair **p2, p1, p3, p2**, e com F6 no ar o mestre confere
pelos números.

**Recarregar a página** no meio de um turno, nas três telas: histórico, barras, fila, turno
aberto e reações pendentes voltam. **Reiniciar o servidor** no meio de uma fila: nenhum cliente
fica mostrando ação que o servidor não tem. As posições das peças voltam de onde estavam — só
depois de B3.

### 6A.9 Decisões desta auditoria

| Decisão | Origem |
|---|---|
| NPC no mapa da partida é NPC da partida (B11) | dono do produto |
| Jogador não vê a velocidade da própria ação até ela abrir; a barra se revela em sequência (F6) | dono do produto |
| Todo escape espera o fechamento; escapar = passar no movimento **e** na esquiva; na falha, o mestre decide a posição final (B13) | dono do produto — rascunho, até o desenho de colisão |
| O servidor carrega o tabuleiro do banco; o navegador do mestre deixa de escrevê-lo (B14) | dono do produto — "o servidor precisa conhecer tudo" |
| Arrastar peça é master action; a posição final de um escape que falhou é resolução de turno, não master action (B13, B14) | dono do produto |
| A ficha na partida é só leitura (F3) | dono do produto |
| Edição do mestre continua na Fase 8 | dono do produto |
| Histórico vem do servidor, não do navegador (F4) | auditoria — o dono delegou o desenho |
| O cliente nunca reenvia ação perdida sozinho (B12) | auditoria — protege "o mestre nunca re-rola" |
| O cálculo do turno aberto nasce só leitura (F7) | auditoria — sem controle que ainda não funciona |
| O cálculo do turno aberto fica no card da ação, na fila; a ação aberta continua na fila, em andamento (F7) | dono do produto |
| A ficha abre no painel, como item do rail (o card também a abre ali); o painel alarga para ela (F3) | dono do produto |
| Ao abrir, a mesa — e o dono — vê a mecânica e as velocidades; acerto, dano e perícias só no fechamento (B2) | regra existente de `acoes.md`, aplicada ao dono |
| A fila é reconciliada, não persistida (B12) | sessão de back — persistir divergiria das barras |
| O tabuleiro — posições, paredes e fog explorado — é salvo por partida, no fechamento de turno, e não no mapa da campanha (B3) | dono do produto |
| Uma partida pode começar de onde outra terminou (B16) | dono do produto |
| O tabuleiro do lobby é o da partida; iniciar a partida não grava no mapa da campanha; o mapa anexado não troca depois do início (B3, F16) | decorre da decisão acima |
| Falhar no movimento do escape faz tomar o golpe; o escape defensivo cai para a defesa (B13) | dono do produto — "não consegue esquivar se falhar no movimento" |
| Arrastar, pôr e tirar peça são master actions: valem com ou sem turno aberto, sempre entram no histórico, e o mestre confirma antes de enviar (B11, B14, F12) | dono do produto |
| O `attack` sai da master action; o mestre ataca pelo NPC, com `enqueue_action`; efeito de ambiente é futuro (B9) | dono do produto |
| Master actions são persistidas em tabela própria, separada de `actions`; as duas aparecem no histórico, e cada leitor vê a master action como a viu ao vivo; `edit_action` não é master action (B14) | dono do produto — o modelo: `actions.actor_uuid` é ficha e `actions.turn_uuid` é obrigatório |
| Tirar peça vale para qualquer peça, e não desinscreve, não mata, não apaga histórico (B11) | dono do produto |
| O mapa anexado não troca depois do início; mudar o mapa no meio da partida é do futuro editor de mapa da partida, e B3 não pode fechar essa porta | dono do produto |
| Troca de regime, troca de cena e round fechado passam a ser persistidos (B15) | "nada deve ser perdido durante a partida" |

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
- **Os botões aparecem para o alvo porque `turn_opened` passa a dizer quem é alvo** (B2, §6A.5).
  Sem isso não há como desenhá-los. Depois de enviar, o alvo vê "reação enviada, aguardando o
  mestre".
- **Dar a palavra** às reações pendentes entra no cálculo do turno aberto, no card da ação na fila do mestre (F7, §6A.6), que
  já as lista.
- **O fantasma de espera** (§10.2): **todo escape** espera o fechamento para mover a peça — ele
  pode falhar (B13, §6A.5). Na abertura, a peça mostra para onde quer ir; no fechamento, vai
  para onde o servidor mandar.

  ⭐ **Reuse o que o movimento puro já tem.** A revisão da Fase 6 construiu o `IntentLayer` (seta
  e marcador de destino), a lista de declaradas e o destaque de slot. O fantasma do escape é o
  mesmo desenho com outra vida — não um segundo mecanismo. Desenhe esta parte com cuidado: é a
  que mais facilmente vira duas implementações paralelas do mesmo gesto.

**Depende de:** B2 e B13 (§6A.5), e do PR de front do fechamento da Fase 6.

**Pronto quando:** três alvos reagem diferente ao mesmo ataque em área, e abrir as reactions em
ordem inversa produz resultado diferente na tela.

## 8. Fase 8 — Regência

**Objetivo:** o mestre com todas as ferramentas.

**Escopo:**
- Edição do mestre: `edit_action` / `action_edited` — rolagem e perícias. É aqui que entra o
  seletor de perícia de dano, `Push` → `Grab` (§4.6).
- Os botões de **editar** no cálculo do turno aberto, no card da ação (F7, §6A.6). Com eles, o cálculo fica
  completo.

> O histórico, a ficha dentro da partida e `change_scene` eram desta fase e **subiram para o
> fechamento da Fase 6** (F4, F3, F8): o dono do produto os quer antes, e nenhum depende da
> edição.

## 9. Inventário e Nen — não planejados

Não existem no backend (`aura?: StatusBar` está tipado como opcional exatamente porque o servidor
não serializa) e **não são o próximo passo** depois da Fase 8 — o próximo é enriquecer a
mecânica de combate (§0). Os itens **Inventário** e **Nen** do rail continuam fora dele até
existirem.

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
| **Intenção declarada** | o envio da action e a abertura dela | o dono — e **o mestre, depois de B1** (§6A.5), que faz a fila dele trazer o destino. Antes de B1, só o dono | **6** |
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

**E quando a peça do escape se move** não depende da categoria: **todo escape espera o
fechamento**, porque pode falhar (B13, §6A.5). Uma versão anterior deste documento tratava "o
Dash rola dado" como "o Dash tem CD" e fazia o escape com Dash esperar e o com Shift não. Errado:
o critério é o escape poder falhar.

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
- **NPC na partida: dois caminhos, e um bug.** Pôr NPC com a sala viva **existe** — `add_npc`
  por WS, que responde `npc_added` à mesa; o REST (`POST /matches/{uuid}/npcs`) monta o roster
  antes de a sala nascer. O bug é que **pôr uma peça de NPC no mapa não inscreve o NPC** — B11,
  §6A.5.
