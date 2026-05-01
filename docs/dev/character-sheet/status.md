# Barras de Status — HP, SP & AP

> Documentação técnica do sistema de barras de status (Health Points, Stamina Points,
> Aura Points) e do padrão Manager que as coordena.
> Pacote: `internal/domain/entity/character_sheet/status/`

## Visão Geral

As barras de status representam os recursos consumíveis do personagem em combate e
exploração. Cada barra é uma entidade reativa: ela **não** armazena XP nem participa
da cascata diretamente. Em vez disso, mantém referências a abilities, atributos e
skills, e recalcula seu `max` quando `Upgrade()` é chamado — tipicamente após cada
inserção de XP em qualquer parte da ficha.

As três barras compartilham uma base comum (`Bar`) e a mesma interface (`IStatusBar`),
mas cada uma tem uma **fórmula de cálculo distinta** com dependências diferentes.

---

## 1. Bar — Struct Base com Clamping

**Arquivo:** `status/status_bar.go`

`Bar` é o alicerce de todas as barras de status. Encapsula três campos inteiros:

| Campo | Significado | Valor inicial |
|-------|-------------|---------------|
| `min` | Piso da barra (mínimo possível) | `0` |
| `curr` | Valor atual do recurso | `0` |
| `max` | Teto da barra (máximo possível) | `0` (recalculado no primeiro `Upgrade()`) |

### Comportamento de clamping

Os métodos `IncreaseAt` e `DecreaseAt` aplicam clamping automático:

- **`IncreaseAt(value)`** — soma `value` ao `curr`, mas limita ao `max` via `min(temp, max)`
- **`DecreaseAt(value)`** — subtrai `value` do `curr`, mas limita ao `min` via `max(temp, min)`

Isso garante que o valor nunca ultrapasse os limites, independente da operação.

### Validação em SetCurrent

`SetCurrent(value)` impõe validação estrita:

```go
if value < b.min || value > b.max {
    return ErrInvalidValue
}
```

Diferente de `IncreaseAt`/`DecreaseAt` que fazem clamping silencioso, `SetCurrent`
**rejeita** valores fora do intervalo `[min, max]` com erro. Isso é intencional:
operações incrementais (dano, cura) podem clampar naturalmente, mas atribuições
diretas (ex: via API REST) devem ser explicitamente válidas.

---

## 2. HP — Pontos de Vida (Health Points)

**Arquivo:** `status/health_points_bar.go`

### Fórmula

```
HP_max = HP_BASE_VALUE + int(coeff * bonus)
```

Onde:

| Componente | Expressão | Fonte |
|------------|-----------|-------|
| `HP_BASE_VALUE` | `20` (constante) | Todo personagem começa com 20 HP base |
| `coeff` | `float64(vitality.GetLevel() + resistance.GetValue())` | Nível da perícia Vitalidade + Valor do atributo Resistência |
| `bonus` | `physicals.GetBonus()` | Bônus da ability Physicals (ver [`abilities-attributes.md`](abilities-attributes.md) §4) |

### Dependências

```
HealthPoints
├── physicals   (ability.IAbility)    → GetBonus()
├── resistance  (IDistributableAttribute) → GetValue() = points + level
└── vitality    (skill.ISkill)        → GetLevel()
```

### Natureza aditiva

A fórmula HP é **aditiva**: `HP_BASE_VALUE + int(coeff * bonus)`. Mesmo com
`coeff` e `bonus` zerados, o personagem terá no mínimo `HP_BASE_VALUE = 20` HP.
Compare com SP e AP, que são multiplicativos (podem resultar em zero).

### Auto-upgrade quando curado

Se o personagem está com vida cheia (`curr == max`), o `Upgrade()` atualiza
automaticamente `curr` para o novo `max`:

```go
if hp.curr == hp.max {
    hp.curr = maxVal
}
```

Isso evita a situação em que um personagem completamente curado recebe XP e fica
com `curr < max` sem motivo narrativo.

### TODOs preservados do código

- `TODO: Implement Min for hit_points` — `Min = -generateStatus.GetLvl()`. O HP
  mínimo poderá ser negativo (representando morte gradual), mas ainda não está
  implementado.
- `TODO: check how the buff interferes here` — Buffs nos atributos podem alterar
  `resistance.GetValue()` via ponteiro compartilhado (ver
  [`abilities-attributes.md`](abilities-attributes.md) §6), mas o impacto na
  fórmula de HP ainda não foi validado.
- `TODO: Implement else case (ex.: hp.current == hp.max - 1 -> threat % case)` —
  Quando o personagem **não** está totalmente curado, o comportamento ideal seria
  manter uma proporção `curr/max` ao recalcular. Atualmente, o `curr` não é
  ajustado nesse cenário.

---

## 3. SP — Pontos de Stamina (Stamina Points)

**Arquivo:** `status/stamina_points_bar.go`

### Fórmula

```
SP_max = SP_COEF_VALUE * int(coeff * bonus)
```

Onde:

| Componente | Expressão | Fonte |
|------------|-----------|-------|
| `SP_COEF_VALUE` | `10` (constante) | Multiplicador base do stamina |
| `coeff` | `float64(energy.GetLevel() + resistance.GetValue())` | Nível da perícia Energia + Valor do atributo Resistência |
| `bonus` | `physicals.GetBonus()` | Bônus da ability Physicals |

### Dependências

```
StaminaPoints
├── physicals   (ability.IAbility)    → GetBonus()
├── resistance  (IDistributableAttribute) → GetValue() = points + level
└── energy      (skill.ISkill)        → GetLevel()
```

### Natureza multiplicativa

Diferente do HP (aditivo), SP é **multiplicativo**: `SP_COEF_VALUE * int(coeff * bonus)`.
Se `coeff` ou `bonus` for zero, o SP máximo será zero. Isso é intencional: um
personagem nível 0 com zero pontos distribuídos não possui stamina para ações
físicas prolongadas.

Note que `resistance` aparece tanto em HP quanto em SP — é o atributo compartilhado.
Treinar Resistência beneficia ambas as barras simultaneamente.

### Mesma lógica de auto-upgrade e TODOs

O comportamento de `curr == max` e os TODOs são idênticos ao HP:

- `TODO: check how the buff interferes here`
- `TODO: Implement Min for stamina_points`
- `TODO: Implement else case (ex.: sp.current == sp.max - 1 -> threat % case)`

---

## 4. AP — Pontos de Aura (Aura Points)

**Arquivo:** `status/aura_points_bar.go`

### Fórmula

```
AP_max = int(AP_COEF_VALUE * coef * float64(bonus))
```

Onde:

| Componente | Expressão | Fonte |
|------------|-----------|-------|
| `AP_COEF_VALUE` | `10` (constante, originalmente `1000` — ver TODO abaixo) | Multiplicador base da aura |
| `coef` | `float64(mop.GetLevel() + conscienceNen.GetLevel())` | Nível da perícia MOP + Nível do atributo ConscienceNen |
| `bonus` | `int(spirituals.GetBonus())` | Bônus da ability Spirituals, **truncado para int** |

### Dependências

```
AuraPoints
├── spirituals    (ability.IAbility)     → GetBonus() → truncado para int
├── conscienceNen (attribute.IGameAttribute) → GetLevel()
└── mop           (skill.ISkill)         → GetLevel()
```

### Diferenças fundamentais em relação a HP/SP

1. **Usa `GetLevel()`, não `GetValue()`** — tanto `conscienceNen` quanto `mop`
   contribuem apenas com seu nível. Diferente de HP/SP que usam
   `resistance.GetValue()` (que inclui `points + level`). Isso reflete que
   atributos espirituais (`SpiritualAttribute`) **não possuem** campo `points`
   nem método `GetValue()` — apenas `GetLevel()` e `GetPower()`. Veja
   [`abilities-attributes.md`](abilities-attributes.md) §1.

2. **Interface `IGameAttribute`**, não `IDistributableAttribute` — `conscienceNen`
   é um atributo espiritual que não aceita distribuição de pontos. A interface
   mais restrita reflete essa limitação.

3. **Bônus truncado** — `int(ap.spirituals.GetBonus())` converte o bônus de
   `float64` para `int` **antes** da multiplicação, perdendo a parte fracionária.
   Em HP/SP, o bônus participa como `float64` na multiplicação com `coeff` e só
   é truncado no resultado final.

4. **Tripla multiplicação** — a fórmula é `AP_COEF_VALUE * coef * bonus`,
   totalmente multiplicativa. Qualquer fator zero resulta em AP zero.

5. **Validação nil no construtor** — `NewAuraPoints` retorna erro se
   `spirituals == nil`. Isso é necessário porque AP é condicional: personagens
   sem Nen (ex: `HalfSheet`) não possuem ability espiritual.

### Raciocínio da fórmula (do owner)

O código contém uma justificativa detalhada da fórmula atual:

> *"estou assumindo que o mopLvl é 0, o conscienceNenLvl é 1 (liberação dos
> shoukos), AP_COEF_VALUE é 10 e o bonus: (spiritLvl + charLvl) / 2 é maior
> que 5 e menor que 10. logo fica em torno de 600 e 900 (parece razoável).
> O MOP vai subir bastante naturalmente mesmo sem treino pelos outros parâmetros
> e um cálculo de padeiro para valores grandes também pareceu razoável."*

A fórmula anterior (comentada) usava `GetValue()` + `bonus` de forma aditiva:
`AP_COEF_VALUE * (mop.GetLevel() + conscienceNen.GetValue() + bonus)`. A versão
atual é mais multiplicativa e escala melhor para valores altos.

### TODOs preservados do código

- `TODO: review this value to upgrade AP formula` — `AP_COEF_VALUE` era
  originalmente `1000`, agora é `10`. O valor final ainda não foi decidido.
- `TODO: review formula` — a fórmula atual é considerada experimental.
- `TODO: check how the buff interferes here`
- `TODO: validar essa fórmula ousada que está no lugar da comentada acima` —
  refere-se à troca da fórmula aditiva pela multiplicativa.
- `TODO: Implement else case (ex.: ap.current == ap.max - 1 -> threat % case)`

---

## 5. Manager — Coordenação das Barras

**Arquivo:** `status/status_manager.go`

O `Manager` armazena todas as barras em um mapa `map[enum.StatusName]IStatusBar`
e fornece operações genéricas sobre elas.

### Upgrade() — recálculo global

```go
func (sm *Manager) Upgrade() error {
    for name := range sm.status {
        status, err := sm.Get(name)
        if err != nil { return err }
        status.Upgrade()
    }
    return nil
}
```

`Upgrade()` é chamado por `CharacterSheet` após **qualquer** inserção de XP
(perícia, atributo, princípio, categoria, proficiência). Ele itera todas as
barras registradas e delega para cada `Upgrade()` individual. Como cada barra
mantém referências por ponteiro às abilities/atributos/skills, ela enxerga os
valores já atualizados pela cascata.

### SetCurrent — validação delegada

`SetCurrent(name, value)` localiza a barra por nome e delega ao `SetCurrent`
do `Bar`, que aplica a validação `[min, max]`. É o ponto de entrada para
alteração direta de HP/SP/AP via API.

### Outros métodos

| Método | Retorno | Descrição |
|--------|---------|-----------|
| `Get(name)` | `IStatusBar, error` | Busca barra por enum. Retorna `ErrStatusNotFound` se inexistente |
| `GetMaxOf(name)` | `int, error` | Retorna `max` de uma barra específica |
| `GetMinOf(name)` | `int, error` | Retorna `min` de uma barra específica |
| `GetCurrentOf(name)` | `int, error` | Retorna `curr` de uma barra específica |
| `GetAllMaximuns()` | `map[StatusName]int` | Retorna `max` de todas as barras (usado após distribuição de pontos) |
| `GetAllStatus()` | `map[StatusName]IStatusBar` | Retorna o mapa completo (usado na renderização `ToString()`) |

---

## 6. IStatusBar — Interface Comum

**Arquivo:** `status/i_status_bar.go`

```go
type IStatusBar interface {
    IncreaseAt(value int) int
    DecreaseAt(value int) int
    Upgrade()
    GetMin() int
    GetCurrent() int
    GetMax() int
    SetCurrent(value int) error
}
```

Toda barra de status deve implementar estes 7 métodos. O `Manager` opera
exclusivamente via `IStatusBar`, sem conhecer o tipo concreto.

### Composição vs herança

`HealthPoints`, `StaminaPoints` e `AuraPoints` usam **embedding** de `*Bar`
para herdar `IncreaseAt`, `DecreaseAt`, `GetMin`, `GetCurrent`, `GetMax` e
`SetCurrent`. Cada uma **sobrescreve** apenas `Upgrade()` com sua fórmula
específica.

Isso significa que adicionar uma nova barra de status requer:

1. Criar uma struct com `*Bar` embeddado
2. Implementar `Upgrade()` com a fórmula desejada
3. Registrar no mapa do `Manager` via `BuildStatusManager` no factory

### Wiring no factory

O factory conecta cada barra às suas dependências (veja
[`factory.md`](factory.md) §1 para o fluxo completo):

```
HP = NewHealthPoints(physAbility, resistance, vitality)
SP = NewStaminaPoints(physAbility, resistance, energy)
AP = NewAuraPoints(spiritualAbility, conscience, mop)  ← condicional: só se spiritualAbility != nil
```

AP só é criado quando o personagem possui sistema espiritual. Personagens sem
Nen (criados via `BuildHalfSheet`) terão apenas HP e SP no mapa do Manager.

---

## Erros do Domínio

**Arquivo:** `status/error.go`

| Erro | Quando ocorre |
|------|---------------|
| `ErrStatusNotFound` | `Manager.Get()` com nome inexistente |
| `ErrSpiritualIsNil` | `NewAuraPoints()` com `spirituals == nil` |
| `ErrInvalidValue` | `Bar.SetCurrent()` com valor fora de `[min, max]` |

---

## Comparação das Fórmulas

| Barra | Fórmula | Tipo | Valor mínimo possível |
|-------|---------|------|----------------------|
| HP | `20 + int(coeff × bonus)` | Aditiva | 20 (base garantida) |
| SP | `10 × int(coeff × bonus)` | Multiplicativa | 0 |
| AP | `int(10 × coef × float64(bonus))` | Multiplicativa tripla | 0 |

| Barra | Skill | Atributo | Ability | Método do atributo |
|-------|-------|----------|---------|-------------------|
| HP | Vitality | Resistance | Physicals | `GetValue()` |
| SP | Energy | Resistance | Physicals | `GetValue()` |
| AP | MOP | ConscienceNen | Spirituals | `GetLevel()` |

---

## Referências de Código

| Conceito | Arquivo |
|----------|---------|
| Struct base (Bar) | `status/status_bar.go` |
| Health Points (HP) | `status/health_points_bar.go` |
| Stamina Points (SP) | `status/stamina_points_bar.go` |
| Aura Points (AP) | `status/aura_points_bar.go` |
| Status Manager | `status/status_manager.go` |
| Interface IStatusBar | `status/i_status_bar.go` |
| Erros de domínio | `status/error.go` |
| Bônus de ability (GetBonus) | `ability/ability.go` |
| PrimaryAttribute (GetValue) | `attribute/primary_attribute.go` |
| SpiritualAttribute (GetLevel) | `attribute/spiritual_attribute.go` |
| Wiring no factory (BuildStatusManager) | `sheet/character_sheet_factory.go` |

> Todos os paths são relativos a `internal/domain/entity/character_sheet/`.
