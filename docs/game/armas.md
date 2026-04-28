# Armas (Weapons)

> Sistema de armas do HxH RPG.

## Propriedades de uma Arma

Toda arma possui os seguintes atributos:

| Propriedade | Tipo | Descrição |
|-------------|------|-----------|
| Dados (Dice) | Lista de inteiros | Lados dos dados rolados no ataque |
| Dano Base (Damage) | Inteiro | Bônus fixo de dano adicionado ao resultado dos dados |
| Defesa (Defense) | Inteiro | Bônus defensivo ao bloquear com a arma |
| Peso (Weight) | Decimal | Determina penalidade e custo de stamina |
| Altura (Height) | Decimal | Tamanho da arma em metros |
| Volume | Inteiro | Espaço que a arma ocupa no inventário |
| Arma de Fogo (IsFireWeapon) | Booleano | Distingue armas à distância modernas |

## Penalidade (Penalty)

A penalidade de uma arma é subtraída dos atributos **Agilidade (Agility)**, **Ataque** e **Flexibilidade (Flexibility)** do personagem enquanto equipada.

### Cálculo:
- **Corpo a corpo:** penalidade = peso da arma
- **Arma de fogo:** penalidade = `1.0` se peso ≥ 1.0, caso contrário `0.0`

> Armas de fogo leves (peso < 1.0) não impõem penalidade de mobilidade.

## Custo de Stamina (Stamina Cost)

O custo em Pontos de Stamina (SP) por ataque realizado.

### Cálculo:
- **Corpo a corpo:** custo = peso da arma
- **Arma de fogo:** custo = sempre `1.0` (independente do peso)

> Armas de fogo consomem pouca energia física do usuário comparadas ao seu peso.

## Segurança de Dados

Os dados (dice) de uma arma são retornados como **cópia**. Isso garante que clients externos não possam modificar acidentalmente os dados internos da arma.

## Armas Corpo a Corpo (Melee Weapons)

| Nome (PT-BR) | Nome (EN) | Dados | Dano | Peso | Altura |
|--------------|-----------|-------|------|------|--------|
| Adaga | Dagger | D8 | 5 | 0.4 | 0.3 |
| Adaga de Arremesso | ThrowingDagger | D8 | 2 | 0.3 | 0.2 |
| Espada | Sword | D10, D4 | 2 | 1.5 | 0.8 |
| Espada Longa | Longsword | D12, D10, D4 | 2 | 2.5 | 1.2 |
| Katana | Katana | D4, D12 | 7 | 1.3 | 1.0 |
| Cimitarra | Scimitar | D6, D4 | 4 | 1.2 | 0.9 |
| Rapieira | Rapier | D4, D4 | 5 | 1.2 | 1.0 |
| Katar | Katar | D6 | 6 | 0.8 | 0.4 |
| Lança | Spear | D8, D4 | 3 | 2.5 | 2.0 |
| Lança Longa | Longspear | D12, D8, D4 | 3 | 4.5 | 3.0 |
| Alabarda | Halberd | D12, D10, D6 | 1 | 7.0 | 2.2 |
| Tridente | Trident | D8, D8, D8 | 3 | 2.5 | 1.5 |
| Machado | Axe | D10, D6 | 1 | 2.5 | 0.7 |
| Machado Longo | Longaxe | D10, D6, D12 | 1 | 4.5 | 1.2 |
| Machado de Arremesso | ThrowingAxe | D10 | 1 | 1.5 | 0.4 |
| Picareta | Pickaxe | D8, D6 | 2 | 3.5 | 0.9 |
| Martelo | Hammer | D12, D6 | 0 | 2.5 | 0.6 |
| Martelo de Guerra | Warhammer | D12, D12, D6 | 0 | 6.0 | 1.2 |
| Martelo de Arremesso | ThrowingHammer | D12 | 0 | 1.5 | 0.4 |
| Massa | Massa | D12, D4 | 1 | 3.5 | 0.7 |
| Massa Longa | Longmass | D12, D12, D4 | 1 | 6.0 | 1.2 |
| Mangual | Mangual | D12, D4 | 1 | 4.0 | 0.8 |
| Cajado | Staff | D10, D8 | 0 | 2.5 | 1.7 |
| Chicote | Whip | D4, D4, D8 | 0 | 1.3 | 2.5 |
| Foice | Scythe | D4, D4, D6 | 2 | 3.0 | 1.6 |
| Foice Longa | Longscythe | D4, D4, D6, D12 | 2 | 6.0 | 2.4 |
| Clava | Club | D8, D8 | 1 | 2.0 | 0.6 |
| Clava Longa | Longclub | D8, D8, D12 | 1 | 4.0 | 1.2 |
| Tchaco | Tchaco | D10 | 4 | 2.0 | 0.7 |
| Punho | Fist | D6, D6, D4 | 0 | 0.8 | 0.2 |
| Bomba | Bomb | D12, D12, D12 | 0 | 2.0 | 0.2 |

## Armas de Fogo (Fire Weapons)

| Nome (PT-BR) | Nome (EN) | Dados | Dano | Peso | Altura |
|--------------|-----------|-------|------|------|--------|
| Besta | Crossbow | D12, D12, D12 | 2 | 4.0 | 0.9 |
| AK-47 | Ak47 | D10, D10, D10 | 1 | 4.5 | 0.8 |
| AR-15 | Ar15 | D10, D10 | 6 | 3.5 | 0.9 |
| Metralhadora | MachineGun | D12, D10 | 3 | 3.0 | 6.0 |
| Pistola .38 | Pistol38 | D12 | 4 | 0.9 | 1.3 |
| Rifle | Rifle | D12, D10 | 8 | 6.0 | 1.2 |
| Uzi | Uzi | D12, D8 | 1 | 3.0 | 0.4 |

## Gerenciador de Armas (Weapons Manager)

Cada personagem possui um gerenciador que controla seu inventário de armas com operações:
- **Adicionar** (Add) — registrar nova arma
- **Remover** (Delete) — descartar arma
- **Consultar** (Get) — verificar arma específica
- **Listar Todas** (GetAll) — ver inventário completo

O gerenciador também delega cálculos de penalidade, custo de stamina e dados de dano para a arma ativa.
