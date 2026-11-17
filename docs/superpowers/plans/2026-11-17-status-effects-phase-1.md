# Status Effects — Phase 1 (back-end base) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Status effects exist as data bound to the character sheet (any campaign, any match), declared as combinations of effect pieces in one catalog, and the master applies, edits and removes them by hand over the match WebSocket, each one visible to a list of viewers (no record = the kind's default).

**Architecture:** A pure domain package `internal/domain/statuseffect` holds the catalog (kinds, params, defaults, pieces), the effect entity, the set operations (apply/edit/remove) and the visibility rule. A new table `character_status_effects` (FK → `character_sheets`) is read when the match session is born and written by a gateway that runs on a pool or inside `PersistTurnClose`'s transaction. The `MatchSession` holds the effects in memory; `appmatch.StatusEffectsUC` mutates them (no I/O); the `Room` projects each change per recipient and decides WHEN the write happens: at once outside a turn, with the turn's close inside one, retried on the next write if it failed.

**Tech Stack:** Go 1.23, PostgreSQL + goose, pgx, gorilla/websocket (existing), standard `testing`.

**Spec:** `docs/superpowers/specs/2026-11-15-status-effects-design.pt-br.md` (owner-approved, PR #110). Read §4, §5, §8, §9, §10, §14 before any task. Decision history and examples: `/home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System/docs/superpowers/specs/2026-11-07-status-brainstorm-notas.md` (untracked, main checkout only — the spec wins on conflict).

## Global Constraints

- **Branch** `feat/status-effects-base`, worktree `System_X_System/.worktrees/status-effects`, cut from `origin/docs/status-effects-spec` (the spec travels with it). PR is stacked: base `docs/status-effects-spec`. Never touch the main checkout (`System_X_System/`, another session's branch).
- **Language:** identifiers, file names and test titles (`TestXxx`, `t.Run("...")`) in **English**. Comments, docs (`docs/dev/`) and log/error *prose* in **Portuguese**. Wire values in English camelCase.
- **Wire format is camelCase** (CLAUDE.md invariant). New payloads follow `message.go` style (explicit json tags).
- **"Testes automáticos" in the scope line means the game's automatic resistance tests (spec §6) — out of scope. Go tests are mandatory (TDD, every layer per `integration-tests.instructions.md`).**
- **Out of scope (do NOT build):** front; automatic resistance tests (§6); negative HP, critical, death/"Morto" (§7 — phase 2); Fatigue; enforcing any piece in the engine (blocking `enqueue_action`, refusing reactions, penalties in rolls, the Hidden status hiding the piece/bars/list — consumer phases); damage over time and recovery over rounds/clock (phase 4); history rows for status changes (§9 history — later phase; note it in the dev doc); a catalog REST endpoint (labels and alias suggestions are front copy, listed in the contract); renaming Push/Carry (§12).
- **Inside an open turn nothing is written before its close** (house rule, spec §10): status writes made while a turn is open ride `TurnCloseData` in `PersistTurnClose`'s transaction.
- **Comments/TODOs:** never delete a comment or TODO outside this scope; what this work makes contradictory is corrected (Task 1 rewrites the AGENTS.md rule to say exactly this).
- **Gateway rules** (`gateway-conventions.instructions.md`): timestamps from Go (`time.Now()` passed as `$N`), unconditional `Rollback` in defer, an index for every `WHERE`/`JOIN` column. New migration file only — never edit an applied one (memory: edited migrations diverged dev/Neon).
- **Per-task verification:** `go vet ./... && go vet -tags=integration ./internal/gateway/pg/...`; gateway tasks also run their package's integration tests. Local DB: `set -a && . ./.env && set +a && export TEST_DATABASE_URL="postgres://$PG_DB_USER:$PG_DB_PASS@localhost:5432/hxh_rpg_test?sslmode=disable"`. Full integration suite only with `-p 1`.
- **Before the PR:** `golangci-lint cache clean && ~/go/bin/golangci-lint run --max-same-issues 0 --max-issues-per-linter 0` (CI runs the lint; vet/test do not catch staticcheck).
- **Do not poll CI.**

## Decisions this plan makes (not in the spec; builder's call per §14)

1. **Re-applying a status the character already has is refused** (`status_effect_exists`); the master edits the existing one. With several characters, the command is all-or-nothing. This decides no game rule: how *sources* merge (§1 "fontes somam") belongs to the automatic sources of later phases.
2. **Numbers without a default in the spec are required** on apply: recovery points of Stunned/Paralyzed/Stupefied (§2 says the master types it), HP per round of Burning/Freezing/Poisoned, Slowed's penalty, the sense level, Tact's faces, Injured's domains, Compelled's conduct. Bleeding defaults to 1 (§5.3: "1 por round"); Paralyzed's domains default to all four (§5.3); Weakened and impaired senses default to disadvantage (§5.3).
3. **Visibility storage:** a nullable JSONB `audience` on the row. `NULL` = the kind's default (no record, §9). Otherwise `{"everyone":true}` or `{"players":[uuid…]}`; `{"players":[]}` = master only. The master always sees. The owner sees only if listed (so "só o mestre" is expressible).
4. **Default visibility** (no record): public kinds → everyone; hidden kinds → master + owner on a PC, master only on an NPC (§9); `MatchRules.PlayerStatusEffectsPublic` (default `false`, §14) makes a PC's hidden-by-default kinds public.
5. **Public statuses are not fog-gated**: the table already learns every NPC's existence (`npc_added` is broadcast); hiding a character is the Hidden status's job (consumer phase).
6. **Unconscious brings Prone** on apply (§5.5) when Prone is absent; removing Unconscious leaves Prone.
7. **Recovery is not inherited** through containment: §5.2's "contains every piece" covers what the status *does*; how it *ends* is per status (§5.3 "Acaba" column — Unconscious has no Tenacity recovery).
8. **Players never see an effect's visibility**; only the master gets `visibility`/`public`. No `updatedAt` on the wire (an edit to master-only fields must not reach a player as a change).
9. Text limits: alias ≤ 60 runes, conduct 1–255 runes (trimmed).

## Gap for the owner (stop point — do not decide)

- **G1 — Tact, balance face, effect on attack/defense/dodge/movement.** §5.3: "Só o equilíbrio afetado pesa sozinho em ataque, defesa, esquiva e movimento". Not said: *how much* (the status's own disadvantage/number when impaired?) and *what* when the sense is nullified. Phase 1 declares only the balance test trigger (→ Prone) and leaves `// TODO(G1, dono)` where that penalty piece goes. No phase-1 behavior depends on the answer; the consumer phase does.

## Review Focus

1. **A turn close that fails after the master changed statuses inside it** → memory keeps the change and the next successful write (another close or an out-of-turn change) writes it; nothing is written mid-turn. (Task 9 tests.)
2. **Changing who sees a status** → only recipients whose projected list actually changed get `status_effects_updated`; a player losing sight gets the list without it; a hidden change never reaches anyone else, not even as an empty message. (Task 8 tests.)
3. **Apply to several characters where one already has the status** → nothing changes for anyone and the master gets `status_effect_exists`. (Task 7 test.)
4. **Unconscious on a character already Prone; removing Unconscious** → no duplicate Prone, Prone stays. (Task 2 test.)
5. **Reconnect** → `match_full_state.statusEffects` carries, per viewer, exactly what that viewer may see; an NPC's hidden statuses never reach a player. (Task 8 test.)

---

## File Structure

| File | Responsibility |
|---|---|
| `AGENTS.md` | Task 1: TODO/comment rule rewritten; Task 10: Known Issues entry |
| `internal/domain/match/character_status.go` | Task 1: correction note under "não será persistido" |
| `internal/domain/statuseffect/kind.go` | Kind, Sense, Aptitude, Domain, levels, faces; parsing; default visibility per kind |
| `internal/domain/statuseffect/effect.go` | `Effect`, `Params`, `ParamsInput`, `Identity`, `Clone`, `Write` |
| `internal/domain/statuseffect/normalize.go` | `Normalize`: per-kind defaults and validation |
| `internal/domain/statuseffect/set.go` | `Apply` / `Edit` / `Remove` on one character's set |
| `internal/domain/statuseffect/piece.go` | Piece vocabulary (§5.1) and `Effect.Pieces()` (the catalog's composition) |
| `internal/domain/statuseffect/audience.go` | `Audience`, `Viewer`, `Subject`, `Visible` |
| `internal/domain/statuseffect/error.go` | domain errors |
| `internal/domain/match/match_rules.go` | `PlayerStatusEffectsPublic` |
| `migrations/20261117000000_character_status_effects.sql` | the table |
| `internal/gateway/pg/statuseffect/repository.go` | `ListBySheets`, `Write` |
| `internal/gateway/pg/round/persist_turn_close.go` | writes `TurnCloseData.StatusEffects` |
| `internal/application/match/i_repository.go` | `TurnCloseData.StatusEffects`, `IStatusEffectReader` |
| `internal/domain/match/matchsession/status_effects.go` | session storage + accessors |
| `internal/application/match/init_match_session.go` | `WithStatusEffects` + load at birth |
| `internal/application/match/status_effects.go` | `StatusEffectsUC` (Apply/Edit/Remove, no I/O) |
| `internal/app/game/message.go` | message types + payloads; `MatchFullStatePayload.StatusEffects` |
| `internal/app/game/status_effects.go` | WS arms, projection, announce, persistence timing |
| `internal/app/game/room_deps.go`, `room.go`, `turn_writes.go`, `master_board.go` | wiring points |
| `cmd/game/main.go` | wiring |
| `docs/dev/api/match-status-effects.md` | WS contract (new) |
| `docs/dev/match/status-effects.md` | design doc (new) |
| `docs/dev/api/match-combat-ws.md`, `docs/documentation-map.yaml`, `.github/instructions/*.md` | doc updates |

---

### Task 1: Rewrite the TODO rule and correct the persistence note

Spec §14, "logo no início". Docs/comments only.

**Files:**
- Modify: `AGENTS.md` (line `- **NEVER remove TODO comments** — intentional markers by the owner`)
- Modify: `internal/domain/match/character_status.go` (the block starting `// O CharacterStatus não será persistido`)

- [ ] **Step 1: Replace the AGENTS.md bullet** with (English, AGENTS.md is English):

```markdown
- **Comments and TODOs are the owner's markers.** Never delete one outside the scope of your task —
  sessions used to wipe other people's TODOs. What your change makes contradictory (a comment that
  now describes the code wrongly) must be corrected or removed in the same change, saying why in
  the commit.
```

- [ ] **Step 2: Correct the note without trimming the owner's block** (the header says "Kept verbatim… Do not trim it"). Insert right after the line `// - considerar statusEngine, ou algo assim`:

```go
//
// [Correção, 2026-11 — spec de efeitos de status §10] Os efeitos de status (caído, atordoado,
// sangrando…) SÃO persistidos, numa tabela própria da ficha (character_status_effects,
// pacote statuseffect) e valem em qualquer campanha. O que continua só em memória é este
// CharacterStatus: barras de ação/movimento, ledger, postura e velocidade.
```

- [ ] **Step 3: Verify** `go vet ./internal/domain/match/` → no output.
- [ ] **Step 4: Commit** `docs: rewrite the TODO rule and correct the status persistence note`

---

### Task 2: Domain — kinds, params, effect and set operations

**Files:**
- Create: `internal/domain/statuseffect/{kind.go,effect.go,normalize.go,set.go,error.go}`
- Test: `internal/domain/statuseffect/{normalize_test.go,set_test.go}` (`package statuseffect_test`)

**Interfaces — Produces:**

```go
package statuseffect

type Kind string // wire values below
const (
	KindParalyzed Kind = "paralyzed"      // Paralisado
	KindStupefied Kind = "stupefied"      // Atônito
	KindStunned Kind = "stunned"          // Atordoado
	KindUnconscious Kind = "unconscious"  // Inconsciente
	KindRestrained Kind = "restrained"    // Preso
	KindInjured Kind = "injured"          // Lesionado
	KindSlowed Kind = "slowed"            // Lento
	KindSense Kind = "sense"              // Sentido afetado (Visão anulada…)
	KindWeakened Kind = "weakened"        // Debilitado (Corpo/Mente/Espírito debilitado)
	KindBleeding Kind = "bleeding"; KindBurning Kind = "burning"
	KindFreezing Kind = "freezing"; KindPoisoned Kind = "poisoned"
	KindAsphyxiating Kind = "asphyxiating"
	KindProne Kind = "prone"; KindAirborne Kind = "airborne"; KindOpenGuard Kind = "openGuard"
	KindCompelled Kind = "compelled"; KindControlled Kind = "controlled"
	KindHidden Kind = "hidden"            // Escondido
)
type Sense string      // "vision" "hearing" "smell" "tact" "taste"; Sense.Skill() enum.SkillName (Vision…Taste)
type Aptitude string   // "physical" "mental" "spiritual"; Aptitude.Ability() enum.AbilityName (Physicals…)
type Domain string     // "attack" "defense" "movement" "dodge"; AllDomains() []Domain in that order
type SenseLevel string // "impaired" "nullified"
type InjuryLevel string // "hurt" "disabled"
type TouchFace string  // "touch" "balance" "preemption"
func ParseKind(s string) (Kind, error) // ErrUnknownKind; same Parse* for each type above
func (k Kind) PublicByDefault() bool   // §9 list; see Step 3

type Params struct { // normalized: only the fields the kind owns are non-zero
	Domains []Domain; Injuries map[Domain]InjuryLevel; Level SenseLevel; Faces []TouchFace
	Points, HPPerRound, Penalty int; Disadvantage bool
	Conduct string; Irresistible bool; ControllerID *uuid.UUID
}
type ParamsInput struct { // what the master sent; nil/empty = "use the default"
	Domains []Domain; Injuries map[Domain]InjuryLevel; Level SenseLevel; Faces []TouchFace
	Points, HPPerRound, Penalty *int; Disadvantage *bool
	Conduct string; Irresistible *bool; ControllerID *uuid.UUID
}
type Identity struct{ Kind Kind; Variant string } // Variant = string(Sense) | string(Aptitude) | ""
type Effect struct {
	ID, CharacterID uuid.UUID; Kind Kind; Sense Sense; Aptitude Aptitude
	Params Params; Alias string
	Audience *Audience // Task 4; nil = default visibility (no record)
	AppliedAt, UpdatedAt time.Time
}
func (e Effect) Identity() Identity
func (e Effect) Clone() Effect // deep copy of slices, map, pointers, Audience
type Write struct{ Delete bool; Effect Effect } // Delete: only Effect.ID matters

type Draft struct { Kind Kind; Sense Sense; Aptitude Aptitude; Params ParamsInput; Alias string; Audience *Audience }
type Patch struct { Params *ParamsInput; Alias *string; SetAudience bool; Audience *Audience }

func Normalize(k Kind, s Sense, a Aptitude, in ParamsInput) (Params, error)
func Apply(current []Effect, characterID uuid.UUID, d Draft, newID func() uuid.UUID, now time.Time) (next, added []Effect, err error)
func Edit(current []Effect, id uuid.UUID, p Patch, now time.Time) (next []Effect, edited Effect, err error)
func Remove(current []Effect, id uuid.UUID) (next []Effect, removed Effect, err error)

var ErrUnknownKind, ErrInvalidParams, ErrAlreadyApplied, ErrNotFound error // error.go; ErrInvalidAudience added in Task 4
```

`Audience` is defined in Task 4; for this task declare `type Audience struct{ Everyone bool; Players []uuid.UUID }` in `audience.go` already (Task 4 adds behavior and validation).

- [ ] **Step 1: Write `normalize_test.go`** — table-driven `TestNormalize`, one row per rule below. Each row: kind, sense, aptitude, input, want params or `errors.Is(err, ErrInvalidParams)`. Rows (all from the table in Step 3):
  - `"paralyzed defaults to the four domains"`: input `{Points:&5}` → `Domains: [attack defense movement dodge], Points: 5`.
  - `"paralyzed requires points"`: `{}` → ErrInvalidParams. Same row for stupefied and stunned.
  - `"bleeding defaults to one hp per round"`: `{}` → `HPPerRound: 1`.
  - `"burning requires hp per round"`: `{}` → ErrInvalidParams (also freezing, poisoned).
  - `"sense requires a level"`; `"impaired sense defaults to disadvantage"` (`{Level: impaired}` on vision → `Disadvantage: true`); `"impaired sense with only a penalty"` (`Disadvantage:&false, Penalty:&2` → `Disadvantage:false, Penalty:2`); `"impaired sense needs disadvantage or penalty"` (`&false`, no penalty → error); `"nullified sense refuses disadvantage and penalty"`.
  - `"tact requires at least one face"`; `"faces only on tact"` (vision + faces → error); `"duplicate faces refused"`.
  - `"weakened defaults to disadvantage"`; `"weakened needs disadvantage or penalty"`; `"weakened requires an aptitude"`; `"aptitude only on weakened"`; `"sense only on sense kind"`.
  - `"injured requires at least one domain"`; `"injured keeps per-domain levels"` (`{attack: disabled, movement: hurt}` preserved).
  - `"slowed requires a penalty"`.
  - `"compelled requires a conduct"`; `"conduct is trimmed and capped at 255 runes"` (256 runes → error); `"irresistible defaults to false"`.
  - `"controlled accepts no controller"` and `"controlled keeps the controller"`.
  - `"param foreign to the kind is refused"` (prone + `Points:&1` → error; restrained + domains → error).
  - `"negative or zero numbers refused"` (points 0, hpPerRound 0, penalty −1 where required ≥1; penalty 0 allowed where optional).
- [ ] **Step 2: Run** `go test ./internal/domain/statuseffect/ -run TestNormalize` → FAIL (package does not compile).
- [ ] **Step 3: Implement `kind.go`, `effect.go`, `normalize.go`, `error.go`.** Per-kind table (anything not listed for a kind must be zero/empty in the input, else `ErrInvalidParams`, wrapped with `fmt.Errorf("%w: <campo>", ErrInvalidParams)` naming the field):

  | Kind | Identity variant | Params owned (default) |
  |---|---|---|
  | paralyzed | — | domains (all four, distinct), points ≥1 required |
  | stupefied, stunned | — | points ≥1 required |
  | unconscious, restrained, asphyxiating, prone, airborne, openGuard, hidden | — | none |
  | injured | — | injuries ≥1 entry, valid domain → valid level |
  | slowed | — | penalty ≥1 required |
  | sense | sense required | level required; impaired: disadvantage (default true), penalty ≥0, at least one; nullified: neither; faces: tact → ≥1 distinct required, other senses → none |
  | weakened | aptitude required | disadvantage (default true), penalty ≥0, at least one |
  | bleeding | — | hpPerRound ≥1 (default 1) |
  | burning, freezing, poisoned | — | hpPerRound ≥1 required |
  | compelled | — | conduct (trimmed, 1–255 runes) required; irresistible (default false) |
  | controlled | — | controllerId optional |

  `PublicByDefault()` true for: prone, airborne, restrained, burning, unconscious, stunned, asphyxiating, openGuard. Everything else false (§9). Sort `Domains` and `Faces` in declaration order so equal inputs normalize equal.
- [ ] **Step 4: Run** `go test ./internal/domain/statuseffect/ -run TestNormalize -v` → PASS.
- [ ] **Step 5: Write `set_test.go`** with a fixed `now` and a counter `newID`:
  - `TestApply/"adds a normalized effect"`: empty set, `Draft{Kind: prone}` → `len(next)==1`, `added[0].CharacterID==charID`, `AppliedAt==UpdatedAt==now`, `Audience==nil`.
  - `TestApply/"refuses the same identity"`: set with prone → apply prone → `ErrAlreadyApplied`, `next` nil.
  - `TestApply/"different senses coexist"`: vision then hearing → both kept. `"same sense refused"`: vision impaired then vision nullified → `ErrAlreadyApplied`.
  - `TestApply/"one weakened per aptitude"`: physical + mental coexist; second physical refused.
  - `TestApply/"unconscious brings prone"`: empty set → added = [unconscious, prone].
  - `TestApply/"unconscious keeps an existing prone"` (Review Focus 4): set with prone → added = [unconscious] only; one prone in `next`.
  - `TestApply/"alias over 60 runes refused"` → `ErrInvalidParams`; alias is trimmed.
  - `TestApply/"does not mutate the input slice"`.
  - `TestEdit/"replaces params whole"`: paralyzed{points 5, all domains} edited with `Params:{Points:&3, Domains:[attack]}` → points 3, domains [attack]; `UpdatedAt==now2`, `AppliedAt` unchanged.
  - `TestEdit/"nil params keep params"`; `"alias empty clears"`; `"set audience nil returns to default"`; `"invalid params leave the set untouched"` (error, `current[i]` unchanged); `"unknown id"` → `ErrNotFound`.
  - `TestRemove/"removes by id"`; `"removing unconscious keeps prone"` (Review Focus 4); `"unknown id"` → `ErrNotFound`.
- [ ] **Step 6: Run** → FAIL. **Step 7: Implement `set.go`** (all functions return fresh slices of `Clone()`d effects; Edit re-runs `Normalize` with the effect's own identity; Apply validates `Draft.Audience` only via Task 4 later — for now store it as given). **Step 8: Run** `go test ./internal/domain/statuseffect/ -v` → PASS; `go vet ./...`.
- [ ] **Step 9: Commit** `feat(statuseffect): catalog kinds, params and set operations`

---

### Task 3: Domain — effect pieces and the catalog's composition

Spec §5.1 (pieces), §5.2 (containment), §5.3/§5.4 (what each status does). Data only; nothing consumes it in this phase.

**Files:**
- Create: `internal/domain/statuseffect/piece.go`
- Test: `internal/domain/statuseffect/piece_test.go`

**Interfaces — Produces:**

```go
type PieceKind string
const (
	PieceNoAct PieceKind = "noAct"; PieceNoReact = "noReact"
	PieceDomainLocked = "domainLocked"; PieceDomainWarning = "domainWarning"
	PieceReactionRestricted = "reactionRestricted"; PieceTestPenalty = "testPenalty"
	PieceAutoFail = "autoFail"; PieceNoDisplacement = "noDisplacement"; PieceLosesHP = "losesHp"
	PieceRecovery = "recovery"; PieceTriggeredTest = "triggeredTest"
	PieceConduct = "conduct"; PieceControl = "control"; PieceCounter = "counter"
)
type Reaction string    // "reflexDodge" "evasion" "defense" "repel" "escape"
type Restriction string // "forbidden" "disadvantage" "rolledOnly"
type Trigger string     // "onHit" "onMove"
type Scope struct { Domain Domain; Aptitude Aptitude; Sense Sense; Face TouchFace; MoveSpeed bool } // exactly one set
type TestRef struct { Skill enum.SkillName; Attribute enum.AttributeName } // exactly one set
type Piece struct {
	Kind PieceKind
	Domain Domain                   // domainLocked, domainWarning (one piece per domain)
	Reaction Reaction; Restriction Restriction // reactionRestricted
	Scope Scope                      // testPenalty, autoFail
	Disadvantage int; Penalty int    // testPenalty (levels of disadvantage; −N)
	HPPerRound int                   // losesHp
	Test TestRef; Points int         // recovery; triggeredTest uses Test
	Triggers []Trigger; OnFail Kind  // triggeredTest
	Conduct string; Irresistible bool // conduct
	ControllerID *uuid.UUID          // control
}
func (e Effect) Pieces() []Piece
```

- [ ] **Step 1: Write `piece_test.go`.** Helper `has(pieces, want Piece) bool` (compare by value; slices by content). Tests:
  - `TestPieces_Composition` table — each row: effect → the exact pieces (order-insensitive):
    - paralyzed{domains [attack, movement], points 4} → domainLocked attack, domainLocked movement, noDisplacement (movement marked), recovery{Tenacity, 4}. With domains [attack] only → no noDisplacement.
    - stupefied{points 3} → noAct, noReact, recovery{Attribute: Resilience, 3}.
    - stunned{points 8} → domainLocked ×4, noDisplacement, noAct, noReact, autoFail{Sense} ×5, recovery{Tenacity, 8}.
    - unconscious → same as stunned **without** recovery.
    - restrained → noDisplacement, reactionRestricted{escape, forbidden}.
    - injured{attack: hurt, movement: disabled} → testPenalty{Domain attack, Disadvantage 1}; testPenalty{Domain movement, Disadvantage 1} + domainWarning movement (disabled contains hurt, §5.2).
    - slowed{penalty 2} → testPenalty{MoveSpeed, Penalty 2}.
    - sense vision impaired{disadvantage, penalty 1} → testPenalty{Sense vision, 1, 1}. Vision nullified → autoFail{Sense vision} + testPenalty{Domain attack, Disadvantage 1} + testPenalty{Domain defense, Disadvantage 1}. Hearing nullified → autoFail{Sense hearing} only.
    - sense tact impaired faces [touch, balance] → testPenalty{Face touch} + testPenalty{Face balance} (each with the status's disadvantage/penalty) + triggeredTest{[onHit, onMove], Tact, OnFail prone}. Tact nullified [balance] → autoFail{Face balance} + the same triggeredTest. Tact [touch] → no triggeredTest.
    - weakened mental{penalty 2, no disadvantage} → testPenalty{Aptitude mental, 0, 2}.
    - bleeding{3} → losesHp 3 (same for burning, freezing, poisoned).
    - asphyxiating → counter.
    - prone → reactionRestricted{reflexDodge, disadvantage}, {repel, disadvantage}.
    - airborne → {reflexDodge, forbidden}, {escape, forbidden}.
    - openGuard → {reflexDodge, forbidden}, {repel, forbidden}, {defense, rolledOnly}, {escape, disadvantage}.
    - compelled{conduct "fugir", irresistible} → conduct piece with both. controlled{controller X} → control X.
    - hidden → no pieces.
  - `TestPieces_Containment` (§5.2 as an invariant): pieces of `paralyzed{all four}` minus recovery ⊆ stunned; stupefied minus recovery ⊆ stunned; stunned minus recovery ⊆ unconscious; restrained's noDisplacement ⊆ paralyzed{movement}.
- [ ] **Step 2: Run** `go test ./internal/domain/statuseffect/ -run TestPieces` → FAIL.
- [ ] **Step 3: Implement `piece.go`.** Build unconscious from stunned's builder minus recovery, stunned from paralyzed(all)+stupefied minus their recoveries plus the five autoFails — the containment lives in the code, not in copy-pasted lists. Leave these markers as comments (Portuguese) where the piece would go, without inventing it:
  - stupefied/stunned: `// TODO Nen: "não usa habilidades" — peça a criar com a migração do Nen (spec §13).`
  - tact balance face: `// TODO(G1, dono): o equilíbrio afetado pesa em ataque, defesa, esquiva e movimento (spec §5.3), mas o spec não diz quanto nem o que o anulado faz — lacuna levada ao dono; peça não declarada até a resposta.`
  - prone and airborne movement: `// Provisório até o movimento (spec §13): só rolar/deslizar (Caído) e só a trajetória (No ar).`
  - asphyxiating: `// Fôlego, barra de afogamento e testes: fase 4 (spec §5.3 Ar).`
- [ ] **Step 4: Run** → PASS. `go vet ./...`.
- [ ] **Step 5: Commit** `feat(statuseffect): declare effect pieces and the catalog composition`

---

### Task 4: Domain — audience and visibility rule; match rule flag

Spec §9, §14 ("Configuração da mesa… parâmetro em MatchRules").

**Files:**
- Modify: `internal/domain/statuseffect/audience.go`, `set.go` (validate audience in Apply/Edit), `error.go`
- Modify: `internal/domain/match/match_rules.go` (+ `match_rules_test.go`)
- Test: `internal/domain/statuseffect/audience_test.go`

**Interfaces — Produces:**

```go
type Audience struct{ Everyone bool; Players []uuid.UUID } // Everyone ⇒ Players empty; Players: distinct
type Viewer struct{ UserID uuid.UUID; IsMaster bool }
type Subject struct{ OwnerID uuid.UUID; IsNPC bool } // the character the effect is on
func (a Audience) Validate() error // ErrInvalidAudience: Everyone with players, duplicates, uuid.Nil
func Visible(e Effect, v Viewer, s Subject, pcPublicByDefault bool) bool
var ErrInvalidAudience error
```
`MatchRules.PlayerStatusEffectsPublic bool` — default `false` in `NewDefaultMatchRules()` (doc comment PT: configuração da mesa do spec §9; sem tela ainda).

- [ ] **Step 1: Write `audience_test.go`** — `TestVisible` table over (kind public/hidden, audience, subject NPC/PC, viewer master/owner/other, flag):
  - master sees everything, always (incl. `{Players: []}`).
  - no record, public kind (prone) → owner and other see.
  - no record, hidden kind (poisoned), PC → owner sees, other does not.
  - no record, hidden kind, NPC (owner = master) → other player does not.
  - no record, hidden kind, PC, `pcPublicByDefault=true` → other sees; same on NPC → other does NOT (flag is for PCs only).
  - `{Everyone: true}` on hidden kind → other sees.
  - `{Players: []}` on a PC → owner does NOT see ("só o mestre").
  - `{Players: [other]}` → other sees, owner does not.
  - `{Players: [owner]}` on public kind → other does not (explicit list overrides the default).
  - `TestAudienceValidate`: everyone+players, duplicate player, `uuid.Nil` → `ErrInvalidAudience`.
  - In `set_test.go`: `TestApply/"invalid audience refused"`, `TestEdit/"invalid audience refused"` → `ErrInvalidAudience`, set untouched.
  - `match_rules_test.go`: defaults have `PlayerStatusEffectsPublic == false`.
- [ ] **Step 2: Run** → FAIL. **Step 3: Implement.** **Step 4: Run** `go test ./internal/domain/statuseffect/ ./internal/domain/match/ -v` → PASS.
- [ ] **Step 5: Commit** `feat(statuseffect): who sees each effect — audience list with kind defaults`

---

### Task 5: Migration and gateway

**Files:**
- Create: `migrations/20261117000000_character_status_effects.sql`
- Create: `internal/gateway/pg/statuseffect/repository.go` (package `pgstatuseffect`)
- Test: `internal/gateway/pg/statuseffect/statuseffect_integration_test.go`
- Modify: `internal/application/match/i_repository.go` (reader interface)

**Interfaces — Produces:**

```go
// i_repository.go
type IStatusEffectReader interface {
	ListBySheets(ctx context.Context, sheetUUIDs []uuid.UUID) (map[uuid.UUID][]statuseffect.Effect, error)
}
// pgstatuseffect
func NewRepository(q pgfs.IQuerier) *Repository
func (r *Repository) ListBySheets(ctx context.Context, sheetUUIDs []uuid.UUID) (map[uuid.UUID][]statuseffect.Effect, error) // never nil map; each slice ordered by applied_at, uuid
func (r *Repository) Write(ctx context.Context, ops []statuseffect.Write) error // in order, atomically
```

- [ ] **Step 1: Write the migration** (same goose layout as `20261010000000_cancelled_actions.sql`, with a PT comment block explaining: tabela da ficha, vale em qualquer campanha (§10); `variant` = sentido ou Aptidão, '' nos demais, parte da identidade; `params` só os campos do tipo; `audience NULL` = visibilidade padrão do tipo (§9, sem registro = padrão)):

```sql
CREATE TABLE IF NOT EXISTS character_status_effects (
  uuid                  UUID         PRIMARY KEY,
  character_sheet_uuid  UUID         NOT NULL REFERENCES character_sheets(uuid) ON DELETE CASCADE,
  kind                  VARCHAR(32)  NOT NULL,
  variant               VARCHAR(32)  NOT NULL DEFAULT '',
  params                JSONB        NOT NULL DEFAULT '{}',
  alias                 VARCHAR(60)  NOT NULL DEFAULT '',
  audience              JSONB,
  applied_at            TIMESTAMPTZ  NOT NULL,
  updated_at            TIMESTAMPTZ  NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_character_status_effects_identity
  ON character_status_effects(character_sheet_uuid, kind, variant);
```
Down: drop index, drop table.

- [ ] **Step 2: Write the integration tests** (`//go:build integration`, pattern of `cancelledaction_integration_test.go`; `pgtest.TruncateAll` per sub-test; sheets via `pgtest.InsertTestCharacterSheet`):
  - `TestWrite/"upsert inserts then updates"`: write upsert of a sense effect (vision, impaired, alias, `Audience{Players:[p]}`), list → equal (times truncated to µs); upsert again with new params/alias/`Audience nil`/UpdatedAt → list shows the update, `AppliedAt` unchanged, `Audience == nil`.
  - `"delete removes and is idempotent"`: delete twice → no error, list empty.
  - `"ops apply in order"`: [upsert A(prone), delete A, upsert B(prone, new uuid)] in one call → only B (the unique identity index would reject B before A's delete).
  - `"atomic"`: [upsert valid, upsert with a sheet UUID that does not exist] → error, list empty.
  - `"runs inside an outer transaction"`: `tx := pool.Begin`; `NewRepository(tx).Write`; rollback → list empty.
  - `TestListBySheets/"groups by sheet and orders"`: two sheets, three effects → map with two keys, order by applied_at; a sheet with none is absent; empty input → empty map, no query error.
  - `"round-trips every params field"`: injured map, faces, controllerId, conduct, irresistible, disadvantage false + penalty.
- [ ] **Step 3: Run** `go test -tags=integration ./internal/gateway/pg/statuseffect/...` → FAIL.
- [ ] **Step 4: Implement `repository.go`.** `Write`: `tx, err := r.q.Begin(ctx)` (a pool begins a transaction; a `pgx.Tx` opens a savepoint — the same code serves both callers), unconditional rollback defer, commit. Upsert: `INSERT … ON CONFLICT (uuid) DO UPDATE SET params, alias, audience, updated_at` (sheet, kind, variant, applied_at are immutable). Params/audience JSON via gateway-local row structs with camelCase tags (`domains, injuries, level, faces, points, hpPerRound, penalty, disadvantage, conduct, irresistible, controllerId`; audience `{"everyone":true}` / `{"players":[…]}`) — the domain stays tag-free. `variant` = `string(Sense)` or `string(Aptitude)`. `ListBySheets`: `WHERE character_sheet_uuid = ANY($1) ORDER BY applied_at, uuid`.
- [ ] **Step 5: Run** the package's integration tests → PASS; `go vet -tags=integration ./internal/gateway/pg/...`.
- [ ] **Step 6: Commit** `feat(statuseffect): character_status_effects table and repository`

---

### Task 6: Status writes in the turn's transaction

**Files:**
- Modify: `internal/application/match/i_repository.go` (`TurnCloseData`)
- Modify: `internal/gateway/pg/round/persist_turn_close.go`
- Test: `internal/gateway/pg/round/round_integration_test.go`

**Interfaces — Produces:** `TurnCloseData.StatusEffects []statuseffect.Write` — doc comment (PT): o que o mestre mudou em efeitos de status enquanto o turno estava aberto, mais o que uma gravação anterior não conseguiu gravar (primeiro), aplicado em ordem nesta transação (§10).

- [ ] **Step 1: Test** (in the existing `PersistTurnClose` test function, reuse its fixture builder): `"writes status effects in the turn transaction"` — close with `StatusEffects: [upsert prone on the victim sheet]` → `pgstatuseffect.NewRepository(pool).ListBySheets` has it. `"a failing close writes no status effect"` — make the close fail the way an existing failure sub-test does (or an upsert on a non-existent sheet) → nothing listed and no turn row.
- [ ] **Step 2: Run** `go test -tags=integration ./internal/gateway/pg/round/...` → FAIL.
- [ ] **Step 3: Implement**: after the cancelled actions block, `if len(d.StatusEffects) > 0 { if err := pgstatuseffect.NewRepository(tx).Write(ctx, d.StatusEffects); err != nil { return fmt.Errorf("PersistTurnClose status effects: %w", err) } }`.
- [ ] **Step 4: Run** → PASS; vet.
- [ ] **Step 5: Commit** `feat(round): write status effect changes with the turn close`

---

### Task 7: Session storage, loading at birth, and the use case

**Files:**
- Create: `internal/domain/match/matchsession/status_effects.go` (+ `status_effects_test.go`)
- Modify: `internal/application/match/init_match_session.go` (+ test)
- Create: `internal/application/match/status_effects.go` (+ `status_effects_test.go`)
- Modify: `internal/application/match/error.go` (`ErrUnknownCharacter` if no equivalent exists — check first)

**Interfaces — Consumes:** Task 2/4 domain API. **Produces:**

```go
// matchsession — the caller holds the room's r.mu (session has no lock of its own)
func (s *MatchSession) SetStatusEffects(byChar map[uuid.UUID][]statuseffect.Effect) // at birth
func (s *MatchSession) StatusEffectsOf(charID uuid.UUID) []statuseffect.Effect      // clones, ordered by AppliedAt, ID; never nil
func (s *MatchSession) ReplaceStatusEffects(charID uuid.UUID, effects []statuseffect.Effect) // empty ⇒ entry deleted
func (s *MatchSession) FindStatusEffect(id uuid.UUID) (statuseffect.Effect, bool)
func (s *MatchSession) CharactersWithStatusEffects() []uuid.UUID
// (reuse an existing "is this user a player participant" accessor if there is one; else add IsPlayerParticipant(userID) bool)

// application/match
func (uc *InitMatchSessionUC) WithStatusEffects(r IStatusEffectReader) *InitMatchSessionUC // optional; nil = none loaded

type IStatusEffects interface {
	Apply(s *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, in ApplyStatusEffectInput, now time.Time) ([]statuseffect.Write, error)
	Edit(s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID, p statuseffect.Patch, now time.Time) (charID uuid.UUID, w []statuseffect.Write, err error)
	Remove(s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID) (charID uuid.UUID, w []statuseffect.Write, err error)
}
type ApplyStatusEffectInput struct{ CharacterIDs []uuid.UUID; Draft statuseffect.Draft }
type StatusEffectsUC struct{}; func NewStatusEffectsUC() *StatusEffectsUC
```

- [ ] **Step 1: Session tests** (`status_effects_test.go`, session built with `NewMatchSession` + two participants): set/get round-trip returns clones (mutating the returned slice does not change the session); `ReplaceStatusEffects(id, nil)` removes the entry; `FindStatusEffect` across characters.
- [ ] **Step 2: Init tests**: with a fake reader returning effects for one participant sheet → `session.StatusEffectsOf(sheet)` has them; the reader receives exactly the participants' sheet UUIDs; reader error → `Init` returns it (a session must not be born without its statuses — same policy as the clock); without `WithStatusEffects` → no reader call, empty.
- [ ] **Step 3: UC tests** (`TestStatusEffectsUC`): 
  - `"applies to several characters"` → each gets one effect; writes = one upsert per added effect (unconscious → 2 per character).
  - `"all or nothing when one already has it"` (Review Focus 3): A has prone, apply prone to [A, B] → `statuseffect.ErrAlreadyApplied`, B still has none, no writes.
  - `"refuses a non-master"` → `ErrNotMatchMaster`; `"refuses an ended match"` → `matchsession.ErrMatchEnded`; `"refuses an unknown character"` → `ErrUnknownCharacter`; `"refuses empty or repeated character ids"` → `ErrInvalidParams` (wrapped); `"refuses a controller outside the match or equal to the target"`; `"refuses an audience player who is not a player of the match"` → `ErrInvalidAudience`.
  - `"edit returns the character and an upsert"`; `"remove returns a delete"`; unknown id → `statuseffect.ErrNotFound`.
- [ ] **Step 4: Run** `go test ./internal/domain/match/matchsession/ ./internal/application/match/ -run 'StatusEffect|Init'` → FAIL. **Step 5: Implement.** The UC computes every character's next set first and only then calls `ReplaceStatusEffects` for all. **Step 6: Run** → PASS; `go vet ./...`.
- [ ] **Step 7: Commit** `feat(match): status effects in the session, loaded at birth, changed by the master`

---

### Task 8: WS commands, per-recipient projection and reconnect

Contract shapes below are final — Task 10 documents them as they are.

**Files:**
- Modify: `internal/app/game/message.go`, `room.go` (three `case` arms delegating; `buildMatchFullState`), `room_deps.go`
- Create: `internal/app/game/status_effects.go`
- Test: `internal/app/game/status_effects_e2e_test.go` (on `combatFixture`: `connectTable`, `withNPCTarget`, `withBystander`, `openAttackTurn`, `collector`)

**Interfaces — Produces (message.go):**

```go
MsgTypeApplyStatusEffect    MessageType = "apply_status_effect"    // client → server, master only
MsgTypeEditStatusEffect     MessageType = "edit_status_effect"
MsgTypeRemoveStatusEffect   MessageType = "remove_status_effect"
MsgTypeStatusEffectsUpdated MessageType = "status_effects_updated" // server → client, projected

type StatusEffectParamsPayload struct {
	Domains []string `json:"domains,omitempty"`; Injuries map[string]string `json:"injuries,omitempty"`
	Level string `json:"level,omitempty"`; Faces []string `json:"faces,omitempty"`
	Points *int `json:"points,omitempty"`; HPPerRound *int `json:"hpPerRound,omitempty"`
	Penalty *int `json:"penalty,omitempty"`; Disadvantage *bool `json:"disadvantage,omitempty"`
	Conduct string `json:"conduct,omitempty"`; Irresistible *bool `json:"irresistible,omitempty"`
	ControllerID *uuid.UUID `json:"controllerId,omitempty"`
}
type StatusEffectVisibilityPayload struct {
	Mode string `json:"mode"` // "default" | "public" | "players"
	PlayerIDs []uuid.UUID `json:"playerIds,omitempty"` // mode "players"; [] = só o mestre
}
type ApplyStatusEffectPayload struct {
	CharacterIDs []uuid.UUID `json:"characterIds"`; Kind string `json:"kind"`
	Sense string `json:"sense,omitempty"`; Aptitude string `json:"aptitude,omitempty"`
	Params StatusEffectParamsPayload `json:"params"`; Alias string `json:"alias,omitempty"`
	Visibility *StatusEffectVisibilityPayload `json:"visibility,omitempty"` // absent = default
}
type EditStatusEffectPayload struct {
	StatusEffectID uuid.UUID `json:"statusEffectId"`
	Params *StatusEffectParamsPayload `json:"params,omitempty"`       // absent = untouched; present = replaces all
	Alias *string `json:"alias,omitempty"`                            // absent = untouched; "" clears
	Visibility *StatusEffectVisibilityPayload `json:"visibility,omitempty"` // absent = untouched; mode "default" = back to default
}
type RemoveStatusEffectPayload struct{ StatusEffectID uuid.UUID `json:"statusEffectId"` }
type StatusEffectPayload struct {
	ID uuid.UUID `json:"id"`; Kind string `json:"kind"`
	Sense string `json:"sense,omitempty"`; Aptitude string `json:"aptitude,omitempty"`
	Params StatusEffectParamsPayload `json:"params"` // every field the kind owns, defaults applied; the rest omitted
	Alias string `json:"alias,omitempty"`; AppliedAt time.Time `json:"appliedAt"`
	Visibility *StatusEffectVisibilityPayload `json:"visibility,omitempty"` // master only
	Public *bool `json:"public,omitempty"`                               // master only: every player sees it now
}
type StatusEffectsUpdatedPayload struct {
	CharacterID uuid.UUID `json:"characterId"`
	StatusEffects []StatusEffectPayload `json:"statusEffects"` // the recipient's WHOLE visible list for the character; never null
}
// MatchFullStatePayload gains:
StatusEffects map[uuid.UUID][]StatusEffectPayload `json:"statusEffects,omitempty"` // projected; characters with nothing visible absent
```
`RoomDeps.StatusEffectsUC appmatch.IStatusEffects` — nil ⇒ the three commands answer `game_error` (the usual "no capability").

Error codes (`error` message): `forbidden` (not master), `invalid_payload` (bad JSON, unknown enum string, empty/duplicate `characterIds`, `statusEffectId` nil), `match_not_started` (no session — lobby), `match_already_ended`, `unknown_character`, `status_effect_invalid` (`ErrUnknownKind`/`ErrInvalidParams`/`ErrInvalidAudience`; message carries the field), `status_effect_exists`, `status_effect_not_found`, `game_error` (anything else).

- [ ] **Step 1: Write the e2e tests** (fixture with a PC owned by the fixture player, a bystander player, and an NPC):
  - `TestStatusEffects_ApplyPublic`: master applies prone to the victim → master, owner and bystander each get exactly one `status_effects_updated{characterId: victim}` with one effect `kind: "prone"`; only the master's carries `visibility {mode:"default"}` and `public: true`.
  - `TestStatusEffects_ApplyHiddenOnPC`: poisoned{hpPerRound 2} → master and owner get it; bystander gets **no message at all**.
  - `TestStatusEffects_ApplyHiddenOnNPC`: poisoned on the NPC → only the master.
  - `TestStatusEffects_VisibilityChange` (Review Focus 2): hidden on PC; edit `visibility {mode:"public"}` → bystander gets a list with it, owner gets nothing (their list did not change), master gets it; then edit `{mode:"players", playerIds: []}` → owner and bystander each get an empty list; then edit only `alias` → only the master gets a message.
  - `TestStatusEffects_Remove`: remove a public effect → everyone gets `statusEffects: []`.
  - `TestStatusEffects_MultiTargetAllOrNothing`: one target already prone → master gets `error status_effect_exists`, nobody gets `status_effects_updated`.
  - `TestStatusEffects_Refusals`: player sends apply → `forbidden`; unknown kind → `status_effect_invalid`; prone + points → `status_effect_invalid`; unknown character → `unknown_character`; edit unknown id → `status_effect_not_found`; lobby fixture (`inLobby`) → `match_not_started`.
  - `TestStatusEffects_Reconnect` (Review Focus 5): public effect on the NPC, hidden on the NPC, hidden on the PC; bystander reconnects → `match_full_state.statusEffects` has only the NPC's public one; owner reconnects → the NPC's public one and their PC's hidden one; master → all three, with `visibility`.
- [ ] **Step 2: Run** `go test ./internal/app/game/ -run TestStatusEffects` → FAIL.
- [ ] **Step 3: Implement** `status_effects.go`:
  - `handleApplyStatusEffect/handleEditStatusEffect/handleRemoveStatusEffect(client *Client, raw json.RawMessage)`: master check, decode + parse wire strings into domain types (wire → `statuseffect.Draft`/`Patch`; `mode "default"` ⇒ `Audience nil`), `session == nil` ⇒ `match_not_started`.
  - Under `r.mu.Lock()`: snapshot **before** = per connected client, per affected character, the JSON of the projected list; call the UC with `r.now()`; on error unlock and answer `statusEffectErrorCode(err)`; on success hand the writes to `r.holdStatusEffectWritesLocked(writes)` (Task 9 — in this task a stub that does nothing, so Task 8 stays testable alone); snapshot **after**; unlock. Then `dispatchPerPlayer` sends `status_effects_updated` to each recipient whose before ≠ after for that character (a client that connected in between has no before entry and is skipped — its `match_full_state` was built after the change).
  - `projectStatusEffectsLocked(charID uuid.UUID, viewer statuseffect.Viewer) []StatusEffectPayload`: owner = `session.GetCharToPlayer()[charID.String()]`; `IsNPC = owner == r.masterUUID`; filter with `statuseffect.Visible(…, session.GetRules().PlayerStatusEffectsPublic)`; master gets `visibility` and `public` (computed as `Visible` for a hypothetical non-owner, non-master viewer, i.e. `uuid.Nil` user).
  - `buildMatchFullState`: fill `StatusEffects` for the viewer from `CharactersWithStatusEffects()`.
- [ ] **Step 4: Run** → PASS; then the whole package `go test ./internal/app/game/` (no regression) and `go vet ./...`.
- [ ] **Step 5: Commit** `feat(game): master applies, edits and removes status effects over WS, projected per viewer`

---

### Task 9: When the write happens — outside a turn, with the turn, and healing

**Files:**
- Modify: `internal/app/game/status_effects.go`, `turn_writes.go` (field `statusEffects []statuseffect.Write`), `room.go` (`persistClosedTurn`; Room field `unwrittenStatusEffects []statuseffect.Write`), `room_deps.go`, `master_board.go` (live NPC), `cmd/game/main.go`
- Test: `internal/app/game/status_effects_e2e_test.go` (fake store `fakeStatusEffectStore` recording each `Write` call and able to fail on demand)

**Interfaces — Produces:**

```go
// room_deps.go
type IStatusEffectRepo interface {
	ListBySheets(ctx context.Context, sheetUUIDs []uuid.UUID) (map[uuid.UUID][]statuseffect.Effect, error)
	Write(ctx context.Context, ops []statuseffect.Write) error
}
RoomDeps.StatusEffectRepo IStatusEffectRepo // nil: nothing is written (and nothing accumulates)
// status_effects.go
func (r *Room) holdStatusEffectWritesLocked(ops []statuseffect.Write) (flushNow bool) // caller holds r.mu (write)
func (r *Room) flushStatusEffects()                                                   // caller holds NO lock
```

- [ ] **Step 1: Tests:**
  - `"outside a turn writes at once"`: apply → the store got exactly one `Write` with one upsert, before the master's `status_effects_updated` assertion completes (await with the fixture's polling helper pattern).
  - `"inside a turn waits for the close"` (house rule): `openAttackTurn`; apply + edit → store got nothing; close the turn → `mockRoundRepoHandler`'s recorded `TurnCloseData.StatusEffects` == [upsert, upsert] in order; store still got nothing.
  - `"a failed out-of-turn write is retried with the next one"` (Review Focus 1): store fails once; apply A → no error to the master, memory has A; apply B → the store's next call carries [A upsert, B upsert].
  - `"a failed close keeps the turn's status writes for the next write"` (Review Focus 1): round repo fails the close; inside the turn apply A; close → fails; next out-of-turn apply B → store gets [A, B].
  - `"unwritten writes ride the next close first"`: store fails on apply A (outside turn); open turn; apply B; close → `TurnCloseData.StatusEffects == [A, B]`; then a later out-of-turn write does not resend A.
  - `"live NPC arrives with its effects"`: store `ListBySheets` returns a public prone for the NPC sheet; `add_npc` → the table gets `status_effects_updated` for the NPC (after `npc_added`), master included.
  - `"no repo, no accumulation"`: `StatusEffectRepo` nil → apply works in memory; nothing panics; close's `StatusEffects` empty.
- [ ] **Step 2: Run** → FAIL.
- [ ] **Step 3: Implement:**
  - `holdStatusEffectWritesLocked`: open turn (`openTurnIDLocked() != uuid.Nil`) ⇒ append to `turnWritesLocked(id).statusEffects`, return false; else if `StatusEffectRepo == nil` ⇒ drop, return false; else append to `r.unwrittenStatusEffects`, return true. The arms call `r.flushStatusEffects()` after unlocking when it returned true.
  - `flushStatusEffects`: `r.persistMu.Lock()` (same serializer as board saves and closes; lock order `persistMu` → `r.mu`); copy `r.unwrittenStatusEffects` under `r.mu`; `Write` outside it; success ⇒ under `r.mu` drop exactly that prefix; failure ⇒ `log.Printf` (PT) saying how many writes stay pending and that the next write retries them.
  - `persistClosedTurn`: in its existing critical section, `pending := clone(r.unwrittenStatusEffects)`; `StatusEffects: append(pending, inTurn.statusEffects...)`; success ⇒ drop `len(pending)` from the front of `r.unwrittenStatusEffects`; failure ⇒ append `inTurn.statusEffects` to `r.unwrittenStatusEffects` (memory still holds the change; the next write carries it — same healing as the HP in `unwritten.go`). Mention status effects in the existing failure log line.
  - Live NPC (`enrollLiveNPC` in `master_board.go`): after the session has the NPC and `npc_added` is queued, `ListBySheets([npc])` with no lock held; under `r.mu`, `ReplaceStatusEffects` + before/after diff (before = empty); announce via the Task 8 path. A read error is logged and the NPC plays without statuses (enrolment already happened; same "log and go on" as the board load).
  - `cmd/game/main.go`: `statusEffectRepository := pgstatuseffect.NewRepository(pool)`; `initSessionUC.WithStatusEffects(statusEffectRepository)`; `RoomDeps{… StatusEffectsUC: match.NewStatusEffectsUC(), StatusEffectRepo: statusEffectRepository}`.
- [ ] **Step 4: Run** `go test ./internal/app/game/ -race` → PASS; `go build ./... && go vet ./... && go vet -tags=integration ./...`.
- [ ] **Step 5: Commit** `feat(game): status effect writes follow the turn rule and heal after a failure`

---

### Task 10: Contract and docs

**Files:**
- Create: `docs/dev/api/match-status-effects.md`, `docs/dev/match/status-effects.md`
- Modify: `docs/dev/api/match-combat-ws.md` (§3 index: link the new contract; `match_full_state`: the `statusEffects` field), `docs/documentation-map.yaml`, `AGENTS.md` (Known Issues), `.github/instructions/domain-map.instructions.md`, `.github/instructions/game-server.instructions.md`

- [ ] **Step 1: Contract** `match-status-effects.md` (PT, same structure as `match-clock.md`): who may send what; the three commands and `status_effects_updated` with JSON examples taken from the Task 8 tests; the per-kind params table (Task 2 Step 3) with wire names, identity variant and defaults; enum values; visibility modes and the default rule (decision 3–5 above); who receives what and that only changed views are sent; `match_full_state.statusEffects`; error codes; inside a turn the change is live at once but durable only with the close. Front-only material: PT display names (§5.3 names, "Corpo/Mente/Espírito debilitado", sense statuses as "Visão prejudicada/anulada") and alias suggestions per kind (copy §8's list verbatim). State plainly what is NOT there yet (no effect on actions/rolls, Hidden does not hide the piece, no history entry, no death).
- [ ] **Step 2: Design doc** `docs/dev/match/status-effects.md` (PT): package map; pieces and `Effect.Pieces()` (containment built in code; decision 7; G1 pending); visibility model; persistence timing and healing; known limitation: a sheet live in two rooms at once keeps two memories (the last write wins on disk); history rows for status changes still to come.
- [ ] **Step 3: Index updates:** `documentation-map.yaml` entries for `internal/domain/statuseffect/`, `internal/gateway/pg/statuseffect/`, `internal/app/game/status_effects.go`, `internal/application/match/status_effects.go` → both new docs; `AGENTS.md` Known Issues: one paragraph "Efeitos de status — fase 1" (what exists, what is deliberately missing, pointer to the docs); domain-map row + current-state line; game-server: a short "Status effects" section (session holds them; room decides the write moment; `persistMu`).
- [ ] **Step 4: Commit** `docs: status effects phase 1 contract and design`

---

### Task 11: Prove it for real, then PR and handoff

- [ ] **Step 1: Full local suite:** `go test ./... -race`; `go test -tags=integration -p 1 ./internal/gateway/pg/...` (with `TEST_DATABASE_URL`); `go vet -tags=integration ./... && go vet -tags=smoke ./...`; `golangci-lint cache clean && ~/go/bin/golangci-lint run --max-same-issues 0 --max-issues-per-linter 0`. All clean — paste the tails in the PR.
- [ ] **Step 2: Real run** (CLAUDE.md "Entrega" step 1; `docs/dev/seed.md`): `make migrate-up` on the dev DB; `go run ./cmd/devseed`; start API and game server from this worktree. With a throwaway WS script outside the repo (scratchpad; Node `ws` or Go `gorilla/websocket`), as master: apply `prone` (public) and `poisoned` (hidden) to the seed player's character; as the seed player and a third account: confirm who received what; `psql` shows two rows in `character_status_effects` (`audience` NULL). **Restart the game server**, reconnect: `match_full_state.statusEffects` still has both, projected. Open a turn, apply one more, check no row before closing, row after. Never use the owner's own logged-in tabs (memory: verification never kicks the owner's tabs).
- [ ] **Step 3: PR** (base `docs/status-effects-spec`, stacked on #110): what was verified and how (Step 1 tails, Step 2 transcript summary); what was not (no front, so nothing visual); the decisions list above; G1 for the owner; out-of-scope list. End with the attribution line.
- [ ] **Step 4: Handoff:** if Step 2 covered everything, say so in the PR and do not run `dev-checkout.sh`; otherwise run `./dev-checkout.sh feat/status-effects-base` from `System_X_System_Project/` and say in the PR what to look at.
