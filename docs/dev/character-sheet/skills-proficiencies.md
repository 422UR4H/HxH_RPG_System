# Perícias & Proficiências — Tipos, Cascade e Manager

> Documentação técnica dos pacotes `skill/` e `proficiency/`, cobrindo tipos de perícia,
> proficiências de armas, fluxos de cascata, sistema de buffs e o papel dos Managers.
> Pacotes: `internal/domain/entity/character_sheet/skill/`,
> `internal/domain/entity/character_sheet/proficiency/`

## Visão Geral

Perícias (skills) e proficiências (proficiencies) são os **pontos de entrada** do
sistema de cascata — é por elas que o XP entra na ficha de personagem. Cada uma tem
variantes "common" (simples) e "joint" (composta), gerenciadas por um `Manager` que
adiciona uma camada de buffs e resolução de lookup.

Para entender o mecanismo de cascata em si (como XP se propaga de perícia → atributo →
ability → characterExp), consulte [`experience.md`](./experience.md).

---

## 1. Prioridade de Lookup no Skill Manager

**Arquivo:** `skill/skills_manager.go` — método `Get(name)`

O `Manager` resolve perícias com uma **prioridade definida**: joint skills são
verificadas primeiro, depois common skills.

```
Manager.Get(name SkillName)
│
├── 1. Itera TODAS as joint skills
│   └── Para cada JointSkill: chama Contains(name)
│       └── Se true → retorna a JointSkill imediatamente
│
└── 2. Busca no mapa de common skills
    └── Se encontrou → retorna a CommonSkill
    └── Senão → ErrSkillNotFound
```

### Implicação prática

Se uma `CommonSkill` é parte de uma `JointSkill`, o `Get()` retorna a **JointSkill**,
não a `CommonSkill` individual. Isso significa que operações feitas via `Manager.Get()`
(como `GetValueForTestOf`) operam sobre a JointSkill como unidade.

### TODOs relevantes no código

O código atual contém dois TODOs nesse método que indicam que esse comportamento está
sendo reconsiderado:

- `// TODO: maybe do not get jointSkills here` — questiona se joint skills deveriam
  ser resolvidas pelo `Get()` ou ter um método separado.
- `// TODO: study if should return sum of both joint and common skills` — sugere que
  talvez o correto seja combinar os valores de ambas.

---

## 2. CommonSkill vs JointSkill — Diferenças Estruturais

**Arquivos:** `skill/common_skill.go`, `skill/joint_skill.go`

### CommonSkill

A `CommonSkill` é a perícia mais simples. Ela tem:

- **`name`** — nome da perícia (enum)
- **`exp`** — instância própria de `Exp`
- **`attribute`** — referência ao atributo associado (`IGameAttribute`)
- **`abilitySkillsExp`** — referência à ability de perícias (`ICascadeUpgrade`)

Todas as dependências são injetadas na construção — não precisa de `Init()`.

### JointSkill

A `JointSkill` agrupa múltiplas perícias (via interface `ISkill`) sob um nome composto
(ex: Ladinagem, Caça, Atleta, Hack). Ela tem:

- **`name`** — nome como `string` (não enum, pois JointSkills são compostas)
- **`exp`** — instância própria de `Exp`
- **`buff`** — buff interno (`int`)
- **`attribute`** — referência ao atributo associado
- **`commonSkills`** — mapa de perícias que compõem esta joint skill (`map[enum.SkillName]ISkill`)
- **`abilitySkillsExp`** — referência à ability de perícias (injetada via `Init()`)

| Aspecto | CommonSkill | JointSkill |
|---------|-------------|------------|
| Nome | `enum.SkillName` | `string` |
| Init necessário | Não | Sim (`Init()`) |
| Buff interno | Não | Sim (`buff int`) |
| Contém sub-perícias | Não | Sim (`commonSkills map[SkillName]ISkill`) |
| Multiplicação de XP | Não | Sim (×`len(commonSkills)`) |

---

## 3. JointSkill — Multiplicação de XP na Cascata

**Arquivo:** `skill/joint_skill.go` — método `CascadeUpgradeTrigger`

A `JointSkill` implementa uma **multiplicação de XP** antes de delegar à ability.
A lógica é:

```go
exp := values.GetExp()
js.exp.IncreasePoints(exp)           // JointSkill recebe XP original
js.attribute.CascadeUpgrade(values)  // atributo recebe XP original
values.SetExp(exp * len(js.commonSkills))  // MULTIPLICA pelo nº de componentes
values.Skills[js.name] = SkillCascade{...} // registra no coletor ANTES da ability
js.abilitySkillsExp.CascadeUpgrade(values) // ability recebe XP multiplicado
```

### Por que multiplicar?

A JointSkill representa múltiplas perícias sendo treinadas simultaneamente. O XP que
chega à ability de perícias é multiplicado pelo número de componentes para refletir
que o treinamento beneficiou vários skills de uma vez.

### Diferença de ordem em relação à CommonSkill

A **ordem** das cascatas é diferente entre CommonSkill e JointSkill, e isso é intencional:

```
CommonSkill:                          JointSkill:
1. skill.exp += XP                    1. skill.exp += XP
2. attribute.CascadeUpgrade(XP)       2. attribute.CascadeUpgrade(XP)
3. abilitySkillsExp.CascadeUpgrade(XP) 3. values.SetExp(XP × len(commonSkills))
4. registra no coletor                4. registra no coletor
                                      5. abilitySkillsExp.CascadeUpgrade(XP×N)
```

Na `CommonSkill`, o atributo e a ability recebem o **mesmo** XP. Na `JointSkill`, a
multiplicação acontece **entre** as duas cascatas — o atributo recebe XP original e a
ability recebe XP multiplicado. Note que na `JointSkill` o registro no coletor acontece
**antes** da cascata para `abilitySkillsExp`.

### TODO relevante

```go
// TODO: upgrade to evolve abilitySkillsExp just like it was done with jointProfs
```

Isso indica que o tratamento de `abilitySkillsExp` na JointSkill está sendo alinhado
ao padrão da `JointProficiency`, que já cascateia para `abilitySkillsExp` de forma
mais completa.

---

## 4. JointSkill — Init() e Proteção contra Double-Init

**Arquivo:** `skill/joint_skill.go` — método `Init()`

A `JointSkill` requer inicialização explícita via `Init()` antes de ser usada. Isso
existe porque a `abilitySkillsExp` não é conhecida no momento da construção — ela
depende da ability de perícias que é montada separadamente.

```go
func (js *JointSkill) Init(abilitySkillsExp ICascadeUpgrade) error {
    if js.abilitySkillsExp != nil {
        return ErrAbilitySkillsAlreadyInitialized  // double-init bloqueado
    }
    if abilitySkillsExp == nil {
        return ErrAbilitySkillsCannotBeNil         // nil bloqueado
    }
    js.abilitySkillsExp = abilitySkillsExp
    return nil
}
```

### Validação no Manager

O `Manager.AddJointSkill()` verifica que a JointSkill **já foi inicializada** antes
de aceitá-la:

```go
func (m *Manager) AddJointSkill(js *JointSkill) error {
    if !js.IsInitialized() { return ErrJointSkillNotInitialized }
    // ...
}
```

Isso difere do padrão de `JointProficiency`, onde o Manager chama `Init()` ele
mesmo (ver seção 6).

| Padrão | JointSkill | JointProficiency |
|--------|------------|------------------|
| Quem chama Init() | Código externo | `Manager.AddJoint()` |
| Manager valida | `IsInitialized()` | N/A (ele mesmo inicializa) |
| Parâmetros de Init | `abilitySkillsExp` | `physSkillsExp` + `abilitySkillsExp` |

---

## 5. Cascade de Proficiência — Diferenças em Relação à Perícia

**Arquivos:** `proficiency/proficiency.go`, `proficiency/joint_proficiency.go`

A cascata de proficiência é **mais simples** que a de perícia, com diferenças
estruturais importantes:

### Proficiency (comum)

```
Proficiency.CascadeUpgradeTrigger(values)
│
├── 1. prof.exp.IncreasePoints(values.GetExp())  ← proficiência recebe XP
├── 2. prof.physSkillsExp.CascadeUpgrade(values)  ← delega à ability física
└── 3. values.Proficiency[weapon] = ProficiencyCascade{...}
```

**Diferença crítica:** a `Proficiency` cascateia apenas para `physSkillsExp` (ability
de perícias físicas). Ela **não** cascateia para um atributo — proficiências não têm
atributo associado. Contraste com `CommonSkill`, que cascateia para **dois** destinos
(atributo + ability).

### JointProficiency

```
JointProficiency.CascadeUpgradeTrigger(values)
│
├── 1. jp.exp.IncreasePoints(values.GetExp())       ← proficiência recebe XP
├── 2. jp.physSkillsExp.CascadeUpgrade(values)       ← delega à ability física
├── 3. jp.abilitySkillsExp.CascadeUpgrade(values)    ← delega à ability de perícias
└── 4. values.Proficiency[name] = ProficiencyCascade{...}
```

A `JointProficiency` cascateia para **dois** destinos: `physSkillsExp` e
`abilitySkillsExp`. Isso é interessante porque a `JointSkill` ainda **não** faz
isso para `abilitySkillsExp` de forma equivalente (o TODO no código da JointSkill
referencia exatamente esse padrão).

### Comparação de cascatas

| Entidade | Cascata para atributo | Cascata para physSkillsExp | Cascata para abilitySkillsExp | Multiplicação de XP |
|----------|----------------------|---------------------------|-------------------------------|---------------------|
| CommonSkill | ✅ | ❌ | ✅ | ❌ |
| JointSkill | ✅ | ❌ | ✅ (com XP×N) | ✅ |
| Proficiency | ❌ | ✅ | ❌ | ❌ |
| JointProficiency | ❌ | ✅ | ✅ | ❌ |

### Proficiência e testes: questão em aberto

O código de `Proficiency.GetValueForTest()` retorna apenas o nível, sem somar Power
de atributo (a parte do atributo está comentada):

```go
// TODO: validate this
// proficiência realiza teste? isso faz sentido mesmo?
// proficiência é, realmente, uma skill? isso faz sentido?
func (p *Proficiency) GetValueForTest() int {
    return p.exp.GetLevel() //+ p.attribute.GetPower()
}
```

Note que `GetValueForTest()` nem está na interface `IProficiency` (está comentado).
Isso indica que o papel da proficiência em testes ainda está sendo definido.

---

## 6. JointProficiency — Agrupamento Multi-Arma e Buffs

**Arquivo:** `proficiency/joint_proficiency.go`

A `JointProficiency` agrupa múltiplas armas sob uma proficiência composta. Sua
estrutura inclui:

- **`weapons []WeaponName`** — lista de armas agrupadas
- **`buff int`** — buff único (não per-weapon no struct, apesar do `SetBuff` receber `WeaponName`)
- **`physSkillsExp`** e **`abilitySkillsExp`** — duas referências de cascata

### SetBuff — buff por arma (semântico, não estrutural)

```go
func (jp *JointProficiency) SetBuff(name WeaponName, value int) int {
    jp.buff = value
    return jp.GetLevel() + jp.buff
}
```

O `SetBuff` recebe o nome da arma como parâmetro, mas internamente armazena apenas
**um** valor de buff. Ou seja, ativar buff para qualquer arma da JointProficiency
substitui o buff de qualquer arma anterior. O retorno é `level + buff`.

### Init — duas dependências obrigatórias

Diferente da `JointSkill` (que recebe apenas `abilitySkillsExp`), a `JointProficiency`
requer **dois** parâmetros no `Init()`:

```go
func (jp *JointProficiency) Init(
    physSkillsExp ICascadeUpgrade,
    abilitySkillsExp ICascadeUpgrade,
) error
```

E a inicialização é feita **pelo próprio Manager** no `AddJoint()`:

```go
func (m *Manager) AddJoint(
    proficiency *JointProficiency,
    physSkillsExp ICascadeUpgrade,
    abilitySkillsExp ICascadeUpgrade,
) error {
    // ...
    if err := proficiency.Init(physSkillsExp, abilitySkillsExp); err != nil { return err }
    m.jointProficiencies[name] = proficiency
    return nil
}
```

---

## 7. Sistema de Buffs nos Managers

Os Managers de perícia e proficiência implementam um **sistema de buffs independente**
que se sobrepõe aos níveis das skills/proficiências. Existem **dois** mecanismos
de buff no sistema, e é importante entender onde cada um vive.

### Buff no Manager (camada externa)

Tanto `skill.Manager` quanto `proficiency.Manager` mantêm mapas de buff:

```go
// skill.Manager
buffs map[enum.SkillName]int

// proficiency.Manager
buffs map[enum.WeaponName]int
```

Esses buffs são **adicionados no momento do cálculo** de `GetValueForTestOf()` e não
afetam o nível real da skill/proficiência:

```go
// skill.Manager.GetValueForTestOf
testVal := skill.GetValueForTest()
if buff, ok := m.buffs[name]; ok {
    testVal += buff
}
```

O Manager oferece `SetBuff(name, value)` e `DeleteBuff(name)` para gerenciar buffs.

### Buff interno na JointSkill / JointProficiency

Adicionalmente, `JointSkill` e `JointProficiency` possuem um campo `buff int` interno.
Porém, apenas a `JointSkill` soma esse buff diretamente no `GetValueForTest()`:

```go
// JointSkill.GetValueForTest — inclui buff
func (js *JointSkill) GetValueForTest() int {
    return js.exp.GetLevel() + js.attribute.GetPower() + js.buff
}
```

A `JointProficiency.GetValueForTest()` **não** inclui o buff — retorna apenas o nível
(com o Power do atributo comentado, assim como a `Proficiency` comum):

```go
// JointProficiency.GetValueForTest — NÃO inclui buff
func (jp *JointProficiency) GetValueForTest() int {
    return jp.exp.GetLevel() //+ jp.attr.GetPower()
}
```

### Dualidade de buffs

Isso cria uma **dualidade** no sistema de buffs:

| Local do buff | Tipo de chave | Escopo | Gerenciado por |
|---------------|---------------|--------|----------------|
| `Manager.buffs` | `SkillName` / `WeaponName` | Por nome de skill/arma | `Manager.SetBuff()` |
| `JointSkill.buff` / `JointProficiency.buff` | N/A (campo único) | Por joint skill/prof inteira | `SetBuff()` no struct |

Na prática, quando `GetValueForTestOf()` é chamado no Manager para uma JointSkill, o
valor final inclui **ambos** os buffs:

```
valor_final = JointSkill.GetValueForTest()  +  Manager.buffs[name]
            = (level + attrPower + jsBuff)  +  managerBuff
```

Para uma `CommonSkill`, que não tem buff interno:

```
valor_final = CommonSkill.GetValueForTest()  +  Manager.buffs[name]
            = (level + attrPower)            +  managerBuff
```

---

## 8. Cálculo de Value for Test — Fórmula com Integração de Buff

O **valor para teste** (`GetValueForTest()`) é o número final usado em rolagens de
dado. A fórmula varia conforme o tipo de entidade:

### CommonSkill

```
ValueForTest = skill.Level + attribute.Power
```

Onde `attribute.Power` vem do `PrimaryAttribute`:

```go
// PrimaryAttribute.GetPower
func (pa *PrimaryAttribute) GetPower() int {
    return pa.GetValue() + int(pa.GetAbilityBonus()) + *pa.buff
}
```

Ou seja, o Power do atributo já embute o bônus de habilidade e buff de atributo.

### JointSkill

```
ValueForTest = jointSkill.Level + attribute.Power + jointSkill.buff
```

A JointSkill adiciona seu próprio buff interno.

### Proficiency

```
ValueForTest = proficiency.Level
```

A proficiência usa apenas o nível (o atributo está comentado — ver seção 5).

### Valor final via Manager

O Manager adiciona o buff externo ao valor calculado. A fórmula difere entre os dois
Managers:

**Skill Manager** — usa `skill.GetValueForTest()`:

```
FinalValue = skill.GetValueForTest() + Manager.buffs[name]
```

**Proficiency Manager** — usa `prof.GetLevel()` diretamente (com `GetValueForTest()`
comentado e um TODO questionando):

```
FinalValue = prof.GetLevel() + Manager.buffs[name]
```

### Fórmula expandida completa (CommonSkill via Manager)

```
FinalValue = skillLevel
           + attrValue + abilityBonus + attrBuff
           + managerBuff
```

Onde:
- `skillLevel` — nível da perícia (derivado de XP via `ExpTable`)
- `attrValue` — valor base do atributo (derivado de XP via `ExpTable`)
- `abilityBonus` — `(charPoints + abilityLevel) / 2` (ver `experience.md` seção 3)
- `attrBuff` — buff temporário no atributo
- `managerBuff` — buff temporário no Manager de perícias

---

## Manager como Intermediário na Cascata

**Arquivo:** `skill/skills_manager.go` — método `CascadeUpgrade`

O `skill.Manager` implementa `ICascadeUpgrade` e atua como **intermediário** na
cadeia de cascata. Quando XP chega ao Manager (vindo de uma cascata superior), ele:

1. Incrementa sua própria experiência agregada
2. Delega à ability de perícias

```go
func (m *Manager) CascadeUpgrade(values *UpgradeCascade) {
    m.exp.IncreasePoints(values.GetExp())
    m.skillsExp.CascadeUpgrade(values)
}
```

Isso permite que o Manager participe da cadeia sem precisar ser o ponto de entrada.
É diferente do papel do Manager no lookup e buffs — aqui ele é apenas um elo
passando XP adiante.

---

## Referências de Código

| Conceito | Arquivo |
|----------|---------|
| CommonSkill | `skill/common_skill.go` |
| JointSkill | `skill/joint_skill.go` |
| Interface ISkill | `skill/i_skill.go` |
| Skill Manager | `skill/skills_manager.go` |
| Proficiency | `proficiency/proficiency.go` |
| JointProficiency | `proficiency/joint_proficiency.go` |
| Interface IProficiency | `proficiency/I_proficiency.go` |
| Proficiency Manager | `proficiency/proficiency_manager.go` |
| UpgradeCascade (coletor) | `experience/upgrade_cascade.go` |
| Cascade interfaces | `experience/i_cascade_upgrade.go`, `experience/i_trigger_cascade_exp.go` |
| PrimaryAttribute.GetPower | `attribute/primary_attribute.go` |

> Todos os paths são relativos a `internal/domain/entity/character_sheet/`.
>
> Para detalhes sobre o mecanismo de cascata, `ExpTable`, `CharacterExp` e as interfaces
> de cascata, consulte [`experience.md`](./experience.md).
