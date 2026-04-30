# Documentation Separation Design Spec

> Separate game rule documentation (player-facing) from technical documentation (developer-facing).

## Problem Statement

The current `docs/game/` directory contains mixed content: game rules for players alongside implementation details for developers (method names, package references, error types, internal patterns). This makes the documentation unsuitable for two key audiences:

1. **Players/Masters** — who need clear game rules they can reference during play (like a D&D Player's Handbook)
2. **Developers** — who need technical documentation explaining code flows, entities, and how to contribute

Additionally, the game documentation will be exported in the future for:
- A physical/digital RPG rulebook
- Frontend display aligned with UX designed by the project's designer

## Goals

1. **Create** `docs/architecture/` per-domain technical docs with implementation details, code flows, entity relationships, and contribution guidelines
2. **Refactor** `docs/game/` to read as a pure rulebook — accessible to non-developers, modular, exportable
3. **Bridge** the two via developer footers in game docs (where relevant), linking to technical counterparts

## Non-Goals

- Rewriting game rules or changing mechanics
- Documenting the frontend integration
- Creating a formatted book layout (that's a future step)

## Design

### Phase 1: Create Technical Documentation (`docs/dev/`)

Create per-domain technical docs using content already in game docs PLUS reading source code for deeper implementation details. Not every domain needs a dedicated doc — be critical about what genuinely adds value beyond what the code already communicates.

#### Structure

```
docs/dev/
├── overview.md                  (move from docs/architecture/ — enrich)
├── character-sheet/
│   ├── experience.md            ← ExpTable, cascade mechanism, interfaces, coefs
│   ├── abilities-attributes.md  ← entity hierarchy, formulas, Manager pattern
│   ├── skills-proficiencies.md  ← types, distribution, cascade differences
│   ├── spiritual.md             ← Nen principles, hexagon algorithm, Hatsu
│   ├── status.md                ← HP/SP/AP calculations, upgrade triggers
│   └── factory.md              ← CharacterSheetFactory, entity graph, class wrap
├── auth.md                      ← JWT, session (sync.Map + DB), bcrypt, login flow
├── campaigns-scenarios.md       ← CRUD flows, validation rules, repos
├── enrollment.md                ← enrollment flow, JOIN path, match↔sheet relations
├── weapons-dice.md              ← Weapon entity, penalty/stamina calcs, RNG
├── match/
│   ├── scenes.md                ← lifecycle, category vs mode separation
│   ├── turns-rounds.md          ← Turn Engine, free/race, priority queue impl
│   └── actions.md               ← Action struct, speed, combat resolution
└── websocket.md                 ← Hub/Room/Client, state machine, protocol
```

This is a starting point — some docs may be merged or omitted if the code is self-explanatory for that domain.

#### Content Guidelines (Technical Docs)

The code should be self-explanatory for individual entities. Dev docs exist to provide context that ISN'T obvious from reading a single file:

- **Cross-package flows** — explain journeys that span multiple files/packages (e.g., XP cascade crossing 4 packages)
- **Design rationale** — WHY something is designed this way, not WHAT the struct contains
- **Non-obvious relationships** — how entities connect in ways that aren't clear from imports alone
- **Source file references** — link to source files rather than replicating struct definitions
- **Extension guidance** — how to add a new component (new skill, new weapon) when the pattern isn't immediately obvious

Don't document mechanically. Be critical — if a domain's code is clear enough on its own, a doc isn't needed. If a flow crosses 4 packages and would take a developer 30 minutes to trace, a doc that explains it in 2 minutes is valuable.

**Language:** Dev docs are written in PT-BR (same as game docs), since the project's developer community is Brazilian. Code references (type names, method names, file paths) remain in English as they appear in the source.

### Phase 2: Refactor Game Documentation (`docs/game/`)

Rewrite each file to read as a rulebook chapter for players.

Not every domain needs a game doc either — for example, authentication details are not game rules. Be critical about what genuinely serves players.

#### Refactoring Principles

1. **Remove all code references** — no method names (`GetValueForTest`, `CascadeUpgrade`), no package paths, no error type names, no technology names (bcrypt, JWT, crypto/rand, max-heap)
2. **Keep all game mechanics** — formulas (in plain math notation), tables of values, rules
3. **Use accessible language** — a player who has never programmed should understand everything
4. **Tone** — informative, clear, friendly. Like a well-written RPG rulebook chapter
5. **Modular** — each file is a self-contained "chapter" that makes sense independently
6. **Preserve structure** — keep the same files and general organization

#### Developer Footer Convention

At the bottom of game docs (where relevant), include a footer separated by `---`:

```markdown
---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/<path>`](../dev/<path>)
> Código-fonte: `internal/domain/entity/<path>/`
```

This footer:
- Is visually separated from game content
- Clearly marked as developer-only
- Links directly to the technical counterpart
- References the source code location
- Is NOT present in every file — only where it adds value

### Execution Order

The "swap two variables with a third" approach:

1. **First**: CREATE all `docs/dev/` files (the "third variable")
   - Move existing `docs/architecture/overview.md` to `docs/dev/overview.md`
   - Use existing game docs as source for technical content
   - Read source code to enrich beyond what game docs currently contain
   - This ensures no information is lost
2. **Then**: REFACTOR `docs/game/` files
   - Remove technical details (they now live in dev)
   - Rewrite for player readability
   - Add developer footers where relevant
   - Remove files that don't make sense as game rules (e.g., `autenticacao.md`)
3. **Finally**: Enrich `docs/dev/overview.md` with cross-references
4. **Implement examples**: The transformation examples in this spec (dice curiosity section, skills explanation, cascade explanation) should be implemented as-is during execution — they serve as the quality bar for the documentation.

### Priority Order (Cascade Flow)

Follow the experience cascade structure, then expand outward:

1. Experience system
2. Skills & Proficiencies
3. Attributes (physical, mental, spiritual)
4. Abilities
5. Status bars
6. Nen/Spiritual system
7. Classes
8. Weapons & Dice
9. Auth & Sessions
10. Campaigns & Scenarios
11. Matches & Enrollment
12. Scenes, Turns & Rounds
13. Combat Actions

### Examples of Transformation

Each example shows the same content in three forms: the current mixed state, the player-facing rewrite, and the technical dev doc counterpart.

---

#### Example 1: Dice — Random Number Generation

**Current (`docs/game/dados.md`) — Mixed:**

```markdown
### Geração de Resultado
O sistema utiliza aleatoriedade criptograficamente segura (`crypto/rand`)
para gerar resultados. Em caso de falha (extremamente raro), realiza
fallback para `math/rand/v2`.
```

**After — Player-facing (`docs/game/dados.md`):**

```markdown
### Geração de Resultado

O sistema garante que todas as rolagens de dados são **verdadeiramente
aleatórias** — não são pseudo-aleatórias como na maioria dos jogos digitais.
Isso significa que cada rolagem é tão imprevisível quanto um dado físico
na mesa.

#### 🎲 Curiosidade: Por que os dados são realmente aleatórios?

Computadores normalmente geram números "pseudo-aleatórios" — sequências que
parecem aleatórias mas são geradas por uma fórmula matemática determinística.
Se você soubesse a semente (seed), poderia prever todos os resultados futuros.

Neste sistema, os dados utilizam uma fonte de entropia do sistema operacional
— ruído de hardware, timing de eventos, e outras fontes físicas impossíveis
de prever. O resultado é estatisticamente equivalente a rolar dados físicos.

Apenas em um cenário extremamente raro de falha no gerador (quase impossível
em condições normais), o sistema recorre a um gerador pseudo-aleatório como
fallback de segurança.
```

**After — Dev doc (`docs/dev/weapons-dice.md`):**

```markdown
## Random Number Generation

The dice system uses `crypto/rand` for true randomness (hardware entropy).
Fallback to `math/rand/v2` on the extremely rare case of reader failure.

See: `internal/domain/entity/die/die.go`

The `Roll()` method generates a value in `[1, N]` via `crypto/rand.Int()`.
If `rand.Int()` returns an error, it falls back to `rand.IntN()` from
`math/rand/v2` (which is auto-seeded and not predictable, but technically
pseudo-random).
```

> **Note:** This is a good example of enriching game docs with curated content that players find interesting — true randomness is a differentiator worth highlighting. Look for similar opportunities where a technical detail translates into something players appreciate knowing.

---

#### Example 2: Skills — Manager and Joint Skills

**Current (`docs/game/pericias.md`) — Mixed:**

```markdown
## Gerenciador de Perícias (Manager)
O `Manager` centraliza o acesso a todas as perícias do personagem:
- **Init** — inicializa o mapa de perícias (só pode ser chamado uma vez)
- **Get** — busca uma perícia pelo nome; prioriza perícias conjuntas
- **IncreaseExp** — insere experiência e dispara a cascata
- **AddJointSkill** — registra uma perícia conjunta (deve estar inicializada)
- **GetValueForTestOf** — retorna o valor de teste incluindo buffs
- **GetSkillsLevel** — retorna o nível de todas as perícias em um mapa
```

**After — Player-facing (`docs/game/pericias.md`):**

```markdown
## Perícias Conjuntas e Prioridade

Algumas classes de personagem possuem **perícias conjuntas** — perícias
especiais que agrupam duas ou mais perícias comuns em uma única progressão.

Quando seu personagem possui uma perícia conjunta, toda vez que ele usar
qualquer uma das perícias que fazem parte desse grupo, o sistema
automaticamente utiliza a conjunta. Isso significa que treinar qualquer
uma delas beneficia todo o grupo de uma vez.

**Exemplo prático:** Se seu personagem tem a perícia conjunta "Combate
Corpo a Corpo" (que agrupa Empurrar e Agarrar), toda vez que ele empurrar
ou agarrar um oponente, a experiência vai para a conjunta — e ambas as
perícias se beneficiam igualmente.
```

**After — Dev doc (`docs/dev/character-sheet/skills-proficiencies.md`):**

```markdown
## Skill Manager Lookup Priority

When resolving a skill by name, the Manager checks joint skills first,
then falls back to common skills. This ensures joint skill XP routing
is transparent to callers.

Flow: `Manager.Get(name)` → check `jointSkills[name]` → check `skills[name]`

Joint skills must be initialized (`Init()`) before being added to the
Manager. They hold references to their component common skills and
multiply XP propagation by the component count during cascade.

See: `internal/domain/entity/character_sheet/skill/manager.go`
```

> **Note:** The player-facing rewrite for this example is acceptable but not ideal — the concept of joint skills is inherently complex to explain in simple terms. During implementation, invest more effort in finding the right language for this section. Examples 1 (dice) and 3 (cascade) are excellent and should be used as the quality bar.

---

#### Example 3: Cascade Upgrade Flow

**Current (`docs/game/ficha-de-personagem/experiencia.md`) — Mixed:**

```markdown
## Upgrade em Cascata (Cascade Upgrade)
O mecanismo central de progressão. Quando XP é inserido em uma perícia:
1. A **perícia** recebe o XP
2. O **atributo** associado recebe o XP
3. A **habilidade** associada recebe o XP
4. A **experiência do personagem** recebe o XP
5. Todos os **status** são recalculados
```

**After — Player-facing (`docs/game/ficha-de-personagem/experiencia.md`):**

```markdown
## Progressão em Cascata

O sistema de HxH RPG utiliza uma mecânica chamada **cascata**: toda
experiência que você ganha em uma ação específica se propaga para cima,
fortalecendo o personagem como um todo.

**Como funciona na prática:**

Imagine que seu personagem pratica escalada (perícia Acrobacia). Essa
experiência não fica isolada — ela também fortalece:

1. O **atributo Flexibilidade** (ao qual Acrobacia está vinculada)
2. A **habilidade Físicos** (que governa todos os atributos físicos)
3. O **nível geral do personagem**
4. As **barras de status** são recalculadas (HP, SP podem aumentar)

Isso significa que treinar qualquer aspecto do seu personagem contribui
para sua evolução geral. Não existe treinamento "perdido".
```

**After — Dev doc (`docs/dev/character-sheet/experience.md`):**

```markdown
## Cascade Upgrade Flow

The cascade is the core XP propagation mechanism. It crosses 4 packages
in a single call chain:

```
skill.CascadeUpgradeTrigger(values)
  → attribute.CascadeUpgrade(values)
    → ability.CascadeUpgrade(values)
      → characterExp.CascadeUpgrade(values)
```

Each layer:
1. Calls `exp.IncreasePoints(values.GetExp())`
2. Records its state in the `UpgradeCascade` collector struct
3. Invokes the next layer's `CascadeUpgrade`

After cascade completes, `status.Manager.Upgrade()` recalculates HP/SP/AP.

Key interfaces: `ICascadeUpgrade`, `ITriggerCascadeExp`
See: `internal/domain/entity/character_sheet/skill/skill.go` (entry point)
See: `internal/domain/entity/character_sheet/experience/experience.go`
```

## Success Criteria

- [ ] `docs/dev/` files created with focused technical detail (flows, rationale, non-obvious relationships)
- [ ] `docs/game/` files refactored to be developer-reference-free and read as a rulebook
- [ ] Developer footers present in game docs that have technical counterparts
- [ ] `docs/dev/overview.md` enriched with cross-references to domain docs
- [ ] A non-developer can read any game doc and understand the rules fully
- [ ] A developer can trace any cross-package flow via dev docs without guessing
- [ ] Game docs include curated "curiosity" content where technical details translate into something players appreciate (e.g., true randomness in dice)
- [ ] No mechanical repetition of what the code already communicates clearly
