---
applyTo: "docs/**"
---

# Documentation Workflow

## Structure

| Directory | Purpose | Audience | Language |
|-----------|---------|----------|----------|
| `docs/game/` | Game rules (RPG rulebook content) | Players & Masters | PT-BR |
| `docs/dev/` | Technical flows, design rationale | Developers | PT-BR |
| `docs/superpowers/specs/` | Feature design specs | Developers | EN + PT-BR |
| `docs/superpowers/plans/` | Implementation plans | Developers | EN |

**Key rule:** `docs/game/` = ONLY game rules (zero code refs, printable as rulebook). Technical details → `docs/dev/`.

## Conventions

- **Game docs:** PT-BR, player-friendly language (no thechnical jargon for devs)
- **Dev docs:** PT-BR prose with English code references (type names, methods, paths)
- **Developer footers:** Game docs include `> 🔧 Para Desenvolvedores` footer linking to dev docs
- **`docs/game/` is tracked normally.** (An older note here claimed it needed `git add -f`; that
  was wrong — the `/game` pattern in `.gitignore` is root-anchored and only matches the game
  binary, never `docs/game/`.)
- **Specs:** EN + PT-BR versions (`.pt-br.md` suffix) committed together

## Maintenance Workflow

**Rule:** Every PR changing code in `internal/` or `cmd/` MUST verify whether docs need updating.

**Source of truth:** `docs/documentation-map.yaml` maps code paths → affected docs.

### Process (before finishing a branch)

1. Run `check_documentation_impact` tool (or manually diff against map)
2. Classify: `covered` ✅ | `missing` ⚠️ | `unmapped` 🔍
3. Update affected docs or justify skipping in PR description

### When to Update Game Docs

Update if change affects **player-visible behavior:** new mechanics, changed formulas, new character options, modified combat flow.

Skip for: refactoring, performance, gateway changes, test additions.

### When to Update Dev Docs

Update if change affects **developer understanding:** new packages/entities, changed patterns/flows, modified interfaces, new integration patterns.

Skip for: bug fixes (no design change), test additions, dependency bumps.

### Skip Justification Examples

- "Pure refactor — no behavioral change"
- "Test-only change"
- "Bug fix — docs already describe correct behavior"
