# Live sheet preview — back package — design

> Front plan (both repos): HxH_RPG_Environment_React,
> `docs/superpowers/plans/2026-11-01-sheet-live-preview.md`.
> Dev doc: [`../../dev/character-sheet/factory.md`](../../dev/character-sheet/factory.md) ("Paridade com o front").

## Problem

While creating or editing a sheet, the front showed values frozen at the class template: power,
skill values, principles, character level and HP/SP/AP bars did not follow the points the player
distributed or the proficiencies they picked.

## Decisions (product owner, 2026-11-01)

| # | Decision |
|---|---|
| D1 | Two implementations: the back (Go) stays the source of truth; the front re-runs the same assembly in TypeScript only to preview it. No preview endpoint, no WASM. |
| D2 | The front sends only the key values (class, `skillsExps`, `proficienciesExps`, `attributePoints`), as before; the back rebuilds everything from them. |
| D3 | The front engine stays decoupled from the app so it can become a shared library later. |

## Back changes

1. **`charactersheet.AssembleCharacterSheet`** — the single assembly path (validate the
   distribution, apply it over the class, `Build`, `ApplyInitialAttributePoints`). Create and
   update use it; before, each use case repeated these steps.
2. **`GET /classes` exposes `definition`** (`ClassDefinitionResponse`): the raw class build inputs
   (`skillsExps`, `attributesExps`, `proficienciesExps`, `jointProficiencies`, `jointProfExps`).
   The existing fields are the computed template; the front engine needs the inputs to replay
   the assembly. `jointProficiencies` (computed and definition) are sorted by name.
3. **Parity golden file** — `internal/app/api/sheet/sheet_parity_test.go` writes
   `testdata/sheet_parity.json`: for every class and a few scenarios, the input the front sends
   and the response the back builds, plus constants and exp tables. Without `-update` the test
   fails when the file is stale, which forces regeneration whenever a sheet rule changes. The
   front copies the file and runs its engine against every case.

## Findings

- Building the same sheet 6 times yields identical output: Go map iteration order does not change
  the sheet (the cascade is order-independent), which the front engine relies on.
- With today's classes, picking proficiencies never changes the Physicals level (so the number of
  attribute points does not move), but it does raise the character level (e.g. Hunter 6 → 7).
- The front used to send points for derived physical attributes (Strength…) on create; the back
  rejects those. Create and update now share one input builder that sends primaries only.

## Out of scope

`JointSkills` (no class uses them; the parity test fails if one starts to), Nen hexagon/category
percentages (create and update build without them), talent exp, buffs.
