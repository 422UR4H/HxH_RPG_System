# Dados & Armas — Guia de Desenvolvimento

> Documentação técnica dos pacotes `die/` e `item/` do sistema de RPG HxH.

---

## 1. Die — Geração Aleatória

O pacote `die` encapsula a rolagem de dados com duas estratégias de geração:

1. **`crypto/rand` (primária)** — gera valores criptograficamente seguros via `cryptoRand.Int()`. É a fonte padrão para garantir aleatoriedade de qualidade em mecânicas de combate e testes.
2. **`math/rand/v2` (fallback)** — usado apenas quando `crypto/rand` falha (ex.: exaustão de entropia). Nesse cenário, o erro deveria ser logado, mas atualmente é silenciado.

> **TODO preservado do código-fonte:** `// TODO: log err` — o fallback não registra o erro de `crypto/rand`.

### Offset +1 (resultado 1-based)

Ambas as estratégias geram um valor no intervalo `[0, sides)`. O `+1` final garante que o resultado esteja em `[1, sides]`, como um dado físico real:

```go
d.result = int(n.Int64()) + 1  // crypto/rand
d.result = mathRand.IntN(sides) + 1  // fallback
```

### DieSides (enum)

O enum `enum.DieSides` define as faces válidas de um dado. `Die` armazena internamente o enum e expõe o valor inteiro via `GetSides()`. Os dados mais comuns no catálogo de armas são **d4, d6, d8, d10, d12**.

### Ciclo de vida

`NewDie(sides)` → `Roll()` → `GetResult()`. Cada chamada a `Roll()` substitui o resultado anterior. O `Die` é stateful: armazena o último resultado para consultas posteriores.

---

## 2. Weapon — Struct e Propriedades

O `Weapon` modela qualquer item ofensivo/defensivo carregável por um personagem.

> **TODO preservado do código-fonte:** `// TODO: improve weapons`

| Campo | Tipo | Significado no gameplay |
|-------|------|------------------------|
| `dice` | `[]int` | Faces dos dados rolados no ataque. Ex.: `[]int{12, 10, 4}` = rola d12 + d10 + d4. Mais dados = maior variância e potencial de dano. |
| `damage` | `int` | Bônus fixo de dano somado ao resultado dos dados. |
| `defense` | `int` | Bônus fixo de defesa concedido ao portador. |
| `weight` | `float64` | Peso da arma. **Dupla função**: para armas corpo a corpo, determina a penalidade de combate e o custo de stamina. |
| `height` | `float64` | Comprimento/altura da arma. Influencia alcance e requisitos de espaço. |
| `volume` | `int` | Espaço ocupado no inventário do personagem. |
| `isFireWeapon` | `bool` | Distingue armas de fogo (pistolas, rifles, etc.) de armas corpo a corpo. Altera completamente a lógica de penalidade e custo de stamina. |

### Cópia defensiva de `dice`

`GetDice()` retorna uma **cópia** do slice interno, impedindo mutação externa:

```go
func (w *Weapon) GetDice() []int {
    dice := make([]int, len(w.dice))
    copy(dice, w.dice)
    return dice
}
```

---

## 3. Penalidade & Custo de Stamina

A distinção entre armas de fogo e corpo a corpo é o ponto central da mecânica de custo:

### Armas de fogo (`isFireWeapon == true`)

| Métrica | Regra | Justificativa |
|---------|-------|---------------|
| **Penalidade** | `1.0` se `weight >= 1.0`, senão `0.0` | Armas de fogo leves (ex.: pistola compacta) não penalizam. Acima de 1kg, penalidade fixa — o recuo é o limitante, não o peso. |
| **Custo de stamina** | Sempre `1.0` | Disparar exige esforço mínimo e constante, independente do peso da arma. |

### Armas corpo a corpo (`isFireWeapon == false`)

| Métrica | Regra | Justificativa |
|---------|-------|---------------|
| **Penalidade** | `weight` (o próprio peso) | Armas mais pesadas são mais difíceis de manejar. Relação linear direta. |
| **Custo de stamina** | `weight` (o próprio peso) | Balancear golpes pesados consome mais energia. Mesmo valor da penalidade. |

### Fórmulas resumidas

```
GetPenality():
  isFireWeapon && weight >= 1.0  →  1.0
  isFireWeapon && weight <  1.0  →  0.0
  !isFireWeapon                  →  weight

GetStaminaCost():
  isFireWeapon   →  1.0
  !isFireWeapon  →  weight
```

> **Nota:** o nome `GetPenality` no código-fonte é um typo intencional preservado (deveria ser `GetPenalty`).

---

## 4. WeaponsFactory — Catálogo Pré-construído

`WeaponsManagerFactory` implementa o padrão **Factory** para criar um `WeaponsManager` populado com o catálogo completo de ~40 armas do jogo.

### Responsabilidade

O método `Build()` constrói o mapa `map[enum.WeaponName]Weapon` com todas as armas balanceadas pelo game design. Retorna um `*WeaponsManager` pronto para uso.

### Organização do catálogo

As armas seguem padrões de progressão por família:

| Família | Variantes | Padrão |
|---------|-----------|--------|
| Espadas | Sword → Longsword | Versão longa adiciona dados e peso |
| Machados | Axe → Longaxe, ThrowingAxe | Variante arremessável é mais leve |
| Martelos | Hammer → Warhammer, ThrowingHammer | Mesma progressão |
| Lanças | Spear → Longspear | Mais alcance, mais dados |
| Foices | Scythe → Longscythe | Adiciona d12 na versão longa |
| Armas de fogo | Pistol38, Ak47, Ar15, Rifle, Uzi, MachineGun, Crossbow | `isFireWeapon = true` |

### Padrões observáveis

- **Armas de arremesso** (ThrowingDagger, ThrowingAxe, ThrowingHammer) têm volume baixo (2–3) e dado único.
- **Todas as armas de fogo** usam d10 ou d12 como base — alta variância por disparo.
- **`defense`** é `0` para todas as armas no catálogo atual — campo reservado para escudos ou futuras armas defensivas.
- **Bomb** (`isFireWeapon = false`) é classificada como corpo a corpo apesar de ser explosiva — provavelmente uma granada de arremesso manual.

---

## 5. WeaponsManager — CRUD e Acessores Delegados

`WeaponsManager` é o **repositório em memória** das armas de um personagem ou cenário. Encapsula o mapa de armas e expõe operações CRUD + acessores delegados.

### Operações

| Método | Comportamento |
|--------|---------------|
| `Add(name, weapon)` | Insere ou sobrescreve uma arma no mapa |
| `Delete(name)` | Remove a arma (silencioso se não existir) |
| `Get(name)` | Retorna a arma ou `ErrWeaponNotFound` |
| `GetAll()` | Retorna o mapa completo (referência, não cópia!) |

### Acessores delegados

O `WeaponsManager` delega para `Weapon` os métodos: `GetDamage`, `GetDefense`, `GetWeight`, `GetHeight`, `GetVolume`, `IsFireWeapon`, `GetDice`, `GetPenality`, `GetStaminaCost`. Cada um recebe um `enum.WeaponName`, busca a arma no mapa e delega a chamada.

### ⚠️ Incompatibilidade de tipo com IItem

A interface `IItem` define:

```go
type IItem interface {
    GetStaminaCost() int   // retorna int
    // ...
}
```

Porém, `Weapon.GetStaminaCost()` retorna `float64`. Isso significa que **`Weapon` não satisfaz a interface `IItem`** em Go, pois as assinaturas diferem no tipo de retorno. Qualquer tentativa de usar `Weapon` como `IItem` resultará em erro de compilação.

Esse desalinhamento precisa ser resolvido: ou `IItem` deve retornar `float64`, ou `Weapon` deve retornar `int` (com truncamento ou arredondamento).

### Tratamento de erro

O erro `ErrWeaponNotFound` é um `DomainError` criado via `domain.NewDomainError()`, seguindo o padrão de erros do projeto.

---

## Referências de Código

| Conceito | Arquivo | Pacote |
|----------|---------|--------|
| Die (rolagem) | `internal/domain/entity/die/die.go` | `die` |
| DieSides enum | `internal/domain/entity/enum/` | `enum` |
| Weapon struct | `internal/domain/entity/item/weapon.go` | `item` |
| WeaponsManagerFactory | `internal/domain/entity/item/weapons_factory.go` | `item` |
| WeaponsManager | `internal/domain/entity/item/weapons_manager.go` | `item` |
| IItem interface | `internal/domain/entity/item/i_item.go` | `item` |
| ErrWeaponNotFound | `internal/domain/entity/item/error.go` | `item` |
| WeaponName enum | `internal/domain/entity/enum/` | `enum` |
