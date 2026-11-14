# Weapon damage on the sheet's proficiency cards — implementation plan

**Date:** 2026-11-14
**Spec:** none. The design authority is this plan's "Design decisions" section, which
records the product owner's decisions (chat, 2026-11-14) on top of the Claude Design canvas
"Proficiências com dano da arma" (private artifact). Where this plan and the canvas differ,
the plan wins: the owner changed the canvas decisions listed below.

## Goal

The character sheet's "Proficiências" section shows, for every weapon, its damage in the
table format `2 D12 +3`. It must work in view mode and in create / full-edit mode (weapon
slots). The section also stops wasting the row's free space.

## Repos and branches

| Repo | Worktree | Branch |
|---|---|---|
| Backend (Go) | `System_X_System/.worktrees/weapon-damage` | `feat/rules-weapons-catalogue` |
| Frontend (React) | `System_X_System_React/.worktrees/weapon-damage` | `feat/proficiency-weapon-damage` |

Task 1 is backend; Tasks 2–6 are frontend, in order.

## Global Constraints

- **Language:** code in English (identifiers, file names, test titles). UI copy, code
  comments and guard-test messages in Brazilian Portuguese.
- **Wire format is camelCase**, no conversion layer. Types in `src/types/` mirror the wire 1:1.
- **Colors only through `colors.*` tokens** from `src/styles/tokens.ts` (a guard test fails
  on raw hex/rgba). No new palette color and no new token is needed for this work; if you
  think you need one, stop and report.
- **Fonts:** no raw font-family (guard test `noRawFonts`).
- **styled-components only**, no CSS files. The sheet uses container queries (`cqi`).
- **TypeScript:** `verbatimModuleSyntax` is on — type-only imports use `import type`.
- **Never remove TODO comments** (backend).
- **Damage format** is produced only by `formatDamage(dice, flat)` in `src/utils/dice.ts`.
  Never format damage by hand and never hard-code a weapon's damage in the front.
- **No scrolling list inside a proficiency card.**
- **Follow existing site patterns** before styling anything new (reuse `ExpandButton`,
  `iconButtonFx`, `Collapse`, `ProfilePillSelect`).
- Commit only your own files (`git add <files>` + `git commit`); confirm the branch with
  `git branch --show-current` before committing and stop if it is not the one in the table.
- Do not run `./dev-checkout.sh`, do not start dev servers, do not touch the main checkout.

## Design decisions (binding)

D1. **Data source.** A new public endpoint `GET /rules/weapons` serves the system's weapon
catalogue (`name`, `dice`, `flatDamage`). The sheet response is not changed. The front
looks damage up by weapon name.

D2. **Common proficiency card** (one weapon). Top row: weapon name left (weight 500),
`Level: N` right (`textInputDisabled`, no wrap). Second line: the damage, in
`textMuted`, at 0.77× the card text size. EXP bar glued to the bottom edge, as today.
Card text size stays `min(22px, 5cqi)`.

D3. **Joint proficiency card** (several weapons, one level). Same top row. Below it a
single full-width button row: `"{n} armas"` (`"1 arma"` when n = 1) followed by the
site's expand arrow. Collapsed by default. Expanded, it lists every weapon: name left,
damage right (`textMuted`, no wrap), each row separated by a 1 px `borderMuted` top
line. The list flows in CSS columns (`column-width: 220px`, `column-gap: 24px`, rows
never break across columns). When the joint has **more than 8 weapons** the expanded card
takes the whole row; with 8 or fewer it stays in its place. No scroll, ever.

D4. **Expand arrow is a "leaking" icon button** — the site's pattern for icon-only
controls. The arrow is larger than the text line it sits on and overflows the row's
padding instead of being boxed into icon size: width `1.8em` of the button's font size
with a floor of `26px`, with negative block margin so the row does not grow. It must not
cover or steal clicks from anything else, and must not be exaggerated. It reuses the
`ExpandButton` ion (decorative, the surrounding `<button>` toggles) and
`iconButtonFx(1.15)` for hover/focus, exactly like `ExpandableText`.

D5. **Weapon slot (create / full edit).** Top row: `Lv N · X XP` left, the chosen weapon's
damage **right, on the same row** (weight 500, `textPrimary`, 0.9× the card text size, no
wrap). If the two do not fit, the damage wraps to its own line and stays right-aligned.
While no weapon is chosen nothing is shown there — no hint text. Below: the pill select.

D6. **Weapon select.** The empty option and placeholder read `Selecione` (same word as the
class select; `Escolher…` is gone). Every weapon option shows its damage as a second,
smaller line. A weapon already taken by another slot stays disabled and its second line
reads `<damage> · já escolhida`.

D7. **The section fills its row.** Cards are laid out with flex-wrap so every row is
filled: base width 260 px, growing equally (`flex: 1 1 260px`), gap 15 px. Up to three
cards fit side by side at the desktop sheet width; one card takes the full row. The dead
gray area to the right must be gone.

D8. **Section title size.** "Proficiências" must render at exactly the same font size as
the column titles "ATRIBUTOS" / "PERÍCIAS" at every width (those are `8cqi` of their own
column). Formula, in the units of the proficiencies section container (`c` = its content
width): single-column sheet → `calc(8cqi + 3.2px)`; two-column sheet (`c ≥ 510px`) →
`calc(4cqi + 0.4px)`. Margin-bottom `0.625em`, padding-left `0.125em` (the same
proportions the column titles have).

D9. **Unknown damage is silent.** While the catalogue is loading, failed, or does not carry
a weapon, the card/slot/option simply shows no damage. No spinner, no error text.

---

## Task 1 — Backend: `GET /rules/weapons`

**Repo:** backend worktree, branch `feat/rules-weapons-catalogue`.
**Read first:** `internal/app/api/rules/routes.go`, `get_sheet_rules.go`,
`get_sheet_rules_test.go`; `internal/app/api/routes_registration_test.go`
(`TestRoutes_SheetRulesArePublic`); `internal/app/api/sheet/get_combat_catalogue.go` (how
it reads `item.WeaponsManager`); `cmd/api/main.go` around `rulesApi`;
`docs/dev/api/rules.md`; `docs/documentation-map.yaml` (entries for
`internal/app/api/rules/`).

**Build:**

1. `internal/app/api/rules/get_weapons.go`:
   - `WeaponResponse { Name string json:"name"; Dice []int json:"dice"; FlatDamage int json:"flatDamage" }`
   - `GetWeaponsBody { Weapons []WeaponResponse json:"weapons" }`
   - `GetWeaponsResponse { CacheControl string header:"Cache-Control"; Body GetWeaponsBody }`
   - `GetWeaponsHandler(catalogue *item.WeaponsManager) func(context.Context, *struct{}) (*GetWeaponsResponse, error)`
     builds the list **once** (outside the returned closure), iterating
     `enum.GetAllWeaponNames()` so the order is stable; a name the catalogue does not carry
     is skipped. `Dice` is a copy of the catalogue's slice, in catalogue order (the client
     sorts for display). Response header `Cache-Control: public, max-age=300`, same as the
     sheet rules.
2. `routes.go`: add `GetWeaponsHandler` to `rules.Api` and register
   `GET /rules/weapons` (tag `rules`, description "System weapon catalogue: dice and flat
   damage (public)"), with the same nil-safety the package already uses for its other
   handler (check how `routes_registration_test.go` builds `rulesHandler.Api{}` empty and
   keep that test passing).
3. `cmd/api/main.go`: wire `GetWeaponsHandler(item.NewWeaponsManagerFactory().Build())`.

**Tests (TDD, standard `testing`, table-driven, external test package):**

- `get_weapons_test.go`: with the real factory catalogue — every name from
  `enum.GetAllWeaponNames()` is present exactly once and in that order; spot values
  `Katana → dice [4 12], flatDamage 7`, `Bomb → [12 12 12], 0`, `Fist → [6 6 4], 0`;
  `Cache-Control` header value; a catalogue built with `item.NewWeaponsManager` missing a
  weapon skips it instead of failing; mutating the returned `Dice` does not change the next
  response.
- `routes_registration_test.go`: `/rules/weapons` answers 200 **without** a token (extend
  the existing public-routes test the way `/rules/character-sheet` is covered).

**Docs (same commit or a follow-up commit in this task):**

- `docs/dev/api/rules.md`: new section `## GET /rules/weapons — Catálogo de armas` (PT-BR
  prose): auth none; header; response example with real values; field table; note that
  `dice` comes in catalogue order and the client formats (`2 D12 +3`: largest die first,
  equal dice grouped, `+N` = `flatDamage`); note that `GET /charactersheets/{uuid}/combat-catalogue`
  is the per-character list and this one is the system's.
- `docs/documentation-map.yaml`: make the `internal/app/api/rules/` entry mention the new
  route, and map `internal/domain/entity/item/` → `docs/dev/api/rules.md` if it is not
  mapped to it (a change in the weapon table changes this response).

**Verify:** `go build ./...`, `go vet ./...`, `go test ./internal/app/api/... ./internal/domain/entity/item/...`.

**Done when:** tests pass, vet is clean, docs updated, one or two commits on the branch.

---

## Task 2 — Front: weapon catalogue data layer

**Repo:** frontend worktree, branch `feat/proficiency-weapon-damage`.
**Read first:** `src/services/rulesService.ts`, `src/hooks/useSheetRules.ts`,
`src/types/rules.ts`, `src/utils/dice.ts`, `src/types/characterSheet.ts`
(`JointProficiency`), `src/features/sheet/engine/types.ts` (the wire shape the preview
already produces for joint proficiencies), the tests next to the rules service/hook.

**Contract** (`GET /rules/weapons`, public, no auth):

```json
{ "weapons": [ { "name": "Katana", "dice": [4, 12], "flatDamage": 7 } ] }
```

**Build:**

1. `src/types/rules.ts`: `WeaponInfo { name: string; dice: number[]; flatDamage: number }`
   and `WeaponsResponse { weapons: WeaponInfo[] }`.
2. `rulesService.getWeapons(): Promise<WeaponInfo[]>` → `GET /rules/weapons`, returns
   `data.weapons`.
3. `src/features/sheet/utils/weaponDamage.ts` (pure):
   `buildWeaponDamageLookup(weapons: WeaponInfo[] | undefined): (weapon: string) => string | undefined`.
   The lookup is case-insensitive on the whole name (sheet keys arrive as
   `throwingDagger`, class lists as `ThrowingDagger`), returns
   `formatDamage(dice, flatDamage)`, and `undefined` for an unknown weapon or an
   undefined catalogue.
4. `src/hooks/useWeaponDamage.ts`: React Query, key `["weaponsCatalogue"]`,
   `staleTime: Infinity`, `retry: 1` (static content, like `useSheetRules`; public route,
   so no token in the key and no `enabled` guard). Returns the lookup function, memoized
   on the query data (stable identity between renders).
5. `JointProficiency` in `src/types/characterSheet.ts` gains `weapons: string[]` (the
   backend has always sent it — see `docs/dev/api/character-sheet.md` in the backend repo).
   Fix whatever no longer compiles (factories, fixtures) with real values, not casts.

**Tests (vitest):** service hits the right URL and unwraps `weapons`; lookup: exact name,
lower-cased first letter (`throwingDagger`), unknown weapon, undefined catalogue, formatted
output for `Katana` (`D12 + D4 +7`) and `Bomb` (`3 D12`); hook returns a working lookup
after the request resolves and `undefined` results while pending.

**Verify:** `npm run test`, `npm run lint`, `npm run build`.

---

## Task 3 — Front: `PillSelect` option description

**Read first:** `src/components/molecules/PillSelect.tsx`, `PillSelect.styles.ts`,
`__tests__/PillSelect.test.tsx`, `src/components/CLAUDE.md`.

**Build:** `PillSelectOption` gains optional `description?: string`. In the open list an
option with a description renders it as a second line under the label: font-size `0.6em`,
weight 400, `colors.textMuted`, line-height 1.25, wraps normally. Label and description sit
in one column; the ✓ column and the row's min-height are unchanged. Options without a
description render exactly as today (no extra element, no layout change). The closed pill
shows only the label; the `Sizer` logic (closed width = longest label) is untouched.
Typeahead keeps matching on `label` only. A disabled option dims label and description
together (the existing `opacity: 0.5`).

**Tests:** description text is rendered inside its option; options without description
have no description element; the closed trigger text is only the label; typeahead still
finds by label when descriptions are present; a disabled option with a description is still
not selectable.

**Verify:** `npm run test`, `npm run lint`, `npm run build`.

---

## Task 4 — Front: proficiency cards, joint card, section layout and title (view)

**Read first:** `src/features/sheet/ProficiencyCard.tsx`, `ProficienciesList.tsx`,
`CharacterSheetTemplate.tsx` (the `ProficienciesSection`, `SectionTitle`, `GridSection`
blocks), `src/components/ions/ExpandButton.tsx`, `src/components/molecules/ExpandableText.tsx`
(the leaking arrow button), `src/styles/iconButtonFx.ts`, `src/components/atoms/Collapse.tsx`,
`src/features/sheet/CharacterProfile.tsx` (how `Collapse` + `aria-controls` are used),
`src/features/sheet/CLAUDE.md`, `src/components/CLAUDE.md`,
`src/features/sheet/__tests__/ProficienciesList.test.tsx`.

**Build (decisions D2, D3, D4, D7, D8, D9):**

1. `ProficienciesList` calls `useWeaponDamage()` once and passes damage strings down.
2. `ProficiencyCard` gets an optional `damage?: string` prop.
   - Top row: `display: flex; justify-content: space-between; align-items: baseline; gap: 12px;`
     padding `12px 15px 0`. Without damage the row's bottom padding is `10px`.
   - Damage line: padding `2px 15px 10px`, font-size `0.77em` of the card text size,
     `colors.textMuted`.
3. New `src/features/sheet/JointProficiencyCard.tsx` — props `name`, `level`, `currExp`,
   `nextLvlBaseExp`, `weapons: string[]`, `damageOf: (weapon: string) => string | undefined`.
   - Same shell and top row as `ProficiencyCard` (share the styled pieces; do not
     duplicate them — export them from one place).
   - Toggle: a real `<button type="button">` spanning the card width, `aria-expanded`,
     `aria-controls` pointing at the list, text `"{n} armas"` / `"1 arma"`, then
     `<ExpandButton isExpanded={…} />` (decorative). Font-size `0.77em`,
     `colors.textMuted`, padding `4px 15px 10px`, no border/background,
     `${iconButtonFx(1.15)}`. On coarse pointers (`@media (any-pointer: coarse)`) the
     button is at least 44 px tall.
   - Arrow per D4: `width: 1.8em; min-width: 26px; margin-block: -0.35em;` (the row height
     must be the same with and without the arrow).
   - List inside `<Collapse open={…} id={…}>`: a `<ul>` (reset list styles), padding
     `0 15px 8px`, font-size `0.77em`, `column-width: 220px; column-gap: 24px`. Each
     `<li>`: flex, space-between, baseline, `flex-wrap: wrap`, gap `0 12px`, padding
     `6px 0`, `border-top: 1px solid ${colors.borderMuted}`, `break-inside: avoid`. Name
     left; damage right in `colors.textMuted`, `white-space: nowrap` (omitted when unknown).
   - Weapons in the order received. With no weapons at all, render no toggle and no list.
   - Collapsed by default. A constant `JOINT_FULL_ROW_FROM = 9` (commented in PT: from this
     many weapons the open list needs the whole row): expanded **and**
     `weapons.length >= 9` → the card's flex-basis is `100%`.
4. `ProficienciesList` container: `display: flex; flex-wrap: wrap; gap: 15px; align-items: flex-start;`
   every direct child (card or slot) `flex: 1 1 260px; min-width: 0;`. Joint entries render
   `JointProficiencyCard`.
5. Title: in `CharacterSheetTemplate`, the proficiencies section uses a
   `ProficienciesTitle = styled(SectionTitle)` implementing D8 exactly, with a PT comment
   explaining the formula (column title = 8 % of the column; the column is `(c + 10px) / 2`
   in two columns and `c + 40px` in one, `c` = this section's content width; two columns
   start at `c ≥ 510px`). Use an `@container (min-width: 510px)` query (the section already
   is `container-type: inline-size`).

**Tests:** common card shows the damage from the catalogue and shows none when the catalogue
has not loaded; joint card: label `5 armas`, collapsed by default (`aria-expanded="false"`,
content inert), click expands and lists each weapon with its damage, click again collapses,
singular `1 arma`, no toggle without weapons; keyboard: the toggle is a button reachable and
operable. Mock the catalogue through the service/hook the way the suite already mocks
services (see `renderWithProviders` and existing sheet tests).

**Verify:** `npm run test`, `npm run lint`, `npm run build`.

---

## Task 5 — Front: weapon slots in create / full edit

**Read first:** `src/features/sheet/ProficienciesList.tsx` (after Task 4),
`src/features/sheet/ProfileFieldStyles.ts`, `src/components/molecules/CharacterSheetHeader.tsx`
(the class select's `Selecione`), `src/features/sheet/__tests__/ProficienciesList.test.tsx`,
`src/test/pickCombobox.ts`.

**Build (decisions D5, D6, D9):**

1. Slot top row: flex, `justify-content: space-between; align-items: baseline; flex-wrap: wrap; gap: 0 12px`.
   Left: `Lv N · X XP` exactly as today. Right: the chosen weapon's damage — weight 500,
   `colors.textPrimary`, font-size `0.9em` of the slot's level text size, `white-space: nowrap`,
   `margin-left: auto`. Rendered only when a weapon is chosen and its damage is known.
2. Select: the empty option's label is `Selecione`. Each weapon option carries
   `description`: its damage; when the weapon is taken by another slot, the option stays
   disabled and the description is `"<damage> · já escolhida"` (just `"já escolhida"` when
   the damage is unknown).
3. Fixed proficiencies shown beside the slots in create mode use the Task 4 card, damage
   included. Slot selection logic (`handleSlotChange`, `slotSelections`, the camelCase /
   PascalCase handling) must not change behavior.

**Tests:** placeholder reads `Selecione`; choosing a weapon shows its damage in the slot;
no damage text before choosing and no leftover hint text; open list shows each option's
damage; the option taken by the other slot is disabled and reads `… · já escolhida`;
the existing slot tests still pass (update only the placeholder string they look for).

**Verify:** `npm run test`, `npm run lint`, `npm run build`.

---

## Task 6 — Front: docs and the "leaking icon button" rule

**Read first:** `src/components/CLAUDE.md`, `src/features/sheet/CLAUDE.md`,
`docs/dev/frontend-architecture.md` (skim the sheet section), the code delivered by
Tasks 2–5.

**Build (PT-BR prose, English code references):**

1. `src/components/CLAUDE.md`: a short section **"Botões só de ícone vazam"** — the
   site's icon-only controls (expand, collapse, close) are drawn larger than the box an
   icon would normally occupy and overflow onto the neighbouring edge/padding
   (`ExpandableText`: 54 px arrow hanging 22 px below the box; `CharacterProfile`: 6cqi
   arrow over a 3.8cqi bar; `JointProficiencyCard`: 1.8em arrow over the text row). Rules:
   do it whenever it fits; never tiny, never boxed to icon size; never exaggerated; never
   covering or stealing clicks from another control; the "Voltar" button does not leak and
   that is fine; hover/focus always through `iconButtonFx`. State the intent (the designer's
   visual identity; two Claude Design sessions drew small boxed buttons and the owner
   rejected them).
2. `src/features/sheet/CLAUDE.md`: a section **"Proficiências e dano da arma"** — where
   damage comes from (`GET /rules/weapons` → `useWeaponDamage`, never a front table),
   silent when unknown, the fill-the-row layout, the joint card rule (columns, full row
   from 9 weapons, no scroll), and the title-size formula with its why.
3. If `docs/dev/frontend-architecture.md` lists the rules service/hooks, add the weapons
   catalogue there in one line.

No code changes in this task. **Verify:** `npm run lint` (docs only — sanity), and the
statements match the code as delivered (file names, constants, numbers).
