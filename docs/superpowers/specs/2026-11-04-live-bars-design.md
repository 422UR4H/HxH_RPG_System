# Live bars, projected per recipient — design

> Both repos. Back: `bars_updated` / `match_full_state.bars` (`internal/app/game/`,
> `internal/domain/match/`). Front: `src/features/match/combat/` (`GeneralBar`,
> `CharacterBarsStrip`, `OwnBars`, `barsMath`). Branch `fix/live-bars` in both.

## Problem

The header bars (the strip under the general bar) do not show a character when they send an
action — only after the master opens it — and the "balance" they show is frozen for the whole
round. At the same time, the general bar leaks secret information to the players: every pending
action of every character, with a number that reveals its rolled speed.

## Decisions (product owner, 2026-11-04)

| # | Decision |
|---|---|
| D1 | **Pending actions are the master's secret.** The master sees every send live — their own (NPCs) and the players'. A player sees only the cards of what THEY sent and what the master has already opened. Other characters' pending actions are invisible to players until opened. |
| D2 | **No rolled die reaches a player before the master opens the action — not even of their own action.** Otherwise a player could cancel an action whose dice rolled badly ("anti game"). This covers speed, hit, damage, any test. The player's own sent action keeps showing as today: the card, with no number. |
| D3 | The bar shows the **live balance** (Disputed regime): `carry + mean(speeds) − n × price`, moving with every change. For the master it includes the pending actions that will open this round; for a player only what is already public. |
| D4 | Before the first open freezes the price, the master sees a **provisional price** — the smallest pending speed, which is exactly what will freeze — marked as provisional. |
| D5 | In the **Free** regime the strip of bars is hidden (no price, no average there). |
| D6 | The back keeps the exact fractional math (it orders the queue). Rounding is display-only, one decimal, as the rest of the screen already does. |
| D7 | **The frozen price stays public**, as today: players see it once the master opens the round's first action. Accepted knowingly: in a small table a player can sometimes deduce that the price is their own pending speed (or that their speed is at least the price). The player sees their own action speed and move speed only when that action is opened. |

D1 corrects `docs/game/combate/barra-de-acao.md` ("Enxergando a barra"), which said everyone sees
the general bar with everyone's order. That text is rewritten to D1.

## Findings (what is wrong today)

| # | Finding | Where |
|---|---|---|
| F1 | Speeds are recorded only when the master opens an action (`recordActed`), and the price freezes only on the first open. Until then a row has no speed and no price, and the strip hides it (`hasBarsData`). Nothing appears on send. | `match_session.go`, `barsMath.ts` |
| F2 | `actionBalance`/`moveBalance` on the wire is the **carry** from the previous round (`ResourceBar.Balance` only changes in `settleBars`). The bar and the player's "Saldo" panel do not move during the round. | `room.go newBarsUpdatedPayload`, `CharacterBarsStrip`, `OwnBars` |
| F3 | `bars_updated` is broadcast identically to everyone, including `order` — actor, bar and key of every pending action. Before the price freezes the key is `carry + speed`, and `carry` is public, so the rolled speed is readable by everyone. Violates D1/D2. | `broadcastBars` |
| F4 | A reaction's speed is recorded in the bar when it is **attached** (`chargeReactionBars` inside `AttachReaction`), not when it is opened; the next `bars_updated` publishes it in `actionSpeeds`/`moveSpeeds` to everyone. Violates D2. | `match_session.go` ~L1334 |
| F5 | Before the first open, `ProjectOrder` scores unfrozen bars with no price and no gate, so the published order differs from the played one — exactly the case its own doc comment describes (p2 23 & 17, p1 20, p3 11 publishes p2 → p1 → p2 → p3; the table plays p2 → p1 → p3 → p2). | `round_scheduler.go keyOnBar` |
| F6 | The order chip prints the raw key: measured `-5.333333333333332`, `-42.333333333333336`, `-65.47619047619048` (means divide by the action count; fractional carry crosses rounds; plus float noise). | `GeneralBar.tsx` `<Key>` |
| F7 | The strip's `x̄` averages action and move speeds together, and the speed list does not say which bar each number belongs to. Each bar has its own speed. | `CharacterBarsStrip.tsx` |
| F8 | `characters` comes from iterating a Go map (`CharacterIDs`), so the row order changes randomly between updates and the rows jump. | `match_session.go CharacterIDs` |
| F9 | In Free nothing is ever recorded, so the strip never shows anything there, contrary to its own comment. Resolved by D5. | `CharacterBarsStrip.tsx` |

## Design

### Back

1. **Provisional price.** Extract from `FreezePrices` the computation "smallest pending speed per
   unfrozen bar" (Race only) into a read-only function. `FreezePrices` uses it to freeze; the
   projection uses it as the provisional price.
2. **Projection uses the provisional price (F5).** In a Race round, `ProjectOrder` scores an
   unfrozen bar against its provisional price — key AND gate — so the published order is the one
   the master will play if nothing else arrives. `SelectNext` is unaffected (it runs after
   `FreezePrices`). Free stays as is.
3. **What the projection reports.** For each pending action the simulation already decides
   whether it opens this round. It must also expose, per character and bar, the speeds of the
   actions it opens (in the order it opens them) and of those it leaves for the next round, so
   the payload can be built without a second scoring path.
4. **Live balance.** `BarEconomy.Balance(carry, speeds, price)`, no ceiling (the ceiling only
   applies at round close, `CloseBalance`). With no price on that bar (frozen or provisional) the
   live balance is the carry. Rule example: p2 alone at 23, price 11 → +12; with the second
   action at 17 → −2.
5. **Public vs. secret speeds (F4).** A speed is public once its action or reaction has been
   opened. The bar state must be able to tell the public part from the rest for the player view
   — today `ResourceBar.Speeds` is a plain `[]int` with no identity. How to track it is the
   builder's call (e.g. record a reaction's speed with a "not yet public" mark and flip it on
   `open_reaction`, or keep the reaction speeds separately). The ECONOMY keeps using every speed,
   exactly as today; only the projection to players filters.
6. **Projection per recipient (F3).** `bars_updated` leaves `r.broadcast` for the per-recipient
   lane (`dispatchPerPlayer`), like `turn_opened`. `match_full_state.bars` builds the same view
   for its recipient. `seq` is still bumped once per snapshot under `r.mu`, shared by every
   recipient's copy. Master view vs. player view: see the wire table below.
7. **Deterministic order (F8).** `characters` sorted by `characterId`.

### Wire (`bars_updated` / `match_full_state.bars`)

Per character:

| Field | Master | Player |
|---|---|---|
| `characterId` | ✓ | ✓ |
| `actionCarry` / `moveCarry` — **new**: what crossed from the previous round (credit or debt) | ✓ | ✓ |
| `actionBalance` / `moveBalance` — **meaning changes**: live balance (D3), not the carry | acted + pending that open this round | public speeds only |
| `actionSpeeds` / `moveSpeeds` — opened this round, in order | all | public only (F4) |
| `pendingActionSpeeds` / `pendingMoveSpeeds` — **new**: queued, open this round, projected order | ✓ | absent |
| `deferredActionSpeeds` / `deferredMoveSpeeds` — **new**: queued, do not fit this round | ✓ | absent |

Top level:

| Field | Master | Player |
|---|---|---|
| `seq`, `roundTurns` | ✓ | ✓ |
| `prices` — frozen prices (unchanged) | ✓ | ✓ |
| `provisionalPrices` — **new**: bars not frozen yet, Race only | ✓ | absent |
| `order` | ✓ (keys now use the provisional price, F5) | `[]` |

Combined actions keep charging both bars: their speed on each bar appears in that bar's lists.
Arrays are never `null` on the wire for the new fields (empty `[]` or omitted when master-only,
the builder picks one and the contract states it; the front's `normalizeBars` keeps defending).

### Front

1. **Types and normalization** follow the wire table (`combatMessages.ts`, `normalizeWire.ts`).
2. **`GeneralBar`.** Order chip key printed with `fmt` (one decimal, F6). The "Fila" segment
   only renders for the master (players receive `order: []`; "Fila vazia" must not show to
   them). Prices: frozen as today; provisional ones with a "(provisório)" mark. The "Barras"
   toggle and the strip are hidden in Free (D5).
3. **`CharacterBarsStrip`.** Row appears when the character has any speed (acted, pending or
   deferred) or the bar has a price — frozen, or provisional for the master. The bar uses the
   live `actionBalance`/`moveBalance` against the frozen or provisional price. Speeds grouped
   per bar (⚔ … / ➜ …): acted in normal style, pending lighter; deferred ones (master only) come
   after a "Próximo round" separator, not struck through (owner, 2026-11-04). One `x̄` per bar
   (F7), over the same speeds that compose that bar's balance. Rows sorted by display name
   (stable — F8; UX call: a reader tracks a bar by its position, and today's random reorder
   carries no information — who acts next is already in the order).
4. **`OwnBars`** shows the live balance (same field, new meaning).
5. **`QueuePanel`** (master's queue): the per-row "próxima rodada" badge becomes a single
   "Próximo round" separator before the deferred group — same treatment as the strip — and its
   "Ordem geral: chave …" line prints the key with `fmt` (F6, same raw number).

### Docs

- `docs/dev/api/match-combat-ws.md`: `bars_updated` rewritten (destination: per recipient; the
  table above; F5 note on keys; the old "broadcast, not projected" rationale removed).
  `match_full_state.bars` points to it.
- `docs/game/combate/barra-de-acao.md`, "Enxergando a barra": rewritten to D1/D2 in player
  language (the master sees the queue; the player sees their own cards and what was opened).
- `docs/dev/match/combat-engine.md`: provisional price and per-recipient bars, where it
  describes the general bar.
- Front: the combat dev doc that describes the general bar (`docs/dev/match/`).

## Non-goals

- No change to the bar economy (mean, price, gates, carry, ceiling).
- No rounding in the back.
- No economy for the Free regime.
- No change to what `turn_opened`, `action_queued`, `ownQueue` or `resolution_updated` reveal.

## Verification

- Back: unit tests for the provisional price, the projection with provisional price (the
  canonical p1/p2/p3 example must publish p2 → p1 → p3 → p2 before any open), the live balance,
  the public/secret speed split for reactions, and the per-recipient payload (master gets
  pending/deferred/provisional/order; a player gets none of them and no unopened reaction
  speed). Room-level test that a player's `bars_updated` after another player's
  `enqueue_action` carries no trace of it.
- Front: component tests for the strip (pending/deferred styles, per-bar mean, provisional mark,
  hidden in Free), `GeneralBar` (formatted key, no "Fila" for players).
- Browser: `go run ./cmd/devseed`, master and player side by side (`docs/dev/seed.md`): the player
  sends → the master's strip shows the row with the pending speed and provisional price at once;
  the player's header shows nothing new. Master opens → the player's header shows the row.

## Settled after the builder check (author's calls, 2026-11-04)

- **When a reaction's speed turns public.** When the table learns the reaction: on
  `reaction_opened`, or when the turn closes (the settled resolution goes to the table) for a
  reaction that was never opened (that happens: the master may close the turn over unopened
  reactions after `close_turn_refused` and `confirm: true`). A free reaction (`nothing`, the seated `default`) records no
  speed, so it never matters. A player's own reaction follows D2 like their own queued action.
- **A player's balance can jump.** Hidden speeds are left out of the player's number until they
  turn public; the jump when they do is the expected consequence of D2, not a bug.
- **Pending vs. deferred.** Every queued action the projection opens is pending; everything left
  when it stops is deferred. A combined action clears both gates or neither (`keyOf`), so it is
  pending on both bars or deferred on both.
- **A bar with no price** (no frozen, no provisional) shows no meter, only its speeds if any —
  as the strip already does today. The row still appears when the other bar has a price or any
  speed exists.
- **Free regime.** The back sends the payload as in Race (prices empty, no provisional); the
  front hides the strip. D1 holds in Free too: players get `order: []`.
- **Recipients.** `dispatchPerPlayer` knows two kinds: the master and a player. A player sees
  every character — their own included — through the same player view (D2); owning several
  characters changes nothing.

## Left to the builder

- How the bar state marks a speed as public (Design 5).
- Whether master-only fields are omitted or empty for players (state it in the contract).
- Exact visual of pending/deferred speeds and the provisional mark, within the existing tokens.
