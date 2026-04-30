# Habilidades & Atributos — Hierarquia de Entidades

> Documentação técnica dos pacotes `ability/` e `attribute/` e suas relações.
> Pacotes: `internal/domain/entity/character_sheet/ability/` e `attribute/`

## Visão Geral

As habilidades (abilities) e atributos (attributes) formam a camada intermediária
da cascata de XP. Os atributos recebem XP das perícias abaixo e delegam para as
habilidades acima, que por sua vez finalizam a cascata no `CharacterExp`.

A parte não óbvia é que as habilidades **retroalimentam** os atributos: o bônus
calculado por `Ability.GetBonus()` é um componente direto do `Power` de todo
atributo. Isso cria um ciclo onde treinar qualquer perícia eventualmente fortalece
todos os atributos sob a mesma habilidade.

Para o fluxo completo da cascata (como XP viaja de uma perícia até `CharacterExp`),
veja [`experience.md`](experience.md) §2.

---

## 1. Hierarquia de Entidades

### Tipos de atributo

O sistema possui três tipos de atributo, cada um com semântica diferente para
Power e distribuição de pontos:

```
ICascadeUpgrade (interface da cascata)
├── IGameAttribute            (base para todos os atributos)
│   ├── SpiritualAttribute    — atributo espiritual (sem pontos distribuíveis)
│   └── ...
├── IDistributableAttribute   (superset estrutural de IGameAttribute + GetPoints/GetValue)
│   ├── PrimaryAttribute      — atributo primário (ex: Força, Agilidade)
│   └── MiddleAttribute       — atributo derivado de 2+ primários (ex: Constituição)
```

> **Nota Go:** `IDistributableAttribute` e `IGameAttribute` não possuem relação de
> embedding — ambas embeddam `ICascadeUpgrade` independentemente. Porém,
> `IDistributableAttribute` é um superset estrutural (qualquer implementação satisfaz
> `IGameAttribute` também).

| Tipo | Points | Value | Power | Cascade |
|------|--------|-------|-------|---------|
| `PrimaryAttribute` | `points` (manual) | `points + level` | `value + abilityBonus + buff` | Recebe XP, delega à `Ability` |
| `MiddleAttribute` | Média dos primários (`math.Round`) | `points + level` | `value + abilityBonus + buff` | Divide XP entre primários, cada um cascateia |
| `SpiritualAttribute` | — (não existe) | — | `level + abilityBonus + buff` | Recebe XP, delega à `Ability` |

**Diferença-chave:** `SpiritualAttribute` não tem campo `points` nem `GetValue()`.
Seu Power depende apenas do nível, do bônus da habilidade e do buff. Isso reflete
a natureza do Nen no RPG: atributos espirituais não aceitam distribuição manual.

### A Ability

`Ability` fica no topo da cadeia, logo antes de `CharacterExp`. Existem 4
habilidades: `Physicals`, `Mentals`, `Spirituals` e `Skills`. Cada uma agrupa
um conjunto de atributos que compartilham a mesma curva de progressão.

**Arquivo:** `ability/ability.go`

A `Ability` possui:
- Sua própria `Exp` (com coeficiente alto: 20.0 para Physicals/Mentals/Skills, 5.0 para Spirituals)
- Uma referência a `ICharacterExp` para finalizar a cascata e gerenciar character points

### Hierarquia de interfaces

```
IGameAttribute                   IAbility
├── ICascadeUpgrade              ├── ICascadeUpgrade (CascadeUpgrade + GetLevel)
├── GetPower()                   ├── GetBonus() float64
├── GetAbilityBonus()            ├── GetExpReference() ICascadeUpgrade
└── (exp getters)                └── (exp getters)

IDistributableAttribute          (superset de IGameAttribute)
├── ICascadeUpgrade
├── (tudo de IGameAttribute)
├── GetPoints() int
└── GetValue() int
```

`IGameAttribute` é implementado por todos os tipos de atributo. `IDistributableAttribute`
adiciona `GetPoints()` e `GetValue()` — exclusivo para atributos físicos e mentais.
O `Manager` retorna `IDistributableAttribute` em seu `Get()`, enquanto o
`SpiritualManager` retorna `IGameAttribute`.

**Arquivos:** `attribute/i_game_attribute.go`, `attribute/i_distributable_attribute.go`

---

## 2. Padrão Manager

Os atributos são organizados em três managers, agrupados na struct `CharacterAttributes`:

```
CharacterAttributes
├── physicals:  *Manager            (PrimaryAttribute + MiddleAttribute)
├── mentals:    *Manager            (PrimaryAttribute + MiddleAttribute)
└── spirituals: *SpiritualManager   (SpiritualAttribute apenas)
```

### Manager (atributos físicos e mentais)

**Arquivo:** `attribute/attributes_manager.go`

O `Manager` armazena atributos primários e intermediários em mapas separados:

```go
type Manager struct {
    primaryAttributes map[enum.AttributeName]*PrimaryAttribute
    middleAttributes  map[enum.AttributeName]*MiddleAttribute
    buffs             map[enum.AttributeName]*int
}
```

`Get()` busca primeiro nos primários, depois nos intermediários. Retorna
`IDistributableAttribute` — permitindo acesso uniforme a `GetPoints()` e
`GetValue()` independente do tipo concreto.

`GetPrimary()` é o getter especializado que retorna uma **cópia por valor**
(`*primaryAttribute` é desreferenciado). Isso é intencional: impede que o
chamador modifique o estado interno do atributo diretamente. Para alterar pontos,
o caminho correto é `IncreasePointsForPrimary()`.

### SpiritualManager

**Arquivo:** `attribute/spiritual_attributes_manager.go`

Mesmo padrão do `Manager`, mas mais simples: armazena apenas `SpiritualAttribute`
(sem primários/intermediários). Retorna `IGameAttribute` em vez de
`IDistributableAttribute`, pois atributos espirituais não possuem pontos
distribuíveis.

### ability.Manager

**Arquivo:** `ability/abilities_manager.go`

O `ability.Manager` organiza as 4 habilidades e gerencia o `Talent`:

```go
type Manager struct {
    characterExp *exp.CharacterExp
    abilities    map[enum.AbilityName]IAbility
    talent       Talent
}
```

Além de acesso às habilidades por nome, ele expõe `GetExpReferenceOf()` — que
retorna a interface `ICascadeUpgrade` de uma ability. Isso é usado pelas skills
e atributos para conectar-se à ability correta na cadeia de cascata, sem
depender do tipo concreto.

---

## 3. Cálculo do MiddleAttribute

O `MiddleAttribute` (ex: Defesa = Força + Constituição) envolve dois algoritmos
distintos que operam sobre conceitos diferentes:

### 3.1. Divisão de XP na cascata (com rastreamento de resto)

Documentado em [`experience.md`](experience.md) §2 ("MiddleAttribute — divisão de XP").

Resumo: durante `CascadeUpgrade`, o XP é dividido igualmente entre os primários
subjacentes, com acumulação de resto para evitar perda por truncamento inteiro.

### 3.2. Cálculo de Points (média com arredondamento)

**Este é o mecanismo que NÃO está documentado em `experience.md`.**

`GetPoints()` calcula a média dos `points` distribuídos nos primários subjacentes:

```go
func (ma *MiddleAttribute) GetPoints() int {
    points := 0
    for _, primaryAttr := range ma.primaryAttrs {
        points += primaryAttr.points  // acessa campo não-exportado (mesmo pacote)
    }
    return int(math.Round(float64(points) / float64(len(ma.primaryAttrs))))
}
```

Detalhes importantes:

- **Acessa `primaryAttr.points` diretamente** — possível porque `MiddleAttribute`
  está no mesmo pacote. Diferente de `GetPoints()` que retornaria o valor público.
- **Usa `math.Round`** — arredondamento half away from zero. Para 2
  primários com points {3, 4}, o resultado é `Round(3.5) = 4`. Note que
  `Round(2.5) = 3` (arredonda para longe de zero, diferente de banker's rounding).
- **Points é somente leitura** para `MiddleAttribute` — ele não aceita
  `IncreasePoints()`. Os pontos são distribuídos apenas nos primários.

### 3.3. Cálculo do AbilityBonus

`GetAbilityBonus()` também faz média, mas dos bônus de ability (não dos points):

```go
func (ma *MiddleAttribute) GetAbilityBonus() float64 {
    // soma GetAbilityBonus() de cada primário / quantidade de primários
}
```

Isso é necessário porque um `MiddleAttribute` pode depender de primários que
pertencem a abilities diferentes (embora na prática atual todos os primários de
um intermediário compartilhem a mesma ability).

---

## 4. Fórmula do Bônus de Habilidade

**Arquivo:** `ability/ability.go` — `GetBonus()`

```go
func (a *Ability) GetBonus() float64 {
    pts := float64(a.charExp.GetCharacterPoints())
    lvl := float64(a.exp.GetLevel())
    return (pts + lvl) / 2.0
}
```

### Como funciona

- **`characterPoints`** — total de pontos de personagem, ganhos quando *qualquer*
  ability sobe de nível. É um valor **global** (compartilhado por todas as
  abilities via `ICharacterExp`).
- **`level`** — nível da ability específica.
- **Resultado** — média dos dois. Retorna `float64` para preservar precisão,
  mas é truncado para `int` ao ser usado no cálculo de Power.

### Implicação de design

O bônus de *todas* as abilities cresce quando *qualquer uma* sobe de nível
(porque `characterPoints` é global). Isso é intencional: um personagem que
treina muito Mentals eventualmente fortalece também seus atributos físicos,
refletindo a maturidade geral do personagem.

A fórmula `(pts + lvl) / 2.0` balanceia dois fatores:
1. **Progressão individual** (`lvl`): recompensa foco em uma habilidade
2. **Progressão geral** (`pts`): recompensa diversidade de treinamento

### Cascata reversa: como o bônus chega no Power

```
Ability.GetBonus()
    ↓
PrimaryAttribute.GetAbilityBonus()  → delega para ability.GetBonus()
    ↓
PrimaryAttribute.GetPower()         → value + abilityBonus + buff
```

O `MiddleAttribute` calcula a média dos `GetAbilityBonus()` dos seus primários.
O `SpiritualAttribute` delega diretamente para sua ability.

---

## 5. Sistema de Talento

### Talent (pacote ability)

**Arquivo:** `ability/talent.go`

O `Talent` é uma instância de `Exp` independente (coeficiente 2.0) que não
participa da cascata de XP. Ele existe no `ability.Manager` e serve como
um indicador do potencial Nen do personagem.

```go
type Talent struct {
    exp experience.Exp
}
```

`InitWithLvl(lvl)` inicializa o talento com a quantidade de XP acumulada
necessária para alcançar o nível dado. Isso é usado na criação do personagem
quando o nível de talento é determinado pelo `TalentByCategorySet`.

### TalentByCategorySet (pacote sheet)

**Arquivo:** `sheet/talent_by_category_set.go`

Este struct calcula o nível de talento com base nas categorias Nen ativas do
personagem:

```go
const BASE_TALENT_LVL = 20

func (t *TalentByCategorySet) GetTalentLvl() int {
    activeCategoryCount := getActiveCategoryCount(t.categories)
    bonus := activeCategoryCount - 1

    if t.initialHexValue == nil {  // sem hexágono
        bonus *= 2
        if bonus == 0 {
            bonus = 1
        }
    }
    return BASE_TALENT_LVL + bonus
}
```

### Lógica do bônus

O cálculo possui dois caminhos dependendo da presença do hexágono:

| Cenário | Categorias ativas | Bônus | Talento resultante |
|---------|-------------------|-------|--------------------|
| **Com hexágono** | 1 | 0 | 20 |
| **Com hexágono** | 2 | 1 | 21 |
| **Com hexágono** | 3 | 2 | 22 |
| **Sem hexágono** | 1 | 1 (mínimo forçado) | 21 |
| **Sem hexágono** | 2 | 2 | 22 |
| **Sem hexágono** | 3 | 4 | 24 |

**Com hexágono** (`initialHexValue != nil`): bônus linear simples —
`categorias - 1`. O hexágono já confere poder Nen, então o talento cresce
devagar.

**Sem hexágono** (`initialHexValue == nil`): bônus dobrado —
`(categorias - 1) * 2`. Compensa a ausência do hexágono. Caso especial: com
apenas 1 categoria ativa, o bônus é forçado para 1 (não zero), garantindo que
o personagem sempre tenha talento acima da base.

### Fluxo de inicialização

```
CharacterSheet creation
├── TalentByCategorySet.GetTalentLvl()  → calcula nível
├── ability.Manager.InitTalentWithLvl(lvl)
│   └── Talent.InitWithLvl(lvl)
│       └── exp.GetAggregateExpByLvl(lvl)  → converte nível em XP
│       └── exp.IncreasePoints(aggregateXP) → define o XP do talento
```

O talento é inicializado uma vez na criação da ficha. Depois disso, ele cresce
via `IncreaseTalentExp()` chamado pelo engine de progressão.

---

## 6. Sistema de Buff (Compartilhamento por Ponteiro)

**Este é o design mais não-óbvio dos pacotes de atributo.**

### Como funciona

Cada atributo recebe um `*int` (ponteiro para int) no construtor. O `Manager`
mantém um mapa `buffs map[enum.AttributeName]*int` contendo os mesmos ponteiros:

```
Manager.buffs["Strength"] ──→ *int (valor: 5)
                                 ↑
PrimaryAttribute.buff ───────────┘  (mesmo ponteiro)
```

Quando o `Manager` executa `SetBuff()`:

```go
func (m *Manager) SetBuff(name enum.AttributeName, buff int) (...) {
    *m.buffs[name] = buff  // desreferencia o ponteiro e altera o valor
}
```

O atributo vê a mudança imediatamente em `GetPower()`:

```go
func (pa *PrimaryAttribute) GetPower() int {
    return pa.GetValue() + int(pa.GetAbilityBonus()) + *pa.buff  // lê o mesmo ponteiro
}
```

### Por que ponteiros?

Este design elimina a necessidade de notificação ou callback. O Manager não
precisa "avisar" o atributo que o buff mudou — ambos compartilham a mesma
posição de memória. Isso é simples e eficiente, mas exige atenção:

- **Não copie o `*int`** — armazenar `buff := *pa.buff` captura o valor, não o
  ponteiro. O `Clone()` de atributo recebe um `*int` novo explicitamente.
- **Inicialização no factory** — o factory da character sheet cria o mapa de
  buffs e distribui os ponteiros para cada atributo e para o manager. Se um
  atributo receber um ponteiro que não está no mapa do manager, `SetBuff` e
  `RemoveBuff` não terão efeito sobre ele.
- **`RemoveBuff()` zera o valor** — `*m.buffs[name] = 0`, não remove a entrada
  do mapa. O ponteiro continua válido.

### Consistência entre Manager e SpiritualManager

O `SpiritualManager` usa exatamente o mesmo padrão de ponteiros. A consistência
é mantida pela factory que constrói ambos com o mesmo estilo de inicialização.

---

## 7. Distribuição de Pontos

A distribuição manual de pontos segue regras restritivas no design atual:

### Quem aceita pontos?

| Tipo | Aceita distribuição? | Método |
|------|---------------------|--------|
| `PrimaryAttribute` (físico) | ✅ Sim | `IncreasePoints(value)` |
| `PrimaryAttribute` (mental) | ❌ Não* | — |
| `MiddleAttribute` | ❌ Não | Points é calculado (média dos primários) |
| `SpiritualAttribute` | ❌ Não | Não possui campo `points` |

*\* Nota: embora `PrimaryAttribute` mental tenha o método `IncreasePoints()`, o
`CharacterAttributes` só expõe `IncreasePrimaryPhysicalPts()` — que delega
exclusivamente para `ca.physicals`. Não existe um método equivalente para
mentais na API pública de `CharacterAttributes`.*

### Fluxo de distribuição

```
CharacterAttributes.IncreasePrimaryPhysicalPts(name, points)
└── Manager.IncreasePointsForPrimary(name, value)
    ├── valida: nome existe em primaryAttributes? (não aceita middle)
    ├── attr.IncreasePoints(value)
    └── retorna GetAttributesPoints() (todos os pontos atualizados)
```

### Impacto nos cálculos derivados

Quando pontos são adicionados a um `PrimaryAttribute`:

1. **`PrimaryAttribute.GetValue()`** aumenta (`points + level`)
2. **`PrimaryAttribute.GetPower()`** aumenta (`value + abilityBonus + buff`)
3. **`MiddleAttribute.GetPoints()`** reflete a mudança (média recalculada)
4. **`MiddleAttribute.GetValue()`** e `GetPower()` atualizam em cascata

A distribuição de pontos **não** gera XP — é um bônus direto ao valor do atributo.

---

## Referências de Código

| Conceito | Arquivo |
|----------|---------|
| Ability (struct + bônus) | `ability/ability.go` |
| Interface IAbility | `ability/i_ability.go` |
| Abilities Manager | `ability/abilities_manager.go` |
| Talent | `ability/talent.go` |
| TalentByCategorySet | `sheet/talent_by_category_set.go` |
| PrimaryAttribute | `attribute/primary_attribute.go` |
| MiddleAttribute | `attribute/middle_attribute.go` |
| SpiritualAttribute | `attribute/spiritual_attribute.go` |
| Attributes Manager | `attribute/attributes_manager.go` |
| SpiritualManager | `attribute/spiritual_attributes_manager.go` |
| CharacterAttributes | `attribute/character_attributes.go` |
| Interface IGameAttribute | `attribute/i_game_attribute.go` |
| Interface IDistributableAttribute | `attribute/i_distributable_attribute.go` |
| Cascata de XP (fluxo completo) | Veja [`experience.md`](experience.md) |

> Todos os paths são relativos a `internal/domain/entity/character_sheet/`.
