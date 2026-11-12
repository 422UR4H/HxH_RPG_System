# Sheet rules panel — design

**Date:** 2026-11-11 · **Status:** approved in brainstorm; builder check done
**Repos:** `System_X_System` (back, branch `feat/sheet-rules`) and `System_X_System_React`
(front, branch `feat/sheet-rules`). Both branches live in `.worktrees/sheet-rules` of each repo.

## 1. Intent

Testers get lost creating a character sheet: nothing on the site explains attributes, skills,
point distribution or classes. The rules already exist as player-facing game docs in
`System_X_System/docs/game/**`. This feature shows **those docs** inside the sheet, so editing a
doc (and deploying the back) updates what players read. No rule text is duplicated in the front.

The owner plans to turn `docs/game` into a rule **book**. So the docs stay book-shaped: rich
illustrations appear on the site, and the same Markdown still reads well on GitHub or in print.

## 2. Goals

1. A single **"?" button** at the top-right corner of the ATRIBUTOS column opens a **rules panel**
   with *all* rules a player needs for the sheet. It appears on all four sheet screens: view
   (`/charactersheet/:id`), create (`/charactersheet/new`), edit (`/charactersheet/:id/edit`) and
   create NPC (`/campaigns/:campaignId/npcs/new`). All four render `CharacterSheetTemplate`.
2. The panel content is the game docs, served by the back, rendered as Markdown with the styles
   designed in Claude Design.
3. Illustrations are the **site's own diagrams** (physical lozenge, mental lozenge, Nen
   principles) drawn with a new **glow effect** ("luz"). They are placed by markers in the docs.
4. Opening the rules never loses what the player typed in the sheet.
5. Fix the existing Markdown bug in the character background (`DescriptionMarkdown`): headings,
   bold and the like render as plain text.

## 3. Non-goals

- Variant A of the design (a full rules page) is **not** built. It is kept for a future site-wide
  rules page.
- Nothing about the class the player picked shows in the panel. The text is the same for everyone.
- The glow does **not** go onto the sheet's own diagrams yet. It is built reusable, but only the
  panel uses it now.
- No other "?" buttons (on PERÍCIAS, on the class field…). The one in ATRIBUTOS opens everything.
- No editing of rules on the site. No search.

## 4. Anti-goals (would be a failure even if it "works")

- **Losing the player's sheet edits** when opening, closing or pressing the browser back button.
- **New colors in the palette.** Only existing `palette` entries may be used (semantic aliases in
  `colors` pointing at existing palette entries are fine, see `src/styles/CLAUDE.md`).
- **Touching the global CSS reset** (`src/styles/ResetStyle.ts`). It affects the whole site.
- Rule text hardcoded in the front, except short UI copy (button labels, tooltips, loading and
  error messages).
- Developer content (code paths, links to `docs/dev`) visible to players.
- A glow that blinks hard. The motion must be subtle (see §8).

## 5. Content: which docs, in what order

The panel is **one continuous scroll** of chapters, one chapter per doc, in this fixed order:

| # | Chip label | Doc (`docs/game/…`) | Notes |
|---|---|---|---|
| 1 | Criação | `ficha-de-personagem/criacao.md` | **New doc**, see §5.1 |
| 2 | Classes | `classes.md` | edits in §5.2 |
| 3 | Habilidades | `ficha-de-personagem/habilidades.md` | |
| 4 | Atributos | `ficha-de-personagem/atributos.md` | diagram markers, §5.3 |
| 5 | Perícias | `ficha-de-personagem/pericias.md` | |
| 6 | Proficiências | `ficha-de-personagem/proficiencias.md` | |
| 7 | Experiência | `ficha-de-personagem/experiencia.md` | |
| 8 | Vida e estamina | `ficha-de-personagem/status.md` | |
| 9 | Nen | `ficha-de-personagem/sistema-nen.md` | **"Em breve"** badge, Nen diagram marker |
| 10 | Glossário | `glossario.md` | Sigla column, §5.2 |

`armas.md`, `dados.md`, combat docs and the like are out: they are not sheet rules.

The docs are already written for players. Only the **developer footer** must be hidden: the
`---` followed by the `> **🔧 Para Desenvolvedores**` blockquote that closes the doc
(`.github/instructions/docs-workflow.instructions.md`). The footer is **optional**:
`glossario.md` and the new `criacao.md` have none. Strip it only when present.

### 5.1 New doc `criacao.md`

A short chapter: five steps in sheet order. Each step links to the chapter that explains it.
1. Fill in the profile (nickname, name, birthday…).
2. Pick a class → Classes.
3. Distribute physical and mental attribute points until **Pontos para Distribuir** reaches zero → Atributos.
4. If the class offers them, choose the skills and weapons that get extra experience → Perícias, Proficiências.
5. Press **Criar Ficha**.

Base it on the Claude Design text (`ConteudoRegras.dc.html`, section `#resumo`).

### 5.2 Edits to existing docs (back repo)

- **`classes.md`**
  - Frame the class as **what the character dedicated years of their life to becoming, up to the
    moment of creation**.
  - State that **12 classes are available today**: Espadachim, Samurai, Ninja, Ladino, Netrunner,
    Pirata, Mercenário, Terrorista, Monge, Militar, Hunter and Mestre de Armas. These are the ones
    `CharacterClassFactory.Build()` registers, i.e. the list `GET /classes` returns to players.
  - Add that more are on the way, worded so a player does not think they are missing something.
- **`glossario.md`**
  - Add a **Sigla** column to the attribute tables, using the abbreviations the sheet shows: RES,
    AGI, FLX, SEN, Str, Cel, Dex, Con, RSL, ADP, JUD, CRE.
  - Mark Atleta, Tribal, Experimento and Circo as "(em breve)" in the classes table.
- **`habilidades.md`, `experiencia.md`**
  - Keep the "Velocidade de Progressão" tables and the XP-curve curiosity.
  - Make the sentence that explains them one or two short, plain lines: a bigger number means
    more XP to level up. No more detail than that.
- **`atributos.md`:** add a short "Como ler um atributo" section (the node shows *nível · pontos ·
  poder*) with the node marker, and the diagram markers of §5.3.
- **`sistema-nen.md`:** add the Nen principles diagram marker.
- **New `internal/domain/entity/character_class/CLAUDE.md`**
  - When a class is added to (or removed from) `Build()`, update the count and list in
    `docs/game/classes.md` and the "(em breve)" marks in `docs/game/glossario.md`.
  - Say why: those docs are shown to players in the sheet rules panel.
- **`docs/documentation-map.yaml`:** register `criacao.md` and the rules endpoint.

Precisão (glossary and `pericias.md`) was already fixed on this branch. Do not touch it again.

### 5.3 Diagram markers in the Markdown

A diagram goes in as a **fenced code block** whose info string starts with `diagram`. The body is
an ASCII fallback, which GitHub and print show as-is; the site replaces the whole block with the
lit diagram.

````markdown
```diagram physical paint=all
            Resistência
   Força                Constituição
Agilidade                     Sentido
   Celeridade             Destreza
           Flexibilidade
```
````

- **Names:** `physical`, `mental`, `nen-principles`, `attribute-node`.
- **`paint`:**
  - `all`; or
  - a comma list of nodes (sheet abbreviations: `RES`, `Str`, `RSL`, `Ken`, …) and segments
    written `A-B` between adjacent nodes (`AGI-Str`).
  - Anything not listed is lit **white**.
  - Without `paint`, the default is `all`.
- **Markers to place:**

  | Where | Marker |
  |---|---|
  | `atributos.md`, physical diamond (replaces the current ASCII block) | `physical paint=all` |
  | `atributos.md`, average (médio) explanation | `physical paint=RES,AGI,Str,AGI-Str,Str-RES` (Força = Resistência + Agilidade) |
  | `atributos.md`, sentence "um ponto em Agilidade sobe Força e Celeridade" | `physical paint=AGI,Str,Cel,AGI-Str,Cel-AGI` |
  | `atributos.md`, mental section | `mental paint=all` |
  | `atributos.md`, "Como ler um atributo" | `attribute-node` |
  | `sistema-nen.md` | `nen-principles paint=all` |

- Unknown names or a malformed `paint` must not crash. Render the ASCII block as plain code and log
  a warning in dev.

### 5.4 Chapter titles and links

- The doc's `# H1` becomes the chapter title. Inside the panel, heading levels shift down so the
  page outline stays valid.
- Links between game docs (`./atributos.md`, `../classes.md`, with or without `#anchor`) scroll
  inside the panel.
- Heading ids are `<slug>--<heading-slug>` (chapter ids are just `<slug>`), so anchors never collide
  across chapters. A link to `atributos.md#distribuicao-de-pontos` resolves to
  `atributos--distribuicao-de-pontos`.
- Links to anything else that survives stripping open in a new tab.

## 6. Back: serving the docs

- **Endpoint:** `GET /rules/character-sheet`, **public** (no auth).
  - Rules are not secret, and the future campaign `RulesSidebar` can reuse it.
  - Register it before `api.UseMiddleware(authMiddleware)`, like the auth and health routes.
  - Response is camelCase:

    ```json
    { "chapters": [ { "slug": "criacao", "label": "Criação", "title": "Criando seu personagem", "markdown": "…", "comingSoon": false } ] }
    ```

    - The order is as in §5.
    - `slug` is the file name without `.md`.
    - `label` is the short chip label from the §5 table, kept in the back's chapter list, so the
      front hardcodes no rule vocabulary.
    - `title` comes from the H1.
    - `markdown` is the body **without the H1 and without the developer footer**.
- **Embedding:** the docs are embedded in the binary with `go:embed`, so the served text is always
  the one of the deployed code version.
  - `go:embed` cannot reach parent directories, so the embedding Go file must sit at or above
    `docs/game`.
  - `docs/game/` must stay pure rulebook text (`docs-workflow.instructions.md`), so **no `.go` file
    inside it**.
  - Use a small package at the module root that embeds `docs/game/*.md` and
    `docs/game/ficha-de-personagem/*.md`. Exact name and location (root `.go` file or another
    approach that keeps `docs/game` clean) are the builder's choice.
  - **`.dockerignore` currently excludes `docs` and `*.md`.** Re-include `docs/game` (and its
    `.md` files), otherwise the Render Docker build breaks. Run a local `docker build` to prove it.
- **Chapter list:** an ordered list in Go code (slug, label, file, comingSoon). A missing file must fail
  the build or a test.
- **Footer stripping:** a pure function with unit tests.
  - Add a guard test asserting that no served chapter contains `internal/`, `docs/dev` or
    `Para Desenvolvedores`.
- **Contract:** `docs/dev/api/rules.md` (request, response, errors, auth, example), registered in
  `documentation-map.yaml`. Follow `.claude/rules/api-contract-author.md`. The back merges first:
  the contract is the handoff to the front.
- **Caching:** the content only changes on deploy. A `Cache-Control` with a modest max-age is fine
  (builder's choice).

## 7. Front: the panel (variant B of the Claude Design)

Reference files, copied from the Claude Design canvas into
`System_X_System_React/docs/audits/2026-11-11-briefing-regras-da-ficha/claude-design/`. Live
canvas: https://claude.ai/artifact/Cz8pspo4VQRtrtbtMvRK3R. The variant chosen and approved by the
designer is **B**: `B1`–`B4`, `BotaoAjuda`, `EstilosMarkdown`, `shared/regras.css`, plus the green
notes in `canvas.json`. `/_blob/…` images map to existing front assets: `questionmark-sm.svg`, the
red close X used by match/campaign panels, and the chevron.

**Never copy the Claude Design colors as hex.** Map each one to an existing token.

### 7.1 The "?" button

- **Asset:** `questionmark-sm.svg`. 60×60 on desktop, 52×52 on mobile; the icon itself is the hit
  target.
- **Position:** absolute at the top-right of the ATRIBUTOS column. In the distribute modes
  (create/edit/NPC) it is vertically centred on the "Pontos para Distribuir" block. In view mode
  it sits in the same corner. It takes no space in the flow: title and diagrams do not move.
- **Accessibility:** `aria-label` "Regras de atributos e perícias", `aria-expanded`, `aria-controls`.
- **Tooltip:** "Regras" when closed, **"Fechar regras" when the panel is open**. Clicking again
  closes.
- **Focus:** 2px `actionHighlight` ring, offset 3px.

### 7.2 Panel layout

- **≥ 1024px:** docked at the right, width `clamp(400px, 35vw, 520px)`, full height, fixed while
  the sheet scrolls.
  - The sheet takes the remaining width and **stays usable**: points can be placed with the panel
    open.
  - The sheet's container shrinks; it reflows, it is not overlaid. Its typography already scales by
    container queries (`cqi`). Anything in the template that keys off the *viewport* instead of its
    container must keep working at the reduced width.
  - Verify at 1024, 1280 and 1440, with no horizontal scroll.
  - Not modal (`role="complementary"`).
- **< 1024px:** a full-screen dialog over the mounted sheet (`role="dialog"`, `aria-modal`, focus
  trap).
- **Background:** `space.png`, like the sheet. Header 52px with the title "Regras · criação de
  personagem" (mobile: "Regras") and the red X that overflows the header, as in matches and
  campaigns.
- **Chips:** one per chapter, in the §5 order, on top of the content. The active chip follows the
  scroll (IntersectionObserver), and clicking a chip scrolls to its chapter.
- **Nen:** the chapter title and its chip carry the existing `ComingSoonBadge`.
- **Closing:** X, `Esc`, the "?" again, or the browser back button (§7.3). Focus returns to the "?".

### 7.3 State, URL and back button

- **Opening:** pushes a history entry with a query param, e.g. `?regras=atributos`, the slug of the
  chapter it opened at. Scrolling between chapters **never rewrites the URL**; it only updates the
  in-memory last chapter.
  - The **browser back button closes the panel** and keeps the player on the sheet.
  - Closing with X/Esc/"?" while the entry is ours calls `history.back()`, so no dead entry is left.
  - A URL that arrives with `?regras=<slug>` opens the panel at that chapter, which makes links
    shareable. Closing a panel that was opened this way does a `replace` that drops the param, since
    there is no entry of ours to go back to.
- **Must hold:** changing that query param must not remount the sheet or reset its form state.
  Verify on all four screens, including create with points already distributed.
- **Opening by itself for a first character.**
  - On `/charactersheet/new`, the panel opens automatically at **Criação** when the player's sheet
    list (`useCharacterSheets`) has **settled successfully and is empty**. This is the case the
    home already redirects to (`CharacterSheetsPage`).
  - It happens once per page mount. It is an ordinary open: it pushes the history entry, and the
    back button closes it.
  - **Trap:** never treat "not loaded yet" as empty. While the list is pending or fetching, or if it
    errored, the panel does not auto-open.
  - The NPC page never auto-opens.
- **Where the "?" opens:**
  - At **Atributos**, where the button is.
  - Within the same page visit, reopening returns to the last chapter shown. That state lives only
    in memory; **nothing is stored in the browser**.
  - The deep link `?regras=<slug>` wins over both.

### 7.4 Data loading

- **Fetching:** a React Query hook for `GET /rules/character-sheet`, keyed without a token since it
  is public, `staleTime: Infinity` for the session, `retry: 1`. Prefetching on hover/focus of the
  "?" is welcome.
- **Loading:** a skeleton or loader inside the panel. The panel opens immediately.
- **Error:** the existing error style (`errorBox`, dark box + 1px border + red600) with a "Tentar de
  novo" button.

### 7.5 Markdown rendering and the reset fix

- **Root cause of the background bug:** `ResetStyle.ts` sets `font: inherit` (and the like) on
  headings, `strong`, `em`, lists… `DescriptionMarkdown` never restores them.
- **Fix:** do **not** change the reset. Create **one** Markdown style base that restores, *scoped
  inside its own container*, heading sizes and weights, bold, italic, list markers and spacing, and
  table styles. Use it in `DescriptionMarkdown` (the background) and in the rules panel. Nothing
  outside these containers changes.
- **Panel styles on top of that base,** per `EstilosMarkdown.dc.html` / `regras.css`:
  - h2/h3 hierarchy;
  - 60–75ch line length;
  - inline `code` as a "sigla" chip, **in the panel only** (the background keeps the base code
    style);
  - **formula blockquote** (a blockquote whose content is italic only, as the docs write formulas);
  - **note box** (a blockquote that starts with a bold label such as `**Nota:**` or a "Curiosidade"
    heading);
  - tables that never scroll horizontally and stack into label/value rows on narrow widths.

  The builder picks the detection rule for formula vs note and documents it in the component.
- The markers of §5.3 are swapped in through a `code` renderer for the `diagram` info string.

### 7.6 Accessibility and responsiveness

- Contrast 4.5:1 for text.
- 44px touch targets.
- No horizontal page scroll at 390px.
- Headings form a valid outline.
- Diagrams have `role="img"` and an `aria-label` that says what is highlighted, in Portuguese
  (e.g. "Força acende junto com Resistência e Agilidade").

## 8. The glow ("luz") — the diagram module

Approved reference, open it in a browser:
`System_X_System_React/docs/audits/2026-11-11-briefing-regras-da-ficha/brilho-referencia.html`
(spike v7). The implementation must look like it. The rules that define it:

1. **One light.** Each glow layer is **one** blurred SVG group holding *all* pieces of the diagram
   (lines, beads, arcs, arrows). Opacity is applied to the group, never per piece, so beads and
   lines merge and never stack into seams. The reference uses four layers (blur 55/30/18/8, line
   widths 60/40/18/9, bead radius ×3.4/2.4/1.6/1.35, opacities ~.3/.6/.8/.9).
2. **Continuous white filament.** The drawing order is:
   - the bead body: a radial gradient with a near-white centre, then the colour, then fading to
     transparent at the edge;
   - then the white line cores (4px, plus a soft 7px white blur under them) running *through* the
     bead to its centre;
   - then a soft white centre disc (≈0.62 r).

   The white of a line and the white of a bead form one filament, with no break.
3. **Beads by size.** Edge beads (médios) glow less than vertex beads, in proportion to their radius.
4. **Paint = colour; rest = white.**
   - Painted parts use the diagram colour; unpainted parts are lit **white** (not dimmed, not
     transparent).
   - Within each layer, white pieces are drawn first and coloured pieces on top, slightly wider
     (×1.3 line, ×1.2 bead), so the colour spills over the white.
5. **Bleed rule (the module default).** When paint reaches a bead, the colour continues ~32% along
   each adjacent unpainted line, fading out (linear gradient from the bead outwards). This is the
   standard behaviour whenever "progress reaches a bead".
6. **Colours, existing palette only:**

   | Diagram | Colour |
   |---|---|
   | Physical | `red600` |
   | Mental | `purple500` (the magic-wall purple) |
   | Nen principles | `blue400` ("mana") |
   | Light: line cores and bead centres | `neutral0` (white) |
   | Light: unpainted halos | `neutral300` |

   - With the same layer values, a white halo reads much brighter than a coloured one. So the
     unpainted halos use `neutral300`, and the white bead body fades
     `neutral0` → `neutral100` → `neutral200` → transparent. Line cores and bead centres stay pure
     white. The reference (v7) shows the balance to match.
   - `red500` may appear inside the red bead gradient. Add semantic aliases in `colors` if needed;
   no new palette entries.
7. **Motion:**
   - On first appearance the diagram "ignites": lines draw in (~0.7s) and beads pop in sequence.
   - After that, each layer breathes slowly (4–6s), and each bead scales very subtly in its own
     rhythm (≤ ±5%).
   - Every ~2s one bead slowly opens its halo (~2.4s, ≤ +13%) and settles back.
   - **No flashes.**
   - Respect `prefers-reduced-motion`: show the final lit state, with no ignition and no breathing.
8. **Geometry** comes from the existing assets (`physical_lozange.svg`, `mental_lozenge.svg`,
   `nen_principles.svg`), so the diagrams match the sheet. The reference already transcribes the
   coordinates; the Nen arrows and arcs are the asset's own paths.
9. **In the panel,** the lit diagram replaces the sheet diagram's background image under the same
   abbreviation labels, so it looks like the sheet. Numbers and the ability centre (e.g.
   "PHYSICAL 3 4.5") are **hidden**; the `attribute-node` figure explains the numbers.
   - The `attribute-node` figure follows the Claude Design `NoAtributo` idea: the node plus three
     cards (Nível · Pontos · Poder).
   - It must look like the sheet's real node (same font sizes and the green points), not the
     distorted Claude Design version.
   - Reuse the sheet's real node markup, the `AttributeMarker` pieces inside `PhysicalsDiagram.tsx`.
     Extract it into a shared piece when that is clean (promote-don't-duplicate). It gets fixed
     example numbers.
10. **Reusable:** a self-contained module, roughly a `GlowDiagram` that takes geometry, paint and
    colour, plus a pure `resolvePaint(spec)` with unit tests. The owner will extend it later
    (painting on the sheet, progress).

## 9. Decisions made by me (the brainstorm owner), open to the owner's review

- **Endpoint:** public; path `/rules/character-sheet`; the response shape of §6.
- **Markers:** fenced `diagram` blocks with ASCII fallback; numbers hidden in the panel diagrams.
- **Nen colour:** `blue400`.
- **Chapters:** the ten of §5, in that order; the whole glossary is shown.
- **Where the panel opens:**
  - It opens by itself at Criação for a player with no sheets, on the create page, and only once the
    list is known to be empty.
  - The "?" opens at Atributos.
  - The last chapter is remembered only during the page visit, with no browser storage. This is
    the owner's suggestion, refined to avoid the "list not loaded yet" trap.
- **From the builder check:**
  - the footer is optional;
  - the `label` field;
  - the anchor id scheme;
  - URL `replace` on closing a deep link, and no URL rewrite on scroll;
  - the sheet reflows next to the docked panel;
  - the node figure reuses the real node markup;
  - the sigla chip is panel-only.
- **No Docker:** if `docker build` is unavailable locally, substitute a test that proves the files
  are embedded, and say so in the PR.

## 10. Left to the builder

- Go file and package placement for the embed.
- `Cache-Control` value.
- Formula/note detection rule.
- Component names and file layout, following `src/components/CLAUDE.md` (Atomic layers, the
  promote-don't-duplicate rule) and `src/features/sheet/CLAUDE.md`.
- Whether the panel's open state lives in a context or in the template.
- Loader visuals.

## 11. Verification (from the root CLAUDE.md "Entrega")

- **Back:**
  - unit tests (stripping, chapter list, guard);
  - handler test via `humatest`;
  - `curl` the endpoint locally;
  - local `docker build` proving the docs are embedded.
- **Front:**
  - tests for:
    - open/close, back button, focus return;
    - the auto-open rule: empty list settled → opens; pending, fetching, error or non-empty → does
      not;
    - paint resolution and Markdown mapping;
  - then the browser at 1440 and 390 on all four screens, including create with points already
    distributed (open, read, close, back button: nothing lost);
  - the background Markdown now shows headings and bold;
  - `prefers-reduced-motion` honoured.
- **PRs:** one per repo, cross-linked; back first.
