# Status Effects Phase 6 — Game Docs and Strength Skills Rename — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Write the player-facing rules for status effects (principles, combat chapter, master doc, dash fix, Força de Vontade) and rename the Strength skills `Push → Force` ("Forçar") and `Carry → Charge` ("Carga") across back, DB and front.

**Architecture:** Docs-only tasks in `docs/game/` (PT, player language, no code; dev footer carries TODOs). The rename is a pure rename: enum values, sheet columns (`push_exp`/`carry_exp`), every persisted JSONB skill-name value (rewritten only under skill-name keys, never loose text), parity fixture, front sheet keys and the one match constant that lists the damage skills. No rule or combat behaviour changes.

**Tech Stack:** Go 1.23, PostgreSQL + goose, React/TS (Vite, vitest).

**Spec:** `docs/superpowers/specs/2026-11-15-status-effects-design.pt-br.md` (PR #110, not on `main`). Read it from the main checkout, where the file is identical to the PR branch:
`/home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System/docs/superpowers/specs/2026-11-15-status-effects-design.pt-br.md`
(or `git show origin/docs/status-effects-spec:docs/superpowers/specs/2026-11-15-status-effects-design.pt-br.md`). §16 overrides earlier sections. Examples: `…/specs/2026-11-07-status-brainstorm-notas.md`, section "Exemplos para as docs de jogo" (line ~1005). What exists today: `docs/dev/api/match-status-effects.md` (§1, §6, §7, §10).

**Worktrees** (branch `feat/status-effects-phase6` in both):
- Back: `/home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System/.worktrees/status-effects-phase6`
- Front: `/home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System_React/.worktrees/status-effects-phase6`

## Global Constraints

- Code identifiers and test titles in English; comments, UI and `docs/game/` in Portuguese.
- `docs/game/`: zero code references (no type names, paths, wire values, `kind` strings); player language; ends with the footer `---` + `> **🔧 Para Desenvolvedores**` linking the dev docs. Things the spec defines but the game does not do yet go ONLY in that footer, as `> - TODO (<tema>): fora do texto do jogador até <o quê>. <a regra definida, em uma ou duas frases>.` — the pattern in `docs/game/armas.md` lines 105–107.
- Vocabulary (glossary + spec §4): "status" at the table; "Barras de status" = HP/SP/AP; "condição" = the master's roll adjustment; Aptidão Física/Mental/Espiritual; System X System; skill names in PT per `docs/game/glossario.md`. Resiliência is an attribute (no skill).
- New skill names: **Forçar** (`Force`) = empurrar, puxar, arrombar, pressionar. **Carga** (`Charge`) = capacidade de carga e embalo (momentum). **Agarrar** (`Grab`) unchanged. Order everywhere the back lists them: `Force, Grab, Charge`.
- Never rename the verb "push" (slices, `git push`, "Push the filtered…" in `room.go`), nor the action field `Charge *RollCheck` in `attack.go`/`move.go` (the momentum roll — a different thing that keeps its name), nor `carry`/`Carry` meaning bar carry-over (`bars_view.go`, `actionCarry`, `ClockEditor`).
- Historical docs are never edited: `docs/superpowers/plans/**`, `docs/superpowers/specs/**` (both repos).
- Do not delete TODOs/comments outside scope; fix comments the rename makes wrong.
- Out of scope: any rule or combat-code change; Biske's chart; charged blow / charge mechanics; match layout in the front (only the one constant in Task 5).

## Review Focus

1. **A sheet with XP in Push/Carry before the migration keeps the same XP in Force/Charge after it** (columns renamed, not dropped/recreated) — Task 3 smoke compares sums.
2. **History of an old match still shows the damage skill** after the migration: `turns.resolution.damageSkill`, `actions.attack.DamageSkill`, `actions.attack.Damage.SkillName`, `actions.skills[].SkillName` become `Force` — Task 3 smoke counts zero `"Push"`/`"Carry"` under skill keys.
3. **A loose `"Push"` that is not a skill name is untouched** (e.g. a weapon or note value) — Task 3 literal-document check.
4. **The master's damage-skill toggle at the table** still shows two buttons with the current one pressed (`Force`), not a third disabled one — Task 5 test.
5. **A player reading the combat chapter is never promised an automatic test, per-round HP loss or a recovery roll** that the game does not perform — Task 1 grep step.

---

### Task 1: Principles and the status-effects combat chapter

**Files:**
- Create: `docs/game/principios.md`
- Create: `docs/game/combate/efeitos-de-status.md`
- Modify: `docs/documentation-map.yaml` (register the new chapter under the status-effect code paths)

**Interfaces:**
- Produces: anchors other docs link to — `combate/efeitos-de-status.md#quem-vê` is NOT needed; Task 2 links the chapter file itself.

- [ ] **Step 1: Write `docs/game/principios.md`** — H1 "Princípios do sistema"; one short intro line; the two master lines of spec §1 ("Realidade", "Cânon"), stated as being **of the same level**, with the weighing criteria of the Cânon line; then a short "Como isso aparece nas regras" with the design principles of §1 that a player/master benefits from ("poucos status genéricos e combináveis", "um status cabe numa frase", "o sistema propõe, o mestre dispõe" linking `combate/o-mestre.md`, "o status diz o estado, não a quantidade"). No Biske chart (it enters with Fadiga — footer TODO). Footer links `../superpowers/specs/2026-11-15-status-effects-design.pt-br.md` §1.

- [ ] **Step 2: Write `docs/game/combate/efeitos-de-status.md`** with these sections, in order:
  1. **O que é um status** — vocabulary (status ≠ barras ≠ condição; apelido is text, not a new status); "existe um Sangrando, nunca dois"; statuses last outside battle and between sessions; who applies: today **the master**, plus Morto by the rule (Task 2 covers the master's how).
  2. **Os status, um por frase** — grouped as the spec §5.3 (Consciência e corpo, Parte do corpo, Sentidos, Testes, Vida, Ar, Guarda, Mente, Presença) plus Morto. Each: the spec phrase, adapted so it says only what the game does now (table below). Include the Tato three faces, Debilitado's three table names (Corpo/Mente/Espírito debilitado), Lesionado "um só por personagem, lesões novas somam", Veneno é fonte, não status, Confuso/Alucinando are combinations, no provocation mechanic.
  3. **Como os status se encaixam** — the §5.2 diagram verbatim in a ```text block (keep "EM ESPERA: Fadiga"), then its readings (Atordoado = Paralisado + Atônito + sentidos anulados; Preso; Lento; Controlado ⊃ Compelido; guard positions independent; Caído não implica Guarda aberta).
  4. **O que cada status deixa fazer na guarda** — the §5.4 table with §16.4 corrections (Caído escape: desvantagem na esquiva e no movimento; Guarda aberta escape: desvantagem nos dois testes), the "padrão; o mestre muda" note, and "rolar e deslizar ainda não existem: a linha do Caído se completa com o movimento". Link `reacoes.md#do-que-cada-ação-é-feita` for the action sets.
  5. **O que pesa nos testes** — per status (from contract §6 "O que pesa nas rolagens", in player words); "um status pesa uma vez por teste; status diferentes somam"; vantagem/desvantagem accumulate and cancel one by one, one extra D10 per level, best/worst two — one paragraph + link `../dados.md#vantagem-e-desvantagem` (do not duplicate the probability table; one line saying two advantages reach D20's critical chance without its critical failures). Falha sozinho = não rola, sem crítico.
  6. **Qual perícia resiste a quê** — the §6.3 table verbatim (theme → skill), introduced as the reference the master uses when a scene asks for a resistance.
  7. **Crítico e erro crítico** — §7 + §16.4 item 28 in three bullets (total ×2 / ÷2 para baixo; acerto crítico também dobra o dano final; o dado de dano nunca gera crítico — the 2 D12 example) + link `../dados.md#crítico-e-erro-crítico`.
  8. **Vida negativa e morte** — link `../ficha-de-personagem/status.md` for the limit formula; then: passou 1 além do limite → morre no fim do turno; HP fica onde morreu (HP 10, limite −16, golpe de 50 → −40); Morto contém o Inconsciente, não traz Caído, entra só pela regra; o mestre vê antes da mesa (o cálculo do turno avisa "morre") e pode desfazer, escolhendo a Vida da volta (padrão: o limite); todos veem, seguindo a névoa; na campanha a marca de morto entra ao encerrar a partida.
  9. **Exemplos** — from the notes, only those valid today, written with the master applying: terremoto/ponte/gelo (o mestre põe Caído em quem perde o equilíbrio), habilidade de gelo (idem, a área pede o equilíbrio — frame as the master's call), golpe "com tudo" que erra → Caído, se jogar no chão → Caído sem teste, muito dano → o mestre põe Caído e Sangrando, Inconsciente carregado/sentado → sai o Caído, choque elétrico → Paralisado (choque), Kastro × Hisoka → Atônito (não usa o clone, morre por isso), Illumi's needle → Compelido irresistível "fugir quando em perigo", o Gon e a plaqueta do Hisoka (vantagem dupla). Aliases come from spec §8 only (not the notes' list, which has rejected names).
  10. Footer `> **🔧 Para Desenvolvedores**`: links `../../dev/match/status-effects.md` and `../../dev/api/match-status-effects.md`, then one TODO line per item of the "Not yet" column below, each with the spec rule (cite the § in parentheses).

  **What the player text may say (does now) vs footer TODO (not yet):**

  | Status / topic | Player text (does now) | Footer TODO (not yet) |
  |---|---|---|
  | Atônito, Atordoado, Inconsciente, Morto | não age, não reage; os quatro domínios travados; teste de sentido falha sozinho (Atordoado/Inconsciente/Morto); Atordoado/Paralisado/Atônito têm um número (quanto falta para se recuperar), registrado pelo mestre; saem quando o mestre tira | entrada por Tenacidade contra o dano final (§6.2); Recuperação a cada round com a sobra como bônus — the notes' "20 de dano tirando 15 → 5; tira 18 → +13" example goes here (§5.3); "não usa habilidades" (Nen) |
  | Paralisado | domínios marcados travados (padrão: os quatro); sem deslocamento se o movimento travou | Recuperação por Tenacidade; habilidade que não dependa do corpo (Nen) |
  | Inconsciente | traz o Caído junto (o mestre pode tirar o Caído) | Vitalidade contra o HP negativo; sangramento por Cura ao passar de 0 (§7) |
  | Preso | não se desloca; não escapa; age e reage | sair do Preso por ação declarada (Forçar, Deslizar, disputa contra Agarrar — §5.3) |
  | Lesionado | desvantagem nos domínios machucados; inutilizado = o sistema avisa o mestre; atravessa o fim da batalha, o mestre decide quando sai | cura por habilidade/item/recuperação |
  | Lento | penalidade na velocidade do movimento | distância (movimento) |
  | Sentido afetado | prejudicado: desvantagem e/ou número nos testes do sentido (Tato: um teste de Tato numa ação conta como toque); anulado: falha sozinho; Visão anulada: desvantagem no ataque, na defesa e na esquiva; Tato no equilíbrio: pesa em ataque, defesa, esquiva e movimento | teste de equilíbrio ao levar golpe e ao mover → Caído; Recuperação do sentido (três saídas §5.3) |
  | Debilitado | desvantagem e/ou número em todo teste da Aptidão marcada | testes de Nen (espiritual) onde o Nen ainda não existe |
  | Sangrando, Queimando, Congelando, Envenenado | o número por round (começa em 1) é mostrado; saem quando o mestre tira | perda de HP por round; Tenacidade na entrada; estancar com Cura (cada dezena tira 1); primeiros socorros (Legerity/Precisão/Forçar) |
  | Asfixiando | o status existe e é visto | fôlego/afogamento inteiro (§5.3 Ar) |
  | Caído | desvantagem na esquiva por reflexo, no repelir e nos dois testes do escape (provisório) | só rola/desliza; levantar-se (movimento) |
  | No ar | sem esquiva por reflexo e sem escape; não muda a trajetória | aplicado pelo sistema no salto (movimento) |
  | Guarda aberta | sem esquiva por reflexo e sem repelir; defesa só rolada; escape com desvantagem nos dois testes; o mestre tira (verified: the server never removes an effect by itself — contract §10) | sair sozinha no fim da batalha, provisório (§10 "Tempo fora do round"); manter a guarda por botão |
  | Compelido | não bloqueia; o sistema lembra o mestre da conduta; pode ser marcado irresistível | o teste de Resiliência/Força de Vontade pedido pelo mestre |
  | Controlado | o status diz quem controla | o outro declarar as ações |
  | Escondido | some do mapa, das barras, dos status e das ações de quem não o percebeu; o mestre esconde e revela | Furtividade contra a percepção (testes) |
  | Tempo fora do round, revelar barras, visibilidade por jogador, Fadiga, queda | — | one TODO each (§10, §11/§16.3 items 24–25, §9/§17, §13, §13) |

- [ ] **Step 3: Register in `docs/documentation-map.yaml`** — add `docs/game/combate/efeitos-de-status.md` (with a one-line `reason` in PT) to the `affected_docs` of the existing entries `internal/domain/statuseffect/`, `internal/domain/statuseffect/roll_effect.go` and `internal/domain/match/matchsession/death.go`. Follow the file's existing entry shape. Do NOT add either new doc to `internal/application/rules/chapters.go`: the sheet's "?" panel only serves character-creation chapters (spec §12: combat rules are shown elsewhere).

- [ ] **Step 4: Verify**

Run (from the back worktree):
```bash
grep -nE '`[a-z]+[A-Z]|internal/|\.go\b|kind:|"[a-z]+":' docs/game/principios.md docs/game/combate/efeitos-de-status.md | grep -v 'Para Desenvolvedores' 
grep -n -iE 'a cada round (o sistema|rola)|perde [0-9N]+ de (HP|Vida) por round|rola sozinho|automaticamente' docs/game/combate/efeitos-de-status.md
go test ./internal/application/rules/... ./internal/app/api/rules/...
```
Expected: first two greps print only lines inside the footer (lines starting with `>` after `🔧 Para Desenvolvedores`); tests PASS (the embed still builds).

- [ ] **Step 5: Commit**
```bash
git add docs/game/principios.md docs/game/combate/efeitos-de-status.md docs/documentation-map.yaml
git commit -m "docs(game): princípios do sistema e capítulo de efeitos de status"
```

### Task 2: Master doc, dash fix, alias copy

**Files:**
- Modify: `docs/game/combate/o-mestre.md` (new section before "O que os jogadores enxergam"; footer gains links + TODOs)
- Modify: `docs/game/combate/reacoes.md:191`
- Modify: `docs/dev/api/match-status-effects.md:121-135` (alias suggestions)

**Interfaces:**
- Consumes: `docs/game/combate/efeitos-de-status.md` (Task 1) — link it, do not repeat its per-status content.

- [ ] **Step 1: Add section "Efeitos de status na mão" to `o-mestre.md`** — subsections:
  - **Aplicar**: a status on one or several characters at once (all or nothing); only the fields that status has (domains, faces, level, number, conduct); the rest starts at the default and every default is editable; reapplying one the character already has is refused — edit it instead; Morto is never applied by hand.
  - **Editar e tirar**: through the status itself; taking off frees whoever was blocked (Atordoado…). **Desfazer no mesmo turno** (§16.3 item 21): removing in the same open turn it was applied is an undo, "como se nunca tivesse existido"; if it had removed actions from the queue or swapped a reaction, the master is asked "Devolver?" and what can't come back arrives with the reason. Desfazer a morte: §16.4 item 30 (choose the HP; default = the negative limit).
  - **O apelido**: free text shown with the status (up to 60 characters), narrative weight only; suggestions = spec §8 list verbatim, with **Preso — Agarrado, Amarrado, Soterrado, Pendurado**.
  - **Quem vê**: write in the spec's plain words, saying *who* sees (memory rule: never "escondido só para o mestre"). Public by default: Caído, No ar, Preso, Queimando, Inconsciente, Atordoado, Asfixiando, Guarda aberta, Morto. The others are seen, by default, by the master and the character's player (on an NPC: only the master). The master can make one public, or keep it to himself even on a player character (the needle). Players only see a public status of a character while they see that character on the map (no map: everyone); a private status still weighs on the player's tests. History: entry/change/exit are recorded; what a player couldn't see doesn't appear to them until revealed; the master reveals when he wants, even after the match ends, and widening who perceives an Escondido does not reveal the past. Escondido: see the chapter.
  - Footer: keep the existing link; add links to `../../dev/api/match-status-effects.md`; TODOs: the master's screen for statuses does not exist in the front yet (the server already accepts it — front phases); choosing specific players (the server keeps a per-status player list; the text waits for per-player visibility, §9/§17); table setting "status de PJ públicos" (§9, no way to change it yet); revealing to one player only (§16.2 item 13).

- [ ] **Step 2: Fix the dash in `reacoes.md:191`** — replace the sentence `Durante o dash você está "no ar" e **não pode esquivar**: fica exposto. Por isso o escape fechado se move com Shift.` with the §5.3 "No ar" correction: no dash the character is neither exposed nor airborne; who already dashed and is attacked will hardly move again, but can evade, repel, parry or defend. Keep the Shift rationale only if it still reads true without "exposto" (it does not depend on it — drop "Por isso"). Search the file for other "exposto"/"no ar" mentions about the dash and fix them the same way.

- [ ] **Step 3: Alias copy in the contract** — in `match-status-effects.md` "Sugestões de apelido", replace `Agarrado, Enredado, Soterrado, Pendurado` with `Agarrado, Amarrado, Soterrado, Pendurado` (spec §8 is the source).

- [ ] **Step 4: Verify**
```bash
grep -n -i "exposto\|no ar" docs/game/combate/reacoes.md
grep -n "Enredado" docs/dev/api/match-status-effects.md docs/game -r
grep -nE '`[a-z]+[A-Z]|internal/|\.go\b' docs/game/combate/o-mestre.md
```
Expected: no dash sentence calling the character exposed/airborne; no "Enredado"; the last grep prints only footer lines.

- [ ] **Step 5: Commit**
```bash
git add docs/game/combate/o-mestre.md docs/game/combate/reacoes.md docs/dev/api/match-status-effects.md
git commit -m "docs(game): o mestre e os status; dash sem 'no ar'; apelido Amarrado"
```

### Task 3: Rename Push → Force, Carry → Charge in the back (code, migration, fixtures)

**Files:**
- Modify: `internal/domain/entity/enum/skill_name.go` (constants `Force SkillName = "Force"`, `Charge SkillName = "Charge"`; `AllSkillNames` order `Force, Grab, Charge`; the Strength comment)
- Modify: `internal/domain/entity/character_class/character_class_factory.go`, `internal/domain/entity/character_sheet/sheet/{half_sheet.go,character_sheet.go,character_sheet_factory.go}`
- Modify: `internal/domain/match/entity/action/attack.go` (`EffectiveDamageSkill` zero → `enum.Force`), `internal/domain/match/service/{turn_resolver.go,damage.go}` (default skill + comments), `internal/domain/match/matchsession/match_session.go`, `internal/domain/match/entity/action/master_action.go`, `internal/app/game/message.go`, `internal/gateway/pg/round/resolution_record.go` — comments that name the skill "Push"
- Modify: `internal/gateway/pg/sheet/{create,update,read}_character_sheet.go` + the model struct (`PushExp`→`ForceExp`, `CarryExp`→`ChargeExp`, columns `force_exp`, `charge_exp`)
- Create: `migrations/20261127000000_rename_strength_skills.sql`
- Modify: every `_test.go` that uses the skill name (`grep -rln '"Push"\|"Carry"\|enum\.Push\|enum\.Carry' internal cmd`); rename `internal/domain/match/service/actor_push_test.go` → `actor_force_test.go` and `push_repercussion_test.go` → `force_repercussion_test.go` **only if** their "push" is the skill (read them first), renaming identifiers inside accordingly
- Regenerate: `internal/app/api/sheet/testdata/sheet_parity.json`

**Interfaces:**
- Produces: `enum.Force`, `enum.Charge` (wire strings `"Force"`, `"Charge"`; sheet map keys lowercase-first `force`, `charge`); `enum.Push`/`enum.Carry` no longer exist. Task 5 relies on these strings.

- [ ] **Step 1: Rename in Go** — enum first, then fix every compile error and every test string; update the comments that said Push/Carry for the skill. Then:
```bash
go build ./... && go vet ./... && go vet -tags integration ./... && go vet -tags smoke ./...
grep -rn 'enum\.Push\|enum\.Carry\|"Push"\|"Carry"\|PushExp\|CarryExp\|push_exp\|carry_exp' --include=*.go internal cmd
```
Expected: builds clean; grep empty.

- [ ] **Step 2: Regenerate the parity fixture and run the suite**
```bash
go test ./internal/app/api/sheet/ -run TestSheetParityFixtures -update
go test ./...
```
Expected: PASS; `git diff --stat internal/app/api/sheet/testdata/sheet_parity.json` shows only key renames (`grep -c '"Push"\|"Carry"' …sheet_parity.json` → 0, and the numbers are unchanged — confirm with `git diff … | grep '^[-+]' | grep -v 'Push\|Carry\|Force\|Charge' | head` printing nothing but the diff headers).

- [ ] **Step 3: Write the migration** `migrations/20261127000000_rename_strength_skills.sql`, in the style of `20261113000000_rename_weapon_enum_values.sql` (header comment in PT saying where the name lives and that only skill-name keys are rewritten; idempotent):
  - `ALTER TABLE character_sheets RENAME COLUMN push_exp TO force_exp;` and `carry_exp TO charge_exp` (Down: the reverse).
  - A recursive `pg_temp.map_skill_names(doc JSONB, renames JSONB) RETURNS JSONB`, `LANGUAGE plpgsql IMMUTABLE`: object → rebuild with `jsonb_object_agg`; for keys in `('SkillName','skillName','DamageSkill','damageSkill')` whose value is a string present in `renames`, put `renames -> value`; every other value recurses; array → `jsonb_agg(… ORDER BY ordinality)`; scalars returned as-is; empty object/array preserved (`COALESCE(…, '{}'/'[]')`).
  - Up `renames = '{"Push":"Force","Carry":"Charge"}'`, Down the inverse. Apply, each `WHERE col::text ~ '"(Push|Carry)"'` (Down: `Force|Charge`), to: `actions` (`speed, skills, move, attack, defense, dodge, feint, trigger, interact, default_defense, repel`), `cancelled_actions.action`, `turns.resolution`, `master_actions.content`, `overridden_action_values.original_value`, `match_events.payload`.
  - `overridden_action_values`: `field` `'skill:Push.condition'` → `'skill:Force.condition'` (same for Carry); and where `field = 'damageSkill'` and `original_value` is a bare JSON string in `renames`, replace it (check in `internal/domain/match/override.go` / `match_session.go` how that row's value is encoded and match it).

- [ ] **Step 4: Run integration tests** (fresh test DB runs every migration up):
```bash
set -a && . ./.env && set +a && export TEST_DATABASE_URL="postgres://$PG_DB_USER:$PG_DB_PASS@localhost:5432/hxh_rpg_test?sslmode=disable"
go test -tags=integration ./internal/gateway/pg/...
```
Expected: PASS. (The DB smoke on a copy is done by the controller in Task 6.)

- [ ] **Step 5: Commit**
```bash
git add -A internal migrations
git commit -m "refactor(skills): Push vira Force e Carry vira Charge, com migração da experiência e do histórico"
```

### Task 4: Sheet docs, glossary and dev docs for the rename; Força de Vontade

**Files:**
- Modify: `docs/game/ficha-de-personagem/pericias.md` (Força table; Força de Vontade row; intro line 5 only if it names the skill), `docs/game/glossario.md:60-62` (and its Força de Vontade row if any), `.github/instructions/glossary.instructions.md` (add `Force | Forçar (perícia)` and `Charge | Carga (perícia)`), `docs/game/ficha-de-personagem/atributos.md:36` (Força: "forçar, agarrar, carga")
- Modify: current dev docs naming the skill: `docs/dev/character-sheet/factory.md:421`, `docs/dev/api/character-sheet.md:321`, `docs/dev/match/combat-engine.md` (damage skill default, ~1068–1075), `docs/dev/api/match-history.md` (examples + the `damageSkill` note ~302–304: turns recorded before Phase 8 were all Push → now say Force, and that the 2026-11-27 migration renamed stored values), `docs/dev/api/match-combat-ws.md` and any other `docs/dev/**` hit of the skill; `docs/documentation-map.yaml:1481-1482` text; `AGENTS.md` only if it names the skill
- Modify: `docs/documentation-map.yaml` — add the migration's code path if the map lists migrations (follow its convention; skip if it does not)

**Interfaces:**
- Consumes: wire names from Task 3 (`Force`, `Charge`).

- [ ] **Step 1: Sheet rows** — Força table in `pericias.md`:

| Perícia | Na ficha | Descrição |
|---|---|---|
| Forçar | Force | Empurrar, puxar, arrombar ou pressionar: a força aplicada contra algo ou alguém |
| Agarrar | Grab | Capacidade de segurar e imobilizar |
| Carga | Charge | O peso que o personagem carrega e o embalo que acumula: quem carrega muito ou embala demais não para num instante |

  Força de Vontade: keep the current sense and add the canon one (spec §12): a vontade de lutar contra quem é — ou parece — mais forte, sem assumir a própria pior condição. One sentence, then credit "(é o que a Biske cobra do Killua)". Same wording in `glossario.md` if it describes the skill. Footer of `pericias.md`: TODO (golpe carregado e investida): fora do texto do jogador até o spec dos testes — Carga mede o golpe carregado e o acúmulo de velocidade além da Aceleração (§12).

- [ ] **Step 2: Dev docs** — replace the skill name in current docs; where a doc explains the default damage skill, say `Force` and keep the history ("antes de 2026-11-27, `Push`").

- [ ] **Step 3: Verify**
```bash
grep -rn -E '\bPush\b|\bCarry\b|Empurr|Carregar \|' docs AGENTS.md .github | grep -v 'docs/superpowers/' | grep -v -i 'git push\|push no websocket\|carry-over\|carry +\|`carry'
go test ./internal/application/rules/... ./internal/app/api/rules/...
```
Expected: grep prints only lines that are about the verb or bar carry-over (inspect each); tests PASS.

- [ ] **Step 4: Commit**
```bash
git add docs AGENTS.md .github
git commit -m "docs: Forçar e Carga nas docs da ficha, glossário e contratos; Força de Vontade com o sentido canônico"
```

### Task 5: Front — sheet keys, damage-skill constant, fixtures

Work in the **front worktree**.

**Files:**
- Modify: `src/features/sheet/PhysicalSkillsGroup.tsx:28` → `skillsSubList={["charge", "force", "grab"]}` (alphabetical, like the other groups)
- Modify: `src/features/sheet/factories/skill.factory.ts` (`carry`→`charge`, `push`→`force`), `src/features/sheet/utils/distribute.ts:159-161`, `src/features/sheet/engine/buildSheet.ts:85` → `["Strength", ["Force", "Grab", "Charge"]]`
- Modify: `src/features/match/combat/RollEditLine.tsx:8` → `const DAMAGE_SKILLS = ["Force", "Grab"];` and the comment at :54; `src/features/match/combat/combatMessages.ts:259` comment. Nothing else in `features/match` beyond tests (another session is redoing the table layout).
- Modify tests: every `"Push"`/`push:` skill fixture — `src/features/match/combat/__tests__/{ActionComposer,combatOrganisms,normalizeWire,rollEdits}.test.ts(x)`, `src/pages/__tests__/{GameMasterPage,GamePlayerPage}.test.tsx`, `src/services/__tests__/characterSheetsService.test.ts`
- Sync: `src/features/sheet/engine/__fixtures__/sheet_parity.json`
- Modify docs (current behaviour only): `docs/dev/match/combate-fase-8.md`, `docs/dev/match/combate-parede-alvo.md`, `docs/dev/match/combate-reacao-padrao.md` — "Push/Grab" → "Force/Grab"

**Interfaces:**
- Consumes: back strings `Force`, `Charge`; sheet keys `force`, `charge`; regenerated fixture from Task 3.

- [ ] **Step 1: Update the damage-skill test first** — in `combatOrganisms.test.tsx` the test "Push/Grab manda damageSkill na hora…" becomes "Force/Grab sends damageSkill at once; unpressing the current one sends None" with `damageSkill: "Force"` in its fixture and `getByRole("button", { name: "Force" })` asserted `aria-pressed="true"`, clicking `Grab` sends `{ damageSkill: "Grab" }`, and no third disabled button exists (`queryAllByRole("button", { name: /^(Force|Grab)$/ })` length 2). Run `npx vitest run src/features/match/combat/__tests__/combatOrganisms.test.tsx` → FAIL (constant still Push).

- [ ] **Step 2: Rename everything listed above; sync the fixture**
```bash
npm run sync:sheet-fixtures -- /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System/.worktrees/status-effects-phase6
git grep -n -E '"Push"|"Carry"|\bpush:|\bcarry:|"push"|"carry"' -- src | grep -v -E '\.push\(|history|router'
```
Expected: grep empty.

- [ ] **Step 3: Full check**
```bash
npm run typecheck && npm run lint && npm run test
```
Expected: all PASS, including the sheet parity test.

- [ ] **Step 4: Commit**
```bash
git add -A src docs
git commit -m "refactor(sheet): Push vira Force e Carry vira Charge (paridade com o back)"
```

### Task 6 (controller): Delivery

- [ ] Back: `golangci-lint cache clean && ~/go/bin/golangci-lint run --max-same-issues 0 --max-issues-per-linter 0`; `go test ./...`; integration suite.
- [ ] **Migration smoke on a DB copy** (never the dev DB that `main`'s API uses): `docker exec hxh_rpg_system-db-1 pg_dump` of the dev DB → new DB `hxh_phase6_smoke`; record `SELECT sum(push_exp), sum(carry_exp) FROM character_sheets` and the counts of `"Push"`/`"Carry"` per JSONB column; first, in one psql session, create the function from the migration and run it on literal documents: nested object, `skills` array, `{"Weapon":"Push"}` (must not change), `{"DamageSkill":"Carry"}`, `"Push"` bare; then `goose up` on the copy; sums equal under the new names, zero counts under skill keys; `goose down` restores, `goose up` again is clean.
- [ ] Browser: branch API on `localhost:5001` against the copy (`PG_DB_NAME=hxh_phase6_smoke SERVER_ADDR=localhost:5001`), front of the front worktree pointing to it; open a sheet with XP in Strength: shows Charge / Force / Grab with the same XP as before; open the rules panel "Perícias" chapter: Forçar / Carga rows.
- [ ] PRs (one per repo, cross-linked), each with "Decisões tomadas" in PT (below), what was verified and what was not; `./dev-checkout.sh` only if something is left unverified.

## Decisões tomadas (copiar para os PRs, ajustando ao que aconteceu)

1. Os capítulos novos (princípios, efeitos de status) não entram no painel "?" da ficha: ele é da criação de personagem e o spec §12 diz que as regras de combate aparecem em outro lugar do site.
2. Os exemplos das notas que dependem de mecânica que ainda não existe (Recuperação com bônus, Tenacidade contra o dano, salto mortal) foram para o rodapé "Para Desenvolvedores"; os outros foram escritos com o mestre aplicando o status.
3. A tela do mestre para aplicar status ainda não existe no front; a doc do mestre descreve a regra e o rodapé registra a tela como TODO.
4. O servidor já guarda uma lista de jogadores por status, mas escolher jogadores específicos ficou fora do texto do jogador (visibilidade por jogador) e entrou como TODO.
5. A migração renomeia as colunas de experiência e reescreve o nome da perícia em todo o histórico gravado, só nos campos que guardam nome de perícia — nunca em texto solto.
6. Na mesa do front só a constante das perícias de dano mudou (Force/Grab), para o seletor não quebrar; nada de layout.
7. No contrato, a sugestão de apelido do Preso "Enredado" virou "Amarrado" (spec §8).
8. A coluna "Na ficha" e a ficha continuam mostrando o nome em inglês (Force, Charge), como as outras perícias.
