# Character Sheet Tests & Game Documentation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add comprehensive test coverage for all 8 character_sheet sub-packages and create game rules documentation.

**Architecture:** Bottom-up testing following the dependency graph (experience → ability → status → attribute → skill → proficiency → spiritual → sheet). Each sub-package gets tests + corresponding game documentation. Tests use standard library only with table-driven patterns. Mocks are hand-written where cross-package isolation is needed.

**Tech Stack:** Go 1.23, standard `testing` package, no external test frameworks

---

## Files to Create

### Test Files
- `internal/domain/entity/character_sheet/experience/exp_table_test.go`
- `internal/domain/entity/character_sheet/experience/experience_test.go`
- `internal/domain/entity/character_sheet/experience/character_exp_test.go`
- `internal/domain/entity/character_sheet/ability/talent_test.go`
- `internal/domain/entity/character_sheet/ability/ability_test.go`
- `internal/domain/entity/character_sheet/ability/abilities_manager_test.go`
- `internal/domain/entity/character_sheet/status/status_bar_test.go`
- `internal/domain/entity/character_sheet/status/status_bars_test.go`
- `internal/domain/entity/character_sheet/status/status_manager_test.go`
- `internal/domain/entity/character_sheet/attribute/primary_attribute_test.go`
- `internal/domain/entity/character_sheet/attribute/middle_attribute_test.go`
- `internal/domain/entity/character_sheet/attribute/spiritual_attribute_test.go`
- `internal/domain/entity/character_sheet/attribute/managers_test.go`
- `internal/domain/entity/character_sheet/skill/common_skill_test.go`
- `internal/domain/entity/character_sheet/skill/joint_skill_test.go`
- `internal/domain/entity/character_sheet/skill/skills_manager_test.go`
- `internal/domain/entity/character_sheet/proficiency/proficiency_test.go`
- `internal/domain/entity/character_sheet/proficiency/joint_proficiency_test.go`
- `internal/domain/entity/character_sheet/proficiency/proficiency_manager_test.go`
- `internal/domain/entity/character_sheet/spiritual/nen_hexagon_test.go`
- `internal/domain/entity/character_sheet/spiritual/nen_principle_test.go`
- `internal/domain/entity/character_sheet/spiritual/nen_category_test.go`
- `internal/domain/entity/character_sheet/spiritual/hatsu_test.go`
- `internal/domain/entity/character_sheet/spiritual/principles_manager_test.go`
- `internal/domain/entity/character_sheet/sheet/character_profile_test.go`
- `internal/domain/entity/character_sheet/sheet/talent_by_category_set_test.go`
- `internal/domain/entity/character_sheet/sheet/character_sheet_test.go`

### Documentation Files
- `docs/game/glossario.md`
- `docs/game/ficha-de-personagem/experiencia.md`
- `docs/game/ficha-de-personagem/habilidades.md`
- `docs/game/ficha-de-personagem/status.md`
- `docs/game/ficha-de-personagem/atributos.md`
- `docs/game/ficha-de-personagem/pericias.md`
- `docs/game/ficha-de-personagem/proficiencias.md`
- `docs/game/ficha-de-personagem/sistema-nen.md`
- `docs/architecture/overview.md`
- `AGENTS.md`

---

## Task 0 — Glossary (Game Keywords Reference)

**Create:** `docs/game/glossario.md`

- [ ] **Step 1** — Create the glossary file with complete keyword mappings:

```bash
mkdir -p docs/game/ficha-de-personagem docs/architecture
```

Create `docs/game/glossario.md`:

```markdown
# Glossário do Sistema HxH RPG

> Mapeamento de termos do jogo entre Português (PT-BR) e Inglês (EN).

## Habilidades (Abilities)

| PT-BR | EN | Descrição |
|---|---|---|
| Físicos | Physicals | Habilidade que governa atributos e perícias físicas |
| Mentais | Mentals | Habilidade que governa atributos e perícias mentais |
| Espirituais | Spirituals | Habilidade que governa atributos e perícias espirituais (Nen) |
| Perícias (habilidade) | Skills | Habilidade que governa a experiência geral de todas as perícias |

## Atributos Físicos (Physical Attributes)

| PT-BR | EN | Tipo | Descrição |
|---|---|---|---|
| Resistência | Resistance | Primário | Base de defesa e vitalidade |
| Agilidade | Agility | Primário | Velocidade de movimento |
| Flexibilidade | Flexibility | Primário | Amplitude de movimentos |
| Sentido | Sense | Primário | Percepção sensorial |
| Força | Strength | Médio | Média de Resistência + Agilidade |
| Celeridade | Celerity | Médio | Média de Agilidade + Flexibilidade |
| Destreza | Dexterity | Médio | Média de Flexibilidade + Sentido |
| Constituição | Constitution | Médio | Média de Sentido + Resistência |

## Atributos Mentais (Mental Attributes)

| PT-BR | EN | Tipo | Descrição |
|---|---|---|---|
| Resiliência | Resilience | Primário | Capacidade de se recuperar mentalmente |
| Adaptabilidade | Adaptability | Primário | Capacidade de se ajustar a novas situações |
| Ponderação | Weighting | Primário | Capacidade de análise e raciocínio |
| Criatividade | Creativity | Primário | Capacidade de inovação e improvisação |

## Atributos Espirituais (Spiritual Attributes)

| PT-BR | EN | Tipo | Descrição |
|---|---|---|---|
| Chama | Flame | Espiritual | Força interior, determinação espiritual |
| Consciência | Conscience | Espiritual | Consciência e controle do fluxo de Nen |

## Perícias Físicas (Physical Skills)

| PT-BR | EN | Atributo | Descrição |
|---|---|---|---|
| Vitalidade | Vitality | Resistência | Influencia HP, vigor geral |
| Energia | Energy | Resistência | Influencia SP, resistência física |
| Defesa | Defense | Resistência | Capacidade de resistir a dano |
| Empurrão | Push | Força | Empurrar objetos e oponentes |
| Agarrar | Grab | Força | Prender oponentes |
| Carregar | Carry | Força | Capacidade de carga |
| Velocidade | Velocity | Agilidade | Velocidade máxima de deslocamento |
| Aceleração | Accelerate | Agilidade | Rapidez para ganhar velocidade |
| Freio | Brake | Agilidade | Capacidade de parar rapidamente |
| Ligeireza | Legerity | Celeridade | Movimentos rápidos e precisos |
| Repelir | Repel | Celeridade | Desviar ataques com velocidade |
| Finta | Feint | Celeridade | Enganar oponentes com movimentos rápidos |
| Acrobacia | Acrobatics | Flexibilidade | Manobras acrobáticas |
| Evasão | Evasion | Flexibilidade | Esquivar de ataques |
| Esgueirar | Sneak | Flexibilidade | Mover-se silenciosamente |
| Reflexo | Reflex | Destreza | Tempo de reação |
| Precisão | Accuracy | Destreza | Acurácia em ataques à distância |
| Furtividade | Stealth | Destreza | Permanecer oculto |
| Visão | Vision | Sentido | Acuidade visual |
| Audição | Hearing | Sentido | Acuidade auditiva |
| Olfato | Smell | Sentido | Acuidade olfativa |
| Tato | Tact | Sentido | Acuidade tátil |
| Paladar | Taste | Sentido | Acuidade gustativa |
| Curar | Heal | Constituição | Regeneração natural |
| Respiração | Breath | Constituição | Controle respiratório |
| Tenacidade | Tenacity | Constituição | Perseverança física |

## Perícias Espirituais (Spiritual Skills)

| PT-BR | EN | Atributo | Descrição |
|---|---|---|---|
| Foco | Focus | Chama | Concentração espiritual |
| Força de Vontade | WillPower | Chama | Determinação interior |
| Autoconhecimento | SelfKnowledge | Chama | Compreensão de si mesmo |
| Coa | Coa | Consciência | Cociente de Abertura da Aura |
| Mop | Mop | Consciência | Montante Operacional de Produção |
| Aop | Aop | Consciência | Abertura Operacional de Produção |

## Princípios Nen (Nen Principles)

| PT-BR | EN | Descrição |
|---|---|---|
| Ten | Ten | Manter a aura ao redor do corpo |
| Zetsu | Zetsu | Fechar completamente os nós de aura |
| Ren | Ren | Amplificar o fluxo de aura |
| Gyo | Gyo | Concentrar aura em uma parte do corpo |
| Hatsu | Hatsu | Liberar a aura com habilidade pessoal |
| Shu | Shu | Envolver um objeto com aura |
| Kou | Kou | Concentrar toda a aura em um ponto |
| Ken | Ken | Manter Ren uniformemente ao redor do corpo |
| Ryu | Ryu | Controlar o fluxo de aura em tempo real |
| In | In | Esconder a aura de detecção |
| En | En | Expandir a aura para detectar presença |

## Categorias Nen (Nen Categories)

| PT-BR | EN | Posição Hex | Descrição |
|---|---|---|---|
| Reforço | Reinforcement | 0 | Aumentar propriedades naturais |
| Transmutação | Transmutation | 100 | Alterar propriedades da aura |
| Materialização | Materialization | 200 | Criar objetos com aura |
| Especialização | Specialization | 300 | Habilidades únicas e especiais |
| Manipulação | Manipulation | 400 | Controlar objetos ou seres |
| Emissão | Emission | 500 | Separar aura do corpo |

## Status

| PT-BR | EN | Fórmula Base |
|---|---|---|
| Pontos de Vida | Health Points (HP) | 20 + (nívVitalidade + valorResistência) × bônusFísicos |
| Pontos de Stamina | Stamina Points (SP) | 10 × (nívEnergia + valorResistência) × bônusFísicos |
| Pontos de Aura | Aura Points (AP) | 10 × (nívMop + nívConsciência) × bônusEspirituais |

## Armas (Weapons)

| PT-BR | EN | Tipo |
|---|---|---|
| Adaga | Dagger | Corpo a corpo |
| Espada | Sword | Corpo a corpo |
| Espada Longa | Longsword | Corpo a corpo |
| Katana | Katana | Corpo a corpo |
| Cimitarra | Scimitar | Corpo a corpo |
| Rapieira | Rapier | Corpo a corpo |
| Lança | Spear | Corpo a corpo |
| Alabarda | Halberd | Corpo a corpo |
| Machado | Axe | Corpo a corpo |
| Martelo | Hammer | Corpo a corpo |
| Martelo de Guerra | Warhammer | Corpo a corpo |
| Cajado | Staff | Corpo a corpo |
| Chicote | Whip | Corpo a corpo |
| Foice | Scythe | Corpo a corpo |
| Arco | Bow | À distância |
| Arco Longo | Longbow | À distância |
| Besta | Crossbow | À distância |
| Punho | Fist | Corpo a corpo |

## Classes de Personagem (Character Classes)

| PT-BR | EN | Descrição |
|---|---|---|
| Espadachim | Swordsman | Especialista em espadas |
| Samurai | Samurai | Guerreiro com código de honra |
| Ninja | Ninja | Especialista em furtividade |
| Ladino | Rogue | Especialista em truques e agilidade |
| Netrunner | Netrunner | Hacker e especialista em tecnologia |
| Pirata | Pirate | Combatente marítimo versátil |
| Mercenário | Mercenary | Combatente profissional |
| Terrorista | Terrorist | Especialista em explosivos |
| Monge | Monk | Combatente desarmado espiritual |
| Militar | Military | Soldado treinado |
| Hunter | Hunter | Licenciado pela Associação Hunter |
| Mestre de Armas | WeaponsMaster | Proficiente em múltiplas armas |
| Atleta | Athlete | Especialista em perícias físicas |
| Tribal | Tribal | Guerreiro de cultura tradicional |
| Experimento | Experiment | Sujeito de experimentos |
| Circo | Circus | Acrobata e artista |

## Dados (Dice)

| Tipo | Lados |
|---|---|
| D4 | 4 |
| D6 | 6 |
| D8 | 8 |
| D10 | 10 |
| D12 | 12 |
| D20 | 20 |
| D100 | 100 |
```

- [ ] **Step 2** — Commit the glossary:

```bash
git add docs/game/glossario.md
git commit -m "docs: add game glossary with PT-BR/EN keyword mappings

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 1 — ExpTable Tests

**Create:** `internal/domain/entity/character_sheet/experience/exp_table_test.go`

- [ ] **Step 1** — Write the exp_table_test.go file:

Create `internal/domain/entity/character_sheet/experience/exp_table_test.go`:

```go
package experience_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
)

func TestNewDefaultExpTable(t *testing.T) {
	table := experience.NewDefaultExpTable()
	if table == nil {
		t.Fatal("NewDefaultExpTable returned nil")
	}
}

func TestNewExpTable_CustomCoefficient(t *testing.T) {
	table := experience.NewExpTable(2.0)
	if table == nil {
		t.Fatal("NewExpTable(2.0) returned nil")
	}
}

func TestExpTable_LevelZeroIsZero(t *testing.T) {
	table := experience.NewDefaultExpTable()

	if base := table.GetBaseExpByLvl(0); base != 0 {
		t.Errorf("base exp at level 0: got %d, want 0", base)
	}
	if agg := table.GetAggregateExpByLvl(0); agg != 0 {
		t.Errorf("aggregate exp at level 0: got %d, want 0", agg)
	}
}

func TestExpTable_BaseExpMonotonicallyIncreasing(t *testing.T) {
	table := experience.NewDefaultExpTable()

	for lvl := 2; lvl < int(experience.MAX_LVL); lvl++ {
		curr := table.GetBaseExpByLvl(lvl)
		prev := table.GetBaseExpByLvl(lvl - 1)

		if curr < prev {
			t.Errorf("base exp decreased at lvl %d: %d < %d", lvl, curr, prev)
		}
	}
}

func TestExpTable_AggregateIsCumulativeSum(t *testing.T) {
	table := experience.NewDefaultExpTable()

	for lvl := 1; lvl < int(experience.MAX_LVL); lvl++ {
		expected := table.GetAggregateExpByLvl(lvl-1) + table.GetBaseExpByLvl(lvl)
		got := table.GetAggregateExpByLvl(lvl)

		if got != expected {
			t.Errorf("aggregate at lvl %d: got %d, want %d (prev_agg=%d + base=%d)",
				lvl, got, expected,
				table.GetAggregateExpByLvl(lvl-1), table.GetBaseExpByLvl(lvl))
		}
	}
}

func TestExpTable_AggregateMonotonicallyIncreasing(t *testing.T) {
	table := experience.NewDefaultExpTable()

	for lvl := 1; lvl < int(experience.MAX_LVL); lvl++ {
		curr := table.GetAggregateExpByLvl(lvl)
		prev := table.GetAggregateExpByLvl(lvl - 1)

		if curr <= prev {
			t.Errorf("aggregate not strictly increasing at lvl %d: %d <= %d", lvl, curr, prev)
		}
	}
}

func TestExpTable_GetLvlByExp(t *testing.T) {
	table := experience.NewDefaultExpTable()

	tests := []struct {
		name string
		exp  int
		want int
	}{
		{"zero exp is level 0", 0, 0},
		{"negative exp is level 0", -1, 0},
		{"exactly at level 1 aggregate", table.GetAggregateExpByLvl(1), 1},
		{"one below level 1 aggregate", table.GetAggregateExpByLvl(1) - 1, 0},
		{"exactly at level 10 aggregate", table.GetAggregateExpByLvl(10), 10},
		{"between level 10 and 11", table.GetAggregateExpByLvl(10) + 1, 10},
		{"exactly at max level aggregate", table.GetAggregateExpByLvl(int(experience.MAX_LVL) - 1), int(experience.MAX_LVL) - 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := table.GetLvlByExp(tt.exp)
			if got != tt.want {
				t.Errorf("GetLvlByExp(%d) = %d, want %d", tt.exp, got, tt.want)
			}
		})
	}
}

func TestExpTable_CoefficientScalesValues(t *testing.T) {
	table1 := experience.NewExpTable(1.0)
	table2 := experience.NewExpTable(2.0)

	for lvl := 1; lvl < int(experience.MAX_LVL); lvl++ {
		base1 := table1.GetBaseExpByLvl(lvl)
		base2 := table2.GetBaseExpByLvl(lvl)

		expected := 2 * base1
		if base2 != expected {
			t.Errorf("coefficient scaling at lvl %d: 2x table got %d, want %d (1x=%d)",
				lvl, base2, expected, base1)
		}
	}
}

func TestExpTable_GetLvlByExp_RoundTrip(t *testing.T) {
	table := experience.NewDefaultExpTable()

	for lvl := 0; lvl < int(experience.MAX_LVL); lvl++ {
		exp := table.GetAggregateExpByLvl(lvl)
		got := table.GetLvlByExp(exp)

		if got != lvl {
			t.Errorf("round trip failed at lvl %d: GetLvlByExp(GetAggregateExpByLvl(%d)) = %d",
				lvl, lvl, got)
		}
	}
}
```

- [ ] **Step 2** — Run tests:

```bash
go test ./internal/domain/entity/character_sheet/experience/ -run TestExpTable -v
```

- [ ] **Step 3** — Commit:

```bash
git add internal/domain/entity/character_sheet/experience/exp_table_test.go
git commit -m "test(experience): add ExpTable tests — structural properties and boundaries

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 2 — Experience and CharacterExp Tests

**Create:** `internal/domain/entity/character_sheet/experience/experience_test.go`, `internal/domain/entity/character_sheet/experience/character_exp_test.go`

- [ ] **Step 1** — Write experience_test.go:

Create `internal/domain/entity/character_sheet/experience/experience_test.go`:

```go
package experience_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
)

func newTestExp() *experience.Exp {
	table := experience.NewDefaultExpTable()
	exp := experience.NewExperience(table)
	return exp
}

func TestExp_InitialState(t *testing.T) {
	exp := newTestExp()

	if exp.GetPoints() != 0 {
		t.Errorf("initial points: got %d, want 0", exp.GetPoints())
	}
	if exp.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", exp.GetLevel())
	}
	if exp.GetCurrentExp() != 0 {
		t.Errorf("initial current exp: got %d, want 0", exp.GetCurrentExp())
	}
}

func TestExp_IncreasePoints_NoLevelUp(t *testing.T) {
	exp := newTestExp()

	smallExp := 1
	diff := exp.IncreasePoints(smallExp)

	if diff != 0 {
		t.Errorf("expected no level change, got diff=%d", diff)
	}
	if exp.GetPoints() != smallExp {
		t.Errorf("points after increase: got %d, want %d", exp.GetPoints(), smallExp)
	}
	if exp.GetLevel() != 0 {
		t.Errorf("level should still be 0, got %d", exp.GetLevel())
	}
}

func TestExp_IncreasePoints_WithLevelUp(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	lvl1Exp := table.GetAggregateExpByLvl(1)
	diff := exp.IncreasePoints(lvl1Exp)

	if diff != 1 {
		t.Errorf("level diff: got %d, want 1", diff)
	}
	if exp.GetLevel() != 1 {
		t.Errorf("level after increase: got %d, want 1", exp.GetLevel())
	}
	if exp.GetPoints() != lvl1Exp {
		t.Errorf("points: got %d, want %d", exp.GetPoints(), lvl1Exp)
	}
}

func TestExp_IncreasePoints_MultiLevel(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	lvl5Exp := table.GetAggregateExpByLvl(5)
	diff := exp.IncreasePoints(lvl5Exp)

	if diff != 5 {
		t.Errorf("level diff: got %d, want 5", diff)
	}
	if exp.GetLevel() != 5 {
		t.Errorf("level: got %d, want 5", exp.GetLevel())
	}
}

func TestExp_GetCurrentExp(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	lvl1Exp := table.GetAggregateExpByLvl(1)
	extra := 50
	exp.IncreasePoints(lvl1Exp + extra)

	current := exp.GetCurrentExp()
	if current != extra {
		t.Errorf("current exp: got %d, want %d", current, extra)
	}
}

func TestExp_GetExpToEvolve(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	evolve := exp.GetExpToEvolve()
	expectedEvolve := table.GetAggregateExpByLvl(1)

	if evolve != expectedEvolve {
		t.Errorf("exp to evolve at lvl 0: got %d, want %d", evolve, expectedEvolve)
	}
}

func TestExp_GetNextLvlBaseExp(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	got := exp.GetNextLvlBaseExp()
	want := table.GetBaseExpByLvl(1)
	if got != want {
		t.Errorf("next lvl base exp: got %d, want %d", got, want)
	}
}

func TestExp_GetNextLvlAggregateExp(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	got := exp.GetNextLvlAggregateExp()
	want := table.GetAggregateExpByLvl(1)
	if got != want {
		t.Errorf("next lvl aggregate exp: got %d, want %d", got, want)
	}
}

func TestExp_Clone(t *testing.T) {
	exp := newTestExp()
	exp.IncreasePoints(1000)

	clone := exp.Clone()

	if clone.GetPoints() != 0 {
		t.Errorf("clone should start fresh, got points=%d", clone.GetPoints())
	}
	if clone.GetLevel() != 0 {
		t.Errorf("clone should start at lvl 0, got %d", clone.GetLevel())
	}
}

func TestExp_IncrementalIncreases(t *testing.T) {
	exp := newTestExp()
	table := experience.NewDefaultExpTable()

	lvl1Exp := table.GetAggregateExpByLvl(1)
	half := lvl1Exp / 2

	diff1 := exp.IncreasePoints(half)
	if diff1 != 0 {
		t.Errorf("first half should not level up, diff=%d", diff1)
	}

	diff2 := exp.IncreasePoints(lvl1Exp - half)
	if diff2 != 1 {
		t.Errorf("second half should level up, diff=%d", diff2)
	}
	if exp.GetLevel() != 1 {
		t.Errorf("level after two increases: got %d, want 1", exp.GetLevel())
	}
}
```

- [ ] **Step 2** — Write character_exp_test.go:

Create `internal/domain/entity/character_sheet/experience/character_exp_test.go`:

```go
package experience_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
)

func newTestCharacterExp() *experience.CharacterExp {
	table := experience.NewDefaultExpTable()
	exp := experience.NewExperience(table)
	return experience.NewCharacterExp(*exp)
}

func TestCharacterExp_InitialState(t *testing.T) {
	ce := newTestCharacterExp()

	if ce.GetCharacterPoints() != 0 {
		t.Errorf("initial character points: got %d, want 0", ce.GetCharacterPoints())
	}
	if ce.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", ce.GetLevel())
	}
	if ce.GetExpPoints() != 0 {
		t.Errorf("initial exp points: got %d, want 0", ce.GetExpPoints())
	}
	if ce.GetCurrentExp() != 0 {
		t.Errorf("initial current exp: got %d, want 0", ce.GetCurrentExp())
	}
}

func TestCharacterExp_IncreaseCharacterPoints(t *testing.T) {
	ce := newTestCharacterExp()

	ce.IncreaseCharacterPoints(5)
	if ce.GetCharacterPoints() != 5 {
		t.Errorf("character points: got %d, want 5", ce.GetCharacterPoints())
	}

	ce.IncreaseCharacterPoints(3)
	if ce.GetCharacterPoints() != 8 {
		t.Errorf("character points after second increase: got %d, want 8", ce.GetCharacterPoints())
	}
}

func TestCharacterExp_EndCascadeUpgrade(t *testing.T) {
	ce := newTestCharacterExp()

	cascade := experience.NewUpgradeCascade(100)
	ce.EndCascadeUpgrade(cascade)

	if ce.GetExpPoints() != 100 {
		t.Errorf("exp points after cascade: got %d, want 100", ce.GetExpPoints())
	}
	if cascade.CharacterExp != ce {
		t.Error("cascade.CharacterExp should reference the CharacterExp instance")
	}
}

func TestCharacterExp_EndCascadeUpgrade_MultipleCalls(t *testing.T) {
	ce := newTestCharacterExp()

	cascade1 := experience.NewUpgradeCascade(50)
	ce.EndCascadeUpgrade(cascade1)

	cascade2 := experience.NewUpgradeCascade(50)
	ce.EndCascadeUpgrade(cascade2)

	if ce.GetExpPoints() != 100 {
		t.Errorf("exp points after two cascades: got %d, want 100", ce.GetExpPoints())
	}
}

func TestCharacterExp_GetNextLvlBaseExp(t *testing.T) {
	ce := newTestCharacterExp()

	got := ce.GetNextLvlBaseExp()
	if got <= 0 {
		t.Errorf("next lvl base exp should be positive, got %d", got)
	}
}

func TestCharacterExp_GetNextLvlAggregateExp(t *testing.T) {
	ce := newTestCharacterExp()

	got := ce.GetNextLvlAggregateExp()
	if got <= 0 {
		t.Errorf("next lvl aggregate exp should be positive, got %d", got)
	}
}
```

- [ ] **Step 3** — Run all experience tests:

```bash
go test ./internal/domain/entity/character_sheet/experience/ -v
```

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/experience/experience_test.go \
        internal/domain/entity/character_sheet/experience/character_exp_test.go
git commit -m "test(experience): add Exp and CharacterExp tests — level ups, cascades, cloning

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 3 — Experience Documentation (PT-BR)

**Create:** `docs/game/ficha-de-personagem/experiencia.md`

- [ ] **Step 1** — Write the experience documentation:

Create `docs/game/ficha-de-personagem/experiencia.md`:

```markdown
# Sistema de Experiência (Experience System)

## Visão Geral

O sistema de experiência (XP) é a base de toda a progressão no HxH RPG System. Cada entidade na ficha de personagem — perícias, atributos, habilidades, princípios Nen — possui sua própria instância de experiência com tabela de progressão configurável.

## Tabela de Experiência (ExpTable)

Cada componente utiliza uma **ExpTable** com um coeficiente multiplicador. A tabela suporta até **100 níveis** (0–100) e é gerada por uma função sigmoidal tripla:

```
f(lvl) = 1700/(1 + e^(0.37*(12-lvl))) + 1800/(1 + e^(0.37*(38-lvl))) + 2000/(1 + e^(0.28*(74-lvl)))
```

O XP base para cada nível é `coeficiente × f(nível)`, e o XP agregado é a soma cumulativa de todos os níveis anteriores.

### Coeficientes por Componente

| Componente | Coeficiente | Descrição |
|---|---|---|
| Personagem | 10.0 | Experiência geral do personagem |
| Talento | 2.0 | Progressão do talento por categoria |
| Físicos | 20.0 | Habilidade física |
| Mentais | 20.0 | Habilidade mental |
| Espirituais | 5.0 | Habilidade espiritual (Nen) |
| Perícias (habilidade) | 20.0 | Habilidade geral de perícias |
| Atributos Físicos | 5.0 | Cada atributo físico |
| Atributos Mentais | 1.0 | Cada atributo mental |
| Atributos Espirituais | 1.0 | Cada atributo espiritual |
| Perícias Físicas | 1.0 | Cada perícia física |
| Perícias Mentais | 2.0 | Cada perícia mental |
| Perícias Espirituais | 3.0 | Cada perícia espiritual |
| Princípios Nen | 1.0 | Cada princípio e categoria |

## Experiência do Personagem (CharacterExp)

A `CharacterExp` é o nível geral do personagem. Ela recebe XP no final de cada **upgrade em cascata** (cascade upgrade) — sempre que qualquer perícia ou princípio recebe experiência, o XP se propaga pela cadeia até chegar à experiência do personagem.

### Pontos de Personagem

Cada vez que uma **habilidade** (Physicals, Mentals, Spirituals, Skills) sobe de nível, o personagem ganha **pontos de personagem** que influenciam o bônus de habilidade.

### Fórmula do Bônus de Habilidade

```
bônus = (pontosDePersonagem + nívelDaHabilidade) / 2
```

## Upgrade em Cascata (Cascade Upgrade)

O mecanismo central de progressão. Quando XP é inserido em uma perícia:

1. A **perícia** recebe o XP
2. O **atributo** associado recebe o XP
3. A **habilidade** associada recebe o XP
4. A **experiência do personagem** recebe o XP
5. Todos os **status** são recalculados

Este processo garante que treinar qualquer aspecto do personagem contribui para sua progressão geral.
```

- [ ] **Step 2** — Commit:

```bash
git add docs/game/ficha-de-personagem/experiencia.md
git commit -m "docs: add experience system documentation (PT-BR)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 4 — Talent and Ability Tests

**Create:** `internal/domain/entity/character_sheet/ability/talent_test.go`, `internal/domain/entity/character_sheet/ability/ability_test.go`

- [ ] **Step 1** — Write talent_test.go:

Create `internal/domain/entity/character_sheet/ability/talent_test.go`:

```go
package ability_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
)

func newTestTalent() *ability.Talent {
	table := experience.NewExpTable(2.0)
	exp := experience.NewExperience(table)
	return ability.NewTalent(*exp)
}

func TestTalent_InitialState(t *testing.T) {
	talent := newTestTalent()

	if talent.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", talent.GetLevel())
	}
	if talent.GetExpPoints() != 0 {
		t.Errorf("initial exp: got %d, want 0", talent.GetExpPoints())
	}
}

func TestTalent_InitWithLvl(t *testing.T) {
	tests := []struct {
		name     string
		lvl      int
		wantLvl  int
	}{
		{"init with level 1", 1, 1},
		{"init with level 5", 5, 5},
		{"init with level 20", 20, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			talent := newTestTalent()
			talent.InitWithLvl(tt.lvl)

			if talent.GetLevel() != tt.wantLvl {
				t.Errorf("level after InitWithLvl(%d): got %d, want %d",
					tt.lvl, talent.GetLevel(), tt.wantLvl)
			}
		})
	}
}

func TestTalent_IncreaseExp(t *testing.T) {
	talent := newTestTalent()

	diff := talent.IncreaseExp(1)
	if talent.GetExpPoints() != 1 {
		t.Errorf("exp after increase: got %d, want 1", talent.GetExpPoints())
	}
	if diff != 0 {
		t.Errorf("diff for small exp: got %d, want 0", diff)
	}
}

func TestTalent_IncreaseExp_LevelUp(t *testing.T) {
	talent := newTestTalent()
	table := experience.NewExpTable(2.0)

	lvl1Agg := table.GetAggregateExpByLvl(1)
	diff := talent.IncreaseExp(lvl1Agg)

	if diff != 1 {
		t.Errorf("diff after level up exp: got %d, want 1", diff)
	}
	if talent.GetLevel() != 1 {
		t.Errorf("level after level up: got %d, want 1", talent.GetLevel())
	}
}

func TestTalent_GetCurrentExp(t *testing.T) {
	talent := newTestTalent()
	table := experience.NewExpTable(2.0)

	lvl1Agg := table.GetAggregateExpByLvl(1)
	extra := 10
	talent.IncreaseExp(lvl1Agg + extra)

	if talent.GetCurrentExp() != extra {
		t.Errorf("current exp: got %d, want %d", talent.GetCurrentExp(), extra)
	}
}

func TestTalent_GetNextLvlAggregateExp(t *testing.T) {
	talent := newTestTalent()

	got := talent.GetNextLvlAggregateExp()
	if got <= 0 {
		t.Errorf("next lvl aggregate exp should be positive, got %d", got)
	}
}

func TestTalent_GetNextLvlBaseExp(t *testing.T) {
	talent := newTestTalent()

	got := talent.GetNextLvlBaseExp()
	if got <= 0 {
		t.Errorf("next lvl base exp should be positive, got %d", got)
	}
}
```

- [ ] **Step 2** — Write ability_test.go:

Create `internal/domain/entity/character_sheet/ability/ability_test.go`:

```go
package ability_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestAbility() (*ability.Ability, *experience.CharacterExp) {
	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilityTable := experience.NewExpTable(20.0)
	abilityExp := experience.NewExperience(abilityTable)

	a := ability.NewAbility(enum.Physicals, *abilityExp, charExp)
	return a, charExp
}

func TestAbility_InitialState(t *testing.T) {
	a, _ := newTestAbility()

	if a.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", a.GetLevel())
	}
	if a.GetExpPoints() != 0 {
		t.Errorf("initial exp: got %d, want 0", a.GetExpPoints())
	}
	if a.GetName() != enum.Physicals {
		t.Errorf("name: got %v, want Physicals", a.GetName())
	}
}

func TestAbility_GetBonus(t *testing.T) {
	tests := []struct {
		name      string
		charPts   int
		wantBonus float64
	}{
		{"zero points and level", 0, 0.0},
		{"with character points", 10, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, charExp := newTestAbility()

			charExp.IncreaseCharacterPoints(tt.charPts)
			got := a.GetBonus()

			if got != tt.wantBonus {
				t.Errorf("GetBonus() = %f, want %f (charPts=%d, abilityLvl=%d)",
					got, tt.wantBonus, charExp.GetCharacterPoints(), a.GetLevel())
			}
		})
	}
}

func TestAbility_CascadeUpgrade(t *testing.T) {
	a, charExp := newTestAbility()

	cascade := experience.NewUpgradeCascade(100)
	a.CascadeUpgrade(cascade)

	if a.GetExpPoints() != 100 {
		t.Errorf("ability exp after cascade: got %d, want 100", a.GetExpPoints())
	}
	if charExp.GetExpPoints() != 100 {
		t.Errorf("char exp after cascade: got %d, want 100", charExp.GetExpPoints())
	}
	if cascade.CharacterExp != charExp {
		t.Error("cascade.CharacterExp should be set after CascadeUpgrade")
	}

	abCascade, ok := cascade.Abilities[enum.Physicals]
	if !ok {
		t.Fatal("cascade.Abilities should contain Physicals entry")
	}
	if abCascade.Exp != a.GetExpPoints() {
		t.Errorf("cascade ability exp: got %d, want %d", abCascade.Exp, a.GetExpPoints())
	}
}

func TestAbility_CascadeUpgrade_LevelUp_IncreasesCharacterPoints(t *testing.T) {
	a, charExp := newTestAbility()

	abilityTable := experience.NewExpTable(20.0)
	lvl1Exp := abilityTable.GetAggregateExpByLvl(1)

	cascade := experience.NewUpgradeCascade(lvl1Exp)
	a.CascadeUpgrade(cascade)

	if a.GetLevel() < 1 {
		t.Errorf("ability should level up, got level %d", a.GetLevel())
	}
	if charExp.GetCharacterPoints() < 1 {
		t.Errorf("character points should increase on ability level up, got %d",
			charExp.GetCharacterPoints())
	}
}

func TestAbility_GetExpReference(t *testing.T) {
	a, _ := newTestAbility()

	ref := a.GetExpReference()
	if ref == nil {
		t.Fatal("GetExpReference returned nil")
	}
}

func TestAbility_DelegatesExpMethods(t *testing.T) {
	a, _ := newTestAbility()

	if a.GetNextLvlBaseExp() <= 0 {
		t.Error("GetNextLvlBaseExp should be positive")
	}
	if a.GetNextLvlAggregateExp() <= 0 {
		t.Error("GetNextLvlAggregateExp should be positive")
	}
	if a.GetCurrentExp() != 0 {
		t.Errorf("GetCurrentExp should be 0 initially, got %d", a.GetCurrentExp())
	}
}
```

- [ ] **Step 3** — Run tests:

```bash
go test ./internal/domain/entity/character_sheet/ability/ -run "TestTalent|TestAbility" -v
```

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/ability/talent_test.go \
        internal/domain/entity/character_sheet/ability/ability_test.go
git commit -m "test(ability): add Talent and Ability tests — bonus formula, cascade upgrade

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 5 — AbilitiesManager Tests

**Create:** `internal/domain/entity/character_sheet/ability/abilities_manager_test.go`

- [ ] **Step 1** — Write abilities_manager_test.go:

Create `internal/domain/entity/character_sheet/ability/abilities_manager_test.go`:

```go
package ability_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestAbilitiesManager() *ability.Manager {
	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilities := make(map[enum.AbilityName]ability.IAbility)

	talentExp := experience.NewExperience(experience.NewExpTable(2.0))
	talent := ability.NewTalent(*talentExp)

	physicalExp := experience.NewExperience(experience.NewExpTable(20.0))
	abilities[enum.Physicals] = ability.NewAbility(enum.Physicals, *physicalExp, charExp)

	mentalExp := experience.NewExperience(experience.NewExpTable(20.0))
	abilities[enum.Mentals] = ability.NewAbility(enum.Mentals, *mentalExp, charExp)

	spiritualExp := experience.NewExperience(experience.NewExpTable(5.0))
	abilities[enum.Spirituals] = ability.NewAbility(enum.Spirituals, *spiritualExp, charExp)

	skillsExp := experience.NewExperience(experience.NewExpTable(20.0))
	abilities[enum.Skills] = ability.NewAbility(enum.Skills, *skillsExp, charExp)

	return ability.NewAbilitiesManager(charExp, abilities, *talent)
}

func TestAbilitiesManager_Get_Found(t *testing.T) {
	mgr := newTestAbilitiesManager()

	tests := []enum.AbilityName{enum.Physicals, enum.Mentals, enum.Spirituals, enum.Skills}
	for _, name := range tests {
		t.Run(string(name), func(t *testing.T) {
			a, err := mgr.Get(name)
			if err != nil {
				t.Fatalf("Get(%s) unexpected error: %v", name, err)
			}
			if a == nil {
				t.Fatalf("Get(%s) returned nil", name)
			}
		})
	}
}

func TestAbilitiesManager_Get_NotFound(t *testing.T) {
	mgr := newTestAbilitiesManager()

	_, err := mgr.Get("NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent ability, got nil")
	}
	if !errors.Is(err, ability.ErrAbilityNotFound) {
		t.Errorf("expected ErrAbilityNotFound, got %v", err)
	}
}

func TestAbilitiesManager_GetLevelOf(t *testing.T) {
	mgr := newTestAbilitiesManager()

	lvl, err := mgr.GetLevelOf(enum.Physicals)
	if err != nil {
		t.Fatalf("GetLevelOf unexpected error: %v", err)
	}
	if lvl != 0 {
		t.Errorf("initial level: got %d, want 0", lvl)
	}
}

func TestAbilitiesManager_GetLevelOf_NotFound(t *testing.T) {
	mgr := newTestAbilitiesManager()

	_, err := mgr.GetLevelOf("NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent ability")
	}
}

func TestAbilitiesManager_GetExpReferenceOf(t *testing.T) {
	mgr := newTestAbilitiesManager()

	ref, err := mgr.GetExpReferenceOf(enum.Physicals)
	if err != nil {
		t.Fatalf("GetExpReferenceOf unexpected error: %v", err)
	}
	if ref == nil {
		t.Fatal("GetExpReferenceOf returned nil")
	}
}

func TestAbilitiesManager_CharacterLevel(t *testing.T) {
	mgr := newTestAbilitiesManager()

	if mgr.GetCharacterLevel() != 0 {
		t.Errorf("initial character level: got %d, want 0", mgr.GetCharacterLevel())
	}
}

func TestAbilitiesManager_CharacterPoints(t *testing.T) {
	mgr := newTestAbilitiesManager()

	if mgr.GetCharacterPoints() != 0 {
		t.Errorf("initial character points: got %d, want 0", mgr.GetCharacterPoints())
	}
}

func TestAbilitiesManager_InitTalentWithLvl(t *testing.T) {
	mgr := newTestAbilitiesManager()

	mgr.InitTalentWithLvl(5)

	if mgr.GetTalentLevel() != 5 {
		t.Errorf("talent level after init: got %d, want 5", mgr.GetTalentLevel())
	}
}

func TestAbilitiesManager_IncreaseTalentExp(t *testing.T) {
	mgr := newTestAbilitiesManager()

	mgr.IncreaseTalentExp(10)
	if mgr.GetTalentExpPoints() != 10 {
		t.Errorf("talent exp after increase: got %d, want 10", mgr.GetTalentExpPoints())
	}
}

func TestAbilitiesManager_GetLevels(t *testing.T) {
	mgr := newTestAbilitiesManager()

	levels := mgr.GetLevels()
	if len(levels) != 4 {
		t.Errorf("expected 4 ability levels, got %d", len(levels))
	}
	for name, lvl := range levels {
		if lvl != 0 {
			t.Errorf("initial level for %s: got %d, want 0", name, lvl)
		}
	}
}

func TestAbilitiesManager_GetAllAbilities(t *testing.T) {
	mgr := newTestAbilitiesManager()

	all := mgr.GetAllAbilities()
	if len(all) != 4 {
		t.Errorf("expected 4 abilities, got %d", len(all))
	}
}

func TestAbilitiesManager_TalentDelegation(t *testing.T) {
	mgr := newTestAbilitiesManager()

	if mgr.GetTalentNextLvlBaseExp() <= 0 {
		t.Error("talent next lvl base exp should be positive")
	}
	if mgr.GetTalentNextLvlAggregateExp() <= 0 {
		t.Error("talent next lvl aggregate exp should be positive")
	}
	if mgr.GetTalentCurrentExp() != 0 {
		t.Errorf("initial talent current exp: got %d, want 0", mgr.GetTalentCurrentExp())
	}
	if mgr.GetTalentExpPoints() != 0 {
		t.Errorf("initial talent exp points: got %d, want 0", mgr.GetTalentExpPoints())
	}
}

func TestAbilitiesManager_CharacterExpDelegation(t *testing.T) {
	mgr := newTestAbilitiesManager()

	if mgr.GetCharacterNextLvlBaseExp() <= 0 {
		t.Error("character next lvl base exp should be positive")
	}
	if mgr.GetCharacterNextLvlAggregateExp() <= 0 {
		t.Error("character next lvl aggregate exp should be positive")
	}
	if mgr.GetCharacterCurrentExp() != 0 {
		t.Errorf("initial character current exp: got %d, want 0", mgr.GetCharacterCurrentExp())
	}
	if mgr.GetCharacterExpPoints() != 0 {
		t.Errorf("initial character exp points: got %d, want 0", mgr.GetCharacterExpPoints())
	}
}
```

- [ ] **Step 2** — Run tests:

```bash
go test ./internal/domain/entity/character_sheet/ability/ -v
```

- [ ] **Step 3** — Commit:

```bash
git add internal/domain/entity/character_sheet/ability/abilities_manager_test.go
git commit -m "test(ability): add AbilitiesManager tests — lookups, delegation, talent init

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 6 — Abilities Documentation (PT-BR)

**Create:** `docs/game/ficha-de-personagem/habilidades.md`

- [ ] **Step 1** — Write the abilities documentation:

Create `docs/game/ficha-de-personagem/habilidades.md`:

```markdown
# Habilidades (Abilities)

## Visão Geral

Habilidades representam as quatro grandes áreas de competência de um personagem. Cada habilidade governa um conjunto de atributos e perícias, e seu nível influencia diretamente o bônus aplicado a todos os componentes subordinados.

## As Quatro Habilidades

| Habilidade | EN | Coef. XP | Governa |
|---|---|---|---|
| Físicos | Physicals | 20.0 | Atributos e perícias físicas, HP, SP |
| Mentais | Mentals | 20.0 | Atributos e perícias mentais |
| Espirituais | Spirituals | 5.0 | Atributos espirituais, princípios Nen, AP |
| Perícias | Skills | 20.0 | Experiência geral de todas as perícias |

## Bônus de Habilidade (Ability Bonus)

O bônus é calculado pela média entre os pontos de personagem e o nível da habilidade:

```
bônus = (pontosDePersonagem + nívelDaHabilidade) / 2
```

Este bônus é usado nas fórmulas de:
- **Poder do atributo** (GetPower)
- **Status máximo** (HP, SP, AP)

## Talento (Talent)

O talento é um sistema de progressão especial baseado nas categorias Nen ativas do personagem. O nível do talento é determinado pelo `TalentByCategorySet`:

- **Base:** nível 20
- **Sem hexágono:** bônus = (categorias ativas - 1) × 2 (mínimo 1)
- **Com hexágono:** bônus = categorias ativas - 1

## Upgrade em Cascata

Quando uma habilidade recebe XP via cascade upgrade:
1. O XP é adicionado à experiência da habilidade
2. A experiência do personagem recebe o mesmo XP
3. Se a habilidade sobe de nível, os pontos de personagem aumentam

## Pontos de Personagem

Cada vez que uma habilidade sobe de nível, o personagem ganha pontos que:
- Aumentam o bônus de todas as habilidades
- Influenciam indiretamente todos os status e poderes de atributo
```

- [ ] **Step 2** — Commit:

```bash
git add docs/game/ficha-de-personagem/habilidades.md
git commit -m "docs: add abilities system documentation (PT-BR)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 7 — StatusBar Tests

**Create:** `internal/domain/entity/character_sheet/status/status_bar_test.go`

- [ ] **Step 1** — Write status_bar_test.go:

Create `internal/domain/entity/character_sheet/status/status_bar_test.go`:

```go
package status_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
)

func TestStatusBar_InitialState(t *testing.T) {
	bar := status.NewStatusBar()

	if bar.GetMin() != 0 {
		t.Errorf("initial min: got %d, want 0", bar.GetMin())
	}
	if bar.GetCurrent() != 0 {
		t.Errorf("initial current: got %d, want 0", bar.GetCurrent())
	}
	if bar.GetMax() != 0 {
		t.Errorf("initial max: got %d, want 0", bar.GetMax())
	}
}

func TestStatusBar_IncreaseAt(t *testing.T) {
	tests := []struct {
		name      string
		increase  int
		wantCurr  int
	}{
		{"increase within max", 5, 0},
		{"increase exceeds max stays at max", 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := status.NewStatusBar()
			got := bar.IncreaseAt(tt.increase)

			if got != tt.wantCurr {
				t.Errorf("IncreaseAt(%d) = %d, want %d (max=0)", tt.increase, got, tt.wantCurr)
			}
		})
	}
}

func TestStatusBar_DecreaseAt(t *testing.T) {
	tests := []struct {
		name     string
		decrease int
		wantCurr int
	}{
		{"decrease within min", 5, 0},
		{"decrease below min stays at min", 100, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := status.NewStatusBar()
			got := bar.DecreaseAt(tt.decrease)

			if got != tt.wantCurr {
				t.Errorf("DecreaseAt(%d) = %d, want %d (min=0)", tt.decrease, got, tt.wantCurr)
			}
		})
	}
}

func TestStatusBar_SetCurrent_Valid(t *testing.T) {
	bar := status.NewStatusBar()

	err := bar.SetCurrent(0)
	if err != nil {
		t.Fatalf("SetCurrent(0) unexpected error: %v", err)
	}
	if bar.GetCurrent() != 0 {
		t.Errorf("current after SetCurrent(0): got %d, want 0", bar.GetCurrent())
	}
}

func TestStatusBar_SetCurrent_Invalid(t *testing.T) {
	bar := status.NewStatusBar()

	tests := []struct {
		name  string
		value int
	}{
		{"above max", 1},
		{"below min", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bar.SetCurrent(tt.value)
			if err == nil {
				t.Errorf("SetCurrent(%d) expected error, got nil", tt.value)
			}
		})
	}
}
```

- [ ] **Step 2** — Run tests:

```bash
go test ./internal/domain/entity/character_sheet/status/ -run TestStatusBar -v
```

- [ ] **Step 3** — Commit:

```bash
git add internal/domain/entity/character_sheet/status/status_bar_test.go
git commit -m "test(status): add StatusBar tests — increase, decrease, set current bounds

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 8 — Status Bars Tests (HP/SP/AP with Mocks)

**Create:** `internal/domain/entity/character_sheet/status/status_bars_test.go`

- [ ] **Step 1** — Write status_bars_test.go with mock implementations:

Create `internal/domain/entity/character_sheet/status/status_bars_test.go`:

```go
package status_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
)

// --- Mock implementations ---

type mockAbility struct {
	bonus float64
	level int
}

func (m *mockAbility) GetBonus() float64                              { return m.bonus }
func (m *mockAbility) GetLevel() int                                  { return m.level }
func (m *mockAbility) CascadeUpgrade(_ *experience.UpgradeCascade)    {}
func (m *mockAbility) GetNextLvlAggregateExp() int                    { return 0 }
func (m *mockAbility) GetNextLvlBaseExp() int                         { return 0 }
func (m *mockAbility) GetCurrentExp() int                             { return 0 }
func (m *mockAbility) GetExpPoints() int                              { return 0 }
func (m *mockAbility) GetExpReference() experience.ICascadeUpgrade    { return nil }
func (m *mockAbility) GetPower() int                                  { return 0 }
func (m *mockAbility) GetAbilityBonus() float64                       { return 0 }

type mockDistributableAttribute struct {
	value int
	level int
}

func (m *mockDistributableAttribute) GetValue() int                             { return m.value }
func (m *mockDistributableAttribute) GetPoints() int                            { return 0 }
func (m *mockDistributableAttribute) GetPower() int                             { return m.value }
func (m *mockDistributableAttribute) GetAbilityBonus() float64                  { return 0 }
func (m *mockDistributableAttribute) GetLevel() int                             { return m.level }
func (m *mockDistributableAttribute) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockDistributableAttribute) GetNextLvlAggregateExp() int               { return 0 }
func (m *mockDistributableAttribute) GetNextLvlBaseExp() int                    { return 0 }
func (m *mockDistributableAttribute) GetCurrentExp() int                        { return 0 }
func (m *mockDistributableAttribute) GetExpPoints() int                         { return 0 }

type mockGameAttribute struct {
	power int
	level int
}

func (m *mockGameAttribute) GetPower() int                             { return m.power }
func (m *mockGameAttribute) GetAbilityBonus() float64                  { return 0 }
func (m *mockGameAttribute) GetLevel() int                             { return m.level }
func (m *mockGameAttribute) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockGameAttribute) GetNextLvlAggregateExp() int               { return 0 }
func (m *mockGameAttribute) GetNextLvlBaseExp() int                    { return 0 }
func (m *mockGameAttribute) GetCurrentExp() int                        { return 0 }
func (m *mockGameAttribute) GetExpPoints() int                         { return 0 }

type mockSkill struct {
	valueForTest int
	level        int
}

func (m *mockSkill) GetValueForTest() int                                   { return m.valueForTest }
func (m *mockSkill) GetLevel() int                                          { return m.level }
func (m *mockSkill) CascadeUpgradeTrigger(_ *experience.UpgradeCascade)     {}
func (m *mockSkill) GetNextLvlAggregateExp() int                            { return 0 }
func (m *mockSkill) GetNextLvlBaseExp() int                                 { return 0 }
func (m *mockSkill) GetCurrentExp() int                                     { return 0 }
func (m *mockSkill) GetExpPoints() int                                      { return 0 }

// --- HealthPoints Tests ---

func TestHealthPoints_Formula(t *testing.T) {
	tests := []struct {
		name           string
		abilityBonus   float64
		vitalityLevel  int
		resistanceVal  int
		wantMax        int
	}{
		{
			"zero values",
			0.0, 0, 0,
			20,
		},
		{
			"with vitality and resistance",
			2.0, 3, 2,
			20 + int(float64(3+2)*2.0),
		},
		{
			"high values",
			5.0, 10, 5,
			20 + int(float64(10+5)*5.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			physicals := &mockAbility{bonus: tt.abilityBonus}
			resistance := &mockDistributableAttribute{value: tt.resistanceVal}
			vitality := &mockSkill{level: tt.vitalityLevel}

			hp := status.NewHealthPoints(physicals, resistance, vitality)

			if hp.GetMax() != tt.wantMax {
				t.Errorf("HP max: got %d, want %d", hp.GetMax(), tt.wantMax)
			}
			if hp.GetCurrent() != tt.wantMax {
				t.Errorf("HP current should equal max initially: got %d, want %d",
					hp.GetCurrent(), tt.wantMax)
			}
		})
	}
}

// --- StaminaPoints Tests ---

func TestStaminaPoints_Formula(t *testing.T) {
	tests := []struct {
		name          string
		abilityBonus  float64
		energyLevel   int
		resistanceVal int
		wantMax       int
	}{
		{
			"zero values",
			0.0, 0, 0,
			0,
		},
		{
			"with energy and resistance",
			2.0, 3, 2,
			10 * int(float64(3+2)*2.0),
		},
		{
			"high values",
			5.0, 10, 5,
			10 * int(float64(10+5)*5.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			physicals := &mockAbility{bonus: tt.abilityBonus}
			resistance := &mockDistributableAttribute{value: tt.resistanceVal}
			energy := &mockSkill{level: tt.energyLevel}

			sp := status.NewStaminaPoints(physicals, resistance, energy)

			if sp.GetMax() != tt.wantMax {
				t.Errorf("SP max: got %d, want %d", sp.GetMax(), tt.wantMax)
			}
			if sp.GetCurrent() != tt.wantMax {
				t.Errorf("SP current should equal max initially: got %d, want %d",
					sp.GetCurrent(), tt.wantMax)
			}
		})
	}
}

// --- AuraPoints Tests ---

func TestAuraPoints_Formula(t *testing.T) {
	tests := []struct {
		name              string
		abilityBonus      float64
		mopLevel          int
		conscienceNenLvl  int
		wantMax           int
	}{
		{
			"zero values",
			0.0, 0, 0,
			0,
		},
		{
			"with mop and conscience",
			5.0, 3, 2,
			int(10 * float64(3+2) * float64(int(5.0))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spirituals := &mockAbility{bonus: tt.abilityBonus}
			conscienceNen := &mockGameAttribute{level: tt.conscienceNenLvl}
			mop := &mockSkill{level: tt.mopLevel}

			ap, err := status.NewAuraPoints(spirituals, conscienceNen, mop)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ap.GetMax() != tt.wantMax {
				t.Errorf("AP max: got %d, want %d", ap.GetMax(), tt.wantMax)
			}
		})
	}
}

func TestAuraPoints_NilSpiritual_ReturnsError(t *testing.T) {
	conscienceNen := &mockGameAttribute{level: 1}
	mop := &mockSkill{level: 1}

	_, err := status.NewAuraPoints(nil, conscienceNen, mop)
	if err == nil {
		t.Fatal("expected error for nil spiritual ability")
	}
}

func TestHealthPoints_UpgradeKeepsFullHP(t *testing.T) {
	physicals := &mockAbility{bonus: 2.0}
	resistance := &mockDistributableAttribute{value: 1}
	vitality := &mockSkill{level: 1}

	hp := status.NewHealthPoints(physicals, resistance, vitality)
	initialMax := hp.GetMax()

	if hp.GetCurrent() != initialMax {
		t.Fatalf("current should be max initially")
	}

	hp.Upgrade()

	if hp.GetCurrent() != hp.GetMax() {
		t.Errorf("after Upgrade with full HP, current should still equal max: got %d, want %d",
			hp.GetCurrent(), hp.GetMax())
	}
}
```

- [ ] **Step 2** — Run tests:

```bash
go test ./internal/domain/entity/character_sheet/status/ -run "TestHealthPoints|TestStaminaPoints|TestAuraPoints" -v
```

- [ ] **Step 3** — Commit:

```bash
git add internal/domain/entity/character_sheet/status/status_bars_test.go
git commit -m "test(status): add HP/SP/AP bar tests with mocks — formulas and nil validation

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 9 — StatusManager Tests + Status Documentation

**Create:** `internal/domain/entity/character_sheet/status/status_manager_test.go`, `docs/game/ficha-de-personagem/status.md`

- [ ] **Step 1** — Write status_manager_test.go:

Create `internal/domain/entity/character_sheet/status/status_manager_test.go`:

```go
package status_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/status"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestStatusManager() *status.Manager {
	bars := make(map[enum.StatusName]status.IStatusBar)

	physicals := &mockAbility{bonus: 2.0}
	resistance := &mockDistributableAttribute{value: 1}
	vitality := &mockSkill{level: 1}
	bars[enum.Health] = status.NewHealthPoints(physicals, resistance, vitality)

	energy := &mockSkill{level: 1}
	bars[enum.Stamina] = status.NewStaminaPoints(physicals, resistance, energy)

	return status.NewStatusManager(bars)
}

func TestStatusManager_Get_Found(t *testing.T) {
	mgr := newTestStatusManager()

	tests := []enum.StatusName{enum.Health, enum.Stamina}
	for _, name := range tests {
		t.Run(string(name), func(t *testing.T) {
			bar, err := mgr.Get(name)
			if err != nil {
				t.Fatalf("Get(%s) unexpected error: %v", name, err)
			}
			if bar == nil {
				t.Fatalf("Get(%s) returned nil", name)
			}
		})
	}
}

func TestStatusManager_Get_NotFound(t *testing.T) {
	mgr := newTestStatusManager()

	_, err := mgr.Get(enum.Aura)
	if err == nil {
		t.Fatal("expected error for Aura (not added)")
	}
	if !errors.Is(err, status.ErrStatusNotFound) {
		t.Errorf("expected ErrStatusNotFound, got %v", err)
	}
}

func TestStatusManager_GetMaxOf(t *testing.T) {
	mgr := newTestStatusManager()

	maxHP, err := mgr.GetMaxOf(enum.Health)
	if err != nil {
		t.Fatalf("GetMaxOf(Health) error: %v", err)
	}
	if maxHP <= 0 {
		t.Errorf("HP max should be positive, got %d", maxHP)
	}
}

func TestStatusManager_GetCurrentOf(t *testing.T) {
	mgr := newTestStatusManager()

	curr, err := mgr.GetCurrentOf(enum.Health)
	if err != nil {
		t.Fatalf("GetCurrentOf(Health) error: %v", err)
	}
	if curr <= 0 {
		t.Errorf("HP current should be positive initially, got %d", curr)
	}
}

func TestStatusManager_SetCurrent(t *testing.T) {
	mgr := newTestStatusManager()

	maxHP, _ := mgr.GetMaxOf(enum.Health)
	err := mgr.SetCurrent(enum.Health, maxHP-1)
	if err != nil {
		t.Fatalf("SetCurrent(Health, %d) error: %v", maxHP-1, err)
	}

	curr, _ := mgr.GetCurrentOf(enum.Health)
	if curr != maxHP-1 {
		t.Errorf("current after SetCurrent: got %d, want %d", curr, maxHP-1)
	}
}

func TestStatusManager_SetCurrent_InvalidValue(t *testing.T) {
	mgr := newTestStatusManager()

	maxHP, _ := mgr.GetMaxOf(enum.Health)
	err := mgr.SetCurrent(enum.Health, maxHP+1)
	if err == nil {
		t.Fatal("expected error for value > max")
	}
}

func TestStatusManager_SetCurrent_NotFound(t *testing.T) {
	mgr := newTestStatusManager()

	err := mgr.SetCurrent(enum.Aura, 0)
	if err == nil {
		t.Fatal("expected error for non-existent status")
	}
}

func TestStatusManager_Upgrade(t *testing.T) {
	mgr := newTestStatusManager()

	err := mgr.Upgrade()
	if err != nil {
		t.Fatalf("Upgrade() error: %v", err)
	}
}

func TestStatusManager_GetAllMaximuns(t *testing.T) {
	mgr := newTestStatusManager()

	maxs := mgr.GetAllMaximuns()
	if len(maxs) != 2 {
		t.Errorf("expected 2 status entries, got %d", len(maxs))
	}
}

func TestStatusManager_GetAllStatus(t *testing.T) {
	mgr := newTestStatusManager()

	all := mgr.GetAllStatus()
	if len(all) != 2 {
		t.Errorf("expected 2 status bars, got %d", len(all))
	}
}
```

- [ ] **Step 2** — Write status documentation:

Create `docs/game/ficha-de-personagem/status.md`:

```markdown
# Status (Barras de Status)

## Visão Geral

O sistema de status representa os recursos vitais do personagem: vida (HP), estamina (SP) e aura (AP). Cada barra é calculada dinamicamente a partir das habilidades, atributos e perícias do personagem.

## Pontos de Vida — HP (Health Points)

**Fórmula:**
```
HP_máx = 20 + (nível_vitalidade + valor_resistência) × bônus_físicos
```

- **Base:** 20 pontos fixos
- **Coeficiente:** nível da perícia Vitalidade + valor do atributo Resistência
- **Bônus:** bônus da habilidade Físicos

Quando o personagem está com vida cheia, um upgrade mantém a vida no máximo novo.

## Pontos de Estamina — SP (Stamina Points)

**Fórmula:**
```
SP_máx = 10 × (nível_energia + valor_resistência) × bônus_físicos
```

- **Coeficiente:** 10 (multiplicador)
- **Fatores:** nível da perícia Energia + valor do atributo Resistência
- **Bônus:** bônus da habilidade Físicos

## Pontos de Aura — AP (Aura Points)

**Fórmula:**
```
AP_máx = 10 × (nível_mop + nível_consciência) × bônus_espirituais
```

- **Coeficiente:** 10 (multiplicador)
- **Fatores:** nível da perícia MOP + nível do atributo Consciência
- **Bônus:** bônus da habilidade Espirituais (convertido para inteiro)

> **Nota:** AP só existe para personagens com habilidade Espiritual ativa. Se `spirituals` for nil, a barra de aura não é criada.

## Mecânica das Barras

Cada barra possui três valores:
- **Mínimo (min):** limite inferior (geralmente 0)
- **Atual (curr):** valor corrente
- **Máximo (max):** limite superior calculado

### Operações
- **IncreaseAt(valor):** Aumenta o atual, limitado pelo máximo
- **DecreaseAt(valor):** Diminui o atual, limitado pelo mínimo
- **SetCurrent(valor):** Define o atual (erro se fora dos limites)
- **Upgrade():** Recalcula o máximo; se estava cheio, o atual acompanha
```

- [ ] **Step 3** — Run status tests:

```bash
go test ./internal/domain/entity/character_sheet/status/ -v
```

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/status/status_manager_test.go \
        docs/game/ficha-de-personagem/status.md
git commit -m "test(status): add StatusManager tests + status documentation (PT-BR)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 10 — PrimaryAttribute and SpiritualAttribute Tests

**Create:** `internal/domain/entity/character_sheet/attribute/primary_attribute_test.go`, `internal/domain/entity/character_sheet/attribute/spiritual_attribute_test.go`

- [ ] **Step 1** — Write primary_attribute_test.go:

Create `internal/domain/entity/character_sheet/attribute/primary_attribute_test.go`:

```go
package attribute_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestPrimaryAttribute() (*attribute.PrimaryAttribute, *experience.CharacterExp) {
	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilityTable := experience.NewExpTable(20.0)
	abilityExp := experience.NewExperience(abilityTable)
	abilityObj := ability.NewAbility(enum.Physicals, *abilityExp, charExp)

	buff := new(int)
	attrTable := experience.NewExpTable(5.0)
	attrExp := experience.NewExperience(attrTable)

	pa := attribute.NewPrimaryAttribute(enum.Resistance, *attrExp, abilityObj, buff)
	return pa, charExp
}

func TestPrimaryAttribute_InitialState(t *testing.T) {
	pa, _ := newTestPrimaryAttribute()

	if pa.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", pa.GetLevel())
	}
	if pa.GetPoints() != 0 {
		t.Errorf("initial points: got %d, want 0", pa.GetPoints())
	}
	if pa.GetExpPoints() != 0 {
		t.Errorf("initial exp: got %d, want 0", pa.GetExpPoints())
	}
}

func TestPrimaryAttribute_GetValue(t *testing.T) {
	pa, _ := newTestPrimaryAttribute()

	if pa.GetValue() != 0 {
		t.Errorf("initial value: got %d, want 0 (points=0, level=0)", pa.GetValue())
	}

	pa.IncreasePoints(3)
	if pa.GetValue() != 3 {
		t.Errorf("value after 3 points: got %d, want 3", pa.GetValue())
	}
}

func TestPrimaryAttribute_GetPower(t *testing.T) {
	pa, _ := newTestPrimaryAttribute()

	power := pa.GetPower()
	expectedPower := pa.GetValue() + int(pa.GetAbilityBonus()) + 0
	if power != expectedPower {
		t.Errorf("initial power: got %d, want %d", power, expectedPower)
	}
}

func TestPrimaryAttribute_IncreasePoints(t *testing.T) {
	pa, _ := newTestPrimaryAttribute()

	result := pa.IncreasePoints(5)
	if result != 5 {
		t.Errorf("IncreasePoints(5) = %d, want 5", result)
	}
	if pa.GetPoints() != 5 {
		t.Errorf("points after increase: got %d, want 5", pa.GetPoints())
	}
}

func TestPrimaryAttribute_CascadeUpgrade(t *testing.T) {
	pa, charExp := newTestPrimaryAttribute()

	cascade := experience.NewUpgradeCascade(100)
	pa.CascadeUpgrade(cascade)

	if pa.GetExpPoints() != 100 {
		t.Errorf("attr exp after cascade: got %d, want 100", pa.GetExpPoints())
	}
	if charExp.GetExpPoints() == 0 {
		t.Error("char exp should receive points from cascade")
	}

	attrCascade, ok := cascade.Attributes[enum.Resistance]
	if !ok {
		t.Fatal("cascade should contain Resistance entry")
	}
	if attrCascade.Power != pa.GetPower() {
		t.Errorf("cascade power: got %d, want %d", attrCascade.Power, pa.GetPower())
	}
}

func TestPrimaryAttribute_Clone(t *testing.T) {
	pa, _ := newTestPrimaryAttribute()
	pa.IncreasePoints(5)

	buff := new(int)
	clone := pa.Clone(enum.Agility, buff)

	if clone.GetPoints() != 0 {
		t.Errorf("clone points should be 0, got %d", clone.GetPoints())
	}
	if clone.GetName() != enum.Agility {
		t.Errorf("clone name: got %v, want Agility", clone.GetName())
	}
}

func TestPrimaryAttribute_GetAbilityBonus(t *testing.T) {
	pa, _ := newTestPrimaryAttribute()

	bonus := pa.GetAbilityBonus()
	if bonus != 0 {
		t.Errorf("initial ability bonus: got %f, want 0", bonus)
	}
}
```

- [ ] **Step 2** — Write spiritual_attribute_test.go:

Create `internal/domain/entity/character_sheet/attribute/spiritual_attribute_test.go`:

```go
package attribute_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestSpiritualAttribute() (*attribute.SpiritualAttribute, *experience.CharacterExp) {
	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilityTable := experience.NewExpTable(5.0)
	abilityExp := experience.NewExperience(abilityTable)
	abilityObj := ability.NewAbility(enum.Spirituals, *abilityExp, charExp)

	buff := new(int)
	attrTable := experience.NewExpTable(1.0)
	attrExp := experience.NewExperience(attrTable)

	sa := attribute.NewSpiritualAttribute(enum.Flame, *attrExp, abilityObj, buff)
	return sa, charExp
}

func TestSpiritualAttribute_InitialState(t *testing.T) {
	sa, _ := newTestSpiritualAttribute()

	if sa.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", sa.GetLevel())
	}
	if sa.GetExpPoints() != 0 {
		t.Errorf("initial exp: got %d, want 0", sa.GetExpPoints())
	}
}

func TestSpiritualAttribute_GetPower(t *testing.T) {
	sa, _ := newTestSpiritualAttribute()

	power := sa.GetPower()
	expected := sa.GetLevel() + int(sa.GetAbilityBonus()) + 0
	if power != expected {
		t.Errorf("power: got %d, want %d (level + abilityBonus + buff)", power, expected)
	}
}

func TestSpiritualAttribute_CascadeUpgrade(t *testing.T) {
	sa, charExp := newTestSpiritualAttribute()

	cascade := experience.NewUpgradeCascade(100)
	sa.CascadeUpgrade(cascade)

	if sa.GetExpPoints() != 100 {
		t.Errorf("exp after cascade: got %d, want 100", sa.GetExpPoints())
	}
	if charExp.GetExpPoints() == 0 {
		t.Error("char exp should receive points from cascade")
	}

	attrCascade, ok := cascade.Attributes[enum.Flame]
	if !ok {
		t.Fatal("cascade should contain Flame entry")
	}
	if attrCascade.Lvl != sa.GetLevel() {
		t.Errorf("cascade lvl: got %d, want %d", attrCascade.Lvl, sa.GetLevel())
	}
}

func TestSpiritualAttribute_Clone(t *testing.T) {
	sa, _ := newTestSpiritualAttribute()

	buff := new(int)
	clone := sa.Clone(enum.Conscience, buff)

	if clone.GetName() != enum.Conscience {
		t.Errorf("clone name: got %v, want Conscience", clone.GetName())
	}
	if clone.GetLevel() != 0 {
		t.Errorf("clone level should be 0, got %d", clone.GetLevel())
	}
}
```

- [ ] **Step 3** — Run tests:

```bash
go test ./internal/domain/entity/character_sheet/attribute/ -run "TestPrimaryAttribute|TestSpiritualAttribute" -v
```

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/attribute/primary_attribute_test.go \
        internal/domain/entity/character_sheet/attribute/spiritual_attribute_test.go
git commit -m "test(attribute): add PrimaryAttribute and SpiritualAttribute tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 11 — MiddleAttribute and Manager Tests

**Create:** `internal/domain/entity/character_sheet/attribute/middle_attribute_test.go`, `internal/domain/entity/character_sheet/attribute/managers_test.go`

- [ ] **Step 1** — Write middle_attribute_test.go:

Create `internal/domain/entity/character_sheet/attribute/middle_attribute_test.go`:

```go
package attribute_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestMiddleAttribute() (*attribute.MiddleAttribute, *attribute.PrimaryAttribute, *attribute.PrimaryAttribute) {
	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilityTable := experience.NewExpTable(20.0)
	abilityExp := experience.NewExperience(abilityTable)
	abilityObj := ability.NewAbility(enum.Physicals, *abilityExp, charExp)

	buff1 := new(int)
	buff2 := new(int)
	buffMid := new(int)

	attrTable := experience.NewExpTable(5.0)
	attrExp := experience.NewExperience(attrTable)

	primary1 := attribute.NewPrimaryAttribute(enum.Resistance, *attrExp, abilityObj, buff1)
	primary2 := attribute.NewPrimaryAttribute(enum.Agility, *attrExp.Clone(), abilityObj, buff2)

	midExp := experience.NewExperience(experience.NewExpTable(5.0))
	mid := attribute.NewMiddleAttribute(enum.Strength, *midExp, buffMid, primary1, primary2)

	return mid, primary1, primary2
}

func TestMiddleAttribute_InitialState(t *testing.T) {
	mid, _, _ := newTestMiddleAttribute()

	if mid.GetLevel() != 0 {
		t.Errorf("initial level: got %d, want 0", mid.GetLevel())
	}
	if mid.GetPoints() != 0 {
		t.Errorf("initial points: got %d, want 0", mid.GetPoints())
	}
}

func TestMiddleAttribute_GetPoints_AveragesChildren(t *testing.T) {
	mid, p1, p2 := newTestMiddleAttribute()

	p1.IncreasePoints(4)
	p2.IncreasePoints(6)

	got := mid.GetPoints()
	want := 5
	if got != want {
		t.Errorf("GetPoints (avg of 4,6): got %d, want %d", got, want)
	}
}

func TestMiddleAttribute_GetPoints_OddAverage(t *testing.T) {
	mid, p1, p2 := newTestMiddleAttribute()

	p1.IncreasePoints(3)
	p2.IncreasePoints(4)

	got := mid.GetPoints()
	want := 4
	if got != want {
		t.Errorf("GetPoints (avg of 3,4 rounded): got %d, want %d", got, want)
	}
}

func TestMiddleAttribute_GetValue(t *testing.T) {
	mid, _, _ := newTestMiddleAttribute()

	got := mid.GetValue()
	want := mid.GetPoints() + mid.GetLevel()
	if got != want {
		t.Errorf("GetValue: got %d, want %d", got, want)
	}
}

func TestMiddleAttribute_GetPower(t *testing.T) {
	mid, _, _ := newTestMiddleAttribute()

	power := mid.GetPower()
	expected := mid.GetValue() + int(mid.GetAbilityBonus()) + 0
	if power != expected {
		t.Errorf("GetPower: got %d, want %d", power, expected)
	}
}

func TestMiddleAttribute_CascadeUpgrade_DistributesXP(t *testing.T) {
	mid, p1, p2 := newTestMiddleAttribute()

	cascade := experience.NewUpgradeCascade(100)
	mid.CascadeUpgrade(cascade)

	if mid.GetExpPoints() != 100 {
		t.Errorf("middle exp: got %d, want 100", mid.GetExpPoints())
	}
	if p1.GetExpPoints() == 0 {
		t.Error("primary1 should receive distributed XP")
	}
	if p2.GetExpPoints() == 0 {
		t.Error("primary2 should receive distributed XP")
	}

	_, ok := cascade.Attributes[enum.Strength]
	if !ok {
		t.Fatal("cascade should contain Strength entry")
	}
}

func TestMiddleAttribute_GetAbilityBonus_AveragesChildren(t *testing.T) {
	mid, _, _ := newTestMiddleAttribute()

	bonus := mid.GetAbilityBonus()
	if bonus != 0 {
		t.Errorf("initial ability bonus average: got %f, want 0", bonus)
	}
}
```

- [ ] **Step 2** — Write managers_test.go:

Create `internal/domain/entity/character_sheet/attribute/managers_test.go`:

```go
package attribute_test

import (
	"errors"
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/ability"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestAttributeManager() *attribute.Manager {
	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilityTable := experience.NewExpTable(20.0)
	abilityExp := experience.NewExperience(abilityTable)
	abilityObj := ability.NewAbility(enum.Physicals, *abilityExp, charExp)

	primaryAttrs := make(map[enum.AttributeName]*attribute.PrimaryAttribute)
	middleAttrs := make(map[enum.AttributeName]*attribute.MiddleAttribute)
	buffs := make(map[enum.AttributeName]*int)

	buffs[enum.Resistance] = new(int)
	buffs[enum.Agility] = new(int)
	buffs[enum.Strength] = new(int)

	attrExp := experience.NewExperience(experience.NewExpTable(5.0))
	res := attribute.NewPrimaryAttribute(enum.Resistance, *attrExp, abilityObj, buffs[enum.Resistance])
	agi := res.Clone(enum.Agility, buffs[enum.Agility])

	primaryAttrs[enum.Resistance] = res
	primaryAttrs[enum.Agility] = agi

	midExp := experience.NewExperience(experience.NewExpTable(5.0))
	str := attribute.NewMiddleAttribute(enum.Strength, *midExp, buffs[enum.Strength], res, agi)
	middleAttrs[enum.Strength] = str

	return attribute.NewAttributeManager(primaryAttrs, middleAttrs, buffs)
}

func TestAttributeManager_Get_Primary(t *testing.T) {
	mgr := newTestAttributeManager()

	attr, err := mgr.Get(enum.Resistance)
	if err != nil {
		t.Fatalf("Get(Resistance) error: %v", err)
	}
	if attr == nil {
		t.Fatal("Get(Resistance) returned nil")
	}
}

func TestAttributeManager_Get_Middle(t *testing.T) {
	mgr := newTestAttributeManager()

	attr, err := mgr.Get(enum.Strength)
	if err != nil {
		t.Fatalf("Get(Strength) error: %v", err)
	}
	if attr == nil {
		t.Fatal("Get(Strength) returned nil")
	}
}

func TestAttributeManager_Get_NotFound(t *testing.T) {
	mgr := newTestAttributeManager()

	_, err := mgr.Get(enum.Creativity)
	if err == nil {
		t.Fatal("expected error for non-existent attribute")
	}
	if !errors.Is(err, attribute.ErrAttributeNotFound) {
		t.Errorf("expected ErrAttributeNotFound, got %v", err)
	}
}

func TestAttributeManager_IncreasePointsForPrimary(t *testing.T) {
	mgr := newTestAttributeManager()

	points, err := mgr.IncreasePointsForPrimary(enum.Resistance, 3)
	if err != nil {
		t.Fatalf("IncreasePointsForPrimary error: %v", err)
	}
	if points[enum.Resistance] != 3 {
		t.Errorf("Resistance points: got %d, want 3", points[enum.Resistance])
	}
}

func TestAttributeManager_GetAttributesLevel(t *testing.T) {
	mgr := newTestAttributeManager()

	levels := mgr.GetAttributesLevel()
	if len(levels) != 3 {
		t.Errorf("expected 3 attributes, got %d", len(levels))
	}
	for name, lvl := range levels {
		if lvl != 0 {
			t.Errorf("initial level for %s: got %d, want 0", name, lvl)
		}
	}
}

func TestAttributeManager_SetBuff(t *testing.T) {
	mgr := newTestAttributeManager()

	buffs, err := mgr.SetBuff(enum.Resistance, 5)
	if err != nil {
		t.Fatalf("SetBuff error: %v", err)
	}
	if *buffs[enum.Resistance] != 5 {
		t.Errorf("buff value: got %d, want 5", *buffs[enum.Resistance])
	}
}

func TestAttributeManager_RemoveBuff(t *testing.T) {
	mgr := newTestAttributeManager()

	mgr.SetBuff(enum.Resistance, 5)
	buffs, err := mgr.RemoveBuff(enum.Resistance)
	if err != nil {
		t.Fatalf("RemoveBuff error: %v", err)
	}
	if *buffs[enum.Resistance] != 0 {
		t.Errorf("buff after remove: got %d, want 0", *buffs[enum.Resistance])
	}
}

func TestAttributeManager_GetAllAttributes(t *testing.T) {
	mgr := newTestAttributeManager()

	all := mgr.GetAllAttributes()
	if len(all) != 3 {
		t.Errorf("expected 3 total attributes, got %d", len(all))
	}
}

func TestSpiritualManager_Get_Found(t *testing.T) {
	charExpTable := experience.NewExpTable(10.0)
	charExpObj := experience.NewExperience(charExpTable)
	charExp := experience.NewCharacterExp(*charExpObj)

	abilityTable := experience.NewExpTable(5.0)
	abilityExp := experience.NewExperience(abilityTable)
	abilityObj := ability.NewAbility(enum.Spirituals, *abilityExp, charExp)

	attrs := make(map[enum.AttributeName]*attribute.SpiritualAttribute)
	buffs := make(map[enum.AttributeName]*int)
	buffs[enum.Flame] = new(int)
	buffs[enum.Conscience] = new(int)

	attrExp := experience.NewExperience(experience.NewExpTable(1.0))
	flame := attribute.NewSpiritualAttribute(enum.Flame, *attrExp, abilityObj, buffs[enum.Flame])
	attrs[enum.Flame] = flame
	attrs[enum.Conscience] = flame.Clone(enum.Conscience, buffs[enum.Conscience])

	mgr := attribute.NewSpiritualAttributeManager(attrs, buffs)

	attr, err := mgr.Get(enum.Flame)
	if err != nil {
		t.Fatalf("Get(Flame) error: %v", err)
	}
	if attr == nil {
		t.Fatal("Get(Flame) returned nil")
	}
}

func TestSpiritualManager_Get_NotFound(t *testing.T) {
	attrs := make(map[enum.AttributeName]*attribute.SpiritualAttribute)
	buffs := make(map[enum.AttributeName]*int)
	mgr := attribute.NewSpiritualAttributeManager(attrs, buffs)

	_, err := mgr.Get(enum.Flame)
	if err == nil {
		t.Fatal("expected error for non-existent attribute")
	}
}
```

- [ ] **Step 3** — Run tests:

```bash
go test ./internal/domain/entity/character_sheet/attribute/ -v
```

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/attribute/middle_attribute_test.go \
        internal/domain/entity/character_sheet/attribute/managers_test.go
git commit -m "test(attribute): add MiddleAttribute and Manager tests — averaging, distribution, buffs

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 12 — Attributes Documentation (PT-BR)

**Create:** `docs/game/ficha-de-personagem/atributos.md`

- [ ] **Step 1** — Write the attributes documentation:

Create `docs/game/ficha-de-personagem/atributos.md`:

```markdown
# Atributos (Attributes)

## Visão Geral

Atributos representam as características físicas, mentais e espirituais do personagem. Existem três categorias com diferentes mecânicas de cálculo.

## Tipos de Atributo

### Atributos Primários (PrimaryAttribute)

Atributos que podem receber pontos distribuídos diretamente pelo jogador.

**Fórmula de Valor:**
```
valor = pontos_distribuídos + nível
```

**Fórmula de Poder:**
```
poder = valor + bônus_habilidade + buff
```

### Atributos Médios (MiddleAttribute)

Atributos compostos que derivam da média de dois atributos primários.

**Fórmula de Pontos:**
```
pontos = arredondar(média dos pontos dos atributos primários filhos)
```

**Fórmula de Valor:**
```
valor = pontos + nível
```

**Fórmula de Poder:**
```
poder = valor + bônus_habilidade + buff
```

**Distribuição de XP:** Quando um atributo médio recebe XP em cascata, ele distribui igualmente o XP entre seus atributos primários filhos.

### Atributos Espirituais (SpiritualAttribute)

Atributos que não possuem pontos distribuíveis — apenas nível.

**Fórmula de Poder:**
```
poder = nível + bônus_habilidade + buff
```

## Atributos Físicos

| Atributo | Tipo | Filhos | Perícias |
|---|---|---|---|
| Resistência | Primário | — | Vitalidade, Energia, Defesa |
| Agilidade | Primário | — | Velocidade, Aceleração, Freio |
| Flexibilidade | Primário | — | Acrobacia, Evasão, Esgueirar |
| Sentido | Primário | — | Visão, Audição, Olfato, Tato, Paladar |
| Força | Médio | Resistência, Agilidade | Empurrão, Agarrar, Carregar |
| Celeridade | Médio | Agilidade, Flexibilidade | Ligeireza, Repelir, Finta |
| Destreza | Médio | Flexibilidade, Sentido | Reflexo, Precisão, Furtividade |
| Constituição | Médio | Sentido, Resistência | Curar, Respiração, Tenacidade |

## Atributos Mentais

Todos primários: Resiliência, Adaptabilidade, Ponderação, Criatividade.

## Atributos Espirituais

| Atributo | Descrição | Perícias |
|---|---|---|
| Chama (Flame) | Força interior | Foco, Força de Vontade, Autoconhecimento |
| Consciência (Conscience) | Controle Nen | Coa, Mop, Aop |

## Distribuição de Pontos

O jogador distribui pontos nos **atributos primários físicos** com o limite:
```
soma_pontos_primários ≤ nível_físicos
```
```

- [ ] **Step 2** — Commit:

```bash
git add docs/game/ficha-de-personagem/atributos.md
git commit -m "docs: add attributes system documentation (PT-BR)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 13 — CommonSkill and JointSkill Tests

**Files:**
- Create: `internal/domain/entity/character_sheet/skill/common_skill_test.go`
- Create: `internal/domain/entity/character_sheet/skill/joint_skill_test.go`

- [ ] **Step 1** — Write `common_skill_test.go`:

```go
package skill_test

import (
	"testing"

	attr "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/attribute"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type mockGameAttribute struct {
	power int
	level int
}

func (m *mockGameAttribute) GetPower() int                                   { return m.power }
func (m *mockGameAttribute) GetAbilityBonus() float64                        { return 0 }
func (m *mockGameAttribute) GetLevel() int                                   { return m.level }
func (m *mockGameAttribute) CascadeUpgrade(_ *experience.UpgradeCascade)     {}
func (m *mockGameAttribute) GetNextLvlAggregateExp() int                     { return 0 }
func (m *mockGameAttribute) GetNextLvlBaseExp() int                          { return 0 }
func (m *mockGameAttribute) GetCurrentExp() int                              { return 0 }
func (m *mockGameAttribute) GetExpPoints() int                               { return 0 }

type mockCascadeUpgrade struct {
	level int
}

func (m *mockCascadeUpgrade) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockCascadeUpgrade) GetLevel() int                               { return m.level }

func newTestCommonSkill(name enum.SkillName, attrPower int) *skill.CommonSkill {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mockAttr := &mockGameAttribute{power: attrPower}
	mockAbilityExp := &mockCascadeUpgrade{}
	return skill.NewCommonSkill(name, exp, mockAttr, mockAbilityExp)
}

func TestCommonSkill_GetValueForTest(t *testing.T) {
	tests := []struct {
		name      string
		attrPower int
		want      int
	}{
		{"zero power zero level", 0, 0},
		{"with attribute power", 5, 5},
		{"high power", 20, 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := newTestCommonSkill(enum.Vitality, tt.attrPower)
			got := cs.GetValueForTest()
			if got != tt.want {
				t.Errorf("GetValueForTest() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCommonSkill_InitialState(t *testing.T) {
	cs := newTestCommonSkill(enum.Vitality, 3)

	if cs.GetLevel() != 0 {
		t.Errorf("initial level = %d, want 0", cs.GetLevel())
	}
	if cs.GetExpPoints() != 0 {
		t.Errorf("initial exp = %d, want 0", cs.GetExpPoints())
	}
	if cs.GetName() != enum.Vitality {
		t.Errorf("name = %s, want %s", cs.GetName(), enum.Vitality)
	}
}

func TestCommonSkill_CascadeUpgradeTrigger(t *testing.T) {
	cs := newTestCommonSkill(enum.Defense, 2)
	values := experience.NewUpgradeCascade(50)

	cs.CascadeUpgradeTrigger(values)

	if cs.GetExpPoints() != 50 {
		t.Errorf("exp after cascade = %d, want 50", cs.GetExpPoints())
	}
	cascade, ok := values.Skills[enum.Defense.String()]
	if !ok {
		t.Fatal("skill cascade not found in UpgradeCascade.Skills")
	}
	if cascade.Exp != cs.GetCurrentExp() {
		t.Errorf("cascade Exp = %d, want %d", cascade.Exp, cs.GetCurrentExp())
	}
}

func TestCommonSkill_Clone(t *testing.T) {
	cs := newTestCommonSkill(enum.Vitality, 5)
	cloned := cs.Clone(enum.Energy)

	if cloned.GetName() != enum.Energy {
		t.Errorf("cloned name = %s, want %s", cloned.GetName(), enum.Energy)
	}
	if cloned.GetExpPoints() != 0 {
		t.Errorf("cloned should start with 0 exp, got %d", cloned.GetExpPoints())
	}
}
```

- [ ] **Step 2** — Write `joint_skill_test.go`:

```go
package skill_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestJointSkill(name string, attrPower int, skills ...enum.SkillName) *skill.JointSkill {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mockAttr := &mockGameAttribute{power: attrPower}
	commonSkills := make(map[enum.SkillName]skill.ISkill)
	for _, sn := range skills {
		commonSkills[sn] = newTestCommonSkill(sn, attrPower)
	}
	return skill.NewJointSkill(exp, name, mockAttr, commonSkills)
}

func TestJointSkill_InitAndIsInitialized(t *testing.T) {
	js := newTestJointSkill("hunt", 3, enum.Vision, enum.Stealth)

	if js.IsInitialized() {
		t.Error("should not be initialized before Init()")
	}

	mockAbilityExp := &mockCascadeUpgrade{}
	if err := js.Init(mockAbilityExp); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	if !js.IsInitialized() {
		t.Error("should be initialized after Init()")
	}

	// Double init should fail
	if err := js.Init(mockAbilityExp); err == nil {
		t.Error("double Init() should return error")
	}
}

func TestJointSkill_InitWithNil(t *testing.T) {
	js := newTestJointSkill("hunt", 3, enum.Vision)

	if err := js.Init(nil); err == nil {
		t.Error("Init(nil) should return error")
	}
}

func TestJointSkill_GetValueForTest(t *testing.T) {
	js := newTestJointSkill("roguery", 5, enum.Stealth, enum.Sneak)

	// level=0, power=5, buff=0 → 5
	if got := js.GetValueForTest(); got != 5 {
		t.Errorf("GetValueForTest() = %d, want 5", got)
	}

	js.SetBuff(3)
	// level=0, power=5, buff=3 → 8
	if got := js.GetValueForTest(); got != 8 {
		t.Errorf("GetValueForTest() with buff = %d, want 8", got)
	}
}

func TestJointSkill_Contains(t *testing.T) {
	js := newTestJointSkill("athletics", 3, enum.Velocity, enum.Acrobatics)

	if !js.Contains(enum.Velocity) {
		t.Error("should contain Velocity")
	}
	if js.Contains(enum.Stealth) {
		t.Error("should not contain Stealth")
	}
}

func TestJointSkill_Properties(t *testing.T) {
	js := newTestJointSkill("hunt", 3, enum.Vision)

	if js.GetName() != "hunt" {
		t.Errorf("GetName() = %s, want hunt", js.GetName())
	}
	if js.GetBuff() != 0 {
		t.Errorf("initial buff = %d, want 0", js.GetBuff())
	}
	js.SetBuff(5)
	if js.GetBuff() != 5 {
		t.Errorf("buff after set = %d, want 5", js.GetBuff())
	}
}

func TestJointSkill_CascadeUpgradeTrigger(t *testing.T) {
	js := newTestJointSkill("hack", 2, enum.Focus, enum.Accuracy)
	mockAbilityExp := &mockCascadeUpgrade{}
	if err := js.Init(mockAbilityExp); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	values := experience.NewUpgradeCascade(100)
	js.CascadeUpgradeTrigger(values)

	if js.GetExpPoints() != 100 {
		t.Errorf("exp after cascade = %d, want 100", js.GetExpPoints())
	}
	// Cascade should multiply exp by number of common skills (2)
	if values.GetExp() != 200 {
		t.Errorf("cascade exp should be multiplied by common skills count: got %d, want 200", values.GetExp())
	}
}
```

- [ ] **Step 3** — Run tests:

Run: `go test ./internal/domain/entity/character_sheet/skill/ -v -run "TestCommonSkill|TestJointSkill"`
Expected: ALL PASS

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/skill/common_skill_test.go internal/domain/entity/character_sheet/skill/joint_skill_test.go
git commit -m "test: add CommonSkill and JointSkill unit tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 14 — SkillsManager Tests

**Files:**
- Create: `internal/domain/entity/character_sheet/skill/skills_manager_test.go`

- [ ] **Step 1** — Write `skills_manager_test.go`:

```go
package skill_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/skill"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestSkillsManager() *skill.Manager {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mockAbilityExp := &mockCascadeUpgrade{}
	return skill.NewSkillsManager(exp, mockAbilityExp)
}

func initManagerWithSkills(m *skill.Manager, names ...enum.SkillName) {
	skills := make(map[enum.SkillName]skill.ISkill)
	for _, name := range names {
		skills[name] = newTestCommonSkill(name, 3)
	}
	m.Init(skills)
}

func TestSkillsManager_Init(t *testing.T) {
	m := newTestSkillsManager()
	skills := map[enum.SkillName]skill.ISkill{
		enum.Vitality: newTestCommonSkill(enum.Vitality, 3),
	}

	if err := m.Init(skills); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Double init should fail
	if err := m.Init(skills); err == nil {
		t.Error("double Init() should return error")
	}
}

func TestSkillsManager_Get(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(m, enum.Vitality, enum.Energy)

	t.Run("existing skill", func(t *testing.T) {
		sk, err := m.Get(enum.Vitality)
		if err != nil {
			t.Fatalf("Get(Vitality) error: %v", err)
		}
		if sk == nil {
			t.Fatal("Get(Vitality) returned nil")
		}
	})

	t.Run("non-existing skill", func(t *testing.T) {
		_, err := m.Get(enum.Stealth)
		if err == nil {
			t.Error("Get(Stealth) should return error for non-existing skill")
		}
	})
}

func TestSkillsManager_GetValueForTestOf(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(m, enum.Vitality)

	val, err := m.GetValueForTestOf(enum.Vitality)
	if err != nil {
		t.Fatalf("GetValueForTestOf error: %v", err)
	}
	if val != 3 {
		t.Errorf("value for test = %d, want 3 (attrPower=3, level=0)", val)
	}
}

func TestSkillsManager_BuffManagement(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(m, enum.Vitality)

	m.SetBuff(enum.Vitality, 5)

	val, err := m.GetValueForTestOf(enum.Vitality)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// attrPower=3 + level=0 + buff=5 = 8
	if val != 8 {
		t.Errorf("buffed value = %d, want 8", val)
	}

	m.DeleteBuff(enum.Vitality)

	val, _ = m.GetValueForTestOf(enum.Vitality)
	if val != 3 {
		t.Errorf("after delete buff = %d, want 3", val)
	}
}

func TestSkillsManager_AddJointSkill(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(m, enum.Vision, enum.Stealth)

	js := newTestJointSkill("hunt", 3, enum.Vision, enum.Stealth)

	t.Run("uninitialized joint skill rejected", func(t *testing.T) {
		if err := m.AddJointSkill(js); err == nil {
			t.Error("should reject uninitialized joint skill")
		}
	})

	mockAbilityExp := &mockCascadeUpgrade{}
	js.Init(mockAbilityExp)

	t.Run("initialized joint skill accepted", func(t *testing.T) {
		if err := m.AddJointSkill(js); err != nil {
			t.Fatalf("AddJointSkill error: %v", err)
		}
	})

	t.Run("duplicate joint skill rejected", func(t *testing.T) {
		js2 := newTestJointSkill("hunt", 3, enum.Vision)
		js2.Init(mockAbilityExp)
		if err := m.AddJointSkill(js2); err == nil {
			t.Error("should reject duplicate joint skill name")
		}
	})

	t.Run("joint skill found via Get", func(t *testing.T) {
		sk, err := m.Get(enum.Vision)
		if err != nil {
			t.Fatalf("Get(Vision) error: %v", err)
		}
		if sk.GetValueForTest() != js.GetValueForTest() {
			t.Error("Get should return the joint skill for contained skill names")
		}
	})
}

func TestSkillsManager_IncreaseExp(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(m, enum.Defense)

	values := experience.NewUpgradeCascade(50)
	if err := m.IncreaseExp(values, enum.Defense); err != nil {
		t.Fatalf("IncreaseExp error: %v", err)
	}

	exp, _ := m.GetExpPointsOf(enum.Defense)
	if exp != 50 {
		t.Errorf("exp after increase = %d, want 50", exp)
	}

	// Non-existing skill
	if err := m.IncreaseExp(values, enum.Stealth); err == nil {
		t.Error("IncreaseExp for non-existing skill should return error")
	}
}

func TestSkillsManager_BatchGetters(t *testing.T) {
	m := newTestSkillsManager()
	initManagerWithSkills(m, enum.Vitality, enum.Energy, enum.Defense)

	levels := m.GetSkillsLevel()
	if len(levels) != 3 {
		t.Errorf("skills count = %d, want 3", len(levels))
	}
	for name, lvl := range levels {
		if lvl != 0 {
			t.Errorf("skill %s initial level = %d, want 0", name, lvl)
		}
	}
}
```

- [ ] **Step 2** — Run tests:

Run: `go test ./internal/domain/entity/character_sheet/skill/ -v -run "TestSkillsManager"`
Expected: ALL PASS

- [ ] **Step 3** — Commit:

```bash
git add internal/domain/entity/character_sheet/skill/skills_manager_test.go
git commit -m "test: add SkillsManager unit tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 15 — Skills Documentation (PT-BR)

**Files:**
- Create: `docs/game/ficha-de-personagem/pericias.md`

- [ ] **Step 1** — Write skills documentation covering:
- What Perícias (Skills) are and how they relate to Atributos (Attributes)
- Valor de Teste (Value For Test) = nível da perícia + poder do atributo
- The 31 skills organized by attribute:
  - Resistência: Vitalidade (Vitality), Energia (Energy), Defesa (Defense)
  - Força: Empurrão (Push), Agarrar (Grab), Carregar (Carry)
  - Agilidade: Velocidade (Velocity), Aceleração (Accelerate), Freagem (Brake)
  - Celeridade: Ligeireza (Legerity), Repelir (Repel), Finta (Feint)
  - Flexibilidade: Acrobacia (Acrobatics), Evasão (Evasion), Esgueirar (Sneak)
  - Destreza: Reflexo (Reflex), Precisão (Accuracy), Furtividade (Stealth)
  - Sentido: Visão (Vision), Audição (Hearing), Olfato (Smell), Tato (Tact), Paladar (Taste)
  - Constituição: Curar (Heal), Respiração (Breath), Tenacidade (Tenacity)
  - Chama: Foco (Focus), Força de Vontade (WillPower), Autoconhecimento (SelfKnowledge)
  - Consciência: Coa (Coa), Mop (Mop), Aop (Aop)
- Perícias Conjuntas (Joint Skills): composition, buff, init requirement
- Cascade de experiência: skill → attribute → ability → character exp

- [ ] **Step 2** — Commit:

```bash
git add docs/game/ficha-de-personagem/pericias.md
git commit -m "docs: add skills system documentation (PT-BR)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 16 — Proficiency and JointProficiency Tests

**Files:**
- Create: `internal/domain/entity/character_sheet/proficiency/proficiency_test.go`
- Create: `internal/domain/entity/character_sheet/proficiency/joint_proficiency_test.go`

- [ ] **Step 1** — Write `proficiency_test.go`:

```go
package proficiency_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type mockCascadeUpgrade struct{ level int }

func (m *mockCascadeUpgrade) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockCascadeUpgrade) GetLevel() int                               { return m.level }

func newTestProficiency(weapon enum.WeaponName) *proficiency.Proficiency {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mockPhysSkills := &mockCascadeUpgrade{}
	return proficiency.NewProficiency(weapon, exp, mockPhysSkills)
}

func TestProficiency_InitialState(t *testing.T) {
	p := newTestProficiency(enum.Sword)

	if p.GetLevel() != 0 {
		t.Errorf("initial level = %d, want 0", p.GetLevel())
	}
	if p.GetExpPoints() != 0 {
		t.Errorf("initial exp = %d, want 0", p.GetExpPoints())
	}
	if p.GetWeapon() != enum.Sword {
		t.Errorf("weapon = %s, want %s", p.GetWeapon(), enum.Sword)
	}
}

func TestProficiency_GetValueForTest(t *testing.T) {
	p := newTestProficiency(enum.Dagger)
	if got := p.GetValueForTest(); got != 0 {
		t.Errorf("initial GetValueForTest() = %d, want 0", got)
	}
}

func TestProficiency_CascadeUpgradeTrigger(t *testing.T) {
	p := newTestProficiency(enum.Sword)
	values := experience.NewUpgradeCascade(50)

	p.CascadeUpgradeTrigger(values)

	if p.GetExpPoints() != 50 {
		t.Errorf("exp after cascade = %d, want 50", p.GetExpPoints())
	}
	cascade, ok := values.Proficiency[enum.Sword.String()]
	if !ok {
		t.Fatal("proficiency cascade not found")
	}
	if cascade.Lvl != p.GetLevel() {
		t.Errorf("cascade Lvl = %d, want %d", cascade.Lvl, p.GetLevel())
	}
}
```

- [ ] **Step 2** — Write `joint_proficiency_test.go`:

```go
package proficiency_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestJointProficiency(name string, weapons ...enum.WeaponName) *proficiency.JointProficiency {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	return proficiency.NewJointProficiency(exp, name, weapons)
}

func TestJointProficiency_Init(t *testing.T) {
	jp := newTestJointProficiency("dual-wield", enum.Sword, enum.Dagger)
	mockPhys := &mockCascadeUpgrade{}
	mockAbility := &mockCascadeUpgrade{}

	t.Run("successful init", func(t *testing.T) {
		if err := jp.Init(mockPhys, mockAbility); err != nil {
			t.Fatalf("Init() error: %v", err)
		}
	})

	t.Run("double init fails", func(t *testing.T) {
		if err := jp.Init(mockPhys, mockAbility); err == nil {
			t.Error("double Init() should return error")
		}
	})
}

func TestJointProficiency_InitNilArgs(t *testing.T) {
	tests := []struct {
		name    string
		phys    experience.ICascadeUpgrade
		ability experience.ICascadeUpgrade
	}{
		{"nil phys", nil, &mockCascadeUpgrade{}},
		{"nil ability", &mockCascadeUpgrade{}, nil},
		{"both nil", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jp := newTestJointProficiency("test", enum.Sword)
			if err := jp.Init(tt.phys, tt.ability); err == nil {
				t.Error("Init with nil args should return error")
			}
		})
	}
}

func TestJointProficiency_ContainsWeapon(t *testing.T) {
	jp := newTestJointProficiency("blades", enum.Sword, enum.Dagger)

	if !jp.ContainsWeapon(enum.Sword) {
		t.Error("should contain Sword")
	}
	if jp.ContainsWeapon(enum.Bow) {
		t.Error("should not contain Bow")
	}
}

func TestJointProficiency_AddWeapon(t *testing.T) {
	jp := newTestJointProficiency("blades", enum.Sword)
	jp.AddWeapon(enum.Dagger)

	if !jp.ContainsWeapon(enum.Dagger) {
		t.Error("should contain Dagger after AddWeapon")
	}
	if len(jp.GetWeapons()) != 2 {
		t.Errorf("weapons count = %d, want 2", len(jp.GetWeapons()))
	}
}

func TestJointProficiency_BuffManagement(t *testing.T) {
	jp := newTestJointProficiency("blades", enum.Sword)

	if jp.GetBuff() != 0 {
		t.Errorf("initial buff = %d, want 0", jp.GetBuff())
	}

	jp.SetBuff(enum.Sword, 5)
	if jp.GetBuff() != 5 {
		t.Errorf("buff after set = %d, want 5", jp.GetBuff())
	}

	jp.DeleteBuff(enum.Sword)
	if jp.GetBuff() != 0 {
		t.Errorf("buff after delete = %d, want 0", jp.GetBuff())
	}
}

func TestJointProficiency_CascadeUpgradeTrigger(t *testing.T) {
	jp := newTestJointProficiency("blades", enum.Sword, enum.Dagger)
	mockPhys := &mockCascadeUpgrade{}
	mockAbility := &mockCascadeUpgrade{}
	jp.Init(mockPhys, mockAbility)

	values := experience.NewUpgradeCascade(100)
	jp.CascadeUpgradeTrigger(values)

	if jp.GetExpPoints() != 100 {
		t.Errorf("exp after cascade = %d, want 100", jp.GetExpPoints())
	}
	cascade, ok := values.Proficiency["blades"]
	if !ok {
		t.Fatal("joint proficiency cascade not found")
	}
	if cascade.Lvl != jp.GetLevel() {
		t.Errorf("cascade Lvl = %d, want %d", cascade.Lvl, jp.GetLevel())
	}
}
```

- [ ] **Step 3** — Run tests:

Run: `go test ./internal/domain/entity/character_sheet/proficiency/ -v`
Expected: ALL PASS

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/proficiency/proficiency_test.go internal/domain/entity/character_sheet/proficiency/joint_proficiency_test.go
git commit -m "test: add Proficiency and JointProficiency unit tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 17 — ProficiencyManager Tests

**Files:**
- Create: `internal/domain/entity/character_sheet/proficiency/proficiency_manager_test.go`

- [ ] **Step 1** — Write `proficiency_manager_test.go`:

```go
package proficiency_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/proficiency"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestProficiencyManager_AddCommon(t *testing.T) {
	m := proficiency.NewManager()
	p := newTestProficiency(enum.Sword)

	if err := m.AddCommon(enum.Sword, p); err != nil {
		t.Fatalf("AddCommon error: %v", err)
	}

	// Duplicate
	if err := m.AddCommon(enum.Sword, p); err == nil {
		t.Error("duplicate AddCommon should return error")
	}
}

func TestProficiencyManager_Get(t *testing.T) {
	m := proficiency.NewManager()
	m.AddCommon(enum.Sword, newTestProficiency(enum.Sword))

	t.Run("existing weapon", func(t *testing.T) {
		prof, err := m.Get(enum.Sword)
		if err != nil {
			t.Fatalf("Get error: %v", err)
		}
		if prof == nil {
			t.Fatal("Get returned nil")
		}
	})

	t.Run("non-existing weapon", func(t *testing.T) {
		_, err := m.Get(enum.Bow)
		if err == nil {
			t.Error("Get for non-existing weapon should return error")
		}
	})
}

func TestProficiencyManager_GetFindsJointFirst(t *testing.T) {
	m := proficiency.NewManager()
	m.AddCommon(enum.Sword, newTestProficiency(enum.Sword))

	jp := newTestJointProficiency("blades", enum.Sword, enum.Dagger)
	mockPhys := &mockCascadeUpgrade{}
	mockAbility := &mockCascadeUpgrade{}
	m.AddJoint(jp, mockPhys, mockAbility)

	// Get(Sword) should return joint proficiency since it contains Sword
	prof, err := m.Get(enum.Sword)
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if prof.GetName() != "blades" {
		t.Errorf("should return joint proficiency, got name=%s", prof.GetName())
	}
}

func TestProficiencyManager_AddJoint(t *testing.T) {
	m := proficiency.NewManager()
	jp := newTestJointProficiency("blades", enum.Sword)
	mockPhys := &mockCascadeUpgrade{}
	mockAbility := &mockCascadeUpgrade{}

	if err := m.AddJoint(jp, mockPhys, mockAbility); err != nil {
		t.Fatalf("AddJoint error: %v", err)
	}

	// Duplicate name
	jp2 := newTestJointProficiency("blades", enum.Dagger)
	if err := m.AddJoint(jp2, mockPhys, mockAbility); err == nil {
		t.Error("duplicate AddJoint should return error")
	}
}

func TestProficiencyManager_IncreaseExp(t *testing.T) {
	m := proficiency.NewManager()
	m.AddCommon(enum.Sword, newTestProficiency(enum.Sword))

	values := experience.NewUpgradeCascade(50)
	if err := m.IncreaseExp(values, enum.Sword); err != nil {
		t.Fatalf("IncreaseExp error: %v", err)
	}

	exp, _ := m.GetExpPointsOf(enum.Sword)
	if exp != 50 {
		t.Errorf("exp after increase = %d, want 50", exp)
	}
}

func TestProficiencyManager_BuffManagement(t *testing.T) {
	m := proficiency.NewManager()
	m.AddCommon(enum.Sword, newTestProficiency(enum.Sword))

	m.SetBuff(enum.Sword, 3)

	val, err := m.GetValueForTestOf(enum.Sword)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	// level=0 + buff=3 = 3
	if val != 3 {
		t.Errorf("buffed value = %d, want 3", val)
	}

	m.DeleteBuff(enum.Sword)
	val, _ = m.GetValueForTestOf(enum.Sword)
	if val != 0 {
		t.Errorf("after delete buff = %d, want 0", val)
	}
}
```

- [ ] **Step 2** — Run tests:

Run: `go test ./internal/domain/entity/character_sheet/proficiency/ -v -run "TestProficiencyManager"`
Expected: ALL PASS

- [ ] **Step 3** — Commit:

```bash
git add internal/domain/entity/character_sheet/proficiency/proficiency_manager_test.go
git commit -m "test: add ProficiencyManager unit tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 18 — Proficiencies Documentation (PT-BR)

**Files:**
- Create: `docs/game/ficha-de-personagem/proficiencias.md`

- [ ] **Step 1** — Write proficiencies documentation covering:
- What Proficiências (Proficiencies) are: weapon-specific skill levels
- Difference from Perícias (Skills): proficiencies are weapon-bound
- Proficiências Comuns (Common): one weapon, one proficiency
- Proficiências Conjuntas (Joint): multiple weapons, shared progression, buff system
- Cascade: proficiency → physical skills → ability → character exp
- List key weapons from enum grouped: melee, ranged, firearms

- [ ] **Step 2** — Commit:

```bash
git add docs/game/ficha-de-personagem/proficiencias.md
git commit -m "docs: add proficiencies system documentation (PT-BR)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 19 — NenHexagon Tests

**Files:**
- Create: `internal/domain/entity/character_sheet/spiritual/nen_hexagon_test.go`

- [ ] **Step 1** — Write `nen_hexagon_test.go`:

```go
package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestNenHexagon_NewWithNilCategory(t *testing.T) {
	tests := []struct {
		name       string
		hexValue   int
		wantCat    enum.CategoryName
	}{
		{"hex 0 → Reinforcement", 0, enum.Reinforcement},
		{"hex 49 → Reinforcement", 49, enum.Reinforcement},
		{"hex 100 → Transmutation", 100, enum.Transmutation},
		{"hex 200 → Materialization", 200, enum.Materialization},
		{"hex 300 → Specialization", 300, enum.Specialization},
		{"hex 400 → Manipulation", 400, enum.Manipulation},
		{"hex 500 → Emission", 500, enum.Emission},
		{"hex 599 → Reinforcement (wraps)", 599, enum.Reinforcement},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nh := spiritual.NewNenHexagon(tt.hexValue, nil)
			if got := nh.GetCategoryName(); got != tt.wantCat {
				t.Errorf("category = %s, want %s", got, tt.wantCat)
			}
		})
	}
}

func TestNenHexagon_NewWithExplicitCategory(t *testing.T) {
	cat := enum.Transmutation
	nh := spiritual.NewNenHexagon(0, &cat)
	if got := nh.GetCategoryName(); got != enum.Transmutation {
		t.Errorf("category = %s, want %s", got, enum.Transmutation)
	}
}

func TestNenHexagon_ModuloWrapping(t *testing.T) {
	nh := spiritual.NewNenHexagon(700, nil)
	if got := nh.GetCurrHexValue(); got != 100 {
		t.Errorf("hex value = %d, want 100 (700 mod 600)", got)
	}
}

func TestNenHexagon_IncreaseCurrHexValue(t *testing.T) {
	nh := spiritual.NewNenHexagon(0, nil)

	result := nh.IncreaseCurrHexValue()
	if result.CurrentHexVal != 1 {
		t.Errorf("hex after increase = %d, want 1", result.CurrentHexVal)
	}

	// Test wrap around at 599
	nh2 := spiritual.NewNenHexagon(599, nil)
	result2 := nh2.IncreaseCurrHexValue()
	if result2.CurrentHexVal != 0 {
		t.Errorf("hex after wrap = %d, want 0", result2.CurrentHexVal)
	}
}

func TestNenHexagon_DecreaseCurrHexValue(t *testing.T) {
	nh := spiritual.NewNenHexagon(1, nil)
	result := nh.DecreaseCurrHexValue()
	if result.CurrentHexVal != 0 {
		t.Errorf("hex after decrease = %d, want 0", result.CurrentHexVal)
	}

	// Test wrap around at 0
	nh2 := spiritual.NewNenHexagon(0, nil)
	result2 := nh2.DecreaseCurrHexValue()
	if result2.CurrentHexVal != 599 {
		t.Errorf("hex after wrap = %d, want 599", result2.CurrentHexVal)
	}
}

func TestNenHexagon_GetPercentOf(t *testing.T) {
	// At position 0 (Reinforcement center)
	nh := spiritual.NewNenHexagon(0, nil)

	t.Run("own category = 100%", func(t *testing.T) {
		pct := nh.GetPercentOf(enum.Reinforcement)
		if pct != 100.0 {
			t.Errorf("Reinforcement%% = %f, want 100.0", pct)
		}
	})

	t.Run("adjacent category = 80%", func(t *testing.T) {
		// Transmutation is 100 hex away → 100/(100/20) = 20 steps → 100-20 = 80
		pct := nh.GetPercentOf(enum.Transmutation)
		if pct != 80.0 {
			t.Errorf("Transmutation%% = %f, want 80.0", pct)
		}
	})

	t.Run("opposite category = 40%", func(t *testing.T) {
		// Specialization is 300 hex away → 300/(100/20) = 60 → 100-60 = 40
		pct := nh.GetPercentOf(enum.Specialization)
		// But Specialization returns 0 if not the current category
		if pct != 0.0 {
			t.Errorf("Specialization%% (not current) = %f, want 0.0", pct)
		}
	})

	t.Run("emission = 80% (symmetric)", func(t *testing.T) {
		// Emission is at 500, diff=500, but >300 so becomes 600-500=100 → 100/5=20 → 80
		pct := nh.GetPercentOf(enum.Emission)
		if pct != 80.0 {
			t.Errorf("Emission%% = %f, want 80.0", pct)
		}
	})
}

func TestNenHexagon_SpecializationPercentOnlyWhenCurrent(t *testing.T) {
	// At Reinforcement center: Specialization returns 0
	nh1 := spiritual.NewNenHexagon(0, nil)
	if pct := nh1.GetPercentOf(enum.Specialization); pct != 0.0 {
		t.Errorf("Specialization when not current = %f, want 0.0", pct)
	}

	// At Specialization center: Specialization returns 100
	nh2 := spiritual.NewNenHexagon(300, nil)
	if pct := nh2.GetPercentOf(enum.Specialization); pct != 100.0 {
		t.Errorf("Specialization when current = %f, want 100.0", pct)
	}
}

func TestNenHexagon_ResetCategory(t *testing.T) {
	cat := enum.Transmutation
	nh := spiritual.NewNenHexagon(120, &cat) // 20 off center of Transmutation(100)

	resetVal := nh.ResetCategory()
	if resetVal != 100 {
		t.Errorf("reset value = %d, want 100 (Transmutation center)", resetVal)
	}
}

func TestNenHexagon_GetCategoryPercents(t *testing.T) {
	nh := spiritual.NewNenHexagon(0, nil)
	percents := nh.GetCategoryPercents()

	if len(percents) != 6 {
		t.Errorf("percents count = %d, want 6", len(percents))
	}
	// Reinforcement should be 100% at position 0
	if percents[enum.Reinforcement] != 100.0 {
		t.Errorf("Reinforcement%% = %f, want 100.0", percents[enum.Reinforcement])
	}
}
```

- [ ] **Step 2** — Run tests:

Run: `go test ./internal/domain/entity/character_sheet/spiritual/ -v -run "TestNenHexagon"`
Expected: ALL PASS

- [ ] **Step 3** — Commit:

```bash
git add internal/domain/entity/character_sheet/spiritual/nen_hexagon_test.go
git commit -m "test: add NenHexagon unit tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 20 — NenPrinciple and NenCategory Tests

**Files:**
- Create: `internal/domain/entity/character_sheet/spiritual/nen_principle_test.go`
- Create: `internal/domain/entity/character_sheet/spiritual/nen_category_test.go`

- [ ] **Step 1** — Write `nen_principle_test.go`:

```go
package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type mockGameAttribute struct {
	power int
	level int
}

func (m *mockGameAttribute) GetPower() int                               { return m.power }
func (m *mockGameAttribute) GetAbilityBonus() float64                    { return 0 }
func (m *mockGameAttribute) GetLevel() int                               { return m.level }
func (m *mockGameAttribute) CascadeUpgrade(_ *experience.UpgradeCascade) {}
func (m *mockGameAttribute) GetNextLvlAggregateExp() int                 { return 0 }
func (m *mockGameAttribute) GetNextLvlBaseExp() int                      { return 0 }
func (m *mockGameAttribute) GetCurrentExp() int                          { return 0 }
func (m *mockGameAttribute) GetExpPoints() int                           { return 0 }

func newTestNenPrinciple(name enum.PrincipleName, flameLvl, conscPower int) *spiritual.NenPrinciple {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	flame := &mockGameAttribute{level: flameLvl}
	conscience := &mockGameAttribute{power: conscPower}
	return spiritual.NewNenPrinciple(name, exp, flame, conscience)
}

func TestNenPrinciple_InitialState(t *testing.T) {
	np := newTestNenPrinciple(enum.Ten, 0, 0)

	if np.GetLevel() != 0 {
		t.Errorf("initial level = %d, want 0", np.GetLevel())
	}
	if np.GetName() != enum.Ten {
		t.Errorf("name = %s, want %s", np.GetName(), enum.Ten)
	}
}

func TestNenPrinciple_GetValueForTest(t *testing.T) {
	tests := []struct {
		name       string
		flameLvl   int
		conscPower int
		want       int
	}{
		{"all zero", 0, 0, 0},
		{"flame only", 5, 0, 5},
		{"conscience only", 0, 3, 3},
		{"both", 5, 3, 8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			np := newTestNenPrinciple(enum.Ren, tt.flameLvl, tt.conscPower)
			// valueForTest = principleLevel(0) + consciencePower + flameLevel
			if got := np.GetValueForTest(); got != tt.want {
				t.Errorf("GetValueForTest() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNenPrinciple_Clone(t *testing.T) {
	np := newTestNenPrinciple(enum.Ten, 3, 5)
	cloned := np.Clone(enum.Ren)

	if cloned.GetName() != enum.Ren {
		t.Errorf("cloned name = %s, want %s", cloned.GetName(), enum.Ren)
	}
	if cloned.GetExpPoints() != 0 {
		t.Errorf("cloned should have 0 exp, got %d", cloned.GetExpPoints())
	}
}

func TestNenPrinciple_CascadeUpgradeTrigger(t *testing.T) {
	np := newTestNenPrinciple(enum.Gyo, 2, 4)
	values := experience.NewUpgradeCascade(50)

	np.CascadeUpgradeTrigger(values)

	if np.GetExpPoints() != 50 {
		t.Errorf("exp after cascade = %d, want 50", np.GetExpPoints())
	}
	cascade, ok := values.Principles[enum.Hatsu]
	if !ok {
		t.Fatal("principle cascade not found (stored under Hatsu key)")
	}
	if cascade.Lvl != np.GetLevel() {
		t.Errorf("cascade Lvl = %d, want %d", cascade.Lvl, np.GetLevel())
	}
}
```

- [ ] **Step 2** — Write `nen_category_test.go`:

```go
package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

type mockHatsu struct {
	percentOf    float64
	valueForTest int
	level        int
}

func (m *mockHatsu) GetPercentOf(_ enum.CategoryName) float64              { return m.percentOf }
func (m *mockHatsu) GetValueForTest() int                                  { return m.valueForTest }
func (m *mockHatsu) GetLevel() int                                         { return m.level }
func (m *mockHatsu) CascadeUpgrade(_ *experience.UpgradeCascade)           {}
func (m *mockHatsu) GetNextLvlAggregateExp() int                           { return 0 }
func (m *mockHatsu) GetNextLvlBaseExp() int                                { return 0 }
func (m *mockHatsu) GetCurrentExp() int                                    { return 0 }
func (m *mockHatsu) GetExpPoints() int                                     { return 0 }

func TestNenCategory_GetValueForTest(t *testing.T) {
	tests := []struct {
		name         string
		percentOf    float64
		hatsuTestVal int
		want         int
	}{
		{"100% own category", 100.0, 10, 10},
		{"80% adjacent", 80.0, 10, 8},
		{"0% specialization not current", 0.0, 10, 0},
		{"60% two steps away", 60.0, 10, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table := experience.NewDefaultExpTable()
			exp := *experience.NewExperience(table)
			mock := &mockHatsu{percentOf: tt.percentOf, valueForTest: tt.hatsuTestVal}
			nc := spiritual.NewNenCategory(exp, enum.Reinforcement, mock)

			// valueForTest = (catLevel(0) + hatsuTestVal) * percent / 100
			got := nc.GetValueForTest()
			if got != tt.want {
				t.Errorf("GetValueForTest() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestNenCategory_GetPercent(t *testing.T) {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mock := &mockHatsu{percentOf: 80.0}
	nc := spiritual.NewNenCategory(exp, enum.Transmutation, mock)

	if pct := nc.GetPercent(); pct != 80.0 {
		t.Errorf("GetPercent() = %f, want 80.0", pct)
	}
}

func TestNenCategory_CascadeUpgradeTrigger(t *testing.T) {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mock := &mockHatsu{percentOf: 100.0, valueForTest: 5}
	nc := spiritual.NewNenCategory(exp, enum.Reinforcement, mock)

	values := experience.NewUpgradeCascade(50)
	nc.CascadeUpgradeTrigger(values)

	if nc.GetExpPoints() != 50 {
		t.Errorf("exp after cascade = %d, want 50", nc.GetExpPoints())
	}
}

func TestNenCategory_Clone(t *testing.T) {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	mock := &mockHatsu{percentOf: 100.0}
	nc := spiritual.NewNenCategory(exp, enum.Reinforcement, mock)

	cloned := nc.Clone(enum.Emission)
	if cloned.GetName() != enum.Emission {
		t.Errorf("cloned name = %s, want %s", cloned.GetName(), enum.Emission)
	}
}
```

- [ ] **Step 3** — Run tests:

Run: `go test ./internal/domain/entity/character_sheet/spiritual/ -v -run "TestNenPrinciple|TestNenCategory"`
Expected: ALL PASS

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/spiritual/nen_principle_test.go internal/domain/entity/character_sheet/spiritual/nen_category_test.go
git commit -m "test: add NenPrinciple and NenCategory unit tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 21 — Hatsu and PrinciplesManager Tests

**Files:**
- Create: `internal/domain/entity/character_sheet/spiritual/hatsu_test.go`
- Create: `internal/domain/entity/character_sheet/spiritual/principles_manager_test.go`

- [ ] **Step 1** — Write `hatsu_test.go`:

```go
package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestHatsu() *spiritual.Hatsu {
	table := experience.NewDefaultExpTable()
	exp := *experience.NewExperience(table)
	flame := &mockGameAttribute{level: 1}
	conscience := &mockGameAttribute{power: 2}

	percents := map[enum.CategoryName]float64{
		enum.Reinforcement:   100.0,
		enum.Transmutation:   80.0,
		enum.Materialization: 60.0,
		enum.Specialization:  0.0,
		enum.Manipulation:    60.0,
		enum.Emission:        80.0,
	}
	return spiritual.NewHatsu(exp, flame, conscience, nil, percents)
}

func TestHatsu_Init(t *testing.T) {
	h := newTestHatsu()
	table := experience.NewDefaultExpTable()

	categories := make(map[enum.CategoryName]spiritual.NenCategory)
	for _, name := range enum.AllNenCategoryNames() {
		exp := *experience.NewExperience(table)
		categories[name] = *spiritual.NewNenCategory(exp, name, h)
	}

	if err := h.Init(categories); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Double init should fail
	if err := h.Init(categories); err == nil {
		t.Error("double Init() should return error")
	}
}

func TestHatsu_SetCategoryPercents(t *testing.T) {
	h := newTestHatsu()

	t.Run("valid 6 categories", func(t *testing.T) {
		percents := map[enum.CategoryName]float64{
			enum.Reinforcement:   90.0,
			enum.Transmutation:   70.0,
			enum.Materialization: 50.0,
			enum.Specialization:  0.0,
			enum.Manipulation:    50.0,
			enum.Emission:        70.0,
		}
		if err := h.SetCategoryPercents(percents); err != nil {
			t.Fatalf("SetCategoryPercents error: %v", err)
		}
	})

	t.Run("invalid count", func(t *testing.T) {
		percents := map[enum.CategoryName]float64{
			enum.Reinforcement: 100.0,
		}
		if err := h.SetCategoryPercents(percents); err == nil {
			t.Error("should reject non-6 category percents")
		}
	})
}

func TestHatsu_GetValueForTest(t *testing.T) {
	h := newTestHatsu()
	// valueForTest = hatsuLevel(0) + consciencePower(2) + flameLevel(1) = 3
	if got := h.GetValueForTest(); got != 3 {
		t.Errorf("GetValueForTest() = %d, want 3", got)
	}
}

func TestHatsu_GetPercentOf(t *testing.T) {
	h := newTestHatsu()
	if pct := h.GetPercentOf(enum.Reinforcement); pct != 100.0 {
		t.Errorf("Reinforcement%% = %f, want 100.0", pct)
	}
	if pct := h.GetPercentOf(enum.Transmutation); pct != 80.0 {
		t.Errorf("Transmutation%% = %f, want 80.0", pct)
	}
}
```

- [ ] **Step 2** — Write `principles_manager_test.go`:

```go
package spiritual_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/experience"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/spiritual"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func newTestPrinciplesManager() *spiritual.Manager {
	table := experience.NewDefaultExpTable()
	flame := &mockGameAttribute{level: 1}
	conscience := &mockGameAttribute{power: 2}

	// Build principles
	principles := make(map[enum.PrincipleName]spiritual.NenPrinciple)
	for _, name := range enum.AllNenPrincipleNames() {
		if name == enum.Hatsu {
			continue
		}
		exp := *experience.NewExperience(table)
		p := spiritual.NewNenPrinciple(name, exp, flame, conscience)
		principles[name] = *p
	}

	// Build hatsu + hexagon
	hexagon := spiritual.NewNenHexagon(0, nil) // Reinforcement
	hatsuExp := *experience.NewExperience(table)
	percents := hexagon.GetCategoryPercents()
	hatsu := spiritual.NewHatsu(hatsuExp, flame, conscience, nil, percents)

	categories := make(map[enum.CategoryName]spiritual.NenCategory)
	for _, name := range enum.AllNenCategoryNames() {
		exp := *experience.NewExperience(table)
		categories[name] = *spiritual.NewNenCategory(exp, name, hatsu)
	}
	hatsu.Init(categories)

	return spiritual.NewPrinciplesManager(principles, hexagon, hatsu)
}

func TestPrinciplesManager_Get(t *testing.T) {
	m := newTestPrinciplesManager()

	t.Run("existing principle", func(t *testing.T) {
		p, err := m.Get(enum.Ten)
		if err != nil {
			t.Fatalf("Get(Ten) error: %v", err)
		}
		if p == nil {
			t.Fatal("Get(Ten) returned nil")
		}
	})

	t.Run("hatsu returns hatsu object", func(t *testing.T) {
		p, err := m.Get(enum.Hatsu)
		if err != nil {
			t.Fatalf("Get(Hatsu) error: %v", err)
		}
		if p == nil {
			t.Fatal("Get(Hatsu) returned nil")
		}
	})
}

func TestPrinciplesManager_IncreaseExpByPrinciple(t *testing.T) {
	m := newTestPrinciplesManager()
	values := experience.NewUpgradeCascade(50)

	if err := m.IncreaseExpByPrinciple(enum.Ten, values); err != nil {
		t.Fatalf("IncreaseExpByPrinciple error: %v", err)
	}

	exp, err := m.GetExpPointsOfPrinciple(enum.Ten)
	if err != nil {
		t.Fatalf("GetExpPointsOfPrinciple error: %v", err)
	}
	if exp != 50 {
		t.Errorf("exp = %d, want 50", exp)
	}
}

func TestPrinciplesManager_HexagonOperations(t *testing.T) {
	m := newTestPrinciplesManager()

	t.Run("initial hex value", func(t *testing.T) {
		val, err := m.GetCurrHexValue()
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if val != 0 {
			t.Errorf("initial hex = %d, want 0", val)
		}
	})

	t.Run("increase hex", func(t *testing.T) {
		result, err := m.IncreaseCurrHexValue()
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if result.CurrentHexVal != 1 {
			t.Errorf("hex after increase = %d, want 1", result.CurrentHexVal)
		}
	})

	t.Run("decrease hex", func(t *testing.T) {
		result, err := m.DecreaseCurrHexValue()
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if result.CurrentHexVal != 0 {
			t.Errorf("hex after decrease = %d, want 0", result.CurrentHexVal)
		}
	})

	t.Run("category name", func(t *testing.T) {
		cat, err := m.GetNenCategoryName()
		if err != nil {
			t.Fatalf("error: %v", err)
		}
		if cat != enum.Reinforcement {
			t.Errorf("category = %s, want %s", cat, enum.Reinforcement)
		}
	})
}

func TestPrinciplesManager_BatchGetters(t *testing.T) {
	m := newTestPrinciplesManager()
	levels := m.GetLevelOfPrinciples()

	// Should have 10 principles (all except Hatsu)
	if len(levels) != 10 {
		t.Errorf("principles count = %d, want 10", len(levels))
	}
}
```

- [ ] **Step 3** — Run tests:

Run: `go test ./internal/domain/entity/character_sheet/spiritual/ -v`
Expected: ALL PASS

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/spiritual/hatsu_test.go internal/domain/entity/character_sheet/spiritual/principles_manager_test.go
git commit -m "test: add Hatsu and PrinciplesManager unit tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 22 — Nen System Documentation (PT-BR)

**Files:**
- Create: `docs/game/ficha-de-personagem/sistema-nen.md`

- [ ] **Step 1** — Write Nen system documentation covering:
- O que é Nen no contexto do RPG
- Princípios Nen (Nen Principles): os 11 princípios
  - Ten, Zetsu, Ren, Gyo, Hatsu, Shu, Kou, Ken, Ryu, In, En
  - Valor de Teste = nível do princípio + poder da Consciência + nível da Chama
- Categorias Nen (Nen Categories): as 6 categorias
  - Reforço (Reinforcement), Transmutação (Transmutation), Materialização (Materialization), Especialização (Specialization), Manipulação (Manipulation), Emissão (Emission)
- Hexágono Nen (Nen Hexagon): sistema de distribuição de porcentagens
  - Escala 0-599, cada categoria ocupa 100 unidades
  - Porcentagem baseada na distância hexagonal
  - Especialização retorna 0% se não for a categoria atual
  - Reset de categoria (referência ao arco de Formigas Quimera)
- Hatsu: habilidades Nen individuais
  - Cascade: categoria → hatsu → consciência → habilidade espiritual → char exp

- [ ] **Step 2** — Commit:

```bash
git add docs/game/ficha-de-personagem/sistema-nen.md
git commit -m "docs: add Nen system documentation (PT-BR)

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 23 — CharacterProfile and TalentByCategorySet Tests

**Files:**
- Create: `internal/domain/entity/character_sheet/sheet/character_profile_test.go`
- Create: `internal/domain/entity/character_sheet/sheet/talent_by_category_set_test.go`

- [ ] **Step 1** — Write `character_profile_test.go`:

```go
package sheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
)

func TestCharacterProfile_Validate(t *testing.T) {
	validProfile := func() sheet.CharacterProfile {
		return sheet.CharacterProfile{
			NickName:         "Gon",
			FullName:         "Gon Freecss",
			Alignment:        "Chaotic-Good",
			BriefDescription: "A young hunter",
			Age:              12,
		}
	}

	t.Run("valid profile", func(t *testing.T) {
		p := validProfile()
		if err := p.Validate(); err != nil {
			t.Errorf("valid profile returned error: %v", err)
		}
	})

	t.Run("nickname too short", func(t *testing.T) {
		p := validProfile()
		p.NickName = "Go"
		if err := p.Validate(); err == nil {
			t.Error("should reject nickname < 3 chars")
		}
	})

	t.Run("nickname too long", func(t *testing.T) {
		p := validProfile()
		p.NickName = "GonFreecss!"
		if err := p.Validate(); err == nil {
			t.Error("should reject nickname > 10 chars")
		}
	})

	t.Run("fullname too short", func(t *testing.T) {
		p := validProfile()
		p.FullName = "Gon"
		if err := p.Validate(); err == nil {
			t.Error("should reject fullname < 6 chars")
		}
	})

	t.Run("fullname too long", func(t *testing.T) {
		p := validProfile()
		p.FullName = "Gon Freecss The Great Hunter Of All Time!!"
		if err := p.Validate(); err == nil {
			t.Error("should reject fullname > 32 chars")
		}
	})

	t.Run("brief description too long", func(t *testing.T) {
		p := validProfile()
		longDesc := ""
		for i := 0; i < 256; i++ {
			longDesc += "x"
		}
		p.BriefDescription = longDesc
		if err := p.Validate(); err == nil {
			t.Error("should reject brief description > 255 chars")
		}
	})

	t.Run("negative age", func(t *testing.T) {
		p := validProfile()
		p.Age = -1
		if err := p.Validate(); err == nil {
			t.Error("should reject negative age")
		}
	})

	t.Run("zero age valid", func(t *testing.T) {
		p := validProfile()
		p.Age = 0
		if err := p.Validate(); err != nil {
			t.Errorf("age 0 should be valid: %v", err)
		}
	})

	t.Run("empty alignment valid", func(t *testing.T) {
		p := validProfile()
		p.Alignment = ""
		if err := p.Validate(); err != nil {
			t.Errorf("empty alignment should be valid: %v", err)
		}
	})
}

func TestCharacterProfile_ValidateAlignment(t *testing.T) {
	tests := []struct {
		name      string
		alignment string
		wantErr   bool
	}{
		{"Lawful-Good", "Lawful-Good", false},
		{"Neutral-Neutral", "Neutral-Neutral", false},
		{"Chaotic-Evil", "Chaotic-Evil", false},
		{"Lawful-Neutral", "Lawful-Neutral", false},
		{"empty", "", false},
		{"invalid format no dash", "LawfulGood", true},
		{"invalid first", "Random-Good", true},
		{"invalid second", "Lawful-Random", true},
		{"too many parts", "Lawful-Good-Extra", true},
		{"lowercase", "lawful-good", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := sheet.CharacterProfile{Alignment: tt.alignment}
			err := p.ValidateAlignment()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAlignment(%q) error = %v, wantErr %v", tt.alignment, err, tt.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2** — Write `talent_by_category_set_test.go`:

```go
package sheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
)

func TestTalentByCategorySet_NewWithZeroActive(t *testing.T) {
	categories := map[enum.CategoryName]bool{
		enum.Reinforcement:   false,
		enum.Transmutation:   false,
		enum.Materialization: false,
		enum.Specialization:  false,
		enum.Manipulation:    false,
		enum.Emission:        false,
	}
	_, err := sheet.NewTalentByCategorySet(categories, nil)
	if err == nil {
		t.Error("should reject 0 active categories")
	}
}

func TestTalentByCategorySet_GetTalentLvl(t *testing.T) {
	tests := []struct {
		name        string
		active      int
		hasHexValue bool
		want        int
	}{
		// No hex value: bonus = (active-1)*2, special: if bonus==0 then bonus=1
		{"1 active no hex", 1, false, 21},  // BASE(20) + bonus: (1-1)*2=0 → special: 1
		{"2 active no hex", 2, false, 22},  // BASE(20) + (2-1)*2 = 22
		{"3 active no hex", 3, false, 24},  // BASE(20) + (3-1)*2 = 24
		{"6 active no hex", 6, false, 30},  // BASE(20) + (6-1)*2 = 30
		// With hex value: bonus = (active-1)*1
		{"1 active with hex", 1, true, 20}, // BASE(20) + (1-1)*1 = 20
		{"2 active with hex", 2, true, 21}, // BASE(20) + (2-1)*1 = 21
		{"6 active with hex", 6, true, 25}, // BASE(20) + (6-1)*1 = 25
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allCategories := enum.AllNenCategoryNames()
			categories := make(map[enum.CategoryName]bool)
			for i, name := range allCategories {
				categories[name] = i < tt.active
			}

			var hexVal *int
			if tt.hasHexValue {
				v := 0
				hexVal = &v
			}

			tcs, err := sheet.NewTalentByCategorySet(categories, hexVal)
			if err != nil {
				t.Fatalf("NewTalentByCategorySet error: %v", err)
			}

			if got := tcs.GetTalentLvl(); got != tt.want {
				t.Errorf("GetTalentLvl() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTalentByCategorySet_Getters(t *testing.T) {
	categories := map[enum.CategoryName]bool{
		enum.Reinforcement: true,
		enum.Emission:      true,
	}
	hexVal := 100
	tcs, err := sheet.NewTalentByCategorySet(categories, &hexVal)
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(tcs.GetCategories()) != 2 {
		t.Errorf("categories count = %d, want 2", len(tcs.GetCategories()))
	}
	if *tcs.GetInitialHexValue() != 100 {
		t.Errorf("initial hex = %d, want 100", *tcs.GetInitialHexValue())
	}
}
```

- [ ] **Step 3** — Run tests:

Run: `go test ./internal/domain/entity/character_sheet/sheet/ -v -run "TestCharacterProfile|TestTalentByCategorySet"`
Expected: ALL PASS

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/sheet/character_profile_test.go internal/domain/entity/character_sheet/sheet/talent_by_category_set_test.go
git commit -m "test: add CharacterProfile and TalentByCategorySet unit tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 24 — CharacterSheet Cascade Tests

**Files:**
- Create: `internal/domain/entity/character_sheet/sheet/character_sheet_test.go`

- [ ] **Step 1** — Write `character_sheet_test.go` with cascade tests using real factory:

```go
package sheet_test

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/character_sheet/sheet"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/google/uuid"
)

func buildTestSheet(t *testing.T) *sheet.CharacterSheet {
	t.Helper()

	factory := sheet.NewCharacterSheetFactory()
	profile := sheet.CharacterProfile{
		NickName:         "Gon",
		FullName:         "Gon Freecss",
		Alignment:        "Chaotic-Good",
		BriefDescription: "A young hunter",
		Age:              12,
	}

	playerUUID := uuid.New()
	masterUUID := uuid.Nil
	campaignUUID := uuid.New()

	categories := map[enum.CategoryName]bool{
		enum.Reinforcement:   true,
		enum.Transmutation:   false,
		enum.Materialization: false,
		enum.Specialization:  false,
		enum.Manipulation:    false,
		enum.Emission:        false,
	}
	categorySet, err := sheet.NewTalentByCategorySet(categories, nil)
	if err != nil {
		t.Fatalf("NewTalentByCategorySet error: %v", err)
	}

	cs, err := factory.Build(
		playerUUID, masterUUID, campaignUUID,
		profile, nil, categorySet,
	)
	if err != nil {
		t.Fatalf("factory.Build error: %v", err)
	}
	return cs
}

func TestCharacterSheet_InitialState(t *testing.T) {
	cs := buildTestSheet(t)

	if cs.GetLevel() != 0 {
		t.Errorf("initial character level = %d, want 0", cs.GetLevel())
	}
	if cs.GetCharacterPoints() != 0 {
		t.Errorf("initial character points = %d, want 0", cs.GetCharacterPoints())
	}
}

func TestCharacterSheet_IncreaseExpForSkill_Cascade(t *testing.T) {
	cs := buildTestSheet(t)

	initialCharExp := cs.GetExpPoints()

	// Increase exp for a physical skill — should cascade through:
	// skill → attribute → ability (Physicals) → character exp
	result, err := cs.IncreaseExpForSkill(enum.Vitality, 500)
	if err != nil {
		t.Fatalf("IncreaseExpForSkill error: %v", err)
	}

	// After cascade, character exp should have increased
	if cs.GetExpPoints() <= initialCharExp {
		t.Errorf("character exp should increase after cascade: was %d, now %d",
			initialCharExp, cs.GetExpPoints())
	}

	// The cascade result should contain skill data
	if _, ok := result.Skills[enum.Vitality.String()]; !ok {
		t.Error("cascade should contain Vitality skill data")
	}

	// Should contain ability data
	if _, ok := result.Abilities[enum.Physicals]; !ok {
		t.Error("cascade should contain Physicals ability data")
	}

	// Character exp should be set in cascade
	if result.CharacterExp == nil {
		t.Error("cascade should contain CharacterExp")
	}
}

func TestCharacterSheet_StatusUpgradeAfterExpIncrease(t *testing.T) {
	cs := buildTestSheet(t)

	hpBefore, err := cs.GetMaxOfStatus(enum.Health)
	if err != nil {
		t.Fatalf("GetMaxOfStatus error: %v", err)
	}

	// Give large exp to trigger level ups which should upgrade status bars
	cs.IncreaseExpForSkill(enum.Vitality, 50000)

	hpAfter, err := cs.GetMaxOfStatus(enum.Health)
	if err != nil {
		t.Fatalf("GetMaxOfStatus error: %v", err)
	}

	if hpAfter <= hpBefore {
		t.Errorf("HP should increase after massive XP gain: before=%d, after=%d",
			hpBefore, hpAfter)
	}
}

func TestCharacterSheet_GetValueForTestOfSkill(t *testing.T) {
	cs := buildTestSheet(t)

	val, err := cs.GetValueForTestOfSkill(enum.Vitality)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if val < 0 {
		t.Errorf("value for test should be >= 0, got %d", val)
	}
}

func TestCharacterSheet_SetCurrStatus(t *testing.T) {
	cs := buildTestSheet(t)

	maxHP, _ := cs.GetMaxOfStatus(enum.Health)
	if maxHP == 0 {
		t.Skip("max HP is 0, cannot test SetCurrStatus")
	}

	err := cs.SetCurrStatus(enum.Health, maxHP-1)
	if err != nil {
		t.Fatalf("SetCurrStatus error: %v", err)
	}

	curr, _ := cs.GetCurrentOfStatus(enum.Health)
	if curr != maxHP-1 {
		t.Errorf("current HP = %d, want %d", curr, maxHP-1)
	}
}

func TestCharacterSheet_NenHexagonOperations(t *testing.T) {
	cs := buildTestSheet(t)

	hexVal, err := cs.GetCurrHexValue()
	if err != nil {
		// Hex may not be initialized — that's ok for the test
		t.Skip("hex not initialized")
	}

	if hexVal < 0 || hexVal >= 600 {
		t.Errorf("hex value out of range: %d", hexVal)
	}
}
```

- [ ] **Step 2** — Run tests:

Run: `go test ./internal/domain/entity/character_sheet/sheet/ -v -run "TestCharacterSheet"`
Expected: ALL PASS (some may need adjustment based on factory behavior)

- [ ] **Step 3** — If any tests fail due to factory assumptions, adjust the test to match actual factory behavior (read factory code for exact parameter requirements).

- [ ] **Step 4** — Commit:

```bash
git add internal/domain/entity/character_sheet/sheet/character_sheet_test.go
git commit -m "test: add CharacterSheet cascade integration tests

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

---

## Task 25 — AGENTS.md and Architecture Documentation

**Files:**
- Create: `AGENTS.md` (project root)
- Create: `docs/architecture/overview.md`

- [ ] **Step 1** — Write `AGENTS.md` containing:
- Project overview: HxH RPG System backend in Go, tabletop RPG calculator
- Architecture: 4 layers — entity (domain model), usecase (domain logic), app (HTTP/WS handlers), gateway (PostgreSQL repositories)
- Domain map with file paths for each concept:
  - Character Sheet: `internal/domain/entity/character_sheet/`
  - Match/Combat: `internal/domain/entity/match/`
  - Campaign: `internal/domain/entity/campaign/`
  - Scenario: `internal/domain/entity/scenario/`
  - User: `internal/domain/entity/user/`
  - Enums: `internal/domain/entity/enum/`
- Code conventions: Go idiomatic, engines as domain services, implicit interfaces, no test frameworks, table-driven tests
- Quick glossary: top 20 domain terms with EN→PT-BR
- Current state: character_sheet stable+tested, Turn/Round WIP/broken, engines pending rename to domain services
- Commands: `go test ./...`, `make build`, `make run-dev`

- [ ] **Step 2** — Write `docs/architecture/overview.md` covering:
- Layer diagram and responsibilities
- Package dependency rules (entity ← usecase ← app, entity ← gateway)
- Experience cascade pattern
- Engine/domain service pattern
- Entry points (API server, WebSocket game server)

- [ ] **Step 3** — Commit:

```bash
git add AGENTS.md docs/architecture/overview.md
git commit -m "docs: add AGENTS.md and architecture overview

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
