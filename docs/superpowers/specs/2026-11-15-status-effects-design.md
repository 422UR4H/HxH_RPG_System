# Status effects — product design

> English version of `2026-11-15-status-effects-design.pt-br.md`. The PT-BR file is the one the
> product owner reviews; in case of conflict, **the PT-BR version wins**.
> Product spec, born from the brainstorm of 2026-11-07 to 2026-11-15 (product owner + Claude as PO).
> Full source of the decisions, with who decided each one: `2026-11-07-status-brainstorm-notas.md`
> (same folder). In case of conflict, **this document wins**; the notes keep the history and the examples.
> Markers: **[owner]** = decided by the owner; **[Claude]** = Claude's decision, delegated, to be reviewed.

## 1. Intent

The master of a System X System table must not be overloaded, neither in their head nor in clicks.
Today the combat engine computes hit, defense and damage, but **nothing happens after the damage**:
nobody ends up Stunned, Prone, Bleeding or Unconscious, and HP stops at zero. This project creates
the **status effects**: what the character suffers, what that blocks, how it shows up and how it ends.

It is also the groundwork for the next task (skill/attribute tests requested by the master): rolling
a test and having nothing happen is incomplete, and so is a status without a visible test. [owner]

### Guiding lines (same hierarchy level) [owner]

1. **Reality.** The system represents reality. A rule that makes physical, chemical and biological
   sense stays. But it is an RPG model: where fidelity would scale the complexity, the middle ground
   wins.
2. **Canon.** The system is canonical with Hunter x Hunter (official translation of the physical
   volume). When canon and reality conflict, weigh case by case: coherence with the existing
   canonical mechanics, reality in the Physical/Mental Abilities, canon in the Spiritual/Mental ones,
   the middle ground and other factors.

Both lines must be written into the game rules (principles section in `docs/game/`).

### Design principles

- **Few generic, combinable statuses**, like the actions: the narration gives the detail. The master
  who reads the rules must not find an endless list. [owner]
- **A status fits in one sentence** that the master remembers. Extra layers (level, number, faces)
  only where the source work or reality demands them. [owner]
- **The system proposes, the master disposes.** Everything the system applies, the master adds,
  swaps or removes; and they can put bias and a modifier on any test. They can do almost anything, as
  long as they do not break the cohesion of the rules. [owner]
- **The status says the state, not the quantity.** There is one Bleeding, never two: sources of the
  same status add up in its number; different statuses coexist. [owner]
- **Do not promote resemblance to law.** Where two statuses look alike, this document says what each
  one does; do not extend a rule by symmetry.

## 2. Goals, non-goals and anti-goals

**Goals**
- Catalog of status effects with mechanics computed by the system (§5).
- Resistance tests that the system rolls by itself, visible in the turn (§6).
- Negative HP, death and the critical/critical-failure rule (§7).
- The master applying, editing and removing statuses by hand, with a scene nickname (§8).
- Visibility per recipient and a history consistent with it (§9).
- Statuses that last outside battle and across sessions (§10).
- Combat game docs and adjustments to the character sheet docs (§12).

**Non-goals (left out, on purpose)**
- **Fatigue** — on standby; the owner will come back with the idea (§13). Stamina remains without effect.
- **Tests that the master requests** (batch, individual, chain of tests) — next brainstorm and spec.
  This design needs to serve them.
- **Exits and entries that depend on an action with a test** [owner agreed, 2026-11-15]: breaking
  free from Restrained, first aid, reviving, putting out the fire, getting up, and the resistance
  test when the source is the master (they would have to give the difficulty). They come with the
  tests spec. **Until then the master removes these statuses by hand** and types in the number of a
  Paralyzed or Dazed that they apply. The rules are described here (§5.3) for that spec to
  implement. The **automatic** tests (§6) come in now.
- **Movement** (jump, roll, sneak, distance) — the owner will redesign it. What depends on it is
  marked as provisional.
- **Nen abilities and items** — only the basics and TODOs for the Nen migration (§13).
- **Visuals** — a session in Claude Design; here only what needs to appear and where (§11).
- Mapping anatomy (which arm, which hand); recovery through training/rest (last feature of v1).

**Anti-goals** (would be a failure even if it works)
- The master having to confirm each status with a click, or fill in a long form.
- A player finding out, through a test, the history or an icon, a status they were not supposed to see.
- A table of "so many meters = so much damage" or any rule closed in a table where the system uses
  a parameter and a test. [owner]
- The status list growing through nicknames: a nickname is text, not a status.

## 3. Approaches

| | What it is | Trade-off |
|---|---|---|
| 1. Fixed catalog | each status hand-written in the code, with its own fields | simple; every new status (Nen, items) requires code |
| **2. Effect pieces (chosen)** | small set of basic effects; each status is a combination declared in one single place | more initial work; new status = new combination |
| 3. Free labels | master's text, no computation | discarded: automates nothing |

**Choice: 2**, Claude's decision approved by the owner ("go ahead"). It is the design of the actions
(few elements that combine), the ladder Paralyzed ⊂ Stunned ⊂ Unconscious comes out of the pieces
themselves, and future abilities and items only combine what exists. **Limit:** the first delivery
has only the pieces that the statuses in this document use; do not create a piece for the future.

## 4. Vocabulary [Claude, after the owner's doubt]

- **Status bars**: HP, SP, AP. Nothing changes.
- **Status effects**: this document. At the table, "status". In code, `StatusEffect` (does not
  collide with the `status` package of the bars).
- **Master's condition**: remains the master's adjustment on a roll (bias, modifier).
- **Nickname**: scene name of an applied status ("Enjoado" (Nauseous), "Braço direito quebrado" (Broken right arm)).
- **Action domains**: attack, defense, movement, dodge (those of the actions).
- **Dazed** is `stupefied` in code and on the wire (contract `docs/dev/api/match-status-effects.md`).
- **Only rolls when it can fail** [owner]: if even a critical failure would pass, the system does not roll.
- Skill names in PT per `docs/game/glossario.md`; see §12 for the Strength ones. In code:
  Accuracy = Precisão, Sneak = Deslizar, Tact = Tato, Breath = Respiração, Heal = Cura,
  Tenacity = Tenacidade, Vitality = Vitalidade, Defense = Defesa, Acrobatics = Acrobacia.
- **Resilience is a mental attribute, with no skill:** the test is an attribute test and rolls with
  the **attribute's power**. [owner: mental attributes have no skills]
- **The three faces of Tact roll with the same value** (the Tact skill); the face only decides
  whether the status weighs on that test. [Claude]

### How penalties add up

- **Advantages and disadvantages ACCUMULATE** [owner, 2026-11-15 — corrects Claude's proposal]. Each
  source counts: Weakened + nullified Vision = two disadvantages.
- **Advantage and disadvantage cancel one by one** [Claude]: two disadvantages and one advantage =
  one disadvantage.
- **Numbers add up** (−2 and −3 = −5).
- The existing rule applies: bias on a passive test makes it rolled.
- **How each level rolls — one extra D10 per level** [owner, 2026-11-15: "it's the face of this
  system"; "for now"]: the test rolls 2 + N dice; with advantage the **2 best** stay, with
  disadvantage the **2 worst**. Critical = the two remaining dice are 10; critical failure = both are
  1. It keeps the stability that made the system prefer 2 D10 over the D20, and with double advantage
  the critical chance (5.2%) already reaches that of the D20, without the 5% critical failure — the
  case of Gon stealing Hisoka's badge comes out of this, with no special roll.
- Rejected: a second whole roll of 2 D10 with 1 D20 at level 2 (rows B and D20 of the table, which
  stay as a record of the comparison).

  | Scheme | Mean | Std. dev. | Critical | Critical failure | ≥ 8 | ≥ 11 | ≥ 14 | ≥ 17 |
  |---|---|---|---|---|---|---|---|---|
  | 2 D10 normal | 11.0 | 4.1 | 1.0% | 1.0% | 79% | 55% | 28% | 10% |
  | A — advantage 1 (3 D10) | 13.5 | 3.7 | 2.8% | 0.1% | 93% | 79% | 53% | 23% |
  | A — advantage 2 (4 D10) | 15.0 | 3.3 | 5.2% | 0.01% | 98% | 90% | 70% | 37% |
  | A — advantage 3 (5 D10) | 16.0 | 2.9 | 8.2% | ~0% | 99% | 95% | 81% | 49% |
  | A — disadvantage 1 | 8.5 | 3.7 | 0.1% | 2.8% | 57% | 29% | 11% | 2% |
  | A — disadvantage 2 | 7.0 | 3.3 | 0.01% | 5.2% | 40% | 15% | 4% | 0.5% |
  | A — disadvantage 3 | 6.0 | 2.9 | ~0% | 8.2% | 28% | 8% | 1.5% | 0.1% |
  | B — advantage 1 (best of two) | 13.3 | 3.3 | 2.0% | 0.01% | 96% | 80% | 48% | 19% |
  | B — disadvantage 1 (worst of two) | 8.7 | 3.3 | 0.01% | 2.0% | 62% | 30% | 8% | 1% |
  | 1 D20 | 10.5 | 5.8 | 5.0% | 5.0% | 65% | 50% | 35% | 20% |

  (Critical = the two remaining dice are 10; critical failure = both are 1.)

## 5. The catalog

### 5.1 The effect pieces [Claude]

| Piece | What it does |
|---|---|
| **Cannot act** | blocks sending actions; the ones in the queue are removed |
| **Cannot react** | no reaction, not even the passive ones (reflex dodge, default defense) |
| **Locked domain** [domains] | that action domain cannot be used |
| **Domain warning** [domains] | the system warns the master when an action uses the domain; they decide |
| **Restricted reaction** [reaction → forbidden / with disadvantage / rolled only] | changes what the guard allows (table 5.4) |
| **Test penalty** [scope; disadvantage and/or number] | scope = Ability, skill, Tact face, action domain or movement speed |
| **Test fails automatically** [scope] | the test is not even rolled |
| **Does not move** | stays on the square (or only on the trajectory) |
| **Loses HP** [N per round] | continuous damage; outside battle, counted by the story clock (1 round = 1 s) |
| **Recovery** [skill; points] | the system rolls every round and subtracts; at zero, the status leaves |
| **Trigger test** [trigger; skill; failure → status] | e.g. balance when hit |
| **Conduct** [text; resistible or not] | reminds the master when the character acts |
| **Control** [who] | another declares the actions |
| **Counter** | number that rises with time (drowning) |

### 5.2 Set diagram [owner's request; Claude's drawing]

A status inside another contains all of its pieces. What does not fit is a separate domain in the
universe set.

```
┌─────────────────────────────── STATUS EFFECTS ────────────────────────────┐
│                                                                           │
│ CONSCIOUSNESS AND BODY                      MIND (will)                   │
│ ┌─ Unconscious ───────────────────────┐     ┌─ Controlled ─────────────┐  │
│ │ no consciousness                    │     │ another declares actions │  │
│ │ ┌─ Stunned ───────────────────────┐ │     │ ┌─ Compelled ─────────┐  │  │
│ │ │ ┌─ Paralyzed ──────┐┌─ Dazed ─┐ │ │     │ │ imposed conduct     │  │  │
│ │ │ │ locked body      ││ mind    │ │ │     │ └─────────────────────┘  │  │
│ │ │ │ (domains)        ││ locked  │ │ │     └──────────────────────────┘  │
│ │ │ │  ┌─Restrained─┐  ││         │ │ │                                   │
│ │ │ │  │ does not   │  ││         │ │ │     TESTS                         │
│ │ │ │  │ move       │  ││         │ │ │     ┌─ Weakened ───────────────┐  │
│ │ │ │  └────────────┘  ││         │ │ │     │ Body│Mind│Spirit         │  │
│ │ │ └──────────────────┘└─────────┘ │ │     └──────────────────────────┘  │
│ │ │ + all senses nullified          │ │                                   │
│ │ └─────────────────────────────────┘ │     LIFE (HP only)                │
│ └─────────────────────────────────────┘     ┌──────────────────────────┐  │
│                                             │ Bleeding   │ Burning     │  │
│ BODY PART                                   │ Freezing   │ Poisoned    │  │
│ ┌─ Injured: unusable ─────────────────┐     └──────────────────────────┘  │
│ │ ┌─ Injured: hurt ────────────────┐  │                                   │
│ │ │ ┌─ Slowed ──────────────────┐  │  │     AIR                           │
│ │ │ │ (movement speed only)     │  │  │     ┌─ Suffocating ────────────┐  │
│ │ │ └───────────────────────────┘  │  │     │ breath → drowning        │  │
│ │ └────────────────────────────────┘  │     └──────────────────────────┘  │
│ └─────────────────────────────────────┘                                   │
│                                             GUARD (position)              │
│ SENSES (each of the five)                   ┌───────┐┌────────┐┌────────┐ │
│ ┌─ nullified ─────────────────────────┐     │ Prone ││Airborne││ Open   │ │
│ │ ┌─ impaired ─────────────────────┐  │     │       ││        ││ guard  │ │
│ │ └────────────────────────────────┘  │     └───────┘└────────┘└────────┘ │
│ └─────────────────────────────────────┘                                   │
│                                                                           │
│ PRESENCE: Hidden (whoever has not noticed it does not see it anywhere)    │
│ ON HOLD: Fatigue                                                          │
└───────────────────────────────────────────────────────────────────────────┘
```

Readings of the diagram, for the owner to confirm in the review:
- **Stunned = Paralyzed (whole body) + Dazed + all senses nullified.** Dazed is the "mind locked"
  half of the stun; Paralyzed, the "body locked" half. They differ in origin and in skill.
  [Claude — it was born from the drawing]
- **Restrained** is the "does not move" piece that the movement Paralyzed also has; Restrained keeps
  all the reactions.
- **Slowed** is the movement part of the hurt Injured.
- **Controlled contains Compelled**: whoever controls everything imposes any conduct.
- The three of the guard do not contain one another: they are independent positions. Prone does
  **not** imply Open guard. [owner]

### 5.3 The statuses

Columns: what it does (the master's sentence) · how it enters · how it ends.

**Consciousness and body**

| Status | The sentence | Enters | Ends |
|---|---|---|---|
| **Paralyzed** | The marked domains do not obey; the mind is intact. | elemental source (shock, poison) or master; marks domains (default: the four) | Recovery with **Tenacity** every round |
| **Dazed** | The mind locks: does not act, does not react, does not use abilities. | mental source (Hisoka's insanity on Kastro, dread) | Recovery with **Resilience** every round |
| **Stunned** | Out of action: does not act, does not react, does not use abilities or the senses; shaken consciousness. | damage received: **Tenacity** against the **final damage** (§6.2) | Recovery with **Tenacity** every round |
| **Unconscious** | Besides being Stunned, is unconscious. | HP below 0: **Vitality** (§7); drowning | recovers when the cause ceases — HP back to ≥ 0, revival, or the master [Claude] |
| **Restrained** | Does not move; acts and reacts normally (disadvantage at the master's discretion). | grapple, net, wires, rubble, ice | declared action (see below) |

- **Recovery (Stunned, Paralyzed, Dazed)** [owner]: what was missing in the entry test (difficulty −
  result) becomes the status points. **At the start of each round** the system rolls the skill,
  always and by itself, and subtracts the result. At zero, it leaves and already acts in that round.
  What **was left over** from the test that zeroed it enters as a bonus on the bar of the **first
  action** that follows; in a combined action (both bars), **half for each, rounded down**
  (provisional: action and movement will be redesigned). E.g.: had 5, rolled 18 → leaves with +13.
- A new stun on someone already stunned **adds** to the points. [owner]
- While Stunned or Unconscious, **every sense test fails automatically**. [Claude]
- The **Paralyzed** can use an ability that does not depend on the locked body; the Stunned cannot.
  TODO Nen.
- Outside battle none of them is removed by the scene change: they remain in force and are resolved
  by the story clock (see §10, "Time outside the round").
- **Leaving Restrained** [owner]: by someone → contest against the **Grab** of whoever holds; by
  something → the difficulty of the thing. It is an **action declared by the player**; the clean
  interface offers **Force** and **Sneak** and the option of assembling another action freely
  (jumping to throw the opponent on the ground, Grab to crush or break). The player picks the skill,
  in the mechanics; the narration has to reflect the choice — if it does not, the master penalizes,
  nullifies or adjusts the skill before the tests collide. **This holds for every action.** Whoever
  holds can let go; an ally can free.

**Body part**

| Status | The sentence | Enters | Ends |
|---|---|---|---|
| **Injured** | The marked domains are hurt (disadvantage) or unusable (the system warns the master). | master or source; the narration says the part | only with healing (see below) |
| **Slowed** | Penalty on movement speed. | master or source | by the cause |

- Injured [owner]: only one per character; new injuries add domains. Reference: arm ≈ attack +
  defense; leg ≈ movement + dodge; hand ≈ attack. The system does not map anatomy.
- **Crosses the end of battle.** Immediate first aid does **not** give back the use. It leaves with a
  healing ability or item (rule in their configuration, when they exist), healing after the match or
  the future recovery; **until then, the master decides when it leaves**. What never heals (Kite's
  arm) is a change to the character sheet, not a status.
- Slowed: speed only (walks later, escapes worse). **Distance open**, until movement. Forbidding dash
  or jump belongs to Injured (movement unusable).

**Senses**

| Status | The sentence | Enters | Ends |
|---|---|---|---|
| **Impaired sense** (Vision, Hearing, Smell, Tact, Taste) | Impaired: disadvantage and/or number on the sense's tests. Nullified: the test fails automatically. | master or source; resists with the origin's skill | by the cause: three exits |

- **Three exits** [owner]: (1) while the cause lasts (darkness, smoke, blindfold) — no test;
  (2) until the body recovers (flash, blast, sand) — Recovery every round with the origin's skill;
  (3) until treated (pierced eye) — it is an injury.
- **Strong flash = Vision nullified, not a stun**; it leaves by beating **Tenacity**. [owner]
- **Vision nullified → disadvantage on attack and on defense.** Future: training/ability removes it
  (En). [owner]
- One status per sense; the level goes up from impaired to nullified, it does not duplicate.
- **Tact has three faces** [owner]: touch, balance, preemption. The master freely marks 1, 2 or all 3.
  Only the affected **balance** weighs by itself on attack, defense, dodge and movement, and
  triggers a **balance test when taking a blow and on any movement**: failed → Prone (nullified →
  fails automatically). Sneaking and rolling do not knock down: rule by type of movement, to be
  closed with movement. The other senses have no faces.

**Tests**

| Status | The sentence | Enters | Ends |
|---|---|---|---|
| **Weakened** — at the table: **Weakened body**, **Weakened mind**, **Weakened spirit** [owner] | Disadvantage on the tests of the marked Ability (Physical, Mental or Spiritual). | master or source | by the cause (the three exits of the senses) [Claude] |

- Only one level; the number is optional (the master swaps the disadvantage for −N, or adds). There
  is no "nullified Weakened". [Claude, after the owner's "too complex"]
- **Weighs on every test of the Ability**, including resistances (Tenacity, Heal, Vitality, Defense)
  and Sense. [owner]
- One Weakened per Ability; they coexist.
- Spiritual: Nen of Flame and of Consciousness (Focus, Willpower, Self-knowledge, CoA, MOP, AOP)
  **and the other Nen tests** (principles, techniques, categories, abilities) — TODO where Nen does
  not exist yet. [owner]

**Life (HP only)** — "damage" is a word of HP; SP and AP are not life. [owner]

| Status | The sentence | Enters | Ends |
|---|---|---|---|
| **Bleeding** | Loses N HP per round until helped. | HP below 0 (**Heal**, §7); damage or ability that bleeds | only with first aid |
| **Burning** | Loses N HP per round until put out. | **Tenacity** against the source, once | someone puts it out (rolling on the ground, water, smothering) |
| **Freezing** | Loses N HP per round while in the cold. | same | leaves the cold or warms up |
| **Poisoned** | Loses N HP per round. | same | antidote, treatment or the poison's time |

- Fire, cold and poison: Tenacity enters **only at the entry**; afterwards, no roll per round. [owner]
- **Poison is a source, not a status**: a poison can put Poisoned, Paralyzed, Weakened, Vision
  nullified, Unconscious. "Poisoned" is only the damage per round. [owner]
- Bleeding [owner]: 1 per round for having fallen below 0; by ability, whatever it defines. Sources
  add up. Stopping the bleeding (the unconscious rolling **Heal** every round, or first aid): **each
  ten of the result removes 1** from the bleeding per round. First aid, skill chosen by the player:
  **Legerity** (tourniquet; default in battle), **Accuracy** (stitching, bandage), **Force**
  (pressing). Medical kit: large bonus.

**Air**

| Status | The sentence | Enters | Ends |
|---|---|---|---|
| **Suffocating** | Holds the breath with Breath; when it fails, starts to drown. | no air (water, strangling, gas) | breathes again |

All [owner], skills [Claude]:
1. **Breath-holding:** resists with **Breath** against a difficulty that rises **1 every 10
   seconds**. Does not roll while it cannot fail. Each action that spends SP raises the difficulty at
   once and asks for the roll — **gap:** SP spending does not exist yet.
2. **Failed:** the **drowning bar** appears, at 0, rising 1 every 10 s. Only rises; no maximum.
   Appears only in combat, not on the character sheet. **Does not take HP.**
3. **Flailing:** conscious, resists with **Vitality** against the bar's value. Failed → Unconscious;
   the bar continues.
4. **Saved conscious:** the bar disappears; HP intact. **Saved unconscious:** the bar keeps rising
   until someone **revives** — test by the rescuer (default **Force**; others of first aid) against
   the bar's value.
5. **Death:** 15 minutes of story time unconscious without revival (≈10 real minutes, biased upward
   because they are hunters). Adjustable parameter.
- Calibrate later with canon: Morel (half an hour to an hour submerged), Gon (15 min of breath).

**Guard (position)**

| Status | The sentence | Enters | Ends |
|---|---|---|---|
| **Prone** | On the ground: only rolls or slides; reflex and repel with disadvantage. | balance failure, blow, the player throws themselves down, master | gets up (cost: with the movement) |
| **Airborne** | Does not change the trajectory; no reflex and no escape. | during the jump (future) | on landing |
| **Open guard** | Careless: no reflex, no repel, defense only rolled, escapes with disadvantage. | master (or system, in the future) | end of battle (provisional) |

- Airborne [owner]: the system applies it while the jump lasts. It persists when the jump test fails
  and it does not reach the target square in the turn. **Does not apply to the dash.** The sentence
  in `reacoes.md` about the dash ("is in the air… stays exposed") is **wrong** and must be
  corrected: whoever already dashed and is attacked hardly moves again; can evade, repel, parry or
  defend.
- Open guard: TODO — in the future the player keeps the guard through a button ("enter guard",
  connects with `Stance`); then the system stops removing it at the end of battle.
- Unconscious suggests Prone by default; carried or seated, Prone is removed. [owner]

**Mind (will)**

| Status | The sentence | Enters | Ends |
|---|---|---|---|
| **Compelled** | Carries an imposed conduct; the system reminds the master when they act. | fear, ability, master | by the cause |
| **Controlled** | Another (player or master) declares their actions. | manipulation ability | by the ability |

- Compelled [owner/Claude]: the system **does not block**. The master asks for a test (**Resilience
  or Willpower**, by their reading), marks **irresistible** (the test fails automatically —
  manipulation ability) or lets it pass. Fear is Compelled. Illumi's needle on Killua is an
  irresistible Compelled with a condition ("flee when in danger"), not direct control.
- **There is no provocation mechanic**: one provokes through roleplay. [owner]
- Controlled: **only the basics** — the status exists and says who controls. Declaration by another
  person arrives with the abilities.
- Confused and Hallucinating are not statuses: they are combinations. [owner]

**Presence** [owner, 2026-11-15]

| Status | The sentence | Enters | Ends |
|---|---|---|---|
| **Hidden** | Whoever has not noticed it does not see it anywhere at the table. | **Stealth** against the others' perception; or the master | is noticed, reveals itself by acting openly, or the master |

- Applies to NPCs (the enemy in the bush drawn on the map background — the system does not need to
  know there is a bush) and to player characters.
- For whoever does **not** see it, it vanishes from everything: **token on the map, action and
  movement bars of the round/turn, character list, and the actions it takes while hidden** (hidden
  history, with the same rule as §9: appears when revealed).
- **Who sees** is a list, as in the other statuses: **it will be granular** (one player notices and
  the others do not). **In this delivery it uses the default of the others:** the master and the
  character's owner see; when revealed, everyone sees. Model it as a list already.
- The Stealth test against perception (Vision, Hearing… of each observer) is a contest between
  characters: it comes with the tests spec. **Until then the master hides and reveals by hand.**
- Unlike the other statuses, visibility here is the effect itself: what is hidden is the
  **character**, not only the status.

### 5.4 What each status allows in the guard

| | Act | Move | Reflex dodge | Evasion | Defense | Repel | Escape |
|---|---|---|---|---|---|---|---|
| **Airborne** | ✓ | trajectory only | ✗ | ✓ | ✓ | ✓ | ✗ |
| **Open guard** | ✓ | ✓ | ✗ | ✓ | ✓ rolled only | ✗ (parrying counts as armed defense) | ✓ with disadvantage |
| **Prone** | ✓ | only roll / slide | ✓ with disadvantage | ✓ | ✓ | ✓ with disadvantage | ✓ only by rolling or sliding, no extra penalty |
| **Restrained** | ✓ | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ |
| **Paralyzed** | per the domains | per the domains | per (dodge) | per (dodge) | per (defense) | per (defense) | per (movement + dodge) |
| **Dazed / Stunned / Unconscious** | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ | ✗ |

Every automatic disadvantage in this table is the default; the master changes it. Rolling and
sliding do not exist yet as movements: the Prone row is completed with movement.

### 5.5 Combinations that the system makes by itself [Claude]

- Unconscious → applies Prone along with it (removable).
- Failure on the balance test → Prone.
- Failure on the drowning Vitality → Unconscious.
- HP below 0 → unconsciousness and bleeding tests (§7).
- Airborne + (Stunned or Unconscious) → the system **warns** the master (fall); does not compute by
  itself.
Nothing beyond that is automatic: Stunned does not knock down on its own.

## 6. Tests that the system rolls by itself

### 6.1 How they enter and who sees

- An automatic test that fails → the status **enters directly**, with no confirmation. [owner]
- **Inside a turn:** it appears in the turn's calculation, along with the damage, which is already
  the master's alone until closing. The master swaps, removes or adds a condition before closing;
  once closed, it holds for the table.
- **Outside a turn** (recovery per round, breath, poison): enters at once; the master undoes it.
- **Test visibility = status visibility** (§9): master always; the character's owner when the turn
  closes; the others only if the status is public.
- **Test card** [owner]: `Tenacidade 12 contra 20 - Atordoado: 8` (Tenacity 12 against 20 - Stunned:
  8) — skill, result, difficulty, status and its number. Without the word "reserva" (reserve).
  Attached to the action that generated it (today the action is text; the card comes with the
  design).

### 6.2 The order of a blow [owner]

1. The hit lands.
2. The defense reduces the damage.
3. The **final damage** comes out of the HP. **The master must see the reduction** (raw → defense →
   final); whoever implements checks what the screen already shows.
4. **Stun test: Tenacity against the final damage.** Failed → Stunned with the difference. Tenacity
   and not Defense, so that Defense does not count twice in the same blow.
5. If the HP ended below 0: §7.
6. If the target has the balance affected: balance test (Tact) → Prone.

### 6.3 Theme → skill (goes to the game docs)

| Theme | Skill |
|---|---|
| Parrying the blow (reducing damage; the fall) | Defense |
| Enduring what got through: stun, shock, fire, cold, poison, flash | Tenacity |
| Blood: not starting to bleed, stopping the bleeding | Heal |
| Staying conscious (HP below 0, drowning) | Vitality — the body; Willpower stays with exhaustion (Fatigue) and fear. The master can swap the test's skill. [owner agreed] |
| Breath-holding | Breath |
| The mind: dread, disgust, shock | Resilience |
| Fear and imposed conduct | Resilience or Willpower (the master reads the scene) |
| Balance | Tact (balance face) |

## 7. Negative HP, death and critical

- **Max negative HP = 10 + Vitality level + Resistance value + Physical Ability bonus**, rounded
  down. [owner; rounding Claude] E.g. (parity fixture): HP 20 → 14 or 15; HP 30 → 17; HP 51 → 21;
  HP 75 → 25.
- Damage that goes past zero keeps going down. **Passed 1 beyond the negative maximum → dies.** A
  blow greater than current HP + max negative HP kills on the spot.
- **Unconsciousness:** whenever HP goes further below 0 (blow or bleeding), **Vitality against the
  exact negative value** (at −4, difficulty 4).
- **Bleeding when passing 0:** **Heal** against **last damage received + |negative HP|** (took 8 and
  ended at −5 → 13). Failed → Bleeding 1 per round. Whoever was already bleeding makes an **easier**
  test (how much: from the builder, parameter); failed, adds 1.
- Outside the round, HP loss and the unconscious's tests run by the story clock.
- **Critical and critical failure** [owner]:
  - **What generates a critical is the test**, never the damage die: both dice of the test at 10 =
    critical; both at 1 = critical failure. A 2 D12 weapon that rolls 12 and 12 deals 24 damage and
    that's it.
  - **In any test** (hit, dodge, defense, resistances): critical → (dice + test value) × 2;
    critical failure → ÷ 2, rounded down.
  - **On the hit**, in addition: critical **doubles the FINAL damage** — what comes out of the HP,
    after all reductions. Final damage 0 stays 0. Also applies to walls and (future) objects. On a
    critical hit both things apply together: the test total doubles and the final damage doubles.
    [owner confirmed]
  - Today the code only sets the flags. **The rule lives in `MatchRules`** (the match's domain
    layer), **never** in the `die` entity nor in the dice rolls: those entities do not know rules of
    the layers above.
- **Death** [owner]: is **permanent**; the **master can undo it**; the **token stays on the map as a
  body**, can be carried, and can be revived or controlled by an ability (e.g. corpse manipulation)
  — TODO Nen. **Everyone sees** that it died. It appears the way the dead NPC already appears in the
  campaign: **dead tag and darkened card**. (Today an NPC "forced as dead" is left with HP at the
  maximum, which makes no sense: the one dead by the rule keeps the HP where it died.)
- **The master always sees the death before the table** [owner]: inside the turn it appears in the
  calculation, which is only theirs until closing, and they can intervene. Outside the turn, see
  §10. "Dead" is a public status effect that contains Unconscious and has no exit of its own. [the
  way it is a status is Claude's]
- This changes character sheet formulas (minimum of the HP bar): regenerate the back/front parity.

## 8. The master applies by hand [owner]

1. **One click in the simple case:** character → status in a short list → apply. The rest is born
   with the default, and **every default is editable**.
2. **Only the field that the status has** (domains, faces, level, number, conduct). No single form.
3. **Several characters at once** (shortcut; the common path will be the batch test).
4. **Edit and remove** through the status itself.
5. **Nickname:** free field shown to the table next to the status, with **suggestions per status**.
   Narrative weight, not mechanical: the system does not interpret it. Document in `o-mestre.md`.
6. The master manually releases whoever is blocked (Stunned etc.), if they want.

**Suggested nicknames** [Claude, lean] (in Portuguese, as the UI is; glosses in parentheses):
Vision — Cego (Blind), Ofuscado (Dazzled) · Hearing — Surdo (Deaf), Zumbido (Ringing) · Tact
(balance) — Tonto (Dizzy), Vertigem (Vertigo) · Tact (touch) — Dormente (Numb) · Weakened body —
Enjoado (Nauseous), Febril (Feverish), Fraco (Weak) · Weakened mind — Abalado (Shaken), Confuso
(Confused) · Weakened spirit — Aura abalada (Shaken aura), Desanimado (Disheartened) · Paralyzed —
Em choque (In shock), Travado (Locked up) · Dazed — Em pânico (In panic), Horrorizado (Horrified) ·
Restrained — Agarrado (Grabbed), Enredado (Entangled), Soterrado (Buried), Pendurado (Hanging) ·
Compelled — Amedrontado (Frightened), Hipnotizado (Hypnotized), Em fúria (Enraged) (no
"Enfeitiçado" (Bewitched): there is no spell in the hunter world) · Hidden — Furtivo (Stealthy),
Emboscado (Ambushed) · Open guard — Desprevenido (Unprepared), Surpreso (Surprised) · Burning —
Em chamas (On fire) · Suffocating — Afogando (Drowning), Estrangulado (Strangled) · Injured — free
("braço quebrado" (broken arm)).

## 9. Who sees [owner]

- **Default by what can be seen from outside**, for NPCs and for player characters (there is PK):
  - **public:** Prone, Airborne, Restrained, Burning, Unconscious, Stunned, Suffocating and Open
    guard [the last two: Claude];
  - **hidden:** Poisoned, Freezing, Weakened, impaired sense, Injured, Slowed, Paralyzed, Dazed,
    Compelled, Controlled, Bleeding, Hidden [Freezing, Slowed, Paralyzed, Dazed: Claude — the owner
    reviews].
- **Hidden** = only the master, on an NPC; **the character's owner and the master**, on a PC. The
  master can set "master only" (the effect acts without the player knowing — Illumi's needle).
- The master makes it public whenever they want. **Table configuration:** PC statuses public by
  default.
- **Future, to anticipate in the model:** reveal to **specific players**; the same rule will serve
  open/locked doors. Model as a list of who sees; **no record = default**, so as not to spend
  database on the common case. The final form is up to whoever implements.
- **History:** entry, change and exit of statuses and the tests stay in the turn history and on the
  match page. What is private **does not appear to those who do not see it**, until it becomes
  public — even matches later; when revealed, the past appears. [owner; "the past appears" is
  Claude's]
- A private status weighs on the player's tests even without them knowing why.
- **The fog comes on top of all this** [owner, 2026-11-17]: a player only receives a character's
  public statuses while seeing it. Detail in §16.1, item 4.

## 10. Persistence and continuity [owner]

- **Any status, bar or effect holds for the character sheet in any campaign and match** [owner]: the
  Injured gained in one campaign shows up if the sheet plays in another. The table belongs to the
  sheet, not to the match. Negative HP is the sheet's own HP.
- Status effects exist **outside battle and across sessions**. **Its own table referencing the
  sheet**; do not fatten the sheet. Replaces the old note in `character_status.go` ("will not be
  persisted").
- **Leaving combat for roleplay changes nothing:** each status continues until its own exit. The end
  of battle only removes Open guard (provisional).
- Writing follows the house rule: inside an open turn nothing is written before closing; the status
  goes into the closing transaction.
- They apply to NPCs the same way.

### Time outside the round [Claude — review]

- **"End of battle"** = the master taking the table out of the contested turn (change of round mode
  to free turn). It is the only event that removes Open guard.
- Outside the contested turn the system **does not roll second by second**. When the story clock
  advances, each status is resolved **in batch**, for the time that passed:
  - **Recovery** (Stunned, Paralyzed, Dazed, sense recovering): uses the average value of the roll
    per second; if the time was enough, the status leaves. No roll.
  - **Loses HP and drowning counter:** before confirming a time advance with someone bleeding,
    poisoned, burning or drowning, the system **shows the master** how much each would lose and who
    would die or fall unconscious, and they confirm, adjust or treat beforehand. No killing someone
    silently in a ten-minute jump.
  - **Even with the clock running second by second** [owner]: the clock **pauses 1 or 2 seconds
    (rounds) before the character dies** and shows the master a message that they are dying. The
    message has a button to **make it public**, if the master wants. Simple rule; the exact form is
    up to whoever implements.
- Inside the contested turn, the round-by-round described in §5.3 applies.

## 11. Where it appears [owner]

- **On the map token:** nameless icons around the token; on select/click, the names expand.
- **In the actions panel** (bottom sheet on mobile): status of the target and of the character
  itself, along with the bars, the armor (if known) and the like. For master and player.
- **In the table's character list**, with number and nickname; for the master, edit and remove. The
  drowning bar appears here.
- **On the player themselves, with emphasis**, when they gain a status or cannot act because of it
  (sending is blocked while Stunned/Dazed/Unconscious and they need to understand why).
- **Stack above the piece** [owner, 2026-11-19, with the PD session]: bottom to top — the
  **piece**; right above it, the **statuses** (if any; layout of the PD session's "option C");
  above them, the **Health and Stamina bars** (Aura replaces Stamina when it exists), if visible;
  on top, the **action/reaction balloon**, if the character is acting or reacting (the balloon is
  not glued to the character). The bars are plain: the default bar color for the current value and
  the project's darkest color for the empty part. **Each player sees only their own bars by
  default; the master sees all and can reveal other players' bars to everyone.** Reactions will
  appear around the piece (later). ⚠️ **Gap for the back:** today Health and Stamina go only to the
  owner and the master (`character_hp_changed`); revealing a character's bars to other players is
  not in the contract yet. It comes with per-player visibility (§16.3, item 24).
- Follow UI/UX principles; this phase's visuals are decided with the PD session, without Claude
  Design (the project's human designer does the final prototypes later). Screen effects (darkening
  etc.) and the character sheet page outside the match: later.

## 12. Docs and character sheet adjustments

- **Combat rules** (`docs/game/combate/`, displayed elsewhere on the site, not in the sheet's "?"
  panel): status effects with the set diagram (§5.2), the theme → skill table, the guard table, the
  examples from the notes ("Exemplos para as docs de jogo" section) and the dash correction in
  `reacoes.md`. No height table.
- **System principles:** the two guiding lines (§1) and Biske's chart (it is in the notes, with the
  end of D near the middle of line 4) — the latter comes in when Fatigue comes in.
- **Master's doc:** applying by hand, nicknames, visibility.
- **Character sheet:** enrich the description of **Willpower** with the canonical meaning (the will
  to fight someone who seems stronger, without assuming one's own worst condition — Biske to
  Killua).
- **Strength skills** [owner]: **Force / Grab / Charge**. Force = push, pull, break open, press.
  Charge = carrying capacity **and** momentum: charged blow and accumulating speed beyond
  Acceleration (the brute, like Uvogin, who does not brake instantly). Renaming `Push→Force` and
  `Carry→Charge` requires a migration like the weapons one (20261113) and back/front parity. The
  mechanic of charged blow and charge attack belongs to the tests/actions task.

## 13. Reserved, on hold and provisional

**Fatigue — on standby.** Do not implement. What is standing for the return: its own status, set by
the system from SP (everything that worsens the physical condition takes SP); nothing above 50%;
negative Stamina like HP; Willpower to stay awake; the floor comes from the roll itself; bars of
different length per character; **no fixed modifier value**. Sources: Biske's chart and Morel's line
("from zero to 100… I must be at 40… no, 35"), in the notes.

**Fall — rule defined, trigger after movement.** Damage = height factor + what was missing at
landing (or − what was left over) − Defense; ≤ 0 does not hurt. Landing: **Acrobatics**. With no
chance to react: height − Defense. **Height factor is a parameter** (initial value 3 per meter),
never a table; the resistance will receive the Nen share (30 m jumps without damage).

**For the tests and collisions spec — negative damage returns to the attacker** [owner, 2026-11-15;
mandatory rule, do not implement here]: if the defense exceeds the damage (a punch of 20 against a
defense of 50 → −30), the excess **returns to whoever hit**, who reduces it with a test. Owner's
examples, with a test of 20 (value 10 + dice 10): excess 30 → receives 10; excess 40 → receives 20.
Afterwards, a stun test on what was received, and the limb that struck can gain a status (Injured,
numb). **Critical failure on the hit doubles the damage that returns** (10 → 20; 20 → 40); **critical
hit halves it** (10 → 5; 20 → 10). The owner suggested Tenacity to reduce, noting that some test
would repeat. In the examples, "excess 40" is the punch of 20 against a defense of 60 (owner
confirmed the reading). **Direction preferred by the owner (Claude's suggestion; close in the tests
spec):** treat what returns as a blow on the attacker — **Defense reduces, Tenacity decides the
stun** —, the same order as §6.2, without repeating a skill.

**For the tests spec — stacking tests** [owner, 2026-11-15]: the master must be able to **stack**
tests on the same stack (chain) of a test, including the automatic ones here. Does not come in now;
the owner has a draft and prefers to increment after statuses and tests are closed. The design of
the automatic tests (§6) must not prevent this: each test is an item of the action's stack.

**TODO for the Nen migration:** effect of Stunned/Unconscious on the aura (the aura may still
protect them); "broken flow" (active techniques fall); what continues without the user is each
ability's rule (materialization); Paralyzed using an ability; spiritual Weakened on Nen tests; En
removing the dependence on Vision; declaration of the Controlled's actions; abilities and items that
apply and heal statuses.

**Provisional until movement:** distance of Slowed; roll/slide/get up of Prone; Airborne on the jump;
exceptions of the balance test; split of the post-stun bonus.

**Other noted futures:** biotype (weights in the physical diagram); location map and volume of the
injured body; weighting to estimate one's own condition; configuration with real data (each
automatic roll is costly in it — hence few rolls per round).

## 14. What stays with the builder

- **Comments and TODOs** [owner]: whatever this implementation makes **contradictory** must be
  deleted or corrected (e.g. "will not be persisted" in `character_status.go`). **Do not** delete
  comments or TODOs outside this scope — the "NEVER remove TODO comments" rule exists because
  sessions were deleting others' TODOs. Right at the start, **rewrite that rule in `AGENTS.md`** in
  that sense.

- The technical form of the pieces, the table and the visibility list; English names in code
  (identifiers and test titles in English; comments and UI in Portuguese).
- New WS/REST contracts in `docs/dev/api/` and registration in `documentation-map.yaml`.
- The split into phases. Suggestion: (1) base — table, pieces, catalog, apply/edit/remove by hand
  with visibility; (2) critical, negative HP and death; (3) automatic tests at turn closing and
  effects on the rolls; (4) continuous damage and recovery outside the turn via the clock,
  suffocation; (5) front: display, history; (6) game docs and renaming of the Strength skills.
- **Table configuration** (PC statuses public by default): the match settings mechanism does not
  exist; for now it is a parameter in `MatchRules` with this document's default, no screen. [Claude]
- Parameters in `MatchRules`: height factor, suffocation pace (1 per 10 s), 15 min to death,
  discount of the test for whoever is already bleeding.
- Check in the code, before planning: what the master's screen shows today of the reduction by
  defense; where HP is clamped at zero; how the master's "condition" enters a new roll.
- **Process between implementation and spec** [owner, 2026-11-19]: each implementation PR carries,
  in Portuguese, a **"Decisões tomadas"** (decisions taken) section — the owner's and the
  implementer's. The PO session folds them into this spec (PT and EN in the same commit) when the PR
  opens. The next implementation reads the spec, not the chat.

## 15. For the owner to review (Claude's decisions)

1. Stunned = Paralyzed + Dazed + senses nullified (§5.2); Dazed blocks acting and reacting.
2. Unconscious ends when the cause ceases (HP ≥ 0, revival) or by the master.
3. Visibility defaults that you did not mention: Suffocating and Open guard public; Freezing,
   Slowed, Paralyzed and Dazed hidden.
4. When a status is revealed, its past history appears.
5. Automatic combinations limited to those of §5.5.
6. List of suggested nicknames (§8).
7. Reviving and pressing the wound use **Force** (the new name of Push).
9. Penalties (§4): advantage and disadvantage cancel one by one.
11. Dead as a public status effect that contains Unconscious (§7).
10. Time outside the round (§10): batch resolution and a warning to the master before a time jump
   with someone losing HP.
8. Names: Unconscious (not Fainted); the sense status is called "Vision nullified", and "Cego"
   (Blind) is a nickname; the elemental family includes the toxic, listed in full in the docs.

## 16. Implementation decisions (phase 1)

> Taken in the back-end implementation sessions (PR #111 and PR 2, history). Where they conflict
> with the sections above, **these win**. Sources: the plan
> `docs/superpowers/plans/2026-11-17-status-effects-phase-1.md` ("Owner decisions", "Decisions
> this plan makes", "Deferred") and the contract `docs/dev/api/match-status-effects.md`.

### 16.1 Owner — 2026-11-17 (PR #111)

1. **Damage per round starts at 1** (Bleeding, Burning, Freezing, Poisoned).
2. **Recovery points** (Stunned, Paralyzed, Dazed): on manual apply the master types them; with
   the automatic test, they come from the failure.
3. **`npc_added` goes to the master only.** An NPC on the board reaches players through the fog,
   like any piece.
4. **Public statuses depend on the fog.** A player only receives a character's statuses while
   seeing it; the owner and the master always see. Out of sight: the server sends nothing and the
   player keeps the last state seen. Back in sight: the current list. Match without a board:
   everyone sees. With a board, a character with no piece is not seen by others.
5. **The round bars** (action and movement) follow the same sight.
6. **Hidden takes the piece off the map** and hides bars and statuses from whoever is not in the
   list of those who perceive it.
7. **Unconscious brings Prone**; the master may remove Prone and keep only Unconscious.
8. **Tact, balance face:** impaired gives the status's own disadvantage/number on attack, defense,
   dodge and movement; nullified gives one disadvantage on those four. Declared now; applied in the
   rolls phase.
9. **Everything that does not depend on another implementation is applied now** (the action and
   reaction blocks).
10. From the new advantage/disadvantage rule (§4), answered for the plan: a **passive test with N
    levels** rolls 2 + N dice and compares with the base value, as today; **D20 set**: one more
    D20 per level.

### 16.2 Owner — 2026-11-19 (PR 2: history and Hidden)

11. **Each status event in the history keeps its cause:** the specific action or reaction of the
    turn. For now the master gives it, optionally, when applying, editing or removing with a turn
    open (e.g. from the reaction's card). Without a cause, the event sits in the turn, in time
    order; outside a turn, in the round's events. The test, once it exists, fills the same field,
    plus the test's own id. **Never infer from "the last opened reaction":** at closing, all are
    open.
12. **Hidden hides the character's actions**, and with them the tests and statuses tied to those
    actions, from whoever does not perceive it — live and in the history.
13. **Nothing stays hidden forever.** The master reveals whenever he wants, even after the match
    ends:
    - a Hidden character's whole period (actions, targets and statuses of that character in the
      period) or a single status;
    - the reveal applies to everyone (per player is future) and covers what happened up to that
      instant;
    - flow: when removing Hidden, the front asks the master whether to reveal what stayed hidden;
      if not, he can do it later from the event's card.
14. **Widening who perceives the character does NOT reveal the past.** Only the master's command
    reveals. In the future, a perception test may make a player perceive.
15. **Hidden target:** for whoever does not perceive it, it leaves the target list and the result
    (live and in the history) until revealed. It may have been hit by accident.
16. **The closed result of the blow goes to everyone**, even to whoever does not perceive the
    attacker, in this phase.
17. Whoever was watching when Hidden came in **sees it come in** and then sees nothing more of that
    character, but keeps in the history what they had already seen.
18. **Lean storage for the rare case:** very few actions and statuses will actually be hidden.

### 16.3 Owner — 2026-11-20 (PR #112)

19. **Cancelled action in the history:** only the character's owner and the master see the entry;
    other players receive nothing.
20. **A Hidden target leaves no trace** for whoever does not perceive it: no id, no numbers, no
    reaction, no position. If the blow hit it, the result carries only how many hidden targets were
    hit, and the screen shows "Desconhecido 1", "Desconhecido 2"… (Unknown 1, 2). If it did not
    hit, nothing appears. The attack's margin also disappears for that reader. Once the master
    reveals, the real result appears.
21. **Undo within the same turn, with "Devolver?" (Give back?):** when the master removes a status
    in the same turn he applied it, it is an **undo** ("as if it never existed"), not a reveal. If
    the status had pulled actions off the queue or swapped a reaction, the screen asks the master
    whether to give them back and lists what returns. Given back, everything returns as it was: same
    action, same place in the queue, same data and the master's edits. What cannot return reaches
    the master with the reason in Portuguese. Outside the turn, or after closing, there is no
    question. A future general undo (ctrl+Z) uses this same path.
22. **Refusals caused by a status name it in Portuguese** (e.g. "Paralisado").
23. **Revealing after the NPC leaves the match:** stays as is for now.
24. **Health and Stamina bars for other players** [2026-11-21]: they come with **per-player
    visibility** (which character perceived the Hidden one, etc.), in the same session. Today
    Health goes only to the owner and the master, Stamina does not arrive live, and the master
    cannot reveal bars: those three gaps close there.

25. **Revealing status bars** [2026-11-21; rule recorded, per-player visibility is still to be
    refined]: **per character**, with a **shortcut to reveal all player characters' bars to all
    players**. **NPCs are revealed one by one** — the shortcut does not include them. Revealing all
    NPCs at once, if it exists, is a separate button and lives in the **match settings**.
26. **Stamina follows the same who-sees rule as Health** [2026-11-21].
27. **Blow into thin air** [2026-11-21]: an attack whose only target is a Hidden one that dodged
    appears, to whoever does not perceive the target, as an attack with no visible target. It
    already works this way.

### 16.4 From the implementer (the owner was informed)

- Re-applying a status the character already has is refused; the master edits the existing one.
  Several characters at once: all or nothing.
- The viewer list is stored per effect (empty = the kind's default); players never see that list.
- Revealing is a master-only HTTP call; it works during and after the match, and the front reloads
  the history. A WebSocket notice may come later.
- In the history, a status event is seen by whoever saw the character right before or right after
  the change, plus the owner.
- Containment inherits what a status **does**, not how it **ends** (Unconscious has no Tenacity
  recovery).
- The rest (required fields, reaction ↔ piece mapping, effect on the queue and pending reactions,
  text limits) is in the plan, "Decisions this plan makes".

## 17. What comes next and shapes the model

Whoever implements any phase needs to know this up front, because it decides the shape of the data.

- **Most statuses will come from tests** [owner, 2026-11-19]. The tests brainstorm is in progress
  (`2026-11-15-testes-brainstorm-notas.md`). Many will come out of **collisions between the action
  and the reactions**, so one turn applies statuses **at several moments and on several targets**
  (several reactions, several targets). Tests **outside battle** will also apply statuses. The
  master's manual statuses remain.
- **The system will know each status's cause** once tests exist. The cause field (§16.2, item 11)
  is born for that.
- **Reveal** also applies to **hidden tests** (the master's secret test, perception against Hidden).
- **Who sees will be per player.** A player who passes the perception test perceives the Hidden
  character and the others do not; per-player reveal is the same list. The model is already a list;
  the command is missing.
- **Health and Stamina bars above the piece**, with reveal by the master (§11): they come with
  per-player visibility (§16.3, item 24).
- **Stacking tests** (§13) only adds tests to an action's stack; it does not change the order.
