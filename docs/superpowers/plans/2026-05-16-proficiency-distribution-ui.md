# Proficiency Distribution UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Permitir que classes com proficiências distribuíveis (Ninja, Mercenary, Hunter) criem fichas com sucesso, mostrando selects card-by-card na aba de proficiências do modo criação.

**Architecture:** O back-end já expõe `distribution.proficiency_points` como `[]LvlExp` (level + exp). O front recebe esses pontos, renderiza um card/select por slot, grava as escolhas em `charSheet.commonProficiencies`, e valida que todos os slots foram preenchidos antes de submeter.

**Tech Stack:** Go 1.23 (back-end), React + TypeScript + styled-components (front-end), Vite build

---

## Estado atual (já aplicado nesta sessão)

As mudanças abaixo foram feitas antes da sessão de brainstorming e **já estão no disco** — não reescrever:

- `internal/app/api/sheet/character_class_response.go` — `DistributionResponse.ProficiencyPoints` é `[]LvlExp`; level calculado via `experience.NewDefaultExpTable().GetLvlByExp(xp)`
- `System_X_System_React/src/types/characterClass.ts` — `DistributionProfPoint` + `Distribution.proficiencyPoints: DistributionProfPoint[]`
- `System_X_System_React/src/features/sheet/types/proficiencyMode.ts` — `"create"` adicionado ao union type
- `System_X_System_React/src/pages/CreateCharacterSheetPage.tsx` — `proficiencyMode: "create"`

---

## Mapa de arquivos

| Arquivo | Ação |
|---------|------|
| `internal/app/api/sheet/character_class_response_test.go` | Criar — test unitário do `NewCharacterClassResponse` com distribution |
| `System_X_System_React/src/features/sheet/CharacterSheetTemplate.tsx` | Modificar — encontrar `selectedClass`, passar props extras ao `ProficienciesList` |
| `System_X_System_React/src/features/sheet/ProficienciesList.tsx` | Modificar — adicionar props create mode, cards distribuíveis, prevenção de duplicata |
| `System_X_System_React/src/pages/CreateCharacterSheetPage.tsx` | Modificar — `validateCharSheet` verifica slots distribuíveis preenchidos |

---

## Task 1: Teste unitário para `NewCharacterClassResponse` com distribution

**Files:**
- Create: `internal/app/api/sheet/character_class_response_test.go`

- [ ] **Step 1: Escrever o teste**

Crie o arquivo `internal/app/api/sheet/character_class_response_test.go`:

```go
package sheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/app/api/sheet"
	cc "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_class"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestNewCharacterClassResponse_distributionLevels(t *testing.T) {
	profsAllowed := []enum.WeaponName{enum.Dagger, enum.Scimitar}
	dist := &cc.Distribution{
		ProficiencyPoints:    []int{210, 127},
		ProficienciesAllowed: profsAllowed,
	}
	profile := *cc.NewClassProfile(enum.Mercenary, "", "test mercenary", "")
	charClass := *cc.NewCharacterClass(profile, dist, nil, nil, nil, nil, nil, nil, nil)
	hs := buildTestHalfSheet(t)

	resp := sheet.NewCharacterClassResponse(hs, charClass)

	if resp.Distribution == nil {
		t.Fatal("expected non-nil distribution")
	}
	if len(resp.Distribution.ProficiencyPoints) != 2 {
		t.Fatalf("expected 2 proficiency_points, got %d", len(resp.Distribution.ProficiencyPoints))
	}

	wantExp := []int{210, 127}
	for i, pt := range resp.Distribution.ProficiencyPoints {
		if pt.Exp != wantExp[i] {
			t.Errorf("point[%d].Exp = %d, want %d", i, pt.Exp, wantExp[i])
		}
		if pt.Level <= 0 {
			t.Errorf("point[%d].Level = %d, want > 0 (level must be computed)", i, pt.Level)
		}
	}
	// maior XP deve dar nível maior ou igual
	if resp.Distribution.ProficiencyPoints[0].Level < resp.Distribution.ProficiencyPoints[1].Level {
		t.Errorf("expected first point (210 XP) to have level >= second point (127 XP)")
	}
	if len(resp.Distribution.ProficienciesAllowed) != 2 {
		t.Fatalf("expected 2 proficiencies_allowed, got %d", len(resp.Distribution.ProficienciesAllowed))
	}
}

func TestNewCharacterClassResponse_nilDistribution(t *testing.T) {
	profile := *cc.NewClassProfile(enum.Samurai, "", "test samurai", "")
	charClass := *cc.NewCharacterClass(profile, nil, nil, nil, nil, nil, nil, nil, nil)
	hs := buildTestHalfSheet(t)

	resp := sheet.NewCharacterClassResponse(hs, charClass)

	if resp.Distribution != nil {
		t.Errorf("expected nil distribution for class without distribution, got %+v", resp.Distribution)
	}
}
```

- [ ] **Step 2: Rodar o teste**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System
go test ./internal/app/api/sheet/... -run TestNewCharacterClassResponse -v
```

Expected: PASS (o código já foi implementado; o teste verifica que os levels são > 0 e que 210 XP dá nível ≥ 127 XP).

- [ ] **Step 3: Confirmar que todos os testes do pacote passam**

```bash
go test ./internal/app/api/sheet/... -v
```

Expected: PASS em todos.

- [ ] **Step 4: Vet**

```bash
go vet ./internal/app/api/sheet/...
```

Expected: sem output (zero erros).

- [ ] **Step 5: Commit**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System
git add internal/app/api/sheet/character_class_response.go \
        internal/app/api/sheet/character_class_response_test.go
git commit -m "$(cat <<'EOF'
feat(api): return level alongside exp in distribution proficiency_points

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

---

## Task 2: `CharacterSheetTemplate` — conectar classe selecionada ao `ProficienciesList`

**Files:**
- Modify: `System_X_System_React/src/features/sheet/CharacterSheetTemplate.tsx`

- [ ] **Step 1: Adicionar cálculo de `selectedClass` e novos props**

No início da função `CharacterSheetTemplate`, após a desestruturação das props, adicione:

```tsx
const selectedClass = charClasses?.find(
  (cc) => cc.profile.name === charSheet.characterClass
);
```

- [ ] **Step 2: Atualizar a chamada de `ProficienciesList`**

Substitua:

```tsx
<ProficienciesList
  mode={sheetMode.proficiencyMode}
  commonProfs={commonProficiencies}
  jointProfs={jointProficiencies}
/>
```

Por:

```tsx
<ProficienciesList
  mode={sheetMode.proficiencyMode}
  commonProfs={commonProficiencies}
  jointProfs={jointProficiencies}
  distribution={sheetMode.proficiencyMode === "create" ? selectedClass?.distribution : undefined}
  charSheet={sheetMode.proficiencyMode === "create" ? charSheet : undefined}
  setCharSheet={sheetMode.proficiencyMode === "create" ? setCharSheet : undefined}
/>
```

- [ ] **Step 3: Verificar build TypeScript**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System_React
npm run build 2>&1 | tail -20
```

Expected: `✓ built in` sem erros de tipo. Se houver erro de "unused variable `selectedClass`", verifique que o prop foi adicionado na chamada do step anterior.

---

## Task 3: `ProficienciesList` — modo create com cards distribuíveis

**Files:**
- Modify: `System_X_System_React/src/features/sheet/ProficienciesList.tsx`

- [ ] **Step 1: Substituir o arquivo completo**

Substitua o conteúdo de `System_X_System_React/src/features/sheet/ProficienciesList.tsx` por:

```tsx
import { useState, useEffect } from "react";
import styled from "styled-components";
import type { ProficiencyMode } from "./types/proficiencyMode";
import type { JointProficiency, CharacterSheet } from "../../types/characterSheet";
import type { Distribution } from "../../types/characterClass";

interface ProficienciesListProps {
  mode: ProficiencyMode;
  commonProfs?: Record<string, { level: number }>;
  jointProfs?: JointProficiency[];
  distribution?: Distribution;
  charSheet?: CharacterSheet;
  setCharSheet?: (s: CharacterSheet) => void;
}

export default function ProficienciesList({
  mode,
  commonProfs,
  jointProfs,
  distribution,
  charSheet,
  setCharSheet,
}: ProficienciesListProps) {
  const [slotSelections, setSlotSelections] = useState<string[]>(() =>
    Array(distribution?.proficiencyPoints.length ?? 0).fill("")
  );

  useEffect(() => {
    setSlotSelections(Array(distribution?.proficiencyPoints.length ?? 0).fill(""));
  }, [distribution]);

  const handleSlotChange = (slotIndex: number, newWeapon: string) => {
    if (!charSheet || !setCharSheet || !distribution) return;
    const oldWeapon = slotSelections[slotIndex];
    const next = [...slotSelections];
    next[slotIndex] = newWeapon;
    setSlotSelections(next);

    const updatedProfs = { ...charSheet.commonProficiencies };
    if (oldWeapon) delete updatedProfs[oldWeapon];
    if (newWeapon) {
      const point = distribution.proficiencyPoints[slotIndex];
      updatedProfs[newWeapon] = { exp: point.exp, level: point.level };
    }
    setCharSheet({ ...charSheet, commonProficiencies: updatedProfs });
  };

  const distributableSet = new Set(distribution?.proficienciesAllowed ?? []);
  const fixedProfs =
    mode === "create" && distribution
      ? Object.fromEntries(
          Object.entries(commonProfs ?? {}).filter(([name]) => !distributableSet.has(name))
        )
      : commonProfs;

  return (
    <ProficienciesListContainer>
      {fixedProfs &&
        Object.entries(fixedProfs).map(([name, { level }]) => (
          <ProficiencyItem key={name}>
            <ProficiencyName>
              {name.charAt(0).toUpperCase() + name.slice(1)}
            </ProficiencyName>
            <ProficiencyLevel>Level: {level}</ProficiencyLevel>
          </ProficiencyItem>
        ))}

      {mode === "create" &&
        distribution?.proficiencyPoints.map((point, i) => {
          const otherSelected = slotSelections.filter((_, j) => j !== i);
          return (
            <DistributionSlot key={i} $selected={!!slotSelections[i]}>
              <SlotLevel>
                Lv {point.level}{" "}
                <SlotExp>· {point.exp} XP</SlotExp>
              </SlotLevel>
              <SlotSelect
                value={slotSelections[i]}
                onChange={(e) => handleSlotChange(i, e.target.value)}
              >
                <option value="">Escolher arma…</option>
                {distribution.proficienciesAllowed.map((weapon) => (
                  <option
                    key={weapon}
                    value={weapon}
                    disabled={otherSelected.includes(weapon)}
                  >
                    {weapon}
                  </option>
                ))}
              </SlotSelect>
            </DistributionSlot>
          );
        })}

      {jointProfs &&
        jointProfs.length > 0 &&
        jointProfs.map(({ name, level }) => (
          <ProficiencyItem key={name}>
            <ProficiencyName>
              {name.charAt(0).toUpperCase() + name.slice(1)}
            </ProficiencyName>
            <ProficiencyLevel>Level: {level}</ProficiencyLevel>
          </ProficiencyItem>
        ))}
    </ProficienciesListContainer>
  );
}

const ProficienciesListContainer = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 300px));
  gap: 15px;
`;

const ProficiencyItem = styled.div`
  font-size: 24px;
  background-color: #444;
  border-radius: 6px;
  padding: 15px;
  display: flex;
  justify-content: space-between;
  align-items: center;
`;

const ProficiencyName = styled.div`
  font-family: "Roboto", sans-serif;
  font-weight: 500;
  font-size: min(22px, 5cqi);
`;

const ProficiencyLevel = styled.div`
  font-family: "Roboto", sans-serif;
  font-weight: 400;
  font-size: min(22px, 5cqi);
  color: #9f9f9f;
`;

const DistributionSlot = styled.div<{ $selected: boolean }>`
  background-color: #444;
  border-radius: 6px;
  padding: 15px;
  border: 2px solid ${({ $selected }) => ($selected ? "#107135" : "#666")};
  display: flex;
  flex-direction: column;
  gap: 10px;
`;

const SlotLevel = styled.div`
  font-family: "Roboto", sans-serif;
  font-weight: 700;
  font-size: min(22px, 5cqi);
  color: white;
`;

const SlotExp = styled.span`
  font-weight: 400;
  font-size: 0.8em;
  color: #9f9f9f;
`;

const SlotSelect = styled.select`
  background-color: #555;
  color: white;
  border: 1px solid #666;
  border-radius: 4px;
  padding: 8px;
  font-family: "Roboto", sans-serif;
  font-size: min(18px, 4cqi);
  cursor: pointer;
  width: 100%;

  &:focus {
    outline: none;
    border-color: #107135;
  }

  option:disabled {
    color: #555;
  }
`;
```

- [ ] **Step 2: Verificar build TypeScript**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System_React
npm run build 2>&1 | tail -20
```

Expected: `✓ built in` sem erros.

- [ ] **Step 3: Verificar lint**

```bash
npm run lint 2>&1 | tail -20
```

Expected: sem erros (warnings de `any` são aceitáveis se já existiam).

---

## Task 4: Validação front-end dos slots distribuíveis

**Files:**
- Modify: `System_X_System_React/src/pages/CreateCharacterSheetPage.tsx`

- [ ] **Step 1: Adicionar validação em `validateCharSheet`**

Localize a função `validateCharSheet` em `src/pages/CreateCharacterSheetPage.tsx`. Ela já valida outros campos. Adicione o bloco abaixo **antes de** `return errors.length > 0 ? errors.join("\n") : null;`:

```tsx
const selectedClass = charClasses?.find(
  (cc) => cc.profile.name === charSheet.characterClass
);
if (selectedClass?.distribution) {
  const d = selectedClass.distribution;
  const filled = d.proficienciesAllowed.filter(
    (w) => (charSheet.commonProficiencies[w]?.exp ?? 0) > 0
  ).length;
  if (filled < d.proficiencyPoints.length) {
    errors.push("Selecione todas as proficiências distribuíveis antes de criar a ficha.");
  }
}
```

- [ ] **Step 2: Verificar build TypeScript**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System_React
npm run build 2>&1 | tail -20
```

Expected: `✓ built in` sem erros.

---

## Task 5: Commit front-end + smoke test manual

- [ ] **Step 1: Commit das mudanças front-end**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System_React
git add src/types/characterClass.ts \
        src/features/sheet/types/proficiencyMode.ts \
        src/features/sheet/CharacterSheetTemplate.tsx \
        src/features/sheet/ProficienciesList.tsx \
        src/pages/CreateCharacterSheetPage.tsx
git commit -m "$(cat <<'EOF'
feat(sheet): add distributable proficiency selection in create mode

Show one card/select per proficiency slot, prevent duplicate weapon
selection across slots, validate all slots filled before submission.

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
EOF
)"
```

- [ ] **Step 2: Rodar o dev server**

```bash
cd /home/azzurah/Documentos/HxH_RPG_Environment_Project/System_X_System_Project/System_X_System_React
npm run dev
```

- [ ] **Step 3: Smoke test — classe sem distribuição (ex: Samurai)**

1. Abrir `http://localhost:5173/createcharactersheet`
2. Selecionar **Samurai**
3. Seção de proficiências deve mostrar apenas o card fixo "Katana — Level: X"
4. Nenhum select deve aparecer

- [ ] **Step 4: Smoke test — classe com 1 slot (Ninja)**

1. Selecionar **Ninja**
2. Seção de proficiências deve mostrar:
   - Card fixo: "ThrowingDagger — Level: 2"
   - 1 card distribuível: "Lv 2 · 127 XP" com select (opções: Dagger, Katana, Katar)
3. Sem selecionar nada, clicar "Criar Ficha" → mensagem de erro "Selecione todas as proficiências distribuíveis"
4. Selecionar uma arma → borda do card fica verde
5. Preencher outros campos obrigatórios e criar → deve funcionar

- [ ] **Step 5: Smoke test — classe com 2 slots e prevenção de duplicata (Mercenary)**

1. Selecionar **Mercenary**
2. Dois cards distribuíveis: "Lv 3 · 210 XP" e "Lv 2 · 127 XP"
3. Selecionar "Dagger" no primeiro slot
4. No segundo slot, a opção "Dagger" deve aparecer desabilitada (não clicável)
5. Selecionar outra arma no segundo slot → ambos os cards ficam com borda verde
6. Preencher outros campos e criar → deve funcionar sem erro 422

- [ ] **Step 6: Smoke test — troca de classe limpa os slots**

1. Selecionar Mercenary, escolher armas nos dois slots
2. Trocar para Ninja (1 slot)
3. O slot anterior deve estar vazio (sem seleção prévia mantida)
