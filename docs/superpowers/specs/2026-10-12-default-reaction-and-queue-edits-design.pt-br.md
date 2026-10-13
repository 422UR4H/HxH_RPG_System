# Reação padrão, viés em teste passivo e edição na fila — design

> **Escopo:** a pendência que o fechamento da Fase 6 deixou ("a condição do mestre sobre
> esquiva, defesa ou repelir é aceita mas não muda nada") já tinha sido resolvida na Fase 8
> (PR #85) para as reações **abertas**. Sobrava o resto, decidido pelo dono do produto em
> 2026-10-12: a esquiva e a defesa passivas de quem não reagiu, o viés numa leitura passiva, a
> reação não aberta fora do cálculo (bug) e a edição de uma ação ainda na fila. **Uma tarefa
> cross-repo, um PR por repo**: este (`HxH_RPG_System`) e o do front (`HxH_RPG_Environment_React`),
> mesma branch `claude/jolly-sagan-gvfth9`.
>
> Contrato: [`../../dev/api/match-combat-ws.md`](../../dev/api/match-combat-ws.md) —
> `attach_reaction`, `open_reaction`, `edit_action`, `close_turn`, `action_queued`,
> `action_edited`, `resolution_updated`, `match_full_state.ownReactions`.
> Motor: [`../../dev/match/combat-engine.md`](../../dev/match/combat-engine.md) — "A reação
> padrão", "A edição na fila", "Rolagem".
> Regras: [`../../game/combate/reacoes.md`](../../game/combate/reacoes.md),
> [`../../game/combate/o-mestre.md`](../../game/combate/o-mestre.md).

## 1. Decisões do dono do produto (2026-10-12)

| # | Decisão |
|---|---|
| D1 | **A passiva é uma reação.** Quem não respondeu tem a reação padrão — esquiva por reflexo, depois a defesa padrão. Ela tem registro, entra no turno como resposta à action, o mestre a edita como qualquer outra e o jogador a vê no histórico depois do fechamento. |
| D2 | **"Não fazer nada" é escolha explícita** e não tem esquiva nem defesa padrão: recebe o golpe cru. **Nunca precisa ser aberto** — não há narração; a mesa já vê na resolução. Continua **abrível** (2026-10-13), como a reação padrão: posicioná-lo na cadeia ajuda o mestre a controlar o tempo de resolução dessa escolha. |
| D3 | **O mestre pode dar a palavra à reação padrão** — um personagem pode esquivar de muitos jeitos. A decisão é dele; a possibilidade não some. |
| D4 | **A reação que conta é a que o jogador mandou.** Uma reação anexada e não aberta **entra no cálculo** (era um bug: ela cobrava as barras e era ignorada). |
| D5 | **Viés num teste passivo o torna rolado** (P2): vantagem fica com o melhor entre o dado e o valor base (11 no 2D10), desvantagem com o pior. Vale para todas as leituras passivas. Não rolar de novo o que já caiu. |
| D6 | **O mestre edita a ação na fila**: condições e a perícia do dano. Alvos ficam para depois — o desenho deve deixar fácil liberar. Editar a velocidade reordena a fila. |
| D7 | **O dono não vê edição nenhuma** na fila. Se ele cancelar a ação, ou uma reação a consumir, as edições somem com ela. |
| D8 | Fora de escopo: a fuga somar o movimento à esquiva (regra de colisão que o dono ainda vai desenhar). |

## 2. A reação padrão

- **`action.ReactDefault` ("default")**: gratuita, mantém a defesa padrão, não desloca, exige um
  `dodge` (sempre `Reflex`). `ReactionKind.IsPassive()` a distingue; `AwaitsNarration()` é falso só
  para `nothing`.
- **Sentada pela sessão.** `MatchSession.seatDefaultReactions` roda quando o turno abre e quando o
  mestre edita os alvos da ação do turno: senta uma para cada personagem que o ataque nomeia e não
  tem reação; tira a sentada de quem deixou de ser nomeado. Só ataques sentam (só eles colidem).
- **O `Turn` sabe quais são sentadas** (`Seat`, `IsSeated`, `SeatedFor`, `ReplaceSeated`,
  `Unseat`, `AwaitingTheFloor`).
- **A resposta substitui** (`AttachReaction`): a reação que chega toma o lugar da sentada,
  inclusive na ordem de abertura; as edições do mestre na sentada caem. Declarar `default` herda
  os dados e as edições da sentada (`inheritSeated`) — nada rola de novo (D5).
- **Fechamento:** `close_turn` só pede confirmação por reação que um jogador mandou e que tem o que
  narrar — nem a sentada, nem o `nothing`.
- **Reconexão:** `ownReactions` não lista a sentada (o cliente ainda oferece os botões).
- **Persistência:** nada novo. É uma linha de `actions` com `reaction_kind = 'default'`, como
  qualquer reação; `overridden_action_values` e o histórico já funcionam.

## 3. A cadeia (D4)

`buildChainOrder`: as abertas, na ordem em que o mestre abriu; depois os demais alvos, na ordem
de `targetId`, **cada um com a reação anexada a ele, aberta ou não**. `CharacterResult` ganha
`ReactionOpened`; o wire, `targets[].reaction.opened`. `pendingReactions` lista toda reação não
aberta (a padrão sentada e o `nothing` inclusive), com `seated`.

**O `nothing` continua abrível** (dono do produto, 2026-10-13). Um primeiro corte recusava o
`open_reaction` nele; isso tirava um poder que o mestre tinha: abri-lo antes de um aparo para o
golpe pegá-lo cheio. Ele nunca precisa da palavra e não segura o fechamento (`AwaitsNarration`),
mas o mestre pode abri-lo para posicioná-lo.

## 4. Viés numa leitura passiva (D5)

`RollCalculator.Derive`, `readPassiveBias`: com `Passive`, um viés líquido e um conjunto primário já
caído, o `DiceTotal` é `max(base, dados)` na vantagem e `min(base, dados)` na desvantagem; `Dice`
mostra o dado; o crítico só conta quando o dado é o que vale. Sem dado, o valor base fica. **O viés
líquido é o do mestre e o do sistema somados** (dono do produto, 2026-10-13: a Desvantagem da troca
vale numa leitura passiva como o viés do mestre, e a Vantagem do mestre sobre ela lê o valor base).
O dado do sistema cai em `deriveSpeeds` (o único lugar onde ele enviesa), pelo mesmo
`ensureDiceForBias`.

O dado cai em `ApplyMasterAction`, `ensureDiceForBias`: a primeira vez que uma condição com viés
pousa num `RollCheck` sem dados, rola-se o conjunto primário — uma vez, nunca mais (o mesmo
princípio da perícia acrescentada). Leituras passivas: a esquiva e a defesa da `default`, a defesa
padrão atrás de `dodge`/`closedDodge`/`escapeGuard`, o `moveSpeed` do Shift (`closedEscape`) e a
`speed` fora do Race. Um `DefaultDefense` cuja condição é zerada não é apagado se já tem dado. O
valor base vai em `match_full_state.passiveValue` (11 no 2D10), para o editor do mestre nomeá-lo.

## 5. Edição na fila (D6, D7)

- `ApplyMasterAction` encontra o alvo por `editTarget`: um `actionId` que está na fila vale com ou
  sem turno aberto; o resto continua exigindo o turno aberto. Na fila, o resultado é `nil` (não há
  turno a resolver) e não há `AddMasterAction`.
- **Recusados na fila:** `targetIds` e `skills` (`ErrQueuedEditUnsupported`). Chegam ao dono antes
  do fechamento (`ownQueue` em `Declaration`, `turn_opened` em `Opened`). Para liberar: projetar a
  declaração (os originais das capturas) nessas duas superfícies; a edição já funciona.
- **Velocidade reordena:** `deriveSpeeds` roda depois da edição e o escalonador lê a velocidade na
  hora de escolher. Num Race, o preço congelado não se mexe.
- **As capturas** ficam em `s.overrides` pelo ID da ação e são drenadas no fechamento do turno dela.
- **Sair sem abrir descarta:** `CancelAction` e `consumePendingFor` chamam `discardEdits`, que
  escreve de volta os originais (`restoreOriginal`) e descarta as capturas. O `cancelled_actions`
  grava a declaração.
- **Wire:** `action_queued` ganha `conditions` e `damageSkill` (master-only) e é reenviado ao mestre
  depois de cada edição — um `actionId` conhecido é atualização. `action_edited` vai com `turnId`
  zero, seguido de `bars_updated`.

## 5b. Repelir quem o ataque da fila mirava (2026-10-13)

Uma reação cobrada que consome uma ação da fila custa a Desvantagem da troca — salvo quando ela é o
que o personagem já tinha na fila, só que antes. `keepsQueuedIntent` reúne os dois casos: a fuga
para exatamente onde o movimento da fila ia (`sameMoveAsQueued`, 2026-10-11) e, novo, um `repel`
contra o atacante quando toda ação consumida é um ataque puro (sem movimento nem interação) cujo
único alvo é esse atacante (`sameBlowAsQueued`). O mestre pode pôr Desvantagem mesmo assim.

## 6. Limites conhecidos

- O que a mesa vê de uma edição depois do fechamento são os **números** (a resolução liquidada); a
  condição e o motivo continuam do mestre, como em toda edição desde a Fase 8.
- A ordem é pública: um `speed` editado na fila move a chave em `bars_updated` (o porquê não vai).
  O `turn_opened` leva ao dono a velocidade da ação no corte `Opened`, com a edição aplicada.

## 7. Testes

- Domínio: `TestRollCalculator_Derive_PassiveWithTheMastersBias`, `TestTurn_SeatedReactions`,
  `TestChain_UnopenedReactionsAreInTheCalculation`.
- Sessão: `default_reaction_test.go` (senta sem rolar; o viés rola uma vez; a resposta substitui;
  declarar `default` herda; `nothing` é abrível e não segura o fechamento; segue os alvos), `queued_edit_test.go` (pousa e
  vai ao turno; velocidade reordena; alvos/perícias recusados; cancelar e consumir descartam).
- E2E: `TestE2E_EditQueuedAction`, `TestE2E_ReconnectingTargetWithOnlyTheSeatedDefaultHasNoOwnReaction`;
  ajustados `TestE2E_CloseTurn`, `TestE2E_AreaAttackWithThreeTargetsReactingDifferently`,
  `TestE2E_ReconnectingTableGetsOpenedReactionsInOpeningOrder`.
- Integração: `TestPersistTurnClose_TheDefaultReactionIsARealReaction`.
