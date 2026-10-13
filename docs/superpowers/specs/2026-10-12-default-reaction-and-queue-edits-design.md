# Default reaction, bias on a passive test, and queue edits — design

> **Scope:** the leftover the Phase 6 closure flagged ("the master's condition on dodge, defense or
> repel is accepted and changes nothing") had already been fixed in Phase 8 (PR #85) for **opened**
> reactions. What remained, decided by the product owner on 2026-10-12: the passive dodge and
> defense of a target that did not react, the bias on a passive reading, the unopened reaction left
> out of the calculation (a bug), and editing an action still in the queue. **One cross-repo task,
> one PR per repo**: this one (`HxH_RPG_System`) and the front's (`HxH_RPG_Environment_React`), same
> branch `claude/jolly-sagan-gvfth9`.
>
> Contract: [`../../dev/api/match-combat-ws.md`](../../dev/api/match-combat-ws.md) —
> `attach_reaction`, `open_reaction`, `edit_action`, `close_turn`, `action_queued`,
> `action_edited`, `resolution_updated`, `match_full_state.ownReactions`.
> Engine: [`../../dev/match/combat-engine.md`](../../dev/match/combat-engine.md) — "A reação
> padrão", "A edição na fila", "Rolagem".
> Rules: [`../../game/combate/reacoes.md`](../../game/combate/reacoes.md),
> [`../../game/combate/o-mestre.md`](../../game/combate/o-mestre.md).

## 1. Product-owner decisions (2026-10-12)

| # | Decision |
|---|---|
| D1 | **The passives are a reaction.** A target that did not answer has the default reaction — reflex dodge, then the default defense. It has a record, it is part of the turn as an answer to the action, the master edits it like any other, and the player sees it in the history after the close. |
| D2 | **"Do nothing" is an explicit choice** with no default dodge or defense: the blow lands raw. **It never needs to be opened** — there is no narration; the table already sees it in the resolution. It stays **openable** (2026-10-13), like the default reaction: placing it in the chain helps the master time the resolution of that choice. |
| D3 | **The master may give the default reaction the floor** — a character can dodge in many ways. The call is the master's; the possibility stays. |
| D4 | **The reaction that counts is the one the player sent.** An attached, unopened reaction **is in the calculation** (it was a bug: it charged the bars and was ignored). |
| D5 | **A bias on a passive test makes it rolled** (P2): advantage keeps the better of the dice and the base value (11 on 2D10), disadvantage the worse. It applies to every passive reading. What already fell is never rolled again. |
| D6 | **The master edits a queued action**: conditions and the damage skill. Targets come later — the design must make lifting that easy. Editing the speed reorders the queue. |
| D7 | **The owner sees no edit** on the queue. If they cancel the action, or a reaction consumes it, the edits go with it. |
| D8 | Out of scope: an escape adding its movement to the dodge (a collision rule the owner will design later). |

## 2. The default reaction

- **`action.ReactDefault` ("default")**: free, keeps the default defense, does not displace,
  requires a `dodge` (always `Reflex`). `ReactionKind.IsPassive()` tells it apart;
  `AwaitsNarration()` is false only for `nothing`.
- **Seated by the session.** `MatchSession.seatDefaultReactions` runs when the turn opens and when
  the master edits the turn's action targets: it seats one for every character the attack names
  that has no reaction, and removes the seated one of a character no longer named. Only attacks
  seat (only they collide).
- **The `Turn` knows which are seated** (`Seat`, `IsSeated`, `SeatedFor`, `ReplaceSeated`,
  `Unseat`, `AwaitingTheFloor`).
- **The answer replaces it** (`AttachReaction`): the incoming reaction takes the seated one's
  place, opening order included; the master's edits on the seated one are dropped. Declaring
  `default` inherits the seated one's dice and edits (`inheritSeated`) — nothing rolls again (D5).
- **Close:** `close_turn` only asks for confirmation for a reaction a player sent that has
  something to narrate — neither the seated one nor `nothing`.
- **Reconnect:** `ownReactions` does not list the seated one (the client still offers the buttons).
- **Persistence:** nothing new. It is an `actions` row with `reaction_kind = 'default'`, like any
  reaction; `overridden_action_values` and the history already work.

## 3. The chain (D4)

`buildChainOrder`: the opened ones, in the order the master opened them; then the other targets,
in `targetId` order, **each with the reaction attached for it, opened or not**. `CharacterResult`
gains `ReactionOpened`; the wire, `targets[].reaction.opened`. `pendingReactions` lists every
unopened reaction (the seated default and `nothing` included), with `seated`.

**`nothing` stays openable** (owner decision, 2026-10-13). A first cut refused `open_reaction` on
it; that took away a master power the table had: opening it ahead of a repel so the blow hits it
whole. It never needs the floor and never holds the close (`AwaitsNarration`), but the master may
open it to place it.

## 4. Bias on a passive reading (D5)

`RollCalculator.Derive`, `readPassiveBias`: with `Passive`, a bias **from the master**
(`Condition.Bias`) and a primary set already fallen, `DiceTotal` is `max(base, dice)` on advantage
and `min(base, dice)` on disadvantage; `Dice` shows the roll; a critical counts only when the dice
are what counts. With no dice, the base value stands. The system's and the ledger's bias still have
no effect on a passive reading (as before; the rule is the master's power to call for a roll).

The dice fall in `ApplyMasterAction`, `ensureDiceForBias`: the first time a condition with a bias
lands on a `RollCheck` with no dice, its primary set is rolled — once, never again (the same
principle as an added skill). Passive readings: the default reaction's dodge and defense, the
default defense behind `dodge`/`closedDodge`/`escapeGuard`, the Shift `moveSpeed` (`closedEscape`)
and `speed` outside Race. A `DefaultDefense` whose condition is zeroed is not dropped once it holds
dice. The base value itself travels in `match_full_state.passiveValue` (11 on 2D10), so the
master's editor can name it.

## 5. Queue edits (D6, D7)

- `ApplyMasterAction` finds its target through `editTarget`: an `actionId` in the queue works with
  or without an open turn; everything else still needs the open turn. On the queue the result is
  `nil` (no turn to resolve) and there is no `AddMasterAction`.
- **Refused on the queue:** `targetIds` and `skills` (`ErrQueuedEditUnsupported`). Both reach the
  owner before the close (`ownQueue` at `Declaration`, `turn_opened` at `Opened`). To lift it:
  project the declaration (the captures' originals) on those two surfaces; the edit already works.
- **Speed reorders:** `deriveSpeeds` runs after the edit and the scheduler reads the speed at
  selection time. In Race, a frozen price does not move.
- **Captures** stay in `s.overrides` under the action's ID and are drained at its turn's close.
- **Leaving without opening discards:** `CancelAction` and `consumePendingFor` call
  `discardEdits`, which writes the originals back (`restoreOriginal`) and drops the captures.
  `cancelled_actions` records the declaration.
- **Wire:** `action_queued` gains `conditions` and `damageSkill` (master-only) and is re-sent to the
  master after every edit — a known `actionId` is an update. `action_edited` carries a zero
  `turnId`, followed by `bars_updated`.

## 6. Known limits

- What the table sees of an edit after the close is the **numbers** (the settled resolution); the
  condition and its reason stay the master's, as with every edit since Phase 8.
- The order is public: a speed edited on the queue moves the key in `bars_updated` (not the why).
  `turn_opened` carries the action's speed to the owner at the `Opened` cut, edit applied.

## 7. Tests

- Domain: `TestRollCalculator_Derive_PassiveWithTheMastersBias`, `TestTurn_SeatedReactions`,
  `TestChain_UnopenedReactionsAreInTheCalculation`.
- Session: `default_reaction_test.go` (seats without rolling; the bias rolls once; the answer
  replaces; declaring `default` inherits; `nothing` is openable and never holds the close; follows
  the targets),
  `queued_edit_test.go` (lands and rides into the turn; speed reorders; targets/skills refused;
  cancel and consume discard).
- E2E: `TestE2E_EditQueuedAction`, `TestE2E_ReconnectingTargetWithOnlyTheSeatedDefaultHasNoOwnReaction`;
  adjusted `TestE2E_CloseTurn`, `TestE2E_AreaAttackWithThreeTargetsReactingDifferently`,
  `TestE2E_ReconnectingTableGetsOpenedReactionsInOpeningOrder`.
- Integration: `TestPersistTurnClose_TheDefaultReactionIsARealReaction`.
