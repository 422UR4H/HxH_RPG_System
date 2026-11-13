# Armas

> Armas do System X System.

## Propriedades de uma Arma

Toda arma possui as seguintes características:

| Propriedade | Descrição |
|-------------|-----------|
| Dados | Dados rolados no ataque (ex: D10 + D4) |
| Dano Base | Bônus fixo de dano adicionado ao resultado dos dados |
| Defesa | Bônus defensivo ao bloquear com a arma |
| Peso | Quanto a arma pesa |
| Altura | Tamanho da arma em metros |
| Volume | Espaço que a arma ocupa no inventário |
| Arma de Fogo | Indica se é uma arma à distância moderna |

## Lista de Armas

O **dano** junta os dados e o dano base: **2 D12 +3** quer dizer dois dados de 12 lados, mais 3. A altura é em metros.

O sistema rola esses dados sozinho. Uma configuração de campanha e de partida vai permitir que jogadores e mestres rolem dados físicos e informem o resultado, em todas as rolagens ou só nas que escolherem (em breve).

São armas de fogo: Besta, Pistola .38, Uzi, AK-47, AR-15, Rifle e Metralhadora.

### Corpo a Corpo

| Nome | Dano | Peso | Altura |
|---|---|---|---|
| Punho | 2 D6 + D4 | 0,8 | 0,2 |
| Adaga | D8 +5 | 0,4 | 0,3 |
| Katar | D6 +6 | 0,8 | 0,4 |
| Espada | D10 + D4 +2 | 1,5 | 0,8 |
| Katana | D12 + D4 +7 | 1,3 | 1,0 |
| Cimitarra | D6 + D4 +4 | 1,2 | 0,9 |
| Rapieira | 2 D4 +5 | 1,2 | 1,0 |
| Lança | D8 + D4 +3 | 2,5 | 2,0 |
| Tridente | 3 D8 +3 | 2,5 | 1,5 |
| Machado | D10 + D6 +1 | 2,5 | 0,7 |
| Picareta | D8 + D6 +2 | 3,5 | 0,9 |
| Martelo | D12 + D6 | 2,5 | 0,6 |
| Maça | D12 + D4 +1 | 3,5 | 0,7 |
| Mangual | D12 + D4 +1 | 4,0 | 0,8 |
| Clava | 2 D8 +1 | 2,0 | 0,6 |
| Cajado | D10 + D8 | 2,5 | 1,7 |
| Chicote | D8 + 2 D4 | 1,3 | 2,5 |
| Foice | D6 + 2 D4 +2 | 3,0 | 1,6 |
| Tchaco | D10 +4 | 2,0 | 0,7 |

### Corpo a Corpo — Longas

| Nome | Dano | Peso | Altura |
|---|---|---|---|
| Espada Longa | D12 + D10 + D4 +2 | 2,5 | 1,2 |
| Lança Longa | D12 + D8 + D4 +3 | 4,5 | 3,0 |
| Alabarda | D12 + D10 + D6 +1 | 7,0 | 2,2 |
| Machado Longo | D12 + D10 + D6 +1 | 4,5 | 1,2 |
| Martelo de Guerra | 2 D12 + D6 | 6,0 | 1,2 |
| Maça Longa | 2 D12 + D4 +1 | 6,0 | 1,2 |
| Clava Longa | D12 + 2 D8 +1 | 4,0 | 1,2 |
| Foice Longa | D12 + D6 + 2 D4 +2 | 6,0 | 2,4 |

### Arremesso

| Nome | Dano | Peso | Altura |
|---|---|---|---|
| Adaga de Arremesso | D8 +2 | 0,3 | 0,2 |
| Machado de Arremesso | D10 +1 | 1,5 | 0,4 |
| Martelo de Arremesso | D12 | 1,5 | 0,4 |

### Projéteis, Armas de Fogo e Explosivos

| Nome | Dano | Peso | Altura |
|---|---|---|---|
| Arco | D12 +3 | 1,0 | 1,5 |
| Arco Longo | 2 D12 +3 | 2,0 | 2,0 |
| Besta | 3 D12 +2 | 4,0 | 0,9 |
| Pistola .38 | D12 +4 | 0,9 | 1,3 |
| Uzi | D12 + D8 +1 | 3,0 | 0,4 |
| AK-47 | 3 D10 +1 | 4,5 | 0,8 |
| AR-15 | 2 D10 +6 | 3,5 | 0,9 |
| Rifle | D12 + D10 +8 | 6,0 | 1,2 |
| Metralhadora | D12 + D10 +3 | 3,0 | 6,0 |
| Bomba | 3 D12 | 2,0 | 0,2 |

## Inventário de Armas

Cada personagem possui um inventário que permite gerenciar suas armas durante o jogo:

- **Adicionar** — adquirir e registrar uma nova arma no inventário
- **Remover** — descartar ou vender uma arma
- **Consultar** — verificar os detalhes de uma arma específica
- **Listar todas** — ver o inventário completo de armas

A arma atualmente equipada é a que determina os dados de dano nos ataques do personagem.

---

> **🔧 Para Desenvolvedores**
>
> Implementação técnica: [`docs/dev/weapons-dice.md`](../dev/weapons-dice.md)
> Código-fonte: `internal/domain/entity/item/`
>
> - TODO (penalidade): fora do texto do jogador até as armas estarem completas. `Weapon.GetPenality()` já calcula — corpo a corpo = peso da arma; arma de fogo = 1,0 se o peso ≥ 1,0, senão 0,0 — mas nada aplica o valor. O texto antigo dizia que a penalidade sai de Agilidade, "Ataque" e Flexibilidade; "Ataque" não é atributo, e o dono vai definir os atributos quando a penalidade existir.
>
> - TODO (custo de Stamina): fora do texto do jogador até as armas estarem completas. `Weapon.GetStaminaCost()` já calcula — corpo a corpo = peso da arma; arma de fogo = sempre 1,0 — mas nada desconta a Stamina no ataque.
