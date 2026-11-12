# Sheet Rules Panel Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A "?" in the sheet's ATRIBUTOS column opens a rules panel that renders the game docs served
by the back. The sheet's three diagrams, and the ones in the panel, are drawn with the approved glow
effect.

**Architecture:**
- **Back:** embeds `docs/game/**` with `go:embed`. At boot it turns the 10 chapter files into
  `[]Chapter`, with the H1 split off and the developer footer stripped, and serves them on the public
  `GET /rules/character-sheet`.
- **Front:** a reusable glow-diagram module, `src/features/diagrams`, lights the sheet diagrams. A
  rules feature, `src/features/rules`, fetches the chapters, renders them with react-markdown, and
  shows them in a panel: docked at ≥1024px, a dialog below that. The panel's open state lives in the
  URL (`?regras=<slug>`).

**Tech Stack:**
- **Back:** Go 1.23, huma v2 + chi, `embed`/`io/fs`, `testing` + `humatest`.
- **Front:** React + TS strict, styled-components, React Query, react-router-dom v7,
  react-markdown 10 + remark-gfm 4 (already installed), vitest + Testing Library + MSW.

**Spec:** `System_X_System/docs/superpowers/specs/2026-11-11-sheet-rules-panel-design.md`
(PT-BR twin `.pt-br.md`). Read it first; this plan argues from it.

**Repos / branches:** both on `feat/sheet-rules`, in `.worktrees/sheet-rules` of each repo, already
rebased on `main`. Part A (back) ends in PR 1. Part B (front) ends in PR 2, cross-linked. Use MSW
in front tests, so Part B does not need PR 1 merged.

**References:**
- Glow reference, open it in a browser:
  `System_X_System_React/docs/audits/2026-11-11-briefing-regras-da-ficha/brilho-referencia.html`
  (v7).
- Claude Design files: `…/briefing-regras-da-ficha/claude-design/` (variant B).

## Global Constraints

- **Language:**
  - Code identifiers, file names and test titles in **English**.
  - UI copy, code comments and guard-test failure messages in **Portuguese**.
  - Game docs in PT-BR, player-facing, with no code references.
- **Colours:** never a hex/rgba in a component; colours only via `src/styles/tokens.ts`. **No new
  `palette` entries**: semantic aliases in `colors` pointing at existing palette entries only.
- **Global reset:** never edit `src/styles/ResetStyle.ts`.
- **Docs stay pure:** no `.go` file inside `docs/game/`.
- **Wire format:** camelCase JSON.
- **Diagram colours:**
  - physical `red600` (and `red500` only inside the red bead gradient);
  - mental `purple500`;
  - Nen principles `blue400`;
  - light cores `neutral0`;
  - unpainted halos `neutral300`;
  - white bead fade `neutral0 → neutral100 → neutral200 → transparent`.
- **Breakpoint:** `breakpoints.railUp` (1024) in `src/styles/breakpoints.ts`. Docked width:
  `clamp(400px, 35vw, 520px)`.
- **"?" button:**
  - 60×60 desktop, 52×52 mobile;
  - `aria-label="Regras de atributos e perícias"`;
  - tooltip "Regras" / "Fechar regras";
  - focus ring 2px `actionHighlight`, offset 3px.
- **Panel title:** "Regras · criação de personagem" (mobile: "Regras").
- **Chapter order, slugs and labels:**

  | Slug | Label | File | Coming soon |
  |---|---|---|---|
  | criacao | Criação | `ficha-de-personagem/criacao.md` | |
  | classes | Classes | `classes.md` | |
  | habilidades | Habilidades | `ficha-de-personagem/habilidades.md` | |
  | atributos | Atributos | `ficha-de-personagem/atributos.md` | |
  | pericias | Perícias | `ficha-de-personagem/pericias.md` | |
  | proficiencias | Proficiências | `ficha-de-personagem/proficiencias.md` | |
  | experiencia | Experiência | `ficha-de-personagem/experiencia.md` | |
  | status | Vida e estamina | `ficha-de-personagem/status.md` | |
  | sistema-nen | Nen | `ficha-de-personagem/sistema-nen.md` | yes |
  | glossario | Glossário | `glossario.md` | |

- **Glow layers (outer → inner):**

  | Blur | Line width | Bead radius | Opacity (lo → hi) | Breathing |
  |---|---|---|---|---|
  | 55 | 60 | ×3.4 | .26 → .34 | 6.0s |
  | 30 | 40 | ×2.4 | .52 → .66 | 5.2s |
  | 18 | 18 | ×1.6 | .72 → .86 | 4.4s |
  | 8 | 9 | ×1.35 | .80 → .95 | 3.8s |

  - Painted pieces are drawn after unpainted ones, ×1.3 line and ×1.2 bead, when the paint is not
    `all`.
  - Bleed: 32% of the adjacent unpainted segment.
  - Bead breathing ≤ ±5% (`1 + 0.05·r/28.05`).
  - A swell every ~1.8s lasts 2.4s, up to `1 + 0.13·r/28.05`.
  - Ignition: lines draw in 0.7s; beads pop with a 0.05s stagger; the glow fades in over 0.8s after
    0.5s.
  - `prefers-reduced-motion`: the final state, no animation.

## Review Focus

1. **Back button after the automatic open.**
   - A player with no sheets is redirected (`replace`) from `/charactersheets` to
     `/charactersheet/new`, and the panel opens by itself. Pressing back must close the panel and
     stay on the create page.
   - Pinned in B8 (`closes the auto-opened panel on browser back without leaving the page`).
2. **Points already distributed survive the panel.**
   - On the create page with 2 physical points placed: open, read, close (X, Esc, back). The points
     are still there.
   - Pinned in B8 (`keeps distributed points across open, close and back`).
3. **Same heading in several chapters.** "Visão Geral" exists in many docs. Ids must not collide,
   and a link to `atributos.md#distribuicao-de-pontos` must land in the right chapter.
   - Pinned in B6 (`prefixes heading ids with the chapter slug` and
     `routes a cross-doc anchor link to onNavigate`).
4. **A broken diagram fence.** `diagram physical paint=XYZ` or `diagram unknown` renders the ASCII
   fallback, warns in dev and never throws.
   - Pinned in B6 (`falls back to the ASCII block for an unknown diagram or bad paint`).
5. **The rules request fails or is slow.** The panel opens at once with a loader. An error shows the
   error box with "Tentar de novo", and the button refetches.
   - Pinned in B7 (`shows the error box and retries`).

---

# Part A — Back (`System_X_System`, PR 1)

### Task A1: Game doc content for the panel

**Files:**
- Create: `docs/game/ficha-de-personagem/criacao.md`
- Modify: `docs/game/classes.md`, `docs/game/glossario.md`,
  `docs/game/ficha-de-personagem/{habilidades,experiencia,atributos,sistema-nen}.md`
- Create: `internal/domain/entity/character_class/CLAUDE.md`

**Interfaces:**
- Produces: the doc files the A2 chapter list points to.
- Produces: the `diagram` fences B6 renders. They must follow the spec §5.3 grammar exactly.

- [ ] **Step 1: Write `criacao.md`**
  - H1: `# Criando seu personagem`.
  - One intro line, then an ordered list of the five steps from spec §5.1.
  - Each step ends with a relative link: `[Classes](../classes.md)`, `[Atributos](./atributos.md)`,
    `[Perícias](./pericias.md)`, `[Proficiências](./proficiencias.md)`.
  - No developer footer.
- [ ] **Step 2: `classes.md`**
  - Open "Visão Geral" with the framing: the class is what the character dedicated years of their
    life to becoming, up to the moment the sheet is created.
  - Add a section "Classes disponíveis" listing the 12 from spec §5.2, then one sentence saying new
    classes are on the way.
  - Do not list the four upcoming ones here.
- [ ] **Step 3: `glossario.md`**
  - Add a `Sigla` column to "Atributos Físicos" and "Atributos Mentais": RES, AGI, FLX, SEN, Str,
    Cel, Dex, Con / RSL, ADP, JUD, CRE.
  - In "Classes de Personagem", append " (em breve)" to the PT-BR names Atleta, Tribal, Experimento
    and Circo.
- [ ] **Step 4: `habilidades.md` and `experiencia.md`**
  - Under each "Velocidade de Progressão" table, keep or replace the explanation with at most two
    short sentences: a bigger number means more XP to level up.
  - Keep the tables and the "Curiosidade".
- [ ] **Step 5: `atributos.md`**
  - Replace the ASCII block with the `physical paint=all` fence. Its body is the same ASCII.
  - Add `## Como ler um atributo` before "Fórmulas": one sentence (the three numbers under the
    abbreviation are nível · pontos · poder), then the fence ` ```diagram attribute-node ` with the
    body `RES  nível · pontos · poder`.
  - Under "Atributos médios", after the first formula, the fence
    `physical paint=RES,AGI,Str,AGI-Str,Str-RES`, with the ASCII diamond as its body.
  - Right after the sentence "Por isso, um ponto em um primário fortalece também os dois médios
    vizinhos…", the fence `physical paint=AGI,Str,Cel,AGI-Str,Cel-AGI`.
  - In "Atributos Mentais", before the table, the fence `mental paint=all` with a 4-name ASCII
    diamond (Resiliência / Criatividade · Adaptabilidade / Julgamento).
- [ ] **Step 6: `sistema-nen.md`**
  - After the principles table, the fence `nen-principles paint=all`.
  - Its body: a one-line text list of the 11 principles.
- [ ] **Step 7: `character_class/CLAUDE.md`**
  - In PT-BR: when a class enters or leaves `Build()` in `character_class_factory.go`, update the
    count and the list in `docs/game/classes.md` and the "(em breve)" marks in
    `docs/game/glossario.md`.
  - Say why: those docs are shown to players in the sheet's rules panel.
- [ ] **Step 8: Verify**
  - Run: `grep -c '^```diagram' docs/game/ficha-de-personagem/atributos.md docs/game/ficha-de-personagem/sistema-nen.md`
  - Expected: `atributos.md:5`, `sistema-nen.md:1`.
- [ ] **Step 9: Commit**

  ```bash
  git add docs/game internal/domain/entity/character_class/CLAUDE.md
  git commit -m "docs(game): capítulo de criação, 12 classes, siglas e marcadores de diagrama"
  ```

### Task A2: Embedded docs and the chapter loader

**Files:**
- Create: `docs/embed.go` (package `docs`)
- Create: `internal/application/rules/chapters.go`
- Test: `internal/application/rules/chapters_test.go` (package `rules_test`)

**Interfaces:**
- Produces, in `docs`: `var GameFS embed.FS`, embedding `game/*.md` and
  `game/ficha-de-personagem/*.md`.
- Produces, in `rules` (import `github.com/422UR4H/HxH_RPG_System/internal/application/rules`):
  - `type ChapterSource struct { Slug, Label, Path string; ComingSoon bool }`. `Path` is relative to
    `GameFS`, e.g. `game/classes.md`.
  - `var SheetChapterSources []ChapterSource`: the 10 rows of the Global Constraints table, in
    order.
  - `type Chapter struct { Slug, Label, Title, Markdown string; ComingSoon bool }`
  - `var ErrMissingTitle = errors.New("rules: chapter has no H1 title")`
  - `func SplitTitle(md string) (title, body string, err error)`
  - `func StripDevFooter(md string) string`
  - `func LoadChapters(fsys fs.FS, sources []ChapterSource) ([]Chapter, error)`. It wraps any error
    with the path, using `%w`.

- [ ] **Step 1: Write the failing tests**
  - `TestSplitTitle`, table-driven:
    - `"# Atributos\n\nTexto"` → `("Atributos", "Texto", nil)`;
    - leading blank lines before the H1 are allowed;
    - `"## Sub\n"` → `ErrMissingTitle`;
    - `""` → `ErrMissingTitle`.
  - `TestStripDevFooter`, table-driven:
    - `"corpo\n\n---\n\n> **🔧 Para Desenvolvedores**\n>\n> x\n"` → `"corpo\n"`;
    - no footer → unchanged except trailing whitespace normalised to one `"\n"`;
    - a marker without a preceding `---` → cut from the marker line;
    - a `---` in the middle of the body with no marker after it → kept.
  - `TestLoadChapters_FromMapFS`: a `fstest.MapFS` with two files and two sources. Assert the order,
    Slug/Label/ComingSoon copied, Title from the H1, Markdown without the H1 and the footer.
  - `TestLoadChapters_MissingFile`: the error mentions the path.
  - `TestSheetChapters_RealDocs`, against `docs.GameFS` and `rules.SheetChapterSources`:
    - loads without error;
    - `len == 10`;
    - the slugs in the table order;
    - `chapters[8].ComingSoon == true`, every other one false;
    - **guard:** no chapter's Markdown contains `internal/`, `docs/dev` or `Para Desenvolvedores`.
      Failure message in PT: `"capítulo %s vaza conteúdo de desenvolvedor"`.
- [ ] **Step 2: Run:** `go test ./internal/application/rules/ -v`. Expected: compile failure, the
  package is missing.
- [ ] **Step 3: Implement**
  - `docs/embed.go`: `//go:embed game/*.md game/ficha-de-personagem/*.md`.
  - `chapters.go`:
    - `SplitTitle`: the first non-blank line must start with `"# "`.
    - `StripDevFooter`: find the last line beginning with `> **🔧 Para Desenvolvedores**`. Drop it
      and everything after it. Then drop the trailing blank lines and one trailing `---` line before
      it. Then `strings.TrimRight` and add one `"\n"`.
- [ ] **Step 4: Run:** `go test ./internal/application/rules/ -v && go vet ./...`. Expected: PASS.
- [ ] **Step 5: Commit**

  ```bash
  git add docs/embed.go internal/application/rules
  git commit -m "feat(rules): embute docs/game e monta os capítulos da ficha"
  ```

### Task A3: Public endpoint `GET /rules/character-sheet`

**Files:**
- Create: `internal/app/api/rules/routes.go`, `internal/app/api/rules/get_sheet_rules.go`
- Test: `internal/app/api/rules/get_sheet_rules_test.go`
- Modify: `internal/app/api/api.go` (new field + registration before the auth middleware)
- Modify: `internal/app/api/routes_registration_test.go`
- Modify: `cmd/api/main.go` (load chapters at boot; `log.Fatal` style used there on error)

**Interfaces:**
- Consumes: `rules.LoadChapters`, `rules.SheetChapterSources`, `docs.GameFS`, `rules.Chapter` (A2).
- Produces the wire shape B5 consumes:

  ```go
  type RuleChapterResponse struct {
      Slug       string `json:"slug"`
      Label      string `json:"label"`
      Title      string `json:"title"`
      Markdown   string `json:"markdown"`
      ComingSoon bool   `json:"comingSoon"`
  }
  type GetSheetRulesBody struct{ Chapters []RuleChapterResponse `json:"chapters"` }
  type GetSheetRulesResponse struct {
      CacheControl string `header:"Cache-Control"` // "public, max-age=300"
      Body         GetSheetRulesBody
  }
  func GetSheetRulesHandler(chapters []rules.Chapter) func(context.Context, *struct{}) (*GetSheetRulesResponse, error)
  type Api struct{ GetSheetRulesHandler func(context.Context, *struct{}) (*GetSheetRulesResponse, error) }
  func (a *Api) RegisterRoutes(r *chi.Mux, api huma.API, logger *zap.Logger)
  ```

- `api.Api` gains `RulesHandler IApi`. `Routes` calls `a.RulesHandler.RegisterRoutes(r, api, a.Logger)`
  right after `a.AuthHandler.RegisterRoutes`, before `api.UseMiddleware`.

- [ ] **Step 1: Write the failing tests**
  - `TestGetSheetRulesHandler`, humatest with two fixture chapters:
    - 200;
    - `Cache-Control` = `public, max-age=300`;
    - the JSON has the keys `chapters[0].slug/label/title/markdown/comingSoon` with the fixture
      values, in order.
  - In `routes_registration_test.go`:
    - add `RulesHandler: &rulesHandler.Api{}` to the existing graph;
    - add `TestRoutes_SheetRulesArePublic`. Build `api.Api` with stub handlers and a real
      `RulesHandler` over the fixture chapters. Pass a middleware that writes 401 and never calls
      `next`. Serve with `httptest`.
    - `GET /rules/character-sheet` → 200, and `GET /classes` → 401.
- [ ] **Step 2: Run:** `go test ./internal/app/api/... -run 'SheetRules|AllHandlers' -v`. Expected:
  FAIL.
- [ ] **Step 3: Implement** the handler, routes (`Tags: ["rules"]`, `Errors: [500]`), the `api.go`
  change, and the `main.go` wiring. In `main.go`, `LoadChapters` errors abort startup.
- [ ] **Step 4: Run:** `go test ./... && go vet ./...`. Expected: PASS.
- [ ] **Step 5: Commit**

  ```bash
  git add internal/app/api cmd/api/main.go
  git commit -m "feat(rules): GET /rules/character-sheet público"
  ```

### Task A4: Build context, contract, map, smoke

**Files:**
- Modify: `.dockerignore` (after `docs`, add `!docs/embed.go` and `!docs/game`)
- Create: `docs/dev/api/rules.md`
- Modify: `docs/documentation-map.yaml`

**Interfaces:**
- Produces: the contract B5 reads (`docs/dev/api/rules.md`).

- [ ] **Step 1: Prove the Docker build sees the docs**
  - Run: `docker build -t sxs-rules-check .`
  - Expected: success. A missing embed pattern fails the compile, so a successful build is the
    proof.
  - No Docker on the machine: say so in the PR, and rely on `TestSheetChapters_RealDocs` plus a
    `go build ./cmd/api` run from a temporary copy made with the same ignore rules.
- [ ] **Step 2: Write the contract**
  - Follow `.claude/rules/api-contract-author.md`: purpose, auth (none), request, response with a
    real example (one short chapter), `Cache-Control`, errors (500), field notes:
    - `markdown` has no H1 and no developer footer;
    - `diagram` fences follow the spec §5.3 grammar;
    - order is meaningful.
- [ ] **Step 3: Map entries**
  - `code_path: internal/application/rules` → `docs/dev/api/rules.md`, directly affected.
  - `code_path: docs/game` notes that `SheetChapterSources` lists the files the panel shows.
  - Add `criacao.md` where `classes.md` is mapped for character classes.
- [ ] **Step 4: Smoke**
  - Run: `make run-dev`, then
    `curl -s localhost:5000/rules/character-sheet | jq '.chapters | map(.slug)'`.
  - Expected: the 10 slugs in order.
  - Then:
    `curl -s localhost:5000/rules/character-sheet | grep -c 'Para Desenvolvedores'`.
  - Expected: `0`.
- [ ] **Step 5: Commit, then finish the branch** (`superpowers:finishing-a-development-branch`)
  - Open PR 1. Its description says what was verified, including the docker build or its
    substitute.

  ```bash
  git add .dockerignore docs/dev/api/rules.md docs/documentation-map.yaml
  git commit -m "docs(rules): contrato, mapa de docs e build Docker com docs/game"
  ```

---

# Part B — Front (`System_X_System_React`, PR 2)

### Task B1: One Markdown style base; fixes the background

**Files:**
- Create: `src/styles/markdown.ts`
- Modify: `src/components/molecules/DescriptionMarkdown.tsx`
- Test: `src/styles/__tests__/markdown.test.tsx`

**Interfaces:**
- Produces: `export const markdownBase: RuleSet<object>` (a styled-components `css` fragment), for
  use *inside* a container's styles.
  - It restores, under the container: `h1–h4` font-size/weight; `strong, b { font-weight: bold }`;
    `em, i { font-style: italic }`; `ul { list-style: disc }`; `ol { list-style: decimal }`; list
    padding; paragraph/list margins; table borders.
  - It takes the existing rules from `DescriptionMarkdown`'s `MarkdownContainer` and adds the
    missing ones.

- [ ] **Step 1: Write the failing test** `restores typography the reset removes`.
  - Render `<DescriptionMarkdown source={"# T\n\n**b** *i*\n\n- x"} />` with `ServerStyleSheet`, as
    `src/test/readResetCss.tsx` does.
  - Assert the CSS, whitespace stripped, contains `strong,b{font-weight:bold`,
    `em,i{font-style:italic`, `ul{list-style:disc` and an `h1{` rule with a `font-size`.
  - Also assert the DOM has `h1`, `strong` and `em`.
- [ ] **Step 2: Run:** `npx vitest run src/styles/__tests__/markdown.test.tsx`. Expected: FAIL.
- [ ] **Step 3: Implement** `markdownBase`, and use it inside `MarkdownContainer`.
  `ResetStyle.ts` is untouched.
- [ ] **Step 4: Run:** `npx vitest run src/styles src/components/molecules`. Expected: PASS.
- [ ] **Step 5: Commit:** `fix(markdown): devolve títulos, negrito e listas dentro do Markdown (background)`

### Task B2: Diagram tokens, geometry and paint resolution

**Files:**
- Modify: `src/styles/tokens.ts`
- Create: `src/features/diagrams/geometry.ts`, `src/features/diagrams/paint.ts`
- Test: `src/features/diagrams/__tests__/paint.test.ts`

**Interfaces:**
- Produces, in `colors`, each with a PT comment:

  | Alias | Palette entry |
  |---|---|
  | `diagramPhysical` | `red600` |
  | `diagramPhysicalWarm` | `red500` |
  | `diagramMental` | `purple500` |
  | `diagramNen` | `blue400` |
  | `diagramLight` | `neutral0` |
  | `diagramLightHalo` | `neutral300` |
  | `diagramLightFadeNear` | `neutral100` |
  | `diagramLightFadeFar` | `neutral200` |

- Produces, in `geometry.ts`:

  ```ts
  export interface DiagramNode { id: string; x: number; y: number; r: number }
  export type DiagramShape =
    | { kind: "ellipse"; cx: number; cy: number; rx: number; ry: number }
    | { kind: "path"; d: string };
  export interface DiagramGeometry {
    viewBox: string;
    nodes: DiagramNode[];
    segments: [string, string][];
    shapes: DiagramShape[];
  }
  export const PHYSICAL_GEOMETRY: DiagramGeometry;      // viewBox "0 0 1280 1024"
  export const MENTAL_GEOMETRY: DiagramGeometry;        // same viewBox and vertices as physical, NO edge beads
  export const NEN_PRINCIPLES_GEOMETRY: DiagramGeometry; // viewBox "0 0 1280 1024"
  export type DiagramName = "physical" | "mental" | "nen-principles";
  export const GEOMETRY_BY_NAME: Record<DiagramName, DiagramGeometry>;
  ```

  - **Physical:** nodes and segments transcribed from `PHYS` in `brilho-referencia.html`, ids =
    sheet keys (RES, SEN, FLX, AGI, Str, Con, Dex, Cel).
  - **Mental:**
    - nodes RSL = RES (639.26, 107.8), ADP = SEN (1046.51, 521.36), JUD = FLX (641.45, 915.73),
      CRE = AGI (234.2, 519.32), all r = 28.05;
    - segments RSL-ADP, ADP-JUD, JUD-CRE, CRE-RSL.

    These match `MentalsDiagram`'s label positions, which are tuned to `physical_lozange.svg`.
  - **Nen:**

    | Node | Position |
    |---|---|
    | Ken | (357.26, 93.45) |
    | Ryu | (898.77, 103.67) |
    | Ren | (362.37, 293.62) |
    | Gyo | (897.49, 295.67) |
    | Ten | (362.62, 686.05) |
    | Zts | (898.03, 686.05) |
    | Shu | (361.86, 931.16) |
    | In | (897.26, 931.16) |
    | En | (118.28, 507.84) |
    | Kou | (1115.56, 507.84) |

    - r = 34.1 for all of them.
    - Segments: Ryu-Gyo, Kou-Gyo, Kou-Zts, Ken-Ren, En-Ren, Ten-En, Ten-Shu, Zts-In.
    - Shapes: the ellipse `(616.72, 511.87, 498.34, 493.64)` plus the six `<path d>` copied
      verbatim from `src/assets/diagrams/nen_principles.svg`.

- Produces, in `paint.ts`:

  ```ts
  export type PaintSpec = "all" | { nodes: string[]; segments: [string, string][] };
  export interface Bleed { from: { x: number; y: number }; to: { x: number; y: number } }
  export interface ResolvedPaint {
    all: boolean;
    isNodePainted(id: string): boolean;
    isSegmentPainted(a: string, b: string): boolean; // order-insensitive
    bleeds: Bleed[];
  }
  export const BLEED_FRACTION = 0.32;
  export function parsePaintSpec(raw: string | undefined, g: DiagramGeometry): PaintSpec | null;
  export function resolvePaint(g: DiagramGeometry, spec: PaintSpec): ResolvedPaint;
  ```

  - `parsePaintSpec`: `undefined` or `"all"` → `"all"`. Otherwise it splits on commas and trims.
    `A-B` is a segment; anything else is a node. It returns `null` if a node id is unknown or a
    segment is not in `g.segments` (either order).
  - `resolvePaint`: a bleed goes from each painted node along each adjacent **unpainted** segment,
    `BLEED_FRACTION` of the way to the other end. There are no bleeds when the spec is `"all"`.

- [ ] **Step 1: Write the failing tests** in `paint.test.ts`:
  - `parses all and undefined as all`
  - `parses nodes and segments`: `"RES,AGI,Str,AGI-Str,Str-RES"` gives nodes `[RES,AGI,Str]` and
    segments `[[AGI,Str],[Str,RES]]`.
  - `rejects unknown node or non-adjacent segment`: `"XYZ"` → null; `"RES-FLX"` → null.
  - `segment painting is order-insensitive`
  - `computes bleeds for Força`: the Força spec gives exactly 2 bleeds. RES→Con ends at
    (708.6976, 175.4928), AGI→Cel ends at (297.2912, 580.488). Use `toBeCloseTo(…, 3)`.
  - `no bleeds when all`
  - `mental geometry has four nodes and no edge beads`
- [ ] **Step 2: Run:** `npx vitest run src/features/diagrams`. Expected: FAIL.
- [ ] **Step 3: Implement** `tokens.ts` aliases, `geometry.ts` and `paint.ts`.
- [ ] **Step 4: Run:** `npx vitest run src/features/diagrams src/styles`. Expected: PASS, including
  the `noRawColors` guard.
- [ ] **Step 5: Commit:** `feat(diagrams): geometria dos diagramas e resolução da pintura`

### Task B3: `GlowDiagram` — the light

**Files:**
- Create: `src/features/diagrams/GlowDiagram.tsx`, `src/features/diagrams/glowLayers.ts`
- Test: `src/features/diagrams/__tests__/GlowDiagram.test.tsx`

**Interfaces:**
- Consumes: B2 `DiagramGeometry`, `PaintSpec`, `resolvePaint`, the colour tokens.
- Produces:

  ```ts
  export type DiagramColor = "physical" | "mental" | "nen";
  export interface GlowDiagramProps {
    geometry: DiagramGeometry;
    color: DiagramColor;
    paint?: PaintSpec;   // default "all"
    label?: string;      // present → role="img" + aria-label; absent → aria-hidden (decorative)
  }
  export default function GlowDiagram(props: GlowDiagramProps): JSX.Element;
  ```

  - `glowLayers.ts` exports `GLOW_LAYERS` (the four rows of Global Constraints) and the motion
    constants.

- Rendering rules (spec §8, all of them; the reference HTML is the visual oracle):
  - `<svg viewBox overflow="visible">`, so halos may exceed the box.
  - Gradient ids are unique per instance (`useId`).
  - Four `<g data-glow-layer filter=…>` groups. Each holds **every** piece: shapes, segments, bead
    circles.
    - Unpainted pieces first, in `colors.diagramLightHalo`; painted pieces after, in the diagram
      colour.
    - Bleed lines last, stroked with a per-bleed `linearGradient`:
      colour opacity 1 → .6 at .55 → 0.
  - Core, in order:
    - bead bodies with the radial gradient for their colour (red:
      white .16 → `diagramPhysicalWarm` .34 → `diagramPhysical` .55 → faded .78 → transparent;
      mental/Nen: white .16 → colour .46 → faded .75 → transparent; white: white .3 →
      `diagramLightFadeNear` .6 → `diagramLightFadeFar` faded .8 → transparent);
    - then the line cores: a soft 7px white at .45 opacity with a 2.5 blur, plus a crisp 4px white;
    - then the shapes in white;
    - then the centre discs: radius .62·r, white solid until .45, then transparent.
  - Motion: CSS keyframes (layer opacity breathing, bead scale with `transform-box: fill-box`,
    ignition), plus a JS swell: every 1.8s add a class to one random node's beads for 2.4s.
    - With reduced motion: no swell interval, `data-motion="off"`, final state rendered.
    - Off-screen (IntersectionObserver): the swell interval stops and the CSS animations pause
      (`animation-play-state: paused` via `data-paused`).

- [ ] **Step 1: Write the failing tests**
  - `renders four glow layers holding every node and segment`: for physical geometry, each
    `[data-glow-layer]` has 8 circles and 8 lines.
  - `paints with the diagram colour and lights the rest white`: the Força spec. A painted bead's
    fill = `colors.diagramPhysical`; an unpainted one = `colors.diagramLightHalo`.
  - `draws one bleed gradient per bleed`: the Força spec gives 2 `linearGradient` with
    `gradientUnits="userSpaceOnUse"`.
  - `is decorative without a label and an img with one`
  - `turns motion off under prefers-reduced-motion`: use `mockReducedMotion(true)` from
    `src/test/reducedMotion.ts`. Then `data-motion="off"`, and with fake timers no node gets the
    swell class after 5s.
- [ ] **Step 2: Run:** `npx vitest run src/features/diagrams`. Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run tests.** Expected: PASS.
- [ ] **Step 5: Visual check against the reference**
  - Run `npm run dev`.
  - Mount a temporary local preview of the four reference cases: physical all; physical Força;
    mental all; Nen all.
  - Compare side by side with `brilho-referencia.html` at the same size.
  - Remove the preview before committing.
- [ ] **Step 6: Commit:** `feat(diagrams): GlowDiagram com a luz aprovada`

### Task B4: The sheet's diagrams get the light (+ a "rules" variant)

**Files:**
- Modify: `src/features/sheet/PhysicalsDiagram.tsx`, `MentalsDiagram.tsx`,
  `NenPrinciplesDiagram.tsx`
- Create: `src/features/sheet/AttributeNode.tsx` (extracted from `PhysicalsDiagram`'s marker:
  label + level/points/power line)
- Test: `src/features/sheet/__tests__/DiagramsGlow.test.tsx`; the existing
  `AttributeDiagramsPoints.test.tsx` must keep passing.

**Interfaces:**
- Consumes: B3 `GlowDiagram`; B2 `PHYSICAL_GEOMETRY`, `MENTAL_GEOMETRY`, `NEN_PRINCIPLES_GEOMETRY`,
  `PaintSpec`.
- Produces: new optional props on all three diagram components:
  `variant?: "sheet" | "rules"` (default `"sheet"`) and `paint?: PaintSpec` (default `"all"`).
  - `"rules"` keeps the labels and their positions as in view mode. It hides the numbers, the
    ability centre, +/− and "Pontos para Distribuir".
- Produces:
  `export default function AttributeNode(props: { label: string; level: number; points: number; power: number; size?: "sheet" | "figure" })`.
  It renders exactly the markup and styles `PhysicalsDiagram` used for a marker. Physicals and
  Mentals use it.

- [ ] **Step 1: Before screenshots**
  - Screenshot create, view and edit at 1440 and 390 (test account `test@mail.com` / `12345678`).
  - Save them in the scratchpad.
- [ ] **Step 2: Write the failing tests**
  - `sheet diagrams render the glow instead of the image`: no `img[src*=lozange]`; one
    `[data-glow-layer]` set per diagram.
  - `mental diagram has no edge beads`
  - `rules variant shows labels without numbers or buttons`
  - `AttributeNode renders level, points and power`
- [ ] **Step 3: Run:** `npx vitest run src/features/sheet`. Expected: the new tests FAIL.
- [ ] **Step 4: Implement**
  - Replace each `<img>` inside `SVGContainer` with
    `<GlowDiagram geometry=… color=… paint={paint} />`, decorative on the sheet.
  - `SVGContainer` gets `overflow: visible`.
  - Positions, the y shifts of the distribute modes, and every other rule of the pixel-tuned zone
    (`src/components/CLAUDE.md`) stay as they are.
- [ ] **Step 5: Run:** `npm run test && npm run build`. Expected: PASS.
- [ ] **Step 6: After screenshots and a performance check**
  - Same screens and sizes as Step 1: labels, numbers and +/− must not move.
  - In DevTools Performance with 4× CPU throttle, type in a profile field and click +/− for 5s.
    Expected: no long tasks from style recalculation over 50ms.
  - If it janks, keep the ignition and the swell but move the breathing to a single opacity
    animation on the `<svg>` element.
- [ ] **Step 7: Commit:** `feat(sheet): diagramas da ficha com a luz`

### Task B5: Rules data — type, service, hook

**Files:**
- Create: `src/types/rules.ts`, `src/services/rulesService.ts`, `src/hooks/useSheetRules.ts`
- Modify: `src/test/handlers.ts` (MSW handler + fixture)
- Test: `src/hooks/__tests__/useSheetRules.test.tsx`

**Interfaces:**
- Consumes: the `docs/dev/api/rules.md` contract (A4).
- Produces:

  ```ts
  export interface RuleChapter { slug: string; label: string; title: string; markdown: string; comingSoon: boolean }
  export const rulesService: { getSheetRules(): Promise<RuleChapter[]> }   // GET /rules/character-sheet, no auth config
  export function useSheetRules(): UseQueryResult<RuleChapter[]>          // key ["sheetRules"], staleTime Infinity, retry 1
  export const sheetRulesFixture: RuleChapter[] // in src/test/fixtures/rules.ts: criacao, atributos (with a diagram fence and a "Visão Geral" h2), sistema-nen (comingSoon)
  ```

- [ ] **Step 1: Failing test** `loads chapters from the public endpoint without a token`: render the
  hook with `renderWithProviders` and `token: null`; expect the data to equal the fixture.
- [ ] **Step 2: Run.** Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run.** Expected: PASS.
- [ ] **Step 5: Commit:** `feat(rules): cliente do GET /rules/character-sheet`

### Task B6: `RulesMarkdown` — chapters, ids, links, quotes, tables, diagrams

**Files:**
- Create: `src/features/rules/RulesMarkdown.tsx`, `src/features/rules/markdownMapping.ts`,
  `src/features/rules/DiagramBlock.tsx`, `src/features/rules/AttributeNodeFigure.tsx`,
  `src/features/diagrams/describePaint.ts`
- Test: `src/features/rules/__tests__/RulesMarkdown.test.tsx`,
  `src/features/rules/__tests__/markdownMapping.test.ts`

**Interfaces:**
- Consumes:
  - B1 `markdownBase`;
  - B2 `parsePaintSpec`, `GEOMETRY_BY_NAME`, `DiagramName`;
  - B4 diagrams with `variant="rules"` and `AttributeNode`;
  - B5 `RuleChapter`.
- Produces:

  ```ts
  // markdownMapping.ts (pure)
  export function slugify(text: string): string; // NFD, strip accents, lowercase, non [a-z0-9] → "-", collapse, trim "-"
  export function headingId(chapterSlug: string, text: string): string; // `${chapterSlug}--${slugify(text)}`
  export function resolveDocLink(href: string): { slug: string; anchor?: string } | null; // "*.md[#a]" relative → slug = basename w/o .md, anchor slugified; else null
  export function classifyBlockquote(text: { firstStrong: boolean; onlyEmphasis: boolean }): "formula" | "note" | "plain";
  export function parseDiagramFence(lang: string | undefined, meta: string | undefined): { name: DiagramName | "attribute-node"; paintRaw?: string } | null;
  export function rehypeTableLabels(): (tree: Root) => void; // copies each column's <th> text into data-label on its <td>s

  // describePaint.ts
  export function describePaint(name: DiagramName, spec: PaintSpec): string; // PT aria-label, e.g. "Diagrama físico com destaque em Resistência, Agilidade e Força"

  // RulesMarkdown.tsx
  export default function RulesMarkdown(props: { chapter: RuleChapter; onNavigate(slug: string, anchor?: string): void }): JSX.Element;
  ```

- Rendering rules:
  - **Headings:** the body's `##` renders as `h3`, `###` as `h4` and so on, since the panel renders
    the chapter title as `h2`. Every heading has `id = headingId(chapter.slug, text)`.
  - **Links:** a doc link calls `onNavigate`. Any other link gets `target="_blank" rel="noopener noreferrer"`.
  - **Blockquotes:** formula → class `md-formula`; note → class `md-note`.
  - **Inline `code`:** the sigla chip.
  - **Tables:** in a non-scrolling wrapper; they stack under 480px container width, using
    `data-label`.
  - **Fenced `diagram`:** `DiagramBlock`, with the meta from react-markdown's `node.data.meta`.
    - A test proves the meta arrives. If it does not, read it via a tiny remark plugin that copies
      `meta` to `data.hProperties["data-meta"]`.
  - **`DiagramBlock`:**
    - A valid name and paint → the matching sheet diagram with `variant="rules"`, wrapped in
      `role="img"` + `describePaint`.
    - `attribute-node` → `AttributeNodeFigure`: `AttributeNode` AGI 3 · 2 · 8, plus three cards
      "Nível — sobe com a experiência", "Pontos — os seus: + põe, − tira", "Poder — o que conta
      nos testes".
    - Invalid → `<pre><code>{fallback}</code></pre>` and, in dev, `console.warn`.
  - **Styles:** per `claude-design/EstilosMarkdown.dc.html` and `shared/regras.css`, mapped to tokens.

- [ ] **Step 1: Write the failing tests**
  - In `markdownMapping.test.ts`:
    - `slugify("Distribuição de Pontos") === "distribuicao-de-pontos"`;
    - `resolveDocLink("../classes.md")` → `{slug:"classes"}`;
    - `resolveDocLink("./atributos.md#Distribuição-de-Pontos")` →
      `{slug:"atributos", anchor:"distribuicao-de-pontos"}`;
    - `resolveDocLink("https://x.y")` → `null`;
    - `parseDiagramFence("diagram","physical paint=RES,AGI")` →
      `{name:"physical", paintRaw:"RES,AGI"}`;
    - `parseDiagramFence("js", "")` → `null`;
    - `classifyBlockquote` covers its three outputs;
    - `rehypeTableLabels` adds a `data-label` to every `td`.
  - In `RulesMarkdown.test.tsx`:
    - `prefixes heading ids with the chapter slug`: "Visão Geral" in the atributos fixture →
      `id="atributos--visao-geral"`;
    - `routes a cross-doc anchor link to onNavigate`;
    - `renders a diagram fence as a lit diagram with a Portuguese label`;
    - `falls back to the ASCII block for an unknown diagram or bad paint` (no throw; `pre` present);
    - `marks formula and note blockquotes`.
- [ ] **Step 2: Run.** Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run** `npx vitest run src/features/rules src/features/diagrams`. Expected: PASS.
- [ ] **Step 5: Commit:** `feat(rules): renderização dos capítulos (ids, links, fórmulas, notas, tabelas, diagramas)`

### Task B7: `SheetRulesPanel` — the panel UI

**Files:**
- Create: `src/features/rules/SheetRulesPanel.tsx`, `src/features/rules/useActiveChapter.ts`
- Test: `src/features/rules/__tests__/SheetRulesPanel.test.tsx`

**Interfaces:**
- Consumes:
  - B5 `useSheetRules`;
  - B6 `RulesMarkdown`;
  - `ComingSoonBadge`, `CloseButton` (red X; overflow the header like `EdgeDrawer`);
  - the `errorBox` css;
  - `useMediaQuery`, `breakpoints.railUp`.
- Produces:

  ```ts
  export interface SheetRulesPanelProps {
    startSlug: string;                     // chapter to scroll to on open
    onClose(): void;
    onChapterChange(slug: string): void;   // active chapter while scrolling
  }
  export default function SheetRulesPanel(props: SheetRulesPanelProps): JSX.Element;
  export const RULES_PANEL_WIDTH = "clamp(400px, 35vw, 520px)";
  export function useActiveChapter(slugs: string[], root: HTMLElement | null): string | undefined;
  ```

- Behaviour:
  - **≥ `railUp`:** an `aside` with `role="complementary"` and `aria-labelledby` the title, fixed to
    the right, `RULES_PANEL_WIDTH` wide, full height, background `space.png`.
  - **Below `railUp`:** `role="dialog" aria-modal="true"`, full screen, focus kept inside (Tab
    cycles), body scroll locked.
  - **Header:** 52px with the title and `CloseButton` (`aria-label` "Fechar regras").
  - **Chips:** `nav aria-label="Sumário das regras"` with one chip per chapter (`label`), marked
    with `aria-current` when active. A click scrolls to the chapter.
  - **Chapters:** each is a `section id={slug}` with an `h2` title. `ComingSoonBadge` goes next to
    the title and on the chip when `comingSoon`.
  - **Esc** calls `onClose`.
  - **Loading:** a loader in the content area.
  - **Error:** `errorBox` with "Não foi possível carregar as regras." and the button
    "Tentar de novo" (`refetch`).
  - **Opening:** scroll to `startSlug` once the data is there.

- [ ] **Step 1: Write the failing tests**
  - `renders chips and chapters in order with the coming-soon badge on Nen`
  - `scrolls to the start chapter`: assert `scrollIntoView` was called on `#atributos`; mock it.
  - `is a modal dialog below 1024px and a complementary panel above`
  - `closes on Escape and on the X`
  - `shows the error box and retries`: MSW returns 500, then 200.
- [ ] **Step 2: Run.** Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run.** Expected: PASS.
- [ ] **Step 5: Commit:** `feat(rules): painel de regras (encaixado e diálogo)`

### Task B8: Open state, URL, "?" button, template, first-character auto-open

**Files:**
- Create: `src/features/rules/useRulesPanel.ts`, `src/features/rules/RulesHelpButton.tsx`
- Modify: `src/features/sheet/CharacterSheetTemplate.tsx`, `src/pages/CreateCharacterSheetPage.tsx`
- Test: `src/features/rules/__tests__/useRulesPanel.test.tsx`,
  `src/pages/__tests__/CreateCharacterSheetPage.rules.test.tsx`

**Interfaces:**
- Consumes: B7 `SheetRulesPanel`, `RULES_PANEL_WIDTH`; `Tooltip`; `useCharacterSheets`.
- Produces:

  ```ts
  export const RULES_PARAM = "regras";
  export function useRulesPanel(): {
    openSlug: string | null;          // from ?regras
    open(slug: string): void;         // push ?regras=slug with state { rulesOpenedHere: true }
    close(): void;                    // state.rulesOpenedHere ? navigate(-1) : replace without the param
    lastSlug: string | null;          // in-memory last active chapter (no storage)
    setLastSlug(slug: string): void;
  };
  export default function RulesHelpButton(props: { open: boolean; onToggle(): void; size: "desktop" | "mobile" }): JSX.Element; // forwards ref
  ```

  - `CharacterSheetTemplate` gets `data.autoOpenRules?: boolean`.
    - It renders `RulesHelpButton` absolutely inside `AttributesSection`: the top-right corner,
      centred on "Pontos para Distribuir" in the distribute modes.
    - It renders the panel when `openSlug` is set.
    - When docked and open, the sheet wrapper gets `padding-right: RULES_PANEL_WIDTH`. The sheet
      reflows; it is not covered.
    - **Nothing is rendered when `sheetMode.embedded`** (the match).
  - The "?" opens at `lastSlug ?? "atributos"`.
  - Auto-open:
    - `CreateCharacterSheetPage` passes
      `autoOpenRules = sheets.isSuccess && !sheets.isFetching && sheets.data.length === 0`.
    - The template opens `"criacao"` once per mount (ref guard), only when no `?regras` is present.
  - Focus returns to the "?" after any close.

- [ ] **Step 1: Write the failing tests**
  - `useRulesPanel`, in a MemoryRouter with history:
    - `open pushes the param and back closes it`;
    - `close after open goes back instead of leaving a dead entry`;
    - `close on a deep link replaces the URL without the param`;
    - `scrolling never rewrites the URL`: `setLastSlug` leaves `location.search` unchanged.
  - Page tests, on `/charactersheet/new` with MSW:
    - `auto-opens at Criação when the sheet list is empty`;
    - `does not auto-open while the list is pending or on error`: delay or 500 the list;
    - `does not auto-open when the player already has sheets`;
    - `closes the auto-opened panel on browser back without leaving the page` (Review Focus 1);
    - `keeps distributed points across open, close and back` (Review Focus 2): click + twice on
      RES, then open, close, open, history back; RES still shows 2 points;
    - `the "?" opens at Atributos and its tooltip reads "Fechar regras" while open`;
    - `returns focus to the "?" after closing`;
    - `renders no "?" in the embedded match sheet`.
- [ ] **Step 2: Run.** Expected: FAIL.
- [ ] **Step 3: Implement.**
- [ ] **Step 4: Run:** `npm run test && npm run lint && npm run build`. Expected: PASS.
- [ ] **Step 5: Commit:** `feat(rules): "?" na coluna de atributos, URL e abertura no primeiro personagem`

### Task B9: Docs, end-to-end verification, PR 2

**Files:**
- Modify: `src/features/sheet/CLAUDE.md`:
  - the diagrams use `features/diagrams/GlowDiagram`;
  - the `variant`/`paint` props;
  - the mental has no edge beads by design;
  - the panel's URL contract.
- Create: `src/features/diagrams/CLAUDE.md` (PT): the glow rules, in short, with a link to spec §8;
  the bleed rule is the module default; colours only via the aliases.

- [ ] **Step 1: Write the two CLAUDE.md updates.**
- [ ] **Step 2: Browser, end to end**
  - Run the back from its branch (`make run-dev`) and `npm run dev` on 5173.
  - At 1440, 1280, 1024 and 390, on view, create, edit and NPC create:
    - "?" placement, open/close, chips, scroll tracking, back button;
    - no horizontal scroll;
    - the docked sheet stays usable: place a point with the panel open.
  - A test account with no sheets: home → auto-open at Criação.
  - The background Markdown shows headings and bold.
  - Reduced motion (DevTools rendering emulation): no animation.
  - Use headless Chrome via CDP if the extension is unavailable. Never log out the owner's tabs.
- [ ] **Step 3: Commit the docs, finish the branch** (`superpowers:finishing-a-development-branch`)
  - Open PR 2, cross-linked with PR 1. Say what was verified and what was not.
  - Run `./dev-checkout.sh feat/sheet-rules` from `System_X_System_Project/` only if something was
    left unverified.
