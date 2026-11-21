# Status Effects — Phase 1 (back-end) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Status effects exist as data bound to the character sheet (any campaign, any match), declared as combinations of effect pieces in one catalog; the master applies, edits and removes them by hand over the match WebSocket; each one reaches only who may see it (a viewer list, no record = the kind's default) AND who sees the character (fog); Hidden takes the character off the map and bars of who did not perceive it; and every piece that needs no new dice rule already works (cannot act, cannot react, locked domain, no displacement, forbidden reactions, master warnings). A second PR records status changes and a hidden character's actions in the history.

**Architecture:** A pure domain package `internal/domain/statuseffect` holds the catalog (kinds, params, defaults, pieces, a `Summary` of what a set forbids), the effect entity, the set operations and the viewer-list rule. A new table `character_status_effects` (FK → `character_sheets`) is read when the match session is born and written by a gateway that runs on a pool or inside `PersistTurnClose`'s transaction. The `MatchSession` holds the effects and enforces the blocking pieces at the points where actions and reactions enter; `appmatch.StatusEffectsUC` changes them (no I/O); the `Room` owns sight (fog + Hidden) and projection per recipient, and decides WHEN a write happens: at once outside a turn, with the turn's close inside one, retried on the next write if it failed.

**Tech Stack:** Go 1.23, PostgreSQL + goose, pgx, gorilla/websocket (existing), standard `testing`.

**Spec:** `docs/superpowers/specs/2026-11-15-status-effects-design.pt-br.md` (owner-approved, PR #110). Read §1, §4, §5, §8, §9, §10, §14 before any task. Decision history: `/home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System/docs/superpowers/specs/2026-11-07-status-brainstorm-notas.md` (untracked, main checkout; the spec wins on conflict). **This plan's "Owner decisions" section overrides the spec's suggested phase cut (§14) and is final.**

## Global Constraints

- **Two PRs, stacked.** PR 1 (Tasks 1–13): branch `feat/status-effects-base`, worktree `System_X_System/.worktrees/status-effects`, cut from `origin/docs/status-effects-spec`; PR base `docs/status-effects-spec`. PR 2 (Tasks 14–16): branch `feat/status-effects-history` cut from `feat/status-effects-base` after PR 1 is open; PR base `feat/status-effects-base`. Never touch the main checkout (`System_X_System/`, another session's branch).
- **Language:** identifiers, file names and test titles (`TestXxx`, `t.Run("...")`) in **English**. Comments, `docs/dev/` and log prose in **Portuguese**. Wire values in English camelCase.
- **Wire format is camelCase** (CLAUDE.md invariant). New payloads follow `message.go` style (explicit json tags).
- **Out of scope (do NOT build):** front; anything that changes a roll (see "Deferred"); automatic resistance tests (§6); negative HP, critical, death (§7); damage over time and recovery per round/clock; asphyxia mechanics; Fatigue; a catalog REST endpoint (labels and alias suggestions are front copy, listed in the contract); hiding NPCs from the players' participants REST (the front needs it until its own phase); Push/Carry rename (§12).
- **Inside an open turn nothing is written before its close** (house rule, spec §10).
- **Comments/TODOs:** never delete one outside this scope; what this work makes contradictory is corrected (Task 1 rewrites the AGENTS.md rule to say this).
- **Gateway rules** (`gateway-conventions.instructions.md`): timestamps from Go, unconditional `Rollback` in defer, an index for every `WHERE`/`JOIN` column. New migration files only — never edit an applied one.
- **Room locking:** `r.mu` guards session and board state; the session has no lock of its own; `persistMu` → `r.mu` is the only allowed nesting; never hold `r.mu` across I/O or while calling `dispatchPerPlayer` (it takes `r.mu` itself).
- **Per-task verification:** `go vet ./... && go vet -tags=integration ./internal/gateway/pg/...`; `go test` of the touched packages; gateway tasks run their package's integration tests. Local DB: `set -a && . ./.env && set +a && export TEST_DATABASE_URL="postgres://$PG_DB_USER:$PG_DB_PASS@localhost:5432/hxh_rpg_test?sslmode=disable"`. Full integration suite only with `-p 1`.
- **Before each PR:** `golangci-lint cache clean && ~/go/bin/golangci-lint run --max-same-issues 0 --max-issues-per-linter 0`.
- **Do not poll CI.**

## Owner decisions (2026-11-17, in this session — final)

1. Damage per round starts at **1** for Bleeding, Burning, Freezing and Poisoned.
2. Recovery points of Stunned/Paralyzed/Stupefied: when the **master** applies by hand there is no failed test, so he types the points. (When the automatic test exists, the points come from the failure — later phase.)
3. **`npc_added` goes to the master only.** An NPC on the board reaches players through the fog, like any piece.
4. **Public statuses depend on the fog.** A player gets a character's statuses only while seeing it. The owner and the master always see. When the character leaves a player's sight the server sends nothing (the player keeps the last state seen); when it comes back, the player gets the current list. A match without a board has no fog: everyone sees. With a board, a character with no piece on it is not seen by others.
5. **Bars follow the same sight**: a player's `bars_updated`/`match_full_state.bars` list only the characters he owns or sees.
6. **Hidden takes the piece off the map**, the bars and the statuses of whoever is not in its viewer list (the ones who perceived it). Its actions while hidden and the history: PR 2.
7. **Unconscious brings Prone**; the master may remove Prone and keep only Unconscious.
8. **G1 (Tact, balance face):** impaired → attack, defense, dodge and movement get the status's own disadvantage/number; nullified → one disadvantage on those four. Declared now as pieces; applied in the rolls phase.
9. Everything that does not depend on another implementation is applied now (see "Blocking pieces" in Task 11).

## Decisions this plan makes (builder's call per §14; already reported to the owner)

1. Re-applying a status the character already has is refused (`status_effect_exists`); the master edits the existing one. Several characters: all-or-nothing.
2. Required on manual apply (no spec default): recovery points (Stunned/Paralyzed/Stupefied), Slowed's penalty, the sense level, Tact's faces, Injured's domains, Compelled's conduct. Defaults: Paralyzed = four domains; Weakened and impaired senses = disadvantage; damage per round = 1.
3. Viewer list storage: nullable JSONB `audience`. `NULL` = kind default; `{"everyone":true}`; `{"players":[…]}`; `{"players":[]}` = master only. The master always sees; the owner only if listed.
4. Kind default (no record): public kinds → everyone; hidden kinds → master + owner on a PC, master only on an NPC; `MatchRules.PlayerStatusEffectsPublic` (default `false`) opens a PC's hidden kinds. Sight (owner decision 4) applies on top, except for the owner and the master.
5. Containment inherits what a status **does**, not how it **ends** (Unconscious has no Tenacity recovery). Injured "disabled" also carries the "hurt" disadvantage (§5.2 diagram).
6. Players never see an effect's viewer list; no `updatedAt` on the wire.
7. Alias ≤ 60 runes; conduct 1–255 runes (trimmed).
8. Applying a status removes from the queue every pending action of that character the new set would refuse; each is recorded as cancelled by the master (`cancelled_actions`, same rule as `cancel_action`). An unopened reaction of a character who can no longer give it is replaced by the seated default (or by `nothing` when he cannot react at all); what was already opened stays.
9. Reaction ↔ piece mapping (for restrictions and locks): `default` = reflex dodge + default defense; `dodge` = reflex dodge; `closedDodge` = reflex dodge + evasion; `escape`/`escapeGuard` = escape; `closedEscape` = escape + evasion; `repel` = repel; `nothing` always allowed. Domains: dodge → reflex dodge, evasion; defense → default defense, repel; escape needs movement **and** dodge (§5.4). Action components: attack, feint → attack; defense → defense; dodge → dodge; move → movement; interact and skill-only actions are blocked only by "cannot act".
10. "Não usa habilidades" waits for the Nen migration (TODO).

## Deferred, with the reason (reported to the owner)

- **Everything that changes a roll** — test penalties, "defense only rolled", disadvantage on a reaction, auto-fail, Slowed, G1's application: §4 replaces how every advantage/disadvantage rolls (2 + N D10, keep 2), with open points the owner answered (passive with N levels: roll 2+N and compare with the base value as today; D20 set: one more D20 per level). Applying penalties on today's scheme would make the owner validate twice. Next phase.
- **Automatic tests (§6)** — the test shape (stack/chain) is being designed in the paused tests brainstorm.
- **Damage over time, recovery per round** — depend on negative HP and death (phase 2).

## Review Focus

1. **A character the player cannot see** gets a public status, then walks into sight → the player gets the list only when it enters sight; nothing while out of sight; nothing when it leaves. (Task 10 tests.)
2. **A turn close that fails after the master changed statuses inside it** → memory keeps the change; the next successful write carries it; nothing is written mid-turn. (Task 9.)
3. **Changing who sees a status** → only recipients whose projected list changed get `status_effects_updated`; nobody gets an empty "something changed". (Task 8.)
4. **Hidden applied/removed** → the piece leaves/returns for exactly the players outside the viewer list, the master and owner keep it, and a later move of the hidden piece reaches none of them. (Task 10.)
5. **Status applied while the character has queued actions or an unopened reaction** → refused-by-the-new-set ones leave and are recorded as cancelled; opened ones stay; the turn still closes. (Task 11.)

---

## File Structure

| File | Responsibility |
|---|---|
| `AGENTS.md` | Task 1 rule; Task 12 Known Issues |
| `internal/domain/match/character_status.go` | Task 1 correction note |
| `internal/domain/statuseffect/kind.go` | kinds and value types, parsing, default visibility |
| `internal/domain/statuseffect/effect.go` | `Effect`, `Params`, `ParamsInput`, `Identity`, `Clone`, `Write`, `Draft`, `Patch` |
| `internal/domain/statuseffect/normalize.go` | per-kind defaults and validation |
| `internal/domain/statuseffect/set.go` | `Apply` / `Edit` / `Remove` |
| `internal/domain/statuseffect/piece.go` | pieces (§5.1) and `Effect.Pieces()` |
| `internal/domain/statuseffect/summary.go` | `Summarize`: what a set forbids/warns (consumed by the session) |
| `internal/domain/statuseffect/audience.go` | `Audience`, `Viewer`, `Subject`, `Listed` |
| `internal/domain/statuseffect/error.go` | errors |
| `internal/domain/match/match_rules.go` | `PlayerStatusEffectsPublic` |
| `migrations/20261117000000_character_status_effects.sql` | table |
| `internal/gateway/pg/statuseffect/repository.go` | `ListBySheets`, `Write` |
| `internal/gateway/pg/round/persist_turn_close.go` | `TurnCloseData.StatusEffects` |
| `internal/domain/match/matchsession/status_effects.go` | storage, accessors, blocking checks |
| `internal/application/match/{init_match_session.go,status_effects.go}` | load at birth; the use case |
| `internal/app/game/message.go` | messages and payloads |
| `internal/app/game/status_effects.go` | WS arms, projection, write timing |
| `internal/app/game/sight.go` | `seesCharacterLocked`, Hidden in the fog gates, sight sync |
| `internal/app/game/{room.go,room_deps.go,turn_writes.go,master_board.go}` | wiring points |
| `cmd/game/main.go` | wiring |
| `docs/dev/api/match-status-effects.md`, `docs/dev/match/status-effects.md` | contract, design |

---

## PR 1

### Task 1: Rewrite the TODO rule and correct the persistence note

- [ ] **Step 1:** In `AGENTS.md`, replace `- **NEVER remove TODO comments** — intentional markers by the owner` with:

```markdown
- **Comments and TODOs are the owner's markers.** Never delete one outside the scope of your task —
  sessions used to wipe other people's TODOs. What your change makes contradictory (a comment that
  now describes the code wrongly) must be corrected or removed in the same change, saying why in
  the commit.
```

- [ ] **Step 2:** In `internal/domain/match/character_status.go`, the owner's block says "Kept verbatim… Do not trim it": do not delete anything. Insert after `// - considerar statusEngine, ou algo assim`:

```go
//
// [Correção, 2026-11 — spec de efeitos de status §10] Os efeitos de status (caído, atordoado,
// sangrando…) SÃO persistidos, numa tabela própria da ficha (character_status_effects,
// pacote statuseffect) e valem em qualquer campanha. O que continua só em memória é este
// CharacterStatus: barras de ação/movimento, ledger, postura e velocidade.
```

- [ ] **Step 3:** `go vet ./internal/domain/match/` → clean. **Commit** `docs: rewrite the TODO rule and correct the status persistence note`

---

### Task 2: Domain — kinds, params, effect and set operations

**Files:** Create `internal/domain/statuseffect/{kind.go,effect.go,normalize.go,set.go,error.go,audience.go}`; tests `normalize_test.go`, `set_test.go` (`package statuseffect_test`).

**Interfaces — Produces:**

```go
package statuseffect

type Kind string
const (
	KindParalyzed Kind = "paralyzed"; KindStupefied Kind = "stupefied"; KindStunned Kind = "stunned"
	KindUnconscious Kind = "unconscious"; KindRestrained Kind = "restrained"
	KindInjured Kind = "injured"; KindSlowed Kind = "slowed"
	KindSense Kind = "sense"; KindWeakened Kind = "weakened"
	KindBleeding Kind = "bleeding"; KindBurning Kind = "burning"; KindFreezing Kind = "freezing"; KindPoisoned Kind = "poisoned"
	KindAsphyxiating Kind = "asphyxiating"
	KindProne Kind = "prone"; KindAirborne Kind = "airborne"; KindOpenGuard Kind = "openGuard"
	KindCompelled Kind = "compelled"; KindControlled Kind = "controlled"; KindHidden Kind = "hidden"
)
// PT names for comments/docs: Paralisado, Atônito, Atordoado, Inconsciente, Preso, Lesionado, Lento,
// Sentido afetado, Debilitado, Sangrando, Queimando, Congelando, Envenenado, Asfixiando, Caído,
// No ar, Guarda aberta, Compelido, Controlado, Escondido.
type Sense string       // "vision" "hearing" "smell" "tact" "taste"; (Sense) Skill() enum.SkillName
type Aptitude string    // "physical" "mental" "spiritual"; (Aptitude) Ability() enum.AbilityName
type Domain string      // "attack" "defense" "movement" "dodge"; AllDomains() in this order
type SenseLevel string  // "impaired" "nullified"
type InjuryLevel string // "hurt" "disabled"
type TouchFace string   // "touch" "balance" "preemption"
func ParseKind(string) (Kind, error) // + ParseSense, ParseAptitude, ParseDomain, ParseSenseLevel, ParseInjuryLevel, ParseTouchFace
func (k Kind) PublicByDefault() bool

type Params struct {
	Domains []Domain; Injuries map[Domain]InjuryLevel; Level SenseLevel; Faces []TouchFace
	Points, HPPerRound, Penalty int; Disadvantage bool
	Conduct string; Irresistible bool; ControllerID *uuid.UUID
}
type ParamsInput struct {
	Domains []Domain; Injuries map[Domain]InjuryLevel; Level SenseLevel; Faces []TouchFace
	Points, HPPerRound, Penalty *int; Disadvantage *bool
	Conduct string; Irresistible *bool; ControllerID *uuid.UUID
}
type Identity struct{ Kind Kind; Variant string } // Variant = Sense | Aptitude | ""
type Audience struct{ Everyone bool; Players []uuid.UUID } // behavior in Task 4
type Effect struct {
	ID, CharacterID uuid.UUID; Kind Kind; Sense Sense; Aptitude Aptitude
	Params Params; Alias string; Audience *Audience // nil = kind default
	AppliedAt, UpdatedAt time.Time
}
func (e Effect) Identity() Identity
func (e Effect) Clone() Effect
type Write struct{ Delete bool; Effect Effect }
type Draft struct { Kind Kind; Sense Sense; Aptitude Aptitude; Params ParamsInput; Alias string; Audience *Audience }
type Patch struct { Params *ParamsInput; Alias *string; SetAudience bool; Audience *Audience }

func Normalize(k Kind, s Sense, a Aptitude, in ParamsInput) (Params, error)
func Apply(current []Effect, characterID uuid.UUID, d Draft, newID func() uuid.UUID, now time.Time) (next, added []Effect, err error)
func Edit(current []Effect, id uuid.UUID, p Patch, now time.Time) (next []Effect, edited Effect, err error)
func Remove(current []Effect, id uuid.UUID) (next []Effect, removed Effect, err error)
var ErrUnknownKind, ErrInvalidParams, ErrAlreadyApplied, ErrNotFound error
```

Per-kind table (anything not listed for a kind must be empty in the input → `fmt.Errorf("%w: <field>", ErrInvalidParams)`):

| Kind | Variant | Params (default) |
|---|---|---|
| paralyzed | — | domains (all four), points ≥1 required |
| stupefied, stunned | — | points ≥1 required |
| unconscious, restrained, asphyxiating, prone, airborne, openGuard, hidden | — | none |
| injured | — | injuries ≥1 entry |
| slowed | — | penalty ≥1 required |
| sense | sense required | level required; impaired: disadvantage (default true), penalty ≥0, at least one of them; nullified: neither; tact → faces ≥1 distinct required; other senses → no faces |
| weakened | aptitude required | disadvantage (default true), penalty ≥0, at least one |
| bleeding, burning, freezing, poisoned | — | hpPerRound ≥1 (default 1) |
| compelled | — | conduct 1–255 runes (trimmed) required; irresistible (default false) |
| controlled | — | controllerId optional |

`PublicByDefault()` = prone, airborne, restrained, burning, unconscious, stunned, asphyxiating, openGuard (§9). Sort `Domains`/`Faces` in declaration order.

- [ ] **Step 1: `normalize_test.go`** — table `TestNormalize`, one row per table line, including: `"paralyzed defaults to the four domains"`, `"paralyzed requires points"` (+stupefied, stunned), `"damage per round defaults to one"` (bleeding, burning, freezing, poisoned — each row), `"sense requires a level"`, `"impaired sense defaults to disadvantage"`, `"impaired sense with only a penalty"`, `"impaired sense needs disadvantage or penalty"`, `"nullified sense refuses disadvantage and penalty"`, `"tact requires at least one face"`, `"faces only on tact"`, `"duplicate faces refused"`, `"weakened defaults to disadvantage"`, `"weakened needs disadvantage or penalty"`, `"weakened requires an aptitude"`, `"aptitude only on weakened"`, `"sense only on sense kind"`, `"injured requires at least one domain"`, `"injured keeps per-domain levels"`, `"slowed requires a penalty"`, `"compelled requires a conduct"`, `"conduct is trimmed and capped at 255 runes"`, `"controlled with and without controller"`, `"param foreign to the kind is refused"`, `"zero or negative numbers refused"`.
- [ ] **Step 2:** `go test ./internal/domain/statuseffect/` → FAIL. **Step 3:** implement kind/effect/normalize/error (+ empty `Audience` type). **Step 4:** PASS.
- [ ] **Step 5: `set_test.go`** (fixed `now`, counter `newID`): `TestApply` — `"adds a normalized effect"`, `"refuses the same identity"` (`ErrAlreadyApplied`, nil next), `"different senses coexist"`, `"same sense refused"`, `"one weakened per aptitude"`, `"unconscious brings prone"` (added = [unconscious, prone]), `"unconscious keeps an existing prone"`, `"alias over 60 runes refused"`, `"alias is trimmed"`, `"does not mutate the input slice"`. `TestEdit` — `"replaces params whole"` (UpdatedAt moves, AppliedAt stays), `"nil params keep params"`, `"empty alias clears"`, `"set audience nil returns to default"`, `"invalid params leave the set untouched"`, `"unknown id"`. `TestRemove` — `"removes by id"`, `"removing prone keeps unconscious"` (owner decision 7), `"removing unconscious keeps prone"`, `"unknown id"`.
- [ ] **Step 6:** FAIL → **Step 7:** implement `set.go` (fresh cloned slices; Edit re-normalizes with the effect's identity; Prone added by Apply carries default params, no alias, default audience) → **Step 8:** PASS, `go vet ./...`.
- [ ] **Step 9: Commit** `feat(statuseffect): catalog kinds, params and set operations`

---

### Task 3: Domain — pieces, composition and the blocking summary

**Files:** Create `piece.go`, `summary.go`; tests `piece_test.go`, `summary_test.go`.

**Interfaces — Produces:**

```go
type PieceKind string // noAct noReact domainLocked domainWarning reactionRestricted testPenalty autoFail
                      // noDisplacement losesHp recovery triggeredTest conduct control counter
type Reaction string    // "reflexDodge" "evasion" "defense" "repel" "escape"
type Restriction string // "forbidden" "disadvantage" "rolledOnly"
type Trigger string     // "onHit" "onMove"
type Scope struct { Domain Domain; Aptitude Aptitude; Sense Sense; Face TouchFace; MoveSpeed bool } // one set
type TestRef struct { Skill enum.SkillName; Attribute enum.AttributeName }                    // one set
type Piece struct {
	Kind PieceKind; Domain Domain; Reaction Reaction; Restriction Restriction
	Scope Scope; Disadvantage, Penalty int; HPPerRound int; Test TestRef; Points int
	Triggers []Trigger; OnFail Kind; Conduct string; Irresistible bool; ControllerID *uuid.UUID
}
func (e Effect) Pieces() []Piece

type Note struct { Kind Kind; Domain Domain; Conduct string; Irresistible bool } // master-only notes
type Summary struct {
	CannotAct, CannotReact, NoDisplacement bool
	CannotActBy, CannotReactBy, NoDisplacementBy []Kind // which statuses cause it (error messages)
	LockedDomains map[Domain][]Kind
	ForbiddenReactions map[Reaction][]Kind
	Notes []Note // domainWarning → {Kind, Domain}; conduct → {Kind, Conduct, Irresistible}
}
func Summarize(effects []Effect) Summary // reads ONLY these pieces; penalties/autoFail/etc. are ignored here
```

Composition (§5.1–§5.4; containment built in code — unconscious = stunned's builder minus recovery; stunned = paralyzed(all four) + stupefied, minus their recoveries, + autoFail for each of the five senses):

- paralyzed{domains, points}: domainLocked per domain; noDisplacement if movement marked; recovery{Tenacity, points}.
- stupefied{points}: noAct, noReact, recovery{Attribute Resilience, points}.
- stunned{points}: as above + recovery{Tenacity, points}. unconscious: stunned without recovery.
- restrained: noDisplacement; reactionRestricted{escape, forbidden}.
- injured: hurt d → testPenalty{Domain d, Disadvantage 1}; disabled d → same + domainWarning d.
- slowed{penalty}: testPenalty{MoveSpeed, Penalty}.
- sense impaired: testPenalty{Sense s} (tact: one per face) with the status's disadvantage/penalty. nullified: autoFail{Sense s} (tact: per face). Vision nullified also testPenalty{attack, Disadvantage 1}, testPenalty{defense, Disadvantage 1}. Tact with balance face: triggeredTest{[onHit,onMove], Skill Tact, OnFail prone}; **G1:** impaired → testPenalty on attack, defense, dodge, movement with the status's own disadvantage/penalty; nullified → testPenalty{Disadvantage 1} on those four.
- weakened{aptitude}: testPenalty{Aptitude a} with its disadvantage/penalty.
- bleeding/burning/freezing/poisoned: losesHp{n}. asphyxiating: counter.
- prone: reactionRestricted{reflexDodge, disadvantage}, {repel, disadvantage}. airborne: {reflexDodge, forbidden}, {escape, forbidden}. openGuard: {reflexDodge, forbidden}, {repel, forbidden}, {defense, rolledOnly}, {escape, disadvantage}.
- compelled: conduct{text, irresistible}. controlled: control{controller}. hidden: none.

Markers (Portuguese comments, no invented piece): stupefied/stunned `// TODO Nen: "não usa habilidades" (spec §13).`; prone/airborne `// Provisório até o movimento (spec §13): só rolar/deslizar (Caído), só a trajetória (No ar).`; asphyxiating `// Fôlego e afogamento: fase posterior (spec §5.3 Ar).`

- [ ] **Step 1: `piece_test.go`** — `TestPieces_Composition` (one row per bullet above, order-insensitive compare; include: paralyzed [attack] has no noDisplacement; hearing nullified has no domain penalty; tact [touch] has no triggeredTest; G1 impaired and nullified rows) and `TestPieces_Containment` (paralyzed(all) − recovery ⊆ stunned; stupefied − recovery ⊆ stunned; stunned − recovery ⊆ unconscious; restrained's noDisplacement ⊆ paralyzed{movement}).
- [ ] **Step 2: `summary_test.go`** — `TestSummarize`: empty → zero Summary; stunned → CannotAct, CannotReact, NoDisplacement, all four LockedDomains, `CannotActBy == [stunned]`; paralyzed{[attack]} + restrained → LockedDomains{attack:[paralyzed]}, NoDisplacement by [restrained], ForbiddenReactions{escape:[restrained]}; openGuard + airborne → ForbiddenReactions reflexDodge:[airborne, openGuard], repel:[openGuard], escape:[airborne] (disadvantage/rolledOnly NOT in Summary); injured{movement: disabled} → Notes [{injured, movement}]; compelled → Notes [{compelled, conduct, irresistible}].
- [ ] **Step 3:** FAIL → implement → PASS; `go vet ./...`. **Commit** `feat(statuseffect): declare effect pieces, composition and blocking summary`

---

### Task 4: Domain — viewer list and match rule flag

**Files:** `audience.go`, `set.go` (validate), `error.go`; `internal/domain/match/match_rules.go` (+test); test `audience_test.go`.

**Interfaces — Produces:**

```go
type Viewer struct{ UserID uuid.UUID; IsMaster bool }
type Subject struct{ OwnerID uuid.UUID; IsNPC bool }
func (a Audience) Validate() error // ErrInvalidAudience: everyone+players, duplicates, uuid.Nil
// Listed: does the viewer list (or the kind default) include this viewer? Sight is NOT here — the room adds it.
func Listed(e Effect, v Viewer, s Subject, pcPublicByDefault bool) bool
var ErrInvalidAudience error
```
`MatchRules.PlayerStatusEffectsPublic bool`, default `false` (PT doc comment: configuração da mesa, spec §9/§14, sem tela).

- [ ] **Step 1: tests** — `TestListed`: master always (incl. `{Players: []}`); default public kind → owner and other; default hidden kind on PC → owner only; on NPC → nobody but the master; flag on → other sees PC's hidden kinds, not an NPC's; `{Everyone:true}` → other; `{Players: []}` → not the owner; `{Players:[other]}` → other, not owner; `{Players:[owner]}` on a public kind → other does not. `TestAudienceValidate`. In `set_test.go`: `"invalid audience refused"` for Apply and Edit. `match_rules_test.go`: default false.
- [ ] **Step 2:** FAIL → implement → PASS. **Commit** `feat(statuseffect): viewer list with kind defaults`

---

### Task 5: Migration and gateway

**Files:** `migrations/20261117000000_character_status_effects.sql`; `internal/gateway/pg/statuseffect/repository.go` (`pgstatuseffect`); `statuseffect_integration_test.go`; `internal/application/match/i_repository.go`.

**Interfaces — Produces:**

```go
type IStatusEffectReader interface { // application/match
	ListBySheets(ctx context.Context, sheetUUIDs []uuid.UUID) (map[uuid.UUID][]statuseffect.Effect, error)
}
func NewRepository(q pgfs.IQuerier) *Repository
func (r *Repository) ListBySheets(ctx, sheetUUIDs) (map[uuid.UUID][]statuseffect.Effect, error) // never nil; ordered applied_at, uuid
func (r *Repository) Write(ctx context.Context, ops []statuseffect.Write) error                 // in order, atomic
```

- [ ] **Step 1: migration** (goose layout of `20261010000000_cancelled_actions.sql`, PT comment: tabela da ficha, vale em qualquer campanha (§10); `variant` = sentido ou Aptidão, parte da identidade; `audience NULL` = padrão do tipo (§9)):

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

- [ ] **Step 2: integration tests** — `TestWrite`: `"upsert inserts then updates"` (AppliedAt unchanged, audience back to nil), `"delete removes and is idempotent"`, `"ops apply in order"` ([upsert A prone, delete A, upsert B prone] → only B), `"atomic"` (second op on a missing sheet → error, nothing written), `"runs inside an outer transaction"` (rollback leaves nothing). `TestListBySheets`: `"groups by sheet and orders"`, `"empty input"`, `"round-trips every params field"`.
- [ ] **Step 3:** FAIL → **Step 4:** implement. `Write` begins on `r.q` (pool → transaction; `pgx.Tx` → savepoint). Upsert `ON CONFLICT (uuid) DO UPDATE SET params, alias, audience, updated_at`. JSON via gateway-local row structs with camelCase tags (`domains, injuries, level, faces, points, hpPerRound, penalty, disadvantage, conduct, irresistible, controllerId`; audience `{"everyone":true}` / `{"players":[…]}`). → PASS; vet with tags.
- [ ] **Step 5: Commit** `feat(statuseffect): character_status_effects table and repository`

---

### Task 6: Status writes in the turn's transaction

**Files:** `internal/application/match/i_repository.go`; `internal/gateway/pg/round/persist_turn_close.go`; `round_integration_test.go`.

**Produces:** `TurnCloseData.StatusEffects []statuseffect.Write` (PT doc: o que o mestre mudou com o turno aberto, precedido do que uma gravação anterior não gravou; em ordem, nesta transação, §10).

- [ ] **Step 1:** tests in the existing `PersistTurnClose` test: `"writes status effects in the turn transaction"`; `"a failing close writes no status effect"`.
- [ ] **Step 2:** FAIL → **Step 3:** after the cancelled-actions block: `pgstatuseffect.NewRepository(tx).Write(ctx, d.StatusEffects)` when non-empty, error wrapped `PersistTurnClose status effects: %w` → PASS.
- [ ] **Step 4: Commit** `feat(round): write status effect changes with the turn close`

---

### Task 7: Session storage, loading at birth, and the use case

**Files:** Create `internal/domain/match/matchsession/status_effects.go` (+test); modify `internal/application/match/init_match_session.go` (+test); create `internal/application/match/status_effects.go` (+test); `error.go` (`ErrUnknownCharacter` unless an equivalent exists).

**Interfaces — Produces:**

```go
// matchsession (caller holds r.mu)
func (s *MatchSession) SetStatusEffects(byChar map[uuid.UUID][]statuseffect.Effect)
func (s *MatchSession) StatusEffectsOf(charID uuid.UUID) []statuseffect.Effect // clones, ordered; never nil
func (s *MatchSession) ReplaceStatusEffects(charID uuid.UUID, effects []statuseffect.Effect) // empty ⇒ delete
func (s *MatchSession) FindStatusEffect(id uuid.UUID) (statuseffect.Effect, bool)
func (s *MatchSession) CharactersWithStatusEffects() []uuid.UUID
func (s *MatchSession) StatusSummaryOf(charID uuid.UUID) statuseffect.Summary
// (+ IsPlayerParticipant(userID) bool unless an accessor already answers it)

func (uc *InitMatchSessionUC) WithStatusEffects(r IStatusEffectReader) *InitMatchSessionUC // nil = none

type IStatusEffects interface {
	Apply(s *matchsession.MatchSession, masterUUID, callerUUID uuid.UUID, in ApplyStatusEffectInput, now time.Time) (StatusEffectChange, error)
	Edit(s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID, p statuseffect.Patch, now time.Time) (StatusEffectChange, error)
	Remove(s *matchsession.MatchSession, masterUUID, callerUUID, id uuid.UUID) (StatusEffectChange, error)
}
type ApplyStatusEffectInput struct{ CharacterIDs []uuid.UUID; Draft statuseffect.Draft }
type StatusEffectChange struct {
	Characters []uuid.UUID          // whose set changed
	Writes     []statuseffect.Write // in order
	HiddenChanged bool              // a hidden-kind effect was added, removed or had its audience changed (Task 10 reads it)
}
func NewStatusEffectsUC() *StatusEffectsUC
```

- [ ] **Step 1: session tests** — clones on read; `ReplaceStatusEffects(id, nil)` deletes; `FindStatusEffect` across characters; `StatusSummaryOf` = `Summarize(StatusEffectsOf)`.
- [ ] **Step 2: init tests** — fake reader: effects land in the session; reader receives exactly the participants' sheet UUIDs; reader error → `Init` returns it (same policy as the clock); no reader → none.
- [ ] **Step 3: UC tests** `TestStatusEffectsUC`: `"applies to several characters"` (writes = upserts, unconscious → 2 per character); `"all or nothing when one already has it"`; `"refuses a non-master"` (`ErrNotMatchMaster`); `"refuses an ended match"` (`matchsession.ErrMatchEnded`); `"refuses an unknown character"`; `"refuses empty or repeated character ids"`; `"refuses a controller outside the match or equal to the target"`; `"refuses an audience player who is not a player of the match"` (`ErrInvalidAudience`); `"edit returns an upsert"`; `"remove returns a delete"`; `"unknown id"`; `"hidden changes are flagged"` (apply/remove hidden, edit hidden's audience → `HiddenChanged`; edit hidden's alias → false).
- [ ] **Step 4:** FAIL → **Step 5:** implement (compute every character's next set first, then replace all) → PASS; vet.
- [ ] **Step 6: Commit** `feat(match): status effects in the session, loaded at birth, changed by the master`

---

### Task 8: WS commands, per-recipient projection, reconnect

**Files:** `internal/app/game/message.go`, `room.go` (three `case` arms; `buildMatchFullState`), `room_deps.go`; create `status_effects.go`, `sight.go`; test `status_effects_e2e_test.go` (on `combatFixture`: `connectTable`, `withNPCTarget`, `withBystander`, `openAttackTurn`, `seedMoverBoard`/`seedHiddenMoveBoard` for positions and walls, `collector`).

**Interfaces — Produces (message.go):**

```go
MsgTypeApplyStatusEffect    MessageType = "apply_status_effect"    // c→s, master
MsgTypeEditStatusEffect     MessageType = "edit_status_effect"
MsgTypeRemoveStatusEffect   MessageType = "remove_status_effect"
MsgTypeStatusEffectsUpdated MessageType = "status_effects_updated" // s→c, projected

type StatusEffectParamsPayload struct {
	Domains []string `json:"domains,omitempty"`; Injuries map[string]string `json:"injuries,omitempty"`
	Level string `json:"level,omitempty"`; Faces []string `json:"faces,omitempty"`
	Points *int `json:"points,omitempty"`; HPPerRound *int `json:"hpPerRound,omitempty"`
	Penalty *int `json:"penalty,omitempty"`; Disadvantage *bool `json:"disadvantage,omitempty"`
	Conduct string `json:"conduct,omitempty"`; Irresistible *bool `json:"irresistible,omitempty"`
	ControllerID *uuid.UUID `json:"controllerId,omitempty"`
}
type StatusEffectVisibilityPayload struct {
	Mode string `json:"mode"`                        // "default" | "public" | "players"
	PlayerIDs []uuid.UUID `json:"playerIds,omitempty"` // mode "players"; [] = master only
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
	Visibility *StatusEffectVisibilityPayload `json:"visibility,omitempty"` // absent = untouched; "default" = back to default
}
type RemoveStatusEffectPayload struct{ StatusEffectID uuid.UUID `json:"statusEffectId"` }
type StatusEffectPayload struct {
	ID uuid.UUID `json:"id"`; Kind string `json:"kind"`
	Sense string `json:"sense,omitempty"`; Aptitude string `json:"aptitude,omitempty"`
	Params StatusEffectParamsPayload `json:"params"` // fields the kind owns, defaults applied
	Alias string `json:"alias,omitempty"`; AppliedAt time.Time `json:"appliedAt"`
	Visibility *StatusEffectVisibilityPayload `json:"visibility,omitempty"` // master only
	Public *bool `json:"public,omitempty"`                               // master only: listed for every player
}
type StatusEffectsUpdatedPayload struct {
	CharacterID uuid.UUID `json:"characterId"`
	StatusEffects []StatusEffectPayload `json:"statusEffects"` // the recipient's WHOLE visible list; never null
}
// MatchFullStatePayload gains:
StatusEffects map[uuid.UUID][]StatusEffectPayload `json:"statusEffects,omitempty"`
```

```go
// sight.go — the ONE answer to "does this player see this character now" (Task 10 adds Hidden)
func (r *Room) seesCharacterLocked(pid uuid.UUID, charID uuid.UUID) bool
// true when: pid owns charID (charToPlayer); OR the room has no board (!hasBoardLocked()); OR the
// character's piece exists, its Visible flag is not false, and its center is inside GetVisibility(pid).
// status_effects.go
func (r *Room) projectStatusEffectsLocked(charID uuid.UUID, pid uuid.UUID, isMaster bool) (list []StatusEffectPayload, known bool)
// known=false when a non-master, non-owner does not see the character → "send nothing, keep nothing".
```
Room field `statusViews map[uuid.UUID]map[uuid.UUID][]byte` (player → character → JSON of the last list sent; guarded by `r.mu`). `RoomDeps.StatusEffectsUC appmatch.IStatusEffects` (nil ⇒ `game_error`).

Error codes: `forbidden`, `invalid_payload` (bad JSON, unknown enum string, empty/duplicate `characterIds`, nil id), `match_not_started`, `match_already_ended`, `unknown_character`, `status_effect_invalid` (`ErrUnknownKind`/`ErrInvalidParams`/`ErrInvalidAudience`; message names the field), `status_effect_exists`, `status_effect_not_found`, `game_error`.

- [ ] **Step 1: e2e tests** (PC owned by the fixture player, a bystander player, an NPC; a board where everyone sees everyone unless a test moves pieces apart behind a wall):
  - `TestStatusEffects_ApplyPublic`: prone on the victim → master, owner, bystander each get one `status_effects_updated`; only the master's has `visibility{mode:"default"}`, `public:true`.
  - `TestStatusEffects_ApplyHiddenOnPC` / `OnNPC`: poisoned → master + owner / master only; bystander gets **no message**.
  - `TestStatusEffects_VisibilityChange`: hidden → `public` (bystander gets it, owner nothing, master gets it) → `players:[]` (owner and bystander get `[]`) → alias only (master only).
  - `TestStatusEffects_PublicNeedsSight`: NPC behind a wall from the bystander → prone on the NPC → bystander gets nothing; master and (NPC owner = master) get it.
  - `TestStatusEffects_NoBoardEveryoneSees`: room without board → bystander gets the public one.
  - `TestStatusEffects_Remove`: everyone who saw it gets `[]`.
  - `TestStatusEffects_MultiTargetAllOrNothing`; `TestStatusEffects_Refusals` (player → `forbidden`; unknown kind / prone+points → `status_effect_invalid`; unknown character; edit unknown id; `inLobby` → `match_not_started`).
  - `TestStatusEffects_Reconnect`: public + hidden on the NPC, hidden on the PC; bystander reconnects in sight → only the NPC's public; out of sight → none; owner → NPC public (if in sight) + own hidden; master → all, with `visibility`.
- [ ] **Step 2:** FAIL → **Step 3: implement** `status_effects.go` + `sight.go`:
  - Arms: master check, decode, wire → domain (`mode "default"` ⇒ nil audience), `session == nil` ⇒ `match_not_started`.
  - Under `r.mu.Lock()`: UC call with `r.now()`; on error unlock + `error`; on success `r.holdStatusEffectWritesLocked(change.Writes)` (stub returning false in this task — Task 9 fills it); unlock; then `r.syncStatusViews(change.Characters)`.
  - `syncStatusViews(chars []uuid.UUID)` (no lock held by caller): for each connected client (same set `dispatchPerPlayer` iterates) and each char, under `r.mu` build `projectStatusEffectsLocked`; `known=false` ⇒ skip; JSON ≠ `statusViews[pid][char]` ⇒ store and queue; then send via `dispatchPerPlayer` (one message per changed (pid, char)). The master's lists go through the same cache.
  - `buildMatchFullState`: `StatusEffects` for the viewer from `CharactersWithStatusEffects()` (known ones only) and **reset** `statusViews[pid]` to exactly what was sent.
  - `removeClient`: drop `statusViews[pid]`.
- [ ] **Step 4:** PASS; whole package `go test ./internal/app/game/`; vet.
- [ ] **Step 5: Commit** `feat(game): master applies, edits and removes status effects; projected by list and sight`

---

### Task 9: When the write happens — outside a turn, with the turn, healing; live NPC; wiring

**Files:** `status_effects.go`, `turn_writes.go` (`statusEffects []statuseffect.Write`), `room.go` (`persistClosedTurn`; field `unwrittenStatusEffects`), `room_deps.go`, `master_board.go` (live NPC), `cmd/game/main.go`; tests in `status_effects_e2e_test.go` with `fakeStatusEffectStore` (records each `Write`, fails on demand, serves `ListBySheets`).

**Produces:**

```go
type IStatusEffectRepo interface {
	ListBySheets(ctx context.Context, sheetUUIDs []uuid.UUID) (map[uuid.UUID][]statuseffect.Effect, error)
	Write(ctx context.Context, ops []statuseffect.Write) error
}
RoomDeps.StatusEffectRepo IStatusEffectRepo // nil: nothing written, nothing accumulated
func (r *Room) holdStatusEffectWritesLocked(ops []statuseffect.Write) (flushNow bool)
func (r *Room) flushStatusEffects() // caller holds no lock
```

- [ ] **Step 1: tests** — `"outside a turn writes at once"`; `"inside a turn waits for the close"` (`TurnCloseData.StatusEffects` = the ops in order; store untouched); `"a failed out-of-turn write is retried with the next one"`; `"a failed close keeps the turn's status writes for the next write"`; `"unwritten writes ride the next close first"` (and are not resent after); `"live NPC arrives with its effects"`; `"no repo, no accumulation"`.
- [ ] **Step 2:** FAIL → **Step 3: implement:**
  - hold: open turn ⇒ `turnWritesLocked(id).statusEffects`; else repo nil ⇒ drop; else append to `unwrittenStatusEffects`, return true (arm calls `flushStatusEffects` after unlocking).
  - flush: `persistMu.Lock()`; copy pending under `r.mu`; `Write` outside it; success ⇒ drop that prefix under `r.mu`; failure ⇒ PT log with how many stay pending.
  - `persistClosedTurn`: in its critical section `pending := clone(unwrittenStatusEffects)`; pass `append(pending, inTurn.statusEffects...)`; success ⇒ drop `len(pending)`; failure ⇒ append `inTurn.statusEffects` to `unwrittenStatusEffects`; mention status effects in the failure log.
  - live NPC (`enrollLiveNPC`): after enrolment, `ListBySheets([npc])` with no lock; under `r.mu` `ReplaceStatusEffects`; then `syncStatusViews([npc])`. Read error ⇒ PT log, NPC plays without statuses.
  - `cmd/game/main.go`: `statusEffectRepository := pgstatuseffect.NewRepository(pool)`; `initSessionUC.WithStatusEffects(statusEffectRepository)`; `RoomDeps{… StatusEffectsUC: match.NewStatusEffectsUC(), StatusEffectRepo: statusEffectRepository}`.
- [ ] **Step 4:** `go test ./internal/app/game/ -race` PASS; `go build ./... && go vet ./... && go vet -tags=integration ./...`.
- [ ] **Step 5: Commit** `feat(game): status effect writes follow the turn rule and heal after a failure`

---

### Task 10: Sight — Hidden in the fog, bars per player, `npc_added` to the master, sync on sight change

Owner decisions 3–6.

**Files:** `sight.go`, `room.go` (fog gates, `broadcastBars`, `buildMatchFullState` bars), `master_board.go` (`npc_added`), `master_action_record.go` (views), `internal/domain/match/matchsession/bars_view.go`; tests `sight_e2e_test.go` + adjust existing tests that asserted `npc_added` reaching players (they now assert it does not).

**Produces:**

```go
// sight.go
func (r *Room) hiddenFromLocked(pid uuid.UUID, charID uuid.UUID) bool // charID has a Hidden effect not Listed for pid (master/owner never)
func (r *Room) syncSight()                                           // caller holds no lock: broadcastBars + syncStatusViews(all characters with effects)
// matchsession
func (s *MatchSession) BarsViewFor(isMaster bool, sees func(charID uuid.UUID) bool) BarsView // BarsView(isMaster) = BarsViewFor(isMaster, all)
```

- [ ] **Step 1: tests** (`sight_e2e_test.go`):
  - `TestSight_HiddenRemovesPiece`: NPC in everyone's sight; master applies hidden to the NPC with `visibility{players:[owner]}` → bystander gets `piece_removed` for the NPC piece, owner nothing; NPC moves → bystander gets nothing, owner gets `piece_moved`; bystander reconnects → `map_full_state` has no NPC piece, `match_full_state.bars` has no NPC; master removes hidden → bystander gets the piece back (`piece_moved` with its slot) and its bars.
  - `TestSight_StatusOnEnteringSight` (Review Focus 1): NPC behind a wall from the bystander with a public prone → nothing; the NPC moves into sight → bystander gets `piece_moved` then `status_effects_updated` with prone; moves out → `piece_removed`, no status message; master edits prone's alias while out of sight → nothing; back in sight → the new list.
  - `TestSight_BarsFollowSight`: bystander's `bars_updated` omits the NPC while out of sight and includes it in sight; master always has it; a PC owned by the bystander always included.
  - `TestSight_NpcAddedMasterOnly`: `add_npc` → only the master gets `npc_added`.
- [ ] **Step 2:** FAIL → **Step 3: implement:**
  - `seesCharacterLocked` gains `&& !hiddenFromLocked(pid, charID)`.
  - Route every per-player piece gate through it: `gatePieceMoveLocked`, `buildMapFullState` (the `pieceProj` handed to the filters marks a hidden-from-pid piece `Visible:false`), `relayPieceMove`/`relayPieceRemoved`, `gateEscapeLandingsLocked`, the views in `master_action_record.go`, and `buildPieceView` (the watched piece's owner as pid). Find any other reader of `PieceMovedPayload.Visible` per player with `grep -n "\.Visible" internal/app/game/*.go` and route it too.
  - When `StatusEffectChange.HiddenChanged`, after the write hold and before `syncStatusViews`: for each connected non-master player, compare "sees the piece" before/after; lost ⇒ `piece_removed`; gained ⇒ `piece_moved` with the current slot (server message). Then `syncSight()`.
  - `broadcastBars`: per player `BarsViewFor(false, func(c) bool { return seesCharacterLocked(pid, c) })`; the dedupe cache becomes per player (`map[uuid.UUID][]byte`, same "stale" rule); `match_full_state.bars` uses the same per-player view and records it in that player's cache.
  - Call `syncSight()` after: `relayPieceMove`, `relayPieceRemoved`, `pushVisibilityUpdates`, the opened move (`announceOpenedTurn` path) and closed escapes, wall interact/reveal, live NPC enrolment, and status changes.
  - `npc_added`: master only (`sendToMaster`). Update its comment and `message.go`'s comment ("mesa inteira" → "só o mestre").
- [ ] **Step 4:** whole `go test ./internal/app/game/ -race` PASS (fix tests that relied on the old broadcast deliberately, explaining each in the commit); vet.
- [ ] **Step 5: Commit** `feat(game): Hidden leaves the map; bars and statuses follow sight; npc_added to the master`

---

### Task 11: Blocking pieces in the engine

Decisions 8–9. Uses `StatusSummaryOf` (Task 7).

**Files:** `internal/domain/match/matchsession/status_effects.go` (checks), `match_session.go` (`EnqueueAction`, `AttachReaction`, `seatDefaultReactions`/`newDefaultReaction`), `internal/app/game/status_effects.go` (queue cleanup after apply), `message.go` + `room.go` (`ActionQueuedPayload.StatusNotes`); tests `matchsession/status_block_test.go`, `internal/app/game/status_block_e2e_test.go`.

**Produces:**

```go
// matchsession
var ErrBlockedByStatus error // wraps: "<character> não pode <agir|reagir|usar o domínio X|se deslocar|…>: <kinds>"
func (s *MatchSession) StatusRefusal(a *action.Action) error   // nil = allowed; checks actor's Summary against the action (or reaction) shape
func (s *MatchSession) DropRefusedByStatus(charID uuid.UUID) (removed []*action.Action) // queue + unopened reactions; caller records them
// message.go
type StatusNotePayload struct {
	Kind string `json:"kind"` // the status kind
	Domain string `json:"domain,omitempty"`
	Conduct string `json:"conduct,omitempty"`; Irresistible bool `json:"irresistible,omitempty"`
}
ActionQueuedPayload.StatusNotes []StatusNotePayload `json:"statusNotes,omitempty"` // master-only surface already
```
Wire error code for a refusal: `blocked_by_status` (message = the PT text).

Rules (`StatusRefusal`):
- CannotAct ⇒ any non-reaction action refused.
- CannotReact ⇒ any reaction refused except `nothing`.
- A component in a locked domain (decision 9 mapping) ⇒ refused. Escape kinds need movement and dodge free.
- NoDisplacement ⇒ a move component or an escape kind refused.
- A reaction kind using a forbidden reaction (decision 9) ⇒ refused.
- `seatDefaultReactions`: CannotReact ⇒ seat a `nothing` reaction; reflex dodge forbidden or dodge locked ⇒ the default carries no reflex dodge; default defense locked ⇒ no default defense; both gone ⇒ `nothing`. **First check** (read the resolver) that a `ReactDefault` with only one of its two passives resolves; if it does not, make the resolver accept it, with tests.
- Notes: domain warnings for the action's domains (Injured disabled) and every conduct note go on `action_queued.statusNotes` and on `match_full_state.queue` entries.

- [ ] **Step 1: session tests** `TestStatusRefusal` (stunned refuses an attack and a reaction, allows `nothing`; paralyzed{attack} refuses attack, allows a move; restrained refuses a move and an escape, allows a dodge; openGuard refuses `dodge`, `closedDodge`, `repel`, allows `escape`; airborne refuses `escape`, `closedEscape`; paralyzed{dodge} refuses `escape` (needs dodge)); `TestDropRefusedByStatus` (queued attack and move of a character who becomes stunned leave; another character's stay; an unopened reaction is replaced by the seated default/nothing; an opened one stays); `TestSeatDefaultReactionUnderStatus` (openGuard → defense only; stunned → nothing; paralyzed{defense} → reflex only; the turn resolves and closes in each case).
- [ ] **Step 2: e2e tests** — `TestStatusBlock_EnqueueRefused` (`blocked_by_status`); `TestStatusBlock_ApplyDropsQueue` (Review Focus 5: removed actions recorded as cancelled — in `CancelledActionRepo` outside a turn, in `TurnCloseData.CancelledActions` inside one — `cancelledBy` = master; the table gets the same `action_cancelled` message `cancel_action` sends); `TestStatusBlock_NotesToMaster` (injured{attack: disabled} + compelled → master's `action_queued.statusNotes` has both; player's own messages carry none); `TestStatusBlock_RemoveUnblocks`.
- [ ] **Step 3:** FAIL → **Step 4: implement**; queue cleanup runs in the apply/edit arm under `r.mu` right after the UC, recording each removed action through the same path `cancel_action` uses (`holdCancelledActionLocked` / `writeCancelledAction`) and sending the same messages, then `broadcastBars` → **Step 5:** PASS; whole package `-race`; vet.
- [ ] **Step 6: Commit** `feat(match): statuses block acting, reacting, locked domains and displacement`

---

### Task 12: Contract and docs (PR 1)

**Files:** create `docs/dev/api/match-status-effects.md`, `docs/dev/match/status-effects.md`; modify `docs/dev/api/match-combat-ws.md` (§3 index link; `match_full_state` gains `statusEffects`, bars per player; `npc_added` → master only; `action_queued.statusNotes`; `blocked_by_status`), `docs/documentation-map.yaml`, `AGENTS.md` (Known Issues paragraph "Efeitos de status — fase 1"), `.github/instructions/domain-map.instructions.md`, `.github/instructions/game-server.instructions.md` (sight + status write timing).

- [ ] **Step 1: contract** (PT, structure of `match-clock.md`): commands, `status_effects_updated`, examples from the Task 8/10/11 tests; the per-kind params table with wire names, variant and defaults; enum values; visibility modes and default rule; sight (owner decision 4) and "nothing on leaving sight"; Hidden; `match_full_state` fields; error codes; inside a turn: live now, durable at the close. Front material: PT display names (§5.3; "Corpo/Mente/Espírito debilitado"; "Visão prejudicada/anulada") and §8's alias suggestions verbatim. Plainly list what is NOT there yet ("Deferred").
- [ ] **Step 2: design doc** (PT): package map; pieces, containment in code, `Summary`; viewer list + sight; write timing and healing; G1 recorded; limitation: a sheet live in two rooms keeps two memories (last write wins on disk).
- [ ] **Step 3:** index/map updates. **Commit** `docs: status effects phase 1 contract and design`

---

### Task 13: Prove PR 1 for real, open it, hand off

- [ ] **Step 1:** `go test ./... -race`; `go test -tags=integration -p 1 ./internal/gateway/pg/...`; `go vet -tags=integration ./... && go vet -tags=smoke ./...`; `golangci-lint cache clean && ~/go/bin/golangci-lint run --max-same-issues 0 --max-issues-per-linter 0`.
- [ ] **Step 2: real run** (CLAUDE.md "Entrega"; `docs/dev/seed.md`): `make migrate-up`; `go run ./cmd/devseed`; API + game from this worktree. Throwaway WS script in the scratchpad (Node `ws` or Go `gorilla/websocket`), master + seed player + third account: public status reaches only who sees the character; hidden reaches the owner only; Hidden removes the NPC piece for the third account; a stunned character's `enqueue_action` gets `blocked_by_status`; `psql` shows the rows (`audience` NULL by default); **restart the game server** and reconnect → `match_full_state.statusEffects` intact; inside a turn no row before the close, row after. Never the owner's own logged-in tabs.
- [ ] **Step 3: PR 1** (base `docs/status-effects-spec`): what was verified and how; what was not (no front ⇒ nothing visual; the player front still re-fetches participants only on reconnect — `npc_added` no longer reaches it); owner decisions; builder decisions; deferred list. Attribution line at the end.
- [ ] **Step 4:** if Step 2 covered everything, say so and skip `dev-checkout.sh`; otherwise `./dev-checkout.sh feat/status-effects-base` from `System_X_System_Project/` and say what to look at.

---

## PR 2 — history of statuses and hidden actions

Branch `feat/status-effects-history` from `feat/status-effects-base`. Before dispatching Task 14, the orchestrator reads `get_match_history.go`, `find_match_history.go`, `master_action_record.go` and `ProjectAction`/`Viewer` in `internal/domain/match/service/projection.go`, and appends the exact signatures to this section (the shapes below are fixed; the call sites are not yet mapped).

Rule (spec §9, owner-confirmed "o passado aparece"): a reader sees a history entry about a status iff **they saw the character at that instant** (sight, recorded as views like `master_actions.views`) **and** the effect's **latest** viewer list (its last recorded event, across matches) includes them — so revealing a status later shows its past to whoever was looking, and never to who was not.

### Task 14: Status events recorded and served in the history

- Migration `20261118000000_status_effect_events.sql`: `status_effect_events(uuid PK, effect_uuid UUID NOT NULL, character_sheet_uuid UUID NOT NULL REFERENCES character_sheets ON DELETE CASCADE, match_uuid REFERENCES matches ON DELETE CASCADE, scene_uuid, round_uuid, turn_uuid NULL (no FK, like master_actions), change VARCHAR(16) CHECK IN ('applied','edited','removed'), snapshot JSONB NOT NULL /* the effect as it became; for removed, as it was */, views JSONB NOT NULL /* player → "full" */, happened_at TIMESTAMPTZ NOT NULL)`; indexes on `(match_uuid, happened_at)` and `effect_uuid`.
- Recording: every write op from Tasks 8–11 also yields an event built at the instant (views = players for whom `seesCharacterLocked` was true then, plus the owner); same timing rule (turnWrites / immediate / healing), written in the same transaction as the ops.
- `GET /history` (contract `docs/dev/api/match-history.md`): entries of kind `statusEffect` (`change`, `characterId`, the snapshot projected like `StatusEffectPayload` for that reader) placed in the turn/round they happened, filtered by the rule above; the master sees all.
- Tests: gateway integration (write + list with latest-audience join); use case unit tests for the filter (hidden then made public later → past visible to a reader who was in sight, not to one who was not; removed effect keeps its last audience); handler/e2e for the history JSON per reader.

### Task 15: Actions of a hidden character

- Live: for a player from whom the actor is hidden (`hiddenFromLocked`), the turn's action is not shown — `turn_opened`, `reaction_opened`, `resolution_updated` (settled) and `turn_closed` reach that player with no actor or action identity (the projection's lowest cut; if no such cut exists in `ProjectAction`, add a `Viewer.ActorHidden` cut with tests). The damage the player's own character takes still arrives (`character_hp_changed` to the owner).
- History: the action rows record, per player, whether the actor was hidden from them at the instant (extend the existing `move_views`-style recording); `GET /history` hides such actions from that reader until the Hidden effect's latest viewer list includes them (same rule as Task 14).
- Tests: e2e (live cuts for the bystander, full for master and owner of the hidden character), history filter before and after revealing.

### Task 16: Docs, verification and PR 2

- Docs: `match-history.md`, `match-status-effects.md` (history section), `match-combat-ws.md` (hidden actor cut), design doc, documentation map.
- Verification as Task 13 Steps 1–2 (real run: hidden NPC attacks; bystander sees nothing of the action; after revealing, the history shows it).
- PR 2 (base `feat/status-effects-base`), cross-linked with PR 1; same handoff rule.
