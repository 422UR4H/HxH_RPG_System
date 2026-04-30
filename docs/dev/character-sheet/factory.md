# CharacterSheet Factory & Assembly

> Documentação técnica do factory de fichas de personagem: ordem de construção,
> tabela de coeficientes, wiring de atributos, aplicação de classes e HalfSheet.
> Pacote: `internal/domain/entity/character_sheet/sheet/`

## Visão Geral

A `CharacterSheetFactory` é responsável por construir e conectar **todos** os
componentes de uma ficha de personagem. A ordem de construção é crítica porque
cada camada depende de referências das camadas anteriores — abilities precisam
do `CharacterExp`, atributos precisam de abilities, skills precisam de atributos,
status precisa de tudo.

O factory é stateless (`CharacterSheetFactory{}`) e expõe dois métodos públicos:
`Build()` (ficha completa) e `BuildHalfSheet()` (ficha sem Nen).

---

## 1. Ordem de Montagem & Grafo de Dependências

### Fluxo de Build()

A construção segue uma ordem linear estrita:

```
1.  BuildCharacterExp()
        ↓ characterExp
2.  BuildPersonAbilities(characterExp)
        ↓ abilities (Physicals, Mentals, Spirituals, Skills, Talent)
3.  BuildPhysAttrs(physAbility)
        ↓ physAttrs (4 primários + 4 intermediários)
4.  BuildMentalAttrs(mentalAbility)
        ↓ mentalAttrs (4 primários, sem intermediários — TODO)
5.  BuildSpiritualAttrs(spiritAbility)
        ↓ spiritAttrs (Flame, Conscience)
6.  NewCharacterAttributes(physAttrs, mentalAttrs, spiritAttrs)
        ↓ charAttrs
7.  BuildPhysSkills(skills, physAttrs)
        ↓ physSkills (21 perícias)
8.  BuildMentalSkills(skills, mentalAttrs)
        ↓ mentalSkills (Manager vazio — TODO)
9.  BuildSpiritualSkills(skills, spiritAttrs)
        ↓ spiritSkills (6 perícias)
10. NewCharacterSkills(physSkills, mentalSkills, spiritSkills)
        ↓ charSkills
11. BuildHatsu(flame, conscience, categoryPercents)
        ↓ hatsu (com NenCategories)
12. BuildSpiritPrinciples(flame, conscience, nenHexagon, hatsu)
        ↓ spiritPrinciples (PrinciplesManager)
13. BuildStatusManager(abilities, charAttrs, charSkills)
        ↓ status (HP, SP, AP)
14. NewCharacterSheet(...)
        ↓ charSheet (validação de owner)
15. [opcional] Wrap(charSheet, charClass)
        ↓ charSheet (com classe aplicada)
```

### Grafo de dependências

```
CharacterExp ──────────────────────────────────────────┐
    │                                                  │
    ▼                                                  │
Abilities (Physicals, Mentals, Spirituals, Skills)     │
    │          │           │           │                │
    ▼          ▼           ▼           │                │
PhysAttrs  MentalAttrs  SpiritAttrs   │                │
    │          │           │           │                │
    │          │           ├── Flame   │                │
    │          │           └── Conscience               │
    │          │                │                       │
    ▼          ▼                │                       │
PhysSkills MentalSkills    SpiritSkills                │
    │                          │                       │
    │                          ▼                       │
    │                    Hatsu ← NenHexagon             │
    │                      │                           │
    │                      ▼                           │
    │               SpiritPrinciples                   │
    │                                                  │
    ▼                                                  │
StatusManager (HP ← PhysAbility+Res+Vit)              │
              (SP ← PhysAbility+Res+Energy)            │
              (AP ← SpiritAbility+Conscience+MOP)      │
                                                       │
CharacterSheet ← tudo acima ──────────────────────────┘
```

### Por que a ordem importa

- **Abilities antes de atributos** — cada atributo recebe uma referência
  `ability.IAbility` para acessar `GetBonus()` e `GetExpReference()`. Sem a
  ability construída, o atributo não tem como participar da cascata.
- **Atributos antes de skills** — cada `CommonSkill` recebe uma referência
  `IDistributableAttribute` (ou `IGameAttribute` para espirituais) para calcular
  `GetValueForTest()` e participar da cascata.
- **Skills antes de status** — as barras de status precisam de referências a
  skills específicas (Vitality, Energy, MOP) para suas fórmulas.
- **Tudo antes de `NewCharacterSheet`** — o construtor recebe tudo por **valor**
  (structs copiados), não por ponteiro. As referências internas (ability →
  charExp, attr → ability, etc.) já estão conectadas via ponteiros internos.

---

## 2. Tabela de Coeficientes

Os 13 coeficientes definem a velocidade de progressão de cada componente via
`ExpTable`. Todos são constantes no factory.

> `TODO: strike the gavel about these changes` — alguns valores foram alterados
> recentemente e ainda não foram formalmente aprovados. Os valores comentados
> ao lado mostram os anteriores.

### Agrupamento por domínio

**Abilities:**

| Constante | Valor | Anterior | ExpTable alimenta |
|-----------|-------|----------|-------------------|
| `CHARACTER_COEFF` | `10.0` | — | CharacterExp (ponto final da cascata) |
| `TALENT_COEFF` | `2.0` | — | Talent (indicador de potencial Nen) |
| `PHYSICAL_COEFF` | `20.0` | — | Ability Physicals |
| `MENTAL_COEFF` | `20.0` | `15.0` | Ability Mentals |
| `SPIRITUAL_COEFF` | `5.0` | — | Ability Spirituals |
| `SKILLS_COEFF` | `20.0` | `5.0` | Ability Skills |

**Atributos:**

| Constante | Valor | Anterior | ExpTable alimenta |
|-----------|-------|----------|-------------------|
| `PHYSICAL_ATTRIBUTE_COEFF` | `5.0` | — | Atributos físicos (Res, Agi, Flx, Sen + intermediários) |
| `MENTAL_ATTRIBUTE_COEFF` | `1.0` | `3.0` | Atributos mentais (Resilience, Adaptability, Weighting, Creativity) |
| `SPIRITUAL_ATTRIBUTE_COEFF` | `1.0` | — | Atributos espirituais (Flame, Conscience) |

**Perícias:**

| Constante | Valor | ExpTable alimenta |
|-----------|-------|-------------------|
| `PHYSICAL_SKILLS_COEFF` | `1.0` | 21 perícias físicas |
| `MENTAL_SKILLS_COEFF` | `2.0` | Perícias mentais (Manager vazio atualmente) |
| `SPIRITUAL_SKILLS_COEFF` | `3.0` | 6 perícias espirituais (Focus, WillPower, etc.) |
| `SPIRITUAL_PRINCIPLE_COEFF` | `1.0` | Princípios Nen (Ten, Zetsu, Ren, etc.) + Hatsu + NenCategories |

### Implicações de balanceamento

Quanto **maior** o coeficiente, **mais lento** o componente sobe de nível (mais
XP por nível). A tabela revela a intenção de design:

- **Perícias físicas** (`1.0`) sobem muito rápido — feedback imediato ao jogador
- **Abilities** (`20.0`) sobem muito devagar — requerem treino extensivo
- **Espirituais** (`5.0` ability, `3.0` skills) — meio-termo, refletindo a
  dificuldade intermediária do domínio de Nen

Para a curva sigmoidal e como os coeficientes escalam a tabela, veja
[`experience.md`](experience.md) §1.

---

## 3. Wiring de Atributos Físicos — Cadeia Circular

**Método:** `BuildPhysAttrs(physAbility)`

Os atributos físicos formam uma cadeia circular de 4 primários e 4
intermediários. Cada intermediário é derivado de **dois primários adjacentes**,
e o último intermediário conecta o último primário de volta ao primeiro:

```
Resistance ──→ Agility ──→ Flexibility ──→ Sense ──→ (volta a Resistance)
    ↘               ↘              ↘              ↘
   Strength       Celerity      Dexterity    Constitution
  (Res+Agi)      (Agi+Flx)     (Flx+Sen)     (Sen+Res)
```

### Construção detalhada

1. Cria `Resistance` como `PrimaryAttribute` base (com `NewPrimaryAttribute`)
2. **Clona** `Agility` a partir de `Resistance` via `Clone()` — compartilha
   a mesma `ExpTable` e ability, mas recebe buff próprio
3. Cria `Strength` como `MiddleAttribute(Resistance, Agility)`
4. **Clona** `Flexibility` e cria `Celerity(Agility, Flexibility)`
5. **Clona** `Sense` e cria `Dexterity(Flexibility, Sense)`
6. Cria `Constitution(Sense, Resistance)` — **fecha o ciclo**

### Detalhes não óbvios

- **Clone reutiliza a ExpTable** — todos os atributos físicos primários
  compartilham a mesma curva de progressão (coeficiente `5.0`). O `Clone()` cria
  uma nova `Exp` com a mesma tabela.
- **Buffs são independentes** — cada atributo (primário e intermediário) recebe
  seu próprio `*int` buff do mapa criado por `BuildPhysAttrBuffs()`.
- **MiddleAttribute.GetPoints()** calcula a média dos `points` dos primários
  adjacentes — veja [`abilities-attributes.md`](abilities-attributes.md) §3.
- **Cadeia circular não cria dependência circular de XP** — cada
  `MiddleAttribute` divide XP entre seus primários via `CascadeUpgrade`, mas
  os primários delegam à ability (não de volta ao intermediário). O ciclo
  topológico é no *wiring*, não no *fluxo de dados*.

### Atributos mentais — sem intermediários (por enquanto)

`BuildMentalAttrs` cria apenas 4 primários (Resilience, Adaptability, Weighting,
Creativity) sem intermediários:

```go
// TODO: add middle attributes which primary attributes above
return attribute.NewAttributeManager(attrs, nil, buffs)
```

O `nil` para `middleAttributes` indica que o sistema está preparado para
intermediários mentais, mas ainda não os possui.

### Atributos espirituais — simplificados

`BuildSpiritualAttrs` cria apenas 2 `SpiritualAttribute` (Flame, Conscience)
via `SpiritualManager`. Sem pontos distribuíveis, sem intermediários.

---

## 4. CharacterClass — Wrapping

**Método:** `Wrap(charSheet, charClass)`

Após a construção base, `Wrap()` aplica uma `CharacterClass` — adicionando XP
inicial a perícias e atributos, e registrando proficiências e joint skills
da classe.

### Ordem de aplicação

A ordem é significativa porque cada etapa pode disparar cascatas de XP:

```
1. SkillsExps         → IncreaseExpForSkill por perícia
2. JointSkills         → AddJointSkill (registro, sem XP)
3. JointProficiencies  → AddJointProficiency (registro, sem XP)
4. AttributesExps      → IncreaseExpForMentals por atributo
5. ProficienciesExps   → NewProficiency + AddCommonProficiency + IncreaseExpForProficiency
6. JointProfExps       → IncreaseExpForJointProficiency
```

### Detalhes de cada etapa

1. **Skills XP** — itera `charClass.SkillsExps` (mapa `SkillName → int`) e
   insere XP via `IncreaseExpForSkill`. Cada inserção dispara a cascata
   completa (skill → attribute → ability → charExp) e `status.Upgrade()`.

2. **Joint Skills** — adiciona `JointSkill` à ficha. Requer inicialização com
   a referência `ICascadeUpgrade` da ability Physicals (via `skill.Init()`).
   Apenas registra — não insere XP.

3. **Joint Proficiencies** — mesmo padrão: registro sem XP. Requer referências
   tanto de Physicals quanto de Skills ability.

4. **Attributes XP** — insere XP em atributos mentais via
   `IncreaseExpForMentals`. Note que apenas atributos mentais recebem XP da
   classe — atributos físicos evoluem pela cascata das perícias.

5. **Proficiencies** — cria proficiências comuns (`NewProficiency` com
   coeficiente `PHYSICAL_SKILLS_COEFF = 1.0`), registra na ficha, e insere XP.

6. **Joint Proficiency XP** — insere XP em proficiências conjuntas já
   existentes.

### WrapHalf

`WrapHalf()` aplica exatamente a mesma lógica ao `HalfSheet`. O código é
duplicado (não reutilizado), pois `HalfSheet` e `CharacterSheet` são tipos
distintos sem interface comum que cubra todos os métodos necessários.

---

## 5. HalfSheet — Ficha Sem Nen

**Método:** `BuildHalfSheet(profile, charClass)`

`HalfSheet` é uma ficha simplificada para personagens que **não possuem**
sistema espiritual (Nen). Usada para NPCs comuns, personagens pré-despertar,
ou qualquer entidade que precise da mecânica física/mental sem Nen.

### Diferenças em relação a CharacterSheet

| Aspecto | CharacterSheet | HalfSheet |
|---------|----------------|-----------|
| UUIDs (player, master, campaign) | Obrigatórios (com validação XOR) | Ausentes |
| Ability Spirituals | ✅ Presente | ❌ Ausente |
| Atributos espirituais | ✅ Flame, Conscience | ❌ `nil` |
| Perícias espirituais | ✅ 6 perícias | ❌ `nil` |
| NenHexagon / Hatsu / Princípios | ✅ Completo | ❌ Ausente |
| Aura Points (AP) | ✅ Condicional | ❌ Nunca criado |
| Validação de owner | `playerUUID XOR masterUUID` | Nenhuma |

### Fluxo de construção

```
BuildHalfSheet()
├── BuildCharacterExp()
├── BuildPersonAbilitiesHalf(charExp)     ← SEM Spirituals ability
│   └── Physicals, Mentals, Skills + Talent (3 abilities, não 4)
├── BuildPhysAttrs(physAbility)
├── BuildMentalAttrs(mentalAbility)
├── NewCharacterAttributes(phys, mental, nil)   ← nil spiritual
├── BuildPhysSkills(skills, physAttrs)
├── BuildMentalSkills(skills, mentalAttrs)
├── NewCharacterSkills(phys, mental, nil)        ← nil spiritual
├── BuildStatusManager(abilities, attrs, skills)
│   └── HP + SP apenas (spiritualAbility == nil → AP não criado)
├── NewHalfSheet(...)
└── [opcional] WrapHalf(sheet, charClass)
```

### Detalhes não óbvios

- **`BuildPersonAbilitiesHalf`** cria apenas 3 abilities (Physicals, Mentals,
  Skills) — sem `enum.Spirituals`. O `ability.Manager` armazena apenas essas 3.
  `abilities.Get(enum.Spirituals)` retornará `nil`.

- **`BuildStatusManager` trata nil** — dentro do factory, o código verifica
  `spiritualAbility != nil` antes de criar AP:

  ```go
  spiritualAbility, _ := abilities.Get(enum.Spirituals)
  if spiritualAbility != nil {
      // ... cria AP
  }
  ```

  Para `HalfSheet`, `spiritualAbility` será `nil` e AP não será adicionado ao
  mapa do `Manager`. O Manager funcionará normalmente com apenas HP e SP.

- **TODOs preservados:**
  - `TODO: fix after add aura (MOP - spiritual status)` — refere-se ao fato
    de que `BuildStatusManager` é compartilhado entre `Build` e `BuildHalfSheet`,
    e o tratamento de AP nil pode precisar de revisão.
  - `TODO: like Build func above, move to client that calls this func` — código
    de `TalentByCategorySet` comentado que deveria ser movido para o chamador.

### Quando usar HalfSheet

- **NPCs comuns** — inimigos sem Nen que participam de combate com atributos
  físicos e mentais
- **Personagens pré-despertar** — jogadores que ainda não desbloquearam Nen
- **Testes simplificados** — cenários de teste que não precisam do subsistema
  espiritual completo

---

## Wiring de Perícias — Referência

### Perícias Físicas (21 total, 8 grupos)

Cada grupo é vinculado a um atributo (primário ou intermediário):

| Atributo | Perícias |
|----------|----------|
| Resistance | Vitality, Energy, Defense |
| Strength | Push, Grab, Carry |
| Agility | Velocity, Accelerate, Brake |
| Celerity | Legerity, Repel, Feint |
| Flexibility | Acrobatics, Evasion, Sneak |
| Dexterity | Reflex, Accuracy, Stealth |
| Sense | Vision, Hearing, Smell, Tact, Taste (5 perícias) |
| Constitution | Heal, Breath, Tenacity |

> Sense possui 5 perícias (não 3), refletindo os 5 sentidos humanos.

### Perícias Espirituais (6 total, 2 grupos)

| Atributo | Perícias |
|----------|----------|
| Flame | Focus, WillPower, SelfKnowledge |
| Conscience | Coa, Mop, Aop |

### Padrão de construção

Cada grupo segue o mesmo padrão:

1. Obtém o atributo do Manager via `physAttrs.Get(enum.X)`
2. Cria a primeira perícia com `NewCommonSkill(name, exp, attr, skillsManager)`
3. Clona as demais via `firstSkill.Clone(name)` — compartilha mesma `ExpTable`
   e referências, mas com nome e `Exp` próprios

O uso de `Clone()` garante consistência: todas as perícias de um grupo
compartilham o mesmo coeficiente e estão ligadas ao mesmo atributo.

---

## Wiring de Status — Referência

```
HP = NewHealthPoints(physAbility, resistance, vitality)
SP = NewStaminaPoints(physAbility, resistance, energy)
AP = NewAuraPoints(spiritualAbility, conscience, mop)  ← só se spiritualAbility != nil
```

Para detalhes das fórmulas de cada barra, veja [`status.md`](status.md).

---

## Referências de Código

| Conceito | Arquivo |
|----------|---------|
| Factory principal (Build + BuildHalfSheet) | `sheet/character_sheet_factory.go` |
| CharacterSheet (struct + construtor) | `sheet/character_sheet.go` |
| HalfSheet (struct + construtor) | `sheet/half_sheet.go` |
| CharacterClass (dados da classe) | `character_class/` (pacote externo) |
| CharacterExp (ponto final da cascata) | `experience/character_exp.go` |
| ExpTable (curva de progressão) | `experience/exp_table.go` |
| Abilities Manager | `ability/abilities_manager.go` |
| PrimaryAttribute + Clone | `attribute/primary_attribute.go` |
| MiddleAttribute | `attribute/middle_attribute.go` |
| SpiritualAttribute | `attribute/spiritual_attribute.go` |
| CharacterAttributes | `attribute/character_attributes.go` |
| CommonSkill + Clone | `skill/common_skill.go` |
| Skills Manager | `skill/skills_manager.go` |
| CharacterSkills | `skill/character_skills.go` |
| Status Manager (wiring HP/SP/AP) | `status/status_manager.go` |
| Hatsu (construção + Init) | `spiritual/hatsu.go` |
| PrinciplesManager | `spiritual/principles_manager.go` |
| Proficiency Manager | `proficiency/proficiencies_manager.go` |
| TalentByCategorySet | `sheet/talent_by_category_set.go` |

> Todos os paths são relativos a `internal/domain/entity/character_sheet/`,
> exceto `character_class/` que está em `internal/domain/entity/`.
