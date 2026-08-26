# Fase 5 — Regência e visibilidade

> Documento auto-contido. Quem for planejar/implementar não precisa ler as fases anteriores;
> precisa ler **isto** e, quando um item apontar, o trecho indicado de
> `docs/dev/match/combat-engine.md`.
>
> **Antes de tudo: `git pull`.** As Fases 1–4 e o fix do caminho de escrita (PR #69) estão em
> `main`.

## O que esta fase entrega

A mesa funcionando: o mestre encerra o turno de propósito em vez de por efeito colateral,
edita a ação quando a cena pede, e cada pessoa na sala recebe a versão do turno que lhe cabe —
com o histórico legível depois.

## Contexto mínimo

- **Turno** = uma action **e** suas reactions. **Round** = sequência de turnos. **Cena** =
  sequência de rounds.
- O mestre "passa o bastão": `open_next_action` abre a próxima da fila (e **fecha a anterior
  por efeito colateral**, que é o que esta fase muda), `open_reaction` dá a palavra a uma
  reaction anexada.
- Todo cálculo é feito **quando a action/reaction chega**. O mestre **nunca re-rola o dado de
  um jogador**: `RollAttempts` guarda os dois conjuntos desde o sorteio, e `Derive` recalcula
  lendo, sem sortear.
- Wire em **camelCase** dos dois lados.
- `Action.actorID` é o **sheetUUID**, não o do jogador. Autorização é por
  `charToPlayer[actorID] == playerUUID`.

---

## 1. Encerramento explícito do turno

Hoje o turno fecha dentro de `OpenNextAction`/`PullAction`. Passa a existir `close_turn`,
master-only.

**⚠️ `MatchSession.CloseTurn()` não serve como está.** Ele chama `roundOrch.CloseTurnErr`
direto e **pula `closeOpenTurn`** — não resolve, não aplica dano, não mexe nos ledgers. A rota
explícita **tem** que passar por `closeOpenTurn`, ou o dano do turno evapora sem erro nenhum.

**A confirmação é do back.** `close_turn` **recusa** quando há reactions anexadas e não
abertas, devolvendo a lista; aceita com `confirm: true` no payload. Assim o critério é
verificável sem front — a Fase 6 é que desenha o diálogo.

> Essas reactions **entram no cálculo**; o que elas perdem é o momento de narrar. É por isso
> que a confirmação existe.

**Encerrar turno NÃO fecha o round.** A exaustão continua detectada só onde há escalonamento
(`OpenNextActionUC`). Custa um passo a mais ao mestre e mantém **um** ponto de detecção — dois
pontos é como duas versões da mesma regra nascem e divergem.

## 2. A edição do mestre

Detalhe e razão em `combat-engine.md` § *A edição do mestre*. Leia essa seção antes de
planejar este item; o resumo abaixo é o que muda em código.

**A action editada É a action.** O valor do mestre entra na própria action; todo consumidor lê
um lugar só. Não é decisão nova — `RollCondition` já mora em `RollContext`, dentro do
`RollCheck`, dentro da `Action`.

**Duas superfícies:**

| Superfície | Onde mora | O que muda |
|---|---|---|
| `RollCondition` | em `RollContext`, dentro do `RollCheck` | **como um teste é lido** — `Bias`, `Modifier`, `Description` |
| `MasterAction` | no `Turn`, via `AddMasterAction` | **quais testes existem** — `Skills`, `TargetID`, `ActionSpeed`, componentes |

**`MasterAction` não é sobreposição — é a ação do mestre**, e ela opera em dois níveis: dentro
de uma action de jogador, e acima da batalha (o que só ele conduz). O segundo nível ainda não
está escrito e **não é desta fase**. `buildMasterAction` (`action_mapper.go`) mapeia só parte
dela — `Move` e `Attack` caem em `TODO`. É o **mapper** que está incompleto, não a entidade.

**Acrescentar perícia rola dados novos** — não fere "o mestre nunca re-rola o dado de um
jogador": não é re-rolagem, é a primeira rolagem de um teste que não existia.

**Remover perícia guarda os dados na memória** enquanto o turno está aberto — senão tirar e
recolocar seria re-rolagem grátis. No fechamento eles vão para
`overridden_action_values` (item 3), **nunca para o histórico da action**.

**⚠️ A edição muda o desfecho, nunca a economia.** Barras cobradas, `Speeds` registradas e
ordem já jogada não se refazem — inclusive quando a edição mudaria o que `Bars()` responde
(`chargesActionBar()` lê `len(a.Skills) > 0`). `bars_updated` já foi ao ar; refazer o preço
reordenaria o que já foi jogado.

**⚠️ NÃO existe verbo de confirmação.** O mestre edita, `Derive` recalcula na hora, e **passar
o bastão é a confirmação**. O que um "confirmar" daria de real seria *cancelar* — e cancelar é
editar de volta ao valor original, que apaga a captura (item 3) e não grava nada.

**Fora de escopo:** o override do desfecho da cadeia em área. É regra de jogo ainda não
escrita; `combat-engine.md` marca o lugar sem descrevê-la.

**⛔ Buraco conhecido, e ele limita o que esta fase pode provar.** `match_session.go` rola cada
`Skill` da action e **ninguém lê o resultado** — a única leitura de `Skills` em toda a colisão
é a `Evasion` da esquiva fechada, por nome. A regra da **corrente de testes** (margem
atravessando de um teste ao próximo, erro por 10+ matando a corrente) está escrita em
`docs/game/combate/acoes.md` e **não existe em código**. Esta fase **não** a implementa: ela
entrega a superfície de edição e a auditoria. Acrescentar/remover perícia muda a lista e a
lista ainda não decide nada — diga isso no PR, não finja que decide.

## 3. `overridden_action_values`

**O que guarda:** o valor que a edição **atropelou** — não a edição. A action já carrega o
valor novo; guardar os dois é duplicação que diverge.

**Por que o nome:** `SystemData` foi reprovado porque um nome genérico não consegue *recusar*
nada e vira depósito. `edited` nomearia o ato do mestre, e não é o ato que está guardado.
`discarded` descreveria o valor sozinho, perdendo a relação que interessa: **existe um valor
que tomou o lugar dele**, na linha correspondente de `actions` — é ela que permite reconstruir
o original.

**Forma:**

- **Uma linha por campo, com o ORIGINAL.** Não uma linha por edição. O propósito é não perder
  *o que o jogador enviou e o que o sistema calculou*; os intermediários do mestre não são
  nenhum dos dois.
- Identidade em **coluna**: qual action, qual campo, quando, qual mestre, e a origem do valor
  deslocado (`system` | `player`).
- O valor deslocado em **`JSONB`**, porque o formato varia de verdade — um inteiro, uma lista
  de perícias, um conjunto de alvos — e ninguém vai consultar dentro dele.
- **Capturado** na primeira sobreposição, em memória; **gravado** no fechamento do turno, na
  mesma transação do `PersistTurnClose`.
- **Reverter apaga a captura.** Editar de volta ao original não deixa linha nenhuma.

> Não há campo `Source: master` — se só o mestre sobrepõe, ele não teria o que discriminar. O
> viés que o *sistema* aplica já é um `Modifier` no ledger e nunca foi sobreposição.

## 4. Visibilidade por destinatário

Regra durável em `combat-engine.md` § *A política*. O que muda em código:

`resolution_updated` deixa de ser master-only e passa a ser **projetado por destinatário**.
Reusar `Room.dispatchPerPlayer` (`room.go:1087`), que já faz exatamente isso para o fog —
**não inventar outro mecanismo**.

**Público por omissão, com deny-list, para a mesa inteira.** As duas metades vêm de regras que
já existiam: *dano é público, HP não*, e *"o adversário precisa deduzir dos números"* —
deduzir exige ver os números.

**Três classes, não quatro:** mestre (tudo), **dono** da action/reaction (tudo o que é dele),
todo o resto (tudo menos a deny-list). **O alvo não é privilegiado** — uma finta contra você
não te conta que era finta.

| Oculto de terceiros | Por quê |
|---|---|
| HP | o dano é público, o HP não |
| `Feint` | uma finta revelada não é finta |
| `Trigger` | idem, até disparar |
| a entrada de `Evasion` na esquiva fechada, e a reserva que ela gera | o adversário deduz |
| **o próprio `ReactionKind`, nas variantes fechadas** | ⬇ |

**⚠️ O rótulo é o vazamento.** Se `closedDodge` chega público, ninguém precisa deduzir nada — o
rótulo já contou que havia Evasão embutida. Uma esquiva fechada chega aos terceiros
**indistinguível de uma esquiva** (`dodge`); um escape fechado, de um escape.

A dedução continua possível, que é o ponto: `bars_updated` é público desde a Fase 3, e o escape
fechado cobra **uma** barra enquanto o padrão cobra duas. **Deduzir da barra é legítimo; ser
informado não é** — a política inteira cabe nessa frase.

**Também nesta fase:** notificação de action enfileirada passa a ser **só para o mestre**.

**Fora de escopo:** a exceção do percept no início de batalha (só vê os alvos quem percebeu).
Bloqueada — os subatributos mentais não existem.

## 5. O resultado do turno persistido

Hoje **nenhum número derivado é gravado**: margem, dano, degrau da escada, estado da cadeia.
Os *dados* já estão persistidos (`RollCheck` carrega `Attempts` e `Result` e vai inteiro para
as colunas JSONB); o que falta é a **colisão**. E recalcular depois é impossível — o
`ModifierLedger` daquele instante não existe mais.

**`turns.resolution JSONB`**, com a `service.TurnResolution` **liquidada** — a que teve o dano
aplicado, não um dry-run. A resolução é recalculada a cada edição do mestre; **a que se
persiste é a do fechamento**. `room.go` já tem `result.ClosedResolution` na mão no mesmo ponto
em que chama `PersistTurnClose`; a assinatura ganha o parâmetro.

## 6. Action History — o caminho de leitura

Não existe endpoint que devolva os turnos. Precisa da consulta **e** da projeção por campo em
cima dela (item 4 vale igual no histórico).

**A resposta é aninhada, não uma lista plana.** A hierarquia do domínio — Cena → Round → Turno
→ Action — é a mesma da resposta; o front renderiza cards de action **dentro do escopo de cada
cena**. Não achatar.

REST, no padrão de `internal/app/api/match/` (huma + chi, `routes.go` registrando handlers).

## 7. `pull_action` alcançável

O mestre não tem como aprender o ID de uma action pendente, então `pull_action` é
**inalcançável de um client real**. É o mesmo buraco que `PendingReactions` fechou para
`open_reaction` na Fase 4 — mesma solução: o ID tem que chegar num payload que o mestre recebe.

> Uma operação cujo ID o client não consegue aprender é uma operação que o client não consegue
> invocar.

---

## Fora de escopo

- Rostering de NPC. **Não bloqueia esta fase** — os dois critérios de pronto são verificáveis
  com dois clients WS de jogadores. É fatia própria antes da **Fase 6**, que é quem não
  consegue desenhar uma mesa sem NPC.
- Front (Fase 6).
- A corrente de testes em código (item 2).
- O override do desfecho da cadeia em área.
- A exceção do percept.
- Posturas; modelo de armadura (`ChainState.Reduce` usa `const armour = 0` de propósito).

## Pronto quando

- **Dois clients WS** conectados como jogadores diferentes recebem, para o mesmo turno,
  payloads **distintos** — um com o campo oculto, outro sem. Verificável no backend, sem front.
- Uma **esquiva fechada** chega a um terceiro como `dodge`, sem a entrada de `Evasion`, e chega
  ao dono e ao mestre como `closedDodge`.
- `close_turn` **recusa** com reactions anexadas e não abertas, e **aceita** com
  `confirm: true` — e o dano do turno é aplicado nos dois caminhos que fecham turno.
- O Action History devolve um turno fechado, **aninhado por cena**, com os campos ocultos
  respeitados.
- Uma sobreposição do mestre aparece em `overridden_action_values` com o **valor original**; e
  editar de volta ao original **não deixa linha**.
- `pull_action` é invocável a partir de um ID que o mestre recebeu num payload.

## Armadilhas conhecidas

- **`MatchSession.CloseTurn()` pula `closeOpenTurn`** — item 1. É a que custa mais caro,
  porque falha em silêncio.
- **Um teste 2D10 consome QUATRO faces, não duas.** `RollCalculator.Roll` sorteia sempre os
  dois conjuntos (`Primary` e `Secondary`), mesmo sem Vantagem/Desvantagem. Um teste
  roteirizado que reserva 2 faces por teste sorteia metade do que o código consome. Só o dano
  da arma rola um conjunto só. Ver `combat-engine.md` § *Quantos dados cada reação consome*.
- **`PersistTurnClose` engole o erro por design** (`room.go`) — o turno já fechou em memória e
  a mesa não pode parar. Se a persistência desta fase falhar, o teste tem que olhar o banco,
  não o retorno da operação.
- **`r.mu` é a única serialização.** `MatchSession` não tem lock interno; quem mexe em estado
  de sessão segura o lock de escrita, mesmo em método que "só lê" mapa.
