# Documentation Separation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Separate mixed game/technical documentation into pure player-facing game docs (`docs/game/`) and developer-facing technical docs (`docs/dev/`).

**Architecture:** "Swap two variables with a third" approach — first CREATE all `docs/dev/` files (Phase 1), then REFACTOR `docs/game/` files (Phase 2), then enrich cross-references (Phase 3). This ensures no technical content is lost during the transition.

**Tech Stack:** Markdown documentation only. No code changes.

**Spec:** `docs/superpowers/specs/2026-04-29-documentation-separation-design.md` (EN) / `.pt-br.md` (PT-BR)

**Branch:** `docs/documentation-separation` (already exists with spec commit)

---

## Quality Guidelines (apply to ALL tasks)

### Dev docs (`docs/dev/`) — PT-BR, code refs in English
- Focus on cross-package flows, design rationale, non-obvious relationships
- Link to source files instead of replicating struct definitions
- Include section headers, flow diagrams, key interfaces
- DON'T document what the code already says clearly
- See spec Example 3 (cascade) as the quality bar

### Game docs (`docs/game/`) — PT-BR, zero code references
- No method names, package paths, error types, technology names
- Tone: informative, clear, friendly — like a RPG rulebook chapter
- Preserve all game mechanics and formulas (use mathematical notation)
- "Curiosity sections" welcome where technical details translate to player interest
- See spec Example 1 (dice/RNG) as the quality bar

### Developer Footer Convention (game docs where relevant)
```markdown
---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/<path>`](../dev/<path>)
> Código-fonte: `internal/domain/entity/<path>/`
```

### Footer Mapping Table

| Game Doc | Dev Doc(s) | Source Code |
|----------|-----------|-------------|
| `ficha-de-personagem/experiencia.md` | `dev/character-sheet/experience.md` | `entity/character_sheet/experience/` |
| `ficha-de-personagem/habilidades.md` | `dev/character-sheet/abilities-attributes.md` | `entity/character_sheet/ability/` |
| `ficha-de-personagem/atributos.md` | `dev/character-sheet/abilities-attributes.md` | `entity/character_sheet/attribute/` |
| `ficha-de-personagem/pericias.md` | `dev/character-sheet/skills-proficiencies.md` | `entity/character_sheet/skill/` |
| `ficha-de-personagem/proficiencias.md` | `dev/character-sheet/skills-proficiencies.md` | `entity/character_sheet/proficiency/` |
| `ficha-de-personagem/sistema-nen.md` | `dev/character-sheet/spiritual.md` | `entity/character_sheet/spiritual/` |
| `ficha-de-personagem/status.md` | `dev/character-sheet/status.md` | `entity/character_sheet/status/` |
| `classes.md` | `dev/character-sheet/factory.md` | `entity/character_class/` |
| `dados.md` | `dev/weapons-dice.md` | `entity/die/` |
| `armas.md` | `dev/weapons-dice.md` | `entity/item/` |
| `campanhas.md` | `dev/campaigns-scenarios.md` | `entity/campaign/`, `domain/campaign/` |
| `cenarios.md` | `dev/campaigns-scenarios.md` | `entity/scenario/`, `domain/scenario/` |
| `partidas.md` | `dev/enrollment.md`, `dev/match/` | `entity/match/`, `domain/enrollment/` |
| `cenas-e-turnos.md` | `dev/match/scenes.md`, `dev/match/turns-rounds.md` | `entity/match/scene/`, `entity/match/turn/` |
| `combate/acoes.md` | `dev/match/actions.md` | `entity/match/action/` |
| `glossario.md` | *(sem footer — é referência pura)* | — |
| `autenticacao.md` | *(removido dos game docs)* | — |

---

## Phase 1: Create Technical Documentation (`docs/dev/`)

### Task 1: Setup — Directory Structure & Move Overview

**Files:**
- Move: `docs/architecture/overview.md` → `docs/dev/overview.md`
- Create dirs: `docs/dev/character-sheet/`, `docs/dev/match/`

- [ ] **Step 1: Create directory structure**

```bash
mkdir -p docs/dev/character-sheet docs/dev/match
```

- [ ] **Step 2: Move overview.md**

```bash
git mv docs/architecture/overview.md docs/dev/overview.md
rmdir docs/architecture
```

- [ ] **Step 3: Verify file was moved correctly**

```bash
cat docs/dev/overview.md | head -5
ls docs/dev/
```

Expected: overview.md exists in docs/dev/, docs/architecture/ is gone.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "docs(dev): move architecture/overview.md to docs/dev/

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 2: Dev Docs — Experience & Cascade Flow

The cascade is the most critical cross-package flow in the system. This doc should be the most detailed dev doc.

**Files:**
- Create: `docs/dev/character-sheet/experience.md`

**Source code to read:**
- `internal/domain/entity/character_sheet/experience/` (all files)
- `internal/domain/entity/character_sheet/skill/` (cascade entry point)
- `internal/domain/entity/character_sheet/attribute/` (cascade middle)
- `internal/domain/entity/character_sheet/ability/` (cascade top)

**Current game doc:** `docs/game/ficha-de-personagem/experiencia.md`

**Sections to include:**
1. **ExpTable** — sigmoidal function, coefficient system, level range (0-100)
2. **Cascade Upgrade Flow** — the full 4-package call chain with method signatures (use the spec Example 3 as template)
3. **CharacterExp** — character points, ability bonus formula
4. **UpgradeCascade collector** — how cascade data is accumulated and returned
5. **Key interfaces** — `ICascadeUpgrade`, `ITriggerCascadeExp`
6. **Extension guide** — how to add a new cascade participant

- [ ] **Step 1: Read source code for experience package**

Read all files in `internal/domain/entity/character_sheet/experience/`.

- [ ] **Step 2: Read cascade entry/exit points**

Read `skill/skill.go` (CascadeUpgradeTrigger), `attribute/` (CascadeUpgrade), `ability/` (CascadeUpgrade).

- [ ] **Step 3: Write `docs/dev/character-sheet/experience.md`**

Write the dev doc following the sections above. Use the spec Example 3 (cascade flow) as the quality bar. Language: PT-BR with code references in English.

- [ ] **Step 4: Verify content accuracy**

Cross-check method names, interfaces, and flow against source code.

- [ ] **Step 5: Commit**

```bash
git add docs/dev/character-sheet/experience.md
git commit -m "docs(dev): experience system & cascade flow

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 3: Dev Docs — Abilities & Attributes

**Files:**
- Create: `docs/dev/character-sheet/abilities-attributes.md`

**Source code to read:**
- `internal/domain/entity/character_sheet/ability/` (all files)
- `internal/domain/entity/character_sheet/attribute/` (all files)

**Current game docs:**
- `docs/game/ficha-de-personagem/habilidades.md`
- `docs/game/ficha-de-personagem/atributos.md`

**Sections to include:**
1. **Entity hierarchy** — PrimaryAttribute, MiddleAttribute, SpiritualAttribute, Ability
2. **Manager pattern** — how Manager and SpiritualManager organize attributes
3. **Middle attribute calculation** — average with remainder tracking (the XP division algorithm)
4. **Ability bonus formula** — (characterPoints + abilityLevel) / 2
5. **Talent system** — TalentByCategorySet, hexagon-aware bonus calculation
6. **Buff system** — pointer-based buff sharing between Manager and attributes
7. **Distribution** — which attributes accept manual point distribution

- [ ] **Step 1: Read ability and attribute source code**

Read all files in `ability/` and `attribute/` packages.

- [ ] **Step 2: Write `docs/dev/character-sheet/abilities-attributes.md`**

Focus on the entity hierarchy diagram, Manager pattern, middle attribute XP division algorithm, and buff pointer sharing. These are the non-obvious parts.

- [ ] **Step 3: Verify accuracy against source**

- [ ] **Step 4: Commit**

```bash
git add docs/dev/character-sheet/abilities-attributes.md
git commit -m "docs(dev): abilities & attributes entity hierarchy

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 4: Dev Docs — Skills & Proficiencies

**Files:**
- Create: `docs/dev/character-sheet/skills-proficiencies.md`

**Source code to read:**
- `internal/domain/entity/character_sheet/skill/` (all files)
- `internal/domain/entity/character_sheet/proficiency/` (all files)

**Current game docs:**
- `docs/game/ficha-de-personagem/pericias.md`
- `docs/game/ficha-de-personagem/proficiencias.md`

**Sections to include:**
1. **Skill Manager lookup priority** — joint skills checked first, then common (spec Example 2)
2. **CommonSkill vs JointSkill** — structural differences, cascade behavior
3. **JointSkill XP multiplication** — exp × component count during cascade
4. **JointSkill initialization** — Init() requirement, error on double-init
5. **Proficiency cascade differences** — how proficiency cascade differs from skill cascade
6. **JointProficiency** — multi-weapon grouping, buff-per-weapon system
7. **Manager buff system** — independent buff layer on top of skill/proficiency levels
8. **Value for test calculation** — formula with buff integration

- [ ] **Step 1: Read skill and proficiency source code**

- [ ] **Step 2: Write `docs/dev/character-sheet/skills-proficiencies.md`**

The key non-obvious detail is the JointSkill XP multiplication and how the Manager prioritizes joint skills in lookups. Use spec Example 2 as reference.

- [ ] **Step 3: Verify accuracy**

- [ ] **Step 4: Commit**

```bash
git add docs/dev/character-sheet/skills-proficiencies.md
git commit -m "docs(dev): skills & proficiencies — types, cascade, Manager

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 5: Dev Docs — Spiritual/Nen System

**Files:**
- Create: `docs/dev/character-sheet/spiritual.md`

**Source code to read:**
- `internal/domain/entity/character_sheet/spiritual/` (all files)

**Current game doc:** `docs/game/ficha-de-personagem/sistema-nen.md`

**Sections to include:**
1. **Hexagon algorithm** — circular distance calculation, percentage formula, 600-value scale
2. **Specialization exception** — returns 0% unless it's the character's category
3. **Hatsu initialization** — category map requirement, double-init error
4. **Cascade path** — Category → Hatsu → Conscience → Spiritual Ability → CharacterExp
5. **PrinciplesManager** — coordination of principles, hexagon, and Hatsu
6. **Category value for test** — formula with percentage scaling
7. **Reset mechanism** — resetNenCategory inspired by Chimera Ant arc

- [ ] **Step 1: Read spiritual package source code**

- [ ] **Step 2: Write `docs/dev/character-sheet/spiritual.md`**

The hexagon algorithm and cascade path are the most valuable parts — a dev would spend significant time tracing these without docs.

- [ ] **Step 3: Verify accuracy**

- [ ] **Step 4: Commit**

```bash
git add docs/dev/character-sheet/spiritual.md
git commit -m "docs(dev): spiritual/Nen system — hexagon, Hatsu, cascade

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 6: Dev Docs — Status & Factory

**Files:**
- Create: `docs/dev/character-sheet/status.md`
- Create: `docs/dev/character-sheet/factory.md`

**Source code to read:**
- `internal/domain/entity/character_sheet/status/` (all files)
- `internal/domain/entity/character_sheet/sheet/` (all files — factory lives here)

**Current game doc:** `docs/game/ficha-de-personagem/status.md`

**Sections for `status.md`:**
1. **HP/SP/AP formulas** — exact Go expressions with variable names mapped to entities
2. **Upgrade trigger** — when and how Upgrade() is called after cascade
3. **Conditional AP** — nil spirituals check, AP only for Nen users
4. **StatusBar mechanics** — min/curr/max, "full means stays full" on upgrade

**Sections for `factory.md`:**
1. **Entity construction graph** — order of creation, dependencies between entities
2. **Character class wrapping** — how Wrap() applies class bonuses
3. **Coefficient mapping** — which entity gets which coefficient
4. **Distribution validation** — skill/proficiency distribution flow during creation

- [ ] **Step 1: Read status and sheet/factory source code**

- [ ] **Step 2: Write `docs/dev/character-sheet/status.md`**

Short doc. Focus on the exact formulas with Go variable names mapped to entity accessors.

- [ ] **Step 3: Write `docs/dev/character-sheet/factory.md`**

Focus on the entity construction graph — the order matters and isn't obvious from reading the factory alone.

- [ ] **Step 4: Verify accuracy**

- [ ] **Step 5: Commit**

```bash
git add docs/dev/character-sheet/status.md docs/dev/character-sheet/factory.md
git commit -m "docs(dev): status calculations & factory entity graph

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 7: Dev Docs — Weapons, Dice & Auth

**Files:**
- Create: `docs/dev/weapons-dice.md`
- Create: `docs/dev/auth.md`

**Source code to read:**
- `internal/domain/entity/die/die.go`
- `internal/domain/entity/item/` (weapon.go, weapons_factory.go, weapons_manager.go)
- `internal/domain/auth/` (all files)
- `internal/domain/session/` (all files)

**Current game docs:**
- `docs/game/dados.md`
- `docs/game/armas.md`
- `docs/game/autenticacao.md`

**Sections for `weapons-dice.md`:**
1. **RNG mechanism** — crypto/rand primary, math/rand/v2 fallback (spec Example 1)
2. **Weapon entity** — properties, penalty/stamina cost calculations
3. **WeaponsFactory** — how weapons are constructed from enum
4. **WeaponsManager** — inventory management, active weapon delegation
5. **Copy semantics** — why GetDice returns a copy

**Sections for `auth.md`:**
1. **Login flow** — email lookup → bcrypt verify → JWT generate → session store
2. **Dual session storage** — sync.Map (fast) + PostgreSQL (persistent), fire-and-forget pattern
3. **Security decisions** — why "access denied" for both wrong email and wrong password
4. **Registration validation** — field rules, uniqueness checks

- [ ] **Step 1: Read die, item, auth, and session source code**

- [ ] **Step 2: Write `docs/dev/weapons-dice.md`**

Use spec Example 1 (RNG) for the dice section.

- [ ] **Step 3: Write `docs/dev/auth.md`**

The dual session storage pattern (sync.Map + DB with fire-and-forget) is the most valuable non-obvious detail.

- [ ] **Step 4: Verify accuracy**

- [ ] **Step 5: Commit**

```bash
git add docs/dev/weapons-dice.md docs/dev/auth.md
git commit -m "docs(dev): weapons/dice RNG & auth session flow

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 8: Dev Docs — Campaigns, Scenarios, Enrollment

**Files:**
- Create: `docs/dev/campaigns-scenarios.md`
- Create: `docs/dev/enrollment.md`

**Source code to read:**
- `internal/domain/entity/campaign/` (all files)
- `internal/domain/entity/scenario/` (all files)
- `internal/domain/campaign/` (use case)
- `internal/domain/scenario/` (use case)
- `internal/domain/enrollment/` (use case)
- `internal/domain/submission/` (use case)

**Current game docs:**
- `docs/game/campanhas.md`
- `docs/game/cenarios.md`
- `docs/game/partidas.md`

**Sections for `campaigns-scenarios.md`:**
1. **Entity hierarchy** — Scenario → Campaign → Match
2. **Validation rules** — field constraints, date validation logic
3. **Submission flow** — player submits sheet → master accepts/rejects
4. **Visibility model** — public vs private, owner-based access

**Sections for `enrollment.md`:**
1. **Enrollment flow** — sheet accepted in campaign → enroll in match
2. **Validation chain** — sheet ownership → campaign membership → match membership
3. **Cross-entity relationships** — match↔campaign↔sheet↔user

Be critical here — if the code is self-explanatory enough, keep these docs concise. The enrollment cross-entity flow is the most valuable part.

- [ ] **Step 1: Read campaign, scenario, enrollment, submission source code**

- [ ] **Step 2: Write `docs/dev/campaigns-scenarios.md`**

- [ ] **Step 3: Write `docs/dev/enrollment.md`**

- [ ] **Step 4: Verify accuracy**

- [ ] **Step 5: Commit**

```bash
git add docs/dev/campaigns-scenarios.md docs/dev/enrollment.md
git commit -m "docs(dev): campaigns, scenarios & enrollment flows

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 9: Dev Docs — Match Runtime & WebSocket

**Files:**
- Create: `docs/dev/match/scenes.md`
- Create: `docs/dev/match/turns-rounds.md`
- Create: `docs/dev/match/actions.md`
- Create: `docs/dev/websocket.md`

**Source code to read:**
- `internal/domain/entity/match/` (all subdirectories: scene, turn, round, action, battle)
- `internal/domain/match/` (engine use case)
- `internal/app/game/` (WebSocket handlers)

**Current game docs:**
- `docs/game/cenas-e-turnos.md`
- `docs/game/combate/acoes.md`
- `docs/game/partidas.md`

**Important note:** Turn/Round system is under semantic refactoring. Document the CURRENT state and note the WIP status.

**Sections for `scenes.md`:**
1. **Scene lifecycle** — create → execute turns → finalize
2. **Category vs Mode separation** — why scene category doesn't determine turn mode

**Sections for `turns-rounds.md`:**
1. **Turn Engine** — mode-agnostic execution, free vs race
2. **Priority queue** — max-heap implementation for race mode
3. **Round structure** — action + triggered reactions
4. **WIP note** — semantic refactoring in progress

**Sections for `actions.md`:**
1. **Action struct** — components (actor, target, speed, attack, defense, dodge)
2. **Speed resolution** — bar + rollCheck → queue position
3. **Reaction chain** — how reactions enter the queue

**Sections for `websocket.md`:**
1. **Hub/Room/Client pattern** — architecture diagram
2. **State machine** — Lobby → Playing → Closed
3. **Message protocol** — message types and flow
4. **Match lifecycle over WS** — enrollment → lobby → start → play → close

- [ ] **Step 1: Read match entity code (all subdirectories)**

- [ ] **Step 2: Read app/game WebSocket code**

- [ ] **Step 3: Write all four files**

These docs don't need to be long — the match system is WIP. Focus on what's stable and useful.

- [ ] **Step 4: Verify accuracy**

- [ ] **Step 5: Commit**

```bash
git add docs/dev/match/ docs/dev/websocket.md
git commit -m "docs(dev): match runtime, turns/rounds & WebSocket

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Phase 2: Refactor Game Documentation (`docs/game/`)

### General Process for Each Game Doc Refactor

For every file:
1. Read the current game doc
2. Read the corresponding `docs/dev/` file (to know what was already extracted)
3. Remove ALL code references: method names, package paths, error type names, technology names
4. Rewrite technical language in player-accessible terms
5. Preserve all game mechanics, formulas (mathematical notation), tables
6. Add developer footer where a `docs/dev/` counterpart exists
7. Add "curiosity sections" where technical details translate to player interest

---

### Task 10: Game Docs — Experience, Abilities, Attributes

**Files:**
- Modify: `docs/game/ficha-de-personagem/experiencia.md`
- Modify: `docs/game/ficha-de-personagem/habilidades.md`
- Modify: `docs/game/ficha-de-personagem/atributos.md`

**What to remove/rewrite:**

`experiencia.md`:
- Remove: "ExpTable", "`CharacterExp`", "instância de experiência" (software term)
- Remove: "função sigmoidal tripla" formula (too technical, players don't need it)
- Keep: coefficient table (relabel as "Velocidade de Progressão"), cascade explanation
- Rewrite cascade using spec Example 3 (climbing → cascade analogy)
- Add developer footer linking to `docs/dev/character-sheet/experience.md`

`habilidades.md`:
- Remove: "cascade upgrade" (reword as "progressão em cascata")
- Remove: "`TalentByCategorySet`"
- Keep: all ability descriptions, bonus formula, talent system
- Add developer footer

`atributos.md`:
- Remove: ALL Manager references (`Manager`, `SpiritualManager`, `Get`, `GetPrimary`, `IncreasePointsForPrimary`, `SetBuff`, `RemoveBuff`, `GetAllAttributes`, `GetAttributesLevel`, `GetAttributesPoints`)
- Remove: "ponteiro" (pointer), "`math.Round`" (say "arredondamento bancário")
- Keep: formulas (valor, poder), attribute tables, buff concept, distribution rules
- Rewrite "Gerenciadores de Atributos" section as "Como Atributos Funcionam"
- Add developer footer

- [ ] **Step 1: Refactor `experiencia.md`**

Rewrite using the spec Example 3 as the quality bar. The cascade explanation should use the climbing/acrobatics analogy.

- [ ] **Step 2: Refactor `habilidades.md`**

Relatively clean already — main changes are removing code terms.

- [ ] **Step 3: Refactor `atributos.md`**

This file has the most mixing. Remove entire Manager/SpiritualManager sections, rewrite as player-facing explanations of how attributes work.

- [ ] **Step 4: Add developer footers to all three files**

- [ ] **Step 5: Verify no code references remain**

Search all three files for: backtick code references, method names (PascalCase/camelCase), package paths, error type names.

- [ ] **Step 6: Commit**

```bash
git add docs/game/ficha-de-personagem/experiencia.md \
       docs/game/ficha-de-personagem/habilidades.md \
       docs/game/ficha-de-personagem/atributos.md
git commit -m "docs(game): refactor experience, abilities & attributes for players

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 11: Game Docs — Skills, Proficiencies, Nen, Status

**Files:**
- Modify: `docs/game/ficha-de-personagem/pericias.md`
- Modify: `docs/game/ficha-de-personagem/proficiencias.md`
- Modify: `docs/game/ficha-de-personagem/sistema-nen.md`
- Modify: `docs/game/ficha-de-personagem/status.md`

**What to remove/rewrite:**

`pericias.md`:
- Remove: ALL Manager methods (`Init`, `Get`, `IncreaseExp`, `AddJointSkill`, `GetValueForTestOf`, `GetSkillsLevel`)
- Remove: "Gerenciador de Perícias (Manager)" section title → rename to game-friendly equivalent
- Remove: "Fluxo de Experiência (Cascade)" code-style flow diagram
- Remove: Manager buff operations (`SetBuff`, `DeleteBuff`, `GetBuffs`)
- Keep: skill list, value-for-test formula, joint skill explanation, cascade concept
- Rewrite joint skill explanation following spec Example 2 direction
- Add developer footer

`proficiencias.md`:
- Remove: `IProficiency` interface, package path, struct names, `CascadeUpgradeTrigger`
- Remove: "Referência Técnica" section entirely
- Remove: Manager method names (`Get`, `AddCommon`, `AddJoint`, `IncreaseExp`)
- Remove: `Enum` column in weapon tables → use only PT-BR name
- Keep: concept, common vs joint proficiency, cascade concept, weapon lists
- Add developer footer

`sistema-nen.md`:
- Remove: `PrinciplesManager` section with method names
- Remove: "`IncreaseCurrHexValue()`", "`DecreaseCurrHexValue()`", "`ResetNenCategory()`", etc.
- Remove: "inicialização" technical details (Init, double-init error)
- Keep: principles table, categories table, hexagon explanation, formulas, cascade concept
- Rewrite PrinciplesManager section as "Controle do Sistema Nen" for players
- Add developer footer

`status.md`:
- Remove: `IncreaseAt`, `DecreaseAt`, `SetCurrent`, `Upgrade` method names
- Remove: "Operações" section → rewrite as "Como as Barras Funcionam"
- Keep: formulas, mechanics explanation
- Add developer footer

- [ ] **Step 1: Refactor `pericias.md`**

The most important change: remove the "Gerenciador de Perícias" section and rewrite skill management in player terms.

- [ ] **Step 2: Refactor `proficiencias.md`**

Remove the "Referência Técnica" section entirely. Remove Enum column from weapon tables.

- [ ] **Step 3: Refactor `sistema-nen.md`**

Rewrite PrinciplesManager as player-facing content. The hexagon explanation is already good for players — just remove method references.

- [ ] **Step 4: Refactor `status.md`**

Already mostly clean. Remove method names from "Operações" section.

- [ ] **Step 5: Add developer footers**

- [ ] **Step 6: Verify no code references remain**

- [ ] **Step 7: Commit**

```bash
git add docs/game/ficha-de-personagem/pericias.md \
       docs/game/ficha-de-personagem/proficiencias.md \
       docs/game/ficha-de-personagem/sistema-nen.md \
       docs/game/ficha-de-personagem/status.md
git commit -m "docs(game): refactor skills, proficiencies, Nen & status for players

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 12: Game Docs — Classes, Dice, Weapons, Auth

**Files:**
- Modify: `docs/game/classes.md`
- Modify: `docs/game/dados.md`
- Modify: `docs/game/armas.md`
- Delete: `docs/game/autenticacao.md` (conteúdo técnico já coberto por `docs/dev/auth.md`)

**What to remove/rewrite:**

`classes.md`:
- Remove: ALL error type names (`NoSkillDistributionError`, `SkillsCountMismatchError`, `SkillNotAllowedError`, `SkillsPointsMismatchError`, and proficiency equivalents)
- Remove: `ApplySkills`, `ApplyProficiencies` method references
- Remove: "Erros possíveis" section with error type names → rewrite as "Regras de Validação" in plain language
- Remove: "Perfil da Classe (Class Profile)" section (technical metadata)
- Keep: class list, distribution mechanics, validation rules (in player language)
- Add developer footer

`dados.md`:
- Remove: "`crypto/rand`", "`math/rand/v2`" references
- Rewrite using spec Example 1 — true randomness explanation + curiosity section
- Keep: dice types table, combination mechanics, weapon dice examples
- Add developer footer

`armas.md`:
- Remove: "Segurança de Dados" section (copy semantics — irrelevant to players)
- Remove: "Gerenciador de Armas" section → rewrite as "Inventário de Armas" in player terms
- Remove: "(Add)", "(Delete)", "(Get)", "(GetAll)" method references
- Keep: property table, penalty/stamina formulas, weapon tables
- Column "Tipo" in properties table: change "Lista de inteiros" → "Combinação de dados" / "Inteiro" → "Número" / "Decimal" → "Número" / "Booleano" → "Sim/Não"
- Add developer footer

`autenticacao.md`:
- This is NOT a game rule — it's platform usage info
- **Decisão:** Remover de `docs/game/`. Todo conteúdo técnico está coberto em `docs/dev/auth.md`. Informações de registro/login para jogadores, se necessárias, pertencem a um futuro "Guia de Início", não às regras do RPG.

- [ ] **Step 1: Refactor `classes.md`**

Remove error types and method names. Rewrite validation rules in player language.

- [ ] **Step 2: Refactor `dados.md`**

Use spec Example 1 as template. Add the curiosity section about true randomness.

- [ ] **Step 3: Refactor `armas.md`**

Remove copy semantics and Manager sections. Rewrite property types for players.

- [ ] **Step 4: Remove `autenticacao.md`**

```bash
git rm docs/game/autenticacao.md
```

- [ ] **Step 5: Add developer footers to classes, dados, armas**

- [ ] **Step 6: Verify no code references remain**

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "docs(game): refactor classes, dice & weapons; remove auth from game docs

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

### Task 13: Game Docs — Campaigns, Scenarios, Matches, Scenes, Combat

**Files:**
- Modify: `docs/game/campanhas.md`
- Modify: `docs/game/cenarios.md`
- Modify: `docs/game/partidas.md`
- Modify: `docs/game/cenas-e-turnos.md`
- Modify: `docs/game/combate/acoes.md`

**What to remove/rewrite:**

`campanhas.md`:
- Mostly clean already. Minor: remove any code-like language
- Add developer footer

`cenarios.md`:
- Mostly clean. Minor cleanup
- Add developer footer

`partidas.md`:
- Remove: "Turn/Round Engine está em refatoração semântica" (technical note)
- Remove: "spec de design do WebSocket Game Server" reference
- Remove: "WebSocket" and "Game Server" technology names → say "conexão em tempo real"
- Keep: match structure, enrollment rules, hierarchy, real-time flow concept
- Add developer footer

`cenas-e-turnos.md`:
- Remove: "Engine de Turnos" → say "sistema de turnos"
- Remove: "max-heap" → say "fila de prioridade por velocidade"
- Remove: "Turn Engine" reference
- Remove: "spec de design do WebSocket Game Server" reference
- Remove: "WebSocket" → "conexão em tempo real"
- Keep: hierarchy, scene categories, turn modes, round structure, game events
- Add developer footer

`combate/acoes.md`:
- Remove: "max-heap" → "fila de prioridade por velocidade"
- Remove: "RollContext" struct name → say "contexto de rolagem"
- Remove: code-like field descriptions ("Struct", "Referência à ação")
- Remove: "Insert", "ExtractMax", "Peek", "ExtractByID" → describe operations in plain language
- Keep: action structure (in player terms), speed mechanics, priority queue concept, attack/defense structure, game events
- Add developer footer

- [ ] **Step 1: Refactor `campanhas.md`**

Light touch — mostly clean already.

- [ ] **Step 2: Refactor `cenarios.md`**

Light touch — mostly clean.

- [ ] **Step 3: Refactor `partidas.md`**

Remove WebSocket/tech references. Rewrite real-time section in player terms.

- [ ] **Step 4: Refactor `cenas-e-turnos.md`**

Replace technical terms (Engine, max-heap, WebSocket) with player-friendly language.

- [ ] **Step 5: Refactor `combate/acoes.md`**

Remove data structure references. Rewrite priority queue in game terms.

- [ ] **Step 6: Add developer footers to relevant files**

- [ ] **Step 7: Verify no code references remain**

- [ ] **Step 8: Commit**

```bash
git add docs/game/campanhas.md docs/game/cenarios.md \
       docs/game/partidas.md docs/game/cenas-e-turnos.md \
       docs/game/combate/acoes.md
git commit -m "docs(game): refactor campaigns, matches, scenes & combat for players

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Phase 3: Finalization

### Task 14: Enrich Overview, Glossary Review & PR

**Files:**
- Modify: `docs/dev/overview.md`
- Review: `docs/game/glossario.md` (already clean — verify no changes needed)

- [ ] **Step 1: Enrich `docs/dev/overview.md` with cross-references**

Add a "Documentação por Domínio" section linking to each `docs/dev/` file:

```markdown
## Documentação por Domínio

| Domínio | Documentação | Código-fonte |
|---------|-------------|--------------|
| Experiência & Cascata | [experience.md](character-sheet/experience.md) | `entity/character_sheet/experience/` |
| Habilidades & Atributos | [abilities-attributes.md](character-sheet/abilities-attributes.md) | `entity/character_sheet/ability/`, `attribute/` |
| ... | ... | ... |
```

Complete the table with all dev docs created in Phase 1.

- [ ] **Step 2: Review `docs/game/glossario.md`**

Verify it contains no technical references. It should be clean already — just confirm.

- [ ] **Step 3: Final verification — scan all game docs for remaining code references**

```bash
# Search for code patterns AND banned tech terms in game docs
grep -rn '`[A-Z][a-zA-Z]*`\|crypto/\|math/\|internal/\|\.go\b' docs/game/ || true
grep -rin 'websocket\|web socket\|bcrypt\|jwt\|sync\.Map\|PostgreSQL\|pgx\|max-heap\|maxheap\|crypto/rand\|math/rand' docs/game/ || true
grep -rn 'Manager\|Engine\|Factory\|Interface\|Struct\|Init()\|Get()\|Set(' docs/game/ || true
echo "If any matches above, fix them before PR."
```

Verify the spec examples were implemented correctly:
- `dados.md` must contain the 🎲 curiosity section about true randomness
- `experiencia.md` must use the cascade analogy (escalada/acrobacia example)
- `pericias.md` must explain joint skills in player-friendly terms (practical example)

- [ ] **Step 4: Commit**

```bash
git add docs/dev/overview.md
git commit -m "docs(dev): enrich overview with domain cross-references

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

- [ ] **Step 5: Push branch and create PR**

```bash
git push -u origin docs/documentation-separation
gh pr create --title "docs: separate game rules from technical documentation" \
  --body "## Summary

Separates mixed documentation into two audience-specific sets:

- **\`docs/game/\`** — Pure game rules for players (like a RPG rulebook)
- **\`docs/dev/\`** — Technical docs for developers (flows, rationale, architecture)

### Changes
- Created \`docs/dev/\` with per-domain technical documentation
- Refactored all \`docs/game/\` files to remove code references
- Added developer footers linking game docs to their technical counterparts
- Moved \`docs/architecture/overview.md\` → \`docs/dev/overview.md\`
- Removed \`docs/game/autenticacao.md\` (not a game rule)

### Design Spec
See \`docs/superpowers/specs/2026-04-29-documentation-separation-design.md\`"
```

---

## Task Dependencies

```
Task 1 (setup)
  ├─→ Tasks 2-9 (Phase 1 — dev docs, can run in parallel)
  │     ├─→ Tasks 10-13 (Phase 2 — game docs, can run in parallel after Phase 1)
  │     │     └─→ Task 14 (Phase 3 — finalization)
```

**Parallelism opportunities:**
- Phase 1 (Tasks 2-9): All independent — can be dispatched in parallel
- Phase 2 (Tasks 10-13): All independent — can be dispatched in parallel after Phase 1 completes
- Phase 3 (Task 14): Sequential, depends on everything above
