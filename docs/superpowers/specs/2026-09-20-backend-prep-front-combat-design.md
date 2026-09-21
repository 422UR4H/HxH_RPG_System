# Preparação do backend para o combate no front — design

> **Escopo:** o §4 de
> [`2026-09-20-front-combat-phases.md`](2026-09-20-front-combat-phases.md), inteiro.
> **Um PR, no repo Go.** Não é uma fase: é o que a Fase 6 precisa encontrar pronto.
>
> O rostering de NPC **não está aqui** — é fatia própria, em paralelo (decisão do dono do
> produto, 2026-09-20). O repo React **não é tocado** por este PR; o porquê está no §11.

## 1. O que este documento assume que você não sabe

Você vai implementar isto sem a conversa que o gerou. Então, o mínimo:

- **Turno** = uma action **e** suas reactions. **Round** = sequência de turnos. **Cena** =
  sequência de rounds.
- `actorId` é sempre o **sheetUUID** do personagem, nunca o UUID do jogador. A autorização é
  `charToPlayer[actorID] == playerUUID`.
- **Os dados caem quando a action chega** (`rollActionDice`, em `EnqueueAction`/
  `AttachReaction`). Nada rola de novo depois — é isso que deixa o mestre editar e as reações
  colidirem sem re-sortear o dado de ninguém.
- **`room.go` é o dono do lock.** `MatchSession` não tem lock próprio: quem lê ou escreve
  estado de sessão segura `r.mu` antes. Todo `Execute` novo herda essa obrigação.
- **Dois eixos de visibilidade**, e os dois valem aqui: *tempo* (`IsSettled` — turno aberto é
  master-only) e *classe* (mestre / dono / resto). `dispatchPerPlayer` é o mecanismo, e não
  deve nascer um segundo.
- Wire em **camelCase** dos dois lados. Exceção conhecida: valores de enum de domínio
  (`rung`, `applies`, `expiresAt`, `againstKind`) viajam em snake_case porque são
  `String()` de enum, não tags de struct.

O contrato que este PR altera é [`../../dev/api/match-combat-ws.md`](../../dev/api/match-combat-ws.md).
Ele é **entregável deste PR**, não subproduto: a Fase 6 implementa contra ele e não contra o Go.

## 2. As sete tarefas, e por que nesta ordem

| # | Tarefa | Arquivos principais |
|---|---|---|
| 1 | `action_enqueued` devolve o `actionId` | `internal/app/game/message.go`, `room.go` |
| 2 | `Push` entra no `RawDamage` | `internal/domain/match/service/damage.go`, `turn_resolver.go` |
| 3 | Finta: visibilidade temporal, não por classe | `internal/domain/match/service/projection.go`, `internal/application/match/get_match_history.go` |
| 4 | Categoria de movimento dos escapes validada no servidor | `internal/app/game/action_mapper.go` |
| 5 | Catálogo de combate por personagem (REST novo) | `internal/app/api/sheet/` |
| 6 | `match_full_state` | `internal/app/game/message.go`, `room.go` |
| 7 | O movimento aplicado ao tabuleiro | `internal/app/game/room.go` |

Os quatro pequenos vêm primeiro de propósito: eles tocam `message.go` e `room.go`, que as
tarefas 6 e 7 reescrevem em volume. Um `action_enqueued` já consertado e verde é evidência
barata de que o loop de mensagem continua de pé enquanto o resto se move.

As tarefas 1–5 são independentes entre si. A 6 e a 7 são independentes uma da outra.

---

## 3. Tarefa 1 — `action_enqueued` devolve o `actionId`

**O problema.** `room.go:746` responde `NewServerMessage(MsgTypeActionEnqueued, struct{}{})`.
O navegador do jogador não tem como se referir à ação que acabou de mandar: não consegue
cancelá-la, nem destacá-la na barra geral, nem saber que a próxima da fila é a dele.
`acoes.md` diz que uma ação pode ser cancelada — e hoje ela é **inendereçável**.

É o mesmo buraco que `PendingReactions` fechou para o mestre na Fase 4: *uma operação cujo ID
o cliente não recebe é uma operação que o cliente não consegue invocar.*

**O conserto.** Payload novo em `message.go`, ao lado de `ActionQueuedPayload`:

```go
// ActionEnqueuedPayload acks the sender's own enqueue AND names the action.
//
// The name is not decoration and it is not a leak: this goes only to the player who sent the
// action, about their own action. The queue stays secret — what the table cannot learn is
// what OTHER people queued, and action_queued (master-only) is still the only surface that
// names someone else's.
type ActionEnqueuedPayload struct {
	ActionID uuid.UUID `json:"actionId"`
}
```

`room.go:746` passa a mandar `ActionEnqueuedPayload{ActionID: a.GetID()}`. O
`action_queued` do mestre **não muda**.

> O verbo de cancelar em si **não é deste PR** — é da Fase 6. Aqui entrega-se só o ID, que é o
> que falta para ele ser possível.

**Verificação:** e2e no pacote `game` — um cliente jogador manda `enqueue_action` e recebe um
`action_enqueued` cujo `actionId` é não-zero e **igual** ao `actionId` que o mestre recebeu no
`action_queued` do mesmo envio. Essa igualdade é o teste; um ID qualquer não serve.

---

## 4. Tarefa 2 — `Push` entra no `RawDamage`

**A regra.** O dano é medido por **Push** (`enum.Push`, `skill_name.go:15` — já existe, junto
com `Grab`). Hoje `RawDamage(dados, arma, catálogo)` não recebe perícia nenhuma: o total é só
os dados da arma mais o dano plano dela.

**Sem seletor.** O jogador não escolhe isso; a arma é que dá o dano. Trocar `Push` por `Grab`
(ou outra) é prerrogativa do mestre e entra na superfície de edição dele, na **Fase 8**.

**O conserto.** `damage.go:37` ganha um parâmetro:

```go
func RawDamage(dice []int, name *enum.WeaponName, cat *item.WeaponsManager, push int) (int, error)
```

`push` soma ao total, junto com `w.GetDamage()` e os dados.

Dois call sites, **ambos** passam o `Push` do **ator**, lido por
`skillValueOf(actorSheet, enum.Push.String())` (o helper já existe em `turn_resolver.go:538`):

- `turn_resolver.go:386`, em `seedChain` — o ataque contra personagem;
- `turn_resolver.go:313` — o ataque contra parede.

**O dano é o mesmo dano**: não há razão para o punho medir diferente contra uma porta e contra
uma pessoa.

Se a ficha do ator não estiver em escopo em algum dos dois call sites, ela é alcançável do
mesmo jeito que em `turn_resolver.go:443`, que já faz
`skillValueOf(actorSheet, a.Attack.Hit.SkillName)` — o ator de uma resolução é sempre
conhecido, e a própria `ResolveInput.Sheets` o carrega. **Não passe zero para "resolver
depois"**: um `Push` silenciosamente zerado é indistinguível de um personagem fraco, e o teste
abaixo é o que impede isso de passar.

**Efeito no contrato:** `attack.damage.skillName` passa a ser **descartado**. Ele já era
decorativo — nada no resolver o lia; só `attack.hit.skillName` vira `skillValue`. Agora isso
fica escrito, na mesma linha em que `speed` já está documentado como descartado. O campo
continua sendo aceito (e validado contra `enum.SkillName`, como qualquer `RollCheckPayload`);
ele só não decide mais nada.

**Verificação:** teste de unidade em `service` com `Push` injetado — dois personagens com a
mesma arma e os mesmos dados produzem dano diferente porque o `Push` difere. Sem isso o teste
passaria com o parâmetro ignorado.

---

## 5. Tarefa 3 — a finta deixa de ser escondida para sempre

**O problema.** `ProjectAction` (`projection.go:96`) faz `out.Feint = nil` para todo não-dono.
O único chamador é o histórico REST (`get_match_history.go:107` e `:110`). Consequência: uma
finta fica escondida do alvo **meses depois**, num histórico de turno fechado.

**A regra correta é temporal, não por classe.** Se o alvo cai na finta, ele descobre **dentro
da resolução do mesmo turno**: o sucesso dele foi contra um ataque falso, e logo vem o real.
Nesse momento ele vê que caiu. Então: **turno aberto esconde, turno fechado revela** — o mesmo
eixo `IsSettled` que `ProjectResolution` já usa.

**O conserto.** `ProjectAction` ganha o eixo:

```go
// isSettled is the TIME axis, the same one ProjectResolution reads. A feint is secret while
// the turn is open and public once it closes: the target who fell for it finds out inside
// that same turn's resolution, because their success was against a false attack and the real
// one follows. Hiding it after the turn closed hides it forever, which is not the rule.
func ProjectAction(a action.Action, v Viewer, isSettled bool) action.Action
```

`out.Feint = nil` passa a acontecer **só quando `!isSettled`**.

O chamador do histórico passa **`tu.FinishedAt != nil`**, não `true`. É verdade que o
histórico só guarda turno fechado — `PersistTurnClose` é o único caminho de escrita —, mas ler
o fato custa o mesmo que assumi-lo e não quebra no dia em que um turno aberto atravessar.

**O que NÃO muda:**

- **`Trigger` continua escondido.** O §4.7 fala da finta. Não alargue a decisão por simetria —
  `action.Trigger` é objeto vazio hoje e a regra dele não foi escrita.
- **A remoção da entrada `Evasion` de `Skills` continua.** Ela não é a mesma regra: é a outra
  metade do segredo da esquiva fechada, amarrada ao rebaixamento do rótulo
  (`closedDodge` → `dodge`), e esse rebaixamento é permanente por desenho.

**Verificação:** teste em `service` — a mesma action com finta, projetada para um terceiro,
devolve `Feint == nil` com `isSettled=false` e `Feint != nil` com `isSettled=true`. Mais um
teste no histórico provando que um turno fechado entrega a finta a quem não é dono.

---

## 6. Tarefa 4 — a categoria de movimento dos escapes, validada no servidor

**A matriz, fechada** (§11.4 do documento mestre):

| Reação | Movimento |
|---|---|
| `escape` (padrão) | **Dash** |
| `escapeGuard` (defensivo padrão) | **Dash** |
| `closedEscape` (fechado) | **Shift** |

O discriminador é **fechado × aberto**, não defensivo × padrão.

**Por que Shift no fechado:** durante o Dash o personagem está "no ar" e não consegue esquivar
— é exatamente o que a fechada existe para não fazer. O Brake governa o Shift porque ele mede
a capacidade de **frear**; quem acelera além do que freia é rápido mas não é ágil.

**O problema.** `Displaces()` só exige que exista um `Move`, sem olhar a categoria. Se a regra
ficar só no front, **o cliente vira dono dela** — e um cliente que manda `closedEscape` com
Dash ganha o desconto de barra da fechada com a mobilidade da aberta.

**O conserto.** Em `action_mapper.go`, no mesmo bloco que hoje valida os componentes
obrigatórios por `ReactionKind` (o `switch` sobre `action.ComponentMove` / `ComponentRepel`,
por volta da linha 150): depois de confirmar que o `Move` existe, confirmar a categoria.

Mensagem de erro no estilo das vizinhas, que são frases e não códigos:

```
reaction "closedEscape" must move with Shift, not Dash
```

Volta como `invalid_action`, igual às outras recusas de reação.

**Verificação:** tabela em `action_mapper_test.go` — os três kinds × as duas categorias, seis
casos, três aceitos e três recusados.

---

## 7. Tarefa 5 — o catálogo de combate, por personagem

**O problema.** O contrato usa `"Strength"`, `"Deception"` e `"sword"` nos exemplos. Os três
seriam recusados: `Strength` é **atributo**, não perícia; a finta é `Feint`; e
`WeaponNameFrom` é **case-sensitive** (`"Sword"`). Não existe endpoint que entregue o
vocabulário, e é dessa ausência que nasceu o `combat_strength` que o front manda hoje.

**A decisão que muda o desenho** (dono do produto, 2026-09-20): o catálogo de armas de uma
action **não é o catálogo do sistema**. São as armas que **aquele personagem** sabe usar — as
proficiências dele —, mais o **golpe corporal**, sempre.

> Quando existir inventário, isto vira "as armas que o personagem carrega". Hoje não existe
> inventário, e proficiência é a melhor aproximação disponível: é o que ele sabe empunhar.

**A rota.** `GET /charactersheets/{uuid}/combat-catalogue` — `{uuid}`, não `{character_sheet_uuid}`: é o que o `GET /charactersheets/{uuid}` vizinho usa, e o handler dele já liga `path:"uuid"`

```json
{
  "weapons": [
    { "name": "Fist",  "dice": [6, 6, 4], "flatDamage": 0, "defenseBonus": 0, "proficiencyLevel": 0 },
    { "name": "Sword", "dice": [10, 4],   "flatDamage": 2, "defenseBonus": 3, "proficiencyLevel": 4 }
  ],
  "skills": ["Accelerate", "Accuracy", "Acrobatics", "Brake", "..."]
}
```

**De onde sai cada coisa:**

| Campo | Fonte |
|---|---|
| a lista de armas | `proficiency.Manager.GetWeapons()` da ficha — as proficiências **comuns** |
| `Fist`, sempre | acrescentado à lista, esteja ou não nas proficiências |
| `dice`, `flatDamage`, `defenseBonus` | `item.WeaponsManager`: `GetDice`, `GetDamage`, `GetDefense` |
| `proficiencyLevel` | `proficiency.Manager.GetLevelOf(name)` |
| `skills` | `enum.AllSkillNames()`, em `String()` |

**Por que o `Fist` entra sempre.** O golpe corporal não depende de treino, e o catálogo de
armas **já o trata como arma como qualquer outra**: `damage.go:11` define
`const unarmed = enum.Fist` exatamente para que mão vazia não seja um ramo que todo mundo a
jusante precise lembrar. Incluí-lo aqui é a mesma decisão, na superfície de leitura.

**Por que `skills` existe se a Fase 6 não tem seletor de perícia.** Ela não existe para virar
controle — o §11.1 proíbe seletor de perícia na Fase 6, porque a corrente de testes não é
executada e um controle que não muda nada é pior que controle nenhum. Ela existe para o front
**nunca mais inventar uma string**. É lista de validação, não de escolha.

**Visibilidade:** a mesma regra de quem pode ler aquela ficha — siga o que
`GetCharacterSheetHandler` já faz, não invente uma terceira política.

**Registro:** `huma.Register` em `internal/app/api/sheet/routes.go`, no padrão dos vizinhos,
com `Tags: []string{"character_sheets"}`.

**Verificação:** teste de handler HTTP real (`humatest`, como os vizinhos) — uma ficha com duas
proficiências devolve três armas, a terceira sendo `Fist`; uma ficha **com** `Fist` entre as
proficiências devolve `Fist` **uma vez só**, com o nível real. Mais um smoke `curl` contra o
servidor de pé.

---

## 8. Tarefa 6 — `match_full_state`

**O problema.** `map_full_state` cobre o mapa e só. Quem entra no meio — ou **reconecta**, e
`useMatchWs.ts` reconecta até cinco vezes sozinho — fica sem barras, sem regime, sem cena, sem
turno aberto e sem reações pendentes, até alguma coisa mudar por acaso.

**A mensagem.** Servidor→cliente, nova, **projetada** — não broadcast.

```json
{
  "type": "match_full_state",
  "payload": {
    "sceneId": "…", "sceneCategory": "Battle", "sceneBriefDescription": "Arena",
    "roundMode": "Race",
    "bars": { "seq": 7, "prices": {…}, "characters": [...], "order": [...] },
    "openTurn": { "turnId": "…", "actorId": "…" },
    "resolution": { "…": "master-only, isSettled:false" }
  }
}
```

**De onde sai cada coisa:**

| Campo | Fonte |
|---|---|
| cena | `session.GetActiveScene()` |
| `roundMode` | `session.GetActiveRound().GetMode()` |
| `bars` | **`newBarsUpdatedPayload(session)`, reusado inteiro** |
| `openTurn` | `session.CurrentTurnID()`; o ator, do turno corrente do round ativo |
| `resolution` | `session.ResolveTurn(t)` do turno aberto — **master-only** |

**Três invariantes, e cada uma tem um motivo que não é óbvio:**

1. **`bars` carrega o `seq` corrente, não um novo.** O cliente guarda o maior `seq` aplicado e
   descarta qualquer coisa menor. Se a reconexão entregasse um `seq` novo, a guarda do cliente
   atravessaria a reconexão zerada — e o primeiro `bars_updated` atrasado que chegasse depois
   seria aplicado por cima de um estado mais novo. Estampe `r.barsSeq` como está, sem
   incrementar.
2. **`resolution` e `openTurn.resolution` são master-only, pelo eixo do tempo.** O turno está
   aberto, logo o cálculo é do mestre. Um mestre que reconecta no meio de um turno aberto hoje
   perde o cálculo inteiro e não tem como recuperá-lo sem fechar o turno. As
   `pendingReactions` já viajam **dentro** da resolução (`newResolutionUpdatedPayload` as
   monta) — não construa uma segunda lista para elas.
3. **`ResolveTurn` é recomputo puro, não re-rolagem.** Os dados já caíram na chegada da action.
   Chamá-la de novo é seguro e é o que `attach_reaction` e `edit_action` já fazem.

**Onde é enviada.** No `Run()`, no braço `case client := <-r.register`, logo depois do
`map_full_state` — e **só quando `r.session != nil`**. No lobby não há sessão e não há nada a
sincronizar.

**Extraia a montagem do payload numa função própria** — `(playerID, isMaster) → *Message`, como
`buildMapFullState` já faz —, em vez de montá-la inline no braço do `register`. Isso vale por si:
é a forma que o vizinho já tem, e é o que torna a mensagem testável sem subir uma conexão.

> **Sobre o rostering de NPC, que roda em paralelo.** O §13 do documento mestre levanta a
> **possibilidade** de a propagação ao vivo de um NPC recém-adicionado pegar carona nesta
> mensagem, em vez de ganhar verbo de WS próprio. **Isso não é uma decisão tomada, e este PR
> não depende dela** — o prompt do rostering diz para ele não tocar em `message.go` nem em
> `room.go`, e um handler REST alcançando o `Room` seria uma chamada entre camadas que não
> existe hoje.
>
> O que este PR garante é modesto e suficiente: **se** o rostering precisar propagar ao vivo, a
> função já estará extraída e pronta para ser chamada de outro ponto. Ele pode perfeitamente
> terminar **sem propagação ao vivo nenhuma** — a tela de importar NPC é de uma fase posterior,
> e até lá um NPC adicionado aparece na próxima conexão como qualquer outro estado de partida.
>
> O que continua valendo sem depender de nada: **não encoste em `indexParticipants`**
> (`match_session.go`). É o único arquivo que os dois pacotes compartilham, e a função é **do
> rostering** — é lá que o NPC passa a ser mapeado no `charToPlayer`. Mexer nela daqui é o jeito
> mais rápido de criar conflito onde não precisava haver.

Use `dispatchPerPlayer`? **Não neste ponto**: o register é de um cliente só. Monte a mensagem
para aquele destinatário, com o mesmo par `(playerID, isMaster)` que o `buildMapFullState`
vizinho já recebe. O `dispatchPerPlayer` existe para quando a mesa inteira precisa de uma
cópia cada.

**Verificação:** e2e — um jogador e um mestre conectam com um turno já aberto; ambos recebem
`match_full_state`; só o do mestre traz `resolution`; o `seq` de ambos é igual ao do último
`bars_updated` que a mesa viu antes da conexão.

---

## 9. Tarefa 7 — o movimento aplicado ao tabuleiro

**O problema.** Nenhuma mensagem servidor→cliente aplica um movimento resolvido ao tabuleiro.
O `piece_moved` que existe é **cliente→servidor**, do lobby. O motor só usa `move.from` para
checar parede. **Hoje uma ação de mover acontece e a peça não sai do lugar** — e o fog não
recalcula, porque nada se moveu.

**A regra.**

| O movimento… | A peça |
|---|---|
| **não depende de teste** (shift/dash para slot livre) | desloca **na abertura** da action |
| **depende de uma CD** (salto, entrar em slot ocupado, passar colado) | **não desloca** |

**Por que na abertura e não no fechamento:** o dano espera o fechamento porque pode ser
editado; a posição não pode esperar, porque **as reactions seguintes dependem de onde a peça
está**.

⭐ **Invariante: o front nunca calcula onde a peça para.** Ele desenha o pedido e depois desenha
a posição que chegou do servidor — seja o slot pretendido, o meio do caminho, ou lugar nenhum.

**O ramo com CD não tem caso alcançável hoje.** `moveSpeedSkill` (`action_mapper.go`) aceita
**só `Dash` e `Shift`**, e recusa `Back`, `Roll`, `Slide`, `Jump`, `FlatJump` com
`move category "X" is not supported yet`. Nenhum dos dois aceitos rola contra CD. **Implemente
só o ramo sem teste** e registre o outro no contrato como inalcançável enquanto isso valer.
Não invente salto para preencher a tabela.

### 9.1 Reusar a máquina que já existe

`handlePieceMoved` (`room.go:1657`) já faz, nesta ordem: atualiza `r.pieces`, relaia por
jogador com o gate de fog, e recalcula a LOS de quem moveu. O gate é o miolo:

- quem enxerga o destino recebe **`piece_moved`**;
- quem só enxergava a origem recebe **`piece_removed`** (a peça saiu de vista);
- quem não enxerga nenhum dos dois **não recebe nada**;
- o mestre recebe sempre; peça `visible:false` nunca chega a jogador.

**Extraia esse miolo para um helper e chame-o dos dois caminhos.** Não escreva um segundo.

A única diferença do movimento do motor é que **não existe cliente remetente**. Hoje a função
pula quem mandou (`if pid == client.userUUID { return nil }`, porque o navegador dele já
aplicou o arrasto localmente) e recalcula a LOS *dele*. No caminho do motor esses dois papéis
passam a ser **o dono do personagem que se moveu** — e ele **não** deve ser pulado: o servidor
é a autoridade e o navegador dele não antecipou nada.

Assinatura sugerida para o helper, com os dois papéis explícitos em vez de um `*Client`:

```go
// origin is the player whose own browser already applied this move locally and must therefore
// not be echoed back to. It is uuid.Nil when the SERVER is the mover — nobody predicted that
// one, so nobody is skipped.
//
// The owner is not a parameter: it is derived from payload.CharacterID through
// session.GetCharToPlayer(), exactly as handlePieceMoved already derives `ownsPiece` today.
// Whoever owns the moved character gets a fresh map_full_state, because their line of sight
// just changed.
func (r *Room) applyAndRelayPieceMove(payload PieceMovedPayload, origin uuid.UUID)
```

### 9.2 O evento continua sendo `piece_moved`

**Não crie um tipo novo**, por três razões:

1. O par `piece_moved`/`piece_removed` é **indivisível**: o gate de fog precisa dos dois, e um
   tipo novo teria que duplicar esse par para não perder o caso "saiu de vista".
2. A instrução do pacote (`game-server.instructions.md`) manda **preferir menos tipos com
   payload mais rico**, e acrescentar tipo só quando a transição de domínio não for dedutível
   dos campos.
3. Para o front isso é **uma coisa só** — "chegou uma posição, desenhe". Ele nunca precisa
   saber quem mandou a peça para lá; é literalmente o invariante do ⭐ acima.

Se algum dia precisar distinguir, o envelope já distingue: `senderId` é
`00000000-0000-0000-0000-000000000000` em toda mensagem de servidor.

### 9.3 Onde engata

`OpenNextAction` e `PullAction` devolvem um `TurnTransition`
(`match_session.go:599`) que carrega `Opened *turn.Turn`. Em `room.go`, no braço que trata
cada uma: **se a action aberta tem `Move`**, aplicar o deslocamento, recalcular o fog e
transmitir — **antes** do `turn_opened` chegar à mesa, porque a mesa não pode ver o turno abrir
com a peça no lugar velho.

A peça é encontrada por `characterId == actorId` da action aberta. Se nenhuma peça no tabuleiro
casar com o ator, **não é erro**: é um personagem sem peça, e o movimento simplesmente não tem
o que mover. Não emita `error` por isso.

O `r.mu` já está segurado nesse trecho — confira antes de chamar o helper, que também toca
`r.pieces`, para não fechar um deadlock.

**Verificação:** e2e — mestre e jogador conectados, jogador enfileira um `Dash`, mestre abre,
e o jogador recebe `piece_moved` com o slot novo **antes** do `turn_opened`. Segundo teste, o
que de fato prova a projeção: um terceiro jogador que não enxerga nem a origem nem o destino
**não recebe mensagem nenhuma**.

---

## 10. O que NÃO entra neste PR

Registrado para ninguém implementar por engano nem achar que foi esquecido.

| Item | Onde vive |
|---|---|
| **Rostering de NPC** | fatia própria, **rodando em paralelo com este PR**. Não bloqueia o *início* da Fase 6 — o teste dela é jogador contra jogador —, mas precisa entrar antes de ela **fechar**: sem inimigo não é uma mesa, é uma prova de motor. Escreve por REST, não toca `message.go` nem `room.go`, e **não depende deste PR** (ver a nota no §8). O único arquivo compartilhado é `indexParticipants`, e ele é dele |
| **O verbo de cancelar action** | Fase 6 (o ID sai daqui, o verbo não) |
| **Tratar `error` no front** | Fase 6, primeira tarefa |
| **O conserto do `combat_strength`** | Fase 6 — ver §11 |
| **Seletor de perícia de dano (`Push` → `Grab`)** | Fase 8, na edição do mestre |
| **A corrente de testes** | regra escrita que o código não executa (§11.1 do doc mestre) |
| **A resolução da finta** | regra não desenhada (§11.2). Aqui se conserta só a *visibilidade* |
| **Posição intermediária, empilhamento, status de peça** | §10.3 e §10.4 do doc mestre |
| **`ReboundDamage`, armadura, `action.Initiative`, posturas** | §11.5 do doc mestre |
| **Consertos em `reacoes.md`** | Fase 7, junto com quem tocar no assunto (§12) |

## 11. Por que o repo React não é tocado

O §4.4 do documento mestre põe o conserto do `skillName` neste pacote. Ele foi escrito **antes**
da decisão de que o catálogo é por personagem, e essa decisão muda o conserto.

Com o catálogo por ficha, o conserto certo no front **não é trocar `"combat_strength"` por
outra string** — é consumir o endpoint novo e listar as armas daquele personagem. Isso é a
bottom sheet, que é o coração da Fase 6.

E trocar a string sem tratar `error` — que o §6 do doc mestre marca como a **primeira** tarefa
da Fase 6 — consertaria o sintoma conhecido e manteria a cegueira que produz os próximos: toda
recusa do servidor continua sumindo num `catch` vazio.

Então o backend entrega o endpoint e os exemplos corretos; o `combat_strength` morre na Fase 6,
no lugar certo, junto com a causa dele. **Diga isso na descrição do PR**, para não parecer
esquecimento.

## 12. Documentação — entregável, não subproduto

No mesmo PR:

**`docs/dev/api/match-combat-ws.md`:**
- `match_full_state` — mensagem nova, com os dois eixos de visibilidade explicados;
- `piece_moved` como mensagem **de servidor** também, com o gate de fog e o par com
  `piece_removed`;
- `action_enqueued` deixa de ser `{}`;
- a finta: visibilidade temporal;
- `attack.damage.skillName` **descartado**, na mesma tabela em que `speed` já está;
- `nickname` documentado como **opcional** — `handler.go:134` já cai para os 8 primeiros
  caracteres do UUID quando vem vazio. Não é código, é o contrato que estava errado;
- **os exemplos consertados**: `"Strength"` → `"Push"`, `"Deception"` → `"Feint"`,
  `"sword"` → `"Sword"`;
- a seção **"O que este contrato ainda não entrega"**: **conferir, não presumir.** As três
  lacunas que este PR fecha — nada move a peça, não há snapshot de combate, o
  `action_enqueued` é vazio — **nunca estiveram nessa lista**. Elas saíram da auditoria do
  front, não da autoconsciência do contrato. As nove entradas que estão lá (`targetId` ausente
  no `turn_opened`, `actionType` vazio, sem evento de HP, a corrente de testes, `ReboundDamage`,
  armadura, `move`/`attack` do master action, nenhuma projeção de declaração, NPC não age)
  **continuam todas válidas** depois deste PR. Não risque nenhuma.

  > Que o contrato não soubesse dessas três é o dado interessante: uma lista de lacunas
  > escrita sem cliente enxerga o que o autor já sabia faltar, não o que o primeiro leitor
  > descobre. Vale registrar isso na descrição do PR.

**`docs/dev/api/character-sheet.md`**: o `combat-catalogue`, junto das outras rotas de ficha.
É rota de ficha; não abra doc própria para ela.

**`docs/dev/match/flows/05-lacunas.md`**: o que deixou de estar oco.

**`docs/documentation-map.yaml`**: as entradas novas.

> O contrato tem um aviso no topo dizendo que nenhum cliente real o leu ainda, e que uma
> divergência encontrada é **bug do contrato**. Este PR é a primeira vez que alguém o corrige
> por esse motivo. Mantenha o aviso; ele continua valendo para a Fase 6.

## 13. Verificação, no agregado

- `go vet ./...` depois de **cada** tarefa, não só no fim.
- A suíte e2e do pacote `game` já dirige dois clientes WS reais (`combat_e2e_test.go`,
  `reaction_chain_e2e_test.go`). As tarefas 1, 4, 6 e 7 ganham e2e de verdade ali.
- As tarefas 2 e 3 são domínio puro: teste de unidade em `service`.
- A tarefa 5 ganha teste de handler HTTP real (`humatest`) **e** smoke `curl` com o servidor
  de pé — é um endpoint novo, e endpoint novo se prova respondendo.
- CI antes de local: `rtk gh run view <id> --log-failed`.

## 14. Riscos

| Risco | Mitigação |
|---|---|
| **Deadlock no `r.mu`** na tarefa 7 — o helper toca `r.pieces` e o call site já segura o lock | Mapear quem segura o quê antes de escrever; o `handlePieceMoved` original pega e solta o lock em trechos curtos, e o helper deve manter essa forma |
| **`seq` incrementado por engano** no `match_full_state`, quebrando a guarda do cliente | Está escrito no §8 e tem teste próprio |
| **A extração do helper mudar o comportamento do lobby** — `handlePieceMoved` é o caminho que o lobby usa hoje e funciona | A suíte de fog (`fog_*_test.go`) é a rede; rodá-la antes e depois da extração, sem alterar os testes |
| **Revelar a finta cedo demais** ao passar `true` fixo no histórico | Passar `tu.FinishedAt != nil`, que lê o fato em vez de assumi-lo |
