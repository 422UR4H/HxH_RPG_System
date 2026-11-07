# Stacked moves, move-only speed and the bars strip — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** a move rolls only its moveSpeed (also in Free), stacked moves chain from the previous
destination, open in order and cancel together, and the bars strip shows the next action's total in
the bar colours, aligned, with the project tooltip.

**Architecture:** back first (`System_X_System/`, Go), contract in
`docs/dev/api/match-combat-ws.md`, then front (`System_X_System_React/`, React/TS). Both repos are on
branch `fix/live-bars` (back PR #101, front PR #102). Commit to that branch; never push.

**Tech Stack:** Go 1.23 standard `testing`; React 18 + TS strict + styled-components + vitest.

**Spec:** `System_X_System/docs/superpowers/specs/2026-11-06-stacked-moves-design.md` (D1–D7). Read
the decisions your task names before coding.

## Global Constraints

- Code identifiers and test titles in English; comments, UI strings and docs in PT-BR.
- Go: `go vet ./...` and `go test ./...` green after each back task; also `go vet -tags integration ./...`
  when a struct that is persisted (`action.Move`) changes.
- Front: `npx tsc --noEmit`, `npx eslint <touched files>`, `npx vitest run` green after each front task.
- Wire format camelCase; new optional fields `omitempty`.
- Contract changes go into `System_X_System/docs/dev/api/match-combat-ws.md` in the same task; dev docs
  (`docs/dev/match/combat-engine.md` etc.) when the engine behaviour they describe changes.
- Do not weaken existing tests to make them pass: a test that asserted Legerity on a pure move is
  updated to the new rule (D1) and says so; anything else failing is a bug in your change.
- Never remove TODO comments. No subagents.

## Review Focus

1. A combined move+attack still rolls Legerity and keys on both bars (D1/D2) — regression risk.
2. A chain of three moves: cancelling the middle one cancels only the third; the first stays (D5).
3. A stacked move whose predecessor opens: it becomes a normal candidate in the same round (D4).
4. A reaction escape to the head's destination keeps the rest of the chain; to another slot cancels
   it (D6).
5. Strip with one row whose action bar has a speed and another row with nothing: the move bars are at
   the same x (D7).

---

### Task 1: Back — a move has no actionSpeed; Free orders by the bars' speed (D1, D2)

**Files:** `internal/app/game/action_mapper.go`, `internal/domain/match/matchsession/match_session.go`
(`rollActionDice`, `deriveSpeeds`, `resolveRollCheck`), `internal/domain/match/entity/action/priority_queue.go`,
`internal/domain/match/entity/action/bar.go`, tests next to each; docs `docs/dev/match/combat-engine.md`,
`docs/dev/api/match-combat-ws.md` (the `speed` field of an action: empty on a move-only action).

**Interfaces — Produces:**
- `func (a *Action) ChargesActionBar() bool` (exported; `Bars()` contains `BarAction`) — reuse the
  existing unexported helper or `Bars()`; one source.
- `func (a *Action) OrderSpeed() int` — `min SpeedOn(bar)` over `Bars()`, 0 when `Bars()` is empty.
  `reactionKey` in `turn_resolver.go` should reuse it if it is the same rule.

- [ ] Tests first: pure Dash in Race → `Speed.Attempts` empty, `Speed.SkillName == ""`, `Speed.Result == 0`,
  `Move.FinalSpeed` rolled from Accelerate; Shift → Brake passive; attack and combined → Legerity as today;
  escape reaction → no Legerity; condition edit `speed` on a pure move → `ErrConditionTargetMissing`;
  Free queue: a pure move with moveSpeed 20 opens before an attack with Legerity 15, and a combined action
  keys on the lower of its two speeds; a scripted `RollSource` test proves a pure move consumes no dice
  for Legerity.
- [ ] Implement. Update tests that pinned the old behaviour, each with a one-line PT comment of why.
- [ ] Docs; `go vet ./... && go test ./...`; commit `fix(match): a move rolls only its moveSpeed, also in Free`.

### Task 2: Back — stacked moves (D3, D4, D5, D6)

**Files:** `internal/domain/match/entity/action/` (Move: `StackedOn uuid.UUID`, json `stackedOn,omitempty`),
`internal/app/game/room.go` (enqueue arm: origin + wall check from the chain tail; cancel arm: one
`action_cancelled` per cancelled action; reaction attach: cascaded cancels), `internal/domain/match/matchsession/match_session.go`
(`EnqueueAction` or a helper that finds the chain tail; `CancelAction` cascade; `PullAction` refusal;
`consumePendingFor` rule), `internal/domain/match/service/round_scheduler.go` (blocked candidates in
`best`, `ProjectRound`, `BestPendingFor`), `priority_queue.go` (Free `ExtractMax` skips blocked),
`internal/app/wire/actionwire/` (`move.stackedOn`), errors (`ErrStackedOnPending` → `game_error`),
cancelled-actions recording (each cascaded action recorded like a requested cancel), tests (unit + room e2e),
docs: contract (`enqueue_action` origin, `pull_action` error, `cancel_action` cascade, `attach_reaction`
cascade, `move.stackedOn`), `docs/dev/match/combat-engine.md`, `docs/game/combate/` (player rule:
"o segundo movimento parte de onde o primeiro termina, acontece depois dele, e cancelar o primeiro cancela
o segundo"), `docs/documentation-map.yaml` if a new file is mapped.

**Interfaces — Produces (consumed by Task 3 and the front tasks):**
- `action.Move.StackedOn uuid.UUID`; wire `move.stackedOn?: string`.
- `func (s *MatchSession) CancelAction(playerUUID, actionID uuid.UUID) ([]*action.Action, error)` — the
  requested action first, then its dependents in chain order (signature change; update callers).
- Reaction attach result exposes the cascaded cancels so `room.go` can send `action_cancelled` for each
  (to the master and the reactor's owner) and record them.
- `ErrStackedOnPending`.

- [ ] Tests first (domain): enqueue two moves → second `StackedOn` = first id and `From` = first
  `Position`; third stacks on second; a non-move action between them does not break the chain and is not
  stacked; Race: second faster than first still opens after it, and `ProjectRound` lists it after the
  first; first deferred → second deferred; Free: same order rule; `PullAction(second)` →
  `ErrStackedOnPending` and the queue is unchanged; cancel first of three → returns all three in order;
  cancel middle → middle + third; reaction escape to head destination → head consumed, second re-pointed
  to `uuid.Nil` and still queued; escape elsewhere → second cancelled; `BestPendingFor` picks the head even
  when the stacked one is faster.
- [ ] Tests (room/e2e): the enqueue wall check runs from the chain tail (a wall between the tail and the
  new destination refuses; one between the piece and the new destination does not); cascade cancel sends
  one `action_cancelled` per action to master and requester, then `bars_updated`; `pull_action` refusal
  reaches the master as `game_error`.
- [ ] Implement; docs; `go vet ./... && go vet -tags integration ./... && go test ./...`; commit
  `feat(match): stacked moves chain, open in order and cancel together`.

### Task 3: Back — the bar balance counts only the next pending (D7 server)

**Files:** `internal/domain/match/matchsession/bars_view.go` + `bars_view_test.go`, contract
`bars_updated` section (`actionBalance`/`moveBalance` meaning, the worked example recomputed),
`docs/game/combate/barra-de-acao.md` if it describes the shown balance.

- [ ] Test first: master with two pending speeds on one bar → `Balance` = `BarEconomy.Balance(carry,
  acted ++ pending[:1], price)`; player unchanged.
- [ ] Implement; docs; `go test ./...`; commit `fix(match): bar balance counts only the next action`.

### Task 4: Front — move-only cards and the stacked wire field (D1, D3 wire)

**Files:** `src/features/match/combat/combatMessages.ts`, `normalizeWire.ts`, `QueuePanel.tsx`,
`queueOrder.ts` (Free order = D2 rule), every other place that prints the action's `speed`
(grep `speed?.rollCheck`, `ResolutionDetails`, `balloonText`, `combatText`), tests.

- [ ] Tests first: QueuePanel detail for a pure move shows "Velocidade de movimento" and NOT "Velocidade de
  ação"; combined shows both; `queueOrder` in Free orders a pure move by its move speed; normalize keeps
  `move.stackedOn`.
- [ ] Implement; checks; commit `fix(match): a move shows only its move speed`.

### Task 5: Front — the bars strip (D7)

**Files:** `src/features/match/combat/CharacterBarsStrip.tsx`, `barsMath.ts`, `src/styles/tokens.ts`
(semantic tokens for the two bar colours and their track tint, per `src/styles/CLAUDE.md`), tests
(`__tests__/CharacterBarsStrip.test.tsx`, `barsMath.test.ts`), dev doc
`docs/dev/match/combate-polimento-mesa.md` ("Barras ao vivo" section).

**Interfaces — Produces:** `barTotal(c, bar): number | undefined` and `barScale(rows, bars, bar): number`
in `barsMath.ts`.

- [ ] Tests first: total = carry + mean(acted ++ pending[:1]) (pending beyond the first ignored); player
  row (no pending) = carry + mean(acted); scale = max over rows of max(total, price); the number beside
  the bar is the total and becomes "saldo X" on hover/focus; one tick per price multiple inside the scale;
  every row renders the same number of grid cells (empty bars included) so columns align; tooltip uses the
  project `Tooltip` with "Ação: saldo 6 de 9"; no `title` attribute left; no green anywhere; `x̄` gone.
- [ ] Implement: one CSS grid for the strip, rows as `display: contents`; numbers `font-variant-numeric:
  tabular-nums`, `min-width` 3ch for speeds and 5ch for the total (room for one decimal); fill = bar colour,
  track = same hue low alpha; hover/focus darkens `[0, spent]` and highlights the balance; the bar box is
  focusable for keyboard.
- [ ] Checks; commit `feat(match): bars strip shows the next action's total, aligned, in the bar colours`.

### Task 6: Front — stacked moves (D3–D6 front)

**Files:** `useActionComposerState.ts` (composition origin = the actor's chain tail destination: the last
pending own move — declared, not yet opened — else the piece; used by the arrow, reach and
`isPathClear`), `useGameTable.ts` (ghost origin: a stacked move's ghost starts at its predecessor's
destination while the predecessor is pending; master queue ghosts by `move.stackedOn`), `CancelDeclaredDialog.tsx`
(+ the master's NPC cancel path if separate): lists the moves that go with it and confirms "Cancelar os N
movimentos"; reaction dialog (`ReactionConfigDialog`/`useReactionControls`): escape whose destination is
not the head's destination while the head has stacked moves → warning that they will be cancelled;
`matchErrorMessages`/`combatErrorMessages.ts`: PT message for `"action is stacked on a pending move"`;
tests; dev doc.

- [ ] Tests first for each behaviour above (pure helpers where possible: `chainTailOf`, `stackedAfter`).
- [ ] Implement; checks; commit `feat(match): stacked moves start where the previous one ends and cancel together`.
