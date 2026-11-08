# Story clock, end of combat and ending the match — design

> English version of `2026-11-08-story-clock-design.pt-br.md`; the Portuguese file is the original.

> 2026-11-08. Brainstorm with the product owner (session of 07→08/11). Origin:
> `System_X_System_Project/docs/handoffs/2026-11-07-contador-de-tempo.md` and
> `.../2026-10-26-encerrar-partida.md` (project root, not versioned).
> Decisions by the owner are marked **(owner)**; what I decided on my own is marked **(mine)** —
> he went to sleep trusting that I would take this all the way to the PRs, so these are the ones he will check.

## 1. What it is for

1. **Knowing what time it is in the story.** Today nobody knows how much time has passed inside the match:
   it stores only the *date* of the story start (`matches.story_start_at`, `DATE`). The clock is game time,
   never real-world time.
2. **Unblocking "end match"**, which requires the exact moment of the end in the story — and which today
   does not exist anywhere (neither back nor front).
3. **Converting combat into time.** Each contested round is worth 1 second, and closing the combat shows the
   master the *gaps* (pauses without contest) so that he can add up how long they lasted.
4. **Basis for statuses** (parallel session): bleeding and other "per round" effects need to answer "how many
   equivalent rounds have passed" outside combat. Since 1 round = 1 s, the answer is the difference in seconds
   between two instants of the clock.

**For whom:** the master controls; the players only watch — and **every change made by the master stays
explicit for them (owner)**, never goes unnoticed.

## 2. Vocabulary

| Term | Meaning |
|---|---|
| **Clock** (Relógio) | The current instant of the match's story (date + time down to the second). One per match. |
| **Real time (play)** (Tempo real) | The clock running alongside the world clock, for roleplay scenes. |
| **Limit** (piso) (Limite) | The instant before which the clock cannot go back. |
| **Counted round** (Round contado) | A round that had at least one turn opened in the contested regime. Worth 1 s. |
| **Gap** (Hiato) | A contested round that ended with the queue empty, until the first turn of the next round. |

## 3. Clock rules (back, source of truth)

### 3.1 Start
- The match gets an **optional start time in the story**, next to the date (owner). Creating and editing a match
  accept this field; editing remains refused after the start (current rule).
- When starting the match (`start_match`), if the match has a date **and** a time, the clock is born at that
  instant, **paused** (owner: "it always starts paused"). Limit = that instant.
- If only the date exists, the clock is born **with no time defined**. As soon as he enters the table, the master
  receives a **mandatory** dialog asking for the time (the date is the match's date, fixed) (owner).
  While there is no time, the server refuses with `clock_not_set` the master's commands that touch the timeline:
  `open_next_action`, `pull_action`, `change_round_mode`, `change_scene`, `end_match`, and the clock commands
  (except `set_clock_start`). Players can still declare, and the master can still act on the board
  (master actions do not move through time).
  **(mine)** — this is the only way for the limits below to hold from the first turn.
  `set_clock_start` with the time already defined → `clock_already_set`. A master who reloads the page without
  having defined the time gets the dialog again (`match_full_state` carries `storyAt: null`).
- Matches that had already started before this feature (dev data) fall into the same case: no time, mandatory dialog.

### 3.2 The limit (never going back before…) (owner)
The clock never goes back to before:
- the last action that **actually** happened — the limit rises when a turn **opens** and when it closes;
- the last scene change;
- the last regime change;
- (implicit) the start of the match and each counted round.

Each of these moments records `limit = current clock` with the **reason** (`start`, `turn`, `scene`,
`roundMode`, `round`), and the UI tells the master what the limit is and why. Real time running does **not**
raise the limit: the master may roll back the time that was allowed to run (owner), down to the limit.

**The limit of an open turn follows the house rule** (mine, resolving a conflict pointed out in review):
"inside an open turn nothing is recorded before it closes." On **opening**, the limit rises only in memory and
the turn stores the instant (`Turn.storyAt`); closing records both in the turn's transaction
(`turns.story_at` = the opening instant). A restart in the middle of the turn loses it entirely, and the limit
goes back together with the last recorded value — consistent.

A command that happens with an open turn and is recorded right away (`change_round_mode`, the clock commands)
records the limit in memory at that moment — including what the open turn raised it to.

### 3.3 Master changes
- **Relative adjustment** (`adjust_clock`, in seconds, positive or negative). Refused if the result would fall
  below the limit (`clock_below_limit`) or if `delta` is 0.
- **Going to an instant** (`set_clock`, absolute) — the direct edit in the UI. Same limit.
- **More than 2 days forward requires confirmation** (owner): the server refuses a forward move > 172 800 s
  without `confirmLongSkip: true` (`clock_long_skip_unconfirmed`). The UI asks first ("the characters lose the
  training days of this period"). Applies to every command that moves the clock forward (`adjust_clock`,
  `set_clock`, `change_scene` with `storyAt`, `end_match`). The base of the calculation is always the **current
  clock before the command** (without the automatic second of the round).
- They work with the clock running: they apply to the current value and it keeps running.

### 3.4 Real time (play)
- Exists only in the **free regime** (owner). `play_clock` in the contested regime → `clock_play_in_race`.
- **Always starts paused**; the master turns it on explicitly (owner).
- **Pauses by itself** (owner) on: regime change, scene change, master leaving the table (the master's last
  connection drops), ending the match.
- **Master leaves = immediate pause (mine)**, at the instant of the drop, with no grace period: an unstable
  reconnection costs the master one click on ▶; a clock running with no one watching is exactly what the owner
  wants to avoid. The room already knows when the master's connection leaves (`unregister`; an account has only
  one connection, B4 — being replaced by another tab does **not** count as leaving).
- While it runs, the server writes a **checkpoint every 60 s** (mine). If the server goes down, the room that
  comes back finds the clock "running" and **pauses it at the last checkpoint** (it loses at most 1 min, and never
  adds hours of server downtime) (mine).
- The current value is `storyAt + (now − runningSince)`, in whole seconds.

### 3.5 Combat: 1 round = 1 second (owner)
- Each **counted round** (had at least one turn opened while in the contested regime) adds **1 s** to the clock
  **when it closes** — by any path: the automatic end of `open_next_action`, `change_scene`, and `end_match`.
  A regime change in the middle of the round does not undo the mark: the round that already opened a turn in the
  contested regime counts, even if it ends in the free regime. This happens live: the clock moves along with the
  combat, and the limit rises along with it (reason `round`). It cannot be undone by the master (owner: "this
  minimal calculation must not go back").
- A round with no turns at all does not count (and today it does not even close: `ErrRoundHasNoTurns`).
- Players **do not** receive a warning per round: they only see the number change (owner).

### 3.6 Gap (owner, confirmed definition)
A **gap** is a **contested** round that ended through `open_next_action` with an **empty queue** — everyone has
already acted and no one has declared the next action — followed by a round with a turn in the same scene.
A round that ends with actions still in the queue (with no bar to pay) is **not** a gap. For each gap, the combat
summary gives: the **last turn before** (of the round that emptied the queue) and the **first turn after** (of the
next round with a turn), each with the round number within the scene, who acted, and the type of action. The system
does not know what happened in the gap; the master remembers and adds up the time.
A round that emptied and had **no successor with a turn** (the master changed scene right after) is not a gap:
the combat ended there. `rounds.drained` records only the fact (empty queue on close); whether it became a gap is
decided by the summary, looking at the following round.

## 4. End of combat

Combat = **scene of category `battle`**. Combat ends by changing scene.

- When the master opens "Nova cena" (New scene) and the current scene is combat, the front asks for the **combat
  summary** (`get_combat_summary`) and the dialog becomes **"Encerrar combate"** (End combat) with:
  - how many rounds were counted and how many seconds that gave — **already added to the clock** (the round in
    progress appears as "+1 s on ending" when it counts);
  - the list of gaps (last turn before / first turn after) and the message: *"During these intervals nobody
    contested actions — the characters may have talked, repositioned or looked for an opening. Add to the clock
    the time they lasted."* (Nesses intervalos ninguém disputou ações — os personagens podem ter conversado, se
    reposicionado ou procurado uma brecha. Some ao relógio o tempo que eles duraram.)
  - the **clock editor** starting at the minimum (current clock + the second of the round in progress, if it counts),
    unable to go below it: it is the **exact time the combat ended** (owner);
  - **optional final scene description** (255) — `scenes.brief_final_description`, which already exists and was
    never filled;
  - the fields of the new scene (category, initial description), as today.
- Changing scene in **roleplay** only gains the optional final description of the scene that closes (owner: the
  field already exists, so it comes in). No summary and no clock editor.
- `change_scene` gains `briefFinalDescription?` and `storyAt?` (+ `confirmLongSkip?`). The server:
  refuses an open turn (as today); closes the round (adding 1 s if counted); validates `storyAt ≥` the minimum;
  moves the clock; pauses real time; records everything in one transaction (scene and round closed, clock, events);
  limit = new clock, reason `scene`.
- **It is not possible to end the match during a combat** (owner): the scene must be changed first.

## 5. Ending the match

- **Master only**, from the table, via WS `end_match` (mine: it is the in-memory room that must stop and notify
  whoever is connected, as with `start_match`; whoever is offline finds out through the match's `GET`).
- **Refused** with: current scene is combat (`match_in_battle`), open turn (`cannot close round: current turn is
  still open`, the same as `change_scene`), clock with no time (`clock_not_set`), match already ended.
- **Flow (owner):** 1) confirmation warning that it **cannot be undone** (Não dá para desfazer — "It cannot be
  undone"); 2) dialog with the **clock editor** (starts at the current clock; minimum = limit) for the **exact time
  of the end** — mandatory (owner: "we will require this from the master, for now") — and the optional **final summary**
  (`matches.brief_final_description`, 255), following the "(optional)" pattern of the forms.
- The editor's minimum is the greater of the limit and "current clock + 1 s" when the round in progress counts (the same
  calculation as End combat: a contested round in a roleplay scene also is worth 1 s). The combat summary does not
  appear here — the current scene is never combat.
- **On the server**, in one transaction: pauses the clock and sets it to the final instant; closes round and scene;
  `matches.game_end_at = now()` (**real** instant, new column, paired with `game_start_at`);
  `matches.story_end_at = date of the final instant`; `brief_final_description`; history event;
  and **`campaigns.story_current_at` advances to the final instant** (owner, 08/11: "this is the idea"). The column is
  `TIMESTAMP` and already has the "does not regress" rule (`ErrCannotRegressStoryCurrentAt`): it writes
  `GREATEST(COALESCE(story_current_at, instant), instant)` — a match played out of order does not pull the campaign
  backwards (mine). **Every participating sheet** (players and NPCs, `match_participants`) advances
  `character_sheets.story_current_at` (`DATE`) to the final instant's date (owner, 08/11: "the
  participating sheets too! all of them!!"), by the same never-regress rule, in the same transaction.
  The queue and the in-memory reactions are discarded. Afterwards: `match_ended` to the whole table, and the room closes
  the connections with **code 4002 `match ended`** and stops.
- **What marks "ended":** `game_end_at != nil` (mine, hypothesis from the handoff confirmed by the model: `story_end_at`
  goes back to being just the story's date). A method `Match.IsFinished()` replaces all the guards that today look at
  `StoryEndAt` (starting, enrollments, NPC, editing campaign/match) — a fix done together with the refactor.
- **Reconnection** to an ended match (table or lobby): the handler closes with 4002 right after the upgrade, like the
  4001 of the lobby.
- **Front:** everyone goes to the match page with `navigate(..., { replace: true })` — `/game` leaves the browser history,
  so "Back" on the match page **never** returns to the table (owner). The player sees "A partida foi encerrada pelo mestre."
  (The match was ended by the master.) `/game` and `/lobby` for an ended match redirect by themselves (`replace`), which
  covers whoever was offline and whoever types the URL.
  The ended match page hides the buttons to open the table/lobby and uses `gameEndAt` in the "ENCERRADA" (ENDED) badge.
- **Out of scope for this delivery (owner):** showing the full history on the ended match page — that is the next session.
  Character sheets (HP, EXP) are not persisted by ending the match beyond what happens today.

## 6. Data model (migrations additive only)

| Where | What |
|---|---|
| `matches` | `story_start_time TIME NULL` (optional start time); `game_end_at TIMESTAMPTZ NULL` |
| `match_clocks` (new) | `match_uuid PK FK`, `story_at TIMESTAMP NULL` (NULL = time not defined), `limit_at TIMESTAMP NULL`, `limit_reason VARCHAR(16)`, `running_since TIMESTAMPTZ NULL`, `updated_at TIMESTAMPTZ` |
| `scenes` | `story_started_at TIMESTAMP NULL`, `story_finished_at TIMESTAMP NULL` |
| `rounds` | `counted BOOLEAN NOT NULL DEFAULT false` (counted round), `drained BOOLEAN NOT NULL DEFAULT false` (ended with the queue empty) |
| `turns` | `story_at TIMESTAMP NULL` (the clock when the turn opened) |
| `match_events` | new `kind`s: `clockStarted`, `clockAdjusted`, `clockRealTime` (payloads in section 7) |

**Story time is naive (no time zone):** `TIMESTAMP`, never `TIMESTAMPTZ`; in Go, `time.Time` in UTC treated as
"wall clock"; on the wire, the string `"2000-01-01T08:30:00"` **without `Z` and without an offset**. The front
**never** passes this through `new Date(str)` (it would apply the browser's time zone): a custom util splits it into
parts and formats it. This is gotcha #1 of this feature.

**Recording:** "one master command, one transaction" (house rule). The clock goes into the transaction of the turn/round
close (`TurnCloseData`/`PersistRoundClose`), of `change_scene`, and of `end_match`; the clock commands have their own.
The clock state is a snapshot (upsert): a write that fails is healed by the next one; a lost history event stays in the log.
`match_events` requires `scene_uuid`/`round_uuid`: the clock events ensure the row of the active pair beforehand
(`EnsureSceneAndRound`).

**Grouping in the history (mine):** consecutive clicks by the master become **one** event: an `adjust_clock` that arrives
up to 10 s after the previous one, with no other event in between, updates the previous event (`from` stays, `to` and
`deltaSeconds` accumulate) instead of creating another.

The front already re-reads the history when the WS signals ("WS signals, REST fetches"): `clock_changed` joins the list of
signals that invalidate `GET /history`, and the updated event in place appears in the re-read.

**Mandatory search before changing the guard:** every use of `storyEndAt`/`story_end_at` as "ended" in the back (list queries,
campaign filters) and in the front (ENCERRADA badge, `getMatchStatus`, match cards) must now look at `gameEndAt`.

**Status (parallel session):** a domain helper `RoundEquivalents(from, to) int` (= seconds) documents the conversion.
No status is implemented here.

## 7. Protocol (full contract goes in `docs/dev/api/match-clock.md`)

**State** (in `match_full_state.clock` and in every `clock_changed`):
```json
{ "storyAt": "2000-01-01T08:30:00", "running": false,
  "limitAt": "2000-01-01T08:12:40", "limitReason": "turn" }
```
`storyAt: null` = time not defined. With `running: true`, the client counts from the moment it received the message.

**Client → server (master only, `forbidden` for the others):**
`set_clock_start {time:"08:30:00"}` · `adjust_clock {deltaSeconds, confirmLongSkip?}` ·
`set_clock {storyAt, confirmLongSkip?}` · `play_clock {}` · `pause_clock {}` ·
`get_combat_summary {}` · `change_scene` (new fields) · `end_match {storyAt, briefFinalDescription?, confirmLongSkip?}`.

**Server → client:**
- `clock_changed {clock, change}` — whole table. `change.kind` ∈ `start` | `adjust` | `set` |
  `play` | `pause` | `round` | `scene` | `end` | `limit` (`limit`: only the limit rose, when opening a turn — the master's
  panel needs it; no warning); `change.from`, `change.to`, `change.deltaSeconds`;
  in `pause`, `change.reason` ∈ `manual` | `roundMode` | `scene` | `masterLeft` | `end` | `restart`.
- `combat_summary {sceneId, countedRounds, countedSeconds, activeRoundCounts, minimumStoryAt, hiatuses:[{before:{roundNumber,turnId,actorId,actionKind}, after:{…}}]}` — only for whoever asked.
- `match_ended {matchId, storyAt, gameEndAt}` — whole table; then closes with 4002. `storyAt` is story time
  (naive, no offset); `gameEndAt` is a real instant in RFC 3339 with offset, like `gameStartAt`.
- `end_match` also accepts the minimum of the round in progress: the server closes the round (+1 s if it counts)
  before validating `storyAt`.
- New errors: `clock_not_set`, `clock_below_limit`, `clock_long_skip_unconfirmed`,
  `clock_play_in_race`, `clock_already_set`, `match_in_battle`, `not_battle_scene`.

**History (`GET /history`):** events `clockStarted {at}`, `clockAdjusted {from,to,deltaSeconds}`,
`clockRealTime {from,to,elapsedSeconds,reason}` (recorded on pause); scenes with
`storyStartedAt/storyFinishedAt` and `briefFinalDescription`; rounds with `counted`/`drained`; turns
with `storyAt`.

**REST:** creating/editing a match accepts `storyStartTime` (`"HH:MM"` or `"HH:MM:SS"`, optional; an
empty string clears it on edit). The match's `GET` returns `storyStartTime`, `gameEndAt` and, when ended,
`storyEndTime`.

## 8. Front

### 8.1 The clock at the table (master and player)
- **Where (mine, with the owner's suggestion):** a centered pill at the top of the stage, over the map,
  just below the top bar — the classic spot for the world clock in game HUDs. The general bar (`GeneralBar`)
  and the stage notices (`StageNotices`) move down below it. On mobile, the pill shows only the time; the date
  appears in the panel.
- **Closed:** a **stopwatch** icon + `08:30:00` (+ date `01/01/2000` from tablet width up).
  With no time defined: "Hora a definir" (Time to be defined).
- **Live (real time on):** a pulsing "live" dot and the seconds running, for everyone. **Master only**, the
  attention escalates so that he **does not forget** (owner): a light pulse every 10 s, a stronger pulse every
  1 min (with the elapsed time, "▶ 3 min"), and from 5 min on a fixed alert tone + a short notice every 5 min
  ("Tempo real ligado há 10 min" — Real time has been on for 10 min). With `prefers-reduced-motion`, no movement:
  only color/opacity in steps. Only existing tokens (`src/styles/CLAUDE.md`: a new color is the designer's decision).
- **Player:** the pill does not open a panel. Each `clock_changed` made by the master (`adjust`, `set`, `play`,
  `pause`, `scene`, `start`, `end`) shows a notice on the stage for about 6 s: *"O mestre avançou o tempo 2 h 10 min — agora 01/01/2000 10:40:00"*
  (The master advanced the time 2 h 10 min — now 01/01/2000 10:40:00) / *"voltou o tempo 15 min"* (went back 15 min) /
  *"ligou o tempo real"* (turned real time on) / *"pausou o tempo real (+3 min 12 s)"* (paused real time (+3 min 12 s)).
  Consecutive notices **add up into the same notice** while it is on screen (mine). `round` does not warn (owner); it
  only lightly flashes. The History gains the line for each clock event.

### 8.2 The master's panel (owner, depth levels)
1. **Click on the pill** → opens the panel anchored below it:
   - the large time (clickable → level 3) and the date; the limit in small text ("Não volta antes de 08:12:40 — último turno"
     — Does not go back before 08:12:40 — last turn);
   - **▶ Tempo real / ❚❚ Pausar** (▶ Real time / ❚❚ Pause) (disabled in the contested regime, with the reason in a `Tooltip`);
   - **+1 h** and **+1 min** — they change the clock **immediately**, with no "Aplicar" (Apply) (owner);
   - **Passar dias** (Skip days) → opens level 3 already with +1 day;
   - **⋯ Mais** (⋯ More) → reveals level 2.
2. **More** (more hidden, owner): **+1 s** and the going-back ones: **−1 h**, **−1 min**, **−1 s**. A button that would
   cross the limit is disabled, with the limit in the `Tooltip`.
3. **Direct edit** (click on the time): date with **[−] 01/01/2000 [+]** (steps of 1 day — **no calendar**, owner) and
   time in three fields `HH : MM : SS` (typing or arrows); preview of the difference ("+1 dia 2 h" — +1 day 2 h);
   **Aplicar** (Apply) and **Cancelar** (Cancel). Below the limit, Apply is disabled and says why.
   More than 2 days forward → confirmation. This is the only place with "Aplicar" besides skipping days (owner).
- Closes with Esc, a click outside, or a click on the pill.

### 8.3 The reused clock editor
Level 3 is a component (`ClockEditor`) also used in the **initial time dialog**, in **End combat** and in **End match**,
each with its own minimum and its own starting value (owner: "it can be the same component, somewhere else").
Feature-level in `src/features/match/clock/`.

### 8.4 Dialogs
- **Initial time (mandatory):** no Cancel and no closing by clicking outside; the match's date is fixed; fields
  `HH : MM` (seconds optional). The player sees "Hora a definir" until then.
- **New scene / End combat** (section 4) — evolution of `SceneChangeDialog`.
- **End match** (section 5): a dangerous button next to "Nova cena" in the master's controls, disabled with a `Tooltip`
  in combat / open turn / no time; `ConfirmDialog` ("Não dá para desfazer" — It cannot be undone) → dialog with
  `ClockEditor` + optional final summary → waits for `match_ended`.

### 8.5 Match form
Creating/editing gain **"Hora na história (opcional)"** (Time in the story (optional)) next to the date: two numeric fields
`HH : MM`, with no picker that opens on top (owner: no calendar in the user's face). Help text: "Se ficar vazio,
você informa a hora quando a partida começar." (If left empty, you give the time when the match starts.)

## 9. Out of scope
- Full history on the ended match page (next session, owner).
- Time-travel abilities (owner: maybe another feature).
- The statuses themselves (parallel session) — only the conversion helper.
- The enum `GameEventCategory.DateChange` (unused) stays as it is.

## 10. Delivery and verification
- **Back PR** (`feat/story-clock`): migrations, clock domain, WS commands, combat summary, `end_match`, `IsFinished` guards,
  REST for the start time, contracts (`match-clock.md` new; `match-combat-ws.md`, `match.md`, `match-history.md` updated;
  `documentation-map.yaml`), game rule in `docs/game/` (1 round = 1 s, gaps). Tests per layer + integration; `go vet` per
  task; `golangci-lint` with no ceiling before the PR; smoke with `curl` and a WS script.
- **Front PR** (`feat/story-clock`): clock, panel, `ClockEditor`, dialogs, notices, history, redirects, form. Vitest tests;
  **browser verification** with master and player (`go run ./cmd/devseed`, seed accounts — never the owner's accounts),
  mobile, tablet and desktop widths.
- Handoff via `dev-checkout.sh` if anything is left unverified.
