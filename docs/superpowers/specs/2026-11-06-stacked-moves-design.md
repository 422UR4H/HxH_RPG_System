# Stacked moves, move-only speed, and the bars strip — design

**Date:** 2026-11-06 · **Branches:** `fix/live-bars` (back PR #101, front PR #102) · **Builds on:**
`2026-11-04-live-bars-design.md` · PT-BR: `2026-11-06-stacked-moves-design.pt-br.md`

## Why

The product owner ran a table and saw four problems:

1. A pure move showed the character's **Legerity** on the action card. A move has no actionSpeed: it
   rolls only its moveSpeed (Accelerate for Dash, Brake for Shift — keep that).
2. In **Free** the queue is ordered by `Action.Speed.Result` (passive Legerity) for every action,
   pure moves included. It must order by the same speed combat uses.
3. **Stacked moves**: the second move of a character is drawn from the piece, not from where the first
   move ends; and a faster second move opened BEFORE the first (the piece jumped to the final slot,
   the first move's arrow kept pointing at the first destination). A second move cannot happen before
   the first.
4. The bars strip: wrong colours, bars that shift sideways when a speed appears, the native `title`
   tooltip, and a live balance drawn "in advance".

## Decisions (owner, 2026-11-06)

### D1 — A move has no actionSpeed
An action rolls and derives `speed` (Legerity) **only when it charges the action bar**
(`slices.Contains(a.Bars(), action.BarAction)`): attack, interact, feint, skills, defense, dodge,
repel, or a combined move+attack. A pure move (and an escape reaction, whose bars are move-only)
carries an **empty** `speed.rollCheck` (no skill, no dice, result 0). Consequences:
- The mapper stops stamping Legerity on actions that do not charge the action bar.
- `rollActionDice` does not roll `Speed` for them; `deriveSpeeds` does not derive it.
- A master condition edit on field `speed` of such an action is refused with the existing
  `ErrConditionTargetMissing` (as `feint` on an action without feint).
- The front shows "Velocidade de ação" only for actions whose `bars` include `action`.

### D2 — Free orders by the same speed as combat
Free still has no price, mean or carry. Its order key is the **rolled speed on the action's bars,
the lowest when it has two** — exactly `reactionKey`'s rule (`min SpeedOn(bar)` over `Bars()`): a
pure move by its moveSpeed, an attack by its actionSpeed, a combined action by the lower of both.
`PriorityQueue.ExtractMax` uses that key (ties: first inserted, as today). The front's queue order
(`queueOrder.ts`) must agree.

### D3 — A move stacks on the character's last pending move
When a character enqueues an action with a `move` while they already have a pending action with a
`move` in the queue, the new move is **stacked** on the LAST such pending action (the chain tail):
- `Move.StackedOn` = that action's id (new field; `uuid.Nil` = not stacked).
- `Move.From` = that action's `Move.Position` (not the piece's slot). The wall check at enqueue runs
  from there. A move with no pending move before it keeps today's origin (the piece).
- Only moves stack (pure or combined). A non-move action never stacks and is never stacked on.
- Wire: `move.stackedOn` (uuid, omitempty) on every surface that already carries the action's `move`
  to its owner and to the master. Players never see another character's pending actions (unchanged).

### D4 — A stacked move never opens before the one it stacks on (and the delay is the price)
An action whose `StackedOn` is **still in the queue** is **blocked**: it is not a candidate.
- `SelectNext`/`best`, `ProjectRound` (simulated: blocked while its predecessor is still unpicked in
  the simulation), `AnyEligible`, and Free's `ExtractMax` all skip blocked actions. If the predecessor
  is deferred to the next round, the stacked move is deferred with it.
- `pull_action` of a blocked action is refused: new domain error `ErrStackedOnPending`
  (`game_error`, message `"action is stacked on a pending move"`), nothing changes.
- **No new penalty.** Once the first move opens it is an acted speed and pays the price; the second is
  then keyed with that price already counted (`BarEconomy.Key`, the rule every second action of a
  round already follows). Example: price 9, moves 12 then 20 → the 12 opens; the 20 then keys
  `(12+20)/2 − 9 = 7`.

### D5 — Cancelling a move cancels what is stacked on it
`cancel_action` on an action that has stacked dependents cancels the whole chain after it
(transitively), in chain order. Each cancelled action goes through today's cancel path: removed from
the queue, edits discarded, recorded in `cancelled_actions`, one `action_cancelled` per action (the
master and the requester), then one `bars_updated`. The swap Disadvantage mark is what it is today
(one mark, same character). The front **warns before confirming**: the cancel dialog lists the
stacked moves that go with it and asks the user to confirm.

### D6 — A reaction spends the first move of the chain
`BestPendingFor` (the action a charged reaction consumes) only considers **unblocked** actions — the
chain head is "the moment about to be spent". (Fallback: when the bar has no unblocked candidate, any
pending action on that bar, as today.)
After a reaction consumes an action that has stacked dependents:
- If the reaction **lands where the consumed move was going** (the reaction's `Move.Position` equals
  the consumed action's `Move.Position` — the same test `sameMoveAsQueued` uses), the dependents
  **survive**: the first of them is re-pointed to the consumed action's own `StackedOn` (usually
  `uuid.Nil`, so it becomes the head). No penalty (the escape kept the queued intent, as today).
- Otherwise the dependents are **cancelled** through the D5 path (no extra penalty beyond today's swap
  Disadvantage).
The front's reaction dialog warns, for an escape, when the escape's destination is not the head's
destination and the head has stacked moves: those moves will be cancelled.

### D7 — The bars strip
- **Colours:** action bar = the attack red of this screen (`mapAttackIntent`), move bar = the move blue
  (`mapMoveIntent`), through semantic tokens per `src/styles/CLAUDE.md`. Empty track = the same hue at
  low alpha (the bar keeps its identity when empty and stays quiet on the dark panel). No green.
- **What the bar shows (owner chose the prototype):** the **total** of the NEXT action —
  `total = carry + mean(acted ++ [next])`, where `next` is the first pending speed (the master's
  `pending` list is already in projected order, so `pending[0]`); for a player (no pending) `total =
  carry + mean(acted)`. One tick at each multiple of the price. The number beside the bar is that
  total. The scale is shared per bar kind across the strip: full width = the largest `max(total,
  price)` among the rows, so characters compare at a glance.
- **Hover / focus:** the spent part (`spent = total − balance`) darkens, the rest stays lit, and the
  number switches to the balance, highlighted (bold, primary text) — "saldo 6". Negative balance:
  the whole fill darkens and the number reads e.g. "saldo −3".
- **Server:** `BarView.Balance` counts `acted ++ pending[:1]` (not every pending). Players are
  unchanged. Contract: `actionBalance`/`moveBalance` = the balance after the next action pays.
- **Tooltip:** the project `Tooltip` (`src/components/ions/Tooltip.tsx`), same wording as today
  (`"Ação: saldo 6 de 9"`).
- **Alignment:** the strip is one CSS grid; every row renders every column (empty cells when a bar has
  nothing), so the move bar never shifts because the action bar got a speed. Numbers use tabular
  figures with room reserved for 3 digits by default (the total also for one decimal); a wider value
  widens its column for **all** rows at once.
- The `x̄` text goes away: the total beside the bar replaces it.

## Out of scope / known limits
- A failed escape (or the master dragging the piece) leaves a stacked move's `From` at the declared
  chain origin. Destinations stay absolute, as today.
- Attack lines from a stacked position are not changed.
