# Proficiency Distribution UI — Design Spec

**Date:** 2026-05-16  
**Scope:** Back-end API contract change + front-end creation flow for distributable proficiencies  
**Status:** Approved

---

## Problem

Classes with `distribution.proficiency_points` (Ninja, Mercenary, Hunter) fail character sheet creation because the front-end shows no UI for distributable proficiency slots. The back-end `ValidateProficiencies` rejects the request with a count mismatch error since zero proficiencies are sent.

---

## Back-end Change

**File:** `internal/app/api/sheet/character_class_response.go`

`DistributionResponse.ProficiencyPoints` changes from `[]int` to `[]LvlExp`:

```json
"distribution": {
  "proficiency_points": [
    { "level": 3, "exp": 210 },
    { "level": 2, "exp": 127 }
  ],
  "proficiencies_allowed": ["Dagger", "ThrowingDagger", "Scimitar", ...]
}
```

Level is computed once in `NewCharacterClassResponse` via `experience.NewDefaultExpTable().GetLvlByExp(xp)`. No changes to domain validation (`ValidateProficiencies` unchanged).

---

## Front-end Changes

### 1. Type: `types/characterClass.ts`

Add `DistributionProfPoint` and update `Distribution`:

```ts
export interface DistributionProfPoint {
  level: number;
  exp: number;
}

export interface Distribution {
  skillPoints: number | null;
  proficiencyPoints: DistributionProfPoint[]; // was number[]
  skillsAllowed: string[];
  proficienciesAllowed: string[];
}
```

### 2. Mode type: `features/sheet/types/proficiencyMode.ts`

```ts
export type ProficiencyMode = "view" | "edit" | "create";
```

### 3. Create page: `pages/CreateCharacterSheetPage.tsx`

```ts
proficiencyMode: "create", // was "view"
```

### 4. Template: `features/sheet/CharacterSheetTemplate.tsx`

Compute selected class and pass extra props to `ProficienciesList` in create mode:

```tsx
const selectedClass = charClasses?.find(cc => cc.profile.name === charSheet.characterClass);

<ProficienciesList
  mode={sheetMode.proficiencyMode}
  commonProfs={commonProficiencies}
  jointProfs={jointProficiencies}
  distribution={sheetMode.proficiencyMode === "create" ? selectedClass?.distribution : undefined}
  charSheet={sheetMode.proficiencyMode === "create" ? charSheet : undefined}
  setCharSheet={sheetMode.proficiencyMode === "create" ? setCharSheet : undefined}
/>
```

### 5. Component: `features/sheet/ProficienciesList.tsx`

**New props (create mode only):**
```ts
distribution?: Distribution;
charSheet?: CharacterSheet;
setCharSheet?: (s: CharacterSheet) => void;
```

**Internal state:**
```ts
const [slotSelections, setSlotSelections] = useState<string[]>(() =>
  Array(distribution?.proficiencyPoints.length ?? 0).fill("")
);

useEffect(() => {
  setSlotSelections(Array(distribution?.proficiencyPoints.length ?? 0).fill(""));
}, [distribution]);
```

**Rendering (mode === "create"):**

- Fixed proficiencies (`commonProfs`): shown as read-only cards, **excluding** weapons that appear in `distribution.proficienciesAllowed` to avoid duplication
- Distribution slots: one card per `proficiencyPoints[i]`, showing `Lv X · Y XP` and a `<select>` with `proficienciesAllowed` options
- Duplicate prevention: options chosen in other slots are rendered with `disabled`

**Layout:** cards side-by-side using `grid-template-columns: repeat(auto-fill, minmax(180px, 1fr))`, matching the existing proficiency card grid.

**On slot change:**
1. Remove old weapon from `commonProficiencies` (if any)
2. Add new: `commonProficiencies[weapon] = { exp: points[i].exp, level: points[i].level }`
3. Call `setCharSheet`

### 6. Validation: `pages/CreateCharacterSheetPage.tsx` — `validateCharSheet`

Before submitting, verify all distribution slots are filled. `charSheet.commonProficiencies` is the source of truth — each selected slot writes its weapon there with `exp > 0`.

```ts
const selectedClass = charClasses?.find(cc => cc.profile.name === charSheet.characterClass);
if (selectedClass?.distribution) {
  const d = selectedClass.distribution;
  const filled = d.proficienciesAllowed.filter(
    w => (charSheet.commonProficiencies[w]?.exp ?? 0) > 0
  ).length;
  if (filled < d.proficiencyPoints.length) {
    errors.push("Selecione todas as proficiências distribuíveis antes de criar a ficha.");
  }
}
```

---

## Data Flow

```
Class selected (header)
  → buildFromClass resets commonProficiencies to fixed profs only
  → CharacterSheetTemplate finds selectedClass
  → ProficienciesList receives new distribution (different reference)
  → useEffect resets slotSelections to []

User picks weapon in slot i
  → handleSlotChange removes old, adds new to commonProficiencies with correct exp
  → setCharSheet propagates up

User clicks "Criar Ficha"
  → validateCharSheet checks all slots filled (front-end)
  → characterSheetsService.createCharacterSheet sends proficiencies_exps: { weapon: exp }
  → back-end ValidateProficiencies accepts (count + values match)
```

---

## Unchanged

- `ValidateProficiencies` domain logic
- View/edit modes of `ProficienciesList`
- `characterSheetsService.createCharacterSheet` payload format
- All other sheet sections

---

## Classes Affected

| Class | proficiencyPoints | proficienciesAllowed count |
|-------|-------------------|---------------------------|
| Ninja | [127] | 3 |
| Mercenary | [210, 127] | 9 |
| Hunter | [127, 127] | 3 |
