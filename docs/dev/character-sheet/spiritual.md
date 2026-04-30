# Sistema Espiritual / Nen

> Documentação técnica do sistema espiritual (Nen): hexágono de categorias,
> princípios, Hatsu e cascata de XP.
> Pacote: `internal/domain/entity/character_sheet/spiritual/`

## Visão Geral

O sistema espiritual modela o Nen do universo Hunter × Hunter. Ele é composto
por quatro entidades principais que cooperam:

| Entidade | Struct | Responsabilidade |
|----------|--------|------------------|
| Hexágono Nen | `NenHexagon` | Algoritmo circular que determina categoria e porcentagens |
| Categoria Nen | `NenCategory` | Ponto de entrada de XP por categoria; escala valor pelo percentual |
| Princípio Nen | `NenPrinciple` | Ponto de entrada de XP por princípio (Ten, Zetsu, Ren, etc.) |
| Hatsu | `Hatsu` | Coordena categorias, armazena percentuais e participa da cascata |
| Manager | `Manager` | Fachada que unifica princípios, hexágono e Hatsu |

Dois atributos espirituais participam de todo o subsistema:

- **`conscienceNen`** — recebe XP da cascata (leitura + escrita)
- **`flameNen`** — usado somente para cálculo de valor de teste (somente leitura)

---

## 1. Hexágono Nen — Algoritmo Circular de Categorias

**Arquivo:** `spiritual/nen_hexagon.go`

As 6 categorias Nen são dispostas em um espaço circular de 600 valores. Cada
categoria ocupa uma faixa de 100 valores (`categoryRange = maxHexRange / 6`):

```
                    Reinforcement (0)
                         /    \
                        /      \
              Emission (500)    Transmutation (100)
                      |          |
                      |          |
            Manipulation (400)  Materialization (200)
                        \      /
                         \    /
                    Specialization (300)
```

O `currHexValue` (0–599) indica a posição atual do personagem no hexágono.
A categoria é determinada pela posição mais próxima, com uma zona de ±50
valores ao redor do centro de cada categoria.

### Constantes

```go
const (
    percentageJumpByCategory = 20    // queda de % por salto de categoria
    maxHexRange              = 600   // comprimento total do círculo
    categoryRange            = 100   // = maxHexRange / 6
)
```

### Determinação de categoria

Duas funções determinam a categoria a partir de `currHexValue`:

| Função | Uso | Comportamento na fronteira (valor == centro + 50) |
|--------|-----|---------------------------------------------------|
| `getCategoryByHexagon()` | Inicialização (construtor) | Retorna a próxima categoria na ordem |
| `UpdateCategoryByHexagon()` | Incremento/decremento | **Mantém a categoria atual** (tie-breaking) |

A diferença é crítica: ao incrementar/decrementar o hexágono, um personagem que
está exatamente na fronteira entre duas categorias permanece na categoria em que
já estava. Na inicialização (sem estado prévio), não há tie-breaking.

### Cálculo de porcentagem — `GetPercentOf()`

```go
func (nh *NenHexagon) GetPercentOf(category enum.CategoryName) float64 {
    // 1. Exceção de Especialização (ver seção 2)
    if category == enum.Specialization && nh.nenCategoryName != enum.Specialization {
        return 0.0
    }
    // 2. Distância absoluta no círculo
    absHexDiff := math.Abs(float64(nenHexagon[category] - nh.currHexValue))
    if absHexDiff > maxHexRange/2 {
        absHexDiff = maxHexRange - absHexDiff  // wrap circular
    }
    // 3. Conversão para perda percentual
    divisor := categoryRange / percentageJumpByCategory  // = 100 / 20 = 5
    absHexDiff /= float64(divisor)
    // 4. Percentual final
    percent := 100.0 - absHexDiff
    return percent
}
```

**Passo a passo da fórmula:**

1. Calcula a distância absoluta entre `currHexValue` e a posição canônica da
   categoria no mapa `nenHexagon`
2. Se a distância excede metade do círculo (300), usa o caminho mais curto:
   `600 - distância`
3. Divide a distância pelo divisor (`5`), convertendo para "saltos de
   categoria". Cada unidade de distância = 1/5 de ponto percentual
4. Subtrai de 100% para obter o percentual final

**Exemplos (personagem na posição 0 = Reinforcement):**

| Categoria | Posição | Distância | ÷ 5 | Percentual |
|-----------|---------|-----------|-----|------------|
| Reinforcement | 0 | 0 | 0.0 | **100%** |
| Transmutation | 100 | 100 | 20.0 | **80%** |
| Materialization | 200 | 200 | 40.0 | **60%** |
| Specialization | 300 | — | — | **0%** (exceção) |
| Manipulation | 400 | 200* | 40.0 | **60%** |
| Emission | 500 | 100* | 20.0 | **80%** |

\* Via wrap circular: `|400-0| = 400 > 300`, então `600 - 400 = 200`.

---

## 2. Exceção de Especialização

**Arquivo:** `spiritual/nen_hexagon.go` — `GetPercentOf()`

Especialização segue uma regra única no anime: um personagem que **não é**
Especialista sempre tem 0% de afinidade com Especialização, independente da
distância no hexágono.

```go
if category == enum.Specialization && nh.nenCategoryName != enum.Specialization {
    return 0.0
}
```

Se o personagem **é** Especialista (`nenCategoryName == Specialization`), o
cálculo de porcentagem segue normalmente — incluindo a distância para
Especialização (que será 100% se estiver na posição canônica 300).

Isso garante que Especialização seja uma categoria exclusiva: apenas quem nasce
ou evolui para Especialista pode acessá-la.

---

## 3. Inicialização do Hatsu

**Arquivo:** `spiritual/hatsu.go`

`Hatsu` é construído via `NewHatsu()` com um mapa de categorias **vazio**
(o construtor ignora o parâmetro `categories` e inicializa com `make()`).
As categorias reais são injetadas depois, via `Init()`:

```go
func (h *Hatsu) Init(categories map[enum.CategoryName]NenCategory) error {
    if len(h.categories) > 0 {
        return ErrNenHexAlreadyInitialized
    }
    h.categories = categories
    return nil
}
```

### Por que inicialização em duas fases?

Existe uma dependência circular: cada `NenCategory` precisa de uma referência
`IHatsu` para chamar `CascadeUpgrade()` e `GetPercentOf()`, mas o `Hatsu`
precisa do mapa de categorias. Solução:

1. Cria-se o `Hatsu` (vazio de categorias)
2. Cria-se cada `NenCategory` passando o `Hatsu` como `IHatsu`
3. Chama-se `hatsu.Init(categories)` com o mapa completo

### Proteção contra dupla inicialização

`Init()` retorna `ErrNenHexAlreadyInitialized` se chamado com categorias já
presentes. Isso previne que o estado interno seja sobrescrito acidentalmente
após a construção.

---

## 4. Caminho de Cascata

O sistema espiritual possui **dois pontos de entrada** para XP, ambos
terminando em `CharacterExp`:

### 4.1. Cascata por Categoria (via `IncreaseExpByCategory`)

```
[Ponto de entrada]
Manager.IncreaseExpByCategory(name, values)
│
└── Hatsu.IncreaseExp(values, name)
    │
    └── NenCategory.CascadeUpgradeTrigger(values)
        │
        ├── 1. category.exp.IncreasePoints(values.GetExp())   ← categoria recebe XP
        │
        └── 2. hatsu.CascadeUpgrade(values)                   ← delega ao Hatsu
            │
            ├── hatsu.exp.IncreasePoints(values.GetExp())      ← Hatsu recebe XP
            │
            └── conscienceNen.CascadeUpgrade(values)           ← atributo espiritual
                │
                └── Ability(Spirituals).CascadeUpgrade(values) ← habilidade
                    │
                    └── CharacterExp.EndCascadeUpgrade(values)  ← ponto final
```

### 4.2. Cascata por Princípio (via `IncreaseExpByPrinciple`)

```
[Ponto de entrada]
Manager.IncreaseExpByPrinciple(name, values)
│
└── NenPrinciple.CascadeUpgradeTrigger(values)
    │
    ├── 1. principle.exp.IncreasePoints(values.GetExp())  ← princípio recebe XP
    │
    └── 2. conscienceNen.CascadeUpgrade(values)           ← atributo espiritual
        │
        └── (mesma cadeia: Ability → CharacterExp)
```

### Diferença fundamental

| Aspecto | Categoria | Princípio |
|---------|-----------|-----------|
| Caminho | Category → Hatsu → conscienceNen → Ability → CharacterExp | Principle → conscienceNen → Ability → CharacterExp |
| Entidades que recebem XP | 4 (category, hatsu, conscienceNen, ability) | 3 (principle, conscienceNen, ability) |
| Fórmula de teste | Escalada pelo percentual da categoria | Baseada em nível + power + flameNen |

### FlameNen vs ConscienceNen na cascata

Detalhe crítico: tanto `NenPrinciple` quanto `Hatsu` possuem referências a
`flameNen` e `conscienceNen`, mas **somente `conscienceNen` participa da
cascata**. O `flameNen` é usado exclusivamente no cálculo de `GetValueForTest()`.

```go
// NenPrinciple.CascadeUpgradeTrigger — somente conscienceNen
np.conscienceNen.CascadeUpgrade(values)  // ← cascata
// flameNen NÃO é chamado aqui

// NenPrinciple.GetValueForTest — ambos participam
flameNenlvl := np.flameNen.GetLevel()
return np.GetLevel() + int(np.conscienceNen.GetPower()) + flameNenlvl
```

---

## 5. PrinciplesManager — Fachada de Coordenação

**Arquivo:** `spiritual/principles_manager.go`

`Manager` é a fachada que unifica os três componentes do sistema espiritual:
princípios, hexágono e Hatsu. Responsabilidades:

### 5.1. Gerenciamento de princípios e categorias

```go
// Dois pontos de entrada para XP
m.IncreaseExpByPrinciple(name, values)  // → principle.CascadeUpgradeTrigger()
m.IncreaseExpByCategory(name, values)   // → hatsu.IncreaseExp()
```

### 5.2. Hatsu como princípio especial

`Manager.Get()` trata `enum.Hatsu` como caso especial — retorna o `Hatsu`
diretamente ao invés de buscá-lo no mapa de princípios:

```go
func (m *Manager) Get(name enum.PrincipleName) (IPrinciple, error) {
    if name == enum.Hatsu {
        return m.hatsu, nil  // Hatsu não está no mapa de princípios
    }
    if principle, ok := m.principles[name]; ok {
        return principle, nil
    }
    return nil, fmt.Errorf("%w: %s", ErrPrincipleNotFound, name.String())
}
```

Isso funciona porque `Hatsu` implementa `IPrinciple` (através de seus métodos
`GetValueForTest()`, `GetLevel()`, etc.), além de implementar
`experience.ICascadeUpgrade` via `IHatsu`. O Hatsu possui um papel duplo:

- Como **`IPrinciple`**: expõe dados de nível/XP ao Manager
- Como **`ICascadeUpgrade`**: atua como elo intermediário na cascata de categorias

### 5.3. Sincronização hexágono ↔ Hatsu

Toda operação que modifica o hexágono atualiza automaticamente os percentuais
no Hatsu:

```go
func (m *Manager) IncreaseCurrHexValue() (*NenHexagonUpdateResult, error) {
    if m.nenHexagon == nil {
        return nil, ErrNenHexNotInitialized
    }
    result := m.nenHexagon.IncreaseCurrHexValue()
    m.hatsu.SetCategoryPercents(result.PercentList)  // ← sincroniza
    return result, nil
}
```

A mesma sincronização ocorre em `DecreaseCurrHexValue()`, `ResetNenCategory()`,
e `InitNenHexagon()`.

### 5.4. Inicialização do hexágono

O hexágono pode ser `nil` no `Manager` (personagens que ainda não despertaram
Nen). `InitNenHexagon()` injeta o hexágono e sincroniza os percentuais:

```go
func (m *Manager) InitNenHexagon(nenHexagon *NenHexagon) error {
    if m.nenHexagon != nil {
        return ErrNenHexAlreadyInitialized
    }
    m.nenHexagon = nenHexagon
    m.hatsu.SetCategoryPercents(nenHexagon.GetCategoryPercents())
    return nil
}
```

Todas as operações que dependem do hexágono (`Increase`, `Decrease`, `Reset`,
`GetNenCategoryName`, `GetCurrHexValue`) verificam se `nenHexagon != nil` e
retornam `ErrNenHexNotInitialized` caso contrário.

---

## 6. Valor para Teste (GetValueForTest)

### 6.1. NenCategory — fórmula com escala percentual

**Arquivo:** `spiritual/nen_category.go`

```go
func (nc *NenCategory) GetValueForTest() int {
    value := float64((nc.GetLevel() + nc.hatsu.GetValueForTest()))
    return int(value * nc.GetPercent() / 100.0)
}
```

A fórmula combina três componentes:

1. **Nível da categoria** (`nc.GetLevel()`) — progressão individual
2. **Valor de teste do Hatsu** (`nc.hatsu.GetValueForTest()`) — contribuição
   do Hatsu compartilhado por todas as categorias
3. **Percentual da categoria** (`nc.GetPercent()`) — escala tudo pelo percentual
   hexagonal

O percentual é obtido via `hatsu.GetPercentOf(nc.name)`, que consulta o mapa
`categoryPercents` no Hatsu (preenchido pelo hexágono). Isso garante que a
afinidade da categoria influencie diretamente a eficácia nos testes.

**Exemplo:** Um Reinforcement (100%) com nível 10 e Hatsu testVal 20 terá:
`int((10 + 20) * 100 / 100) = 30`. Um Materialization (60%) com os mesmos
valores terá: `int((10 + 20) * 60 / 100) = 18`.

### 6.2. NenPrinciple e Hatsu — fórmula sem escala

**Arquivo:** `spiritual/nen_principle.go`, `spiritual/hatsu.go`

```go
func (np *NenPrinciple) GetValueForTest() int {
    flameNenlvl := np.flameNen.GetLevel()
    return np.GetLevel() + int(np.conscienceNen.GetPower()) + flameNenlvl
}
```

A fórmula é idêntica para `NenPrinciple` e `Hatsu`:

| Componente | Fonte | Descrição |
|------------|-------|-----------|
| Nível próprio | `GetLevel()` | Progressão da entidade |
| Power de conscienceNen | `conscienceNen.GetPower()` | Força do atributo espiritual |
| Nível de flameNen | `flameNen.GetLevel()` | Nível do atributo de chama |

Note que **não há escala percentual** — princípios representam técnicas
fundamentais que todo usuário de Nen domina igualmente.

---

## 7. Mecanismo de Reset — `ResetNenCategory()`

**Arquivo:** `spiritual/nen_hexagon.go` (impl), `spiritual/principles_manager.go` (fachada)

Inspirado no arco de Formigas Quimera, onde Gon perde e precisa reconquistar
seu Nen, o sistema permite resetar a posição hexagonal de um personagem:

```go
// NenHexagon.ResetCategory
func (nh *NenHexagon) ResetCategory() int {
    nh.currHexValue = nenHexagon[nh.nenCategoryName]
    return nh.currHexValue
}

// Manager.ResetNenCategory (fachada)
func (m *Manager) ResetNenCategory() (int, error) {
    if m.nenHexagon == nil {
        return -1, ErrNenHexNotInitialized
    }
    currHexValue := m.nenHexagon.ResetCategory()
    m.hatsu.SetCategoryPercents(m.nenHexagon.GetCategoryPercents())
    return currHexValue, nil
}
```

### O que o reset faz

1. **Reposiciona** `currHexValue` para a posição canônica da categoria atual
   (ex: Reinforcement → 0, Transmutation → 100)
2. **Recalcula** todos os percentuais das categorias
3. **Sincroniza** os novos percentuais com o Hatsu

### O que o reset NÃO faz

- **Não muda a categoria** — `nenCategoryName` permanece inalterada
- **Não zera XP** — a experiência de princípios, categorias e Hatsu é mantida
- **Não zera níveis** — toda a progressão é preservada

O efeito prático é normalizar os percentuais para os valores "puros" da
categoria, desfazendo qualquer desvio causado por incrementos/decrementos
anteriores no hexágono.

---

## Interfaces-Chave

**Arquivos:** `spiritual/i_hatsu.go`, `spiritual/i_principle.go`,
`spiritual/i_category.go`

### IHatsu

```go
type IHatsu interface {
    experience.ICascadeUpgrade          // CascadeUpgrade() + GetLevel()
    GetPercentOf(enum.CategoryName) float64
    GetValueForTest() int
    GetNextLvlAggregateExp() int
    GetNextLvlBaseExp() int
    GetCurrentExp() int
    GetExpPoints() int
}
```

Usado por `NenCategory` para acessar o Hatsu sem depender do tipo concreto.
Combina `ICascadeUpgrade` (participação na cascata) com consulta de percentuais
e dados de XP.

### IPrinciple

```go
type IPrinciple interface {
    GetValueForTest() int
    GetNextLvlAggregateExp() int
    GetNextLvlBaseExp() int
    GetCurrentExp() int
    GetExpPoints() int
    GetLevel() int
}
```

Implementado por `NenPrinciple` e `Hatsu`. Usado pelo `Manager` para
operações genéricas sobre princípios, sem distinguir entre princípios regulares
e o Hatsu.

### ICategory

```go
type ICategory interface {
    experience.ITriggerCascadeExp  // CascadeUpgradeTrigger()
    GetPercent() float64
    GetValueForTest() int
    GetNextLvlAggregateExp() int
    GetNextLvlBaseExp() int
    GetCurrentExp() int
    GetExpPoints() int
    GetLevel() int
}
```

Implementado por `NenCategory`. Usado pelo `Hatsu` para operar sobre categorias
de forma polimórfica.

---

## Erros do Domínio

**Arquivo:** `spiritual/error.go`

| Erro | Quando ocorre |
|------|---------------|
| `ErrNenHexAlreadyInitialized` | `InitNenHexagon()` ou `Hatsu.Init()` chamados quando já inicializados |
| `ErrNenHexNotInitialized` | Operações de hexágono quando `nenHexagon == nil` |
| `ErrInvalidCategoryPercents` | `SetCategoryPercents()` com mapa de tamanho ≠ 6 |
| `ErrCategoryNotFound` | `Hatsu.Get()` com nome de categoria inexistente |
| `ErrPrincipleNotFound` | `Manager.Get()` com nome de princípio inexistente |

---

## Referências de Código

| Conceito | Arquivo |
|----------|---------|
| Hexágono Nen (algoritmo circular) | `spiritual/nen_hexagon.go` |
| Resultado de atualização do hexágono | `spiritual/nen_hexagon_update_result.go` |
| Categoria Nen (cascata + valor de teste) | `spiritual/nen_category.go` |
| Princípio Nen (cascata + valor de teste) | `spiritual/nen_principle.go` |
| Hatsu (coordenação de categorias) | `spiritual/hatsu.go` |
| Manager (fachada) | `spiritual/principles_manager.go` |
| Erros de domínio | `spiritual/error.go` |
| Interface IHatsu | `spiritual/i_hatsu.go` |
| Interface IPrinciple | `spiritual/i_principle.go` |
| Interface ICategory | `spiritual/i_category.go` |

> Todos os paths são relativos a `internal/domain/entity/character_sheet/`.
