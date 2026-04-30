# Armas

> Sistema de armas do HxH RPG.

## Propriedades de uma Arma

Toda arma possui as seguintes características:

| Propriedade | Descrição |
|-------------|-----------|
| Dados | Dados rolados no ataque (ex: D10 + D4) |
| Dano Base | Bônus fixo de dano adicionado ao resultado dos dados |
| Defesa | Bônus defensivo ao bloquear com a arma |
| Peso | Determina penalidade e custo de stamina |
| Altura | Tamanho da arma em metros |
| Volume | Espaço que a arma ocupa no inventário |
| Arma de Fogo | Indica se é uma arma à distância moderna |

## Penalidade

A penalidade de uma arma é subtraída dos atributos **Agilidade**, **Ataque** e **Flexibilidade** do personagem enquanto equipada.

### Cálculo:
- **Corpo a corpo:** penalidade = peso da arma
- **Arma de fogo:** penalidade = 1,0 se o peso for maior ou igual a 1,0; caso contrário, 0,0

> Armas de fogo leves (peso menor que 1,0) não impõem penalidade de mobilidade.

## Custo de Stamina

O custo em Pontos de Stamina (SP) por ataque realizado.

### Cálculo:
- **Corpo a corpo:** custo = peso da arma
- **Arma de fogo:** custo = sempre 1,0 (independente do peso)

> Armas de fogo consomem pouca energia física do usuário comparadas ao seu peso.

## Armas Corpo a Corpo

| Nome | Dados | Dano | Peso | Altura |
|------|-------|------|------|--------|
| Adaga | D8 | 5 | 0,4 | 0,3 |
| Adaga de Arremesso | D8 | 2 | 0,3 | 0,2 |
| Espada | D10, D4 | 2 | 1,5 | 0,8 |
| Espada Longa | D12, D10, D4 | 2 | 2,5 | 1,2 |
| Katana | D4, D12 | 7 | 1,3 | 1,0 |
| Cimitarra | D6, D4 | 4 | 1,2 | 0,9 |
| Rapieira | D4, D4 | 5 | 1,2 | 1,0 |
| Katar | D6 | 6 | 0,8 | 0,4 |
| Lança | D8, D4 | 3 | 2,5 | 2,0 |
| Lança Longa | D12, D8, D4 | 3 | 4,5 | 3,0 |
| Alabarda | D12, D10, D6 | 1 | 7,0 | 2,2 |
| Tridente | D8, D8, D8 | 3 | 2,5 | 1,5 |
| Machado | D10, D6 | 1 | 2,5 | 0,7 |
| Machado Longo | D10, D6, D12 | 1 | 4,5 | 1,2 |
| Machado de Arremesso | D10 | 1 | 1,5 | 0,4 |
| Picareta | D8, D6 | 2 | 3,5 | 0,9 |
| Martelo | D12, D6 | 0 | 2,5 | 0,6 |
| Martelo de Guerra | D12, D12, D6 | 0 | 6,0 | 1,2 |
| Martelo de Arremesso | D12 | 0 | 1,5 | 0,4 |
| Massa | D12, D4 | 1 | 3,5 | 0,7 |
| Massa Longa | D12, D12, D4 | 1 | 6,0 | 1,2 |
| Mangual | D12, D4 | 1 | 4,0 | 0,8 |
| Cajado | D10, D8 | 0 | 2,5 | 1,7 |
| Chicote | D4, D4, D8 | 0 | 1,3 | 2,5 |
| Foice | D4, D4, D6 | 2 | 3,0 | 1,6 |
| Foice Longa | D4, D4, D6, D12 | 2 | 6,0 | 2,4 |
| Clava | D8, D8 | 1 | 2,0 | 0,6 |
| Clava Longa | D8, D8, D12 | 1 | 4,0 | 1,2 |
| Tchaco | D10 | 4 | 2,0 | 0,7 |
| Punho | D6, D6, D4 | 0 | 0,8 | 0,2 |
| Bomba | D12, D12, D12 | 0 | 2,0 | 0,2 |

## Armas de Fogo

| Nome | Dados | Dano | Peso | Altura |
|------|-------|------|------|--------|
| Besta | D12, D12, D12 | 2 | 4,0 | 0,9 |
| AK-47 | D10, D10, D10 | 1 | 4,5 | 0,8 |
| AR-15 | D10, D10 | 6 | 3,5 | 0,9 |
| Metralhadora | D12, D10 | 3 | 3,0 | 6,0 |
| Pistola .38 | D12 | 4 | 0,9 | 1,3 |
| Rifle | D12, D10 | 8 | 6,0 | 1,2 |
| Uzi | D12, D8 | 1 | 3,0 | 0,4 |

## Inventário de Armas

Cada personagem possui um inventário que permite gerenciar suas armas durante o jogo:

- **Adicionar** — adquirir e registrar uma nova arma no inventário
- **Remover** — descartar ou vender uma arma
- **Consultar** — verificar os detalhes de uma arma específica
- **Listar todas** — ver o inventário completo de armas

A arma atualmente equipada é a que determina os dados de dano, a penalidade de mobilidade e o custo de stamina nos ataques do personagem.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/weapons-dice.md`](../dev/weapons-dice.md)
> Código-fonte: `internal/domain/entity/item/`
