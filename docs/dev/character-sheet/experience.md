# Experiência & Cascade Flow

> Documentação técnica do sistema de experiência e do mecanismo de cascata (cascade upgrade).
> Pacote: `internal/domain/entity/character_sheet/experience/`

## Visão Geral

O sistema de experiência é a espinha dorsal de toda a progressão no HxH RPG System.
Cada entidade na ficha de personagem (perícia, atributo, habilidade) possui sua própria
instância de `Exp` com uma `ExpTable` configurável por coeficiente. Quando XP é
inserido em uma perícia, ele **se propaga automaticamente** por 4 pacotes em uma
única cadeia de chamadas — a **cascata** (cascade).

---

## 1. ExpTable — Tabela de Progressão

**Arquivo:** `experience/exp_table.go`

A `ExpTable` gera uma curva de progressão sigmoidal tripla que define quanto XP é
necessário para cada nível (0–100). A função central é:

```
f(lvl) = 1700 / (1 + e^(A/10 * (12 - lvl)))
       + 1800 / (1 + e^(A/10 * (38 - lvl)))
       + 2000 / (1 + e^(B/10 * (74 - lvl)))
```

Onde `A = 3.7` e `B = 2.8` são constantes fixas (`A_PARAM`, `B_PARAM`).

### Por que três sigmoides?

Cada sigmoide cria um "degrau" de dificuldade em uma faixa de níveis diferente:

| Sigmoide | Centro | Amplitude | Efeito |
|----------|--------|-----------|--------|
| 1ª       | lvl 12 | 1700      | Progressão inicial rápida |
| 2ª       | lvl 38 | 1800      | Platô intermediário |
| 3ª       | lvl 74 | 2000      | Dificuldade endgame |

O resultado é uma curva onde os primeiros níveis custam pouco XP, os níveis médios
custam progressivamente mais, e os níveis altos (74+) exigem um investimento massivo.

### Sistema de coeficientes

O coeficiente multiplicador escala a curva inteira:

```go
currExp := int(coefficient * expTableFunction(lvl))
```

Isso permite que a **mesma função** gere tabelas com ritmos diferentes. Uma perícia
com coeficiente `1.0` sobe rápido; uma habilidade com coeficiente `20.0` exige 20×
mais XP por nível.

A tabela pré-computa dois arrays no construtor:

- **`baseTable[lvl]`** — XP necessário para subir do nível `lvl-1` para `lvl`
- **`aggregateTable[lvl]`** — soma cumulativa: total de XP para alcançar o nível `lvl`

### Coeficientes por componente (referência)

| Componente             | Coeficiente | Velocidade de subida |
|------------------------|-------------|----------------------|
| Perícias Físicas       | 1.0         | Muito rápida         |
| Atributos Mentais      | 1.0         | Muito rápida         |
| Atributos Espirituais  | 1.0         | Muito rápida         |
| Princípios Nen         | 1.0         | Muito rápida         |
| Talento                | 2.0         | Rápida               |
| Perícias Mentais       | 2.0         | Rápida               |
| Perícias Espirituais   | 3.0         | Moderada             |
| Atributos Físicos      | 5.0         | Moderada             |
| Espirituais (ability)  | 5.0         | Moderada             |
| Personagem             | 10.0        | Lenta                |
| Físicos (ability)      | 20.0        | Muito lenta          |
| Mentais (ability)      | 20.0        | Muito lenta          |
| Perícias (ability)     | 20.0        | Muito lenta          |

### Resolução de nível a partir de XP

`GetLvlByExp(exp)` faz busca reversa na `aggregateTable` — percorre do nível mais
alto para o mais baixo e retorna o primeiro nível cuja XP agregada é `≤ exp`.
Isso funciona porque `aggregateTable` é estritamente crescente.

---

## 2. Fluxo de Cascade Upgrade

**Este é o mecanismo central de propagação de XP.** Ele atravessa 4 pacotes em uma
única cadeia de chamadas. Compreender esse fluxo é essencial para trabalhar em
qualquer parte do sistema de progressão.

### A cadeia completa (CommonSkill)

```
[Ponto de entrada]
CommonSkill.CascadeUpgradeTrigger(values *UpgradeCascade)
│
├── 1. skill.exp.IncreasePoints(values.GetExp())    ← perícia recebe XP
├── 2. skill.attribute.CascadeUpgrade(values)        ← delega ao atributo
│   │
│   ├── PrimaryAttribute.CascadeUpgrade(values)
│   │   ├── pa.exp.IncreasePoints(values.GetExp())   ← atributo recebe XP
│   │   ├── pa.ability.CascadeUpgrade(values)         ← delega à habilidade
│   │   │   │
│   │   │   └── Ability.CascadeUpgrade(values)
│   │   │       ├── a.exp.IncreasePoints(values.GetExp())  ← habilidade recebe XP
│   │   │       ├── a.charExp.EndCascadeUpgrade(values)     ← delega ao personagem
│   │   │       │   └── CharacterExp.EndCascadeUpgrade(values)
│   │   │       │       ├── ce.exp.IncreasePoints(values.GetExp())
│   │   │       │       └── values.CharacterExp = ce  ← registra no coletor
│   │   │       ├── if diff > 0: a.charExp.IncreaseCharacterPoints(diff)
│   │   │       └── values.Abilities[name] = AbilityCascade{...}
│   │   │
│   │   └── values.Attributes[pa.name] = AttributeCascade{...}
│   │
│   └── (ou MiddleAttribute → divide XP entre PrimaryAttributes)
│
├── 3. skill.abilitySkillsExp.CascadeUpgrade(values) ← ability de perícias
│   │
│   └── (mesma cadeia: ability → characterExp)
│
└── 4. values.Skills[name] = SkillCascade{...}       ← registra no coletor
```

### O que cada camada faz

| Camada | Struct | Método | Responsabilidade |
|--------|--------|--------|------------------|
| Perícia | `CommonSkill` | `CascadeUpgradeTrigger` | Ponto de entrada. Recebe XP e dispara a cascata em duas direções: atributo e ability de perícias |
| Atributo | `PrimaryAttribute` / `MiddleAttribute` / `SpiritualAttribute` | `CascadeUpgrade` | Recebe XP e delega à ability. MiddleAttribute divide o XP entre seus PrimaryAttributes |
| Habilidade | `Ability` | `CascadeUpgrade` | Recebe XP, finaliza a cascata via `EndCascadeUpgrade`, e concede character points se subiu de nível |
| Personagem | `CharacterExp` | `EndCascadeUpgrade` | Ponto final da cascata. Recebe XP do personagem |

### A bifurcação na perícia

Detalhe crítico: `CommonSkill.CascadeUpgradeTrigger` dispara **duas** cascatas
independentes com o **mesmo** `values.GetExp()`:

1. **`attribute.CascadeUpgrade(values)`** — XP sobe pelo atributo → ability do atributo → characterExp
2. **`abilitySkillsExp.CascadeUpgrade(values)`** — XP sobe diretamente pela ability de perícias (Physicals/Mentals/Spirituals/Skills) → characterExp

Isso significa que uma inserção de XP em uma perícia alimenta **duas** abilities
diferentes e chega ao `CharacterExp` **duas vezes**.

### MiddleAttribute — divisão de XP

`MiddleAttribute` (ex: Força, que depende de Resistência e Agilidade) trata XP de
forma diferente dos atributos primários:

```go
func (ma *MiddleAttribute) CascadeUpgrade(values *UpgradeCascade) {
    lenAttrs := len(ma.primaryAttrs)
    remainder := ma.exp.GetPoints() % lenAttrs

    ma.exp.IncreasePoints(values.GetExp())

    exp := remainder + values.GetExp()
    exp /= lenAttrs
    values.SetExp(exp)
    // ...
    for _, attr := range ma.primaryAttrs {
        attr.CascadeUpgrade(values)
    }
}
```

O XP é dividido igualmente entre os `PrimaryAttributes` subjacentes, mas com
tratamento de **resto** — o resto da divisão anterior é acumulado e somado ao
próximo XP antes de dividir. Isso evita perda de XP por truncamento inteiro.

### JointSkill — multiplicação de XP

`JointSkill` (ex: Ladinagem, que agrupa perícias comuns) multiplica o XP pelo
número de perícias componentes antes de delegar à ability:

```go
values.SetExp(exp * len(js.commonSkills))
js.abilitySkillsExp.CascadeUpgrade(values)
```

Isso reflete que uma JointSkill representa múltiplas perícias treinadas de uma vez.

---

## 3. CharacterExp — Experiência do Personagem

**Arquivo:** `experience/character_exp.go`

`CharacterExp` é o ponto final de toda cascata. Ela encapsula:

- **`exp Exp`** — a experiência do personagem (com `ExpTable` de coeficiente `10.0`)
- **`points int`** — pontos de personagem (character points), ganhos quando abilities sobem de nível

### Pontos de personagem e o bônus de habilidade

Toda vez que uma `Ability` sobe de nível durante a cascata, ela chama:

```go
if diff > 0 {
    a.charExp.IncreaseCharacterPoints(diff)
}
```

Esses pontos alimentam a **fórmula do bônus de habilidade** em `Ability.GetBonus()`:

```go
func (a *Ability) GetBonus() float64 {
    pts := float64(a.charExp.GetCharacterPoints())
    lvl := float64(a.exp.GetLevel())
    return (pts + lvl) / 2.0
}
```

O bônus influencia diretamente o **Power** dos atributos:

```go
// PrimaryAttribute
func (pa *PrimaryAttribute) GetPower() int {
    return pa.GetValue() + int(pa.GetAbilityBonus()) + *pa.buff
}
```

Isso cria um **loop de feedback positivo**: treinar perícias → abilities sobem de
nível → mais character points → bônus de habilidade maior → Power dos atributos
cresce → perícias ficam mais fortes nos testes.

### EndCascadeUpgrade vs CascadeUpgrade

`CharacterExp` implementa `IEndCascadeUpgrade` (não `ICascadeUpgrade`), pois é o
ponto **terminal** da cascata — não delega para ninguém acima.

```go
func (ce *CharacterExp) EndCascadeUpgrade(values *UpgradeCascade) {
    ce.exp.IncreasePoints(values.GetExp())
    values.CharacterExp = ce  // registra referência no coletor
}
```

---

## 4. UpgradeCascade — O Struct Coletor

**Arquivo:** `experience/upgrade_cascade.go`

`UpgradeCascade` é um struct **mutável** que viaja por toda a cadeia de cascata,
servindo dois propósitos:

### 4.1. Transportar o XP

O campo `expInserted` carrega o valor de XP atual. Ele é lido via `GetExp()` e
pode ser modificado via `SetExp()` — usado por `MiddleAttribute` para dividir XP
e por `JointSkill` para multiplicar XP.

### 4.2. Coletar resultados

Cada camada registra seu estado atualizado no struct:

```go
type UpgradeCascade struct {
    expInserted  int
    CharacterExp ICharacterExp
    Skills       map[string]SkillCascade
    Proficiency  map[string]ProficiencyCascade
    Abilities    map[enum.AbilityName]AbilityCascade
    Attributes   map[enum.AttributeName]AttributeCascade
    Principles   map[enum.PrincipleName]PrincipleCascade
    Status       map[enum.StatusName]StatusCascade
}
```

Cada sub-struct captura os dados relevantes pós-upgrade:

| Sub-struct          | Campos         | Preenchido por |
|---------------------|----------------|----------------|
| `SkillCascade`      | Exp, Lvl, TestVal | `CommonSkill.CascadeUpgradeTrigger` |
| `AttributeCascade`  | Exp, Lvl, Power   | `PrimaryAttribute.CascadeUpgrade`, `MiddleAttribute.CascadeUpgrade`, `SpiritualAttribute.CascadeUpgrade` |
| `AbilityCascade`    | Exp, Lvl, Bonus   | `Ability.CascadeUpgrade` |
| `PrincipleCascade`  | Exp, Lvl, TestVal | Princípios Nen (pacote `spiritual`) |
| `ProficiencyCascade`| Exp, Lvl          | Proficiências (pacote `proficiency`) |
| `StatusCascade`     | Min, Curr, Max    | Status Manager (pacote `status`) |

### Padrão de uso

```go
// 1. Criar o coletor com o XP a inserir
values := experience.NewUpgradeCascade(expToInsert)

// 2. Disparar a cascata
skill.CascadeUpgradeTrigger(values)

// 3. Após retorno, values contém todos os estados atualizados
//    para enviar ao cliente (API response / WebSocket message)
```

O coletor é criado **antes** da cascata e retornado **depois** com todos os dados
preenchidos. Isso evita que cada camada precise retornar seus próprios valores — um
padrão que seria muito verboso em uma cadeia de 4 níveis.

---

## 5. Interfaces-Chave

**Arquivos:** `experience/i_cascade_upgrade.go`, `experience/i_trigger_cascade_exp.go`,
`experience/i_end_cascade_upgrade.go`, `experience/I_character_exp.go`

### ICascadeUpgrade

```go
type ICascadeUpgrade interface {
    CascadeUpgrade(values *UpgradeCascade)
    GetLevel() int
}
```

Implementado por: `PrimaryAttribute`, `MiddleAttribute`, `SpiritualAttribute`,
`Ability`, `skill.Manager` (que delega à ability de perícias).

Representa qualquer entidade que **recebe** XP da cascata e o propaga para cima.

### ITriggerCascadeExp

```go
type ITriggerCascadeExp interface {
    CascadeUpgradeTrigger(values *UpgradeCascade)
}
```

Implementado por: `CommonSkill`, `JointSkill`.

Representa o **ponto de entrada** da cascata. A diferença para `ICascadeUpgrade` é
semântica: `Trigger` inicia a cadeia; `CascadeUpgrade` é um elo intermediário.

### IEndCascadeUpgrade

```go
type IEndCascadeUpgrade interface {
    EndCascadeUpgrade(values *UpgradeCascade)
}
```

Implementado por: `CharacterExp`.

Representa o **ponto final** da cascata. `EndCascadeUpgrade` não propaga — apenas
recebe XP e registra no coletor.

### ICharacterExp

```go
type ICharacterExp interface {
    IEndCascadeUpgrade
    IncreaseCharacterPoints(int)
    GetCharacterPoints() int
}
```

Combina `IEndCascadeUpgrade` com a gestão de pontos de personagem.
Usado pela `Ability` para finalizar a cascata **e** conceder character points.

### Hierarquia de interfaces

```
ITriggerCascadeExp     (ponto de entrada: skills)
        │
        ▼
ICascadeUpgrade        (elos intermediários: atributos, abilities)
        │
        ▼
IEndCascadeUpgrade     (ponto final: characterExp)
        │
        ▼
ICharacterExp          (= IEndCascadeUpgrade + character points)
```

---

## 6. Guia de Extensão — Adicionando um Novo Participante na Cascata

### Cenário: adicionar uma nova entidade que recebe XP via cascata

**Exemplo:** adicionar um novo tipo de atributo que participa da cascata.

#### Passo 1: Implementar `ICascadeUpgrade`

```go
type MeuAtributo struct {
    name enum.AttributeName
    exp  experience.Exp
    next experience.ICascadeUpgrade  // próxima camada (ability)
}

func (m *MeuAtributo) CascadeUpgrade(values *experience.UpgradeCascade) {
    m.exp.IncreasePoints(values.GetExp())
    m.next.CascadeUpgrade(values)

    values.Attributes[m.name] = experience.AttributeCascade{
        Exp:   m.exp.GetPoints(),
        Lvl:   m.exp.GetLevel(),
        Power: m.GetPower(),
    }
}

func (m *MeuAtributo) GetLevel() int {
    return m.exp.GetLevel()
}
```

#### Passo 2: Registrar dados no UpgradeCascade

Se a nova entidade precisa de um tipo de dados diferente dos existentes
(`AttributeCascade`, `SkillCascade`, etc.), adicione um novo sub-struct e campo
em `UpgradeCascade`:

```go
// Em upgrade_cascade.go
type MeuCascade struct {
    Exp int
    Lvl int
    // campos específicos
}

// Adicionar ao struct UpgradeCascade
type UpgradeCascade struct {
    // ...campos existentes...
    MeusDados map[string]MeuCascade
}
```

Não esqueça de inicializar o mapa em `NewUpgradeCascade()`.

#### Passo 3: Conectar na cadeia existente

A nova entidade precisa ser referenciada por quem vem **abaixo** (quem dispara
`CascadeUpgrade` nela) e referenciar quem vem **acima** (a ability ou
characterExp para onde o XP continua subindo).

#### Passo 4: Testes

Siga o padrão existente: `package experience_test` com testes table-driven.
Verifique que:
1. XP é propagado corretamente pela cadeia
2. O nível é atualizado ao acumular XP suficiente
3. O `UpgradeCascade` contém os dados esperados após a cascata

### Cenário: adicionar um novo ponto de entrada (como uma nova skill)

Implemente `ITriggerCascadeExp`. O método `CascadeUpgradeTrigger` deve:

1. Chamar `exp.IncreasePoints(values.GetExp())` para a própria XP
2. Chamar `attribute.CascadeUpgrade(values)` para o atributo associado
3. Chamar `abilityExp.CascadeUpgrade(values)` para a ability associada
4. Registrar os resultados no `UpgradeCascade`

Ver `skill/common_skill.go` como implementação de referência.

---

## Referências de Código

| Conceito | Arquivo |
|----------|---------|
| Função sigmoidal e tabela | `experience/exp_table.go` |
| Struct de experiência | `experience/experience.go` |
| Experiência do personagem | `experience/character_exp.go` |
| Struct coletor da cascata | `experience/upgrade_cascade.go` |
| Interface ICascadeUpgrade | `experience/i_cascade_upgrade.go` |
| Interface ITriggerCascadeExp | `experience/i_trigger_cascade_exp.go` |
| Interface IEndCascadeUpgrade | `experience/i_end_cascade_upgrade.go` |
| Interface ICharacterExp | `experience/I_character_exp.go` |
| Ponto de entrada (CommonSkill) | `skill/common_skill.go` |
| Ponto de entrada (JointSkill) | `skill/joint_skill.go` |
| Atributo primário | `attribute/primary_attribute.go` |
| Atributo intermediário | `attribute/middle_attribute.go` |
| Atributo espiritual | `attribute/spiritual_attribute.go` |
| Ability | `ability/ability.go` |

> Todos os paths são relativos a `internal/domain/entity/character_sheet/`.
