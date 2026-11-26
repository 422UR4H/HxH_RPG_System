# Efeitos de status — desenho (fases 1 e 2)

Contrato do fio: [`../api/match-status-effects.md`](../api/match-status-effects.md). Spec:
`docs/superpowers/specs/2026-11-15-status-effects-design.md` (e `.pt-br.md`). Plano:
`docs/superpowers/plans/2026-11-17-status-effects-phase-1.md`.

A fase 1 entregou o **catálogo, o armazenamento, o comando do mestre, a visibilidade e os
bloqueios**. A fase 2 (plano `docs/superpowers/plans/2026-11-24-status-effects-phase-2.md`) fez os
efeitos **pesarem nas rolagens** (vantagem/desvantagem, penalidade, falha automática), pôs o
**crítico** e a **morte** (`dead`) — ver "Rolagens" e "Morte" abaixo e
[`combat-engine.md`](combat-engine.md).

## Mapa de pacotes

| Peça | Onde |
|---|---|
| Domínio puro: tipos, parâmetros, peças, conjunto, quem vê | `internal/domain/statuseffect/` |
| Sessão: guarda e consulta os efeitos de cada ficha; recusas e refazer pendências | `internal/domain/match/matchsession/status_effects.go` (+ `error.go`) |
| Caso de uso (sem I/O): aplicar, editar, remover | `internal/application/match/status_effects.go` (`StatusEffectsUC`, `StatusEffectChange`) |
| Carga ao iniciar a partida | `internal/application/match/init_match_session.go` (`WithStatusEffects`) |
| Gateway | `internal/gateway/pg/statuseffect/` (`ListBySheets`, `Write`); migration `20261117000000_character_status_effects.sql` |
| Sala: comandos, projeção por destinatário, gravação | `internal/app/game/status_effects.go`, `sight.go`, `room.go`, `master_board.go` |
| Fio | `internal/app/game/message.go`; forma compartilhada sala + histórico REST em `internal/app/wire/statuswire/` |
| Histórico: eventos, revelações, âncoras | `domain/statuseffect/event.go`; `gateway/pg/statuseffect/{events,reveals}.go`; `application/match/{get_match_history,reveal_status_effect}.go`; sala: `internal/app/game/status_anchor.go` |

O domínio `statuseffect` não importa nada de fora de `domain/entity`, exceto `domain/masteraction` (o `View` das `views` de um evento de histórico, `event.go`).

## As peças e a contenção

Um efeito não carrega regras: **compõe peças** (`Effect.Pieces()`, `piece.go`), as unidades que
o motor lê: `noAct`, `noReact`, `domainLocked`, `domainWarning`, `reactionRestricted`,
`testPenalty`, `autoFail`, `noDisplacement`, `losesHp`, `recovery`, `triggeredTest`, `conduct`,
`control`, `counter`. A **contenção** está no código, não no banco: Atordoado = Paralisado nos
quatro domínios + Atônito + falha de sentidos + Recuperação; Inconsciente herda o que o Atordoado
**faz**, não como ele **termina** (sem Recuperação por Tenacidade); Lesionado `disabled` também
carrega a desvantagem de `hurt`. Isso fecha a decisão 5 do plano: herda-se o que um status faz,
não como acaba.

**Só as peças que bloqueiam** têm consumidor hoje, e todas passam por **`Summary`**
(`summary.go`, `Summarize`): `CannotAct`, `CannotReact`, `NoDisplacement`, `LockedDomains`,
`ForbiddenReactions` e as `Notes` (aviso de domínio e conduta), cada um com a lista de **quem
causa** (`...By`, usada na mensagem de `blocked_by_status`). A sessão chama `StatusRefusal`
em `EnqueueAction` e `AttachReactionWithCancels`, depois de toda validação existente e antes de rolar
dado, consumir ou cobrar. `losesHp`, `recovery`, `triggeredTest` e `control` são **declaradas e
testadas, e ainda não lidas** pelo motor. `testPenalty`, `autoFail` e as `reactionRestricted` de
**desvantagem** e **só rolada** são lidas, desde a fase 2, por `RollEffectOf` ([Rolagens](#rolagens-fase-2)).

Mapa reação ↔ peça (decisão 9 do plano): `default` = esquiva por reflexo + defesa padrão;
`dodge` = esquiva por reflexo; `closedDodge` = reflexo + evasão; `escape`/`escapeGuard` = fuga;
`closedEscape` = fuga + evasão; `repel` = repelir; `nothing` sempre. Domínios (os conjuntos
do diagrama de ações em `docs/dev/match/flows/_source/fluxos.excalidraw`, dono, 2026-11-25):
esquiva → reflexo, evasão; defesa → defesa padrão; **repelir → ataque E defesa** (`needRepel`:
travar qualquer um dos dois o proíbe, e a recusa nomeia o domínio travado — "não pode atacar" /
"não pode defender"); fuga precisa de movimento **e** esquiva (spec §5.4).

**Por que o repelir está em ataque e defesa (R1).** Duas leituras, juntas: (1) repelir é
**atacar o ataque** — você acerta o golpe do oponente; (2) um ataque já declarado contra quem te
ataca pode virar o repelir, à vontade do jogador e sem penalidade (`keepsQueuedIntent`,
`sameBlowAsQueued`). Mesmo assim o repelir é **um teste só** (Repel), que recebe o que pesa no
ataque e o que pesa na defesa — ver [Rolagens](#rolagens-fase-2), R4.

**A reação padrão não é interseção (R2).** A defesa padrão (conjunto defesa) e a esquiva por
reflexo (conjunto esquiva) são isoladas: cada uma no seu conjunto, cada uma com o seu teste. A
**esquiva defensiva** (esquiva ∩ defesa) é **outra ação**, que ainda não existe no código — não é
a reação padrão.

**G1 (Tato, face equilíbrio):** prejudicado → ataque, defesa, esquiva e movimento recebem a
desvantagem/número do próprio status; anulado → uma desvantagem nesses quatro. Declarado em
`sensePieces` e **aplicado desde a fase 2** (peças `testPenalty` com escopo de domínio). O **teste
de equilíbrio** em si (`triggeredTest`, que derrubaria o personagem) continua sem consumidor.

### A reação padrão sentada segue os status

`seatDefaultReactions` agora senta `seatedReactionFor`: a `default` inteira; a `default` com
uma das passivas tirada (`WithoutReflexDodge` / `WithoutDefaultDefense` na `action.Action`, o
resolvedor as lê: sem reflexo pula a esquiva, sem defesa pula a defesa padrão); ou `nothing`,
quando o personagem não pode reagir ou perdeu as duas passivas. **Esses dois sinalizadores não
são persistidos** (`actioncodec` grava `DefaultDefense`, não eles, e nenhum mapper do fio os lê):
o histórico mostra uma `default` simples. O dano aplicado no fechamento é calculado em memória
**com** os sinalizadores, então está correto.

`DropRefusedByStatus` (a cada mudança de status do personagem, remoções inclusive) tira da fila
o que o conjunto recusa (com os movimentos empilhados, pela mesma cascata do `cancel_action`;
as edições do mestre descartadas; sem marca de troca, porque a decisão é do mestre) e refaz as
reações **não abertas** do turno aberto: sentada de mesmo tipo = só os sinalizadores; sentada de
tipo diferente ou enviada agora recusada = `DropUnopened` e nova sentada. Reação **aberta** nunca
é tocada. Uma reação enviada e substituída mantém o que já gastou (barras, ações consumidas):
nada é devolvido nem registrado.

## Rolagens (fase 2)

**Uma função lê, três chamadores.** `statuseffect.RollEffectOf(effects, Test)` soma as peças de
todos os efeitos que alcançam **um teste** e devolve `RollEffect{Bias, Penalty, AutoFail,
RolledOnly}` (`roll_effect.go`). `service.StatusRollOf` o traduz para `service.StatusRoll`, a
quarta origem de viés de `RollCalculator.Derive` (ao lado do sistema, do mestre e do ledger). Quem
chama: o **resolvedor** (acerto do atacante; reações do alvo — `ResolveInput.Effects`), a
**sessão** (`deriveSpeeds`, `deriveSkills`) e nada mais. `Summarize` continua dono do **bloqueio**
(`forbidden`); `RollEffectOf` **não lê** `forbidden`.

**O `Test` descreve o teste, não como rola:** `{Skill, Domains, Reaction, MoveSpeed}` —
`Domains` são os conjuntos em que o teste está (vazio = não é teste de conjunto; o repelir é o
único em dois, `{attack, defense}`). Quais peças o
alcançam (`scopeReaches`): escopo de **aptidão** (Debilitado) → a aptidão da perícia
(`AptitudeOfSkill`: o conjunto Focus/WillPower/SelfKnowledge/Coa/Mop/Aop é espiritual, qualquer
outra perícia é física, não há mental); escopo de **domínio** (Lesionado, Visão anulada, G1) → um dos
domínios do teste (uma vez por efeito, R4 abaixo); escopo de **sentido** → a perícia do sentido (`SenseOfSkill`; no Tato só a face
**toque** — um teste de Tato declarado numa ação é lido como toque, provisório, spec §16.4 item
38); escopo de **moveSpeed** (Lento). `reactionRestricted` com `disadvantage` soma um nível de
viés quando a `Reaction` do teste é a da peça; `rolledOnly` faz a defesa padrão ser rolada.

**Tabela domínio → teste** (decisão 3 do plano, refeita pelos conjuntos do
`fluxos.excalidraw`, 2026-11-25):

| Domínio | Testes |
|---|---|
| ataque | o acerto (`hit`) e o repelir (`repel`) |
| defesa | a defesa padrão e o repelir (`repel`) |
| esquiva | o reflexo (esquiva padrão, `dodge`, `closedDodge` e o reflexo dos escapes) e a evasão |
| movimento | `moveSpeed` (o componente de movimento e o segundo teste dos escapes) |
| — | `actionSpeed` (Legerity) e o dano **não** são testes de domínio |

- **Repelir = ataque ∩ defesa, um teste só (R1).** O `Test` do Repel leva `Domains: {attack,
  defense}` (`resolveRepel`).
- **Escape = movimento ∩ esquiva, dois testes (R3).** Cada teste lê só o seu conjunto: o reflexo
  lê a esquiva; o `moveSpeed`, o movimento. O Caído e a Guarda aberta continuam pesando nos dois
  (são peças de reação, não de conjunto, e são dois status diferentes: somam). O escape defensivo
  (`escapeGuard`) é o mesmo, mais a defesa padrão que ele mantém como rede (que lê a defesa).
- **Um status pesa uma vez só no mesmo teste, pelos conjuntos (R4).** A lista de conjuntos de um
  efeito (Visão anulada: ataque, defesa e esquiva; Tato no equilíbrio: os quatro; Lesionado: um
  por conjunto lesionado) diz **até onde ele alcança**, não **quanto pesa**. `RollEffectOf` junta
  as peças `testPenalty` de escopo de domínio **do mesmo efeito** que alcançam o teste e soma uma
  vez a maior desvantagem e a maior penalidade entre elas. Efeitos diferentes continuam somando;
  as peças de aptidão, sentido, `moveSpeed` e as de reação seguem como antes. Exemplos no Repel:
  Visão anulada → 1 nível (não 2); Lesionado só no ataque → 1 nível; Lesionado no ataque e na
  defesa → 1 nível; Visão anulada + Lesionado no ataque → 2 níveis (dois status); Tato no
  equilíbrio com penalidade N → −N uma vez. Num teste de um conjunto só nada muda.
- **Finta (R5).** O que pesa no ataque pesa na finta — mas a finta ainda não tem total derivado
  (o motor não a resolve), então hoje não recebe nada. `TODO(finta)` em `turn_resolver.go`, ao
  lado do `Test` do acerto.

O Debilitado pesa em **todo** teste da aptidão (acerto, reflexo, evasão, defesa, repelir,
`actionSpeed`, `moveSpeed` e `skills[]`). **`skills[]`** ganhou o total derivado
(`RollCheck.Result`, decisão 5): `deriveSkills` o recalcula na chegada, a cada edição do mestre e a
cada mudança de status, e nada o lê para um desfecho ainda (a corrente de testes lerá).

**Escapes (provisório).** O Caído e a Guarda aberta põem desvantagem nos **dois** testes de **cada
um** dos três escapes (o reflexo e o `moveSpeed`, ambos com `Reaction: escape`); o Caído também
pesa na esquiva por reflexo e no repelir. É **provisório**, TODO até o redesenho do movimento: que o
Caído "role" sem penalidade é só **um ponto possível** desse desenho, não regra (palavras do dono,
spec §16.4 itens 33 e 37). O Caído **não** bloqueia nada; a Guarda aberta proíbe esquiva por
reflexo e repelir e deixa a defesa padrão só rolada.

**Falha automática.** `autoFail` por sentido (Atordoado, Inconsciente, Morto; sentido anulado) faz
`Derive` devolver `Total 0`, `AutoFailed`, sem dados e sem crítico: **nada é rolado** e o teste não
pede dados.

**Mudou um status → re-deriva.** `MatchSession.rederiveForStatus` (chamado ao fim de
`SetStatusEffects`/`ReplaceStatusEffects`) refaz as velocidades das ações **na fila** do
personagem e das reações dele no turno aberto, e o `skills[]` de todas (a fila é reordenada pelo mesmo caminho de uma velocidade editada). Com
turno aberto, a sala republica o cálculo (`turnTouched`).

**Invariante: ler nunca rola, e todo comando que muda o viés resolve sob a trava de escrita.**
`Derive` e `TurnResolver.Resolve` nunca rolam: um teste que pede mais dados do que existem
informa `RollOutcome.Short`/`TurnResolution.DiceShort` e quem **pode** rolar (a sessão: `deriveCheck`,
`ResolveTurn`/`drawShortDice`) tira só os que faltam, **uma vez**. A leitura de somente-leitura
(`PeekTurnResolution`) loga em vez de rolar. Todo comando que muda um viés (edição, abertura,
mudança de status) tem de resolver sob a trava de escrita.

## Morte (fase 2)

Decisões do dono: spec §16.4 itens 28–32 e `docs/superpowers/specs/2026-11-15-status-effects-design.pt-br.md`.

- **Mínimo da barra.** `HealthPoints.min = -NegativeMax()`, onde `NegativeMax = ⌊10 + nível de
  Vitality + valor de Resistance + bônus de Físicos⌋` (recalculado em `Upgrade`). `DecreaseAt` não
  tem piso (o HP exato fica: −40 é −40) e `SetCurrent` só limita pelo máximo.
- **Quando morre.** Quando o HP **aplicado** fica **abaixo** do mínimo (um além; no mínimo é vivo).
  É decidido pelo HP **aplicado no fechamento** (`matchsession/death.go`, `applyDeaths`), não pelo
  `Dies` da resolução, que é só a **prévia** do mestre (`dies` em `service/turn_resolver.go`, por
  alvo, não cumulativo, zerado para quem não é mestre em `ProjectResolution`).
- **Fluxo no fechamento** (`close_turn`, `open_next_action`, `pull_action`, os três por
  `closeOpenTurn`): `applyResolution` → `applyDeaths` (aplica o `dead` a cada ficha danificada que
  ficou abaixo do mínimo e ainda não estava morta, e tira da fila o que ela tinha pelo mesmo caminho
  de qualquer status) → `advanceLedgers`. O resultado vai em `TurnTransition.Deaths` (as gravações,
  uma aplicação por morto) e `DeathDrops` (as ações que saíram da fila). Se a ação que o agendador
  tinha escolhido para abrir era de quem morreu, `open_next_action` escolhe de novo.
- **Sala.** `recordDeathsLocked` carimba os eventos como os de qualquer mudança, **mas no turno que
  fechou** (e no round dele) e os guarda em `turnWrites.deaths`, que o `persistClosedTurn` leva em
  `TurnCloseData`: o Morto entra **na transação do turno que o causou**. As ações derrubadas são
  registradas como canceladas pelo mestre, também nesse turno. **Ordem dos avisos:**
  `broadcastHpChanges` (HP que matou) → `announceDeaths` (`action_cancelled` das ações derrubadas e a
  lista nova de status de cada morto, por `syncStatusViews`, pela névoa) → `broadcastTurnClosed`.
- **O que o `dead` é.** Peça de `Unconscious` (`Pieces` dos dois são `stunnedPieces(0, false)`),
  `PublicByDefault`, sem parâmetros, sem apelido, **não traz o Caído**, sem saída própria. O caso de
  uso recusa aplicá-lo à mão (`status_effect_invalid`, "O Morto entra só pela regra.").
- **Desfazer.** `StatusEffectsUC.Remove(..., hp *int)`: valida **tudo antes** de mexer na sessão;
  `hp` só no `dead`; padrão = o mínimo, só se o HP estiver abaixo (nunca rebaixa); igual ao HP de
  agora não é mudança. A mudança volta em `StatusEffectChange.HP`. A sala marca a ficha em
  `unwrittenSheets` (`markSheetsUnwrittenLocked`): fora de turno, o `flushStatusEffects` grava as
  barras na **mesma transação** dos efeitos (`StatusEffectRepo.Write(ops, bars)`); dentro de um turno,
  o fechamento as leva como leva qualquer ficha dele. **Efeito colateral intencional:** o flush leva
  **toda** ficha em `unwrittenSheets`, inclusive as de um fechamento que falhou antes — as duas
  coisas (status e HP) ficam pendentes juntas ou vão juntas.
- **A marca da ficha (`dead_at`)** só é escrita no **fim da partida**: a sessão registra quem morreu
  e quem teve a morte desfeita (`DeadMarks()`); `PersistMatchEnd` faz `dead_at = COALESCE(dead_at,
  fim)` ou `NULL`, na transação do fim.

## Quem vê (lista) e visão (névoa)

Armazenamento da lista: coluna `audience` JSONB **anulável** (`NULL` = padrão do tipo;
`{"everyone":true}`; `{"players":[…]}`; `{"players":[]}` = só o mestre). `statuseffect.Listed`
é a **única** conta de lista (`audience.go`): o mestre sempre; com lista, só quem está nela
(o dono não entra implicitamente); sem lista, público do tipo → todos, NPC → ninguém, PJ → só o
dono (ou todos, com `MatchRules.PlayerStatusEffectsPublic`).

A **visão** é somada pela sala em `seesCharacterLocked` (`sight.go`): o dono sempre; um
Escondido (`hiddenFromLocked`, que reusa `Listed` com o mesmo `Subject` da projeção —
`statusSubjectLocked` existe para as duas contas nunca divergirem) não é visto por quem está fora
da lista; sem tabuleiro, todos; com tabuleiro, o personagem precisa de peça, a peça não
escondida **para esse jogador** (`pieceHiddenForLocked`: `visible:false` global **ou** Escondido
por jogador, nunca escrito em `Visible`) e o centro dela dentro da visão do jogador.

**Projeção por destinatário** (`projectStatusEffectsLocked`): o mestre vê tudo com `visibility`
e `public`; os demais, o que `Listed` dá, e só se `seesCharacterLocked`. `syncStatusViews` compara
com o que cada um recebeu da última vez (`statusViews`, por jogador e personagem) e manda um
`status_effects_updated` só quando difere; quem não vê o personagem agora não recebe nem tem o
cache tocado (fica com o último estado visto). Lista vazia e sem entrada no cache valem o mesmo
(não há o que apagar).

**`syncSight`** é o ponto de reacerto depois de tudo que pode mudar a visão (peça que andou ou
saiu, parede, NPC que entrou, Escondido que mudou): `broadcastBars` (cada jogador, a vista dele) +
`syncStatusSight`. É adiado no fim de `relayPieceMove`/`relayPieceRemoved` e chamado em
`enrollLiveNPC`. Um Escondido que muda tira antes um retrato de quem vê cada peça
(`pieceSightLocked`), e `relayHiddenChange` manda `piece_removed`/`piece_moved` a quem perdeu ou
ganhou, antes de barras e status.

Barras também são por jogador (`BarsViewFor(isMaster, sees)`; cache de dedupe por jogador) e
`npc_added` vai só ao mestre (decisão 3).

## Quando se grava, e como se cura

Regra da casa (spec §10): **dentro de um turno aberto nada é gravado antes do fechamento.** O
**momento da gravação é decidido pela sala**, não pelo caso de uso (que não faz I/O e só devolve
`StatusEffectChange.Writes`): `holdStatusEffectWritesLocked`.

- Turno aberto: as gravações vão para `turnWrites.statusEffects` e o `persistClosedTurn` as leva
  em `TurnCloseData.StatusEffects`, na transação do turno.
- Fora de turno, com repositório: entram no fim de `unwrittenStatusEffects` e o chamador, já sem
  trava, chama `flushStatusEffects` (**grava antes de avisar**, como as master actions: o aviso
  serve de barreira nos testes).
- Sem repositório: não grava nem acumula.

**Cura:** uma gravação que falha continua pendente (`unwrittenStatusEffects`, em `unwritten.go`)
e vai **na frente** da próxima gravação que dá certo: a próxima mudança de status fora de turno,
o próximo fechamento de turno (que também devolve as do turno que falhou, logo depois do prefixo
que copiou: a ordem real) **ou o `end_match`** (`MatchEndData.StatusEffects`, na transação do fim:
`PersistMatchEnd`). Cada gravador descarta **exatamente o prefixo que copiou**. Uma sala que
fecha com gravações pendentes (ou com um turno aberto) registra no log (`logDroppedTurn`).

Limite conhecido de ordem: entre o caso de uso fechar o turno em memória e o `persistClosedTurn`
tirar o retrato, um **segundo socket do mestre** pode fazer uma mudança fora de turno que entra
em `unwrittenStatusEffects` na frente das do turno que fecha; se ela edita o mesmo efeito, a
ordem de gravação seria edit → apply. Com um socket só não acontece (o fechamento e o próximo
comando dividem a bomba de leitura).

### Cadeia de trava

`persistMu` → `r.mu` é o aninhamento permitido de tudo que grava (tabuleiro, fechamento de
turno, efeitos de status). **Só o `end_match`** segura a cadeia maior: `clockPersistMu` →
`persistMu` → `r.mu` (precisa de `persistMu` para a cópia e o descarte do prefixo de status não
correrem contra um `flushStatusEffects`). Não há deadlock porque nada que segura `persistMu`
toma `clockPersistMu` (`persistClosedTurn`, `flushStatusEffects`, `savePersistedBoard` e o
recarregar+salvar do `StartMatch` não chamam `persistClock` nem o checkpoint). Nunca se segura
`r.mu` durante I/O nem ao chamar `dispatchPerPlayer` (que toma `r.mu`).

## NPC ao vivo

`enrollLiveNPC` (add_npc com sessão viva): sessão recebe o NPC, `npc_added` vai ao mestre,
`loadLiveNPCStatusEffects` lê os efeitos (`ListBySheets`, sem trava; falha = NPC sem status, com
log) e só então `syncSight()` manda barras e status. O `npc_added` (`sendToMaster`) e o `syncSight`
seguinte correm na mesma goroutine e na mesma pista direta: o mestre recebe `npc_added` **antes**
dos efeitos do NPC. A leitura usa
`ReplaceStatusEffects`: um status aplicado por um segundo socket entre a inscrição e a leitura
seria sobrescrito (desprezível).

## Histórico: eventos, listas, revelações e âncoras

Contrato: [`../api/match-history.md`](../api/match-history.md).

**Eventos.** Cada mudança (`applied`/`edited`/`removed`) carrega um `statuseffect.Event`
(`Write.Event`), carimbado pela sala no instante e gravado **na mesma operação** do efeito
(`status_effect_events`: `snapshot`, `listing`, `views`, `hidden_from`, `action_uuid`, `seq` para
desempatar). O caso de uso continua sem I/O. `ResolveListing` resolve a lista do instante (o
padrão do tipo já aplicado, nunca "padrão"); `views` = jogadores que enxergavam o personagem logo
antes ou logo depois, mais o dono (o mestre nunca entra).

**Quem vê um evento** é uma regra só, `statuseffect.VisibleInHistory`: mestre; ou (âncora visível
**e** (revelação do efeito cobre o instante **ou** (leitor em `views`, ou em `hidden_from` com o
Escondido já revelado, **e** a última lista do efeito inclui o leitor))). A última lista
(`LatestListings`) vem de qualquer partida; sem evento anterior, nenhuma → falha fechada.

**`hidden_from`** (sala → banco): esparso, `NULL` = ninguém. `cancelled_actions` **não tem**
`hidden_from`: a action cancelada só chega ao mestre e ao dono do ator (dono do produto,
2026-11-20), que enxergam através do Escondido, então nenhum leitor dependeria de uma lista ali.
Em `status_effect_events` é
`{playerId: hiddenEffectId}` (jogadores da sessão para quem o personagem estava Escondido) — lá
ele só **abre** o evento a quem estava de fora; a lista de permissão do evento é `views`, então
nomear só a sessão não vaza nada. Em `actions` é uma **lista de permissão**
(`statuseffect.HiddenFrom`, `hiddenRecordLocked`): `{charId: {effect, perceivedBy: [playerId…]}}`,
por personagem Escondido com lista que não é "todos", com a lista resolvida do instante
(`ResolveListing`) — quem não está nela não o percebia, inclusive quem não era jogador da sessão
(leitor de fora da partida pública, membro da campanha não inscrito, inscrito depois). O dono e o
mestre são decididos na leitura. Cobre o ator e cada alvo na linha da action (abertura do turno)
e o reator na da reação (na abertura dela; nunca aberta, no fechamento). A sentinela de linha
ilegível é o registro zero (efeito `uuid.Nil`, ninguém): esconde de todos e não é revelável. O
codec é um só (`actioncodec.EncodeHiddenFrom`/`DecodeHiddenFrom`). Nunca sai no wire.

**Revelações** (`status_effect_reveals`, `reveals.go`): só o REST do mestre insere, e só um efeito
revelável na partida (`RevealableInMatch`: evento em qualquer partida + ficha participante desta —
um Escondido trazido de outra partida também); o histórico lê
o `MAX(revealed_at)` por efeito (`RevealedAt`) uma vez por requisição, junto de `LatestListings`.
`RevealLookup.Covers` = revelado em ou depois do instante. Ampliar a lista nunca revela.

**Âncoras** (`actionId`, a causa; `status_anchor.go` + `matchsession/anchor_moves.go`). Só o mestre
informa, e só uma action/reação do turno aberto. Enquanto o turno está aberto a sala as mantém
sob `r.mu`: **seguem** (reação sentada substituída pela do jogador, ou refeita por status → a nova;
sentada que sai sem substituta → a action do turno); **relink** (`relink_status_effect`) as troca
à mão; o mestre vê a âncora viva em `turnActionId`. **Desfazer** (`remove_status_effect` de um
efeito aplicado neste mesmo turno) apaga todas as gravações e eventos pendentes dele e tira o id
dele de `hidden_from` das ações/eventos do turno (`purgeEffectFromTurnLocked`: nas ações, a
entrada do personagem sai inteira); as ações da fila
que ele cancelou continuam canceladas, a não ser que o mestre peça a devolução (abaixo). Depois
do fechamento o histórico é definitivo.

**Devolver o que o status cancelou (D3, dono do produto 2026-11-20).** Quando
`dropRefusedByStatusLocked` tira ações da fila com turno aberto, o turno guarda o **registro do
desfazer** (`turnWrites.statusCancels`): efeito → os `matchsession.StatusDrop` que saíram, em
ordem de remoção, atribuídos a todo efeito do personagem que a mesma mudança aplicou ou editou
(atribuir a mais é seguro: a devolução revalida). O `StatusDrop` é o retrato do instante da
saída, tirado antes de a ação sair: a ordem de chegada dela (`MatchSession.arrivals`, que espelha
a ordem da fila e desempata), as edições do mestre (capturas, dados de perícias retiradas e o
valor editado de cada campo, que o `discardEdits` apaga) e se o movimento se apoiava numa base
ainda na fila, e os valores derivados que a saída recalcula (`derivedSnapshot`: `Speed` inteira,
`Move.Speed`, `Move.FinalSpeed`, `SystemBias`). O registro morre com o turno (fechamento, reinício,
`end_match`): depois dele não há o que devolver nem ponteiro velho. O mestre o vê em
`cancelledActionIds` (só efeito aplicado no turno). No `remove_status_effect`, o registro do
efeito é sempre apagado; só quando `undoInTurnAppliesLocked` desfaz esse efeito **e** veio
`restoreCancelledActions`, `giveBackLocked` chama `MatchSession.RestoreCancelledByStatus` — na
**mesma seção crítica de `r.mu`** da remoção e do desfazer, depois de o status sair da sessão.
A sessão **revalida, nunca repete**: ator na partida; id ainda não na fila (idempotente);
`statusBlock` dos status de agora; a cadeia de movimentos como estava na saída — o que se apoiava
numa base ainda na fila naquele instante volta só se ela continua na fila e é o fim da cadeia; o
que era cabeça (de fato, ou com a base já fora da fila antes do status — aberta, consumida) volta
só sem outro movimento pendente. A distinção é do instante da saída, não do de agora: uma base
cancelada depois da saída e uma que abriu antes dela parecem iguais na fila de agora. Nunca meia
cadeia, nunca garfo. A aceita volta ao mesmo lugar (`insertByArrival` + `PriorityQueue.InsertAt`,
pela ordem de chegada), com as edições do mestre de novo (`reapplyEdits`: valor editado, capturas,
dados retirados) e os valores derivados escritos de volta como estavam antes da saída
(`restoreDerived`) — **não** derivados de novo: o ledger só muda no fechamento de um turno
(pagamentos, `AdvanceTurn`), e um turno fechado entre o enqueue e a saída daria outra velocidade,
outro lugar na ordem do Race; `reseatForStatus` refaz as reações; a recusada volta com motivo em português (`restore_refused`, só ao mestre). Cada
devolvida apaga o seu registro em `turnWrites.cancelledActions` e sai do registro de todo efeito
(`forgetRestored`). Uma recusa por outro status (`RestoreRefusal.BlockedBy`) passa o `StatusDrop`
ao registro de cada efeito desse tipo aplicado no mesmo turno (`passToBlockerLocked`): desfazer
os dois devolve a ação em qualquer ordem. Um empilhado cuja base foi recusada assim na mesma
chamada herda o `BlockedBy` da base e vai junto, depois dela. O cancelamento por status não marca troca (sem
`swapMarks`), então não há marca a desfazer; o fechamento da janela de fusão do relógio que o
cancelamento fez (`clearClockMergeLocked`, em `holdCancelledActionsLocked`) não é desfeito — é
inofensivo: só faz o próximo ajuste do relógio virar um evento próprio. O anúncio é
`action_restored` (mestre em `Full`, dono no corte do `ownQueue`, com `beforeActionId`) e o
`action_queued` do mestre; nunca `action_enqueued`, que é o ack casado pela ordem dos envios do
cliente.

**A reação trocada volta também (revisão final do PR 2).** Uma reação **não aberta** que o status
tirou do turno aberto (`reseatForStatus`: a enviada que o conjunto recusa e a sentada trocada por
outra de tipo diferente) entra no mesmo registro (`statusCancel.reactions`), como
`matchsession.ReactionDrop` — retrato tirado **antes** do `DropUnopened`: o turno, a reação inteira
(cópia; o turno guarda reações por valor e a que saiu não está em lugar nenhum — dados,
velocidades, `ConsumedActionIDs` vêm nela), o lugar entre as reações (`Turn.ReactionIndex`), se era
a sentada, o pouso do mestre e as capturas dele (que o `dropEdits` apaga). Sai em
`DropRefusedByStatusWithReactions` (o `DropRefusedByStatus` de antes é o mesmo sem as reações) e o
mestre o vê em `swappedReactionIds`. Com o mesmo `restoreCancelledActions`, `giveBackLocked` chama,
logo depois do `RestoreCancelledByStatus`, `MatchSession.RestoreSwappedReactions`, que **revalida,
nunca repete**: o turno dela ainda é o aberto; o reator ainda está na partida e ainda é alvo da
ação do turno; nenhuma reação com esse id no turno (idempotente); `statusBlock` dos status de
agora (`ReactionRefusal.BlockedBy`, passado ao outro efeito do turno por
`passReactionToBlockerLocked`, como as ações); o reator não mandou outra reação depois (a dele é a
última palavra) e a sentada que ficou no lugar não foi aberta (o narrado fica). A aceita volta
como estava (`Turn.RestoreUnopened`: mesmo id, mesmo lugar, sentada ou não, pouso; `reapplyEdits`
para as capturas; só as passivas recalculadas), no lugar da sentada que está lá — que sai, com as
edições dela, e passa as âncoras para a devolvida (`noteAnchorMove`, aplicado pelo
`followAnchorsLocked` na mesma seção crítica): a âncora que tinha seguido a reação para a sentada
volta com ela. **A barra fica coerente:** a reação cobrou na chegada (`RecordSecretSpeed`, pelo id
dela) e o status nunca devolveu; com a mesma reação de volta, a cobrança volta a corresponder a
uma reação do turno. A recusada (ou sem o pedido) fica como o status deixou: fora do turno, com a
barra cobrada. O anúncio é `reaction_restored` ao mestre (com `replacedReactionId`) e ao dono só
quando a reação é dele (uma sentada, `RestoredReaction.Seated`, nunca está em `ownReactions`) e o
efeito desfeito era visível a ele (`statuseffect.Listed`: a troca só chegou a ele por esse
efeito, §9); nunca `reaction_attached`, que o front casa com o envio pendente do mesmo personagem
(`status === "sending" && actorId`). A mesa não recebe nada (o `resolution_updated` do turno
aberto é do mestre).

**A regra do ctrl+Z.** O inverso de aplicar um status no turno aberto é o desfazer com
`restoreCancelledActions` (`undoInTurnAppliesLocked` + `giveBackLocked`, que chama
`RestoreCancelledByStatus` para as ações da fila e `RestoreSwappedReactions` para as reações
trocadas). Um futuro ctrl+Z deve chamar este caminho, nunca escrever a sua própria reversão: a
reversão revalida contra o estado de agora, sob `r.mu`, e recusa com motivo o que não cabe mais.
**O que o inverso cobre:** as gravações e eventos pendentes do efeito (nada dele é gravado), o
`hidden_from` dele no turno, as ações da fila que ele cancelou (lugar, edições, velocidades de
antes; o registro de cancelada some) e as reações não abertas que ele trocou (a mesma reação, as
âncoras de volta; a barra cobrada volta a ser coerente). **O que não cobre:** a janela de fusão do
relógio fechada pelo cancelamento (`clearClockMergeLocked`, inofensivo); o que foi recusado na
revalidação (fica como o status deixou: ação cancelada e gravada no fechamento, reação fora do
turno com a barra cobrada — a cobrança dela, aí, não volta); uma reação que já foi **aberta**
(o status nunca a tira); e nada depois do fechamento do turno.

**Horários reais.** O turno guarda a abertura (`Turn.GetOpenedAt`, relógio da sala, µs):
`turns.created_at` é ela; cada reação ganha `actions.opened_at` (NULL = nunca aberta). A regra de
"escondido" compara a revelação com esses instantes.

## Limitação: uma ficha viva em duas salas

Cada sala guarda os efeitos **na própria sessão** e grava o que mudou nela. Uma mesma ficha viva
em duas partidas ao mesmo tempo tem **duas memórias**: a última gravação ganha no banco, sem
merge. Não é tratado nesta fase.

## O que não está aqui

Testes automáticos (resistência ao entrar, equilíbrio do Tato, Recuperação, Inconsciente
automático), dano e recuperação por round, um comando de **editar o HP** do mestre, asfixia e
Fadiga, "não usa habilidades" (Nen), a regra de escape por tipo (provisória, aguarda o redesenho do
movimento), revelar a um jogador só (a revelação é para todos), push de WS depois de revelar e
desfazer depois do fechamento do turno. Lista no contrato, §10.
