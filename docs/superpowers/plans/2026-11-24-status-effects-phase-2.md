# Status Effects — Phase 2 (rolls, critical, negative HP, death) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Advantage/disadvantage accumulate (one extra die per level, keep the best/worst set); the critical and critical failure rule lives in `MatchRules` and doubles/halves every test, and a critical hit doubles the final damage; the status effects the master applies weigh on every roll the engine computes; HP descends below zero to a per-sheet negative maximum, past which the character dies (status `dead`, public, applied by the system at the turn's close, removable by the master with an HP).

**Architecture:** A roll keeps its base set and a pool of extra dice that fall once, when a bias first asks for them (`RollAttempts{Primary, Extra}`); `Derive` stays pure and reports how many dice it was short, and the session tops them up and resolves again. `statuseffect.RollEffectOf(effects, test)` turns a character's effects into bias/penalty/auto-fail for one test descriptor; the resolver and the session build the descriptor at each of the seven `Derive` sites. Death is decided by the session when the close applies damage; the room stores the `dead` write in the closed turn's transaction and broadcasts it like any status change. The front gets a small, linked PR: the HP bar minimum in the sheet engine (parity) and two marks on the master's calculation.

**Tech Stack:** Go 1.23, PostgreSQL + goose, pgx, gorilla/websocket, standard `testing`. Front: Vite/React/TS, vitest.

**Spec:** `docs/superpowers/specs/2026-11-15-status-effects-design.pt-br.md` on branch `docs/status-effects-spec` (PR #110, commit `de1c3cf` or later — read with `git show origin/docs/status-effects-spec:<path>`). Read §4 ("Como as penalidades se somam"), §5.3–§5.4, §7, §9, §16 (all of it; **§16 overrides earlier sections**, items 28–38 are this phase). Phase 1 plan: `docs/superpowers/plans/2026-11-17-status-effects-phase-1.md`. Contract: `docs/dev/api/match-status-effects.md` (§10 is what this phase removes). Gap notes of this session: `docs/superpowers/plans/2026-11-24-status-effects-phase-2-gaps.pt-br.md` (its §3 lists decisions this plan carries).

## Global Constraints

- **Branches/worktrees.** Back: worktree `System_X_System/.worktrees/status-effects-rolls`, branch `feat/status-effects-rolls` (from `origin/main`). Front: worktree `System_X_System_React/.worktrees/status-effects-rolls`, branch `feat/status-effects-rolls` (from `origin/main`), created at Task 13. **Never touch either main checkout.** Copy `.env` into a new worktree (gitignored).
- **Language:** identifiers, file names and test titles in **English**; comments, `docs/dev/`, log text and every user-facing string in **Portuguese**. Wire values camelCase English.
- **The critical rule lives in `match.MatchRules`**, never in `die`, `enum.DieSides` or `RollSource` (spec §7).
- **Roll once.** A die that fell is never rolled again; a bias that grows draws only the missing dice, once. `Derive` and `TurnResolver.Resolve` never roll.
- **Inside an open turn nothing is written before its close** (house rule). **One master command, one transaction.**
- **Comments/TODOs:** never delete one outside this scope; correct what this work makes contradictory (several comments in `roll_calculator.go`, `roll_attempts.go`, `turn_resolver.go` `RollResult`, `health_points_bar.go`, `applySkillEdit`, `docs/dev/match/combat-engine.md` describe the old rules — fix each in the task that changes it).
- **Out of scope (do NOT build):** automatic tests (stun, unconsciousness, bleeding, balance), HP loss and recovery per round, time outside the round, asphyxia, Fatigue, Push/Carry rename, any front visual beyond Task 14, a master "edit HP" command, applying `dead` by hand.
- **Per-task verification:** `go vet ./... && go vet -tags=integration ./internal/gateway/pg/...` and `go test` of the touched packages; gateway tasks run that package's integration tests (DB: `set -a && . ./.env && set +a && export TEST_DATABASE_URL="postgres://$PG_DB_USER:$PG_DB_PASS@localhost:5432/hxh_rpg_test?sslmode=disable"`). Full integration suite only with `-p 1`.
- **Before the PR:** `golangci-lint cache clean && ~/go/bin/golangci-lint run --max-same-issues 0 --max-issues-per-linter 0`. Do not poll CI.

## Owner decisions this plan implements (spec §16.1 item 10, §16.4 items 28–38)

1. One extra D10 per level; advantage keeps the 2 best, disadvantage the 2 worst; levels cancel one by one; numbers add. Passive with N levels rolls 2+N and compares the kept pair with the base value. D20 set: one extra D20 per level, keep 1. Same for the master's condition.
2. Critical (kept dice all at the top face) = (dice + test value + **every number added to the test**) × 2; critical failure (kept dice all 1) = the same ÷ 2, floored. Any test. A critical hit also doubles the **final** damage of each target (after every reduction; 0 stays 0) and on walls.
3. Negative HP max = ⌊10 + Vitality level + Resistance value + Physicals bonus⌋. The HP bar's min is −that. Death when HP < min (one past it); a big blow leaves HP at the exact value (−40 stays −40).
4. Death = status `dead` ("Morto"), public, containing Unconscious, no exit of its own. Applied by the system at the turn's close (inside the turn the master's calculation says "dies"). Follows the fog like every public status. Removing it = the master's undo; he may give the HP it comes back with, default the negative max (only raised, never lowered). The sheet's `dead_at` mark is set only when the match ends; an undone death clears it then.
5. Prone: disadvantage on reflex dodge, repel and on **both** tests of all three escapes (provisional, TODO until movement). Open guard: escape disadvantage on both tests; default defense only rolled. Vision nullified: disadvantage on attack, defense **and dodge**. A Tact test declared in an action reads as the touch face.

## Decisions this plan makes (builder's call; go in the PR's "Decisões tomadas")

1. **Dice shape.** `action.RollAttempts{Primary, Extra []int}` (Secondary removed). Wire `attempts: {primary, extra}`; stored action rows from before keep `"Secondary"` in their JSON and read back with no extras (the stored resolution keeps the totals the table saw).
2. **Max 10 bias levels** per test (technical cap; beyond it reads as 10).
3. **Domain → test.** attack = hit; defense = default defense and repel; dodge = reflex (default, dodge, closed dodge, escapes) and evasion; movement = moveSpeed (move component and escapes). actionSpeed (Legerity) and damage are not domain tests. The feint has no derived total and gets nothing.
4. **Weakened** weighs on every test of its aptitude's skills: hit, reflex, evasion, defense, repel, actionSpeed, moveSpeed and `skills[]`. Skills of the spiritual set (Focus, WillPower, SelfKnowledge, Coa, Mop, Aop) are spiritual; every other skill is physical; mental has none.
5. **`skills[]` now get a derived total** (`RollCheck.Result`) with condition and status, at arrival and on every edit/status change. Nothing reads it for an outcome yet (the test chain will).
6. **Auto-fail:** not rolled, `Total` 0, `AutoFailed` true, no critical.
7. **Status change re-derives** the character's queued actions' speeds (a new penalty reorders the queue) — the same path an edited speed takes.
8. **`dies` and `criticalDamage`** ride on the resolution's per-target result; `dies` is stripped for every non-master reader (the table learns of a death by the `dead` status, through the fog).
9. **`dead` cannot be applied by hand** (`status_effect_invalid`, "O Morto entra só pela regra."); `dead` does not bring Prone.
10. **`dead_at` gets the real time of the match's end**; a sheet that was already dead keeps its earlier date.
11. The HP set by removing `dead`: out of turn it is written in the **same transaction** as the status write; inside a turn it is carried by the turn's close (as any sheet the close writes).

## Review Focus

1. **Bias edited after the dice fell** (master adds +2 to a hit that rolled with 0, then drops to +1) → exactly 2 extra dice fall once; +1 reads the first extra; going back to +2 draws nothing new. (Task 2 tests.)
2. **Status applied while an action waits in the queue** → its speed is re-derived with the penalty and the queue reorders; removing the status restores the number. (Task 6.)
3. **A blow that kills on a target already below zero** (HP −10, min −16, final damage 7) → dies; final damage 6 → lives at −16. (Task 5.)
4. **Removing `dead` with an HP outside [min, max]** → refused in Portuguese; without HP and current already ≥ min → HP unchanged. (Task 10.)
5. **Reconnecting to a match where someone died** → `match_full_state` carries the `dead` status by fog and the sheet's HP below min loads without error (no normalization clamp). (Tasks 7 and 8.)

---

## File Structure

| File | Responsibility |
|---|---|
| `internal/domain/match/entity/action/roll_attempts.go` | `RollAttempts{Primary, Extra}` |
| `internal/domain/match/match_rules.go` | `MaxBiasLevels`, `SetSize`, `CriticalTotal`, `CriticalDamage` |
| `internal/domain/match/service/roll_calculator.go` | new pick, passive with levels, critical, auto-fail, `Short` |
| `internal/domain/match/service/dice_shortage.go` | `DiceShortage`, `DiceRef` (what a resolution was short of) |
| `internal/domain/statuseffect/roll_effect.go` | `Test`, `RollEffect`, `RollEffectOf`, `AptitudeOfSkill`, `SenseOfSkill` |
| `internal/domain/statuseffect/{kind.go,piece.go,normalize.go}` | `KindDead`, Prone escape piece, Vision → dodge |
| `internal/domain/match/service/{turn_resolver.go,reaction_collision.go}` | descriptors at each Derive, crit damage, `Dies`, wall crit |
| `internal/domain/match/matchsession/{match_session.go,dice.go,status_effects.go,death.go}` | top-up loop, session-side derives, re-derive on status change, death at close, dead-at bookkeeping |
| `internal/domain/entity/character_sheet/status/{health_points_bar.go,status_bar.go}` | negative min, no floor for HP |
| `internal/gateway/pg/sheet/read_character_sheet.go` | load HP below min |
| `internal/application/match/status_effects.go` | refuse manual `dead`; remove `dead` with HP |
| `internal/app/game/{message.go,status_effects.go,room.go,end_match.go}` + `internal/app/wire/actionwire/*` | wire, death at close, HP on removal, dead marks at end |
| `internal/gateway/pg/{statuseffect/repository.go,round/persist_match_end.go,round/resolution_record.go}` | bars with status write, `dead_at`, record fields |
| `internal/app/api/sheet/testdata/sheet_parity.json` | regenerated |
| docs: `docs/dev/api/{match-status-effects.md,match-combat-ws.md,match-history.md}`, `docs/dev/match/{status-effects.md,combat-engine.md}`, `docs/game/` contradictions, `docs/documentation-map.yaml`, `AGENTS.md` | Task 12 |

---

### Task 1: Critical rule and dice-set shape in `MatchRules`

**Files:** Modify `internal/domain/match/match_rules.go`; test `match_rules_test.go`.

**Interfaces — Produces:**
```go
const MaxBiasLevels = 10
func (d DiceSet) SetSize() int                       // 2 for 2d10, 1 for d20
func (r MatchRules) CriticalTotal(raw int, critical, failure bool) int // critical → raw*2; failure → floor(raw/2) (true floor, also for negatives); neither → raw
func (r MatchRules) CriticalDamage(final int, critical bool) int        // critical → final*2 (0 stays 0); else final
```
- [ ] **Step 1:** Tests `TestCriticalTotal` (rows: `"critical doubles"` 28→56, `"failure halves down"` 25→12, `"failure floors negatives"` −3→−2, `"neutral keeps"`), `TestCriticalDamage` (`"doubles final damage"` 8→16, `"zero stays zero"`, `"no critical keeps"`), `TestSetSize`.
- [ ] **Step 2:** FAIL → implement (Portuguese comments citing spec §7 / §16.4 item 28) → PASS. **Commit** `feat(match): critical rule in MatchRules`.

### Task 2: Dice pool, accumulated bias, critical and auto-fail in `RollCalculator`

**Files:** Modify `action/roll_attempts.go`, `service/roll_calculator.go`, `app/wire/actionwire/{action.go,from.go}`, `gateway/pg/round/resolution_record.go` (`rollOutcomeRecord` mirrors `RollOutcome` field for field — add the same fields; `Short` with `json:"-"`); tests `service/roll_calculator_test.go`.

**Interfaces — Produces:**
```go
// action
type RollAttempts struct{ Primary, Extra []int } // IsEmpty: both empty
// service
type StatusRoll struct{ Bias, Penalty int; AutoFail, RolledOnly bool } // from statuseffect.RollEffect (Task 4)
type RollInput struct { /* existing fields */ ; Status StatusRoll }
type RollOutcome struct { /* existing */ ; Rolled []int; AutoFailed bool; Short int }
func (rc RollCalculator) Roll(rules match.MatchRules, src RollSource) action.RollAttempts // Primary only now
func (rc RollCalculator) TopUp(rules match.MatchRules, a *action.RollAttempts, need int, src RollSource) // fills Primary first (if empty), then appends to Extra
```
Algorithm (`Derive`): `bias = SystemBias + cond.Bias + ledger bias + Status.Bias`, clamped to ±`MaxBiasLevels`; `modifier = cond.Modifier + ledger amount + Status.Penalty` (Penalty is already negative-signed by Task 4, i.e. Slowed 2 → −2). `Status.AutoFail` → return `{Total:0, AutoFailed:true}` with no dice read. `Status.RolledOnly` turns a passive into a rolled test. Rolled test: `levels=|bias|`, `pool = Primary + Extra[:min(levels,len(Extra))]`, `Short = (levels - len(Extra)) if >0`, plus `SetSize` when Primary is empty; bias>0 keeps the `SetSize` highest, bias<0 the lowest, 0 reads Primary; `Dice` = kept (in pool order), `Rolled` = pool. Passive: base value; with bias it reads the kept set against the base exactly as `readPassiveBias` does today (strictly better/worse replaces; ties keep the base). Critical/failure read on the kept dice only when the dice count. `Total = rules.CriticalTotal(DiceTotal + SkillValue + modifier, crit, fail)`.
- [ ] **Step 1:** Tests (scripted dice): `"advantage two keeps best pair of four"` ([3,4]+[9,10]→19), `"disadvantage keeps worst pair"`, `"levels cancel one by one"` (+2 −1 → 1 level), `"d20 keeps one of three"`, `"short reports missing extras"` (bias +3, 1 extra → Short 2, derives with what exists), `"short counts the base set on an empty passive"`, `"passive with two levels compares kept pair with base"` (both signs, tie keeps base), `"critical doubles total with modifiers"` (10,10 + skill 8 − 3 → 50), `"critical failure halves"` (1,1 + 8 → 5), `"critical read on kept dice only"` (advantage drops a 1,1), `"passive base value never critical"`, `"auto fail skips dice"`, `"rolled only rolls a passive"`, `"bias capped at ten levels"`. Replace the `pickAttempt` tests.
- [ ] **Step 2:** FAIL → implement; rewrite the comments that describe "both sets"/"magnitude only settles the sign" → PASS. `go build ./...` will fail where `Secondary` or `ensureDiceForBias` is used: replace `ensureDiceForBias(rc, bias)` with `s.topUp(rc, need)` (Task 3 generalizes the loop; here just keep behaviour: need = Short of a Derive), and the wire `Secondary` with `Extra` (`json:"extra,omitempty"`). Update `actionwire` tests and `resolution_record` round-trip tests.
- [ ] **Step 3:** `go test ./internal/domain/... ./internal/app/... ./internal/gateway/pg/round/` (unit) PASS. **Commit** `feat(match): accumulated advantage, dice pool and critical in derive`.

### Task 3: Top-up loop — the resolution draws the dice it was short of

**Files:** Create `service/dice_shortage.go`, `matchsession/dice.go`; modify `service/{turn_resolver.go,reaction_collision.go}`, `matchsession/match_session.go` (`ResolveTurn`, `deriveSpeeds`); tests `matchsession/dice_test.go`.

**Interfaces — Produces:**
```go
// service
type DiceRef struct{ ActionID uuid.UUID; Field action.ConditionField; SkillName string } // same addressing as edit_action
type DiceShortage struct{ Ref DiceRef; Need int }
// TurnResolution.DiceShort []DiceShortage — never on the wire, never persisted
// matchsession
func (s *MatchSession) deriveCheck(rc *action.RollCheck, in service.RollInput) service.RollOutcome // derive; if Short>0 TopUp(s.rollSource) and derive again
```
`ResolveTurn`: resolve → for each shortage find the live check (`actionOnTurn` + `resolveRollCheck(a, ConditionEdit{Field|SkillName})`) and `TopUp` it → resolve again (at most twice; a second shortage is a bug → log). Each `Derive` site in the resolver records its shortage with its `DiceRef` (hit = `FieldHit` on the turn action; reflex = `FieldDodge`, evasion = `SkillName "Evasion"`, default defense = `FieldDefense`, repel = `FieldRepel`, escape moveSpeed = `FieldMoveSpeed`, all on the reaction).
- [ ] **Step 1:** Tests: `TestResolveTurnDrawsMissingDiceOnce` (master gives +2 on hit via `ApplyMasterAction`; resolution reads 4 dice; resolving again rolls nothing — scripted `RollSource` counts calls), `TestBiasDownThenUpDrawsNothingNew`, `TestPassiveDefaultDefenseBiasDrawsBaseSet` (replaces the old ensureDiceForBias test).
- [ ] **Step 2:** FAIL → implement → PASS; `go test ./internal/domain/match/... ./internal/app/game/` PASS (e2e may need scripted-dice adjustments where they counted `Secondary`). **Commit** `feat(match): draw missing dice once when a bias grows`.

### Task 4: `statuseffect.RollEffectOf` — what a set of effects does to one test

**Files:** Create `internal/domain/statuseffect/roll_effect.go`, test `roll_effect_test.go`; modify `kind.go` (`KindDead`), `piece.go`, `normalize.go`, `piece_test.go`.

**Interfaces — Produces:**
```go
const KindDead Kind = "dead" // "Morto" / estado "morto"; PublicByDefault true; no params; pieces = stunnedPieces(0,false) (contém o Inconsciente)
type Test struct {
	Skill     enum.SkillName // "" for none
	Domain    Domain         // "" when the test is not a domain test (actionSpeed, skills[])
	Reaction  Reaction       // reflexDodge | evasion | defense | repel | escape | ""
	MoveSpeed bool
}
type RollEffect struct{ Bias, Penalty int; AutoFail, RolledOnly bool; Kinds []Kind }
func RollEffectOf(effects []Effect, t Test) RollEffect
func AptitudeOfSkill(s enum.SkillName) Aptitude // Focus, WillPower, SelfKnowledge, Coa, Mop, Aop → spiritual; any other → physical; "" → ""
func SenseOfSkill(s enum.SkillName) (Sense, bool)
```
Reading rules (sum over every piece of every effect; `Kinds` = effects that contributed, sorted, unique):
- `testPenalty`: matches when its scope's Aptitude = `AptitudeOfSkill(t.Skill)`, or Domain = `t.Domain`, or Sense = `SenseOfSkill(t.Skill)` (for Tact the piece's Face must be `FaceTouch` — spec §16.4 item 38), or `MoveSpeed && t.MoveSpeed` → `Bias -= Disadvantage; Penalty -= Penalty`.
- `autoFail`: Sense scope matches as above → `AutoFail`.
- `reactionRestricted` with `disadvantage` and Reaction = `t.Reaction` → `Bias -= 1`; with `rolledOnly` → `RolledOnly`. (`forbidden` is phase 1's block; ignore here.)
Piece changes: Prone adds `restrict(ReactionEscape, RestrictionDisadvantage)` with `// TODO(movimento): provisório — o desenho do movimento decide por tipo de escape (rolar talvez sem penalidade); spec §16.4 itens 33 e 37.`; Vision nullified adds `testPenalty{Domain dodge, Disadvantage 1}` (item 35). `KindDead` in `allKinds`, labels (`"Morto"`, state `"morto"`), `PublicByDefault`, `Normalize` (no params).
- [ ] **Step 1:** Tests `TestRollEffectOf`: `"weakened physical weighs on hit"`, `"weakened spiritual ignores hit"`, `"weakened number only"`, `"injured hurt weighs on its domain"`, `"vision nullified weighs on attack defense and dodge"`, `"vision nullified auto fails a vision test"`, `"vision impaired on a vision skill"`, `"tact touch weighs on a tact skill test"`, `"tact balance only ignores a tact skill test"`, `"tact balance weighs on the four domains"` (impaired with own numbers; nullified → 1), `"stunned auto fails every sense test"`, `"dead auto fails every sense test"`, `"slowed penalizes move speed"`, `"prone weighs on reflex repel and both escape tests"`, `"open guard escape disadvantage on both tests"`, `"open guard default defense rolled only"`, `"sources accumulate"` (weakened + vision nullified on hit → −2), `"no effects is neutral"`. Add `dead` rows to the existing kind/normalize/piece tests.
- [ ] **Step 2:** FAIL → implement → PASS. **Commit** `feat(statuseffect): roll effects per test, dead kind`.

### Task 5: Effects weigh on the resolver's rolls; critical damage; who dies

**Files:** Modify `service/turn_resolver.go`, `service/reaction_collision.go`, `matchsession/match_session.go` (`ResolveTurn` passes effects); tests `service/turn_resolver_status_test.go` (new), existing resolver tests.

**Interfaces:**
- Consumes: `statuseffect.RollEffectOf`, `Test` (Task 4); `MatchRules.CriticalDamage` (Task 1).
- Produces: `ResolveInput.Effects map[uuid.UUID][]statuseffect.Effect` (nil = none); `ReactionInput.Effects []statuseffect.Effect` (the target's); `CharacterResult.CriticalDamage bool`, `CharacterResult.Dies bool`.

Descriptors: hit `{Skill: hit skill, Domain: attack}` on the actor's effects; reflex `{Reflex, dodge, reflexDodge}` — but `{Reflex, dodge, escape}` when the reaction displaces; evasion `{Evasion, dodge, evasion}`; default defense `{Defense, defense, defense}`; repel `{Repel, defense, repel}`. `StatusRoll` = the `RollEffect` fields. Critical damage: after `EffectiveDamage`, `cr.EffectiveDamage = in.Rules.CriticalDamage(eff, cr.Hit.IsCritical)`, `cr.CriticalDamage = cr.Hit.IsCritical && eff > 0`; the chain's residual is not doubled. `Dies`: target sheet's current HP − `EffectiveDamage` < its HP bar min, and the target has no `dead` effect. Wall branch: derive the hit once (no `AgainstID`, actor's effects) and give the wall `CriticalDamage(effective)`: pass the doubled amount into `ApplyStructuralDamage` by adding a `critical bool` parameter that doubles `effective` before subtracting HP (rebound unchanged).
- [ ] **Step 1:** Tests: `"weakened attacker hits with disadvantage"`, `"vision nullified target dodges with disadvantage"`, `"prone target escape both tests disadvantaged"`, `"open guard default defense rolls"`, `"auto failed hit"` is not possible (attack is not a sense) — instead `"stunned has no effect on resolver numbers beyond blocks"`, `"critical hit doubles each target final damage"` (chain of two; residual unchanged), `"critical on zero final damage stays zero"`, `"critical failure on hit halves the total"`, `"dies when final damage passes the negative max"` (HP 10, min −16: 26 → −16 lives; 27 → −17 dies), `"already dead target does not die again"`, `"wall takes double on critical"`.
- [ ] **Step 2:** FAIL → implement; fix the `RollResult` comment ("No rule consumes them yet") → PASS. **Commit** `feat(match): status effects weigh on the resolver; critical damage`.

### Task 6: Session-side rolls: speeds, moveSpeed, `skills[]`; re-derive on status change

**Files:** Modify `matchsession/match_session.go` (`deriveSpeeds`, `applySkillEdit` comment), `matchsession/status_effects.go`; tests `matchsession/status_rolls_test.go` (new).

**Interfaces — Produces:** `func (s *MatchSession) deriveSkills(a *action.Action)` (each `a.Skills[i].Result = deriveCheck(...).Total`, descriptor `{Skill: name}`); `deriveSpeeds` builds actionSpeed `{Legerity}` and moveSpeed `{skill, Domain movement, MoveSpeed: true}` (an escape reaction's moveSpeed: `Reaction escape` too); both go through `deriveCheck`. After every status change of a character (`apply/edit/remove` path that already calls `DropRefusedByStatus…`), re-derive speeds and skills of that character's queued actions and of its reaction on the open turn, and reorder the queue the way an edited `speed` does.
- [ ] **Step 1:** Tests: `"slowed lowers queued move speed and reorders"`, `"removing slowed restores the speed"`, `"weakened lowers action speed"`, `"vision nullified auto fails a vision skill entry"`, `"skill entry total carries the master condition"`, `"skill entry tact reads the touch face"`.
- [ ] **Step 2:** FAIL → implement → PASS; `go test ./internal/domain/match/... ./internal/app/game/` PASS. **Commit** `feat(match): status effects weigh on speeds and declared skills`.

### Task 7: HP below zero on the sheet

**Files:** Modify `character_sheet/status/{health_points_bar.go,status_bar.go}`, `gateway/pg/sheet/read_character_sheet.go`; tests in `status/` and the sheet gateway; regenerate `internal/app/api/sheet/testdata/sheet_parity.json`.

**Interfaces — Produces:** `func (hp *HealthPoints) NegativeMax() int` (⌊10 + vitality level + resistance value + physicals bonus⌋, recomputed in `Upgrade`; `min = −NegativeMax()`); HP's `DecreaseAt` has no floor; HP's `SetCurrent` accepts any value ≤ max (a dead character's HP sits below min). Stamina/Aura unchanged. Replace the `TODO: Implement Min for hit_points` block by the implementation and a note citing spec §7 (keep `// TODO: check how the buff interferes here`).
- [ ] **Step 1:** Tests: `TestNegativeMax` with the spec's fixtures (HP 20 → 14 and 15, HP 30 → 17, HP 51 → 21, HP 75 → 25 — build sheets from the parity cases or the same inputs), `"health decreases past min"`, `"health set below min"`, `"stamina still floors at zero"`; gateway integration `"health below min loads unchanged"`.
- [ ] **Step 2:** FAIL → implement → PASS → `go test ./internal/app/api/sheet/ -run TestSheetParityFixtures -update`, then the test without `-update` PASS. **Commit** `feat(sheet): negative HP bar minimum`.

### Task 8: Death at the turn's close (session + room)

**Files:** Create `matchsession/death.go`; modify `matchsession/match_session.go` (`applyDamage`, `TurnTransition`), `app/game/room.go` (the three close paths — extract one helper `r.recordDeathsLocked(...)` used by all), `app/game/status_effects.go`; tests `matchsession/death_test.go`, `app/game/death_e2e_test.go`.

**Interfaces — Produces:** `TurnTransition.Deaths []statuseffect.Write` (one `dead` apply per character whose HP went below min this close and had no `dead`; event `ChangeApplied`); `func (s *MatchSession) IsDead(charID uuid.UUID) bool`. Room, under `r.mu`, before `persistClosedTurn`: stamp the events (cause = the closed turn's action id, same `statusInstant` shape `runStatusEffectChange` builds), append the writes to the **closed** turn's `turnWrites.statusEffects` so `PersistTurnClose` carries them; after `broadcastHpChanges`, `syncStatusViews` for those characters (fog decides who gets it). Queue/reaction drops for the dead go through `dropRefusedByStatusLocked` like any status.
- [ ] **Step 1:** Session tests: `"close kills past the negative max"`, `"close below zero but within max does not kill"`, `"dead keeps the exact hp"`, `"dead character cannot act or react"`. E2E (real `Room`, mock repos as the phase-1 e2e tests do): `"death goes to the master and to who sees the piece"`, `"player out of sight gets dead when the piece enters sight"`, `"death is in the turn close write"` (repo mock sees the `dead` write inside `PersistTurnClose`).
- [ ] **Step 2:** FAIL → implement → PASS. **Commit** `feat(match): death at the turn's close`.

### Task 9: Wire — resolution and contract fields

**Files:** Modify `app/game/message.go` (`CharacterResultPayload.CriticalDamage bool json:"criticalDamage,omitempty"`, `Dies bool json:"dies,omitempty"`; `RollResultPayload`/rolls: `autoFailed` where a `RollOutcome` is put on the wire), `service/projection.go` (strip `Dies` for non-master), `gateway/pg/round/resolution_record.go` (persist both), `actionwire` (`skills[].rollCheck.result` now real); tests `resolution_payload_test.go`, projection tests, record round-trip.
- [ ] **Step 1:** Tests `"critical damage on the wire"`, `"dies only for the master"` (also on the settled resolution a player receives), `"record keeps critical damage and dies"`, `"skill entry result travels"`.
- [ ] **Step 2:** FAIL → implement → PASS. **Commit** `feat(game): critical damage and death on the resolution wire`.

### Task 10: `dead` by hand refused; removing `dead` sets the HP

**Files:** Modify `application/match/status_effects.go`, `app/game/{message.go,status_effects.go}`, `gateway/pg/statuseffect/repository.go` (+ the room's repo interface in `room_deps.go`); tests UC, gateway integration, e2e.

**Interfaces — Produces:** `RemoveStatusEffectPayload.HP *int json:"hp,omitempty"`; `StatusEffectsUC.Remove(s, master, caller, id uuid.UUID, hp *int)`; `StatusEffectChange.HP []HPChange{CharacterID; Sheet *csSheet.CharacterSheet; NewHP int}`; repo `Write(ctx, ops []statuseffect.Write, bars []appmatch.SheetStatusBars) error` (same transaction; reuse `pgsheet.UpdateStatusBars` on the tx, as `PersistTurnClose` does). Rules: `hp` only on a `dead` effect (`status_effect_invalid`, "Só a remoção do Morto aceita HP."); range `[min, max]` of the sheet's HP (`status_effect_invalid`, "O HP precisa ficar entre <min> e <max>."); nil → HP raised to `min` only if below it. Apply of `KindDead` → `status_effect_invalid`, "O Morto entra só pela regra." Room: emit `character_hp_changed` for the HP change (same projection as damage); out of turn the flush carries the sheet's bars with the status write; inside a turn mark the sheet unwritten so the close writes it.
- [ ] **Step 1:** Tests: UC `"apply dead refused"`, `"remove dead raises hp to min"`, `"remove dead with hp"`, `"hp out of range refused"`, `"hp on another kind refused"`, `"remove dead with hp already above min keeps it"`; gateway `"write carries bars in the same transaction"` (a failing bar write rolls the status back); e2e `"remove dead out of turn writes status and hp together"`, `"remove dead inside a turn waits for the close"`.
- [ ] **Step 2:** FAIL → implement → PASS (`go test -tags=integration ./internal/gateway/pg/statuseffect/`). **Commit** `feat(match): remove death with an HP; no manual death`.

### Task 11: The sheet's death mark at the match's end

**Files:** Modify `matchsession/status_effects.go` (remember who had `dead` when the effects were loaded), `application/match/i_repository.go` (`MatchEndData.DeadMarks []DeadMark{SheetID uuid.UUID; Dead bool}`), `app/game/end_match.go`, `gateway/pg/round/persist_match_end.go`; tests session, end_match e2e, `match_end_integration_test.go`.

Rule: `Dead: true` for every sheet with `dead` now → `dead_at = COALESCE(dead_at, GameEndAt)`; `Dead: false` for every sheet that had `dead` at load (or got it in this match) and has it no longer → `dead_at = NULL`. Sheets never dead in this match are not touched (a master's manual mark stays).
- [ ] **Step 1:** Tests: integration `"end marks the dead"`, `"end keeps an earlier death date"`, `"end clears an undone death"`, `"end leaves a manual mark alone"`; session `"dead marks list"`.
- [ ] **Step 2:** FAIL → implement → PASS. **Commit** `feat(match): death reaches the campaign when the match ends`.

### Task 12: Contracts and docs

- [ ] `docs/dev/api/match-status-effects.md`: `dead` in the kind table (público, contém o Inconsciente, só a regra o aplica), `remove_status_effect.hp`, new error texts, §6 (Caído/Guarda aberta/Visão no que pesam), §10 rewritten (what still does not exist: automatic tests, per-round HP/recovery, asphyxia, Fatigue, Nen), death at the close and fog.
- [ ] `docs/dev/api/match-combat-ws.md`: `attempts.extra` (no `secondary`), `criticalDamage`, `dies` (master-only), `autoFailed`, `skills[].rollCheck.result`, accumulated bias in conditions; `character_hp_changed` may carry negative HP and comes from the removal of `dead`.
- [ ] `docs/dev/api/match-history.md` if the history payload shows any of these; `docs/dev/api/character-sheet.md`: `status.health.min` negative, `deadAt` set at match end.
- [ ] `docs/dev/match/combat-engine.md` and `docs/dev/match/status-effects.md`: the dice pool, top-up, critical, descriptors table (domain → test), death flow, provisional escape rule (movement redesign; "rolar" is only a possible point).
- [ ] `docs/game/`: grep `vantagem`, `desvantagem`, `crítico`, `HP` / `vida` and correct what now contradicts the rules (no new chapters — the combat docs are phase 6).
- [ ] `docs/documentation-map.yaml` entries for new files; `AGENTS.md` Known Issues: rewrite the phase-1 "nada aqui muda rolagem" sentence into the phase-2 state; domain map entry.
- [ ] **Commit** `docs: status effects phase 2 contracts and engine docs`.

### Task 13: Front — sheet parity and the master's marks (PR linked to the back's)

**Files (front worktree):** `src/features/sheet/engine/status.ts` (HealthPoints min = −⌊10 + vitality level + resistance value + physicals bonus⌋), fixtures via `npm run sync:sheet-fixtures` (point the script at the back worktree's `sheet_parity.json` if it reads the main checkout), `src/features/match/combat/combatMessages.ts` (`criticalDamage?`, `dies?`, `attempts.extra?`), `ResolutionDetails.tsx` (after the damage line: `crítico ×2` when `criticalDamage`; a line `morre` when `dies`, using the existing error/danger tokens — follow `src/styles/CLAUDE.md`), `src/types/matchHistory.ts` (`extra?`, drop `secondary`). Tests: the engine parity test, a `ResolutionDetails` test for both marks.
- [ ] `npm run test` and `npm run build` PASS. **Commit** `feat(match): negative HP minimum and critical/death marks for the master`.

### Task 14: Prove it, open the PRs, leave the environment ready

- [ ] Back: full `go test ./...`, `go test -tags=integration -p 1 ./internal/gateway/pg/...`, `go vet -tags smoke ./...`, lint (Global Constraints).
- [ ] Real run: `migrate-up`, `go run ./cmd/devseed`, API on 5000 + game server; a WS script (master + player) that: applies Corpo debilitado to an NPC, sees the hit derived with 4 dice under +2 condition, forces a kill on a low-HP NPC (master condition on damage), checks `dies` in the master's `resolution_updated`, the `dead` status after the close for the player who sees the piece, removes `dead` with `hp` and checks `character_hp_changed`; `end_match` and `deadAt` on `GET` of the sheet. If driving a kill end-to-end needs undocumented flows out of proportion, substitute with e2e against the real handlers and say so in the PR.
- [ ] Front: browser on 5173 with the back of this branch — the master's calculation shows "crítico ×2"/"morre" (scripted dice or a forced condition), the sheet editor shows the negative min.
- [ ] Open both PRs (cross-linked). The back PR body, in Portuguese: what changed, **"Decisões tomadas"** (owner's items 28–38 and this plan's list), what was verified and what was not. Then `./dev-checkout.sh feat/status-effects-rolls feat/status-effects-rolls` from `System_X_System_Project/` if anything remained unverified (say what to look at).
